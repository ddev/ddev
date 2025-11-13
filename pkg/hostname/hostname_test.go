package hostname

// This file contains tests for ddev-hostname functionality that require passwordless sudo.
// These tests only run in CI environments (GitHub Actions) where passwordless sudo is available.
// The tests verify that ddev-hostname can properly add and remove hostnames from the hosts file
// without user interaction when DDEV_NONINTERACTIVE is unset.
//
// Related issue: https://github.com/ddev/ddev/issues/7790

import (
	"os"
	"os/exec"
	"testing"

	exec2 "github.com/ddev/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDdevHostnameWithPasswordlessSudo tests ddev-hostname functionality when passwordless sudo is available.
// This test only runs when:
// 1. We're in a CI environment (GITHUB_ACTIONS=true)
// 2. Passwordless sudo is available
func TestDdevHostnameWithPasswordlessSudo(t *testing.T) {
	// Skip if not in GitHub Actions
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		t.Skip("Skipping because not in GitHub Actions (GITHUB_ACTIONS != true)")
	}

	// Check if passwordless sudo is available
	cmd := exec.Command("sudo", "-n", "true")
	err := cmd.Run()
	if err != nil {
		t.Skip("Skipping because passwordless sudo is not available")
	}

	assert := asrt.New(t)

	// Save and restore original DDEV_NONINTERACTIVE value
	origNonInteractive := os.Getenv("DDEV_NONINTERACTIVE")
	t.Cleanup(func() {
		_ = os.Setenv("DDEV_NONINTERACTIVE", origNonInteractive)
	})

	// Unset DDEV_NONINTERACTIVE to allow hostname manipulation
	err = os.Setenv("DDEV_NONINTERACTIVE", "")
	require.NoError(t, err)

	// Use a unique hostname for testing
	testHostname := "test-ddev-hostname.local"
	testIP := "127.0.0.1"

	// Get the binary name
	binary := GetDdevHostnameBinary()

	// Clean up any existing entry first
	cleanupCmd := exec.Command(binary, "--remove", testHostname, testIP)
	_ = cleanupCmd.Run() // Ignore error if entry doesn't exist

	t.Cleanup(func() {
		// Clean up test hostname at the end
		cleanupCmd := exec.Command(binary, "--remove", testHostname, testIP)
		_ = cleanupCmd.Run()
	})

	// Test adding hostname
	addOut, err := exec2.RunHostCommand(binary, testHostname, testIP)
	assert.NoError(err, "ddev-hostname add should succeed, output: %s", addOut)
	assert.Contains(addOut, "Added", "output should indicate hostname was added: %s", addOut)

	// Test checking hostname exists
	checkCmd := exec.Command(binary, "--check", testHostname, testIP)
	err = checkCmd.Run()
	assert.NoError(err, "ddev-hostname --check should succeed for existing entry")

	// Test that hostname is actually in hosts file
	exists, err := IsHostnameInHostsFile(testHostname)
	assert.NoError(err, "IsHostnameInHostsFile should not error")
	assert.True(exists, "hostname should be in hosts file")

	// Test removing hostname
	removeOut, err := exec2.RunHostCommand(binary, "--remove", testHostname, testIP)
	assert.NoError(err, "ddev-hostname --remove should succeed, output: %s", removeOut)
	assert.Contains(removeOut, "Removed", "output should indicate hostname was removed: %s", removeOut)

	// Verify hostname is removed
	checkCmd = exec.Command(binary, "--check", testHostname, testIP)
	err = checkCmd.Run()
	assert.Error(err, "ddev-hostname --check should fail for removed entry")

	// Verify hostname is not in hosts file
	exists, err = IsHostnameInHostsFile(testHostname)
	assert.NoError(err, "IsHostnameInHostsFile should not error")
	assert.False(exists, "hostname should not be in hosts file after removal")
}

// TestElevateToAddRemoveHostEntry tests the ElevateToAddHostEntry and ElevateToRemoveHostEntry functions.
// This test only runs when:
// 1. We're in a CI environment (GITHUB_ACTIONS=true)
// 2. Passwordless sudo is available
func TestElevateToAddRemoveHostEntry(t *testing.T) {
	// Skip if not in GitHub Actions
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		t.Skip("Skipping because not in GitHub Actions (GITHUB_ACTIONS != true)")
	}

	// Check if passwordless sudo is available
	cmd := exec.Command("sudo", "-n", "true")
	err := cmd.Run()
	if err != nil {
		t.Skip("Skipping because passwordless sudo is not available")
	}

	assert := asrt.New(t)

	// Save and restore original DDEV_NONINTERACTIVE value
	origNonInteractive := os.Getenv("DDEV_NONINTERACTIVE")
	t.Cleanup(func() {
		_ = os.Setenv("DDEV_NONINTERACTIVE", origNonInteractive)
	})

	// Unset DDEV_NONINTERACTIVE to allow hostname manipulation
	err = os.Setenv("DDEV_NONINTERACTIVE", "")
	require.NoError(t, err)

	// Use a unique hostname for testing
	testHostname := "test-elevate-hostname.local"
	testIP := "127.0.0.1"

	// Clean up any existing entry first
	_, _ = ElevateToRemoveHostEntry(testHostname, testIP) // Ignore error if entry doesn't exist

	t.Cleanup(func() {
		// Clean up test hostname at the end
		_, _ = ElevateToRemoveHostEntry(testHostname, testIP)
	})

	// Test ElevateToAddHostEntry
	out, err := ElevateToAddHostEntry(testHostname, testIP)
	assert.NoError(err, "ElevateToAddHostEntry should succeed, output: %s", out)
	if err == nil {
		assert.Contains(out, "Added", "output should indicate hostname was added: %s", out)
	}

	// Verify hostname is in hosts file
	exists, err := IsHostnameInHostsFile(testHostname)
	assert.NoError(err, "IsHostnameInHostsFile should not error")
	assert.True(exists, "hostname should be in hosts file")

	// Test ElevateToRemoveHostEntry
	out, err = ElevateToRemoveHostEntry(testHostname, testIP)
	assert.NoError(err, "ElevateToRemoveHostEntry should succeed, output: %s", out)
	if err == nil {
		assert.Contains(out, "Removed", "output should indicate hostname was removed: %s", out)
	}

	// Verify hostname is not in hosts file
	exists, err = IsHostnameInHostsFile(testHostname)
	assert.NoError(err, "IsHostnameInHostsFile should not error")
	assert.False(exists, "hostname should not be in hosts file after removal")
}
