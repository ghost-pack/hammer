package opentofu

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ghost-pack/hammer/internal/ci"
	"github.com/ghost-pack/hammer/internal/gcp"
	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/ghost-pack/hammer/internal/runner"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	ci.Register("opentofu", New)
}

func New(component oam.Component, app oam.App, client ci.DependencyClients) (ci.Pipeline, error) {
	if component.Type != "opentofu" {
		return nil, fmt.Errorf("opentofu component must be of type opentofu")
	}

	return &Pipeline{
		component:          &component,
		runner:             runner.New(),
		cloudStorageClient: client.CloudStorage,
	}, nil
}

type properties struct {
	Path string `yaml:"path"`
}
type Pipeline struct {
	component          *oam.Component
	app                *oam.App
	runner             runner.Runner
	cloudStorageClient gcp.CloudStorageClient
}

type Properties struct {
	Path string `yaml:"path"`
}

func parseOpenTofuPath(p *Pipeline, properties *properties) error {
	if err := p.component.Properties.Decode(&properties); err != nil {
		return fmt.Errorf("decoding properties: %w", err)
	}
	if properties.Path == "" {
		properties.Path = "opentofu"
	}
	return nil
}

func (p *Pipeline) ComponentType() string {
	return p.component.Type
}

func (p *Pipeline) CI(ctx context.Context) (*ci.Artifact, error) {
	ctx, span := tracing.Tracer(fmt.Sprintf("%s CI", p.ComponentType())).Start(ctx, fmt.Sprintf("%s CI", p.ComponentType()),
		trace.WithAttributes(
			attribute.String("cmd", fmt.Sprintf("%s CI", p.ComponentType()))))
	defer span.End()

	// Before doing anything:
	// Enumerate environments in Terraform and in the tenant. Diff them, use them to create "supported environments" array.
	supportedEnvironments, err := p.getSupportedEnvironments(ctx)
	if err != nil {
		return nil, err
	}
	if len(supportedEnvironments) == 0 {
		return nil, fmt.Errorf("no supported environments found")
	}
	// Next, try to find tfstate in each environment. Save off next to env. I guess env thing can be a map.

	// Before looping through each environment, do:
	// 1. tofu fmt -recursive -check.
	// 2. tflint
	// 3. checkov

	var phases []phase

	// For each support environment,
	// 1. tofu init
	// 2. tofu validate
	// 3. tofu plan

	// If all of those phases pass,
	// 1. Tar the entire opentofu directory.
	// 2. Upload to cloud storage. Artifact should look like this:
	//"type": "opentofu",
	//"properties": {
	//	"packageUri": "gs://hammer-release/acme-corp/deployments/opentofu/1234567.tar.gz",
	//	"supportedEnvironments": ["dev", "prod"]
	//}

	for _, env := range []string{"dev", "prod"} {
		for _, ph := range phases {
			slog.InfoContext(ctx, "phase start", "phase", ph.name)
			if err := ph.run(ctx, env); err != nil {
				slog.ErrorContext(ctx, "phase error", "phase", ph.name, "error", err)
				return nil, fmt.Errorf("phase %s error: %w", ph.name, err)
			}
		}
	}

	return nil, nil
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

type phase struct {
	name string
	run  func(context.Context, string) error
}
