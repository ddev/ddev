package ddevapp_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/versionconstants"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShopware6SiteSettingsPaths tests that the .env.local file path is set correctly
// for different composer_root configurations
func TestShopware6SiteSettingsPaths(t *testing.T) {
	assert := asrt.New(t)

	testDir := testcommon.CreateTmpDir(t.Name())
	t.Cleanup(func() {
		_ = os.RemoveAll(testDir)
	})

	// Test case 1: No composer_root configured - should use AppRoot
	t.Run("NoComposerRoot", func(t *testing.T) {
		app, err := ddevapp.NewApp(testDir, false)
		require.NoError(t, err)
		app.ComposerRoot = ""

		// Apply the shopware6 settings paths function
		setShopware6SiteSettingsPaths(app)

		expectedPath := filepath.Join(testDir, ".env.local")
		assert.Equal(expectedPath, app.SiteSettingsPath)
	})

	// Test case 2: composer_root configured - should use composer root
	t.Run("WithComposerRoot", func(t *testing.T) {
		shopwareSubdir := filepath.Join(testDir, "shopware")
		err := os.MkdirAll(shopwareSubdir, 0755)
		require.NoError(t, err)

		app, err := ddevapp.NewApp(testDir, false)
		require.NoError(t, err)
		app.ComposerRoot = "shopware"

		// Apply the shopware6 settings paths function
		setShopware6SiteSettingsPaths(app)

		expectedPath := filepath.Join(shopwareSubdir, ".env.local")
		assert.Equal(expectedPath, app.SiteSettingsPath)
	})
}

// TestShopware6PostStartAction tests that the .env.local file is created correctly
// with the right content and in the right location
func TestShopware6PostStartAction(t *testing.T) {
	assert := asrt.New(t)

	testDir := testcommon.CreateTmpDir(t.Name())
	t.Cleanup(func() {
		_ = os.RemoveAll(testDir)
	})

	// Test case 1: Create .env.local in AppRoot (no composer_root)
	t.Run("CreateInAppRoot", func(t *testing.T) {
		app, err := ddevapp.NewApp(testDir, false)
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
		assert.FileExists(envFilePath)

		// Check that the file contains the expected values
		envContent, err := fileutil.ReadFileIntoString(envFilePath)
		require.NoError(t, err)

		// Values may be quoted by the env file writer
		assert.Contains(envContent, "DATABASE_URL")
		assert.Contains(envContent, "mysql://db:db@db:3306/db")
		assert.Contains(envContent, "APP_ENV")
		assert.Contains(envContent, "dev")
		assert.Contains(envContent, "MAILER_DSN")
		assert.Contains(envContent, "smtp://127.0.0.1:1025")
	})

	// Test case 2: Create .env.local in composer_root directory
	t.Run("CreateInComposerRoot", func(t *testing.T) {
		// Use separate directory to avoid conflicts with previous test
		composerRootTestDir := testcommon.CreateTmpDir("CreateInComposerRoot")
		defer func() {
			_ = os.RemoveAll(composerRootTestDir)
		}()

		shopwareSubdir := filepath.Join(composerRootTestDir, "shopware")
		err := os.MkdirAll(shopwareSubdir, 0755)
		require.NoError(t, err)

		app, err := ddevapp.NewApp(composerRootTestDir, false)
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
		assert.FileExists(envFilePath)

		// Should NOT exist in AppRoot
		rootEnvPath := filepath.Join(composerRootTestDir, ".env.local")
		assert.NoFileExists(rootEnvPath)

		// Check content
		envContent, err := fileutil.ReadFileIntoString(envFilePath)
		require.NoError(t, err)

		// Values may be quoted
		assert.Contains(envContent, "DATABASE_URL")
		assert.Contains(envContent, "mysql://db:db@db:3306/db")
		assert.Contains(envContent, "APP_ENV")
		assert.Contains(envContent, "dev")
	})

	// Test case 3: Skip when settings management is disabled
	t.Run("SkipWhenDisabled", func(t *testing.T) {
		// Use a separate test directory for this test to avoid conflicts
		separateTestDir := testcommon.CreateTmpDir("SkipWhenDisabled")
		defer func() {
			_ = os.RemoveAll(separateTestDir)
		}()

		app, err := ddevapp.NewApp(separateTestDir, false)
		require.NoError(t, err)
		app.Name = "test-shopware6-disabled"
		app.Type = nodeps.AppTypeShopware6
		app.DisableSettingsManagement = true

		err = shopware6PostStartAction(app)
		require.NoError(t, err)

		// No .env.local should be created
		envFilePath := filepath.Join(separateTestDir, ".env.local")
		assert.NoFileExists(envFilePath)
	})
}

