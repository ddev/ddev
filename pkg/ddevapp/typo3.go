package ddevapp

import (
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/drud/ddev/pkg/appports"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"os"
	"path/filepath"
)

// Typo3Settings encapsulates all the configurations for a typo3 site.
type Typo3Settings struct {
	DeployName       string
	DeployURL        string
	DatabaseName     string
	DatabaseUsername string
	DatabasePassword string
	DatabaseHost     string
	DatabaseDriver   string
	DatabasePort     string
	DatabasePrefix   string
	Signature        string
}

// NewTypo3Settings produces a Typo3Settings object with default.
func NewTypo3Settings() *Typo3Settings {
	return &Typo3Settings{
		DatabaseName:     "db",
		DatabaseUsername: "db",
		DatabasePassword: "db",
		DatabaseHost:     "db",
		DatabaseDriver:   "mysql",
		DatabasePort:     appports.GetPort("db"),
		DatabasePrefix:   "",
		Signature:        DdevFileSignature,
	}
}

const typo3AdditionalConfigTemplate = `<?php
{{ $config := . }}
/**
 {{ $config.Signature }}: Automatically generated Typo3 AdditionalConfiguration.php file.
 ddev manages this file and may delete or overwrite the file unless this comment is removed.
 */

$GLOBALS['TYPO3_CONF_VARS']['DB'] = array_merge($GLOBALS['TYPO3_CONF_VARS']['DB'], [
                    'database' => 'db',
                    'host' => 'db',
                    'password' => 'db',
                    'user' => 'db',
]);`

// createTypo3SettingsFile creates the app's LocalConfiguration.php and
// AdditionalConfiguration.php, adding things like database host, name, and
// password. Returns the fullpath to settings file and error
func createTypo3SettingsFile(app *DdevApp) (string, error) {

	if !fileutil.FileExists(app.SiteSettingsPath) {
		util.Failed("Typo3 does not seem to have been set up yet, missing LocalConfiguration.php (%s)", app.SiteLocalSettingsPath)
	}

	settingsFilePath, err := app.DetermineSettingsPathLocation()
	if err != nil {
		return "", fmt.Errorf("Failed to get Typo3 AdditionalConfiguration.php file path: %v", err)
	}
	output.UserOut.Printf("Generating %s file for database connection.", filepath.Base(settingsFilePath))

	// Currently there isn't any customization done for the typo3 config, but
	// we may want to do some kind of customization in the future.
	typo3Config := NewTypo3Settings()

	err = writeTypo3SettingsFile(app, typo3Config)
	if err != nil {
		return settingsFilePath, fmt.Errorf("Failed to write Typo3 settings file: %v", err)
	}

	return settingsFilePath, nil
}

// writeTypo3SettingsFile produces AdditionalSettings.php file
// It's assumed that the LocalConfiguration.php must already exist, and we're
// overriding the db config values in it.
func writeTypo3SettingsFile(app *DdevApp, settings *Typo3Settings) error {

	filePath := app.SiteLocalSettingsPath
	tmpl, err := template.New("settings").Funcs(sprig.TxtFuncMap()).Parse(typo3AdditionalConfigTemplate)
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

// getTypo3UploadDir just returns a static upload files (public files) dir.
// This can be made more sophisticated in the future, for example by adding
// the directory to the ddev config.yaml.
func getTypo3UploadDir(app *DdevApp) string {
	// @todo: Check to see if this can be overridden in LocalConfiguration.php
	return "uploads"
}

// Typo3Hooks adds a typo3-specific hooks example for post-import-db
const Typo3Hooks = `
#     - exec: "hostname"`

// getTypo3Hooks for appending as byte array
func getTypo3Hooks() []byte {
	// We don't have anything new to add yet, so just use Typo37 version
	return []byte(Typo3Hooks)
}

// setTypo3SiteSettingsPaths sets the paths to settings.php/settings.local.php
// for templating.
func setTypo3SiteSettingsPaths(app *DdevApp) {
	settingsFileBasePath := filepath.Join(app.AppRoot, app.Docroot)
	var settingsFilePath, localSettingsFilePath string
	settingsFilePath = filepath.Join(settingsFileBasePath, "typo3conf", "LocalConfiguration.php")
	localSettingsFilePath = filepath.Join(settingsFileBasePath, "typo3conf", "AdditionalConfiguration.php")
	app.SiteSettingsPath = settingsFilePath
	app.SiteLocalSettingsPath = localSettingsFilePath
}

// isTypoApp returns true if the app is of type typo3
func isTypo3App(app *DdevApp) bool {
	if _, err := os.Stat(filepath.Join(app.AppRoot, app.Docroot, "typo3")); err == nil {
		return true
	}
	return false
}
