package service

import (
	"context"
	"fmt"

	"dagger.io/dagger"
)

type BuildService interface {
	Build(ctx context.Context) error
}

type buildServiceImpl struct {
	client *dagger.Client
}

// TODO: Make a wrapper interface around the dagger client
func NewBuildService(client *dagger.Client) BuildService {
	return &buildServiceImpl{client: client}
}

// Build performs the actual build using Dagger.
func (s *buildServiceImpl) Build(ctx context.Context) error {
	out, err := s.client.Container().
		From("alpine:latest").
		WithExec([]string{"echo", "Building with Dagger!"}).
		Stdout(ctx)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}
