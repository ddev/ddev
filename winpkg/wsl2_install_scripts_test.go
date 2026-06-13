//go:build windows

package main

import (
	"context"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
)

// getDefaultDistro returns the name of the current default WSL2 distro, or ""
// if it cannot be determined. The default distro is the one marked with "*" in
// `wsl -l -v` output (which is UTF-16, so NUL bytes are stripped).
func getDefaultDistro(t *testing.T) string {
	out, err := exec.RunHostCommand("wsl.exe", "-l", "-v")
	if err != nil {
		t.Logf("Failed to list WSL distros: %v", err)
		return ""
	}
	clean := strings.ReplaceAll(out, "\x00", "")
	for _, line := range strings.Split(clean, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "*") {
			fields := strings.Fields(strings.TrimPrefix(trimmed, "*"))
			if len(fields) > 0 {
				return fields[0]
			}
		}
	}
	return ""
}

// TestWSL2InstallScripts exercises the manual WSL2 install PowerShell scripts
// (scripts/install_ddev_wsl2_docker_inside.ps1 and
// scripts/install_ddev_wsl2_docker_desktop.ps1) against current Ubuntu. These
// scripts predate the GUI installer and otherwise have no automated coverage.
//
// The scripts operate on the *default* WSL2 distro, so each case temporarily
// sets its target instance as the default and restores the prior default
// afterward. The named instances are provisioned out-of-band (see
// docs/content/developers/buildkite-testmachine-setup.md) and reused here.
func TestWSL2InstallScripts(t *testing.T) {
	if nodeps.IsEnvFalse("DDEV_TEST_USE_REAL_INSTALLER") {
		t.Skip("Skipping WSL2 install-script test, set DDEV_TEST_USE_REAL_INSTALLER=true to run")
	}

	testCases := []struct {
		name          string
		script        string
		distro        string
		skipCondition func() bool
	}{
		{
			name:          "docker-inside",
			script:        "../scripts/install_ddev_wsl2_docker_inside.ps1",
			distro:        "ddev-test-ubuntu-ce",
			skipCondition: func() bool { return false }, // always run
		},
		{
			name:          "docker-desktop",
			script:        "../scripts/install_ddev_wsl2_docker_desktop.ps1",
			distro:        "ddev-test-ubuntu-desktop",
			skipCondition: func() bool { return !isDockerProviderAvailable("ddev-test-ubuntu-desktop") },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipCondition() {
				t.Skipf("Skipping %s test - distro %s must have a working Docker provider", tc.name, tc.distro)
			}

			require := require.New(t)

			// Verify the pre-provisioned distro exists (fails fast with guidance).
			configureTestWSL2Distro(t, tc.distro)

			// The scripts act on the default distro; set ours as default and
			// restore the prior default afterward.
			origDefault := getDefaultDistro(t)
			t.Logf("Current default WSL2 distro: %q", origDefault)
			t.Cleanup(func() {
				if origDefault != "" && origDefault != tc.distro {
					t.Logf("Restoring default WSL2 distro to %q", origDefault)
					_, _ = exec.RunHostCommand("wsl.exe", "--set-default", origDefault)
				}
			})

			// Reset the distro to a pre-ddev state for a meaningful install.
			cleanupTestEnv(t, tc.distro)

			// Ensure ddev is powered off after this case, even on failure.
			t.Cleanup(func() {
				t.Logf("Cleaning up %s test - powering off ddev", tc.name)
				_, _ = exec.RunHostCommand("wsl.exe", "-d", tc.distro, "bash", "-c", "ddev poweroff")
				_, _ = exec.RunHostCommand("wsl.exe", "-d", tc.distro, "bash", "-c", "ddev delete -Oy tp")
			})

			_, err := exec.RunHostCommand("wsl.exe", "--set-default", tc.distro)
			require.NoError(err, "Failed to set %s as default WSL2 distro", tc.distro)

			// Resolve the script's absolute path.
			wd, err := os.Getwd()
			require.NoError(err)
			scriptFullPath := filepath.Join(wd, tc.script)
			require.True(fileutil.FileExists(scriptFullPath), "Install script not found at %s", scriptFullPath)

			// Run the PowerShell script with a 15-minute timeout. Docker CE
			// installs + image pulls can legitimately take several minutes.
			t.Logf("Running install script: %s (default distro=%s)", scriptFullPath, tc.distro)
			const scriptTimeout = 15 * time.Minute
			ctx, cancel := context.WithTimeout(context.Background(), scriptTimeout)
			defer cancel()

			cmd := osexec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptFullPath)
			scriptOutput, err := cmd.CombinedOutput()
			out := string(scriptOutput)

			if ctx.Err() == context.DeadlineExceeded {
				require.Fail("Script timeout", "%s did not complete within %v, output: %s", tc.script, scriptTimeout, out)
			}
			require.NoError(err, "Install script %s failed: %v, output: %s", tc.script, err, out)
			t.Logf("Install script completed, output: %s", out)

			// The scripts set CAROOT and WSLENV in the Windows user environment
			// via setx; assert they landed in the registry.
			userCarootReg, userCarootRegErr := exec.RunHostCommand("reg.exe", "query", "HKEY_CURRENT_USER\\Environment", "/v", "CAROOT")
			require.NoError(userCarootRegErr, "CAROOT should be set in registry after running %s", tc.script)
			caRootValue := parseRegQueryValue(userCarootReg)
			require.NotEmpty(caRootValue, "CAROOT registry value should not be empty after running %s", tc.script)
			t.Logf("CAROOT registry value: %q", caRootValue)

			userWslenvReg, userWslenvRegErr := exec.RunHostCommand("reg.exe", "query", "HKEY_CURRENT_USER\\Environment", "/v", "WSLENV")
			require.NoError(userWslenvRegErr, "WSLENV should be set in registry after running %s", tc.script)
			wslenvValue := parseRegQueryValue(userWslenvReg)
			require.Contains(wslenvValue, "CAROOT/up", "WSLENV should contain CAROOT/up after running %s", tc.script)
			t.Logf("WSLENV registry value: %q", wslenvValue)

			// End-to-end: ddev installed and a project starts.
			testDdevInstallation(t, tc.distro)
			testBasicDdevFunctionality(t, tc.distro)
		})
	}
}
