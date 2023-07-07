package ddevapp

import (
	"fmt"
	"github.com/pkg/errors"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
)

// appTypeFuncs prototypes
//
// settingsCreator
type settingsCreator func(*DdevApp) (string, error)

// uploadDirs
type uploadDirs func(*DdevApp) []string

// hookDefaultComments should probably change its arg from string to app when
// config refactor is done.
type hookDefaultComments func() []byte

// appTypeSettingsPaths
type appTypeSettingsPaths func(app *DdevApp)

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
type importFilesAction func(app *DdevApp, uploadDir, importPath, extractPath string) error

// defaultWorkingDirMap returns the app type's default working directory map
type defaultWorkingDirMap func(app *DdevApp, defaults map[string]string) map[string]string

// appTypeFuncs struct defines the functions that can be called (if populated)
// for a given appType.
type appTypeFuncs struct {
	settingsCreator
	uploadDirs
	hookDefaultComments
	appTypeSettingsPaths
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
var appTypeMatrix map[string]appTypeFuncs

func init() {
	appTypeMatrix = map[string]appTypeFuncs{
		nodeps.AppTypeBackdrop: {
			settingsCreator:      createBackdropSettingsFile,
			uploadDirs:           getBackdropUploadDirs,
			hookDefaultComments:  getBackdropHooks,
			appTypeSettingsPaths: setBackdropSiteSettingsPaths,
			appTypeDetect:        isBackdropApp,
			postImportDBAction:   backdropPostImportDBAction,
			postStartAction:      backdropPostStartAction,
			importFilesAction:    backdropImportFilesAction,
			defaultWorkingDirMap: docrootWorkingDir,
		},

		nodeps.AppTypeCraftCms: {
			importFilesAction:    craftCmsImportFilesAction,
			appTypeDetect:        isCraftCmsApp,
			configOverrideAction: craftCmsConfigOverrideAction,
			postStartAction:      craftCmsPostStartAction,
		},

		nodeps.AppTypeDjango4: {
			settingsCreator:      django4SettingsCreator,
			appTypeDetect:        isDjango4App,
			configOverrideAction: django4ConfigOverrideAction,
			postConfigAction:     django4PostConfigAction,
			postStartAction:      django4PostStartAction,
			importFilesAction:    genericImportFilesAction,
		},

		nodeps.AppTypeDrupal6: {
			settingsCreator:      createDrupalSettingsPHP,
			uploadDirs:           getDrupalUploadDirs,
			hookDefaultComments:  getDrupal6Hooks,
			appTypeSettingsPaths: setDrupalSiteSettingsPaths,
			appTypeDetect:        isDrupal6App,
			configOverrideAction: drupal6ConfigOverrideAction,
			postStartAction:      drupal6PostStartAction,
			importFilesAction:    drupalImportFilesAction,
			defaultWorkingDirMap: docrootWorkingDir,
		},

		nodeps.AppTypeDrupal7: {
			settingsCreator:      createDrupalSettingsPHP,
			uploadDirs:           getDrupalUploadDirs,
			hookDefaultComments:  getDrupal7Hooks,
			appTypeSettingsPaths: setDrupalSiteSettingsPaths,
			appTypeDetect:        isDrupal7App,
			postStartAction:      drupal7PostStartAction,
			importFilesAction:    drupalImportFilesAction,
			defaultWorkingDirMap: docrootWorkingDir,
		},

		nodeps.AppTypeDrupal8: {
			settingsCreator:      createDrupalSettingsPHP,
			uploadDirs:           getDrupalUploadDirs,
			hookDefaultComments:  getDrupal8Hooks,
			appTypeSettingsPaths: setDrupalSiteSettingsPaths,
			appTypeDetect:        isDrupal8App,
			configOverrideAction: drupal8ConfigOverrideAction,
			postStartAction:      drupal8PostStartAction,
			importFilesAction:    drupalImportFilesAction,
		},

		nodeps.AppTypeDrupal9: {
			settingsCreator:      createDrupalSettingsPHP,
			uploadDirs:           getDrupalUploadDirs,
			hookDefaultComments:  getDrupal8Hooks,
			appTypeSettingsPaths: setDrupalSiteSettingsPaths,
			appTypeDetect:        isDrupal9App,
			postStartAction:      drupalPostStartAction,
			importFilesAction:    drupalImportFilesAction,
		},

		nodeps.AppTypeDrupal10: {
			settingsCreator:      createDrupalSettingsPHP,
			uploadDirs:           getDrupalUploadDirs,
			hookDefaultComments:  getDrupal8Hooks,
			appTypeSettingsPaths: setDrupalSiteSettingsPaths,
			appTypeDetect:        isDrupal10App,
			configOverrideAction: drupal10ConfigOverrideAction,
			postStartAction:      drupalPostStartAction,
			importFilesAction:    drupalImportFilesAction,
		},

		nodeps.AppTypeLaravel: {
			appTypeDetect:        isLaravelApp,
			postStartAction:      laravelPostStartAction,
			configOverrideAction: laravelConfigOverrideAction,
			importFilesAction:    genericImportFilesAction,
		},

		nodeps.AppTypeSilverstripe: {
			appTypeDetect:        isSilverstripeApp,
			postStartAction:      silverstripePostStartAction,
			configOverrideAction: silverstripeConfigOverrideAction,
		},

		nodeps.AppTypeMagento: {
			settingsCreator:      createMagentoSettingsFile,
			uploadDirs:           getMagentoUploadDirs,
			appTypeSettingsPaths: setMagentoSiteSettingsPaths,
			appTypeDetect:        isMagentoApp,
			configOverrideAction: magentoConfigOverrideAction,
			importFilesAction:    magentoImportFilesAction,
		},

		nodeps.AppTypeMagento2: {
			settingsCreator:      createMagento2SettingsFile,
			uploadDirs:           getMagento2UploadDirs,
			appTypeSettingsPaths: setMagento2SiteSettingsPaths,
			appTypeDetect:        isMagento2App,
			configOverrideAction: magento2ConfigOverrideAction,
			importFilesAction:    magentoImportFilesAction,
		},

		nodeps.AppTypePHP: {
			postStartAction:   phpPostStartAction,
			importFilesAction: genericImportFilesAction,
		},

		nodeps.AppTypePython: {
			appTypeDetect:        isPythonApp,
			configOverrideAction: pythonConfigOverrideAction,
			postConfigAction:     pythonPostConfigAction,
			importFilesAction:    genericImportFilesAction,
		},

		nodeps.AppTypeShopware6: {
			appTypeDetect:        isShopware6App,
			appTypeSettingsPaths: setShopware6SiteSettingsPaths,
			uploadDirs:           getShopwareUploadDirs,
			postStartAction:      shopware6PostStartAction,
			importFilesAction:    shopware6ImportFilesAction,
		},

		nodeps.AppTypeTYPO3: {
			settingsCreator:      createTypo3SettingsFile,
			uploadDirs:           getTypo3UploadDirs,
			hookDefaultComments:  getTypo3Hooks,
			appTypeSettingsPaths: setTypo3SiteSettingsPaths,
			appTypeDetect:        isTypo3App,
			importFilesAction:    typo3ImportFilesAction,
		},

		nodeps.AppTypeWordPress: {
			settingsCreator:      createWordpressSettingsFile,
			uploadDirs:           getWordpressUploadDirs,
			hookDefaultComments:  getWordpressHooks,
			appTypeSettingsPaths: setWordpressSiteSettingsPaths,
			appTypeDetect:        isWordpressApp,
			importFilesAction:    wordpressImportFilesAction,
		},
	}
}

// CreateSettingsFile creates the settings file (like settings.php) for the
// provided app is the apptype has a settingsCreator function.
// It also preps the ddev directory, including setting up the .ddev gitignore
func (app *DdevApp) CreateSettingsFile() (string, error) {
	err := PrepDdevDirectory(app)
	if err != nil {
		util.Warning("Unable to PrepDdevDirectory: %v", err)
	}

	app.SetApptypeSettingsPaths()

	if app.DisableSettingsManagement && app.Type != nodeps.AppTypePHP {
		util.Warning("Not creating CMS settings files because disable_settings_management=true")
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
			util.Warning("Unable to create settings file '%s': %v", app.SiteSettingsPath, err)
		}

		// Don't create gitignore if it would be in top-level directory, where
		// there is almost certainly already a gitignore (like backdrop)
		if path.Dir(app.SiteSettingsPath) != app.AppRoot {
			if err = CreateGitIgnore(filepath.Dir(app.SiteSettingsPath), filepath.Base(app.SiteDdevSettingsFile), "drushrc.php"); err != nil {
				util.Warning("Failed to write .gitignore in %s: %v", filepath.Dir(app.SiteDdevSettingsFile), err)
			}
		}
		return settingsPath, nil
	}

	// If the project is not running, it makes no sense to sync it
	if s, _ := app.SiteStatus(); s == SiteRunning {
		err = app.MutagenSyncFlush()
		if err != nil {
			return "", err
		}
	}

	return "", nil
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
	if appFuncs, ok := appTypeMatrix[app.Type]; ok && appFuncs.appTypeSettingsPaths != nil {
		appFuncs.appTypeSettingsPaths(app)
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

	return nodeps.AppTypePHP
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

// dispatchImportFilesAction executes the relevant import files workflow for each app type.
func (app *DdevApp) dispatchImportFilesAction(uploadDir, importPath, extractPath string) error {
	if strings.TrimSpace(uploadDir) == "" {
		return errors.Errorf("upload_dirs is not set for this project (%s)", app.Type)
	}

	if appFuncs, ok := appTypeMatrix[app.Type]; ok && appFuncs.importFilesAction != nil {
		return appFuncs.importFilesAction(app, uploadDir, importPath, extractPath)
	}

	return fmt.Errorf("this project type (%s) does not support import-files", app.Type)
}

// DefaultWorkingDirMap returns the app type's default working directory map.
func (app *DdevApp) DefaultWorkingDirMap() map[string]string {
	_, _, username := util.GetContainerUIDGid()
	// Default working directory values are defined here.
	// Services working directories can be overridden by app types if needed.
	defaults := map[string]string{
		"web": "/var/www/html/",
		"db":  "/home/" + username,
	}

	if appFuncs, ok := appTypeMatrix[app.Type]; ok && appFuncs.defaultWorkingDirMap != nil {
		return appFuncs.defaultWorkingDirMap(app, defaults)
	}

	if app.Database.Type == nodeps.Postgres {
		defaults["db"] = "/var/lib/postgresql"
	}
	return defaults
}

// docrootWorkingDir handles the shared case in which the web service working directory is the docroot.
func docrootWorkingDir(app *DdevApp, defaults map[string]string) map[string]string {
	defaults["web"] = path.Join("/var/www/html", app.Docroot)

	return defaults
}

// IsValidAppType is a helper function to determine if an app type is valid, returning
// true if the given app type is valid and configured and false otherwise.
func IsValidAppType(apptype string) bool {
	if _, ok := appTypeMatrix[apptype]; !ok {
		return false
	}

	return true
}

// GetValidAppTypes returns the valid apptype keys from the appTypeMatrix
func GetValidAppTypes() []string {
	keys := make([]string, 0, len(appTypeMatrix))
	for k := range appTypeMatrix {
		keys = append(keys, k)
		sort.Strings(keys)
	}
	return keys
}
