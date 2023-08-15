package ddevapp

import (
	"fmt"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"

	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/ddev/ddev/pkg/fileutil"

	"github.com/ddev/ddev/pkg/archive"
)

// DrupalSettings encapsulates all the configurations for a Drupal site.
type DrupalSettings struct {
	DeployName       string
	DeployURL        string
	DatabaseName     string
	DatabaseUsername string
	DatabasePassword string
	DatabaseHost     string
	DatabaseDriver   string
	DatabasePort     string
	HashSalt         string
	Signature        string
	SitePath         string
	SiteSettings     string
	SiteSettingsDdev string
	SyncDir          string
	DockerIP         string
	DBPublishedPort  int
}

// NewDrupalSettings produces a DrupalSettings object with default.
func NewDrupalSettings(app *DdevApp) *DrupalSettings {
	dockerIP, _ := dockerutil.GetDockerIP()
	dbPublishedPort, _ := app.GetPublishedPort("db")

	settings := &DrupalSettings{
		DatabaseName:     "db",
		DatabaseUsername: "db",
		DatabasePassword: "db",
		DatabaseHost:     "db",
		DatabaseDriver:   "mysql",
		DatabasePort:     GetExposedPort(app, "db"),
		HashSalt:         util.HashSalt(app.Name),
		Signature:        nodeps.DdevFileSignature,
		SitePath:         path.Join("sites", "default"),
		SiteSettings:     "settings.php",
		SiteSettingsDdev: "settings.ddev.php",
		SyncDir:          path.Join("files", "sync"),
		DockerIP:         dockerIP,
		DBPublishedPort:  dbPublishedPort,
	}
	if app.Type == "drupal6" {
		settings.DatabaseDriver = "mysqli"
	}
	if app.Database.Type == nodeps.Postgres {
		settings.DatabaseDriver = "pgsql"
	}
	return settings
}

// settingsIncludeStanza defines the template that will be appended to
// a project's settings.php in the event that the file already exists.
const settingsIncludeStanza = `
// Automatically generated include for settings managed by ddev.
$ddev_settings = dirname(__FILE__) . '/settings.ddev.php';
if (getenv('IS_DDEV_PROJECT') == 'true' && is_readable($ddev_settings)) {
  require $ddev_settings;
}
`

// manageDrupalSettingsFile will direct inspecting and writing of settings.php.
func manageDrupalSettingsFile(app *DdevApp, drupalConfig *DrupalSettings, appType string) error {
	// We'll be writing/appending to the settings files and parent directory, make sure we have permissions to do so
	if err := drupalEnsureWritePerms(app); err != nil {
		return err
	}

	if !fileutil.FileExists(app.SiteSettingsPath) {
		output.UserOut.Printf("No %s file exists, creating one", drupalConfig.SiteSettings)

		if err := writeDrupalSettingsPHP(app.SiteSettingsPath, appType); err != nil {
			return fmt.Errorf("failed to write: %v", err)
		}
	}

	included, err := settingsHasInclude(drupalConfig, app.SiteSettingsPath)
	if err != nil {
		return fmt.Errorf("failed to check for include: %v", err)
	}

	if included {
		util.Debug("Existing %s file includes %s", drupalConfig.SiteSettings, drupalConfig.SiteSettingsDdev)
	} else {
		util.Debug("Existing %s file does not include %s, modifying to include ddev settings", drupalConfig.SiteSettings, drupalConfig.SiteSettingsDdev)

		if err := appendIncludeToDrupalSettingsFile(app.SiteSettingsPath, app.Type); err != nil {
			return fmt.Errorf("failed to include %s in %s: %v", drupalConfig.SiteSettingsDdev, drupalConfig.SiteSettings, err)
		}
	}

	return nil
}

// writeDrupalSettingsPHP creates the project's settings.php if it doesn't exist
func writeDrupalSettingsPHP(filePath string, appType string) error {
	content, err := bundledAssets.ReadFile(path.Join("drupal", appType, "settings.php"))
	if err != nil {
		return err
	}

	// Ensure target directory exists and is writable
	dir := filepath.Dir(filePath)
	if err = os.Chmod(dir, 0755); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// Create file
	err = os.WriteFile(filePath, content, 0755)
	if err != nil {
		return err
	}

	return nil
}

// createDrupalSettingsPHP manages creation and modification of settings.php and settings.ddev.php.
// If a settings.php file already exists, it will be modified to ensure that it includes
// settings.ddev.php, which contains ddev-specific configuration.
func createDrupalSettingsPHP(app *DdevApp) (string, error) {
	// Currently there isn't any customization done for the drupal config, but
	// we may want to do some kind of customization in the future.
	drupalConfig := NewDrupalSettings(app)

	if err := manageDrupalSettingsFile(app, drupalConfig, app.Type); err != nil {
		return "", err
	}

	if err := writeDrupalSettingsDdevPhp(drupalConfig, app.SiteDdevSettingsFile, app); err != nil {
		return "", fmt.Errorf("`failed to write` Drupal settings file %s: %v", app.SiteDdevSettingsFile, err)
	}

	return app.SiteDdevSettingsFile, nil
}

