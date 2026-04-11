package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"net"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/spf13/cobra"
)

// portProcess holds information about a process listening on a port.
type portProcess struct {
	PID     int
	Name    string
	CmdLine string
	Side    string // "Linux", "macOS", "Windows"
}

// PortDiagnoseCmd implements the ddev utility port-diagnose command.
var PortDiagnoseCmd = &cobra.Command{
	Use:   "port-diagnose",
	Short: "Identify processes occupying ports needed by DDEV",
	Long: `Check which ports the current DDEV project needs and identify any
processes that are already occupying those ports.

When run inside a DDEV project directory, checks the ports configured for that
project (HTTP, HTTPS, Mailpit, XHGui). When run outside a project, checks the
default ports 80 and 443.

On WSL2, both the Linux side and the Windows side are checked separately.

Use this command when DDEV reports port conflicts on startup.`,
	Example: `ddev utility port-diagnose
ddev ut port-diagnose`,
	Run: func(cmd *cobra.Command, args []string) {
		exitCode := runPortDiagnose()
		os.Exit(exitCode)
	},
}

func init() {
	DebugCmd.AddCommand(PortDiagnoseCmd)
}

// namedPort pairs a port number with a human-readable label.
type namedPort struct {
	port  string
	label string
}

// runPortDiagnose checks DDEV project ports (or defaults) and reports conflicts.
// Returns 0 if all ports are available, 1 if any conflicts are found.
func runPortDiagnose() int {
	sudoMessageShown = false

	// Check for running DDEV projects or router first — they legitimately use ports.
	activeProjects := ddevapp.GetActiveProjects()
	router, _ := ddevapp.FindDdevRouter()
	if len(activeProjects) > 0 || router != nil {
		var reasons []string
		if len(activeProjects) > 0 {
			names := make([]string, 0, len(activeProjects))
			for _, app := range activeProjects {
				names = append(names, app.Name)
			}
			reasons = append(reasons, fmt.Sprintf("running projects: %s", strings.Join(names, ", ")))
		}
		if router != nil {
			reasons = append(reasons, "ddev-router is running")
		}
		output.UserErr.Printf("DDEV is currently active (%s).\n", strings.Join(reasons, "; "))
		output.UserErr.Println("Running DDEV services use ports that will show as false conflicts.")
		output.UserErr.Println("Please run 'ddev poweroff' first, then re-run this command.")
		return 2
	}

	app, err := ddevapp.GetActiveApp("")
	var ports []namedPort
	inProject := err == nil && app.AppRoot != ""

	if inProject {
		output.UserOut.Printf("Port diagnostics for project: %s\n", app.Name)
		httpPort := app.GetPrimaryRouterHTTPPort()
		httpsPort := app.GetPrimaryRouterHTTPSPort()
		mailpitHTTP := app.GetMailpitHTTPPort()
		mailpitHTTPS := app.GetMailpitHTTPSPort()
		xhguiHTTP := app.GetXHGuiHTTPPort()
		xhguiHTTPS := app.GetXHGuiHTTPSPort()

		for _, np := range []namedPort{
			{httpPort, "router HTTP"},
			{httpsPort, "router HTTPS"},
			{mailpitHTTP, "Mailpit HTTP"},
			{mailpitHTTPS, "Mailpit HTTPS"},
			{xhguiHTTP, "XHGui HTTP"},
			{xhguiHTTPS, "XHGui HTTPS"},
		} {
			if np.port != "" {
				ports = append(ports, np)
			}
		}
	} else {
		output.UserOut.Println("Not in a DDEV project directory — checking default ports 80 and 443.")
		ports = []namedPort{
			{"80", "HTTP"},
			{"443", "HTTPS"},
		}
	}

	hasConflict := false

	for _, np := range ports {
		allProcs := findPortProcesses(np.port)

		// On WSL2, also check the Windows side.
		if nodeps.IsWSL2() {
			allProcs = append(allProcs, findWindowsPortProcesses(np.port)...)
		}

		if len(allProcs) == 0 {
			// Nothing found without elevated privileges.
			// Try to bind the port — if it succeeds, the port is free.
			// This is more reliable than IsPortActive (which checks the Docker IP,
			// not 127.0.0.1) and correctly detects listeners on all interfaces.
			if isPortBindable(np.port) {
				output.UserOut.Printf("Port %s (%s): Available\n", np.port, np.label)
				continue
			}
			// Port is active. Try sudo lsof to identify the owner.
			if !nodeps.IsWindows() && hasCommand("sudo") {
				if !sudoMessageShown {
					output.UserOut.Println("Unable to identify the process without elevated privileges.")
					output.UserOut.Printf("Running: sudo %s -iTCP:%s -sTCP:LISTEN -n -P\n", lsofPath(), np.port)
					if isTerminal(os.Stdin) {
						output.UserOut.Println("You may be prompted for your password.")
					}
					sudoMessageShown = true
				}
				allProcs, _ = findPortProcessesSudoLsof(np.port)
			}
		}

		if len(allProcs) == 0 {
			// Port responds but we still can't identify who owns it.
			hasConflict = true
			hint := fmt.Sprintf("try 'sudo lsof -iTCP:%s -sTCP:LISTEN'", np.port)
			if nodeps.IsWindows() {
				hint = fmt.Sprintf("try 'Get-NetTCPConnection -LocalPort %s -State Listen' in PowerShell", np.port)
			}
			output.UserOut.Printf("Port %s (%s): IN USE (unable to identify process — %s)\n", np.port, np.label, hint)
			continue
		}

		hasConflict = true

		// Deduplicate by process name (e.g. apache2 parent + workers all listen on same port).
		for _, p := range deduplicateByName(allProcs) {
			side := ""
			if p.Side != "" {
				side = fmt.Sprintf(" [%s]", p.Side)
			}
			cmdInfo := ""
			if p.CmdLine != "" && p.CmdLine != p.Name {
				cmdInfo = fmt.Sprintf(", cmd=%s", p.CmdLine)
			}
			output.UserOut.Printf("Port %s (%s): IN USE by %s (PID %d%s)%s\n", np.port, np.label, p.Name, p.PID, cmdInfo, side)
			for _, hint := range portHints(p.Name, p.CmdLine, p.Side, p.PID, np.port) {
				output.UserOut.Printf("  %s\n", hint)
			}
		}
	}

	if !hasConflict {
		output.UserOut.Println("All required ports are available.")
	}

	if hasConflict {
		return 1
	}
	return 0
}

