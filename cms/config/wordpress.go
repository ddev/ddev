package config

import (
	"os"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/drud/bootstrap/cli/cms/model"
)

const (
	wordpressTemplate = `<?php
{{ $config := . }}
/* Automatically generated WordPress wp-config.php file. */

// ============================
// Production database settings
// ============================
if ( ! defined( 'DB_NAME' ) )
    define( 'DB_NAME', "{{ $config.DatabaseName }}" );
if ( ! defined( 'DB_USER' ) )
    define( 'DB_USER', "{{ $config.DatabaseUsername }}" );
if ( ! defined( 'DB_PASSWORD' ) )
    define( 'DB_PASSWORD', "{{ $config.DatabasePassword }}" );
if ( ! defined( 'DB_HOST' ) )
    define( 'DB_HOST', "{{ $config.DatabaseHost }}" );

// ======================================================
// Additonal DB settings, you do not want to change these
// ======================================================
if (file_exists(__DIR__ . '/custom.settings.php')) {
  include __DIR__ . '/custom.settings.php';
}

define( 'DB_CHARSET', 'utf8' );
define( 'DB_COLLATE', '' );

// =================
// Site and WP URL's
// =================
if ( ! defined( 'WP_SITEURL' ) ) {
  // define( 'WP_SITEURL', 'http://' . $_SERVER['HTTP_HOST'] . '/wp' );
  {{ if $config.WPGeneric }}
  define( 'WP_SITEURL', '{{ $config.DeployURL }}' );
  {{ else }}
  define( 'WP_SITEURL', '{{ $config.DeployURL }}/wp' );
  {{ end }}
}
if ( ! defined( 'WP_HOME' ) ) {
  // define( 'WP_HOME', 'http://' . $_SERVER['HTTP_HOST'] );
  define( 'WP_HOME', '{{ $config.DeployURL }}' );
}
// ============================================
// Custom Content Directory.
// Can also change them in local/dev-config.php
// ============================================
if ( ! defined( 'WP_CONTENT_DIR' ) ) {
  {{ if $config.WPGeneric }}
  define( 'WP_CONTENT_DIR', dirname( __FILE__ ) . '/wp-content' );
  {{ else }}
  define( 'WP_CONTENT_DIR', dirname( __FILE__ ) . '/content' );
  {{ end }}
  //define( 'WP_CONTENT_DIR', '{{ $config.Docroot }}/content' );
}
if ( ! defined( 'WP_CONTENT_URL' ) )
  {{ if $config.WPGeneric }}
  define( 'WP_CONTENT_URL', WP_HOME . '/wp-content' );
  {{ else }}
  define( 'WP_CONTENT_URL', WP_HOME . '/content' );
  {{ end }}

// Allows for WP to work behind an reverse proxy with HTTPS
if (isset($_SERVER['HTTP_X_FORWARDED_PROTO']) && $_SERVER['HTTP_X_FORWARDED_PROTO'] == 'https')
       $_SERVER['HTTPS']='on';

// ==============================================================
// Salts, for security
// Grab these from: https://api.wordpress.org/secret-key/1.1/salt
// ==============================================================
define( 'AUTH_KEY',         '{{ $config.AuthKey }}' );
define( 'SECURE_AUTH_KEY',  '{{ $config.SecureAuthKey }}' );
define( 'LOGGED_IN_KEY',    '{{ $config.LoggedInKey }}' );
define( 'NONCE_KEY',        '{{ $config.NonceKey }}' );
define( 'AUTH_SALT',        '{{ $config.AuthSalt }}' );
define( 'SECURE_AUTH_SALT', '{{ $config.SecureAuthSalt }}' );
define( 'LOGGED_IN_SALT',   '{{ $config.LoggedInSalt }}' );
define( 'NONCE_SALT',       '{{ $config.NonceSalt }}' );


// ==============================================================
// Table prefix
// Change this if you have multiple installs in the same database
// ==============================================================
$table_prefix  = '{{ $config.TablePrefix }}';

// ================================
// Language
// Leave blank for American English
// ================================
if ( !defined( 'WPLANG' ) )
  define( 'WPLANG', '' );

// ===========
// Hide errors
// ===========
ini_set( 'display_errors', 0 );
if ( !defined( 'WP_DEBUG_DISPLAY' ) )
  define( 'WP_DEBUG_DISPLAY', false );

// ============================================
// Debug mode
// Can also enable them in local/dev-config.php
// ============================================
{{ if eq $config.DeployName "default" }}
define( 'WP_DEBUG', true );
if ( !defined( 'SAVEQUERIES' ) )
  define( 'SAVEQUERIES', true );
if ( !defined( 'SCRIPT_DEBUG' ) )
  define('SCRIPT_DEBUG', true);
if ( !defined( 'WP_ENV' ) )
  define('WP_ENV', 'development');
{{ else }}
define( 'WP_DEBUG', false );
if ( !defined( 'SAVEQUERIES' ) )
  define( 'SAVEQUERIES', false );
if ( !defined( 'SCRIPT_DEBUG' ) )
  define('SCRIPT_DEBUG', false);
if ( !defined( 'WP_ENV' ) )
  define('WP_ENV', 'production');
{{ end }}

// =====================================
// Change Autosave Interval - in seconds
// =====================================
if ( !defined( 'AUTOSAVE_INTERVAL' ) )
  define('AUTOSAVE_INTERVAL', 240 );

// ==============================================================
// Configure Post Revisions - false if you don't want to save any
// ==============================================================
if ( !defined( 'WP_POST_REVISIONS' ) )
  define( 'WP_POST_REVISIONS', 3 ); // or false

// ========================================
// Remove Trash - In days, WP default is 30
// ========================================
define( 'EMPTY_TRASH_DAYS', 60 );

// =========================
// Increase PHP Memory Limit
// =========================
if ( !defined( 'WP_MEMORY_LIMIT' ) )
  define( 'WP_MEMORY_LIMIT', '128M' );

// =============================================
// Dis-Allow Plugin / Theme - Editing / Updating
// =============================================
if ( !defined( 'DISALLOW_FILE_EDIT' ) ) // editor
  define('DISALLOW_FILE_EDIT', true);
if ( !defined( 'DISALLOW_FILE_MODS' ) ) // updates
  define( 'DISALLOW_FILE_MODS', true );

// =====================================
// WP - Core only updates the core files
// No Akisemet or Hello Dolly
// =====================================
if ( !defined( 'CORE_UPGRADE_SKIP_NEW_BUNDLED' ) )
  define( 'CORE_UPGRADE_SKIP_NEW_BUNDLED', true );

// ===========================================
// Override default permissions
// If you want allow direct plugin downloading
// ===========================================
if ( !defined( 'FS_CHMOD_DIR' ) )
  define( 'FS_CHMOD_DIR', ( 0755 & ~ umask() ) );
if ( !defined( 'FS_CHMOD_FILE' ) )
  define( 'FS_CHMOD_FILE', ( 0644 & ~ umask() ) );

// ======================================
// Load a Memcached config if we have one
// ======================================
if ( file_exists( dirname( __FILE__ ) . '/memcached.php' ) )
  $memcached_servers = include( dirname( __FILE__ ) . '/memcached.php' );

// ===========================================================================================
// This can be used to programatically set the stage when deploying (e.g. production, staging)
// ===========================================================================================
define( 'WP_STAGE', 'xxxx' );
define( 'STAGING_DOMAIN', 'xxxx' ); // Does magic in WP Stack to handle staging domain rewriting

// ========================================
// Absolute path to the WordPress directory
// ========================================
if ( !defined( 'ABSPATH' ) ) {
  // define( 'ABSPATH', dirname( __FILE__ ) . '/wp/' );
  define( 'ABSPATH', '{{ $config.Docroot }}' . '/wp/' );
}

// ====================
// Bootstraps WordPress
// ====================
require_once( ABSPATH . 'wp-settings.php' );
`
)

// WriteWordpressConfig dynamically produces valid wp-config.php file by combining a configuration
// object with a data-driven template.
func WriteWordpressConfig(wordpressConfig *model.WordpressConfig, filePath string) error {
	tmpl, err := template.New("wordpressConfig").Funcs(sprig.TxtFuncMap()).Parse(wordpressTemplate)
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
	return nil
}
