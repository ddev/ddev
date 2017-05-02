package cmd

import (
	"github.com/drud/ddev/pkg/util"
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
			util.Failed("Failed to ssh: %v", err)
		}

		labels := map[string]string{
			"com.ddev.site-name":      app.GetName(),
			"com.ddev.container-type": serviceType,
		}
		container, err := util.FindContainerByLabels(labels)
		nameContainer := util.ContainerName(container)

		if app.SiteStatus() != "running" {
			util.Failed("App not running locally. Try `ddev start`.")
		}

		app.DockerEnv()

		err = util.ContainerExec(nameContainer, []string{"bash"})
		if err != nil {
			util.Failed("Failed to ssh %s: %s", app.GetName(), err)
		}

	},
}

func init() {
	LocalDevSSHCmd.Flags().StringVarP(&serviceType, "service", "s", "web", "Which service to send the command to. [web, db]")
	RootCmd.AddCommand(LocalDevSSHCmd)
}
