package cmd

import (
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
	"os"
)

// DdevRestoreSnapshotCommand provides the ability to revert to a database snapshot
var DdevRestoreSnapshotCommand = &cobra.Command{
	Hidden: true,
	Use:    "restore-snapshot [snapshot_name]",
	Short:  "Restore a project's database to the provided snapshot version.",
	Long:   "Please use \"snapshot restore\" command",
	Run: func(_ *cobra.Command, _ []string) {
		util.Failed("Please use \"ddev snapshot restore\".")
		os.Exit(1)
	},
}

func init() {
	RootCmd.AddCommand(DdevRestoreSnapshotCommand)
}
