package cmd

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// generateSelfSignedCA creates a self-signed CA cert+key PEM pair in dir.
// Returns the CA cert, CA key, and CA cert pool.
func generateSelfSignedCA(t *testing.T, dir string) (*x509.Certificate, *ecdsa.PrivateKey, *x509.CertPool) {
	t.Helper()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "mkcert test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	caDERBytes, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	caCert, err := x509.ParseCertificate(caDERBytes)
	require.NoError(t, err)

	// Write rootCA.pem
	caPEMPath := filepath.Join(dir, "rootCA.pem")
	f, err := os.Create(caPEMPath)
	require.NoError(t, err)
	require.NoError(t, pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: caDERBytes}))
	require.NoError(t, f.Close())

	// Write rootCA-key.pem
	caKeyDER, err := x509.MarshalECPrivateKey(caKey)
	require.NoError(t, err)
	caKeyPEMPath := filepath.Join(dir, "rootCA-key.pem")
	fk, err := os.Create(caKeyPEMPath)
	require.NoError(t, err)
	require.NoError(t, pem.Encode(fk, &pem.Block{Type: "EC PRIVATE KEY", Bytes: caKeyDER}))
	require.NoError(t, fk.Close())

	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	return caCert, caKey, pool
}

// generateLeafCert issues a leaf cert signed by the given CA for the given
// hostnames. Writes it to certPath and returns the DER bytes.
func generateLeafCert(t *testing.T, caCert *x509.Certificate, caKey *ecdsa.PrivateKey, hostnames []string, notAfter time.Time, certPath string) []byte {
	t.Helper()
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: hostnames[0]},
		DNSNames:     hostnames,
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, caCert, &leafKey.PublicKey, caKey)
	require.NoError(t, err)

	f, err := os.Create(certPath)
	require.NoError(t, err)
	require.NoError(t, pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: leafDER}))
	require.NoError(t, f.Close())

	return leafDER
}

// realCARoot returns the mkcert CAROOT path, skipping the test if mkcert is
// not installed. It does NOT modify any state.
func realCARoot(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("mkcert"); err != nil {
		t.Skip("mkcert not in PATH — skipping")
	}
	out, err := exec.Command("mkcert", "-CAROOT").Output()
	require.NoError(t, err, "mkcert -CAROOT failed")
	caRoot := strings.TrimSpace(string(out))
	require.NotEmpty(t, caRoot, "mkcert -CAROOT returned empty path")
	return caRoot
}

// ---------------------------------------------------------------------------
// Smoke / integration
// ---------------------------------------------------------------------------

// TestTLSDiagnoseSmoke verifies that runTLSDiagnose completes without panicking
// and returns a valid exit code. On a properly configured machine (mkcert
// installed, CA trusted) exit code 0 is expected.
func TestTLSDiagnoseSmoke(t *testing.T) {
	restore := util.CaptureUserOut()
	exitCode := runTLSDiagnose()
	restore()

	require.Contains(t, []int{0, 1}, exitCode, "runTLSDiagnose must return 0 or 1")
}

// TestTLSDiagnoseOutputSections verifies that every expected section header
// appears in the diagnostic output regardless of whether issues are found.
// This protects against accidental removal of sections.
func TestTLSDiagnoseOutputSections(t *testing.T) {
	restore := util.CaptureUserOut()
	exitCode := runTLSDiagnose()
	out := restore()

	require.Contains(t, []int{0, 1}, exitCode)

	sections := []string{
		"mkcert Installation",
		"OS Trust Store",
		"Firefox",
		"Certificate Files",
		"Summary",
	}
	for _, section := range sections {
		require.Contains(t, out, section, "output must contain section %q", section)
	}
	if nodeps.IsWSL2() {
		require.Contains(t, out, "WSL2 Configuration", "WSL2 output must contain WSL2 section")
	}
}

// TestTLSDiagnoseCommandRegistered verifies that tls-diagnose is registered as
// a subcommand of the utility/debug command.
func TestTLSDiagnoseCommandRegistered(t *testing.T) {
	found := false
	for _, sub := range DebugCmd.Commands() {
		if sub.Name() == "tls-diagnose" {
			found = true
			break
		}
	}
	require.True(t, found, "tls-diagnose must be registered as a subcommand of DebugCmd")
}

// ---------------------------------------------------------------------------
// Real mkcert installation checks (no Docker, no state changes)
// ---------------------------------------------------------------------------

