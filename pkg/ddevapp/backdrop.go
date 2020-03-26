package ddevapp

import (
	"fmt"
	"github.com/drud/ddev/pkg/dockerutil"
	"os"
	"path/filepath"
	"text/template"

	"io/ioutil"

	"github.com/drud/ddev/pkg/archive"
	"github.com/drud/ddev/pkg/fileutil"
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
	SiteSettings     string
	SiteSettingsDdev string
	DockerIP         string
	DBPublishedPort  int
}

// NewBackdropSettings produces a BackdropSettings object with default values.
func NewBackdropSettings(app *DdevApp) *BackdropSettings {
	dockerIP, _ := dockerutil.GetDockerIP()
	dbPublishedPort, _ := app.GetPublishedPort("db")

	return &BackdropSettings{
		DatabaseName:     "db",
		DatabaseUsername: "db",
		DatabasePassword: "db",
		DatabaseHost:     "db",
		DatabaseDriver:   "mysql",
		DatabasePort:     GetPort("db"),
		DatabasePrefix:   "",
		HashSalt:         util.RandString(64),
		Signature:        DdevFileSignature,
		SiteSettings:     "settings.php",
		SiteSettingsDdev: "settings.ddev.php",
		DockerIP:         dockerIP,
		DBPublishedPort:  dbPublishedPort,
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
if (file_exists(__DIR__ . '/{{ $config.SiteSettingsDdev }}') && getenv('IS_DDEV_PROJECT') == 'true')) {
  include __DIR__ . '/{{ $config.SiteSettingsDdev }}';
}
`

// backdropSettingsAppendTemplate defines the template that will be appended to
// settings.php in the event that one exists.
const backdropSettingsAppendTemplate = `{{ $config := . }}
// Automatically generated include for settings managed by ddev.
if (file_exists(__DIR__ . '/{{ $config.SiteSettingsDdev }}') && getenv('IS_DDEV_PROJECT') == 'true') {
  include __DIR__ . '/{{ $config.SiteSettingsDdev }}';
}
`

// BackdropDdevSettingsTemplate defines the template that will become settings.ddev.php.
const BackdropDdevSettingsTemplate = `<?php
{{ $config := . }}
/**
 {{ $config.Signature }}: Automatically generated Backdrop settings.ddev.php file.
 ddev manages this file and may delete or overwrite the file unless this comment is removed.
 */

$host = "{{ $config.DatabaseHost }}";
$port = {{ $config.DatabasePort }};

// If DDEV_PHP_VERSION is not set but IS_DDEV_PROJECT *is*, it means we're running (drush) on the host,
// so use the host-side bind port on docker IP
if (empty(getenv('DDEV_PHP_VERSION') && getenv('IS_DDEV_PROJECT') == "true")) {
  $host = "{{ $config.DockerIP }}";
  $port = {{ $config.DBPublishedPort }};
} 

$database = "{{ $config.DatabaseDriver }}://{{ $config.DatabaseUsername }}:{{ $config.DatabasePassword }}@$host:$port/{{ $config.DatabaseName }}";
$database_prefix = '{{ $config.DatabasePrefix }}';

$settings['update_free_access'] = FALSE;
$settings['hash_salt'] = '{{ $config.HashSalt }}';
$settings['backdrop_drupal_compatibility'] = TRUE;
`

// createBackdropSettingsFile manages creation and modification of settings.php and settings.ddev.php.
// If a settings.php file already exists, it will be modified to ensure that it includes
// settings.ddev.php, which contains ddev-specific configuration.
func createBackdropSettingsFile(app *DdevApp) (string, error) {
	settings := NewBackdropSettings(app)

	if !fileutil.FileExists(app.SiteSettingsPath) {
		output.UserOut.Printf("No %s file exists, creating one", settings.SiteSettings)
		if err := writeBackdropMainSettingsFile(settings, app.SiteSettingsPath); err != nil {
			return "", err
		}
	}

	included, err := fileutil.FgrepStringInFile(app.SiteSettingsPath, settings.SiteSettingsDdev)
	if err != nil {
		return "", err
	}

	if included {
		output.UserOut.Printf("Existing %s includes %s", settings.SiteSettings, settings.SiteSettingsDdev)
	} else {
		output.UserOut.Printf("Existing %s file does not include %s, modifying to include ddev settings", settings.SiteSettings, settings.SiteSettingsDdev)

		if err = appendIncludeToBackdropSettingsFile(settings, app.SiteSettingsPath); err != nil {
			return "", fmt.Errorf("failed to include %s in %s: %v", settings.SiteSettingsDdev, settings.SiteSettings, err)
		}
	}

	if err = writeBackdropDdevSettingsFile(settings, app.SiteDdevSettingsFile); err != nil {
		return "", fmt.Errorf("failed to write Drupal settings file %s: %v", app.SiteDdevSettingsFile, err)
	}

	return app.SiteDdevSettingsFile, nil
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
	if fileutil.FileExists(filePath) {
		// Check if the file is managed by ddev.
		signatureFound, err := fileutil.FgrepStringInFile(filePath, DdevFileSignature)
		if err != nil {
			return err
		}

		// If the signature wasn't found, warn the user and return.
		if !signatureFound {
			util.Warning("%s already exists and is managed by the user.", filepath.Base(filePath))
			return nil
		}
	}
	tmpl, err := template.New("settings").Funcs(getTemplateFuncMap()).Parse(BackdropDdevSettingsTemplate)
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
	backdropHooks := `#  post-import-db:
#    - exec: drush cc all
`
	return []byte(backdropHooks)
}

// setBackdropSiteSettingsPaths sets the paths to settings.php for templating.
func setBackdropSiteSettingsPaths(app *DdevApp) {
	settings := NewBackdropSettings(app)
	settingsFileBasePath := filepath.Join(app.AppRoot, app.Docroot)
	app.SiteSettingsPath = filepath.Join(settingsFileBasePath, settings.SiteSettings)
	app.SiteDdevSettingsFile = filepath.Join(settingsFileBasePath, settings.SiteSettingsDdev)
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
	err := WriteDrushrc(app, filepath.Join(filepath.Dir(app.SiteSettingsPath), "drushrc.php"))
	if err != nil {
		util.Warning("Failed to WriteDrushrc: %v", err)
	}

	if _, err = app.CreateSettingsFile(); err != nil {
		return fmt.Errorf("failed to write settings file %s: %v", app.SiteDdevSettingsFile, err)
	}
	return nil
}
