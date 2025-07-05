//go:build windows
// +build windows

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
	testDistroName = "ddev-test-ubuntu-22.04"
	installerPath  = "../.gotmp/bin/windows_amd64/ddev_windows_amd64_installer.exe"
)

// TestWindowsInstallerWSL2DockerCE tests the WSL2 with Docker CE installation path
func TestWindowsInstallerWSL2DockerCE(t *testing.T) {
	if os.Getenv("DDEV_TEST_USE_REAL_INSTALLER") == "" {
		t.Skip("Skipping installer test, set DDEV_TEST_USE_REAL_INSTALLER=true to run")
	}

	require := require.New(t)

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

	// Run installer with WSL2 Docker CE option
	t.Logf("Running installer: %s", installerFullPath)
	out, err := exec.RunHostCommand(fmt.Sprintf(`"%s" /docker-ce /distro=%s /S`, installerFullPath, testDistroName))
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
	out, err := exec.RunHostCommand(fmt.Sprintf(`"%s" /docker-desktop /distro=%s /S`, installerFullPath, testDistroName))
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
	out, err := exec.RunHostCommand(fmt.Sprintf(`"%s" /traditional /S`, installerFullPath))
	require.NoError(err, "Installer failed: %v, output: %s", err, out)
	t.Logf("Installer output: %s", out)

	// Test that ddev.exe is installed in Program Files
	programFiles := os.Getenv("PROGRAMFILES")
	ddevPath := filepath.Join(programFiles, "DDEV", "ddev.exe")
	require.True(fileutil.FileExists(ddevPath), "ddev.exe not found at %s", ddevPath)

	// Test ddev version
	out, err = exec.RunHostCommand(fmt.Sprintf(`"%s" version`, ddevPath))
	require.NoError(err, "ddev version failed: %v, output: %s", err, out)
	require.Contains(out, "ddev version")
	t.Logf("ddev version output: %s", out)
}

// Helper functions

