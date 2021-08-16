package ddevapp

import (
	"embed"
	"github.com/drud/ddev/pkg/globalconfig"
)

//The bundled assets for the project .ddev directory are in directory dotddev_assets
//go:embed dotddev_assets global_dotddev_assets
var bundledAssets embed.FS

// PopulateExamplesCommandsHomeadditions grabs embedded assets and
// installs them into the named directory
func PopulateExamplesCommandsHomeadditions(appName string) error {
	app, err := GetActiveApp(appName)
	// If we have an error in GetActiveApp, it means we're not in a project directory
	// That's no an error, just means we can't do this work, so return nil.
	if err != nil {
		return nil
	}

	err = CopyEmbedAssets(bundledAssets, "dotddev_assets", app.GetConfigPath(""))
	if err != nil {
		return err
	}
	err = CopyEmbedAssets(bundledAssets, "global_dotddev_assets", globalconfig.GetGlobalDdevDir())
	if err != nil {
		return err
	}

	return nil
}
