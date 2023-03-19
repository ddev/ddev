package ddevapp

import (
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
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
	return nil
}
