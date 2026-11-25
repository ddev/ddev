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
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/compose-spec/compose-go/v2/consts"
	"github.com/compose-spec/compose-go/v2/override"
	"github.com/compose-spec/compose-go/v2/paths"
	"github.com/compose-spec/compose-go/v2/types"
)

// extendable are the top-level sections whose entries can extend another entry
// of the same section: a service extends a service, a job extends a job.
var extendable = []string{"services", "jobs"}

// kindOf returns the singular of an extendable section, for error messages.
func kindOf(section string) string {
	return strings.TrimSuffix(section, "s")
}

func ApplyExtends(ctx context.Context, dict map[string]any, opts *Options, tracker *cycleTracker, post PostProcessor) error {
	for _, key := range extendable {
		a, ok := dict[key]
		if !ok {
			continue
		}
		entries, ok := a.(map[string]any)
		if !ok {
			return fmt.Errorf("%s must be a mapping", key)
		}
		for name := range entries {
			merged, err := applyServiceExtends(ctx, key, name, entries, opts, tracker, post)
			if err != nil {
				return err
			}
			entries[name] = merged
		}
		dict[key] = entries
	}
	return nil
}

// applyServiceExtends resolves the extends of the entry name of the section
// (services or jobs), among the entries of that section.
func applyServiceExtends(ctx context.Context, section, name string, services map[string]any, opts *Options, tracker *cycleTracker, post PostProcessor) (any, error) {
	s := services[name]
	if s == nil {
		return nil, nil
	}
	service, ok := s.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s.%s must be a mapping", section, name)
	}
	extends, ok := service["extends"]
	if !ok {
		return s, nil
	}
	filename := ctx.Value(consts.ComposeFileKey{}).(string)
	var (
		err  error
		ref  string
		file any
	)
	switch v := extends.(type) {
	case map[string]any:
		ref, ok = v["service"].(string)
		if !ok {
			return nil, fmt.Errorf("extends.%s.service is required", name)
		}
		file = v["file"]
		opts.ProcessEvent("extends", v)
	case string:
		ref = v
		opts.ProcessEvent("extends", map[string]any{"service": ref})
	}

	var (
		base      any
		processor = post
	)

	if file != nil {
		refFilename, ok := file.(string)
		if !ok {
			return nil, fmt.Errorf("extends.%s.file must be a string", name)
		}
		services, processor, err = getExtendsBaseFromFile(ctx, section, name, ref, filename, refFilename, opts, tracker)
		if err != nil {
			return nil, err
		}
		filename = refFilename
		// extends declared in the referenced file are not part of the config
		// files passed to the loader: don't report events for them
		if len(opts.Listeners) > 0 {
			opts = opts.clone()
			opts.Listeners = nil
		}
	} else {
		_, ok := services[ref]
		if !ok {
			return nil, fmt.Errorf("cannot extend %s %q in %s: %s %q not found", kindOf(section), name, filename, kindOf(section), ref)
		}
	}

	tracker, err = tracker.Add(filename, name)
	if err != nil {
		return nil, err
	}

	// recursively apply `extends`
	base, err = applyServiceExtends(ctx, section, ref, services, opts, tracker, processor)
	if err != nil {
		return nil, err
	}

	if base == nil {
		return service, nil
	}
	source := deepClone(base).(map[string]any)

	err = post.Apply(map[string]any{
		section: map[string]any{
			name: source,
		},
	})
	if err != nil {
		return nil, err
	}

	merged, err := override.ExtendService(source, service)
	if err != nil {
		return nil, err
	}

	delete(merged, "extends")
	services[name] = merged
	return merged, nil
}

type extendsCacheKey struct{}

// extendsRef identifies an extends.file base within an extends cache scope.
type extendsRef struct {
	path       string
	workingDir string
	section    string
}

type extendsBase struct {
	// services are the entries of the section of the file the extending entry
	// belongs to: the services of a service, the jobs of a job
	services  map[string]any
	processor PostProcessor
}

