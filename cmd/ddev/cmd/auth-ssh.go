package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/heredoc"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/spf13/cobra"
)

var sshKeyFiles, sshKeyDirs []string

// AuthSSHCommand implements the "ddev auth ssh" command
var AuthSSHCommand = &cobra.Command{
	Use:   "ssh",
	Short: "Add SSH private key authentication to the ddev-ssh-agent container",
	Long:  `Use this command to provide the password to your SSH private key to the ddev-ssh-agent container, where it can be used by other containers. The command can be executed multiple times to add more keys.`,
	Example: heredoc.DocI2S(`
		ddev auth ssh
		ddev auth ssh -d ~/custom/path/to/ssh
		ddev auth ssh -f ~/.ssh/id_ed25519 -f ~/.ssh/id_rsa
	`),
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		if len(args) > 0 {
			util.Failed("This command takes no arguments.")
		}

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
		util.Debug("Found %d file paths to process", len(files))

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
				util.Debug("Added SSH private key: %s", key)
			}
		}
		if len(keys) == 0 {
			util.Failed("No SSH private keys found in %s", strings.Join(append(sshKeyDirs, sshKeyFiles...), ", "))
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
			util.Failed("Failed to start %s container: %v", nodeps.DdevSSHAgentContainer, err)
		}
		util.Debug("%s is running", nodeps.DdevSSHAgentContainer)

		output.UserOut.Printf("Adding %d SSH private key(s)...", len(keys))

		exitCode, err := runSSHAuthContainer(keys)

		if err != nil {
			helpMessage := ""
			// Add more helpful message to the obscure error from Docker
			// Can be triggered if the key is in /tmp on macOS
			if strings.Contains(err.Error(), "bind source path does not exist") {
				helpMessage = "\n\nThe specified SSH private key path is not shared with your Docker provider."
			}
			util.Error("Failed to execute command `%s`: %v %s", cmd.CommandPath(), err, helpMessage)
			output.UserErr.Exit(exitCode)
		}
		util.Success("Successfully added %d SSH private key(s).", len(keys))
	},
}

