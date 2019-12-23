package ddevapp

import (
	"fmt"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/nodeps"

	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"

	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/drud/ddev/pkg/fileutil"

	"github.com/drud/ddev/pkg/archive"
)

// DrupalSettings encapsulates all the configurations for a Drupal site.
type DrupalSettings struct {
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
	Signature        string
	SitePath         string
	SiteSettings     string
	SiteSettingsDdev string
	SyncDir          string
	DockerIP         string
	DBPublishedPort  int
}

// NewDrupalSettings produces a DrupalSettings object with default.
func NewDrupalSettings(app *DdevApp) *DrupalSettings {
	dockerIP, _ := dockerutil.GetDockerIP()
	dbPublishedPort, _ := app.GetPublishedPort("db")

	return &DrupalSettings{
		DatabaseName:     "db",
		DatabaseUsername: "db",
		DatabasePassword: "db",
		DatabaseHost:     "db",
		DatabaseDriver:   "mysql",
		DatabasePort:     GetPort("db"),
		DatabasePrefix:   "",
		HashSalt:         util.RandString(64),
		Signature:        DdevFileSignature,
		SitePath:         path.Join("sites", "default"),
		SiteSettings:     "settings.php",
		SiteSettingsDdev: "settings.ddev.php",
		SyncDir:          path.Join("files", "sync"),
		DockerIP:         dockerIP,
		DBPublishedPort:  dbPublishedPort,
	}
}

// drupal8SettingsTemplate defines the template that will become a Drupal 8 app's settings.php
// in the event that one does not already exist.
const drupal8SettingsTemplate = `<?php
{{ $config := . }}
// {{ $config.Signature }}: Automatically generated Drupal settings file.
if (file_exists($app_root . '/' . $site_path . '/{{ $config.SiteSettingsDdev }}')) {
  include $app_root . '/' . $site_path . '/{{ $config.SiteSettingsDdev }}';
}
`

// drupal8SettingsAppendTemplate defines the template that will be appended to
// a Drupal 8 app's settings.php in the event that one exists.
const drupal8SettingsAppendTemplate = `{{ $config := . }}
// Automatically generated include for settings managed by ddev.
if (file_exists($app_root . '/' . $site_path . '/{{ $config.SiteSettingsDdev }}')) {
  include $app_root . '/' . $site_path . '/{{ $config.SiteSettingsDdev }}';
}
`

// drupal7SettingsTemplate defines the template that will become a Drupal 7
// app's settings.php in the event that one does not already exist.
const drupal7SettingsTemplate = `<?php
{{ $config := . }}
// {{ $config.Signature }}: Automatically generated Drupal settings file.
$ddev_settings = dirname(__FILE__) . '/{{ $config.SiteSettingsDdev }}';
if (is_readable($ddev_settings)) {
  require $ddev_settings;
}
`

// drupal7SettingsAppendTemplate defines the template that will be appended to
// a Drupal 7 app's settings.php in the event that one exists.
const drupal7SettingsAppendTemplate = `{{ $config := . }}
// Automatically generated include for settings managed by ddev.
$ddev_settings = dirname(__FILE__) . '/{{ $config.SiteSettingsDdev }}';
if (is_readable($ddev_settings)) {
  require $ddev_settings;
}
`

// drupal6SettingsTemplate defines the template that will become a Drupal 6
// app's settings.php in the event that one does not already exist.
const drupal6SettingsTemplate = drupal7SettingsTemplate

// drupal7SettingsAppendTemplate defines the template that will be appended to
// a Drupal 7 app's settings.php in the event that one exists.
const drupal6SettingsAppendTemplate = drupal7SettingsAppendTemplate

