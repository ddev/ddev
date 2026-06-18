//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
)

const (
	installerPath = "../.gotmp/bin/windows_amd64/ddev_windows_amd64_installer.exe"
)

// getInstallerDebugLogs finds and returns the contents of the most recent installer debug log
func getInstallerDebugLogs(t *testing.T) string {
	t.Helper()

	// Get Windows temp directory
	tempDir := os.Getenv("TEMP")
	if tempDir == "" {
		tempDir = os.Getenv("TMP")
	}
	if tempDir == "" {
		t.Log("Could not determine Windows temp directory")
		return ""
	}

	// Find all installer debug logs
	pattern := filepath.Join(tempDir, "ddev_installer_debug_*.log")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Logf("Error finding installer debug logs: %v", err)
		return ""
	}

	if len(matches) == 0 {
		t.Logf("No installer debug logs found in %s", tempDir)
		return ""
	}

	// Sort by modification time (newest first)
	sort.Slice(matches, func(i, j int) bool {
		fi, _ := os.Stat(matches[i])
		fj, _ := os.Stat(matches[j])
		if fi == nil || fj == nil {
			return false
		}
		return fi.ModTime().After(fj.ModTime())
	})

	// Read the most recent log
	logPath := matches[0]
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Logf("Error reading installer debug log %s: %v", logPath, err)
		return ""
	}

	return fmt.Sprintf("=== Installer Debug Log: %s ===\n%s", logPath, string(content))
}

