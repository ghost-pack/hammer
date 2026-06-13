// Package cli wires up the hammer command tree using Cobra.
package cli

import (
	"context"
	"log/slog"

	_ "github.com/ghost-pack/hammer/internal/pipeline/cloudbuild"
	_ "github.com/ghost-pack/hammer/internal/pipeline/goservice"
	_ "github.com/ghost-pack/hammer/internal/pipeline/toolkit"
	"github.com/spf13/cobra"
)

func Execute(ctx context.Context) error {
	root := newRootCmd()
	return root.ExecuteContext(ctx)
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hammer",
		Short: "hammer is the platform CLI for building, scanning, and deploying services",
		Long: `hammer takes an oam.yaml describing your application and handles the
infrastructure for building it, scanning it, and deploying it to GCP.`,
		SilenceUsage:  true, // don't print usage on every error
		SilenceErrors: true, // we log errors ourselves in main()
		// PersistentPreRunE runs before every subcommand. Useful for
		// per-command setup (e.g., loading oam.yaml).
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			slog.DebugContext(cmd.Context(), "running command",
				"command", cmd.CommandPath(),
				"args", args,
			)
			return nil
		},
	}

	// Persistent flags: available on every subcommand.
	addPersistentFlags(cmd)

	// Subcommands.
	cmd.AddCommand(
		newVersionCmd(),
		newCICmd(),
	)

	return cmd
}
