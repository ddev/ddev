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
	scriptPath := app.GetConfigPath("share-providers", providerName+".sh")

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
func (app *DdevApp) GetShareProviderEnvironment(providerName string) []string {
	env := os.Environ()

	// Add DDEV_LOCAL_URL
	localURL := app.GetWebContainerDirectHTTPURL()
	env = append(env, fmt.Sprintf("DDEV_LOCAL_URL=%s", localURL))

	// Add provider-specific args
	switch providerName {
	case "ngrok":
		args := app.ShareNgrokArgs
		if args == "" && app.NgrokArgs != "" {
			args = app.NgrokArgs // Backward compatibility
		}
		if args != "" {
			env = append(env, fmt.Sprintf("DDEV_SHARE_NGROK_ARGS=%s", args))
		}
	case "cloudflared":
		if app.ShareCloudflaredArgs != "" {
			env = append(env, fmt.Sprintf("DDEV_SHARE_CLOUDFLARED_ARGS=%s", app.ShareCloudflaredArgs))
		}
	}

	return env
}