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

package paths

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/compose-spec/compose-go/v2/tree"
	"github.com/compose-spec/compose-go/v2/types"
)

type resolver func(any) (any, error)

// ResolveRelativePaths make relative paths absolute
func ResolveRelativePaths(project map[string]any, base string, remotes []RemoteResource) error {
	r := relativePathsResolver{
		workingDir: base,
		remotes:    remotes,
	}
	r.resolvers = map[tree.Path]resolver{
		"services.*.build.context":               r.absContextPath,
		"services.*.build.additional_contexts.*": r.absAdditionalContextPath,
		"services.*.build.ssh.*":                 r.absSSHPath,
		"services.*.env_file.*.path":             r.absPath,
		"services.*.label_file.*":                r.absPath,
		"services.*.extends.file":                r.absExtendsPath,
		"services.*.develop.watch.*.path":        r.absSymbolicLink,
		"services.*.volumes.*":                   r.absVolumeMount,
		"jobs.*.build.context":                   r.absContextPath,
		"jobs.*.build.additional_contexts.*":     r.absAdditionalContextPath,
		"jobs.*.build.ssh.*":                     r.absSSHPath,
		"jobs.*.env_file.*.path":                 r.absPath,
		"jobs.*.label_file.*":                    r.absPath,
		"jobs.*.extends.file":                    r.absExtendsPath,
		"jobs.*.volumes.*":                       r.absVolumeMount,
		// pre_start hooks are full container specifications (compose-spec#656)
		"services.*.pre_start.*.env_file.*.path": r.absPath,
		"services.*.pre_start.*.label_file.*":    r.absPath,
		"services.*.pre_start.*.volumes.*":       r.absVolumeMount,
		"configs.*.file":                         r.maybeUnixPath,
		"secrets.*.file":                         r.maybeUnixPath,
		"include.path":                           r.absPath,
		"include.project_directory":              r.absPath,
		"include.env_file":                       r.absPath,
		"volumes.*":                              r.volumeDriverOpts,
	}
	_, err := r.resolveRelativePaths(project, tree.NewPath())
	return err
}

type RemoteResource func(path string) bool

type relativePathsResolver struct {
	workingDir string
	remotes    []RemoteResource
	resolvers  map[tree.Path]resolver
}

func (r *relativePathsResolver) isRemoteResource(path string) bool {
	for _, remote := range r.remotes {
		if remote(path) {
			return true
		}
	}
	return false
}

func (r *relativePathsResolver) resolveRelativePaths(value any, p tree.Path) (any, error) {
	for pattern, resolver := range r.resolvers {
		if p.Matches(pattern) {
			resolved, err := resolver(value)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", p, err)
			}
			return resolved, nil
		}
	}
	switch v := value.(type) {
	case map[string]any:
		for k, e := range v {
			resolved, err := r.resolveRelativePaths(e, p.Next(k))
			if err != nil {
				return nil, err
			}
			v[k] = resolved
		}
	case []any:
		for i, e := range v {
			resolved, err := r.resolveRelativePaths(e, p.Next("[]"))
			if err != nil {
				return nil, err
			}
			v[i] = resolved
		}
	}
	return value, nil
}

func (r *relativePathsResolver) absPath(value any) (any, error) {
	switch v := value.(type) {
	case []any:
		for i, s := range v {
			abs, err := r.absPath(s)
			if err != nil {
				return nil, err
			}
			v[i] = abs
		}
		return v, nil
	case string:
		v = ExpandUser(v)
		if filepath.IsAbs(v) {
			return v, nil
		}
		if v != "" {
			return filepath.Join(r.workingDir, v), nil
		}
		return v, nil
	}

	return nil, fmt.Errorf("unexpected type %T", value)
}

// absSSHPath resolves the path of a build ssh key. The value of a mapping may
// be a number or a boolean, which the model reads as the text it is written
// with: it is resolved like that text, as it would be if it was written as one.
func (r *relativePathsResolver) absSSHPath(value any) (any, error) {
	switch value.(type) {
	case nil, string:
	default:
		value = fmt.Sprint(value)
	}
	return r.maybeUnixPath(value)
}

func (r *relativePathsResolver) absVolumeMount(a any) (any, error) {
	switch vol := a.(type) {
	case map[string]any:
		if vol["type"] != types.VolumeTypeBind {
			return vol, nil
		}
		src, ok := vol["source"]
		if !ok || src == "" {
			return nil, errors.New(`invalid mount config for type "bind": field Source must not be empty`)
		}
		// a source that is not a string is left as is, like any non canonical
		// value: the schema validation rejects it, and without validation the
		// model decoding reads it as text
		abs, err := r.maybeUnixPath(src)
		if err != nil {
			return nil, err
		}
		vol["source"] = abs
		return vol, nil
	default:
		// not using canonical format, skip
		return a, nil
	}
}

func (r *relativePathsResolver) volumeDriverOpts(a any) (any, error) {
	if a == nil {
		return nil, nil
	}
	vol, ok := a.(map[string]any)
	if !ok {
		// not a volume definition: the model decoding reports it
		return a, nil
	}
	if vol["driver"] != "local" {
		return vol, nil
	}
	do, ok := vol["driver_opts"]
	if !ok {
		return vol, nil
	}
	opts, ok := do.(map[string]any)
	if !ok {
		return vol, nil
	}
	if dev, ok := opts["device"]; opts["o"] == "bind" && ok {
		// This is actually a bind mount
		path, err := r.maybeUnixPath(dev)
		if err != nil {
			return nil, err
		}
		opts["device"] = path
	}
	return vol, nil
}
