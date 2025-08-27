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
	"github.com/ddev/ddev/pkg/mkcert"
	"github.com/stretchr/testify/require"
)

const (
	installerPath = "../.gotmp/bin/windows_amd64/ddev_windows_amd64_installer.exe"
)

// TestWindowsInstallerWSL2 tests WSL2 installation paths using a test matrix
func TestWindowsInstallerWSL2(t *testing.T) {
	if os.Getenv("DDEV_TEST_USE_REAL_INSTALLER") == "" {
		t.Skip("Skipping installer test, set DDEV_TEST_USE_REAL_INSTALLER=true to run")
	}

	testCases := []struct {
		name          string
		distro        string
		installerArgs []string
		skipCondition func() bool
	}{
		{
			name:          "DockerCE",
			distro:        "TestDockerCE",
			installerArgs: []string{"/docker-ce", "/distro=TestDockerCE", "/S"},
			skipCondition: func() bool { return false }, // always run
		},
		{
			name:          "DockerDesktop",
			distro:        "TestDesktop",
			installerArgs: []string{"/docker-desktop", "/distro=TestDesktop", "/S"},
			skipCondition: func() bool { return !isDockerProviderAvailable("TestDesktop") },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipCondition() {
				t.Skipf("Skipping %s test - Desktop distro must have integration with Rancher/Docker Desktop", tc.name)
			}

			require := require.New(t)

			// Create fresh test WSL2 distro
			cleanupTestEnv(t, tc.distro)
			configureTestWSL2Distro(t, tc.distro)

			// Ensure ddev is powered off after this test case, even if it fails
			t.Cleanup(func() {
				t.Logf("Cleaning up %s test - powering off ddev", tc.name)
				_, _ = exec.RunHostCommand("wsl.exe", "-d", tc.distro, "bash", "-c", "ddev poweroff")
				_, _ = exec.RunHostCommand("wsl.exe", "-d", tc.distro, "-u", "root", "bash", "-c", "apt-get remove -y ddev ddev-wsl2 docker-ce-cli docker-ce")

				// Install system ddev to ensure subsequent tests have a working ddev
				t.Logf("Installing system ddev for subsequent tests")
				installSystemDdev(t)
			})

			// Get absolute path to installer
			wd, err := os.Getwd()
			require.NoError(err)
			installerFullPath := filepath.Join(wd, installerPath)
			require.True(fileutil.FileExists(installerFullPath), "Installer not found at %s", installerFullPath)

			// Run installer with specified args
			t.Logf("Running installer: %s %v", installerFullPath, tc.installerArgs)

			// Add some debugging before installer run
			t.Logf("Pre-installer: checking system state...")
			out, _ := exec.RunHostCommand("tasklist.exe", "/FI", "IMAGENAME eq msiexec.exe")
			t.Logf("MSI processes running: %s", out)

			out, err = exec.RunHostCommand(installerFullPath, tc.installerArgs...)
			if err != nil {
				t.Logf("Installer failed with error trying to run with '%s': %v", installerFullPath+" "+strings.Join(tc.installerArgs, " "), err)
				t.Logf("Installer output: %s", out)

				// Check for specific error patterns
				if strings.Contains(err.Error(), "0xc0000005") {
					t.Logf("ACCESS VIOLATION detected in installer")
					// Try to get more info about what was running
					procOut, _ := exec.RunHostCommand("tasklist.exe", "/FI", "IMAGENAME eq ddev_windows_amd64_installer.exe")
					t.Logf("Installer processes: %s", procOut)
				}

				require.NoError(err, "Installer failed: %v, output: %s", err, out)
			}
			t.Logf("Installer completed successfully but may be asynchronous, output: %s", out)

			// Check $CAROOT env var on Windows side
			caRootOut, caRootErr := exec.RunHostCommand("cmd.exe", "/c", "echo %CAROOT%")
			t.Logf("Windows $CAROOT env var: %s (err: %v)", strings.TrimSpace(caRootOut), caRootErr)

			// 1. Check $WSLENV env var on Windows side
			out, err = exec.RunHostCommand("cmd.exe", "/c", "echo %WSLENV%")
			t.Logf("Windows WSLENV env var: %s (err: %v)", strings.TrimSpace(out), caRootErr)

			// Check user environment variables in registry
			userCarootReg, userCarootRegErr := exec.RunHostCommand("reg.exe", "query", "HKEY_CURRENT_USER\\Environment", "/v", "CAROOT")
			t.Logf("Registry HKCU\\Environment CAROOT: %s (err: %v)", strings.TrimSpace(userCarootReg), userCarootRegErr)

			userWslenvReg, userWslenvRegErr := exec.RunHostCommand("reg.exe", "query", "HKEY_CURRENT_USER\\Environment", "/v", "WSLENV")
			t.Logf("Registry HKCU\\Environment WSLENV: %s (err: %v)", strings.TrimSpace(userWslenvReg), userWslenvRegErr)

			// Check system environment variables in registry
			systemCarootReg, systemCarootRegErr := exec.RunHostCommand("reg.exe", "query", "HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment", "/v", "CAROOT")
			t.Logf("Registry HKLM\\System\\Environment CAROOT: %s (err: %v)", strings.TrimSpace(systemCarootReg), systemCarootRegErr)

			systemWslenvReg, systemWslenvRegErr := exec.RunHostCommand("reg.exe", "query", "HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment", "/v", "WSLENV")
			t.Logf("Registry HKLM\\System\\Environment WSLENV: %s (err: %v)", strings.TrimSpace(systemWslenvReg), systemWslenvRegErr)

			// Check DDEV integrated mkcert CAROOT
			ca := mkcert.NewCA()
			mkcertOut := ca.GetCAROOT()
			t.Logf("DDEV integrated mkcert CAROOT: %s", strings.TrimSpace(mkcertOut))

			// Check WSLENV inside the distro
			out, err = exec.RunHostCommand("wsl.exe", "-d", tc.distro, "bash", "-c", "echo $WSLENV")
			t.Logf("WSL distro $WSLENV: %s (err: %v)", strings.TrimSpace(out), err)

			distroCAROOTOut, distroCAROOTErr := exec.RunHostCommand("wsl.exe", "-d", tc.distro, "bash", "-c", "echo $CAROOT")
			t.Logf("WSL distro $CAROOT: %s (err: %v)", strings.TrimSpace(distroCAROOTOut), distroCAROOTErr)

			// Wait for installation completion by monitoring status file
			const maxTries = 60
			statusFile := "/tmp/ddev_installation_status.txt"

			for i := 0; i < maxTries; i++ {
				out, err := exec.RunHostCommand("wsl.exe", "-d", tc.distro, "bash", "-c", "tail -1l "+statusFile+" 2>/dev/null")
				if err == nil {
					t.Logf("Installation status on try %d: %s", i, strings.TrimSpace(out))
					break
				}
				out, err = exec.RunHostCommand("wsl.exe", "-d", tc.distro, "bash", "-c", "grep '^COMPLETED' "+statusFile+" 2>/dev/null")
				if err == nil {
					t.Logf("Installation completion confirmed: %s", strings.TrimSpace(out))
					break
				}

				// Check for errors
				errOut, errErr := exec.RunHostCommand("wsl.exe", "-d", tc.distro, "bash", "-c", "grep '^ERROR' "+statusFile+" 2>/dev/null")
				if errErr == nil {
					require.Error(errErr, "Installation failed", "Error found in status file: %s", strings.TrimSpace(errOut))
				}
				if i == maxTries-1 {
					require.Less(i, maxTries, "Installation timeout", "No completion marker found after %d tries", i)
				}

				time.Sleep(1 * time.Second)
			}

			out, err = exec.RunHostCommand("wsl.exe", "-d", tc.distro, "bash", "-c", "ddev config global --instrumentation-opt-in=false")
			require.NoError(err, "Failed to set global instrumentation opt-in: %v, output: %s", err, out)

			// Check if ddev is available to verify installer waited for completion
			out, err = exec.RunHostCommand("wsl.exe", "-d", tc.distro, "bash", "-c", "ddev version")
			require.NoError(err, "ddev version check failed after installer: %v, output: %s", err, out)

			// Test that ddev is installed and working
			testDdevInstallation(t, tc.distro)

			// Test basic ddev functionality
			testBasicDdevFunctionality(t, tc.distro)
		})
	}
}

