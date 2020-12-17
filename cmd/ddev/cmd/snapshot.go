package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
	"strings"
)

var snapshotAll bool
var snapshotList bool
var snapshotName string

// DdevSnapshotCommand provides the snapshot command
var DdevSnapshotCommand = &cobra.Command{
	Use:   "snapshot [projectname projectname...]",
	Short: "Create a database snapshot for one or more projects.",
	Long:  `Uses mariabackup or xtrabackup command to create a database snapshot in the .ddev/db_snapshots folder. These are compatible with server backups using the same tools and can be restored with "ddev restore-snapshot".`,
	Example: `ddev snapshot
ddev snapshot --name some_descriptive_name
ddev snapshot --list
ddev snapshot --all`,
	Run: func(cmd *cobra.Command, args []string) {
		apps, err := getRequestedProjects(args, snapshotAll)
		if err != nil {
			util.Failed("Unable to get project(s) %v: %v", args, err)
		}
		if len(apps) > 0 {
			instrumentationApp = apps[0]
		}

		for _, app := range apps {
			if snapshotList {
				listAppSnapshots(app)

				continue // do not create snapshot
			}

			createAppSnapshot(app)
		}
	},
}

func listAppSnapshots(app *ddevapp.DdevApp) {
	if snapshotNames, err := app.ListSnapshots(); err != nil {
		util.Failed("Failed to list snapshots %s: %v", app.GetName(), err)
	} else {
		util.Success("Snapshots of project %s: %s", app.GetName(), strings.Join(snapshotNames, ", "))
	}
}

func createAppSnapshot(app *ddevapp.DdevApp) {
	if snapshotNameOutput, err := app.Snapshot(snapshotName); err != nil {
		util.Failed("Failed to snapshot %s: %v", app.GetName(), err)
	} else {
		util.Success("Created snapshot %s", snapshotNameOutput)
	}
}

func init() {
	DdevSnapshotCommand.Flags().BoolVarP(&snapshotAll, "all", "a", false, "Snapshot all running projects")
	DdevSnapshotCommand.Flags().BoolVarP(&snapshotList, "list", "l", false, "List snapshots")
	DdevSnapshotCommand.Flags().StringVarP(&snapshotName, "name", "n", "", "provide a name for the snapshot")
	RootCmd.AddCommand(DdevSnapshotCommand)
}
