package cmd

import (
	"fmt"
	"log"

	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// LocalDevReconfigCmd rebuilds an apps settings
var LocalDevReconfigCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the local development environment for a site.",
	Long:  `Restart stops the containers for site's environment and starts them back up again.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		client := dockerutil.GetDockerClient()

		err := dockerutil.EnsureNetwork(client, netName)
		if err != nil {
			log.Fatal(err)
		}

	},
	Run: func(cmd *cobra.Command, args []string) {
		app, err := getActiveApp()
		if err != nil {
			util.Failed("Failed to restart: %v", err)
		}

		fmt.Printf("Restarting environment for %s...", app.GetName())
		err = app.Stop()
		if err != nil {
			util.Failed("Failed to restart %s: %v", app.GetName(), err)
		}

		err = app.Start()
		if err != nil {
			util.Failed("Failed to restart %s: %v", app.GetName(), err)
		}

		fmt.Println("Waiting for the environment to become ready. This may take a couple of minutes...")
		err = app.Wait("web")
		if err != nil {
			util.Failed("Failed to restart %s: %v", app.GetName(), err)
		}

		util.Success("Successfully restarted %s", app.GetName())
		util.Success("Your application can be reached at: %s", app.URL())
	},
}

func init() {
	RootCmd.AddCommand(LocalDevReconfigCmd)

}
