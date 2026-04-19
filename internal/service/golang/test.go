//package golang
//
//import (
//	"context"
//
//	"github.com/ghost-pack/hammer/internal/dagger"
//	"go.opentelemetry.io/otel/attribute"
//)
//
//type TestService interface {
//	Test(ctx context.Context) error
//}
//
//type buildServiceImpl struct {
//	client dagger.DaggerClient
//}
//
//func NewBuildService(client dagger.DaggerClient) BuildService {
//	return &buildServiceImpl{client: client}
//}
//
//func (s *buildServiceImpl) Build(ctx context.Context) error {
//	ctx, span := tracer.Start(ctx, "build")
//	defer span.End()
//	_, err := s.client.RunCommandWithMount(
//		ctx,
//		"alpine:latest",
//		[]string{"ls", "-la"},
//		"/src",
//		".",
//	)
//	if err != nil {
//		return err
//	}
//	rollValueAttr := attribute.Int("roll.value", 1)
//	span.SetAttributes(rollValueAttr)
//
//	return nil
//}
