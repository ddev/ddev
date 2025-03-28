package ddevapp

import (
	"fmt"
	"path/filepath"

	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	copy2 "github.com/otiai10/copy"
)

// isOpenMageApp returns true if the app is of type magento
func isOpenMageApp(app *DdevApp) bool {
	ism1, err := fileutil.FgrepStringInFile(filepath.Join(app.GetAbsDocroot(false), "README.md"), `Magento - Long Term Support`)
	if err == nil && ism1 {
		return true
	}
	return false
}

// createOpenMageSettingsFile manages creation and modification of local.xml.
func createOpenMageSettingsFile(app *DdevApp) (string, error) {

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

		content, err := bundledAssets.ReadFile("openmage/local.xml")
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

// setOpenMageSiteSettingsPaths sets the paths to local.xml for templating.
func setOpenMageSiteSettingsPaths(app *DdevApp) {
	app.SiteSettingsPath = filepath.Join(app.GetAbsDocroot(false), "app", "etc", "local.xml")
}

// openmageImportFilesAction defines the magento workflow for importing project files.
func openmageImportFilesAction(app *DdevApp, uploadDir, importPath, extPath string) error {
	destPath := app.calculateHostUploadDirFullPath(uploadDir)

	// parent of destination dir should exist
	if !fileutil.FileExists(filepath.Dir(destPath)) {
		return fmt.Errorf("unable to import to %s: parent directory does not exist", destPath)
	}

	// parent of destination dir should be writable.
	if err := util.Chmod(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// If the destination path exists, remove it as was warned
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

// getOpenMageUploadDirs will return the default paths.
func getOpenMageUploadDirs(_ *DdevApp) []string {
	return []string{"media"}
}
