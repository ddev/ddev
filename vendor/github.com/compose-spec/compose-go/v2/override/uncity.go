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

package override

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/format"
	"github.com/compose-spec/compose-go/v2/tree"
)

type indexer func(any, tree.Path) (string, error)

// mergeSpecials defines the custom rules applied by compose when merging yaml trees
var unique = map[tree.Path]indexer{}

func init() {
	unique["networks.*.labels"] = keyValueIndexer
	unique["networks.*.ipam.options"] = keyValueIndexer
	// services and jobs share the container specification: same rules
	for _, prefix := range []tree.Path{"services", "jobs"} {
		unique[prefix+".*.annotations"] = keyValueIndexer
		unique[prefix+".*.build.args"] = keyValueIndexer
		unique[prefix+".*.build.additional_contexts"] = keyValueIndexer
		unique[prefix+".*.build.platform"] = keyValueIndexer
		unique[prefix+".*.build.tags"] = keyValueIndexer
		unique[prefix+".*.build.labels"] = keyValueIndexer
		unique[prefix+".*.cap_add"] = keyValueIndexer
		unique[prefix+".*.cap_drop"] = keyValueIndexer
		unique[prefix+".*.configs"] = mountIndexer("")
		unique[prefix+".*.dns"] = keyValueIndexer
		unique[prefix+".*.dns_opt"] = keyValueIndexer
		unique[prefix+".*.dns_search"] = keyValueIndexer
		unique[prefix+".*.environment"] = keyValueIndexer
		unique[prefix+".*.env_file"] = envFileIndexer
		unique[prefix+".*.expose"] = exposeIndexer
		unique[prefix+".*.labels"] = keyValueIndexer
		unique[prefix+".*.networks.*.aliases"] = keyValueIndexer
		unique[prefix+".*.networks.*.link_local_ips"] = keyValueIndexer
		unique[prefix+".*.ports"] = portIndexer
		unique[prefix+".*.profiles"] = keyValueIndexer
		unique[prefix+".*.secrets"] = mountIndexer("/run/secrets")
		unique[prefix+".*.sysctls"] = keyValueIndexer
		unique[prefix+".*.tmpfs"] = keyValueIndexer
		unique[prefix+".*.volumes"] = volumeIndexer
		unique[prefix+".*.devices"] = deviceMappingIndexer
	}
	// attributes only services have
	unique["services.*.deploy.labels"] = keyValueIndexer
	unique["services.*.links"] = keyValueIndexer
}

// EnforceUnicity removes redefinition of elements declared in a sequence
func EnforceUnicity(value map[string]any) (map[string]any, error) {
	uniq, err := enforceUnicity(value, tree.NewPath())
	if err != nil {
		return nil, err
	}
	return uniq.(map[string]any), nil
}

func enforceUnicity(value any, p tree.Path) (any, error) {
	switch v := value.(type) {
	case map[string]any:
		for k, e := range v {
			u, err := enforceUnicity(e, p.Next(k))
			if err != nil {
				return nil, err
			}
			v[k] = u
		}
		return v, nil
	case []any:
		for pattern, indexer := range unique {
			if p.Matches(pattern) {
				return uniqueEntries(v, indexer, p)
			}
		}
	}
	return value, nil
}

// uniqueEntries keeps one entry per key: the last entry for a key takes effect,
// at the position of the first one.
func uniqueEntries(entries []any, indexer indexer, p tree.Path) ([]any, error) {
	seq := []any{}
	keys := map[string]int{}
	for i, entry := range entries {
		key, err := indexer(entry, p.Next(fmt.Sprintf("[%d]", i)))
		if err != nil {
			return nil, err
		}
		if j, ok := keys[key]; ok {
			seq[j] = entry
		} else {
			seq = append(seq, entry)
			keys[key] = len(seq) - 1
		}
	}
	return seq, nil
}

func keyValueIndexer(v any, p tree.Path) (string, error) {
	switch value := v.(type) {
	case string:
		key, _, found := strings.Cut(value, "=")
		if found {
			return key, nil
		}
		return value, nil
	default:
		return "", fmt.Errorf("%s: unexpected type %T", p, v)
	}
}

func volumeIndexer(y any, p tree.Path) (string, error) {
	switch value := y.(type) {
	case map[string]any:
		target, ok := value["target"].(string)
		if !ok {
			return "", fmt.Errorf("service volume %s is missing a mount target", p)
		}
		return target, nil
	case string:
		volume, err := format.ParseVolume(value)
		if err != nil {
			return "", err
		}
		return volume.Target, nil
	}
	return "", nil
}

func deviceMappingIndexer(y any, p tree.Path) (string, error) {
	switch value := y.(type) {
	case map[string]any:
		target, ok := value["target"].(string)
		if !ok {
			return "", fmt.Errorf("service device %s is missing a mount target", p)
		}
		return target, nil
	case string:
		arr := strings.Split(value, ":")
		if len(arr) == 1 {
			return arr[0], nil
		}
		return arr[1], nil
	}
	return "", nil
}

func exposeIndexer(a any, path tree.Path) (string, error) {
	switch v := a.(type) {
	case string:
		return v, nil
	case int:
		return strconv.Itoa(v), nil
	default:
		return "", fmt.Errorf("%s: unsupported expose value %s", path, a)
	}
}

func mountIndexer(defaultPath string) indexer {
	return func(a any, path tree.Path) (string, error) {
		switch v := a.(type) {
		case string:
			return fmt.Sprintf("%s/%s", defaultPath, v), nil
		case map[string]any:
			if t, ok := v["target"]; ok {
				target, ok := t.(string)
				if !ok {
					return "", fmt.Errorf("%s: unexpected type %T for target", path, t)
				}
				return target, nil
			}
			return fmt.Sprintf("%s/%s", defaultPath, v["source"]), nil
		default:
			return "", fmt.Errorf("%s: unsupported expose value %s", path, a)
		}
	}
}

func portIndexer(y any, p tree.Path) (string, error) {
	switch value := y.(type) {
	case int:
		return strconv.Itoa(value), nil
	case map[string]any:
		target, ok := value["target"]
		if !ok {
			return "", fmt.Errorf("service ports %s is missing a target port", p)
		}
		published, ok := value["published"]
		if !ok {
			// try to parse it as an int
			if pub, ok := value["published"]; ok {
				published = fmt.Sprintf("%d", pub)
			}
		}
		host, ok := value["host_ip"]
		if !ok {
			host = "0.0.0.0"
		}
		protocol, ok := value["protocol"]
		if !ok {
			protocol = "tcp"
		}
		return fmt.Sprintf("%s:%s:%d/%s", host, published, target, protocol), nil
	case string:
		return value, nil
	}
	return "", nil
}

func envFileIndexer(y any, p tree.Path) (string, error) {
	switch value := y.(type) {
	case string:
		return value, nil
	case map[string]any:
		if pathValue, ok := value["path"]; ok {
			path, ok := pathValue.(string)
			if !ok {
				return "", fmt.Errorf("%s: unexpected type %T for path", p, pathValue)
			}
			return path, nil
		}
		return "", fmt.Errorf("environment path attribute %s is missing", p)
	}
	return "", nil
}
