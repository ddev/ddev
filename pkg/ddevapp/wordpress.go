package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
)

// WordpressConfig encapsulates all the configurations for a WordPress site.
type WordpressConfig struct {
	WPGeneric        bool
	DeployName       string
	DeployURL        string
	DatabaseName     string
	DatabaseUsername string
	DatabasePassword string
	DatabaseHost     string
	AuthKey          string
	SecureAuthKey    string
	LoggedInKey      string
	NonceKey         string
	AuthSalt         string
	SecureAuthSalt   string
	LoggedInSalt     string
	NonceSalt        string
	Docroot          string
	TablePrefix      string
	Signature        string
	SiteSettings     string
	SiteSettingsDdev string
	AbsPath          string
	DbCharset        string
	DbCollate        string
}

// NewWordpressConfig produces a WordpressConfig object with defaults.
func NewWordpressConfig(app *DdevApp, absPath string) *WordpressConfig {
	return &WordpressConfig{
		WPGeneric:        false,
		DatabaseName:     "db",
		DatabaseUsername: "db",
		DatabasePassword: "db",
		DatabaseHost:     "ddev-" + app.Name + "-db",
		DeployURL:        app.GetPrimaryURL(),
		Docroot:          "/var/www/html/docroot",
		TablePrefix:      "wp_",
		AuthKey:          util.RandString(64),
		AuthSalt:         util.RandString(64),
		LoggedInKey:      util.RandString(64),
		LoggedInSalt:     util.RandString(64),
		NonceKey:         util.RandString(64),
		NonceSalt:        util.RandString(64),
		SecureAuthKey:    util.RandString(64),
		SecureAuthSalt:   util.RandString(64),
		Signature:        nodeps.DdevFileSignature,
		SiteSettings:     "wp-config.php",
		SiteSettingsDdev: "wp-config-ddev.php",
		AbsPath:          absPath,
		DbCharset:        "utf8",
		DbCollate:        "",
	}
}

// wordPressHooks adds a wp-specific hooks example for post-start
const wordPressHooks = `# Un-comment to emit the WP CLI version after ddev start.
#  post-start:
#    - exec: wp cli version
`

// getWordpressHooks for appending as byte array
func getWordpressHooks() []byte {
	return []byte(wordPressHooks)
}

// getWordpressUploadDirs will return the default paths.
func getWordpressUploadDirs(_ *DdevApp) []string {
	return []string{"wp-content/uploads"}
}

const wordpressConfigInstructions = `
An existing user-managed wp-config.php file has been detected!
Project ddev settings have been written to:

%s

Please comment out any database connection settings in your wp-config.php and
add the following snippet to your wp-config.php, near the bottom of the file
and before the include of wp-settings.php:

// Include for ddev-managed settings in wp-config-ddev.php.
$ddev_settings = dirname(__FILE__) . '/wp-config-ddev.php';
if (is_readable($ddev_settings) && !defined('DB_USER')) {
  require_once($ddev_settings);
}

If you don't care about those settings, or config is managed in a .env
file, etc, then you can eliminate this message by putting a line that says
// wp-config-ddev.php not needed
in your wp-config.php
`

// createWordpressSettingsFile creates a Wordpress settings file from a
// template. Returns full path to location of file + err
func createWordpressSettingsFile(app *DdevApp) (string, error) {
	absPath, err := wordpressGetRelativeAbsPath(app)
	if err != nil {
		if strings.Contains(err.Error(), "multiple") {
			util.Warning("Unable to determine ABSPATH: %v", err)
		}
	}

	config := NewWordpressConfig(app, absPath)

	//  write ddev settings file
	if err := writeWordpressDdevSettingsFile(config, app.SiteDdevSettingsFile); err != nil {
		return "", err
	}

	// Check if an existing WordPress settings file exists
	if fileutil.FileExists(app.SiteSettingsPath) {
		// Check if existing WordPress settings file is ddev-managed
		sigExists, err := fileutil.FgrepStringInFile(app.SiteSettingsPath, nodeps.DdevFileSignature)
		if err != nil {
			return "", err
		}

		if sigExists {
			// Settings file is ddev-managed, overwriting is safe
			if err := writeWordpressSettingsFile(config, app.SiteSettingsPath); err != nil {
				return "", err
			}
		} else {
			// Settings file exists and is not ddev-managed, alert the user to the location
			// of the generated ddev settings file
			includeExists, err := fileutil.FgrepStringInFile(app.SiteSettingsPath, "wp-config-ddev.php")
			if err != nil {
				util.Warning("Unable to check that the ddev settings file has been included: %v", err)
			}

			if includeExists {
				util.Success("Include of %s found in %s", app.SiteDdevSettingsFile, app.SiteSettingsPath)
			} else {
				util.Warning(wordpressConfigInstructions, app.SiteDdevSettingsFile)
			}
		}
	} else {
		// If settings file does not exist, write basic settings file including it
		if err := writeWordpressSettingsFile(config, app.SiteSettingsPath); err != nil {
			return "", err
		}
	}

	return app.SiteDdevSettingsFile, nil
}

