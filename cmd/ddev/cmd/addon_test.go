package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/stretchr/testify/require"

	asrt "github.com/stretchr/testify/assert"
)

// TestCmdAddon tests various `ddev add-on` commands.
func TestCmdAddon(t *testing.T) {
	if os.Getenv("DDEV_RUN_GET_TESTS") != "true" {
		t.Skip("Skipping because DDEV_RUN_GET_TESTS is not set")
	}
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err = exec.RunHostCommand(DdevBin, "add-on", "remove", "memcached")
		assert.NoError(err)
		_, err = exec.RunHostCommand(DdevBin, "add-on", "remove", "example")
		assert.NoError(err)

		err = os.Chdir(origDir)
		assert.NoError(err)
		_ = os.RemoveAll(filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/web/global-touched"))
		_ = os.RemoveAll(filepath.Join(globalconfig.GetGlobalDdevDir(), "file-with-no-ddev-generated.txt"))
		_ = os.RemoveAll(filepath.Join(globalconfig.GetGlobalDdevDir(), "globalextras"))
	})

	// Make sure 'ddev add-on list' works first
	out, err := exec.RunHostCommand(DdevBin, "add-on", "list")
	assert.NoError(err, "failed ddev add-on list: %v (%s)", err, out)
	assert.Contains(out, "ddev/ddev-memcached")

	tarballFile := filepath.Join(origDir, "testdata", t.Name(), "ddev-memcached.tar.gz")

	// Test with many input styles
	for _, arg := range []string{
		"ddev/ddev-memcached",
		"https://github.com/ddev/ddev-memcached/archive/refs/tags/v1.1.1.tar.gz",
		tarballFile} {
		out, err := exec.RunHostCommand(DdevBin, "add-on", "get", arg)
		assert.NoError(err, "failed ddev add-on get %s", arg)
		assert.Contains(out, "Installed ")
		assert.FileExists(app.GetConfigPath("docker-compose.memcached.yaml"))
	}

	// Test with a directory-path input
	exampleDir := filepath.Join(origDir, "testdata", t.Name(), "example-repo")
	err = fileutil.TemplateStringToFile("no signature here", nil, app.GetConfigPath("file-with-no-ddev-generated.txt"))
	require.NoError(t, err)
	err = fileutil.TemplateStringToFile("no signature here", nil, filepath.Join(globalconfig.GetGlobalDdevDir(), "file-with-no-ddev-generated.txt"))
	require.NoError(t, err)

	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", exampleDir)
	assert.NoError(err, "output=%s", out)
	assert.FileExists(app.GetConfigPath("i-have-been-touched"))
	assert.FileExists(app.GetConfigPath("docker-compose.example.yaml"))
	exists, err := fileutil.FgrepStringInFile(app.GetConfigPath("file-with-no-ddev-generated.txt"), "install should result in a warning")
	require.NoError(t, err)
	assert.False(exists, "the file with no ddev-generated.txt should not have been replaced")

	assert.FileExists(filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/web/global-touched"))
	assert.FileExists(filepath.Join(globalconfig.GetGlobalDdevDir(), "globalextras/okfile.txt"))

	exists, err = fileutil.FgrepStringInFile(filepath.Join(globalconfig.GetGlobalDdevDir(), "file-with-no-ddev-generated.txt"), "install should result in a warning")
	require.NoError(t, err)
	assert.False(exists, "the file with no ddev-generated.txt should not have been replaced")

	assert.Contains(out, fmt.Sprintf("NOT overwriting %s", app.GetConfigPath("file-with-no-ddev-generated.txt")))
	assert.Contains(out, fmt.Sprintf("NOT overwriting %s", filepath.Join(globalconfig.GetGlobalDdevDir(), "file-with-no-ddev-generated.txt")))
}

