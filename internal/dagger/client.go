package dagger

import (
	"context"
	"os"

	"dagger.io/dagger"
	"dagger.io/dagger/dag"
)

type DaggerClient interface {
	RunCommand(ctx context.Context, image string, command []string) (string, error)
	RunCommandWithMount(ctx context.Context, image string, command []string, mountPath, hostDir string) (string, error)
	Close() error
}

type DaggerClientImpl struct {
	client *dagger.Client
}

func NewDaggerClient(ctx context.Context) (DaggerClient, error) {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
	if err != nil {
		return nil, err
	}
	return &DaggerClientImpl{client}, nil
}

func (r *DaggerClientImpl) RunCommand(ctx context.Context, image string, command []string) (string, error) {
	return r.client.Container().
		From(image).
		WithExec(command).
		Stdout(ctx)
}

func (r *DaggerClientImpl) RunCommandWithMount(ctx context.Context, image string, command []string, mountPath, hostDir string) (string, error) {
	dir := r.client.Host().Directory(hostDir)

	return r.client.Container().
		From(image).
		WithMountedDirectory(mountPath, dir).
		WithWorkdir(mountPath).
		WithExec(command, dagger.ContainerWithExecOpts{ExperimentalPrivilegedNesting: true}).
		WithUnixSocket(
			"/var/run/docker.sock",
			dag.Host().UnixSocket("/var/run/docker.sock"),
		).
		With(r.TestContainersSupport()).
		Stdout(ctx)
}

func (r *DaggerClientImpl) Close() error {
	return r.client.Close()
}

func (r *DaggerClientImpl) TestContainersSupport() dagger.WithContainerFunc {
	container := r.client.Container().From("cgr.dev/chainguard/docker-dind")

	docker := container.
		WithoutEnvVariable("DOCKER_TLS_CERTDIR").
		WithExposedPort(2375).
		WithDefaultArgs([]string{"--tls=false"}).
		AsService(dagger.ContainerAsServiceOpts{UseEntrypoint: true, ExperimentalPrivilegedNesting: true, InsecureRootCapabilities: true})
	return func(c *dagger.Container) *dagger.Container {
		return c.WithEnvVariable("TESTCONTAINERS_RYUK_DISABLED", "true").
			//WithEnvVariable("TESTCONTAINERS_HUB_IMAGE_NAME_PREFIX", "true"). commenting out until we have our own image registry.
			WithEnvVariable("DOCKER_HOST", "tcp://docker:2375").
			WithoutEnvVariable("DOCKER_TLS_CERTDIR").
			WithServiceBinding("docker", docker)
	}
}
