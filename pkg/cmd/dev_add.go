package cmd

import (
	"log"
	"strings"

	"github.com/drud/ddev/local"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const netName = "drud_default"

var (
	scaffold    bool
	serviceType string
	webImage    string
	dbImage     string
	webImageTag string
	dbImageTag  string
	skipYAML    bool
	appClient   string
)

// DevAddCmd represents the add command
var DevAddCmd = &cobra.Command{
	Use:   "add [app_name] [environment_name]",
	Short: "Add an existing application to your local development environment",
	Long: `Add an existing application to your local dev environment.  This involves
	downloading of containers, media, and databases.`,
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

		app := local.PluginMap[strings.ToLower(plugin)]

		opts := local.AppOptions{
			Name:        activeApp,
			Environment: activeDeploy,
			Client:      appClient,
			WebImage:    webImage,
			WebImageTag: webImageTag,
			DbImage:     dbImage,
			DbImageTag:  dbImageTag,
			SkipYAML:    skipYAML,
			CFG:         cfg,
		}

		app.Init(opts)

		err := app.GetResources()
		if err != nil {
			log.Println(err)
			Failed("Failed to gather resources.")
		}

		err = app.UnpackResources()
		if err != nil {
			log.Println(err)
			Failed("Failed to unpack application resources.")
		}

		siteURL := ""
		if !scaffold {
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

			siteURL, err = app.Wait()
			if err != nil {
				log.Println(err)
				Failed("Site never became ready")
			}
		}

		color.Cyan("Successfully added %s-%s", activeApp, activeDeploy)
		if siteURL != "" {
			color.Cyan("Your application can be reached at: %s", siteURL)
		}

	},
}

func init() {
	DevAddCmd.Flags().BoolVarP(&scaffold, "scaffold", "s", false, "Add the app but don't run or config it.")
	DevAddCmd.Flags().StringVarP(&webImage, "web-image", "", "", "Change the image used for the app's web server")
	DevAddCmd.Flags().StringVarP(&dbImage, "db-image", "", "", "Change the image used for the app's database server")
	DevAddCmd.Flags().StringVarP(&webImageTag, "web-image-tag", "", "", "Override the default web image tag")
	DevAddCmd.Flags().StringVarP(&dbImageTag, "db-image-tag", "", "", "Override the default web image tag")
	DevAddCmd.Flags().StringVarP(&plugin, "plugin", "p", "legacy", "Choose which plugin to use")
	DevAddCmd.Flags().BoolVarP(&skipYAML, "skip-yaml", "", false, "Skip creating the docker-compose.yaml.")
	DevAddCmd.Flags().StringVarP(&appClient, "client", "c", "", "Client name")

	RootCmd.AddCommand(DevAddCmd)
}