// TestCheckMkcertInstallationHealthy calls checkMkcertInstallation() directly
// on a machine where mkcert is properly installed (i.e. every DDEV dev/CI
// machine). Verifies the function finds the CA files and returns no issues.
func TestCheckMkcertInstallationHealthy(t *testing.T) {
	caRoot := realCARoot(t)

	if nodeps.IsWSL2() && !strings.HasPrefix(caRoot, "/mnt/") && nodeps.IsEnvFalse("DDEV_RUN_TEST_ANYWAY") {
		// buildkite-agent (systemd) doesn't inherit WSLENV, so CAROOT points to the
		// Linux-local mkcert dir rather than the shared Windows one. Skip until the
		// runner hooks/environment is configured to set CAROOT from the Windows registry.
		t.Skipf("CAROOT (%s) doesn't point to Windows filesystem on WSL2 — hooks/environment not yet configured; skip unless DDEV_RUN_TEST_ANYWAY=true", caRoot)
	}

	// Verify the files mkcert -install creates actually exist before calling
	// the function, so any failure is attributable to the function itself.
	require.FileExists(t, filepath.Join(caRoot, "rootCA.pem"))
	require.FileExists(t, filepath.Join(caRoot, "rootCA-key.pem"))

	restore := util.CaptureUserOut()
	gotCARoot, hasIssues := checkMkcertInstallation()
	restore()

	require.False(t, hasIssues, "mkcert installation should be healthy in a configured environment")
	require.Equal(t, caRoot, gotCARoot, "returned CAROOT should match mkcert -CAROOT output")
}

// TestLoadCACertPoolWithRealCA verifies that loadCACertPool succeeds against
// the actual mkcert CA installed on the machine.
func TestLoadCACertPoolWithRealCA(t *testing.T) {
	caRoot := realCARoot(t)

	pool, err := loadCACertPool(caRoot)
	require.NoError(t, err)
	require.NotNil(t, pool)
}

// TestCertThumbprintRoundTrip verifies that certThumbprintFromFile returns
// the same value on repeated calls (deterministic) and is the right length.
func TestCertThumbprintRoundTrip(t *testing.T) {
	caRoot := realCARoot(t)
	pemPath := filepath.Join(caRoot, "rootCA.pem")

	t1, err := certThumbprintFromFile(pemPath)
	require.NoError(t, err)
	t2, err := certThumbprintFromFile(pemPath)
	require.NoError(t, err)

	require.Equal(t, t1, t2, "certThumbprintFromFile must be deterministic")
	require.Len(t, t1, 40, "SHA1 thumbprint must be 40 hex characters")
	require.Equal(t, strings.ToUpper(t1), t1, "thumbprint must be uppercase")
}

// TestCheckCertFileWithRealMkcert generates a certificate using the real
// mkcert binary (same pipeline DDEV uses), then validates it through
// checkCertFile. This exercises the complete chain: real CA → real cert
// → Go x509 verification.
func TestCheckCertFileWithRealMkcert(t *testing.T) {
	caRoot := realCARoot(t)

	dir := t.TempDir()
	certPath := filepath.Join(dir, "test.crt")
	keyPath := filepath.Join(dir, "test.key")

	hostname := "myproject.ddev.site"
	out, err := exec.Command("mkcert",
		"-cert-file", certPath,
		"-key-file", keyPath,
		hostname,
	).CombinedOutput()
	require.NoError(t, err, "mkcert should generate a cert: %s", out)
	require.FileExists(t, certPath)

	pool, err := loadCACertPool(caRoot)
	require.NoError(t, err)

	restore := util.CaptureUserOut()
	issues := checkCertFile(certPath, pool, []string{hostname}, "real-mkcert cert")
	restore()

	require.False(t, issues, "mkcert-generated cert must validate against the real CA")
}

// TestCheckCertFileRealMkcertWrongHostname verifies that a real mkcert cert
// does NOT validate when a different hostname is expected.
func TestCheckCertFileRealMkcertWrongHostname(t *testing.T) {
	caRoot := realCARoot(t)

	dir := t.TempDir()
	certPath := filepath.Join(dir, "test.crt")
	keyPath := filepath.Join(dir, "test.key")

	out, err := exec.Command("mkcert",
		"-cert-file", certPath,
		"-key-file", keyPath,
		"other.ddev.site",
	).CombinedOutput()
	require.NoError(t, err, "mkcert should generate a cert: %s", out)

	pool, err := loadCACertPool(caRoot)
	require.NoError(t, err)

	restore := util.CaptureUserOut()
	issues := checkCertFile(certPath, pool, []string{"myproject.ddev.site"}, "hostname mismatch")
	restore()

	require.True(t, issues, "cert for other.ddev.site should not validate for myproject.ddev.site")
}

