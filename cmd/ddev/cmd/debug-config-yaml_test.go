package cmd

import (
	"testing"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDebugConfigYamlCmd tests that ddev debug configyaml works
func TestDebugConfigYamlCmd(t *testing.T) {
	assert := assert.New(t)

	// Create a temporary directory and switch to it.
	tmpdir := testcommon.CreateTmpDir(t.Name())
	defer testcommon.CleanupDir(tmpdir)
	defer testcommon.Chdir(tmpdir)()

	// Create a basic config
	args := []string{
		"config",
		"--docroot", ".",
		"--project-name", "config-yaml-test",
		"--project-type", "php",
	}

	_, err := exec.RunCommand(DdevBin, args)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = exec.RunCommand(DdevBin, []string{"delete", "-Oy", "config-yaml-test"})
	})

	// Test basic configyaml output (field-by-field mode)
	t.Run("basic configyaml output", func(t *testing.T) {
		out, err := exec.RunCommand(DdevBin, []string{"debug", "configyaml"})
		assert.NoError(err)
		// Note: "These config files were loaded" now goes to stderr, not stdout
		assert.Contains(out, "name: config-yaml-test")
		assert.Contains(out, "type: php")
		assert.Contains(out, "docroot: .")
	})

	// Test --full-yaml mode
	t.Run("full-yaml mode", func(t *testing.T) {
		out, err := exec.RunCommand(DdevBin, []string{"debug", "configyaml", "--full-yaml"})
		assert.NoError(err)
		// Note: "These config files were loaded" now goes to stderr, not stdout
		assert.Contains(out, "# Complete processed project configuration:")
		assert.Contains(out, "name: config-yaml-test")
		assert.Contains(out, "type: php")
		assert.Contains(out, "docroot: .")
		// Should be valid YAML format
		assert.Contains(out, "webserver_type:")
		assert.Contains(out, "php_version:")
	})

	// Test --omit-keys functionality in regular mode
	t.Run("omit-keys in regular mode", func(t *testing.T) {
		out, err := exec.RunCommand(DdevBin, []string{"debug", "configyaml", "--omit-keys=name,type"})
		assert.NoError(err)
		assert.NotContains(out, "name: config-yaml-test")
		assert.NotContains(out, "type: php")
		assert.Contains(out, "docroot: .") // This should still be present
	})

	// Test --omit-keys functionality in full-yaml mode
	t.Run("omit-keys in full-yaml mode", func(t *testing.T) {
		out, err := exec.RunCommand(DdevBin, []string{"debug", "configyaml", "--full-yaml", "--omit-keys=name,type"})
		assert.NoError(err)
		assert.Contains(out, "# Complete processed project configuration:")
		assert.NotContains(out, "name: config-yaml-test")
		assert.NotContains(out, "type: php")
		assert.Contains(out, "docroot: .") // This should still be present
	})

	// Test omitting environment variables (the main security use case)
	t.Run("omit environment variables", func(t *testing.T) {
		// Create a config with web_environment using ddev config command
		_, err := exec.RunCommand(DdevBin, []string{
			"config",
			"--web-environment=SECRET_KEY=supersecret,API_TOKEN=sensitive",
		})
		require.NoError(t, err)

		// Test that environment variables appear by default
		out, err := exec.RunCommand(DdevBin, []string{"debug", "configyaml", "--full-yaml"})
		assert.NoError(err)
		assert.Contains(out, "SECRET_KEY=supersecret")
		assert.Contains(out, "API_TOKEN=sensitive")

		// Test that they're hidden with --omit-keys
		out, err = exec.RunCommand(DdevBin, []string{"debug", "configyaml", "--full-yaml", "--omit-keys=web_environment"})
		assert.NoError(err)
		assert.NotContains(out, "SECRET_KEY=supersecret")
		assert.NotContains(out, "API_TOKEN=sensitive")
		assert.Contains(out, "name: config-yaml-test") // Other fields should still be present
	})

	// Test with spaces in omit-keys (should handle trimming)
	t.Run("omit-keys with spaces", func(t *testing.T) {
		out, err := exec.RunCommand(DdevBin, []string{"debug", "configyaml", "--omit-keys= name , type "})
		assert.NoError(err)
		assert.NotContains(out, "name: config-yaml-test")
		assert.NotContains(out, "type: php")
		assert.Contains(out, "docroot: .") // This should still be present
	})

	// Test that non-existent project name fails gracefully
	t.Run("non-existent project", func(t *testing.T) {
		_, err := exec.RunCommand(DdevBin, []string{"debug", "configyaml", "non-existent-project"})
		assert.Error(err)
	})
}