// TestWindowsInstallerTraditional tests the Traditional Windows installation path
func TestWindowsInstallerTraditional(t *testing.T) {
	if os.Getenv("DDEV_TEST_USE_REAL_INSTALLER") == "" {
		t.Skip("Skipping installer test, set DDEV_TEST_USE_REAL_INSTALLER=true to run")
	}
	// Check if Docker Desktop is working on Windows
	if !isDockerDesktopWorkingOnWindows() {
		t.Skip("Skipping Traditional Windows test - Docker Desktop not working on Windows")
	}

	require := require.New(t)

	// Clean up any existing DDEV installation
	cleanupTraditionalWindowsEnv(t)

	// Ensure cleanup after test
	t.Cleanup(func() {
		t.Logf("Cleaning up Traditional Windows test")
		cleanupTraditionalWindowsEnv(t)

		// Install system ddev to ensure subsequent tests have a working ddev
		t.Logf("Installing system ddev for subsequent tests")
		installSystemDdev(t)
	})

	// Get absolute path to installer
	wd, err := os.Getwd()
	require.NoError(err)
	installerFullPath := filepath.Join(wd, installerPath)
	require.True(fileutil.FileExists(installerFullPath), "Installer not found at %s", installerFullPath)

	// Run installer with Traditional Windows option
	t.Logf("Running installer: %s /traditional /S", installerFullPath)
	out, err := exec.RunHostCommand(installerFullPath, "/traditional", "/S")
	require.NoError(err, "Installer failed: %v, output: %s", err, out)
	t.Logf("Installer output: %s", out)

	// Test that ddev is installed and working on Windows
	testDdevTraditionalInstallation(t)

	// Test basic ddev functionality on Windows
	testBasicDdevTraditionalFunctionality(t)
}

