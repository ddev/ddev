package cmd

import (
	"github.com/ddev/ddev/pkg/amplitude"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugInstrumentationCleanCmd implements the ddev debug instrumentation clean command
var DebugInstrumentationCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Removes usage statistics from the local cache",
	Run: func(_ *cobra.Command, _ []string) {
		amplitude.Clean()

		util.Success("Usage statistics deleted.")
	},
}

func init() {
	DebugInstrumentationCmd.AddCommand(DebugInstrumentationCleanCmd)
}
