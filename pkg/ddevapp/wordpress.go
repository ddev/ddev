package ddevapp

import (
	"github.com/Masterminds/sprig"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"os"
	"path/filepath"
	"text/template"
)

// WordpressConfig encapsulates all the configurations for a WordPress site.
type WordpressConfig struct {
	WPGeneric        bool
	DeployName       string
	DeployURL        string
	DatabaseName     string
	DatabaseUsername string
	DatabasePassword string
	DatabaseHost     string
	AuthKey          string
	SecureAuthKey    string
	LoggedInKey      string
	NonceKey         string
	AuthSalt         string
	SecureAuthSalt   string
	LoggedInSalt     string
	NonceSalt        string
	Docroot          string
	TablePrefix      string
	Signature        string
}

// NewWordpressConfig produces a WordpressConfig object with defaults.
func NewWordpressConfig() *WordpressConfig {
	return &WordpressConfig{
		WPGeneric:        false,
		DatabaseName:     "db",
		DatabaseUsername: "db",
		DatabasePassword: "db",
		DatabaseHost:     "db",
		Docroot:          "/var/www/html/docroot",
		TablePrefix:      "wp_",
		AuthKey:          util.RandString(64),
		AuthSalt:         util.RandString(64),
		LoggedInKey:      util.RandString(64),
		LoggedInSalt:     util.RandString(64),
		NonceKey:         util.RandString(64),
		NonceSalt:        util.RandString(64),
		SecureAuthKey:    util.RandString(64),
		SecureAuthSalt:   util.RandString(64),
		Signature:        DdevFileSignature,
	}
}

// wordPressHooks adds a wp-specific hooks example for post-import-db
const wordPressHooks = `
# Un-comment and enter the production url and local url
# to replace in your database after import.
#post-import-db:
#  - exec: "wp search-replace <production-url> <local-url>"`

// getWordpressHooks for appending as byte array
func getWordpressHooks() []byte {
	return []byte(wordPressHooks)
}

// getWordpressUploadDir just returns a static upload files directory string.
func getWordpressUploadDir(app *DdevApp) string {
	return "wp-content/uploads"
}

const (
	wordpressTemplate = `<?php
{{ $config := . }}
/**
 {{ $config.Signature }}: Automatically generated WordPress wp-config.php file.
 ddev manages this file and may delete or overwrite the file unless this comment is removed.
 */

// ** MySQL settings - You can get this info from your web host ** //
/** The name of the database for WordPress */
define('DB_NAME', '{{ $config.DatabaseName }}');

/** MySQL database username */
define('DB_USER', '{{ $config.DatabaseUsername }}');

/** MySQL database password */
define('DB_PASSWORD', '{{ $config.DatabasePassword }}');

/** MySQL hostname */
define('DB_HOST', '{{ $config.DatabaseHost }}');

/** Database Charset to use in creating database tables. */
define('DB_CHARSET', 'utf8mb4');

/** The Database Collate type. Don't change this if in doubt. */
define('DB_COLLATE', '');

/**
 * WordPress Database Table prefix.
 */
$table_prefix  = '{{ $config.TablePrefix }}';

/**
 * For developers: WordPress debugging mode.
 */
define('WP_DEBUG', false);

/**#@+
 * Authentication Unique Keys and Salts.
 */
define( 'AUTH_KEY',         '{{ $config.AuthKey }}' );
define( 'SECURE_AUTH_KEY',  '{{ $config.SecureAuthKey }}' );
define( 'LOGGED_IN_KEY',    '{{ $config.LoggedInKey }}' );
define( 'NONCE_KEY',        '{{ $config.NonceKey }}' );
define( 'AUTH_SALT',        '{{ $config.AuthSalt }}' );
define( 'SECURE_AUTH_SALT', '{{ $config.SecureAuthSalt }}' );
define( 'LOGGED_IN_SALT',   '{{ $config.LoggedInSalt }}' );
define( 'NONCE_SALT',       '{{ $config.NonceSalt }}' );

/* That's all, stop editing! Happy blogging. */

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
)

// createWordpressSettingsFile creates a wordpress settings file from a
// template. Returns fullpath to location of file + err
func createWordpressSettingsFile(app *DdevApp) (string, error) {
	settingsFilePath := app.SiteSettingsPath
	if settingsFilePath == "" {
		settingsFilePath = app.SiteLocalSettingsPath
	}
	output.UserOut.Printf("Generating %s file for database connection.", filepath.Base(settingsFilePath))
	wpConfig := NewWordpressConfig()
	wpConfig.DeployURL = app.GetHTTPURL()
	err := WriteWordpressConfig(wpConfig, settingsFilePath)
	return settingsFilePath, err
}

// WriteWordpressConfig dynamically produces valid wp-config.php file by combining a configuration
// object with a data-driven template.
func WriteWordpressConfig(wordpressConfig *WordpressConfig, filePath string) error {
	tmpl, err := template.New("wordpressConfig").Funcs(sprig.TxtFuncMap()).Parse(wordpressTemplate)
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
	err = tmpl.Execute(file, wordpressConfig)
	if err != nil {
		return err
	}
	util.CheckClose(file)
	return nil
}

// setWordpressSiteSettingsPaths sets the expected settings files paths for
// a wordpress site.
func setWordpressSiteSettingsPaths(app *DdevApp) {
	settingsFileBasePath := filepath.Join(app.AppRoot, app.Docroot)
	var settingsFilePath, localSettingsFilePath string
	settingsFilePath = filepath.Join(settingsFileBasePath, "wp-config.php")
	localSettingsFilePath = filepath.Join(settingsFileBasePath, "wp-config-local.php")
	app.SiteSettingsPath = settingsFilePath
	app.SiteLocalSettingsPath = localSettingsFilePath
}

// isWordpressApp returns true if the app of of type wordpress
func isWordpressApp(app *DdevApp) bool {
	if _, err := os.Stat(filepath.Join(app.AppRoot, app.Docroot, "wp-login.php")); err == nil {
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
