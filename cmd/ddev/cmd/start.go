package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/spf13/cobra"
)

const netName = "ddev_default"

var (
	serviceType string
	webImage    string
	dbImage     string
	webImageTag string
	dbImageTag  string
)

// StartCmd represents the add command
var StartCmd = &cobra.Command{
	Use:     "start",
	Aliases: []string{"add"},
	Short:   "Start the local development environment for a site.",
	Long:    `Start initializes and configures the web server and database containers to provide a working environment for development.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			cmd.Usage()
			os.Exit(0)
		}

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
			log.Fatalf("Could not find an active ddev configuration, have you run 'ddev config'?: %v", err)
		}

		fmt.Printf("Starting environment for %s...\n", app.GetName())

		err = app.Start()
		if err != nil {
			Failed("Failed to start %s: %s", app.GetName(), err)
		}

		fmt.Println("Waiting for the environment to become ready. This may take a couple of minutes...")
		siteURL, err := app.Wait()
		if err != nil {
			Failed("The environment for %s never became ready: %s", app.GetName(), err)
		}

		Success("Successfully started %s", app.GetName())
		Success("Your application can be reached at: %s", siteURL)

	},
}

func init() {
	StartCmd.Flags().StringVarP(&webImage, "web-image", "", "", "Change the image used for the app's web server")
	StartCmd.Flags().StringVarP(&dbImage, "db-image", "", "", "Change the image used for the app's database server")
	StartCmd.Flags().StringVarP(&webImageTag, "web-image-tag", "", "", "Override the default web image tag")
	StartCmd.Flags().StringVarP(&dbImageTag, "db-image-tag", "", "", "Override the default web image tag")

	RootCmd.AddCommand(StartCmd)
}
