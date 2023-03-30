package ddevapp

import (
	"fmt"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// isDjango4App returns true if the app is of type django4
func isDjango4App(app *DdevApp) bool {
	return fileutil.FileExists(filepath.Join(app.AppRoot, app.Docroot, "manage.py"))
}

// django4ConfigOverrideAction sets up webserverType and anything else
// we might need for django.
func django4ConfigOverrideAction(app *DdevApp) error {
	if app.WebserverType == nodeps.WebserverDefault {
		app.WebserverType = nodeps.WebserverNginxGunicorn
	}
	if app.Database == DatabaseDefault {
		app.Database.Type = nodeps.Postgres
		app.Database.Version = nodeps.Postgres14
	}

	return nil
}

// django4PostConfigAction just reminds people that they may need DJANGO_SETTINGS_MODULE env var
func django4PostConfigAction(_ *DdevApp) error {
	util.Warning("Your project may need a DJANGO_SETTINGS_MODULE environment variable to work correctly")
	return nil
}

// django4PostStartAction handles creating settings for the project
func django4PostStartAction(app *DdevApp) error {
	// Return early because we aren't expected to manage settings.
	if app.DisableSettingsManagement {
		return nil
	}

	// Sort out what they need
	settingsFile, _, err := app.Exec(&ExecOpts{
		Cmd: "find-django-settings-file.py",
	})
	if err != nil {
		return err
	}

	settingsFile = strings.Trim(settingsFile, "\n")
	settingsFile = strings.TrimPrefix(settingsFile, "/var/www/html/")

	settingsDdevPy := path.Join("/var/www/html/.ddev", "settings.ddev.py")
	django4SettingsIncludeStanza := fmt.Sprintf(`
    if os.environ.get('IS_DDEV_PROJECT') == 'true':
		s = Path(%s)
        if s.is_file():
            from s import *
	`, settingsDdevPy)

	// Add the inclusion
	file, err := os.OpenFile(settingsFile, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer util.CheckClose(file)

	_, err = file.Write([]byte(django4SettingsIncludeStanza))
	if err != nil {
		return err
	}

	// Add the settings.django.py; should use the type of db we're using

	err = app.MutagenSyncFlush()
	return err
}
