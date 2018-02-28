package cmd

import (
	"os"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DdevRestartCmd rebuilds an apps settings
var DdevRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the development environment for a project.",
	Long:  `Restart stops the containers for project and starts them back up again.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			err := cmd.Usage()
			util.CheckErr(err)
			os.Exit(0)
		}

		dockerutil.EnsureDdevNetwork()
	},
	Run: func(cmd *cobra.Command, args []string) {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Failed to restart: %v", err)
		}

		output.UserOut.Printf("Restarting project %s...", app.GetName())
		err = app.Stop()
		if err != nil {
			util.Failed("Failed to restart %s: %v", app.GetName(), err)
		}

		err = app.Start()
		if err != nil {
			util.Failed("Failed to restart %s: %v", app.GetName(), err)
		}

		util.Success("Successfully restarted %s", app.GetName())
		util.Success("Your project can be reached at: %s and %s", app.GetHTTPURL(), app.GetHTTPSURL())
	},
}

func init() {
	RootCmd.AddCommand(DdevRestartCmd)

}
