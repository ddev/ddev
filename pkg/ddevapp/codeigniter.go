package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
)

// createCodeIgniterSettingsFile creates/updates the .env file for CodeIgniter 4
func createCodeIgniterSettingsFile(app *DdevApp) (string, error) {
	envFilePath := filepath.Join(app.AppRoot, app.ComposerRoot, ".env")

	_, envText, err := ReadProjectEnvFile(envFilePath)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("unable to read .env file: %v", err)
	}

	// If .env doesn't exist, try to copy from env or .env.example
	if os.IsNotExist(err) {
		envExamplePath := filepath.Join(app.AppRoot, app.ComposerRoot, "env")
		if !fileutil.FileExists(envExamplePath) {
			envExamplePath = filepath.Join(app.AppRoot, app.ComposerRoot, ".env.example")
		}
		if fileutil.FileExists(envExamplePath) {
			if err := fileutil.CopyFile(envExamplePath, envFilePath); err != nil {
				return "", fmt.Errorf("failed to copy %s to .env: %v", envExamplePath, err)
			}
			_, envText, err = ReadProjectEnvFile(envFilePath)
			if err != nil {
				return "", err
			}
		} else {
			util.Debug("CodeIgniter: env file does not exist yet, not trying to process it")
			return "", nil
		}
	}

	// Build env map with CodeIgniter settings
	envMap := map[string]string{
		"CI_ENVIRONMENT": "development",
		"app.baseURL":    app.GetPrimaryURL(),
	}

	// Only set database configuration if db container is not omitted
	if !slices.Contains(app.OmitContainers, "db") {
		driver := "MySQLi"
		port := "3306"
		charset := "utf8mb4"
		if app.Database.Type == nodeps.Postgres {
			driver = "Postgre"
			port = "5432"
			charset = "utf8"
		}

		envMap["database.default.hostname"] = "db"
		envMap["database.default.database"] = "db"
		envMap["database.default.username"] = "db"
		envMap["database.default.password"] = "db"
		envMap["database.default.DBDriver"] = driver
		envMap["database.default.port"] = port
		envMap["database.default.charset"] = charset
	}

	err = WriteProjectEnvFile(envFilePath, envMap, envText)
	if err != nil {
		return "", err
	}

	return envFilePath, nil
}

// getCodeIgniterUploadDirs returns upload directories for CodeIgniter 4
func getCodeIgniterUploadDirs(_ *DdevApp) []string {
	return []string{"writable/uploads"}
}

// getCodeIgniterHooks returns example hook comments for CodeIgniter
func getCodeIgniterHooks() []byte {
	return []byte(`#  post-start:
#    - exec: php spark migrate
#    - exec: php spark db:seed`)
}

// setCodeIgniterSiteSettingsPaths sets the paths to settings files
func setCodeIgniterSiteSettingsPaths(app *DdevApp) {
	app.SiteSettingsPath = filepath.Join(app.AppRoot, app.ComposerRoot, ".env")
	app.SiteDdevSettingsFile = ""
}

// isCodeIgniterApp detects if this is a CodeIgniter 4 application
func isCodeIgniterApp(app *DdevApp) bool {
	// Check for CodeIgniter 4 specific files
	sparkPath := filepath.Join(app.AppRoot, app.ComposerRoot, "spark")
	appConfigPath := filepath.Join(app.AppRoot, app.ComposerRoot, "app", "Config", "App.php")
	publicIndexPath := filepath.Join(app.AppRoot, app.ComposerRoot, "public", "index.php")

	return fileutil.FileExists(sparkPath) &&
		fileutil.FileExists(appConfigPath) &&
		fileutil.FileExists(publicIndexPath)
}

// codeIgniterImportFilesAction handles file imports
func codeIgniterImportFilesAction(app *DdevApp, uploadDir, importPath, extPath string) error {
	destPath := app.calculateHostUploadDirFullPath(uploadDir)

	// Ensure destination directory exists
	err := os.MkdirAll(destPath, 0755)
	if err != nil {
		return err
	}

	// Copy files from import path to destination
	if isTar(importPath) {
		err = archive.Untar(importPath, destPath, extPath)
	} else {
		err = fileutil.CopyDir(importPath, destPath)
	}

	return err
}