// writeWordpressSettingsFile dynamically produces valid wp-config.php file by combining a configuration
// object with a data-driven template.
func writeWordpressSettingsFile(wordpressConfig *WordpressConfig, filePath string) error {
	t, err := template.New("wp-config.php").ParseFS(bundledAssets, "wordpress/wp-config.php")
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

	//nolint: revive
	if err = t.Execute(file, wordpressConfig); err != nil {
		return err
	}

	return nil
}

// writeWordpressDdevSettingsFile unconditionally creates the file that contains ddev-specific settings.
func writeWordpressDdevSettingsFile(config *WordpressConfig, filePath string) error {
	if fileutil.FileExists(filePath) {
		// Check if the file is managed by ddev.
		signatureFound, err := fileutil.FgrepStringInFile(filePath, nodeps.DdevFileSignature)
		if err != nil {
			return err
		}

		// If the signature wasn't found, warn the user and return.
		if !signatureFound {
			util.Warning("%s already exists and is managed by the user.", filepath.Base(filePath))
			return nil
		}
	}

	t, err := template.New("wp-config-ddev.php").ParseFS(bundledAssets, "wordpress/wp-config-ddev.php")
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

	err = t.Execute(file, config)
	return err
}

// setWordpressSiteSettingsPaths sets the expected settings files paths for
// a wordpress site.
func setWordpressSiteSettingsPaths(app *DdevApp) {
	config := NewWordpressConfig(app, "")

	settingsFileBasePath := filepath.Join(app.AppRoot, app.Docroot)
	app.SiteSettingsPath = filepath.Join(settingsFileBasePath, config.SiteSettings)
	app.SiteDdevSettingsFile = filepath.Join(settingsFileBasePath, config.SiteSettingsDdev)
}

// isWordpressApp returns true if the app of of type wordpress
func isWordpressApp(app *DdevApp) bool {
	_, err := wordpressGetRelativeAbsPath(app)
	if err != nil {
		// Multiple abspath candidates is an issue, but is still a valid
		// indicator that this is a WordPress app
		if strings.Contains(err.Error(), "multiple") {
			return true
		}

		return false
	}

	return true
}

// wordpressImportFilesAction defines the Wordpress workflow for importing project files.
// The Wordpress workflow is currently identical to the Drupal import-files workflow.
func wordpressImportFilesAction(app *DdevApp, target, importPath, extPath string) error {
	destPath := app.calculateHostUploadDirFullPath(target)

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

	//nolint: revive
	if err := fileutil.CopyDir(importPath, destPath); err != nil {
		return err
	}

	return nil
}

// wordpressGetRelativeAbsPath returns the portion of the ABSPATH value that will come after "/" in wp-config.php -
// this is done by searching (at a max depth of one directory from the docroot) for wp-settings.php, the
// file we're using as a signal to indicate that this is a WordPress project.
func wordpressGetRelativeAbsPath(app *DdevApp) (string, error) {
	needle := "wp-settings.php"

	curDirMatches, err := filepath.Glob(filepath.Join(app.AppRoot, app.Docroot, needle))
	if err != nil {
		return "", err
	}

	if len(curDirMatches) > 0 {
		return "", nil
	}

	subDirMatches, err := filepath.Glob(filepath.Join(app.AppRoot, app.Docroot, "*", needle))
	if err != nil {
		return "", err
	}

	if len(subDirMatches) == 0 {
		return "", fmt.Errorf("unable to find %s in subdirectories", needle)
	}

	if len(subDirMatches) > 1 {
		return "", fmt.Errorf("multiple subdirectories contain %s", needle)
	}

	absPath := filepath.Base(filepath.Dir(subDirMatches[0]))

	return absPath, nil
}
