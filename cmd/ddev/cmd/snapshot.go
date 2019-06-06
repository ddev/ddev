package cmd

import (
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var snapshotAll bool
var snapshotName string

// DdevSnapshotCommand provides the snapshot command
var DdevSnapshotCommand = &cobra.Command{
	Use:   "snapshot [projectname projectname...]",
	Short: "Create a database snapshot for one or more projects.",
	Long:  `Uses mariabackup command to create a database snapshot in the .ddev/db_snapshots folder.`,
	Run: func(cmd *cobra.Command, args []string) {
		apps, err := getRequestedProjects(args, snapshotAll)
		if err != nil {
			util.Failed("Unable to get project(s) %v: %v", args, err)
		}

		for _, app := range apps {
			if snapshotNameOutput, err := app.Snapshot(snapshotName); err != nil {
				util.Failed("Failed to snapshot %s: %v", app.GetName(), err)
			} else {
				util.Success("Created snapshot %s", snapshotNameOutput)
			}
		}
	},
}

func init() {
	DdevSnapshotCommand.Flags().BoolVarP(&snapshotAll, "all", "a", false, "Snapshot all running sites")
	DdevSnapshotCommand.Flags().StringVarP(&snapshotName, "name", "n", "", "provide a name for the snapshot")
	RootCmd.AddCommand(DdevSnapshotCommand)
}
