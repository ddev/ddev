package composer

import (
	"encoding/json"
	"os"
	"strings"
)

type Manifest struct {
	Filename    string
	RootPackage map[string]interface{}
}

func NewManifest(filename string) (*Manifest, error) {
	manifest := &Manifest{
		Filename: filename,
	}

	err := manifest.load()

	if err != nil {
		return nil, err
	}

	return manifest, nil
}

func (m *Manifest) load() (err error) {
	content, err := os.ReadFile(m.Filename)
	if err != nil {
		return
	}

	err = json.Unmarshal(content, &m.RootPackage)

	return
}

// keyExists traverses the given value until key is found or returns false.
func keyExists(value *map[string]interface{}, key *string) bool {
	path := strings.SplitN(*key, ".", 2)

	v, found := (*value)[path[0]]
	if !found {
		return false
	}

	if len(path) == 1 {
		return true
	}

	childMap, ok := v.(map[string]interface{})

	if !ok {
		return false
	}

	return keyExists(&childMap, &path[1])
}

// keyExists returns true if the given key exists. The key is a dot separated
// value e.g. "config.allow-plugins".
func (m *Manifest) keyExists(key string) bool {
	return keyExists(&m.RootPackage, &key)
}

func (m *Manifest) HasPostRootPackageInstallScript() bool {
	return m.keyExists("scripts.post-root-package-install")
}

func (m *Manifest) HasPostCreateProjectCmdScript() bool {
	return m.keyExists("scripts.post-create-project-cmd")
}
