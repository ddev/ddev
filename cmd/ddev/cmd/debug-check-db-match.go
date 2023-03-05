package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugCheckDBMatch verified that the DB Type/Version in container matches configured
var DebugCheckDBMatch = &cobra.Command{
	Use:   "check-db-match",
	Short: "Verify that the database in the ddev-db server matches the configured type/version",
	Run: func(cmd *cobra.Command, args []string) {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Can't find active project: %v", err)
		}

		dbType, err := app.GetExistingDBType()
		if err != nil {
			util.Failed("Failed to check existing DB type: %v", err)
		}
		expected := app.Database.Type + ":" + app.Database.Version
		if expected != dbType {
			util.Failed("configured database type=%s but database type in volume is %s", expected, dbType)
		}
		util.Success("database in volume matches configured database: %s", expected)
	},
}

func init() {
	DebugCmd.AddCommand(DebugCheckDBMatch)
}
