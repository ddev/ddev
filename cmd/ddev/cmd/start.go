package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/manifoldco/promptui"
	"os"
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

		selectFlag, err := cmd.Flags().GetBool("select")

		if err != nil {
			util.Failed(err.Error())
		}

		if selectFlag {
			inactiveProjects, err := ddevapp.GetInactiveProjects()

			if err != nil {
				util.Failed(err.Error())
			}

			if len(inactiveProjects) == 0 {
				util.Warning("No project to start available")
				os.Exit(0)
			}

			inactiveProjectNames := ddevapp.ExtractProjectNames(inactiveProjects)

			prompt := promptui.Select{
				Label: "Projects",
				Items: inactiveProjectNames,
				Templates: &promptui.SelectTemplates{
					Label: "{{ . | cyan }}:",
				},
				StartInSearchMode: true,
				Searcher: func(input string, idx int) bool {
					return strings.Contains(inactiveProjectNames[idx], input)
				},
			}

			_, projectName, err := prompt.Run()

			if err != nil {
				util.Failed(err.Error())
			}

			args = append(args, projectName)
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
			httpURLs, httpsURLs, _ := project.GetAllURLs()
			if !nodeps.IsGitpod() && (globalconfig.GetCAROOT() == "" || ddevapp.IsRouterDisabled(project)) {
				httpsURLs = httpURLs
			}
			util.Success("Project can be reached at %s", strings.Join(httpsURLs, " "))
		}
	},
}

func init() {
	StartCmd.Flags().BoolVarP(&startAll, "all", "a", false, "Start all projects")
	StartCmd.Flags().BoolP("skip-confirmation", "y", false, "Skip any confirmation steps")
	StartCmd.Flags().BoolP("select", "s", false, "Interactively select a project to start")
	RootCmd.AddCommand(StartCmd)
}
