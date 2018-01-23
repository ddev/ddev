package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"
)

type settingsCreator func(*DdevApp) (string, error)
type uploadDir func(*DdevApp) string

// hookDefaultComments should probably change its arg from string to app when
// config refactor is done.
type hookDefaultComments func() []byte

type apptypeSettingsPaths func(app *DdevApp)

// appTypeDetect returns true if the app is of the specified type
type appTypeDetect func(app *DdevApp) bool

// postImportDBAction can take actions after import (like warning user about
// required actions on Wordpress.
type postImportDBAction func(app *DdevApp) error

// configOverrideAction allows a particular apptype to override elements
// of the config for that apptype. Key example is drupal6 needing php56
type configOverrideAction func(app *DdevApp) error

// postConfigAction allows actions to take place at the end of ddev config
type postConfigAction func(app *DdevApp) error

// AppTypeFuncs struct defines the functions that can be called (if populated)
// for a given appType.
type AppTypeFuncs struct {
	settingsCreator
	uploadDir
	hookDefaultComments
	apptypeSettingsPaths
	appTypeDetect
	postImportDBAction
	configOverrideAction
	postConfigAction
}

// appTypeMatrix is a static map that defines the various functions to be called
// for each apptype (CMS).
// Example: appTypeMatrix["drupal"]["7"] == { settingsCreator etc }
var appTypeMatrix map[string]AppTypeFuncs

func init() {
	appTypeMatrix = map[string]AppTypeFuncs{
		"php": {},
		"drupal7": {
			createDrupalSettingsFile, getDrupalUploadDir, getDrupal7Hooks, setDrupalSiteSettingsPaths, isDrupal7App, nil, drupal7ConfigOverrideAction, nil,
		},
		"drupal8": {
			createDrupalSettingsFile, getDrupalUploadDir, getDrupal8Hooks, setDrupalSiteSettingsPaths, isDrupal8App, nil, nil, nil,
		},
		"wordpress": {
			createWordpressSettingsFile, getWordpressUploadDir, getWordpressHooks, setWordpressSiteSettingsPaths, isWordpressApp, wordpressPostImportDBAction, nil, nil,
		},
		"drupal6": {
			createDrupal6SettingsFile, getDrupalUploadDir, getDrupal6Hooks, setDrupalSiteSettingsPaths, isDrupal6App, nil, drupal6ConfigOverrideAction, drupal6PostConfigAction,
		},
		"typo3": {
			createTypo3SettingsFile, getTypo3UploadDir, getTypo3Hooks, setTypo3SiteSettingsPaths, isTypo3App, nil, nil, nil,
		},
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

// GetHookDefaultComments gets the actual text of the config.yaml hook suggestions
// for a given apptype
func (app *DdevApp) GetHookDefaultComments() []byte {
	if appFuncs, ok := appTypeMatrix[app.Type]; ok && appFuncs.hookDefaultComments != nil {
		suggestions := appFuncs.hookDefaultComments()
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

// DetectAppType calls each apptype's detector until it finds a match,
// or returns 'php' as a last resort.
func (app *DdevApp) DetectAppType() string {
	for appName, appFuncs := range appTypeMatrix {
		if appFuncs.appTypeDetect != nil && appFuncs.appTypeDetect(app) {
			return appName
		}
	}
	return "php"
}

// PostImportDBAction calls each apptype's detector until it finds a match,
// or returns 'php' as a last resort.
func (app *DdevApp) PostImportDBAction() error {

	if appFuncs, ok := appTypeMatrix[app.Type]; ok && appFuncs.postImportDBAction != nil {
		return appFuncs.postImportDBAction(app)
	}

	return nil
}

// ConfigFileOverrideAction gives a chance for an apptype to override any element
// of config.yaml that it needs to.
func (app *DdevApp) ConfigFileOverrideAction() error {
	if appFuncs, ok := appTypeMatrix[app.Type]; ok && appFuncs.configOverrideAction != nil {
		return appFuncs.configOverrideAction(app)
	}

	return nil
}

// PostConfigAction gives a chance for an apptype to override do something at
// the end of ddev config.
func (app *DdevApp) PostConfigAction() error {
	if appFuncs, ok := appTypeMatrix[app.Type]; ok && appFuncs.postConfigAction != nil {
		return appFuncs.postConfigAction(app)
	}

	// In the future if these get more specialized we may want to add this explicitly
	// to the apptype matrix instead of doing it here.
	return genericPostConfigAction(app)
}
