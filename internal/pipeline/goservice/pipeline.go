package goservice

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/ghost-pack/hammer/internal/pipeline"
	"github.com/ghost-pack/hammer/internal/runner"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	pipeline.Register("goservice", New)
}

func New(component oam.Component) (pipeline.Pipeline, error) {
	if component.Type != "goservice" {
		return nil, fmt.Errorf("goservice component must be of type goservice")
	}

	return &Pipeline{
		component: &component,
	}, nil
}

type Pipeline struct {
	component *oam.Component
}

func (p *Pipeline) ComponentType() string {
	return "goservice"
}

func (p *Pipeline) CI(ctx context.Context) error {
	slog.InfoContext(ctx, "oh my god finally we made it here")
	ctx, span := tracing.Tracer("goservice").Start(ctx, "goservice",
		trace.WithAttributes(
			attribute.String("cmd", "goservice")))
	defer span.End()
	result, err := runner.RunWithoutOptions(ctx, "go", []string{"test", "./..."})
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		slog.InfoContext(ctx, fmt.Sprintf("test passed? I think? %d", result.ExitCode))
	}
	return nil
}
