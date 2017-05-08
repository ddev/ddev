package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/drud/ddev/pkg/util"
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
			err := cmd.Usage()
			util.CheckErr(err)
			os.Exit(0)
		}

		client := util.GetDockerClient()

		err := util.EnsureNetwork(client, netName)
		if err != nil {
			log.Fatalf("Unable to create/ensure docker network %s, error: %v", netName, err)
		}

	},
	Run: func(cmd *cobra.Command, args []string) {
		app, err := getActiveApp()
		if err != nil {
			util.Failed("Failed to start: %s", err)
		}

		fmt.Printf("Starting environment for %s...\n", app.GetName())

		err = app.Start()
		if err != nil {
			util.Failed("Failed to start %s: %v", app.GetName(), err)
		}

		fmt.Println("Waiting for the environment to become ready. This may take a couple of minutes...")
		err = app.Wait("web")
		if err != nil {
			util.Failed("The environment for %s never became ready: %v", app.GetName(), err)
		}

		util.Success("Successfully started %s", app.GetName())
		util.Success("Your application can be reached at: %s", app.URL())

	},
}

func init() {
	StartCmd.Flags().StringVarP(&webImage, "web-image", "", "", "Change the image used for the app's web server")
	StartCmd.Flags().StringVarP(&dbImage, "db-image", "", "", "Change the image used for the app's database server")
	StartCmd.Flags().StringVarP(&webImageTag, "web-image-tag", "", "", "Override the default web image tag")
	StartCmd.Flags().StringVarP(&dbImageTag, "db-image-tag", "", "", "Override the default web image tag")

	RootCmd.AddCommand(StartCmd)
}
