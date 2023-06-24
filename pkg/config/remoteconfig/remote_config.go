package remoteconfig

import (
	"sync"
	"time"

	"github.com/ddev/ddev/pkg/config/remoteconfig/internal"
	"github.com/ddev/ddev/pkg/config/remoteconfig/storage"
	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	statetypes "github.com/ddev/ddev/pkg/config/state/types"
	"github.com/ddev/ddev/pkg/util"
)

// New creates and returns a new RemoteConfig.
func New(config *Config, stateManager statetypes.State, isInternetActive func() bool) types.RemoteConfig {
	defer util.TimeTrack()()

	// Create RemoteConfig.
	cfg := &remoteConfig{
		state:            newState(stateManager),
		fileStorage:      storage.NewFileStorage(config.getLocalSourceFileName()),
		updateInterval:   config.UpdateInterval,
		tickerInterval:   config.TickerInterval,
		isInternetActive: isInternetActive,
	}

	// Load local remote config.
	cfg.loadFromLocalStorage()

	// Configure remote and initiate update.
	cfg.githubStorage = storage.NewGithubStorage(
		config.getRemoteSourceOwner(&cfg.remoteConfig),
		config.getRemoteSourceRepo(&cfg.remoteConfig),
		config.getRemoteSourceFilepath(&cfg.remoteConfig),
		storage.Options{Ref: config.getRemoteSourceRef(&cfg.remoteConfig)},
	)
	cfg.updateFromGithub()

	return cfg
}

const (
	localFileName = ".remote-config"
	// Default intervals in hours
	updateInterval        = 10
	notificationsInterval = 20
	tickerInterval        = 20
)

// remoteConfig is a in memory representation of the DDEV RemoteConfig.
type remoteConfig struct {
	state        *state
	remoteConfig internal.RemoteConfig

	fileStorage   types.RemoteConfigStorage
	githubStorage types.RemoteConfigStorage

	updateInterval   int
	tickerInterval   int
	isInternetActive func() bool

	mu sync.Mutex
}

// write saves the remote config to the local storage.
func (c *remoteConfig) write() {
	defer util.TimeTrack()()

	err := c.fileStorage.Write(c.remoteConfig)

	if err != nil {
		util.Debug("Error while writing remote config: %s", err)
	}
}

// loadFromLocalStorage loads the remote config from the local storage and
// initiates an update from the remote asynchronously.
func (c *remoteConfig) loadFromLocalStorage() {
	defer util.TimeTrack()()

	c.mu.Lock()
	defer c.mu.Unlock()

	var err error

	c.remoteConfig, err = c.fileStorage.Read()

	if err != nil {
		util.Debug("Error while loading remote config: %s", err)
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
	if c.state.UpdatedAt.Add(c.getUpdateInterval()).Before(time.Now()) {
		util.Debug("Downloading remote config.")

		c.mu.Lock()
		defer c.mu.Unlock()

		var err error

		// Download the remote config.
		c.remoteConfig, err = c.githubStorage.Read()

		if err != nil {
			util.Debug("Error while downloading remote config: %s", err)

			return
		}

		c.write()

		// Update state.
		c.state.UpdatedAt = time.Now()
		if err = c.state.save(); err != nil {
			util.Debug("Error while saving state: %s", err)
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
