package cmd

import (
	"fmt"

	"github.com/drud/ddev/pkg/util"
	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/spf13/cobra"
)

// LocalDevSSHCmd represents the ssh command.
var LocalDevSSHCmd = &cobra.Command{
	Use:   "ssh",
	Short: "SSH to an app container.",
	Long:  `Connects user to the running container.`,
	Run: func(cmd *cobra.Command, args []string) {
		app, err := getActiveApp()
		if err != nil {
			util.Failed("Failed to ssh %s: %s", app.GetName(), err)
		}

		nameContainer := fmt.Sprintf("%s-%s", app.ContainerName(), serviceType)
		if !dockerutil.IsRunning(nameContainer) {
			util.Failed("App not running locally. Try `ddev start`.")
		}
		app.DockerEnv()
		err = dockerutil.DockerCompose(
			"-f", app.DockerComposeYAMLPath(),
			"exec",
			nameContainer,
			"bash",
		)
		if err != nil {
			util.Failed("Failed to ssh %s: %s", app.GetName(), err)
		}

	},
}

func init() {
	LocalDevSSHCmd.Flags().StringVarP(&serviceType, "service", "s", "web", "Which service to send the command to. [web, db]")
	RootCmd.AddCommand(LocalDevSSHCmd)
}
