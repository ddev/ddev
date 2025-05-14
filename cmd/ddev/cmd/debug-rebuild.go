package cmd

import (
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	exec2 "github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var (
	buildAll bool
	service  string
)

// DebugRebuildCmd implements the ddev debug rebuild command
var DebugRebuildCmd = &cobra.Command{
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 1),
	Use:               "rebuild",
	Short:             "Rebuilds the project's Docker cache with verbose output and restarts the project or the specified service.",
	Aliases:           []string{"refresh"},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			util.Failed("This command only takes one optional argument: project name")
		}

		projectName := ""
		if len(args) == 1 {
			projectName = args[0]
		}

		if cmd.Flags().Changed("all") && cmd.Flags().Changed("service") {
			util.Failed("--all flag cannot be used with --service flag")
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

		buildDurationStart := util.ElapsedDuration(time.Now())
		composeRenderedPath := app.DockerComposeFullRenderedYAMLPath()
		withoutCache := !cmd.Flags().Changed("cache")

		buildArgs := []string{"-f", composeRenderedPath, "--progress", "plain", "build"}

		if !buildAll {
			buildArgs = append(buildArgs, service)
		}

		if withoutCache {
			buildArgs = append(buildArgs, "--no-cache")
			output.UserOut.Printf("Rebuilding project images without Docker cache...")
		} else {
			output.UserOut.Printf("Rebuilding project images using Docker cache...")
		}

		output.UserOut.Printf("Executing `%s %v`", composeBinaryPath, prettyCmd(buildArgs))
		err = exec2.RunInteractiveCommand(composeBinaryPath, buildArgs)
		if err != nil {
			util.Failed("Failed to execute `%s %v`: %v", composeBinaryPath, prettyCmd(buildArgs), err)
		}

		buildDuration := util.FormatDuration(buildDurationStart())
		if buildAll {
			util.Success("Rebuilt %s cache in %s", app.Name, buildDuration)
		} else {
			util.Success("Rebuilt %s service cache for %s in %s", service, app.Name, buildDuration)
		}

		// Restart the entire project only when changing "web",
		// since app.Start() includes a lot of extra logic
		// and just restarting the web service isn't enough here
		if buildAll || service == "web" {
			err = app.Restart()
			if err != nil {
				util.Failed("Failed to restart project: %v", err)
			}
			util.Success("Restarted %s", app.GetName())
			return
		}

		labels := map[string]string{
			"com.ddev.site-name":         app.GetName(),
			"com.docker.compose.oneoff":  "False",
			"com.docker.compose.service": service,
		}

		// Restart the specified service using docker-compose, if it is running
		if container, err := dockerutil.FindContainerByLabels(labels); err == nil && container != nil {
			restartArgs := []string{"-f", composeRenderedPath, "--progress", "plain", "restart", service}
			output.UserOut.Printf("Executing `%s %v`", composeBinaryPath, prettyCmd(restartArgs))
			err = exec2.RunInteractiveCommand(composeBinaryPath, restartArgs)
			if err != nil {
				util.Failed("Failed to execute `%s %v`: %v", composeBinaryPath, prettyCmd(restartArgs), err)
			}
			output.UserOut.Printf("Waiting %ds for project container [%v] to become ready...", app.GetMaxContainerWaitTime(), service)
			err = app.WaitByLabels(labels)
			if err != nil {
				util.Failed("Failed to wait for project container [%v] to become ready: %v", service, err)
			}
			if !ddevapp.IsRouterDisabled(app) {
				util.Debug("Starting %s if necessary...", nodeps.RouterContainer)
				err = ddevapp.StartDdevRouter()
				if err != nil {
					util.Failed("Failed to start %s: %v", nodeps.RouterContainer, err)
				}
			}
			util.Success("Restarted %s service for %s", service, app.GetName())
		}
	},
}

func init() {
	DebugCmd.AddCommand(DebugRebuildCmd)
	DebugRebuildCmd.Flags().BoolVarP(&buildAll, "all", "a", false, "Rebuild all services and restart the project")
	DebugRebuildCmd.Flags().Bool("cache", false, "Keep Docker cache")
	DebugRebuildCmd.Flags().StringVarP(&service, "service", "s", "web", "Rebuild the specified service and restart it")
}
