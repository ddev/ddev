package manifest

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/manifest/internal"
	"github.com/ddev/ddev/pkg/manifest/storage"
	"github.com/ddev/ddev/pkg/manifest/types"
	"github.com/ddev/ddev/pkg/util"
)

func New(configPath string, isInternetActive bool, updateInterval time.Duration, disableTips bool) types.Manifest {
	manifest := &Manifest{
		fileStorage: storage.NewFileStorage(getLocalFileName(configPath)),
		// TODO change to ddev repo before merge
		githubStorage: storage.NewGithubStorage(
			globalconfig.DdevGlobalConfig.Manifest.SourceOwner,
			globalconfig.DdevGlobalConfig.Manifest.SourceRepo,
			`manifest.json`,
			storage.Options{Ref: globalconfig.DdevGlobalConfig.Manifest.SourceRef},
		),
		isInternetActive: isInternetActive,
		updateInterval:   updateInterval,
		tipsDisabled:     disableTips,
	}
	manifest.loadFromLocalStorage()

	return manifest
}

var (
	manifest types.Manifest
)

// GetManifest returns a global stored manifest. This is for the time being to
// avoid import cycles.
// TODO inject to a global interface e.g. command factory as soon as it exists.
func GetManifest() types.Manifest {
	if manifest == nil {
		// Load manifest
		updateInterval := globalconfig.DdevGlobalConfig.Manifest.UpdateInterval
		if updateInterval <= 0 {
			updateInterval = 24
		}

		manifest = New(
			globalconfig.GetGlobalConfigPath(),
			globalconfig.IsInternetActive(),
			time.Duration(updateInterval)*time.Hour,
			globalconfig.DdevGlobalConfig.Manifest.DisableTips,
		)
	}

	return manifest
}

const localFileName = ".manifest"

// Manifest is a in memory representation of the DDEV manifest file.
type Manifest struct {
	manifest internal.Manifest

	fileStorage   types.ManifestStorage
	githubStorage types.ManifestStorage

	isInternetActive bool
	updateInterval   time.Duration
	tipsDisabled     bool

	mu sync.RWMutex
}

func (m *Manifest) write() {
	defer util.TimeTrack(time.Now(), "write")()

	m.mu.Lock()
	defer m.mu.Unlock()

	err := m.fileStorage.Push(m.manifest)

	if err != nil {
		util.Error("Error while writing manifest: %s", err)
	}
}

func (m *Manifest) loadFromLocalStorage() {
	defer util.TimeTrack(time.Now(), "loadFromLocalStorage")()

	m.mu.Lock()
	defer func() {
		m.mu.Unlock()
		go m.updateFromGithub()
	}()

	var err error

	m.manifest, err = m.fileStorage.Pull()

	if err != nil {
		util.Error("Error while loading manifest: %s", err)
	}
}

func (m *Manifest) updateFromGithub() {
	defer util.TimeTrack(time.Now(), "updateFromGithub")()

	if !m.isInternetActive {
		util.Debug("No internet connection.")

		return
	}

	// Check if an update is needed.
	if m.fileStorage.LastUpdate().Add(m.updateInterval).Before(time.Now()) {
		util.Debug("Downloading manifest.")

		m.mu.Lock()
		backupLast := m.manifest.Messages.Tips.Last

		defer func() {
			m.manifest.Messages.Tips.Last = backupLast
			m.mu.Unlock()
			m.write()
		}()

		// Download the manifest.
		var err error
		m.manifest, err = m.githubStorage.Pull()

		if err != nil {
			util.Error("Error while downloading manifest: %s", err)
		}
	}
}

// getLocalFileName returns the filename of the local storage.
func getLocalFileName(configPath string) string {
	return filepath.Join(configPath, localFileName)
}
