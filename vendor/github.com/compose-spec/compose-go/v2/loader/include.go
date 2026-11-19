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
	"crypto/sha256"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/compose-spec/compose-go/v2/dotenv"
	interp "github.com/compose-spec/compose-go/v2/interpolation"
	"github.com/compose-spec/compose-go/v2/override"
	"github.com/compose-spec/compose-go/v2/tree"
	"github.com/compose-spec/compose-go/v2/types"
)

type includeCacheKey struct{}

// withIncludeCache attaches a fresh include cache to ctx. The cache is scoped
// to a single load so a file reachable through several include paths (a
// "diamond" include graph) is parsed and expanded only once per distinct
// (paths, directories, environment) tuple.
func withIncludeCache(ctx context.Context) context.Context {
	return context.WithValue(ctx, includeCacheKey{}, map[string]map[string]any{})
}

func includeCache(ctx context.Context) map[string]map[string]any {
	cache, _ := ctx.Value(includeCacheKey{}).(map[string]map[string]any)
	return cache
}

// includeModelKey covers every input that determines the model loaded for an
// include: resolved paths, working directory, project directory and effective
// environment. Interpolate.Substitute and TypeCastMapping are intentionally
// excluded: they are invariant across includes within a single load. If a
// future option allows them to vary per include, they must be folded into the
// key. Fields are length-prefixed and collections count-prefixed so the byte
// stream is uniquely decodable.
func includeModelKey(paths types.StringList, workingDir, projectDir string, env types.Mapping) string {
	h := sha256.New()
	write := func(s string) {
		fmt.Fprintf(h, "%d:%s", len(s), s)
	}
	fmt.Fprintf(h, "%d;", len(paths))
	for _, p := range paths {
		write(p)
	}
	write(workingDir)
	write(projectDir)
	fmt.Fprintf(h, "%d;", len(env))
	for _, k := range slices.Sorted(maps.Keys(env)) {
		write(k)
		write(env[k])
	}
	return string(h.Sum(nil))
}

// loadIncludeConfig parse the required config from raw yaml
func loadIncludeConfig(source any) ([]types.IncludeConfig, error) {
	if source == nil {
		return nil, nil
	}
	configs, ok := source.([]any)
	if !ok {
		return nil, fmt.Errorf("`include` must be a list, got %s", source)
	}
	for i, config := range configs {
		if v, ok := config.(string); ok {
			configs[i] = map[string]any{
				"path": v,
			}
		}
	}
	var requires []types.IncludeConfig
	err := Transform(source, &requires)
	return requires, err
}

func ApplyInclude(ctx context.Context, workingDir string, environment types.Mapping, model map[string]any, options *Options, included []string, processor PostProcessor) error {
	includeConfig, err := loadIncludeConfig(model["include"])
	if err != nil {
		return err
	}

	i := &includer{
		ctx:         ctx,
		workingDir:  workingDir,
		environment: environment,
		model:       model,
		options:     options,
		included:    included,
		processor:   processor,
	}
	for _, r := range includeConfig {
		if err := i.apply(&r); err != nil {
			return err
		}
	}
	delete(model, "include")
	return nil
}

// includer imports the resources of the files listed by the `include`
// section of a model into this very model
type includer struct {
	ctx context.Context
	// workingDir is the working directory of the file declaring the includes
	workingDir  string
	environment types.Mapping
	// model is the model declaring the includes, receiving the included resources
	model   map[string]any
	options *Options
	// included is the stack of files being included, used to detect cycles
	included  []string
	processor PostProcessor
}

// apply loads the files of one include entry, and imports their resources
func (i *includer) apply(r *types.IncludeConfig) error {
	for _, listener := range i.options.Listeners {
		listener("include", map[string]any{
			"path":       r.Path,
			"workingdir": i.workingDir,
		})
	}

	relworkingdir, err := i.resolvePaths(r)
	if err != nil {
		return err
	}

	loadOptions := i.loadOptions(r)

	if err := i.resolveEnvFiles(r); err != nil {
		return err
	}

	envFromFile, err := dotenv.GetEnvFromFile(i.environment, r.EnvFile)
	if err != nil {
		return err
	}

	config := types.ConfigDetails{
		WorkingDir:  relworkingdir,
		ConfigFiles: types.ToConfigFiles(r.Path),
		Environment: i.environment.Clone().Merge(envFromFile),
	}
	loadOptions.Interpolate = &interp.Options{
		Substitute:      i.options.Interpolate.Substitute,
		LookupValue:     config.LookupEnv,
		TypeCastMapping: i.options.Interpolate.TypeCastMapping,
	}
	imported, err := i.loadModel(config, loadOptions, r)
	if err != nil {
		return err
	}
	return importResources(imported, i.model, i.processor)
}