// TestCheckCertFileDefaultCert validates the real DDEV global default cert
// (~/.ddev/traefik/certs/default_cert.crt) against the real mkcert CA.
// Skipped if the cert has not yet been generated (no project has ever started).
func TestCheckCertFileDefaultCert(t *testing.T) {
	caRoot := realCARoot(t)

	defaultCert := filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik", "certs", "default_cert.crt")
	if _, err := os.Stat(defaultCert); os.IsNotExist(err) {
		t.Skip("default_cert.crt not found — start any DDEV project once to generate it")
	}

	pool, err := loadCACertPool(caRoot)
	require.NoError(t, err)

	restore := util.CaptureUserOut()
	// No expected hostnames: the default cert covers a wildcard, not a
	// single project hostname, so we only check validity and CA trust.
	issues := checkCertFile(defaultCert, pool, []string{}, "default_cert.crt")
	restore()

	require.False(t, issues, "real default_cert.crt must be valid against the current mkcert CA")
}

// TestCheckCertificateFilesHealthy calls checkCertificateFiles directly with
// the real CA root and no project (app=nil), which checks only the global
// default cert. Skipped if the cert does not exist yet.
func TestCheckCertificateFilesHealthy(t *testing.T) {
	caRoot := realCARoot(t)

	defaultCert := filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik", "certs", "default_cert.crt")
	if _, err := os.Stat(defaultCert); os.IsNotExist(err) {
		t.Skip("default_cert.crt not found — start any DDEV project once to generate it")
	}

	restore := util.CaptureUserOut()
	issues := checkCertificateFiles(caRoot, nil)
	restore()

	require.False(t, issues, "checkCertificateFiles must find no issues in a healthy environment")
}

// TestCheckCertificateFilesRotatedCA verifies that checkCertificateFiles
// detects a CA mismatch: when we pass a freshly generated CA that did not
// sign the real default cert, it must report issues.
func TestCheckCertificateFilesRotatedCA(t *testing.T) {
	defaultCert := filepath.Join(globalconfig.GetGlobalDdevDir(), "traefik", "certs", "default_cert.crt")
	if _, err := os.Stat(defaultCert); os.IsNotExist(err) {
		t.Skip("default_cert.crt not found — start any DDEV project once to generate it")
	}

	// Use a synthetic CA that did not sign the real cert — simulates a
	// rotated CA where old certs are no longer trusted.
	dir := t.TempDir()
	generateSelfSignedCA(t, dir)

	restore := util.CaptureUserOut()
	issues := checkCertificateFiles(dir, nil)
	restore()

	require.True(t, issues, "real cert signed by a different CA must be flagged as untrusted")
}

// ---------------------------------------------------------------------------
// Live connectivity (requires a running DDEV project — Docker)
// ---------------------------------------------------------------------------

