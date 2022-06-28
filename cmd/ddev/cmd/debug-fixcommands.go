package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugFixCommandsCmd fixes up custom/shell commands without having to do a ddev start
var DebugFixCommandsCmd = &cobra.Command{
	Use:   "fix-commands",
	Short: "Fix up custom/shell commands without running ddev start",
	Run: func(cmd *cobra.Command, args []string) {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Can't find active project: %v", err)
		}
		err = ddevapp.PopulateCustomCommandFiles(app)
		if err != nil {
			util.Warning("Failed to populate custom command files: %v", err)
		}
	},
}

func init() {
	DebugCmd.AddCommand(DebugFixCommandsCmd)
}
