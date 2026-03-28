package cmd

import (
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// TLSDiagnoseCmd implements the ddev utility tls-diagnose command
var TLSDiagnoseCmd = &cobra.Command{
	Use:   "tls-diagnose",
	Short: "Diagnose TLS/HTTPS certificate trust issues",
	Long: `Check mkcert installation, CA trust stores, certificates, and live HTTPS connectivity.

This command checks:
- mkcert installation and CAROOT configuration
- OS trust store installation
- WSL2-specific CA sharing requirements (when running in WSL2)
- Certificate files and their validity
- Live HTTPS connectivity (when a project is running)

On WSL2, this is especially useful to diagnose why Windows browsers (Chrome, Edge)
do not trust DDEV HTTPS certificates.`,
	Example: `ddev utility tls-diagnose
ddev ut tls-diagnose`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 0 {
			fmt.Fprintln(os.Stderr, "This command takes no additional arguments")
			os.Exit(1)
		}
		exitCode := runTLSDiagnose()
		os.Exit(exitCode)
	},
}

func init() {
	DebugCmd.AddCommand(TLSDiagnoseCmd)
}

// runTLSDiagnose performs all TLS diagnostic checks.
// Returns exit code: 0 if no issues, 1 if issues found.
func runTLSDiagnose() int {
	hasIssues := false

	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("TLS/HTTPS Diagnostics")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println()

	// Try to load app from current directory (optional)
	app, _ := ddevapp.GetActiveApp("")

	caRoot, mkcertIssues := checkMkcertInstallation()
	if mkcertIssues {
		hasIssues = true
	}

	trustIssues := checkOSTrustStore()
	if trustIssues {
		hasIssues = true
	}

	// WSLg detection: when Linux-side browsers are installed inside WSL2 the
	// Windows certificate store is irrelevant — setup is the same as plain Linux.
	// Ask interactively only when we actually find a Linux browser.
	wslgMode := false
	if nodeps.IsWSL2() {
		wslgBrowsers := detectWSLgBrowsers()
		if len(wslgBrowsers) > 0 {
			output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			output.UserOut.Println("WSLg Browser Detection")
			output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			output.UserOut.Printf("  ℹ Linux-side browser(s) found in WSL2: %s\n", strings.Join(wslgBrowsers, ", "))
			output.UserOut.Println("    These may be running via WSLg (Linux GUI app support).")
			output.UserOut.Println("    For WSLg browsers, the Windows certificate store is irrelevant —")
			output.UserOut.Println("    only the Linux-side trust store (already checked above) matters.")
			output.UserOut.Println()
			wslgMode = util.ConfirmTo("Are you troubleshooting a browser running inside WSL2 via WSLg (not a Windows browser)?", false)
			output.UserOut.Println()
			if wslgMode {
				output.UserOut.Println("  ℹ WSLg mode selected — skipping Windows-side certificate checks.")
				output.UserOut.Println("    The Linux trust store and mkcert -install inside WSL2 are all that is needed.")
				output.UserOut.Println()
			}
		}

		if !wslgMode {
			wsl2Issues := checkWSL2Configuration(caRoot)
			if wsl2Issues {
				hasIssues = true
			}
		}
	}

	firefoxWarnings := checkFirefoxStatus(wslgMode)
	if firefoxWarnings {
		hasIssues = true
	}

	certIssues := checkCertificateFiles(caRoot, app)
	if certIssues {
		hasIssues = true
	}

	if app != nil && app.AppRoot != "" {
		connIssues := checkLiveConnectivity(caRoot, app)
		if connIssues {
			hasIssues = true
		}
	}

	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Summary")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if hasIssues {
		tlsFail("Issues found — see sections above for details and fixes")
		output.UserOut.Println()
		output.UserOut.Println("  Nuclear option (when all else fails):")
		output.UserOut.Println("    ddev poweroff && mkcert -uninstall && rm -rf \"$(mkcert -CAROOT)\" && mkcert -install && ddev start")
	} else {
		output.UserOut.Println("  ✓ No issues found — TLS configuration looks good!")
	}
	output.UserOut.Println()

	if hasIssues {
		return 1
	}
	return 0
}