// TestCheckLiveConnectivityWithProject starts TestSites[0], runs
// checkLiveConnectivity against it, then stops it. This is the most realistic
// integration test: it exercises tls.Dial against a real DDEV HTTPS endpoint.
//
// On WSL2, checkLiveConnectivity performs two independent checks:
//   - Linux/WSL2-side: tls.Dial to localhost (what the DDEV process can reach)
//   - Windows-side: PowerShell Invoke-WebRequest (what Chrome/Edge would see)
//
// Both are asserted separately so a failure on one side doesn't obscure the other.
func TestCheckLiveConnectivityWithProject(t *testing.T) {
	if len(TestSites) == 0 {
		t.Skip("no TestSites configured")
	}
	caRoot := realCARoot(t)

	site := TestSites[0]
	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = app.Stop(false, false)
	})

	require.NoError(t, app.Start(), "failed to start TestSites[0]")

	restore := util.CaptureUserOut()
	linuxIssues, windowsIssues := checkLiveConnectivity(caRoot, app)
	out := restore()

	// Linux/WSL2-side TLS must always succeed on a properly configured machine.
	require.False(t, linuxIssues, "Linux/WSL2-side TLS connection to running project must succeed")
	require.Contains(t, out, "TLS verified", "output must confirm Linux-side TLS success")

	if nodeps.IsWSL2() {
		caRootIsWindowsMount := strings.HasPrefix(caRoot, "/mnt/")
		if caRootIsWindowsMount || !nodeps.IsEnvFalse("DDEV_RUN_TEST_ANYWAY") {
			// Windows-side check: only assert when CAROOT points to the shared Windows CA
			// (buildkite-agent with hooks/environment configured) or forced via env var.
			require.Contains(t, out, "Checking Windows trust", "Windows-side connectivity check must run on WSL2")
			require.False(t, windowsIssues, "Windows-side TLS connection to running project must succeed on WSL2")
			require.Contains(t, out, "Windows Invoke-WebRequest: TRUSTED", "Windows must trust the DDEV certificate")
		} else {
			t.Logf("Skipping Windows connectivity assertions: CAROOT=%s is not a Windows mount — buildkite-agent likely lacks WSLENV propagation", caRoot)
		}
	}
}

// TestCheckLiveConnectivityNotRunning verifies that checkLiveConnectivity
// returns false (no issues) and skips gracefully when the project is stopped.
// A stopped project is not a TLS problem.
func TestCheckLiveConnectivityNotRunning(t *testing.T) {
	if len(TestSites) == 0 {
		t.Skip("no TestSites configured")
	}
	caRoot := realCARoot(t)

	site := TestSites[0]
	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)

	// Ensure the project is stopped
	_ = app.Stop(false, false)

	restore := util.CaptureUserOut()
	linuxIssues, windowsIssues := checkLiveConnectivity(caRoot, app)
	out := restore()

	require.False(t, linuxIssues, "a stopped project is not a Linux-side TLS issue — should return no issues")
	require.False(t, windowsIssues, "a stopped project is not a Windows-side TLS issue — should return no issues")
	require.Contains(t, out, "not running", "output should explain why connectivity was skipped")
}

// ---------------------------------------------------------------------------
// verifyCATrustedByOS
// ---------------------------------------------------------------------------

// TestVerifyCATrustedByOSSynthetic generates a CA entirely in memory (never
// installed in any trust store) and confirms verifyCATrustedByOS returns false.
func TestVerifyCATrustedByOSSynthetic(t *testing.T) {
	dir := t.TempDir()
	caCert, caKey, _ := generateSelfSignedCA(t, dir)

	trusted, err := verifyCATrustedByOS(caCert, caKey)
	require.NoError(t, err, "verifyCATrustedByOS should not error for a synthetic CA")
	require.False(t, trusted, "synthetic CA should not be trusted by the OS")
}

// TestVerifyCATrustedByOSReal loads the actual mkcert CA cert and key from
// CAROOT and confirms the OS trusts them. This passes only on machines where
// mkcert -install has been run — i.e. every properly configured DDEV dev and
// CI machine.
func TestVerifyCATrustedByOSReal(t *testing.T) {
	caRoot := realCARoot(t)

	caCertBytes, err := os.ReadFile(filepath.Join(caRoot, "rootCA.pem"))
	require.NoError(t, err)
	caKeyBytes, err := os.ReadFile(filepath.Join(caRoot, "rootCA-key.pem"))
	require.NoError(t, err)

	caCert, err := parseCertFromPEM(caCertBytes)
	require.NoError(t, err)
	caKey, err := parsePKCS8Key(caKeyBytes)
	require.NoError(t, err)

	trusted, err := verifyCATrustedByOS(caCert, caKey)
	require.NoError(t, err)
	require.True(t, trusted, "real mkcert CA must be trusted by the OS — run mkcert -install if this fails")
}

// TestParsePKCS8Key verifies that parsePKCS8Key handles the key formats mkcert
// has historically used. Uses a synthetic ECDSA key (simplest to generate in
// tests); RSA path is exercised implicitly by TestVerifyCATrustedByOSReal on
// machines that have an RSA mkcert CA.
func TestParsePKCS8Key(t *testing.T) {
	dir := t.TempDir()
	generateSelfSignedCA(t, dir) // writes rootCA-key.pem as EC PRIVATE KEY (our test helper uses ECDSA)

	keyBytes, err := os.ReadFile(filepath.Join(dir, "rootCA-key.pem"))
	require.NoError(t, err)

	key, err := parsePKCS8Key(keyBytes)
	require.NoError(t, err)
	require.NotNil(t, key)

	// Error on empty input
	_, err = parsePKCS8Key([]byte("not a key"))
	require.Error(t, err)
}

