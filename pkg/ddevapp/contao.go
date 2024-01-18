package ddevapp

import (
	"errors"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"os"
	"path/filepath"
)

// isContaoApp returns true if the app is of type contao
func isContaoApp(app *DdevApp) bool {
	isContao, err := fileutil.FgrepStringInFile(filepath.Join(app.AppRoot, app.ComposerRoot, "vendor/composer/installed.json"), `"name": "contao/core-bundle"`)
	if err == nil && isContao {
		return true
	}
	return false
}

// setContaoSiteSettingsPaths sets the paths to .env.local file.
func setContaoSiteSettingsPaths(app *DdevApp) {
	app.SiteSettingsPath = filepath.Join(app.AppRoot, ".env.local")
}

// contaoConfigOverrideAction sets up the default webserverType
func contaoConfigOverrideAction(app *DdevApp) error {
	if app.WebserverType == nodeps.WebserverDefault {
		app.WebserverType = nodeps.WebserverApacheFPM
	}
	return nil
}

// contaoPostStartAction sets up the .env.local file
func contaoPostStartAction(app *DdevApp) error {
	if app.DisableSettingsManagement {
		return nil
	}

	envFilePath := filepath.Join(app.AppRoot, ".env.local")
	_, envText, err := ReadProjectEnvFile(envFilePath)
	var envMap = map[string]string{
		"DATABASE_URL": `mysql://db:db@db/db`,
		"MAILER_DSN":   `smtp://127.0.0.1:1025`,
	}

	switch {
	case err == nil:
		util.Warning("Updating %s with %v", envFilePath, envMap)
		fallthrough
	case errors.Is(err, os.ErrNotExist):
		err := WriteProjectEnvFile(envFilePath, envMap, envText)
		if err != nil {
			return err
		}
	default:
		util.Warning("error opening %s: %v", envFilePath, err)
	}

	return nil
}

// getContaoUploadDirs will return the default paths.
func getContaoUploadDirs(_ *DdevApp) []string {
	return []string{"files"}
}
