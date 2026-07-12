package cli

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/ghost-pack/hammer/internal/tenant"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func newTenantCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tenant",
		Short: "Run the Tenant creation pipeline for a tenant.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			ctx, span := tracing.Tracer("cobra").Start(ctx, "tenant",
				trace.WithAttributes(
					attribute.String("cmd", "tenant"),
					attribute.StringSlice("args", args)))
			defer span.End()

			slog.InfoContext(ctx, "tenant start",
				"folder", flagTenantFolder,
			)

			onMain := os.Getenv("BRANCH_NAME") == "main"
			if !onMain {
				slog.InfoContext(ctx, "tenant flow short-circuited as we are not on main")
				return nil
			}

			entries, err := os.ReadDir(flagTenantFolder)
			if err != nil {
				return fmt.Errorf("reading tenants dir: %w", err)
			}

			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				if filepath.Ext(entry.Name()) != ".yaml" {
					continue
				}

				path := filepath.Join("tenants", entry.Name())
				_, err := tenant.Load(path)
				if err != nil {
					return err
				}
				// interpret tenant.
			}

			slog.InfoContext(ctx, "tenant flow complete")
			return nil
		},
	}
	return cmd
}
