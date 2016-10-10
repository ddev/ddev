package config

import (
	"os"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/drud/bootstrap/cli/cms/model"
)

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

$drupal_hash_salt = '{{ $config.HashSalt }}';
$base_url = '{{ $config.DeployURL }}';

ini_set('session.gc_probability', 1);
ini_set('session.gc_divisor', 100);
ini_set('session.gc_maxlifetime', 200000);
ini_set('session.cookie_lifetime', 2000000);

$conf['404_fast_paths_exclude'] = '/\/(?:styles)\//';
$conf['404_fast_paths'] = '/\.(?:txt|png|gif|jpe?g|css|js|ico|swf|flv|cgi|bat|pl|dll|exe|asp)$/i';
$conf['404_fast_html'] = '<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML+RDFa 1.0//EN" "http://www.w3.org/MarkUp/DTD/xhtml-rdfa-1.dtd"><html xmlns="http://www.w3.org/1999/xhtml"><head><title>404 Not Found</title></head><body><h1>Not Found</h1><p>The requested URL "@path" was not found on this server.</p></body></html>';

if (isset($_SERVER['HTTP_X_FORWARDED_PROTO']) &&
  $_SERVER['HTTP_X_FORWARDED_PROTO'] == 'https') {
  $_SERVER['HTTPS'] = 'on';
}

$conf['preprocess_css'] = '1';
$conf['preprocess_js'] = '1';
$conf['cache'] = '1';

if (file_exists(__DIR__ . '/custom.settings.php')) {
  include __DIR__ . '/custom.settings.php';
}

// This is super ugly but it determines whether or not drush should include a custom settings file which allows
// it to work both within a docker container and natively on the host system.
if (!empty($_SERVER["argv"]) && strpos($_SERVER["argv"][0], "drush") && empty($_ENV['DEPLOY_NAME'])) {
  include __DIR__ . '../../../../drush.settings.php';
}
`
)

const drushTemplate = `<?php
{{ $config := . }}
$databases['default']['default'] = array(
  'database' => "data",
  'username' => "root",
  'password' => "root",
  'host' => "127.0.0.1",
  'driver' => "mysql",
  'port' => {{ $config.DatabasePort }},
  'prefix' => "",
);

`

// WriteDrupalConfig dynamically produces valid settings.php file by combining a configuration
// object with a data-driven template.
func WriteDrupalConfig(drupalConfig *model.DrupalConfig, filePath string) error {
	tmpl, err := template.New("drupalConfig").Funcs(sprig.TxtFuncMap()).Parse(drupalTemplate)
	if err != nil {
		return err
	}
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	err = tmpl.Execute(file, drupalConfig)
	if err != nil {
		return err
	}
	return nil
}

func WriteDrushConfig(drushConfig *model.DrushConfig, filePath string) error {
	tmpl, err := template.New("drushConfig").Funcs(sprig.TxtFuncMap()).Parse(drushTemplate)
	if err != nil {
		return err
	}
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	err = tmpl.Execute(file, drushConfig)
	if err != nil {
		return err
	}
	return nil
}
