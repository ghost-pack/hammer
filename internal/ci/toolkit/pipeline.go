package toolkit

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ghost-pack/hammer/internal/ci"
	"github.com/ghost-pack/hammer/internal/docker"
	"github.com/ghost-pack/hammer/internal/gcp/project"
	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/ghost-pack/hammer/internal/runner"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	ci.Register("toolkit", New)
}

func New(component oam.Component, app oam.App, clients ci.DependencyClients) (ci.Pipeline, error) {
	if component.Type != "toolkit" {
		return nil, fmt.Errorf("toolkit component must be of type toolkit")
	}

	return &Pipeline{
		component:    &component,
		runner:       runner.New(),
		garClient:    clients.GarClient,
		dockerClient: clients.DockerClient,
	}, nil
}

type Pipeline struct {
	component    *oam.Component
	runner       runner.Runner
	garClient    project.GarClient
	dockerClient docker.Client
}

func (p *Pipeline) ComponentType() string {
	return p.component.Type
}

func (p *Pipeline) CI(ctx context.Context) (*ci.Artifact, error) {
	ctx, span := tracing.Tracer(fmt.Sprintf("%s CI", p.ComponentType())).Start(ctx, fmt.Sprintf("%s CI", p.ComponentType()),
		trace.WithAttributes(
			attribute.String("cmd", fmt.Sprintf("%s CI", p.ComponentType()))))
	defer span.End()

	var phases []phase

	phases = []phase{
		{"apkobuild", p.build},
		{"ensureGarExists", p.createGar},
		{"push", p.push},
	}

	for _, ph := range phases {
		slog.InfoContext(ctx, "phase start", "phase", ph.name)
		if err := ph.run(ctx); err != nil {
			slog.ErrorContext(ctx, "phase error", "phase", ph.name, "error", err)
			return nil, fmt.Errorf("phase %s error: %w", ph.name, err)
		}
	}
	return nil, nil
}

func (p *Pipeline) Analyze(ctx context.Context) error {
	ctx, span := tracing.Tracer(fmt.Sprintf("%s analyze", p.ComponentType())).Start(ctx, fmt.Sprintf("%s analyze", p.ComponentType()),
		trace.WithAttributes(
			attribute.String("cmd", fmt.Sprintf("%s analyze", p.ComponentType()))))
	defer span.End()

	phases := []phase{
		{"apko build", p.build},
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

func (p *Pipeline) Deploy(ctx context.Context) error {
	return nil
}

type phase struct {
	name string
	run  func(context.Context) error
}
