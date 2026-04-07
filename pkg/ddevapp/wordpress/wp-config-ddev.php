<?php
/**
 * #ddev-generated: Automatically generated WordPress settings file.
 * ddev manages this file and may delete or overwrite the file unless this comment is removed.
 *
 * @package ddevapp
 */

if ( getenv( 'IS_DDEV_PROJECT' ) !== 'true' ) {
	return;
}

/** The name of the database for WordPress */
defined( 'DB_NAME' ) || define( 'DB_NAME', getenv( 'DB_NAME' ) ?: 'db' );

/** MySQL database username */
defined( 'DB_USER' ) || define( 'DB_USER', getenv( 'DB_USER' ) ?: 'db' );

/** MySQL database password */
defined( 'DB_PASSWORD' ) || define( 'DB_PASSWORD', getenv( 'DB_PASSWORD' ) ?: 'db' );

/** MySQL hostname */
defined( 'DB_HOST' ) || define( 'DB_HOST', getenv( 'DB_HOST' ) ?: 'db' );

/** WP_HOME URL */
defined( 'WP_HOME' ) || define( 'WP_HOME', getenv( 'DDEV_PRIMARY_URL' ) ?: 'http://localhost' );

/** WP_SITEURL location */
defined( 'WP_SITEURL' ) || define(
	'WP_SITEURL',
	WP_HOME . '/' . ltrim(
		str_replace(
			realpath( getenv( 'DDEV_APPROOT' ) . '/' . getenv( 'DDEV_DOCROOT' ) ),
			'',
			realpath( ABSPATH )
		),
		'/'
	)
);

/** Enable debug (can be disabled with `ddev config --web-environment-add=WP_DEBUG=false`) */
defined( 'WP_DEBUG' ) || define( 'WP_DEBUG', getenv( 'WP_DEBUG' ) === false || getenv( 'WP_DEBUG' ) === 'true' );

/**
 * Set WordPress Database Table prefix if not already set.
 *
 * @global string $table_prefix
 */
if ( empty( $table_prefix ) ) {
	// phpcs:disable WordPress.WP.GlobalVariablesOverride.Prohibited
	$table_prefix = getenv('DB_PREFIX') ?: 'wp_';
	// phpcs:enable
}
