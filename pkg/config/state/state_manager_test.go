package state_test

import (
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/config/state/storage/yaml"
	"github.com/ddev/ddev/pkg/config/state/types"
	"github.com/stretchr/testify/suite"
)

func TestStateTestSuite(t *testing.T) {
	suite.Run(t, new(StateTestSuite))
}

type TestStateSubEntry struct {
	BoolField1   bool    `yaml:"bool_field1,omitempty"`
	IntField1    int     `yaml:"int_field1,omitempty"`
	UIntField1   uint    `yaml:"uint_field1,omitempty"`
	FloatField1  float32 `yaml:"float_field1,omitempty"`
	StringField1 string  `yaml:"string_field1,omitempty"`
}

type TestStateEntry struct {
	BoolField1   bool              `yaml:"bool_field1,omitempty"`
	BoolField2   bool              `yaml:"bool_field2,omitempty"`
	IntField1    int               `yaml:"int_field1,omitempty"`
	IntField2    int16             `yaml:"int_field2,omitempty"`
	UIntField1   uint              `yaml:"uint_field1,omitempty"`
	UIntField2   uint16            `yaml:"uint_field2,omitempty"`
	FloatField1  float32           `yaml:"float_field1,omitempty"`
	FloatField2  float64           `yaml:"float_field2,omitempty"`
	StringField1 string            `yaml:"string_field1,omitempty"`
	StringField2 string            `yaml:"string_field2,omitempty"`
	Sub          TestStateSubEntry `yaml:"sub,omitempty"`
}

type StateTestSuite struct {
	suite.Suite
}

func (suite *StateTestSuite) TestGetReturnsCorrectState() {
	var tests = []struct {
		Key      types.StateEntryKey
		Expected TestStateEntry
	}{
		// key1 to 3 are available in the test data.
		{
			Key: "key1",
			Expected: TestStateEntry{
				BoolField1:   true,
				BoolField2:   false,
				IntField1:    11,
				IntField2:    12,
				UIntField1:   112,
				UIntField2:   123,
				FloatField1:  1.1,
				FloatField2:  1.2,
				StringField1: "string11",
				StringField2: "string12",
			},
		},
		{
			Key: "key2",
			Expected: TestStateEntry{
				BoolField1:   false,
				BoolField2:   true,
				IntField1:    21,
				IntField2:    22,
				UIntField1:   212,
				UIntField2:   223,
				FloatField1:  2.1,
				FloatField2:  2.2,
				StringField1: "string21",
				StringField2: "string22",
			},
		},
		{
			Key: "key3",
			Expected: TestStateEntry{
				BoolField1:   true,
				BoolField2:   true,
				IntField1:    31,
				IntField2:    32,
				UIntField1:   312,
				UIntField2:   323,
				FloatField1:  3.1,
				FloatField2:  3.2,
				StringField1: "string31",
				StringField2: "string32",
			},
		},
		// Tests if the initialization works properly.
		{
			Key:      "dummy",
			Expected: TestStateEntry{},
		},
	}

	// Use the state file from the test data.
	state := yaml.NewState(filepath.Join("testdata", "state.yml"))

	// Variable must be initialized before the for loop to make the last test
	// working which needs values to be set.
	stateEntry := TestStateEntry{}

	for _, tt := range tests {
		suite.Run(tt.Key, func() {
			suite.NoError(state.Get(tt.Key, &stateEntry))
			suite.EqualValues(tt.Expected, stateEntry)
		})
	}
}

func (suite *StateTestSuite) TestSetUpdatesState() {
	// Use a non existing state file from the temp dir.
	state := yaml.NewState(filepath.Join(suite.T().TempDir(), "state.yml"))

	stateEntryOut := TestStateEntry{
		BoolField1:   true,
		BoolField2:   false,
		FloatField1:  12.34,
		FloatField2:  23.45,
		IntField1:    12,
		IntField2:    23,
		UIntField1:   123,
		UIntField2:   234,
		StringField1: "string1",
		StringField2: "string2",
		Sub: TestStateSubEntry{
			BoolField1:   true,
			IntField1:    1,
			UIntField1:   2,
			FloatField1:  3.1,
			StringField1: "string1",
		},
	}
	stateEntryIn := TestStateEntry{}

	suite.NoError(state.Set("test_state_entry", stateEntryOut))
	suite.NoError(state.Get("test_state_entry", &stateEntryIn))
	suite.EqualValues(stateEntryOut, stateEntryIn)
}

