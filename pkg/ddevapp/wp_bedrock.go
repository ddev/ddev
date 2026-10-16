package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/util"
)

// isWPBedrockApp returns true if the app is a Roots Bedrock project.
// It checks for config/application.php, which is Bedrock's main
// configuration file and is not present in standard WordPress.
func isWPBedrockApp(app *DdevApp) bool {
	return fileutil.FileExists(filepath.Join(app.AppRoot, app.ComposerRoot, "config", "application.php"))
}

// wpBedrockPostStartAction checks to see if the .env file is set up
func wpBedrockPostStartAction(app *DdevApp) error {
	// We won't touch env if disable_settings_management: true
	if app.DisableSettingsManagement {
		return nil
	}
	envFilePath := filepath.Join(app.AppRoot, app.ComposerRoot, ".env")
	envText, err := ReadEnvFileText(envFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("unable to read .env file: %v", err)
	}
	setSaltAndKeys := false
	if os.IsNotExist(err) {
		err = fileutil.CopyFile(filepath.Join(app.AppRoot, app.ComposerRoot, ".env.example"), filepath.Join(app.AppRoot, app.ComposerRoot, ".env"))
		if err != nil {
			util.Debug("Bedrock: .env.example does not exist yet, not trying to process it")
			return nil
		}
		envText, err = ReadEnvFileText(envFilePath)
		if err != nil {
			return err
		}
		setSaltAndKeys = true
	}
	envMap := map[string]string{
		"WP_ENV":      "development",
		"WP_HOME":     app.GetPrimaryURL(),
		"WP_SITEURL":  app.GetPrimaryURL() + "/wp",
		"DB_NAME":     "db",
		"DB_USER":     "db",
		"DB_PASSWORD": "db",
		"DB_HOST":     "db",
	}

	if setSaltAndKeys {
		envMap["AUTH_KEY"] = util.RandString(64)
		envMap["SECURE_AUTH_KEY"] = util.RandString(64)
		envMap["LOGGED_IN_KEY"] = util.RandString(64)
		envMap["NONCE_KEY"] = util.RandString(64)
		envMap["AUTH_SALT"] = util.RandString(64)
		envMap["SECURE_AUTH_SALT"] = util.RandString(64)
		envMap["LOGGED_IN_SALT"] = util.RandString(64)
		envMap["NONCE_SALT"] = util.RandString(64)
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
