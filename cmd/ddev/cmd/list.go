package cmd

import (
	"fmt"
	"os"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/spf13/cobra"
)

// DevListCmd represents the list command
var DevListCmd = &cobra.Command{
	Use:   "list",
	Short: "List applications that exist locally",
	Long:  `List applications that exist locally.`,
	Run: func(cmd *cobra.Command, args []string) {
		apps := platform.GetApps()

		if len(apps) < 1 {
			fmt.Println("There are no running ddev applications.")
			os.Exit(0)
		}

		for platformType, sites := range apps {
			platform.RenderAppTable(platformType, sites)
		}
	},
}

func init() {
	RootCmd.AddCommand(DevListCmd)
}
