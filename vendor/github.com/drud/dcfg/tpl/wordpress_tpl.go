package tpl

const (
	wordpressTemplate = `<?php
{{ $config := . }}
/* Automatically generated WordPress wp-config.php file. */

// ** MySQL settings - You can get this info from your web host ** //
/** The name of the database for WordPress */
define('DB_NAME', '{{ $config.DatabaseName }}');

/** MySQL database username */
define('DB_USER', '{{ $config.DatabaseUsername }}');

/** MySQL database password */
define('DB_PASSWORD', '{{ $config.DatabasePassword }}');

/** MySQL hostname */
define('DB_HOST', '{{ $config.DatabaseHost }}:{{ $config.DatabasePort }}');

// This allows you to provide a configuration file in your site's code base for
// configurations that should be present in any environment.
if (file_exists(__DIR__ . '/wp-config.custom.php')) {
  include __DIR__ . '/wp-config.custom.php';
}

// This allows you to provide a configuration file in your site's code base for
// configurations that should be present for a local development environment.
if (file_exists(__DIR__ . '/wp-config.local.php')) {
  include __DIR__ . '/wp-config.local.php';
}

/** Database Charset to use in creating database tables. */
define('DB_CHARSET', 'utf8');

/** The Database Collate type. Don't change this if in doubt. */
define('DB_COLLATE', '');

/**#@+
 * Authentication Unique Keys and Salts.
 */
define('AUTH_KEY',         '{{ $config.AuthKey }}');
define('SECURE_AUTH_KEY',  '{{ $config.SecureAuthKey }}');
define('LOGGED_IN_KEY',    '{{ $config.LoggedInKey }}');
define('NONCE_KEY',        '{{ $config.NonceKey }}');
define('AUTH_SALT',        '{{ $config.AuthSalt }}');
define('SECURE_AUTH_SALT', '{{ $config.SecureAuthSalt }}');
define('LOGGED_IN_SALT',   '{{ $config.LoggedInSalt }}');
define('NONCE_SALT',       '{{ $config.NonceSalt }}');

/**
 * WordPress Database Table prefix.
 */
$table_prefix  = '{{ $config.DatabasePrefix }}';

/**
 * For developers: WordPress debugging mode.
 */
define('WP_DEBUG', false);


/**
 * WP_SITEURL allows the WordPress address (URL) to be defined. The value defined is the
 * address where your WordPress core files reside.
 */
if ( !defined('WP_SITEURL') ) {
  define( 'WP_SITEURL', '{{ $config.SiteURL }}{{ $config.CoreDir }}' );
}

/**
 * WP_HOME overrides the wp_options table value for home but does not change it in the
 * database. home is the address you want people to type in their browser to reach your WordPress site.
 */
if ( !defined('WP_HOME') ) {
  define( 'WP_HOME', '{{ $config.SiteURL }}' );
}

/**
 * You can move the wp-content directory, which holds your themes, plugins,
 * and uploads, outside of the WordPress application directory.
 */

// Set WP_CONTENT_DIR to the full local path of this directory (no trailing slash)
if ( !defined('WP_CONTENT_DIR') ) {
  define( 'WP_CONTENT_DIR', dirname( __FILE__ ) . '/{{ $config.ContentDir }}' );
}

// Set WP_CONTENT_URL to the full URL of this directory (no trailing slash)
if ( !defined('WP_CONTENT_URL')  ) {
  define( 'WP_CONTENT_URL', WP_HOME . '/{{ $config.ContentDir }}' );
}

define( 'UPLOADS', '{{ $config.UploadDir }}' );

/* That's all, stop editing! Happy blogging. */

/** Absolute path to the WordPress directory. */
if ( !defined('ABSPATH') )
	define('ABSPATH', dirname(__FILE__) . '{{ $config.CoreDir }}');

/** Sets up WordPress vars and included files. */
require_once(ABSPATH . '/wp-settings.php');
`
)
