package remoteconfig

import (
	"time"

	statetypes "github.com/ddev/ddev/pkg/config/state/types"
	"github.com/ddev/ddev/pkg/util"
)

func newState(stateManager statetypes.State) *state {
	state := &state{
		stateManager: stateManager,
	}

	if err := state.load(); err != nil {
		util.Debug("Error while loading state: %s", err)
	}

	return state
}

type state struct {
	stateManager statetypes.State
	stateEntry
}

type stateEntry struct {
	UpdatedAt          time.Time `yaml:"updated_at"`
	LastNotificationAt time.Time `yaml:"last_notification_at"`
	LastTickerAt       time.Time `yaml:"last_ticker_at"`
	LastTickerMessage  int       `yaml:"last_ticker_message"`
}

const stateKey = "remote_config"

func (s *state) load() error {
	return s.stateManager.Get(stateKey, &s.stateEntry)
}

func (s *state) save() (err error) {
	err = s.stateManager.Set(stateKey, s.stateEntry)
	if err != nil {
		return
	}

	return s.stateManager.Save()
}