// TestParseCertFromPEM verifies that parseCertFromPEM handles valid and invalid
// PEM input correctly.
func TestParseCertFromPEM(t *testing.T) {
	dir := t.TempDir()
	generateSelfSignedCA(t, dir)

	certBytes, err := os.ReadFile(filepath.Join(dir, "rootCA.pem"))
	require.NoError(t, err)

	cert, err := parseCertFromPEM(certBytes)
	require.NoError(t, err)
	require.NotNil(t, cert)
	require.True(t, cert.IsCA)

	// Error on non-PEM input
	_, err = parseCertFromPEM([]byte("not a cert"))
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Pure-Go unit tests (no external tools, no Docker)
// ---------------------------------------------------------------------------

// TestWslenvHasCARootEntry verifies that wslenvHasCARootEntry correctly detects
// a missing CAROOT entry and a malformed WSLENV value. The malformed case
// reproduces the real-world failure where WSLENV="CAROOT/up;1" passes a naive
// substring check but semicolons are not valid WSLENV separators and silently
// prevent CAROOT from propagating into WSL2.
func TestWslenvHasCARootEntry(t *testing.T) {
	tests := []struct {
		input         string
		wantHasCARoot bool
		wantMalformed bool
	}{
		// Happy paths
		{"CAROOT/up", true, false},
		{"WT_SESSION:WT_PROFILE_ID:CAROOT/up", true, false},
		{"CAROOT", true, false},
		// Missing CAROOT
		{"", false, false},
		{"WT_SESSION:WT_PROFILE_ID", false, false},
		// Malformed: semicolons in the value (the exact case reported)
		{"WT_SESSION:WT_PROFILE_ID:CAROOT/up;1", false, true},
		{"CAROOT/up;", false, true},
		{"CAROOT/up;extra", false, true},
	}

	for _, tc := range tests {
		hasCARoot, malformed := wslenvHasCARootEntry(tc.input)
		require.Equal(t, tc.wantHasCARoot, hasCARoot, "hasCARoot for input %q", tc.input)
		require.Equal(t, tc.wantMalformed, malformed, "malformed for input %q", tc.input)
	}
}

// TestWindowsPathToWSL verifies the Windows→WSL path conversion helper.
func TestWindowsPathToWSL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`C:\Users\foo\AppData\Local\mkcert`, "/mnt/c/Users/foo/AppData/Local/mkcert"},
		{`D:\mkcert`, "/mnt/d/mkcert"},
		// Path already in WSL form — pass through unchanged
		{"/mnt/c/Users/foo", "/mnt/c/Users/foo"},
		// Trailing carriage-return from Windows line endings is stripped
		{"C:\\Users\\foo\r", "/mnt/c/Users/foo"},
	}

	for _, tc := range tests {
		result := windowsPathToWSL(tc.input)
		require.Equal(t, tc.expected, result, "input: %q", tc.input)
	}
}

// TestWindowsPathFromWSLCAROOT verifies the WSL CAROOT → Windows path helper.
func TestWindowsPathFromWSLCAROOT(t *testing.T) {
	if !nodeps.IsWSL2() {
		t.Skip("windowsPathFromWSLCAROOT is only exercised on WSL2")
	}

	t.Setenv("CAROOT", "/mnt/c/Users/testuser/AppData/Local/mkcert")
	result := windowsPathFromWSLCAROOT()
	require.Equal(t, `C:\Users\testuser\AppData\Local\mkcert`, result)
}

// TestCertThumbprintFromFile verifies certThumbprintFromFile with synthetic
// certs: correct format on a valid PEM, errors on missing/malformed files.
func TestCertThumbprintFromFile(t *testing.T) {
	dir := t.TempDir()
	_, _, _ = generateSelfSignedCA(t, dir)

	pemPath := filepath.Join(dir, "rootCA.pem")
	thumbprint, err := certThumbprintFromFile(pemPath)
	require.NoError(t, err)
	require.NotEmpty(t, thumbprint)
	require.Equal(t, strings.ToUpper(thumbprint), thumbprint, "thumbprint should be uppercase")
	require.Len(t, thumbprint, 40, "SHA1 hex is 40 characters")

	_, err = certThumbprintFromFile(filepath.Join(dir, "nonexistent.pem"))
	require.Error(t, err)

	badPath := filepath.Join(dir, "bad.pem")
	require.NoError(t, os.WriteFile(badPath, []byte("not a pem"), 0600))
	_, err = certThumbprintFromFile(badPath)
	require.Error(t, err)
}

