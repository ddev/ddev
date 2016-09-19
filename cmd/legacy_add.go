package cmd

import (
	"log"

	"github.com/drud/bootstrap/cli/local"
	"github.com/spf13/cobra"
)

const legacyComposeTemplate = `version: '2'
services:
  {{.name}}-db:
    container_name: {{.name}}-db
    image: drud/mysql-docker-local:5.7
    volumes:
      - "./data:/db"
    restart: always
    environment:
      MYSQL_DATABASE: data
      MYSQL_ROOT_PASSWORD: root
    ports:
      - "3306"
  {{.name}}-web:
    container_name: {{.name}}-web
    image: {{.image}}
    volumes:
      - "./src:/var/www/html"
    restart: always
    depends_on:
      - {{.name}}-db
    links:
      - {{.name}}-db:db
    ports:
      - "80"
    working_dir: "/var/www/html/docroot"
    environment:
      - DEPLOY_NAME=local
`

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
			Template:    legacyComposeTemplate,
		}

		err := local.WriteLocalAppYAML(app)
		if err != nil {
			log.Fatalln(err)
		}

		err = local.CloneSource(app)
		if err != nil {
			log.Fatalln(err)
		}

	},
}

func init() {
	LegacyAddCmd.Flags().StringVarP(&appType, "type", "t", "drupal", "Type of application ('drupal' or 'wp')")
	LegacyCmd.AddCommand(LegacyAddCmd)
}
