<?php
/**
 * #ddev-generated: Automatically generated WordPress settings file.
 * ddev manages this file and may delete or overwrite the file unless this comment is removed.
 *
 * @package ddevapp
 */

if ( getenv( 'IS_DDEV_PROJECT' ) == 'true' ) {
	/** The name of the database for WordPress */
	defined( 'DB_NAME' ) || define( 'DB_NAME', 'db' );

	/** MySQL database username */
	defined( 'DB_USER' ) || define( 'DB_USER', 'db' );

	/** MySQL database password */
	defined( 'DB_PASSWORD' ) || define( 'DB_PASSWORD', 'db' );

	/** MySQL hostname */
	defined( 'DB_HOST' ) || define( 'DB_HOST', 'db' );

	/** WP_HOME URL comes from DDEV_PRIMARY_URL */
	// Use a ddev-prefixed variable name to avoid conflicts with plugins or snippets that may define $wp_home.
	$ddev_wp_home = getenv( 'DDEV_PRIMARY_URL' );
	defined( 'WP_HOME' ) || define( 'WP_HOME', $ddev_wp_home );

	/** WP_SITEURL location */
	defined( 'WP_SITEURL' ) || define( 'WP_SITEURL', WP_HOME );

	/** Enable debug */
	defined( 'WP_DEBUG' ) || define( 'WP_DEBUG', true );

	/**
	 * Set WordPress Database Table prefix if not already set.
	 *
	 * @global string $table_prefix
	 */
	if ( ! isset( $table_prefix ) || empty( $table_prefix ) ) {
		// phpcs:disable WordPress.WP.GlobalVariablesOverride.Prohibited
		$table_prefix = 'wp_';
		// phpcs:enable
	}
}
