<?php
{{ $config := . }}/**
 * #ddev-generated: Automatically generated WordPress settings file.
 * ddev manages this file and may delete or overwrite the file unless this comment is removed.
 * It is recommended that you leave this file alone.
 *
 * @package ddevapp
 */

/** Database charset to use in creating database tables. */
define( 'DB_CHARSET', '{{ $config.DbCharset }}' );

/** The database collate type. Don't change this if in doubt. */
define( 'DB_COLLATE', '{{ $config.DbCollate }}' );

/** Authentication Unique Keys and Salts. */
define( 'AUTH_KEY', '{{ $config.AuthKey }}' );
define( 'SECURE_AUTH_KEY', '{{ $config.SecureAuthKey }}' );
define( 'LOGGED_IN_KEY', '{{ $config.LoggedInKey }}' );
define( 'NONCE_KEY', '{{ $config.NonceKey }}' );
define( 'AUTH_SALT', '{{ $config.AuthSalt }}' );
define( 'SECURE_AUTH_SALT', '{{ $config.SecureAuthSalt }}' );
define( 'LOGGED_IN_SALT', '{{ $config.LoggedInSalt }}' );
define( 'NONCE_SALT', '{{ $config.NonceSalt }}' );

/** Absolute path to the WordPress directory. */
defined( 'ABSPATH' ) || define( 'ABSPATH', dirname( __FILE__ ) . '/{{ $config.AbsPath }}' );

// Include for settings managed by ddev.
$ddev_settings = dirname( __FILE__ ) . '/wp-config-ddev.php';
if ( ! defined( 'DB_USER' ) && getenv( 'IS_DDEV_PROJECT' ) == 'true' && is_readable( $ddev_settings ) ) {
	require_once( $ddev_settings );
}

/** Include wp-settings.php */
if ( file_exists( ABSPATH . '/wp-settings.php' ) ) {
	require_once ABSPATH . '/wp-settings.php';
}
