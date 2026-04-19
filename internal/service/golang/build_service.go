package golang

import (
	"context"

	"github.com/ghost-pack/hammer/internal/dagger"
)

type BuildServiceImpl struct {
	client dagger.DaggerClient
}

func NewBuildService(client dagger.DaggerClient) *BuildServiceImpl {
	return &BuildServiceImpl{client: client}
}

func (s *BuildServiceImpl) Build(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "build")
	defer span.End()
	_, err := s.client.RunCommandWithMount(
		ctx,
		"cgr.dev/chainguard/go",
		[]string{"go", "build", "."},
		"/src",
		".",
	)
	if err != nil {
		return err
	}

	return nil
}
