package types

// RawState is used to hold a weak type in-memory representation of the state.
type RawState = map[string]any

type StateStorage interface {
	// Read loads the state from the storage.
	//
	// Must not return an error in case the state does not exist in the
	// storage but an empty RawState.
	Read() (RawState, error)

	// Write saves the state to the storage.
	Write(RawState) error

	// TagName returns the tag name used for additional field configuration.
	TagName() string
}