// TestCmdAddonInstalled tests `ddev add-on list --installed` and `ddev add-on remove`
func TestCmdAddonInstalled(t *testing.T) {
	if os.Getenv("DDEV_RUN_GET_TESTS") != "true" {
		t.Skip("Skipping because DDEV_RUN_GET_TESTS is not set")
	}
	origDdevDebug := os.Getenv("DDEV_DEBUG")
	_ = os.Unsetenv("DDEV_DEBUG")
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err = exec.RunHostCommand(DdevBin, "add-on", "remove", "memcached")
		assert.NoError(err)
		_, err = exec.RunHostCommand(DdevBin, "add-on", "remove", "redis")
		assert.NoError(err)

		err = os.Chdir(origDir)
		assert.NoError(err)
		_ = os.RemoveAll(filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/web/global-touched"))
		_ = os.Setenv("DDEV_DEBUG", origDdevDebug)
	})

	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-memcached", "--json-output")
	require.NoError(t, err, "failed ddev add-on get ddev/ddev-memcached: %v (output='%s')", err, out)

	memcachedManifest := getManifestFromLogs(t, out)
	require.NoError(t, err)

	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-redis", "--json-output")
	require.NoError(t, err, "failed ddev add-on get ddev/ddev-redis: %v (output='%s')", err, out)

	redisManifest := getManifestFromLogs(t, out)
	require.NoError(t, err)

	installedOutput, err := exec.RunHostCommand(DdevBin, "add-on", "list", "--installed", "--json-output")
	require.NoError(t, err, "failed ddev add-on list --installed --json-output: %v (output='%s')", err, installedOutput)
	installedManifests := getManifestMapFromLogs(t, installedOutput)

	require.NotEmptyf(t, memcachedManifest["Version"], "memcached manifest is empty: %v", memcachedManifest)
	require.NotEmptyf(t, redisManifest["Version"], "redis manifest is empty: %v", redisManifest)

	assert.Equal(memcachedManifest["Version"], installedManifests["memcached"]["Version"])
	assert.Equal(redisManifest["Version"], installedManifests["redis"]["Version"])

	// Now try the remove using other techniques (full repo name, partial repo name)
	for _, n := range []string{"ddev/ddev-redis", "ddev-redis", "redis"} {
		out, err = exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-redis", "--json-output")
		require.NoError(t, err, "failed ddev add-on get %s: %v (output='%s')", n, err, out)
		out, err = exec.RunHostCommand(DdevBin, "add-on", "remove", n)
		require.NoError(t, err, "unable to ddev add-on remove %s: %v, output='%s'", n, err, out)
	}
	// Now make sure we put it back so it can be removed in cleanu
	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-redis")
	assert.NoError(err, "unable to ddev add-on get redis: %v, output='%s'", err, out)
}

