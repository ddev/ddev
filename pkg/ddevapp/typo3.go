package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"text/template"

	"github.com/Masterminds/semver/v3"
	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/composer"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	copy2 "github.com/otiai10/copy"
)

// createTypo3SettingsFile creates the app's settings.php and
// additional.php, adding things like database host, name, and
// password. Returns the fullpath to settings file and error
func createTypo3SettingsFile(app *DdevApp) (string, error) {
	if filepath.Dir(app.SiteDdevSettingsFile) == app.AppRoot {
		// As long as the final settings folder is not defined, early return
		return app.SiteDdevSettingsFile, nil
	}

	if !fileutil.FileExists(app.SiteSettingsPath) {
		util.Warning("TYPO3 does not seem to have been set up yet, missing %s (%s)", filepath.Base(app.SiteSettingsPath), app.SiteSettingsPath)
	}

	// TYPO3 DDEV settings file will be additional.php (app.SiteDdevSettingsFile).
	// Check if the file already exists.
	if fileutil.FileExists(app.SiteDdevSettingsFile) {
		// Check if the file is managed by ddev.
		signatureFound, err := fileutil.FgrepStringInFile(app.SiteDdevSettingsFile, nodeps.DdevFileSignature)
		if err != nil {
			return "", err
		}

		// If the signature wasn't found, warn the user and return.
		if !signatureFound {
			util.Warning("%s already exists and is managed by the user.", filepath.Base(app.SiteDdevSettingsFile))
			return app.SiteDdevSettingsFile, nil
		}
	}

	output.UserOut.Printf("Generating %s file for database connection.", filepath.Base(app.SiteDdevSettingsFile))
	if err := writeTypo3SettingsFile(app); err != nil {
		return "", fmt.Errorf("failed to write TYPO3 %s file: %v", app.SiteDdevSettingsFile, err.Error())
	}

	return app.SiteDdevSettingsFile, nil
}

// writeTypo3SettingsFile produces AdditionalConfiguration.php file
// It's assumed that the settings.php already exists, and we're
// overriding the db config values in it. The typo3conf/ directory will
// be created if it does not yet exist.
func writeTypo3SettingsFile(app *DdevApp) error {
	filePath := app.SiteDdevSettingsFile

	// Ensure target directory is writable.
	dir := filepath.Dir(filePath)
	var perms os.FileMode = 0755
	if err := util.Chmod(dir, perms); err != nil {
		if !os.IsNotExist(err) {
			// The directory exists, but chmod failed.
			return err
		}

		// The directory doesn't exist, create it with the appropriate permissions.
		if err := os.MkdirAll(dir, perms); err != nil {
			return err
		}
	}
	dbDriver := "mysqli" // mysqli is the driver used in default LocalConfiguration.php
	if app.Database.Type == nodeps.Postgres {
		dbDriver = "pdo_pgsql"
	}
	settings := map[string]interface{}{"DBHostname": "db", "DBDriver": dbDriver, "DBPort": GetExposedPort(app, "db")}

	// Ensure target directory exists and is writable
	if err := util.Chmod(dir, 0755); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	f, err := os.Create(filePath)
	if err != nil {
		return err
	}

	t, err := template.New("AdditionalConfiguration.php").ParseFS(bundledAssets, "typo3/AdditionalConfiguration.php")
	if err != nil {
		return err
	}

	if err = t.Execute(f, settings); err != nil {
		return err
	}
	if err != nil {
		return err
	}
	return nil
}

// getTypo3UploadDirs will return the default paths.
func getTypo3UploadDirs(_ *DdevApp) []string {
	return []string{"fileadmin"}
}

// Typo3Hooks adds a TYPO3-specific hooks example for post-import-db
const Typo3Hooks = `#  post-start:
#    - exec: composer install -d /var/www/html
`

// getTypo3Hooks for appending as byte array
func getTypo3Hooks() []byte {
	// We don't have anything new to add yet.
	return []byte(Typo3Hooks)
}