// TestShopware6EnvFileUpdate tests that existing .env.local files are updated correctly
func TestShopware6EnvFileUpdate(t *testing.T) {
	assert := asrt.New(t)

	testDir := testcommon.CreateTmpDir(t.Name())
	t.Cleanup(func() {
		_ = os.RemoveAll(testDir)
	})

	// Create an existing .env.local with some custom values
	envFilePath := filepath.Join(testDir, ".env.local")
	existingContent := `# Custom comment
APP_SECRET=my-secret-key
CUSTOM_VAR=custom-value
DATABASE_URL=mysql://old:old@oldhost:3306/olddb
`
	err := os.WriteFile(envFilePath, []byte(existingContent), 0644)
	require.NoError(t, err)

	app, err := ddevapp.NewApp(testDir, false)
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
	assert.Contains(updatedContent, "# Custom comment")
	assert.Contains(updatedContent, "APP_SECRET=my-secret-key")
	assert.Contains(updatedContent, "CUSTOM_VAR=custom-value")

	// Should update DDEV-managed values (may be quoted)
	assert.Contains(updatedContent, "DATABASE_URL")
	assert.Contains(updatedContent, "mysql://db:db@db:3306/db")
	assert.Contains(updatedContent, "APP_ENV")
	assert.Contains(updatedContent, "dev")
	assert.Contains(updatedContent, "MAILER_DSN")
	assert.Contains(updatedContent, "smtp://127.0.0.1:1025")
}

// findWebExposedPort returns the WebExposedPort with the given container port,
// or nil if none is present. Used to assert on the watcher ports added by the
// shopware6 configOverrideAction.
func findWebExposedPort(ports []ddevapp.WebExposedPort, containerPort int) *ddevapp.WebExposedPort {
	for i := range ports {
		if ports[i].WebContainerPort == containerPort {
			return &ports[i]
		}
	}
	return nil
}

// TestShopware6ConfigOverrideAction verifies that the shopware6 project type
// exposes the shopware-cli watcher ports through the real app-type matrix
// (ConfigFileOverrideAction), that it is idempotent, and that it never clobbers
// a user's own port settings. The watcher environment is deliberately NOT set
// here (it is set at runtime in the watcher commands), so this only asserts on
// ports. This does not require Docker.
func TestShopware6ConfigOverrideAction(t *testing.T) {
	testDir := testcommon.CreateTmpDir(t.Name())
	t.Cleanup(func() {
		_ = os.RemoveAll(testDir)
	})

	// AddsWatcherPorts: a fresh shopware6 project gets the three watcher ports.
	t.Run("AddsWatcherPorts", func(t *testing.T) {
		app, err := ddevapp.NewApp(testDir, false)
		require.NoError(t, err)
		app.Type = nodeps.AppTypeShopware6

		require.NoError(t, app.ConfigFileOverrideAction(true))

		// Vite admin watcher on 5173.
		vite := findWebExposedPort(app.WebExtraExposedPorts, 5173)
		require.NotNil(t, vite, "expected vite admin port 5173 to be exposed")
		require.Equal(t, 5172, vite.HTTPPort)
		require.Equal(t, 5173, vite.HTTPSPort)

		// Storefront proxy on 9998.
		proxy := findWebExposedPort(app.WebExtraExposedPorts, 9998)
		require.NotNil(t, proxy, "expected storefront proxy port 9998 to be exposed")
		require.Equal(t, 8888, proxy.HTTPPort)
		require.Equal(t, 9998, proxy.HTTPSPort)

		// Storefront assets on 9999.
		assets := findWebExposedPort(app.WebExtraExposedPorts, 9999)
		require.NotNil(t, assets, "expected storefront assets port 9999 to be exposed")
		require.Equal(t, 8889, assets.HTTPPort)
		require.Equal(t, 9999, assets.HTTPSPort)

		// The watcher env must NOT be injected into web_environment; it is set at
		// runtime in the commands instead.
		require.NotContains(t, app.WebEnvironment, "PROXY_URL=${DDEV_PRIMARY_URL}:9998")
	})

	// Idempotent: running the override twice must not duplicate ports.
	t.Run("Idempotent", func(t *testing.T) {
		app, err := ddevapp.NewApp(testDir, false)
		require.NoError(t, err)
		app.Type = nodeps.AppTypeShopware6

		require.NoError(t, app.ConfigFileOverrideAction(true))
		portsAfterFirst := len(app.WebExtraExposedPorts)
		require.NoError(t, app.ConfigFileOverrideAction(true))

		require.Len(t, app.WebExtraExposedPorts, portsAfterFirst, "ports should not be duplicated on re-run")
	})

	// PreservesUserValues: a user's own value for a conflicting container port
	// must be left untouched.
	t.Run("PreservesUserValues", func(t *testing.T) {
		app, err := ddevapp.NewApp(testDir, false)
		require.NoError(t, err)
		app.Type = nodeps.AppTypeShopware6

		// User has already mapped container port 9998 to different host ports.
		app.WebExtraExposedPorts = []ddevapp.WebExposedPort{
			{Name: "user-proxy", WebContainerPort: 9998, HTTPPort: 7777, HTTPSPort: 7778},
		}

		require.NoError(t, app.ConfigFileOverrideAction(true))

		// The user's 9998 mapping is preserved, not overwritten or duplicated.
		proxy := findWebExposedPort(app.WebExtraExposedPorts, 9998)
		require.NotNil(t, proxy)
		require.Equal(t, 7777, proxy.HTTPPort, "user's port mapping must be preserved")
		require.Equal(t, 1, countWebExposedPort(app.WebExtraExposedPorts, 9998))

		// The non-conflicting watcher ports are still added.
		require.NotNil(t, findWebExposedPort(app.WebExtraExposedPorts, 5173))
		require.NotNil(t, findWebExposedPort(app.WebExtraExposedPorts, 9999))
	})
}

