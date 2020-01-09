package ddevapp

import (
	"os"
	"path/filepath"
)

// isMagentoApp returns true if the app is of type magento
func isMagentoApp(app *DdevApp) bool {
	if _, err := os.Stat(filepath.Join(app.AppRoot, "get.php")); err == nil {
		return true
	}
	return false
}

// isMagento2App returns true if the app is of type magento2
func isMagento2App(app *DdevApp) bool {
	if _, err := os.Stat(filepath.Join(app.AppRoot, "pub", "static.php")); err == nil {
		return true
	}
	return false
}
