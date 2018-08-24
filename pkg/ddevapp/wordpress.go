package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"io/ioutil"

	"github.com/Masterminds/sprig"
	"github.com/drud/ddev/pkg/archive"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
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
}

// NewWordpressConfig produces a WordpressConfig object with defaults.
func NewWordpressConfig() *WordpressConfig {
	return &WordpressConfig{
		WPGeneric:         false,
		DatabaseName:      "db",
		DatabaseUsername:  "db",
		DatabasePassword:  "db",
		DatabaseHost:      "db",
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
	}
}

// wordPressHooks adds a wp-specific hooks example for post-import-db
const wordPressHooks = `
# Un-comment and enter the production url and local url
# to replace in your database after import.
#  post-import-db:
#    - exec: wp search-replace <production-url> <local-url>`

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

// Automatically generated include for settings managed by ddev.
if (file_exists(getenv('NGINX_DOCROOT') . '/{{ $config.SiteSettingsLocal }}')) {
	require_once getenv('NGINX_DOCROOT') . '/{{ $config.SiteSettingsLocal }}';
}
`

const wordpressSettingsAppendTemplate = `{{ $config := . }}
// Automatically generated include for settings managed by ddev.
if (file_exists(getenv('NGINX_DOCROOT') . '/{{ $config.SiteSettingsLocal }}')) {
	require_once getenv('NGINX_DOCROOT') . '/{{ $config.SiteSettingsLocal }}';
}
`

const wordpressDdevSettingsTemplate = `<?php
{{ $config := . }}
/**
{{ $config.Signature }}: Automatically generated WordPress settings file.
This file is managed by ddev and may be deleted or overwritten.
*/

/** The name of the database for WordPress */
if (!defined('DB_NAME'))
	define( 'DB_NAME', '{{ $config.DatabaseName }}' );

/** MySQL database username */
if (!defined('DB_USER'))
	define( 'DB_USER', '{{ $config.DatabaseUsername }}' );

/** MySQL database password */
if (!defined('DB_PASSWORD'))
	define( 'DB_PASSWORD', '{{ $config.DatabasePassword }}' );

/** MySQL hostname */
if (!defined('DB_HOST'))
 	define( 'DB_HOST', '{{ $config.DatabaseHost }}' );

/** Database Charset to use in creating database tables. */
define( 'DB_CHARSET', 'utf8mb4' );

/** The Database Collate type. */
define( 'DB_COLLATE', '' );

/**
 * WordPress Database Table prefix.
 */
if(!isset($table_prefix))
	$table_prefix  = '{{ $config.TablePrefix }}';

/**
 * For developers: WordPress debugging mode.
 */
 if (!defined('WP_DEBUG'))
	define('WP_DEBUG', false);

/**#@+
 * Authentication Unique Keys and Salts.
 */
if ( !defined('AUTH_KEY') )
	define( 'AUTH_KEY',         	'{{ $config.AuthKey }}' );

if ( !defined('SECURE_AUTH_KEY') )
	define( 'SECURE_AUTH_KEY',  	'{{ $config.SecureAuthKey }}' );

if ( !defined('LOGGED_IN_KEY') )
	define( 'LOGGED_IN_KEY',    	'{{ $config.LoggedInKey }}' );

if ( !defined('NONCE_KEY') )
	define( 'NONCE_KEY',        	'{{ $config.NonceKey }}' );

if ( !defined('AUTH_SALT') )
	define( 'AUTH_SALT',        	'{{ $config.AuthSalt }}' );

if ( !defined('SECURE_AUTH_SALT') )
	define( 'SECURE_AUTH_SALT', 	'{{ $config.SecureAuthSalt }}' );

if ( !defined('LOGGED_IN_SALT') )
	define( 'LOGGED_IN_SALT',   	'{{ $config.LoggedInSalt }}' );

if ( !defined('NONCE_SALT') )
	define( 'NONCE_SALT',       	'{{ $config.NonceSalt }}' );

/** site URL */
if ( !defined('WP_HOME') )
	define('WP_HOME', '{{ $config.DeployURL }}');

/** WP_ENV */
define('WP_ENV', getenv('DDEV_ENV_NAME') ? getenv('DDEV_ENV_NAME') : 'production');

/** Absolute path to the WordPress directory. */
if ( !defined('ABSPATH') )
	define('ABSPATH', dirname(__FILE__) . '/');

