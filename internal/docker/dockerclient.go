package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Client interface {
	Build(ctx context.Context, baseImage, binaryPath, tarPath string) error
	Push(ctx context.Context, tarPath, imageTag string) error
}

type ClientImpl struct {
	keychain authn.Keychain
}

func NewClient() Client {
	return &ClientImpl{
		keychain: google.Keychain,
	}
}

func (b *ClientImpl) Build(ctx context.Context, baseImage, binaryPath, tarPath string) error {
	ctx, span := tracing.Tracer("image build").Start(ctx, "image build",
		trace.WithAttributes(
			attribute.String("baseImage", baseImage),
			attribute.String("binaryPath", binaryPath),
		))
	defer span.End()

	img, err := assembleImage(ctx, baseImage, binaryPath, b.keychain)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("assembling image: %w", err)
	}

	tag, err := name.NewTag("myapp:local")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("parsing image tag %q: %w", "myapp:local", err)
	}

	if err := tarball.WriteToFile(tarPath, tag, img); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("writing image tar to %s: %w", tarPath, err)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func (b *ClientImpl) Push(ctx context.Context, tarPath, imageTag string) error {
	ctx, span := tracing.Tracer("image push").Start(ctx, "image push",
		trace.WithAttributes(
			attribute.String("imageTag", imageTag),
			attribute.String("tarPath", tarPath),
		))
	defer span.End()

	img, err := tarball.ImageFromPath(tarPath, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("loading image from %s: %w", tarPath, err)
	}

	tag, err := name.NewTag(imageTag)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("parsing image tag %q: %w", imageTag, err)
	}

	if err := remote.Write(tag, img,
		remote.WithContext(ctx),
		remote.WithAuthFromKeychain(google.Keychain),
	); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("pushing %s: %w", imageTag, err)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func assembleImage(ctx context.Context, baseImageRef, binaryPath string, keychain authn.Keychain) (v1.Image, error) {
	ref, err := name.ParseReference(baseImageRef)
	if err != nil {
		return nil, fmt.Errorf("parsing base image reference %q: %w", baseImageRef, err)
	}

	base, err := remote.Image(ref,
		remote.WithContext(ctx),
		remote.WithAuthFromKeychain(keychain),
	)
	if err != nil {
		return nil, fmt.Errorf("pulling base image %q: %w", baseImageRef, err)
	}

	layer, err := appLayer(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("creating app layer: %w", err)
	}

	img, err := mutate.AppendLayers(base, layer)
	if err != nil {
		return nil, fmt.Errorf("appending layer: %w", err)
	}

	cf, err := img.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("reading image config: %w", err)
	}
	cf = cf.DeepCopy()
	cf.Config.Entrypoint = []string{"/usr/local/bin/app"}
	cf.Config.Labels = map[string]string{
		"org.opencontainers.image.revision": os.Getenv("COMMIT_SHA"),
		"org.opencontainers.image.version":  os.Getenv("COMMIT_SHA"),
	}

	return mutate.ConfigFile(img, cf)
}

func appLayer(binaryPath string) (v1.Layer, error) {
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("reading binary %q: %w", binaryPath, err)
	}

	opener := func() (io.ReadCloser, error) {
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)

		if err := tw.WriteHeader(&tar.Header{
			Name:     "usr/local/bin/app",
			Mode:     0755,
			Size:     int64(len(data)),
			Typeflag: tar.TypeReg,
			ModTime:  time.Time{},
		}); err != nil {
			return nil, fmt.Errorf("tar header: %w", err)
		}
		if _, err := tw.Write(data); err != nil {
			return nil, fmt.Errorf("tar write: %w", err)
		}
		if err := tw.Close(); err != nil {
			return nil, fmt.Errorf("tar close: %w", err)
		}
		return io.NopCloser(&buf), nil
	}

	return tarball.LayerFromOpener(opener)
}
