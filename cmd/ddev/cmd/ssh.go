package cmd

import (
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// LocalDevSSHCmd represents the ssh command.
var LocalDevSSHCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Starts a shell session in the container for a service. Uses web service by default.",
	Long:  `Starts a shell session in the container for a service. Uses web service by default. To start a shell session for another service, run "ddev ssh --service <service>`,
	Run: func(cmd *cobra.Command, args []string) {
		app, err := getActiveApp("")
		if err != nil {
			util.Failed("Failed to ssh: %v", err)
		}

		if app.SiteStatus() != "running" {
			util.Failed("App not running locally. Try `ddev start`.")
		}

		app.DockerEnv()

		err = app.Exec(serviceType, false, "bash")
		if err != nil {
			util.Failed("Failed to ssh %s: %s", app.GetName(), err)
		}

	},
}

func init() {
	LocalDevSSHCmd.Flags().StringVarP(&serviceType, "service", "s", "web", "Defines the service to connect to. [e.g. web, db]")
	RootCmd.AddCommand(LocalDevSSHCmd)
}
