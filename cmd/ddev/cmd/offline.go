package cmd

import (
	"log"
	"os"

	"github.com/drud/ddev/pkg/plugins/platform"

	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// OfflineCmd represents the local command
var OfflineCmd = &cobra.Command{
	Use:   "offline [enable/disable]",
	Short: "Manage your hostfile entries.",
	Long:  `Manage your hostfile entries.`,
	Run: func(cmd *cobra.Command, args []string) {
		switch {
		case args[0] == "enable":
			err := platform.SetOfflineMode()
			if err != nil {
				log.Fatal(err)
			}
		case args[0] == "disable":
			err := platform.UnsetOfflineMode()
			if err != nil {
				log.Fatal(err)
			}
		default:
			err := cmd.Usage()
			util.CheckErr(err)
			os.Exit(0)
		}
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {},
}

func init() {
	RootCmd.AddCommand(OfflineCmd)
}
