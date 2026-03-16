package ddevapp

import (
	"strings"

	"dario.cat/mergo"
	"github.com/ddev/ddev/pkg/util"
)

// mergeAdditionalConfigIntoApp takes the provided yaml `config.*.yaml` and merges
// it into "app"
func (app *DdevApp) mergeAdditionalConfigIntoApp(configPath string) error {
	newConfig := DdevApp{}
	err := newConfig.LoadConfigYamlFile(configPath)
	if err != nil {
		return err
	}

	// If override_config is set in the config.*.yaml, load it on top of the app.
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

// EnvToUniqueEnv() makes sure that only the last occurrence of an env (NAME=val or bare NAME)
// slice is actually retained. Bare variable names without a value (e.g. "MY_VAR") are passed
// through as-is; docker-compose resolves them from the host environment at container start time.
func EnvToUniqueEnv(inSlice *[]string) []string {
	mapStore := map[string]string{}

	for _, s := range *inSlice {
		// Both "KEY=value" and bare "KEY" are supported.
		// strings.Cut returns the part before "=" as the key in both cases.
		// Last entry for a given key wins.
		k, _, _ := strings.Cut(s, "=")
		mapStore[k] = s
	}
	newSlice := make([]string, 0, len(mapStore))
	for _, v := range mapStore {
		newSlice = append(newSlice, v)
	}
	if len(newSlice) == 0 {
		return nil
	}
	return newSlice
}
