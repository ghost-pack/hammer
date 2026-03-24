package service

import (
	"context"
	"fmt"

	"github.com/ghost-pack/hammer/internal/dagger"
)

type BuildService interface {
	Build(ctx context.Context) error
}

type buildServiceImpl struct {
	client dagger.DaggerClient
}

func NewBuildService(client dagger.DaggerClient) BuildService {
	return &buildServiceImpl{client: client}
}

func (s *buildServiceImpl) Build(ctx context.Context) error {
	out, err := s.client.RunCommand(ctx, "alpine:latest", []string{"echo", "Building with Dagger!"})
	if err != nil {
		return err
	}
	fmt.Println(out)
	fmt.Println("sup also this is brian")
	return nil
}
