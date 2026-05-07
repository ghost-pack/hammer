package cli

import (
	"log/slog"

	"github.com/spf13/cobra"
)

func newDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy the app in oam.yaml to the target environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			slog.InfoContext(ctx, "deploy start",
				"file", flagOAMFile,
				"env", flagEnv,
				"dry_run", flagDryRun,
			)

			// TODO: dispatch to pipeline.RunDeploy(ctx)

			slog.InfoContext(ctx, "deploy complete (placeholder)")
			return nil
		},
	}
	return cmd
}
