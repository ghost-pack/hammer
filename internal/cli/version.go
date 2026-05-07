package cli

import (
	"fmt"

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
			fmt.Fprintln(cmd.OutOrStdout(), "hammer", version)
			fmt.Fprintln(cmd.OutOrStdout(), "  commit:", commit)
			fmt.Fprintln(cmd.OutOrStdout(), "  built: ", date)
		},
	}
}
