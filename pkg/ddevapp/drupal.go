package ddevapp

import (
	"fmt"
	"github.com/Masterminds/sprig"
	"github.com/drud/ddev/pkg/appports"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"os"
	"path/filepath"
	"text/template"
)

// DrupalConfig encapsulates all the configurations for a Drupal site.
type DrupalConfig struct {
	DeployName       string
	DeployURL        string
	DatabaseName     string
	DatabaseUsername string
	DatabasePassword string
	DatabaseHost     string
	DatabaseDriver   string
	DatabasePort     string
	DatabasePrefix   string
	HashSalt         string
	IsDrupal8        bool
	Signature        string
}

// NewDrupalConfig produces a DrupalConfig object with default.
func NewDrupalConfig() *DrupalConfig {
	return &DrupalConfig{
		DatabaseName:     "db",
		DatabaseUsername: "db",
		DatabasePassword: "db",
		DatabaseHost:     "db",
		DatabaseDriver:   "mysql",
		DatabasePort:     appports.GetPort("db"),
		DatabasePrefix:   "",
		IsDrupal8:        false,
		HashSalt:         util.RandString(64),
		Signature:        DdevSettingsFileSignature,
	}
}

// DrushConfig encapsulates configuration for a drush settings file.
type DrushConfig struct {
	DatabasePort string
	DatabaseHost string
	IsDrupal8    bool
}

// NewDrushConfig produces a DrushConfig object with default.
func NewDrushConfig() *DrushConfig {
	return &DrushConfig{
		DatabaseHost: "127.0.0.1",
		DatabasePort: appports.GetPort("db"),
		IsDrupal8:    false,
	}
}

const (
	drupalTemplate = `<?php
{{ $config := . }}
/**
 {{ $config.Signature }}: Automatically generated Drupal settings.php file.
 ddev manages this file and may delete or overwrite the file unless this comment is removed.
 */

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


{{ else }}

$drupal_hash_salt = '{{ $config.HashSalt }}';
{{ end }}


// This is super ugly but it determines whether or not drush should include a custom settings file which allows
// it to work both within a docker container and natively on the host system.
if (!empty($_SERVER["argv"]) && strpos($_SERVER["argv"][0], "drush") && empty($_ENV['DEPLOY_NAME'])) {
  include __DIR__ . '../../../drush.settings.php';
}
`
)

const drushTemplate = `<?php
{{ $config := . }}
$databases['default']['default'] = array(
  'database' => "db",
  'username' => "db",
  'password' => "db",
  'host' => "127.0.0.1",
  'driver' => "mysql",
  'port' => {{ $config.DatabasePort }},
  'prefix' => "",
);
`

// CreateDrupalSettingsFile creates the app's settings.php or equivalent,
// adding things like database host, name, and password
func CreateDrupalSettingsFile(l *DdevApp) error {

	settingsFilePath, err := setUpSettingsFile(l)
	if err != nil {
		return fmt.Errorf("Failed to set up settings file: %v", err)
	}

	output.UserOut.Printf("Generating %s file for database connection.", filepath.Base(settingsFilePath))

	// Currently there isn't any customization done for the drupal config, but
	// we may want to in the future.
	drupalConfig := NewDrupalConfig()

	err = WriteDrupalSettingsFile(drupalConfig, settingsFilePath)
	if err != nil {
		return err
	}

	return nil
}

// WriteDrupalSettingsFile dynamically produces valid settings.php file by combining a configuration
// object with a data-driven template.
func WriteDrupalSettingsFile(drupalConfig *DrupalConfig, filePath string) error {
	tmpl, err := template.New("drupalConfig").Funcs(sprig.TxtFuncMap()).Parse(drupalTemplate)
	if err != nil {
		return err
	}

	// Ensure target directory is writable.
	dir := filepath.Dir(filePath)
	err = os.Chmod(dir, 0755)
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
	util.CheckClose(file)
	return nil
}

// WriteDrushConfig writes out a drush config based on passed-in values.
func WriteDrushConfig(drushConfig *DrushConfig, filePath string) error {
	tmpl, err := template.New("drushConfig").Funcs(sprig.TxtFuncMap()).Parse(drushTemplate)
	if err != nil {
		return err
	}

	// Ensure target directory is writable.
	dir := filepath.Dir(filePath)
	err = os.Chmod(dir, 0755)
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
	util.CheckClose(file)
	return nil
}

// GetDrupalUploadDir just returns a static upload files (public files) dir.
// This can be made more sophisticated in the future, for example by adding
// the directory to the ddev config.yaml.
func GetDrupalUploadDir(l *DdevApp) string {
	return "sites/default/files"
}

// Drupal8Hooks adds a d8-specific hooks example for post-import-db
const Drupal8Hooks = `
#     - exec: "drush cr"`

// Drupal7Hooks adds a d7-specific hooks example for post-import-db
const Drupal7Hooks = `
#     - exec: "drush cc all"`

// GetDrupal7Hooks for appending as byte array
func GetDrupal7Hooks() []byte {
	return []byte(Drupal7Hooks)
}

// GetDrupal8Hooks for appending as byte array
func GetDrupal8Hooks() []byte {
	return []byte(Drupal8Hooks)
}

func SetDrupalSiteSettingsPaths(app *DdevApp) {
	settingsFileBasePath := filepath.Join(app.AppRoot, app.Docroot)
	var settingsFilePath, localSettingsFilePath string
	settingsFilePath = filepath.Join(settingsFileBasePath, "sites", "default", "settings.php")
	localSettingsFilePath = filepath.Join(settingsFileBasePath, "sites", "default", "settings.local.php")
	app.SiteSettingsPath = settingsFilePath
	app.SiteLocalSettingsPath = localSettingsFilePath
}
