package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
)

// isMagentoApp returns true if the app is of type magento
func isMagentoApp(app *DdevApp) bool {
	ism1, err := fileutil.FgrepStringInFile(filepath.Join(app.AppRoot, app.Docroot, "README.md"), `Magento - Long Term Support`)
	if err == nil && ism1 {
		return true
	}
	return false
}

// isMagento2App returns true if the app is of type magento2
func isMagento2App(app *DdevApp) bool {
	ism2, err := fileutil.FgrepStringInFile(filepath.Join(app.AppRoot, app.Docroot, "..", "SECURITY.md"), `https://hackerone.com/magento`)
	if err == nil && ism2 {
		return true
	}
	return false
}

// createMagentoSettingsFile manages creation and modification of local.xml.
func createMagentoSettingsFile(app *DdevApp) (string, error) {

	if fileutil.FileExists(app.SiteSettingsPath) {
		// Check if the file is managed by ddev.
		signatureFound, err := fileutil.FgrepStringInFile(app.SiteSettingsPath, nodeps.DdevFileSignature)
		if err != nil {
			return "", err
		}

		// If the signature wasn't found, warn the user and return.
		if !signatureFound {
			util.Warning("%s already exists and is managed by the user.", app.SiteSettingsPath)
			return "", nil
		}
	} else {
		output.UserOut.Printf("No %s file exists, creating one", app.SiteSettingsPath)

		content, err := bundledAssets.ReadFile("magento/local.xml")
		if err != nil {
			return "", err
		}
		templateVars := map[string]interface{}{"DBHostname": "db"}
		err = fileutil.TemplateStringToFile(string(content), templateVars, app.SiteSettingsPath)
		if err != nil {
			return "", err
		}
	}

	return app.SiteDdevSettingsFile, nil
}

// setMagentoSiteSettingsPaths sets the paths to settings.php for templating.
func setMagentoSiteSettingsPaths(app *DdevApp) {
	app.SiteSettingsPath = filepath.Join(app.AppRoot, app.Docroot, "app", "etc", "local.xml")
}

// magentoImportFilesAction defines the magento workflow for importing project files.
func magentoImportFilesAction(app *DdevApp, uploadDir, importPath, extPath string) error {
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

// getMagentoUploadDirs will return the default paths.
func getMagentoUploadDirs(_ *DdevApp) []string {
	return []string{"media"}
}

// getMagento2UploadDirs will return the default paths.
func getMagento2UploadDirs(_ *DdevApp) []string {
	return []string{"media"}
}

// createMagento2SettingsFile manages creation and modification of app/etc/env.php.
func createMagento2SettingsFile(app *DdevApp) (string, error) {

	if fileutil.FileExists(app.SiteSettingsPath) {
		// Check if the file is managed by ddev.
		signatureFound, err := fileutil.FgrepStringInFile(app.SiteSettingsPath, nodeps.DdevFileSignature)
		if err != nil {
			return "", err
		}

		// If the signature wasn't found, warn the user and return.
		if !signatureFound {
			util.Warning("%s already exists and is managed by the user.", app.SiteSettingsPath)
			return "", nil
		}
	} else {
		output.UserOut.Printf("No %s file exists, creating one", app.SiteSettingsPath)

		content, err := bundledAssets.ReadFile("magento/env.php")
		if err != nil {
			return "", err
		}

		templateVars := map[string]interface{}{"DBHostname": "db"}
		err = fileutil.TemplateStringToFile(string(content), templateVars, app.SiteSettingsPath)
		if err != nil {
			return "", err
		}
	}

	return app.SiteDdevSettingsFile, nil
}

// setMagento2SiteSettingsPaths sets the paths to settings.php for templating.
func setMagento2SiteSettingsPaths(app *DdevApp) {
	app.SiteSettingsPath = filepath.Join(app.AppRoot, app.Docroot, "..", "app", "etc", "env.php")
}

func magentoConfigOverrideAction(app *DdevApp) error {
	app.PHPVersion = nodeps.PHP74
	return nil
}

// Latest magento2 requires php8.1
func magento2ConfigOverrideAction(app *DdevApp) error {
	app.PHPVersion = nodeps.PHP81
	return nil
}
