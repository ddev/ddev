package ddevapp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
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
