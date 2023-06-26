package cmd

import (
	"github.com/ddev/ddev/pkg/amplitude"
	"github.com/ddev/ddev/pkg/globalconfig"
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

		debugBackup := globalconfig.DdevDebug
		globalconfig.DdevDebug = true
		defer func() {
			globalconfig.DdevDebug = debugBackup
		}()

		amplitude.FlushForce()
	},
}

func init() {
	DebugInstrumentationCmd.AddCommand(DebugInstrumentationFlushCmd)
}
