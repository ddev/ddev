package cmd

import (
	"os"
	"path/filepath"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/spf13/cobra"
)

// sshKeyPath is the full path to the *directory* containing SSH keys.
var sshKeyPath string

// AuthSSHCommand implements the "ddev auth ssh" command
var AuthSSHCommand = &cobra.Command{
	Use:     "ssh",
	Short:   "Add SSH key authentication to the ddev-ssh-agent container",
	Long:    `Use this command to provide the password to your SSH key to the ddev-ssh-agent container, where it can be used by other containers. Normal usage is "ddev auth ssh", or if your key is not in ~/.ssh, ddev auth ssh --ssh-key-path=/some/path/.ssh"`,
	Example: `ddev auth ssh`,
	Run: func(_ *cobra.Command, args []string) {
		var err error
		if len(args) > 0 {
			util.Failed("This command takes no arguments.")
		}

		uidStr, _, _ := util.GetContainerUIDGid()

		if sshKeyPath == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				util.Failed("Unable to determine home directory: %v", err)
			}
			sshKeyPath = filepath.Join(homeDir, ".ssh")
		}
		if !filepath.IsAbs(sshKeyPath) {
			sshKeyPath, err = filepath.Abs(sshKeyPath)
			if err != nil {
				util.Failed("Failed to derive absolute path for SSH key path %s: %v", sshKeyPath, err)
			}
		}
		fi, err := os.Stat(sshKeyPath)
		if os.IsNotExist(err) {
			util.Failed("The SSH key path %s was not found", sshKeyPath)
		}
		if err != nil {
			util.Failed("Failed to check status of SSH key path %s: %v", sshKeyPath, err)
		}

		var paths []string
		var files []string
		if !fi.IsDir() {
			files = append(files, sshKeyPath)
		} else {
			files, err = fileutil.ListFilesInDirFullPath(sshKeyPath, true)
			if err != nil {
				util.Failed("Failed to list files in %s: %v", sshKeyPath, err)
			}
		}
		// Get real paths to key files in case they are symlinks
		for _, file := range files {
			realPath, err := filepath.EvalSymlinks(file)
			if err != nil {
				util.Failed("Error resolving symlinks for %s: %v", file, err)
			}
			paths = append(paths, realPath)
		}

		app, err := ddevapp.GetActiveApp("")
		if err != nil || app == nil {
			// We don't actually have to start ssh-agent in a project directory, so use a dummy app.
			app = &ddevapp.DdevApp{OmitContainersGlobal: globalconfig.DdevGlobalConfig.OmitContainersGlobal}
		}
		omitted := app.GetOmittedContainers()
		if nodeps.ArrayContainsString(omitted, nodeps.DdevSSHAgentContainer) {
			util.Failed("ddev-ssh-agent is omitted in your configuration so ssh auth cannot be used")
		}

		err = app.EnsureSSHAgentContainer()
		if err != nil {
			util.Failed("Failed to start ddev-ssh-agent container: %v", err)
		}

		var mounts []string
		for _, keyPath := range paths {
			filename := filepath.Base(keyPath)
			keyPath = util.WindowsPathToCygwinPath(keyPath)
			mounts = append(mounts, "--mount=type=bind,src="+keyPath+",dst=/tmp/sshtmp/"+filename)
		}

		dockerCmd := []string{"run", "-it", "--rm", "--volumes-from=" + ddevapp.SSHAuthName, "--user=" + uidStr, "--entrypoint="}
		dockerCmd = append(dockerCmd, mounts...)
		dockerCmd = append(dockerCmd, versionconstants.SSHAuthImage+":"+versionconstants.SSHAuthTag+"-built", "bash", "-c", `cp -r /tmp/sshtmp ~/.ssh && chmod -R go-rwx ~/.ssh && cd ~/.ssh && grep -l '^-----BEGIN .* PRIVATE KEY-----' * | xargs -d '\n' ssh-add`)

		err = exec.RunInteractiveCommand("docker", dockerCmd)

		if err != nil {
			util.Failed("Docker command 'docker %v' failed: %v", dockerCmd, err)
		}
	},
}

func init() {
	AuthSSHCommand.Flags().StringVarP(&sshKeyPath, "ssh-key-path", "d", "", "full path to SSH key directory or file")

	AuthCmd.AddCommand(AuthSSHCommand)
}
