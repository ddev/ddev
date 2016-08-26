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
