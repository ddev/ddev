package cmd

import (
	"testing"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

// TestDebugConfigYamlCmd tests that ddev utility configyaml works
func TestDebugConfigYamlCmd(t *testing.T) {
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
		out, err := exec.RunCommand(DdevBin, []string{"utility", "configyaml"})
		require.NoError(t, err)
		// Note: "These config files were loaded" now goes to stderr, not stdout
		require.Contains(t, out, "name: config-yaml-test")
		require.Contains(t, out, "type: php")
		require.Contains(t, out, "docroot: .")
	})

	// Test --full-yaml mode
	t.Run("full-yaml mode", func(t *testing.T) {
		out, err := exec.RunCommand(DdevBin, []string{"utility", "configyaml", "--full-yaml"})
		require.NoError(t, err)
		// Note: "These config files were loaded" now goes to stderr, not stdout
		require.Contains(t, out, "# Complete processed project configuration:")
		require.Contains(t, out, "name: config-yaml-test")
		require.Contains(t, out, "type: php")
		require.Contains(t, out, "docroot: .")
		// Should be valid YAML format
		require.Contains(t, out, "webserver_type:")
		require.Contains(t, out, "php_version:")
	})

	// Test --omit-keys functionality in regular mode
	t.Run("omit-keys in regular mode", func(t *testing.T) {
		out, err := exec.RunCommand(DdevBin, []string{"utility", "configyaml", "--omit-keys=name,type"})
		require.NoError(t, err)
		require.NotContains(t, out, "name: config-yaml-test")
		require.NotContains(t, out, "type: php")
		require.Contains(t, out, "docroot: .") // This should still be present
	})

	// Test --omit-keys functionality in full-yaml mode
	t.Run("omit-keys in full-yaml mode", func(t *testing.T) {
		out, err := exec.RunCommand(DdevBin, []string{"utility", "configyaml", "--full-yaml", "--omit-keys=name,type"})
		require.NoError(t, err)
		require.Contains(t, out, "# Complete processed project configuration:")
		require.NotContains(t, out, "name: config-yaml-test")
		require.NotContains(t, out, "type: php")
		require.Contains(t, out, "docroot: .") // This should still be present
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
		out, err := exec.RunCommand(DdevBin, []string{"utility", "configyaml", "--full-yaml"})
		require.NoError(t, err)
		require.Contains(t, out, "SECRET_KEY=supersecret")
		require.Contains(t, out, "API_TOKEN=sensitive")

		// Test that they're hidden with --omit-keys
		out, err = exec.RunCommand(DdevBin, []string{"utility", "configyaml", "--full-yaml", "--omit-keys=web_environment"})
		require.NoError(t, err)
		require.NotContains(t, out, "SECRET_KEY=supersecret")
		require.NotContains(t, out, "API_TOKEN=sensitive")
		require.Contains(t, out, "name: config-yaml-test") // Other fields should still be present
	})

	// Test with spaces in omit-keys (should handle trimming)
	t.Run("omit-keys with spaces", func(t *testing.T) {
		out, err := exec.RunCommand(DdevBin, []string{"utility", "configyaml", "--omit-keys= name , type "})
		require.NoError(t, err)
		require.NotContains(t, out, "name: config-yaml-test")
		require.NotContains(t, out, "type: php")
		require.Contains(t, out, "docroot: .") // This should still be present
	})

	// Test that non-existent project name fails gracefully
	t.Run("non-existent project", func(t *testing.T) {
		_, err := exec.RunCommand(DdevBin, []string{"utility", "configyaml", "non-existent-project"})
		require.Error(t, err)
	})
}
