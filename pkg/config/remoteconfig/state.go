package remoteconfig

import (
	"time"

	statetypes "github.com/ddev/ddev/pkg/config/state/types"
	"github.com/ddev/ddev/pkg/util"
)

func NewState(stateManager statetypes.State) *StateEntry {
	state := &StateEntry{
		stateManager: stateManager,
	}

	if err := state.Load(); err != nil {
		util.Debug("Error while loading state: %s", err)
	}

	return state
}

type StateEntry struct {
	stateManager statetypes.State

	UpdatedAt           time.Time `yaml:"updated_at"`
	LastTickerMessageAt time.Time `yaml:"last_ticker_message_at"`
	LastTickerMessage   int       `yaml:"last_ticker_message"`
}

const stateKey = "remote_config"

func (s *StateEntry) Load() error {
	return s.stateManager.Get(stateKey, s)
}

func (s *StateEntry) Save() error {
	return s.stateManager.Set(stateKey, *s)
}
