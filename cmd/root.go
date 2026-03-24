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

func Execute() error {
	ctx := context.Background()
	client, err := NewDaggerClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	svcs := service.NewServices(client)

	// Slice of constructors that accept *Services
	commands := []func(*service.Services) *cobra.Command{
		cli.NewBuildCmd,
		//cli.NewTerraformBuildCmd,
		//cli.NewNodeBuildCmd,
		//cli.NewTrivyOnImageCmd,
	}

	for _, cmdConstructor := range commands {
		rootCmd.AddCommand(cmdConstructor(svcs))
	}

	return rootCmd.Execute()
}
