package ddevapp

import (
	"errors"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/imdario/mergo"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

// mergeConfigToApp takes the provided yaml `config.*.yaml` and merges
// it into "app"
func (app *DdevApp) mergeConfigToApp(configPath string) error {

	// Moving the config.*.yaml file to a tmp dir is a hack, perhaps temporary,
	// to avoid refactoring NewApp() so it (optionally?) takes a file instead of a dir.
	// If this experiment works out, we may want to rework NewApp() to (optionally?) take a specific file instead of dir
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Join(tmpDir, ".ddev"), 0755)
	if err != nil {
		return err
	}
	err = fileutil.CopyFile(configPath, filepath.Join(tmpDir, ".ddev", "config.yaml"))
	if err != nil {
		return err
	}

	newConfig, err := NewApp(tmpDir, false)
	if err != nil {
		return err
	}

	// These items can't be overridden
	newConfig.Name = app.Name
	newConfig.AppRoot = app.AppRoot
	newConfig.Type = app.Type
	newConfig.Docroot = app.Docroot

	err = newConfig.ValidateConfig()
	if err != nil {
		return err
	}
	err = mergo.Merge(app, newConfig, mergo.WithAppendSlice, mergo.WithOverride)
	if err != nil {
		return err
	}

	return nil
}

// merge an arbitrary string list.
func (app *DdevApp) mergeStringList(ptr interface{}, oldList []string) error {
	results := []string{}
	results = append(results, oldList...)

	newList, ok := ptr.(*[]string)
	if !ok {
		return errors.New("unexpected type for DdevApp item")
	}

	re, _ := regexp.Compile(`^\s*(!*)\s*(\S+)\s*$`)

	// support for a future delete syntax. This stuff runs, but
	// is not yet used
	for _, inItem := range *newList {
		matches := re.FindStringSubmatch(inItem)
		if matches != nil {
			if matches[1] != "!" {
				results = append(results, matches[2])
			}
		} else {
			log.Println("found an invalid string")
		}
	}

	// save back the merged results
	*newList = results
	return nil
}

// merge the web_environment string list.
func (app *DdevApp) mergeWebEnvironment(ptr interface{}, oldEnv []string) error {

	result := []string{}
	newEnv := app.WebEnvironment

	// ENV=value or ENV=
	re, err := regexp.Compile(`^([^=]+)=(\S*)`)
	if err != nil {
		return nil
	}

	// start by walking the old env. replace any
	// changed strings, keep any unchanged.
	for _, oldItem := range oldEnv {

		// check new for any matches
		matches := re.FindStringSubmatch(oldItem)
		if matches == nil {
			// does not look like an env string
			continue
		}
		key := matches[1]

		// does new have this key?
		// if so, replace it
		for _, newItem := range newEnv {
			matches = re.FindStringSubmatch(newItem)
			if matches != nil && key == matches[1] {
				oldItem = newItem // match overrides
			}
		}
		// winner added to result list
		result = append(result, oldItem)
	}

	// Now add any non-matched new keys into the results
	// since new wins, we find exact matches or nothing.
	for _, newItem := range newEnv {
		found := false
		for _, rsltItem := range result {
			if rsltItem == newItem {
				found = true
			}
		}
		if !found {
			result = append(result, newItem)
		}
	}
	app.WebEnvironment = result
	return nil
}

func (app *DdevApp) mergeHooks(ptr interface{}, oldHooks map[string][]YAMLTask) error {
	mergedHooks := map[string][]YAMLTask{}
	// new hooks will contain at least the contents of the new
	newHooks := ptr.(*map[string][]YAMLTask)

	// check to see if there is anything to merge.
	if len(oldHooks) == 0 && len(*newHooks) == 0 {
		// not an error, but return early.
		return nil
	}

	// We add any hook used in old but not in new, and merge anything that is
	// shared.
	if len(oldHooks) > 0 {
		for key, items := range oldHooks {
			ytaskList, found := (*newHooks)[key]
			if !found {
				// add it to newHooks
				mergedHooks[key] = items
			} else {
				// no replacement, so just create a joint list
				items = append(items, ytaskList...)
				mergedHooks[key] = items
			}
		}
	} else if len(*newHooks) > 0 {
		// nothing in old hooks, load in new hooks
		for key, items := range *newHooks {
			mergedHooks[key] = items
		}
	}

	app.Hooks = mergedHooks
	return nil
}
