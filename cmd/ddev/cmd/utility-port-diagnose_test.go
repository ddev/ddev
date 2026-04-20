package cmd

import (
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/moby/moby/api/types/container"
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
	// macOS nc does not allow -p with -l (port is positional); Linux nc requires -p.
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command(ncPath, "-l", "-k", port)
	} else {
		cmd = exec.Command(ncPath, "-l", "-k", "-p", port)
	}
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

// setGlobalRouterPorts temporarily sets the global router HTTP/HTTPS ports
// and registers a t.Cleanup to restore the original values.
func setGlobalRouterPorts(t *testing.T, httpPort, httpsPort string) {
	t.Helper()
	orig := globalconfig.DdevGlobalConfig
	t.Cleanup(func() {
		globalconfig.DdevGlobalConfig.RouterHTTPPort = orig.RouterHTTPPort
		globalconfig.DdevGlobalConfig.RouterHTTPSPort = orig.RouterHTTPSPort
		_ = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	})
	globalconfig.DdevGlobalConfig.RouterHTTPPort = httpPort
	globalconfig.DdevGlobalConfig.RouterHTTPSPort = httpsPort
	require.NoError(t, globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig))
}

// TestPortDiagnoseConflict verifies that runPortDiagnose detects a conflict
// and reports it. We configure global router ports to unprivileged test ports
// so no root is needed, then bind a listener to force the conflict.
func TestPortDiagnoseConflict(t *testing.T) {
	if nodeps.IsWindows() {
		t.Skip("Windows port detection uses PowerShell; covered by TestFindPortProcessesOwnProcess_Windows")
	}
	// PowerOff stops all projects and the router (which may hold ports on its own).
	ddevapp.PowerOff()
	t.Cleanup(func() { ddevapp.PowerOff() })

	httpPort := getFreePort(t)
	httpsPort := getFreePort(t)
	setGlobalRouterPorts(t, httpPort, httpsPort)

	// Hold the HTTP port to trigger a conflict.
	l, err := net.Listen("tcp", "127.0.0.1:"+httpPort)
	require.NoError(t, err)
	t.Cleanup(func() { l.Close() })

	exitCode := runPortDiagnose()
	require.Equal(t, 1, exitCode, "expected exit code 1 when a port is in use")
}

// TestPortDiagnoseAllFree verifies that runPortDiagnose returns 0 when both
// configured ports are free.
func TestPortDiagnoseAllFree(t *testing.T) {
	if nodeps.IsWindows() {
		t.Skip("Windows port detection uses PowerShell; covered by TestFindPortProcessesOwnProcess_Windows")
	}
	// PowerOff stops all projects and the router (which may hold ports on its own).
	ddevapp.PowerOff()
	t.Cleanup(func() { ddevapp.PowerOff() })

	httpPort := getFreePort(t)
	httpsPort := getFreePort(t)
	setGlobalRouterPorts(t, httpPort, httpsPort)

	exitCode := runPortDiagnose()
	require.Equal(t, 0, exitCode, "expected exit code 0 when both ports are free")
}

