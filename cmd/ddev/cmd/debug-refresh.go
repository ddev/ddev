package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
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

		status, _ := app.SiteStatus()
		if status != ddevapp.SiteRunning {
			if err = app.Start(); err != nil {
				util.Failed("Failed to start %s: %v", app.Name, err)
			}
		}

		app.DockerEnv()
		if err = app.WriteDockerComposeYAML(); err != nil {
			util.Failed("Failed to get compose-config: %v", err)
		}

		util.Debug("Executing docker-compose -f %s build --no-cache", app.DockerComposeFullRenderedYAMLPath())
		out, stderr, err := dockerutil.ComposeCmd([]string{app.DockerComposeFullRenderedYAMLPath()}, "build", "--no-cache")
		output.UserOut.Printf("docker-compose build output:\n%s\n\n", out)
		if err != nil {
			util.Failed("Failed to execute docker-compose -f %s build --no-cache: %v; stderr=\n%s\n\n", err, stderr)
		}
		err = app.Restart()
		if err != nil {
			util.Failed("Failed to restart project: %v", err)
		}

		util.Success("Refreshed docker cache for project %s", app.Name)
	},
}

func init() {
	DebugCmd.AddCommand(DebugRefreshCmd)
}
