package cli

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/ghost-pack/hammer/internal/provisioner"
	"github.com/ghost-pack/hammer/internal/tenant"
	"github.com/stretchr/testify/require"
)

func init() {
	provisioner.Register("TenantTest", NewProvisioner)
	provisioner.Register("TenantBad", NewBadProvisioner)
}

// The noop component is only used for testing.
func NewProvisioner(tenant *tenant.Tenant, client *provisioner.DependencyClients) (provisioner.Provisioner, error) {
	if tenant.Kind != "TenantTest" {
		return nil, fmt.Errorf("TenantTest resource must be of kind TenantTest")
	}

	return &Provisioner{
		tenant: tenant,
	}, nil
}

type Provisioner struct {
	tenant *tenant.Tenant
}

func (p *Provisioner) Apply(ctx context.Context) error {
	return nil
}

func NewBadProvisioner(tenant *tenant.Tenant, client *provisioner.DependencyClients) (provisioner.Provisioner, error) {
	if tenant.Kind != "TenantBad" {
		return nil, fmt.Errorf("TenantBad component must be of type TenantBad")
	}

	return &BadProvisioner{
		tenant: tenant,
	}, nil
}

type BadProvisioner struct {
	tenant *tenant.Tenant
}

func (p *BadProvisioner) Apply(ctx context.Context) error {
	return fmt.Errorf("bad apply")
}

func TestTenantApply(t *testing.T) {
	tests := []struct {
		name         string
		tenantFolder string
		wantErr      bool
	}{
		{
			name:         "successful apply execution",
			tenantFolder: "testdata/tenantGood",
		},
		{
			name:         "failed apply bad kind",
			tenantFolder: "testdata/tenantBad",
			wantErr:      true,
		},
		{
			name:         "failed apply unparseable",
			tenantFolder: "testdata/tenantUnparseable",
			wantErr:      true,
		},
		{
			name:         "failed apply non-existent kind",
			tenantFolder: "testdata/tenantNonexistentKind",
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("BRANCH_NAME", "main")
			rootCmd := newRootCmd()

			rootCmd.SetArgs([]string{"tenant", "--tenantFolder", tt.tenantFolder})

			err := rootCmd.Execute()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTenantMiscellaneous(t *testing.T) {
	t.Run("short circuit when not on main", func(t *testing.T) {
		os.Setenv("BRANCH_NAME", "whatever")
		rootCmd := newRootCmd()

		rootCmd.SetArgs([]string{"tenant", "--tenantFolder", "testdata/tenantGood"})

		err := rootCmd.Execute()
		require.NoError(t, err)
	})

	t.Run("error with bad tenant folder", func(t *testing.T) {
		os.Setenv("BRANCH_NAME", "main")
		rootCmd := newRootCmd()

		rootCmd.SetArgs([]string{"tenant", "--tenantFolder", "testdata/tenantGooasdfasdfasdfasdfd"})

		err := rootCmd.Execute()
		require.Error(t, err)
	})
}
