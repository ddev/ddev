package cmd

import (
	"log"

	"github.com/drud/bootstrap/cli/local"
	"github.com/spf13/cobra"
)

var appType string

// LegacyAddCmd represents the add command
var LegacyAddCmd = &cobra.Command{
	Use:   "add [app_name] [deploy_name]",
	Short: "Add an existing application to your local development environment",
	Long: `Add an existing application to your local dev environment.  This involves
	downloading of containers, media, and databases.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			log.Fatalln("app_name and deploy_name are expected as arguments.")
		}

		app := local.LegacyApp{
			Name:        args[0],
			Environment: args[1],
			AppType:     appType,
			Template:    local.legacyComposeTemplate,
		}

		err := local.WriteLocalAppYAML(app)
		if err != nil {
			log.Fatalln(err)
		}

		err = local.CloneSource(app)
		if err != nil {
			log.Fatalln(err)
		}

		err = app.GetResources()
		if err != nil {
			log.Fatalln(err)
		}

		err = app.UnpackResources()
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

	},
}

func init() {
	LegacyAddCmd.Flags().StringVarP(&appType, "type", "t", "drupal", "Type of application ('drupal' or 'wp')")
	LegacyCmd.AddCommand(LegacyAddCmd)
}
