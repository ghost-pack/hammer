package service

import (
	"context"

	"github.com/ghost-pack/hammer/internal/dagger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

const name = "github.com/ghost-pack/hammer"

var (
	tracer = otel.Tracer(name)
	//meter   = otel.Meter(name)
	//logger = otelslog.NewLogger(name)
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
