package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"io/ioutil"

	"github.com/drud/ddev/pkg/appports"
	"github.com/drud/ddev/pkg/archive"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
)

// BackdropSettings holds database connection details for Backdrop.
type BackdropSettings struct {
	DatabaseName      string
	DatabaseUsername  string
	DatabasePassword  string
	DatabaseHost      string
	DatabaseDriver    string
	DatabasePort      string
	DatabasePrefix    string
	HashSalt          string
	Signature         string
	SiteSettings      string
	SiteSettingsLocal string
}

// NewBackdropSettings produces a BackdropSettings object with default values.
func NewBackdropSettings() *BackdropSettings {
	return &BackdropSettings{
		DatabaseName:      "db",
		DatabaseUsername:  "db",
		DatabasePassword:  "db",
		DatabaseHost:      "db",
		DatabaseDriver:    "mysql",
		DatabasePort:      appports.GetPort("db"),
		DatabasePrefix:    "",
		HashSalt:          util.RandString(64),
		Signature:         DdevFileSignature,
		SiteSettings:      "settings.php",
		SiteSettingsLocal: "settings.ddev.php",
	}
}

// backdropMainSettingsTemplate defines the template that will become settings.php in
// the event that one does not already exist.
const backdropMainSettingsTemplate = `<?php
{{ $config := . }}
// {{ $config.Signature }}: Automatically generated Backdrop settings file.

// Providing these in the main settings file is most likely to match the default config_directories.
$database = 'mysql://user:pass@localhost/database_name';
$config_directories['active'] = 'files/config_' . md5($database) . '/active';
$config_directories['staging'] = 'files/config_' . md5($database) . '/staging';
if (file_exists(__DIR__ . '/{{ $config.SiteSettingsLocal }}')) {
  include __DIR__ . '/{{ $config.SiteSettingsLocal }}';
}
`

// backdropSettingsAppendTemplate defines the template that will be appended to
// settings.php in the event that one exists.
const backdropSettingsAppendTemplate = `{{ $config := . }}
// Automatically generated include for settings managed by ddev.
if (file_exists(__DIR__ . '/{{ $config.SiteSettingsLocal }}')) {
  include __DIR__ . '/{{ $config.SiteSettingsLocal }}';
}
`

// backdropLocalSettingsTemplate defines the template that will become settings.ddev.php.
const backdropLocalSettingsTemplate = `<?php
{{ $config := . }}
/**
 {{ $config.Signature }}: Automatically generated Backdrop settings file.
 ddev manages this file and may delete or overwrite the file unless this comment is removed.
 */

$database = 'mysql://{{ $config.DatabaseUsername }}:{{ $config.DatabasePassword }}@{{ $config.DatabaseHost }}/{{ $config.DatabaseName }}';
$database_prefix = '{{ $config.DatabasePrefix }}';

$settings['update_free_access'] = FALSE;
$settings['hash_salt'] = '{{ $config.HashSalt }}';
$settings['backdrop_drupal_compatibility'] = TRUE;

ini_set('session.gc_probability', 1);
ini_set('session.gc_divisor', 100);
ini_set('session.gc_maxlifetime', 200000);
ini_set('session.cookie_lifetime', 2000000);

// This determines whether or not drush should include a custom settings file which allows
// it to work both within a docker container and natively on the host system.
$drush_settings = __DIR__ . '/ddev_drush_settings.php';
if (empty(getenv('DDEV_PHP_VERSION')) && file_exists($drush_settings)) {
  include $drush_settings;
}

`

// createBackdropSettingsFile manages creation and modification of settings.php and settings.ddev.php.
// If a settings.php file already exists, it will be modified to ensure that it includes
// settings.ddev.php, which contains ddev-specific configuration.
func createBackdropSettingsFile(app *DdevApp) (string, error) {
	settings := NewBackdropSettings()

	if !fileutil.FileExists(app.SiteSettingsPath) {
		output.UserOut.Printf("No %s file exists, creating one", settings.SiteSettings)
		if err := writeBackdropMainSettingsFile(settings, app.SiteSettingsPath); err != nil {
			return "", err
		}
	}

	included, err := fileutil.FgrepStringInFile(app.SiteSettingsPath, settings.SiteSettingsLocal)
	if err != nil {
		return "", err
	}

	if included {
		output.UserOut.Printf("Existing %s includes %s", settings.SiteSettings, settings.SiteSettingsLocal)
	} else {
		output.UserOut.Printf("Existing %s file does not include %s, modifying to include ddev settings", settings.SiteSettings, settings.SiteSettingsLocal)

		if err = appendIncludeToBackdropSettingsFile(settings, app.SiteSettingsPath); err != nil {
			return "", fmt.Errorf("failed to include %s in %s: %v", settings.SiteSettingsLocal, settings.SiteSettings, err)
		}
	}

	if err = writeBackdropDdevSettingsFile(settings, app.SiteLocalSettingsPath); err != nil {
		return "", fmt.Errorf("failed to write Drupal settings file %s: %v", app.SiteLocalSettingsPath, err)
	}

	return app.SiteLocalSettingsPath, nil
}

