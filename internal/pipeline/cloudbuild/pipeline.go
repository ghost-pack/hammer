package cloudbuild

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ghost-pack/hammer/internal/docker"
	"github.com/ghost-pack/hammer/internal/gcp"
	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/ghost-pack/hammer/internal/pipeline"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	pipeline.Register("cloudbuild", New)
}

func New(component oam.Component, _ docker.Client, _ gcp.GarClient, cloudBuildClient gcp.CloudBuildClient) (pipeline.Pipeline, error) {
	if component.Type != "cloudbuild" {
		return nil, fmt.Errorf("cloudbuild component must be of type cloudbuild")
	}

	return &Pipeline{
		component:        &component,
		cloudBuildClient: cloudBuildClient,
	}, nil
}

type Pipeline struct {
	component        *oam.Component
	cloudBuildClient gcp.CloudBuildClient
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

	phases = []phase{
		{"lint", p.lint},
		{"submittest", p.submitTest},
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
		{"lint", p.lint},
		{"submittest", p.submitTest},
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
