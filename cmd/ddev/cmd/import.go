package cmd

import (
	"log"
	"strings"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// ImportCmd represents the add command
var ImportCmd = &cobra.Command{
	Use:   "import [app_name] [environment_name]",
	Short: "Import an existing site to the local dev environment",
	Long:  `Import the database and file assets of an existing site into the local development environment.`,
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

		err = app.Config()
		if err != nil {
			log.Println(err)
			Failed("Failed to configure application.")
		}

		siteURL, err := app.Wait()
		if err != nil {
			log.Println(err)
			Failed("Site never became ready")
		}

		color.Cyan("Successfully imported %s-%s", activeApp, activeDeploy)
		if siteURL != "" {
			color.Cyan("Your application can be reached at: %s", siteURL)
		}

	},
}

func init() {
	RootCmd.AddCommand(ImportCmd)
}
