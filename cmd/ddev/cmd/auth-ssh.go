package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
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
	Use:   "ssh",
	Short: "Add ssh key authentication to the ddev-ssh-auth container",
	Long:  `Use this command to provide the password to your ssh key to the ddev-ssh-agent container, where it can be used by other containers. Normal usage is just "ddev auth ssh", or if your key is not in ~/.ssh, ddev auth ssh --keydir=/some/path/.ssh"`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		if len(args) > 0 {
			util.Failed("This command takes no arguments.")
		}

		_, _, uidStr, _ := util.GetContainerUIDGid()

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

		// We don't actually have to start ssh-agent in a project directory, so use a dummy app.
		app := ddevapp.DdevApp{}
		err = app.EnsureSSHAgentContainer()
		if err != nil {
			util.Failed("Failed to start ddev-ssh-agent container: %v", err)
		}
		sshKeyPath = dockerutil.MassageWindowsHostMountpoint(sshKeyPath)
		useWinPty := util.IsCommandAvailable("winpty")
		dockerCmd := fmt.Sprintf("docker run -it --rm --volumes-from=%s --mount 'type=bind,src=%s,dst=/tmp/.ssh' -u %s %s:%s ssh-add", ddevapp.SSHAuthName, sshKeyPath, uidStr, version.SSHAuthImage, version.SSHAuthTag)
		if useWinPty {
			dockerCmd = "winpty" + dockerCmd
		}
		err = exec.RunInteractiveCommand("sh", []string{"-c", dockerCmd})

		if err != nil {
			util.Failed("Docker command '%s' failed: %v", dockerCmd, err)
		}
	},
}

func init() {
	AuthSSHCommand.Flags().StringVarP(&sshKeyPath, "ssh-key-path", "d", "", "full path to ssh key directory")

	AuthCmd.AddCommand(AuthSSHCommand)
}
