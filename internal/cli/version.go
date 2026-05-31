package cli

import (
	"log/slog"

	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags:
//
//	go build -ldflags "-X github.com/you/hammer/internal/cli.version=v1.2.3"
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print hammer version information",
		Run: func(cmd *cobra.Command, args []string) {
			slog.InfoContext(cmd.Context(), "hammer", "version", version)
			slog.InfoContext(cmd.Context(), "  commit:", "commit", commit)
			slog.InfoContext(cmd.Context(), "  built: ", "date", date)
		},
	}
}
