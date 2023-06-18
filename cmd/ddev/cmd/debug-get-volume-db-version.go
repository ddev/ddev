package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugGetVolumeDBVersion gets the DB Type/Version in the ddev-dbserver
var DebugGetVolumeDBVersion = &cobra.Command{
	Use:   "get-volume-db-version",
	Short: "Get the database type/version found in the ddev-dbserver's database volume, which may not be the same as the configured database type/version",
	Run: func(cmd *cobra.Command, args []string) {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Can't find active project: %v", err)
		}

		dbType, err := app.GetExistingDBType()
		if err != nil {
			util.Failed("Failed to get existing DB type/version: %v", err)
		}
		util.Success(dbType)
	},
}

func init() {
	DebugCmd.AddCommand(DebugGetVolumeDBVersion)
}
