package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

// TestUtilityDiagnoseCmd tests basic functionality of ddev utility diagnose
func TestUtilityDiagnoseCmd(t *testing.T) {
	// Test in non-project directory
	t.Run("NonProjectDirectory", func(t *testing.T) {
		origDir, _ := os.Getwd()
		tmpDir := testcommon.CreateTmpDir("TestUtilityDiagnose-NonProject")
		t.Cleanup(func() {
			_ = os.RemoveAll(tmpDir)
			_ = os.Chdir(origDir)
		})

		err := os.Chdir(tmpDir)
		require.NoError(t, err)

		// Should run but warn about not being in a project
		out, _ := exec.RunHostCommand(DdevBin, "utility", "diagnose")
		// May fail due to not being in project, but should produce output
		require.Contains(t, out, "DDEV Diagnostic Report")
		require.Contains(t, out, "Environment")
		require.Contains(t, out, "Docker Environment")
	})

	// Test with basic project
	t.Run("BasicProject", func(t *testing.T) {
		origDir, _ := os.Getwd()
		testDir := filepath.Join(origDir, "testdata", t.Name())
		err := os.Chdir(testDir)
		require.NoError(t, err)
		t.Cleanup(func() {
			err = os.Chdir(origDir)
			require.NoError(t, err)
		})

		out, err := exec.RunHostCommand(DdevBin, "utility", "diagnose")
		require.NoError(t, err)
		require.Contains(t, out, "DDEV Diagnostic Report")
		require.Contains(t, out, "Current Project")
		require.Contains(t, out, "Name: test-diagnose-basic")
		require.Contains(t, out, "Type: php")
	})

	// Test with customizations
	t.Run("ProjectWithCustomizations", func(t *testing.T) {
		origDir, _ := os.Getwd()
		testDir := filepath.Join(origDir, "testdata", t.Name())
		err := os.Chdir(testDir)
		require.NoError(t, err)
		t.Cleanup(func() {
			err = os.Chdir(origDir)
			require.NoError(t, err)
		})

		out, err := exec.RunHostCommand(DdevBin, "utility", "diagnose")
		require.NoError(t, err)
		require.Contains(t, out, "customized configuration file(s)")
		require.Contains(t, out, ".ddev/.env")
	})

	// Test outside home directory warning
	t.Run("OutsideHomeDirectory", func(t *testing.T) {
		// Only test on systems where we can easily test outside home
		if os.Getenv("DDEV_TEST_OUTSIDE_HOME") != "true" {
			t.Skip("Skipping outside home test - set DDEV_TEST_OUTSIDE_HOME=true to enable")
		}

		origDir, _ := os.Getwd()
		tmpDir := testcommon.CreateTmpDir("TestUtilityDiagnose-OutsideHome")
		t.Cleanup(func() {
			_ = os.RemoveAll(tmpDir)
			_ = os.Chdir(origDir)
		})

		// Copy test project to /tmp or similar
		testDir := filepath.Join(origDir, "testdata", "TestUtilityDiagnoseCmd/BasicProject")
		err := fileutil.CopyDir(testDir, tmpDir)
		require.NoError(t, err)

		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		out, _ := exec.RunHostCommand(DdevBin, "utility", "diagnose")
		// Should warn about not being in home directory
		if os.Getenv("HOME") != "" && tmpDir[:len(os.Getenv("HOME"))] != os.Getenv("HOME") {
			require.Contains(t, out, "should usually be in a subdirectory of your home directory")
		}
	})
}
