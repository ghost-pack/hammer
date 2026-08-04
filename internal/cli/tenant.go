package cli

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ghost-pack/hammer/internal/gcp"
	"github.com/ghost-pack/hammer/internal/gcp/project"
	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/ghost-pack/hammer/internal/provisioner"
	"github.com/ghost-pack/hammer/internal/tenant"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func newTenantCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tenant",
		Short: "Run the Tenant creation pipeline for each tenant in tenant folder",
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

			billingClient, err := project.NewBillingClient(cmd.Context())
			if err != nil {
				return err
			}
			defer billingClient.Close()

			iamClient, err := project.NewIAMClient(cmd.Context())
			if err != nil {
				return err
			}
			defer iamClient.Close()

			orgPolicy, err := project.NewOrgPolicyClient(cmd.Context())
			if err != nil {
				return err
			}
			defer orgPolicy.Close()

			resourceManager, err := project.NewResourceManagerClient(cmd.Context())
			if err != nil {
				return err
			}
			defer resourceManager.Close()

			serviceUsage, err := project.NewServiceUsageClient(cmd.Context())
			if err != nil {
				return err
			}
			defer serviceUsage.Close()

			cloudStorageClient, err := gcp.NewCloudStorageClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cloudStorageClient.Close()

			clients := provisioner.DependencyClients{
				ResourceManager: resourceManager,
				ServiceUsage:    serviceUsage,
				CloudStorage:    cloudStorageClient,
				Billing:         billingClient,
				IAM:             iamClient,
				OrgPolicy:       orgPolicy,
			}

			for _, entry := range entries {
				path := filepath.Join(flagTenantFolder, entry.Name())
				ten, err := tenant.Load(path)
				if err != nil {
					return err
				}
				tenantProvisioner, err := provisioner.For(ten, &clients)
				if err != nil {
					return err
				}

				slog.InfoContext(ctx, fmt.Sprintf("applying tenant %s", ten.Metadata.Name))
				err = tenantProvisioner.Apply(ctx)
				if err != nil {
					return err
				}
			}

			slog.InfoContext(ctx, "tenant flow complete")
			return nil
		},
	}
	return cmd
}
