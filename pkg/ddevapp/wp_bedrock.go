package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/util"
)

// isWPBedrockApp returns true if the app is a Roots Bedrock project.
// It checks for config/application.php, which is Bedrock's main
// configuration file and is not present in standard WordPress.
func isWPBedrockApp(app *DdevApp) bool {
	return fileutil.FileExists(filepath.Join(app.AppRoot, app.ComposerRoot, "config", "application.php"))
}

// wpBedrockPostStartAction manages the .env file for Bedrock projects,
// setting database credentials and URLs for the DDEV environment.
func wpBedrockPostStartAction(app *DdevApp) error {
	if app.DisableSettingsManagement {
		return nil
	}
	envFilePath := filepath.Join(app.AppRoot, app.ComposerRoot, ".env")
	_, envText, err := ReadProjectEnvFile(envFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("unable to read .env file: %v", err)
	}
	if os.IsNotExist(err) {
		err = fileutil.CopyFile(filepath.Join(app.AppRoot, app.ComposerRoot, ".env.example"), filepath.Join(app.AppRoot, app.ComposerRoot, ".env"))
		if err != nil {
			util.Debug("Bedrock: .env.example does not exist yet, not trying to process it")
			return nil
		}
		_, envText, err = ReadProjectEnvFile(envFilePath)
		if err != nil {
			return err
		}
	}

	envMap := map[string]string{
		"WP_HOME": app.GetPrimaryURL(),
		"WP_ENV":  "development",
	}

	// Only set database configuration if db container is not omitted
	if !slices.Contains(app.OmitContainers, "db") {
		envMap["DB_NAME"] = "db"
		envMap["DB_USER"] = "db"
		envMap["DB_PASSWORD"] = "db"
		envMap["DB_HOST"] = "db"
	}

	err = WriteProjectEnvFile(envFilePath, envMap, envText)
	if err != nil {
		return err
	}

	return nil
}

// wpBedrockConfigOverrideAction sets Bedrock-specific defaults.
// Bedrock always uses "web" as its docroot.
func wpBedrockConfigOverrideAction(app *DdevApp) error {
	if app.Docroot == "" {
		app.Docroot = "web"
	}
	return nil
}

// getWPBedrockUploadDirs returns the upload directories for Bedrock.
// Bedrock moves wp-content to app/ inside the docroot.
func getWPBedrockUploadDirs(_ *DdevApp) []string {
	return []string{"app/uploads"}
}