/**
Sets up WordPress vars and included files.

wp-settings.php is typically included in wp-config.php. This check ensures it is not
included again if this file is written to wp-config-local.php.
*/
if (basename(__FILE__) == "wp-config.php") {
	require_once(ABSPATH . '/wp-settings.php');
}
`

// createWordpressSettingsFile creates a Wordpress settings file from a
// template. Returns full path to location of file + err
func createWordpressSettingsFile(app *DdevApp) (string, error) {
	config := NewWordpressConfig()

	// If the settings file does not exist, create it
	if !fileutil.FileExists(app.SiteSettingsPath) {
		output.UserOut.Printf("No %s file exists, creating one", config.SiteSettings)

		if err := writeWordpressSettingsFile(config, app.SiteSettingsPath); err != nil {
			return "", fmt.Errorf("failed to write %s: %v", app.SiteSettingsPath, err)
		}
	}

	// If the settings file does not include a reference to the ddev settings, append the include
	included, err := fileutil.FgrepStringInFile(app.SiteSettingsPath, config.SiteSettingsLocal)
	if err != nil {
		return "", fmt.Errorf("failed to check for include in %s: %v", app.SiteSettingsPath, err)
	}

	if included {
		output.UserOut.Printf("Existing %s file includes %s", config.SiteSettings, config.SiteSettingsLocal)
	} else {
		output.UserOut.Printf("Existing %s file does not include %s, modifying to include ddev settings", config.SiteSettings, config.SiteSettingsLocal)

		if err := appendIncludeToWordpressSettingsFile(config, app.SiteSettingsPath); err != nil {
			return "", fmt.Errorf("failed to include %s in %s: %v", config.SiteSettingsLocal, config.SiteSettings, err)
		}
	}

	// Always write a new ddev-specific settings file
	if err := writeWordpressDdevSettingsFile(config, app.SiteLocalSettingsPath); err != nil {
		return "", fmt.Errorf("failed to write %s: %v", app.SiteLocalSettingsPath, err)
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

// appendIncludeToWordpressSettingsFile modifies the settings file to include the ddev-specific settings file.
func appendIncludeToWordpressSettingsFile(config *WordpressConfig, siteSettingsPath string) error {
	// Check if file is empty
	contents, err := ioutil.ReadFile(siteSettingsPath)
	if err != nil {
		return err
	}

	// If the file is empty, write the complete settings template and return
	if len(contents) == 0 {
		return writeWordpressSettingsFile(config, siteSettingsPath)
	}

	// The file is not empty, open it for appending
	file, err := os.OpenFile(siteSettingsPath, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer util.CheckClose(file)

	tmpl, err := template.New("settings").Funcs(getTemplateFuncMap()).Parse(wordpressSettingsAppendTemplate)
	if err != nil {
		return err
	}

	// Write the template to the file
	if err := tmpl.Execute(file, config); err != nil {
		return err
	}

	return nil
}

// setWordpressSiteSettingsPaths sets the expected settings files paths for
// a wordpress site.
func setWordpressSiteSettingsPaths(app *DdevApp) {
	config := NewWordpressConfig()

	settingsFileBasePath := filepath.Join(app.AppRoot, app.Docroot)
	app.SiteSettingsPath = filepath.Join(settingsFileBasePath, config.SiteSettings)
	app.SiteLocalSettingsPath = filepath.Join(settingsFileBasePath, config.SiteSettingsLocal)
}

// isWordpressApp returns true if the app of of type wordpress
func isWordpressApp(app *DdevApp) bool {
	if _, err := os.Stat(filepath.Join(app.AppRoot, app.Docroot, "wp-login.php")); err == nil {
		return true
	}
	// check for WP installed in a sub-directory
	// TODO: Add wildcard or ENV var to make more flexible, ie wordpress/
	if _, err := os.Stat(filepath.Join(app.AppRoot, app.Docroot, "wp/wp-login.php")); err == nil {
		return true
	}
	return false
}

// wordpressPostImportDBAction just emits a warning about updating URLs as is
// required with wordpress when running on a different URL.
func wordpressPostImportDBAction(app *DdevApp) error {
	util.Warning("Wordpress sites require a search/replace of the database when the URL is changed. You can run \"ddev exec wp search-replace [http://www.myproductionsite.example] %s\" to update the URLs across your database. For more information, see http://wp-cli.org/commands/search-replace/", app.GetHTTPURL())
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
