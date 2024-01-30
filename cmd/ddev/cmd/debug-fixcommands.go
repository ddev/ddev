package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugFixCommandsCmd fixes up global container commands without having to do a ddev start
var DebugFixCommandsCmd = &cobra.Command{
	Use:   "fix-commands",
	Short: "Fix up global container commands without running ddev start",
	Run: func(_ *cobra.Command, _ []string) {
		err := ddevapp.PopulateGlobalCustomCommandFiles()
		if err != nil {
			util.Warning("Failed to populate custom command files: %v", err)
		}
	},
}

func init() {
	DebugCmd.AddCommand(DebugFixCommandsCmd)
}