func (suite *StateTestSuite) TestSetPreservesExistingStates() {
	var tests = []struct {
		Key   types.StateEntryKey
		Entry TestStateEntry
	}{
		{
			Key: "key1",
			Entry: TestStateEntry{
				BoolField1:   true,
				BoolField2:   false,
				IntField1:    11,
				IntField2:    12,
				UIntField1:   112,
				UIntField2:   123,
				FloatField1:  1.1,
				FloatField2:  1.2,
				StringField1: "string11",
				StringField2: "string12",
			},
		},
		{
			Key: "key2",
			Entry: TestStateEntry{
				BoolField1:   false,
				BoolField2:   true,
				IntField1:    21,
				IntField2:    22,
				UIntField1:   212,
				UIntField2:   223,
				FloatField1:  2.1,
				FloatField2:  2.2,
				StringField1: "string21",
				StringField2: "string22",
			},
		},
		{
			Key: "key3",
			Entry: TestStateEntry{
				BoolField1:   true,
				BoolField2:   true,
				IntField1:    31,
				IntField2:    32,
				UIntField1:   312,
				UIntField2:   323,
				FloatField1:  3.1,
				FloatField2:  3.2,
				StringField1: "string31",
				StringField2: "string32",
			},
		},
	}

	// Use a non existing state file from the temp dir.
	state := yaml.NewState(filepath.Join(suite.T().TempDir(), "state.yml"))

	for _, tt := range tests {
		suite.NoError(state.Set(tt.Key, tt.Entry))
	}

	// Save and reload states.
	suite.NoError(state.Save())
	suite.NoError(state.Load())

	// Add test state.
	testState := TestStateEntry{
		BoolField1: true,
		BoolField2: true,
	}
	suite.NoError(state.Set("test", testState))

	// Verify previous states.
	stateEntry := TestStateEntry{}

	for _, tt := range tests {
		suite.Run(tt.Key, func() {
			suite.NoError(state.Get(tt.Key, &stateEntry))
			suite.EqualValues(tt.Entry, stateEntry)
		})
	}

	// Verify test state.
	suite.NoError(state.Get("test", &stateEntry))
	suite.EqualValues(testState, stateEntry)
}

func (suite *StateTestSuite) TestGetExpectsPointer() {
	// Use a non existing state file from the temp dir.
	state := yaml.NewState(filepath.Join(suite.T().TempDir(), "state.yml"))

	stateEntry := TestStateEntry{}

	suite.ErrorContains(state.Get("test_state_entry", stateEntry), "stateEntry must be a pointer")
}

func (suite *StateTestSuite) TestSetExpectsNonPointer() {
	// Use a non existing state file from the temp dir.
	state := yaml.NewState(filepath.Join(suite.T().TempDir(), "state.yml"))

	stateEntry := TestStateEntry{}

	suite.ErrorContains(state.Set("test_state_entry", &stateEntry), "stateEntry must not be a pointer")
}

func (suite *StateTestSuite) TestLoadedIsProperlySet() {
	var tests = []struct {
		Name string
		File string
	}{
		{
			Name: "StateExists",
			File: filepath.Join("testdata", "state.yml"),
		},
		{
			Name: "StateNotExists",
			File: filepath.Join(suite.T().TempDir(), "state.yml"),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.Name, func() {
			state := yaml.NewState(tt.File)

			suite.False(state.Loaded())
			suite.NoError(state.Load())
			suite.True(state.Loaded())
			suite.NoError(state.Save())
			suite.True(state.Loaded())
		})
	}
}

func (suite *StateTestSuite) TestChangedIsProperlySet() {
	state := yaml.NewState(filepath.Join(suite.T().TempDir(), "state.yml"))

	suite.False(state.Changed())
	suite.NoError(state.Load())
	suite.False(state.Changed())
	suite.NoError(state.Set("test_state_entry", TestStateEntry{}))
	suite.True(state.Changed())
	suite.NoError(state.Save())
	suite.False(state.Changed())
}