// TestCmdAddonProjectFlag tests the `--project` flag in `ddev add-on` subcommands
func TestCmdAddonProjectFlag(t *testing.T) {
	if os.Getenv("DDEV_RUN_GET_TESTS") != "true" {
		t.Skip("Skipping because DDEV_RUN_GET_TESTS is not set")
	}
	origDdevDebug := os.Getenv("DDEV_DEBUG")
	_ = os.Unsetenv("DDEV_DEBUG")
	assert := asrt.New(t)

	site := TestSites[0]
	// Explicitly don't chdir to the project

	t.Cleanup(func() {
		_, err := exec.RunHostCommand(DdevBin, "add-on", "remove", "redis", "--project", site.Name)
		assert.NoError(err)
		_ = os.RemoveAll(filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/web/global-touched"))
		_ = os.Setenv("DDEV_DEBUG", origDdevDebug)
	})

	// Install the add-on using the `--project` flag
	out, err := exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-redis", "--project", site.Name, "--json-output")
	require.NoError(t, err, "failed ddev add-on get ddev/ddev-redis --project %s --json-output: %v (output='%s')", site.Name, err, out)

	redisManifest := getManifestFromLogs(t, out)
	require.NoError(t, err)

	installedOutput, err := exec.RunHostCommand(DdevBin, "add-on", "list", "--installed", "--project", site.Name, "--json-output")
	require.NoError(t, err, "failed ddev add-on list --installed --project %s --json-output: %v (output='%s')", site.Name, err, installedOutput)
	installedManifests := getManifestMapFromLogs(t, installedOutput)

	require.NotEmptyf(t, redisManifest["Version"], "redis manifest is empty: %v", redisManifest)
	assert.Equal(redisManifest["Version"], installedManifests["redis"]["Version"])

	// Remove the add-on using the `--project` flag
	out, err = exec.RunHostCommand(DdevBin, "add-on", "remove", "ddev/ddev-redis", "--project", site.Name)
	require.NoError(t, err, "unable to ddev add-on remove ddev/ddev-redis --project %s: %v, output='%s'", site.Name, err, out)

	// Now make sure we put it back so it can be removed in cleanup
	out, err = exec.RunHostCommand(DdevBin, "add-on", "get", "ddev/ddev-redis", "--project", site.Name)
	assert.NoError(err, "unable to ddev add-on get ddev/ddev-redis --project %s: %v, output='%s'", site.Name, err, out)
}

// getManifestFromLogs returns the manifest built from 'raw' section of
// ddev add-on get <project> -j output
func getManifestFromLogs(t *testing.T, jsonOut string) map[string]interface{} {
	assert := asrt.New(t)

	logItems, err := unmarshalJSONLogs(jsonOut)
	require.NoError(t, err)
	data := logItems[len(logItems)-1]
	assert.EqualValues(data["level"], "info")

	m, ok := data["raw"].(map[string]interface{})
	require.True(t, ok)
	return m
}

// getManifestMapFromLogs returns the manifest array built from 'raw' section of
// ddev add-on list --installed -j output
func getManifestMapFromLogs(t *testing.T, jsonOut string) map[string]map[string]interface{} {
	assert := asrt.New(t)

	logItems, err := unmarshalJSONLogs(jsonOut)
	require.NoError(t, err)
	data := logItems[len(logItems)-1]
	assert.EqualValues(data["level"], "info")

	m, ok := data["raw"].([]interface{})
	require.True(t, ok)
	masterMap := map[string]map[string]interface{}{}
	for _, item := range m {
		itemMap := item.(map[string]interface{})
		masterMap[itemMap["Name"].(string)] = itemMap
	}
	return masterMap
}

// TestCmdAddonPHP tests the new PHP execution functionality in addons
func TestCmdAddonPHP(t *testing.T) {
	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)

	t.Cleanup(func() {
		addonList, err := exec.RunHostCommand("bash", "-c", fmt.Sprintf("%s add-on list --installed -j | docker run -i --rm ddev/ddev-utilities jq -r .raw.[].Name", DdevBin))
		require.NoError(t, err)
		addonList = strings.TrimSpace(addonList)
		addons := strings.Split(addonList, "\n")
		for _, item := range addons {
			_, err = exec.RunHostCommand(DdevBin, "add-on", "remove", item)
			require.NoError(t, err)
		}

		_ = os.Chdir(origDir)
	})

	// Test basic PHP addon
	t.Run("BasicPHPAddon", func(t *testing.T) {
		basicAddonDir := filepath.Join(origDir, "testdata", t.Name())
		out, err := exec.RunHostCommand(DdevBin, "add-on", "get", basicAddonDir, "--verbose")
		require.NoError(t, err, "failed to install basic PHP addon: %v, output: %s", err, out)

		// Check that PHP output is present
		require.Contains(t, out, "PHP: Setting up project:")
		require.Contains(t, out, "PHP: Config directory accessible")
		require.Contains(t, out, "PHP: Created test file")
		require.Contains(t, out, "Bash: This is a regular bash action after PHP")
		require.Contains(t, out, "PHP: Post-install PHP action executed")

		// Check that descriptions are displayed
		require.Contains(t, out, "👍  Initialize basic PHP addon")
		require.Contains(t, out, "👍  Execute bash validation step")
		require.Contains(t, out, "👍  Finalize basic PHP addon setup")

		// Verify the PHP-created file exists
		require.FileExists(t, app.GetConfigPath("php-test-created.txt"))
	})

	// Test complex PHP addon with yaml_read_files
	t.Run("ComplexPHPAddon", func(t *testing.T) {
		complexAddonDir := filepath.Join(origDir, "testdata", t.Name())

		// First create the test config file
		testConfigContent := `database:
  version: "8.0"
services:
  redis:
    enabled: true`
		err := os.WriteFile(app.GetConfigPath("test-config.yaml"), []byte(testConfigContent), 0644)
		require.NoError(t, err)

		out, err := exec.RunHostCommand(DdevBin, "add-on", "get", complexAddonDir, "--verbose")
		require.NoError(t, err, "failed to install complex PHP addon: %v, output: %s", err, out)

		// Check that PHP processed the YAML config
		require.Contains(t, out, "PHP: Database version from config: 8.0")
		require.Contains(t, out, "PHP: Service redis configured")
		require.Contains(t, out, "PHP: Generated docker-compose.complex-php-addon.yaml")

		// Check that descriptions are displayed
		require.Contains(t, out, "👍  Process complex PHP addon configuration")

		// Verify the generated docker-compose file exists and has expected content
		composePath := app.GetConfigPath("docker-compose.complex-php-addon.yaml")
		require.FileExists(t, composePath)

		content, err := os.ReadFile(composePath)
		require.NoError(t, err)
		require.Contains(t, string(content), "complex-php-addon")
		require.Contains(t, string(content), "PROJECT_NAME")
		require.Contains(t, string(content), "PROJECT_TYPE")

		// Clean up test config
		_ = os.Remove(app.GetConfigPath("test-config.yaml"))
	})

	// Test mixed bash/PHP addon
	t.Run("MixedAddon", func(t *testing.T) {
		mixedAddonDir := filepath.Join(origDir, "testdata", t.Name())
		out, err := exec.RunHostCommand(DdevBin, "add-on", "get", mixedAddonDir, "--verbose")
		require.NoError(t, err, "failed to install mixed addon: %v, output: %s", err, out)

		// Check that both bash and PHP actions executed in correct order
		require.Contains(t, out, "Bash: Starting mixed addon installation")
		require.Contains(t, out, "PHP: Mixed addon PHP action")
		require.Contains(t, out, "Bash: Continuing after PHP")
		require.Contains(t, out, fmt.Sprintf("PHP: Project name is %s", app.Name))
		require.Contains(t, out, "Bash: Final bash action")

		// Check that PHP action descriptions are displayed
		require.Contains(t, out, "👍  Start mixed addon installation")
		require.Contains(t, out, "👍  Execute first PHP action")
		require.Contains(t, out, "👍  Continue with bash processing")
		require.Contains(t, out, "👍  Read project configuration")
		require.Contains(t, out, "👍  Complete mixed addon installation")
	})

	// Test custom PHP image
	t.Run("CustomImageAddon", func(t *testing.T) {
		customImageAddonDir := filepath.Join(origDir, "testdata", t.Name())
		out, err := exec.RunHostCommand(DdevBin, "add-on", "get", customImageAddonDir, "--verbose")
		require.NoError(t, err, "failed to install custom image addon: %v, output: %s", err, out)

		// Check that custom PHP image (8.1-cli-alpine) was used
		require.Contains(t, out, "PHP: Running on 8.1")
		require.Contains(t, out, "PHP: OS info:")
		require.Contains(t, out, fmt.Sprintf("PHP: Custom image working for project: %s", app.Name))

		// Check that PHP action description is displayed
		require.Contains(t, out, "👍  Testing custom PHP image")
	})

	// Test Varnish PHP addon - demonstrates real-world addon conversion
	t.Run("VarnishPHPAddon", func(t *testing.T) {
		varnishAddonDir := filepath.Join(origDir, "testdata", t.Name())

		out, err := exec.RunHostCommand(DdevBin, "add-on", "get", varnishAddonDir, "--verbose")
		require.NoError(t, err, "failed to install varnish PHP addon: %v, output: %s", err, out)

		// Check that PHP processed the DDEV config and handled installation
		require.Contains(t, out, fmt.Sprintf("PHP: Installing Varnish for project: %s", app.Name))
		require.Contains(t, out, "PHP: Static files (docker-compose.varnish.yaml, varnish/, commands/) will be installed")
		require.Contains(t, out, "PHP: Generated docker-compose.varnish_extras.yaml")
		require.Contains(t, out, "PHP: Varnish installation complete!")

		// Check that descriptions are displayed
		require.Contains(t, out, "👍  Initialize Varnish addon installation")
		require.Contains(t, out, "👍  Complete Varnish configuration")

		// Verify the generated docker-compose file exists and has expected content
		varnishComposePath := app.GetConfigPath("docker-compose.varnish.yaml")
		require.FileExists(t, varnishComposePath)

		content, err := os.ReadFile(varnishComposePath)
		require.NoError(t, err)
		require.Contains(t, string(content), "ddev-${DDEV_SITENAME}-varnish")
		require.Contains(t, string(content), "varnish:6.0")
		require.Contains(t, string(content), "./varnish:/etc/varnish")

		// Verify varnish directory and VCL file were created
		varnishDir := app.GetConfigPath("varnish")
		require.DirExists(t, varnishDir)

		vclPath := app.GetConfigPath("varnish/default.vcl")
		require.FileExists(t, vclPath)

		vclContent, err := os.ReadFile(vclPath)
		require.NoError(t, err)
		require.Contains(t, string(vclContent), "#ddev-generated")
		require.Contains(t, string(vclContent), "vcl 4.1")
		require.Contains(t, string(vclContent), `backend default`)

		// Verify varnish_extras file was created
		extrasPath := app.GetConfigPath("docker-compose.varnish_extras.yaml")
		require.FileExists(t, extrasPath)

		extrasContent, err := os.ReadFile(extrasPath)
		require.NoError(t, err)
		require.Contains(t, string(extrasContent), "novarnish.${DDEV_HOSTNAME}")
		require.Contains(t, string(extrasContent), "#ddev-generated")

		// Verify commands directory was installed
		commandsDir := app.GetConfigPath("commands/varnish")
		require.DirExists(t, commandsDir)

		// Check for some key varnish commands
		require.FileExists(t, app.GetConfigPath("commands/varnish/varnishadm"))
		require.FileExists(t, app.GetConfigPath("commands/varnish/varnishlog"))
	})

	// Test repository access addon - demonstrates full project access
	t.Run("RepoAccessAddon", func(t *testing.T) {
		repoAccessAddonDir := filepath.Join(origDir, "testdata", t.Name())

		out, err := exec.RunHostCommand(DdevBin, "add-on", "get", repoAccessAddonDir, "--verbose")
		require.NoError(t, err, "failed to install repo access addon: %v, output: %s", err, out)

		// Check that PHP processed repository files
		require.Contains(t, out, "PHP: Found")
		require.Contains(t, out, "files in project root")
		require.Contains(t, out, "PHP: Created test file in project root")
		require.Contains(t, out, "PHP: Created settings file in web/sites/default/")
		require.Contains(t, out, "PHP: Test file exists and contains:")
		require.Contains(t, out, "PHP: Settings file exists and is readable")
		require.Contains(t, out, "PHP: Repository access test completed successfully")

		// Check that descriptions are displayed
		require.Contains(t, out, "👍  Test repository access capabilities")
		require.Contains(t, out, "👍  Verify repository file access")

		// Verify the files were actually created in the project
		testFile := filepath.Join(app.GetAppRoot(), "php-addon-test.txt")
		require.FileExists(t, testFile)

		testContent, err := os.ReadFile(testFile)
		require.NoError(t, err)
		require.Contains(t, string(testContent), "Test file created by PHP addon")
		require.Contains(t, string(testContent), "Created at:")

		// Verify settings directory and file were created
		settingsDir := filepath.Join(app.GetAppRoot(), "web", "sites", "default")
		require.DirExists(t, settingsDir)

		settingsFile := filepath.Join(settingsDir, "addon-settings.php")
		require.FileExists(t, settingsFile)

		settingsContent, err := os.ReadFile(settingsFile)
		require.NoError(t, err)
		require.Contains(t, string(settingsContent), "<?php")
		require.Contains(t, string(settingsContent), "Settings file created by PHP addon")
		require.Contains(t, string(settingsContent), "$addon_config")
		require.Contains(t, string(settingsContent), "repo-access-test")
	})

	// Test environment variables addon - validates all DDEV environment variables are available
	t.Run("EnvironmentVariablesAddon", func(t *testing.T) {
		envVarsAddonDir := filepath.Join(origDir, "testdata", t.Name())

		// Set some custom environment variables to test
		out, err := exec.RunHostCommand(DdevBin, "dotenv", "set", ".ddev/.env.env-vars-test", "--custom-variable", "show $ sign", "--extra-var", "One more extra variable")
		require.NoError(t, err, "failed to set custom variable in .ddev/.env.env-vars-test file: %v, output: %s", err, out)

		out, err = exec.RunHostCommand(DdevBin, "add-on", "get", envVarsAddonDir, "--verbose")
		require.NoError(t, err, "failed to install environment variables addon: %v, output: %s", err, out)

		// Check that PHP received and validated all environment variables
		require.Contains(t, out, "PHP: Testing environment variables...")

		// Define expected environment variables with their expected values or patterns
		expectedEnvVars := map[string]interface{}{
			"DDEV_SITENAME":        app.Name,
			"DDEV_PROJECT":         app.Name,
			"DDEV_PROJECT_TYPE":    app.Type,
			"DDEV_PHP_VERSION":     app.PHPVersion,
			"DDEV_WEBSERVER_TYPE":  app.WebserverType,
			"DDEV_APPROOT":         "/var/www/html",
			"DDEV_DOCROOT":         "", // Value varies, just check presence
			"DDEV_DATABASE":        "", // Value varies, just check presence
			"DDEV_DATABASE_FAMILY": "", // Value varies, just check presence
			"DDEV_FILES_DIRS":      "", // Value varies, just check presence
			"DDEV_MUTAGEN_ENABLED": "", // Value varies, just check presence
			"DDEV_VERSION":         "", // Value varies, just check presence
			"DDEV_TLD":             "", // Value varies, just check presence
			"IS_DDEV_PROJECT":      "true",
			"CUSTOM_VARIABLE":      "show $ sign",
			"EXTRA_VAR":            "One more extra variable",
		}

		// Verify each environment variable is present in the output
		for envVar, expectedValue := range expectedEnvVars {
			if expectedValue != "" {
				// Check for specific value
				require.Contains(t, out, fmt.Sprintf("PHP: Found %s=%s", envVar, expectedValue),
					"Environment variable %s should have expected value", envVar)
			} else {
				// Just check for presence
				require.Contains(t, out, fmt.Sprintf("PHP: Found %s=", envVar),
					"Environment variable %s should be present", envVar)
			}
		}

		require.Contains(t, out, fmt.Sprintf("PHP: SUCCESS - All %d environment variables found", len(expectedEnvVars)))
		require.Contains(t, out, "PHP: Environment variable validation completed successfully")

		// Check that description is displayed
		require.Contains(t, out, "👍  Test environment variables in PHP actions")
	})

	// Test configuration access addon - validates PHP actions can access processed configuration
	t.Run("ConfigurationAccessAddon", func(t *testing.T) {
		configAccessAddonDir := filepath.Join(origDir, "testdata", t.Name())
		out, err := exec.RunHostCommand(DdevBin, "add-on", "get", configAccessAddonDir, "--verbose")
		require.NoError(t, err, "failed to install configuration access addon: %v, output: %s", err, out)

		// Check that PHP accessed and validated configuration files successfully
		require.Contains(t, out, "PHP: ✓ Project name matches environment:")
		require.Contains(t, out, "PHP: ✓ Project type matches environment:")
		// Note: PHP version and webserver type may not always be available in the YAML structure
		// The test validates what's actually present in the configuration
		require.Contains(t, out, "PHP: Successfully validated")
		require.Contains(t, out, "configuration properties")
		require.Contains(t, out, "PHP: Configuration data validation PASSED")

		// Check that description is displayed
		require.Contains(t, out, "👍  Test configuration file access and data validation")

		// Verify that .ddev-config directory was cleaned up after installation
		configDir := app.GetConfigPath(".ddev-config")
		require.NoDirExists(t, configDir, "Configuration directory should be cleaned up after installation")
	})

	// Test PHP removal actions - install an addon with PHP removal actions and then remove it
	t.Run("PHPRemovalActions", func(t *testing.T) {
		// Use the test add-on which has PHP removal actions
		addonDir := filepath.Join(origDir, "testdata", t.Name())

		// Install the addon first
		out, err := exec.RunHostCommand(DdevBin, "add-on", "get", addonDir)
		require.NoError(t, err, "failed to install varnish PHP addon: %v, output: %s", err, out)

		// Verify the addon was installed - should generate varnish_extras file
		extrasFile := app.GetConfigPath("docker-compose.varnish_extras.yaml")
		require.FileExists(t, extrasFile, "Varnish extras file should be created")

		// Stop the project to test removal actions work without running project
		err = app.Stop(false, false)
		require.NoError(t, err, "failed to stop project for removal test")

		// Remove the addon - this should execute the PHP removal action
		out, err = exec.RunHostCommand(DdevBin, "add-on", "remove", "php-removal-actions")
		require.NoError(t, err, "failed to remove varnish PHP addon: %v, output: %s", err, out)

		// Verify the PHP removal action was executed
		require.Contains(t, out, "Remove generated varnish extras file", "Removal action description should be shown")
		require.Contains(t, out, "PHP: Varnish removal action completed", "PHP removal output should be shown")

		// Verify the PHP removal action created the test file (even when project stopped)
		removalTestFile := app.GetConfigPath("php-removal-test.txt")
		require.FileExists(t, removalTestFile, "PHP removal test file should be created by removal action")

		// Clean up the test file
		_ = os.Remove(removalTestFile)

		// Verify the varnish extras file was removed by the PHP action
		require.NoFileExists(t, extrasFile, "Varnish extras file should be removed by PHP removal action")

		// Verify that .ddev-config directory was cleaned up after removal
		configDir := app.GetConfigPath(".ddev-config")
		require.NoDirExists(t, configDir, "Configuration directory should be cleaned up after removal")
	})

	// Test explicit image without php-yaml (e.g., php:8.3)
	t.Run("ExplicitImageNoPHPYaml", func(t *testing.T) {
		noYamlAddonDir := filepath.Join(origDir, "testdata", t.Name())
		out, err := exec.RunHostCommand(DdevBin, "add-on", "get", noYamlAddonDir, "--verbose")
		require.Error(t, err, "expected error when using image without php-yaml")
		require.Contains(t, out, "undefined function yaml_parse_file", "Should show PHP fatal error due to missing php-yaml")
	})

	// Test explicit image that does not have PHP at all (e.g., alpine:latest)
	t.Run("ExplicitImageNoPHP", func(t *testing.T) {
		noPHPAddonDir := filepath.Join(origDir, "testdata", t.Name())
		out, err := exec.RunHostCommand(DdevBin, "add-on", "get", noPHPAddonDir, "--verbose")
		require.Error(t, err, "expected error when using image without PHP")
		require.Contains(t, out, "php: not found", "Should show error about PHP not found")
	})
}

