package cli

import (
	"log/slog"

	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func newTenantCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tenant",
		Short: "Run the Tenant creation pipeline for a tenant.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			ctx, span := tracing.Tracer("cobra").Start(ctx, "tenant",
				trace.WithAttributes(
					attribute.String("cmd", "tenant"),
					attribute.StringSlice("args", args)))
			defer span.End()

			slog.InfoContext(ctx, "tenant start",
				"file", flagTenantFile,
				"env", flagEnv,
			)

			// Just giving this to every pipeline for now.
			//dockerClient := docker.NewClient()
			//
			//garClient, err := gcp.NewGarClient(cmd.Context())
			//if err != nil {
			//	return err
			//}
			//defer garClient.Close()
			//
			//cloudBuildClient, err := gcp.NewCloudBuildClient(cmd.Context())
			//if err != nil {
			//	return err
			//}
			//defer cloudBuildClient.Close()
			//
			//cloudStorageClient, err := gcp.NewCloudStorageClient(cmd.Context())
			//if err != nil {
			//	return err
			//}
			//defer cloudStorageClient.Close()
			//
			//app, err := oam.Load(flagOAMFile)
			//if err != nil {
			//	return err
			//}
			//
			//onMain := os.Getenv("BRANCH_NAME") == "main"
			//
			//for _, component := range app.Spec.Components {
			//	componentPipeline, err := pipeline.For(component, pipeline.DependencyClients{DockerClient: dockerClient, GarClient: garClient, CloudBuild: cloudBuildClient, CloudStorage: cloudStorageClient})
			//	if err != nil {
			//		return err
			//	}
			//	var run func(context.Context) error
			//	if onMain {
			//		run = componentPipeline.CI
			//	} else {
			//		run = componentPipeline.Analyze
			//	}
			//
			//	if err := run(ctx); err != nil {
			//		return err
			//	}
			//
			//	if onMain {
			//		err := componentPipeline.Deploy(ctx)
			//		if err != nil {
			//			return err
			//		}
			//	}
			//}

			slog.InfoContext(ctx, "tenant flow complete")
			return nil
		},
	}
	return cmd
}