var sudoMessageShown bool

// findPortProcesses returns processes listening on port without elevated privileges.
// On Windows-native (not WSL2), delegates entirely to findWindowsPortProcesses.
// It tries lsof (macOS and most Linux distros), then ss and /proc/net/tcp (Linux only).
// Elevated (sudo) detection is handled by the caller after an IsPortActive confirmation.
func findPortProcesses(port string) []portProcess {
	if nodeps.IsWindows() {
		return findWindowsPortProcesses(port)
	}

	// Try lsof first (available on macOS and most Linux distros).
	if hasCommand("lsof") || hasCommand("/usr/sbin/lsof") {
		procs, err := findPortProcessesLsof(port)
		if err == nil && len(procs) > 0 {
			return procs
		}
	}

	// Fallback: ss (Linux only).
	if runtime.GOOS == "linux" {
		if procs := findPortProcessesSS(port); len(procs) > 0 {
			return procs
		}
		// Last resort: parse /proc/net/tcp to find the inode, then match to a PID.
		return findPortProcessesProcNet(port)
	}

	return nil
}

// findPortProcessesSudoLsof tries lsof with sudo to see processes owned by other users.
// Connects stdin only when running in an interactive terminal so sudo can prompt for a
// password. In non-interactive contexts (CI, scripts) sudo fails immediately rather than
// blocking indefinitely waiting for input.
func findPortProcessesSudoLsof(port string) ([]portProcess, error) {
	cmd := exec.Command("sudo", lsofPath(), "-iTCP:"+port, "-sTCP:LISTEN", "-n", "-P", "-F", "pcnT")
	if isTerminal(os.Stdin) {
		cmd.Stdin = os.Stdin
	}
	// Suppress sudo's own error messages (e.g. "a terminal is required")
	// since we handle failure gracefully by falling through to other methods.
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseLsofOutput(out)
}

