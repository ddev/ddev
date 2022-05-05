package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugRefreshCmd implements the ddev debug refresh command
var DebugRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refreshes docker cache for project",
	Run: func(cmd *cobra.Command, args []string) {
		projectName := ""

		if len(args) > 1 {
			util.Failed("This command only takes one optional argument: project name")
		}

		if len(args) == 1 {
			projectName = args[0]
		}

		app, err := ddevapp.GetActiveApp(projectName)
		if err != nil {
			util.Failed("Failed to get project: %v", err)
		}

		app.DockerEnv()
		if err = app.WriteDockerComposeYAML(); err != nil {
			util.Failed("Failed to get compose-config: %v", err)
		}

		util.Debug("Executing docker-compose -f %s build --no-cache", app.DockerComposeFullRenderedYAMLPath())
		_, _, err = dockerutil.ComposeCmd([]string{app.DockerComposeFullRenderedYAMLPath()}, "build", "--no-cache")

		if err != nil {
			util.Failed("Failed to execute docker-compose -f %s build --no-cache: %v", err)
		}
		util.Success("Refreshed docker cache for project %s", app.Name)
	},
}

func init() {
	DebugCmd.AddCommand(DebugRefreshCmd)
}
