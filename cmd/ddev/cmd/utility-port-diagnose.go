package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/netutil"
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
	// Check for running DDEV projects first — they legitimately use ports.
	activeProjects := ddevapp.GetActiveProjects()
	if len(activeProjects) > 0 {
		names := make([]string, 0, len(activeProjects))
		for _, app := range activeProjects {
			names = append(names, app.Name)
		}
		output.UserErr.Printf("DDEV projects currently running: %s\n", strings.Join(names, ", "))
		output.UserErr.Println("Running projects use ports that will show as conflicts.")
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
		active := netutil.IsPortActive(np.port)

		// On WSL2, also check the Windows side even if the Linux side looks free.
		var windowsProcs []portProcess
		if nodeps.IsWSL2() {
			windowsProcs = findWindowsPortProcesses(np.port)
		}

		if !active && len(windowsProcs) == 0 {
			output.UserOut.Printf("Port %s (%s): Available\n", np.port, np.label)
			continue
		}

		hasConflict = true

		// Collect all processes for this port.
		var allProcs []portProcess
		if active {
			allProcs = findPortProcesses(np.port)
		}
		allProcs = append(allProcs, windowsProcs...)

		if len(allProcs) == 0 {
			output.UserOut.Printf("Port %s (%s): IN USE (process unidentifiable — may be Docker or a container)\n", np.port, np.label)
		} else {
			for _, p := range allProcs {
				side := ""
				if p.Side != "" {
					side = fmt.Sprintf(" [%s]", p.Side)
				}
				cmdInfo := ""
				if p.CmdLine != "" && p.CmdLine != p.Name {
					cmdInfo = fmt.Sprintf(", cmd=%s", p.CmdLine)
				}
				output.UserOut.Printf("Port %s (%s): IN USE by %s (PID %d%s)%s\n", np.port, np.label, p.Name, p.PID, cmdInfo, side)
				for _, hint := range portHints(p.Name, p.Side, p.PID) {
					output.UserOut.Printf("  %s\n", hint)
				}
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

// findPortProcesses returns processes listening on port on the local (Linux/macOS) side.
// On Windows-native (not WSL2), delegates entirely to findWindowsPortProcesses.
// It tries multiple detection methods: lsof, sudo lsof, ss, and /proc/net/tcp.
func findPortProcesses(port string) []portProcess {
	if nodeps.IsWindows() {
		return findWindowsPortProcesses(port)
	}

	// Try lsof first (available on macOS and most Linux distros).
	if hasCommand("lsof") {
		procs, err := findPortProcessesLsof(port)
		if err == nil && len(procs) > 0 {
			return procs
		}

		// On Linux, lsof without root can't see processes owned by other users.
		// Try sudo lsof (non-interactive) if regular lsof returned nothing.
		if runtime.GOOS == "linux" && hasCommand("sudo") {
			procs, err = findPortProcessesSudoLsof(port)
			if err == nil && len(procs) > 0 {
				return procs
			}
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
func findPortProcessesSudoLsof(port string) ([]portProcess, error) {
	out, err := exec.Command("sudo", "-n", "lsof", "-i", ":"+port, "-sTCP:LISTEN", "-n", "-P", "-F", "pcn").Output()
	if err != nil {
		return nil, err
	}
	return parseLsofOutput(out)
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
		for _, fd := range fds {
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

// findPortProcessesLsof uses lsof to identify listening processes.
func findPortProcessesLsof(port string) ([]portProcess, error) {
	out, err := exec.Command("lsof", "-i", ":"+port, "-sTCP:LISTEN", "-n", "-P", "-F", "pcn").Output()
	if err != nil {
		return nil, err
	}
	return parseLsofOutput(out)
}

// parseLsofOutput parses lsof -F pcn output into portProcess entries.
// lsof -F output lines: p<pid>, c<command>, n<name> (address:port)
func parseLsofOutput(out []byte) ([]portProcess, error) {
	type entry struct {
		pid  int
		name string
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
			if current.pid != 0 {
				cmdLine := getCommandLine(current.pid)
				side := getSide()
				results = appendUniquePID(results, portProcess{
					PID:     current.pid,
					Name:    current.name,
					CmdLine: cmdLine,
					Side:    side,
				})
			}
			pid, _ := strconv.Atoi(line[1:])
			current = entry{pid: pid}
		case 'c':
			current.name = line[1:]
		}
	}
	// flush last entry
	if current.pid != 0 {
		cmdLine := getCommandLine(current.pid)
		side := getSide()
		results = appendUniquePID(results, portProcess{
			PID:     current.pid,
			Name:    current.name,
			CmdLine: cmdLine,
			Side:    side,
		})
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
	psScript := fmt.Sprintf(`
Get-NetTCPConnection -LocalPort %s -State Listen -ErrorAction SilentlyContinue |
  ForEach-Object {
    $pid = $_.OwningProcess
    $proc = Get-Process -Id $pid -ErrorAction SilentlyContinue
    $path = ""
    try { $path = $proc.MainModule.FileName } catch {}
    "$pid|$($proc.ProcessName)|$path"
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

// portHints returns actionable fix strings based on process name and side.
func portHints(name string, side string, pid int) []string {
	lower := strings.ToLower(name)
	isWindows := strings.Contains(side, "Windows")

	switch {
	case lower == "apache2" || lower == "apache" || lower == "httpd":
		if isWindows {
			return []string{
				"Fix     : Stop-Service Apache2",
				"Disable : Set-Service Apache2 -StartupType Disabled",
			}
		}
		var hints []string
		if hasCommand("systemctl") {
			hints = append(hints,
				"Fix     : sudo systemctl stop apache2",
				"Disable : sudo systemctl disable apache2",
			)
		} else {
			hints = append(hints, "Fix     : sudo apachectl stop")
		}
		if hasCommand("apt-get") {
			hints = append(hints, "Remove  : sudo apt-get remove apache2")
		} else if hasCommand("brew") {
			hints = append(hints, "Remove  : brew uninstall httpd")
		}
		return hints

	case lower == "nginx":
		if isWindows {
			return []string{
				"Fix     : Stop-Service nginx  (or net stop nginx)",
				"Disable : Set-Service nginx -StartupType Disabled",
			}
		}
		var hints []string
		if hasCommand("systemctl") {
			hints = append(hints,
				"Fix     : sudo systemctl stop nginx",
				"Disable : sudo systemctl disable nginx",
			)
		} else if hasCommand("brew") {
			hints = append(hints, "Fix     : brew services stop nginx")
		}
		if hasCommand("apt-get") {
			hints = append(hints, "Remove  : sudo apt-get remove nginx")
		} else if hasCommand("brew") {
			hints = append(hints, "Remove  : brew uninstall nginx")
		}
		return hints

	case lower == "caddy":
		if hasCommand("systemctl") {
			return []string{
				"Fix     : sudo systemctl stop caddy",
				"Disable : sudo systemctl disable caddy",
			}
		}
		return []string{"Fix     : sudo caddy stop"}

	case lower == "w3wp" || lower == "iisexpress" || lower == "iis":
		return []string{
			"Fix     : Stop-Service W3SVC  (run in Windows PowerShell as Administrator)",
			"Disable : Set-Service W3SVC -StartupType Disabled",
		}

	case strings.HasPrefix(lower, "com.docker") || lower == "docker desktop" || lower == "dockerd":
		return []string{
			"Note    : This port appears to be used by Docker Desktop itself.",
			"Fix     : Restart Docker Desktop, or check your Docker Compose port mappings.",
		}

	case strings.HasPrefix(lower, "com.orbstack") || lower == "orbstack":
		return []string{
			"Note    : This port appears to be used by OrbStack.",
			"Fix     : Restart OrbStack, or check your container port mappings.",
		}

	case lower == "lando" || lower == "traefik":
		return []string{
			"Fix     : lando poweroff",
			"Note    : Lando's Traefik router is occupying this port.",
		}

	default:
		if isWindows {
			return []string{fmt.Sprintf("Fix     : Stop-Process -Id %d  (run in Windows PowerShell as Administrator)", pid)}
		}
		return []string{fmt.Sprintf("Fix     : sudo kill %d", pid)}
	}
}

// hasCommand returns true if the named executable is in PATH.
func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