// isPortBindable returns true if nothing is currently listening on the port
// on any local interface. It attempts to bind 0.0.0.0:<port> and immediately
// releases it; success means the port is free.
func isPortBindable(port string) bool {
	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return false
	}
	l.Close()
	return true
}

// isTerminal returns true if f is connected to an interactive terminal.
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// findPortProcessesProcNet parses /proc/net/tcp to find the process using a port.
// This works without elevated privileges to find the inode, then scans /proc/*/fd
// to match the inode to a PID.
func findPortProcessesProcNet(port string) []portProcess {
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return nil
	}
	hexPort := fmt.Sprintf("%04X", portNum)

	// Read /proc/net/tcp and /proc/net/tcp6
	var inodes []string
	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for rawLine := range strings.SplitSeq(string(raw), "\n") {
			fields := strings.Fields(rawLine)
			if len(fields) < 10 {
				continue
			}
			// fields[1] = local_address (hex_ip:hex_port), fields[3] = state (0A = LISTEN)
			if fields[3] != "0A" {
				continue
			}
			addrParts := strings.Split(fields[1], ":")
			if len(addrParts) == 2 && addrParts[1] == hexPort {
				inodes = append(inodes, fields[9])
			}
		}
	}

	if len(inodes) == 0 {
		return nil
	}

	// Scan /proc/*/fd to find which PID holds this socket inode
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}

	var results []portProcess
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		fdDir := fmt.Sprintf("/proc/%d/fd", pid)
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}
		matched := false
		for _, fd := range fds {
			if matched {
				break
			}
			link, err := os.Readlink(fmt.Sprintf("%s/%s", fdDir, fd.Name()))
			if err != nil {
				continue
			}
			for _, inode := range inodes {
				if link == "socket:["+inode+"]" {
					name := readProcComm(pid)
					cmdLine := getCommandLine(pid)
					results = appendUniquePID(results, portProcess{
						PID:     pid,
						Name:    name,
						CmdLine: cmdLine,
						Side:    getSide(),
					})
					matched = true
					break
				}
			}
		}
	}
	return results
}

// readProcComm reads the process name from /proc/<pid>/comm.
func readProcComm(pid int) string {
	raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(raw))
}

// lsofPath returns the path to lsof, preferring /usr/sbin/lsof (macOS, some Linux).
func lsofPath() string {
	if hasCommand("/usr/sbin/lsof") {
		return "/usr/sbin/lsof"
	}
	return "lsof"
}

// findPortProcessesLsof uses lsof to identify listening processes.
func findPortProcessesLsof(port string) ([]portProcess, error) {
	out, err := exec.Command(lsofPath(), "-iTCP:"+port, "-sTCP:LISTEN", "-n", "-P", "-F", "pcnT").Output()
	if err != nil {
		return nil, err
	}
	return parseLsofOutput(out)
}

