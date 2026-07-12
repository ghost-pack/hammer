package cli

import (
	"github.com/spf13/cobra"
)

var (
	flagOAMFile      string
	flagTenantFolder string
	flagEnv          string
	flagVerbose      bool
)

func addPersistentFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&flagOAMFile, "file", "f", "oam.yaml",
		"path to oam.yaml")
	cmd.PersistentFlags().StringVarP(&flagTenantFolder, "tenantFolder", "t", "tenants",
		"path to tenant folder")
	cmd.PersistentFlags().StringVarP(&flagEnv, "env", "e", "dev",
		"deployment environment (dev/staging/prod/preview)")
	cmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false,
		"verbose output (equivalent to LOG_LEVEL=debug)")
}