// checkMkcertInstallation checks mkcert binary, CAROOT, and CA files.
// Returns (caRoot, hasIssues).
func checkMkcertInstallation() (string, bool) {
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("mkcert Installation")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	hasIssues := false

	// Check mkcert in PATH (WSL2-side / Linux)
	mkcertPath, err := exec.LookPath("mkcert")
	if err != nil {
		tlsFail("mkcert not found in PATH")
		output.UserOut.Println("    → Install: brew install mkcert / choco install -y mkcert / or use DDEV installer")
		output.UserOut.Println()
		return "", true
	}
	output.UserOut.Printf("  ✓ mkcert found: %s\n", mkcertPath)

	// On WSL2 also confirm mkcert.exe is available on the Windows side
	if nodeps.IsWSL2() {
		winMkcertCheck, err := exec.Command("cmd.exe", "/c", "where mkcert").Output()
		if err != nil {
			tlsFail("Windows-side mkcert.exe not found in Windows PATH")
			output.UserOut.Println("    → Install mkcert on Windows: choco install -y mkcert  OR  winget install mkcert")
			hasIssues = true
		} else {
			winMkcertPath := strings.TrimSpace(strings.TrimSuffix(string(winMkcertCheck), "\r\n"))
			output.UserOut.Printf("  ✓ Windows mkcert found: %s\n", winMkcertPath)
		}
	}

	// Get CAROOT via mkcert -CAROOT
	caRootOut, err := exec.Command("mkcert", "-CAROOT").Output()
	if err != nil {
		tlsFail("mkcert -CAROOT failed — mkcert may be broken")
		output.UserOut.Println()
		return "", true
	}
	caRoot := strings.TrimSpace(string(caRootOut))
	output.UserOut.Printf("  ✓ CAROOT: %s\n", caRoot)

	// Also show $CAROOT env var if set and mismatched
	caRootEnv := os.Getenv("CAROOT")
	if caRootEnv != "" && caRootEnv != caRoot {
		output.UserOut.Printf("  ⚠ $CAROOT env var (%s) differs from mkcert -CAROOT (%s)\n", caRootEnv, caRoot)
		output.UserOut.Println("    → This mismatch can cause certificate issues")
		hasIssues = true
	}

	// Check rootCA.pem — if it doesn't exist locally, clear caRoot so downstream
	// checks don't try (and fail) to read from a path that isn't accessible on
	// this OS (e.g. CAROOT pointing to a Windows path on macOS).
	rootCAPEM := filepath.Join(caRoot, "rootCA.pem")
	if _, err := os.Stat(rootCAPEM); err != nil {
		if !nodeps.IsPathOnWindowsFilesystem(caRoot) || nodeps.IsWSL2() {
			tlsFail("rootCA.pem not found in CAROOT (%s)", caRoot)
			output.UserOut.Println("    → Run: mkcert -install")
		} else {
			tlsFail("CAROOT (%s) points to a Windows path that is not accessible on this OS", caRoot)
			output.UserOut.Println("    → Unset $CAROOT from your shell, or set it to the local mkcert CA directory")
		}
		hasIssues = true
		caRoot = "" // clear so downstream checks skip CA-dependent work
	} else {
		output.UserOut.Println("  ✓ rootCA.pem found")
	}

	// Check rootCA-key.pem (only when caRoot is still valid)
	if caRoot != "" {
		rootCAKey := filepath.Join(caRoot, "rootCA-key.pem")
		if _, err := os.Stat(rootCAKey); err != nil {
			tlsFail("rootCA-key.pem not found or not readable")
			output.UserOut.Println("    → Run: mkcert -install")
			hasIssues = true
		} else {
			output.UserOut.Println("  ✓ rootCA-key.pem readable")
		}
	}

	// Cross-check with globalconfig
	gcCARoot := globalconfig.GetCAROOT()
	if gcCARoot == "" && caRoot != "" {
		output.UserOut.Println("  ⚠ DDEV global config does not see a valid CAROOT (rootCA.pem may be missing)")
		hasIssues = true
	}

	output.UserOut.Println()
	return caRoot, hasIssues
}

