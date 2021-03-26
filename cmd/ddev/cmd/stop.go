package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// For single remove only: remove db and related data
var removeData bool

// Stop all projects (but not available with -a
var stopAll bool

// create a snapshot during remove (default to false with regular remove, default to true with rm --remove-data
var createSnapshot bool

// force omission of snapshot during remove-data
var omitSnapshot bool

// Stop the ddev-ssh-agent
var stopSSHAgent bool

var unlist bool

// DdevStopCmd represents the remove command
var DdevStopCmd = &cobra.Command{
	Use:     "stop [projectname ...]",
	Aliases: []string{"rm", "remove"},
	Short:   "Stop and remove the containers of a project. Does not lose or harm anything unless you add --remove-data.",
	Long: `Stop and remove the containers of a project. You can run 'ddev stop'
from a project directory to stop/remove that project, or you can stop/remove projects in
any directory by running 'ddev stop projectname [projectname ...]' or 'ddev stop -a'.

By default, stop is a non-destructive operation and will leave database
contents intact. It never touches your code or files directories.

To remove database contents and global listing, 
use "ddev delete" or "ddev stop --remove-data".

To snapshot the database on stop, use "ddev stop --snapshot"; A snapshot is automatically created on
"ddev stop --remove-data" unless you use "ddev stop --remove-data --omit-snapshot".
`,
	Example: `ddev stop
ddev stop proj1 proj2 proj3
ddev stop --all
ddev stop --all --stop-ssh-agent
ddev stop --remove-data`,
	Run: func(cmd *cobra.Command, args []string) {
		if createSnapshot && omitSnapshot {
			util.Failed("Illegal option combination: --snapshot and --omit-snapshot:")
		}

		projects, err := getRequestedProjects(args, stopAll)
		if err != nil {
			util.Failed("Failed to get project(s): %v", err)
		}
		if len(projects) > 0 {
			instrumentationApp = projects[0]
		}

		// Iterate through the list of projects built above, removing each one.
		for _, project := range projects {
			if project.SiteStatus() == ddevapp.SiteStopped {
				util.Success("Project %s is already stopped.", project.GetName())
			}

			// We do the snapshot if either --snapshot or --remove-data UNLESS omit-snapshot is set
			doSnapshot := (createSnapshot || removeData) && !omitSnapshot
			if err := project.Stop(removeData, doSnapshot); err != nil {
				util.Failed("Failed to stop project %s: \n%v", project.GetName(), err)
			}
			if unlist {
				project.RemoveGlobalProjectInfo()
			}

			util.Success("Project %s has been stopped.", project.GetName())
		}

		if stopSSHAgent {
			if err := ddevapp.RemoveSSHAgentContainer(); err != nil {
				util.Error("Failed to remove ddev-ssh-agent: %v", err)
			}
		}
	},
}

func init() {
	DdevStopCmd.Flags().BoolVarP(&removeData, "remove-data", "R", false, "Remove stored project data (MySQL, logs, etc.)")
	DdevStopCmd.Flags().BoolVarP(&createSnapshot, "snapshot", "S", false, "Create database snapshot")
	DdevStopCmd.Flags().BoolVarP(&omitSnapshot, "omit-snapshot", "O", false, "Omit/skip database snapshot")

	DdevStopCmd.Flags().BoolVarP(&stopAll, "all", "a", false, "Stop and remove all running or container-stopped projects and remove from global projects list")
	DdevStopCmd.Flags().BoolVarP(&stopSSHAgent, "stop-ssh-agent", "", false, "Stop the ddev-ssh-agent container")
	DdevStopCmd.Flags().BoolVarP(&unlist, "unlist", "U", false, "Remove the project from global project list, it won't show in ddev list until started again")

	RootCmd.AddCommand(DdevStopCmd)
}