// setTypo3SiteSettingsPaths sets the paths to settings files for templating. TYPO3 supports different setup structures,
// composer, legacy and mono repository which differs in places and naming of these files depending on mode and version.
// Thus different detection function are provided to determine the best suitable place and filenames.
func setTypo3SiteSettingsPaths(app *DdevApp) {
	var settingsFilePath, localSettingsFilePath string

	if isTypo3ComposerV12OrHigher(app) {
		// Since TYPO3 v12 the configuration files are named `settings.php` and `additional.php` and expected
		// to be in `project-folder/config/system` for composer mode installations. Set them now to ensure ddev
		// writes them in the correct place.
		settingsFileBasePath := filepath.Join(app.AppRoot, app.ComposerRoot)
		settingsFilePath = filepath.Join(settingsFileBasePath, "config", "system", "settings.php")
		localSettingsFilePath = filepath.Join(settingsFileBasePath, "config", "system", "additional.php")
	} else if isTypo3LegacyV12OrHigher(app) {
		// Since TYPO3 v12 the configuration files are names `settings.php` and `additional.php` and expected to
		// be in `docroot/typo3conf/system/` which differs from TYPO3 v12 or higher composer mode installations
		// above. TYPO3 v12 or higher Core Development mono repository is basically a non-composer mode installation
		// albeit having a composer.json in the project root folder. Set now the correct path and configuration file
		// names suitable for these setups.
		settingsFileBasePath := filepath.Join(app.AppRoot, app.Docroot)
		settingsFilePath = filepath.Join(settingsFileBasePath, "typo3conf", "system", "settings.php")
		localSettingsFilePath = filepath.Join(settingsFileBasePath, "typo3conf", "system", "additional.php")
	} else if isTypo3LegacyV11OrLower(app) {
		// Up to TYPO3 v11 configuration files had the same naming and resided in the `docroot/typo3conf/` folder
		// which dates back to times before composer has been a gamechanger in the PHP ecosystem. TYPO3 Core Development
		// mono repository shared the same and we do not have to differenciate between composer and legacy mode here.
		settingsFileBasePath := filepath.Join(app.AppRoot, app.Docroot)
		settingsFilePath = filepath.Join(settingsFileBasePath, "typo3conf", "LocalConfiguration.php")
		localSettingsFilePath = filepath.Join(settingsFileBasePath, "typo3conf", "AdditionalConfiguration.php")
	} else {
		// As long as TYPO3 is not installed, the file paths are set to the AppRoot to avoid the creation of the
		// .gitignore in the wrong location. Set the longstanding and old names here as it does not matter at all.
		// createTypo3SettingsFile skips early if these configuration are returned to be in the AppRoot which has
		// never been a suitable place for TYPO3 literally not writing them and flags them skipable.
		settingsFilePath = filepath.Join(app.AppRoot, "LocalConfiguration.php")
		localSettingsFilePath = filepath.Join(app.AppRoot, "AdditionalConfiguration.php")
	}

	// Update file paths to the above determined paths
	app.SiteSettingsPath = settingsFilePath
	app.SiteDdevSettingsFile = localSettingsFilePath
}

// isTypo3App returns true if the app is of type typo3 including composer mode, legacy mode and mono repository
func isTypo3App(app *DdevApp) bool {
	if isTypo3ComposerV12OrHigher(app) {
		return true
	}
	if isTypo3LegacyV12OrHigher(app) {
		return true
	}
	if isTypo3LegacyV11OrLower(app) {
		return true
	}
	return false
}

// Up to TYPO3 v11 legacy and composer mode installation provided the system extensions and the configuration in the
// same structure within the docroot. Only difference is that for composer mode a public subfolder is used as docroot
// which make no difference in the way to detect both variants. TYPO3 Core Development monorepository also shared the
// same filestem layout. Thus using only a simple docroot/typo3 folder check here and be fine.
func isTypo3LegacyV11OrLower(app *DdevApp) bool {
	typo3Folder := filepath.Join(app.AppRoot, app.Docroot, "typo3")

	// Check if the folder exists, fails if a symlink target does not exist.
	if _, err := os.Stat(typo3Folder); !os.IsNotExist(err) {
		return true
	}

	// Check if a symlink exists, succeeds even if the target does not exist.
	if _, err := os.Lstat(typo3Folder); !os.IsNotExist(err) {
		return true
	}

	return false
}

