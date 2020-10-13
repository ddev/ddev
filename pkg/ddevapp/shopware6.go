package ddevapp

import (
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/nodeps"
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

//// createShopwareSettingsFile manages creation and modification of local.xml.
//func createShopwareSettingsFile(app *DdevApp) (string, error) {
//
//	return app.SiteDdevSettingsFile, nil
//}

// setShopware6SiteSettingsPaths sets the paths to settings.php for templating.
func setShopware6SiteSettingsPaths(app *DdevApp) {
	app.SiteSettingsPath = filepath.Join(app.AppRoot, ".env")
}

// shopware6ImportFilesAction defines the magento workflow for importing project files.
//func shopware6ImportFilesAction(app *DdevApp, importPath, extPath string) error {
//	destPath := filepath.Join(app.GetAppRoot(), app.GetDocroot(), app.GetUploadDir())
//
//	// parent of destination dir should exist
//	if !fileutil.FileExists(filepath.Dir(destPath)) {
//		return fmt.Errorf("unable to import to %s: parent directory does not exist", destPath)
//	}
//
//	// parent of destination dir should be writable.
//	if err := os.Chmod(filepath.Dir(destPath), 0755); err != nil {
//		return err
//	}
//
//	// If the destination path exists, remove it as was warned
//	if fileutil.FileExists(destPath) {
//		if err := os.RemoveAll(destPath); err != nil {
//			return fmt.Errorf("failed to cleanup %s before import: %v", destPath, err)
//		}
//	}
//
//	if isTar(importPath) {
//		if err := archive.Untar(importPath, destPath, extPath); err != nil {
//			return fmt.Errorf("failed to extract provided archive: %v", err)
//		}
//
//		return nil
//	}
//
//	if isZip(importPath) {
//		if err := archive.Unzip(importPath, destPath, extPath); err != nil {
//			return fmt.Errorf("failed to extract provided archive: %v", err)
//		}
//
//		return nil
//	}
//
//	if err := fileutil.CopyDir(importPath, destPath); err != nil {
//		return err
//	}
//
//	return nil
//}

//// getShopwareUploadDir will return a custom upload dir if defined, returning a default path if not.
//func getShopwareUploadDir(app *DdevApp) string {
//	if app.UploadDir == "" {
//		return "media"
//	}
//
//	return app.UploadDir
//}

func shopware6PostStartAction(app *DdevApp) error {
	//TODO: Set this up to use the right warning message

	//if fileutil.FileExists(filepath.Join(app.AppRoot, ".env")) {
	//	isConfiguredDbConnection, _ := fileutil.FgrepStringInFile(app.SiteSettingsPath, `DATABASE_URL="mysql://db:db@db:3306/db"`)
	//	isAppURLCorrect, err := fileutil.FgrepStringInFile(app.SiteSettingsPath, fmt.Sprintf(`APP_URL="%s"`, app.GetPrimaryURL()))
	//	if err == nil && !isConfiguredDbConnection && !isAppURLCorrect {
	//		envSettingsWarning(WarnTypeNotConfigured)
	//	}
	//} else {
	//	envSettingsWarning(WarnTypeAbsent)
	//}

	return nil
}

// shopware6ConfigOverrideAction overrides mariadb_version for shopware6,
// since it requires at least 10.3
func shopware6ConfigOverrideAction(app *DdevApp) error {
	app.MariaDBVersion = nodeps.MariaDB103
	return nil
}
