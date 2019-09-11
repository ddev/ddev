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
	Use:   "restore-snapshot [snapshot_name]",
	Short: "Restore a project's database to the provided snapshot version.",
	Long: `Uses mariabackup command to restore a project database to a particular snapshot from the .ddev/db_snapshots folder.
Example: "ddev restore-snapshot d8git_20180717203845"`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			util.Warning("Please provide the name of the snapshot you want to restore." +
				"\nThe available snapshots are in .ddev/db_snapshots.")
			_ = cmd.Usage()
			os.Exit(1)
		}

		snapshotName := args[0]
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Failed to find active project: %v", err)
		}

		if err := app.RestoreSnapshot(snapshotName); err != nil {
			util.Failed("Failed to restore snapshot %s for project %s: %v", snapshotName, app.GetName(), err)
		}
	},
}

func init() {
	app, err := ddevapp.GetActiveApp("")
	if err == nil && app != nil && !nodeps.ArrayContainsString(app.OmitContainers, "db") {
		RootCmd.AddCommand(DdevRestoreSnapshotCommand)
	}
}