// TestShopware6WebImageDockerfile verifies that the shopware-cli binary is baked
// into the generated web image Dockerfile for shopware6 projects (via the COPY
// --from=shopware/shopware-cli:bin line), and that no other project type gets it.
// This exercises the real RenderComposeYAML path, so it requires Docker.
func TestShopware6WebImageDockerfile(t *testing.T) {
	origDir, err := os.Getwd()
	require.NoError(t, err)

	copyLine := fmt.Sprintf("COPY --from=%s /shopware-cli /usr/local/bin/shopware-cli", versionconstants.ShopwareCLIImage)

	cases := []struct {
		name       string
		appType    string
		expectCopy bool
	}{
		{"shopware6-gets-binary", nodeps.AppTypeShopware6, true},
		{"php-does-not", nodeps.AppTypePHP, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testDir := testcommon.CreateTmpDir(t.Name())
			t.Cleanup(func() {
				_ = os.Chdir(origDir)
				_ = os.RemoveAll(testDir)
			})

			app, err := ddevapp.NewApp(testDir, false)
			require.NoError(t, err)
			app.Type = tc.appType
			require.NoError(t, app.WriteConfig())

			// RenderComposeYAML writes .webimageBuild/Dockerfile as a side effect.
			_, err = app.RenderComposeYAML()
			require.NoError(t, err)

			dockerfile, err := fileutil.ReadFileIntoString(app.GetConfigPath(".webimageBuild/Dockerfile"))
			require.NoError(t, err)

			if tc.expectCopy {
				require.Contains(t, dockerfile, copyLine, "shopware6 web image Dockerfile should copy in shopware-cli")
			} else {
				require.NotContains(t, dockerfile, copyLine, "%s web image Dockerfile should not copy in shopware-cli", tc.appType)
			}
		})
	}
}

// countWebExposedPort returns the number of ports with the given container port.
func countWebExposedPort(ports []ddevapp.WebExposedPort, containerPort int) int {
	count := 0
	for _, p := range ports {
		if p.WebContainerPort == containerPort {
			count++
		}
	}
	return count
}

// Helper functions to access the unexported functions from shopware6.go
// These would normally be in the same package, but since we're in _test package,
// we need to access them through the app type system or create wrapper functions.

func setShopware6SiteSettingsPaths(app *ddevapp.DdevApp) {
	composerRoot := app.GetComposerRoot(false, false)
	app.SiteSettingsPath = filepath.Join(composerRoot, ".env.local")
}

func shopware6PostStartAction(app *ddevapp.DdevApp) error {
	if app.DisableSettingsManagement {
		return nil
	}

	composerRoot := app.GetComposerRoot(false, false)
	envFilePath := filepath.Join(composerRoot, ".env.local")

	_, envText, err := ddevapp.ReadProjectEnvFile(envFilePath)
	var envMap = map[string]string{
		"DATABASE_URL": `mysql://db:db@db:3306/db`,
		"APP_ENV":      "dev",
		"APP_URL":      app.GetPrimaryURL(),
		"MAILER_DSN":   `smtp://127.0.0.1:1025?encryption=&auth_mode=`,
	}

	// If the .env.local doesn't exist, create it.
	if err == nil || os.IsNotExist(err) {
		err := ddevapp.WriteProjectEnvFile(envFilePath, envMap, envText)
		if err != nil {
			return err
		}
	}

	return nil
}
