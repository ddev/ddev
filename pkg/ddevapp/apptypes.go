package ddevapp

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/drud/ddev/pkg/util"
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

// postStartAction allows actions to take place at the end of ddev start
type postStartAction func(app *DdevApp) error

// importFilesAction
type importFilesAction func(app *DdevApp, importPath, extPath string) error

// defaultWorkingDirMap returns the app type's default working directory map
type defaultWorkingDirMap func(app *DdevApp, defaults map[string]string) map[string]string

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
	postStartAction
	importFilesAction
	defaultWorkingDirMap
}

// appTypeMatrix is a static map that defines the various functions to be called
// for each apptype (CMS).
var appTypeMatrix map[string]AppTypeFuncs

func init() {
	appTypeMatrix = map[string]AppTypeFuncs{
		AppTypePHP: {},
		AppTypeDrupal6: {
			settingsCreator: createDrupal6SettingsFile, uploadDir: getDrupalUploadDir, hookDefaultComments: getDrupal6Hooks, apptypeSettingsPaths: setDrupalSiteSettingsPaths, appTypeDetect: isDrupal6App, postImportDBAction: nil, configOverrideAction: drupal6ConfigOverrideAction, postConfigAction: nil, postStartAction: drupal6PostStartAction, importFilesAction: drupalImportFilesAction, defaultWorkingDirMap: docrootWorkingDir,
		},
		AppTypeDrupal7: {
			settingsCreator: createDrupal7SettingsFile, uploadDir: getDrupalUploadDir, hookDefaultComments: getDrupal7Hooks, apptypeSettingsPaths: setDrupalSiteSettingsPaths, appTypeDetect: isDrupal7App, postImportDBAction: nil, configOverrideAction: nil, postConfigAction: nil, postStartAction: drupal7PostStartAction, importFilesAction: drupalImportFilesAction, defaultWorkingDirMap: docrootWorkingDir,
		},
		AppTypeDrupal8: {
			settingsCreator: createDrupal8SettingsFile, uploadDir: getDrupalUploadDir, hookDefaultComments: getDrupal8Hooks, apptypeSettingsPaths: setDrupalSiteSettingsPaths, appTypeDetect: isDrupal8App, postImportDBAction: nil, configOverrideAction: nil, postConfigAction: nil, postStartAction: drupal8PostStartAction, importFilesAction: drupalImportFilesAction, defaultWorkingDirMap: docrootWorkingDir,
		},
		AppTypeWordPress: {
			settingsCreator: createWordpressSettingsFile, uploadDir: getWordpressUploadDir, hookDefaultComments: getWordpressHooks, apptypeSettingsPaths: setWordpressSiteSettingsPaths, appTypeDetect: isWordpressApp, postImportDBAction: nil, configOverrideAction: nil, postConfigAction: nil, postStartAction: wordpressPostStartAction, importFilesAction: wordpressImportFilesAction,
		},
		AppTypeTYPO3: {
			settingsCreator: createTypo3SettingsFile, uploadDir: getTypo3UploadDir, hookDefaultComments: getTypo3Hooks, apptypeSettingsPaths: setTypo3SiteSettingsPaths, appTypeDetect: isTypo3App, postImportDBAction: nil, configOverrideAction: typo3ConfigOverrideAction, postConfigAction: nil, postStartAction: typo3PostStartAction, importFilesAction: typo3ImportFilesAction,
		},
		AppTypeBackdrop: {
			settingsCreator: createBackdropSettingsFile, uploadDir: getBackdropUploadDir, hookDefaultComments: getBackdropHooks, apptypeSettingsPaths: setBackdropSiteSettingsPaths, appTypeDetect: isBackdropApp, postImportDBAction: backdropPostImportDBAction, configOverrideAction: nil, postConfigAction: nil, postStartAction: backdropPostStartAction, importFilesAction: backdropImportFilesAction, defaultWorkingDirMap: docrootWorkingDir,
		},
	}
}

