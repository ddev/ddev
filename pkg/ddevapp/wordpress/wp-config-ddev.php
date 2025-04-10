<?php
{{ $config := . }}/**
 * #ddev-generated: Automatically generated WordPress settings file.
 * ddev manages this file and may delete or overwrite the file unless this comment is removed.
 *
 * @package ddevapp
 */

if ( getenv( 'IS_DDEV_PROJECT' ) == 'true' ) {
	/** The name of the database for WordPress */
	defined( 'DB_NAME' ) || define( 'DB_NAME', '{{ $config.DatabaseName }}' );

	/** MySQL database username */
	defined( 'DB_USER' ) || define( 'DB_USER', '{{ $config.DatabaseUsername }}' );

	/** MySQL database password */
	defined( 'DB_PASSWORD' ) || define( 'DB_PASSWORD', '{{ $config.DatabasePassword }}' );

	/** MySQL hostname */
	defined( 'DB_HOST' ) || define( 'DB_HOST', '{{ $config.DatabaseHost }}' );

	/** WP_HOME URL */
	defined( 'WP_HOME' ) || define( 'WP_HOME', '{{ $config.DeployURL }}' );

	/** WP_SITEURL location */
	defined( 'WP_SITEURL' ) || define( 'WP_SITEURL', WP_HOME . '/{{ $config.AbsPath  }}' );

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
