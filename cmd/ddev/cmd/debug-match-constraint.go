package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugMatchConstraintCmd Compares a constraint against the installed DDEV version.
var DebugMatchConstraintCmd = &cobra.Command{
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
	DebugCmd.AddCommand(DebugMatchConstraintCmd)
}
