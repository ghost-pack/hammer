package golang

import (
	"context"

	"github.com/ghost-pack/hammer/internal/dagger"
	"github.com/ghost-pack/hammer/internal/service/observability"
	"go.opentelemetry.io/otel/attribute"
)

var (
	tracer = observability.Tracer("golang")
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
		"alpine:latest",
		[]string{"ls", "-la"},
		"/src",
		".",
	)
	if err != nil {
		return err
	}
	rollValueAttr := attribute.Int("roll.value", 1)
	span.SetAttributes(rollValueAttr)

	return nil
}
