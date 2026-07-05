<?php
/**
 * #ddev-generated: Automatically generated MODX config.inc.php file.
 * ddev manages this file and may delete or overwrite the file unless this comment is removed.
 * It is recommended that you leave this file alone.
 *
 * This file is valid for both MODX Revolution 2.x and 3.x. The database
 * credentials are the fixed DDEV values, and the path/URL constants are derived
 * from MODX_CORE_PATH (defined by the entry script's config.core.php, or its
 * fallback) so the file is independent of the absolute install path and docroot.
 */

$database_type = 'mysql';
$database_server = '{{ .DBHostname }}';
$database_user = 'db';
$database_password = 'db';
$database_connection_charset = '{{ .DBCharset }}';
$dbase = 'db';
$table_prefix = '{{ .TablePrefix }}';
$database_dsn = 'mysql:host={{ .DBHostname }};dbname=db;charset={{ .DBCharset }}';
$config_options = [];
$driver_options = [];

$lastInstallTime = 0;

$site_id = 'ddevmodxsite';
$site_sessionname = 'modxddev';
$https_port = '443';
$uuid = '';

/* Derive the filesystem base path from MODX_CORE_PATH, which the entry script
 * (config.core.php, or the index.php fallback) defines before this file loads.
 * config.inc.php lives at core/config/, so this file's grandparent is the core's
 * parent (the web root) for a standard install. */
$modx_core_path = defined('MODX_CORE_PATH') ? MODX_CORE_PATH : dirname(__DIR__) . '/';
$modx_base_path = dirname($modx_core_path) . '/';

if (!defined('MODX_CORE_PATH')) {
    define('MODX_CORE_PATH', $modx_core_path);
}
if (!defined('MODX_PROCESSORS_PATH')) {
    // Legacy constant; only used by MODX 2.x and older third-party processors.
    define('MODX_PROCESSORS_PATH', MODX_CORE_PATH . 'model/modx/processors/');
}
if (!defined('MODX_CONNECTORS_PATH')) {
    define('MODX_CONNECTORS_PATH', $modx_base_path . 'connectors/');
    define('MODX_CONNECTORS_URL', '/connectors/');
}
if (!defined('MODX_MANAGER_PATH')) {
    define('MODX_MANAGER_PATH', $modx_base_path . 'manager/');
    define('MODX_MANAGER_URL', '/manager/');
}
if (!defined('MODX_BASE_PATH')) {
    define('MODX_BASE_PATH', $modx_base_path);
    define('MODX_BASE_URL', '/');
}
if (!defined('MODX_ASSETS_PATH')) {
    define('MODX_ASSETS_PATH', $modx_base_path . 'assets/');
    define('MODX_ASSETS_URL', '/assets/');
}

/* Determine the request scheme and host dynamically so the generated file works
 * at the project's ddev.site URL without hard-coding it. The '{{ .HTTPHost }}'
 * literal is only the fallback host for CLI/API requests, where HTTP_HOST is
 * not present. */
if (defined('PHP_SAPI') && (PHP_SAPI === 'cli' || PHP_SAPI === 'embed')) {
    $isSecureRequest = false;
} else {
    $isSecureRequest = ((isset($_SERVER['HTTPS']) && !empty($_SERVER['HTTPS']) && strtolower($_SERVER['HTTPS']) !== 'off') || (isset($_SERVER['HTTP_HOST']) && parse_url('http://' . $_SERVER['HTTP_HOST'], PHP_URL_PORT) == $https_port));
}
if (!defined('MODX_URL_SCHEME')) {
    define('MODX_URL_SCHEME', $isSecureRequest ? 'https://' : 'http://');
}
if (!defined('MODX_HTTP_HOST')) {
    if (defined('PHP_SAPI') && (PHP_SAPI === 'cli' || PHP_SAPI === 'embed')) {
        define('MODX_HTTP_HOST', '{{ .HTTPHost }}');
    } else {
        $http_host = array_key_exists('HTTP_HOST', $_SERVER) ? parse_url(MODX_URL_SCHEME . $_SERVER['HTTP_HOST'], PHP_URL_HOST) : '{{ .HTTPHost }}';
        $http_port = array_key_exists('HTTP_HOST', $_SERVER) ? parse_url(MODX_URL_SCHEME . $_SERVER['HTTP_HOST'], PHP_URL_PORT) : null;
        $http_host .= in_array($http_port, [null, 80, 443]) ? '' : ':' . $http_port;
        define('MODX_HTTP_HOST', $http_host);
    }
}
if (!defined('MODX_SITE_URL')) {
    define('MODX_SITE_URL', MODX_URL_SCHEME . MODX_HTTP_HOST . MODX_BASE_URL);
}

if (!defined('MODX_LOG_LEVEL_FATAL')) {
    define('MODX_LOG_LEVEL_FATAL', 0);
    define('MODX_LOG_LEVEL_ERROR', 1);
    define('MODX_LOG_LEVEL_WARN', 2);
    define('MODX_LOG_LEVEL_INFO', 3);
    define('MODX_LOG_LEVEL_DEBUG', 4);
}
