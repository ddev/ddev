// Package state provides a simple framework to handle states from a storage.
//
// The types.State interface represents the main interface for the interaction
// with states.
//
// The types.StateStorage interface represents the interface to read and write
// states from a storage e.g. a file. At the moment the implementation for YAML
// files is done but any other type could be provided too.
//
// # Example Usage
//
//	import (
//		"github.com/ddev/ddev/pkg/config/state/storage/yaml"
//	)
//
//	type MyState struct {
//		BoolField   bool   `yaml:"bool_field"`
//		StringField string `yaml:"string_field,omitempty"`
//	}
//
//	state := yaml.NewState(filepath.Join("config", "state.yml"))
//
//	myState := MyState{}
//
//	// Retrieve state, Get() implicitly calls Load()
//	err := state.Get("my_state", &myState)
//	if err != nil {
//		// add error handling
//	}
//
//	// Access state fields
//	printf("Value of BoolField is: %v", myState.BoolField)
//	printf("Value of StringField is: %v", myState.StringField)
//
//	// Update states
//	myState.BoolField = true
//	myState.StringField = "New value for string field"
//
//	err := state.Set("my_state", myState)
//	if err != nil {
//		// add error handling
//	}
//
//	// Save state to file
//	err := state.Save()
//	if err != nil {
//		// add error handling
//	}
package state
