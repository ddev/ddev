package cmd

import (
	"os"
	"os/exec"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// sshDirArg allows a configurable container destination directory.
var sshDirArg string

// DdevSSHCmd represents the SSH command.
var DdevSSHCmd = &cobra.Command{
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("active", 1),
	Use:               "ssh [projectname]",
	Short:             "Starts a shell session in the container for a service. Uses web service by default.",
	Long:              `Starts a shell session in the container for a service. Uses web service by default. To start a shell session for another service, run "ddev ssh --service <service>`,
	Example: `ddev ssh
ddev ssh -s db
ddev ssh -s db -u root
ddev ssh <projectname>
ddev ssh -d /var/www/html`,
	Args: cobra.MaximumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		projects, err := getRequestedProjects(args, false)
		if err != nil || len(projects) == 0 {
			util.Failed("Failed to ddev ssh: %v", err)
		}
		app := projects[0]
		instrumentationApp = app

		_ = app.DockerEnv()

		// Use Bash for our containers, sh for 3rd-party containers
		// that may not have Bash.
		shell := app.GetXDdevExtension(serviceType).SSHShell

		err = app.ExecWithTty(&ddevapp.ExecOpts{
			Service: serviceType,
			Cmd:     shell + " -l",
			Dir:     sshDirArg,
			User:    serviceUser,
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
	_ = DdevSSHCmd.RegisterFlagCompletionFunc("service", ddevapp.GetServiceNamesFunc(true))
	DdevSSHCmd.Flags().StringVarP(&sshDirArg, "dir", "d", "", "Defines the destination directory within the container")
	DdevSSHCmd.Flags().StringVarP(&serviceUser, "user", "u", "", "Defines the user to use within the container")
	RootCmd.AddCommand(DdevSSHCmd)
}
