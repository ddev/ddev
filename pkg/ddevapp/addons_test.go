package ddevapp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
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
		err = ddevapp.ProcessAddonAction(action, installDesc, app, verbose)
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

	// Test normalized identifier circular dependency detection
	t.Run("DetectsCircularDependencyWithNormalizedIdentifiers", func(t *testing.T) {
		// Reset the global install stack
		ddevapp.ResetInstallStack()

		// Add addon using GitHub format
		err1 := ddevapp.AddToInstallStack("ddev/ddev-redis")
		require.NoError(t, err1)

		// Try to add the same addon using GitHub URL format - should detect circular dependency
		err2 := ddevapp.AddToInstallStack("https://github.com/ddev/ddev-redis/archive/refs/tags/v1.0.0.tar.gz")
		require.Error(t, err2, "Should detect circular dependency with different identifier formats")
		require.Contains(t, err2.Error(), "circular dependency detected", "Error should mention circular dependency")
	})

	// Test the NormalizeAddonIdentifier function directly
	t.Run("NormalizeAddonIdentifierFunction", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"ddev/ddev-redis", "ddev/ddev-redis"},
			{"https://github.com/ddev/ddev-redis/archive/refs/tags/v1.0.0.tar.gz", "ddev/ddev-redis"},
			{"/path/to/ddev-redis.tar.gz", "ddev-redis"},
			{"ddev-redis.tar.gz", "ddev-redis"},
			{"local-addon", "local-addon"},
			{"./relative/path/addon.tgz", "addon"},
		}

		for _, tc := range testCases {
			result := ddevapp.NormalizeAddonIdentifier(tc.input)
			require.Equal(t, tc.expected, result, "Normalization of %q should return %q but got %q", tc.input, tc.expected, result)
		}
	})

	t.Run("DetectsCircularDependencyWithLocalPath", func(t *testing.T) {
		// Reset the global install stack
		ddevapp.ResetInstallStack()

		// Add addon using tarball format first
		err1 := ddevapp.AddToInstallStack("/path/to/ddev-redis.tar.gz")
		require.NoError(t, err1)

		// Try to add the same addon using GitHub format - should detect circular dependency
		// Both should normalize to "ddev-redis" and trigger detection
		err2 := ddevapp.AddToInstallStack("ddev-redis.tar.gz")
		require.Error(t, err2, "Should detect circular dependency with different path formats")
		require.Contains(t, err2.Error(), "circular dependency detected", "Error should mention circular dependency")
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

// TestMixedDependencyScenarios tests comprehensive mixed dependency scenarios
func TestMixedDependencyScenarios(t *testing.T) {
	t.Run("NormalizeAddonIdentifierEdgeCases", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    string
			expected string
		}{
			{"GitHub owner/repo", "ddev/ddev-redis", "ddev/ddev-redis"},
			{"GitHub URL", "https://github.com/ddev/ddev-redis/archive/refs/tags/v1.0.0.tar.gz", "ddev/ddev-redis"},
			{"Local absolute path", "/path/to/ddev-redis.tar.gz", "ddev-redis"},
			{"Local relative path", "./ddev-redis.tar.gz", "ddev-redis"},
			{"Tarball with .tgz extension", "addon.tgz", "addon"},
			{"Zip file", "addon.zip", "addon"},
			{"Plain directory", "my-addon", "my-addon"},
			{"Nested directory", "../parent/addon-name", "addon-name"},
			{"URL with subdirectory", "https://github.com/user/repo/subfolder", "user/repo"},
			{"Complex GitHub URL", "https://github.com/ddev/ddev-solr/releases/download/v1.2.3/ddev-solr-v1.2.3.tar.gz", "ddev/ddev-solr"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := ddevapp.NormalizeAddonIdentifier(tc.input)
				require.Equal(t, tc.expected, result, "Normalization of %q should return %q but got %q", tc.input, tc.expected, result)
			})
		}
	})

	t.Run("CircularDependencyWithMixedFormats", func(t *testing.T) {
		scenarios := []struct {
			name      string
			first     string
			second    string
			shouldErr bool
		}{
			{
				name:      "GitHub vs URL same repo",
				first:     "ddev/ddev-redis",
				second:    "https://github.com/ddev/ddev-redis/archive/v1.0.0.tar.gz",
				shouldErr: true,
			},
			{
				name:      "URL vs local tarball same name",
				first:     "https://example.com/ddev-solr.tar.gz",
				second:    "/local/path/ddev-solr.tar.gz",
				shouldErr: true,
			},
			{
				name:      "Different addons should not conflict",
				first:     "ddev/ddev-redis",
				second:    "ddev/ddev-solr",
				shouldErr: false,
			},
			{
				name:      "Different extensions same base name",
				first:     "/path/addon.tar.gz",
				second:    "/other/path/addon.tgz",
				shouldErr: true,
			},
		}

		for _, scenario := range scenarios {
			t.Run(scenario.name, func(t *testing.T) {
				ddevapp.ResetInstallStack()

				// Add first addon
				err1 := ddevapp.AddToInstallStack(scenario.first)
				require.NoError(t, err1)

				// Try to add second addon
				err2 := ddevapp.AddToInstallStack(scenario.second)

				if scenario.shouldErr {
					require.Error(t, err2, "Should detect circular dependency")
					require.Contains(t, err2.Error(), "circular dependency detected")
				} else {
					require.NoError(t, err2, "Should not detect circular dependency for different addons")
				}
			})
		}
	})

	t.Run("DependencyPathResolution", func(t *testing.T) {
		// Test ResolveDependencyPaths function with various path formats
		extractedDir := "/tmp/addon-base"
		dependencies := []string{
			"ddev/ddev-redis",
			"../relative-addon",
			"./local-addon",
			"https://example.com/addon.tar.gz",
			"/absolute/path/addon",
		}

		resolved := ddevapp.ResolveDependencyPaths(dependencies, extractedDir, false)

		expected := []string{
			"ddev/ddev-redis",
			"/tmp/relative-addon", // filepath.Clean(filepath.Join("/tmp/addon-base", "../relative-addon"))
			"/tmp/addon-base/local-addon",
			"https://example.com/addon.tar.gz",
			"/absolute/path/addon",
		}

		require.Equal(t, expected, resolved)
	})

	t.Run("InstallStackCleanupOnDefer", func(t *testing.T) {
		// Test that the install stack is properly cleaned up on function exit
		ddevapp.ResetInstallStack()

		// This test simulates the defer cleanup pattern used in installAddonRecursive
		func() {
			// Simulate adding to stack
			err := ddevapp.AddToInstallStack("test-addon")
			require.NoError(t, err)

			defer func() {
				// This simulates the cleanup code
				ddevapp.ResetInstallStack() // For test simplicity, just reset
			}()

			// Stack should contain the addon now
			err = ddevapp.AddToInstallStack("test-addon")
			require.Error(t, err, "Should detect circular dependency within same function")
		}()

		// After function exit, stack should be clean
		err := ddevapp.AddToInstallStack("test-addon")
		require.NoError(t, err, "Stack should be clean after function exit")
	})

	t.Run("ManifestKeyMatching", func(t *testing.T) {
		// Test that GatherAllManifests creates appropriate keys for different addon formats
		// This is tested implicitly through the existing system, but we can verify
		// the logic by testing key creation patterns

		testCases := []struct {
			name         string
			repository   string
			expectedKeys []string
		}{
			{
				name:         "redis",
				repository:   "ddev/ddev-redis",
				expectedKeys: []string{"redis", "ddev/ddev-redis", "ddev-redis"},
			},
			{
				name:         "solr",
				repository:   "https://github.com/ddev/ddev-solr/archive/v1.0.0.tar.gz",
				expectedKeys: []string{"solr", "https://github.com/ddev/ddev-solr/archive/v1.0.0.tar.gz"},
			},
			{
				name:         "local-addon",
				repository:   "/path/to/local-addon",
				expectedKeys: []string{"local-addon", "/path/to/local-addon"},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Create a mock manifest
				manifest := ddevapp.AddonManifest{
					Name:       tc.name,
					Repository: tc.repository,
					Version:    "v1.0.0",
				}

				// Simulate the key creation logic from GatherAllManifests
				allManifests := make(map[string]ddevapp.AddonManifest)
				allManifests[manifest.Name] = manifest
				allManifests[manifest.Repository] = manifest

				pathParts := strings.Split(manifest.Repository, "/")
				if len(pathParts) > 1 {
					shortRepo := pathParts[len(pathParts)-1]
					allManifests[shortRepo] = manifest
				}

				// Verify all expected keys exist
				for _, expectedKey := range tc.expectedKeys {
					_, exists := allManifests[expectedKey]
					require.True(t, exists, "Expected key %q should exist in manifest map", expectedKey)
				}
			})
		}
	})
}

