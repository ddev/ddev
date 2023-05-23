package yaml_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/config/state/storage/yaml"
	"github.com/ddev/ddev/pkg/config/state/types"
	"github.com/stretchr/testify/suite"
)

func TestYamlStorageTestSuite(t *testing.T) {
	suite.Run(t, new(YamlStorageTestSuite))
}

type YamlStorageTestSuite struct {
	suite.Suite
}

func (suite *YamlStorageTestSuite) TestReadProperlyLoadsState() {
	var tests = []struct {
		Key   types.StateEntryKey
		Value string
	}{
		// key1 to 3 are available in the test data.
		{
			Key:   "key1",
			Value: "string1",
		},
		{
			Key:   "key2",
			Value: "string2",
		},
		{
			Key:   "key3",
			Value: "string3",
		},
	}

	storage := yaml.New(filepath.Join("testdata", "state.yml"))
	data, err := storage.Read()

	suite.NoError(err)
	suite.NotNil(data)

	for _, tt := range tests {
		suite.Run(tt.Key, func() {
			suite.EqualValues(tt.Value, data[tt.Key].(map[string]interface{})["value"])
		})
	}
}

func (suite *YamlStorageTestSuite) TestReadReturnsStateWithoutStateFile() {
	storage := yaml.New(filepath.Join("testdata", "not_existing.yml"))
	data, err := storage.Read()

	suite.NoError(err)
	suite.NotNil(data)
	suite.EqualValues(types.RawState{}, data)
}

func (suite *YamlStorageTestSuite) TestReadReturnsErrorForInvalidStateFile() {
	storage := yaml.New(filepath.Join("testdata", "invalid.yml"))
	data, err := storage.Read()

	suite.ErrorContains(err, "yaml: unmarshal errors")
	suite.Nil(data)
}

func (suite *YamlStorageTestSuite) TestWriteProperlyWritesState() {
	stateFile := filepath.Join(suite.T().TempDir(), "state.yml")
	storage := yaml.New(stateFile)

	// Simulation of the state.yml.
	err := storage.Write(types.RawState{
		"key1": map[string]string{
			"value": "string1",
		},
		"key2": map[string]string{
			"value": "string2",
		},
		"key3": map[string]string{
			"value": "string3",
		},
	})

	suite.NoError(err)
	suite.FileExists(stateFile)

	want, err := os.ReadFile(filepath.Join("testdata", "state.yml"))
	if err != nil {
		suite.FailNow("error reading state file:", err)
	}

	has, err := os.ReadFile(stateFile)
	if err != nil {
		suite.FailNow("error reading written state file:", err)
	}

	suite.Equal(want, has)
}

func BenchmarkRead(b *testing.B) {
	storage := yaml.New(filepath.Join("testdata", "state.yml"))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = storage.Read()
	}
}

func BenchmarkWrite(b *testing.B) {
	storage := yaml.New(filepath.Join(b.TempDir(), "state.yml"))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = storage.Write(types.RawState{
			"key1": map[string]string{
				"value": "string1",
			},
			"key2": map[string]string{
				"value": "string2",
			},
			"key3": map[string]string{
				"value": "string3",
			},
		})
	}
}
