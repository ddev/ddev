package cmd

import (
	"strings"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DdevShellCmd represents the shell command.
var DdevShellCmd = &cobra.Command{
	Use:     "shell",
	Aliases: []string{"ssh"},
	Short:   "Starts a shell session in the container for a service. Uses web service by default.",
	Long:    `Starts a shell session in the container for a service. Uses web service by default. To start a shell session for another service, run "ddev shell --service <service>`,
	Run: func(cmd *cobra.Command, args []string) {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Failed to open shell: %v", err)
		}

		if strings.Contains(app.SiteStatus(), ddevapp.SiteNotFound) {
			util.Failed("Project is not currently running. Try 'ddev start'.")
		}

		if strings.Contains(app.SiteStatus(), ddevapp.SiteStopped) {
			util.Failed("Project is stopped. Run 'ddev start' to start the environment.")
		}

		app.DockerEnv()

		err = app.ExecWithTty(serviceType, "bash")

		if err != nil {
			util.Failed("Failed to open shell in %s: %s", app.GetName(), err)
		}
	},
}

func init() {
	DdevShellCmd.Flags().StringVarP(&serviceType, "service", "s", "web", "Defines the service to connect to. [e.g. web, db]")
	RootCmd.AddCommand(DdevShellCmd)
}
