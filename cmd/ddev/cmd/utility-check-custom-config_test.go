package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/github"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

// TestUtilityCheckCustomConfigCmd tests ddev debug check-custom-config
func TestUtilityCheckCustomConfigCmd(t *testing.T) {
	// Create a temporary directory and switch to it.
	tmpdir := testcommon.CreateTmpDir(t.Name())
	defer testcommon.CleanupDir(tmpdir)
	defer testcommon.Chdir(tmpdir)()

	// Use tmpdir name as project name
	projectName := filepath.Base(tmpdir)

	// Create a basic config
	args := []string{
		"config",
		"--docroot", ".",
		"--project-name", projectName,
		"--project-type", "php",
	}

	_, err := exec.RunCommand(DdevBin, args)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = exec.RunCommand(DdevBin, []string{"delete", "-Oy", projectName})
	})

	// Test with no custom config
	t.Run("no custom config", func(t *testing.T) {
		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "No custom configuration detected in project '"+projectName+"'.")
	})

	// Test with custom config files
	t.Run("with custom config files", func(t *testing.T) {
		// Create .ddev directory if it doesn't exist
		ddevDir := filepath.Join(tmpdir, ".ddev")
		phpDir := filepath.Join(ddevDir, "php")
		err := os.MkdirAll(phpDir, 0755)
		require.NoError(t, err)

		// Create a custom PHP config file
		customPHPFile := filepath.Join(phpDir, "custom.ini")
		err = os.WriteFile(customPHPFile, []byte("memory_limit = 512M\n"), 0644)
		require.NoError(t, err)

		// Run check-custom-config
		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "PHP")
		require.Contains(t, out, "custom.ini")
	})

	// Test with silenced custom config file
	t.Run("with silenced custom config", func(t *testing.T) {
		ddevDir := filepath.Join(tmpdir, ".ddev")
		phpDir := filepath.Join(ddevDir, "php")

		// Create a silenced custom PHP config file
		silencedFile := filepath.Join(phpDir, "silenced.ini")
		err := os.WriteFile(silencedFile, []byte("#ddev-silent-no-warn\nmemory_limit = 256M\n"), 0644)
		require.NoError(t, err)

		// Run check-custom-config (should show all including silenced)
		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "PHP")
		require.Contains(t, out, "silenced.ini")
		require.Contains(t, out, "(#ddev-silent-no-warn)")
	})

	// Test with multiple custom config types
	t.Run("with multiple custom config types", func(t *testing.T) {
		ddevDir := filepath.Join(tmpdir, ".ddev")

		// Create nginx snippets
		nginxDir := filepath.Join(ddevDir, "nginx")
		err := os.MkdirAll(nginxDir, 0755)
		require.NoError(t, err)
		nginxFile := filepath.Join(nginxDir, "custom.conf")
		err = os.WriteFile(nginxFile, []byte("# Custom nginx config\n"), 0644)
		require.NoError(t, err)

		// Create MySQL config
		mysqlDir := filepath.Join(ddevDir, "mysql")
		err = os.MkdirAll(mysqlDir, 0755)
		require.NoError(t, err)
		mysqlFile := filepath.Join(mysqlDir, "custom.cnf")
		err = os.WriteFile(mysqlFile, []byte("[mysqld]\nmax_connections = 500\n"), 0644)
		require.NoError(t, err)

		// Run check-custom-config
		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "PHP")
		require.Contains(t, out, "Web server")
		require.Contains(t, out, "Database")
	})

	// Test with fake/suspicious DDEV-generated file (has #ddev-generated but not whitelisted)
	t.Run("with suspicious ddev-generated file", func(t *testing.T) {
		ddevDir := filepath.Join(tmpdir, ".ddev")
		phpDir := filepath.Join(ddevDir, "php")

		// Create a file with #ddev-generated marker but in a location where it's not expected
		// This simulates a fake or suspicious DDEV-generated file
		suspiciousFile := filepath.Join(phpDir, "suspicious.ini")
		err := os.WriteFile(suspiciousFile, []byte("#ddev-generated\n; This file claims to be DDEV-generated but isn't whitelisted\nmemory_limit = 1G\n"), 0644)
		require.NoError(t, err)

		// Run check-custom-config
		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		require.Contains(t, out, "PHP")
		require.Contains(t, out, "suspicious.ini")
		// Should be detected as custom even though it has #ddev-generated marker
		// because it's not in the expected/whitelisted files list
	})

	// Test with addon-generated files
	t.Run("with addon-generated files", func(t *testing.T) {
		if !github.HasGitHubToken() {
			t.Skip("Skipping because DDEV_GITHUB_TOKEN is not set")
		}

		// Install a simple addon (redis) to test addon file detection
		_, err := exec.RunCommand(DdevBin, []string{"add-on", "get", "ddev/ddev-phpmyadmin"})
		require.NoError(t, err)

		// Run check-custom-config
		out, err := exec.RunCommand(DdevBin, []string{"utility", "check-custom-config"})
		require.NoError(t, err)
		require.Contains(t, out, "Custom configuration detected in project '"+projectName+"':")
		// Should show addon files with (#ddev-generated) marker
		require.Contains(t, out, "docker-compose.phpmyadmin.yaml")
		require.Contains(t, out, "(#ddev-generated)")
	})
}
