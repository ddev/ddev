package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// DdevSnapshotRestoreCommand handles ddev snapshot restore
var DdevSnapshotRestoreCommand = &cobra.Command{
	Use:   "restore [snapshot_name|path]",
	Short: "Restore a project's database to the provided snapshot version.",
	Long: `Uses mariabackup or xtrabackup command to restore a project database to a particular snapshot, either by name from the .ddev/db_snapshots folder or by the path to a snapshot file elsewhere.

If the project is a git worktree, snapshots from the same project checked out in other worktrees of the same repository are also offered.`,
	Example: `ddev snapshot restore d8git_20180717203845
ddev snapshot restore --latest
ddev snapshot restore --force t3v14-latest
ddev snapshot restore $HOME/tmp/mysnapshot-mariadb_11.8.zst`,
	Run: func(_ *cobra.Command, args []string) {
		var snapshotName string

		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Failed to find active project: %v", err)
		}
		if nodeps.ArrayContainsString(app.OmitContainers, "db") {
			util.Failed("Snapshots are not available when database container is omitted")
		}

		if err = app.StartAppIfNotRunning(); err != nil {
			util.Failed("Failed to start app %s: %v", app.GetName(), err)
		}

		if snapshotRestoreLatest {
			target, err := app.GetLatestSnapshotRestoreTarget()
			if err != nil {
				util.Failed("Failed to get latest snapshot of project %s: %v", app.GetName(), err)
			}
			snapshotName = target.Path
		} else {
			if len(args) != 1 { // If the name of the snapshot isn't provided, do prompted restore
				targets, err := app.ListSnapshotRestoreTargets()
				if err != nil {
					util.Failed("Cannot list snapshots of project %s: %v", app.GetName(), err)
				}

				if len(targets) == 0 {
					util.Failed("No snapshots found for project %s", app.GetName())
				}

				labels := make([]string, 0, len(targets))
				for _, target := range targets {
					labels = append(labels, target.Label)
				}

				templates := &promptui.SelectTemplates{
					Label: "{{ . | cyan }}:",
				}

				prompt := promptui.Select{
					Label:     "Snapshot",
					Items:     labels,
					Templates: templates,
				}

				selected, _, err := prompt.Run()
				if err != nil {
					util.Failed("Prompt failed %v", err)
				}
				snapshotName = targets[selected].Path
			} else { // Snapshot name was given on command-line, use it.
				snapshotName = args[0]
			}
		}

		// Normalize the snapshot name

		if err := app.RestoreSnapshot(snapshotName, snapshotRestoreForce); err != nil {
			util.Failed("Failed to restore snapshot %s for project %s: %v", snapshotName, app.GetName(), err)
		}
	},
}

func init() {
	DdevSnapshotRestoreCommand.Flags().BoolVarP(&snapshotRestoreLatest, "latest", "", false, "use latest snapshot")
	DdevSnapshotRestoreCommand.Flags().BoolVarP(&snapshotRestoreForce, "force", "f", false, "Restore a snapshot made by a different version of the same database server, which may not come up")
	DdevSnapshotCommand.AddCommand(DdevSnapshotRestoreCommand)
}
