package goservice

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ghost-pack/hammer/internal/docker"
	"github.com/ghost-pack/hammer/internal/gcp/project"
	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/ghost-pack/hammer/internal/pipeline"
	"github.com/ghost-pack/hammer/internal/runner"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	pipeline.Register("goservice", New)
	pipeline.Register("gocli", New)
}

func New(component oam.Component, clients pipeline.DependencyClients) (pipeline.Pipeline, error) {
	if component.Type != "goservice" && component.Type != "gocli" {
		return nil, fmt.Errorf("goservice component must be of type goservice")
	}

	return &Pipeline{
		component:    &component,
		runner:       runner.New(),
		dockerClient: clients.DockerClient,
		garClient:    clients.GarClient,
	}, nil
}

type Pipeline struct {
	component    *oam.Component
	runner       runner.Runner
	dockerClient docker.Client
	garClient    project.GarClient
}

func (p *Pipeline) ComponentType() string {
	return p.component.Type
}

func (p *Pipeline) CI(ctx context.Context) error {
	ctx, span := tracing.Tracer(fmt.Sprintf("%s CI", p.ComponentType())).Start(ctx, fmt.Sprintf("%s CI", p.ComponentType()),
		trace.WithAttributes(
			attribute.String("cmd", fmt.Sprintf("%s CI", p.ComponentType()))))
	defer span.End()

	var phases []phase

	if p.ComponentType() == "gocli" {
		phases = []phase{
			{"test", p.test},
			{"build", p.build},
			{"containerize", p.containerize},
			{"ensureGarExists", p.createGar},
			{"push", p.push},
		}
	} else if p.ComponentType() == "goservice" {
		phases = []phase{
			{"test", p.test},
			{"build", p.build},
			{"containerize", p.containerize},
			{"ensureGarExists", p.createGar},
			{"push", p.push},
			// deploy to cloud run also
		}
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

func (p *Pipeline) Analyze(ctx context.Context) error {
	ctx, span := tracing.Tracer(fmt.Sprintf("%s analyze", p.ComponentType())).Start(ctx, fmt.Sprintf("%s analyze", p.ComponentType()),
		trace.WithAttributes(
			attribute.String("cmd", fmt.Sprintf("%s analyze", p.ComponentType()))))
	defer span.End()

	phases := []phase{
		{"test", p.test},
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

// Fill out for Goservice eventually
func (p *Pipeline) Deploy(ctx context.Context) error {
	return nil
}
