package cmd

import (
	"crypto/tls"
	"crypto/x509"
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

	if nodeps.IsWSL2() {
		wsl2Issues := checkWSL2Configuration(caRoot)
		if wsl2Issues {
			hasIssues = true
		}
	}

	firefoxWarnings := checkFirefoxStatus()
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
		output.UserOut.Println("  ✗ Issues found — see sections above for details and fixes")
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

	// Check mkcert in PATH
	mkcertPath, err := exec.LookPath("mkcert")
	if err != nil {
		output.UserOut.Println("  ✗ mkcert not found in PATH")
		output.UserOut.Println("    → Install: brew install mkcert / choco install -y mkcert / or use DDEV installer")
		output.UserOut.Println()
		return "", true
	}
	output.UserOut.Printf("  ✓ mkcert found: %s\n", mkcertPath)

	// Get CAROOT via mkcert -CAROOT
	caRootOut, err := exec.Command("mkcert", "-CAROOT").Output()
	if err != nil {
		output.UserOut.Println("  ✗ mkcert -CAROOT failed — mkcert may be broken")
		output.UserOut.Println()
		return "", true
	}
	caRoot := strings.TrimSpace(string(caRootOut))
	output.UserOut.Printf("  ✓ CAROOT: %s\n", caRoot)

	// Also show $CAROOT env var if set
	caRootEnv := os.Getenv("CAROOT")
	if caRootEnv != "" && caRootEnv != caRoot {
		output.UserOut.Printf("  ⚠ $CAROOT env var (%s) differs from mkcert -CAROOT (%s)\n", caRootEnv, caRoot)
		output.UserOut.Println("    → This mismatch can cause certificate issues")
		hasIssues = true
	}

	// Check rootCA.pem
	rootCAPEM := filepath.Join(caRoot, "rootCA.pem")
	if _, err := os.Stat(rootCAPEM); err != nil {
		output.UserOut.Println("  ✗ rootCA.pem not found in CAROOT")
		output.UserOut.Println("    → Run: mkcert -install")
		hasIssues = true
	} else {
		output.UserOut.Println("  ✓ rootCA.pem found")
	}

	// Check rootCA-key.pem
	rootCAKey := filepath.Join(caRoot, "rootCA-key.pem")
	if _, err := os.Stat(rootCAKey); err != nil {
		output.UserOut.Println("  ✗ rootCA-key.pem not found or not readable")
		output.UserOut.Println("    → Run: mkcert -install")
		hasIssues = true
	} else {
		output.UserOut.Println("  ✓ rootCA-key.pem readable")
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
		output.UserOut.Printf("  ✗ mkcert -install failed: %v\n", err)
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
		output.UserOut.Println("  ✗ $CAROOT env var is not set in WSL2")
		output.UserOut.Println("    → In Windows PowerShell/CMD, run:")
		output.UserOut.Println("      $env:CAROOT = mkcert -CAROOT")
		output.UserOut.Println("      setx CAROOT $env:CAROOT")
		output.UserOut.Println("    → Then in WSL2, add to ~/.bashrc or ~/.zshrc:")
		output.UserOut.Println("      export WSLENV=CAROOT/up")
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
		output.UserOut.Println("  ✗ WSLENV does not contain CAROOT — environment not propagated to Windows")
		output.UserOut.Println("    → In Windows PowerShell, run:")
		output.UserOut.Println("      setx WSLENV \"CAROOT/up\"")
		output.UserOut.Println("    → Then restart WSL2")
		hasIssues = true
	} else {
		output.UserOut.Printf("  ✓ WSLENV contains CAROOT (%s)\n", wslEnv)
	}

	// Check Windows-side mkcert via cmd.exe
	winMkcertOut, err := exec.Command("cmd.exe", "/c", "mkcert -CAROOT").Output()
	if err != nil {
		output.UserOut.Println("  ✗ Windows-side mkcert not found")
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

	// Check Windows certificate store for mkcert CA via PowerShell (both LocalMachine and CurrentUser)
	psScript := `
$certs = @(Get-ChildItem Cert:\LocalMachine\Root | Where-Object { $_.Subject -like "*mkcert*" }) +
         @(Get-ChildItem Cert:\CurrentUser\Root | Where-Object { $_.Subject -like "*mkcert*" })
if ($certs.Count -gt 0) {
    Write-Output "FOUND:$($certs[0].Subject)"
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
		if subject, found := strings.CutPrefix(psResult, "FOUND:"); found {
			output.UserOut.Printf("  ✓ mkcert CA found in Windows certificate store: %s\n", subject)
		} else {
			output.UserOut.Println("  ✗ mkcert CA NOT found in Windows certificate store")
			output.UserOut.Println("    → In Windows PowerShell (as Administrator), run: mkcert -install")
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
func checkFirefoxStatus() bool {
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Firefox")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	hasWarnings := false

	if nodeps.IsWSL2() {
		// Check for Firefox on Windows side via PowerShell registry query
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
    # Also check common install locations
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
	} else if nodeps.IsMacOS() {
		// Check for Firefox variants on macOS
		firefoxApps := []string{
			"/Applications/Firefox.app",
			"/Applications/Firefox Nightly.app",
			"/Applications/Firefox Developer Edition.app",
		}
		for _, appPath := range firefoxApps {
			if _, err := os.Stat(appPath); err == nil {
				appName := filepath.Base(appPath)
				appName = strings.TrimSuffix(appName, ".app")
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
	} else if nodeps.IsLinux() {
		// Check for Firefox on Linux
		firefoxVariants := []string{"firefox", "firefox-nightly", "firefox-developer-edition", "firefox-esr"}
		foundAny := false
		for _, variant := range firefoxVariants {
			if _, err := exec.LookPath(variant); err == nil {
				output.UserOut.Printf("  ℹ %s detected — uses certutil/NSS for certificate trust\n", variant)
				foundAny = true
			}
		}
		// Check snap and flatpak Firefox
		if snapOut, err := exec.Command("snap", "list", "firefox").Output(); err == nil && strings.Contains(string(snapOut), "firefox") {
			output.UserOut.Println("  ⚠ Firefox (snap) detected — snap Firefox has its own NSS database")
			output.UserOut.Println("    → Run: mkcert -install  (mkcert handles snap Firefox automatically on some systems)")
			hasWarnings = true
			foundAny = true
		}
		if !foundAny {
			output.UserOut.Println("  ✓ No Firefox detected on Linux")
		}
	}

	output.UserOut.Println()
	return hasWarnings
}

// checkCertificateFiles checks the default and project certificates for validity.
func checkCertificateFiles(caRoot string, app *ddevapp.DdevApp) bool {
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Certificate Files")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	hasIssues := false

	if caRoot == "" {
		output.UserOut.Println("  ✗ Cannot check certificates — CAROOT not available")
		output.UserOut.Println()
		return true
	}

	// Load the CA cert pool
	caPool, err := loadCACertPool(caRoot)
	if err != nil {
		output.UserOut.Printf("  ✗ Failed to load CA certificate from CAROOT: %v\n", err)
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
		output.UserOut.Printf("  ✗ %s not found: %s\n", label, certPath)
		output.UserOut.Printf("    (path: %s)\n", certPath)
		return true
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		output.UserOut.Printf("  ✗ %s: failed to decode PEM\n", label)
		return true
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		output.UserOut.Printf("  ✗ %s: failed to parse certificate: %v\n", label, err)
		return true
	}

	// Check expiration
	now := time.Now()
	if now.After(cert.NotAfter) {
		output.UserOut.Printf("  ✗ %s: EXPIRED (expired %s)\n", label, cert.NotAfter.Format("2006-01-02"))
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
		output.UserOut.Printf("  ✗ %s: NOT verified against current CAROOT: %v\n", label, err)
		output.UserOut.Println("    → CA may have been rotated; regenerate certificates")
		hasIssues = true
	} else {
		output.UserOut.Printf("  ✓ %s verified against current CAROOT\n", label)
	}

	// Check hostname coverage
	for _, hostname := range expectedHostnames {
		if err := cert.VerifyHostname(hostname); err != nil {
			output.UserOut.Printf("  ✗ %s does not cover hostname %s\n", label, hostname)
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
	httpsPort := globalconfig.DdevGlobalConfig.RouterHTTPSPort
	if httpsPort == "" {
		httpsPort = nodeps.DdevDefaultRouterHTTPSPort
	}

	// Load CA cert pool for verification
	caPool, err := loadCACertPool(caRoot)
	if err != nil {
		output.UserOut.Printf("  ✗ Cannot load CA for live check: %v\n", err)
		hasIssues = true
	} else {
		// Linux/WSL2-side TLS check via tls.Dial
		addr := fmt.Sprintf("localhost:%s", httpsPort)
		tlsConfig := &tls.Config{
			ServerName: hostname,
			RootCAs:    caPool,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			output.UserOut.Printf("  ✗ TLS connection to %s (SNI: %s) failed: %v\n", addr, hostname, err)
			output.UserOut.Println("    → Router may not be running or certificate is not trusted")
			hasIssues = true
		} else {
			conn.Close()
			output.UserOut.Printf("  ✓ TLS verified: localhost:%s with SNI %s\n", httpsPort, hostname)
		}
	}

	// Windows-side check (WSL2 only)
	if nodeps.IsWSL2() {
		projectURL := fmt.Sprintf("https://%s", hostname)
		if httpsPort != "443" {
			projectURL = fmt.Sprintf("https://%s:%s", hostname, httpsPort)
		}
		output.UserOut.Printf("  Checking Windows trust (Chrome/Edge test) via PowerShell: %s\n", projectURL)

		psScript := fmt.Sprintf(`
try {
    $null = Invoke-WebRequest -Uri '%s' -UseBasicParsing -TimeoutSec 10
    Write-Output "TRUSTED"
} catch [System.Net.WebException] {
    $msg = $_.Exception.Message
    if ($_.Exception.Response -ne $null) {
        Write-Output "CERT_ERROR:$msg"
    } else {
        Write-Output "CONNECT_ERROR:$msg"
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
				output.UserOut.Printf("  ✗ Windows CERT_ERROR: %s\n", msg)
				output.UserOut.Println("    → Windows does not trust the mkcert CA")
				output.UserOut.Println("    → In Windows PowerShell (as Administrator): mkcert -install")
				output.UserOut.Println("    → Make sure CAROOT points to the Windows mkcert directory")
				hasIssues = true
			} else if msg, ok := strings.CutPrefix(result, "CONNECT_ERROR:"); ok {
				output.UserOut.Printf("  ⚠ Windows CONNECT_ERROR (DNS/network, not a cert issue): %s\n", msg)
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
