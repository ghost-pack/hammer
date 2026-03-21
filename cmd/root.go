package cmd

import (
	"context"

	"dagger.io/dagger"
	"github.com/ghost-pack/hammer/internal/cli"
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

	commands := []func(*dagger.Client) *cobra.Command{
		cli.NewBuildCmd,
	}

	for _, cmdConstructor := range commands {
		rootCmd.AddCommand(cmdConstructor(client))
	}

	return rootCmd.Execute()
}
