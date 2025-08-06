package ddevapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestProcessPHPAction(t *testing.T) {
	// Create a temporary test app (use ~/tmp which is mountable by Docker)
	homeDir, _ := os.UserHomeDir()
	testDir := filepath.Join(homeDir, "tmp", "test-app")
	app := &DdevApp{
		AppRoot: testDir,
		Name:    "test-project",
		Type:    "php",
	}

	// Create the .ddev directory and config.yaml for testing
	err := os.MkdirAll(app.AppConfDir(), 0755)
	require.NoError(t, err)

	configContent := `name: test-project
type: php
`
	err = os.WriteFile(filepath.Join(app.AppConfDir(), "config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	defer os.RemoveAll(testDir)

	// Test PHP syntax validation with intentionally broken PHP
	t.Run("PHPSyntaxValidation", func(t *testing.T) {
		// Test with syntax error
		brokenAction := `<?php
echo "This has a syntax error
// Missing closing quote above
?>`

		dict := map[string]interface{}{}
		err := processPHPAction(brokenAction, dict, "", true, app)
		require.Error(t, err, "Broken PHP action should fail syntax validation")
		require.Contains(t, err.Error(), "syntax error", "Error should mention syntax error")
	})

	// Test PHP strict error handling
	t.Run("PHPStrictErrorHandling", func(t *testing.T) {
		// This should trigger an error due to undefined variable
		strictTestAction := `<?php
echo "Testing strict mode\n";
echo $undefined_variable; // This should trigger strict error handling
?>`

		dict := map[string]interface{}{}
		err := processPHPAction(strictTestAction, dict, "", true, app)
		// The error should be caught by our strict error handling
		require.Error(t, err, "PHP action with undefined variable should fail due to strict mode")
	})

	// Test basic PHP action
	t.Run("BasicPHPAction", func(t *testing.T) {
		action := `<?php
echo "Hello from PHP test\n";
echo "This is working\n";
?>`

		dict := map[string]interface{}{
			"DdevProjectConfig": map[string]interface{}{
				"name": "test-project",
				"type": "php",
			},
		}

		err := processPHPAction(action, dict, "", true, app)
		require.NoError(t, err, "PHP action should execute without error")
	})

	// Test PHP action with config access
	t.Run("PHPActionWithConfig", func(t *testing.T) {
		action := `<?php
$configPath = 'config.yaml';
if (file_exists($configPath)) {
    $configContent = file_get_contents($configPath);
    if (preg_match('/^name:\s*(.+)$/m', $configContent, $matches)) {
        $projectName = trim($matches[1]);
        echo "Project name: $projectName\n";
    }
}
?>`

		dict := map[string]interface{}{
			"DdevProjectConfig": map[string]interface{}{
				"name": "test-project",
				"type": "php",
			},
		}

		err := processPHPAction(action, dict, "", true, app)
		require.NoError(t, err, "PHP action with config should execute without error")
	})

	// Test custom PHP image
	t.Run("CustomPHPImage", func(t *testing.T) {
		action := `<?php
echo "PHP Version: " . PHP_VERSION . "\n";
?>`

		dict := map[string]interface{}{
			"DdevProjectConfig": map[string]interface{}{
				"name": "test-project",
			},
		}

		err := processPHPAction(action, dict, "php:8.1-cli", true, app)
		require.NoError(t, err, "PHP action with custom image should execute without error")
	})

	// Test working directory is set to /var/www/html/.ddev
	t.Run("WorkingDirectoryTest", func(t *testing.T) {
		action := `<?php
// Test that we're in the correct working directory
$workingDir = getcwd();
echo "Working directory: $workingDir\n";

// Test relative path access to config.yaml
if (file_exists('config.yaml')) {
    echo "Found config.yaml in working directory\n";
    $configContent = file_get_contents('config.yaml');
    if (strpos($configContent, 'test-project') !== false) {
        echo "Config contains expected project name\n";
    }
} else {
    echo "config.yaml not found in working directory\n";
}

// Test relative path access to parent directory (../composer.json would be at project root)
if (file_exists('../')) {
    echo "Parent directory accessible\n";
} else {
    echo "Parent directory not accessible\n";
}
?>`

		dict := map[string]interface{}{
			"DdevProjectConfig": map[string]interface{}{
				"name": "test-project",
				"type": "php",
			},
		}

		err := processPHPAction(action, dict, "", true, app)
		require.NoError(t, err, "PHP action with working directory test should execute without error")
	})

	// Test file writing with relative paths
	t.Run("RelativeFileWriteTest", func(t *testing.T) {
		action := `<?php
// Test writing a file in the current directory (.ddev)
$testFile = 'test-output.txt';
$testContent = "Test content from PHP addon\n";
file_put_contents($testFile, $testContent);

if (file_exists($testFile)) {
    echo "Successfully wrote file: $testFile\n";
    $readContent = file_get_contents($testFile);
    if ($readContent === $testContent) {
        echo "File content matches expected content\n";
    }
    // Clean up
    unlink($testFile);
} else {
    echo "Failed to write file: $testFile\n";
}

// Test writing a file in parent directory (project root)
$parentTestFile = '../test-parent.txt';
$parentContent = "Test content in parent directory\n";
file_put_contents($parentTestFile, $parentContent);

if (file_exists($parentTestFile)) {
    echo "Successfully wrote file in parent directory\n";
    // Clean up
    unlink($parentTestFile);
} else {
    echo "Failed to write file in parent directory\n";
}
?>`

		dict := map[string]interface{}{
			"DdevProjectConfig": map[string]interface{}{
				"name": "test-project",
				"type": "php",
			},
		}

		err := processPHPAction(action, dict, "", true, app)
		require.NoError(t, err, "PHP action with relative file write test should execute without error")
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

func TestGetProcessedProjectConfigYAML(t *testing.T) {
	// Create a temporary test app
	homeDir, _ := os.UserHomeDir()
	testDir := filepath.Join(homeDir, "tmp", "test-config-app")
	app := &DdevApp{
		AppRoot:    testDir,
		Name:       "test-config-project",
		Type:       "drupal",
		ConfigPath: filepath.Join(testDir, ".ddev", "config.yaml"),
	}

	// Create the .ddev directory and config files for testing
	err := os.MkdirAll(app.AppConfDir(), 0755)
	require.NoError(t, err)

	// Create base config.yaml
	baseConfig := `name: test-config-project
type: drupal
docroot: web
php_version: "8.2"
database:
  type: mysql
  version: "8.0"
webserver_type: nginx-fpm
additional_hostnames:
  - test.ddev.site
`
	err = os.WriteFile(filepath.Join(app.AppConfDir(), "config.yaml"), []byte(baseConfig), 0644)
	require.NoError(t, err)

	// Create config.override.yaml to test merging
	overrideConfig := `php_version: "8.3"
additional_hostnames:
  - override.ddev.site
webserver_type: apache-fpm
`
	err = os.WriteFile(filepath.Join(app.AppConfDir(), "config.override.yaml"), []byte(overrideConfig), 0644)
	require.NoError(t, err)

	// Clean up
	defer func() {
		_ = os.RemoveAll(testDir)
	}()

	// Test GetProcessedProjectConfigYAML
	configYAML, err := app.GetProcessedProjectConfigYAML()
	require.NoError(t, err)
	require.NotEmpty(t, configYAML)

	// Parse the result to verify it contains merged values
	var parsedConfig map[string]interface{}
	err = yaml.Unmarshal(configYAML, &parsedConfig)
	require.NoError(t, err)

	// Verify base values are preserved
	require.Equal(t, "test-config-project", parsedConfig["name"])
	require.Equal(t, "drupal", parsedConfig["type"])
	require.Equal(t, "web", parsedConfig["docroot"])

	// Verify override values took precedence
	require.Equal(t, "8.3", parsedConfig["php_version"])           // Should be overridden
	require.Equal(t, "apache-fpm", parsedConfig["webserver_type"]) // Should be overridden

	// Verify arrays were merged (both hostnames should be present)
	hostnames, ok := parsedConfig["additional_hostnames"]
	require.True(t, ok)
	hostnameSlice, ok := hostnames.([]interface{})
	require.True(t, ok)
	require.Len(t, hostnameSlice, 2) // Should have both hostnames

	// Convert to strings for easier checking
	var hostnameStrings []string
	for _, h := range hostnameSlice {
		hostnameStrings = append(hostnameStrings, h.(string))
	}
	require.Contains(t, hostnameStrings, "test.ddev.site")
	require.Contains(t, hostnameStrings, "override.ddev.site")
}

func TestGetGlobalConfigYAML(t *testing.T) {
	// Test GetGlobalConfigYAML
	globalConfigYAML, err := GetGlobalConfigYAML()
	require.NoError(t, err)
	require.NotEmpty(t, globalConfigYAML)

	// Parse the result to verify it's valid YAML
	var parsedGlobalConfig map[string]interface{}
	err = yaml.Unmarshal(globalConfigYAML, &parsedGlobalConfig)
	require.NoError(t, err)

	// The global config should contain some expected fields
	// Note: We can't test specific values since they depend on the system
	// but we can verify the structure is reasonable
	require.NotNil(t, parsedGlobalConfig)
}
