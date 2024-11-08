package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	dockerContainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
)

// sshKeyPath is the full path to the path containing SSH keys.
var sshKeyPath string

// AuthSSHCommand implements the "ddev auth ssh" command
var AuthSSHCommand = &cobra.Command{
	Use:   "ssh",
	Short: "Add SSH key authentication to the ddev-ssh-agent container",
	Long:  `Use this command to provide the password to your SSH key to the ddev-ssh-agent container, where it can be used by other containers. Normal usage is "ddev auth ssh", or if your key is not in ~/.ssh, ddev auth ssh --ssh-key-path=/some/path/.ssh"`,
	Example: `ddev auth ssh
ddev auth ssh -d ~/.ssh
ddev auth ssh -f ~/.ssh/id_ed25519`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		if len(args) > 0 {
			util.Failed("This command takes no arguments.")
		}

		if cmd.Flags().Changed("ssh-key-path") {
			sshKeyPath = cmd.Flag("ssh-key-path").Value.String()
		} else if cmd.Flags().Changed("ssh-key-file") {
			sshKeyPath = cmd.Flag("ssh-key-file").Value.String()
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
			if cmd.Flags().Changed("ssh-key-path") {
				util.Failed("SSH key path %s is not a directory", sshKeyPath)
			}
			files = append(files, sshKeyPath)
		} else {
			if cmd.Flags().Changed("ssh-key-file") {
				util.Failed("SSH key path %s is not a file", sshKeyPath)
			}
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

		var mounts []mount.Mount
		for _, keyPath := range paths {
			mount := mount.Mount{
				Type:     mount.TypeBind,
				Source:   keyPath,
				Target:   "/tmp/sshtmp/" + filepath.Base(keyPath),
				ReadOnly: true,
			}
			mounts = append(mounts, mount)
		}
		sshAddCmd := []string{"bash", "-c", `cp -r /tmp/sshtmp ~/.ssh && chmod -R go-rwx ~/.ssh && cd ~/.ssh && key_files=$(grep -l '^-----BEGIN .* PRIVATE KEY-----' *); if [ -z "$key_files" ]; then echo 'No private keys found.' >&2; exit 1; else echo $key_files | xargs -d '\n' ssh-add; fi`}
		config := &dockerContainer.Config{
			Entrypoint: []string{},
		}
		hostConfig := &dockerContainer.HostConfig{
			Mounts:      mounts,
			VolumesFrom: []string{ddevapp.SSHAuthName},
		}
		_, out, err := dockerutil.RunSimpleContainerExtended(docker.GetSSHAuthImage()+"-built", "auth-ssh-"+util.RandString(6), sshAddCmd, uidStr, true, false, config, hostConfig)
		out = strings.TrimSpace(out)
		if err != nil {
			util.Failed("Unable to run auth ssh: %v; output='%s'", err, out)
		}
		output.UserOut.Println(out)
	},
}

func init() {
	AuthSSHCommand.Flags().StringP("ssh-key-file", "f", "", "full path to SSH key file")
	AuthSSHCommand.Flags().StringP("ssh-key-path", "d", "", "full path to SSH key directory")
	AuthSSHCommand.MarkFlagsMutuallyExclusive("ssh-key-file", "ssh-key-path")

	AuthCmd.AddCommand(AuthSSHCommand)
}
