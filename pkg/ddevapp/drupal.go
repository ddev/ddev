package ddevapp

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	copy2 "github.com/otiai10/copy"
)

// DefaultDrupalSettingsVersion is the version used for settings.php/settings.ddev.php
// when no known Drupal version is detected
const DefaultDrupalSettingsVersion = "10"

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
func manageDrupalSettingsFile(app *DdevApp, drupalConfig *DrupalSettings) error {
	// We'll be writing/appending to the settings files and parent directory, make sure we have permissions to do so
	if err := drupalEnsureWritePerms(app); err != nil {
		return err
	}

	if !fileutil.FileExists(app.SiteSettingsPath) {
		output.UserOut.Printf("No %s file exists, creating one", drupalConfig.SiteSettings)

		if err := writeDrupalSettingsPHP(app); err != nil {
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
		util.Debug("Existing %s file does not include %s, modifying to include DDEV settings", drupalConfig.SiteSettings, drupalConfig.SiteSettingsDdev)

		if err := appendIncludeToDrupalSettingsFile(app); err != nil {
			return fmt.Errorf("failed to include %s in %s: %v", drupalConfig.SiteSettingsDdev, drupalConfig.SiteSettings, err)
		}
	}

	return nil
}

// writeDrupalSettingsPHP creates the project's settings.php if it doesn't exist
func writeDrupalSettingsPHP(app *DdevApp) error {

	var appType string
	if app.Type == nodeps.AppTypeBackdrop {
		appType = app.Type
	} else {
		drupalVersion, err := GetDrupalVersion(app)
		if err != nil || drupalVersion == "" {
			drupalVersion = DefaultDrupalSettingsVersion
		}
		appType = "drupal" + drupalVersion
	}

	content, err := bundledAssets.ReadFile(path.Join("drupal", appType, "settings.php"))
	if err != nil {
		return err
	}

	// Ensure target directory exists and is writable
	dir := filepath.Dir(app.SiteSettingsPath)
	if err = util.Chmod(dir, 0755); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// Create file
	err = os.WriteFile(app.SiteSettingsPath, content, 0755)
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

	if err := manageDrupalSettingsFile(app, drupalConfig); err != nil {
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

	drupalVersion, err := GetDrupalVersion(app)
	if err != nil || drupalVersion == "" {
		drupalVersion = DefaultDrupalSettingsVersion
	}
	t, err := template.New("settings.ddev.php").ParseFS(bundledAssets, path.Join("drupal", "drupal"+drupalVersion, "settings.ddev.php"))
	if err != nil {
		return err
	}

	// Ensure target directory exists and is writable
	dir := filepath.Dir(filePath)
	if err = util.Chmod(dir, 0755); os.IsNotExist(err) {
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
 * DDEV manages this file and may delete or overwrite it unless this comment is removed.
 * Remove this comment if you don't want DDEV to manage this file.
 */

if (getenv('IS_DDEV_PROJECT') == 'true') {
  $options['l'] = "` + uri + `";
}
`)

	// Ensure target directory exists and is writable
	dir := filepath.Dir(filePath)
	if err := util.Chmod(dir, 0755); os.IsNotExist(err) {
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

// DrupalHooks adds d8+-specific hooks example for post-import-db
const DrupalHooks = `# post-import-db:
#   - exec: drush sql:sanitize
#   - exec: drush updatedb
#   - exec: drush cache:rebuild
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
	// We don't have anything new to add yet, so use Drupal7 version
	return []byte(Drupal7Hooks)
}

// getDrupalHooks for appending as byte array
func getDrupalHooks() []byte {
	return []byte(DrupalHooks)
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

// GetDrupalVersion finds the drupal8+ version so it can be used
// for setting requirements.
// It can only work if there is configured Drupal8+ code
func GetDrupalVersion(app *DdevApp) (string, error) {
	// For drupal6/7 we use the apptype provided as version
	switch app.Type {
	case nodeps.AppTypeDrupal6:
		return "6", nil
	case nodeps.AppTypeDrupal7:
		return "7", nil
	}
	// Otherwise figure out the version from existing code
	f := filepath.Join(app.AppRoot, app.Docroot, "core/lib/Drupal.php")
	hasVersion, matches, err := fileutil.GrepStringInFile(f, `const VERSION = '([0-9]+)`)
	v := ""
	if hasVersion {
		v = matches[1]
	}
	return v, err
}

// isDrupalApp returns true if the app is drupal
func isDrupalApp(app *DdevApp) bool {
	v, err := GetDrupalVersion(app)
	if err == nil && v != "" {
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

// drupal6ConfigOverrideAction overrides php_version for D6
func drupal6ConfigOverrideAction(app *DdevApp) error {
	app.PHPVersion = nodeps.PHP56
	return nil
}

// drupal7ConfigOverrideAction overrides php_version for D7
func drupal7ConfigOverrideAction(app *DdevApp) error {
	app.PHPVersion = nodeps.PHP82
	return nil
}

// drupalConfigOverrideAction selects proper versions for
func drupalConfigOverrideAction(app *DdevApp) error {
	v, err := GetDrupalVersion(app)
	if err != nil || v == "" {
		util.Warning("Unable to detect Drupal version, continuing")
		return nil
	}
	// If there is no database, update it to the default one,
	// otherwise show a warning to the user.
	if !nodeps.ArrayContainsString(app.GetOmittedContainers(), "db") {
		if dbType, err := app.GetExistingDBType(); err == nil && dbType == "" {
			app.Database = DatabaseDefault
		} else if app.Database != DatabaseDefault && v != "8" {
			defaultType := DatabaseDefault.Type + ":" + DatabaseDefault.Version
			util.Warning("Default database type is %s, but the current actual database type is %s, you may want to migrate with 'ddev debug migrate-database %s'.", defaultType, dbType, defaultType)
		}
	}
	switch v {
	case "8":
		app.PHPVersion = nodeps.PHP74
		app.Database = DatabaseDesc{Type: nodeps.MariaDB, Version: nodeps.MariaDB104}
	case "9":
		app.PHPVersion = nodeps.PHP81
	case "10":
		app.PHPVersion = nodeps.PHP83
	case "11":
		app.PHPVersion = nodeps.PHP83
		app.CorepackEnable = true
	}
	return nil
}

func drupalPostStartAction(app *DdevApp) error {
	if !nodeps.ArrayContainsString(app.GetOmittedContainers(), "db") && (isDrupalApp(app)) {
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
			_, _, err := app.Exec(&ExecOpts{
				Service:   "db",
				Cmd:       `mysql -uroot -proot -e "SET GLOBAL TRANSACTION ISOLATION LEVEL READ COMMITTED;" >/dev/null 2>&1`,
				NoCapture: false,
			})
			if err != nil {
				util.Warning("Unable to SET GLOBAL TRANSACTION ISOLATION LEVEL READ COMMITTED: %v", err)
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

		if err := util.Chmod(o, stat.Mode()|writePerms); err != nil {
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
// This is done by looking for the DDEV settings file (settings.ddev.php) in settings.php.
func settingsHasInclude(drupalConfig *DrupalSettings, siteSettingsPath string) (bool, error) {
	included, err := fileutil.FgrepStringInFile(siteSettingsPath, drupalConfig.SiteSettingsDdev)
	if err != nil {
		return false, err
	}

	return included, nil
}

// appendIncludeToDrupalSettingsFile modifies the settings.php file to include the settings.ddev.php
// file, which contains ddev-specific configuration.
func appendIncludeToDrupalSettingsFile(app *DdevApp) error {
	// Check if file is empty
	contents, err := os.ReadFile(app.SiteSettingsPath)
	if err != nil {
		return err
	}

	// If the file is empty, write the complete settings file and return
	if len(contents) == 0 {
		return writeDrupalSettingsPHP(app)
	}

	// The file is not empty, open it for appending
	file, err := os.OpenFile(app.SiteSettingsPath, os.O_RDWR|os.O_APPEND, 0644)
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

	// Parent of destination dir should exist
	if !fileutil.FileExists(filepath.Dir(destPath)) {
		return fmt.Errorf("unable to import to %s: parent directory does not exist", destPath)
	}

	// Parent of destination dir should be writable.
	if err := util.Chmod(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// If the destination path exists, remove contents as was warned
	// We do not remove the directory as it may be a docker bind-mount in
	// various situations, especially when mutagen is in use.
	if fileutil.FileExists(destPath) {
		if err := fileutil.PurgeDirectory(destPath); err != nil {
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

	if err := copy2.Copy(importPath, destPath); err != nil {
		return err
	}

	return nil
}

// getDrupalComposerCreateAllowedPaths returns fullpaths that are allowed to be present when running composer create
func getDrupalComposerCreateAllowedPaths(app *DdevApp) ([]string, error) {
	var allowed []string

	// Return early because we aren't expected to manage settings.
	if app.DisableSettingsManagement {
		return []string{}, nil
	}

	drupalConfig := NewDrupalSettings(app)

	if app.Type == "drupal6" || app.Type == "drupal7" {
		// drushrc.php path
		allowed = append(allowed, filepath.Join(filepath.Dir(app.SiteSettingsPath), "drushrc.php"))
	} else {
		// Sync path
		allowed = append(allowed, path.Join(app.GetDocroot(), "sites/default", drupalConfig.SyncDir))
	}

	return allowed, nil
}
