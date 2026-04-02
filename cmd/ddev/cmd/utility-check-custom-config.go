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

By default, shows only files that would warn on startup: user-created files
and files with an unexpected #ddev-generated marker. Recognized add-on files
and files with #ddev-silent-no-warn are excluded.

Use --all to show all custom configuration with annotations:
  (add-on <name>) for add-on files
  (#ddev-generated) for add-on files with a #ddev-generated marker
  (unexpected #ddev-generated) for unrecognized #ddev-generated files
  (#ddev-silent-no-warn) for silenced files`,
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
	UtilityCheckCustomConfig.Flags().Bool("all", false, `Show all custom configuration with annotations for add-on files, #ddev-generated, and #ddev-silent-no-warn`)
	DebugCmd.AddCommand(UtilityCheckCustomConfig)
}
