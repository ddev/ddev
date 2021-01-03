package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
	"os"
)

// DdevRestoreSnapshotCommand provides the ability to revert to a database snapshot
var DdevRestoreSnapshotCommand = &cobra.Command{
	Hidden: true,
	Use:    "restore-snapshot [snapshot_name]",
	Short:  "Restore a project's database to the provided snapshot version.",
	Long:   "Please use \"snapshot restore\" command",
	Run: func(cmd *cobra.Command, args []string) {
		util.Failed("Please use \"ddev snapshot restore\".")
		os.Exit(1)
	},
}

func init() {
	app, err := ddevapp.GetActiveApp("")
	if err == nil && app != nil && !nodeps.ArrayContainsString(app.OmitContainers, "db") {
		RootCmd.AddCommand(DdevRestoreSnapshotCommand)
	}
	RootCmd.AddCommand(DdevRestoreSnapshotCommand)
}
