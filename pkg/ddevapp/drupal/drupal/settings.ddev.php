<?php
{{ $config := . }}
/**
 * @file
 * {{ $config.Signature }}: Automatically generated Drupal settings file.
 * ddev manages this file and may delete or overwrite the file unless this
 * comment is removed.  It is recommended that you leave this file alone.
 */

$host = "{{ $config.DatabaseHost }}";
$port = {{ $config.DatabasePort }};
$driver = "{{ $config.DatabaseDriver }}";

$databases['default']['default']['database'] = "{{ $config.DatabaseName }}";
$databases['default']['default']['username'] = "{{ $config.DatabaseUsername }}";
$databases['default']['default']['password'] = "{{ $config.DatabasePassword }}";
$databases['default']['default']['host'] = $host;
$databases['default']['default']['port'] = $port;
$databases['default']['default']['driver'] = $driver;

$settings['hash_salt'] = '{{ $config.HashSalt }}';

// This will prevent Drupal from setting read-only permissions on sites/default.
$settings['skip_permissions_hardening'] = TRUE;

// This will ensure the site can only be accessed through the intended host
// names. Additional host patterns can be added for custom configurations.
$settings['trusted_host_patterns'] = ['.*'];

// Don't use Symfony's APCLoader. ddev includes APCu; Composer's APCu loader has
// better performance.
$settings['class_loader_auto_detect'] = FALSE;

// Set $settings['config_sync_directory'] if not set in settings.php.
if (empty($settings['config_sync_directory'])) {
  $settings['config_sync_directory'] = 'sites/default/files/sync';
}

// Override drupal/symfony_mailer default config to use Mailpit
$config['symfony_mailer.settings']['default_transport'] = 'sendmail';
$config['symfony_mailer.mailer_transport.sendmail']['plugin'] = 'smtp';
$config['symfony_mailer.mailer_transport.sendmail']['configuration']['user']='';
$config['symfony_mailer.mailer_transport.sendmail']['configuration']['pass']='';
$config['symfony_mailer.mailer_transport.sendmail']['configuration']['host']='localhost';
$config['symfony_mailer.mailer_transport.sendmail']['configuration']['port']='1025';

// Enable verbose logging for errors.
// https://www.drupal.org/forum/support/post-installation/2018-07-18/enable-drupal-8-backend-errorlogdebugging-mode
$config['system.logging']['error_level'] = 'verbose';
