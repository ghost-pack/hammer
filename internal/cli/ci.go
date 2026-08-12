package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/ghost-pack/hammer/internal/docker"
	"github.com/ghost-pack/hammer/internal/gcp"
	"github.com/ghost-pack/hammer/internal/gcp/project"
	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/ghost-pack/hammer/internal/pipeline"
	"github.com/ghost-pack/hammer/internal/tenant"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/yaml.v3"
)

func newCICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ci",
		Short: "Run the CI pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			ci := &CICmd{OAMFile: flagOAMFile}
			// Create real clients inside RunE (they need the request context).
			// Alternatively, you could pass the context to a factory.
			dockerClient := docker.NewClient() // likely already an interface?
			garClient, err := project.NewGarClient(cmd.Context())
			if err != nil {
				return err
			}
			defer garClient.Close()

			cloudBuildClient, err := gcp.NewCloudBuildClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cloudBuildClient.Close()

			cloudStorageClient, err := gcp.NewCloudStorageClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cloudStorageClient.Close()

			pubsubClient, err := gcp.NewPubsubClient(cmd.Context())
			if err != nil {
				return err
			}
			defer pubsubClient.Close()

			ci.Docker = dockerClient
			ci.Gar = garClient
			ci.CloudBuild = cloudBuildClient
			ci.CloudStorage = cloudStorageClient
			ci.PubSub = pubsubClient

			return ci.Execute(cmd.Context(), args)
		},
	}
	return cmd
}

type CICmd struct {
	Docker       docker.Client
	Gar          project.GarClient
	CloudBuild   gcp.CloudBuildClient
	CloudStorage gcp.CloudStorageClient
	PubSub       gcp.PubsubClient
	OAMFile      string
}

func (c *CICmd) Execute(ctx context.Context, args []string) (err error) {
	ctx, span := tracing.Tracer("cobra").Start(ctx, "CI",
		trace.WithAttributes(
			attribute.String("cmd", "CI"),
			attribute.StringSlice("args", args)))
	defer span.End()
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "CI complete")
		}
		span.End()
	}()

	slog.InfoContext(ctx, "ci start", "file", c.OAMFile)

	app, err := oam.Load(c.OAMFile)
	if err != nil {
		return err
	}

	err = c.checkTenantValidity(ctx, app)
	if err != nil {
		return err
	}

	err = c.ensureBucketExistence(ctx)
	if err != nil {
		return err
	}

	onMain := os.Getenv("BRANCH_NAME") == "main"

	artifacts := make(map[string]pipeline.Artifact)
	for _, component := range app.Spec.Components {
		componentPipeline, err := pipeline.For(
			component,
			*app,
			pipeline.DependencyClients{
				DockerClient: c.Docker,
				GarClient:    c.Gar,
				CloudBuild:   c.CloudBuild,
				CloudStorage: c.CloudStorage,
				PubSub:       c.PubSub,
			})
		if err != nil {
			return err
		}
		if onMain {
			artifact, err := componentPipeline.CI(ctx)

			if err != nil {
				return err
			}

			if artifact != nil {
				artifacts[component.Name] = *artifact
			}
		} else {
			if err := componentPipeline.Analyze(ctx); err != nil {
				return err
			}
		}
	}

	if onMain {
		if len(artifacts) == 0 {
			slog.InfoContext(ctx, "ci complete - no artifacts uploaded")
			return nil
		}

		if app.Metadata.Annotations["platform.hammerplatform.dev/deployment-strategy"] == "per-component" {
			err = c.triggerPerComponentCdPipeline(ctx, app, artifacts)
		} else {
			err = c.triggerUnifiedCdPipeline(ctx, app, artifacts)
		}
		if err != nil {
			return err
		}
	}

	slog.InfoContext(ctx, "ci complete")
	return nil
}

func (c *CICmd) triggerPerComponentCdPipeline(ctx context.Context, app *oam.App, artifacts map[string]pipeline.Artifact) error {
	return c.triggerCdPipeline(ctx, app, artifacts, true)
}

func (c *CICmd) triggerUnifiedCdPipeline(ctx context.Context, app *oam.App, artifacts map[string]pipeline.Artifact) error {
	return c.triggerCdPipeline(ctx, app, artifacts, false)
}

func (c *CICmd) triggerCdPipeline(ctx context.Context, app *oam.App, artifacts map[string]pipeline.Artifact, perComponent bool) error {
	oamPath, err := c.uploadOamFile(ctx, app)
	if err != nil {
		return err
	}

	routingSlip, err := c.createRoutingSlip(ctx, app)
	if err != nil {
		return err
	}

	topicLocation, routingSlip := routingSlip[0], routingSlip[1:]

	mainMsg := buildCdPubSubMessage(ctx, app, oamPath, artifacts, routingSlip, true)
	if err := c.pushToPubSub(ctx, topicLocation, mainMsg); err != nil {
		return err
	}

	if perComponent {
		for componentName, artifact := range artifacts {
			componentArtifacts := map[string]pipeline.Artifact{componentName: artifact}
			componentMsg := buildCdPubSubMessage(ctx, app, oamPath, componentArtifacts, routingSlip, false)
			if err := c.pushToPubSub(ctx, topicLocation, componentMsg); err != nil {
				return err
			}
		}
	}

	return nil
}

