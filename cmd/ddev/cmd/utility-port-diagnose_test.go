package cmd

import (
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
)

// getFreePort returns an available TCP port.
func getFreePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := strconv.Itoa(l.Addr().(*net.TCPAddr).Port)
	l.Close()
	return port
}

// startNCListener starts nc listening on the given port and returns a cleanup function.
// Skips the test if nc is not available.
func startNCListener(t *testing.T, port string) func() {
	t.Helper()
	ncPath, err := exec.LookPath("nc")
	if err != nil {
		t.Skip("nc not available — skipping")
	}
	cmd := exec.Command(ncPath, "-l", "-k", "-p", port)
	require.NoError(t, cmd.Start(), "failed to start nc on port %s", port)
	return func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}

// TestPortDiagnoseAvailablePort verifies that runPortDiagnose runs without panic.
func TestPortDiagnoseAvailablePort(t *testing.T) {
	exitCode := runPortDiagnose()
	// runPortDiagnose checks project ports or defaults.
	// Exit code 0 = all clear, 1 = conflicts, 2 = DDEV running.
	// We just verify it runs and returns a valid code.
	require.Contains(t, []int{0, 1, 2}, exitCode)
}

// TestFindPortProcessesOwnProcess verifies that we can find our own Go
// test process when it holds a port. This works on all Unix platforms.
func TestFindPortProcessesOwnProcess(t *testing.T) {
	if nodeps.IsWindows() {
		t.Skip("uses Unix lsof/ss detection — see TestFindWindowsPortProcesses")
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	port := strconv.Itoa(l.Addr().(*net.TCPAddr).Port)

	procs := findPortProcesses(port)
	if len(procs) == 0 {
		t.Skip("lsof/ss not available or returned no results")
	}

	found := false
	for _, p := range procs {
		if p.PID == os.Getpid() {
			found = true
			require.NotEmpty(t, p.Name, "process name should not be empty")
		}
	}
	require.True(t, found, "expected our own PID (%d) in results for port %s", os.Getpid(), port)
}

// TestFindPortProcessesNC_Linux verifies detection of nc on Linux (non-WSL2)
// using the lsof -> ss -> /proc/net/tcp chain.
func TestFindPortProcessesNC_Linux(t *testing.T) {
	if runtime.GOOS != "linux" || nodeps.IsWSL2() {
		t.Skip("Linux-only test (non-WSL2)")
	}

	port := getFreePort(t)
	cleanup := startNCListener(t, port)
	defer cleanup()

	procs := findPortProcesses(port)
	require.NotEmpty(t, procs, "expected to find nc on port %s", port)
	require.Equal(t, "nc", procs[0].Name)
	require.NotZero(t, procs[0].PID)
	require.Contains(t, procs[0].Side, "Linux")
}

// TestFindPortProcessesNC_WSL2 verifies detection of nc on WSL2.
func TestFindPortProcessesNC_WSL2(t *testing.T) {
	if !nodeps.IsWSL2() {
		t.Skip("WSL2-only test")
	}

	port := getFreePort(t)
	cleanup := startNCListener(t, port)
	defer cleanup()

	procs := findPortProcesses(port)
	require.NotEmpty(t, procs, "expected to find nc on port %s", port)
	require.Equal(t, "nc", procs[0].Name)
	require.NotZero(t, procs[0].PID)
	require.Contains(t, procs[0].Side, "WSL2")
}

// TestFindPortProcessesNC_macOS verifies detection of nc on macOS using lsof.
func TestFindPortProcessesNC_macOS(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-only test")
	}

	port := getFreePort(t)
	cleanup := startNCListener(t, port)
	defer cleanup()

	procs := findPortProcesses(port)
	require.NotEmpty(t, procs, "expected to find nc on port %s", port)
	require.Equal(t, "nc", procs[0].Name)
	require.NotZero(t, procs[0].PID)
	require.Equal(t, "macOS", procs[0].Side)
}

