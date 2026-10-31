package cmd

import (
	"path/filepath"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugFixCommandsCmd copies global container commands into running containers
var DebugFixCommandsCmd = &cobra.Command{
	Use:   "fix-commands",
	Short: "Copy global container commands into running containers",
	Long:  "Copy global container commands into the current project's running containers, or into those of every running project when run outside a project, so changes to them take effect without a restart.",
	Run: func(_ *cobra.Command, _ []string) {
		app, err := ddevapp.GetActiveApp("")
		inProject := err == nil
		apps := []*ddevapp.DdevApp{app}
		if !inProject {
			apps = ddevapp.GetActiveProjects()
		}
		sourceDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "commands")
		refreshed := false
		for _, app := range apps {
			if status, _ := app.SiteStatus(); status != ddevapp.SiteRunning {
				if inProject {
					util.Warning("Project %s is not running; 'ddev start' copies the global commands into its containers", app.Name)
				}
				continue
			}
			services, err := ddevapp.PopulateGlobalCustomCommandFiles(app)
			if err != nil {
				util.Warning("Failed to copy global commands into project %s: %v", app.Name, err)
			}
			if len(services) > 0 {
				util.Success("Copied global commands from %s to %s in the %s container(s) of project %s", sourceDir, ddevapp.GlobalCommandsContainerDir, strings.Join(services, ", "), app.Name)
				refreshed = true
			}
		}
		if !refreshed && !inProject {
			util.Warning("No running project has a container with a directory in %s", sourceDir)
		}
	},
}

func registerDebugFixcommandsCmd() {
	DebugCmd.AddCommand(DebugFixCommandsCmd)
}
