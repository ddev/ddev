package ddevapp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

// processTestAddon is a helper function to process addon install.yaml files from testdata
func processTestAddon(app *ddevapp.DdevApp, testName, addonName string, verbose bool) error {
	addonPath := filepath.Join("testdata", testName, addonName)
	yamlFile := filepath.Join(addonPath, "install.yaml")

	yamlContent, err := os.ReadFile(yamlFile)
	if err != nil {
		return err
	}

	var installDesc ddevapp.InstallDesc
	err = yaml.Unmarshal(yamlContent, &installDesc)
	if err != nil {
		return err
	}

	// Test dependency installation (matches addon-get.go logic)
	if len(installDesc.Dependencies) > 0 {
		err := ddevapp.InstallDependencies(app, installDesc.Dependencies, verbose)
		if err != nil {
			return err
		}
	}

	// Copy project files to the app directory
	for _, file := range installDesc.ProjectFiles {
		srcFile := filepath.Join(addonPath, file)
		destFile := filepath.Join(app.AppConfDir(), file)

		// Ensure directory exists
		err = os.MkdirAll(filepath.Dir(destFile), 0755)
		if err != nil {
			return err
		}

		content, err := os.ReadFile(srcFile)
		if err != nil {
			return err
		}

		err = os.WriteFile(destFile, content, 0644)
		if err != nil {
			return err
		}
	}

	// Process post install actions
	for _, action := range installDesc.PostInstallActions {
		err = ddevapp.ProcessAddonAction(action, installDesc, app, "", verbose)
		if err != nil {
			return err
		}
	}

	return nil
}

func TestProcessPHPAction(t *testing.T) {
	tmpDir := testcommon.CreateTmpDir(t.Name())

	// Create the project directory
	projectDir := filepath.Join(tmpDir, t.Name())
	err := os.MkdirAll(projectDir, 0755)
	require.NoError(t, err)

	// Create and initialize the app properly using NewApp
	app, err := ddevapp.NewApp(projectDir, true)
	require.NoError(t, err)

	// Write the config using the proper DDEV method
	err = app.WriteConfig()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = app.Stop(true, false)
		_ = os.RemoveAll(tmpDir)
	})

	// Test basic PHP action
	t.Run("BasicPHPAction", func(t *testing.T) {
		err := processTestAddon(app, "TestProcessPHPAction", "BasicPHPAddon", true)
		require.NoError(t, err, "PHP action should execute without error")
	})
}

func TestProcessAddonActionPHPDetection(t *testing.T) {
	// Test PHP detection
	t.Run("PHPDetection", func(t *testing.T) {
		phpAction := "<?php echo 'Hello'; ?>"
		bashAction := "echo 'Hello'"

		// Check that PHP actions are detected
		require.True(t, strings.HasPrefix(strings.TrimSpace(phpAction), "<?php"))
		require.False(t, strings.HasPrefix(strings.TrimSpace(bashAction), "<?php"))
	})

	// Test mixed whitespace PHP detection
	t.Run("PHPDetectionWithWhitespace", func(t *testing.T) {
		phpAction := `
		<?php echo 'Hello'; ?>`

		require.True(t, strings.HasPrefix(strings.TrimSpace(phpAction), "<?php"))
	})
}

func TestValidatePHPIncludesAndRequires(t *testing.T) {
	tmpDir := testcommon.CreateTmpDir(t.Name())

	// Create the project directory
	projectDir := filepath.Join(tmpDir, t.Name())
	err := os.MkdirAll(projectDir, 0755)
	require.NoError(t, err)

	// Create and initialize the app properly using NewApp
	app, err := ddevapp.NewApp(projectDir, true)
	require.NoError(t, err)

	// Write the config using the proper DDEV method
	err = app.WriteConfig()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = app.Stop(true, false)
		_ = os.RemoveAll(tmpDir)
	})

	// Test multiple include/require statements are found and validated
	t.Run("MultipleIncludesFound", func(t *testing.T) {
		err := processTestAddon(app, "TestValidatePHPIncludesAndRequires", "MultipleIncludesAddon", false)
		require.NoError(t, err, "Multiple valid includes should pass validation")
	})

	// Test that syntax errors in included files are caught
	t.Run("SyntaxErrorInIncludedFile", func(t *testing.T) {
		err := processTestAddon(app, "TestValidatePHPIncludesAndRequires", "SyntaxErrorAddon", false)
		require.Error(t, err, "Syntax error in included file should cause validation to fail")
		require.Contains(t, err.Error(), "Parse error", "Error should mention parse error")
	})

	// Test that missing included files cause failure during execution (not just validation)
	t.Run("MissingIncludedFiles", func(t *testing.T) {
		err := processTestAddon(app, "TestValidatePHPIncludesAndRequires", "MissingIncludesAddon", false)
		require.Error(t, err, "Missing included files cause execution to fail")
		require.Contains(t, err.Error(), "Failed to open stream", "Error should mention file not found")
	})

	// Test various include/require syntax patterns to ensure regex patterns work correctly
	t.Run("RegexEdgeCases", func(t *testing.T) {
		err := processTestAddon(app, "TestValidatePHPIncludesAndRequires", "RegexEdgeCaseAddon", false)
		require.NoError(t, err, "Various include/require syntax patterns should be parsed and validated correctly")
	})

	// Regression test: This would pass silently with the old broken regex but should fail now
	t.Run("RegressionTestCatchesMissedValidation", func(t *testing.T) {
		err := processTestAddon(app, "TestValidatePHPIncludesAndRequires", "RegressionTestAddon", false)
		require.Error(t, err, "Should catch syntax errors in included files (would be silently missed with old regex)")
		require.Contains(t, err.Error(), "Parse error", "Should contain PHP parse error")
	})
}