// CreateSettingsFile creates the settings file (like settings.php) for the
// provided app is the apptype has a settingsCreator function.
func (app *DdevApp) CreateSettingsFile() (string, error) {
	app.SetApptypeSettingsPaths()

	// If neither settings file options are set, then don't continue. Return
	// a nil error because this should not halt execution if the apptype
	// does not have a settings definition.
	if app.SiteDdevSettingsFile == "" && app.SiteSettingsPath == "" {
		util.Warning("Project type has no settings paths configured, so not creating settings file.")
		return "", nil
	}

	// Drupal and WordPress love to change settings files to be unwriteable.
	// Chmod them to something we can work with in the event that they already
	// exist.
	chmodTargets := []string{filepath.Dir(app.SiteSettingsPath), app.SiteDdevSettingsFile}
	for _, fp := range chmodTargets {
		fileInfo, err := os.Stat(fp)
		if err != nil {
			// We're not doing anything about this error other than warning,
			// and will have to deal with the same check in settingsCreator.
			if !os.IsNotExist(err) {
				util.Warning("Unable to ensure write permissions: %v", err)
			}

			continue
		}

		perms := 0644
		if fileInfo.IsDir() {
			perms = 0755
		}

		err = os.Chmod(fp, os.FileMode(perms))
		if err != nil {
			return "", fmt.Errorf("could not change permissions on file %s to make it writeable: %v", fp, err)
		}
	}

	// If we have a function to do the settings creation, do it, otherwise
	// just ignore.
	if appFuncs, ok := appTypeMatrix[app.GetType()]; ok && appFuncs.settingsCreator != nil {
		settingsPath, err := appFuncs.settingsCreator(app)
		if err != nil {
			util.Warning("Unable to create settings file: %v", err)
		}
		if err := CreateGitIgnore(filepath.Dir(app.SiteSettingsPath), filepath.Base(app.SiteDdevSettingsFile), "ddev_drush_settings.php"); err != nil {
			util.Warning("Failed to write .gitignore in %s: %v", filepath.Dir(app.SiteDdevSettingsFile), err)
		}

		return settingsPath, nil
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
	return AppTypePHP
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
// of config.yaml that it needs to (on initial creation, but not after that)
func (app *DdevApp) ConfigFileOverrideAction() error {
	if appFuncs, ok := appTypeMatrix[app.Type]; ok && appFuncs.configOverrideAction != nil && !app.ConfigExists() {
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

	return nil
}

// PostStartAction gives a chance for an apptype to do something after the app
// has been started.
func (app *DdevApp) PostStartAction() error {
	if appFuncs, ok := appTypeMatrix[app.Type]; ok && appFuncs.postStartAction != nil {
		return appFuncs.postStartAction(app)
	}

	return nil
}

// ImportFilesAction executes the relevant import files workflow for each app type.
func (app *DdevApp) ImportFilesAction(importPath, extPath string) error {
	if appFuncs, ok := appTypeMatrix[app.Type]; ok && appFuncs.importFilesAction != nil {
		return appFuncs.importFilesAction(app, importPath, extPath)
	}

	return fmt.Errorf("this project type (%s) does not support import-files", app.Type)
}

// DefaultWorkingDirMap returns the app type's default working directory map.
func (app *DdevApp) DefaultWorkingDirMap() map[string]string {
	// Default working directory values are defined here.
	// Services working directories can be overridden by app types if needed.
	defaults := map[string]string{
		"web": "/var/www/html/",
		"db":  "/home",
		"dba": "/home",
	}

	if appFuncs, ok := appTypeMatrix[app.Type]; ok && appFuncs.defaultWorkingDirMap != nil {
		return appFuncs.defaultWorkingDirMap(app, defaults)
	}

	return defaults
}

// docrootWorkingDir handles the shared case in which the web service working directory is the docroot.
func docrootWorkingDir(app *DdevApp, defaults map[string]string) map[string]string {
	defaults["web"] = path.Join("/var/www/html", app.Docroot)

	return defaults
}
