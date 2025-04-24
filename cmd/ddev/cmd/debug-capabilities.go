package cmd

import (
	"strings"

	"github.com/ddev/ddev/pkg/output"
	"github.com/spf13/cobra"
)

// DebugCapabilitiesCmd implements the ddev debug capabilities command
var DebugCapabilitiesCmd = &cobra.Command{
	Use:        "capabilities",
	Short:      "Show capabilities of this version of DDEV",
	Deprecated: "\nddev debug capabilities is no longer maintained \nand may be removed in a future release.\nUse ddev_version_constraint to specify versions of DDEV instead. \nhttps://ddev.readthedocs.io/en/stable/users/configuration/config/#ddev_version_constraint\n\n",
	Hidden:     true,
	Run: func(_ *cobra.Command, _ []string) {
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
			"debian-bookworm",
			"corepack",
		}
		output.UserOut.WithField("raw", capabilities).Print(strings.Join(capabilities, "\n"))
	},
}

func init() {
	DebugCmd.AddCommand(DebugCapabilitiesCmd)
}
