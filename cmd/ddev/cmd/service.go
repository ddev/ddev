package cmd

import (
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// ServiceCmd is the top-level "ddev service" command
var ServiceCmd = &cobra.Command{
	Use:        "service [command]",
	Short:      "The service commands have been deprecated and removed and replaced by ddev add-on",
	Hidden:     true,
	Deprecated: `true`,
	Run: func(cmd *cobra.Command, _ []string) {
		err := cmd.Usage()
		util.CheckErr(err)
	},
}

func init() {
	RootCmd.AddCommand(ServiceCmd)
}
