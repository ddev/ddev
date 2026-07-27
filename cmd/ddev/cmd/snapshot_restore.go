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
	Use:   "restore [snapshot_name]",
	Short: "Restore a project's database to the provided snapshot version.",
	Long: `Uses mariabackup command to restore a project database to a particular snapshot from the .ddev/db_snapshots folder.
If the current project is part of a git worktree, snapshots from the corresponding project paths in other worktrees are also listed.
Example: "ddev snapshot restore d8git_20180717203845"`,
	Run: func(_ *cobra.Command, args []string) {
		// snapshotName is declared in cmd/snapshot.go as a package var; do not redeclare it here

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

		var restoreTarget *ddevapp.SnapshotRestoreTarget
		if snapshotRestoreLatest {
			restoreTarget, err = app.GetLatestSnapshotRestoreTarget()
			if err != nil {
				util.Failed("Failed to get latest snapshot of project %s: %v", app.GetName(), err)
			}
		} else {
			if len(args) != 1 { // If the name of the snapshot isn't provided, do prompted restore
				targets, err := app.ListSnapshotRestoreTargets()
				if err != nil {
				util.Failed("Cannot list snapshots of project %s: %v", app.GetName(), err)
				}

				if len(targets) == 0 {
				util.Failed("No snapshots found for project %s", app.GetName())
				}

				snapshotLabels := make([]string, len(targets))
				for i, target := range targets {
				snapshotLabels[i] = target.Label
				}

				templates := &promptui.SelectTemplates{
				Label: "{{ . | cyan }}:",
				}

				prompt := promptui.Select{
				Label:     "Snapshot",
				Items:     snapshotLabels,
				Templates: templates,
				}

				selectedIndex, _, err := prompt.Run()
				if err != nil {
				util.Failed("Prompt failed %v", err)
				}
				restoreTarget = &targets[selectedIndex]
			} else { // Snapshot name was given on command-line, use it.
				targets, err := app.ListSnapshotRestoreTargets()
				if err != nil {
				util.Failed("Cannot list snapshots of project %s: %v", app.GetName(), err)
				}
				for i := range targets {
					if targets[i].Name == args[0] {
						restoreTarget = &targets[i]
						break
					}
				}
				if restoreTarget == nil || restoreTarget.Name == "" {
					restoreTarget = &ddevapp.SnapshotRestoreTarget{Name: args[0], SourceRoot: app.AppRoot, Label: args[0]}
				}
			}
		}

		if err := app.RestoreSnapshotFromWorktree(restoreTarget.Name, restoreTarget.SourceRoot); err != nil {
			util.Failed("Failed to restore snapshot %s for project %s: %v", restoreTarget.Name, app.GetName(), err)
		}
	},
}

func init() {
	DdevSnapshotRestoreCommand.Flags().BoolVarP(&snapshotRestoreLatest, "latest", "", false, "use latest snapshot")
	DdevSnapshotCommand.AddCommand(DdevSnapshotRestoreCommand)
}
