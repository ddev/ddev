package ddevapp

import (
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"path/filepath"
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
func django4PostConfigAction(app *DdevApp) error {
	util.Warning("Your project may need a DJANGO_SETTINGS_MODULE environment variable to work correctly")
	return nil
}
