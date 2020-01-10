package ddevapp

import (
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/gobuffalo/packr/v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

// isMagentoApp returns true if the app is of type magento
func isMagentoApp(app *DdevApp) bool {
	if _, err := os.Stat(filepath.Join(app.AppRoot, "get.php")); err == nil {
		return true
	}
	return false
}

// isMagento2App returns true if the app is of type magento2
func isMagento2App(app *DdevApp) bool {
	if _, err := os.Stat(filepath.Join(app.AppRoot, "pub", "static.php")); err == nil {
		return true
	}
	return false
}

// createMagentoSettingsFile manages creation and modification of settings.php and settings.ddev.php.
// If a settings.php file already exists, it will be modified to ensure that it includes
// settings.ddev.php, which contains ddev-specific configuration.
func createMagentoSettingsFile(app *DdevApp) (string, error) {

	if fileutil.FileExists(app.SiteSettingsPath) {
		// Check if the file is managed by ddev.
		signatureFound, err := fileutil.FgrepStringInFile(app.SiteSettingsPath, DdevFileSignature)
		if err != nil {
			return "", err
		}

		// If the signature wasn't found, warn the user and return.
		if !signatureFound {
			util.Warning("%s already exists and is managed by the user.", app.SiteSettingsPath)
			return "", nil
		}
	} else {
		output.UserOut.Printf("No %s file exists, creating one", app.SiteSettingsPath)

		box := packr.New("magento_packr_assets", "./magento_packr_assets")
		content, err := box.Find("local.xml")
		if err != nil {
			return "", err
		}
		err = ioutil.WriteFile(app.SiteSettingsPath, content, 0644)
		if err != nil {
			return "", err
		}
	}

	return app.SiteDdevSettingsFile, nil
}

// setMagentoSiteSettingsPaths sets the paths to settings.php for templating.
func setMagentoSiteSettingsPaths(app *DdevApp) {
	settingsFileBasePath := filepath.Join(app.AppRoot, app.Docroot)
	app.SiteSettingsPath = filepath.Join(settingsFileBasePath, "app", "etc", "local.xml")
}
