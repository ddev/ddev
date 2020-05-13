package ddevapp

import (
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/util"
	"path/filepath"
)

const (
	WarnTypeAbsent        = iota
	WarnTypeNotConfigured = iota
)

// isLaravelApp returns true if the app is of type laravel
func isLaravelApp(app *DdevApp) bool {
	return fileutil.FileExists(filepath.Join(app.AppRoot, "artisan"))
}

// setLaravelSiteSettingsPaths sets the paths to .env
func setLaravelSiteSettingsPaths(app *DdevApp) {
	app.SiteSettingsPath = filepath.Join(app.AppRoot, ".env")
}

func envSettingsWarning(status int) {
	var srcFile = ".env"
	var message = "Don't forget to configure the database in your .env file"

	if WarnTypeAbsent == status {
		srcFile += ".example"
		message = "Don't forget to create the .env file with proper database settings"
	}
	util.Warning(message)
	util.Warning("You can do it with this one-liner:")
	util.Warning("ddev exec \"cat %v | sed  -E 's/DB_(HOST|DATABASE|USERNAME|PASSWORD)=(.*)/DB_\\1=db/g' > .env\"", srcFile)
}

func laravelPostStartAction(app *DdevApp) error {
	if fileutil.FileExists(app.SiteSettingsPath) {
		isConfigured, err := fileutil.FgrepStringInFile(app.SiteSettingsPath, `DB_HOST=db`)
		if err == nil && !isConfigured {
			envSettingsWarning(WarnTypeNotConfigured)
		}
	} else {
		envSettingsWarning(WarnTypeAbsent)
	}

	return nil
}
