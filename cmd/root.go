package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "build",
	Short: "Command to build... something",
	Long:  `Trying to build something heck yeah`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Hello, we are building")
	},
}

func Execute() {
	fmt.Printf("sup")
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
