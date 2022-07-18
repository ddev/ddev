package cmd

import (
	"github.com/drud/ddev/pkg/output"
	"github.com/spf13/cobra"
	"strings"
)

// DebugCapabilitiesCmd implements the ddev debug capabilities command
var DebugCapabilitiesCmd = &cobra.Command{
	Use:   "capabilities",
	Short: "Show capabilities of this version of ddev",
	Run: func(cmd *cobra.Command, args []string) {
		capabilities := []string{"multiple-dockerfiles", "interactive-project-selection", "ddev-get-yaml-interpolation", "pre-dockerfile-insertion"}
		output.UserOut.WithField("raw", capabilities).Print(strings.Join(capabilities, "\n"))
	},
}

func init() {
	DebugCmd.AddCommand(DebugCapabilitiesCmd)
}
}
