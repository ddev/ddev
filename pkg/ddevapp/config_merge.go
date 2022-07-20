package ddevapp

import (
	"github.com/imdario/mergo"
)

// mergeAdditionalConfigIntoApp takes the provided yaml `config.*.yaml` and merges
// it into "app"
func (app *DdevApp) mergeAdditionalConfigIntoApp(configPath string) error {

	newConfig := DdevApp{}
	err := newConfig.LoadConfigYamlFile(configPath)
	if err != nil {
		return err
	}

	err = mergo.Merge(app, newConfig, mergo.WithAppendSlice, mergo.WithOverride)
	if err != nil {
		return err
	}

	return nil
}