// TestFindPortProcessesOwnProcess verifies that we can find our own Go
// test process when it holds a port. Works on all Unix platforms.
// On Windows, see TestFindPortProcessesOwnProcess_Windows.
func TestFindPortProcessesOwnProcess(t *testing.T) {
	if nodeps.IsWindows() {
		t.Skip("uses Unix lsof/ss detection — see TestFindPortProcessesOwnProcess_Windows")
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

// TestFindPortProcessesOwnProcess_Windows verifies that findPortProcesses
// (which delegates to findWindowsPortProcesses on native Windows) can find
// the Go test process holding a port.
func TestFindPortProcessesOwnProcess_Windows(t *testing.T) {
	if !nodeps.IsWindows() {
		t.Skip("Windows-native test")
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	port := strconv.Itoa(l.Addr().(*net.TCPAddr).Port)

	procs := findPortProcesses(port)
	require.NotEmpty(t, procs, "expected to find our own process on port %s", port)

	found := false
	for _, p := range procs {
		if p.PID == os.Getpid() {
			found = true
			require.NotEmpty(t, p.Name, "process name should not be empty")
			require.Equal(t, "Windows", p.Side)
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

// TestFindPortProcessesNC_macOS verifies detection of nc on macOS using non-sudo lsof.
// It calls findPortProcessesLsof directly to avoid the sudo fallback, which would hang
// in CI environments where sudo requires a password but a pseudo-terminal is attached.
// The test skips gracefully if non-sudo lsof cannot see nc (macOS CI restriction).
func TestFindPortProcessesNC_macOS(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-only test")
	}
	if !hasCommand("lsof") && !hasCommand("/usr/sbin/lsof") {
		t.Skip("lsof not available")
	}

	port := getFreePort(t)
	cleanup := startNCListener(t, port)
	t.Cleanup(cleanup)

	procs, err := findPortProcessesLsof(port)
	if err != nil || len(procs) == 0 {
		t.Skip("non-sudo lsof cannot see nc on this system (elevated privileges required)")
	}

	require.Equal(t, "nc", procs[0].Name)
	require.NotZero(t, procs[0].PID)
	require.Equal(t, "macOS", procs[0].Side)
}

// TestFindPortProcessesFreePort verifies that a free port returns no processes.
func TestFindPortProcessesFreePort(t *testing.T) {
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

// TestFindWindowsPortProcesses starts a TCP listener on the Windows side
// via PowerShell and verifies that findWindowsPortProcesses detects it.
func TestFindWindowsPortProcesses(t *testing.T) {
	if !nodeps.IsWindows() && !nodeps.IsWSL2() {
		t.Skip("Windows/WSL2-only test")
	}
	if !hasCommand("powershell.exe") {
		t.Skip("powershell.exe not available")
	}
	// In WSL2 mirrored networking mode, listeners created via WSL2 interop
	// (exec.Command("powershell.exe")) are not visible to Get-NetTCPConnection.
	// Native Windows applications are still detectable; this test setup is not.
	if nodeps.IsWSL2MirroredMode() {
		t.Skip("Get-NetTCPConnection cannot see WSL2-interop-created listeners in mirrored networking mode")
	}

	port := getFreePort(t)

	// Start a .NET TCP listener on the Windows side. The script prints
	// "LISTENING" once the socket is ready, then waits until stdin closes.
	psScript := fmt.Sprintf(`
$l = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, %s)
$l.Start()
Write-Output "LISTENING"
[Console]::In.ReadLine() | Out-Null
$l.Stop()
`, port)

	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psScript)
	// Provide a pipe for stdin so we can signal shutdown.
	stdinPipe, err := cmd.StdinPipe()
	require.NoError(t, err)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())

	defer func() {
		// Close stdin to let the PowerShell script exit, then wait.
		_ = stdinPipe.Close()
		_ = cmd.Wait()
	}()

	// Wait for "LISTENING" to confirm the socket is ready.
	buf := make([]byte, 256)
	n, err := stdout.Read(buf)
	require.NoError(t, err)
	require.Contains(t, string(buf[:n]), "LISTENING", "PowerShell listener should print LISTENING")

	procs := findWindowsPortProcesses(port)
	require.NotEmpty(t, procs, "expected to find Windows listener on port %s", port)
	require.NotEmpty(t, procs[0].Name, "process name should not be empty")
	require.NotZero(t, procs[0].PID, "process PID should not be zero")
	require.Equal(t, "Windows", procs[0].Side)
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

// TestSuppressWSLRelayIfRedundant verifies that wslrelay is dropped when a more
// specific Docker provider process is also present, but kept when it is the only entry.
func TestSuppressWSLRelayIfRedundant(t *testing.T) {
	relay := portProcess{PID: 10028, Name: "wslrelay", Side: "Windows"}
	backend := portProcess{PID: 24204, Name: "com.docker.backend", Side: "Windows"}
	other := portProcess{PID: 999, Name: "nginx", Side: "Windows"}

	// wslrelay alone — must be kept.
	result := suppressWSLRelayIfRedundant([]portProcess{relay})
	require.Len(t, result, 1)
	require.Equal(t, "wslrelay", result[0].Name)

	// wslrelay.exe alone — must be kept.
	relayExe := portProcess{PID: 10028, Name: "wslrelay.exe", Side: "Windows"}
	result = suppressWSLRelayIfRedundant([]portProcess{relayExe})
	require.Len(t, result, 1)

	// wslrelay alongside com.docker.backend — wslrelay suppressed.
	result = suppressWSLRelayIfRedundant([]portProcess{relay, backend})
	require.Len(t, result, 1)
	require.Equal(t, "com.docker.backend", result[0].Name)

	// wslrelay alongside any other process — wslrelay suppressed.
	result = suppressWSLRelayIfRedundant([]portProcess{relay, other})
	require.Len(t, result, 1)
	require.Equal(t, "nginx", result[0].Name)

	// Non-wslrelay processes — returned unchanged.
	result = suppressWSLRelayIfRedundant([]portProcess{backend, other})
	require.Len(t, result, 2)
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

// TestContainerNameForPort verifies that containerNameForPort finds containers by
// PublicPort regardless of whether the IP field is populated. This specifically
// guards against a regression where Podman's container-list API (which omits the
// IP field) caused the lookup to always return an empty string.
func TestContainerNameForPort(t *testing.T) {
	dockerIP := netip.MustParseAddr("0.0.0.0")

	tests := []struct {
		name       string
		hostPort   int
		containers []container.Summary
		want       string
	}{
		{
			name:     "docker-ce style with IP populated",
			hostPort: 80,
			containers: []container.Summary{
				{Names: []string{"/my-container"}, Ports: []container.PortSummary{
					{PublicPort: 80, PrivatePort: 80, Type: "tcp", IP: dockerIP},
				}},
			},
			want: "my-container",
		},
		{
			name:     "podman style with IP absent (zero netip.Addr)",
			hostPort: 80,
			containers: []container.Summary{
				{Names: []string{"/podman-container"}, Ports: []container.PortSummary{
					{PublicPort: 80, PrivatePort: 80, Type: "tcp"},
				}},
			},
			want: "podman-container",
		},
		{
			name:     "port not matched",
			hostPort: 443,
			containers: []container.Summary{
				{Names: []string{"/other"}, Ports: []container.PortSummary{
					{PublicPort: 80, PrivatePort: 80, Type: "tcp"},
				}},
			},
			want: "",
		},
		{
			name:     "unexposed port (PublicPort zero) does not match",
			hostPort: 80,
			containers: []container.Summary{
				{Names: []string{"/no-publish"}, Ports: []container.PortSummary{
					{PublicPort: 0, PrivatePort: 80, Type: "tcp"},
				}},
			},
			want: "",
		},
		{
			name:     "leading slash stripped from name",
			hostPort: 8080,
			containers: []container.Summary{
				{Names: []string{"/ddev-myproject-web"}, Ports: []container.PortSummary{
					{PublicPort: 8080, PrivatePort: 80, Type: "tcp"},
				}},
			},
			want: "ddev-myproject-web",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containerNameForPort(tt.hostPort, tt.containers)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestPortHints verifies hint generation for well-known process names.
func TestPortHints(t *testing.T) {
	tests := []struct {
		name     string
		cmdLine  string
		side     string
		pid      int
		contains string
	}{
		// Apache hints vary by platform (systemctl on Linux, apachectl on macOS),
		// but always mention "apache" somewhere in the output.
		{"apache2", "", "Linux", 1, "apache"},
		{"httpd", "", "macOS", 1, "apache"},
		// nginx/caddy non-Windows hints depend on hasCommand (systemctl/brew)
		// at runtime, so test the Windows path here (deterministic) and test
		// platform-native paths in TestPortHintsPlatformSpecific.
		{"nginx", "", "Windows", 1, "nginx"},
		{"caddy", "", "Linux", 1, "caddy"},
		{"w3wp", "", "Windows", 1, "W3SVC"},
		// com.docker.backend: when Docker Desktop is the active provider the hint
		// describes the container; when it is not active it names Docker Desktop.
		// Both paths mention "port".
		{"com.docker.backend", "", "macOS", 1, "port"},
		{"com.orbstack.backend", "", "macOS", 1, "port"},
		// Docker-provider hints are environment-dependent (output varies based on which
		// provider is currently active), so just verify we don't fall through to the
		// generic "kill" hint — all provider branches mention "port".
		{"OrbStack Helper", "", "macOS", 1, "port"},
		{"docker-proxy", "", "Linux", 1, "port"},
		// Docker rootless and Podman process names — output varies based on active provider,
		// but all branches mention "port" (container hints) or the provider name.
		{"rootlesskit", "", "Linux", 1, "port"},
		{"rootlessk", "", "Linux", 1, "port"},
		{"rootlessport", "", "Linux", 1, "port"},
		{"rootlessp", "", "Linux", 1, "port"},
		{"ssh", "/.colima/_lima/colima/ssh.sock", "macOS", 1, "port"},
		{"ssh", "/rancher-desktop/lima/0/ssh.sock", "macOS", 1, "port"},
		{"limactl", "/.lima/default/ha.sock", "macOS", 1, "port"},
		// wslrelay: port="" so findContainerForPort("") returns "" → "no container" path.
		// Both name variants (wslrelay and wslrelay.exe) must be handled.
		{"wslrelay", "", "Windows", 1, "WSL2"},
		{"wslrelay.exe", "", "Windows", 1, "WSL2"},
		{"lando", "", "Linux", 1, "lando poweroff"},
		{"traefik", "", "Linux", 1, "lando poweroff"},
		{"someunknown", "", "Linux", 42, "kill 42"},
		{"someunknown", "", "Windows", 42, "Stop-Process"},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_"+tt.side, func(t *testing.T) {
			hints := portHints(tt.name, tt.cmdLine, tt.side, tt.pid, "")
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
		hints := portHints("apache2", "", "Linux", 1, "")
		combined := strings.Join(hints, " ")
		if hasCommand("systemctl") {
			require.Contains(t, combined, "systemctl")
		} else {
			require.Contains(t, combined, "apachectl")
		}
		// nginx on Linux
		hints = portHints("nginx", "", "Linux", 1, "")
		combined = strings.Join(hints, " ")
		require.Contains(t, combined, "nginx")
	case "darwin":
		hints := portHints("apache2", "", "macOS", 1, "")
		combined := strings.Join(hints, " ")
		// macOS should never suggest systemctl
		require.NotContains(t, combined, "systemctl")
		require.Contains(t, combined, "apachectl")
		// nginx on macOS
		hints = portHints("nginx", "", "macOS", 1, "")
		combined = strings.Join(hints, " ")
		require.Contains(t, combined, "nginx")
	case "windows":
		// Windows apache hints should use Stop-Service
		hints := portHints("apache2", "", "Windows", 1, "")
		combined := strings.Join(hints, " ")
		require.Contains(t, combined, "Stop-Service")
		// Unknown process should suggest Stop-Process
		hints = portHints("someprocess", "", "Windows", 999, "")
		combined = strings.Join(hints, " ")
		require.Contains(t, combined, "Stop-Process")
		require.Contains(t, combined, "999")
	default:
		t.Skip("platform-specific hint test not applicable")
	}
}
