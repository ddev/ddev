package ddevapp

import (
	"embed"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
)

//The bundled assets for the project .ddev directory are in directory dotddev_assets
// And the global .ddev assets are in directory global_dotddev_assets
//go:embed dotddev_assets/* dotddev_assets/commands/.gitattributes global_dotddev_assets/* global_dotddev_assets/.gitignore global_dotddev_assets/commands/.gitattributes app_compose_template.yaml router_compose_template.yaml ssh_auth_compose_template.yaml drupal/* magento/* wordpress/* typo3/* postgres/* healthcheck/*
var bundledAssets embed.FS

// PopulateExamplesCommandsHomeadditions grabs embedded assets and
// installs them into the named directory
// Note that running this with no appName (very common) can result in updating
// a *different* projects assets. So `ddev start something` will first update the current
// directory's assets (if it's a project) and then later (when called with appName) update
// the actual project's assets.
func PopulateExamplesCommandsHomeadditions(appName string) error {
	app, err := GetActiveApp(appName)
	// If we have an error from GetActiveApp, it means we're not in a project directory
	// That's not an error, just means we can't do this work, so return nil.
	if err != nil {
		return nil
	}

	err = fileutil.CopyEmbedAssets(bundledAssets, "dotddev_assets", app.GetConfigPath(""))
	if err != nil {
		return err
	}
	err = fileutil.CopyEmbedAssets(bundledAssets, "global_dotddev_assets", globalconfig.GetGlobalDdevDir())
	if err != nil {
		return err
	}

	return nil
}
