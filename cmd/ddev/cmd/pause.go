package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var pauseAllProjects bool

// DdevPauseCommand represents the stop command
var DdevPauseCommand = &cobra.Command{
	Use:   "pause [projectname ...]",
	Short: "uses 'docker stop' to pause/stop the containers belonging to a project.",
	Long: `Uses "docker-compose stop" to pause/stop the containers belonging to a project. This leaves the containers instantiated instead of removing them like ddev stop does. You can run 'ddev pause'
from a project directory to stop the containers of that project, or you can stop running projects
in any directory by running 'ddev pause projectname [projectname ...]' or pause all with 'ddev pause --all'`,
	Aliases: []string{"sc", "stop-containers"},
	Run: func(cmd *cobra.Command, args []string) {
		projects, err := getRequestedProjects(args, pauseAllProjects)
		if err != nil {
			util.Failed("Unable to get project(s): %v", err)
		}
		if len(projects) > 0 {
			instrumentationApp = projects[0]
		}

		for _, project := range projects {
			if err := ddevapp.CheckForMissingProjectFiles(project); err != nil {
				util.Failed("Failed to pause/stop-containers %s: %v", project.GetName(), err)
			}

			if err := project.Pause(); err != nil {
				util.Failed("Failed to pause/stop-containers %s: %v", project.GetName(), err)
			}

			util.Success("Project %s has been paused.", project.GetName())
		}
	},
}

func init() {
	DdevPauseCommand.Flags().BoolVarP(&pauseAllProjects, "all", "a", false, "Pause all running projects")
	RootCmd.AddCommand(DdevPauseCommand)
}