// parseLsofOutput parses lsof -F pcnT output into portProcess entries,
// filtering to only LISTEN-state connections. lsof -F output lines:
// p<pid>, c<command>, n<address:port>, TST=<state>
func parseLsofOutput(out []byte) ([]portProcess, error) {
	type entry struct {
		pid           int
		name          string
		isListen      bool
		hasStateField bool // true if we saw any T line for this entry
	}

	flushEntry := func(current entry) *portProcess {
		// Accept if: explicitly LISTEN, or no state field seen (trust -sTCP:LISTEN filter).
		if current.pid == 0 || (current.hasStateField && !current.isListen) {
			return nil
		}
		cmdLine := getCommandLine(current.pid)
		side := getSide()
		return &portProcess{
			PID:     current.pid,
			Name:    current.name,
			CmdLine: cmdLine,
			Side:    side,
		}
	}

	var (
		results []portProcess
		current entry
	)

	for rawLine := range strings.SplitSeq(string(out), "\n") {
		line := strings.TrimSpace(rawLine)
		if len(line) < 2 {
			continue
		}
		switch line[0] {
		case 'p':
			if p := flushEntry(current); p != nil {
				results = appendUniquePID(results, *p)
			}
			pid, _ := strconv.Atoi(line[1:])
			current = entry{pid: pid}
		case 'c':
			current.name = line[1:]
		case 'T':
			// TCP state field: TST=LISTEN, TST=ESTABLISHED, etc.
			if strings.HasPrefix(line, "TST=") {
				current.hasStateField = true
				if strings.Contains(line, "LISTEN") {
					current.isListen = true
				}
			}
		}
	}
	// flush last entry
	if p := flushEntry(current); p != nil {
		results = appendUniquePID(results, *p)
	}

	return results, nil
}

// findPortProcessesSS uses ss (Linux) to identify listening processes.
// ss -tlnp output example (one line per socket):
// LISTEN 0 128 0.0.0.0:80 0.0.0.0:* users:(("nginx",pid=1234,fd=6))
func findPortProcessesSS(port string) []portProcess {
	out, err := exec.Command("ss", "-tlnp", "sport", "=", ":"+port).Output()
	if err != nil {
		return nil
	}

	var results []portProcess
	for rawLine := range strings.SplitSeq(string(out), "\n") {
		line := strings.TrimSpace(rawLine)
		if !strings.Contains(line, "users:") {
			continue
		}
		// Extract users:(...) section
		usersStart := strings.Index(line, "users:(")
		if usersStart < 0 {
			continue
		}
		usersSection := line[usersStart:]

		// Parse entries like ("nginx",pid=1234,fd=6)
		for part := range strings.SplitSeq(usersSection, "(") {
			if !strings.Contains(part, "pid=") {
				continue
			}
			name := ""
			pid := 0
			for field := range strings.SplitSeq(part, ",") {
				field = strings.Trim(field, "\"()")
				if pidStr, ok := strings.CutPrefix(field, "pid="); ok {
					pid, _ = strconv.Atoi(pidStr)
				} else if !strings.Contains(field, "=") && field != "" {
					name = field
				}
			}
			if pid != 0 && name != "" {
				cmdLine := getCommandLine(pid)
				results = appendUniquePID(results, portProcess{
					PID:     pid,
					Name:    name,
					CmdLine: cmdLine,
					Side:    getSide(),
				})
			}
		}
	}
	return results
}

// findWindowsPortProcesses uses PowerShell Get-NetTCPConnection to identify
// processes listening on port on the Windows side. Works from WSL2 or Windows.
func findWindowsPortProcesses(port string) []portProcess {
	// Validate port is numeric before interpolating into PowerShell script.
	if _, err := strconv.Atoi(port); err != nil {
		return nil
	}
	psScript := fmt.Sprintf(`
Get-NetTCPConnection -LocalPort %s -State Listen -ErrorAction SilentlyContinue |
  ForEach-Object {
    $procId = $_.OwningProcess
    $proc = Get-Process -Id $procId -ErrorAction SilentlyContinue
    $path = ""
    try { $path = $proc.MainModule.FileName } catch {}
    "$procId|$($proc.ProcessName)|$path"
  }
`, port)

	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psScript)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var results []portProcess
	for rawLine := range strings.SplitSeq(string(out), "\n") {
		line := strings.TrimSpace(rawLine)
		// Strip Windows CRLF
		line = strings.TrimSuffix(line, "\r")
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 2 {
			continue
		}
		pid, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		if pid == 0 {
			continue
		}
		name := strings.TrimSpace(parts[1])
		cmdLine := ""
		if len(parts) == 3 {
			cmdLine = strings.TrimSpace(parts[2])
		}
		results = appendUniquePID(results, portProcess{
			PID:     pid,
			Name:    name,
			CmdLine: cmdLine,
			Side:    "Windows",
		})
	}
	return results
}

