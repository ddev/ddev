package cmd

import (
	"fmt"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/heredoc"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var sshKeyFiles, sshKeyDirs []string

// AuthSSHCommand implements the "ddev auth ssh" command
var AuthSSHCommand = &cobra.Command{
	Use:   "ssh",
	Short: "Add SSH key authentication to the ddev-ssh-agent container",
	Long:  `Use this command to provide the password to your SSH key to the ddev-ssh-agent container, where it can be used by other containers.`,
	Example: heredoc.DocI2S(`
		ddev auth ssh
		ddev auth ssh -d ~/custom/path/to/ssh
		ddev auth ssh -f ~/.ssh/id_ed25519 -f ~/.ssh/id_rsa
		ddev auth ssh -d ~/.ssh -f ~/custom/path/to/ssh/id_ed25519
	`),
	Run: func(_ *cobra.Command, args []string) {
		var err error
		if len(args) > 0 {
			util.Failed("This command takes no arguments.")
		}

		uidStr, _, _ := util.GetContainerUIDGid()

		// Use ~/.ssh if nothing is provided
		if sshKeyFiles == nil && sshKeyDirs == nil {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				util.Failed("Unable to determine home directory: %v", err)
			}
			sshKeyDirs = append(sshKeyDirs, filepath.Join(homeDir, ".ssh"))
		}

		files := getSSHKeyPaths(sshKeyDirs, true, false)
		files = append(files, getSSHKeyPaths(sshKeyFiles, false, true)...)

		var keys []string
		// Get real paths to key files in case they are symlinks
		for _, file := range files {
			key, err := filepath.EvalSymlinks(file)
			if err != nil {
				util.Warning("Unable to read %s file: %v", file, err)
				continue
			}
			if !slices.Contains(keys, key) && fileIsPrivateKey(key) {
				keys = append(keys, key)
			}
		}
		if len(keys) == 0 {
			util.Failed("No SSH keys found in %s", strings.Join(append(sshKeyDirs, sshKeyFiles...), ", "))
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
		// Map to track already added keys
		addedKeys := make(map[string]struct{})
		for i, keyPath := range keys {
			filename := filepath.Base(keyPath)
			// If it has the same name, change it to avoid conflicts
			// This can happen if you have symlinks to the same key
			if _, exists := addedKeys[filename]; exists {
				filename = fmt.Sprintf("%s_%d", filename, i)
			}
			addedKeys[filename] = struct{}{}
			keyPath = util.WindowsPathToCygwinPath(keyPath)
			mounts = append(mounts, "--mount=type=bind,src="+keyPath+",dst=/tmp/sshtmp/"+filename)
		}

		dockerCmd := []string{"run", "-it", "--rm", "--volumes-from=" + ddevapp.SSHAuthName, "--user=" + uidStr, "--entrypoint="}
		dockerCmd = append(dockerCmd, mounts...)
		dockerCmd = append(dockerCmd, docker.GetSSHAuthImage()+"-built", "/add-keys.sh")

		err = exec.RunInteractiveCommand("docker", dockerCmd)

		if err != nil {
			helpMessage := ""
			// Add more helpful message to the obscure error from Docker
			// Can be triggered if the key is in /tmp on macOS
			if strings.Contains(err.Error(), "bind source path does not exist") {
				helpMessage = "\n\nThe specified SSH key path is not shared with your Docker provider."
			}
			util.Failed("Docker command 'docker %v' failed: %v %v", echoDockerCmd(dockerCmd), err, helpMessage)
		}
	},
}

// getSSHKeyPaths returns an array of full paths to SSH keys
// with checks to ensure they are valid.
func getSSHKeyPaths(sshKeyPathArray []string, acceptsDirsOnly bool, acceptsFilesOnly bool) []string {
	var files []string
	for _, sshKeyPath := range sshKeyPathArray {
		if !filepath.IsAbs(sshKeyPath) {
			cwd, err := os.Getwd()
			if err != nil {
				util.Failed("Failed to get current working directory: %v", err)
			}
			fullPath, err := filepath.Abs(filepath.Join(cwd, sshKeyPath))
			if err != nil {
				util.Failed("Failed to derive absolute path for SSH key path %s: %v", sshKeyPath, err)
			}
			sshKeyPath = fullPath
		}
		fi, err := os.Stat(sshKeyPath)
		if os.IsNotExist(err) {
			util.Failed("The SSH key path %s was not found", sshKeyPath)
		}
		if err != nil {
			util.Failed("Failed to check status of SSH key path %s: %v", sshKeyPath, err)
		}
		if !fi.IsDir() {
			if acceptsDirsOnly {
				util.Failed("SSH key path %s is not a directory", sshKeyPath)
			}
			files = append(files, sshKeyPath)
		} else {
			if acceptsFilesOnly {
				util.Failed("SSH key path %s is not a file", sshKeyPath)
			}
			files, err = fileutil.ListFilesInDirFullPath(sshKeyPath, true)
			if err != nil {
				util.Failed("Failed to list files in %s: %v", sshKeyPath, err)
			}
		}
	}
	return files
}

// fileIsPrivateKey checks if a file is readable and that it is a private key.
// Regex isn't used here because files can be huge.
// The full check if it's really a private key is done with grep -l '^-----BEGIN .* PRIVATE KEY-----'.
func fileIsPrivateKey(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	// nolint: errcheck
	defer file.Close()
	prefix := []byte("-----BEGIN")
	buffer := make([]byte, len(prefix))
	_, err = file.Read(buffer)
	if err != nil {
		return false
	}
	return string(buffer) == string(prefix)
}

// echoDockerCmd formats the Docker command to be more readable.
func echoDockerCmd(dockerCmd []string) string {
	for i, arg := range dockerCmd {
		if strings.Contains(arg, " ") {
			dockerCmd[i] = `"` + arg + `"`
		}
	}
	return strings.Join(dockerCmd, " ")
}

func init() {
	AuthSSHCommand.Flags().StringArrayVarP(&sshKeyFiles, "ssh-key-file", "f", nil, "full path to SSH key file")
	AuthSSHCommand.Flags().StringArrayVarP(&sshKeyDirs, "ssh-key-path", "d", nil, "full path to SSH key directory")

	AuthCmd.AddCommand(AuthSSHCommand)
}
