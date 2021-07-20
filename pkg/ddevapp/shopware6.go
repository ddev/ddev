package ddevapp

import (
	"fmt"
	"github.com/drud/ddev/pkg/archive"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/util"
	"os"
	"path/filepath"
)

// isShopware6App returns true if the app is of type shopware6
func isShopware6App(app *DdevApp) bool {
	isShopware6, err := fileutil.FgrepStringInFile(filepath.Join(app.AppRoot, "config", "README.md"), "packages/shopware.yaml")
	if err == nil && isShopware6 {
		return true
	}
	return false
}

// setShopware6SiteSettingsPaths sets the paths to settings.php for templating.
func setShopware6SiteSettingsPaths(app *DdevApp) {
	app.SiteSettingsPath = filepath.Join(app.AppRoot, ".env")
}

// shopware6ImportFilesAction defines the shopware6 workflow for importing user-generated files.
func shopware6ImportFilesAction(app *DdevApp, importPath, extPath string) error {
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

	//nolint: revive
	if err := fileutil.CopyDir(importPath, destPath); err != nil {
		return err
	}

	return nil
}

// getShopwareUploadDir will return a custom upload dir if defined,
//returning a default path if not; this is relative to the docroot
func getShopwareUploadDir(app *DdevApp) string {
	if app.UploadDir == "" {
		return "media"
	}

	return app.UploadDir
}

// shopware6PostStartAction checks to see if the .env file is set up
func shopware6PostStartAction(app *DdevApp) error {
	envFile := filepath.Join(app.AppRoot, ".env")
	var addOnConfig string
	expectedDatabaseURL := `DATABASE_URL="mysql://db:db@db:3306/db"`
	expectedPrimaryURL := fmt.Sprintf(`APP_URL="%s"`, app.GetPrimaryURL())
	expectedMailerURL := `MAILER_URL="smtp://localhost:1025?encryption=&auth_mode="`

	if fileutil.FileExists(envFile) {
		isConfiguredDbConnection, _ := fileutil.FgrepStringInFile(app.SiteSettingsPath, expectedDatabaseURL)
		isAppURLCorrect, _ := fileutil.FgrepStringInFile(app.SiteSettingsPath, expectedPrimaryURL)
		isMailhogConfigCorrect, _ := fileutil.FgrepStringInFile(envFile, expectedMailerURL)

		if !isConfiguredDbConnection {
			addOnConfig = addOnConfig + expectedDatabaseURL + "\n"
		}
		if !isAppURLCorrect {
			addOnConfig = addOnConfig + expectedPrimaryURL + "\n"
		}
		if !isMailhogConfigCorrect {
			addOnConfig = addOnConfig + expectedMailerURL + "\n"
		}
		if addOnConfig != "" {
			addOnConfig = "# =================\n# Configuration added by ddev\n" + addOnConfig
			err := fileutil.AppendStringToFile(envFile, addOnConfig)
			if err != nil {
				return err
			}
			util.Warning("ddev configuration added to %s", envFile)
		}
	} else {
		util.Warning("the .env file has not yet been created (%s)", envFile)
	}

	return nil
}
