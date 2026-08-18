package ddevapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
)

// TestShopware6SiteSettingsPaths tests that the .env.local file path is set correctly
// for different composer_root configurations
func TestShopware6SiteSettingsPaths(t *testing.T) {
	testDir := t.TempDir()

	// Test case 1: No composer_root configured - should use AppRoot
	t.Run("NoComposerRoot", func(t *testing.T) {
		app, err := NewApp(testDir, false)
		require.NoError(t, err)
		app.ComposerRoot = ""

		// Apply the real shopware6 settings paths function
		setShopware6SiteSettingsPaths(app)

		expectedPath := filepath.Join(testDir, ".env.local")
		require.Equal(t, expectedPath, app.SiteSettingsPath)
	})

	// Test case 2: composer_root configured - should use composer root
	t.Run("WithComposerRoot", func(t *testing.T) {
		shopwareSubdir := filepath.Join(testDir, "shopware")
		err := os.MkdirAll(shopwareSubdir, 0755)
		require.NoError(t, err)

		app, err := NewApp(testDir, false)
		require.NoError(t, err)
		app.ComposerRoot = "shopware"

		// Apply the real shopware6 settings paths function
		setShopware6SiteSettingsPaths(app)

		expectedPath := filepath.Join(shopwareSubdir, ".env.local")
		require.Equal(t, expectedPath, app.SiteSettingsPath)
	})
}

// TestShopware6PostStartAction tests that the .env.local file is created correctly
// with the right content and in the right location
func TestShopware6PostStartAction(t *testing.T) {
	testDir := t.TempDir()

	// Test case 1: Create .env.local in AppRoot (no composer_root)
	t.Run("CreateInAppRoot", func(t *testing.T) {
		app, err := NewApp(testDir, false)
		require.NoError(t, err)
		app.Name = "test-shopware6"
		app.Type = nodeps.AppTypeShopware6
		app.ComposerRoot = ""
		app.DisableSettingsManagement = false

		// Set up app for testing
		app.RouterHTTPSPort = "443"
		app.RouterHTTPPort = "80"

		err = shopware6PostStartAction(app)
		require.NoError(t, err)

		envFilePath := filepath.Join(testDir, ".env.local")
		require.FileExists(t, envFilePath)

		// Check that the file contains the expected values
		envContent, err := fileutil.ReadFileIntoString(envFilePath)
		require.NoError(t, err)

		// Values may be quoted by the env file writer
		require.Contains(t, envContent, "DATABASE_URL")
		require.Contains(t, envContent, "mysql://db:db@db:3306/db")
		require.Contains(t, envContent, "APP_ENV")
		require.Contains(t, envContent, "dev")
		require.Contains(t, envContent, "MAILER_DSN")
		require.Contains(t, envContent, "smtp://127.0.0.1:1025")
	})

	// Test case 2: Create .env.local in composer_root directory
	t.Run("CreateInComposerRoot", func(t *testing.T) {
		// Use separate directory to avoid conflicts with previous test
		composerRootTestDir := t.TempDir()

		shopwareSubdir := filepath.Join(composerRootTestDir, "shopware")
		err := os.MkdirAll(shopwareSubdir, 0755)
		require.NoError(t, err)

		app, err := NewApp(composerRootTestDir, false)
		require.NoError(t, err)
		app.Name = "test-shopware6-composerroot"
		app.Type = nodeps.AppTypeShopware6
		app.ComposerRoot = "shopware"
		app.DisableSettingsManagement = false

		// Set up app for testing
		app.RouterHTTPSPort = "443"
		app.RouterHTTPPort = "80"

		err = shopware6PostStartAction(app)
		require.NoError(t, err)

		// The .env.local should be created in the shopware subdirectory
		envFilePath := filepath.Join(shopwareSubdir, ".env.local")
		require.FileExists(t, envFilePath)

		// Should NOT exist in AppRoot
		rootEnvPath := filepath.Join(composerRootTestDir, ".env.local")
		require.NoFileExists(t, rootEnvPath)

		// Check content
		envContent, err := fileutil.ReadFileIntoString(envFilePath)
		require.NoError(t, err)

		// Values may be quoted
		require.Contains(t, envContent, "DATABASE_URL")
		require.Contains(t, envContent, "mysql://db:db@db:3306/db")
		require.Contains(t, envContent, "APP_ENV")
		require.Contains(t, envContent, "dev")
	})

	// Test case 3: Skip when settings management is disabled
	t.Run("SkipWhenDisabled", func(t *testing.T) {
		// Use a separate test directory for this test to avoid conflicts
		separateTestDir := t.TempDir()

		app, err := NewApp(separateTestDir, false)
		require.NoError(t, err)
		app.Name = "test-shopware6-disabled"
		app.Type = nodeps.AppTypeShopware6
		app.DisableSettingsManagement = true

		err = shopware6PostStartAction(app)
		require.NoError(t, err)

		// No .env.local should be created
		envFilePath := filepath.Join(separateTestDir, ".env.local")
		require.NoFileExists(t, envFilePath)
	})
}

