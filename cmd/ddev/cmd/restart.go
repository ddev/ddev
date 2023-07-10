package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"strings"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var restartAll bool

// RestartCmd rebuilds an apps settings
var RestartCmd = &cobra.Command{
	Use:   "restart [projects]",
	Short: "Restart a project or several projects.",
	Long:  `Stops named projects and then starts them back up again.`,
	Example: `ddev restart
ddev restart <project1> <project2>
ddev restart --all`,
	PreRun: func(cmd *cobra.Command, args []string) {
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

		skip, err := cmd.Flags().GetBool("skip-confirmation")
		if err != nil {
			util.Failed(err.Error())
		}

		// Look for version change and opt-in to instrumentation if it has changed.
		err = checkDdevVersionAndOptInInstrumentation(skip)
		if err != nil {
			util.Failed(err.Error())
		}

		for _, app := range projects {

			output.UserOut.Printf("Restarting project %s...", app.GetName())
			err = app.Restart()
			if err != nil {
				util.Failed("Failed to restart %s: %v", app.GetName(), err)
			}

			util.Success("Restarted %s", app.GetName())
			httpURLs, urlList, _ := app.GetAllURLs()
			if globalconfig.GetCAROOT() == "" || ddevapp.IsRouterDisabled(app) {
				urlList = httpURLs
			}
			util.Success("Your project can be reached at %s", strings.Join(urlList, " "))
		}
	},
}

func init() {
	RestartCmd.Flags().BoolP("skip-confirmation", "y", false, "Skip any confirmation steps")
	RestartCmd.Flags().BoolVarP(&restartAll, "all", "a", false, "restart all projects")
	RootCmd.AddCommand(RestartCmd)
}