// cleanupTestDistro removes the test WSL2 distro if it exists
func cleanupTestDistro(t *testing.T) {
	t.Logf("Cleaning up test distro: %s", testDistroName)
	
	// Check if distro exists
	out, err := exec.RunHostCommand("wsl.exe -l -v")
	if err != nil {
		t.Logf("Failed to list WSL distros: %v", err)
		return
	}
	
	if strings.Contains(out, testDistroName) {
		// Terminate the distro first
		_, _ = exec.RunHostCommand(fmt.Sprintf("wsl.exe --terminate %s", testDistroName))
		time.Sleep(2 * time.Second)
		
		// Unregister (delete) the distro
		out, err := exec.RunHostCommand(fmt.Sprintf("wsl.exe --unregister %s", testDistroName))
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
	
	// Download Ubuntu 22.04 appx if not present
	tempDir := os.TempDir()
	ubuntuAppx := filepath.Join(tempDir, "Ubuntu2204.appx")
	
	if !fileutil.FileExists(ubuntuAppx) {
		t.Logf("Downloading Ubuntu 22.04...")
		// Use PowerShell to download Ubuntu 22.04
		cmd := fmt.Sprintf(`powershell.exe -Command "Invoke-WebRequest -Uri 'https://aka.ms/wslubuntu2204' -OutFile '%s'"`, ubuntuAppx)
		out, err := exec.RunHostCommand(cmd)
		require.NoError(err, "Failed to download Ubuntu 22.04: %v, output: %s", err, out)
	}
	
	// Install the WSL distro
	t.Logf("Installing WSL distro from %s", ubuntuAppx)
	cmd := fmt.Sprintf(`wsl.exe --import %s "%s" "%s"`, testDistroName, filepath.Join(tempDir, testDistroName), ubuntuAppx)
	out, err := exec.RunHostCommand(cmd)
	require.NoError(err, "Failed to import WSL distro: %v, output: %s", err, out)
	
	// Wait a moment for the distro to be ready
	time.Sleep(5 * time.Second)
	
	// Verify the distro is WSL2
	out, err = exec.RunHostCommand("wsl.exe -l -v")
	require.NoError(err, "Failed to list WSL distros: %v", err)
	require.Contains(out, testDistroName)
	require.Contains(out, "2", "Test distro should be WSL2, got: %s", out)
	
	// Set up basic user in the distro
	t.Logf("Setting up test user in distro")
	cmd = fmt.Sprintf(`wsl.exe -d %s -u root bash -c "useradd -m -s /bin/bash testuser && echo 'testuser:testpass' | chpasswd && usermod -aG sudo testuser"`, testDistroName)
	_, err = exec.RunHostCommand(cmd)
	require.NoError(err, "Failed to set up test user: %v", err)
	
	t.Logf("Test WSL2 distro %s created successfully", testDistroName)
}

// testDdevInstallation verifies that ddev is properly installed in WSL2
func testDdevInstallation(t *testing.T) {
	require := require.New(t)
	t.Logf("Testing ddev installation in %s", testDistroName)
	
	// Test ddev version
	out, err := exec.RunHostCommand(fmt.Sprintf("wsl.exe -d %s ddev version", testDistroName))
	require.NoError(err, "ddev version failed: %v, output: %s", err, out)
	require.Contains(out, "ddev version")
	t.Logf("ddev version output: %s", out)
	
	// Test ddev-hostname
	out, err = exec.RunHostCommand(fmt.Sprintf("wsl.exe -d %s ddev-hostname --help", testDistroName))
	require.NoError(err, "ddev-hostname failed: %v, output: %s", err, out)
	t.Logf("ddev-hostname available")
	
	// Test mkcert is available
	out, err = exec.RunHostCommand(fmt.Sprintf("wsl.exe -d %s mkcert -version", testDistroName))
	require.NoError(err, "mkcert not available: %v, output: %s", err, out)
	t.Logf("mkcert available: %s", strings.TrimSpace(out))
}

// testBasicDdevFunctionality tests basic ddev project creation and start
func testBasicDdevFunctionality(t *testing.T) {
	require := require.New(t)
	t.Logf("Testing basic ddev functionality in %s", testDistroName)
	
	projectDir := "/tmp/ddev-test-project"
	
	// Clean up any existing test project
	_, _ = exec.RunHostCommand(fmt.Sprintf("wsl.exe -d %s rm -rf %s", testDistroName, projectDir))
	
	// Create test project directory
	out, err := exec.RunHostCommand(fmt.Sprintf("wsl.exe -d %s mkdir -p %s", testDistroName, projectDir))
	require.NoError(err, "Failed to create project directory: %v, output: %s", err, out)
	
	// Create a simple index.html
	cmd := fmt.Sprintf(`wsl.exe -d %s bash -c "echo '<html><head><title>DDEV Test</title></head><body><h1>Hello from DDEV!</h1></body></html>' > %s/index.html"`, testDistroName, projectDir)
	_, err = exec.RunHostCommand(cmd)
	require.NoError(err, "Failed to create index.html: %v", err)
	
	// Initialize ddev project
	cmd = fmt.Sprintf(`wsl.exe -d %s bash -c "cd %s && ddev config --project-type=html --project-name=test-project --auto"`, testDistroName, projectDir)
	out, err = exec.RunHostCommand(cmd)
	require.NoError(err, "ddev config failed: %v, output: %s", err, out)
	t.Logf("ddev config output: %s", out)
	
	// Start the project
	cmd = fmt.Sprintf(`wsl.exe -d %s bash -c "cd %s && ddev start"`, testDistroName, projectDir)
	out, err = exec.RunHostCommand(cmd)
	require.NoError(err, "ddev start failed: %v, output: %s", err, out)
	t.Logf("ddev start output: %s", out)
	
	// Wait a moment for the site to be ready
	time.Sleep(10 * time.Second)
	
	// Test HTTP response
	cmd = fmt.Sprintf(`wsl.exe -d %s bash -c "cd %s && curl -s https://test-project.ddev.site"`, testDistroName, projectDir)
	out, err = exec.RunHostCommand(cmd)
	require.NoError(err, "curl to HTTPS site failed: %v, output: %s", err, out)
	require.Contains(out, "Hello from DDEV!")
	t.Logf("HTTPS site responding correctly")
	
	// Test that the site has valid HTTPS
	cmd = fmt.Sprintf(`wsl.exe -d %s bash -c "cd %s && curl -s -I https://test-project.ddev.site | head -1"`, testDistroName, projectDir)
	out, err = exec.RunHostCommand(cmd)
	require.NoError(err, "HTTPS check failed: %v, output: %s", err, out)
	require.Contains(out, "200 OK")
	t.Logf("HTTPS certificate working")
	
	// Clean up - stop the project
	cmd = fmt.Sprintf(`wsl.exe -d %s bash -c "cd %s && ddev stop"`, testDistroName, projectDir)
	_, _ = exec.RunHostCommand(cmd)
	
	// Remove test project
	_, _ = exec.RunHostCommand(fmt.Sprintf("wsl.exe -d %s rm -rf %s", testDistroName, projectDir))
	
	t.Logf("Basic ddev functionality test completed successfully")
}

// isDockerDesktopAvailable checks if Docker Desktop is installed and running
func isDockerDesktopAvailable() bool {
	// Check if Docker Desktop process is running
	out, err := exec.RunHostCommand(`tasklist.exe /FI "IMAGENAME eq Docker Desktop.exe"`)
	if err != nil {
		return false
	}
	return strings.Contains(out, "Docker Desktop.exe")
}