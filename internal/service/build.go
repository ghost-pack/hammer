package service

import (
    "context"
    "fmt"

    "dagger.io/dagger"
)

// BuildService handles build operations.
type BuildService struct {
    client *dagger.Client
}

// NewBuildService creates a new build service.
func NewBuildService(client *dagger.Client) *BuildService {
    return &BuildService{client: client}
}

// Build performs the actual build using Dagger.
func (s *BuildService) Build(ctx context.Context) error {
    // Example: run a container
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