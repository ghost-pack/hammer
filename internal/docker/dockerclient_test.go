package docker

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// shared setup — one test registry per test run is fine
func setupRegistry(t *testing.T) string {
	t.Helper()
	s := httptest.NewServer(registry.New())
	t.Cleanup(s.Close)
	return strings.TrimPrefix(s.URL, "http://")
}

// push a random image into the test registry to use as a base
func seedBaseImage(t *testing.T, host string) string {
	t.Helper()
	ref := fmt.Sprintf("%s/base:latest", host)
	img, err := random.Image(512, 1)
	if err != nil {
		t.Fatalf("random.Image: %v", err)
	}
	tag, err := name.NewTag(ref, name.Insecure)
	if err != nil {
		t.Fatalf("name.NewTag: %v", err)
	}
	if err := remote.Write(tag, img, remote.WithAuthFromKeychain(authn.DefaultKeychain)); err != nil {
		t.Fatalf("seeding base image: %v", err)
	}
	return ref
}

func fakeBinary(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "app")
	if err := os.WriteFile(path, []byte("not a real binary but good enough"), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func newTestBuilder() *ClientImpl {
	return &ClientImpl{
		keychain: authn.DefaultKeychain,
	}
}

func TestBuild(t *testing.T) {
	host := setupRegistry(t)
	baseImage := seedBaseImage(t, host)
	binary := fakeBinary(t)

	tests := []struct {
		name       string
		baseImage  string
		binaryPath string
		wantErr    bool
	}{
		{
			name:       "happy path",
			baseImage:  baseImage,
			binaryPath: binary,
		},
		{
			name:       "binary does not exist",
			baseImage:  baseImage,
			binaryPath: "/nonexistent/app",
			wantErr:    true,
		},
		{
			name:       "invalid base image ref",
			baseImage:  ":::not-a-ref:::",
			binaryPath: binary,
			wantErr:    true,
		},
		{
			name:       "base image not in registry",
			baseImage:  host + "/doesnotexist:latest",
			binaryPath: binary,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tarPath := filepath.Join(t.TempDir(), "image.tar")
			builder := newTestBuilder()

			err := builder.Build(context.Background(), tt.baseImage, tt.binaryPath, tarPath)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Build() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if _, err := os.Stat(tarPath); err != nil {
					t.Errorf("expected tar to exist at %s: %v", tarPath, err)
				}
			}
		})
	}
}

func TestPush(t *testing.T) {
	host := setupRegistry(t)
	baseImage := seedBaseImage(t, host)
	binary := fakeBinary(t)

	// build a real tar once and share it across push test cases
	tarPath := filepath.Join(t.TempDir(), "image.tar")
	builder := newTestBuilder()
	if err := builder.Build(context.Background(), baseImage, binary, tarPath); err != nil {
		t.Fatalf("setup Build: %v", err)
	}

	tests := []struct {
		name     string
		tarPath  string
		imageTag string
		wantErr  bool
	}{
		{
			name:     "happy path",
			tarPath:  tarPath,
			imageTag: fmt.Sprintf("%s/myapp:abc1234", host),
		},
		{
			name:     "tar does not exist",
			tarPath:  "/nonexistent/image.tar",
			imageTag: fmt.Sprintf("%s/myapp:abc1234", host),
			wantErr:  true,
		},
		{
			name:     "invalid image tag",
			tarPath:  tarPath,
			imageTag: ":::bad:::",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := builder.Push(context.Background(), tt.tarPath, tt.imageTag)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Push() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				// pull it back and verify it actually landed
				ref, _ := name.ParseReference(tt.imageTag, name.Insecure)
				img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
				if err != nil {
					t.Fatalf("image not found in registry after push: %v", err)
				}
				layers, _ := img.Layers()
				// base has 1 layer, we added 1, so expect 2
				if len(layers) != 2 {
					t.Errorf("expected 2 layers, got %d", len(layers))
				}
			}
		})
	}
}
