package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/spf13/cobra"
)

// LocalDevReconfigCmd rebuilds an apps settings
var LocalDevReconfigCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the local development environment for a site.",
	Long:  `Restart stops the containers for site's environment and starts them back up again.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Restarting environment for %s...", activeApp)

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
		app := platform.PluginMap[strings.ToLower(plugin)]
		app.Init()

		err := app.Stop()
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
			Failed("The environment for %s never became ready: %s", activeApp, err)
		}

		Success("Successfully restarted %s", activeApp)
		Success("Your application can be reached at: %s", siteURL)
	},
}

func init() {
	RootCmd.AddCommand(LocalDevReconfigCmd)

}
