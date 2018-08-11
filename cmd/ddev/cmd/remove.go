package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// For single remove only: remove db and related data
var removeData bool

// Remove all projects (but not available with -a
var removeAll bool

// create a snapshot during remove (default to false with regular remove, default to true with rm --remove-data
var createSnapshot bool

// force omission of snapshot during remove-data
var omitSnapshot bool

// DdevRemoveCmd represents the remove command
var DdevRemoveCmd = &cobra.Command{
	Use:     "remove [projectname ...]",
	Aliases: []string{"rm"},
	Short:   "Remove the development environment for a project.",
	Long: `Remove the development environment for a project. You can run 'ddev remove'
from a project directory to remove that project, or you can remove projects in
any directory by running 'ddev remove projectname [projectname ...]'.

By default, remove is a non-destructive operation and will leave database
contents intact. Remove never touches your code or files directories.

To remove database contents, you may use "ddev remove --remove-data".

To snapshot the database on remove, use "ddev remove --snapshot"; A snapshot is automatically created on
"ddev remove --remove-data" unless you use "ddev remove --remove-data --omit-snapshot".
`,
	Run: func(cmd *cobra.Command, args []string) {
		// Prevent users from destroying everything
		if removeAll && removeData {
			util.Failed("Illegal option combination: --all and --remove-data")
		}
		if createSnapshot && omitSnapshot {
			util.Failed("Illegal option combination: --snapshot and --omit-snapshot:")
		}

		apps, err := getRequestedApps(args, removeAll)
		if err != nil {
			util.Failed("Unable to get project(s): %v", err)
		}

		// Iterate through the list of apps built above, removing each one.
		for _, app := range apps {
			if app.SiteStatus() == ddevapp.SiteNotFound {
				util.Warning("Project %s is not currently running. Try 'ddev start'.", app.GetName())
			}

			// We do the snapshot if either --snapshot or --remove-data UNLESS omit-snapshot is set
			doSnapshot := ((createSnapshot || removeData) && !omitSnapshot)
			if err := app.Down(removeData, doSnapshot); err != nil {
				util.Failed("Failed to remove ddev project %s: %v", app.GetName(), err)
			}

			util.Success("Project %s has been removed.", app.GetName())
		}
	},
}

func init() {
	DdevRemoveCmd.Flags().BoolVarP(&removeData, "remove-data", "R", false, "Remove stored project data (MySQL, logs, etc.)")
	DdevRemoveCmd.Flags().BoolVarP(&createSnapshot, "snapshot", "S", false, "Create database snapshot")
	DdevRemoveCmd.Flags().BoolVarP(&omitSnapshot, "omit-snapshot", "O", false, "Omit/skip database snapshot")

	DdevRemoveCmd.Flags().BoolVarP(&removeAll, "all", "a", false, "Remove all running and stopped projects")
	RootCmd.AddCommand(DdevRemoveCmd)
}
