package cmd

import (
	"context"

	"github.com/ghost-pack/hammer/internal/cli"
	"github.com/ghost-pack/hammer/internal/service"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "hammer",
	Short: "Root command for hammer",
	Long:  `Use this to hammer things out.`,
}

var newDaggerClient = NewDaggerClient

func Execute() error {
	ctx := context.Background()
	client, err := newDaggerClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	// 2. Create service layer (depends on client)
	buildSvc := service.NewBuildService(client)

	// 3. Create CLI commands (depend on service)
	buildCmd := cli.NewBuildCmd(buildSvc)
	rootCmd.AddCommand(buildCmd)

	return rootCmd.Execute()
}
