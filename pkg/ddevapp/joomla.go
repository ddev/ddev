package ddevapp

import (
	"path/filepath"

	"github.com/ddev/ddev/pkg/fileutil"
)

// isJoomlaApp returns true if the app is of type joomla
func isJoomlaApp(app *DdevApp) bool {
	return fileutil.FileExists(filepath.Join(app.AppRoot, app.Docroot, "includes", "defines.php")) &&
		fileutil.FileExists(filepath.Join(app.AppRoot, app.Docroot, "administrator", "index.php"))
}
