package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ddev/ddev/pkg/fileutil"
)

// GetShareProviderScript returns the absolute path to a share provider script
func (app *DdevApp) GetShareProviderScript(providerName string) (string, error) {
	scriptPath := app.GetConfigPath(filepath.Join("share-providers", providerName+".sh"))

	if !fileutil.FileExists(scriptPath) {
		return "", fmt.Errorf("share provider '%s' not found at %s", providerName, scriptPath)
	}

	return scriptPath, nil
}

// ListShareProviders returns all available share provider names
func (app *DdevApp) ListShareProviders() ([]string, error) {
	providerDir := app.GetConfigPath("share-providers")
	if !fileutil.IsDirectory(providerDir) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(providerDir)
	if err != nil {
		return nil, err
	}

	var providers []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sh" {
			name := strings.TrimSuffix(entry.Name(), ".sh")
			providers = append(providers, name)
		}
	}

	return providers, nil
}

// GetShareProviderEnvironment builds environment variables for provider script
// providerArgsOverride allows command-line args to override config file args
func (app *DdevApp) GetShareProviderEnvironment(providerName string, providerArgsOverride string) []string {
	env := os.Environ()

	// Add DDEV_LOCAL_URL
	localURL := app.GetWebContainerDirectHTTPURL()
	env = append(env, fmt.Sprintf("DDEV_LOCAL_URL=%s", localURL))

	// Determine args with priority:
	// 1. Command-line override (--provider-args)
	// 2. Generic config (share_provider_args)
	// 3. Legacy ngrok_args (for ngrok only, backward compatibility)
	var args string
	if providerArgsOverride != "" {
		args = providerArgsOverride
	} else if app.ShareProviderArgs != "" {
		args = app.ShareProviderArgs
	} else if providerName == "ngrok" && app.NgrokArgs != "" {
		// Legacy backward compatibility for ngrok_args
		args = app.NgrokArgs
	}

	// Set the generic DDEV_SHARE_ARGS for provider scripts to use
	if args != "" {
		env = append(env, fmt.Sprintf("DDEV_SHARE_ARGS=%s", args))
	}

	return env
}
