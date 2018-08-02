package cmd

import (
	"strings"

	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var startAll bool

// StartCmd provides the ddev start command
var StartCmd = &cobra.Command{
	Use:     "start [projectname ...]",
	Aliases: []string{"add"},
	Short:   "Start a ddev project.",
	Long: `Start initializes and configures the web server and database containers
to provide a local development environment. You can run 'ddev start' from a
project directory to start that project, or you can start stopped projects in
any directory by running 'ddev start projectname [projectname ...]'`,
	PreRun: func(cmd *cobra.Command, args []string) {
		dockerutil.EnsureDdevNetwork()
	},
	Run: func(cmd *cobra.Command, args []string) {
		apps, err := getRequestedApps(args, startAll)
		if err != nil {
			util.Failed("Unable to get project(s): %v", err)
		}

		if len(apps) == 0 {
			output.UserOut.Printf("There are no projects to start.")
		}

		for _, app := range apps {
			output.UserOut.Printf("Starting %s...", app.GetName())

			if err := app.Start(); err != nil {
				util.Failed("Failed to start %s: %v", app.GetName(), err)
				continue
			}

			util.Success("Successfully started %s", app.GetName())
			util.Success("Project can be reached at %s", strings.Join(app.GetAllURLs(), ", "))
		}
	},
}

func init() {
	StartCmd.Flags().BoolVarP(&startAll, "all", "a", false, "Start all stopped projects")
	RootCmd.AddCommand(StartCmd)
}