func TestCmdAddonSearch(t *testing.T) {
	if os.Getenv("DDEV_RUN_GET_TESTS") != "true" {
		t.Skip("Skipping because DDEV_RUN_GET_TESTS is not set")
	}
	assert := asrt.New(t)

	// Test search for redis
	out, err := exec.RunHostCommand(DdevBin, "add-on", "search", "redis")
	assert.NoError(err, "failed ddev add-on search redis: %v (%s)", err, out)
	assert.Contains(out, "ddev/ddev-redis")
	assert.Contains(out, "repositories found matching 'redis'")

	// Test search with multiple keywords
	out, err = exec.RunHostCommand(DdevBin, "add-on", "search", "redis cache")
	assert.NoError(err, "failed ddev add-on search 'redis cache': %v (%s)", err, out)
	assert.Contains(out, "ddev/ddev-redis")
	assert.Contains(out, "repositories found matching 'redis cache'")

	// Test search with no results
	out, err = exec.RunHostCommand(DdevBin, "add-on", "search", "nonexistentservice")
	assert.NoError(err, "failed ddev add-on search nonexistentservice: %v (%s)", err, out)
	assert.Contains(out, "No add-ons found matching 'nonexistentservice'")

	// Test search with --all flag
	out, err = exec.RunHostCommand(DdevBin, "add-on", "search", "redis", "--all")
	assert.NoError(err, "failed ddev add-on search redis --all: %v (%s)", err, out)
	assert.Contains(out, "repositories found matching 'redis'")
}
