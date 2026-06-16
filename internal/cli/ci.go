package cli

import (
	"context"
	"log/slog"
	"os"

	"github.com/ghost-pack/hammer/internal/docker"
	"github.com/ghost-pack/hammer/internal/gcp"
	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/ghost-pack/hammer/internal/pipeline"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func newCICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ci",
		Short: "Run the CI pipeline (build, scan, push) for the app in oam.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			ctx, span := tracing.Tracer("cobra").Start(ctx, "CI",
				trace.WithAttributes(
					attribute.String("cmd", "CI"),
					attribute.StringSlice("args", args)))
			defer span.End()

			slog.InfoContext(ctx, "ci start",
				"file", flagOAMFile,
				"env", flagEnv,
			)

			// Just giving this to every pipeline for now.
			dockerClient := docker.NewClient()

			garClient, err := gcp.NewGarClient(cmd.Context())
			if err != nil {
				return err
			}
			defer garClient.Close()

			cloudBuildClient, err := gcp.NewCloudBuildClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cloudBuildClient.Close()

			app, err := oam.Load(flagOAMFile)
			if err != nil {
				return err
			}

			onMain := os.Getenv("BRANCH_NAME") == "main"

			for _, component := range app.Spec.Components {
				componentPipeline, err := pipeline.For(component, dockerClient, garClient, cloudBuildClient)
				if err != nil {
					return err
				}
				var run func(context.Context) error
				if onMain {
					run = componentPipeline.CI
				} else {
					run = componentPipeline.Analyze
				}

				if err := run(ctx); err != nil {
					return err
				}

				if onMain {
					err := componentPipeline.Deploy(ctx)
					if err != nil {
						return err
					}
				}
			}

			slog.InfoContext(ctx, "ci complete")
			return nil
		},
	}
	return cmd
}
