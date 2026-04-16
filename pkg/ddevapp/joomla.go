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

// getJoomlaUploadDirs will return the default paths.
func getJoomlaUploadDirs(_ *DdevApp) []string {
	uploadDirs := []string{"images"}

	return uploadDirs
}
