package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/util"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// DdevSnapshotRestoreCommand handles ddev snapshot restore
var DdevSnapshotRestoreCommand = &cobra.Command{
	Use:   "restore [snapshot_name]",
	Short: "Restore a project's database to the provided snapshot version.",
	Long: `Uses mariabackup command to restore a project database to a particular snapshot from the .ddev/db_snapshots folder.
Example: "ddev snapshot restore d8git_20180717203845"`,
	Run: func(cmd *cobra.Command, args []string) {
		var snapshotName string

		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Failed to find active project: %v", err)
		}

		if snapshotRestoreLatest {
			if snapshotName, err = app.GetLatestSnapshot(); err != nil {
				util.Failed("Failed to get latest snapshot of project %s: %v", app.GetName(), err)
			}
		} else {
			if len(args) != 1 {

				snapshots, err := app.ListSnapshots()
				if err != nil {
					util.Failed("Cannot list snapshots of project %s: %v", app.GetName(), err)
				}

				if len(snapshots) == 0 {
					util.Failed("No snapshots found for project %s", app.GetName())
				}

				templates := &promptui.SelectTemplates{
					Label: "{{ . | cyan }}:",
				}

				prompt := promptui.Select{
					Label:     "Snapshot",
					Items:     snapshots,
					Templates: templates,
				}

				_, snapshotName, err = prompt.Run()

				if err != nil {
					util.Failed("Prompt failed %v", err)
				}
			} else {
				snapshotName = args[0]
			}
		}

		if err := app.RestoreSnapshot(snapshotName); err != nil {
			util.Failed("Failed to restore snapshot %s for project %s: %v", snapshotName, app.GetName(), err)
		}
	},
}

func init() {
	app, err := ddevapp.GetActiveApp("")
	if err == nil && app != nil && !nodeps.ArrayContainsString(app.OmitContainers, "db") {
		DdevSnapshotRestoreCommand.Flags().BoolVarP(&snapshotRestoreLatest, "latest", "", false, "use latest snapshot")
		DdevSnapshotCommand.AddCommand(DdevSnapshotRestoreCommand)
	}
}
