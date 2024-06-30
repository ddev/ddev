package cmd

import (
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/spf13/cobra"
)

// DebugVersionConstraintCmd implements the ddev debug version-constraint command
var DebugVersionConstraintCmd = &cobra.Command{
	Use:   "version-constraint [constraint]",
	Short: "Checks if the current version of DDEV meets the given constraint",
	Example: `ddev debug version-constraint ">=v1.23.0-alpha1"

if [ "$(ddev debug version-constraint ">=v1.23.3" 2>/dev/null)" != "true" ]; then
    echo "This add-on requires DDEV v1.23.3 or higher, please upgrade." && exit 2
fi`,
	Run: func(_ *cobra.Command, args []string) {

		if len(args) == 0 || len(args) > 1 {
			util.Failed("This command only takes one optional argument: [constraint]")
		}

		constraint := args[0]
		if !strings.Contains(constraint, "-") {
			// Allow pre-releases to be included in the constraint validation
			// @see https://github.com/Masterminds/semver#working-with-prerelease-versions
			constraint += "-0"
		}
		c, err := semver.NewConstraint(constraint)
		if err != nil {
			util.Failed("'%s' constraint is invalid. See https://github.com/Masterminds/semver#checking-version-constraints for valid constraints format.", constraint)
		}

		// Make sure we do this check with valid released versions
		v, err := semver.NewVersion(versionconstants.DdevVersion)
		if err == nil {
			if !c.Check(v) {
				util.Failed("Your DDEV version '%s' doesn't meet the constraint '%s'.", versionconstants.DdevVersion, constraint)
			}
		}

		output.UserOut.WithField("raw", "true").Print("true")
	},
}

func init() {
	DebugCmd.AddCommand(DebugVersionConstraintCmd)
}
