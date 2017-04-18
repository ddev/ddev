package cmd

import (
	"fmt"
	"log"

	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// LocalDevReconfigCmd rebuilds an apps settings
var LocalDevReconfigCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the local development environment for a site.",
	Long:  `Restart stops the containers for site's environment and starts them back up again.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		client, err := util.GetDockerClient()
		if err != nil {
			log.Fatal(err)
		}

		err = util.EnsureNetwork(client, netName)
		if err != nil {
			log.Fatal(err)
		}

	},
	Run: func(cmd *cobra.Command, args []string) {
		app, err := getActiveApp()
		if err != nil {
			log.Fatalf("Could not find an active ddev configuration, have you run 'ddev config'?: %v", err)
		}

		fmt.Printf("Restarting environment for %s...", app.GetName())
		err = app.Stop()
		if err != nil {
			log.Println(err)
			util.Failed("Failed to stop application.")
		}

		err = app.Start()
		if err != nil {
			log.Println(err)
			util.Failed("Failed to start application.")
		}

		fmt.Println("Waiting for the environment to become ready. This may take a couple of minutes...")
		siteURL, err := app.Wait()
		if err != nil {
			util.Failed("The environment for %s never became ready: %s", app.GetName(), err)
		}

		util.Success("Successfully restarted %s", app.GetName())
		util.Success("Your application can be reached at: %s", siteURL)
	},
}

func init() {
	RootCmd.AddCommand(LocalDevReconfigCmd)

}
