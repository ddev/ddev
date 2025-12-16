package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
)

// createCodeIgniterSettingsFile creates/updates the .env file for CodeIgniter 4
func createCodeIgniterSettingsFile(app *DdevApp) (string, error) {
	envFilePath := filepath.Join(app.AppRoot, app.ComposerRoot, ".env")

	// Check if .env already has DDEV config
	if fileutil.FileExists(envFilePath) {
		content, err := fileutil.ReadFileIntoString(envFilePath)
		if err == nil && strings.Contains(content, "# ddev-generated") {
			return envFilePath, nil
		}
	} else {
		// .env doesn't exist, try to copy from env or .env.example
		envExamplePath := filepath.Join(app.AppRoot, "env")
		if !fileutil.FileExists(envExamplePath) {
			envExamplePath = filepath.Join(app.AppRoot, ".env.example")
		}
		if fileutil.FileExists(envExamplePath) {
			if err := fileutil.CopyFile(envExamplePath, envFilePath); err != nil {
				return "", fmt.Errorf("failed to copy %s to .env: %v", envExamplePath, err)
			}
		}
	}

	// Build DB config dynamically from DDEV configuration.
	dbConfig := buildCodeIgniterDBConfig(app)

	// Always add base URL and set the enviroment to development
	cfg := fmt.Sprintf(`# ddev-generated
%s
#--------------------------------------------------------------------
# ENVIRONMENT
#--------------------------------------------------------------------

CI_ENVIRONMENT = development

#--------------------------------------------------------------------
# APP
#--------------------------------------------------------------------

app.baseURL = '%s'
`, dbConfig, app.GetPrimaryURL())

	f, err := os.OpenFile(envFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := f.WriteString(cfg); err != nil {
		return "", err
	}

	return envFilePath, nil
}

// buildCodeIgniterDBConfig returns the database stanza for CodeIgniter 4 based on DDEV config.
// If the DB container is omitted, it returns an empty string.
func buildCodeIgniterDBConfig(app *DdevApp) string {
	// No DB requested: omit_containers: [db]
	if isDBOmitted(app) {
		return "# Database omitted by DDEV configuration"
	}

	// Simple mapping: default to MySQL/MariaDB; switch to Postgres when requested.
	driver := "MySQLi"
	port := 3306
	if app != nil && app.Database.Type == nodeps.Postgres {
		driver = "Postgre"
		port = 5432
	}

	// Standard DDEV credentials/hostnames
	return fmt.Sprintf(`
#--------------------------------------------------------------------
# DATABASE
#--------------------------------------------------------------------

database.default.hostname = db
database.default.database = db
database.default.username = db
database.default.password = db
database.default.DBDriver = %s
database.default.DBPrefix =
database.default.port = %d
	`, driver, port)
}

// isDBOmitted returns true if "db" is in omit_containers.
func isDBOmitted(app *DdevApp) bool {
	if app == nil {
		return false
	}
	for _, s := range app.OmitContainers {
		if s == "db" {
			return true
		}
	}
	return false
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
	sparkPath := filepath.Join(app.AppRoot, "spark")
	appConfigPath := filepath.Join(app.AppRoot, "app", "Config", "App.php")
	publicIndexPath := filepath.Join(app.AppRoot, "public", "index.php")

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
