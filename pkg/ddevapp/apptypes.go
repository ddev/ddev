package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"
)

type settingsCreator func(*DdevApp) (string, error)
type uploadDir func(*DdevApp) string

// hookSuggestions should probably change its arg from string to app when
// config refactor is done.
type hookSuggestions func() []byte

type apptypeSettingsPaths func(app *DdevApp)

// AppTypeFuncs struct defines the functions that can be called (if populated)
// for a given appType.
type AppTypeFuncs struct {
	settingsCreator
	uploadDir
	hookSuggestions
	apptypeSettingsPaths
}

// appTypeMatrix is a static map that defines the various functions to be called
// for each apptype (CMS).
// Example: appTypeMatrix["drupal"]["7"] == { settingsCreator etc }
var appTypeMatrix map[string]AppTypeFuncs

func init() {
	appTypeMatrix = map[string]AppTypeFuncs{
		"php": {},
		"drupal7": {
			createDrupalSettingsFile, getDrupalUploadDir, getDrupal7Hooks, setDrupalSiteSettingsPaths,
		},
		"drupal8": {
			createDrupalSettingsFile, getDrupalUploadDir, getDrupal8Hooks, setDrupalSiteSettingsPaths,
		},
		"wordpress": {
			createWordpressSettingsFile, getWordpressUploadDir, getWordpressHooks, setWordpressSiteSettingsPaths,
		},
		"backdrop": {},
		"typo3":    {},
	}
}

// GetValidAppTypes returns the valid apptype keys from the appTypeMatrix
func GetValidAppTypes() []string {
	keys := make([]string, 0, len(appTypeMatrix))
	for k := range appTypeMatrix {
		keys = append(keys, k)
	}
	return keys
}

// IsValidAppType checks to see if the given apptype string is a valid configured
// apptype.
func IsValidAppType(apptype string) bool {
	if _, ok := appTypeMatrix[apptype]; ok {
		return true
	}
	return false
}

// CreateSettingsFile creates the settings file (like settings.php) for the
// provided app is the apptype has a settingsCreator function.
func (app *DdevApp) CreateSettingsFile() (string, error) {
	app.SetApptypeSettingsPaths()

	// If neither settings file options are set, then don't continue
	if app.SiteLocalSettingsPath == "" && app.SiteSettingsPath == "" {
		return "", fmt.Errorf("Neither SiteLocalSettingsPath nor SiteSettingsPath is set")
	}

	// Drupal and WordPress love to change settings files to be unwriteable.
	// Chmod them to something we can work with in the event that they already
	// exist.
	chmodTargets := []string{filepath.Dir(app.SiteSettingsPath), app.SiteLocalSettingsPath}
	for _, fp := range chmodTargets {
		if fileInfo, err := os.Stat(fp); !os.IsNotExist(err) {
			perms := 0644
			if fileInfo.IsDir() {
				perms = 0755
			}

			err = os.Chmod(fp, os.FileMode(perms))
			if err != nil {
				return "", fmt.Errorf("could not change permissions on file %s to make it writeable: %v", fp, err)
			}
		}
	}

	// If we have a function to do the settings creation, do it, otherwise
	// just ignore.
	if appFuncs, ok := appTypeMatrix[app.GetType()]; ok && appFuncs.settingsCreator != nil {
		settingsPath, err := appFuncs.settingsCreator(app)
		return settingsPath, err
	}
	return "", nil
}

// GetUploadDir returns the upload (public files) directory for the given app
func (app *DdevApp) GetUploadDir() string {
	if appFuncs, ok := appTypeMatrix[app.GetType()]; ok && appFuncs.uploadDir != nil {
		uploadDir := appFuncs.uploadDir(app)
		return uploadDir
	}
	return ""
}

// GetHookSuggestions gets the actual text of the config.yaml hook suggestions
// for a given apptype
func (app *DdevApp) GetHookSuggestions() []byte {
	if appFuncs, ok := appTypeMatrix[app.Type]; ok && appFuncs.hookSuggestions != nil {
		suggestions := appFuncs.hookSuggestions()
		return suggestions
	}
	return []byte("")
}

// SetApptypeSettingsPaths chooses and sets the settings.php/settings.local.php
// and related paths for a given app.
func (app *DdevApp) SetApptypeSettingsPaths() {
	if appFuncs, ok := appTypeMatrix[app.Type]; ok && appFuncs.apptypeSettingsPaths != nil {
		appFuncs.apptypeSettingsPaths(app)
	}
}
