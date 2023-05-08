package manifest

import (
	"path/filepath"
	"time"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/manifest/storages"
	"github.com/ddev/ddev/pkg/manifest/types"
	"github.com/ddev/ddev/pkg/util"
)

var (
	fileStorage   types.ManifestStorage
	githubStorage types.ManifestStorage
)

// GetManifest updates the manifest from the Github repo if needed and returns it.
func GetManifest(updateInterval time.Duration) (manifest types.Manifest) {
	var err error

	if fileStorage == nil {
		messageFile := filepath.Join(globalconfig.GetGlobalDdevDir(), ".manifest")
		fileStorage = storages.NewFileStorage(messageFile)
	}

	// Check if an update is needed.
	if fileStorage.LastUpdate().Add(updateInterval).Before(time.Now()) {
		if githubStorage == nil {
			githubStorage = storages.NewGithubStorage(`ddev`, `ddev`, `manifest.json`)
		}

		// Download the manifest.
		manifest, err = githubStorage.Pull()

		if err == nil {
			// Push the downloaded manifest to the local storage.
			err = fileStorage.Push(&manifest)

			if err != nil {
				util.Error("Error while writing manifest: %s", err)
			}
		} else {
			util.Error("Error while downloading manifest: %s", err)
		}
	}

	// Pull the messages to return
	manifest, err = fileStorage.Pull()

	if err != nil {
		util.Error("Error while loading manifest: %s", err)
	}

	return
}
