package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// UtilityCheckCustomConfig displays custom configuration in the current project
var UtilityCheckCustomConfig = &cobra.Command{
	Use:   "check-custom-config",
	Short: "Display custom configuration files in the current project",
	Run: func(_ *cobra.Command, _ []string) {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Can't find active project: %v", err)
		}

		app.CheckCustomConfig(true)
	},
}

func init() {
	DebugCmd.AddCommand(UtilityCheckCustomConfig)
}
