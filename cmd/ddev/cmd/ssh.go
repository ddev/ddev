package cmd

import (
	"os"
	"os/exec"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// sshDirArg allows a configurable container destination directory.
var sshDirArg string

// DdevSSHCmd represents the ssh command.
var DdevSSHCmd = &cobra.Command{
	Use:   "ssh [projectname]",
	Short: "Starts a shell session in the container for a service. Uses web service by default.",
	Long:  `Starts a shell session in the container for a service. Uses web service by default. To start a shell session for another service, run "ddev ssh --service <service>`,
	Example: `ddev ssh
ddev ssh -s db
ddev ssh <projectname>
ddev ssh -d /var/www/html`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projects, err := getRequestedProjects(args, false)
		if err != nil || len(projects) == 0 {
			util.Failed("Failed to ddev ssh: %v", err)
		}
		app := projects[0]
		instrumentationApp = app

		app.DockerEnv()

		// Use bash for our containers, sh for 3rd-party containers
		// that may not have bash.
		shell := "bash"
		if !nodeps.ArrayContainsString([]string{"web", "db", "solr"}, serviceType) {
			shell = "sh"
		}
		err = app.ExecWithTty(&ddevapp.ExecOpts{
			Service: serviceType,
			Cmd:     shell + " -l",
			Dir:     sshDirArg,
		})
		if err != nil {
			if exiterr, ok := err.(*exec.ExitError); ok {
				os.Exit(exiterr.ExitCode())
			}
			util.Failed("ddev ssh failed: %v", err)
		}
	},
}

func init() {
	DdevSSHCmd.Flags().StringVarP(&serviceType, "service", "s", "web", "Defines the service to connect to. [e.g. web, db]")
	DdevSSHCmd.Flags().StringVarP(&sshDirArg, "dir", "d", "", "Defines the destination directory within the container")
	RootCmd.AddCommand(DdevSSHCmd)
}
