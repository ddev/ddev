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

package loader

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/override"
	"github.com/compose-spec/compose-go/v2/types"
)

// Normalize compose project by moving deprecated attributes to their canonical position and injecting implicit defaults
//
// The model is expected in canonical form, as the loader hands it over. An
// attribute holding a value of another shape is reported as an error naming
// it, instead of crashing.
func Normalize(dict map[string]any, env types.Mapping) (map[string]any, error) {
	if err := normalizeNetworks(dict); err != nil {
		return nil, err
	}

	lookup := func(s string) (string, bool) {
		v, ok := env[s]
		return v, ok
	}

	for _, key := range []string{"services", "jobs"} {
		d, ok := dict[key]
		if !ok {
			continue
		}
		containers, err := expectMapping(d, key)
		if err != nil {
			return nil, err
		}
		for name, s := range containers {
			where := key + "." + name
			container, err := expectMapping(s, where)
			if err != nil {
				return nil, err
			}
			if err := normalizeContainer(container, lookup, where); err != nil {
				return nil, err
			}
			if key == "services" {
				// a job runs to completion: it has no restart policy, nor `deploy`
				carryRestartPolicy(name, container)
			}
			containers[name] = container
		}

		dict[key] = containers
	}
	if err := setNameFromKey(dict); err != nil {
		return nil, err
	}

	return dict, nil
}

// expectMapping returns v, the value of the attribute found at the config path
// where, as a mapping, or an error naming the attribute when it holds something
// else.
func expectMapping(v any, where string) (map[string]any, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be a mapping", where)
	}
	return m, nil
}

// expectList returns v, the value of the attribute found at the config path
// where, as a list, or an error naming the attribute when it holds something
// else.
func expectList(v any, where string) ([]any, error) {
	l, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be a list", where)
	}
	return l, nil
}

// expectString returns v, the value of the attribute found at the config path
// where, as a string, or an error naming the attribute when it holds something
// else.
func expectString(v any, where string) (string, error) {
	str, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", where)
	}
	return str, nil
}

// normalizeContainer normalizes in place the service or job definition found at
// the config path where.
func normalizeContainer(container map[string]any, lookup func(string) (string, bool), where string) error {
	if container["pull_policy"] == types.PullPolicyIfNotPresent {
		container["pull_policy"] = types.PullPolicyMissing
	}

	if b, ok := container["build"]; ok {
		build, err := expectMapping(b, where+".build")
		if err != nil {
			return err
		}
		normalizeBuild(build, lookup)
	}

	if e, ok := container["environment"]; ok {
		container["environment"], _ = resolve(e, lookup, true)
	}

	dependsOn := map[string]any{}
	if d, ok := container["depends_on"]; ok {
		var err error
		dependsOn, err = expectMapping(d, where+".depends_on")
		if err != nil {
			return err
		}
	}
	for _, step := range []func(map[string]any, map[string]any, string) error{
		addLinksDependencies,
		addNamespaceDependencies,
		cleanVolumeTargets,
		addVolumesFromDependencies,
	} {
		if err := step(container, dependsOn, where); err != nil {
			return err
		}
	}
	if len(dependsOn) > 0 {
		container["depends_on"] = dependsOn
	}

	return resolvePreStartHooks(container)
}

// normalizeBuild injects the implicit build context and dockerfile, and
// resolves build args declared without a value from the environment
func normalizeBuild(build map[string]any, lookup func(string) (string, bool)) {
	if build["context"] == nil {
		build["context"] = "."
	}
	if build["dockerfile"] == nil && build["dockerfile_inline"] == nil {
		build["dockerfile"] = "Dockerfile"
	}

	if a, ok := build["args"]; ok {
		build["args"], _ = resolve(a, lookup, false)
	}
}

// implicitDependency builds the depends_on entry for a dependency the user
// did not declare but which the service definition implies
func implicitDependency(restart bool) map[string]any {
	return map[string]any{
		"condition": types.ServiceConditionStarted,
		"restart":   restart,
		"required":  true,
	}
}

// addLinksDependencies makes each linked service a dependency, unless
// declared as such already
func addLinksDependencies(container map[string]any, dependsOn map[string]any, where string) error {
	l, ok := container["links"]
	if !ok {
		return nil
	}
	links, err := expectList(l, where+".links")
	if err != nil {
		return err
	}
	for i, e := range links {
		link, err := expectString(e, fmt.Sprintf("%s.links[%d]", where, i))
		if err != nil {
			return err
		}
		parts := strings.Split(link, ":")
		if len(parts) == 2 {
			link = parts[0]
		}
		if _, ok := dependsOn[link]; !ok {
			dependsOn[link] = implicitDependency(true)
		}
	}
	return nil
}