// writeDrupalSettingsDdevPhp dynamically produces valid settings.ddev.php file by combining a configuration
// object with a data-driven template.
func writeDrupalSettingsDdevPhp(settings *DrupalSettings, filePath string, app *DdevApp) error {
	if fileutil.FileExists(filePath) {
		// Check if the file is managed by ddev.
		signatureFound, err := fileutil.FgrepStringInFile(filePath, nodeps.DdevFileSignature)
		if err != nil {
			return err
		}

		// If the signature wasn't found, warn the user and return.
		if !signatureFound {
			util.Warning("%s already exists and is managed by the user.", filepath.Base(filePath))
			return nil
		}
	}

	t, err := template.New("settings.ddev.php").ParseFS(bundledAssets, path.Join("drupal", app.Type, "settings.ddev.php"))
	if err != nil {
		return err
	}

	// Ensure target directory exists and is writable
	dir := filepath.Dir(filePath)
	if err = os.Chmod(dir, 0755); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	err = t.Execute(file, settings)
	if err != nil {
		return err
	}
	util.CheckClose(file)
	return nil
}

// WriteDrushrc writes out drushrc.php based on passed-in values.
// This works on Drupal 6 and Drupal 7 or with drush8 and older
func WriteDrushrc(app *DdevApp, filePath string) error {
	if fileutil.FileExists(filePath) {
		// Check if the file is managed by ddev.
		signatureFound, err := fileutil.FgrepStringInFile(filePath, nodeps.DdevFileSignature)
		if err != nil {
			return err
		}

		// If the signature wasn't found, warn the user and return.
		if !signatureFound {
			util.Warning("%s already exists and is managed by the user.", filepath.Base(filePath))
			return nil
		}
	}

	uri := app.GetPrimaryURL()
	drushContents := []byte(`<?php

/**
 * @file
 * ` + nodeps.DdevFileSignature + `: Automatically generated drushrc.php file (for Drush 8)
 * ddev manages this file and may delete or overwrite the file unless this comment is removed.
 * Remove this comment if you don't want ddev to manage this file.
 */

if (getenv('IS_DDEV_PROJECT') == 'true') {
  $options['l'] = "` + uri + `";
}
`)

	// Ensure target directory exists and is writable
	dir := filepath.Dir(filePath)
	if err := os.Chmod(dir, 0755); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	err := os.WriteFile(filePath, drushContents, 0666)
	if err != nil {
		return err
	}

	return nil
}

// getDrupalUploadDirs will return the default paths.
func getDrupalUploadDirs(_ *DdevApp) []string {
	uploadDirs := []string{"sites/default/files"}

	return uploadDirs
}

// Drupal8Hooks adds a d8-specific hooks example for post-import-db
const Drupal8Hooks = `# post-import-db:
#   - exec: drush cr
#   - exec: drush updb
`

// Drupal7Hooks adds a d7-specific hooks example for post-import-db
const Drupal7Hooks = `#  post-import-db:
#    - exec: drush cc all
`

// getDrupal7Hooks for appending as byte array
func getDrupal7Hooks() []byte {
	return []byte(Drupal7Hooks)
}

// getDrupal6Hooks for appending as byte array
func getDrupal6Hooks() []byte {
	// We don't have anything new to add yet, so just use Drupal7 version
	return []byte(Drupal7Hooks)
}

// getDrupal8Hooks for appending as byte array
func getDrupal8Hooks() []byte {
	return []byte(Drupal8Hooks)
}

// setDrupalSiteSettingsPaths sets the paths to settings.php/settings.ddev.php
// for templating.
func setDrupalSiteSettingsPaths(app *DdevApp) {
	drupalConfig := NewDrupalSettings(app)
	settingsFileBasePath := filepath.Join(app.AppRoot, app.Docroot)
	app.SiteSettingsPath = filepath.Join(settingsFileBasePath, drupalConfig.SitePath, drupalConfig.SiteSettings)
	app.SiteDdevSettingsFile = filepath.Join(settingsFileBasePath, drupalConfig.SitePath, drupalConfig.SiteSettingsDdev)
}

// isDrupal7App returns true if the app is of type drupal7
func isDrupal7App(app *DdevApp) bool {
	if _, err := os.Stat(filepath.Join(app.AppRoot, app.Docroot, "misc/ajax.js")); err == nil {
		return true
	}
	return false
}

