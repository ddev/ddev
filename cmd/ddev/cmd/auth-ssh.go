package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
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

		sshKeyPath = dockerutil.MassageWindowsHostMountpoint(sshKeyPath)
		useWinPty := fileutil.IsCommandAvailable("winpty")
		dockerCmd := fmt.Sprintf("docker run -it --rm --volumes-from=%s --mount 'type=bind,src=%s,dst=/tmp/.ssh' -u %s %s:%s ssh-add", ddevapp.SSHAuthName, sshKeyPath, uidStr, version.SSHAuthImage, version.SSHAuthTag)
		if useWinPty {
			dockerCmd = "winpty" + dockerCmd
		}
		err := exec.RunInteractiveCommand("sh", []string{"-c", dockerCmd})

		if err != nil {
			util.Failed("Docker command '%s' failed: %v", dockerCmd, err)
		}
	},
}

func init() {
	AuthSSHCommand.Flags().StringVarP(&sshKeyPath, "ssh-key-path", "d", "", "full path to ssh key directory")

	AuthCmd.AddCommand(AuthSSHCommand)
}
