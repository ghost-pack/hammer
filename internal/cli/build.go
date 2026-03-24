// internal/cli/build.go
package cli

import (
	"github.com/ghost-pack/hammer/internal/service"
	"github.com/spf13/cobra"
)

func NewBuildCmd(services *service.Services) *cobra.Command {
	return &cobra.Command{
		Use:   "build",
		Short: "Build something",
		RunE: func(cmd *cobra.Command, args []string) error {
			return services.Build.Build(cmd.Context())
		},
	}
}
