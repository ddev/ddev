package ddevapp

import (
	"embed"
	"fmt"
	"path/filepath"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
)

// The bundled assets for the project .ddev directory are in directory dotddev_assets
// And the global .ddev assets are in directory global_dotddev_assets
//
//go:embed dotddev_assets/* dotddev_assets/commands/.gitattributes
//go:embed mysql_config_assets/*
//go:embed global_dotddev_assets/* global_dotddev_assets/.gitignore global_dotddev_assets/commands/.gitattributes
//go:embed app_compose_template.yaml
//go:embed router_compose_template.yaml
//go:embed ssh_auth_compose_template.yaml
//go:embed traefik_config_template.yaml
//go:embed traefik_static_config_template.yaml
//go:embed traefik_global_config_template.yaml
//go:embed drupal/*
//go:embed magento/*
//go:embed wordpress/*
//go:embed typo3/*
//go:embed postgres/*
//go:embed healthcheck/*
var bundledAssets embed.FS

// PopulateExamplesCommandsHomeadditions grabs embedded assets and
// installs them into the named directory
// Note that running this with no appName (very common) can result in updating
// a *different* projects assets. So `ddev start something` will first update the current
// directory's assets (if it's a project) and then later (when called with appName) update
// the actual project's assets.
func PopulateExamplesCommandsHomeadditions(appName string) error {
	defer util.TimeTrackC(fmt.Sprintf("CopyEmbedAssets in PopulateExamplesCommandsHomeadditions(%s)", appName))()

	err := fileutil.CopyEmbedAssets(bundledAssets, "global_dotddev_assets", globalconfig.GetGlobalDdevDir(), nil)
	if err != nil {
		return err
	}

	// We don't want to populate the project's .ddev directory
	// unless the project name is explicitly specified.
	if appName == "" {
		return nil
	}

	app, err := GetActiveApp(appName)
	// If we have an error from GetActiveApp, it means we're not in a project directory
	// That is not an error. It means we can not do this work, so return nil.
	if err != nil {
		return nil
	}

	err = fileutil.CopyEmbedAssets(bundledAssets, "dotddev_assets", app.GetConfigPath(""), GetInstalledAddonProjectFiles(app))
	if err != nil {
		return err
	}

	// Provide .ddev/mysql README and example by default
	if app.Database.Type == nodeps.MySQL || app.Database.Type == nodeps.MariaDB {
		err = fileutil.CopyEmbedAssets(bundledAssets, "mysql_config_assets", app.GetConfigPath("mysql"), nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func IsBundledCustomCommand(globalCommand bool, service, command string) bool {
	var baseDir string
	if globalCommand {
		baseDir = "global_dotddev_assets"
	} else {
		baseDir = "dotddev_assets"
	}

	_, err := bundledAssets.ReadFile(filepath.Join(baseDir, "commands", service, command))

	return err == nil
}

func IsBundledCustomProvider(provider string) bool {
	paths := []string{
		filepath.Join("dotddev_assets", "providers", provider) + ".yaml",
		filepath.Join("dotddev_assets", "providers", provider) + ".yaml.example",
	}
	for _, path := range paths {
		if _, err := bundledAssets.ReadFile(path); err == nil {
			return true
		}
	}
	return false
}
