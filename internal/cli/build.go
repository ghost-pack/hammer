package cli

import (
	"dagger.io/dagger"
	"github.com/spf13/cobra"

	"github.com/ghost-pack/hammer/internal/service"
)

func NewBuildCmd(client *dagger.Client) *cobra.Command {
	svc := service.NewBuildService(client)
	return &cobra.Command{
		Use:   "build",
		Short: "Build something",
		RunE: func(cmd *cobra.Command, args []string) error {
			return svc.Build(cmd.Context())
		},
	}
}
