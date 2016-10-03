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
`
)

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