const (
	drupal8DdevSettingsTemplate = `<?php
{{ $config := . }}
/**
 * @file
 * {{ $config.Signature }}: Automatically generated Drupal settings file.
 * ddev manages this file and may delete or overwrite the file unless this
 * comment is removed.
 */

$host = "{{ $config.DatabaseHost }}";
$port = {{ $config.DatabasePort }};

// If DDEV_PHP_VERSION is not set, it means we're running on the host,
// so use the host-side bind port on docker IP
if (empty(getenv('DDEV_PHP_VERSION'))) {
  $host = "{{ $config.DockerIP }}";
  $port = {{ $config.DBPublishedPort }};
} 

$databases['default']['default'] = array(
  'database' => "{{ $config.DatabaseName }}",
  'username' => "{{ $config.DatabaseUsername }}",
  'password' => "{{ $config.DatabasePassword }}",
  'host' => $host,
  'driver' => "{{ $config.DatabaseDriver }}",
  'port' => $port,
  'prefix' => "{{ $config.DatabasePrefix }}",
);

ini_set('session.gc_probability', 1);
ini_set('session.gc_divisor', 100);
ini_set('session.gc_maxlifetime', 200000);
ini_set('session.cookie_lifetime', 2000000);

$settings['hash_salt'] = '{{ $config.HashSalt }}';

// This will prevent Drupal from setting read-only permissions on sites/default.
$settings['skip_permissions_hardening'] = TRUE;

// This will ensure the site can only be accessed through the intended host
// names. Additional host patterns can be added for custom configurations.
$settings['trusted_host_patterns'] = ['.*'];

// Don't use Symfony's APCLoader. ddev includes APCu; Composer's APCu loader has
// better performance.
$settings['class_loader_auto_detect'] = FALSE;

// This specifies the default configuration sync directory.
if (empty($config_directories[CONFIG_SYNC_DIRECTORY])) {
  $config_directories[CONFIG_SYNC_DIRECTORY] = '{{ joinPath $config.SitePath $config.SyncDir }}';
}


`
)

const (
	drupal7DdevSettingsTemplate = `<?php
{{ $config := . }}
/**
 * @file
 * {{ $config.Signature }}: Automatically generated Drupal settings file.
 * ddev manages this file and may delete or overwrite the file unless this
 * comment is removed.
 */

$host = "{{ $config.DatabaseHost }}";
$port = {{ $config.DatabasePort }};

// If DDEV_PHP_VERSION is not set, it means we're running on the host,
// so use the host-side bind port on docker IP
if (empty(getenv('DDEV_PHP_VERSION'))) {
  $host = "{{ $config.DockerIP }}";
  $port = {{ $config.DBPublishedPort }};
} 

$databases['default']['default'] = array(
  'database' => "{{ $config.DatabaseName }}",
  'username' => "{{ $config.DatabaseUsername }}",
  'password' => "{{ $config.DatabasePassword }}",
  'host' => $host,
  'driver' => "{{ $config.DatabaseDriver }}",
  'port' => $port,
  'prefix' => "{{ $config.DatabasePrefix }}",
);

ini_set('session.gc_probability', 1);
ini_set('session.gc_divisor', 100);
ini_set('session.gc_maxlifetime', 200000);
ini_set('session.cookie_lifetime', 2000000);

$drupal_hash_salt = '{{ $config.HashSalt }}';


`
)

const (
	drupal6DdevSettingsTemplate = `<?php
{{ $config := . }}
/**
 * @file
 * {{ $config.Signature }}: Automatically generated Drupal settings file.
 * ddev manages this file and may delete or overwrite the file unless this
 * comment is removed.
 */
$host = "{{ $config.DatabaseHost }}";
$port = {{ $config.DatabasePort }};

// If DDEV_PHP_VERSION is not set, it means we're running on the host,
// so use the host-side bind port on docker IP
if (empty(getenv('DDEV_PHP_VERSION'))) {
  $host = "{{ $config.DockerIP }}";
  $port = {{ $config.DBPublishedPort }};
} 

$db_url = "{{ $config.DatabaseDriver }}://{{ $config.DatabaseUsername }}:{{ $config.DatabasePassword }}@$host:$port/{{ $config.DatabaseName }}";

ini_set('session.gc_probability', 1);
ini_set('session.gc_divisor', 100);
ini_set('session.gc_maxlifetime', 200000);
ini_set('session.cookie_lifetime', 2000000);
`
)