// isDrupal8App returns true if the app is of type drupal8
func isDrupal8App(app *DdevApp) bool {
	isD8, err := fileutil.FgrepStringInFile(filepath.Join(app.AppRoot, app.Docroot, "core/lib/Drupal.php"), `const VERSION = '8`)
	if err == nil && isD8 {
		return true
	}
	return false
}

// isDrupal9App returns true if the app is of type drupal9
func isDrupal9App(app *DdevApp) bool {
	isD9, err := fileutil.FgrepStringInFile(filepath.Join(app.AppRoot, app.Docroot, "core/lib/Drupal.php"), `const VERSION = '9`)
	if err == nil && isD9 {
		return true
	}
	return false
}

// isDrupal10App returns true if the app is of type drupal10
func isDrupal10App(app *DdevApp) bool {
	isD10, err := fileutil.FgrepStringInFile(filepath.Join(app.AppRoot, app.Docroot, "core/lib/Drupal.php"), `const VERSION = '10`)
	if err == nil && isD10 {
		return true
	}
	return false
}

// isDrupal6App returns true if the app is of type Drupal6
func isDrupal6App(app *DdevApp) bool {
	if _, err := os.Stat(filepath.Join(app.AppRoot, app.Docroot, "misc/ahah.js")); err == nil {
		return true
	}
	return false
}

// drupal6ConfigOverrideAction overrides php_version for D6, since it is incompatible
// with php7+
func drupal6ConfigOverrideAction(app *DdevApp) error {
	app.PHPVersion = nodeps.PHP56
	return nil
}

func drupal8ConfigOverrideAction(app *DdevApp) error {
	app.PHPVersion = nodeps.PHP74
	return nil
}

// drupal0ConfigOverrideAction overrides php_version for D10, requires PHP8.0
//func drupal9ConfigOverrideAction(app *DdevApp) error {
//	app.PHPVersion = nodeps.PHP80
//	return nil
//}

// drupal10ConfigOverrideAction overrides php_version for D10, requires PHP8.0
func drupal10ConfigOverrideAction(app *DdevApp) error {
	app.PHPVersion = nodeps.PHP81
	return nil
}

// drupal8PostStartAction handles default post-start actions for D8 apps, like ensuring
// useful permissions settings on sites/default.
func drupal8PostStartAction(app *DdevApp) error {
	// Return early because we aren't expected to manage settings.
	if app.DisableSettingsManagement {
		return nil
	}
	if err := createDrupal8SyncDir(app); err != nil {
		return err
	}

	//nolint: revive
	if err := drupalEnsureWritePerms(app); err != nil {
		return err
	}
	return nil
}

func drupalPostStartAction(app *DdevApp) error {
	if isDrupal9App(app) || isDrupal10App(app) {
		err := app.Wait([]string{nodeps.DBContainer})
		if err != nil {
			return err
		}
		// pg_trm extension is required in Drupal9.5+
		if app.Database.Type == nodeps.Postgres {
			stdout, stderr, err := app.Exec(&ExecOpts{
				Service:   "db",
				Cmd:       `psql -q -c "CREATE EXTENSION IF NOT EXISTS pg_trgm;" 2>/dev/null`,
				NoCapture: false,
			})
			if err != nil {
				util.Warning("unable to CREATE EXTENSION pg_trm: stdout='%s', stderr='%s', err=%v", stdout, stderr, err)
			}
		}
		// SET GLOBAL TRANSACTION ISOLATION LEVEL READ COMMITTED required in Drupal 9.5+
		if app.Database.Type == nodeps.MariaDB || app.Database.Type == nodeps.MySQL {
			stdout, stderr, err := app.Exec(&ExecOpts{
				Service:   "db",
				Cmd:       `mysql -uroot -proot -e "SET GLOBAL TRANSACTION ISOLATION LEVEL READ COMMITTED;"`,
				NoCapture: false,
			})
			if err != nil {
				util.Warning("unable to SET GLOBAL TRANSACTION ISOLATION LEVEL READ COMMITTED: stdout='%s', stderr='%s', err=%v", stdout, stderr, err)
			}
		}
	}
	// Return early because we aren't expected to manage settings.
	if app.DisableSettingsManagement {
		return nil
	}
	if err := createDrupal8SyncDir(app); err != nil {
		return err
	}

	//nolint: revive
	if err := drupalEnsureWritePerms(app); err != nil {
		return err
	}
	return nil
}

// drupal7PostStartAction handles default post-start actions for D7 apps, like ensuring
// useful permissions settings on sites/default.
func drupal7PostStartAction(app *DdevApp) error {
	// Return early because we aren't expected to manage settings.
	if app.DisableSettingsManagement {
		return nil
	}
	if err := drupalEnsureWritePerms(app); err != nil {
		return err
	}

	err := WriteDrushrc(app, filepath.Join(filepath.Dir(app.SiteSettingsPath), "drushrc.php"))
	if err != nil {
		util.Warning("Failed to WriteDrushrc: %v", err)
	}

	return nil
}