// checkOSTrustStore runs mkcert -install and checks the result.
func checkOSTrustStore() bool {
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("OS Trust Store")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	hasIssues := false

	// Run mkcert -install to check/install the CA
	cmd := exec.Command("mkcert", "-install")
	out, err := cmd.CombinedOutput()
	installOutput := strings.TrimSpace(string(out))

	if err != nil {
		tlsFail("mkcert -install failed: %v", err)
		if installOutput != "" {
			output.UserOut.Printf("    Output: %s\n", installOutput)
		}
		hasIssues = true
	} else {
		if strings.Contains(installOutput, "already installed") {
			output.UserOut.Println("  ✓ CA already installed in system trust store")
		} else if strings.Contains(installOutput, "installed") {
			output.UserOut.Println("  ✓ CA installed in system trust store")
		} else {
			output.UserOut.Println("  ✓ mkcert -install succeeded")
		}
		if installOutput != "" && !strings.Contains(installOutput, "already installed") {
			// Show any extra output (e.g., warnings about certutil)
			for line := range strings.SplitSeq(installOutput, "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					output.UserOut.Printf("    ℹ %s\n", line)
				}
			}
		}
	}

	// On Linux, check for certutil (needed for Firefox/NSS)
	if nodeps.IsLinux() && !nodeps.IsWSL2() {
		_, err := exec.LookPath("certutil")
		if err != nil {
			output.UserOut.Println("  ⚠ certutil not found — Firefox will not trust DDEV certificates")
			output.UserOut.Println("    → Install: sudo apt install libnss3-tools  OR  brew install nss")
			hasIssues = true
		} else {
			output.UserOut.Println("  ✓ certutil found (Firefox NSS support available)")
		}
	}

	output.UserOut.Println()
	return hasIssues
}