// manageDrupalSettingsFile will direct inspecting and writing of settings.php.
func manageDrupalSettingsFile(app *DdevApp, drupalConfig *DrupalSettings, settingsTemplate, appendTemplate string) error {
	// We'll be writing/appending to the settings files and parent directory, make sure we have permissions to do so
	if err := drupalEnsureWritePerms(app); err != nil {
		return err
	}

	if !fileutil.FileExists(app.SiteSettingsPath) {
		output.UserOut.Printf("No %s file exists, creating one", drupalConfig.SiteSettings)

		if err := writeDrupalSettingsFile(drupalConfig, app.SiteSettingsPath, settingsTemplate); err != nil {
			return fmt.Errorf("failed to write: %v", err)
		}
	}

	included, err := settingsHasInclude(drupalConfig, app.SiteSettingsPath)
	if err != nil {
		return fmt.Errorf("failed to check for include: %v", err)
	}

	if included {
		output.UserOut.Printf("Existing %s file includes %s", drupalConfig.SiteSettings, drupalConfig.SiteSettingsDdev)
	} else {
		output.UserOut.Printf("Existing %s file does not include %s, modifying to include ddev settings", drupalConfig.SiteSettings, drupalConfig.SiteSettingsDdev)

		if err := appendIncludeToDrupalSettingsFile(drupalConfig, app.SiteSettingsPath, appendTemplate); err != nil {
			return fmt.Errorf("failed to include %s in %s: %v", drupalConfig.SiteSettingsDdev, drupalConfig.SiteSettings, err)
		}
	}

	return nil
}

// writeDrupalSettingsFile creates the app's settings.php or equivalent,
// which does nothing more than import the ddev-managed settings.ddev.php.
func writeDrupalSettingsFile(drupalConfig *DrupalSettings, filePath string, versionTemplate string) error {
	tmpl, err := template.New("settings").Funcs(getTemplateFuncMap()).Parse(versionTemplate)
	if err != nil {
		return err
	}

	// Ensure target directory exists and is writable
	dir := filepath.Dir(filePath)
	if err = os.Chmod(dir, 0755); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	} else if err != nil {
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
	drupalConfig := NewDrupalSettings(app)

	if err := manageDrupalSettingsFile(app, drupalConfig, drupal7SettingsTemplate, drupal7SettingsAppendTemplate); err != nil {
		return "", err
	}

	if err := writeDrupal7DdevSettingsFile(drupalConfig, app.SiteDdevSettingsFile); err != nil {
		return "", fmt.Errorf("`failed to write` Drupal settings file %s: %v", app.SiteDdevSettingsFile, err)
	}

	return app.SiteDdevSettingsFile, nil
}

// createDrupal8SettingsFile manages creation and modification of settings.php and settings.ddev.php.
// If a settings.php file already exists, it will be modified to ensure that it includes
// settings.ddev.php, which contains ddev-specific configuration.
func createDrupal8SettingsFile(app *DdevApp) (string, error) {
	// Currently there isn't any customization done for the drupal config, but
	// we may want to do some kind of customization in the future.
	drupalConfig := NewDrupalSettings(app)

	if err := manageDrupalSettingsFile(app, drupalConfig, drupal8SettingsTemplate, drupal8SettingsAppendTemplate); err != nil {
		return "", err
	}

	if err := writeDrupal8DdevSettingsFile(drupalConfig, app.SiteDdevSettingsFile); err != nil {
		return "", fmt.Errorf("failed to write Drupal settings file %s: %v", app.SiteDdevSettingsFile, err)
	}

	return app.SiteDdevSettingsFile, nil
}

// createDrupal6SettingsFile manages creation and modification of settings.php and settings.ddev.php.
// If a settings.php file already exists, it will be modified to ensure that it includes
// settings.ddev.php, which contains ddev-specific configuration.
func createDrupal6SettingsFile(app *DdevApp) (string, error) {
	// Currently there isn't any customization done for the drupal config, but
	// we may want to do some kind of customization in the future.
	drupalConfig := NewDrupalSettings(app)
	// mysqli is required in latest D6LTS and works fine in ddev in old D6
	drupalConfig.DatabaseDriver = "mysqli"

	if err := manageDrupalSettingsFile(app, drupalConfig, drupal6SettingsTemplate, drupal6SettingsAppendTemplate); err != nil {
		return "", err
	}

	if err := writeDrupal6DdevSettingsFile(drupalConfig, app.SiteDdevSettingsFile); err != nil {
		return "", fmt.Errorf("failed to write Drupal settings file %s: %v", app.SiteDdevSettingsFile, err)
	}

	return app.SiteDdevSettingsFile, nil
}