// getSSHKeyPaths returns an array of full paths to SSH private keys
// with checks to ensure they are valid.
func getSSHKeyPaths(sshKeyPathArray []string, acceptsDirsOnly bool, acceptsFilesOnly bool) []string {
	var files []string
	for _, sshKeyPath := range sshKeyPathArray {
		sshKeyPath, _ = util.ExpandHomedir(sshKeyPath)
		if !filepath.IsAbs(sshKeyPath) {
			cwd, err := os.Getwd()
			if err != nil {
				util.Failed("Failed to get current working directory: %v", err)
			}
			fullPath, err := filepath.Abs(filepath.Join(cwd, sshKeyPath))
			if err != nil {
				util.Failed("Failed to derive absolute path for SSH private key path %s: %v", sshKeyPath, err)
			}
			sshKeyPath = fullPath
		}
		fi, err := os.Stat(sshKeyPath)
		if os.IsNotExist(err) {
			util.Failed("The SSH private key path %s was not found", sshKeyPath)
		}
		if err != nil {
			util.Failed("Failed to check status of SSH private key path %s: %v", sshKeyPath, err)
		}
		if !fi.IsDir() {
			if acceptsDirsOnly {
				util.Failed("SSH private key path %s is not a directory", sshKeyPath)
			}
			files = append(files, sshKeyPath)
		} else {
			if acceptsFilesOnly {
				util.Failed("SSH private key path %s is not a file", sshKeyPath)
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
// The full check if it's really a private key is done with grep -l '^-----BEGIN .*PRIVATE KEY-----'.
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

// getCertificateForPrivateKey returns path and name for optional OpenSSH certificate
// https://www.man7.org/linux/man-pages/man1/ssh-keygen.1.html#CERTIFICATES
// https://github.com/ddev/ddev/issues/6832
func getCertificateForPrivateKey(path string, name string) (string, string) {
	cert := path + "-cert.pub"
	if !fileutil.FileExists(cert) {
		return "", ""
	}
	certPath, err := filepath.EvalSymlinks(cert)
	if err != nil {
		util.Warning("Unable to read %s file: %v", cert, err)
		return "", ""
	}
	if !fileutil.FileIsReadable(certPath) {
		util.Warning("Unable to read %s file: file is not readable", certPath)
		return "", ""
	}
	certName := name + "-cert.pub"
	return certPath, certName
}

// GetAuthSSHCmd wraps the command to be run inside the SSH auth container.
// SSH auth container mounts the SSH keys into /tmp/sshtmp location,
// copies them to ~/.ssh (to avoid permission issues with mounted volumes),
// sets ownership and permissions, and filters the private keys from provided files.
// We have:
// "ssh-add" - adds all found private keys to ssh-agent
// "//test.expect.passphrase" - used for testing, adds a single key with passphrase
func GetAuthSSHCmd(command string) string {
	uid, gid, username := dockerutil.GetContainerUser()

	commandToRun := command
	if dockerutil.IsDockerRootless() {
		// Run command as container user, not root
		commandToRun = fmt.Sprintf("setpriv --reuid=%s --regid=%s --init-groups -- %s", uid, gid, command)
	}

	if command == "ssh-add" {
		commandToRun = fmt.Sprintf(`
for key in "${keys[@]}"; do \
  # Show which key is being added
  printf "%[1]s\n" "$key" >&2; \
  # Add the key to ssh-agent or exit immediately on failure
  %[2]s "$key" || exit $?; \
done`, util.ColorizeText("Adding key %s", "yellow"), commandToRun)
	}

	return fmt.Sprintf(`
# Copy SSH files and set proper ownership and permissions
cp -r /tmp/sshtmp /home/%[1]s/.ssh && \
chown -R %[2]s:%[3]s /home/%[1]s/.ssh && \
chmod -R go-rwx /home/%[1]s/.ssh && \
cd /home/%[1]s/.ssh && \
# Find all private key files
mapfile -t keys < <(grep -l '^-----BEGIN .*PRIVATE KEY-----' *) && \
# Verify at least one key exists
((${#keys[@]})) || { echo "No SSH private keys found." >&2; exit 1; } && \
%[4]s`, username, uid, gid, commandToRun)
}

// runSSHAuthContainer runs the SSH auth container using Docker client API
func runSSHAuthContainer(keys []string) (int, error) {
	uid, _, _ := dockerutil.GetContainerUser()
	// Run container as root to be able to change ownership of files
	if dockerutil.IsDockerRootless() {
		uid = "0"
	}

	config := &container.Config{
		Image:       docker.GetSSHAuthImage() + "-built",
		Cmd:         []string{"bash", "-c", GetAuthSSHCmd("ssh-add")},
		Entrypoint:  []string{},
		AttachStdin: true,
		User:        uid,
	}

	// Prepare mounts for Docker API
	var mounts []mount.Mount
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
		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   keyPath,
			Target:   "/tmp/sshtmp/" + filename,
			ReadOnly: true,
		})
		util.Debug("Binding SSH private key %s into container as /tmp/sshtmp/%s", keyPath, filename)
		// Mount optional OpenSSH certificate
		if certPath, certName := getCertificateForPrivateKey(keyPath, filename); certPath != "" && certName != "" {
			mounts = append(mounts, mount.Mount{
				Type:     mount.TypeBind,
				Source:   certPath,
				Target:   "/tmp/sshtmp/" + certName,
				ReadOnly: true,
			})
			util.Debug("Binding SSH certificate %s into container as /tmp/sshtmp/%s", certPath, certName)
		}
	}

	// Host configuration with volume mounts
	hostConfig := &container.HostConfig{
		Mounts:      mounts,
		VolumesFrom: []string{ddevapp.SSHAuthName},
	}

	containerName := "ddev-ssh-auth-" + util.RandString(6)
	_, _, err := dockerutil.RunSimpleContainerExtended(containerName, config, hostConfig, false, false)

	exitCode := 0
	if err != nil {
		exitCode = 1
		var code int
		n, scanErr := fmt.Sscanf(err.Error(), "container run failed with exit code %d", &code)
		if n == 1 && scanErr == nil {
			exitCode = code
		}
	}
	return exitCode, err
}

func init() {
	AuthSSHCommand.Flags().StringArrayVarP(&sshKeyFiles, "ssh-key-file", "f", nil, "path to SSH private key file, use the flag multiple times to add more keys")
	AuthSSHCommand.Flags().StringArrayVarP(&sshKeyDirs, "ssh-key-path", "d", nil, "path to directory with SSH private key(s), use the flag multiple times to add more directories")
	// While both flags work well with each other, don't allow them to be passed at the same time to make it easier to use.
	AuthSSHCommand.MarkFlagsMutuallyExclusive("ssh-key-file", "ssh-key-path")

	AuthCmd.AddCommand(AuthSSHCommand)
}
