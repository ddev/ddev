package ddevapp

import (
	"github.com/imdario/mergo"
)

// mergeAdditionalConfigIntoApp takes the provided yaml `config.*.yaml` and merges
// it into "app"
func (app *DdevApp) mergeAdditionalConfigIntoApp(configPath string) error {

	newConfig, err := NewAppFromConfigFileOnly(app.AppRoot, configPath)
	if err != nil {
		return err
	}

	// These items can't be overridden
	newConfig.Name = app.Name
	newConfig.AppRoot = app.AppRoot
	//newConfig.Docroot = app.Docroot

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
