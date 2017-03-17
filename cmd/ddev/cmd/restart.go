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
	Short: "Stop and Start the app.",
	Long:  `Restart is useful for when the port of your local app has changed due to a system reboot or some other failure.`,
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

		fmt.Println("Waiting for site readiness. This may take a couple minutes...")
		siteURL, err := app.Wait()
		if err != nil {
			log.Println(err)
			Failed("Site never became ready")
		}

		Success("Successfully restarted %s", activeApp)
		Success("Your application can be reached at: %s", siteURL)
	},
}

func init() {
	RootCmd.AddCommand(LocalDevReconfigCmd)

}
