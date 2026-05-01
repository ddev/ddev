package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

// TestUtilityAddonUpdateCheckerCmd tests basic functionality of ddev utility addon-update-checker
func TestUtilityAddonUpdateCheckerCmd(t *testing.T) {
	// Test 1: directory contains install.yaml directly — single run
	t.Run("SingleAddon", func(t *testing.T) {
		dir := testcommon.CreateTmpDir(filepath.Base(t.Name()))
		defer testcommon.CleanupDir(dir)

		err := os.WriteFile(filepath.Join(dir, "install.yaml"), []byte("name: test-addon\n"), 0644)
		require.NoError(t, err)

		// Script may exit non-zero when add-on is out of date; ignore the error.
		out, _ := exec.RunHostCommand(DdevBin, "utility", "addon-update-checker", "--dir", dir)
		require.Contains(t, out, dir)
		require.Contains(t, out, "Exit code:")
	})

	// Test 2: workspace with subdirectories each containing install.yaml — runs in each
	t.Run("MultipleAddons", func(t *testing.T) {
		workspace := testcommon.CreateTmpDir(filepath.Base(t.Name()))
		defer testcommon.CleanupDir(workspace)

		for _, name := range []string{"addon-one", "addon-two"} {
			subdir := filepath.Join(workspace, name)
			require.NoError(t, os.Mkdir(subdir, 0755))
			require.NoError(t, os.WriteFile(filepath.Join(subdir, "install.yaml"), []byte("name: "+name+"\n"), 0644))
		}

		out, _ := exec.RunHostCommand(DdevBin, "utility", "addon-update-checker", "--dir", workspace)
		require.Contains(t, out, "addon-one")
		require.Contains(t, out, "addon-two")
		// Both runs must report an exit code
		require.Equal(t, 2, strings.Count(out, "Exit code:"))
	})

	// Test 3: no install.yaml anywhere — must fail with a clear error
	t.Run("NoInstallYaml", func(t *testing.T) {
		dir := testcommon.CreateTmpDir(filepath.Base(t.Name()))
		defer testcommon.CleanupDir(dir)

		out, err := exec.RunHostCommand(DdevBin, "utility", "addon-update-checker", "--dir", dir)
		require.Error(t, err)
		require.Contains(t, out, "No install.yaml found")
	})
}
