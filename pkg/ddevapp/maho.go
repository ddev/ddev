package ddevapp

import (
	"os"
	"path/filepath"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
)

// isMahoApp returns true if the app is of type maho
func isMahoApp(app *DdevApp) bool {
	isMaho, err := fileutil.FgrepStringInFile(filepath.Join(app.AppRoot, app.ComposerRoot, "composer.json"), `"mahocommerce/maho"`)
	if err == nil && isMaho {
		return true
	}
	return false
}

// createMahoSettingsFile manages creation and modification of app/etc/local.xml.
func createMahoSettingsFile(app *DdevApp) (string, error) {
	if fileutil.FileExists(app.SiteSettingsPath) {
		// Check if the file is managed by ddev.
		signatureFound, err := fileutil.FgrepStringInFile(app.SiteSettingsPath, nodeps.DdevFileSignature)
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

		// app/etc does not exist in a fresh maho-starter project
		if err := os.MkdirAll(filepath.Dir(app.SiteSettingsPath), 0755); err != nil {
			return "", err
		}

		content, err := bundledAssets.ReadFile("maho/local.xml")
		if err != nil {
			return "", err
		}
		templateVars := map[string]any{"DBHostname": "db"}
		err = fileutil.TemplateStringToFile(string(content), templateVars, app.SiteSettingsPath)
		if err != nil {
			return "", err
		}
	}

	return app.SiteDdevSettingsFile, nil
}

// setMahoSiteSettingsPaths sets the paths to local.xml for templating.
// Maho keeps app/etc/local.xml at the project root, outside the public docroot.
func setMahoSiteSettingsPaths(app *DdevApp) {
	app.SiteSettingsPath = filepath.Join(app.AppRoot, "app", "etc", "local.xml")
}

// getMahoUploadDirs will return the default paths.
func getMahoUploadDirs(_ *DdevApp) []string {
	return []string{"media"}
}
