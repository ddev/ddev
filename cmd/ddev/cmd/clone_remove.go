package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// CloneRemoveCmd implements the "ddev clone remove" command.
var CloneRemoveCmd = &cobra.Command{
	Use:               "remove <clone-name>",
	Short:             "Remove a clone and clean up all its resources",
	Long:              "Remove a clone and clean up all its Docker resources, git worktree, and project registration.",
	ValidArgsFunction: ddevapp.GetCloneNamesFunc(1),
	Example: `ddev clone remove feature-x
ddev clone remove feature-x --force`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cloneName := args[0]

		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Unable to get active project: %v", err)
		}

		force, _ := cmd.Flags().GetBool("force")

		if err := ddevapp.CloneRemove(app, cloneName, force); err != nil {
			util.Failed("Failed to remove clone: %v", err)
		}
	},
}

func init() {
	CloneCmd.AddCommand(CloneRemoveCmd)
	CloneRemoveCmd.Flags().Bool("force", false, "Skip confirmation for dirty worktrees")
}