// Helper functions

// cleanupTestEnv removes the test WSL2 distro and runs the uninstaller if it exists
func cleanupTestEnv(t *testing.T, distroName string) {
	t.Logf("Cleaning up test environment")

	// First, run the uninstaller to clean up Windows-side components
	// Try common installation locations for the uninstaller
	possiblePaths := []string{
		`C:\Program Files\DDEV\ddev_uninstall.exe`,
	}

	var uninstallerPath string
	for _, path := range possiblePaths {
		if fileutil.FileExists(path) {
			uninstallerPath = path
			break
		}
	}

	if uninstallerPath != "" {
		t.Logf("Running uninstaller: %s", uninstallerPath)
		out, err := exec.RunHostCommand(uninstallerPath, "/S")
		t.Logf("Uninstaller result - err: %v, output: %s", err, out)
	} else {
		t.Logf("No uninstaller found (DDEV may not be installed yet)")
	}

	// Clean up test distro
	t.Logf("Cleaning up test distro: %s", distroName)

	// Check if distro exists
	out, err := exec.RunHostCommand("wsl.exe", "-l", "-q")
	if err != nil {
		t.Logf("Failed to list WSL distros: %v", err)
		return
	}

	// Convert UTF-16 output to UTF-8 by removing null bytes
	cleanOut := strings.ReplaceAll(out, "\x00", "")
	//t.Logf("WSL distros list: %q", cleanOut)

	if strings.Contains(cleanOut, distroName) {

		// Get distro back to a fairly normal pre-ddev state.
		// Makes test run much faster than completely deleting the distro.
		out, _ := exec.RunHostCommand("wsl.exe", "-d", distroName, "bash", "-c", "(ddev poweroff 2>/dev/null || true) && (ddev stop --unlist -a 2>/dev/null) && rm -rf ~/tp")
		t.Logf("ddev poweroff/stop/unlist: err=%v, output: %s", err, out)

		out, err = exec.RunHostCommand("wsl.exe", "-d", distroName, "-u", "root", "bash", "-c", "(apt-get remove -y ddev ddev-wsl2 docker-ce-cli docker-ce 2>/dev/null)")
		t.Logf("distro cleanup: err=%v, output: %s", err, out)
	}
}

