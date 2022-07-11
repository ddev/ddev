package ddevapp

import (
	"errors"
	"regexp"

	"gopkg.in/yaml.v2"
)

// mergeConfigToApp does an unmarshall with merging
func (app *DdevApp) mergeConfigToApp(source []byte) error {
	// save away state before merges.
	unmergedApp := DdevApp{}
	unmergedApp = *app

	type mergeData struct {
		newData interface{}
		oldData interface{}
	}

	// add merges here.  Default strategy is to clobber old
	// keys.
	mergeableItems := map[string]mergeData{
		"web_environment": {
			&app.WebEnvironment,
			unmergedApp.WebEnvironment,
		},
		"additional_hostnames": {
			&app.AdditionalHostnames,
			unmergedApp.AdditionalHostnames,
		},
		"hooks": {
			&app.Hooks,
			unmergedApp.Hooks,
		},
	}

	// get the updated settings. Note that we will replace
	// anything else from the upstream config for any
	// key except for web_environment.
	err := yaml.Unmarshal(source, app)
	if err != nil {
		return err
	}

	// loop over the items we know how to merge.
	for item, data := range mergeableItems {
		switch item {
		case "web_environment":
			err = app.mergeWebEnvironment(data.newData, data.oldData.([]string))
		case "additional_hostnames":
			// default case is a simple string list merge
			err = app.mergeStringList(data.newData, data.oldData.([]string))
		case "hooks":
			// merge w/o replacement
			oldHookData := data.oldData.(map[string][]YAMLTask)
			err = app.mergeHooks(data.newData, oldHookData)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

// merge an arbitrary string list.
func (app *DdevApp) mergeStringList(ptr interface{}, oldList []string) error {
	// first pass: merge w/o replacement.
	results := []string{}
	newList, ok := ptr.(*[]string)
	if !ok {
		return errors.New("unexpected type for DdevApp item")
	}

	re, err := regexp.Compile(`^\s*(!*)\s*(\S+)\s*$`)
	if err != nil {
		// this is from the code, if it ain't right, well, that ain't right.
		panic(err)
	}

	// add new to the results, dropping any delete instructions.
	processDeletes := false
	for _, inItem := range *newList {
		matches := re.FindStringSubmatch(inItem)
		if matches != nil {
			if matches[1] != "!" {
				results = append(results, matches[2])
			} else {
				processDeletes = true
			}
		}
	}

	// add any non-matching oldItems for the merge.
	for _, oldItem := range oldList {
		remove := false
		for _, newItem := range *newList {
			if processDeletes {
				// check for delete instructions
				matches := re.FindStringSubmatch(newItem)
				if matches != nil {
					newKey := matches[2]
					if newKey == oldItem {
						if matches[1] == "!" {
							// keep the item
							remove = true
						}
					}
				}

			}
		}
		if !remove {
			results = append(results, oldItem)
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

	// new hooks will contain at least the contents of the new
	newHooks := ptr.(*map[string][]YAMLTask)

	// We add any hook used in old but not in new, and merge anything that is
	// shared.
	for key, items := range oldHooks {
		ytaskList, found := (*newHooks)[key]
		if !found {
			// add it to newHooks
			(*newHooks)[key] = items
		} else {
			// no replacement, so just create a joint list
			items = append(items, ytaskList...)
			(*newHooks)[key] = items
		}
	}

	app.Hooks = *newHooks
	return nil
}
