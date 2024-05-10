package cmd

import (
	exec2 "github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugRefreshCmd implements the ddev debug refresh command
var DebugRefreshCmd = &cobra.Command{
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 1),
	Use:               "refresh",
	Short:             "Refreshes Docker cache for project",
	Run: func(_ *cobra.Command, args []string) {
		projectName := ""

		if len(args) > 1 {
			util.Failed("This command only takes one optional argument: project name")
		}

		if len(args) == 1 {
			projectName = args[0]
		}

		_, err := dockerutil.DownloadDockerComposeIfNeeded()
		if err != nil {
			util.Failed("could not download docker-compose: %v", err)
		}
		composeBinaryPath, err := globalconfig.GetDockerComposePath()
		if err != nil {
			util.Failed("could not GetDockerComposePath(): %v", err)
		}

		app, err := ddevapp.GetActiveApp(projectName)
		if err != nil {
			util.Failed("Failed to get project: %v", err)
		}

		app.DockerEnv()
		if err = app.WriteDockerComposeYAML(); err != nil {
			util.Failed("Failed to get compose-config: %v", err)
		}

		output.UserOut.Printf("Rebuilding project images...")
		buildDurationStart := util.ElapsedDuration(time.Now())
		composeRenderedPath := app.DockerComposeFullRenderedYAMLPath()
		util.Success("Rebuilding web image with `%s -f %s build web --no-cache`", composeBinaryPath, composeRenderedPath)

		err = exec2.RunInteractiveCommand(composeBinaryPath, []string{"-f", composeRenderedPath, "build", "web", "--no-cache"})
		if err != nil {
			util.Failed("Failed to execute %s -f %s build web --no-cache: %v", composeBinaryPath, composeRenderedPath, err)
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
