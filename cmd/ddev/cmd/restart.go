package cmd

import (
	"fmt"
	"log"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/spf13/cobra"
)

// LocalDevReconfigCmd rebuilds an apps settings
var LocalDevReconfigCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the local development environment for a site.",
	Long:  `Restart stops the containers for site's environment and starts them back up again.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		client, err := platform.GetDockerClient()
		if err != nil {
			log.Fatal(err)
		}

		err = EnsureNetwork(client, netName)
		if err != nil {
			log.Fatal(err)
		}

	},
	Run: func(cmd *cobra.Command, args []string) {
		app, err := getActiveApp()
		if err != nil {
			log.Fatalf("Could not find an active ddev configuration, have you ran 'ddev config'?: %v", err)
		}

		fmt.Printf("Restarting environment for %s...", app.GetName())
		err = app.Stop()
		if err != nil {
			log.Println(err)
			Failed("Failed to stop application.")
		}

		err = app.Start()
		if err != nil {
			log.Println(err)
			Failed("Failed to start application.")
		}

		fmt.Println("Waiting for the environment to become ready. This may take a couple of minutes...")
		siteURL, err := app.Wait()
		if err != nil {
			Failed("The environment for %s never became ready: %s", app.GetName(), err)
		}

		Success("Successfully restarted %s", app.GetName())
		Success("Your application can be reached at: %s", siteURL)
	},
}

func init() {
	RootCmd.AddCommand(LocalDevReconfigCmd)

}
