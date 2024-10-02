package cmd

import (
	exec2 "github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"strings"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var (
	buildAll bool
	service  string
)

// DebugRefreshCmd implements the ddev debug refresh command
var DebugRefreshCmd = &cobra.Command{
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 1),
	Use:               "refresh",
	Short:             "Refreshes Docker cache for project with verbose output",
	Run: func(cmd *cobra.Command, args []string) {
		projectName := ""

		if len(args) > 1 {
			util.Failed("This command only takes one optional argument: project name")
		}

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

		output.UserOut.Printf("Rebuilding project images...")
		buildDurationStart := util.ElapsedDuration(time.Now())
		composeRenderedPath := app.DockerComposeFullRenderedYAMLPath()
		withoutCache := !cmd.Flags().Changed("cache")

		buildArgs := []string{"-f", composeRenderedPath, "--progress", "plain", "build"}

		if !buildAll {
			buildArgs = append(buildArgs, cmd.Flag("service").Value.String())
		}

		if withoutCache {
			buildArgs = append(buildArgs, "--no-cache")
		}

		util.Success("Rebuilding project %s with `%s %v`", app.Name, composeBinaryPath, strings.Join(buildArgs, " "))

		err = exec2.RunInteractiveCommand(composeBinaryPath, buildArgs)
		if err != nil {
			util.Failed("Failed to execute `%s %v`: %v", composeBinaryPath, strings.Join(buildArgs, " "), err)
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
	DebugRefreshCmd.Flags().BoolVarP(&buildAll, "all", "a", false, "Rebuild all services")
	DebugRefreshCmd.Flags().Bool("cache", false, "Keep Docker cache")
	DebugRefreshCmd.Flags().StringVarP(&service, "service", "s", "web", "Rebuild specified service")
}
