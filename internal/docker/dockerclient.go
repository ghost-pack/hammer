package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/moby/moby/client"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type DockerClient interface {
	Build(ctx context.Context, binaryPath, imageTag string) error
}

type DockerClientImpl struct {
	client.APIClient
}

func NewDockerClient() (DockerClient, error) {
	cli, err := client.New(
		client.FromEnv,
	)
	if err != nil {
		return nil, fmt.Errorf("creating docker client: %w", err)
	}
	return &DockerClientImpl{cli}, nil
}

func (b *DockerClientImpl) Build(ctx context.Context, binaryPath, imageTag string) error {
	ctx, span := tracing.Tracer("docker docker").Start(ctx, "docker docker",
		trace.WithAttributes(
			attribute.String("cmd", "docker"),
			attribute.StringSlice("args", []string{"docker", "-f", binaryPath, "-t", imageTag})))
	defer span.End()

	buildCtx, err := tarContext(binaryPath)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("creating docker context: %w", err)
	}
	defer buildCtx.Close()

	resp, err := b.ImageBuild(ctx, buildCtx, client.ImageBuildOptions{
		Tags:       []string{imageTag},
		Dockerfile: "Dockerfile",
		Remove:     true,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("image docker: %w", err)
	}
	defer resp.Body.Close()

	if err := streamBuildOutput(resp.Body); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("streaming docker output: %w", err)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// tarContext creates an in-memory tar archive containing:
//   - the Go binary (renamed to "app")
//   - a Dockerfile that layers it onto cgr.dev/chainguard/static
func tarContext(binaryPath string) (io.ReadCloser, error) {
	binaryData, err := os.ReadFile(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("reading binary %q: %w", binaryPath, err)
	}

	dockerfile := []byte(`FROM cgr.dev/chainguard/static:latest
COPY app /usr/local/bin/app
ENTRYPOINT ["/usr/local/bin/app"]
`)

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	files := []struct {
		name string
		mode int64
		data []byte
	}{
		{"Dockerfile", 0644, dockerfile},
		{"app", 0755, binaryData},
	}

	for _, f := range files {
		hdr := &tar.Header{
			Name: f.name,
			Mode: f.mode,
			Size: int64(len(f.data)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, fmt.Errorf("tar header %s: %w", f.name, err)
		}
		if _, err := tw.Write(f.data); err != nil {
			return nil, fmt.Errorf("tar write %s: %w", f.name, err)
		}
	}

	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("closing tar: %w", err)
	}

	return io.NopCloser(&buf), nil
}

// buildEvent mirrors the JSON lines Docker streams during a docker.
type buildEvent struct {
	Stream string `json:"stream"`
	Error  string `json:"error"`
}

// streamBuildOutput prints docker log lines and returns the first error seen.
func streamBuildOutput(r io.Reader) error {
	dec := json.NewDecoder(r)
	for {
		var ev buildEvent
		if err := dec.Decode(&ev); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if ev.Stream != "" {
			fmt.Print(ev.Stream)
		}
		if ev.Error != "" {
			return fmt.Errorf("%s", ev.Error)
		}
	}
}
