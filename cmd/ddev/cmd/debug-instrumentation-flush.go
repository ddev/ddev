package cmd

import (
	"github.com/ddev/ddev/pkg/amplitude"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugInstrumentationFlushCmd implements the ddev debug instrumentation flush command
var DebugInstrumentationFlushCmd = &cobra.Command{
	Use:   "flush",
	Short: "Transmits usage statistics from the local cache",
	Run: func(cmd *cobra.Command, args []string) {
		if amplitude.IsDisabled() {
			util.Warning("Instrumentation is currently disabled.")

			return
		}

		amplitude.FlushForce()

		util.Success("Usage statistics transmitted.")
	},
}

func init() {
	DebugInstrumentationCmd.AddCommand(DebugInstrumentationFlushCmd)
}
