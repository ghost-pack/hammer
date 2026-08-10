package cloudbuild

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ghost-pack/hammer/internal/gcp"
	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/ghost-pack/hammer/internal/pipeline"
	"go.opentelemetry.io/otel/attribute"
	otelCodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	pipeline.Register("cloudbuild", New)
}

type properties struct {
	Path             string       `yaml:"path"`
	TriggerType      string       `yaml:"trigger_type"`
	ManuallyApproved bool         `yaml:"manually_approved"`
	PubSubTopic      string       `yaml:"pubsub_topic"`
	ServiceAccount   string       `yaml:"service_account"`
	Tests            []testConfig `yaml:"tests"`
}
type testConfig struct {
	Path     string `yaml:"path"`
	Required *bool  `yaml:"required"` // use pointer to detect if field was explicitly set
}

func New(component oam.Component, clients pipeline.DependencyClients) (pipeline.Pipeline, error) {
	if component.Type != "cloudbuild" {
		return nil, fmt.Errorf("cloudbuild component must be of type cloudbuild")
	}

	return &Pipeline{
		component:             &component,
		cloudBuildClient:      clients.CloudBuild,
		pubsubClient:          clients.PubSub,
		platformProject:       "hammer-central-prod",
		platformProjectNumber: "598451979611",
	}, nil
}

type Pipeline struct {
	component             *oam.Component
	cloudBuildClient      gcp.CloudBuildClient
	pubsubClient          gcp.PubsubClient
	platformProject       string
	platformProjectNumber string
	cioutput              string
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

	var properties properties
	err := parseCloudBuildPath(p, &properties)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return err
	}

	p.cioutput = properties.Path
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

func (p *Pipeline) Deploy(ctx context.Context) error {
	ctx, span := tracing.Tracer(fmt.Sprintf("%s Deploy", p.ComponentType())).Start(ctx, fmt.Sprintf("%s Deploy", p.ComponentType()),
		trace.WithAttributes(
			attribute.String("cmd", fmt.Sprintf("%s Deploy", p.ComponentType()))))
	defer span.End()
	if p.cioutput == "" {
		err := fmt.Errorf("no ci output")
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, "no ci output")
		return err
	}

	var phases []phase

	phases = []phase{
		{"createOrUpdateTrigger", p.createOrUpdateTrigger},
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
