package goservice

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ghost-pack/hammer/internal/docker"
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

func New(component oam.Component, dockerClient docker.DockerClient) (pipeline.Pipeline, error) {
	if component.Type != "goservice" {
		return nil, fmt.Errorf("goservice component must be of type goservice")
	}

	return &Pipeline{
		component:    &component,
		runner:       runner.New(),
		dockerClient: dockerClient,
	}, nil
}

type Pipeline struct {
	component    *oam.Component
	runner       runner.Runner
	dockerClient docker.DockerClient
}

func (p *Pipeline) ComponentType() string {
	return "goservice"
}

func (p *Pipeline) CI(ctx context.Context) error {
	ctx, span := tracing.Tracer("goservice").Start(ctx, "goservice",
		trace.WithAttributes(
			attribute.String("cmd", "goservice")))
	defer span.End()

	phases := []phase{
		{"test", p.test},
		{"build", p.build},
		{"containerize", p.containerize},
		//{"scan", p.scan},
		//{"push", p.push},
	}

	for _, ph := range phases {
		slog.InfoContext(ctx, "phase start", "phase", ph.name)
		if err := ph.run(ctx); err != nil {
			slog.ErrorContext(ctx, "phase error", "phase", ph.name, "error", err)
			return fmt.Errorf("phase %s error: %w", ph.name, err)
		}
	}
	return nil
}

type phase struct {
	name string
	run  func(context.Context) error
}
