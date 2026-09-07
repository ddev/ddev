package cmd

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/styles"
	"github.com/ddev/ddev/pkg/util"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

var snapshotAll bool
var snapshotCleanup bool
var snapshotList bool
var snapshotName string
var snapshotRestoreLatest bool
var snapshotUncompressed bool
var snapshotRestoreForce bool

// noConfirm: If true, --yes, we won't stop and prompt before each deletion
var snapshotCleanupNoConfirm bool

// DdevSnapshotCommand provides the snapshot command
var DdevSnapshotCommand = &cobra.Command{
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 0),
	Use:               "snapshot [projectname projectname...]",
	Short:             "Create a database snapshot for one or more projects.",
	Long:              `Uses mariadb-backup or xtrabackup command to create a database snapshot in the .ddev/db_snapshots folder. These are compatible with server backups using the same tools and can be restored with "ddev snapshot restore".`,
	Example: `ddev snapshot
ddev snapshot --name some_descriptive_name
ddev snapshot --cleanup
ddev snapshot --cleanup --name my_snapshot_name
ddev snapshot --cleanup -y
ddev snapshot --list
ddev snapshot --all
ddev snapshot --uncompressed`,
	Run: func(_ *cobra.Command, args []string) {
		apps, err := getRequestedProjects(args, snapshotAll)
		if err != nil {
			util.Failed("Unable to get project(s) %v: %v", args, err)
		}
		if len(apps) > 0 {
			instrumentationApp = apps[0]
		}

		if snapshotList {
			listSnapshots(apps)
			return
		}

		for _, app := range apps {
			switch {
			case snapshotCleanup:
				deleteAppSnapshot(app)
			default:
				createAppSnapshot(app)
			}
		}
	},
}

func listSnapshots(apps []*ddevapp.DdevApp) {
	var out bytes.Buffer
	t := table.NewWriter()
	t.SetOutputMirror(&out)
	styles.SetGlobalTableStyle(t, false)

	allTargets := make(map[string][]ddevapp.SnapshotRestoreTarget)
	hasWorktreeSnapshot := false
	for _, app := range apps {
		targets, err := app.ListSnapshotRestoreTargets()
		if err != nil {
			util.Failed("Failed to list snapshots %s: %v", app.GetName(), err)
		}
		allTargets[app.GetName()] = targets
		for _, target := range targets {
			if target.SourceRoot != app.AppRoot {
				hasWorktreeSnapshot = true
			}
		}
	}

	var columns table.Row
	if len(apps) > 1 {
		columns = append(columns, "Project")
	}
	columns = append(columns, "Snapshot", "Created", "Size", "DB Version", "Compression")
	if hasWorktreeSnapshot {
		columns = append(columns, "Worktree")
	}

	if !globalconfig.DdevGlobalConfig.SimpleFormatting {
		var colConfig []table.ColumnConfig
		for _, col := range columns {
			colConfig = append(colConfig, table.ColumnConfig{
				Name: fmt.Sprint(col),
			})
		}
		t.SetColumnConfigs(colConfig)
	}
	t.AppendHeader(columns)

	for _, app := range apps {
		targets := allTargets[app.GetName()]
		if len(targets) > 0 {
			for _, target := range targets {
				worktree := ""
				if target.SourceRoot != app.AppRoot {
					worktree = filepath.Base(target.SourceRoot)
				}
				row := table.Row{}
				if len(apps) > 1 {
					row = append(row, app.GetName())
				}
				row = append(row, target.Name, target.Created.Format("2006-01-02"), util.FormatBytes(target.Size), target.DBVersion, target.Compression)
				if hasWorktreeSnapshot {
					row = append(row, worktree)
				}
				t.AppendRow(row)
			}
		} else {
			row := table.Row{}
			if len(apps) > 1 {
				row = append(row, app.GetName())
			}
			row = append(row, text.Italic.Sprint("No snapshots"), "", "", "", "")
			if hasWorktreeSnapshot {
				row = append(row, "")
			}
			t.AppendRow(row)
		}
	}

	t.Render()
	output.UserOut.WithField("raw", allTargets).Println(out.String())
}

