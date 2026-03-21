package cli

import (
    "github.com/spf13/cobra"

    "github.com/ghost-pack/hammer/internal/service"
)

func NewBuildCmd(svc *service.BuildService) *cobra.Command {
    return &cobra.Command{
        Use:   "build",
        Short: "Build something",
        RunE: func(cmd *cobra.Command, args []string) error {
            return svc.Build(cmd.Context())
        },
    }
}