// Since TYPO3 v12 system extensions are no longer installed into `public/typo3/sysext/` and extension no longer into
// `public/typo3conf/ext/`. This function verifies for a composer mode installation first and extract the TYPO3 version
// from the `Typo3Version` php class if composer dependencies are installed and looks that source file up within the
// composer vendor folder. The TYPO3 Core Development mono repository has a root composer.json but does not have the
// system extensions within the vendor folder like composer mode installations and thus making it safe to check for the
// Typo3Version class within the typo3/cms-core package folder.
func isTypo3ComposerV12OrHigher(app *DdevApp) bool {
	composerManifest, _ := composer.NewManifest(filepath.Join(app.AppRoot, app.ComposerRoot, "composer.json"))
	vendorDir := "vendor"
	if composerManifest != nil {
		vendorDir = composerManifest.GetVendorDir()
	}

	versionFilePath := filepath.Join(app.AppRoot, app.ComposerRoot, vendorDir, "typo3", "cms-core", "Classes", "Information", "Typo3Version.php")
	versionFile, err := fileutil.ReadFileIntoString(versionFilePath)

	// Typo3Version class exists since v10.3.0. Before v11.5.0 the core was always
	// installed into the folder public/typo3 so we can early return if the file
	// is not found in the vendor folder.
	if err != nil {
		util.Debug("TYPO3 version class not found in '%s' for project %s, installed version is assumed to be older than 11.5.0: %v", versionFilePath, app.Name, err)
		return false
	}

	// We may have a TYPO3 version 11 or higher and therefor have to parse the
	// class file to properly detect the version.
	re := regexp.MustCompile(`const\s+VERSION\s*=\s*'([^']+)`)

	matches := re.FindStringSubmatch(versionFile)

	if len(matches) < 2 {
		util.Warning("Unexpected Typo3Version found for project %s in %v.", app.Name, versionFile)
		return false
	}

	version, err := semver.NewVersion(matches[1])
	if err != nil {
		// This case never should happen
		util.Warning("Unexpected error while parsing TYPO3 version ('%s') for project %s: %v.", matches[1], app.Name, err)
		return false
	}

	util.Debug("Found TYPO3 version %v for project %s.", version.Original(), app.Name)

	return version.Major() >= 12
}

// TYPO3 v12 changed some stuff, mostly for composer mode. Place and names of main and additional configuration changed
// for composer and legacy mode with different places and also differs in places where system extensions can be found or
// user extensions are installed. This function verifies if the current project reflects a TYPO3 v12 or higher legacy
// mode installation to set the correct configuration folder and names in function `setTypo3SiteSettingsPaths`.
// The TYPO3 Core Development mono repository uses literally the same layout albeit having a composer.json in the root
// folder. In the mono repository the TYPO3 system extensions resides not within the composer vendor folder.
func isTypo3LegacyV12OrHigher(app *DdevApp) bool {
	if !isTypo3LegacyV11OrLower(app) {
		return false
	}

	typo3Folder := filepath.Join(app.AppRoot, app.Docroot, "typo3")

	versionFilePath := filepath.Join(typo3Folder, "sysext", "core", "Classes", "Information", "Typo3Version.php")
	versionFile, err := fileutil.ReadFileIntoString(versionFilePath)

	// Typo3Version class exists since v10.3.0. Before v11.5.0 the core was always
	// installed into the folder public/typo3 so we can early return if the file
	// is not found in the vendor folder.
	if err != nil {
		util.Debug("TYPO3 version class not found in '%s' for project %s, installed version is assumed to be older than 11.5.0: %v", versionFilePath, app.Name, err)
		return false
	}

	// We may have a TYPO3 version 11 or higher and therefor have to parse the
	// class file to properly detect the version.
	re := regexp.MustCompile(`const\s+VERSION\s*=\s*'([^']+)`)

	matches := re.FindStringSubmatch(versionFile)

	if len(matches) < 2 {
		util.Warning("Unexpected Typo3Version found for project %s in %v.", app.Name, versionFile)
		return false
	}

	version, err := semver.NewVersion(matches[1])
	if err != nil {
		// This case never should happen
		util.Warning("Unexpected error while parsing TYPO3 version ('%s') for project %s: %v.", matches[1], app.Name, err)
		return false
	}

	util.Debug("Found TYPO3 version %v for project %s.", version.Original(), app.Name)

	return version.Major() >= 12
}

// typo3ImportFilesAction defines the TYPO3 workflow for importing project files.
// The TYPO3 import-files workflow is currently identical to the Drupal workflow.
func typo3ImportFilesAction(app *DdevApp, uploadDir, importPath, extPath string) error {
	destPath := app.calculateHostUploadDirFullPath(uploadDir)

	// parent of destination dir should exist
	if !fileutil.FileExists(filepath.Dir(destPath)) {
		return fmt.Errorf("unable to import to %s: parent directory does not exist", destPath)
	}

	// parent of destination dir should be writable.
	if err := util.Chmod(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// If the destination path exists, purge it as was warned
	if fileutil.FileExists(destPath) {
		if err := fileutil.PurgeDirectory(destPath); err != nil {
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

	if err := copy2.Copy(importPath, destPath); err != nil {
		return err
	}

	return nil
}
