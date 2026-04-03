package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/ghost-pack/hammer/internal/dagger"
	"github.com/ghost-pack/hammer/internal/service"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
)

const name = "github.com/ghost-pack/hammer"

var (
	tracer = otel.Tracer(name)
	//meter   = otel.Meter(name)
	//logger = otelslog.NewLogger(name)
)

var rootCmd = &cobra.Command{
	Use:   "hammer",
	Short: "Root command for hammer",
	Long:  `Use this to hammer things out.`,
}

func Execute() error {
	ctx := context.Background()
	otelShutdown, err := setupOTelSDK(ctx)
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	client, err := dagger.NewDaggerClient(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := client.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing Dagger client: %v\n", err)
		}
	}()

	svcs := service.NewServices(client)

	commands := []func(*service.Services) *cobra.Command{
		NewBuildCmd,
		//cli.NewTerraformBuildCmd,
		//cli.NewNodeBuildCmd,
		//cli.NewTrivyOnImageCmd,
	}

	for _, cmdConstructor := range commands {
		rootCmd.AddCommand(cmdConstructor(svcs))
	}
	ctx, span := tracer.Start(ctx, "hammer")
	defer span.End()

	return rootCmd.ExecuteContext(ctx)
}
