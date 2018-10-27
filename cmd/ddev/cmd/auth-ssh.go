package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"os/user"
	"path/filepath"
)

// SSHKeyPath is the full path to the *directory* containing ssh keys.
var SSHKeyPath string

// AuthSSHCommand implements the "ddev auth ssh" command
var AuthSSHCommand = &cobra.Command{
	Use:   "ssh",
	Short: "Add ssh key authentication to the ddev-ssh-auth container",
	Long:  `Use this command to provide the password to your ssh key to the ddev-ssh-agent container, where it can be used by other containers. Normal usage is just "ddev auth ssh", or if your key is not in ~/.ssh, ddev auth ssh --keydir=/some/path/.ssh"`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) > 1 {
			util.Failed("This command only takes one optional argument: ssh-key-path")
		}

		if len(args) == 1 {
			SSHKeyPath = args[0]
		}
		curUser, err := user.Current()
		if err != nil {
			util.Failed("user.Current() failed: %v", err)
		}

		if SSHKeyPath == "" {
			homeDir, err := homedir.Dir()
			if err != nil {
				util.Failed("Unable to determine home directory: %v", err)
			}
			SSHKeyPath = filepath.Join(homeDir, ".ssh")
		}

		err = exec.RunInteractiveCommand("docker", []string{"run", "-it", "--rm", "--volumes-from=" + ddevapp.SSHAuthName, "-v", SSHKeyPath + ":/tmp/.ssh", "-u", curUser.Uid, version.SSHAuthImage + ":" + version.SSHAuthTag, "ssh-add"})

		if err != nil {
			util.Failed("Docker command failed: %v", err)
		}
	},
}

func init() {
	AuthSSHCommand.Flags().StringVarP(&SSHKeyPath, "ssh-key-path", "d", "", "full path to ssh key directory")

	AuthCmd.AddCommand(AuthSSHCommand)
}