func buildCdPubSubMessage(ctx context.Context, app *oam.App, oamPath string, artifacts map[string]pipeline.Artifact, routingSlip []pipeline.RoutingSlipEntry, reconcile bool) *pipeline.CIPubSubMessage {
	// Create a carrier to hold the W3C trace headers
	carrier := propagation.MapCarrier{}

	// Inject the current span's context into the carrier
	propagator := propagation.TraceContext{}
	propagator.Inject(ctx, carrier)

	return &pipeline.CIPubSubMessage{
		Tenant:      app.Metadata.Name,
		CommitSha:   os.Getenv("COMMIT_SHA"),
		Branch:      "main",
		PublishedAt: time.Now(),
		OAMPath:     oamPath,
		Artifacts:   artifacts,
		Reconcile:   reconcile,
		RoutingSlip: routingSlip,
		Traceparent: carrier.Get("traceparent"),
	}
}
func (c *CICmd) uploadOamFile(ctx context.Context, app *oam.App) (string, error) {
	// Going to put file in a place like this:
	//"oamPath": "gs://hammer-release/tenant/deployments/oam/a1b2c3d.yaml",
	specData, err := yaml.Marshal(app)
	if err != nil {
		return "", fmt.Errorf("marshalling oam file: %w", err)
	}

	oamPath := fmt.Sprintf("%s/deployments/oam/%s.yaml", app.Metadata.Name, string([]rune(os.Getenv("COMMIT_SHA"))[:7]))
	if err := c.CloudStorage.WriteObject(
		ctx,
		"hammer-release",
		oamPath,
		specData,
		nil,
	); err != nil {
		return "", fmt.Errorf("writing oam file to release bucket: %w", err)
	}

	return "gs://hammer-release/" + oamPath, nil
}

func (c *CICmd) ensureBucketExistence(ctx context.Context) error {
	err := c.CloudStorage.EnsureBucketExists(
		ctx,
		"hammer-central-prod",
		"us-central1",
		"hammer-release",
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *CICmd) createRoutingSlip(ctx context.Context, app *oam.App) ([]pipeline.RoutingSlipEntry, error) {
	tenantBytes, err := c.CloudStorage.GetObject(
		ctx,
		"hammer-registry",
		fmt.Sprintf("tenants/%s/spec.yaml", app.Metadata.Name),
	)
	if err != nil {
		return nil, err
	}

	var currentTenant *tenant.Tenant
	if err := yaml.Unmarshal(tenantBytes, &currentTenant); err != nil {
		return nil, fmt.Errorf("unmarshalling tenant state: %w", err)
	}

	var routingSlip []pipeline.RoutingSlipEntry
	for _, env := range currentTenant.Spec.Environments {
		routingSlip = append(routingSlip, pipeline.RoutingSlipEntry{
			Env:         env,
			PubSubTopic: "NOT SURE YET",
		})
	}
	return routingSlip, nil
}

func (c *CICmd) pushToPubSub(ctx context.Context, entry pipeline.RoutingSlipEntry, pubSubMessage *pipeline.CIPubSubMessage) error {
	jsonBytes, err := json.Marshal(pubSubMessage)
	if err != nil {
		return err
	}

	_, err = c.PubSub.PublishMessage(
		ctx,
		"hammer-central-prod",
		entry.PubSubTopic,
		jsonBytes,
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *CICmd) checkTenantValidity(ctx context.Context, app *oam.App) error {
	tenantPrefixes, err := c.CloudStorage.ListPrefixes(ctx, "hammer-registry", "tenants/", "/")
	// tenantPrefixes: ["tenants/hammer-central/", "tenants/tenant-2/", ...]
	if err != nil {
		return err
	}

	existingTenants := make(map[string]bool)
	for _, p := range tenantPrefixes {
		name := extractTenantName(p, "tenants/")
		existingTenants[name] = true
	}

	// Now check the OAM component's metadata.name:
	if !existingTenants[app.Metadata.Name] {
		return fmt.Errorf("tenant %q not found in registry", app.Metadata.Name)
	}
	return nil
}

func extractTenantName(fullPrefix, basePrefix string) string {
	// Remove the leading base prefix, e.g., "tenants/"
	trimmed := strings.TrimPrefix(fullPrefix, basePrefix)
	// Remove the trailing slash
	return strings.TrimSuffix(trimmed, "/")
}
