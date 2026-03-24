package dagger

import (
	"context"
	"os"

	"dagger.io/dagger"
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
	// Get the host directory (relative to the current working directory)
	dir := r.client.Host().Directory(hostDir)

	return r.client.Container().
		From(image).
		WithMountedDirectory(mountPath, dir). // Mount at the given path
		WithWorkdir(mountPath).               // Set that as working directory
		WithExec(command).
		Stdout(ctx)
}

func (r *DaggerClientImpl) Close() error {
	return r.client.Close()
}
