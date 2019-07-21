package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
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
		projects, err := getRequestedProjects(args, startAll)
		if err != nil {
			util.Failed("Failed to get project(s): %v", err)
		}

		// Look for version change and opt-in Sentry if it has changed.
		err = checkDdevVersionAndOptInInstrumentation()
		if err != nil {
			util.Failed(err.Error())
		}

		for _, project := range projects {
			if err := ddevapp.CheckForMissingProjectFiles(project); err != nil {
				util.Failed("Failed to start %s: %v", project.GetName(), err)
			}

			output.UserOut.Printf("Starting %s...", project.GetName())
			if err := project.Start(); err != nil {
				util.Failed("Failed to start %s: %v", project.GetName(), err)
				continue
			}

			util.Success("Successfully started %s", project.GetName())
			util.Success("Project can be reached at %s", strings.Join(project.GetAllURLs(), ", "))
			if project.WebcacheEnabled {
				util.Warning("All contents were copied to fast docker filesystem,\nbut bidirectional sync operation may not be fully functional for a few minutes.")
			}
		}
	},
}

func init() {
	StartCmd.Flags().BoolVarP(&startAll, "all", "a", false, "Start all projects")
	RootCmd.AddCommand(StartCmd)
}