func createAppSnapshot(app *ddevapp.DdevApp) {
	// If the database is omitted, do not snapshot
	omittedContainers := app.GetOmittedContainers()
	if nodeps.ArrayContainsString(omittedContainers, "db") {
		util.Warning("Database is omitted for project %s, skipping snapshot", app.GetName())
		return
	}

	appStatus, _ := app.SiteStatus()
	// If the app is not running, then start it to create a snapshot.
	if appStatus != ddevapp.SiteRunning {
		util.Warning("Project %s is %s, starting it to create a snapshot", app.GetName(), appStatus)
		if err := app.Start(); err != nil {
			util.Failed("Failed to start %s: %v", app.GetName(), err)
		}
	}
	// If there is an error from Snapshot, show a warning message
	// allow the command to continue, there may be other snapshots needed
	if snapshotNameOutput, err := app.Snapshot(snapshotName, snapshotUncompressed); err != nil {
		errorMsg := util.ColorizeText("Failed to snapshot %s: %v", "red")
		util.Warning(errorMsg, app.GetName(), err)
	} else {
		util.Success("Created database snapshot %s", snapshotNameOutput)
		util.Success("Restore this snapshot with 'ddev snapshot restore %s'", snapshotNameOutput)
	}
	// Return the app to its previous state, stopped or paused.
	if appStatus == ddevapp.SiteStopped {
		if err := app.Stop(false, false); err != nil {
			util.Failed("Failed to stop %s: %v", app.GetName(), err)
		}
	}
	if appStatus == ddevapp.SitePaused {
		if err := app.Pause(); err != nil {
			util.Failed("Failed to pause %s: %v", app.GetName(), err)
		}
	}
}

func deleteAppSnapshot(app *ddevapp.DdevApp) {
	var snapshotsToDelete []string
	var prompt string
	var err error

	if !snapshotCleanupNoConfirm {
		if snapshotName == "" {
			prompt = fmt.Sprintf("OK to delete all snapshots of %s.", app.GetName())
		} else {
			prompt = fmt.Sprintf("OK to delete the snapshot '%s' of project '%s'", snapshotName, app.GetName())
		}
		if !util.Confirm(prompt) {
			return
		}
	}

	if snapshotName == "" {
		snapshotsToDelete, err = app.ListSnapshotNames()
		if err != nil {
			util.Failed("Failed to detect snapshots %s: %v", app.GetName(), err)
		}
	} else {
		snapshotsToDelete = append(snapshotsToDelete, snapshotName)
	}

	for _, snapshotToDelete := range snapshotsToDelete {
		if err := app.DeleteSnapshot(snapshotToDelete); err != nil {
			util.Failed("Failed to delete snapshot %s: %v", app.GetName(), err)
		}
	}
}

func init() {
	DdevSnapshotCommand.Flags().BoolVarP(&snapshotAll, "all", "a", false, "Snapshot all projects. Will start the project if it is stopped or paused")
	DdevSnapshotCommand.Flags().BoolVarP(&snapshotList, "list", "l", false, "List snapshots")
	DdevSnapshotCommand.Flags().BoolVarP(&snapshotCleanup, "cleanup", "C", false, "Cleanup snapshots")
	DdevSnapshotCommand.Flags().BoolVarP(&snapshotCleanupNoConfirm, "yes", "y", false, "Yes - skip confirmation prompt")
	DdevSnapshotCommand.Flags().StringVarP(&snapshotName, "name", "n", "", "provide a name for the snapshot")
	DdevSnapshotCommand.Flags().BoolVar(&snapshotUncompressed, "uncompressed", false, "Write the snapshot as an uncompressed mariadb-backup/xtrabackup stream instead of compressing it. This skips decompression on restore, but the file can be roughly as large as the database's datadir, many times bigger than a compressed snapshot. Not available for PostgreSQL projects.")
	RootCmd.AddCommand(DdevSnapshotCommand)
}
