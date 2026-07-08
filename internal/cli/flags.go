package cli

import (
	"github.com/spf13/cobra"
)

var (
	flagOAMFile    string
	flagTenantFile string
	flagEnv        string
	flagVerbose    bool
)

func addPersistentFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&flagOAMFile, "file", "f", "oam.yaml",
		"path to oam.yaml")
	cmd.PersistentFlags().StringVarP(&flagTenantFile, "tenantFile", "t", "tenant.yaml",
		"path to tenant.yaml")
	cmd.PersistentFlags().StringVarP(&flagEnv, "env", "e", "dev",
		"deployment environment (dev/staging/prod/preview)")
	cmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false,
		"verbose output (equivalent to LOG_LEVEL=debug)")
}
