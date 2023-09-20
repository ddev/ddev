package cmd

import (
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugRefreshCmd implements the ddev debug refresh command
var DebugRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refreshes Docker cache for project",
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

		output.UserOut.Printf("Rebuilding project images... This can take some time.")
		buildDurationStart := util.ElapsedDuration(time.Now())
		util.Debug("Executing docker-compose -f %s build --no-cache", app.DockerComposeFullRenderedYAMLPath())
		out, stderr, err := dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
			ComposeFiles: []string{app.DockerComposeFullRenderedYAMLPath()},
			Action:       []string{"build", "--no-cache"},
			Progress:     true,
		})
		output.UserOut.Printf("docker-compose build output:\n%s\n\n", out)
		if err != nil {
			util.Failed("Failed to execute docker-compose -f %s build --no-cache: %v; stderr=\n%s\n\n", err, stderr)
		}
		buildDuration := util.FormatDuration(buildDurationStart())
		util.Success("Refreshed Docker cache for project %s in %s", app.Name, buildDuration)

		err = app.Restart()
		if err != nil {
			util.Failed("Failed to restart project: %v", err)
		}
	},
}

func init() {
	DebugCmd.AddCommand(DebugRefreshCmd)
}
