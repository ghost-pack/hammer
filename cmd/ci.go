// internal/cli/build_service.go
package cmd

import (
	"github.com/ghost-pack/hammer/internal/service"
	"github.com/spf13/cobra"
)

func NewCICommand(services *service.Services) *cobra.Command {
	return &cobra.Command{
		Use:   "ci",
		Short: "Continuous Integration Commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := services.Test.Test(cmd.Context())
			if err != nil {
				return err
			}
			err = services.Build.Build(cmd.Context())
			if err != nil {
				return err
			}
			return nil
		},
	}
}