// TestShopware6EnvFileUpdate tests that existing .env.local files are updated correctly
func TestShopware6EnvFileUpdate(t *testing.T) {
	testDir := t.TempDir()

	// Create an existing .env.local with some custom values
	envFilePath := filepath.Join(testDir, ".env.local")
	existingContent := `# Custom comment
APP_SECRET=my-secret-key
CUSTOM_VAR=custom-value
DATABASE_URL=mysql://old:old@oldhost:3306/olddb
`
	err := os.WriteFile(envFilePath, []byte(existingContent), 0644)
	require.NoError(t, err)

	app, err := NewApp(testDir, false)
	require.NoError(t, err)
	app.Name = "test-shopware6-update"
	app.Type = nodeps.AppTypeShopware6
	app.DisableSettingsManagement = false

	// Set up app for testing
	app.RouterHTTPSPort = "443"
	app.RouterHTTPPort = "80"

	err = shopware6PostStartAction(app)
	require.NoError(t, err)

	// Read the updated content
	updatedContent, err := fileutil.ReadFileIntoString(envFilePath)
	require.NoError(t, err)

	// Should preserve custom values and comments
	require.Contains(t, updatedContent, "# Custom comment")
	require.Contains(t, updatedContent, "APP_SECRET=my-secret-key")
	require.Contains(t, updatedContent, "CUSTOM_VAR=custom-value")

	// Should update DDEV-managed values (may be quoted)
	require.Contains(t, updatedContent, "DATABASE_URL")
	require.Contains(t, updatedContent, "mysql://db:db@db:3306/db")
	require.Contains(t, updatedContent, "APP_ENV")
	require.Contains(t, updatedContent, "dev")
	require.Contains(t, updatedContent, "MAILER_DSN")
	require.Contains(t, updatedContent, "smtp://127.0.0.1:1025")
}

// TestShopware6ConfigOverrideAction verifies that the shopware6 project type
// exposes the shopware-cli watcher ports through the real app-type matrix
// (ConfigFileOverrideAction), that it is idempotent, and that it never clobbers
// a user's own port settings. The watcher environment is deliberately NOT set
// here (it is set at runtime in the watcher commands), so this only asserts on
// ports. This does not require Docker.
func TestShopware6ConfigOverrideAction(t *testing.T) {
	testDir := t.TempDir()

	// The exact set of watcher ports shopware6ConfigOverrideAction is expected to
	// add. Kept in sync with the production watcherPorts slice in shopware6.go.
	expectedWatcherPorts := []WebExposedPort{
		{Name: "shopware-vite-admin", WebContainerPort: 5173, HTTPPort: 19172, HTTPSPort: 5173},
		{Name: "shopware-storefront-proxy", WebContainerPort: 9998, HTTPPort: 19998, HTTPSPort: 9998},
		{Name: "shopware-storefront-assets", WebContainerPort: 9999, HTTPPort: 19999, HTTPSPort: 9999},
	}

	// AddsWatcherPorts: a fresh shopware6 project gets exactly the watcher ports.
	t.Run("AddsWatcherPorts", func(t *testing.T) {
		app, err := NewApp(testDir, false)
		require.NoError(t, err)
		app.Type = nodeps.AppTypeShopware6

		require.NoError(t, app.ConfigFileOverrideAction(true))

		require.ElementsMatch(t, expectedWatcherPorts, app.WebExtraExposedPorts)

		// The watcher env must NOT be injected into web_environment; it is set at
		// runtime in the commands instead.
		require.NotContains(t, app.WebEnvironment, "PROXY_URL=${DDEV_PRIMARY_URL}:9998")
	})

	// Idempotent: running the override twice must not duplicate ports.
	t.Run("Idempotent", func(t *testing.T) {
		app, err := NewApp(testDir, false)
		require.NoError(t, err)
		app.Type = nodeps.AppTypeShopware6

		require.NoError(t, app.ConfigFileOverrideAction(true))
		require.NoError(t, app.ConfigFileOverrideAction(true))

		require.ElementsMatch(t, expectedWatcherPorts, app.WebExtraExposedPorts,
			"ports should not be duplicated on re-run")
	})

	// PreservesUserValues: a user's own value for a conflicting container port
	// must be left untouched.
	t.Run("PreservesUserValues", func(t *testing.T) {
		app, err := NewApp(testDir, false)
		require.NoError(t, err)
		app.Type = nodeps.AppTypeShopware6

		// User has already mapped container port 9998 to different host ports.
		userProxy := WebExposedPort{Name: "user-proxy", WebContainerPort: 9998, HTTPPort: 7777, HTTPSPort: 7778}
		app.WebExtraExposedPorts = []WebExposedPort{userProxy}

		require.NoError(t, app.ConfigFileOverrideAction(true))

		// The user's 9998 mapping is preserved, not overwritten or duplicated,
		// and the non-conflicting watcher ports are still added.
		require.ElementsMatch(t, []WebExposedPort{
			userProxy,
			{Name: "shopware-vite-admin", WebContainerPort: 5173, HTTPPort: 19172, HTTPSPort: 5173},
			{Name: "shopware-storefront-assets", WebContainerPort: 9999, HTTPPort: 19999, HTTPSPort: 9999},
		}, app.WebExtraExposedPorts)
	})
}
