package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var restartAll bool

// RestartCmd rebuilds an apps settings
var RestartCmd = &cobra.Command{
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 0),
	Use:               "restart [projects]",
	Short:             "Restart a project or several projects.",
	Long:              `Stops named projects and then starts them back up again.`,
	Example: `ddev restart
ddev restart <project1> <project2>
ddev restart --all`,
	PreRun: func(_ *cobra.Command, _ []string) {
		dockerutil.EnsureDdevNetwork()
	},
	Run: func(cmd *cobra.Command, args []string) {
		projects, err := getRequestedProjects(args, restartAll)
		if err != nil {
			util.Failed("Failed to get project(s): %v", err)
		}
		if len(projects) > 0 {
			instrumentationApp = projects[0]
		}

		// Look for version change and opt-in to instrumentation if it has changed.
		// Do it here rather than waiting for app.Start() so the poweroff prompt
		// comes before any of the restart output.
		ddevapp.RunUpgradeCheck()

		noCache, _ := cmd.Flags().GetBool("no-cache")

		for _, app := range projects {
			app.NoCache = noCache
			output.UserOut.Printf("Restarting project %s...", app.GetName())
			err = app.Restart()
			if err != nil {
				util.Failed("Failed to restart %s: %v", app.GetName(), err)
			}

			util.Success("Restarted %s", app.GetName())
			emitReachProjectMessage(app)
		}
	},
}

func init() {
	RestartCmd.Flags().BoolP("skip-confirmation", "y", false, "Skip any confirmation steps")
	RestartCmd.Flags().BoolP("no-cache", "", false, "Rebuild custom Docker image layers without cache")
	RestartCmd.Flags().BoolVarP(&restartAll, "all", "a", false, "Restart all projects")
	RootCmd.AddCommand(RestartCmd)
}