// waitForDockerDesktopWSL2Integration polls until Docker Desktop's WSL2
// integration is active for the given distro. It first polls briefly; if
// integration is still absent, it restarts Docker Desktop via
// .buildkite/restart-docker-desktop.sh (which forces Docker Desktop to
// re-inject its /usr/bin/docker symlink into all configured distros).
// Returns true when `docker ps` succeeds inside the distro, false on timeout.
func waitForDockerDesktopWSL2Integration(t *testing.T, distro string) bool {
	t.Helper()
	// Poll briefly before resorting to a Docker Desktop restart. Give Docker Desktop
	// ~40s to inject integration naturally after sanetestbot.sh starts it. If it
	// hasn't appeared by then, restart-docker-desktop.sh (stop+start) reliably
	// restores it in another ~30s.
	const quickAttempts = 4
	const delay = 10 * time.Second

	for attempt := 1; attempt <= quickAttempts; attempt++ {
		// When Docker Desktop integration is active it symlinks /usr/bin/docker →
		// /mnt/wsl/docker-desktop/cli-tools/usr/bin/docker (a real Linux binary).
		// When inactive, /usr/bin/docker does not exist; PATH falls through to the
		// /Docker/host/bin/docker stub which outputs the "integration not enabled" error.
		out, err := exec.RunHostCommand("wsl.exe", "-d", distro, "--", "docker", "ps")
		if err == nil {
			// docker ps works — also verify Docker API is fully ready via docker info.
			// docker ps can succeed while Docker Desktop is still initializing; the
			// installer's 'ddev version' calls docker info/version and hangs with HTTP 500
			// if the Docker daemon isn't fully ready yet.
			infoOut, infoErr := exec.RunHostCommand("wsl.exe", "-d", distro, "--", "docker", "info", "--format", "{{.ServerVersion}}")
			if infoErr == nil && strings.TrimSpace(infoOut) != "" {
				t.Logf("Docker Desktop fully ready in %s (attempt %d/%d), server version: %s", distro, attempt, quickAttempts, strings.TrimSpace(infoOut))
				return true
			}
			t.Logf("Docker Desktop integration present but API not fully ready in %s (attempt %d/%d): %v", distro, attempt, quickAttempts, infoErr)
			// Fall through — retry after delay
		}
		t.Logf("Docker Desktop WSL2 integration not yet active for %s (attempt %d/%d): %v\nOutput: %s", distro, attempt, quickAttempts, err, out)

		// On first failure, dump diagnostics to help understand the environment.
		if attempt == 1 {
			if diagOut, diagErr := exec.RunHostCommand("wsl.exe", "--list", "--running"); diagErr == nil {
				t.Logf("WSL running distros:\n%s", diagOut)
			}
			if diagOut, _ := exec.RunHostCommand("wsl.exe", "-d", distro, "--", "bash", "-lc",
				`echo "PATH=$PATH"; which docker 2>&1 || echo "docker not in PATH"; ls -la /var/run/docker.sock 2>&1; ls /usr/local/bin/docker /usr/bin/docker 2>&1`); diagOut != "" {
				t.Logf("Diagnostics inside %s:\n%s", distro, diagOut)
			}
		}

		if attempt < quickAttempts {
			time.Sleep(delay)
		}
	}

	// Before resorting to a full Docker Desktop restart, try fixing WSL interop.
	// The Docker Desktop CLI at /mnt/wsl/docker-desktop/cli-tools/usr/bin/docker
	// uses WSL interop internally. If WSLInterop was cleared from binfmt_misc
	// (e.g. by docker-ce post-remove scripts during cleanupTestEnv), the symlink
	// exists and the mount looks fine but the binary cannot execute. wsl-fix-interop
	// re-registers the binfmt_misc entry and is much cheaper than a full restart.
	t.Logf("Docker Desktop WSL2 integration absent for %s after %d attempts — trying wsl-fix-interop first", distro, quickAttempts)
	if fixOut, fixErr := exec.RunHostCommand("wsl.exe", "-d", distro, "bash", "-c", "sudo wsl-fix-interop"); fixErr != nil {
		t.Logf("wsl-fix-interop in %s failed: %v\n%s", distro, fixErr, fixOut)
	} else {
		t.Logf("wsl-fix-interop in %s: %s", distro, strings.TrimSpace(fixOut))
	}
	// Give interop a moment to take effect, then re-check.
	time.Sleep(5 * time.Second)
	if out, err := exec.RunHostCommand("wsl.exe", "-d", distro, "--", "docker", "ps"); err == nil {
		t.Logf("Docker Desktop WSL2 integration confirmed for %s after wsl-fix-interop", distro)
		_ = out
		return true
	}

	// Last resort: full Docker Desktop restart via 'docker desktop restart',
	// which re-injects the /usr/bin/docker integration symlink into the distro.
	// (We dropped the "lightweight proxy re-inject" attempt: it ran Docker
	// Desktop's own docker-desktop-user-distro proxy binary, but that is exactly
	// the binary DD itself cannot run in the failure modes we hit — the 0-byte
	// post-update stub and the noexec/permission cases — so it never succeeded
	// and only added ~40s before this restart, which does work.)
	t.Logf("Docker Desktop WSL2 integration absent for %s after wsl-fix-interop — performing full Docker Desktop restart", distro)
	wd, err := os.Getwd()
	if err != nil {
		t.Logf("Could not get working directory: %v", err)
		return false
	}
	restartScript := filepath.Join(wd, "..", ".buildkite", "restart-docker-desktop.sh")
	// The restart script stops Docker Desktop, waits for it to stop, starts it again,
	// and waits for WSL2 integration to appear — up to ~7 minutes total.
	const restartTimeout = 10 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), restartTimeout)
	defer cancel()
	cmd := osexec.CommandContext(ctx, "bash", restartScript, distro)
	restartOutBytes, restartErr := cmd.CombinedOutput()
	restartOut := string(restartOutBytes)
	t.Logf("Docker Desktop restart output:\n%s", restartOut)
	if restartErr != nil {
		t.Logf("Docker Desktop restart failed: %v", restartErr)
		return false
	}
	// After restart, docker ps works but Docker Desktop may still be initializing
	// its internal components. The NSIS installer exercises docker more thoroughly
	// (docker info, image operations) and hangs if Docker isn't fully ready.
	// Wait for docker info to succeed inside the distro as a stronger readiness check.
	t.Logf("Waiting for docker info to confirm Docker is fully initialized in %s...", distro)
	const dockerInfoAttempts = 12
	const dockerInfoDelay = 10 * time.Second
	for i := 1; i <= dockerInfoAttempts; i++ {
		out, err := exec.RunHostCommand("wsl.exe", "-d", distro, "--", "docker", "info", "--format", "{{.ServerVersion}}")
		if err == nil && strings.TrimSpace(out) != "" {
			t.Logf("Docker fully initialized in %s (attempt %d/%d), server version: %s", distro, i, dockerInfoAttempts, strings.TrimSpace(out))
			return true
		}
		t.Logf("Docker not yet fully initialized in %s (attempt %d/%d): %v", distro, i, dockerInfoAttempts, err)
		if i < dockerInfoAttempts {
			time.Sleep(dockerInfoDelay)
		}
	}
	t.Logf("WARNING: docker info did not succeed in %s after restart — proceeding anyway", distro)
	return true
}

