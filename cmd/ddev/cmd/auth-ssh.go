package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

// sshKeyPath is the full path to the *directory* containing ssh keys.
var sshKeyPath string

// AuthSSHCommand implements the "ddev auth ssh" command
var AuthSSHCommand = &cobra.Command{
	Use:     "ssh",
	Short:   "Add ssh key authentication to the ddev-ssh-auth container",
	Long:    `Use this command to provide the password to your ssh key to the ddev-ssh-agent container, where it can be used by other containers. Normal usage is just "ddev auth ssh", or if your key is not in ~/.ssh, ddev auth ssh --ssh-key-path=/some/path/.ssh"`,
	Example: `ddev auth ssh`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		if len(args) > 0 {
			util.Failed("This command takes no arguments.")
		}

		uidStr, _, _ := util.GetContainerUIDGid()

		if sshKeyPath == "" {
			homeDir, err := homedir.Dir()
			if err != nil {
				util.Failed("Unable to determine home directory: %v", err)
			}
			sshKeyPath = filepath.Join(homeDir, ".ssh")
		}
		if !filepath.IsAbs(sshKeyPath) {
			sshKeyPath, err = filepath.Abs(sshKeyPath)
			if err != nil {
				util.Failed("Failed to derive absolute path for ssh key path %s: %v", sshKeyPath, err)
			}
		}
		fi, err := os.Stat(sshKeyPath)
		if os.IsNotExist(err) {
			util.Failed("The ssh key directory %s was not found", sshKeyPath)
		}
		if err != nil {
			util.Failed("Failed to check status of ssh key directory %s: %v", sshKeyPath, err)
		}
		if !fi.IsDir() {
			util.Failed("The ssh key directory (%s) must be a directory", sshKeyPath)
		}

		app, err := ddevapp.GetActiveApp("")
		if err != nil || app == nil {
			// We don't actually have to start ssh-agent in a project directory, so use a dummy app.
			app = &ddevapp.DdevApp{OmitContainerGlobal: globalconfig.DdevGlobalConfig.OmitContainersGlobal}
		}
		omitted := app.GetOmittedContainers()
		if nodeps.ArrayContainsString(omitted, nodeps.DdevSSHAgentContainer) {
			util.Failed("ddev-ssh-agent is omitted in your configuration so ssh auth cannot be used")
		}

		err = app.EnsureSSHAgentContainer()
		if err != nil {
			util.Failed("Failed to start ddev-ssh-agent container: %v", err)
		}
		sshKeyPath = dockerutil.MassageWindowsHostMountpoint(sshKeyPath)

		dockerCmd := []string{"run", "-it", "--rm", "--volumes-from=" + ddevapp.SSHAuthName, "--user=" + uidStr, "--entrypoint=", "--mount=type=bind,src=" + sshKeyPath + ",dst=/tmp/sshtmp", version.SSHAuthImage + ":" + version.SSHAuthTag + "-built", "bash", "-c", `cp -r /tmp/sshtmp ~/.ssh && chmod -R go-rwx ~/.ssh && cd ~/.ssh && ssh-add $(file * | awk -F: "/private key/ { print \$1 }")`}

		err = exec.RunInteractiveCommand("docker", dockerCmd)

		if err != nil {
			util.Failed("Docker command 'docker %v' failed: %v", dockerCmd, err)
		}
	},
}

func init() {
	AuthSSHCommand.Flags().StringVarP(&sshKeyPath, "ssh-key-path", "d", "", "full path to ssh key directory")

	AuthCmd.AddCommand(AuthSSHCommand)
}
