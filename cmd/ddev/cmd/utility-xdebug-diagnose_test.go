package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/stretchr/testify/require"
)

// TestCmdXdebugDiagnose tests the `ddev utility xdebug-diagnose` command
func TestCmdXdebugDiagnose(t *testing.T) {
	pwd, _ := os.Getwd()
	// Use first test site
	v := TestSites[0]

	err := os.Chdir(v.Dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		// Ensure xdebug is off after test
		_, _ = exec.RunCommand(DdevBin, []string{"xdebug", "off"})
		err := os.Chdir(pwd)
		require.NoError(t, err)
	})

	// Start the project if not already running
	_, err = exec.RunHostCommand(DdevBin, "start")
	require.NoError(t, err)

	// Run xdebug-diagnose command
	out, _ := exec.RunHostCommand(DdevBin, "utility", "xdebug-diagnose")
	// Command may exit with 0 or 1 depending on diagnostic results
	// We just check that it runs and produces expected output
	t.Logf("xdebug-diagnose output: %s", out)

	// Check for expected sections in output
	require.Contains(t, out, "Xdebug Diagnostics for Project")
	require.Contains(t, out, "Port 9003 Pre-Check")
	require.Contains(t, out, "host.docker.internal Configuration")
	require.Contains(t, out, "Global Configuration")
	require.Contains(t, out, "Connection Test")
	require.Contains(t, out, "Xdebug Status")
	require.Contains(t, out, "Summary")

	// Check that we get information about host.docker.internal
	require.Contains(t, out, "IP address:")
	require.Contains(t, out, "Derivation:")

	// Check for xdebug_ide_location check
	require.True(t, strings.Contains(out, "xdebug_ide_location") || strings.Contains(out, "Global Configuration"))
}

// TestCmdXdebugDiagnoseNotInProject tests running the command outside a project
func TestCmdXdebugDiagnoseNotInProject(t *testing.T) {
	tmpDir := t.TempDir()
	pwd, _ := os.Getwd()

	err := os.Chdir(tmpDir)
	require.NoError(t, err)

	t.Cleanup(func() {
		err := os.Chdir(pwd)
		require.NoError(t, err)
	})

	// Run xdebug-diagnose command outside a project
	out, err := exec.RunHostCommand(DdevBin, "utility", "xdebug-diagnose")
	require.Error(t, err)
	require.Contains(t, out, "Not in a DDEV project directory")
}

// TestCmdXdebugDiagnoseInteractiveFlag tests that --interactive flag exists and
// falls back to standard mode when DDEV_NONINTERACTIVE is set
func TestCmdXdebugDiagnoseInteractiveFlag(t *testing.T) {
	pwd, _ := os.Getwd()
	v := TestSites[0]

	err := os.Chdir(v.Dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		err := os.Chdir(pwd)
		require.NoError(t, err)
	})

	// Test that --interactive flag is recognized (with DDEV_NONINTERACTIVE to skip prompts)
	t.Setenv("DDEV_NONINTERACTIVE", "true")
	out, _ := exec.RunHostCommand(DdevBin, "utility", "xdebug-diagnose", "--interactive")

	// Should see the fallback warning when DDEV_NONINTERACTIVE is set
	require.Contains(t, out, "Interactive mode requested but DDEV_NONINTERACTIVE is set")
	// Should still run standard diagnostics
	require.Contains(t, out, "Xdebug Diagnostics for Project")
}
