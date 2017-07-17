package config

import (
	"os"
	"text/template"

	"path/filepath"

	"github.com/Masterminds/sprig"
	"github.com/drud/ddev/pkg/cms/model"
	"github.com/drud/ddev/pkg/util"
)

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
if (__FILE__ == "wp-config.php") {
	require_once(ABSPATH . '/wp-settings.php');
}
`
)

// WriteWordpressConfig dynamically produces valid wp-config.php file by combining a configuration
// object with a data-driven template.
func WriteWordpressConfig(wordpressConfig *model.WordpressConfig, filePath string) error {
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
