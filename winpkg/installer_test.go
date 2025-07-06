//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/stretchr/testify/require"
)

const (
	testDistroName = "Ubuntu-22.04"
	installerPath  = "../.gotmp/bin/windows_amd64/ddev_windows_amd64_installer.exe"
)

// TestWindowsInstallerWSL2DockerCE tests the WSL2 with Docker CE installation path
func TestWindowsInstallerWSL2DockerCE(t *testing.T) {
	if os.Getenv("DDEV_TEST_USE_REAL_INSTALLER") == "" {
		t.Skip("Skipping installer test, set DDEV_TEST_USE_REAL_INSTALLER=true to run")
	}

	require := require.New(t)

	// Create fresh test WSL2 distro
	createTestWSL2Distro(t)
	//t.Cleanup(func() {
	//	// Cleanup any existing test distro
	//	cleanupTestDistro(t)
	//})

	// Get absolute path to installer
	wd, err := os.Getwd()
	require.NoError(err)
	installerFullPath := filepath.Join(wd, installerPath)
	require.True(fileutil.FileExists(installerFullPath), "Installer not found at %s", installerFullPath)

	// Run installer with WSL2 Docker CE option
	t.Logf("Running installer: %s", installerFullPath)
	out, err := exec.RunHostCommand(installerFullPath, "/docker-ce", fmt.Sprintf("/distro=%s", testDistroName), "/S")
	require.NoError(err, "Installer failed: %v, output: %s", err, out)
	t.Logf("Installer output: %s", out)

	// Test that ddev is installed and working
	testDdevInstallation(t)

	// Test basic ddev functionality
	testBasicDdevFunctionality(t)
}

// TestWindowsInstallerWSL2DockerDesktop tests the WSL2 with Docker Desktop installation path
func TestWindowsInstallerWSL2DockerDesktop(t *testing.T) {
	if os.Getenv("DDEV_TEST_USE_REAL_INSTALLER") == "" {
		t.Skip("Skipping installer test, set DDEV_TEST_USE_REAL_INSTALLER=true to run")
	}

	require := require.New(t)

	// Skip if Docker Desktop is not available
	if !isDockerDesktopAvailable() {
		t.Skip("Docker Desktop not available, skipping Docker Desktop installer test")
	}

	// Cleanup any existing test distro
	cleanupTestDistro(t)

	// Create fresh test WSL2 distro
	createTestWSL2Distro(t)
	defer cleanupTestDistro(t)

	// Get absolute path to installer
	wd, err := os.Getwd()
	require.NoError(err)
	installerFullPath := filepath.Join(wd, installerPath)
	require.True(fileutil.FileExists(installerFullPath), "Installer not found at %s", installerFullPath)

	// Run installer with WSL2 Docker Desktop option
	t.Logf("Running installer: %s", installerFullPath)
	out, err := exec.RunHostCommand(installerFullPath, "/docker-desktop", fmt.Sprintf("/distro=%s", testDistroName), "/S")
	require.NoError(err, "Installer failed: %v, output: %s", err, out)
	t.Logf("Installer output: %s", out)

	// Test that ddev is installed and working
	testDdevInstallation(t)

	// Test basic ddev functionality
	testBasicDdevFunctionality(t)
}

// TestWindowsInstallerTraditional tests the traditional Windows installation path
func TestWindowsInstallerTraditional(t *testing.T) {
	if os.Getenv("DDEV_TEST_USE_REAL_INSTALLER") == "" {
		t.Skip("Skipping installer test, set DDEV_TEST_USE_REAL_INSTALLER=true to run")
	}

	require := require.New(t)

	// Get absolute path to installer
	wd, err := os.Getwd()
	require.NoError(err)
	installerFullPath := filepath.Join(wd, installerPath)
	require.True(fileutil.FileExists(installerFullPath), "Installer not found at %s", installerFullPath)

	// Run installer with traditional Windows option
	t.Logf("Running installer: %s", installerFullPath)
	out, err := exec.RunHostCommand(installerFullPath, "/traditional", "/S")
	require.NoError(err, "Installer failed: %v, output: %s", err, out)
	t.Logf("Installer output: %s", out)

	// Test that ddev.exe is installed in Program Files
	programFiles := os.Getenv("PROGRAMFILES")
	ddevPath := filepath.Join(programFiles, "DDEV", "ddev.exe")
	require.True(fileutil.FileExists(ddevPath), "ddev.exe not found at %s", ddevPath)

	// Test ddev version
	out, err = exec.RunHostCommand(ddevPath, "version")
	require.NoError(err, "ddev version failed: %v, output: %s", err, out)
	require.Contains(out, "ddev version")
	t.Logf("ddev version output: %s", out)
}

