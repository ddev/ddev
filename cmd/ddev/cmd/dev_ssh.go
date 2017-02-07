package cmd

import (
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/drud/ddev/pkg/local"
	"github.com/drud/drud-go/utils"
	"github.com/spf13/cobra"
)

// LocalDevSSHCmd represents the ssh command.
var LocalDevSSHCmd = &cobra.Command{
	Use:   "ssh [app_name] [environment_name]",
	Short: "SSH to an app container.",
	Long:  `Connects user to the running container.`,
	Run: func(cmd *cobra.Command, args []string) {
		app := local.PluginMap[strings.ToLower(plugin)]

		opts := local.AppOptions{
			Name:        activeApp,
			Environment: activeDeploy,
		}
		app.SetOpts(opts)

		nameContainer := fmt.Sprintf("%s-%s", app.ContainerName(), serviceType)
		if !utils.IsRunning(nameContainer) {
			Failed("App not running locally. Try `drud legacy add`.")
		}

		if !local.ComposeFileExists(app) {
			Failed("No docker-compose yaml for this site. Try `drud legacy add`.")
		}

		err := utils.DockerCompose(
			"-f", path.Join(app.AbsPath(), "docker-compose.yaml"),
			"exec",
			nameContainer,
			"bash",
		)
		if err != nil {
			log.Println(err)
			Failed("Failed to run exec command.")
		}

	},
}

func init() {
	LocalDevSSHCmd.Flags().StringVarP(&serviceType, "service", "s", "web", "Which service to send the command to. [web, db]")
	RootCmd.AddCommand(LocalDevSSHCmd)
}
