package ddevapp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/testcommon"
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
