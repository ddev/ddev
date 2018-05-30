package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/drud/ddev/pkg/appports"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
)

// BackdropSettings holds database connection details for Backdrop.
type BackdropSettings struct {
	DatabaseName     string
	DatabaseUsername string
	DatabasePassword string
	DatabaseHost     string
	DatabaseDriver   string
	DatabasePort     string
	DatabasePrefix   string
	HashSalt         string
	Signature        string
}

// NewBackdropSettings produces a BackdropSettings object with default values.
func NewBackdropSettings() *BackdropSettings {
	return &BackdropSettings{
		DatabaseName:     "db",
		DatabaseUsername: "db",
		DatabasePassword: "db",
		DatabaseHost:     "db",
		DatabaseDriver:   "mysql",
		DatabasePort:     appports.GetPort("db"),
		DatabasePrefix:   "",
		HashSalt:         util.RandString(64),
		Signature:        DdevFileSignature,
	}
}

// Note that this template will almost always be used for settings.local.php
// because Backdrop ships with it's own default settings.php.
const backdropTemplate = `<?php
{{ $config := . }}
/**
 {{ $config.Signature }}: Automatically generated Backdrop settings.php file.
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

// Include local settings if it exists.
if (file_exists(__DIR__ . '/settings.ddev-local.php')) {
	require_once __DIR__ . '/settings.ddev-local.php';
}
`

// createBackdropSettingsFile creates the app's settings.php or equivalent,
// adding things like database host, name, and password.
// Returns the full path to the settings file and err.
func createBackdropSettingsFile(app *DdevApp) (string, error) {
	settingsFilePath, err := app.DetermineSettingsPathLocation()
	if err != nil {
		return "", fmt.Errorf("Failed to get Backdrop settings file path: %v", err.Error())
	}
	output.UserOut.Printf("Generating %s file for database connection.", filepath.Base(settingsFilePath))

	backdropConfig := NewBackdropSettings()

	err = writeBackdropSettingsFile(backdropConfig, settingsFilePath)
	if err != nil {
		return settingsFilePath, fmt.Errorf("Failed to write Backdrop settings file: %v", err.Error())
	}

	return settingsFilePath, nil
}

// writeBackdropSettingsFile dynamically produces a valid settings.php file by
// combining a configuration object with a data-driven template.
func writeBackdropSettingsFile(settings *BackdropSettings, filePath string) error {
	tmpl, err := template.New("settings").Funcs(sprig.TxtFuncMap()).Parse(backdropTemplate)
	if err != nil {
		return err
	}

	// Ensure target directory is writable.
	dir := filepath.Dir(filePath)
	err = os.Chmod(dir, 0755)
	if err != nil {
		return err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	err = tmpl.Execute(file, settings)
	if err != nil {
		return err
	}
	util.CheckClose(file)
	return nil
}

// getBackdropUploadDir returns the path to the directory where uploaded files are
// stored.
func getBackdropUploadDir(app *DdevApp) string {
	return "files"
}

// getBackdropHooks for appending as byte array.
func getBackdropHooks() []byte {
	backdropHooks := `
#  post-import-db:
#    - exec: "drush cc all"`
	return []byte(backdropHooks)
}

// setBackdropSiteSettingsPaths sets the paths to settings.php for templating.
func setBackdropSiteSettingsPaths(app *DdevApp) {
	settingsFileBasePath := filepath.Join(app.AppRoot, app.Docroot)
	app.SiteSettingsPath = filepath.Join(settingsFileBasePath, "settings.local.php")
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
