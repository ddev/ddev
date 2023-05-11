package manifest

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/manifest/storages"
	"github.com/ddev/ddev/pkg/manifest/types"
	"github.com/ddev/ddev/pkg/util"
)

func NewManifest(updateInterval time.Duration) *Manifest {
	manifest := &Manifest{
		fileStorage:    storages.NewFileStorage(getLocalFileName()),
		githubStorage:  storages.NewGithubStorage(`ddev`, `ddev`, `manifest.json`),
		updateInterval: updateInterval,
	}
	manifest.loadFromLocalStorage()

	return manifest
}

const localFileName = ".manifest"

type Manifest struct {
	Manifest types.Manifest

	fileStorage   types.ManifestStorage
	githubStorage types.ManifestStorage

	updateInterval time.Duration

	mu sync.RWMutex
}

func (m *Manifest) Write() {
	m.mu.Lock()
	defer m.mu.Unlock()

	err := m.fileStorage.Push(m.Manifest)

	if err != nil {
		util.Error("Error while writing manifest: %s", err)
	}
}

func (m *Manifest) loadFromLocalStorage() {
	m.mu.Lock()
	defer func() {
		m.mu.Unlock()
		go m.updateFromGithub()
	}()

	var err error

	m.Manifest, err = m.fileStorage.Pull()

	if err != nil {
		util.Error("Error while loading manifest: %s", err)
	}
}

func (m *Manifest) updateFromGithub() {
	if !globalconfig.IsInternetActive() {
		return
	}

	// Check if an update is needed.
	if m.fileStorage.LastUpdate().Add(m.updateInterval).Before(time.Now()) {
		m.mu.Lock()
		backupLast := m.Manifest.Messages.Tips.Last

		defer func() {
			m.Manifest.Messages.Tips.Last = backupLast
			m.mu.Unlock()
			m.Write()
		}()

		// Download the manifest.
		var err error
		m.Manifest, err = m.githubStorage.Pull()

		if err != nil {
			util.Error("Error while downloading manifest: %s", err)
		}
	}
}

// getLocalFileName returns the filename of the local storage.
func getLocalFileName() string {
	return filepath.Join(globalconfig.GetGlobalDdevDir(), localFileName)
}
