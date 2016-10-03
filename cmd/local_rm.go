package cmd

import (
	"log"
	"path"

	drudutils "github.com/drud/drud-go/utils"
	"github.com/spf13/cobra"
)

// localStopCmd represents the stop command
var localRMCmd = &cobra.Command{
	Use:   "rm",
	Short: "Remove an application's local services.",
	Long:  `Remove will delete the local service containers from this machine..`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			log.Fatalln("app_name and deploy_name are expected as arguments.")
		}

		if appClient == "" {
			appClient = cfg.Client
		}

		basePath := path.Join(homedir, ".drud", appClient, args[0], args[1])
		err := drudutils.DockerCompose("-f", path.Join(basePath, "docker-compose.yaml"), "down")
		if err != nil {
			log.Fatalln(err)
		}

	},
}

func init() {
	localRMCmd.Flags().StringVarP(&appClient, "client", "c", "", "Client name")
	LocalCmd.AddCommand(localRMCmd)

}
