package cmd

import (
	"fmt"
	"log"

	"github.com/drud/bootstrap/cli/local"
	"github.com/spf13/cobra"
)

// LegacyReconfigCmd rebuilds an apps settings
var LegacyReconfigCmd = &cobra.Command{
	Use:   "restart",
	Short: "Stop and Start the app.",
	Long:  `Restart is useful for when the port of your local app has changed due to a system reboot or some other failure.`,
	PreRun: func(cmd *cobra.Command, args []string) {

		client, err := local.GetDockerClient()
		if err != nil {
			log.Fatal(err)
		}

		err = EnsureNetwork(client, netName)
		if err != nil {
			log.Fatal(err)
		}

	},
	Run: func(cmd *cobra.Command, args []string) {
		if activeApp == "" {
			log.Fatalln("Must set app flag to dentoe which app you want to work with.")
		}

		app := local.NewLegacyApp(activeApp, activeDeploy)
		app.Template = local.LegacyComposeTemplate
		app.SkipYAML = skipYAML

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

		err = app.Config()
		if err != nil {
			log.Println(err)
			Failed("Failed to configure application.")
		}

		fmt.Println("Waiting for site readiness. This may take a couple minutes...")
		siteURL, err := app.Wait()
		if err != nil {
			log.Println(err)
			Failed("Site never became ready")
		}

		Success("Successfully restarted %s %s", activeApp, activeDeploy)
		if siteURL != "" {
			Success("Your application can be reached at: %s", siteURL)
		}
	},
}

func init() {
	LegacyReconfigCmd.Flags().BoolVarP(&skipYAML, "skip-yaml", "", false, "Skip creating the docker-compose.yaml.")
	LegacyCmd.AddCommand(LegacyReconfigCmd)

}