// TestFindPortProcessesFreePort verifies that a free port returns no processes.
func TestFindPortProcessesFreePort(t *testing.T) {
	if nodeps.IsWindows() {
		t.Skip("uses Unix detection — see TestFindWindowsPortProcesses")
	}

	port := getFreePort(t)
	procs := findPortProcesses(port)
	require.Empty(t, procs, "expected no processes on free port %s", port)
}

// TestFindPortProcessesLsof directly tests the lsof detection method.
func TestFindPortProcessesLsof(t *testing.T) {
	if !hasCommand("lsof") && !hasCommand("/usr/sbin/lsof") {
		t.Skip("lsof not available")
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	port := strconv.Itoa(l.Addr().(*net.TCPAddr).Port)

	procs, err := findPortProcessesLsof(port)
	require.NoError(t, err)
	require.NotEmpty(t, procs, "lsof should find our own process on port %s", port)

	found := false
	for _, p := range procs {
		if p.PID == os.Getpid() {
			found = true
		}
	}
	require.True(t, found, "lsof should find PID %d on port %s", os.Getpid(), port)
}

// TestFindPortProcessesSS directly tests the ss detection method (Linux only).
func TestFindPortProcessesSS(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("ss is Linux-only")
	}
	if !hasCommand("ss") {
		t.Skip("ss not available")
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	port := strconv.Itoa(l.Addr().(*net.TCPAddr).Port)

	procs := findPortProcessesSS(port)
	// ss without root may not show process info for our own process,
	// so we just verify it doesn't error. If it does return results, check them.
	for _, p := range procs {
		require.NotEmpty(t, p.Name)
		require.NotZero(t, p.PID)
	}
}

// TestFindPortProcessesProcNet directly tests the /proc/net/tcp detection method (Linux only).
func TestFindPortProcessesProcNet(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("/proc/net/tcp is Linux-only")
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	port := strconv.Itoa(l.Addr().(*net.TCPAddr).Port)

	procs := findPortProcessesProcNet(port)
	require.NotEmpty(t, procs, "/proc/net/tcp should find our own process on port %s", port)

	found := false
	for _, p := range procs {
		if p.PID == os.Getpid() {
			found = true
			require.NotEmpty(t, p.Name)
		}
	}
	require.True(t, found, "/proc/net/tcp should find PID %d on port %s", os.Getpid(), port)
}

// TestFindWindowsPortProcesses tests Windows-side detection (Windows or WSL2 only).
func TestFindWindowsPortProcesses(t *testing.T) {
	if !nodeps.IsWindows() && !nodeps.IsWSL2() {
		t.Skip("Windows/WSL2-only test")
	}
	if !hasCommand("powershell.exe") {
		t.Skip("powershell.exe not available")
	}

	// Port 445 (SMB) is almost always in use on Windows.
	procs := findWindowsPortProcesses("445")
	// We can't guarantee SMB is running, but if results come back, verify structure.
	for _, p := range procs {
		require.NotEmpty(t, p.Name, "Windows process name should not be empty")
		require.NotZero(t, p.PID, "Windows process PID should not be zero")
		require.Equal(t, "Windows", p.Side)
	}
}

// TestParseLsofOutputFiltersListen verifies that parseLsofOutput only includes
// LISTEN-state connections and ignores ESTABLISHED/other states.
func TestParseLsofOutputFiltersListen(t *testing.T) {
	// Simulated lsof -F pcnT output with a mix of LISTEN and ESTABLISHED connections.
	// This reproduces the macOS issue where Chrome/Discord outbound connections
	// to remote port 443 were incorrectly reported as port conflicts.
	lsofOutput := []byte(`p26365
chttpd
n*:443
TST=LISTEN
p1014
cGoogle Chrome Helper
n10.0.0.1:54321->142.250.80.46:443
TST=ESTABLISHED
p1861
cDiscord Helper
n10.0.0.1:54322->162.159.128.233:443
TST=ESTABLISHED
`)
	procs, err := parseLsofOutput(lsofOutput)
	require.NoError(t, err)
	// Only httpd (LISTEN) should be included; Chrome and Discord (ESTABLISHED) should be filtered out.
	require.Len(t, procs, 1, "should only include LISTEN connections")
	require.Equal(t, "httpd", procs[0].Name)
	require.Equal(t, 26365, procs[0].PID)
}

// TestParseLsofOutputNoStateField verifies that entries without a TST= field
// are accepted (some macOS lsof versions may not emit the T field).
func TestParseLsofOutputNoStateField(t *testing.T) {
	lsofOutput := []byte(`p12345
cnc
n*:8142
`)
	procs, err := parseLsofOutput(lsofOutput)
	require.NoError(t, err)
	require.Len(t, procs, 1, "entry without TST= should be accepted")
	require.Equal(t, "nc", procs[0].Name)
	require.Equal(t, 12345, procs[0].PID)
}

// TestDeduplicateByName verifies that multiple PIDs with the same name are collapsed.
func TestDeduplicateByName(t *testing.T) {
	procs := []portProcess{
		{PID: 100, Name: "apache2", Side: "Linux"},
		{PID: 101, Name: "apache2", Side: "Linux"},
		{PID: 102, Name: "apache2", Side: "Linux"},
		{PID: 200, Name: "nginx", Side: "Linux"},
	}
	result := deduplicateByName(procs)
	require.Len(t, result, 2)
	require.Equal(t, "apache2", result[0].Name)
	require.Equal(t, 100, result[0].PID, "should keep the first PID")
	require.Equal(t, "nginx", result[1].Name)
}

// TestPortHints verifies hint generation for well-known process names.
func TestPortHints(t *testing.T) {
	tests := []struct {
		name     string
		side     string
		pid      int
		contains string
	}{
		// Apache hints vary by platform (systemctl on Linux, apachectl on macOS),
		// but always mention "apache" somewhere in the output.
		{"apache2", "Linux", 1, "apache"},
		{"httpd", "macOS", 1, "apache"},
		{"nginx", "macOS", 1, "nginx"},
		{"nginx", "Linux", 1, "nginx"},
		{"caddy", "Linux", 1, "caddy"},
		{"w3wp", "Windows", 1, "W3SVC"},
		{"com.docker.backend", "macOS", 1, "Docker Desktop"},
		{"com.orbstack.backend", "macOS", 1, "OrbStack"},
		{"lando", "Linux", 1, "lando poweroff"},
		{"traefik", "Linux", 1, "lando poweroff"},
		{"someunknown", "Linux", 42, "sudo kill 42"},
		{"someunknown", "Windows", 42, "Stop-Process"},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_"+tt.side, func(t *testing.T) {
			hints := portHints(tt.name, tt.side, tt.pid)
			require.NotEmpty(t, hints, "hints should not be empty for process %q", tt.name)
			combined := strings.Join(hints, " ")
			require.Contains(t, combined, tt.contains, "hints for %q on %s should contain %q", tt.name, tt.side, tt.contains)
		})
	}
}

// TestPortHintsPlatformSpecific verifies that hints use the right commands
// for the current platform (relies on runtime hasCommand checks).
func TestPortHintsPlatformSpecific(t *testing.T) {
	switch runtime.GOOS {
	case "linux":
		hints := portHints("apache2", "Linux", 1)
		combined := strings.Join(hints, " ")
		if hasCommand("systemctl") {
			require.Contains(t, combined, "systemctl")
		} else {
			require.Contains(t, combined, "apachectl")
		}
	case "darwin":
		hints := portHints("apache2", "macOS", 1)
		combined := strings.Join(hints, " ")
		// macOS should never suggest systemctl
		require.NotContains(t, combined, "systemctl")
		require.Contains(t, combined, "apachectl")
	default:
		t.Skip("platform-specific hint test not applicable")
	}
}
