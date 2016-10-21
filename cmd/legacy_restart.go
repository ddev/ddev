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
	Run: func(cmd *cobra.Command, args []string) {
		if activeApp == "" {
			log.Fatalln("Must set app flag to dentoe which app you want to work with.")
		}

		app := local.LegacyApp{
			Name:        activeApp,
			Environment: activeDeploy,
			Template:    local.LegacyComposeTemplate,
		}

		err := app.Stop()
		if err != nil {
			log.Fatalln(err)
		}

		err = app.Start()
		if err != nil {
			log.Fatalln(err)
		}

		err = app.Config()
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Println("Waiting for site readiness. This may take a couple minutes...")
		siteURL, err := app.Wait()
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Println("Successfully restarted", activeApp, activeDeploy)
		if siteURL != "" {
			fmt.Println("Your application can be reached at:", siteURL)
		}
	},
}

func init() {

	LegacyCmd.AddCommand(LegacyReconfigCmd)

}