// getCommandLine returns the full command line for a PID.
// On Linux reads /proc/<pid>/cmdline; on macOS uses ps.
func getCommandLine(pid int) string {
	if runtime.GOOS == "linux" {
		raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if err != nil {
			return ""
		}
		// Arguments are NUL-separated.
		return strings.ReplaceAll(strings.TrimRight(string(raw), "\x00"), "\x00", " ")
	}

	// macOS / other
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "command=").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// getSide returns a human-readable label for the current OS side.
func getSide() string {
	switch {
	case nodeps.IsWSL2():
		return "Linux (WSL2)"
	case nodeps.IsMacOS():
		return "macOS"
	case nodeps.IsWindows():
		return "Windows"
	default:
		return "Linux"
	}
}

// appendUniquePID appends p to results only if no entry with the same PID exists.
func appendUniquePID(results []portProcess, p portProcess) []portProcess {
	for _, existing := range results {
		if existing.PID == p.PID {
			return results
		}
	}
	return append(results, p)
}

// deduplicateByName returns one entry per unique process name, keeping the first (lowest PID).
func deduplicateByName(procs []portProcess) []portProcess {
	seen := make(map[string]bool)
	var result []portProcess
	for _, p := range procs {
		if !seen[p.Name] {
			seen[p.Name] = true
			result = append(result, p)
		}
	}
	return result
}

// portHints returns actionable fix strings based on process name, cmdline, side, and port.
func portHints(name string, cmdLine string, side string, pid int, port string) []string {
	lower := strings.ToLower(name)
	isWindows := strings.Contains(side, "Windows")

	switch {
	case lower == "apache2" || lower == "apache" || lower == "httpd":
		if isWindows {
			return []string{"Stop-Service Apache2; Set-Service Apache2 -StartupType Disabled"}
		}
		if hasCommand("systemctl") {
			hints := []string{"sudo systemctl stop apache2 && sudo systemctl disable apache2"}
			if hasCommand("apt-get") {
				hints = append(hints, "Remove: sudo apt-get remove apache2")
			}
			return hints
		}
		return []string{"sudo apachectl stop"}

	case lower == "nginx":
		if isWindows {
			return []string{"Stop-Service nginx; Set-Service nginx -StartupType Disabled"}
		}
		if hasCommand("systemctl") {
			hints := []string{"sudo systemctl stop nginx && sudo systemctl disable nginx"}
			if hasCommand("apt-get") {
				hints = append(hints, "Remove: sudo apt-get remove nginx")
			}
			return hints
		}
		if hasCommand("brew") {
			return []string{"brew services stop nginx"}
		}
		return []string{fmt.Sprintf("Stop nginx, e.g. 'kill %d'", pid)}

	case lower == "caddy":
		if isWindows {
			return []string{"Stop caddy or Stop-Service caddy (PowerShell as Admin)"}
		}
		if hasCommand("systemctl") {
			return []string{"sudo systemctl stop caddy && sudo systemctl disable caddy"}
		}
		return []string{"sudo caddy stop"}

	case lower == "w3wp" || lower == "iisexpress" || lower == "iis":
		return []string{"Stop-Service W3SVC; Set-Service W3SVC -StartupType Disabled (PowerShell as Admin)"}

	case lower == "docker-proxy":
		return dockerContainerHints(port)

	case (lower == "ssh" || lower == "limactl") && strings.Contains(cmdLine, ".colima"):
		return dockerProviderHints("Colima", port)

	case (lower == "ssh" || lower == "limactl") && strings.Contains(cmdLine, "/.lima/"):
		return dockerProviderHints("Lima", port)

	case strings.HasPrefix(lower, "com.docker") || lower == "docker desktop" || lower == "dockerd":
		return dockerProviderHints("Docker Desktop", port)

	case strings.HasPrefix(lower, "com.orbstack") || strings.HasPrefix(lower, "orbstack"):
		return dockerProviderHints("OrbStack", port)

	case lower == "wslrelay" || lower == "wslrelay.exe":
		return []string{"This port is forwarded from WSL2. Run 'ddev poweroff' inside WSL2 and stop any services using this port there."}

	case lower == "lando" || lower == "traefik":
		return []string{"Lando's Traefik router — run: lando poweroff"}

	default:
		if isWindows {
			return []string{fmt.Sprintf("Stop-Process -Id %d (PowerShell as Admin)", pid)}
		}
		return []string{fmt.Sprintf("Consider stopping this process using OS tools, e.g. 'kill %d'", pid)}
	}
}

