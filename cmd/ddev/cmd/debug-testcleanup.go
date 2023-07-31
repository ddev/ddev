package cmd

import (
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/output"
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
		if len(allProjects) < 1 {
			output.UserOut.Println("No ddev projects were found.")
			return
		}
		for _, project := range allProjects {
			name := project.GetName()
			if !strings.HasPrefix(name, "tryddevproject-") {
				continue
			}
			output.UserOut.Printf("Running ddev delete -Oy %v", name)
			err = DeleteCmd.Flags().Set("omit-snapshot", "true")
			if err != nil {
				util.Failed("failed setting a flag --omit-snapshot for ddev delete: %v", err)
			}
			err = DeleteCmd.Flags().Set("yes", "true")
			if err != nil {
				util.Failed("failed setting a flag --yes for ddev delete: %v", err)
			}
			DeleteCmd.Run(cmd, []string{name})
		}
		util.Success("Finished cleaning ddev diagnostic projects")
	},
}

func init() {
	DebugCmd.AddCommand(DebugTestCleanupCmd)
}