// checkWSL2Configuration checks the WSL2-specific CA sharing requirements.
func checkWSL2Configuration(caRoot string) bool {
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("WSL2 Configuration")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("  ✓ Running in WSL2")

	hasIssues := false

	// Check CAROOT points to Windows filesystem
	caRootEnv := os.Getenv("CAROOT")
	if caRootEnv == "" {
		tlsFail("$CAROOT env var is not set in WSL2")
		output.UserOut.Println("    → In Windows PowerShell, run:")
		output.UserOut.Println("      $env:CAROOT = mkcert -CAROOT")
		output.UserOut.Println("      setx CAROOT $env:CAROOT")
		output.UserOut.Println("      setx WSLENV \"CAROOT/up\"")
		output.UserOut.Println("    → Then restart WSL2: wsl --shutdown")
		hasIssues = true
	} else if !nodeps.IsPathOnWindowsFilesystem(caRootEnv) {
		output.UserOut.Printf("  ⚠ $CAROOT (%s) does not point to Windows filesystem (/mnt/...)\n", caRootEnv)
		output.UserOut.Println("    → Windows browsers won't trust DDEV certificates")
		output.UserOut.Println("    → Set CAROOT to the Windows mkcert CAROOT (e.g., /mnt/c/Users/<user>/AppData/Local/mkcert)")
		hasIssues = true
	} else {
		output.UserOut.Printf("  ✓ CAROOT points to Windows filesystem: %s\n", caRootEnv)
	}

	// Check WSLENV contains CAROOT/up
	wslEnv := os.Getenv("WSLENV")
	if !strings.Contains(wslEnv, "CAROOT") {
		tlsFail("WSLENV does not contain CAROOT — environment not propagated to Windows")
		output.UserOut.Println("    → In Windows PowerShell, run:")
		output.UserOut.Println("      setx WSLENV \"CAROOT/up\"")
		output.UserOut.Println("    → Then restart WSL2: wsl --shutdown")
		hasIssues = true
	} else {
		output.UserOut.Printf("  ✓ WSLENV contains CAROOT (%s)\n", wslEnv)
	}

	// Check Windows-side mkcert CAROOT and compare paths
	winMkcertOut, err := exec.Command("cmd.exe", "/c", "mkcert -CAROOT").Output()
	if err != nil {
		tlsFail("Windows-side mkcert CAROOT could not be determined")
		output.UserOut.Println("    → Install mkcert on Windows: choco install -y mkcert  OR  winget install mkcert")
		hasIssues = true
	} else {
		winCARoot := strings.TrimSpace(string(winMkcertOut))
		winCARoot = strings.TrimSuffix(winCARoot, "\r")
		output.UserOut.Printf("  ✓ Windows mkcert CAROOT: %s\n", winCARoot)

		// Translate Windows path to WSL path for comparison
		wslWinCARoot := windowsPathToWSL(winCARoot)
		if caRoot != "" && wslWinCARoot != "" && !strings.EqualFold(wslWinCARoot, caRoot) {
			output.UserOut.Printf("  ⚠ Windows CAROOT (%s) != WSL2 CAROOT (%s)\n", wslWinCARoot, caRoot)
			output.UserOut.Println("    → Run the Windows PowerShell CAROOT setup again")
			hasIssues = true
		} else if wslWinCARoot != "" {
			output.UserOut.Println("  ✓ Windows CAROOT matches WSL2 CAROOT")
		}
	}

	// Compute SHA1 thumbprint of the WSL2-side rootCA.pem for comparison
	wslThumbprint := ""
	if caRoot != "" {
		wslThumbprint, err = certThumbprintFromFile(filepath.Join(caRoot, "rootCA.pem"))
		if err != nil {
			output.UserOut.Printf("  ⚠ Could not read WSL2 rootCA.pem for fingerprint: %v\n", err)
		}
	}

	// Check Windows certificate store for mkcert CA via PowerShell (both LocalMachine and CurrentUser)
	// Also retrieve the thumbprint to compare with the WSL2-side CA.
	psScript := `
$certs = @(Get-ChildItem Cert:\LocalMachine\Root | Where-Object { $_.Subject -like "*mkcert*" }) +
         @(Get-ChildItem Cert:\CurrentUser\Root | Where-Object { $_.Subject -like "*mkcert*" })
if ($certs.Count -gt 0) {
    Write-Output "FOUND:$($certs[0].Thumbprint):$($certs[0].Subject)"
} else {
    Write-Output "NOTFOUND"
}
`
	psOut, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psScript).Output()
	if err != nil {
		output.UserOut.Println("  ⚠ Could not query Windows certificate store via PowerShell")
	} else {
		psResult := strings.TrimSpace(string(psOut))
		psResult = strings.TrimSuffix(psResult, "\r")
		if after, found := strings.CutPrefix(psResult, "FOUND:"); found {
			// Format: "thumbprint:subject"
			parts := strings.SplitN(after, ":", 2)
			winThumbprint := strings.ToUpper(strings.TrimSpace(parts[0]))
			subject := ""
			if len(parts) == 2 {
				subject = strings.TrimSpace(parts[1])
			}
			output.UserOut.Printf("  ✓ mkcert CA found in Windows certificate store: %s\n", subject)

			// Compare thumbprints to confirm WSL2 and Windows use the same CA
			if wslThumbprint != "" && winThumbprint != "" {
				if strings.EqualFold(wslThumbprint, winThumbprint) {
					output.UserOut.Printf("  ✓ WSL2 and Windows CA fingerprints match (%s)\n", winThumbprint)
				} else {
					output.UserOut.Printf("  ⚠ CA fingerprint mismatch: WSL2=%s  Windows=%s\n", wslThumbprint, winThumbprint)
					output.UserOut.Println("    → WSL2 is using a different CA than the one trusted by Windows")
					output.UserOut.Println("    → Run mkcert -install in PowerShell, then restart WSL2")
					hasIssues = true
				}
			}
		} else {
			tlsFail("mkcert CA NOT found in Windows certificate store")
			output.UserOut.Println("    → In Windows PowerShell (as yourself, not Administrator), run: mkcert -install")
			hasIssues = true
		}
	}

	// Report networking mode
	mode, err := nodeps.GetWSL2NetworkingMode()
	if err != nil {
		output.UserOut.Println("  ℹ WSL2 networking mode: unknown")
	} else {
		output.UserOut.Printf("  ℹ WSL2 networking mode: %s\n", mode)
		if mode == "mirrored" {
			if !nodeps.IsWSL2HostAddressLoopbackEnabled() {
				output.UserOut.Println("  ⚠ WSL2 mirrored mode: hostAddressLoopback is not enabled in .wslconfig")
				output.UserOut.Println("    → Add under [experimental] in <Windows user home>\\.wslconfig:")
				output.UserOut.Println("      hostAddressLoopback=true")
			} else {
				output.UserOut.Println("  ✓ WSL2 mirrored mode: hostAddressLoopback=true")
			}
		}
	}

	output.UserOut.Println()
	return hasIssues
}

