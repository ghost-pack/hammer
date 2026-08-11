package cli

import (
	"context"
	"log/slog"
	"os"

	"github.com/ghost-pack/hammer/internal/docker"
	"github.com/ghost-pack/hammer/internal/gcp"
	"github.com/ghost-pack/hammer/internal/gcp/project"
	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/ghost-pack/hammer/internal/pipeline"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type CICmd struct {
	Docker       docker.Client
	Gar          project.GarClient
	CloudBuild   gcp.CloudBuildClient
	CloudStorage gcp.CloudStorageClient
	PubSub       gcp.PubsubClient
	OAMFile      string
}

// Execute runs the CI logic. All I/O goes through the interfaces.
func (c *CICmd) Execute(ctx context.Context, args []string) error {
	// tracing setup (as before)
	ctx, span := tracing.Tracer("cobra").Start(ctx, "CI",
		trace.WithAttributes(
			attribute.String("cmd", "CI"),
			attribute.StringSlice("args", args)))
	defer span.End()

	slog.InfoContext(ctx, "ci start", "file", c.OAMFile)

	app, err := oam.Load(c.OAMFile)
	if err != nil {
		return err
	}

	onMain := os.Getenv("BRANCH_NAME") == "main"

	for _, component := range app.Spec.Components {
		componentPipeline, err := pipeline.For(component,
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
			if _, err := componentPipeline.CI(ctx); err != nil {
				return err
			}
		} else {
			if err := componentPipeline.Analyze(ctx); err != nil {
				return err
			}
		}
	}

	// Steps mentioned in the comments (only on main)
	if onMain {
		// 1. Upload OAM file to GCS
		// Use c.CloudStorage.WriteObject(...)
		// 2. Build Pubsub message from CI output
		// 3. Publish message using c.PubSub
		// (implement these later, but they'll be testable because you mock the clients)
	}

	slog.InfoContext(ctx, "ci complete")
	return nil
}

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