// TestWindowsInstallerWSL2 tests WSL2 installation paths using a test matrix
func TestWindowsInstallerWSL2(t *testing.T) {
	if nodeps.IsEnvFalse("DDEV_TEST_USE_REAL_INSTALLER") {
		t.Skip("Skipping installer test, set DDEV_TEST_USE_REAL_INSTALLER=true to run")
	}

	// The test matrix is one entry per (base distro × Docker provider). The
	// instance name is also the subtest name, so `-run` filters read identically
	// to the WSL distro name, e.g. `-run TestWindowsInstallerWSL2/ddev-test-debian-ce`.
	// The named instances are provisioned out-of-band on each runner (see
	// docs/content/developers/buildkite-testmachine-setup.md); the test only
	// resets and reuses them. baseDistro is documentation of the provisioning
	// source and is not used at test runtime.
	matrix := []struct {
		instance   string // WSL instance name == subtest name
		baseDistro string // WSL catalog distro the instance is provisioned from
		provider   string // "docker-ce" or "docker-desktop"
	}{
		// "Ubuntu" in the WSL catalog installs the current LTS (Ubuntu 26.04 as of 2026).
		{instance: "ddev-test-ubuntu-ce", baseDistro: "Ubuntu", provider: "docker-ce"},
		{instance: "ddev-test-ubuntu-desktop", baseDistro: "Ubuntu", provider: "docker-desktop"},
		// {instance: "ddev-test-ubuntu2404-ce", baseDistro: "Ubuntu-24.04", provider: "docker-ce"},
		// {instance: "ddev-test-ubuntu2404-desktop", baseDistro: "Ubuntu-24.04", provider: "docker-desktop"},
		{instance: "ddev-test-debian-ce", baseDistro: "Debian", provider: "docker-ce"},
		{instance: "ddev-test-debian-desktop", baseDistro: "Debian", provider: "docker-desktop"},
	}

	providerArg := map[string]string{
		"docker-ce":      "/docker-ce",
		"docker-desktop": "/docker-desktop",
	}

	type testCase struct {
		name          string
		distro        string
		installerArgs []string
	}
	testCases := make([]testCase, 0, len(matrix))
	for _, m := range matrix {
		testCases = append(testCases, testCase{
			name:          m.instance,
			distro:        m.instance,
			installerArgs: []string{providerArg[m.provider], "/distro=" + m.instance, "/S"},
		})
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)

			// For docker-desktop cases verify integration is active before running
			// the installer. Docker Desktop frequently loses WSL2 integration; this
			// produces an actionable message instead of a cryptic installer error.
			if strings.HasSuffix(tc.name, "-desktop") {
				if !waitForDockerDesktopWSL2Integration(t, tc.distro) {
					t.Skipf("SKIPPED: Docker Desktop WSL2 integration is not active for %s after retries.\n"+
						"Re-enable it: Docker Desktop → Settings → Resources → WSL Integration → enable %s → Apply & Restart.\n"+
						"Then verify with: wsl -d %s docker ps",
						tc.distro, tc.distro, tc.distro)
				}
			}

			// Dump installer debug logs on test failure (registered first, runs last)
			t.Cleanup(func() {
				if t.Failed() {
					t.Log("Test failed - dumping installer debug logs:")
					if logs := getInstallerDebugLogs(t); logs != "" {
						t.Log(logs)
					}
				}
			})

			// Create fresh test WSL2 distro
			cleanupTestEnv(t, tc.distro)
			configureTestWSL2Distro(t, tc.distro)

			// cleanupTestEnv just removed docker-ce-cli. Desktop distros must use
			// Docker Desktop's own /usr/bin/docker integration symlink, not the
			// docker-ce-cli binary (which masks broken integration and then breaks
			// mid-install). Re-verify integration here so Docker Desktop re-injects
			// its symlink — which removing docker-ce-cli may have just deleted —
			// before the installer needs docker.
			if strings.HasSuffix(tc.name, "-desktop") {
				if !waitForDockerDesktopWSL2Integration(t, tc.distro) {
					t.Skipf("SKIPPED: Docker Desktop WSL2 integration not active for %s after cleanup/retries.\n"+
						"Verify with: wsl -d %s docker ps", tc.distro, tc.distro)
				}
			}

			// Ensure ddev is powered off after this test case, even if it fails
			t.Cleanup(func() {
				t.Logf("Cleaning up %s test - powering off ddev", tc.name)
				_, _ = exec.RunHostCommand("wsl.exe", "-d", tc.distro, "bash", "-c", "ddev poweroff")
				_, _ = exec.RunHostCommand("wsl.exe", "-d", tc.distro, "bash", "-c", "ddev delete -Oy tp")
				// Remove docker-ce-cli/docker-ce on ALL distros, including docker-desktop.
				// On desktop distros docker-ce-cli must NOT be present: it owns
				// /usr/bin/docker (Docker Desktop's integration symlink path) and, while
				// installed, masks broken integration (docker ps succeeds via its binary
				// + DD's socket) so the precondition falsely passes and docker breaks
				// mid-install. With it gone, /usr/bin/docker is Docker Desktop's own
				// symlink, which the precondition can reliably detect and restart-repair.
				_, _ = exec.RunHostCommand("wsl.exe", "-d", tc.distro, "-u", "root", "bash", "-c", "apt-get remove -y ddev ddev-wsl2 docker-ce-cli docker-ce")
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

			// Run installer with a 10-minute timeout to prevent infinite hangs.
			// Each case runs as its own decoupled Buildkite job, so an install
			// that exceeds 10 minutes is treated as a real failure (the test then
			// dumps the installer debug log rather than letting the job hang).
			const installerTimeout = 15 * time.Minute
			ctx, cancel := context.WithTimeout(context.Background(), installerTimeout)
			defer cancel()

			cmd := osexec.CommandContext(ctx, installerFullPath, tc.installerArgs...)
			installerOutput, err := cmd.CombinedOutput()
			out = string(installerOutput)

			if ctx.Err() == context.DeadlineExceeded {
				t.Logf("Installer TIMED OUT after %v", installerTimeout)
				t.Logf("Dumping installer debug log for timeout diagnosis:")
				if logs := getInstallerDebugLogs(t); logs != "" {
					t.Log(logs)
				}
				require.Fail("Installer timeout", "Installer did not complete within %v", installerTimeout)
			}

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

				// Dump debug log on any error
				t.Logf("Dumping installer debug log for error diagnosis:")
				if logs := getInstallerDebugLogs(t); logs != "" {
					t.Log(logs)
				}

				require.NoError(err, "Installer failed: %v, output: %s", err, out)
			}
			t.Logf("Installer completed successfully, output: %s", out)

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

			// Assert CAROOT is set in registry
			require.NoError(userCarootRegErr, "CAROOT should be set in registry after install")
			caRootValue := parseRegQueryValue(userCarootReg)
			require.NotEmpty(caRootValue, "CAROOT registry value should not be empty after install")
			t.Logf("CAROOT registry value: %q", caRootValue)

			// Assert WSLENV is set correctly: must contain CAROOT/up, must not contain semicolons
			require.NoError(userWslenvRegErr, "WSLENV should be set in registry after WSL2 install")
			wslenvValue := parseRegQueryValue(userWslenvReg)
			require.NotEmpty(wslenvValue, "WSLENV registry value should not be empty after install")
			require.Contains(wslenvValue, "CAROOT/up", "WSLENV should contain CAROOT/up after install")
			require.NotContains(wslenvValue, ";", "WSLENV must not contain semicolons — they are not valid WSLENV separators and prevent CAROOT from propagating to WSL2")
			t.Logf("WSLENV registry value: %q (valid)", wslenvValue)

			// Check system environment variables in registry
			systemCarootReg, systemCarootRegErr := exec.RunHostCommand("reg.exe", "query", "HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment", "/v", "CAROOT")
			t.Logf("Registry HKLM\\System\\Environment CAROOT: %s (err: %v)", strings.TrimSpace(systemCarootReg), systemCarootRegErr)

			systemWslenvReg, systemWslenvRegErr := exec.RunHostCommand("reg.exe", "query", "HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment", "/v", "WSLENV")
			t.Logf("Registry HKLM\\System\\Environment WSLENV: %s (err: %v)", strings.TrimSpace(systemWslenvReg), systemWslenvRegErr)

			// Check mkcert -CAROOT on Windows side
			mkcertOut, mkcertErr := exec.RunHostCommand("mkcert.exe", "-CAROOT")
			t.Logf("Windows mkcert -CAROOT: %s (err: %v)", strings.TrimSpace(mkcertOut), mkcertErr)

			// Check WSLENV inside the distro
			out, err = exec.RunHostCommand("wsl.exe", "-d", tc.distro, "bash", "-c", "echo $WSLENV")
			t.Logf("WSL distro $WSLENV: %s (err: %v)", strings.TrimSpace(out), err)

			distroCAROOTOut, distroCAROOTErr := exec.RunHostCommand("wsl.exe", "-d", tc.distro, "bash", "-c", "echo $CAROOT")
			t.Logf("WSL distro $CAROOT: %s (err: %v)", strings.TrimSpace(distroCAROOTOut), distroCAROOTErr)

			// Wait for installation completion by monitoring status file
			const maxTries = 60
			statusFile := "/tmp/ddev_installation_status.txt"

			for i := 0; i < maxTries; i++ {
				// Log current status for debugging
				statusOut, _ := exec.RunHostCommand("wsl.exe", "-d", tc.distro, "bash", "-c", "tail -1l "+statusFile+" 2>/dev/null")
				if statusOut != "" {
					t.Logf("Installation status on try %d: %s", i, strings.TrimSpace(statusOut))
				}

				// Check for COMPLETED marker - only break when installation is actually done
				out, err := exec.RunHostCommand("wsl.exe", "-d", tc.distro, "bash", "-c", "grep '^COMPLETED' "+statusFile+" 2>/dev/null")
				if err == nil && strings.Contains(out, "COMPLETED") {
					t.Logf("Installation completion confirmed: %s", strings.TrimSpace(out))
					break
				}

				// Check for errors
				errOut, errErr := exec.RunHostCommand("wsl.exe", "-d", tc.distro, "bash", "-c", "grep '^ERROR' "+statusFile+" 2>/dev/null")
				if errErr == nil && strings.Contains(errOut, "ERROR") {
					require.Fail("Installation failed", "Error found in status file: %s", strings.TrimSpace(errOut))
				}
				if i == maxTries-1 {
					require.Fail("Installation timeout", "No completion marker found after %d tries. Last status: %s", maxTries, strings.TrimSpace(statusOut))
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
	if nodeps.IsEnvFalse("DDEV_TEST_USE_REAL_INSTALLER") {
		t.Skip("Skipping installer test, set DDEV_TEST_USE_REAL_INSTALLER=true to run")
	}
	// Check if Docker Desktop is working on Windows
	if !isDockerDesktopWorkingOnWindows() {
		t.Skip("Skipping Traditional Windows test - Docker Desktop not working on Windows")
	}

	require := require.New(t)

	// Dump installer debug logs on test failure (registered first, runs last)
	t.Cleanup(func() {
		if t.Failed() {
			t.Log("Test failed - dumping installer debug logs:")
			if logs := getInstallerDebugLogs(t); logs != "" {
				t.Log(logs)
			}
		}
	})

	// Clean up any existing DDEV installation
	cleanupTraditionalWindowsEnv(t)

	// Ensure cleanup after test
	t.Cleanup(func() {
		t.Logf("Cleaning up Traditional Windows test")
		cleanupTraditionalWindowsEnv(t)
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

	// Wait for installer to complete by checking for ddev.exe at expected location
	// NSIS installers in silent mode may return before fully completing
	localAppData := os.Getenv("LOCALAPPDATA")
	ddevPath := filepath.Join(localAppData, "Programs", "DDEV", "ddev.exe")
	const maxWaitSeconds = 60
	for i := 0; i < maxWaitSeconds; i++ {
		if fileutil.FileExists(ddevPath) {
			t.Logf("ddev.exe found at %s after %d seconds", ddevPath, i)
			break
		}
		if i == maxWaitSeconds-1 {
			require.Fail("ddev.exe not found at expected location after %d seconds: %s", maxWaitSeconds, ddevPath)
		}
		time.Sleep(1 * time.Second)
	}

	// Test that ddev is installed and working on Windows
	testDdevTraditionalInstallation(t)

	// Test basic ddev functionality on Windows
	testBasicDdevTraditionalFunctionality(t)
}

// parseRegQueryValue parses the value from reg.exe query output.
// reg.exe outputs lines like: "    VARNAME    REG_SZ    value"
// Returns the value string or empty string if not found.
func parseRegQueryValue(output string) string {
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "REG_SZ") {
			parts := strings.SplitN(line, "REG_SZ", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// Helper functions

// cleanupTestEnv removes the test WSL2 distro
func cleanupTestEnv(t *testing.T, distroName string) {
	t.Logf("Cleaning up test environment")

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
		// Ensure WSL interop is working before any .exe operations in this distro.
		// The binfmt_misc WSLInterop entry can go missing after a distro restart or if
		// systemd hasn't fully started. Without it, calling any Windows .exe from within
		// WSL (e.g. mkcert.exe, reg.exe) fails with "Exec format error". In the PS1
		// install scripts this causes mkcert.exe -install to fail silently, leaving the
		// mkcert CA out of the Windows cert store and breaking all PowerShell HTTPS checks.
		// wsl-fix-interop re-registers the entry idempotently. Requires one-time install
		// per distro — see docs/content/developers/buildkite-testmachine-setup.md.
		// See https://github.com/rfay/wsl-fix-interop
		if fixOut, fixErr := exec.RunHostCommand("wsl.exe", "-d", distroName, "bash", "-c", "sudo wsl-fix-interop"); fixErr != nil {
			t.Logf("wsl-fix-interop not available or failed in %s (install it per testmachine setup docs): %v\n%s", distroName, fixErr, fixOut)
		} else {
			t.Logf("wsl-fix-interop: %s", strings.TrimSpace(fixOut))
		}

		// Get distro back to a fairly normal pre-ddev state.
		// Makes test run much faster than completely deleting the distro.
		// Delete the test project fully (including Docker volumes) before unlisting.
		// Old certs in Docker volumes from a broken-interop run are signed by a CA that
		// is not in the Windows cert store; without this delete, DDEV reuses them and the
		// Windows HTTPS check fails even after interop and the CA are restored.
		out, _ := exec.RunHostCommand("wsl.exe", "-d", distroName, "bash", "-c", "(ddev delete -Oy tp 2>/dev/null || true) && (ddev poweroff 2>/dev/null || true) && (ddev stop --unlist -a 2>/dev/null) && rm -rf ~/tp")
		t.Logf("ddev poweroff/stop/unlist: err=%v, output: %s", err, out)

		// Temp allow all sudo to let mkcert -uninstall work as normal user
		out, err := exec.RunHostCommand("wsl.exe", "-d", distroName, "-u", "root", "bash", "-c", `echo "ALL ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/temp-mkcert-install`)
		require.NoError(t, err)

		// Now do mkcert -uninstall as normal user if mkcert is installed
		out, err = exec.RunHostCommand("wsl.exe", "-d", distroName, "bash", "-c", "if command -v mkcert >/dev/null 2>&1; then mkcert -uninstall; fi")
		require.NoError(t, err)

		// Now take away temp sudo and remove ddev/docker apt sources leftovers (*.list and *.sources create conflicts in apt-get)
		out, err = exec.RunHostCommand("wsl.exe", "-d", distroName, "-u", "root", "bash", "-c", "rm -f /etc/sudoers.d/temp-mkcert-install /etc/apt/sources.list.d/ddev.* /etc/apt/sources.list.d/docker.*")
		require.NoError(t, err)

		// Remove docker-ce-cli/docker-ce on ALL distros, including docker-desktop.
		// docker-desktop distros must use Docker Desktop's own /usr/bin/docker
		// integration symlink, not the docker-ce-cli binary: while docker-ce-cli is
		// installed it owns /usr/bin/docker and masks broken DD integration (docker ps
		// works via its binary + DD's socket), so the integration check falsely passes
		// and docker breaks mid-install. The post-cleanup integration re-verify (in the
		// test) restarts Docker Desktop to re-inject the symlink that this removal deletes.
		out, err = exec.RunHostCommand("wsl.exe", "-d", distroName, "-u", "root", "bash", "-c", "(apt-get remove -y ddev ddev-wsl2 docker-ce-cli docker-ce 2>/dev/null)")
		t.Logf("distro cleanup: err=%v, output: %s", err, out)

		// Re-run wsl-fix-interop after apt-get remove: docker-ce's post-remove scripts
		// clear binfmt_misc entries (for QEMU multi-arch support), which also removes
		// the WSLInterop entry. Re-register it so the next test's mkcert.exe calls work.
		if fixOut, fixErr := exec.RunHostCommand("wsl.exe", "-d", distroName, "bash", "-c", "sudo wsl-fix-interop"); fixErr != nil {
			t.Logf("wsl-fix-interop post-cleanup not available in %s: %v\n%s", distroName, fixErr, fixOut)
		} else {
			t.Logf("wsl-fix-interop post-cleanup: %s", strings.TrimSpace(fixOut))
		}
	}
}

// configureTestWSL2Distro verifies that the pre-provisioned test WSL2 distro
// exists. Test distros are provisioned out-of-band, once per runner (see
// docs/content/developers/buildkite-testmachine-setup.md). The test does not
// create them — it only resets and reuses them (see cleanupTestEnv). This
// function fails fast with a clear message if the instance is missing.
func configureTestWSL2Distro(t *testing.T, distroName string) {
	require := require.New(t)
	t.Logf("Verifying pre-provisioned test distro: %s", distroName)

	out, err := exec.RunHostCommand("wsl.exe", "-l", "-q")
	require.NoError(err, "Failed to list WSL distros")

	// Convert UTF-16 output to UTF-8 by removing null bytes
	cleanOut := strings.ReplaceAll(out, "\x00", "")
	if !strings.Contains(cleanOut, distroName) {
		t.Fatalf("Test distro %q is not registered on this runner. Provision it once "+
			"(wsl --install <base> --name %s) per "+
			"docs/content/developers/buildkite-testmachine-setup.md before running installer tests.",
			distroName, distroName)
	}

	t.Logf("Test WSL2 distro %s present", distroName)
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

	// Make sure previous has been deleted
	_, _ = exec.RunHostCommand("wsl.exe", "-d", distroName, "bash", "-c", "ddev delete -Oy tp")

	// Clean up any existing test project
	_, _ = exec.RunHostCommand("wsl.exe", "-d", distroName, "rm", "-rf", projectDir)

	// Create test project directory
	out, err := exec.RunHostCommand("wsl.exe", "-d", distroName, "mkdir", "-p", projectDir)
	require.NoError(err, "Failed to create project directory: %v, output: %s", err, out)

	// Create a simple index.html
	_, err = exec.RunHostCommand("wsl.exe", "-d", distroName, "bash", "-c", fmt.Sprintf("echo 'Hello from DDEV!' > %s/index.html", projectDir))
	require.NoError(err, "Failed to create index.html: %v", err)

	// Ensure DDEV's mkcert_caroot is set to the correct Windows CAROOT before ddev start.
	// Issue #8485: the NSIS installer calls SetEnvironmentVariable("CAROOT", NULL) to unset
	// CAROOT in its own process before running mkcert.exe, which means WSL2 processes
	// spawned from NSIS during installation run with empty CAROOT. readCAROOT() then
	// returns "" and DDEV writes empty mkcert_caroot to ~/.ddev/global_config.yaml. When
	// ddev start later runs, it generates a new CA at the Linux default path rather than
	// using the Windows-side CA at CAROOT — that new CA is not in the Windows cert store.
	// Fix: get CAROOT from cmd.exe (the test process environment, always reliable), convert
	// to Linux path via wslpath, and force-set ddev config global. This is idempotent.
	caRootForConfig, _ := exec.RunHostCommand("cmd.exe", "/c", "echo %CAROOT%")
	caRootForConfig = strings.TrimSpace(caRootForConfig)
	if caRootForConfig != "" && caRootForConfig != "%CAROOT%" {
		// Convert Windows path to WSL path in Go rather than via wslpath.
		// Passing a Windows path with backslashes to 'wsl.exe wslpath' causes bash
		// to interpret \U, \t, \A, \L etc. as escape sequences, stripping the backslashes.
		// Direct conversion: C:\Users\foo -> /mnt/c/Users/foo
		wslCARoot := windowsPathToWSL(caRootForConfig)
		// ddev config global has no --mkcert-caroot flag; set it directly in the YAML.
		// This bypasses readCAROOT() which may return "" if rootCA-key.pem permissions
		// prevent reading from WSL2 (issue #8485).
		t.Logf("Setting DDEV mkcert_caroot to %s in global_config.yaml (CAROOT=%s)", wslCARoot, caRootForConfig)
		setCmd := fmt.Sprintf(
			`mkdir -p ~/.ddev; `+
				`if grep -q "^mkcert_caroot:" ~/.ddev/global_config.yaml 2>/dev/null; then `+
				`  sed -i "s|^mkcert_caroot:.*|mkcert_caroot: %s|" ~/.ddev/global_config.yaml; `+
				`else `+
				`  printf "mkcert_caroot: %s\n" >> ~/.ddev/global_config.yaml; `+
				`fi; `+
				`grep mkcert_caroot ~/.ddev/global_config.yaml`, wslCARoot, wslCARoot)
		setOut, setErr := exec.RunHostCommand("wsl.exe", "-d", distroName, "bash", "-c", setCmd)
		t.Logf("Set mkcert_caroot in global_config.yaml: err=%v out=%s", setErr, strings.TrimSpace(setOut))
	} else {
		t.Logf("WARNING: CAROOT not set in test process environment — DDEV may use wrong CA")
	}

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

	// Dump Windows cert store mkcert entries and compare thumbprint with rootCA.pem.
	// A mismatch (old cert in store, new rootCA.pem on disk) is the most common reason
	// the PowerShell HTTPS check fails even though mkcert.exe -install reports "already installed".
	caRootDir, _ := exec.RunHostCommand("cmd.exe", "/c", "echo %CAROOT%")
	caRootDir = strings.TrimSpace(caRootDir)
	if storeOut, storeErr := exec.RunHostCommand("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command",
		"Get-ChildItem Cert:\\CurrentUser\\Root | Where-Object {$_.Subject -like '*mkcert*'} | Select-Object Subject,Thumbprint,NotBefore,NotAfter | Format-List"); storeErr == nil {
		t.Logf("Windows cert store mkcert entries:\n%s", storeOut)
	}
	if caRootDir != "" && caRootDir != "%CAROOT%" {
		if thumbOut, thumbErr := exec.RunHostCommand("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command",
			fmt.Sprintf("(New-Object System.Security.Cryptography.X509Certificates.X509Certificate2('%s\\rootCA.pem')).Thumbprint", caRootDir)); thumbErr == nil {
			t.Logf("rootCA.pem thumbprint (CAROOT=%s): %s", caRootDir, strings.TrimSpace(thumbOut))
		}
	}

	// Dump the actual TLS certificate served by the site so we can see which CA signed it.
	// This runs regardless of trust (ServerCertificateValidationCallback={$true}) so it
	// works even when the cert is not trusted, letting us compare the signing CA thumbprint
	// against the Windows cert store entries logged above.
	if certChainOut, certChainErr := exec.RunHostCommand("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command",
		fmt.Sprintf(`$req=[Net.HttpWebRequest]::Create("https://%s.ddev.site"); $req.ServerCertificateValidationCallback={$true}; try{$req.GetResponse().Dispose()}catch{}; $c=$req.ServicePoint.Certificate; if($c){$x=New-Object Security.Cryptography.X509Certificates.X509Certificate2($c); "Site cert thumbprint: "+$x.Thumbprint; "Site cert subject: "+$x.Subject; "Site cert issuer: "+$x.Issuer; "Site cert notbefore: "+$x.NotBefore; "Site cert notafter: "+$x.NotAfter}`, projectName)); certChainErr == nil {
		t.Logf("Site TLS certificate (served, ignoring trust):\n%s", certChainOut)
	}

	// Test using windows PowerShell to check HTTPS
	psInvoke := fmt.Sprintf("powershell.exe -NoProfile -ExecutionPolicy Bypass -Command Invoke-RestMethod 'https://%s.ddev.site' -ErrorAction Stop", projectName)
	out, err = exec.RunHostCommand("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", fmt.Sprintf("Invoke-RestMethod 'https://%s.ddev.site' -ErrorAction Stop", projectName))
	require.NoError(err, "HTTPS check from Windows failed (`%s`): %v, output: %s", psInvoke, err, out)
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

// windowsPathToWSL converts a Windows path to its WSL2 mount path in Go,
// avoiding the need to call wslpath (which receives args through bash and
// has backslashes stripped as escape sequences).
// Example: C:\Users\testbot\AppData\Local\mkcert → /mnt/c/Users/testbot/AppData/Local/mkcert
func windowsPathToWSL(winPath string) string {
	if len(winPath) >= 2 && winPath[1] == ':' {
		drive := strings.ToLower(string(winPath[0]))
		rest := strings.ReplaceAll(winPath[2:], "\\", "/")
		return "/mnt/" + drive + rest
	}
	return strings.ReplaceAll(winPath, "\\", "/")
}

// isDockerDesktopWorkingOnWindows checks if Docker Desktop is working on Windows
func isDockerDesktopWorkingOnWindows() bool {
	// Check if docker ps works directly on Windows
	_, err := exec.RunHostCommand("docker.exe", "ps")
	return err == nil
}

// cleanupTraditionalWindowsEnv cleans up Traditional Windows test environment
func cleanupTraditionalWindowsEnv(t *testing.T) {
	t.Logf("Cleaning up Traditional Windows environment")

	// Use full path since the current process doesn't see the PATH update from the installer
	localAppData := os.Getenv("LOCALAPPDATA")
	ddevPath := filepath.Join(localAppData, "Programs", "DDEV", "ddev.exe")

	// Stop any running DDEV projects (ignore errors if ddev not installed)
	if fileutil.FileExists(ddevPath) {
		_, _ = exec.RunHostCommand(ddevPath, "poweroff")
	}
}

// testDdevTraditionalInstallation verifies that ddev is properly installed on Windows
func testDdevTraditionalInstallation(t *testing.T) {
	require := require.New(t)
	t.Logf("Testing ddev installation on Windows")

	// Use full paths since the current process doesn't see the PATH update from the installer
	localAppData := os.Getenv("LOCALAPPDATA")
	ddevPath := filepath.Join(localAppData, "Programs", "DDEV", "ddev.exe")
	hostnamePath := filepath.Join(localAppData, "Programs", "DDEV", "ddev-hostname.exe")

	// Verify files exist in expected per-user location
	require.True(fileutil.FileExists(ddevPath), "ddev.exe not found at %s", ddevPath)
	require.True(fileutil.FileExists(hostnamePath), "ddev-hostname.exe not found at %s", hostnamePath)

	// Test ddev version using full path
	out, err := exec.RunHostCommand(ddevPath, "version")
	require.NoError(err, "ddev version failed: %v, output: %s", err, out)
	require.Contains(out, "DDEV version")
	t.Logf("ddev version output: %s", out)

	// Test ddev-hostname using full path
	out, err = exec.RunHostCommand(hostnamePath, "--help")
	require.NoError(err, "ddev-hostname failed: %v, output: %s", err, out)
	t.Logf("ddev-hostname available")
}

// testBasicDdevTraditionalFunctionality tests basic ddev project creation and start on Windows
func testBasicDdevTraditionalFunctionality(t *testing.T) {
	require := require.New(t)
	t.Logf("Testing basic ddev functionality on Windows")

	// Use full path since the current process doesn't see the PATH update from the installer
	localAppData := os.Getenv("LOCALAPPDATA")
	ddevPath := filepath.Join(localAppData, "Programs", "DDEV", "ddev.exe")

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

	// Initialize ddev project using full path
	out, err := exec.RunHostCommand(ddevPath, "config", "--auto")
	require.NoError(err, "ddev config failed: %v, output: %s", err, out)
	t.Logf("ddev config output: %s", out)

	// Start the project using full path
	out, err = exec.RunHostCommand(ddevPath, "start", "-y")
	require.NoError(err, "ddev start failed: %v, output: %s", err, out)
	t.Logf("ddev start output: %s", out)

	// Ensure cleanup using full path
	t.Cleanup(func() {
		_, _ = exec.RunHostCommand(ddevPath, "delete", "-Oy")
		_, _ = exec.RunHostCommand(ddevPath, "poweroff")
	})

	// Test HTTPS response from Windows
	siteURL := fmt.Sprintf("https://%s.ddev.site", projectName)
	out, err = exec.RunHostCommand("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", fmt.Sprintf("Invoke-RestMethod '%s' -ErrorAction Stop", siteURL))
	require.NoError(err, "HTTPS check failed: %v, output: %s", err, out)
	require.Contains(out, "Hello from DDEV Traditional!")
	t.Logf("Project working and accessible from Windows: %s", siteURL)

	t.Logf("Basic Traditional Windows ddev functionality test completed successfully")
}