// checkFirefoxStatus checks for Firefox installations and issues warnings.
func checkFirefoxStatus(wslgMode bool) bool {
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Firefox")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	hasWarnings := false

	switch {
	case nodeps.IsWSL2() && wslgMode:
		// WSLg mode: browser runs on Linux inside WSL2, same as plain Linux.
		output.UserOut.Println("  ℹ WSLg mode: checking Linux-side Firefox (Windows Firefox is not relevant)")
		hasWarnings = checkLinuxFirefox()

	case nodeps.IsWSL2():
		// Standard WSL2: browser runs on Windows side.
		psScript := `
$firefoxPaths = @(
    "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\firefox.exe",
    "HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\firefox.exe"
)
$found = $false
foreach ($path in $firefoxPaths) {
    if (Test-Path $path) {
        $found = $true
        break
    }
}
if (-not $found) {
    $commonPaths = @(
        "$env:ProgramFiles\Mozilla Firefox\firefox.exe",
        "$env:ProgramFiles(x86)\Mozilla Firefox\firefox.exe",
        "$env:LOCALAPPDATA\Mozilla Firefox\firefox.exe"
    )
    foreach ($path in $commonPaths) {
        if (Test-Path $path) {
            $found = $true
            break
        }
    }
}
if ($found) { Write-Output "FOUND" } else { Write-Output "NOTFOUND" }
`
		psOut, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psScript).Output()
		if err == nil {
			result := strings.TrimSpace(strings.TrimSuffix(string(psOut), "\r"))
			if result == "FOUND" {
				output.UserOut.Println("  ⚠ Firefox detected on Windows — Firefox cannot use the Windows system certificate store")
				output.UserOut.Println("    → You must manually import the mkcert CA into Firefox:")
				output.UserOut.Println("      Firefox Settings → Privacy & Security → View Certificates → Import")
				output.UserOut.Printf("      CA file: %s\\rootCA.pem\n", windowsPathFromWSLCAROOT())
				output.UserOut.Println("    → See: https://docs.ddev.com/en/stable/users/install/configuring-browsers/")
				hasWarnings = true
			} else {
				output.UserOut.Println("  ✓ Firefox not detected on Windows")
			}
		} else {
			output.UserOut.Println("  ℹ Could not check for Firefox on Windows")
		}

	case nodeps.IsWindows():
		// Traditional Windows (not WSL2). Firefox does not use the Windows system
		// certificate store, so the mkcert CA must be imported manually.
		firefoxPaths := []string{
			filepath.Join(os.Getenv("ProgramFiles"), "Mozilla Firefox", "firefox.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Mozilla Firefox", "firefox.exe"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Mozilla Firefox", "firefox.exe"),
		}
		firefoxFound := false
		for _, p := range firefoxPaths {
			if _, err := os.Stat(p); err == nil {
				firefoxFound = true
				break
			}
		}
		if firefoxFound {
			output.UserOut.Println("  ⚠ Firefox detected on Windows — Firefox does not use the Windows system certificate store")
			output.UserOut.Println("    → You must manually import the mkcert CA into Firefox:")
			output.UserOut.Println("      Firefox Settings → Privacy & Security → View Certificates → Import")
			caRoot := globalconfig.GetCAROOT()
			if caRoot != "" {
				output.UserOut.Printf("      CA file: %s\\rootCA.pem\n", caRoot)
			}
			output.UserOut.Println("    → Firefox Nightly, Developer Edition, and ESR each have separate trust stores")
			output.UserOut.Println("    → See: https://docs.ddev.com/en/stable/users/install/configuring-browsers/")
			hasWarnings = true
		} else {
			output.UserOut.Println("  ✓ Firefox not detected on Windows")
		}

	case nodeps.IsMacOS():
		firefoxApps := []string{
			"/Applications/Firefox.app",
			"/Applications/Firefox Nightly.app",
			"/Applications/Firefox Developer Edition.app",
		}
		for _, appPath := range firefoxApps {
			if _, err := os.Stat(appPath); err == nil {
				appName := strings.TrimSuffix(filepath.Base(appPath), ".app")
				if appName == "Firefox" {
					output.UserOut.Printf("  ⚠ %s detected — may not use macOS Keychain for certificate trust\n", appName)
				} else {
					output.UserOut.Printf("  ⚠ %s detected — has a separate NSS trust store, needs separate CA import\n", appName)
				}
				output.UserOut.Println("    → See: https://docs.ddev.com/en/stable/users/install/configuring-browsers/")
				hasWarnings = true
			}
		}
		if !hasWarnings {
			output.UserOut.Println("  ✓ No Firefox variants detected on macOS")
		}

	case nodeps.IsLinux():
		hasWarnings = checkLinuxFirefox()
	}

	output.UserOut.Println()
	return hasWarnings
}

