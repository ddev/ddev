package cmd

import (
	"os"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// LocalDevRestartCmd rebuilds an apps settings
var LocalDevRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the local development environment for a site.",
	Long:  `Restart stops the containers for site's environment and starts them back up again.`,
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

		output.UserOut.Printf("Restarting environment for %s...", app.GetName())
		err = app.Stop()
		if err != nil {
			util.Failed("Failed to restart %s: %v", app.GetName(), err)
		}

		err = app.Start()
		if err != nil {
			util.Failed("Failed to restart %s: %v", app.GetName(), err)
		}

		util.Success("Successfully restarted %s", app.GetName())
		util.Success("Your application can be reached at: %s", app.URL())
	},
}

func init() {
	RootCmd.AddCommand(LocalDevRestartCmd)

}