// writeDrupal8DdevSettingsFile dynamically produces valid settings.ddev.php file by combining a configuration
// object with a data-driven template.
func writeDrupal8DdevSettingsFile(settings *DrupalSettings, filePath string) error {
	if fileutil.FileExists(filePath) {
		// Check if the file is managed by ddev.
		signatureFound, err := fileutil.FgrepStringInFile(filePath, DdevFileSignature)
		if err != nil {
			return err
		}

		// If the signature wasn't found, warn the user and return.
		if !signatureFound {
			util.Warning("%s already exists and is managed by the user.", filepath.Base(filePath))
			return nil
		}
	}

	tmpl, err := template.New("settings").Funcs(getTemplateFuncMap()).Parse(drupal8DdevSettingsTemplate)
	if err != nil {
		return err
	}

	// Ensure target directory exists and is writable
	dir := filepath.Dir(filePath)
	if err = os.Chmod(dir, 0755); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	} else if err != nil {
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

// writeDrupal7DdevSettingsFile dynamically produces valid settings.ddev.php file by combining a configuration
// object with a data-driven template.
func writeDrupal7DdevSettingsFile(settings *DrupalSettings, filePath string) error {
	if fileutil.FileExists(filePath) {
		// Check if the file is managed by ddev.
		signatureFound, err := fileutil.FgrepStringInFile(filePath, DdevFileSignature)
		if err != nil {
			return err
		}

		// If the signature wasn't found, warn the user and return.
		if !signatureFound {
			util.Warning("%s already exists and is managed by the user.", filepath.Base(filePath))
			return nil
		}
	}

	tmpl, err := template.New("settings").Funcs(getTemplateFuncMap()).Parse(drupal7DdevSettingsTemplate)
	if err != nil {
		return err
	}

	// Ensure target directory exists and is writable
	dir := filepath.Dir(filePath)
	if err = os.Chmod(dir, 0755); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	} else if err != nil {
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

// writeDrupal6DdevSettingsFile dynamically produces valid settings.ddev.php file by combining a configuration
// object with a data-driven template.
func writeDrupal6DdevSettingsFile(settings *DrupalSettings, filePath string) error {
	if fileutil.FileExists(filePath) {
		// Check if the file is managed by ddev.
		signatureFound, err := fileutil.FgrepStringInFile(filePath, DdevFileSignature)
		if err != nil {
			return err
		}

		// If the signature wasn't found, warn the user and return.
		if !signatureFound {
			util.Warning("%s already exists and is managed by the user.", filepath.Base(filePath))
			return nil
		}
	}
	tmpl, err := template.New("settings").Funcs(getTemplateFuncMap()).Parse(drupal6DdevSettingsTemplate)
	if err != nil {
		return err
	}

	// Ensure target directory exists and is writable
	dir := filepath.Dir(filePath)
	if err = os.Chmod(dir, 0755); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	} else if err != nil {
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

// WriteDrushrc writes out drushrc.php based on passed-in values.
// This works on Drupal 6 and Drupal 7 or with drush8 and older
func WriteDrushrc(app *DdevApp, filePath string) error {
	if fileutil.FileExists(filePath) {
		// Check if the file is managed by ddev.
		signatureFound, err := fileutil.FgrepStringInFile(filePath, DdevFileSignature)
		if err != nil {
			return err
		}

		// If the signature wasn't found, warn the user and return.
		if !signatureFound {
			util.Warning("%s already exists and is managed by the user.", filepath.Base(filePath))
			return nil
		}
	}

	uri := app.GetHTTPSURL()
	if GetCAROOT() == "" {
		uri = app.GetHTTPURL()
	}
	drushContents := []byte(`<?php
/** ` + DdevFileSignature + `: Automatically generated drushrc.php file.
 ddev manages this file and may delete or overwrite the file unless this comment is removed.
 */
$options['l'] = "` + uri + `";
`)

	// Ensure target directory exists and is writable
	dir := filepath.Dir(filePath)
	if err := os.Chmod(dir, 0755); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	err := ioutil.WriteFile(filePath, drushContents, 0666)
	if err != nil {
		return err
	}

	return nil
}

// WriteDrushYML writes a drush.yaml to set the default uri
func WriteDrushYML(app *DdevApp, filePath string) error {
	if fileutil.FileExists(filePath) {

		// Check if the file is managed by ddev.
		signatureFound, err := fileutil.FgrepStringInFile(filePath, DdevFileSignature)
		if err != nil {
			return err
		}

		// If the signature wasn't found, warn the user and return.
		if !signatureFound {
			util.Warning("%s already exists and is managed by the user.", filepath.Base(filePath))
			return nil
		}
	}

	uri := app.GetHTTPSURL()
	if GetCAROOT() == "" {
		uri = app.GetHTTPURL()
	}
	drushContents := []byte(`
#` + DdevFileSignature + `: Automatically generated drush.yml file.
# ddev manages this file and may delete or overwrite the file unless this comment is removed.

options:
  uri: "` + uri + `"
`)

	// Ensure target directory exists and is writable
	dir := filepath.Dir(filePath)
	if err := os.Chmod(dir, 0755); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	err := ioutil.WriteFile(filePath, drushContents, 0666)
	if err != nil {
		return err
	}

	return nil
}

// getDrupalUploadDir will return a custom upload dir if defined, returning a default path if not.
func getDrupalUploadDir(app *DdevApp) string {
	if app.UploadDir == "" {
		return "sites/default/files"
	}

	return app.UploadDir
}

// Drupal8Hooks adds a d8-specific hooks example for post-import-db
const Drupal8Hooks = `# post-import-db:
#   - exec: drush cr
#   - exec: drush updb
`

// Drupal7Hooks adds a d7-specific hooks example for post-import-db
const Drupal7Hooks = `#  post-import-db:
#    - exec: drush cc all
`

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

// setDrupalSiteSettingsPaths sets the paths to settings.php/settings.ddev.php
// for templating.
func setDrupalSiteSettingsPaths(app *DdevApp) {
	drupalConfig := NewDrupalSettings(app)
	settingsFileBasePath := filepath.Join(app.AppRoot, app.Docroot)
	app.SiteSettingsPath = filepath.Join(settingsFileBasePath, drupalConfig.SitePath, drupalConfig.SiteSettings)
	app.SiteDdevSettingsFile = filepath.Join(settingsFileBasePath, drupalConfig.SitePath, drupalConfig.SiteSettingsDdev)
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

// drupal6ConfigOverrideAction overrides php_version for D6, since it is incompatible
// with php7+
func drupal6ConfigOverrideAction(app *DdevApp) error {
	app.PHPVersion = nodeps.PHP56
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

	// Write both drush.yml and drushrc.php for Drupal 8, because we can't know
	// what version of drush may be in use. drush8 is happy with drushrc.php
	// drush9 wants drush.yml
	err := WriteDrushYML(app, filepath.Join(filepath.Dir(app.SiteSettingsPath), "..", "all", "drush", "drush.yml"))
	if err != nil {
		util.Warning("Failed to WriteDrushYML: %v", err)
	}

	err = WriteDrushrc(app, filepath.Join(filepath.Dir(app.SiteSettingsPath), "drushrc.php"))
	if err != nil {
		util.Warning("Failed to WriteDrushrc: %v", err)
	}

	if _, err = app.CreateSettingsFile(); err != nil {
		return fmt.Errorf("failed to write settings file %s: %v", app.SiteDdevSettingsFile, err)
	}
	return nil
}

// drupal7PostStartAction handles default post-start actions for D7 apps, like ensuring
// useful permissions settings on sites/default.
func drupal7PostStartAction(app *DdevApp) error {
	if err := drupalEnsureWritePerms(app); err != nil {
		return err
	}

	err := WriteDrushrc(app, filepath.Join(filepath.Dir(app.SiteSettingsPath), "drushrc.php"))
	if err != nil {
		util.Warning("Failed to WriteDrushrc: %v", err)
	}

	if _, err = app.CreateSettingsFile(); err != nil {
		return fmt.Errorf("failed to write settings file %s: %v", app.SiteDdevSettingsFile, err)
	}
	return nil
}

// drupal6PostStartAction handles default post-start actions for D6 apps, like ensuring
// useful permissions settings on sites/default.
func drupal6PostStartAction(app *DdevApp) error {
	if err := drupalEnsureWritePerms(app); err != nil {
		return err
	}

	err := WriteDrushrc(app, filepath.Join(filepath.Dir(app.SiteSettingsPath), "drushrc.php"))
	if err != nil {
		util.Warning("Failed to WriteDrushrc: %v", err)
	}
	if _, err = app.CreateSettingsFile(); err != nil {
		return fmt.Errorf("failed to write settings file %s: %v", app.SiteDdevSettingsFile, err)
	}
	return nil
}

// drupalEnsureWritePerms will ensure sites/default and sites/default/settings.php will
// have the appropriate permissions for development.
func drupalEnsureWritePerms(app *DdevApp) error {
	output.UserOut.Printf("Ensuring write permissions for %s", app.GetName())
	var writePerms os.FileMode = 0200

	settingsDir := path.Dir(app.SiteSettingsPath)
	makeWritable := []string{
		settingsDir,
		app.SiteSettingsPath,
		app.SiteDdevSettingsFile,
		path.Join(settingsDir, "services.yml"),
	}

	for _, o := range makeWritable {
		stat, err := os.Stat(o)
		if err != nil {
			if !os.IsNotExist(err) {
				util.Warning("Unable to ensure write permissions: %v", err)
			}

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
	drupalConfig := NewDrupalSettings(app)

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
	included, err := fileutil.FgrepStringInFile(siteSettingsPath, drupalConfig.SiteSettingsDdev)
	if err != nil {
		return false, err
	}

	return included, nil
}

// appendIncludeToDrupalSettingsFile modifies the settings.php file to include the settings.ddev.php
// file, which contains ddev-specific configuration.
func appendIncludeToDrupalSettingsFile(drupalConfig *DrupalSettings, siteSettingsPath string, appendTemplate string) error {
	// Check if file is empty
	contents, err := ioutil.ReadFile(siteSettingsPath)
	if err != nil {
		return err
	}

	// If the file is empty, write the complete settings template and return
	if len(contents) == 0 {
		return writeDrupalSettingsFile(drupalConfig, siteSettingsPath, appendTemplate)
	}

	// The file is not empty, open it for appending
	file, err := os.OpenFile(siteSettingsPath, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer util.CheckClose(file)

	tmpl, err := template.New("settings").Funcs(getTemplateFuncMap()).Parse(appendTemplate)
	if err != nil {
		return err
	}

	// Write the template to the file
	if err := tmpl.Execute(file, drupalConfig); err != nil {
		return err
	}

	return nil
}

// drupalImportFilesAction defines the Drupal workflow for importing project files.
func drupalImportFilesAction(app *DdevApp, importPath, extPath string) error {
	destPath := filepath.Join(app.GetAppRoot(), app.GetDocroot(), app.GetUploadDir())

	// parent of destination dir should exist
	if !fileutil.FileExists(filepath.Dir(destPath)) {
		return fmt.Errorf("unable to import to %s: parent directory does not exist", destPath)
	}

	// parent of destination dir should be writable.
	if err := os.Chmod(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// If the destination path exists, remove it as was warned
	if fileutil.FileExists(destPath) {
		if err := os.RemoveAll(destPath); err != nil {
			return fmt.Errorf("failed to cleanup %s before import: %v", destPath, err)
		}
	}

	if isTar(importPath) {
		if err := archive.Untar(importPath, destPath, extPath); err != nil {
			return fmt.Errorf("failed to extract provided archive: %v", err)
		}

		return nil
	}

	if isZip(importPath) {
		if err := archive.Unzip(importPath, destPath, extPath); err != nil {
			return fmt.Errorf("failed to extract provided archive: %v", err)
		}

		return nil
	}

	if err := fileutil.CopyDir(importPath, destPath); err != nil {
		return err
	}

	return nil
}
