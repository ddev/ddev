package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// CloneCreateCmd implements the "ddev clone create" command.
var CloneCreateCmd = &cobra.Command{
	Use:   "create <clone-name>",
	Short: "Create a clone of the current DDEV project",
	Long: `Create a clone of the current DDEV project using git worktree and Docker volume duplication.

Creates a git worktree at ../<project>-clone-<name>, copies all Docker volumes,
configures a new DDEV project, and starts it.`,
	Example: `ddev clone create feature-x
ddev clone create feature-x --branch existing-branch
ddev clone create feature-x --no-start`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cloneName := args[0]

		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Unable to get active project: %v", err)
		}

		branch, _ := cmd.Flags().GetString("branch")
		noStart, _ := cmd.Flags().GetBool("no-start")

		if err := ddevapp.CloneCreate(app, cloneName, branch, noStart); err != nil {
			util.Failed("Failed to create clone: %v", err)
		}
	},
}

func init() {
	CloneCmd.AddCommand(CloneCreateCmd)
	CloneCreateCmd.Flags().StringP("branch", "b", "", "Check out an existing branch instead of creating a new one")
	CloneCreateCmd.Flags().Bool("no-start", false, "Configure the clone but do not start it")
}