// Helper functions

// cleanupTestDistro removes the test WSL2 distro if it exists
func cleanupTestDistro(t *testing.T) {
	t.Logf("Cleaning up test distro: %s", testDistroName)

	// Check if distro exists
	out, err := exec.RunHostCommand("wsl.exe", "-l", "-q")
	if err != nil {
		t.Logf("Failed to list WSL distros: %v", err)
		return
	}

	// Convert UTF-16 output to UTF-8 by removing null bytes
	cleanOut := strings.ReplaceAll(out, "\x00", "")
	t.Logf("WSL distros list: %q", cleanOut)

	if strings.Contains(cleanOut, testDistroName) {
		t.Logf("Test distro %s exists, attempting to remove", testDistroName)

		// Unregister (delete) the distro
		out, err := exec.RunHostCommand("wsl.exe", "--unregister", testDistroName)
		if err != nil {
			t.Logf("Failed to unregister distro %s: %v, output: %s", testDistroName, err, out)
		} else {
			t.Logf("Successfully removed test distro: %s", testDistroName)
		}
	}
}

// createTestWSL2Distro creates a fresh Ubuntu 22.04 WSL2 distro for testing
func createTestWSL2Distro(t *testing.T) {
	require := require.New(t)
	t.Logf("Creating test WSL2 distro: %s", testDistroName)

	// Install the WSL distro without launching
	t.Logf("Installing WSL distro %s", testDistroName)
	out, err := exec.RunHostCommand("wsl.exe", "--install", testDistroName, "--no-launch")
	require.NoError(err, "Failed to install WSL distro: %v, output: %s", err, out)

	// Wait a moment for the distro to be ready
	time.Sleep(5 * time.Second)

	// Complete distro setup with root user (avoids interactive user setup)
	t.Logf("Completing distro setup with root user")
	userProfile := os.Getenv("USERPROFILE")
	// Convert Ubuntu-22.04 to ubuntu2204.exe
	exeName := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(testDistroName, "-", ""), ".", "")) + ".exe"
	ubuntuExePath := filepath.Join(userProfile, "AppData", "Local", "Microsoft", "WindowsApps", exeName)
	out, err = exec.RunHostCommand(ubuntuExePath, "install", "--root")
	// Note: distro.exe install --root is undocumented but works, though it returns non-zero exit code
	t.Logf("Distro setup output: %s, error: %v", out, err)

	// Wait a moment for the distro to be fully registered
	time.Sleep(3 * time.Second)

	// Create an unprivileged default user
	t.Logf("Creating unprivileged default user")
	out, err = exec.RunHostCommand("wsl.exe", "-d", testDistroName, "-u", "root", "bash", "-c", "useradd -m -s /bin/bash testuser && echo 'testuser:testpass' | chpasswd && usermod -aG sudo testuser")
	require.NoError(err, "Failed to create test user: %v, output=%v", err, out)

	// Set testuser as the default user using wsl --manage
	t.Logf("Setting testuser as default user")
	_, err = exec.RunHostCommand("wsl.exe", "--manage", testDistroName, "--set-default-user", "testuser")
	require.NoError(err, "Failed to set default user: %v", err)

	t.Logf("Test WSL2 distro %s created successfully", testDistroName)
}

