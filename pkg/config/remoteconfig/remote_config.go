package remoteconfig

import (
	"sync"
	"time"

	"github.com/ddev/ddev/pkg/config/remoteconfig/internal"
	"github.com/ddev/ddev/pkg/config/remoteconfig/storage"
	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	"github.com/ddev/ddev/pkg/util"
)

// New creates and returns a new RemoteConfig.
func New(config *Config, isInternetActive func() bool) types.RemoteConfig {
	defer util.TimeTrack()()

	// Create RemoteConfig.
	cfg := &remoteConfig{
		fileStorage: storage.NewFileStorage(config.getLocalSourceFileName()),
		githubStorage: storage.NewGithubStorage(
			config.getRemoteSourceOwner(),
			config.getRemoteSourceRepo(),
			config.getRemoteSourceFilepath(),
			storage.Options{Ref: config.getRemoteSourceRef()},
		),
		localSource:      config.LocalSource,
		remoteSource:     config.RemoteSource,
		updateInterval:   config.UpdateInterval,
		tickerDisabled:   config.TickerDisabled,
		isInternetActive: isInternetActive,
	}

	// Load local remote config, also initiates update from remote.
	cfg.loadFromLocalStorage()

	return cfg
}

const (
	localFileName  = ".remote-config"
	updateInterval = 6 // default update interval in hours
)

// remoteConfig is a in memory representation of the DDEV RemoteConfig.
type remoteConfig struct {
	remoteConfig internal.RemoteConfig

	fileStorage   types.RemoteConfigStorage
	githubStorage types.RemoteConfigStorage

	localSource  LocalSource
	remoteSource RemoteSource

	updateInterval   int
	tickerDisabled   bool
	isInternetActive func() bool

	mu sync.RWMutex
}

// write saves the remote config to the local storage.
func (c *remoteConfig) write() {
	defer util.TimeTrack()()

	c.mu.Lock()
	defer c.mu.Unlock()

	err := c.fileStorage.Push(c.remoteConfig)

	if err != nil {
		util.Error("Error while writing remote config: %s", err)
	}
}

// loadFromLocalStorage loads the remote config from the local storage and
// initiates an update from the remote asynchronously.
func (c *remoteConfig) loadFromLocalStorage() {
	defer util.TimeTrack()()

	c.mu.Lock()
	defer func() {
		c.mu.Unlock()
		go c.updateFromGithub()
	}()

	var err error

	c.remoteConfig, err = c.fileStorage.Pull()

	if err != nil {
		util.Error("Error while loading remote config: %s", err)
	}
}

// updateFromGithub downloads the remote config from Github.
func (c *remoteConfig) updateFromGithub() {
	defer util.TimeTrack()()

	if !c.isInternetActive() {
		util.Debug("No internet connection.")

		return
	}

	// Check if an update is needed.
	if c.fileStorage.LastUpdate().Add(c.getUpdateInterval()).Before(time.Now()) {
		util.Debug("Downloading remote config.")

		c.mu.Lock()
		backupLast := c.remoteConfig.Messages.Ticker.Last

		defer func() {
			c.remoteConfig.Messages.Ticker.Last = backupLast
			c.mu.Unlock()
			c.write()
		}()

		// Download the remote config.
		var err error
		c.remoteConfig, err = c.githubStorage.Pull()

		if err != nil {
			util.Error("Error while downloading remote config: %s", err)
		}
	}
}

// getUpdateInterval returns the update interval for the remote config. The
// processing order is defined as follows, the first defined value is returned:
//   - global config
//   - remote config
//   - const updateInterval
func (c *remoteConfig) getUpdateInterval() time.Duration {
	if c.updateInterval > 0 {
		return time.Duration(c.updateInterval) * time.Hour
	}

	if c.remoteConfig.UpdateInterval > 0 {
		return time.Duration(c.remoteConfig.UpdateInterval) * time.Hour
	}

	return time.Duration(updateInterval) * time.Hour
}