// addNamespaceDependencies makes a service whose network, ipc, pid, uts or
// cgroup namespace is shared with another service depend on it, unless
// declared as such already
func addNamespaceDependencies(container map[string]any, dependsOn map[string]any, where string) error {
	for _, namespace := range []string{"network_mode", "ipc", "pid", "uts", "cgroup"} {
		n, ok := container[namespace]
		if !ok {
			continue
		}
		ref, err := expectString(n, where+"."+namespace)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(ref, types.ServicePrefix) {
			continue
		}
		shared := ref[len(types.ServicePrefix):]
		if _, ok := dependsOn[shared]; !ok {
			dependsOn[shared] = implicitDependency(true)
		}
	}
	return nil
}

// cleanVolumeTargets lexically cleans the container path of volumes
func cleanVolumeTargets(container map[string]any, _ map[string]any, containerPath string) error {
	v, ok := container["volumes"]
	if !ok {
		return nil
	}
	volumes, err := expectList(v, containerPath+".volumes")
	if err != nil {
		return err
	}
	for i, volume := range volumes {
		volumePath := fmt.Sprintf("%s.volumes[%d]", containerPath, i)
		vol, err := expectMapping(volume, volumePath)
		if err != nil {
			return err
		}
		target, err := expectString(vol["target"], volumePath+".target")
		if err != nil {
			return err
		}
		vol["target"] = path.Clean(target)
		volumes[i] = vol
	}
	container["volumes"] = volumes
	return nil
}

// addVolumesFromDependencies makes a service mounting the volumes of another
// service depend on it (without restarting it on update), unless declared as
// such already. Volumes taken from an external container are not a service
// dependency.
func addVolumesFromDependencies(container map[string]any, dependsOn map[string]any, where string) error {
	n, ok := container["volumes_from"]
	if !ok {
		return nil
	}
	volumesFrom, err := expectList(n, where+".volumes_from")
	if err != nil {
		return err
	}
	for i, v := range volumesFrom {
		vol, err := expectString(v, fmt.Sprintf("%s.volumes_from[%d]", where, i))
		if err != nil {
			return err
		}
		if strings.HasPrefix(vol, types.ContainerPrefix) {
			continue
		}
		spec := strings.Split(vol, ":")
		if _, ok := dependsOn[spec[0]]; !ok {
			dependsOn[spec[0]] = implicitDependency(false)
		}
	}
	return nil
}

// resolvePreStartHooks materializes pre_start inheritance at load time: each
// hook's container specification is completed with the service's own
// container-spec attributes, per the compose merge rules — the hook's
// declarations win on conflicts, and accumulated entries strictly identical
// to an inherited one are not duplicated, which keeps this resolution
// idempotent when an already-resolved model (a `config` output) is loaded
// again. This subsumes the image inheritance of compose-spec#647.
//
// Volumes are deliberately not inherited: mounts inherit at runtime through
// volumes_from, the only mechanism able to share the parent's anonymous and
// image volumes; the hook keeps only its own volume declarations.
func resolvePreStartHooks(service map[string]any) error {
	hooks, ok := service["pre_start"].([]any)
	if !ok || len(hooks) == 0 {
		return nil
	}
	base := map[string]any{}
	for k, v := range service {
		if k == "volumes" || !containerSpecKeys[k] {
			continue
		}
		base[k] = v
	}
	for i, h := range hooks {
		hook, ok := h.(map[string]any)
		if !ok {
			continue
		}
		// wrapped as a service entry so the per-attribute merge rules
		// registered under services.* apply to the hook's specification
		merged, err := override.Merge(
			map[string]any{"services": map[string]any{"hook": deepClone(base)}},
			map[string]any{"services": map[string]any{"hook": hook}},
		)
		if err != nil {
			return fmt.Errorf("resolving pre_start hook specification: %w", err)
		}
		hooks[i] = merged["services"].(map[string]any)["hook"]
	}
	return nil
}