// testDdevInstallation verifies that ddev is properly installed in WSL2
func testDdevInstallation(t *testing.T) {
	require := require.New(t)
	t.Logf("Testing ddev installation in %s", testDistroName)

	// Test ddev version
	out, err := exec.RunHostCommand("wsl.exe", "-d", testDistroName, "ddev", "version")
	require.NoError(err, "ddev version failed: %v, output: %s", err, out)
	require.Contains(out, "DDEV version")
	t.Logf("ddev version output: %s", out)

	// Test ddev-hostname
	out, err = exec.RunHostCommand("wsl.exe", "-d", testDistroName, "ddev-hostname", "--help")
	require.NoError(err, "ddev-hostname failed: %v, output: %s", err, out)
	t.Logf("ddev-hostname available")

	// Test mkcert is available
	out, err = exec.RunHostCommand("wsl.exe", "-d", testDistroName, "mkcert", "-version")
	require.NoError(err, "mkcert not available: %v, output: %s", err, out)
	t.Logf("mkcert available: %s", strings.TrimSpace(out))
}

// testBasicDdevFunctionality tests basic ddev project creation and start
func testBasicDdevFunctionality(t *testing.T) {
	require := require.New(t)
	t.Logf("Testing basic ddev functionality in %s", testDistroName)

	projectDir := "/tmp/ddev-test-project"

	// Clean up any existing test project
	_, _ = exec.RunHostCommand("wsl.exe", "-d", testDistroName, "rm", "-rf", projectDir)

	// Create test project directory
	out, err := exec.RunHostCommand("wsl.exe", "-d", testDistroName, "mkdir", "-p", projectDir)
	require.NoError(err, "Failed to create project directory: %v, output: %s", err, out)

	// Create a simple index.html
	_, err = exec.RunHostCommand("wsl.exe", "-d", testDistroName, "bash", "-c", fmt.Sprintf("echo '<html><head><title>DDEV Test</title></head><body><h1>Hello from DDEV!</h1></body></html>' > %s/index.html", projectDir))
	require.NoError(err, "Failed to create index.html: %v", err)

	// Initialize ddev project
	out, err = exec.RunHostCommand("wsl.exe", "-d", testDistroName, "bash", "-c", fmt.Sprintf("cd %s && ddev config --auto", projectDir))
	require.NoError(err, "ddev config failed: %v, output: %s", err, out)
	t.Logf("ddev config output: %s", out)

	// Start the project
	out, err = exec.RunHostCommand("wsl.exe", "-d", testDistroName, "bash", "-c", fmt.Sprintf("cd %s && ddev start -y", projectDir))
	require.NoError(err, "ddev start failed: %v, output: %s", err, out)
	t.Logf("ddev start -y output: %s", out)

	// Wait a moment for the site to be ready
	time.Sleep(10 * time.Second)

	// Test HTTP response
	out, err = exec.RunHostCommand("wsl.exe", "-d", testDistroName, "bash", "-c", fmt.Sprintf("cd %s && curl -s https://test-project.ddev.site", projectDir))
	require.NoError(err, "curl to HTTPS site failed: %v, output: %s", err, out)
	require.Contains(out, "Hello from DDEV!")
	t.Logf("HTTPS site responding correctly")

	// Test that the site has valid HTTPS
	out, err = exec.RunHostCommand("wsl.exe", "-d", testDistroName, "bash", "-c", fmt.Sprintf("cd %s && curl -s -I https://test-project.ddev.site | head -1", projectDir))
	require.NoError(err, "HTTPS check failed: %v, output: %s", err, out)
	require.Contains(out, "200 OK")
	t.Logf("HTTPS certificate working")

	_, _ = exec.RunHostCommand("wsl.exe", "-d", testDistroName, "bash", "-c", "ddev poweroff")

	t.Logf("Basic ddev functionality test completed successfully")
}

// isDockerDesktopAvailable checks if Docker Desktop is installed and running
func isDockerDesktopAvailable() bool {
	// Check if Docker Desktop process is running
	out, err := exec.RunHostCommand("tasklist.exe", "/FI", "IMAGENAME eq Docker Desktop.exe")
	if err != nil {
		return false
	}
	return strings.Contains(out, "Docker Desktop.exe")
}
