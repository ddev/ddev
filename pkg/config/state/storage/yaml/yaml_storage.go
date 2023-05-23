package yaml

import (
	"os"
	"path/filepath"

	"github.com/ddev/ddev/pkg/config/state"
	"github.com/ddev/ddev/pkg/config/state/types"
	"gopkg.in/yaml.v3"
)

// New returns a new StateStorage interface based on a YAML file.
//
// Parameter 'stateFilePath' is the path and file name to the storage file.
func New(stateFilePath string) types.StateStorage {
	return &yamlStorage{
		filePath: stateFilePath,
	}
}

// NewState returns a new State using a YAML storage.
func NewState(stateFilePath string) types.State {
	return state.New(New(stateFilePath))
}

// yamlStorage is the in-memory representation of the storage.
type yamlStorage struct {
	filePath string
}

// Read see interface description.
func (s *yamlStorage) Read() (raw types.RawState, err error) {
	content, err := os.ReadFile(s.filePath)
	if os.IsNotExist(err) {
		return make(types.RawState), nil
	} else if err != nil {
		return
	}

	err = yaml.Unmarshal(content, &raw)

	return
}

// Write see interface description.
func (s *yamlStorage) Write(raw types.RawState) (err error) {
	content, err := yaml.Marshal(&raw)
	if err != nil {
		return
	}

	err = os.MkdirAll(filepath.Dir(s.filePath), 0755)
	if err != nil {
		return
	}

	// Prefix content with YAML document start.
	finalContent := []byte("---\n")
	finalContent = append(finalContent, content...)

	err = os.WriteFile(s.filePath, finalContent, 0600)

	return
}

func (s *yamlStorage) TagName() string {
	return "yaml"
}
