package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
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
	// Include the volume's creation timestamp in the cache key so that deleting
	// and recreating the volume (e.g. between CI runs) always triggers a fresh copy,
	// even when the source directory hash is unchanged.
	volCreatedAt := dockerutil.VolumeCreatedAt("ddev-global-cache")
	cacheKey := currentHash + "|" + volCreatedAt
	hashFilePath := filepath.Join(globalconfig.GetGlobalDdevDir(), globalCommandsHashFile)
	if savedKey, err := os.ReadFile(hashFilePath); err == nil && string(savedKey) == cacheKey && volCreatedAt != "" {
		util.Debug("PopulateGlobalCustomCommandFiles: skipping, global commands unchanged")
		return nil
	}

	commandDirInVolume := "/mnt/ddev-global-cache/global-commands/"

	// Use CopyIntoVolume with destroyExisting=true to combine the rm + copy
	// into a single container operation (previously these were separate).
	uid, _, _ := dockerutil.GetContainerUser()
	err = copyIntoGlobalCache(sourceGlobalCommandPath, "global-commands", uid, "host", true)
	if err != nil {
		return err
	}

	// Make sure all commands can be executed
	_, stderr, err := performTaskInContainer([]string{"sh", "-c", "chmod -R u+rwx " + commandDirInVolume})
	if err != nil {
		return fmt.Errorf("unable to chmod %s: %v (stderr=%s)", commandDirInVolume, err, stderr)
	}

	// Save the cache key so we can skip next time if unchanged
	_ = os.WriteFile(hashFilePath, []byte(cacheKey), 0644)

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
	return dockerutil.RunSimpleContainer(versionconstants.UtilitiesImage, containerName, command, nil, nil, []string{dockerutil.GlobalCacheMount()}, uid, true, false, nil, nil, nil)
}