// resolvePaths replaces the paths of the include entry by the local files
// served by the resource loaders. It returns the working directory, relative
// when possible, of the main (first) file.
func (i *includer) resolvePaths(r *types.IncludeConfig) (string, error) {
	var relworkingdir string
	for idx, p := range r.Path {
		resolved, relDir, err := i.resolvePath(r, idx, p)
		if err != nil {
			return "", err
		}
		if idx == 0 {
			relworkingdir = relDir
		}
		r.Path[idx] = resolved
	}
	return relworkingdir, nil
}

// resolvePath loads the idx-th path of the include entry with every resource
// loader accepting it. The first path is the "main" file, used to define
// project-directory, the others are overrides: for this one the working
// directory is returned too, and an include cycle is reported.
//
// There is no break after the first loader accepting the path, on purpose. The
// local loader is last and accepts everything, so a path first loaded by a
// remote loader is handled a second time by the local loader, on the local copy
// the remote loader returned. For the first path of the entry, the first pass
// sets the project directory and returns the remote loader's Dir of the local
// copy; the second pass finds the project directory set and returns it, as an
// absolute path, instead. That is what makes the relative paths of a remote
// file resolve: the remote loaders of Docker Compose key Dir on the reference
// given to Load, but Dir is called here with the local copy and so returns
// nothing. Stopping at the first loader would leave that working directory
// empty.
func (i *includer) resolvePath(r *types.IncludeConfig, idx int, p string) (string, string, error) {
	var relworkingdir string
	for _, loader := range i.options.ResourceLoaders {
		if !loader.Accept(p) {
			continue
		}
		path, err := loader.Load(i.ctx, p)
		if err != nil {
			return "", "", err
		}
		p = path

		if idx == 0 {
			relworkingdir = i.setProjectDirectory(r, loader, path)
			if err := i.checkCycle(path); err != nil {
				return "", "", err
			}
		}
	}
	return p, relworkingdir, nil
}

// setProjectDirectory defines the absolute project directory of the include
// entry, from the main file or the declared project_directory, and returns
// the working directory to load the included files with
func (i *includer) setProjectDirectory(r *types.IncludeConfig, loader ResourceLoader, path string) string {
	switch {
	case r.ProjectDirectory == "":
		r.ProjectDirectory = filepath.Dir(path)
		return loader.Dir(path)
	case !filepath.IsAbs(r.ProjectDirectory):
		relworkingdir := loader.Dir(r.ProjectDirectory)
		r.ProjectDirectory = filepath.Join(i.projectDirectory(), r.ProjectDirectory)
		return relworkingdir
	default:
		return r.ProjectDirectory
	}
}

// checkCycle reports an error if path is already being included
func (i *includer) checkCycle(path string) error {
	for _, f := range i.included {
		if f == path {
			i.included = append(i.included, path)
			return fmt.Errorf("include cycle detected:\n%s\n include %s", i.included[0], strings.Join(i.included[1:], "\n include "))
		}
	}
	return nil
}

// loadOptions derives the options to load the files of the include entry
func (i *includer) loadOptions(r *types.IncludeConfig) *Options {
	loadOptions := i.options.clone()
	loadOptions.ResolvePaths = true
	loadOptions.SkipNormalization = true
	loadOptions.SkipConsistencyCheck = true
	// include and extends events are only reported for declarations in the
	// config files passed to the loader, not for those in included files
	loadOptions.Listeners = nil
	loadOptions.ResourceLoaders = append(loadOptions.RemoteResourceLoaders(), localResourceLoader{
		WorkingDir: r.ProjectDirectory,
	})
	return loadOptions
}

