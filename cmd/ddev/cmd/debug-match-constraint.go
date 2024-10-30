package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugCheckDBMatch verified that the DB Type/Version in container matches configured
var DebugMatchConstraint = &cobra.Command{
	Use:     "match-constraint",
	Short:   "Check if the currently installed ddev matches the specified version constraint.",
	Args:    cobra.ExactArgs(1),
	Example: `ddev debug match-constraint ">= 1.24.0"`,
	Run: func(_ *cobra.Command, args []string) {
		versionConstraint := args[0]

		err := ddevapp.CheckDdevVersionConstraint(versionConstraint, "", "")
		if err != nil {
			util.Failed(err.Error())
		}
	},
}

func init() {
	DebugCmd.AddCommand(DebugMatchConstraint)
}