// containerSpecKeys indexes types.ContainerSpecAttributes (generated at
// build time from the ContainerSpec type, so the boundary cannot drift from
// the model). The filter matters for the dict form of the model
// (LoadModel/ToModel consumers): schema validation runs before normalization
// and the typed decode drops unknown keys, so nothing else prevents workload
// or service-only attributes (ports, build, depends_on, x-*) from leaking
// into the rendered hooks.
var containerSpecKeys = func() map[string]bool {
	keys := make(map[string]bool, len(types.ContainerSpecAttributes))
	for _, k := range types.ContainerSpecAttributes {
		keys[k] = true
	}
	return keys
}()

// connectToDefaultNetwork connects a container that declares no network, or an
// empty networks section, to the default network. It tells whether the
// container is connected to the default network, explicitly or not.
func connectToDefaultNetwork(container map[string]any, where string) (bool, error) {
	n, ok := container["networks"]
	if !ok {
		// If none explicitly declared, container is connected to default network
		container["networks"] = map[string]any{"default": nil}
		return true, nil
	}
	networks, err := expectMapping(n, where+".networks")
	if err != nil {
		return false, err
	}
	if len(networks) == 0 {
		// networks section declared but empty (corner case)
		container["networks"] = map[string]any{"default": nil}
		return true, nil
	}
	_, ok = networks["default"]
	return ok, nil
}

func normalizeNetworks(dict map[string]any) error {
	networks := map[string]any{}
	if n, ok := dict["networks"]; ok {
		var err error
		networks, err = expectMapping(n, "networks")
		if err != nil {
			return err
		}
	}

	// implicit `default` network must be introduced only if actually used by some service
	usesDefaultNetwork := false

	for _, key := range []string{"services", "jobs"} {
		s, ok := dict[key]
		if !ok {
			continue
		}
		containers, err := expectMapping(s, key)
		if err != nil {
			return err
		}
		for name, se := range containers {
			container, err := expectMapping(se, key+"."+name)
			if err != nil {
				return err
			}
			if _, ok := container["provider"]; ok {
				continue
			}
			if _, ok := container["network_mode"]; ok {
				continue
			}
			onDefault, err := connectToDefaultNetwork(container, key+"."+name)
			if err != nil {
				return err
			}
			usesDefaultNetwork = usesDefaultNetwork || onDefault
			containers[name] = container
		}
		dict[key] = containers
	}

	if _, ok := networks["default"]; !ok && usesDefaultNetwork {
		// If not declared explicitly, Compose model involves an implicit "default" network
		networks["default"] = nil
	}

	if len(networks) > 0 {
		dict["networks"] = networks
	}
	return nil
}

func resolve(a any, fn func(s string) (string, bool), keepEmpty bool) (any, bool) {
	switch v := a.(type) {
	case []any:
		var resolved []any
		for _, val := range v {
			if r, ok := resolve(val, fn, keepEmpty); ok {
				resolved = append(resolved, r)
			}
		}
		return resolved, true
	case map[string]any:
		resolved := map[string]any{}
		for key, val := range v {
			if val != nil {
				resolved[key] = val
				continue
			}
			if s, ok := fn(key); ok {
				resolved[key] = s
			} else if keepEmpty {
				resolved[key] = nil
			}
		}
		return resolved, true
	case string:
		if !strings.Contains(v, "=") {
			if val, ok := fn(v); ok {
				return fmt.Sprintf("%s=%s", v, val), true
			}
			if keepEmpty {
				return v, true
			}
			return "", false
		}
		return v, true
	default:
		return v, false
	}
}

// Resources with no explicit name are actually named by their key in map
func setNameFromKey(dict map[string]any) error {
	for _, kind := range []string{"networks", "volumes", "configs", "secrets"} {
		a, ok := dict[kind]
		if !ok {
			continue
		}
		toplevel, err := expectMapping(a, kind)
		if err != nil {
			return err
		}
		for key, r := range toplevel {
			resource := map[string]any{}
			if r != nil {
				resource, err = expectMapping(r, kind+"."+key)
				if err != nil {
					return err
				}
			}
			if resource["name"] == nil {
				if x, ok := resource["external"]; ok && isTrue(x) {
					resource["name"] = key
				} else {
					resource["name"] = fmt.Sprintf("%s_%s", dict["name"], key)
				}
			}
			toplevel[key] = resource
		}
	}
	return nil
}

func isTrue(x any) bool {
	// with interpolation skipped, `external` can still be a string: read it as
	// the decoding of the model does
	if s, ok := x.(string); ok {
		switch strings.ToLower(s) {
		case "true", "y", "yes", "on":
			return true
		}
	}
	parseBool, _ := strconv.ParseBool(fmt.Sprint(x))
	return parseBool
}
