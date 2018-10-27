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

// AuthSSHCommand implements the "ddev auth ssh" command
var AuthSSHCommand = &cobra.Command{
	Use:   "ssh",
	Short: "Add ssh key authentication to the ddev-ssh-auth container",
	Run: func(cmd *cobra.Command, args []string) {
		//keyPath := ""

		if len(args) > 0 {
			util.Failed("This command only takes one optional argument: ssh-key-path")
		}

		//if len(args) == 1 {
		//	projectName = args[0]
		//}
		curUser, err := user.Current()
		if err != nil {
			util.Failed("user.Current() failed: %v", err)
		}
		homeDir, err := homedir.Dir()
		if err != nil {
			util.Failed("Unable to determine home directory")
		}
		sshDir := filepath.Join(homeDir, ".ssh")

		err = exec.RunInteractiveCommand("docker", []string{"run", "-it", "--rm", "--volumes-from=" + ddevapp.SSHAuthName, "-v", sshDir + ":/tmp/.ssh", "-u", curUser.Uid, version.SSHAuthImage + ":" + version.SSHAuthTag, "ssh-add", "/tmp/.ssh/id_rsa"})

		if err != nil {
			util.Failed("Docker command failed: %v", err)
		}

		//output.UserOut.Printf(strings.TrimSpace(out))
	},
}

func init() {
	AuthCmd.AddCommand(AuthSSHCommand)
}
