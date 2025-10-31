package cmd

import (
	"github.com/ddev/ddev/pkg/amplitude"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugInstrumentationFlushCmd implements the ddev utility instrumentation flush command
var DebugInstrumentationFlushCmd = &cobra.Command{
	Use:   "flush",
	Short: "Transmits usage statistics from the local cache",
	Run: func(_ *cobra.Command, _ []string) {
		amplitude.CheckSetUp()

		if !globalconfig.IsInternetActive() {
			util.WarningOnce("Warning: %v", globalconfig.IsInternetActiveErr)
		}

		if amplitude.IsDisabled() {
			util.Failed("Instrumentation is currently disabled.")
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
