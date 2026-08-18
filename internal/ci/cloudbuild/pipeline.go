package cloudbuild

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/ghost-pack/hammer/internal/ci"
	"github.com/ghost-pack/hammer/internal/gcp"
	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	ci.Register("cloudbuild", New)
}

// Needs to handle overrides. But also most of these are only useful for CD?
// Like PubSubTopic, ManuallyApproved, and ServiceAccount probably could be overridden, but that's it.
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

func parseCloudBuildPath(p *Pipeline) (properties, error) {
	var props properties

	if err := p.component.Properties.Decode(&props); err != nil {
		return properties{}, fmt.Errorf("decoding properties: %w", err)
	}

	if props.Path == "" {
		props.Path = "./cloudbuild.yaml"
	}

	return props, nil
}

func New(component oam.Component, app oam.App, clients ci.DependencyClients) (ci.Pipeline, error) {
	if component.Type != "cloudbuild" {
		return nil, fmt.Errorf("cloudbuild component must be of type cloudbuild")
	}

	return &Pipeline{
		component:             &component,
		app:                   &app,
		cloudBuildClient:      clients.CloudBuild,
		pubsubClient:          clients.PubSub,
		cloudStorageClient:    clients.CloudStorage,
		releaseBucket:         "hammer-release",
		shortCommitSha:        os.Getenv("COMMIT_SHA")[:7],
		platformProject:       "hammer-central-prod",
		platformProjectNumber: "598451979611",
	}, nil
}

type Pipeline struct {
	component             *oam.Component
	app                   *oam.App
	cloudBuildClient      gcp.CloudBuildClient
	pubsubClient          gcp.PubsubClient
	cloudStorageClient    gcp.CloudStorageClient
	platformProject       string
	platformProjectNumber string
	releaseBucket         string
	shortCommitSha        string
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
		{"lint", p.lint},
		{"submittest", p.submitTest},
		{"writeOutput", p.writeOutput},
	}

	for _, ph := range phases {
		slog.InfoContext(ctx, "phase start", "phase", ph.name)
		if err := ph.run(ctx); err != nil {
			slog.ErrorContext(ctx, "phase error", "phase", ph.name, "error", err)
			return nil, fmt.Errorf("phase %s error: %w", ph.name, err)
		}
	}

	artifact := &ci.Artifact{
		Type: ci.ArtifactTypeCloudBuild,
		Properties: map[string]string{
			"cloudBuildYaml": fmt.Sprintf("gs://%s/%s/deployments/cloudbuild/%s.yaml", p.releaseBucket, p.app.Metadata.Name, p.shortCommitSha),
		},
	}
	return artifact, nil
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
