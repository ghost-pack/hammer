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
	Build(ctx context.Context, baseImage, binaryPath, imageTag string) error
	Tag(ctx context.Context, source, target string) error
	Push(ctx context.Context, image string) error
	Close() error
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

func (b *DockerClientImpl) Close() error {
	return b.APIClient.Close()
}

func (b *DockerClientImpl) Tag(ctx context.Context, source, target string) error {
	ctx, span := tracing.Tracer("docker tag").Start(ctx, "docker tag",
		trace.WithAttributes(
			attribute.String("source", source),
			attribute.String("target", target),
		))
	defer span.End()
	_, err := b.ImageTag(ctx, client.ImageTagOptions{
		Source: source,
		Target: target,
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("tagging image %s → %s: %w", source, target, err)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func (b *DockerClientImpl) Push(ctx context.Context, image string) error {
	ctx, span := tracing.Tracer("docker push").Start(ctx, "docker push",
		trace.WithAttributes(attribute.String("image", image)),
	)
	defer span.End()

	registryAuth, err := resolveRegistryAuth(image)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("resolving registry auth: %w", err)
	}

	pushResp, err := b.ImagePush(ctx, image, client.ImagePushOptions{
		RegistryAuth: registryAuth,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("pushing image %s: %w", image, err)
	}
	defer pushResp.Close()

	if err := streamPushOutput(pushResp); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("streaming push output: %w", err)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func (b *DockerClientImpl) Build(ctx context.Context, baseImage, binaryPath, imageTag string) error {
	ctx, span := tracing.Tracer("docker build").Start(ctx, "docker build",
		trace.WithAttributes(
			attribute.String("cmd", "docker"),
			attribute.StringSlice("args", []string{"build", "-f", binaryPath, "-t", imageTag})))
	defer span.End()

	buildCtx, err := tarContext(binaryPath, baseImage)
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
		Labels: map[string]string{
			"org.opencontainers.image.revision": os.Getenv("COMMIT_SHA"),
			"org.opencontainers.image.version":  os.Getenv("COMMIT_SHA"),
		},
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

func tarContext(binaryPath string, baseImage string) (io.ReadCloser, error) {
	binaryData, err := os.ReadFile(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("reading binary %q: %w", binaryPath, err)
	}

	dockerfile := []byte(fmt.Sprintf(`FROM %s
COPY app /usr/local/bin/app
ENTRYPOINT ["/usr/local/bin/app"]
`, baseImage))

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Point of this function is to package up the Dockerfile and the binary (we just made) into a tar file.
	files := []struct {
		name string
		mode int64 // setting mode here skips chmod
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

type pushMessage struct {
	Status   string `json:"status"`
	Progress string `json:"progress"`
	ID       string `json:"id"`
	Error    string `json:"error"`
}

func streamPushOutput(r io.Reader) error {
	dec := json.NewDecoder(r)
	for {
		var msg pushMessage
		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if msg.Error != "" {
			return fmt.Errorf("push error: %s", msg.Error)
		}
		if msg.ID != "" && msg.Status != "" {
			fmt.Printf("%s: %s %s\n", msg.ID, msg.Status, msg.Progress)
		} else if msg.Status != "" {
			fmt.Println(msg.Status)
		}
	}
}
