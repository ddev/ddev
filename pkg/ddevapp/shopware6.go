package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/fileutil"
)

// isShopware6App returns true if the app is of type shopware6
func isShopware6App(app *DdevApp) bool {
	isShopware6, err := fileutil.FgrepStringInFile(filepath.Join(app.AppRoot, "config", "README.md"), "packages/shopware.yaml")
	if err == nil && isShopware6 {
		return true
	}
	return false
}

// setShopware6SiteSettingsPaths sets the paths to .env file.
func setShopware6SiteSettingsPaths(app *DdevApp) {
	app.SiteSettingsPath = filepath.Join(app.AppRoot, ".env")
}

// shopware6ImportFilesAction defines the shopware6 workflow for importing user-generated files.
func shopware6ImportFilesAction(app *DdevApp, uploadDir, importPath, extPath string) error {
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

// getShopwareUploadDirs will return the default paths.
func getShopwareUploadDirs(_ *DdevApp) []string {
	return []string{"media"}
}

// shopware6PostStartAction checks to see if the .env file is set up
func shopware6PostStartAction(app *DdevApp) error {
	if app.DisableSettingsManagement {
		return nil
	}
	envFilePath := filepath.Join(app.AppRoot, ".env")
	_, envText, err := ReadProjectEnvFile(envFilePath)
	var envMap = map[string]string{
		"DATABASE_URL": `mysql://db:db@db:3306/db`,
		"APP_URL":      app.GetPrimaryURL(),
		"MAILER_URL":   `smtp://127.0.0.1:1025?encryption=&auth_mode=`,
	}
	// Shopware 6 refuses to do bin/console system:setup if the env file exists,
	// so if it doesn't exist, wait for it to be created
	if err == nil {
		err := WriteProjectEnvFile(envFilePath, envMap, envText)
		if err != nil {
			return err
		}
	}

	return nil
}
