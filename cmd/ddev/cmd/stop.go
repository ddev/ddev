package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var stopAll bool

// DdevStopCmd represents the stop command
var DdevStopCmd = &cobra.Command{
	Use:   "stop [projectname ...]",
	Short: "Stop the development environment for a project.",
	Long: `Stop the development environment for a project. You can run 'ddev stop'
from a project directory to stop that project, or you can stop running projects
in any directory by running 'ddev stop projectname [projectname ...]'`,
	Run: func(cmd *cobra.Command, args []string) {
		projects, err := getRequestedProjects(args, stopAll)
		if err != nil {
			util.Failed("Unable to get project(s): %v", err)
		}

		for _, project := range projects {
			if err := ddevapp.CheckForMissingProjectFiles(project); err != nil {
				util.Failed("Failed to stop %s: %v", project.GetName(), err)
			}

			if err := project.Stop(); err != nil {
				util.Failed("Failed to stop %s: %v", project.GetName(), err)
			}

			util.Success("Project %s has been stopped.", project.GetName())
		}
	},
}

func init() {
	DdevStopCmd.Flags().BoolVarP(&stopAll, "all", "a", false, "Stop all running projects")
	RootCmd.AddCommand(DdevStopCmd)
}
