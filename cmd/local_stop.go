package cmd

import (
	"log"
	"path"

	drudutils "github.com/drud/drud-go/utils"
	"github.com/spf13/cobra"
)

// localStopCmd represents the stop command
var localStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop an application's local services.",
	Long:  `Stop will turn off the local containers and not remove them.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			log.Fatalln("app_name and deploy_name are expected as arguments.")
		}

		if appClient == "" {
			appClient = cfg.Client
		}

		basePath := path.Join(homedir, ".drud", appClient, args[0], args[1])
		err := drudutils.DockerCompose(
			"-f", path.Join(basePath, "docker-compose.yaml"),
			"stop",
		)

		if err != nil {
			log.Fatalln(err)
		}

	},
}

func init() {
	localStopCmd.Flags().StringVarP(&appClient, "client", "c", "", "Client name")
	LocalCmd.AddCommand(localStopCmd)

}