func TestCircularDependencyDetection(t *testing.T) {
	// Test the circular dependency detection logic directly
	t.Run("DetectsCircularDependency", func(t *testing.T) {
		// Reset the global install stack
		ddevapp.ResetInstallStack()

		// Simulate a circular dependency scenario
		err1 := ddevapp.AddToInstallStack("addon-a")
		require.NoError(t, err1)

		err2 := ddevapp.AddToInstallStack("addon-b")
		require.NoError(t, err2)

		// This should detect the circular dependency
		err3 := ddevapp.AddToInstallStack("addon-a")
		require.Error(t, err3, "Should detect circular dependency")
		require.Contains(t, err3.Error(), "circular dependency detected", "Error should mention circular dependency")
		require.Contains(t, err3.Error(), "addon-a -> addon-b -> addon-a", "Error should show dependency chain")
	})

	t.Run("AllowsValidDependencyChain", func(t *testing.T) {
		// Reset the global install stack
		ddevapp.ResetInstallStack()

		// Simulate a valid dependency chain
		err1 := ddevapp.AddToInstallStack("addon-c")
		require.NoError(t, err1)

		err2 := ddevapp.AddToInstallStack("addon-d")
		require.NoError(t, err2)

		// Should not detect circular dependency
		err3 := ddevapp.AddToInstallStack("addon-e")
		require.NoError(t, err3)
	})
}

func TestGetGitHubRelease(t *testing.T) {
	if os.Getenv("DDEV_RUN_GET_TESTS") != "true" {
		t.Skip("Skipping because DDEV_RUN_GET_TESTS is not set")
	}

	// Test getting latest release
	t.Run("GetLatestRelease", func(t *testing.T) {
		tarballURL, version, err := ddevapp.GetGitHubRelease("ddev", "ddev-redis", "")
		require.NoError(t, err, "Should successfully get latest release")
		require.NotEmpty(t, tarballURL, "Tarball URL should not be empty")
		require.NotEmpty(t, version, "Version should not be empty")
		require.Contains(t, tarballURL, "github.com", "Tarball URL should be from GitHub")
	})

	// Test getting specific version
	t.Run("GetSpecificVersion", func(t *testing.T) {
		tarballURL, version, err := ddevapp.GetGitHubRelease("ddev", "ddev-redis", "v1.0.4")
		require.NoError(t, err, "Should successfully get specific version")
		require.Equal(t, "v1.0.4", version, "Should return requested version")
		require.NotEmpty(t, tarballURL, "Tarball URL should not be empty")
	})

	// Test non-existent repository
	t.Run("NonExistentRepo", func(t *testing.T) {
		_, _, err := ddevapp.GetGitHubRelease("ddev", "non-existent-repo", "")
		require.Error(t, err, "Should fail for non-existent repository")
		require.Contains(t, err.Error(), "unable to get releases", "Error should mention inability to get releases")
	})

	// Test non-existent version
	t.Run("NonExistentVersion", func(t *testing.T) {
		_, _, err := ddevapp.GetGitHubRelease("ddev", "ddev-redis", "v999.999.999")
		require.Error(t, err, "Should fail for non-existent version")
		require.Contains(t, err.Error(), "no release found", "Error should mention no release found")
	})

	// Test real addon with dependencies
	t.Run("RealAddonWithDependencies", func(t *testing.T) {
		// Test ddev-redis-commander which depends on ddev-redis
		tarballURL, version, err := ddevapp.GetGitHubRelease("ddev", "ddev-redis-commander", "")
		require.NoError(t, err, "Should successfully get ddev-redis-commander release")
		require.NotEmpty(t, tarballURL, "Tarball URL should not be empty")
		require.NotEmpty(t, version, "Version should not be empty")
		require.Contains(t, tarballURL, "github.com", "Tarball URL should be from GitHub")
	})
}

// TestParseRuntimeDependencies tests runtime dependency file parsing
func TestParseRuntimeDependencies(t *testing.T) {
	assert := assert.New(t)

	// Create a temporary .runtime-deps file
	tmpDir, err := os.MkdirTemp("", "test-runtime-deps")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	runtimeDepsFile := filepath.Join(tmpDir, ".runtime-deps")
	content := `# Runtime dependencies
ddev/ddev-redis

# Another comment
../local/addon
https://example.com/addon.tar.gz

# Empty line above should be ignored
`
	err = os.WriteFile(runtimeDepsFile, []byte(content), 0644)
	require.NoError(t, err)

	// Test parsing
	deps, err := ddevapp.ParseRuntimeDependencies(runtimeDepsFile)
	assert.NoError(err)
	assert.Equal([]string{"ddev/ddev-redis", "../local/addon", "https://example.com/addon.tar.gz"}, deps)

	// Test non-existent file
	deps, err = ddevapp.ParseRuntimeDependencies(filepath.Join(tmpDir, "nonexistent"))
	assert.NoError(err)
	assert.Nil(deps)
}