// withExtendsCache attaches a fresh extends.file cache to ctx. The cache is
// scoped to a single loadYamlModel call: within that scope interpolation
// options and environment are fixed, so the resolved path and working
// directory fully identify the loaded file.
func withExtendsCache(ctx context.Context) context.Context {
	return context.WithValue(ctx, extendsCacheKey{}, map[extendsRef]extendsBase{})
}

func extendsCache(ctx context.Context) map[extendsRef]extendsBase {
	cache, _ := ctx.Value(extendsCacheKey{}).(map[extendsRef]extendsBase)
	return cache
}

func getExtendsBaseFromFile(
	ctx context.Context,
	section, name, ref string,
	path, refPath string,
	opts *Options,
	ct *cycleTracker,
) (map[string]any, PostProcessor, error) {
	for _, loader := range opts.ResourceLoaders {
		if !loader.Accept(refPath) {
			continue
		}
		local, err := loader.Load(ctx, refPath)
		if err != nil {
			return nil, nil, err
		}
		relworkingdir := loader.Dir(refPath)

		cache := extendsCache(ctx)
		cacheKey := extendsRef{path: local, workingDir: relworkingdir, section: section}
		base, hit := cache[cacheKey]
		if hit {
			// hand out a copy: resolving an extends chain writes merged
			// services back into this map (see applyServiceExtends)
			base.services = deepClone(base.services).(map[string]any)
		} else {
			base, err = loadExtendsBase(ctx, section, name, local, relworkingdir, opts, ct)
			if err != nil {
				return nil, nil, err
			}
			if cache != nil {
				cache[cacheKey] = extendsBase{
					services:  deepClone(base.services).(map[string]any),
					processor: base.processor,
				}
			}
		}

		if _, ok := base.services[ref]; !ok {
			return nil, nil, fmt.Errorf(
				"cannot extend %s %q in %s: %s %q not found in %s",
				kindOf(section),
				name,
				path,
				kindOf(section),
				ref,
				refPath,
			)
		}
		return base.services, base.processor, nil
	}
	return nil, nil, fmt.Errorf("cannot read %s", refPath)
}

func loadExtendsBase(ctx context.Context, section, name, local, relworkingdir string, opts *Options, ct *cycleTracker) (extendsBase, error) {
	extendsOpts := opts.clone()
	// replace localResourceLoader with a new flavour, using extended file base path
	extendsOpts.ResourceLoaders = append(opts.RemoteResourceLoaders(), localResourceLoader{
		WorkingDir: filepath.Dir(local),
	})
	extendsOpts.ResolvePaths = false // we do relative path resolution after file has been loaded
	extendsOpts.SkipNormalization = true
	extendsOpts.SkipConsistencyCheck = true
	extendsOpts.SkipInclude = true
	extendsOpts.SkipExtends = true    // we manage extends recursively based on raw service definition
	extendsOpts.SkipValidation = true // we validate the merge result
	extendsOpts.SkipDefaultValues = true
	source, processor, err := loadYamlFile(ctx, types.ConfigFile{Filename: local},
		extendsOpts, relworkingdir, nil, ct, map[string]any{}, nil)
	if err != nil {
		return extendsBase{}, err
	}
	m, ok := source[section]
	if !ok {
		return extendsBase{}, fmt.Errorf("cannot extend %s %q in %s: no %s section", kindOf(section), name, local, section)
	}
	services, ok := m.(map[string]any)
	if !ok {
		return extendsBase{}, fmt.Errorf("cannot extend %s %q in %s: %s must be a mapping", kindOf(section), name, local, section)
	}

	var remotes []paths.RemoteResource
	for _, loader := range opts.RemoteResourceLoaders() {
		remotes = append(remotes, loader.Accept)
	}
	err = paths.ResolveRelativePaths(source, relworkingdir, remotes)
	if err != nil {
		return extendsBase{}, err
	}

	return extendsBase{services: services, processor: processor}, nil
}

func deepClone(value any) any {
	switch v := value.(type) {
	case []any:
		cp := make([]any, len(v))
		for i, e := range v {
			cp[i] = deepClone(e)
		}
		return cp
	case map[string]any:
		cp := make(map[string]any, len(v))
		for k, e := range v {
			cp[k] = deepClone(e)
		}
		return cp
	default:
		return value
	}
}
