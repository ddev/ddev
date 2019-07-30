package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// PoweroffCommand contains the "ddev share" command
var PoweroffCommand = &cobra.Command{
	Use:     "poweroff",
	Short:   "Completely stop all projects and containers",
	Long:    `ddev poweroff stops all projects and containers, equivalent to ddev stop -a --stop-ssh-agent`,
	Example: `ddev poweroff`,
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		powerOff()
	},
}

func init() {
	RootCmd.AddCommand(PoweroffCommand)
}

func powerOff() {
	projects, err := ddevapp.GetProjects(true)
	if err != nil {
		util.Failed("Failed to get project(s): %v", err)
	}

	// Iterate through the list of projects built above, removing each one.
	for _, project := range projects {
		if project.SiteStatus() == ddevapp.SiteStopped {
			util.Warning("Project %s is not currently running. Try 'ddev start'.", project.GetName())
		}

		// We do the snapshot if either --snapshot or --remove-data UNLESS omit-snapshot is set
		doSnapshot := (createSnapshot || removeData) && !omitSnapshot
		if err := project.Stop(removeData, doSnapshot); err != nil {
			util.Failed("Failed to remove project %s: \n%v", project.GetName(), err)
		}
		if unlist {
			project.RemoveGlobalProjectInfo()
		}

		util.Success("Project %s has been stopped.", project.GetName())
	}

	if err := ddevapp.RemoveSSHAgentContainer(); err != nil {
		util.Error("Failed to remove ddev-ssh-agent: %v", err)
	}
}