// checkLinuxFirefox checks Firefox on Linux (also used for WSLg browsers inside WSL2).
// Returns true if warnings were found.
func checkLinuxFirefox() bool {
	hasWarnings := false

	// Whether Firefox trusts DDEV certificates depends on certutil (libnss3-tools).
	// mkcert -install registers the CA into Firefox's NSS database only when certutil is present.
	_, certutilErr := exec.LookPath("certutil")
	certutilFound := certutilErr == nil

	firefoxVariants := []string{"firefox", "firefox-nightly", "firefox-developer-edition", "firefox-esr"}
	foundAny := false
	for _, variant := range firefoxVariants {
		if _, err := exec.LookPath(variant); err == nil {
			foundAny = true
			if certutilFound {
				output.UserOut.Printf("  ✓ %s detected — certutil available, mkcert -install registers the CA via NSS\n", variant)
				output.UserOut.Printf("    ⚠ Note: some %s builds (Flatpak, certain snap versions) maintain a separate NSS database\n", variant)
				output.UserOut.Println("      and may still require manual CA import if HTTPS warnings appear")
			} else {
				output.UserOut.Printf("  ⚠ %s detected but certutil not found — Firefox may not trust DDEV certificates\n", variant)
				output.UserOut.Println("    → Install certutil: sudo apt install libnss3-tools  OR  brew install nss")
				output.UserOut.Println("    → Then run: mkcert -install")
				output.UserOut.Println("    → Or manually import: Firefox Settings → Privacy & Security → View Certificates → Import")
				hasWarnings = true
			}
		}
	}
	// Snap Firefox has its own NSS database separate from the system one.
	if snapOut, err := exec.Command("snap", "list", "firefox").Output(); err == nil && strings.Contains(string(snapOut), "firefox") {
		output.UserOut.Println("  ⚠ Firefox (snap) detected — snap Firefox uses its own NSS database")
		output.UserOut.Println("    → mkcert -install may or may not reach the snap profile")
		output.UserOut.Println("    → If HTTPS warnings appear, manually import the CA in Firefox")
		hasWarnings = true
		foundAny = true
	}
	if !foundAny {
		output.UserOut.Println("  ✓ No Firefox detected on Linux")
	}
	return hasWarnings
}

// checkCertificateFiles checks the default and project certificates for validity.
func checkCertificateFiles(caRoot string, app *ddevapp.DdevApp) bool {
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Certificate Files")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	hasIssues := false

	if caRoot == "" {
		tlsFail("Cannot check certificates — CAROOT not available")
		output.UserOut.Println()
		return true
	}

	// Load the CA cert pool
	caPool, err := loadCACertPool(caRoot)
	if err != nil {
		tlsFail("Failed to load CA certificate from CAROOT: %v", err)
		output.UserOut.Println("    → Run: mkcert -install")
		output.UserOut.Println()
		return true
	}
	output.UserOut.Println("  ✓ CA certificate loaded from CAROOT")

	// Check global default cert
	globalTraefikCertsDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik", "certs")
	defaultCert := filepath.Join(globalTraefikCertsDir, "default_cert.crt")
	defaultIssues := checkCertFile(defaultCert, caPool, []string{}, "Default cert")
	if defaultIssues {
		hasIssues = true
		output.UserOut.Println("    → Run: ddev poweroff && ddev start  to regenerate certificates")
	}

	// Check project cert if in a project
	if app != nil && app.AppRoot != "" {
		projectCertsDir := app.GetConfigPath("traefik/certs")
		projectCertFile := filepath.Join(projectCertsDir, app.Name+".crt")
		hostnames := []string{app.GetHostname()}
		projectIssues := checkCertFile(projectCertFile, caPool, hostnames, fmt.Sprintf("Project cert (%s)", app.Name))
		if projectIssues {
			hasIssues = true
			output.UserOut.Println("    → Run: ddev restart  to regenerate project certificate")
		}
	}

	output.UserOut.Println()
	return hasIssues
}

// checkCertFile validates a single certificate file against the CA pool.
// Returns true if issues found.
func checkCertFile(certPath string, caPool *x509.CertPool, expectedHostnames []string, label string) bool {
	hasIssues := false

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		tlsFail("%s not found: %s", label, certPath)
		return true
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		tlsFail("%s: failed to decode PEM", label)
		return true
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		tlsFail("%s: failed to parse certificate: %v", label, err)
		return true
	}

	// Check expiration
	now := time.Now()
	if now.After(cert.NotAfter) {
		tlsFail("%s: EXPIRED (expired %s)", label, cert.NotAfter.Format("2006-01-02"))
		hasIssues = true
	} else {
		output.UserOut.Printf("  ✓ %s exists and is valid (expires %s)\n", label, cert.NotAfter.Format("2006-01-02"))
	}

	// Verify against CA
	_, err = cert.Verify(x509.VerifyOptions{
		Roots:       caPool,
		CurrentTime: now,
	})
	if err != nil {
		tlsFail("%s: NOT verified against current CAROOT: %v", label, err)
		output.UserOut.Println("    → CA may have been rotated; regenerate certificates")
		hasIssues = true
	} else {
		output.UserOut.Printf("  ✓ %s verified against current CAROOT\n", label)
	}

	// Check hostname coverage
	for _, hostname := range expectedHostnames {
		if err := cert.VerifyHostname(hostname); err != nil {
			tlsFail("%s does not cover hostname %s", label, hostname)
			hasIssues = true
		} else {
			output.UserOut.Printf("  ✓ %s covers hostname %s\n", label, hostname)
		}
	}

	return hasIssues
}

