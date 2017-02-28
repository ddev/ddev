package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/fatih/color"
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

		opts := platform.AppOptions{
			Name:        activeApp,
			Environment: activeDeploy,
			Client:      appClient,
			SkipYAML:    skipYAML,
			CFG:         cfg,
		}
		app.Init(opts)

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

		color.Cyan("Successfully restarted %s %s", activeApp, activeDeploy)
		if siteURL != "" {
			color.Cyan("Your application can be reached at: %s", siteURL)
		}
	},
}

func init() {
	LocalDevReconfigCmd.Flags().BoolVarP(&skipYAML, "skip-yaml", "", false, "Skip creating the docker-compose.yaml.")
	RootCmd.AddCommand(LocalDevReconfigCmd)

}
