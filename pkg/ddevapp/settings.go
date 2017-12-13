package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"
)

// setUpSettingsFile does the grunt work of determining which and where the
// settings file will be created, then sets permissions so it can do the right
// thing.
func setUpSettingsFile(app *DdevApp) (string, error) {

	// If neither settings file options are set, then
	if app.SiteLocalSettingsPath == "" && app.SiteSettingsPath == "" {
		return "", fmt.Errorf("Neither SiteLocalSettingsPath nor SiteSettingsPath is set")
	}

	settingsFilePath, err := app.DetermineDrupalSettingsPath()
	if err != nil {
		return "", err
	}

	// Drupal and WordPress love to change settings files to be unwriteable.
	// Chmod them to something we can work with in the event that they already
	// exist.
	chmodTargets := []string{filepath.Dir(settingsFilePath), settingsFilePath}
	for _, fp := range chmodTargets {
		if fileInfo, err := os.Stat(fp); !os.IsNotExist(err) {
			perms := 0644
			if fileInfo.IsDir() {
				perms = 0755
			}

			err = os.Chmod(fp, os.FileMode(perms))
			if err != nil {
				return "", fmt.Errorf("could not change permissions on %s to make the file writeable: %v", fp, err)
			}
		}
	}
	return settingsFilePath, nil
}