// drupal6PostStartAction handles default post-start actions for D6 apps, like ensuring
// useful permissions settings on sites/default.
func drupal6PostStartAction(app *DdevApp) error {
	// Return early because we aren't expected to manage settings.
	if app.DisableSettingsManagement {
		return nil
	}

	if err := drupalEnsureWritePerms(app); err != nil {
		return err
	}

	err := WriteDrushrc(app, filepath.Join(filepath.Dir(app.SiteSettingsPath), "drushrc.php"))
	if err != nil {
		util.Warning("Failed to WriteDrushrc: %v", err)
	}
	return nil
}

// drupalEnsureWritePerms will ensure sites/default and sites/default/settings.php will
// have the appropriate permissions for development.
func drupalEnsureWritePerms(app *DdevApp) error {
	util.Debug("Ensuring write permissions for %s", app.GetName())
	var writePerms os.FileMode = 0200

	settingsDir := path.Dir(app.SiteSettingsPath)
	makeWritable := []string{
		settingsDir,
		app.SiteSettingsPath,
		app.SiteDdevSettingsFile,
		path.Join(settingsDir, "services.yml"),
	}

	for _, o := range makeWritable {
		stat, err := os.Stat(o)
		if err != nil {
			if !os.IsNotExist(err) {
				util.Warning("Unable to ensure write permissions: %v", err)
			}

			continue
		}

		if err := os.Chmod(o, stat.Mode()|writePerms); err != nil {
			// Warn the user, but continue.
			util.Warning("Unable to set permissions: %v", err)
		}
	}

	return nil
}

// createDrupal8SyncDir creates a Drupal 8 app's sync directory
func createDrupal8SyncDir(app *DdevApp) error {
	// Currently there isn't any customization done for the drupal config, but
	// we may want to do some kind of customization in the future.
	drupalConfig := NewDrupalSettings(app)

	syncDirPath := path.Join(app.GetAppRoot(), app.GetDocroot(), "sites/default", drupalConfig.SyncDir)
	if fileutil.FileExists(syncDirPath) {
		return nil
	}

	if err := os.MkdirAll(syncDirPath, 0755); err != nil {
		return fmt.Errorf("failed to create sync directory (%s): %v", syncDirPath, err)
	}

	return nil
}

// settingsHasInclude determines if the settings.php or equivalent includes settings.ddev.php or equivalent.
// This is done by looking for the ddev settings file (settings.ddev.php) in settings.php.
func settingsHasInclude(drupalConfig *DrupalSettings, siteSettingsPath string) (bool, error) {
	included, err := fileutil.FgrepStringInFile(siteSettingsPath, drupalConfig.SiteSettingsDdev)
	if err != nil {
		return false, err
	}

	return included, nil
}

// appendIncludeToDrupalSettingsFile modifies the settings.php file to include the settings.ddev.php
// file, which contains ddev-specific configuration.
func appendIncludeToDrupalSettingsFile(siteSettingsPath string, appType string) error {
	// Check if file is empty
	contents, err := os.ReadFile(siteSettingsPath)
	if err != nil {
		return err
	}

	// If the file is empty, write the complete settings file and return
	if len(contents) == 0 {
		return writeDrupalSettingsPHP(siteSettingsPath, appType)
	}

	// The file is not empty, open it for appending
	file, err := os.OpenFile(siteSettingsPath, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer util.CheckClose(file)

	_, err = file.Write([]byte(settingsIncludeStanza))
	if err != nil {
		return err
	}
	return nil
}

// drupalImportFilesAction defines the Drupal workflow for importing project files.
func drupalImportFilesAction(app *DdevApp, uploadDir, importPath, extPath string) error {
	destPath := app.calculateHostUploadDirFullPath(uploadDir)

	// parent of destination dir should exist
	if !fileutil.FileExists(filepath.Dir(destPath)) {
		return fmt.Errorf("unable to import to %s: parent directory does not exist", destPath)
	}

	// parent of destination dir should be writable.
	if err := os.Chmod(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// If the destination path exists, remove it as was warned
	if fileutil.FileExists(destPath) {
		if err := os.RemoveAll(destPath); err != nil {
			return fmt.Errorf("failed to cleanup %s before import: %v", destPath, err)
		}
	}

	if isTar(importPath) {
		if err := archive.Untar(importPath, destPath, extPath); err != nil {
			return fmt.Errorf("failed to extract provided archive: %v", err)
		}

		return nil
	}

	if isZip(importPath) {
		if err := archive.Unzip(importPath, destPath, extPath); err != nil {
			return fmt.Errorf("failed to extract provided archive: %v", err)
		}

		return nil
	}

	//nolint: revive
	if err := fileutil.CopyDir(importPath, destPath); err != nil {
		return err
	}

	return nil
}