// activeDockerProvider returns a human-readable name for the currently active Docker provider.
func activeDockerProvider() string {
	switch {
	case dockerutil.IsColima():
		return "Colima"
	case dockerutil.IsOrbStack():
		return "OrbStack"
	case dockerutil.IsDockerDesktop():
		return "Docker Desktop"
	case dockerutil.IsRancherDesktop():
		return "Rancher Desktop"
	case dockerutil.IsLima():
		return "Lima"
	default:
		return ""
	}
}

// findContainerForPort uses the Docker API to find a running container publishing
// the given host port. Returns the container name, or empty string if not found.
func findContainerForPort(port string) string {
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return ""
	}
	containers, err := dockerutil.GetDockerContainers(false)
	if err != nil {
		return ""
	}
	for _, c := range containers {
		for _, p := range c.Ports {
			if int(p.PublicPort) == portNum && p.IP.IsValid() {
				if len(c.Names) > 0 {
					return strings.TrimPrefix(c.Names[0], "/")
				}
			}
		}
	}
	return ""
}

// dockerContainerHints builds hints when a Docker-internal proxy process holds the port.
// It tries to identify the specific container via docker ps.
func dockerContainerHints(port string) []string {
	if container := findContainerForPort(port); container != "" {
		return []string{
			fmt.Sprintf("Container '%s' is holding this port.", container),
			fmt.Sprintf("Run: docker stop %s", container),
		}
	}
	return []string{
		"A Docker container is holding this port.",
		"Run 'docker ps' to find it and 'docker stop <name>' to free the port.",
	}
}

// dockerProviderHints builds hints when a known Docker provider process holds the port.
// If the provider matches the active one, the port is held by a container — look it up.
// If it's a different provider, suggest stopping that provider instead.
func dockerProviderHints(provider string, port string) []string {
	active := activeDockerProvider()
	if active == provider {
		// The active provider is forwarding this port — find the container responsible.
		return dockerContainerHints(port)
	}
	// A non-active provider is holding the port.
	switch provider {
	case "Colima":
		return []string{
			"Colima is running and holding this port (but is not your active Docker provider).",
			"Stop Colima: colima stop",
		}
	case "OrbStack":
		return []string{
			"OrbStack is running and holding this port (but is not your active Docker provider).",
			"Stop OrbStack from the menu bar or: open -a OrbStack",
		}
	case "Docker Desktop":
		return []string{
			"Docker Desktop is running and holding this port (but is not your active Docker provider).",
			"Quit Docker Desktop from the menu bar or: killall 'Docker Desktop'",
		}
	case "Lima":
		return []string{
			"Lima is running and holding this port (but is not your active Docker provider).",
			"Stop Lima: limactl stop default",
		}
	case "Rancher Desktop":
		return []string{
			"Rancher Desktop is running and holding this port (but is not your active Docker provider).",
			"Quit Rancher Desktop from the menu bar.",
		}
	default:
		return []string{
			fmt.Sprintf("%s is running and holding this port.", provider),
			"Stop it if you are not using it, or stop any containers it is running.",
		}
	}
}

// hasCommand returns true if the named executable is in PATH.
func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
