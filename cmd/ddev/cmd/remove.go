package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var removeData bool
var removeAll bool

// DdevRemoveCmd represents the remove command
var DdevRemoveCmd = &cobra.Command{
	Use:     "remove [projectname ...]",
	Aliases: []string{"rm"},
	Short:   "Remove the development environment for a project.",
	Long: `Remove the development environment for a project. You can run 'ddev remove'
from a project directory to remove that project, or you can specify projects
to remove by running 'ddev remove [projectname ...]'. By default, remove is
a non-destructive operation and will leave database contents intact. Remove
never touches your code or files directories.

To remove database contents, you may use the --remove-data flag with remove.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Prevent users from destroying everything
		if removeAll && removeData {
			util.Failed("Illegal option combination: --all and --remove-data")
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

			if err := app.Down(removeData); err != nil {
				util.Failed("Failed to remove %s: %v", app.GetName(), err)
			}

			util.Success("Project %s has been removed.", app.GetName())
		}
	},
}

func init() {
	DdevRemoveCmd.Flags().BoolVarP(&removeData, "remove-data", "R", false, "Remove stored project data (MySQL, logs, etc.)")
	DdevRemoveCmd.Flags().BoolVarP(&removeAll, "all", "a", false, "Remove all running and stopped projects")
	RootCmd.AddCommand(DdevRemoveCmd)
}
