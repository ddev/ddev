package yaml

import (
	"os"
	"path/filepath"

	"github.com/ddev/ddev/pkg/config/state"
	"github.com/ddev/ddev/pkg/config/state/types"
	"github.com/ddev/ddev/pkg/util"
	"go.yaml.in/yaml/v4"
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

	// The state is a disposable cache, so an empty file (left by a crash
	// mid-write) or an unparseable one is reset rather than blocking DDEV.
	if err = yaml.Unmarshal(content, &raw); err != nil {
		util.Debug("Ignoring unreadable state file %s: %v", s.filePath, err)
		return make(types.RawState), nil
	}
	if raw == nil {
		raw = make(types.RawState)
	}

	return raw, nil
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

	// Write to a temp file and rename it into place so a crash mid-write
	// cannot leave a truncated state file behind.
	tmp, err := os.CreateTemp(filepath.Dir(s.filePath), filepath.Base(s.filePath)+".*.tmp")
	if err != nil {
		return
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	if _, err = tmp.Write(finalContent); err != nil {
		_ = tmp.Close()
		return
	}
	if err = tmp.Close(); err != nil {
		return
	}

	return os.Rename(tmp.Name(), s.filePath)
}

func (s *yamlStorage) TagName() string {
	return "yaml"
}
