package cmd

import (
	"github.com/spf13/cobra"
)

// CloneCmd is the top-level "ddev clone" command container for managing project clones.
var CloneCmd = &cobra.Command{
	Use:   "clone [command]",
	Short: "A collection of commands for managing project clones",
	Long:  "Create, list, remove, and prune cloned DDEV project environments using git worktrees and Docker volume duplication.",
	Example: `ddev clone create feature-x
ddev clone list
ddev clone remove feature-x
ddev clone prune`,
}

func init() {
	RootCmd.AddCommand(CloneCmd)
}
