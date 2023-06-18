package ddevapp

import (
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"os"
	"path/filepath"
)

// isPythonApp returns true if the app is otherwise indeterminate python
func isPythonApp(app *DdevApp) bool {
	docroot := filepath.Join(app.AppRoot, app.Docroot)
	// Don't trigger on django4
	if fileutil.FileExists(filepath.Join(docroot, "manage.py")) {
		return false
	}

	files, err := os.ReadDir(docroot)
	if err != nil {
		return false
	}

	// If there are .py files in the docroot, assume python type
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".py" {
			return true
		}
	}
	return false
}

// pythonConfigOverrideAction sets up webserverType and anything else
// we might need for generic python.
func pythonConfigOverrideAction(app *DdevApp) error {
	if app.WebserverType == nodeps.WebserverDefault {
		app.WebserverType = nodeps.WebserverNginxGunicorn
	}
	if app.Database == DatabaseDefault {
		app.Database.Type = nodeps.Postgres
		app.Database.Version = nodeps.Postgres14
	}
	return nil
}

// pythonPostConfigAction just reminds people that they may need WSGI_APP env var
func pythonPostConfigAction(_ *DdevApp) error {
	util.Warning("Your project may need a WSGI_APP environment variable to work correctly")
	return nil
}
