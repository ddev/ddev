package ddevapp

import (
	"fmt"
	"io/ioutil"

	"github.com/Masterminds/sprig"
	"github.com/drud/ddev/pkg/appports"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"

	"os"
	"path/filepath"
	"text/template"
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
	IsDrupal8        bool
	Signature        string
}

// NewDrupalSettings produces a DrupalSettings object with default.
func NewDrupalSettings() *DrupalSettings {
	return &DrupalSettings{
		DatabaseName:     "db",
		DatabaseUsername: "db",
		DatabasePassword: "db",
		DatabaseHost:     "db",
		DatabaseDriver:   "mysql",
		DatabasePort:     appports.GetPort("db"),
		DatabasePrefix:   "",
		IsDrupal8:        false,
		HashSalt:         util.RandString(64),
		Signature:        DdevFileSignature,
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

const (
	drupal6Template = `<?php
{{ $config := . }}
/**
 {{ $config.Signature }}: Automatically generated Drupal settings.php file.
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

// createDrupalSettingsFile creates the app's settings.php or equivalent,
// adding things like database host, name, and password
// Returns the fullpath to settings file and err
func createDrupalSettingsFile(app *DdevApp) (string, error) {

	settingsFilePath, err := app.DetermineSettingsPathLocation()
	if err != nil {
		return "", fmt.Errorf("Failed to get Drupal settings file path: %v", err)
	}
	output.UserOut.Printf("Generating %s file for database connection.", filepath.Base(settingsFilePath))

	// Currently there isn't any customization done for the drupal config, but
	// we may want to do some kind of customization in the future.
	drupalConfig := NewDrupalSettings()

	err = writeDrupalSettingsFile(drupalConfig, settingsFilePath)
	if err != nil {
		return settingsFilePath, fmt.Errorf("Failed to write Drupal settings file: %v", err)
	}

	return settingsFilePath, nil
}

// createDrupal6SettingsFile creates the app's settings.php or equivalent,
// adding things like database host, name, and password
// Returns the fullpath to settings file and err
func createDrupal6SettingsFile(app *DdevApp) (string, error) {

	settingsFilePath, err := app.DetermineSettingsPathLocation()
	if err != nil {
		return "", fmt.Errorf("Failed to get Drupal settings file path: %v", err)
	}
	output.UserOut.Printf("Generating %s file for database connection.", filepath.Base(settingsFilePath))

	// Currently there isn't any customization done for the drupal config, but
	// we may want to do some kind of customization in the future.
	drupalConfig := NewDrupalSettings()

	err = writeDrupal6SettingsFile(drupalConfig, settingsFilePath)
	if err != nil {
		return settingsFilePath, fmt.Errorf("Failed to write Drupal settings file: %v", err)
	}

	return settingsFilePath, nil
}

// writeDrupalSettingsFile dynamically produces valid settings.php file by combining a configuration
// object with a data-driven template.
func writeDrupalSettingsFile(settings *DrupalSettings, filePath string) error {
	tmpl, err := template.New("settings").Funcs(sprig.TxtFuncMap()).Parse(drupalTemplate)
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

// writeDrupal6SettingsFile dynamically produces valid settings.php file by combining a configuration
// object with a data-driven template.
func writeDrupal6SettingsFile(settings *DrupalSettings, filePath string) error {
	tmpl, err := template.New("settings").Funcs(sprig.TxtFuncMap()).Parse(drupal6Template)
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

// getDrupalUploadDir just returns a static upload files (public files) dir.
// This can be made more sophisticated in the future, for example by adding
// the directory to the ddev config.yaml.
func getDrupalUploadDir(app *DdevApp) string {
	return "sites/default/files"
}

// Drupal8Hooks adds a d8-specific hooks example for post-import-db
const Drupal8Hooks = `
#     - exec: "drush cr"`

// Drupal7Hooks adds a d7-specific hooks example for post-import-db
const Drupal7Hooks = `
#     - exec: "drush cc all"`

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
	settingsFileBasePath := filepath.Join(app.AppRoot, app.Docroot)
	var settingsFilePath, localSettingsFilePath string
	settingsFilePath = filepath.Join(settingsFileBasePath, "sites", "default", "settings.php")
	localSettingsFilePath = filepath.Join(settingsFileBasePath, "sites", "default", "settings.local.php")
	app.SiteSettingsPath = settingsFilePath
	app.SiteLocalSettingsPath = localSettingsFilePath
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

// dtNginxConfig provides an nginx config override for Drupal 6
const d6NginxConfig = DdevFileSignature + `

# Set https to 'on' if x-forwarded-proto is https
map $http_x_forwarded_proto $fcgi_https {
    default off;
    https on;
}

server {
    listen 80; ## listen for ipv4; this line is default and implied
    listen [::]:80 default ipv6only=on; ## listen for ipv6
    # The NGINX_DOCROOT variable is substituted with
    # its value when the container is started.
    root $NGINX_DOCROOT;
    index index.php index.htm index.html;

    # Make site accessible from http://localhost/
    server_name _;

    # Disable sendfile as per https://docs.vagrantup.com/v2/synced-folders/virtualbox.html
    sendfile off;
    error_log /var/log/nginx/error.log info;
    access_log /var/log/nginx/error.log;

    location / {
        # First attempt to serve request as file, then
        # as directory, then fall back to index.html
        try_files $uri @rewrite;

    }

    location @rewrite {
        rewrite ^/(.*)$ /index.php?q=$1;
    }

    # Handle image styles for Drupal 7+
    location ~ ^/sites/.*/files/styles/ {
        try_files $uri @rewrite;
    }

    # pass the PHP scripts to FastCGI server listening on socket
    location ~ \.php$ {
        try_files $uri =404;
        fastcgi_split_path_info ^(.+\.php)(/.+)$;
        fastcgi_pass unix:/run/php-fpm.sock;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        fastcgi_param SCRIPT_NAME $fastcgi_script_name;
        fastcgi_index index.php;
        include fastcgi_params;
        fastcgi_intercept_errors on;
        # fastcgi_read_timeout should match max_execution_time in php.ini
        fastcgi_read_timeout 240;
        fastcgi_param SERVER_NAME $host;
        fastcgi_param HTTPS $fcgi_https;
    }

    # Expire rules for static content
    # Feed
    location ~* \.(?:rss|atom)$ {
        expires 1h;
    }

    # Media: images, icons, video, audio, HTC
    location ~* \.(?:jpg|jpeg|gif|png|ico|cur|gz|svg|svgz|mp4|ogg|ogv|webm|htc)$ {
        expires 1M;
        access_log off;
        add_header Cache-Control "public";
    }

    # Prevent clients from accessing hidden files (starting with a dot)
    # This is particularly important if you store .htpasswd files in the site hierarchy
    # Access to /.well-known/ is allowed.
    # https://www.mnot.net/blog/2010/04/07/well-known
    # https://tools.ietf.org/html/rfc5785
    location ~* /\.(?!well-known\/) {
        deny all;
    }

    # Prevent clients from accessing to backup/config/source files
    location ~* (?:\.(?:bak|conf|dist|fla|in[ci]|log|psd|sh|sql|sw[op])|~)$ {
        deny all;
    }

    ## Regular private file serving (i.e. handled by Drupal).
    location ^~ /system/files/ {
        ## For not signaling a 404 in the error log whenever the
        ## system/files directory is accessed add the line below.
        ## Note that the 404 is the intended behavior.
        log_not_found off;
        access_log off;
        expires 30d;
        try_files $uri @rewrite;
    }

    ## provide a health check endpoint
    location /healthcheck {
        access_log off;
        return 200;
    }

    error_page 400 401 /40x.html;
    location = /40x.html {
            root   /usr/share/nginx/html;
    }

}
`

// drupal6PostConfigAction allows us to override the nginx config for d6, since
// the default is oriented around d7/d8.
func drupal6PostConfigAction(app *DdevApp) error {
	// Write a nginx-site.conf
	filePath := app.GetConfigPath("nginx-site.conf")

	if fileutil.FileExists(filePath) {
		signatureFound, err := fileutil.FgrepStringInFile(filePath, DdevFileSignature)
		util.CheckErr(err) // Really can't happen as we already checked for the file existence

		if !signatureFound {
			util.Warning("A custom non-ddev-generated %s already exists, so not creating a Drupal 6 version of it.", filePath)
			return nil
		}
	}

	// Replace or create nginx-site.conf
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filePath, []byte(d6NginxConfig), 0644)
	if err != nil {
		return err
	}
	util.CheckClose(file)
	return nil
}
