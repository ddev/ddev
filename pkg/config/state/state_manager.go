package state

import (
	"errors"
	"reflect"

	"github.com/ddev/ddev/pkg/config/state/types"
	"github.com/mitchellh/mapstructure"
)

// New returns a new State interface to get and set states based on the
// stateManager implementation bellow. The 'storage' parameter defines the
// storage to be used to read and write states from and to.
func New(storage types.StateStorage) types.State {
	return &stateManager{
		storage: storage,
	}
}

// stateManager is the default implementation of the State interface. It
// includes an in-memory cache of the state to reduce read/write operations
// on the storage.
type stateManager struct {
	storage      types.StateStorage
	state        types.RawState
	stateChanged bool
	stateLoaded  bool
}

// Load see interface description.
func (m *stateManager) Load() (err error) {
	m.state, err = m.storage.Read()

	if err == nil {
		m.stateChanged = false
		m.stateLoaded = true
	}

	return
}

// Save see interface description.
func (m *stateManager) Save() (err error) {
	err = m.storage.Write(m.state)

	if err == nil {
		m.stateChanged = false
		m.stateLoaded = true
	}

	return
}

// Loaded see interface description.
func (m *stateManager) Loaded() bool {
	return m.stateLoaded
}

// Changed see interface description.
func (m *stateManager) Changed() bool {
	return m.stateChanged
}

// Get see interface description.
func (m *stateManager) Get(key types.StateEntryKey, stateEntry types.StateEntry) (err error) {
	// Check stateEntry is a pointer.
	val := reflect.ValueOf(stateEntry)
	if val.Kind() != reflect.Ptr {
		return errors.New("stateEntry must be a pointer")
	}

	// Clear the stateEntry to have a clean state.
	val.Elem().SetZero()

	// Use mapstructure to convert the raw data to the desired state entry.
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ZeroFields:       true,
		WeaklyTypedInput: true,
		Result:           stateEntry,
		TagName:          m.storage.TagName(),
	})
	if err != nil {
		return
	}

	if err = m.ensureLoaded(); err != nil {
		return
	}

	return decoder.Decode(m.state[key])
}

// Set see interface description.
func (m *stateManager) Set(key types.StateEntryKey, stateEntry types.StateEntry) (err error) {
	// Check stateEntry is not a pointer.
	val := reflect.ValueOf(stateEntry)
	if val.Kind() == reflect.Ptr {
		return errors.New("stateEntry must not be a pointer")
	}

	if err = m.ensureLoaded(); err != nil {
		return
	}

	// Update or add state entry.
	m.state[key] = stateEntry
	m.stateChanged = true

	return
}

// ensureLoaded loads the state if not already loaded before.
func (m *stateManager) ensureLoaded() (err error) {
	if !m.stateLoaded {
		err = m.Load()
	}

	return
}
