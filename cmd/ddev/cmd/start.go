package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/globalconfig"
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
	Example: `ddev start
ddev start <project1> <project2>
ddev start --all`,
	PreRun: func(cmd *cobra.Command, args []string) {
		dockerutil.EnsureDdevNetwork()
	},
	Run: func(cmd *cobra.Command, args []string) {

		skip, err := cmd.Flags().GetBool("skip-confirmation")
		if err != nil {
			util.Failed(err.Error())
		}

		// Look for version change and opt-in to instrumentation if it has changed.
		err = checkDdevVersionAndOptInInstrumentation(skip)
		if err != nil {
			util.Failed(err.Error())
		}

		projects, err := getRequestedProjects(args, startAll)
		if err != nil {
			util.Failed("Failed to get project(s): %v", err)
		}
		if len(projects) > 0 {
			instrumentationApp = projects[0]
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
			httpURLs, urlList, _ := project.GetAllURLs()
			if globalconfig.GetCAROOT() == "" {
				urlList = httpURLs
			}
			util.Success("Project can be reached at %s", strings.Join(urlList, " "))
		}
	},
}

func init() {
	StartCmd.Flags().BoolVarP(&startAll, "all", "a", false, "Start all projects")
	StartCmd.Flags().BoolP("skip-confirmation", "y", false, "Skip any confirmation steps")
	RootCmd.AddCommand(StartCmd)
}