// configureTestWSL2Distro configures an existing Ubuntu WSL2 distro for testing
func configureTestWSL2Distro(t *testing.T, distroName string) {
	require := require.New(t)
	t.Logf("Configuring test distro: %s", distroName)

	// Check if distro exists, if not install it
	out, err := exec.RunHostCommand("wsl.exe", "-l", "-q")
	if err != nil {
		t.Logf("Failed to list WSL distros: %v", err)
		require.NoError(err, "Failed to list WSL distros")
	}

	// Convert UTF-16 output to UTF-8 by removing null bytes
	cleanOut := strings.ReplaceAll(out, "\x00", "")
	if !strings.Contains(cleanOut, distroName) {
		// Install the WSL distro without launching
		t.Logf("Installing WSL distro %s", distroName)
		out, err := exec.RunHostCommand("wsl.exe", "--install", distroName, "--no-launch")
		require.NoError(err, "Failed to install WSL distro: %v, output: %s", err, out)

		// Complete distro setup with root user (avoids interactive user setup)
		t.Logf("Completing distro setup with root user only")
		userProfile := os.Getenv("USERPROFILE")
		// Convert Ubuntu-22.04 to ubuntu2204.exe
		exeName := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(distroName, "-", ""), ".", "")) + ".exe"
		ubuntuExePath := filepath.Join(userProfile, "AppData", "Local", "Microsoft", "WindowsApps", exeName)
		out, err = exec.RunHostCommand(ubuntuExePath, "install", "--root")
		// Note: distro.exe install --root is undocumented but works, though it returns non-zero exit code
		t.Logf("Distro setup output: %s, error: %v", out, err)

		// Wait a moment for the distro to be fully registered
		time.Sleep(1 * time.Second)
	}

	// Create an unprivileged default user if it doesn't exist
	t.Logf("Ensuring unprivileged default user exists")
	out, err = exec.RunHostCommand("wsl.exe", "-d", distroName, "-u", "root", "bash", "-c", "if ! id -u testuser; then useradd -m -s /bin/bash testuser && echo 'testuser:testpass' | chpasswd && usermod -aG sudo testuser; fi")
	require.NoError(err, "Failed to create test user: %v, output=%v", err, out)

	// Set testuser as the default user using wsl --manage
	t.Logf("Setting testuser as default user")
	_, err = exec.RunHostCommand("wsl.exe", "--manage", distroName, "--set-default-user", "testuser")
	require.NoError(err, "Failed to set default user: %v", err)

	t.Logf("Test WSL2 distro %s configured successfully", distroName)
}

// testDdevInstallation verifies that ddev is properly installed in WSL2
func testDdevInstallation(t *testing.T, distroName string) {
	require := require.New(t)
	t.Logf("Testing ddev installation in %s", distroName)

	// Test ddev version
	out, err := exec.RunHostCommand("wsl.exe", "-d", distroName, "bash", "-c", "ddev version | egrep 'DDEV|docker'")
	require.NoError(err, "ddev version failed: %v, output: %s", err, out)
	require.Contains(out, "DDEV version")
	t.Logf("ddev version output: %s", out)

	// Test ddev-hostname
	out, err = exec.RunHostCommand("wsl.exe", "-d", distroName, "ddev-hostname", "--help")
	require.NoError(err, "ddev-hostname failed: %v, output: %s", err, out)
	t.Logf("ddev-hostname available")

	out, err = exec.RunHostCommand("wsl.exe", "-d", distroName, "command", "-v", "ddev-hostname.exe")
	require.NoError(err, "ddev-hostname.exe failed: %v, output: %s", err, out)
	out = strings.TrimSpace(out)
	require.EqualValues("/usr/bin/ddev-hostname.exe", out, "Expected ddev-hostname.exe to be at /usr/bin/ddev-hostname.exe, got %s", out)
	t.Logf("ddev-hostname.exe available at %s", out)
}

