package ddevapp

import (
	"fmt"

	"github.com/drud/ddev/pkg/appports"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"

	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/drud/ddev/pkg/fileutil"
)

// DrupalSettings encapsulates all the configurations for a Drupal site.
type DrupalSettings struct {
	DeployName        string
	DeployURL         string
	DatabaseName      string
	DatabaseUsername  string
	DatabasePassword  string
	DatabaseHost      string
	DatabaseDriver    string
	DatabasePort      string
	DatabasePrefix    string
	HashSalt          string
	Signature         string
	SitePath          string
	SiteSettings      string
	SiteSettingsLocal string
	SyncDir           string
}

// NewDrupalSettings produces a DrupalSettings object with default.
func NewDrupalSettings() *DrupalSettings {
	return &DrupalSettings{
		DatabaseName:      "db",
		DatabaseUsername:  "db",
		DatabasePassword:  "db",
		DatabaseHost:      "db",
		DatabaseDriver:    "mysql",
		DatabasePort:      appports.GetPort("db"),
		DatabasePrefix:    "",
		HashSalt:          util.RandString(64),
		Signature:         DdevFileSignature,
		SitePath:          filepath.Join("sites", "default"),
		SiteSettings:      "settings.php",
		SiteSettingsLocal: "settings.ddev.php",
		SyncDir:           filepath.Join("files", "sync"),
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

// drupalCommonSettingsTemplate defines the template that will become settings.php
// in the even that one does not already exist.
const drupalCommonSettingsTemplate = `<?php
{{ $config := . }}
// {{ $config.Signature }}: Automatically generated Drupal settings file.
if (file_exists('{{ joinPath $config.SitePath $config.SiteSettingsLocal }}')) {
  include '{{ joinPath $config.SitePath $config.SiteSettingsLocal }}';
}
`

// drupalCommonSettingsAppendTemplate defines the template that will be appended to
// settings.php in the event that one exists.
const drupalCommonSettingsAppendTemplate = `{{ $config := . }}
// Automatically generated include for settings managed by ddev.
if (file_exists('{{ joinPath $config.SitePath $config.SiteSettingsLocal }}')) {
  include '{{ joinPath $config.SitePath $config.SiteSettingsLocal }}';
}
`

const (
	drupal8Template = `<?php
{{ $config := . }}
/**
 {{ $config.Signature }}: Automatically generated Drupal settings file.
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

$settings['hash_salt'] = '{{ $config.HashSalt }}';

$settings['file_scan_ignore_directories'] = [
  'node_modules',
  'bower_components',
];

// This will prevent Drupal from setting read-only permissions on sites/default.
$settings['skip_permissions_hardening'] = TRUE;

// This will ensure the site can only be accessed through the intended host names.
// Additional host patterns can be added for custom configurations.
$settings['trusted_host_patterns'] = ['.*'];

// This specifies the default configuration sync directory.
if (empty($config_directories[CONFIG_SYNC_DIRECTORY])) {
  $config_directories[CONFIG_SYNC_DIRECTORY] = '{{ joinPath $config.SitePath $config.SyncDir }}';
}

// This determines whether or not drush should include a custom settings file which allows
// it to work both within a docker container and natively on the host system.
if (!empty($_SERVER["argv"]) && strpos($_SERVER["argv"][0], "drush") && empty($_ENV['DEPLOY_NAME'])) {
  include __DIR__ . '../../../drush.settings.php';
}
`
)

const (
	drupal7Template = `<?php
{{ $config := . }}
/**
 {{ $config.Signature }}: Automatically generated Drupal settings file.
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

$drupal_hash_salt = '{{ $config.HashSalt }}';

// This determines whether or not drush should include a custom settings file which allows
// it to work both within a docker container and natively on the host system.
if (!empty($_SERVER["argv"]) && strpos($_SERVER["argv"][0], "drush") && empty($_ENV['DEPLOY_NAME'])) {
  include __DIR__ . '../../../drush.settings.php';
}
`
)

const (
	drupal6Template = `<?php
{{ $config := . }}
/**
 {{ $config.Signature }}: Automatically generated Drupal settings file.
 ddev manages this file and may delete or overwrite the file unless this comment is removed.
 */

$db_url = '{{ $config.DatabaseDriver }}://{{ $config.DatabaseUsername }}:{{ $config.DatabasePassword }}@{{ $config.DatabaseHost }}:{{ $config.DatabasePort }}/{{ $config.DatabaseName }}';

ini_set('session.gc_probability', 1);
ini_set('session.gc_divisor', 100);
ini_set('session.gc_maxlifetime', 200000);
ini_set('session.cookie_lifetime', 2000000);

// This determines whether or not drush should include a custom settings file which allows
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

// manageDrupalCommonSettingsFile will direct inspecting and writing of settings.php.
func manageDrupalCommonSettingsFile(app *DdevApp, drupalConfig *DrupalSettings) error {
	if !fileutil.FileExists(app.SiteSettingsPath) {
		output.UserOut.Printf("No %s file exists, creating one", drupalConfig.SiteSettings)

		if err := writeDrupalCommonSettingsFile(drupalConfig, app.SiteSettingsPath); err != nil {
			return fmt.Errorf("failed to write %s: %v", app.SiteSettingsPath, err)
		}
	}

	included, err := settingsHasInclude(drupalConfig, app.SiteSettingsPath)
	if err != nil {
		return fmt.Errorf("failed to check for include in %s: %v", app.SiteSettingsPath, err)
	}

	if included {
		output.UserOut.Printf("Existing %s file includes %s", drupalConfig.SiteSettings, drupalConfig.SiteSettingsLocal)
	} else {
		output.UserOut.Printf("Existing %s file does not include %s, modifying to include ddev settings", drupalConfig.SiteSettings, drupalConfig.SiteSettingsLocal)

		if err := appendIncludeToDrupalSettingsFile(drupalConfig, app.SiteSettingsPath); err != nil {
			return fmt.Errorf("failed to include %s in %s: %v", drupalConfig.SiteSettingsLocal, drupalConfig.SiteSettings, err)
		}
	}

	return nil
}

// writeDrupalCommonSettingsFile creates the app's settings.php or equivalent,
// which does nothing more than import the ddev-managed settings.ddev.php.
func writeDrupalCommonSettingsFile(drupalConfig *DrupalSettings, filePath string) error {
	tmpl, err := template.New("settings").Funcs(getTemplateFuncMap()).Parse(drupalCommonSettingsTemplate)
	if err != nil {
		return err
	}

	// Ensure target directory is writable.
	dir := filepath.Dir(filePath)
	if err = os.Chmod(dir, 0755); err != nil {
		return err
	}

	// Create file
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer util.CheckClose(file)

	if err := tmpl.Execute(file, drupalConfig); err != nil {
		return err
	}

	return nil
}

// createDrupal7SettingsFile manages creation and modification of settings.php and settings.ddev.php.
// If a settings.php file already exists, it will be modified to ensure that it includes
// settings.ddev.php, which contains ddev-specific configuration.
func createDrupal7SettingsFile(app *DdevApp) (string, error) {
	// Currently there isn't any customization done for the drupal config, but
	// we may want to do some kind of customization in the future.
	drupalConfig := NewDrupalSettings()

	if err := manageDrupalCommonSettingsFile(app, drupalConfig); err != nil {
		return "", err
	}

	if err := writeDrupal7SettingsFile(drupalConfig, app.SiteLocalSettingsPath); err != nil {
		return "", fmt.Errorf("failed to write Drupal settings file %s: %v", app.SiteLocalSettingsPath, err)
	}

	if err := createGitIgnore(filepath.Dir(app.SiteLocalSettingsPath), drupalConfig.SiteSettingsLocal); err != nil {
		output.UserOut.Warnf("Failed to write .gitignore in %s: %v", filepath.Dir(app.SiteLocalSettingsPath), err)
	}

	return app.SiteLocalSettingsPath, nil
}

// createDrupal8SettingsFile manages creation and modification of settings.php and settings.ddev.php.
// If a settings.php file already exists, it will be modified to ensure that it includes
// settings.ddev.php, which contains ddev-specific configuration.
func createDrupal8SettingsFile(app *DdevApp) (string, error) {
	// Currently there isn't any customization done for the drupal config, but
	// we may want to do some kind of customization in the future.
	drupalConfig := NewDrupalSettings()

	if err := manageDrupalCommonSettingsFile(app, drupalConfig); err != nil {
		return "", err
	}

	if err := writeDrupal8SettingsFile(drupalConfig, app.SiteLocalSettingsPath); err != nil {
		return "", fmt.Errorf("failed to write Drupal settings file %s: %v", app.SiteLocalSettingsPath, err)
	}

	if err := createGitIgnore(filepath.Dir(app.SiteSettingsPath), drupalConfig.SiteSettingsLocal); err != nil {
		output.UserOut.Warnf("Failed to write .gitignore in %s: %v", filepath.Dir(app.SiteLocalSettingsPath), err)
	}

	return app.SiteLocalSettingsPath, nil
}

// createDrupal6SettingsFile manages creation and modification of settings.php and settings.ddev.php.
// If a settings.php file already exists, it will be modified to ensure that it includes
// settings.ddev.php, which contains ddev-specific configuration.
func createDrupal6SettingsFile(app *DdevApp) (string, error) {
	// Currently there isn't any customization done for the drupal config, but
	// we may want to do some kind of customization in the future.
	drupalConfig := NewDrupalSettings()

	if err := manageDrupalCommonSettingsFile(app, drupalConfig); err != nil {
		return "", err
	}

	if err := writeDrupal6SettingsFile(drupalConfig, app.SiteLocalSettingsPath); err != nil {
		return "", fmt.Errorf("failed to write Drupal settings file %s: %v", app.SiteLocalSettingsPath, err)
	}

	if err := createGitIgnore(filepath.Dir(app.SiteSettingsPath), drupalConfig.SiteSettingsLocal); err != nil {
		output.UserOut.Warnf("Failed to write .gitignore in %s: %v", filepath.Dir(app.SiteLocalSettingsPath), err)
	}

	return app.SiteLocalSettingsPath, nil
}

// writeDrupal8SettingsFile dynamically produces valid settings.ddev.php file by combining a configuration
// object with a data-driven template.
func writeDrupal8SettingsFile(settings *DrupalSettings, filePath string) error {
	tmpl, err := template.New("settings").Funcs(getTemplateFuncMap()).Parse(drupal8Template)
	if err != nil {
		return err
	}

	// Ensure target directory is writable.
	dir := filepath.Dir(filePath)
	if err = os.Chmod(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer util.CheckClose(file)

	if err := tmpl.Execute(file, settings); err != nil {
		return err
	}

	return nil
}

// writeDrupal7SettingsFile dynamically produces valid settings.ddev.php file by combining a configuration
// object with a data-driven template.
func writeDrupal7SettingsFile(settings *DrupalSettings, filePath string) error {
	tmpl, err := template.New("settings").Funcs(getTemplateFuncMap()).Parse(drupal7Template)
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
	err = tmpl.Execute(file, settings)
	if err != nil {
		return err
	}
	util.CheckClose(file)
	return nil
}

// writeDrupal6SettingsFile dynamically produces valid settings.ddev.php file by combining a configuration
// object with a data-driven template.
func writeDrupal6SettingsFile(settings *DrupalSettings, filePath string) error {
	tmpl, err := template.New("settings").Funcs(getTemplateFuncMap()).Parse(drupal6Template)
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
	err = tmpl.Execute(file, settings)
	if err != nil {
		return err
	}
	util.CheckClose(file)
	return nil
}

// WriteDrushConfig writes out a drush config based on passed-in values.
func WriteDrushConfig(drushConfig *DrushConfig, filePath string) error {
	tmpl, err := template.New("drushConfig").Funcs(getTemplateFuncMap()).Parse(drushTemplate)
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

// getDrupalUploadDir just returns a static upload files (public files) dir.
// This can be made more sophisticated in the future, for example by adding
// the directory to the ddev config.yaml.
func getDrupalUploadDir(app *DdevApp) string {
	return "sites/default/files"
}

// Drupal8Hooks adds a d8-specific hooks example for post-import-db
const Drupal8Hooks = `
# post-import-db:
#   - exec: drush cr
#   - exec: drush updb
`

// Drupal7Hooks adds a d7-specific hooks example for post-import-db
const Drupal7Hooks = `
#  post-import-db:
#    - exec: "drush cc all"`

// getDrupal7Hooks for appending as byte array
func getDrupal7Hooks() []byte {
	return []byte(Drupal7Hooks)
}

// getDrupal6Hooks for appending as byte array
func getDrupal6Hooks() []byte {
	// We don't have anything new to add yet, so just use Drupal7 version
	return []byte(Drupal7Hooks)
}

// getDrupal8Hooks for appending as byte array
func getDrupal8Hooks() []byte {
	return []byte(Drupal8Hooks)
}

// setDrupalSiteSettingsPaths sets the paths to settings.php/settings.local.php
// for templating.
func setDrupalSiteSettingsPaths(app *DdevApp) {
	drupalConfig := NewDrupalSettings()
	settingsFileBasePath := filepath.Join(app.AppRoot, app.Docroot)
	app.SiteSettingsPath = filepath.Join(settingsFileBasePath, drupalConfig.SitePath, drupalConfig.SiteSettings)
	app.SiteLocalSettingsPath = filepath.Join(settingsFileBasePath, drupalConfig.SitePath, drupalConfig.SiteSettingsLocal)
}

// isDrupal7App returns true if the app is of type drupal7
func isDrupal7App(app *DdevApp) bool {
	if _, err := os.Stat(filepath.Join(app.AppRoot, app.Docroot, "misc/ajax.js")); err == nil {
		return true
	}
	return false
}

// isDrupal8App returns true if the app is of type drupal8
func isDrupal8App(app *DdevApp) bool {
	if _, err := os.Stat(filepath.Join(app.AppRoot, app.Docroot, "core/scripts/drupal.sh")); err == nil {
		return true
	}
	return false
}

// isDrupal6App returns true if the app is of type Drupal6
func isDrupal6App(app *DdevApp) bool {
	if _, err := os.Stat(filepath.Join(app.AppRoot, app.Docroot, "misc/ahah.js")); err == nil {
		return true
	}
	return false
}

// drupal7ConfigOverrideAction sets a safe php_version for D7
func drupal7ConfigOverrideAction(app *DdevApp) error {
	app.PHPVersion = "7.1"
	return nil
}

// drupal6ConfigOverrideAction overrides php_version for D6, since it is incompatible
// with php7+
func drupal6ConfigOverrideAction(app *DdevApp) error {
	app.PHPVersion = "5.6"
	return nil
}

// drupal8PostStartAction handles default post-start actions for D8 apps, like ensuring
// useful permissions settings on sites/default.
func drupal8PostStartAction(app *DdevApp) error {
	if err := createDrupal8SyncDir(app); err != nil {
		return err
	}

	if err := drupalEnsureWritePerms(app); err != nil {
		return err
	}

	return nil
}

// drupal7PostStartAction handles default post-start actions for D7 apps, like ensuring
// useful permissions settings on sites/default.
func drupal7PostStartAction(app *DdevApp) error {
	if err := drupalEnsureWritePerms(app); err != nil {
		return err
	}

	return nil
}

// drupal6PostStartAction handles default post-start actions for D6 apps, like ensuring
// useful permissions settings on sites/default.
func drupal6PostStartAction(app *DdevApp) error {
	if err := drupalEnsureWritePerms(app); err != nil {
		return err
	}

	return nil
}

// drupalEnsureWritePerms will ensure sites/default and sites/default/settings.php will
// have the appropriate permissions for development.
func drupalEnsureWritePerms(app *DdevApp) error {
	output.UserOut.Printf("Ensuring write permissions for %s...", app.GetName())
	var writePerms os.FileMode = 0200

	settingsDir := path.Dir(app.SiteSettingsPath)
	makeWritable := []string{
		settingsDir,
		app.SiteSettingsPath,
		app.SiteLocalSettingsPath,
		path.Join(settingsDir, "services.yml"),
	}

	for _, o := range makeWritable {
		stat, err := os.Stat(o)
		// If the file doesn't exist, don't try to set the permissions.
		if os.IsNotExist(err) {
			continue
		}

		if err := os.Chmod(o, stat.Mode()|writePerms); err != nil {
			// Warn the user, but continue.
			util.Warning("Unable to set permissions: %v", err)
		}
	}

	return nil
}

// createDrupal8SyncDir creates a Drupal 8 app's sync directory
func createDrupal8SyncDir(app *DdevApp) error {
	// Currently there isn't any customization done for the drupal config, but
	// we may want to do some kind of customization in the future.
	drupalConfig := NewDrupalSettings()

	syncDirPath := path.Join(app.GetAppRoot(), app.GetDocroot(), drupalConfig.SyncDir)
	if fileutil.FileExists(syncDirPath) {
		return nil
	}

	if err := os.MkdirAll(syncDirPath, 0755); err != nil {
		return fmt.Errorf("failed to create sync directory (%s): %v", syncDirPath, err)
	}

	return nil
}

// settingsHasInclude determines if the settings.php or equivalent includes settings.ddev.php or equivalent.
// This is done by looking for the ddev settings file (settings.ddev.php) in settings.php.
func settingsHasInclude(drupalConfig *DrupalSettings, siteSettingsPath string) (bool, error) {
	included, err := fileutil.FgrepStringInFile(siteSettingsPath, drupalConfig.SiteSettingsLocal)
	if err != nil {
		return false, err
	}

	return included, nil
}

// appendIncludeToDrupalSettingsFile modifies the settings.php file to include the settings.ddev.php
// file, which contains ddev-specific configuration.
func appendIncludeToDrupalSettingsFile(drupalConfig *DrupalSettings, siteSettingsPath string) error {
	// Check if file is empty
	contents, err := ioutil.ReadFile(siteSettingsPath)
	if err != nil {
		return err
	}

	// If the file is empty, write the complete settings template and return
	if len(contents) == 0 {
		return writeDrupalCommonSettingsFile(drupalConfig, siteSettingsPath)
	}

	// The file is not empty, open it for appending
	file, err := os.OpenFile(siteSettingsPath, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer util.CheckClose(file)

	tmpl, err := template.New("settings").Funcs(getTemplateFuncMap()).Parse(drupalCommonSettingsAppendTemplate)
	if err != nil {
		return err
	}

	// Write the template to the file
	if err := tmpl.Execute(file, drupalConfig); err != nil {
		return err
	}

	return nil
}
