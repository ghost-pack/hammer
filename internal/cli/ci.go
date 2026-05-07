package cli

import (
	"log/slog"

	"github.com/ghost-pack/hammer/internal/observability/tracing"
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
			ctx, span := tracing.Tracer("runner").Start(ctx, "CI",
				trace.WithAttributes(
					attribute.String("cmd", "CI"),
					attribute.StringSlice("args", args)))
			defer span.End()

			slog.InfoContext(ctx, "ci start",
				"file", flagOAMFile,
				"env", flagEnv,
				"dry_run", flagDryRun,
			)

			// TODO: load oam.yaml, dispatch to pipeline.
			// app, err := oam.Load(flagOAMFile)
			// if err != nil { return err }
			// p, err := pipeline.For(app)
			// if err != nil { return err }
			// return p.RunCI(ctx)

			slog.InfoContext(ctx, "ci complete (placeholder)")
			return nil
		},
	}
	return cmd
}