// testBasicDdevFunctionality tests basic ddev project creation and start
func testBasicDdevFunctionality(t *testing.T, distroName string) {
	require := require.New(t)
	t.Logf("Testing basic ddev functionality in %s", distroName)

	projectDir := "~/tp"
	projectName := "tp"

	// Clean up any existing test project
	_, _ = exec.RunHostCommand("wsl.exe", "-d", distroName, "rm", "-rf", projectDir)

	// Create test project directory
	out, err := exec.RunHostCommand("wsl.exe", "-d", distroName, "mkdir", "-p", projectDir)
	require.NoError(err, "Failed to create project directory: %v, output: %s", err, out)

	// Create a simple index.html
	_, err = exec.RunHostCommand("wsl.exe", "-d", distroName, "bash", "-c", fmt.Sprintf("echo 'Hello from DDEV!' > %s/index.html", projectDir))
	require.NoError(err, "Failed to create index.html: %v", err)

	// Initialize ddev project
	out, err = exec.RunHostCommand("wsl.exe", "-d", distroName, "bash", "-c", fmt.Sprintf("cd %s && ddev config --auto && ddev start -y", projectDir))
	require.NoError(err, "ddev config/start failed: %v, output: %s", err, out)
	t.Logf("ddev config/start output: %s", out)

	// Test HTTP response from inside WSL distro
	insideCurl := fmt.Sprintf("curl -s http://%s.ddev.site", projectName)
	out, err = exec.RunHostCommand("wsl.exe", "-d", distroName, "bash", "-c", insideCurl)
	require.NoError(err, "`%s` failed inside distro: %v, output: %s", insideCurl, err, out)
	require.Contains(out, "Hello from DDEV!")
	t.Logf("HTTPS project responding correctly inside distro")

	// Test using windows PowerShell to check HTTPS
	psInvoke := fmt.Sprintf("powershell.exe -NoProfile -ExecutionPolicy Bypass -Command Invoke-RestMethod 'https://%s.ddev.site' -ErrorAction Stop", projectName)
	out, err = exec.RunHostCommand("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", fmt.Sprintf("Invoke-RestMethod 'https://%s.ddev.site' -ErrorAction Stop", projectName))
	require.NoError(err, "HTTPS check from Windows failed (`%s`) (note that mkcert.exe -install must be run previously on test runner): %v, output: %s", psInvoke, err, out)
	require.Contains(out, "Hello from DDEV!")
	t.Logf("Project working and accessible from Windows")

	_, _ = exec.RunHostCommand("wsl.exe", "-d", distroName, "bash", "-c", "ddev poweroff")

	t.Logf("Basic ddev functionality test completed successfully")
}

// isDockerProviderAvailable checks if docker ps works in the named distro
func isDockerProviderAvailable(distroName string) bool {
	// Check if docker ps works in the distro
	_, err := exec.RunHostCommand("wsl.exe", "-d", distroName, "docker", "ps")
	return err == nil
}

// isDockerDesktopWorkingOnWindows checks if Docker Desktop is working on Windows
func isDockerDesktopWorkingOnWindows() bool {
	// Check if docker ps works directly on Windows
	_, err := exec.RunHostCommand("docker.exe", "ps")
	return err == nil
}

// cleanupTraditionalWindowsEnv removes DDEV Traditional Windows installation
func cleanupTraditionalWindowsEnv(t *testing.T) {
	t.Logf("Cleaning up Traditional Windows environment")

	// Stop any running DDEV projects
	_, _ = exec.RunHostCommand("ddev.exe", "poweroff")

	// Run the uninstaller to clean up Windows-side components
	possiblePaths := []string{
		`C:\Program Files\DDEV\ddev_uninstall.exe`,
	}

	var uninstallerPath string
	for _, path := range possiblePaths {
		if fileutil.FileExists(path) {
			uninstallerPath = path
			break
		}
	}

	if uninstallerPath != "" {
		t.Logf("Running uninstaller: %s", uninstallerPath)
		out, err := exec.RunHostCommand(uninstallerPath, "/S")
		t.Logf("Uninstaller result - err: %v, output: %s", err, out)
	} else {
		t.Logf("No uninstaller found (DDEV may not be installed yet)")
	}
}

