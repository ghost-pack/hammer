package opentofu

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ghost-pack/hammer/internal/gcp"
	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/ghost-pack/hammer/internal/pipeline"
	"github.com/ghost-pack/hammer/internal/runner"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	pipeline.Register("opentofu", New)
}

func New(component oam.Component, client pipeline.DependencyClients) (pipeline.Pipeline, error) {
	if component.Type != "opentofu" {
		return nil, fmt.Errorf("opentofu component must be of type opentofu")
	}

	return &Pipeline{
		component:          &component,
		runner:             runner.New(),
		cloudStorageClient: client.CloudStorage,
	}, nil
}

type Pipeline struct {
	component          *oam.Component
	runner             runner.Runner
	cloudStorageClient gcp.CloudStorageClient
}

type Properties struct {
	Path string `yaml:"path"`
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

	//phases = []phase{
	//	{"ensureBucketExists", p.ensureBucketExists},
	//	{"format", p.format},
	//	{"init", p.init},
	//	{"validate", p.validate},
	//	{"tflint", p.tflint},
	//	{"checkov", p.checkov},
	//	{"plan", p.plan},
	//}

	for _, env := range []string{"dev", "prod"} {
		for _, ph := range phases {
			slog.InfoContext(ctx, "phase start", "phase", ph.name)
			if err := ph.run(ctx, env); err != nil {
				slog.ErrorContext(ctx, "phase error", "phase", ph.name, "error", err)
				return fmt.Errorf("phase %s error: %w", ph.name, err)
			}
		}
	}

	return nil
}

func (p *Pipeline) Analyze(ctx context.Context) error {
	//ctx, span := tracing.Tracer(fmt.Sprintf("%s analyze", p.ComponentType())).Start(ctx, fmt.Sprintf("%s analyze", p.ComponentType()),
	//	trace.WithAttributes(
	//		attribute.String("cmd", fmt.Sprintf("%s analyze", p.ComponentType()))))
	//defer span.End()
	//
	//phases := []phase{
	//	{"lint", p.lint},
	//	{"submittest", p.submitTest},
	//}
	//
	//for _, ph := range phases {
	//	slog.InfoContext(ctx, "phase start", "phase", ph.name)
	//	if err := ph.run(ctx); err != nil {
	//		slog.ErrorContext(ctx, "phase error", "phase", ph.name, "error", err)
	//		return fmt.Errorf("phase %s error: %w", ph.name, err)
	//	}
	//}
	return nil
}

func (p *Pipeline) Deploy(ctx context.Context) error {
	//ctx, span := tracing.Tracer(fmt.Sprintf("%s Deploy", p.ComponentType())).Start(ctx, fmt.Sprintf("%s Deploy", p.ComponentType()),
	//	trace.WithAttributes(
	//		attribute.String("cmd", fmt.Sprintf("%s Deploy", p.ComponentType()))))
	//defer span.End()
	//if p.cioutput == "" {
	//	err := fmt.Errorf("no ci output")
	//	span.RecordError(err)
	//	span.SetStatus(otelCodes.Error, "no ci output")
	//	return err
	//}
	//
	//var phases []phase
	//
	//phases = []phase{
	//	{"createOrUpdateTrigger", p.createOrUpdateTrigger},
	//}
	//
	//for _, ph := range phases {
	//	slog.InfoContext(ctx, "phase start", "phase", ph.name)
	//	if err := ph.run(ctx); err != nil {
	//		slog.ErrorContext(ctx, "phase error", "phase", ph.name, "error", err)
	//		return fmt.Errorf("phase %s error: %w", ph.name, err)
	//	}
	//}
	return nil
}

type phase struct {
	name string
	run  func(context.Context, string) error
}
