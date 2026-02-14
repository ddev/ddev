package cmd

import (
	"github.com/ddev/ddev/pkg/tui"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// TUICmd launches the interactive TUI dashboard.
var TUICmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive TUI dashboard",
	Long:  `Launch an interactive terminal dashboard for managing DDEV projects. Provides a visual interface for starting, stopping, and monitoring projects.`,
	Run: func(_ *cobra.Command, _ []string) {
		if err := tui.Launch(); err != nil {
			util.Failed("TUI failed: %v", err)
		}
	},
}

func init() {
	RootCmd.AddCommand(TUICmd)
}
