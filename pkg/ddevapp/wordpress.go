package ddevapp

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/drud/ddev/pkg/archive"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/util"
)

// WordpressConfig encapsulates all the configurations for a WordPress site.
type WordpressConfig struct {
	WPGeneric         bool
	DeployName        string
	DeployURL         string
	DatabaseName      string
	DatabaseUsername  string
	DatabasePassword  string
	DatabaseHost      string
	AuthKey           string
	SecureAuthKey     string
	LoggedInKey       string
	NonceKey          string
	AuthSalt          string
	SecureAuthSalt    string
	LoggedInSalt      string
	NonceSalt         string
	Docroot           string
	TablePrefix       string
	Signature         string
	SiteSettings      string
	SiteSettingsLocal string
	AbsPath           string
}

// NewWordpressConfig produces a WordpressConfig object with defaults.
func NewWordpressConfig(app *DdevApp) *WordpressConfig {
	absPath, _ := getRelativeAbsPath(app)

	return &WordpressConfig{
		WPGeneric:         false,
		DatabaseName:      "db",
		DatabaseUsername:  "db",
		DatabasePassword:  "db",
		DatabaseHost:      "db",
		DeployURL:         app.GetHTTPURL(),
		Docroot:           "/var/www/html/docroot",
		TablePrefix:       "wp_",
		AuthKey:           util.RandString(64),
		AuthSalt:          util.RandString(64),
		LoggedInKey:       util.RandString(64),
		LoggedInSalt:      util.RandString(64),
		NonceKey:          util.RandString(64),
		NonceSalt:         util.RandString(64),
		SecureAuthKey:     util.RandString(64),
		SecureAuthSalt:    util.RandString(64),
		Signature:         DdevFileSignature,
		SiteSettings:      "wp-config.php",
		SiteSettingsLocal: "wp-config-ddev.php",
		AbsPath:           absPath,
	}
}

// wordPressHooks adds a wp-specific hooks example for post-start
const wordPressHooks = `
# Un-comment to emit the WP CLI version after ddev start.
#  post-start:
#    - exec: wp cli version`

// getWordpressHooks for appending as byte array
func getWordpressHooks() []byte {
	return []byte(wordPressHooks)
}

// getWordpressUploadDir will return a custom upload dir if defined, returning a default path if not.
func getWordpressUploadDir(app *DdevApp) string {
	if app.UploadDir == "" {
		return "wp-content/uploads"
	}

	return app.UploadDir
}

const wordpressSettingsTemplate = `<?php
{{ $config := . }}
/**
 {{ $config.Signature }}: Automatically generated WordPress settings file.
 ddev manages this file and may delete or overwrite the file unless this comment is removed.
 */

/** Absolute path to the WordPress directory. */
define('ABSPATH', dirname(__FILE__) . '/{{ $config.AbsPath }}');

/** Automatically generated include for settings managed by ddev. */
if (file_exists(getenv('NGINX_DOCROOT') . '/{{ $config.SiteSettingsLocal }}')) {
	require_once getenv('NGINX_DOCROOT') . '/{{ $config.SiteSettingsLocal }}';
}

/** Include wp-settings.php */
if (file_exists(ABSPATH . '/wp-settings.php')) {
	require_once ABSPATH . '/wp-settings.php';
}
`

const wordpressDdevSettingsTemplate = `<?php
{{ $config := . }}
/**
{{ $config.Signature }}: Automatically generated WordPress settings file.
This file is managed by ddev and may be deleted or overwritten.
*/

/** The name of the database for WordPress */
define('DB_NAME', '{{ $config.DatabaseName }}');

/** MySQL database username */
define('DB_USER', '{{ $config.DatabaseUsername }}');

/** MySQL database password */
define('DB_PASSWORD', '{{ $config.DatabasePassword }}');

/** MySQL hostname */
define('DB_HOST', '{{ $config.DatabaseHost }}');

/** Enable debug */
define('WP_DEBUG', true);

/** WP_HOME URL */
define('WP_HOME', '{{ $config.DeployURL }}');

/** WP_SITEURL location */
define('WP_SITEURL', WP_HOME . '/{{ $config.AbsPath  }}');

/** WP_ENV */
define('WP_ENV', getenv('DDEV_ENV_NAME') ? getenv('DDEV_ENV_NAME') : 'production');

/** Define the database table prefix */
$table_prefix  = 'wp_';
`

