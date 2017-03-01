package tpl

const (
	drupalTemplate = `<?php
{{ $config := . }}
/* Automatically generated Drupal settings.php file. */

$databases['default']['default'] = array(
  'database' => "{{ $config.DatabaseName }}",
  'username' => "{{ $config.DatabaseUsername }}",
  'password' => "{{ $config.DatabasePassword }}",
  'host' => "{{ $config.DatabaseHost }}",
  'driver' => "{{ $config.DatabaseDriver }}",
  'port' => {{ $config.DatabasePort }},
  'prefix' => "{{ $config.DatabasePrefix }}",
);

ini_set('session.gc_probability', 1);
ini_set('session.gc_divisor', 100);
ini_set('session.gc_maxlifetime', 200000);
ini_set('session.cookie_lifetime', 2000000);

{{ if $config.IsDrupal8 }}

$settings['hash_salt'] = '{{ $config.HashSalt }}';

$settings['file_scan_ignore_directories'] = [
  'node_modules',
  'bower_components',
];

 $config_directories = array(
   CONFIG_SYNC_DIRECTORY => '{{ $config.ConfigSyncDir }}',
 );


{{ else }}

$drupal_hash_salt = '{{ $config.HashSalt }}';
$base_url = '{{ $config.SiteURL }}';

if (isset($_SERVER['HTTP_X_FORWARDED_PROTO']) &&
  $_SERVER['HTTP_X_FORWARDED_PROTO'] == 'https') {
  $_SERVER['HTTPS'] = 'on';
}
{{ end }}

// This allows you to provide a configuration file in your site's code base for
// configurations that should be present in any environment.
if (file_exists(__DIR__ . '/settings.custom.php')) {
  include __DIR__ . '/settings.custom.php';
}

// This allows you to provide a configuration file in your site's code base for
// configurations that should be present for a local development environment.
if (isset($_ENV['DEPLOY_NAME']) && $_ENV['DEPLOY_NAME'] == 'local' && file_exists(__DIR__ . '/settings.local.php')) {
  include __DIR__ . '/settings.local.php';
}


// This is super ugly but it determines whether or not drush should include a custom settings file which allows
// it to work both within a docker container and natively on the host system.
if (!empty($_SERVER["argv"]) && strpos($_SERVER["argv"][0], "drush") && empty($_ENV['DEPLOY_NAME'])) {
  include __DIR__ . '../../../../drush.settings.php';
}
`
)