// testDdevTraditionalInstallation verifies that ddev is properly installed on Windows
func testDdevTraditionalInstallation(t *testing.T) {
	require := require.New(t)
	t.Logf("Testing ddev installation on Windows")

	// Test ddev version
	out, err := exec.RunHostCommand("ddev.exe", "version")
	require.NoError(err, "ddev version failed: %v, output: %s", err, out)
	require.Contains(out, "DDEV version")
	t.Logf("ddev version output: %s", out)

	// Test ddev-hostname
	out, err = exec.RunHostCommand("ddev-hostname.exe", "--help")
	require.NoError(err, "ddev-hostname failed: %v, output: %s", err, out)
	t.Logf("ddev-hostname available")

	// Verify files exist in expected location
	ddevPath := `C:\Program Files\DDEV\ddev.exe`
	hostnameePath := `C:\Program Files\DDEV\ddev-hostname.exe`
	require.True(fileutil.FileExists(ddevPath), "ddev.exe not found at %s", ddevPath)
	require.True(fileutil.FileExists(hostnameePath), "ddev-hostname.exe not found at %s", hostnameePath)
}

// testBasicDdevTraditionalFunctionality tests basic ddev project creation and start on Windows
func testBasicDdevTraditionalFunctionality(t *testing.T) {
	require := require.New(t)
	t.Logf("Testing basic ddev functionality on Windows")

	// Create a temporary directory for the test project
	tempDir, err := os.MkdirTemp("", "ddev-test-")
	require.NoError(err, "Failed to create temp directory: %v", err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	projectName := "testproj"
	projectDir := filepath.Join(tempDir, projectName)

	// Create test project directory
	err = os.MkdirAll(projectDir, 0755)
	require.NoError(err, "Failed to create project directory: %v", err)

	// Create a simple index.html
	indexPath := filepath.Join(projectDir, "index.html")
	err = os.WriteFile(indexPath, []byte("Hello from DDEV Traditional!"), 0644)
	require.NoError(err, "Failed to create index.html: %v", err)

	// Change to project directory and initialize ddev project
	originalDir, err := os.Getwd()
	require.NoError(err, "Failed to get current directory: %v", err)
	t.Cleanup(func() { os.Chdir(originalDir) })

	err = os.Chdir(projectDir)
	require.NoError(err, "Failed to change to project directory: %v", err)

	// Initialize ddev project
	out, err := exec.RunHostCommand("ddev.exe", "config", "--auto")
	require.NoError(err, "ddev config failed: %v, output: %s", err, out)
	t.Logf("ddev config output: %s", out)

	// Start the project
	out, err = exec.RunHostCommand("ddev.exe", "start", "-y")
	require.NoError(err, "ddev start failed: %v, output: %s", err, out)
	t.Logf("ddev start output: %s", out)

	// Ensure cleanup
	t.Cleanup(func() {
		_, _ = exec.RunHostCommand("ddev.exe", "delete", "-Oy")
		_, _ = exec.RunHostCommand("ddev.exe", "poweroff")
	})

	// Test HTTPS response from Windows
	siteURL := fmt.Sprintf("https://%s.ddev.site", projectName)
	out, err = exec.RunHostCommand("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", fmt.Sprintf("Invoke-RestMethod '%s' -ErrorAction Stop", siteURL))
	require.NoError(err, "HTTPS check failed: %v, output: %s", err, out)
	require.Contains(out, "Hello from DDEV Traditional!")
	t.Logf("Project working and accessible from Windows: %s", siteURL)

	t.Logf("Basic Traditional Windows ddev functionality test completed successfully")
}

// installSystemDdev installs a system ddev using the traditional Windows installer
func installSystemDdev(t *testing.T) {
	t.Logf("Installing system ddev for subsequent tests using traditional Windows installer")

	// Get absolute path to installer (same as used in tests)
	wd, err := os.Getwd()
	if err != nil {
		t.Logf("Failed to get working directory: %v", err)
		return
	}
	installerFullPath := filepath.Join(wd, installerPath)

	if !fileutil.FileExists(installerFullPath) {
		t.Logf("Installer not found at %s, cannot install system ddev", installerFullPath)
		return
	}

	// Run installer in traditional Windows mode with silent flag
	t.Logf("Running traditional Windows installer in silent mode: %s", installerFullPath)
	out, err := exec.RunHostCommand(installerFullPath, "/traditional", "/S")
	if err != nil {
		t.Logf("Failed to install system ddev via traditional installer: %v, output: %s", err, out)
	} else {
		t.Logf("Successfully installed system ddev via traditional installer: %s", out)
	}
}
