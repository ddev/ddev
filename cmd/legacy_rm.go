package cmd

import (
	"fmt"
	"log"

	"github.com/drud/bootstrap/cli/local"
	"github.com/spf13/cobra"
)

// LegacyRMCmd represents the stop command
var LegacyRMCmd = &cobra.Command{
	Use:   "rm",
	Short: "Remove an application's local services.",
	Long:  `Remove will delete the local service containers from this machine..`,
	Run: func(cmd *cobra.Command, args []string) {
		if activeApp == "" {
			log.Fatalln("Must set app flag to denote which app you want to work with.")
		}

		app := local.LegacyApp{
			Name:        activeApp,
			Environment: activeDeploy,
		}

		err := app.Down()

		if err != nil {
			log.Fatalf("Could not remove site: %s", err)
		}

		fmt.Printf("Successfully removed the %s deploy for the %s application.\n", activeDeploy, activeApp)
	},
}

func init() {
	LegacyCmd.AddCommand(LegacyRMCmd)
}