// checkLiveConnectivity checks HTTPS connectivity to a running project.
func checkLiveConnectivity(caRoot string, app *ddevapp.DdevApp) bool {
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Live Connectivity")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	hasIssues := false

	status, _ := app.SiteStatus()
	if status != ddevapp.SiteRunning {
		output.UserOut.Printf("  ℹ Project '%s' is not running — skipping live connectivity check\n", app.Name)
		output.UserOut.Println("    → Run: ddev start  then re-run tls-diagnose")
		output.UserOut.Println()
		return false
	}
	output.UserOut.Printf("  ✓ Project '%s' is running\n", app.Name)

	hostname := app.GetHostname()
	httpsPort := app.GetPrimaryRouterHTTPSPort()
	if httpsPort == "" {
		httpsPort = "443"
	}

	// Load CA cert pool for verification.
	// Prefer an explicit pool from CAROOT so we test against the exact CA that signed
	// the certs. Fall back to the system pool when CAROOT is unavailable locally
	// (e.g. CAROOT points to a Windows path that doesn't exist on this OS), which
	// works correctly on macOS/Linux after a successful mkcert -install.
	caPool, err := loadCACertPool(caRoot)
	var caPoolNote string
	if err != nil {
		sysPool, sysErr := x509.SystemCertPool()
		if sysErr != nil {
			tlsFail("Cannot load CA for live check: %v", err)
			hasIssues = true
			caPool = nil
		} else {
			caPool = sysPool
			caPoolNote = " (using system CA pool — CAROOT not readable locally)"
		}
	}

	if caPool != nil {
		// Linux/WSL2/macOS-side TLS check via tls.Dial
		addr := fmt.Sprintf("localhost:%s", httpsPort)
		tlsConfig := &tls.Config{
			ServerName: hostname,
			RootCAs:    caPool,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			tlsFail("TLS connection to %s (SNI: %s) failed: %v", addr, hostname, err)
			output.UserOut.Println("    → Router may not be running or certificate is not trusted")
			hasIssues = true
		} else {
			conn.Close()
			output.UserOut.Printf("  ✓ TLS verified: localhost:%s with SNI %s%s\n", httpsPort, hostname, caPoolNote)
		}
	}

	// Windows-side check (WSL2 only)
	if nodeps.IsWSL2() {
		projectURL := fmt.Sprintf("https://%s", hostname)
		if httpsPort != "443" {
			projectURL = fmt.Sprintf("https://%s:%s", hostname, httpsPort)
		}
		output.UserOut.Printf("  Checking Windows trust (Chrome/Edge test) via PowerShell: %s\n", projectURL)

		// Use TrustFailure status to distinguish certificate errors from network errors.
		// A certificate error means the CA is not trusted on the Windows side.
		// A connect/DNS error means a network issue unrelated to certificates.
		psScript := fmt.Sprintf(`
try {
    $null = Invoke-WebRequest -Uri '%s' -UseBasicParsing -TimeoutSec 10
    Write-Output "TRUSTED"
} catch [System.Net.WebException] {
    $status = $_.Exception.Status
    $msg = $_.Exception.Message
    if ($status -eq [System.Net.WebExceptionStatus]::TrustFailure) {
        Write-Output "CERT_ERROR:$msg"
    } elseif ($status -eq [System.Net.WebExceptionStatus]::ConnectFailure -or
              $status -eq [System.Net.WebExceptionStatus]::NameResolutionFailure) {
        Write-Output "CONNECT_ERROR:$msg"
    } else {
        Write-Output "CERT_ERROR:$msg"
    }
} catch {
    Write-Output "CONNECT_ERROR:$($_.Exception.Message)"
}
`, projectURL)

		psOut, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psScript).Output()
		if err != nil {
			output.UserOut.Println("  ⚠ Could not run Windows PowerShell connectivity test")
		} else {
			result := strings.TrimSpace(strings.TrimSuffix(string(psOut), "\r"))
			if result == "TRUSTED" {
				output.UserOut.Println("  ✓ Windows Invoke-WebRequest: TRUSTED (Chrome/Edge would trust this)")
			} else if msg, ok := strings.CutPrefix(result, "CERT_ERROR:"); ok {
				tlsFail("Windows certificate not trusted: %s", msg)
				output.UserOut.Println("    → Windows does not trust the mkcert CA")
				output.UserOut.Println("    → In Windows PowerShell (as yourself, not Administrator): mkcert -install")
				output.UserOut.Println("    → Make sure CAROOT points to the Windows mkcert directory")
				hasIssues = true
			} else if msg, ok := strings.CutPrefix(result, "CONNECT_ERROR:"); ok {
				output.UserOut.Printf("  ⚠ Windows connection error (DNS/network issue, not a certificate problem): %s\n", msg)
			} else {
				output.UserOut.Printf("  ⚠ Windows connectivity check returned: %s\n", result)
			}
		}
	}

	output.UserOut.Println()
	return hasIssues
}

