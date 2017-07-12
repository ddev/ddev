package cmd

import (
	"fmt"
	"os"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// StartCmd represents the add command
var StartCmd = &cobra.Command{
	Use:     "start",
	Aliases: []string{"add"},
	Short:   "Start the local development environment for a site.",
	Long: `Start initializes and configures the web server and database containers to
provide a working environment for development.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			err := cmd.Usage()
			util.CheckErr(err)
			os.Exit(0)
		}

		dockerNetworkPreRun()
	},
	Run: func(cmd *cobra.Command, args []string) {
		appStart()
	},
}

func appStart() {
	app, err := platform.GetActiveApp("")
	if err != nil {
		util.Failed("Failed to start: %s", err)
	}

	fmt.Printf("Starting environment for %s...\n", app.GetName())

	err = app.Start()
	if err != nil {
		util.Failed("Failed to start %s: %v", app.GetName(), err)
	}

	util.Success("Successfully started %s", app.GetName())
	util.Success("Your application can be reached at: %s", app.URL())

}
func init() {
	RootCmd.AddCommand(StartCmd)
}
