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
	Long: `Display custom configuration files in the current project.

By default, shows only files that would warn on startup (user-created files
without a #ddev-generated or #ddev-silent-no-warn marker).

Use --all to also show add-on files (labeled "addon <name>") and silenced
files (labeled #ddev-silent-no-warn).`,
	Run: func(cmd *cobra.Command, _ []string) {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Can't find active project: %v", err)
		}

		showAll, _ := cmd.Flags().GetBool("all")
		message, hasWarnings := app.CheckCustomConfig(showAll)
		if hasWarnings {
			util.Warning(message)
		} else {
			util.Success(message)
		}
	},
}

func init() {
	UtilityCheckCustomConfig.Flags().Bool("all", false, `Include add-on files (labeled "addon <name>") and silenced files (#ddev-silent-no-warn)`)
	DebugCmd.AddCommand(UtilityCheckCustomConfig)
}
