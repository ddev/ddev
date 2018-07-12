package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var snapshotName string

// DdevRevertToSnapshotCommand provides the ability to revert to a database snapshot
var DdevRevertToSnapshotCommand = &cobra.Command{
	Use:   "revert-to-snapshot projectname_HHHHMMDDHHMMSS",
	Short: "Revert a project's database to a named snapshot version.",
	Long:  `Uses mariabackup command to revert a project database to a particular snapshot from the .ddev/db_snapshots folder.`,
	Run: func(cmd *cobra.Command, args []string) {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Failed to find active project: %v", err)
		}

		if err := app.RevertToSnapshot(snapshotName); err != nil {
			util.Failed("Failed to snapshot %s: %v", app.GetName(), err)
		}
	},
}

func init() {
	DdevRevertToSnapshotCommand.Flags().StringVarP(&snapshotName, "snapshot-name", "", "", "Provide the snapshot directory name from .ddev/db_snapshots")
	RootCmd.AddCommand(DdevRevertToSnapshotCommand)
}
