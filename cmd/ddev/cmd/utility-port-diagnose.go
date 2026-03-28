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
	app, err := ddevapp.GetActiveApp("")
	var ports []namedPort
	inProject := err == nil && app.AppRoot != ""

	if inProject {
		output.UserOut.Printf("Port diagnostics for project: %s\n\n", app.Name)
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
		output.UserOut.Printf("Port %s (%s):\n", np.port, np.label)

		active := netutil.IsPortActive(np.port)

		// On WSL2, also check the Windows side even if the Linux side looks free.
		var windowsProcs []portProcess
		if nodeps.IsWSL2() {
			windowsProcs = findWindowsPortProcesses(np.port)
		}

		if !active && len(windowsProcs) == 0 {
			output.UserOut.Printf("  ✓ Available\n\n")
			continue
		}

		hasConflict = true

		// --- Linux/macOS side ---
		if active {
			linuxProcs := findPortProcesses(np.port)
			if len(linuxProcs) == 0 {
				// IsPortActive says busy but we can't identify the process (e.g. Docker itself)
				output.UserOut.Printf("  ✗ IN USE (process unidentifiable — may be Docker or a container)\n")
			} else {
				for _, p := range linuxProcs {
					printProcess(p)
					for _, hint := range portHints(p.Name, p.Side, p.PID) {
						output.UserOut.Printf("    %s\n", hint)
					}
				}
			}
		}

		// --- Windows side (WSL2 only) ---
		for _, p := range windowsProcs {
			printProcess(p)
			for _, hint := range portHints(p.Name, p.Side, p.PID) {
				output.UserOut.Printf("    %s\n", hint)
			}
		}

		output.UserOut.Println()
	}

	if !hasConflict {
		output.UserOut.Println("All required ports are available.")
	}

	if hasConflict {
		return 1
	}
	return 0
}

// printProcess prints one portProcess entry.
func printProcess(p portProcess) {
	side := ""
	if p.Side != "" {
		side = fmt.Sprintf(" [%s]", p.Side)
	}
	output.UserOut.Printf("  ✗ IN USE%s\n", side)
	output.UserOut.Printf("    Process : %s (PID %d)\n", p.Name, p.PID)
	if p.CmdLine != "" {
		output.UserOut.Printf("    Command : %s\n", p.CmdLine)
	}
}

// findPortProcesses returns processes listening on port on the local (Linux/macOS) side.
// On Windows-native (not WSL2), delegates entirely to findWindowsPortProcesses.
func findPortProcesses(port string) []portProcess {
	if nodeps.IsWindows() {
		return findWindowsPortProcesses(port)
	}

	// Try lsof first (available on macOS and most Linux distros).
	procs, err := findPortProcessesLsof(port)
	if err == nil && len(procs) > 0 {
		return procs
	}

	// Fallback: ss (Linux only).
	if runtime.GOOS == "linux" {
		return findPortProcessesSS(port)
	}

	return nil
}

// findPortProcessesLsof uses lsof to identify listening processes.
func findPortProcessesLsof(port string) ([]portProcess, error) {
	out, err := exec.Command("lsof", "-i", ":"+port, "-sTCP:LISTEN", "-n", "-P", "-F", "pcn").Output()
	if err != nil {
		return nil, err
	}

	// lsof -F output lines look like:
	// p<pid>
	// c<command>
	// n<name>   (address:port)
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
