package cmd

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/drud/drud-go/drudapi"
	"github.com/spf13/cobra"
)

// localSSHCmd represents the ssh command.
var localSSHCmd = &cobra.Command{
	Use:   "ssh",
	Short: "SSH to an app container.",
	Long:  `Connects user to the running container.`,
	Run: func(cmd *cobra.Command, args []string) {

		if cfg.ActiveApp == "" || cfg.ActiveDeploy == "" {
			log.Fatalln("Must set ActiveApp and ActiveDeploy in drud.yaml. Use config set to achieve this.")
		}

		if appClient == "" {
			appClient = cfg.Client
		}

		//cmdSplit := strings.Split(args[0], " ")

		basePath := path.Join(homedir, ".drud", appClient, cfg.ActiveApp, cfg.ActiveDeploy)
		nameContainer := fmt.Sprintf("%s-%s-%s-%s", appClient, cfg.ActiveApp, cfg.ActiveDeploy, serviceType)

		if !checkLocalRunning(nameContainer) {
			log.Fatal("App not runnign locally. Try `drud local add`.")
		}

		composeLOC := path.Join(basePath, "docker-compose.yaml")
		if _, err := os.Stat(composeLOC); os.IsNotExist(err) {
			log.Fatalln("No docker-compose yaml for this site. Try `drud local add`.")
		}

		err := drudapi.DockerCompose(
			"-f", composeLOC,
			"exec",
			nameContainer,
			"bash",
		)
		if err != nil {
			log.Fatalln(err)
		}

	},
}

func init() {
	localSSHCmd.Flags().StringVarP(&appClient, "client", "c", "", "Client name")
	localSSHCmd.Flags().StringVarP(&serviceType, "service", "s", "web", "Which service to send the command to. [web, db]")
	LocalCmd.AddCommand(localSSHCmd)
}