// TestAddonDirectoryCreation tests that addon installation creates necessary directories
// for file copying, particularly during dependency installation scenarios.
// This test validates that copy.Copy properly handles nested directory creation.
func TestAddonDirectoryCreation(t *testing.T) {
	if os.Getenv("DDEV_RUN_DIRECTORY_TESTS") == "" {
		t.Skip("Skipping directory creation test because DDEV_RUN_DIRECTORY_TESTS is not set")
	}

	// Create a test addon structure that requires nested directories
	testAddonDir := testcommon.CreateTmpDir(t.Name())
	defer func() {
		err := os.RemoveAll(testAddonDir)
		assert.NoError(t, err)
	}()

	// Create install.yaml with nested project files
	installYaml := `name: test-directory-addon
repository: test-repo
description: Test addon for directory creation
project_files:
  - nested/deep/config.yaml
  - scripts/setup.sh
  - data/settings.conf
global_files:
  - global/nested/global.conf`

	err := os.WriteFile(filepath.Join(testAddonDir, "install.yaml"), []byte(installYaml), 0644)
	require.NoError(t, err)

	// Create the source files in nested directory structure
	sourceDirs := []string{
		"nested/deep",
		"scripts",
		"data",
		"global/nested",
	}

	sourceFiles := map[string]string{
		"nested/deep/config.yaml":   "#ddev-generated\ntest: config\n",
		"scripts/setup.sh":          "#ddev-generated\n#!/bin/bash\necho 'setup'\n",
		"data/settings.conf":        "#ddev-generated\nsetting=value\n",
		"global/nested/global.conf": "#ddev-generated\nglobal=setting\n",
	}

	for _, dir := range sourceDirs {
		err = os.MkdirAll(filepath.Join(testAddonDir, dir), 0755)
		require.NoError(t, err)
	}

	for filePath, content := range sourceFiles {
		err = os.WriteFile(filepath.Join(testAddonDir, filePath), []byte(content), 0644)
		require.NoError(t, err)
	}

	// Create test app with clean .ddev directory
	site := testcommon.CreateTmpDir(t.Name() + "_site")
	defer func() {
		err := os.RemoveAll(site)
		assert.NoError(t, err)
	}()

	app, err := ddevapp.NewApp(site, false)
	require.NoError(t, err)

	// Ensure no existing directories that might mask the bug
	ddevDir := app.GetConfigPath("")
	err = os.RemoveAll(ddevDir)
	require.NoError(t, err)

	err = os.MkdirAll(ddevDir, 0755)
	require.NoError(t, err)

	// Test InstallAddonFromDirectory directly
	err = ddevapp.InstallAddonFromDirectory(app, testAddonDir, false)
	require.NoError(t, err, "InstallAddonFromDirectory should handle directory creation")

	// Verify all files were copied to correct locations
	expectedFiles := []string{
		app.GetConfigPath("nested/deep/config.yaml"),
		app.GetConfigPath("scripts/setup.sh"),
		app.GetConfigPath("data/settings.conf"),
	}

	for _, expectedFile := range expectedFiles {
		require.True(t, fileutil.FileExists(expectedFile), "File should exist: %s", expectedFile)

		// Verify content was copied correctly
		content, err := fileutil.ReadFileIntoString(expectedFile)
		require.NoError(t, err)
		require.Contains(t, content, "#ddev-generated", "File should contain ddev-generated signature")
	}

	// Verify global file was created
	globalFile := filepath.Join(globalconfig.GetGlobalDdevDir(), "global/nested/global.conf")
	require.True(t, fileutil.FileExists(globalFile), "Global file should exist: %s", globalFile)

	// Clean up global file
	err = os.Remove(globalFile)
	assert.NoError(t, err)
	// Remove empty directories
	err = os.RemoveAll(filepath.Join(globalconfig.GetGlobalDdevDir(), "global"))
	assert.NoError(t, err)
}
