package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// LocalDevStartCmd represents the start cmd
var LocalDevStartCmd = &cobra.Command{
	Use:   "start [app_name] [environment_name]",
	Short: "Start an application's local services.",
	Long:  `Start will turn on the local containers that were previously stopped for an app.`,
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
		}
		app.SetOpts(opts)

		err := app.Start()
		if err != nil {
			log.Println(err)
			Failed("Failed to start site.")
		}

		fmt.Println("Waiting for site readiness. This may take a couple minutes...")

		siteURL, err := app.Wait()
		if err != nil {
			log.Println(err)
			Failed("Site failed to achieve readiness.")
		}

		color.Cyan("Successfully started %s %s", activeApp, activeDeploy)
		if siteURL != "" {
			color.Cyan("Your application can be reached at: %s", siteURL)
		}

	},
}

func init() {
	RootCmd.AddCommand(LocalDevStartCmd)
}
