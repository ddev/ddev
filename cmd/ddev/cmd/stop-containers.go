package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var stopAllContainers bool

// DdevStopContainersCmd represents the stop command
var DdevStopContainersCmd = &cobra.Command{
	Use:   "stop-containers [projectname ...]",
	Short: "uses 'docker stop' to stop the containers belonging to a project.",
	Long: `Use "docker stop" to stop the containers belonging to a project a project. You can run 'ddev stop-containers'
from a project directory to stop the containers of that project, or you can stop running projects
in any directory by running 'ddev stop-containers projectname [projectname ...]'`,
	Run: func(cmd *cobra.Command, args []string) {
		projects, err := getRequestedProjects(args, stopAllContainers)
		if err != nil {
			util.Failed("Unable to get project(s): %v", err)
		}

		for _, project := range projects {
			if err := ddevapp.CheckForMissingProjectFiles(project); err != nil {
				util.Failed("Failed to stop-containers %s: %v", project.GetName(), err)
			}

			if err := project.StopContainers(); err != nil {
				util.Failed("Failed to stop-containers %s: %v", project.GetName(), err)
			}

			util.Success("Containers for project %s have been stopped.", project.GetName())
		}
	},
}

func init() {
	DdevStopContainersCmd.Flags().BoolVarP(&stopAllContainers, "all", "a", false, "Stop-containers on all running projects")
	RootCmd.AddCommand(DdevStopContainersCmd)
}
