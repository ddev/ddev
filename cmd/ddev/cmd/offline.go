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
	Short: "Enable or disable offline mode.",
	Long: `Enable or disable offline mode. When offline mode is enabled, ddev will use
hosts file entries for local development domains instead of relying on remote
DNS for ddev.site.`,
	Run: func(cmd *cobra.Command, args []string) {
		switch {
		case args[0] == "enable":
			err := platform.SetOfflineMode()
			if err != nil {
				log.Fatal(err)
			}
			util.Success("Offline mode enabled. ddev will use hosts file entries for local development domains.")
		case args[0] == "disable":
			err := platform.UnsetOfflineMode()
			if err != nil {
				log.Fatal(err)
			}
			util.Success("Offline mode disabled. Hosts entries will no longer be added for sites.")
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
