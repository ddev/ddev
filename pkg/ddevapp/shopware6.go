package ddevapp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/util"
	copy2 "github.com/otiai10/copy"
)

// isShopware6App returns true if the app is of type shopware6
func isShopware6App(app *DdevApp) bool {
	isShopware6, err := fileutil.FgrepStringInFile(filepath.Join(app.AppRoot, "composer.json"), `"name": "shopware/production"`)
	if err == nil && isShopware6 {
		return true
	}
	return false
}

// setShopware6SiteSettingsPaths sets the paths to .env.local file.
func setShopware6SiteSettingsPaths(app *DdevApp) {
	app.SiteSettingsPath = filepath.Join(app.AppRoot, ".env.local")
}

// shopware6ImportFilesAction defines the shopware6 workflow for importing user-generated files.
func shopware6ImportFilesAction(app *DdevApp, uploadDir, importPath, extPath string) error {
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

// getShopwareUploadDirs will return the default paths.
func getShopwareUploadDirs(_ *DdevApp) []string {
	return []string{"media"}
}

// shopware6PostStartAction checks to see if the .env.local file is set up
func shopware6PostStartAction(app *DdevApp) error {
	if app.DisableSettingsManagement {
		return nil
	}
	envFilePath := filepath.Join(app.AppRoot, ".env.local")
	_, envText, err := ReadProjectEnvFile(envFilePath)
	var envMap = map[string]string{
		"DATABASE_URL": `mysql://db:db@db:3306/db`,
		"APP_ENV":      "dev",
		"APP_URL":      app.GetPrimaryURL(),
		"MAILER_DSN":   `smtp://127.0.0.1:1025?encryption=&auth_mode=`,
	}
	// If the .env.local doesn't exist, create it.
	switch {
	case err == nil:
		util.Warning("Updating %s with %v", envFilePath, envMap)
		fallthrough
	case errors.Is(err, os.ErrNotExist):
		err := WriteProjectEnvFile(envFilePath, envMap, envText)
		if err != nil {
			return err
		}
	default:
		util.Warning("error opening %s: %v", envFilePath, err)
	}

	return nil
}