// TestLoadCACertPool verifies that loadCACertPool succeeds with a valid CA dir
// and fails on a non-existent path.
func TestLoadCACertPool(t *testing.T) {
	dir := t.TempDir()
	generateSelfSignedCA(t, dir)

	pool, err := loadCACertPool(dir)
	require.NoError(t, err)
	require.NotNil(t, pool)

	_, err = loadCACertPool(filepath.Join(dir, "nonexistent"))
	require.Error(t, err)
}

// TestCheckCertFileValid verifies that checkCertFile returns false (no issues)
// for a valid cert signed by the CA, covering the expected hostname.
func TestCheckCertFileValid(t *testing.T) {
	dir := t.TempDir()
	caCert, caKey, caPool := generateSelfSignedCA(t, dir)

	certPath := filepath.Join(dir, "leaf.crt")
	generateLeafCert(t, caCert, caKey, []string{"myproject.ddev.site"}, time.Now().Add(24*time.Hour), certPath)

	restore := util.CaptureUserOut()
	issues := checkCertFile(certPath, caPool, []string{"myproject.ddev.site"}, "test cert")
	restore()

	require.False(t, issues, "valid cert with correct hostname should report no issues")
}

// TestCheckCertFileExpired verifies that checkCertFile flags an expired cert.
func TestCheckCertFileExpired(t *testing.T) {
	dir := t.TempDir()
	caCert, caKey, caPool := generateSelfSignedCA(t, dir)

	certPath := filepath.Join(dir, "expired.crt")
	generateLeafCert(t, caCert, caKey, []string{"myproject.ddev.site"}, time.Now().Add(-time.Hour), certPath)

	restore := util.CaptureUserOut()
	issues := checkCertFile(certPath, caPool, []string{"myproject.ddev.site"}, "expired cert")
	restore()

	require.True(t, issues, "expired cert should report issues")
}

// TestCheckCertFileWrongHostname verifies that checkCertFile flags a hostname
// that is not covered by the cert's SANs.
func TestCheckCertFileWrongHostname(t *testing.T) {
	dir := t.TempDir()
	caCert, caKey, caPool := generateSelfSignedCA(t, dir)

	certPath := filepath.Join(dir, "wrong.crt")
	generateLeafCert(t, caCert, caKey, []string{"other.ddev.site"}, time.Now().Add(24*time.Hour), certPath)

	restore := util.CaptureUserOut()
	issues := checkCertFile(certPath, caPool, []string{"myproject.ddev.site"}, "wrong-hostname cert")
	restore()

	require.True(t, issues, "cert not covering expected hostname should report issues")
}

// TestCheckCertFileWrongCA verifies that checkCertFile flags a cert that was
// signed by a different CA than the one in caPool.
func TestCheckCertFileWrongCA(t *testing.T) {
	dir := t.TempDir()
	caCert, caKey, _ := generateSelfSignedCA(t, dir)

	dir2 := t.TempDir()
	_, _, otherPool := generateSelfSignedCA(t, dir2)

	certPath := filepath.Join(dir, "leaf.crt")
	generateLeafCert(t, caCert, caKey, []string{"myproject.ddev.site"}, time.Now().Add(24*time.Hour), certPath)

	restore := util.CaptureUserOut()
	issues := checkCertFile(certPath, otherPool, []string{"myproject.ddev.site"}, "wrong-CA cert")
	restore()

	require.True(t, issues, "cert signed by different CA should report issues")
}

// TestCheckCertFileMissing verifies that checkCertFile flags a missing file.
func TestCheckCertFileMissing(t *testing.T) {
	dir := t.TempDir()
	_, _, caPool := generateSelfSignedCA(t, dir)

	restore := util.CaptureUserOut()
	issues := checkCertFile(filepath.Join(dir, "nonexistent.crt"), caPool, []string{}, "missing cert")
	restore()

	require.True(t, issues, "missing cert file should report issues")
}
