package cmd

import (
	"fmt"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// ClonePruneCmd implements the "ddev clone prune" command.
var ClonePruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Clean up stale clone references",
	Long:  "Clean up clones whose worktree directories no longer exist.",
	Example: `ddev clone prune
ddev clone prune --dry-run`,
	Run: func(cmd *cobra.Command, _ []string) {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Unable to get active project: %v", err)
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		pruned, err := ddevapp.ClonePrune(app, dryRun)
		if err != nil {
			util.Failed("Failed to prune clones: %v", err)
		}

		sourceProjectName := ddevapp.GetSourceProjectNameExported(app)

		if len(pruned) == 0 {
			util.Success("No stale clones found for project '%s'.", sourceProjectName)
			return
		}

		if dryRun {
			util.Success("%d stale clone(s) would be pruned.", len(pruned))
		} else {
			util.Success(fmt.Sprintf("Pruned %d stale clone(s).", len(pruned)))
		}
	},
}

func init() {
	CloneCmd.AddCommand(ClonePruneCmd)
	ClonePruneCmd.Flags().Bool("dry-run", false, "Show what would be cleaned up without taking action")
}
