package ddevapp

import (
	"fmt"
	"github.com/ddev/ddev/pkg/util"
	"github.com/imdario/mergo"
	"strings"
)

// mergeAdditionalConfigIntoApp takes the provided yaml `config.*.yaml` and merges
// it into "app"
func (app *DdevApp) mergeAdditionalConfigIntoApp(configPath string) error {
	newConfig := DdevApp{}
	err := newConfig.LoadConfigYamlFile(configPath)
	if err != nil {
		return err
	}

	// If override_config is set in the config.*.yaml, then just load it on top of app
	// Otherwise (the normal default case) merge.
	if newConfig.OverrideConfig {
		err = app.LoadConfigYamlFile(configPath)
		if err != nil {
			return err
		}
	} else {
		err = mergo.Merge(app, newConfig, mergo.WithAppendSlice, mergo.WithOverride)
		if err != nil {
			return err
		}
	}

	// We don't need this set; it's only a flag to determine behavior above
	app.OverrideConfig = false

	// Make sure we don't have absolutely identical items in our resultant arrays
	for _, arr := range []*[]string{&app.WebImageExtraPackages, &app.DBImageExtraPackages, &app.AdditionalHostnames, &app.AdditionalFQDNs, &app.OmitContainers} {
		*arr = util.SliceToUniqueSlice(arr)
	}

	for _, arr := range []*[]string{&app.WebEnvironment} {
		*arr = EnvToUniqueEnv(arr)
	}

	return nil
}

// EnvToUniqueEnv() makes sure that only the last occurrence of an env (NAME=val)
// slice is actually retained.
func EnvToUniqueEnv(inSlice *[]string) []string {
	mapStore := map[string]string{}
	newSlice := []string{}

	for _, s := range *inSlice {
		// config.yaml vars look like ENV1=val1 and ENV2=val2
		// Split them and then make sure the last one wins
		k, v, found := strings.Cut(s, "=")
		// If we didn't find the "=" delimiter, it wasn't an env
		if !found {
			continue
		}
		mapStore[k] = v
	}
	for k, v := range mapStore {
		newSlice = append(newSlice, fmt.Sprintf("%s=%v", k, v))
	}
	if len(newSlice) == 0 {
		return nil
	}
	return newSlice
}