// createWordpressSettingsFile creates a Wordpress settings file from a
// template. Returns full path to location of file + err
func createWordpressSettingsFile(app *DdevApp) (string, error) {
	config := NewWordpressConfig(app)

	// Unconditionally write ddev settings file
	if err := writeWordpressDdevSettingsFile(config, app.SiteLocalSettingsPath); err != nil {
		return "", err
	}

	// Check if an existing WordPress settings file exists
	if fileutil.FileExists(app.SiteSettingsPath) {
		// Check if existing WordPress settings file is ddev-managed
		sigExists, err := fileutil.FgrepStringInFile(app.SiteSettingsPath, DdevFileSignature)
		if err != nil {
			return "", err
		}

		if sigExists {
			// Settings file is ddev-managed, overwriting is safe
			if err := writeWordpressSettingsFile(config, app.SiteSettingsPath); err != nil {
				return "", err
			}
		} else {
			// Settings file exists and is not ddev-managed, alert the user to the location
			// of the generated ddev settings file
			util.Warning("\nAn existing user-managed %s file has been detected", path.Base(app.SiteSettingsPath))
			util.Warning("ddev settings have been written to %s", app.SiteLocalSettingsPath)
			util.Warning("Please alter your settings to include these values before starting this project\n")
		}
	} else {
		// If settings file does not exist, write basic settings file including it
		if err := writeWordpressSettingsFile(config, app.SiteSettingsPath); err != nil {
			return "", err
		}
	}

	return app.SiteLocalSettingsPath, nil
}

// writeWordpressSettingsFile dynamically produces valid wp-config.php file by combining a configuration
// object with a data-driven template.
func writeWordpressSettingsFile(wordpressConfig *WordpressConfig, filePath string) error {
	tmpl, err := template.New("wordpressConfig").Funcs(sprig.TxtFuncMap()).Parse(wordpressSettingsTemplate)
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

	if err = tmpl.Execute(file, wordpressConfig); err != nil {
		return err
	}

	return nil
}

// writeWordpressDdevSettingsFile unconditionally creates the file that contains ddev-specific settings.
func writeWordpressDdevSettingsFile(config *WordpressConfig, filePath string) error {
	tmpl, err := template.New("wordpressConfig").Funcs(sprig.TxtFuncMap()).Parse(wordpressDdevSettingsTemplate)
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

	if err = tmpl.Execute(file, config); err != nil {
		return err
	}

	return nil
}

// setWordpressSiteSettingsPaths sets the expected settings files paths for
// a wordpress site.
func setWordpressSiteSettingsPaths(app *DdevApp) {
	config := NewWordpressConfig(app)

	settingsFileBasePath := filepath.Join(app.AppRoot, app.Docroot)
	app.SiteSettingsPath = filepath.Join(settingsFileBasePath, config.SiteSettings)
	app.SiteLocalSettingsPath = filepath.Join(settingsFileBasePath, config.SiteSettingsLocal)
}

// isWordpressApp returns true if the app of of type wordpress
func isWordpressApp(app *DdevApp) bool {
	_, err := getRelativeAbsPath(app)
	if err != nil {
		return false
	}

	return true
}

// wordpressPostImportDBAction just emits a warning about updating URLs as is
// required with wordpress when running on a different URL.
func wordpressPostImportDBAction(app *DdevApp) error {
	return nil
}

// wordpressImportFilesAction defines the Wordpress workflow for importing project files.
// The Wordpress workflow is currently identical to the Drupal import-files workflow.
func wordpressImportFilesAction(app *DdevApp, importPath, extPath string) error {
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

// getRelativeAbsPath returns the portion of the ABSPATH value that will come after "/" in wp-config.php -
// this is done by searching (at a max depth of one directory from the docroot) for wp-settings.php, the
// file we're using as a signal to indicate that this is a WordPress project.
func getRelativeAbsPath(app *DdevApp) (string, error) {
	needle := "wp-settings.php"

	// Check if the docroot is the abspath
	if fileutil.FileExists(filepath.Join(app.AppRoot, app.Docroot, needle)) {
		return "", nil
	}

	// Gather directories in approot
	objs, err := ioutil.ReadDir(filepath.Join(app.AppRoot, app.Docroot))
	if err != nil {
		return "", err
	}

	for _, obj := range objs {
		if !obj.IsDir() {
			continue
		}

		potentials, err := ioutil.ReadDir(filepath.Join(app.AppRoot, app.Docroot, obj.Name()))
		if err != nil {
			return "", err
		}

		for _, potential := range potentials {
			if potential.Name() == needle {
				return obj.Name(), nil
			}
		}
	}

	return "", fmt.Errorf("unable to determine ABSPATH")
}