// writeBackdropMainSettingsFile dynamically produces a valid settings.php file by
// combining a configuration object with a data-driven template.
func writeBackdropMainSettingsFile(settings *BackdropSettings, filePath string) error {
	tmpl, err := template.New("settings").Funcs(getTemplateFuncMap()).Parse(backdropMainSettingsTemplate)
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
	defer util.CheckClose(file)

	if err := tmpl.Execute(file, settings); err != nil {
		return err
	}

	return nil
}

// writeBackdropDdevSettingsFile dynamically produces a valid settings.ddev.php file
// by combining a configuration object with a data-driven template.
func writeBackdropDdevSettingsFile(settings *BackdropSettings, filePath string) error {
	tmpl, err := template.New("settings").Funcs(getTemplateFuncMap()).Parse(backdropLocalSettingsTemplate)
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
	defer util.CheckClose(file)

	if err := tmpl.Execute(file, settings); err != nil {
		return err
	}

	return nil
}

// getBackdropUploadDir will return a custom upload dir if defined, returning a default path if not.
func getBackdropUploadDir(app *DdevApp) string {
	if app.UploadDir == "" {
		return "files"
	}

	return app.UploadDir
}

// getBackdropHooks for appending as byte array.
func getBackdropHooks() []byte {
	backdropHooks := `
#  post-import-db:
#    - exec: drush cc all`
	return []byte(backdropHooks)
}

// setBackdropSiteSettingsPaths sets the paths to settings.php for templating.
func setBackdropSiteSettingsPaths(app *DdevApp) {
	settings := NewBackdropSettings()
	settingsFileBasePath := filepath.Join(app.AppRoot, app.Docroot)
	app.SiteSettingsPath = filepath.Join(settingsFileBasePath, settings.SiteSettings)
	app.SiteLocalSettingsPath = filepath.Join(settingsFileBasePath, settings.SiteSettingsLocal)
}

// isBackdropApp returns true if the app is of type "backdrop".
func isBackdropApp(app *DdevApp) bool {
	if _, err := os.Stat(filepath.Join(app.AppRoot, app.Docroot, "core/scripts/backdrop.sh")); err == nil {
		return true
	}
	return false
}

// backdropPostImportDBAction emits a warning about moving configuration into place
// appropriately in order for Backdrop to function properly.
func backdropPostImportDBAction(app *DdevApp) error {
	util.Warning("Backdrop sites require your config JSON files to be located in your site's \"active\" configuration directory. Please refer to the Backdrop documentation (https://backdropcms.org/user-guide/moving-backdrop-site) for more information about this process.")
	return nil
}

// appendIncludeToBackdropSettingsFile modifies the settings.php file to include the settings.ddev.php
// file, which contains ddev-specific configuration.
func appendIncludeToBackdropSettingsFile(settings *BackdropSettings, siteSettingsPath string) error {
	// Check if file is empty
	contents, err := ioutil.ReadFile(siteSettingsPath)
	if err != nil {
		return err
	}

	// If the file is empty, write the complete settings template and return
	if len(contents) == 0 {
		return writeBackdropMainSettingsFile(settings, siteSettingsPath)
	}

	// The file is not empty, open it for appending
	file, err := os.OpenFile(siteSettingsPath, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer util.CheckClose(file)

	tmpl, err := template.New("settings").Funcs(getTemplateFuncMap()).Parse(backdropSettingsAppendTemplate)
	if err != nil {
		return err
	}

	// Write the template to the file
	if err := tmpl.Execute(file, settings); err != nil {
		return err
	}

	return nil
}

// backdropImportFilesAction defines the Backdrop workflow for importing project files.
// The Backdrop workflow is currently identical to the Drupal import-files workflow.
func backdropImportFilesAction(app *DdevApp, importPath, extPath string) error {
	destPath := filepath.Join(app.GetAppRoot(), app.GetDocroot(), app.GetUploadDir())

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

	if err := fileutil.CopyDir(importPath, destPath); err != nil {
		return err
	}

	return nil
}

// backdropPostStartAction handles default post-start actions for backdrop apps, like ensuring
// useful permissions settings on sites/default.
func backdropPostStartAction(app *DdevApp) error {
	// Drush config has to be written after start because we don't know the ports until it's started
	drushConfig := NewDrushConfig(app)
	err := WriteDrushConfig(drushConfig, filepath.Join(filepath.Dir(app.SiteSettingsPath), "ddev_drush_settings.php"))
	if err != nil {
		util.Warning("Failed to WriteDrushConfig: %v", err)
	}

	return nil
}
