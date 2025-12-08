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

	// Check if executable
	info, err := os.Stat(scriptPath)
	if err != nil {
		return "", err
	}
	if info.Mode()&0111 == 0 {
		return "", fmt.Errorf("share provider '%s' is not executable (chmod +x %s)", providerName, scriptPath)
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

	// Determine args: command-line override takes precedence over config
	var args string
	if providerArgsOverride != "" {
		args = providerArgsOverride
	} else {
		// Use provider-specific config
		switch providerName {
		case "ngrok":
			args = app.ShareNgrokArgs
			if args == "" && app.NgrokArgs != "" {
				args = app.NgrokArgs // Backward compatibility
			}
		case "cloudflared":
			args = app.ShareCloudflaredArgs
		}
	}

	// Set the generic DDEV_SHARE_ARGS for any provider to use
	if args != "" {
		env = append(env, fmt.Sprintf("DDEV_SHARE_ARGS=%s", args))
	}

	// Also set provider-specific env var for backward compatibility
	if args != "" {
		upperProvider := strings.ToUpper(providerName)
		env = append(env, fmt.Sprintf("DDEV_SHARE_%s_ARGS=%s", upperProvider, args))
	}

	return env
}
