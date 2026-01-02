package ddevapp

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
)

// isExpressionEngineApp returns true if the app is of type expressionengine
func isExpressionEngineApp(app *DdevApp) bool {
	// Check for ExpressionEngine core file in the system directory
	systemPath := findExpressionEngineSystemPath(app)
	if systemPath == "" {
		return false
	}
	return fileutil.FileExists(filepath.Join(systemPath, "ee/ExpressionEngine.php"))
}

// setExpressionEngineSettingsPaths sets the path to the ExpressionEngine .env.php file
func setExpressionEngineSettingsPaths(app *DdevApp) {
	// Find the system directory path from index.php
	systemPath := findExpressionEngineSystemPath(app)
	if systemPath == "" {
		// Fallback to default location if we can't parse index.php
		app.SiteSettingsPath = filepath.Join(app.AppRoot, ".env.php")
		return
	}

	// .env.php lives at the same level as the system directory
	envPath := filepath.Join(filepath.Dir(systemPath), ".env.php")
	app.SiteSettingsPath = envPath
}

// findExpressionEngineSystemPath parses index.php to find the system directory path
func findExpressionEngineSystemPath(app *DdevApp) string {
	// Look for index.php in the docroot
	indexPath := filepath.Join(app.AppRoot, app.Docroot, "index.php")
	if !fileutil.FileExists(indexPath) {
		// Try alternate location in project root
		indexPath = filepath.Join(app.AppRoot, "index.php")
		if !fileutil.FileExists(indexPath) {
			return ""
		}
	}

	// Read index.php content
	indexContent, err := fileutil.ReadFileIntoString(indexPath)
	if err != nil {
		return ""
	}

	// Parse for $system_path = '...' or $system_path = "..."
	re := regexp.MustCompile(`\$system_path\s*=\s*['"]([^'"]+)['"]`)
	matches := re.FindStringSubmatch(indexContent)
	if len(matches) < 2 {
		return ""
	}

	systemPathRel := matches[1]

	// Resolve the path relative to the directory containing index.php
	indexDir := filepath.Dir(indexPath)
	systemPath := filepath.Join(indexDir, systemPathRel)

	// Clean the path
	systemPath = filepath.Clean(systemPath)

	return systemPath
}

// updateExpressionEngineDotEnv creates or updates the .env.php file for ExpressionEngine
func updateExpressionEngineDotEnv(app *DdevApp) (string, error) {
	// If settings management is disabled, do nothing
	if app.DisableSettingsManagement {
		return "", nil
	}

	envFilePath := app.SiteSettingsPath

	// Warn if we couldn't determine the proper path
	if envFilePath == "" {
		util.Warning("Could not determine ExpressionEngine .env.php location")
		return "", nil
	}

	// ExpressionEngine database configuration
	newEnvMap := map[string]string{
		"BASE_URL": app.GetPrimaryURL(),
		"DB_HOST":  "db",
		"DB_NAME":  "db",
		"DB_USER":  "db",
		"DB_PASS":  "db",
	}

	// Read existing .env.php file (preserves comments/structure)
	_, existingEnvText, err := ReadProjectEnvFile(envFilePath)
	// If envFilePath doesn't exist, that's not really an error, continue
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	// Write merged configuration
	err = WriteProjectEnvFile(envFilePath, newEnvMap, existingEnvText)
	if err != nil {
		return "", err
	}

	return "", nil
}
