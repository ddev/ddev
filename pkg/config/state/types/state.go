package types

type StateEntry = interface{}
type StateEntryKey = string

type State interface {
	// Load reads the state from the storage into the memory.
	Load() error

	// Save writes the state from the memory into the storage.
	Save() error

	// Loaded returns true if the state is loaded into the memory.
	Loaded() bool

	// Changed returns true if the in-memory state has changed.
	Changed() bool

	// Get retrieves the state for the key into the state entry which must be
	// provided as pointer see example.
	//
	// For example:
	//
	//     type T struct {
	//         F int `yaml:"a,omitempty"`
	//         B int
	//     }
	//     var t T
	//     State.Get("my_key", &t)
	//
	Get(StateEntryKey, StateEntry) error

	// Set updates the state entry for the key. The state entry must not be
	// provided as pointer see example.
	//
	// For example:
	//
	//     type T struct {
	//         F int `yaml:"a,omitempty"`
	//         B int
	//     }
	//     var t T
	//     State.Set("my_key", t)
	//
	Set(StateEntryKey, StateEntry) error
}
