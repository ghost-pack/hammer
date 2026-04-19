package golang

import (
	"context"

	"github.com/ghost-pack/hammer/internal/dagger"
	"go.opentelemetry.io/otel/codes"
)

type TestServiceImpl struct {
	client dagger.DaggerClient
}

func NewTestService(client dagger.DaggerClient) *TestServiceImpl {
	return &TestServiceImpl{client: client}
}

func (s *TestServiceImpl) Test(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "test")
	defer span.End()
	_, err := s.client.RunCommandWithMount(
		ctx,
		"cgr.dev/chainguard/go",
		[]string{"go", "test", "./..."},
		"/src",
		".",
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "Tests passed.")
	return nil
}
