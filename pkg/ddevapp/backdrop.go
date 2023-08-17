package ddevapp

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
)

// BackdropSettings holds database connection details for Backdrop.
type BackdropSettings struct {
	DatabaseName     string
	DatabaseUsername string
	DatabasePassword string
	DatabaseHost     string
	DatabaseDriver   string
	DatabasePort     string
	HashSalt         string
	Signature        string
	SiteSettings     string
	SiteSettingsDdev string
	DockerIP         string
	DBPublishedPort  int
}

// NewBackdropSettings produces a BackdropSettings object with default values.
func NewBackdropSettings(app *DdevApp) *BackdropSettings {
	dockerIP, _ := dockerutil.GetDockerIP()
	dbPublishedPort, _ := app.GetPublishedPort("db")

	return &BackdropSettings{
		DatabaseName:     "db",
		DatabaseUsername: "db",
		DatabasePassword: "db",
		DatabaseHost:     "ddev-" + app.Name + "-db",
		DatabaseDriver:   "mysql",
		DatabasePort:     GetExposedPort(app, "db"),
		HashSalt:         util.HashSalt(app.Name),
		Signature:        nodeps.DdevFileSignature,
		SiteSettings:     "settings.php",
		SiteSettingsDdev: "settings.ddev.php",
		DockerIP:         dockerIP,
		DBPublishedPort:  dbPublishedPort,
	}
}

// createBackdropSettingsFile manages creation and modification of settings.php and settings.ddev.php.
// If a settings.php file already exists, it will be modified to ensure that it includes
// settings.ddev.php, which contains ddev-specific configuration.
func createBackdropSettingsFile(app *DdevApp) (string, error) {
	settings := NewBackdropSettings(app)

	if !fileutil.FileExists(app.SiteSettingsPath) {
		output.UserOut.Printf("No %s file exists, creating one", settings.SiteSettings)
		if err := writeDrupalSettingsPHP(app.SiteSettingsPath, app.Type); err != nil {
			return "", err
		}
	}

	included, err := fileutil.FgrepStringInFile(app.SiteSettingsPath, settings.SiteSettingsDdev)
	if err != nil {
		return "", err
	}

	if included {
		output.UserOut.Printf("Existing %s includes %s", settings.SiteSettings, settings.SiteSettingsDdev)
	} else {
		output.UserOut.Printf("Existing %s file does not include %s, modifying to include ddev settings", settings.SiteSettings, settings.SiteSettingsDdev)

		if err = appendIncludeToDrupalSettingsFile(app.SiteSettingsPath, app.Type); err != nil {
			return "", fmt.Errorf("failed to include %s in %s: %v", settings.SiteSettingsDdev, settings.SiteSettings, err)
		}
	}

	if err = writeBackdropSettingsDdevPHP(settings, app.SiteDdevSettingsFile, app); err != nil {
		return "", fmt.Errorf("failed to write Drupal settings file %s: %v", app.SiteDdevSettingsFile, err)
	}

	return app.SiteDdevSettingsFile, nil
}

// writeBackdropSettingsDdevPHP dynamically produces a valid settings.ddev.php file
// by combining a configuration object with a data-driven template.
func writeBackdropSettingsDdevPHP(settings *BackdropSettings, filePath string, _ *DdevApp) error {
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
	t, err := template.New("settings.ddev.php").ParseFS(bundledAssets, path.Join("drupal/backdrop/settings.ddev.php"))
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

	err = t.Execute(file, settings)
	return err
}

// getBackdropUploadDirs will return the default paths.
func getBackdropUploadDirs(_ *DdevApp) []string {
	return []string{"files"}
}

// getBackdropHooks for appending as byte array.
func getBackdropHooks() []byte {
	backdropHooks := `#  post-import-db:
#    - exec: drush cc all
`
	return []byte(backdropHooks)
}

// setBackdropSiteSettingsPaths sets the paths to settings.php for templating.
func setBackdropSiteSettingsPaths(app *DdevApp) {
	settings := NewBackdropSettings(app)
	settingsFileBasePath := filepath.Join(app.AppRoot, app.Docroot)
	app.SiteSettingsPath = filepath.Join(settingsFileBasePath, settings.SiteSettings)
	app.SiteDdevSettingsFile = filepath.Join(settingsFileBasePath, settings.SiteSettingsDdev)
}

// isBackdropApp returns true if the app is of type "backdrop".
func isBackdropApp(app *DdevApp) bool {
	if _, err := os.Stat(filepath.Join(app.AppRoot, app.Docroot, "core/scripts/backdrop.sh")); err == nil {
		return true
	}
	return false
}

// backdropPostImportDBAction emits a warning about moving configuration into place
// appropriately in order for Backdrop to function properly.
func backdropPostImportDBAction(_ *DdevApp) error {
	util.Warning("Backdrop sites require your config JSON files to be located in your site's \"active\" configuration directory. Please refer to the Backdrop documentation (https://backdropcms.org/user-guide/moving-backdrop-site) for more information about this process.")
	return nil
}

// backdropImportFilesAction defines the Backdrop workflow for importing project files.
// The Backdrop workflow is currently identical to the Drupal import-files workflow.
func backdropImportFilesAction(app *DdevApp, uploadDir, importPath, extPath string) error {
	destPath := app.calculateHostUploadDirFullPath(uploadDir)

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

// backdropPostStartAction handles default post-start actions for backdrop apps, like ensuring
// useful permissions settings on sites/default.
func backdropPostStartAction(app *DdevApp) error {
	// Drush config has to be written after start because we don't know the ports until it's started
	err := WriteDrushrc(app, filepath.Join(filepath.Dir(app.SiteSettingsPath), "drushrc.php"))
	if err != nil {
		util.Warning("Failed to WriteDrushrc: %v", err)
	}

	if _, err = app.CreateSettingsFile(); err != nil {
		return fmt.Errorf("failed to write settings file %s: %v", app.SiteDdevSettingsFile, err)
	}
	return nil
}
