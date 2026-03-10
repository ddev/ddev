package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"

	dockerImages "github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
)

// globalCommandsHashFile is the name of the file that stores the fingerprint
// of the global commands directory to avoid unnecessary re-copying.
const globalCommandsHashFile = ".global-commands-hash"

// PopulateGlobalCustomCommandFiles sets up the custom command files in the project
// directories where they need to go.
func PopulateGlobalCustomCommandFiles() error {
	sourceGlobalCommandPath := filepath.Join(globalconfig.GetGlobalDdevDir(), "commands")
	err := os.MkdirAll(sourceGlobalCommandPath, 0755)
	if err != nil {
		return nil
	}

	// Check if the source directory has changed since last copy.
	// If not, skip the expensive container operations.
	currentHash, err := fileutil.HashDir(sourceGlobalCommandPath)
	if err != nil {
		util.Warning("unable to hash global commands directory: %v", err)
	}
	hashFilePath := filepath.Join(globalconfig.GetGlobalDdevDir(), globalCommandsHashFile)
	if savedHash, err := os.ReadFile(hashFilePath); err == nil && string(savedHash) == currentHash {
		util.Debug("PopulateGlobalCustomCommandFiles: skipping, global commands unchanged")
		return nil
	}

	commandDirInVolume := "/mnt/ddev-global-cache/global-commands/"

	// Use CopyIntoVolume with destroyExisting=true to combine the rm + copy
	// into a single container operation (previously these were separate).
	uid, _, _ := dockerutil.GetContainerUser()
	err = dockerutil.CopyIntoVolume(sourceGlobalCommandPath, "ddev-global-cache", "global-commands", uid, "host", true)
	if err != nil {
		return err
	}

	// Make sure all commands can be executed
	_, stderr, err := performTaskInContainer([]string{"sh", "-c", "chmod -R u+rwx " + commandDirInVolume})
	if err != nil {
		return fmt.Errorf("unable to chmod %s: %v (stderr=%s)", commandDirInVolume, err, stderr)
	}

	// Save the fingerprint so we can skip next time if unchanged
	_ = os.WriteFile(hashFilePath, []byte(currentHash), 0644)

	return nil
}

// performTaskInContainer runs a command in the web container if it's available,
// but uses an anonymous container otherwise.
func performTaskInContainer(command []string) (string, string, error) {
	app, err := GetActiveApp("")
	if err == nil {
		status, _ := app.SiteStatus()
		if status == SiteRunning {
			// Prepare docker exec command
			opts := &ExecOpts{
				RawCmd:    command,
				Tty:       false,
				NoCapture: false,
			}
			return app.Exec(opts)
		}
	}

	// If there is no running active site, use an anonymous container instead.
	containerName := "performTaskInContainer" + nodeps.RandomString(12)
	uid, _, _ := dockerutil.GetContainerUser()
	return dockerutil.RunSimpleContainer(dockerImages.GetWebImage(), containerName, command, nil, nil, []string{"ddev-global-cache:/mnt/ddev-global-cache"}, uid, true, false, map[string]string{"com.ddev.site-name": ""}, nil, &dockerutil.NoHealthCheck)
}
