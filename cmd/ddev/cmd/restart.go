package cmd

import (
	"github.com/drud/ddev/pkg/globalconfig"
	"strings"

	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
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

		for _, app := range projects {

			output.UserOut.Printf("Restarting project %s...", app.GetName())
			err = app.Stop(false, false)
			if err != nil {
				util.Failed("Failed to restart %s: %v", app.GetName(), err)
			}

			err = app.Start()
			if err != nil {
				util.Failed("Failed to restart %s: %v", app.GetName(), err)
			}

			util.Success("Restarted %s", app.GetName())
			httpURLs, urlList, _ := app.GetAllURLs()
			if globalconfig.GetCAROOT() == "" {
				urlList = httpURLs
			}

			util.Success("Your project can be reached at %s", strings.Join(urlList, " "))
		}
	},
}

func init() {
	RestartCmd.Flags().BoolVarP(&restartAll, "all", "a", false, "restart all projects")
	RootCmd.AddCommand(RestartCmd)
}
