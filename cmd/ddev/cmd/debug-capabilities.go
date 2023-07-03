package cmd

import (
	"strings"

	"github.com/ddev/ddev/pkg/output"
	"github.com/spf13/cobra"
)

// DebugCapabilitiesCmd implements the ddev debug capabilities command
var DebugCapabilitiesCmd = &cobra.Command{
	Use:   "capabilities",
	Short: "Show capabilities of this version of ddev",
	Run: func(cmd *cobra.Command, args []string) {
		capabilities := []string{
			"multiple-dockerfiles",
			"interactive-project-selection",
			"ddev-get-yaml-interpolation",
			"config-star-yaml-merging",
			"pre-dockerfile-insertion",
			"user-env-var",
			"npm-yarn-caching",
			"exposed-ports-configuration",
			"daemon-run-configuration",
			"get-volume-db-version",
			"migrate-database",
			"web-start-hooks",
			"add-on-versioning",
			"multiple-upload-dirs",
		}
		output.UserOut.WithField("raw", capabilities).Print(strings.Join(capabilities, "\n"))
	},
}

func init() {
	DebugCmd.AddCommand(DebugCapabilitiesCmd)
}
