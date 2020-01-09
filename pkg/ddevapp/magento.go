package ddevapp

import (
	"os"
	"path/filepath"
)

// isMagento2App returns true if the app is of type magento
func isMagento2App(app *DdevApp) bool {
	if _, err := os.Stat(filepath.Join(app.AppRoot, "pub", "static.php")); err == nil {
		return true
	}
	return false
}
