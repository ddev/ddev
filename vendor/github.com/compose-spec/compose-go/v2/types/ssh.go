/*
   Copyright 2020 The Compose Specification Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package types

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type SSHKey struct {
	ID   string `yaml:"id,omitempty" json:"id,omitempty"`
	Path string `path:"path,omitempty" json:"path,omitempty"`
}

// SSHConfig is a mapping type for SSH build config
type SSHConfig []SSHKey

func (s SSHConfig) Get(id string) (string, error) {
	for _, sshKey := range s {
		if sshKey.ID == id {
			return sshKey.Path, nil
		}
	}
	return "", fmt.Errorf("ID %s not found in SSH keys", id)
}

// format is the `id=path` form of the key, as the Docker CLI and buildx read
// `--ssh`, and as it loads back from a list. The path is left out for the
// default agent (`default`), which is the only id a list accepts alone.
func (s SSHKey) format() string {
	if s.Path == "" && s.ID == "default" {
		return s.ID
	}
	return s.ID + "=" + s.Path
}

// MarshalYAML makes SSHKey implement yaml.Marshaller
func (s SSHKey) MarshalYAML() (interface{}, error) {
	return s.format(), nil
}

// MarshalJSON makes SSHKey implement json.Marshaller
func (s SSHKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.format())
}

func (s *SSHConfig) DecodeMapstructure(value interface{}) error {
	v, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid ssh config type %T", value)
	}
	result := make(SSHConfig, 0, len(v))
	for id, path := range v {
		// `id=path` is the way a key is written, and read back: an id that has
		// a `=` would be cut in two
		if strings.Contains(id, "=") {
			return fmt.Errorf("invalid ssh key id %q: must not contain \"=\"", id)
		}
		key := SSHKey{ID: id}
		if path != nil {
			key.Path = fmt.Sprint(path)
		}
		result = append(result, key)
	}
	// a mapping has no order: keep the keys in a stable one, so that the same
	// file always loads, and marshals, the same way
	slices.SortFunc(result, func(a, b SSHKey) int { return strings.Compare(a.ID, b.ID) })
	*s = result
	return nil
}
