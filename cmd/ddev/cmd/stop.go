package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var stopAll bool

// DdevStopCmd represents the stop command
var DdevStopCmd = &cobra.Command{
	Use:   "stop [projectname]",
	Short: "Stop the development environment for a project.",
	Long: `Stop the development environment for a project. You can run 'ddev stop'
from a project directory to stop that project, or you can specify a running project
to stop by running 'ddev stop <projectname>.`,
	Run: func(cmd *cobra.Command, args []string) {
		apps, err := determineAppList(args, stopAll)
		if err != nil {
			util.Failed("Unable to stop project(s): %v", err)
		}

		for _, app := range apps {
			if app.SiteStatus() == ddevapp.SiteStopped {
				continue
			}

			if err := app.Stop(); err != nil {
				util.Warning("Failed to stop %s: %v", app.GetName(), err)
				continue
			}

			if app.SiteStatus() != ddevapp.SiteStopped {
				util.Warning("Failed to stop %s", app.GetName())
				continue
			}

			util.Success("Project %s has been stopped.", app.GetName())
		}
	},
}

func init() {
	DdevStopCmd.Flags().BoolVarP(&stopAll, "all", "a", false, "Stop all running sites")
	RootCmd.AddCommand(DdevStopCmd)
}
