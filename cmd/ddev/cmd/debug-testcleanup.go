package cmd

import (
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugTestCleanupCmd implements the ddev debug testcleanup command
var DebugTestCleanupCmd = &cobra.Command{
	Use:     "testcleanup",
	Short:   "Removes diagnostic projects prefixed with 'tryddevproject-'",
	Example: "ddev debug testcleanup",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 0 {
			util.Failed("This command takes no additional arguments")
		}
		allProjects, err := ddevapp.GetProjects(false)
		if err != nil {
			util.Failed("failed getting GetProjects: %v", err)
		}
		hasDeletedProjects := false
		for _, project := range allProjects {
			if strings.HasPrefix(project.GetName(), "tryddevproject-") {
				if err := project.Stop(true, false); err != nil {
					util.Failed("Failed to remove project %s: \n%v", project.GetName(), err)
				}
				hasDeletedProjects = true
			}
		}
		if hasDeletedProjects {
			util.Success("Finished cleaning ddev diagnostic projects")
		} else {
			util.Warning("No ddev diagnostic projects were found")
		}
	},
}

func init() {
	DebugCmd.AddCommand(DebugTestCleanupCmd)
}