// resolveEnvFiles makes the env files of the include entry absolute, and
// defaults to the .env file of the project directory when none is declared
func (i *includer) resolveEnvFiles(r *types.IncludeConfig) error {
	if len(r.EnvFile) == 0 {
		f := filepath.Join(r.ProjectDirectory, ".env")
		if s, err := os.Stat(f); err == nil && !s.IsDir() {
			r.EnvFile = types.StringList{f}
		}
		return nil
	}

	envFile := []string{}
	for _, f := range r.EnvFile {
		if f == "/dev/null" {
			continue
		}
		if !filepath.IsAbs(f) {
			var err error
			f, err = i.checkRelativeEnvFile(f)
			if err != nil {
				return err
			}
		}
		envFile = append(envFile, f)
	}
	r.EnvFile = envFile
	return nil
}

// projectDirectory is the directory a relative path declared by an include
// entry, `env_file` or `project_directory`, is resolved against, as the file
// system sees it: the project directory of the file declaring the entry.
//
// It is workingDir when that one is absolute, as it is for the main project
// and for the files served by a remote resource loader. It is not, in general:
// workingDir is also the directory the model of the file is relative to, for
// the files of an included project, that of the project including it. The paths
// of the model are resolved from it level by level, until the main project
// makes them absolute. That is right for the paths of the model, but such a
// workingDir is not a path of the file system, and looking for a file with it
// finds it from the working directory of the process. The local resource loader
// the including entry set in the options holds the project directory to use.
func (i *includer) projectDirectory() string {
	if filepath.IsAbs(i.workingDir) {
		return i.workingDir
	}
	for _, loader := range slices.Backward(i.options.ResourceLoaders) {
		if local, ok := loader.(localResourceLoader); ok && local.WorkingDir != "" {
			return local.WorkingDir
		}
	}
	return i.workingDir
}

// checkRelativeEnvFile resolves a relative env file against the project
// directory of the including file, and checks it is an existing file
func (i *includer) checkRelativeEnvFile(f string) (string, error) {
	f = filepath.Join(i.projectDirectory(), f)
	s, err := os.Stat(f)
	if err != nil {
		return "", err
	}
	if s.IsDir() {
		return "", fmt.Errorf("%s is not a file", f)
	}
	return f, nil
}

// loadModel loads the model of the included files, or gets it from the
// include cache when the very same include was loaded already
func (i *includer) loadModel(config types.ConfigDetails, loadOptions *Options, r *types.IncludeConfig) (map[string]any, error) {
	cache := includeCache(i.ctx)
	cacheKey := includeModelKey(r.Path, config.WorkingDir, r.ProjectDirectory, config.Environment)
	if imported, hit := cache[cacheKey]; hit {
		// hand out a copy, the cached model must stay pristine
		return deepClone(imported).(map[string]any), nil
	}
	imported, err := loadYamlModel(i.ctx, config, loadOptions, &cycleTracker{}, i.included)
	if err != nil {
		return nil, err
	}
	if cache != nil {
		// store a pristine copy: the model returned to the caller is
		// merged into the parent and mutated by later loading phases
		cache[cacheKey] = deepClone(imported).(map[string]any)
	}
	return imported, nil
}

// importResources import into model all resources defined by imported, and report error on conflict
func importResources(source map[string]any, target map[string]any, processor PostProcessor) error {
	for _, key := range []string{"services", "jobs", "volumes", "networks", "secrets", "configs", "models"} {
		if err := importResource(source, target, key, processor); err != nil {
			return err
		}
	}
	return nil
}

func importResource(source map[string]any, target map[string]any, key string, processor PostProcessor) error {
	from := source[key]
	if from != nil {
		resources, err := expectMapping(from, key)
		if err != nil {
			return err
		}
		// a section left empty (`volumes:`) is declared, but holds nothing
		to := map[string]any{}
		if v := target[key]; v != nil {
			to, err = expectMapping(v, key)
			if err != nil {
				return err
			}
		}
		for name, a := range resources {
			conflict, ok := to[name]
			if !ok {
				to[name] = a
				continue
			}
			if reflect.DeepEqual(a, conflict) {
				// Same resource reached through multiple include paths (a
				// diamond); re-merging identical definitions would append
				// duplicate entries to list-valued fields.
				continue
			}
			err := processor.Apply(map[string]any{
				key: map[string]any{
					name: a,
				},
			})
			if err != nil {
				return err
			}

			merged, err := override.MergeYaml(a, conflict, tree.NewPath(key, name))
			if err != nil {
				return err
			}
			to[name] = merged
		}
		target[key] = to
	}
	return nil
}