// loadCACertPool loads the mkcert rootCA.pem into an x509.CertPool.
func loadCACertPool(caRoot string) (*x509.CertPool, error) {
	rootCAPEM := filepath.Join(caRoot, "rootCA.pem")
	caCertBytes, err := os.ReadFile(rootCAPEM)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", rootCAPEM, err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCertBytes) {
		return nil, fmt.Errorf("no valid certificates found in %s", rootCAPEM)
	}
	return pool, nil
}

// certThumbprintFromFile returns the uppercase SHA1 hex thumbprint of the first
// certificate in the PEM file at path. This matches the Thumbprint shown by
// Windows Get-ChildItem Cert:\.
func certThumbprintFromFile(pemPath string) (string, error) {
	data, err := os.ReadFile(pemPath)
	if err != nil {
		return "", err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return "", fmt.Errorf("no PEM block found in %s", pemPath)
	}
	sum := sha1.Sum(block.Bytes) //nolint:gosec // SHA1 used only for display/comparison, not security
	return strings.ToUpper(hex.EncodeToString(sum[:])), nil
}

// windowsPathToWSL converts a Windows path (e.g. C:\Users\foo) to a WSL path (/mnt/c/Users/foo).
func windowsPathToWSL(winPath string) string {
	winPath = strings.TrimSuffix(winPath, "\r")
	winPath = strings.ReplaceAll(winPath, "\\", "/")
	if len(winPath) >= 2 && winPath[1] == ':' {
		drive := strings.ToLower(string(winPath[0]))
		return "/mnt/" + drive + winPath[2:]
	}
	return winPath
}

// windowsPathFromWSLCAROOT returns the Windows-style CAROOT path for display purposes.
func windowsPathFromWSLCAROOT() string {
	caRootEnv := os.Getenv("CAROOT")
	if caRootEnv == "" {
		return "<CAROOT>"
	}
	// Convert /mnt/c/Users/foo -> C:\Users\foo
	if strings.HasPrefix(caRootEnv, "/mnt/") {
		rest := caRootEnv[5:] // strip /mnt/
		if len(rest) >= 1 {
			drive := strings.ToUpper(string(rest[0]))
			path := strings.ReplaceAll(rest[1:], "/", "\\")
			return drive + ":" + path
		}
	}
	return caRootEnv
}

// tlsFail prints a red ✗ failure line to the diagnostic output.
func tlsFail(format string, a ...any) {
	mark := util.ColorizeText("✗", "red")
	msg := fmt.Sprintf(format, a...)
	output.UserOut.Println("  " + mark + " " + msg)
}

// detectWSLgBrowsers returns the names of Linux-side browsers found in PATH.
// These browsers may be running via WSLg (Linux GUI app support), in which case
// the Windows certificate store is irrelevant.
func detectWSLgBrowsers() []string {
	candidates := []string{
		"google-chrome",
		"google-chrome-stable",
		"chromium",
		"chromium-browser",
		"firefox",
		"microsoft-edge",
		"microsoft-edge-stable",
		"brave-browser",
		"brave",
	}
	var found []string
	for _, name := range candidates {
		if _, err := exec.LookPath(name); err == nil {
			found = append(found, name)
		}
	}
	return found
}
