package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// noConfirm: If true, --yes, we won't stop and prompt before each deletion
var noConfirm bool

// if deleteAll is true, we'll delete all projects
var deleteAll bool

// DeleteCmd provides the delete command
var DeleteCmd = &cobra.Command{
	Use:   "delete [projectname ...]",
	Short: "Remove all project information (including database) for an existing project",
	Long:  `Removes all ddev project information (including database) for an existing project, but does not touch the project codebase or the codebase's .ddev folder.'.`,
	Example: `ddev delete
ddev delete proj1 proj2 proj3
ddev delete --omit-snapshot proj1
ddev delete --omit-snapshot --yes proj1 proj2
ddev delete -Oy
ddev delete --all`,
	Run: func(cmd *cobra.Command, args []string) {
		if noConfirm && deleteAll {
			util.Failed("Sorry, it's not possible to use flags --all and --yes together")
		}
		projects, err := getRequestedProjects(args, deleteAll)
		if err != nil {
			util.Failed("Failed to get project(s): %v", err)
		}
		if len(projects) > 0 {
			instrumentationApp = projects[0]
		}

		// Iterate through the list of projects built above, removing each one.
		for _, project := range projects {
			if !noConfirm {
				prompt := "OK to delete this project and its database?\n  %s in %s\nThe code and its .ddev directory will not be touched.\n"
				if !omitSnapshot {
					prompt = prompt + "A database snapshot will be made before the database is deleted.\n"
				}
				if !util.Confirm(fmt.Sprintf(prompt+"OK to delete %s?", project.Name, project.AppRoot, project.Name)) {
					continue
				}
			}
			// Explanation of what's going on (including where the project is)
			// Stop it.
			// Delete database
			// Delete any other associated volumes

			// We do the snapshot UNLESS omit-snapshot is set; the project may have to be
			// started to do the snapshot.
			if project.SiteStatus() != ddevapp.SiteRunning && !omitSnapshot {
				util.Warning("project must be started to do the snapshot")
				err = project.Start()
				if err != nil {
					util.Failed("Failed to start project %s: %v", project.Name, err)
				}
			}
			if err := project.Stop(true, !omitSnapshot); err != nil {
				util.Failed("Failed to remove project %s: \n%v", project.GetName(), err)
			}
		}
	},
}

func init() {
	DeleteCmd.Flags().Bool("clean-containers", true, "Clean up all ddev docker containers which are not required by this version of ddev")
	DeleteCmd.Flags().BoolVarP(&omitSnapshot, "omit-snapshot", "O", false, "Omit/skip database snapshot")
	DeleteCmd.Flags().BoolVarP(&noConfirm, "yes", "y", false, "Yes - skip confirmation prompt")
	DeleteCmd.Flags().BoolVarP(&deleteAll, "all", "a", false, "Delete all projects")

	RootCmd.AddCommand(DeleteCmd)
}
