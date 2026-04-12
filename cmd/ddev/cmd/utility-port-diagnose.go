package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"errors"
	"net"
	"syscall"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/moby/moby/api/types/container"
	"github.com/spf13/cobra"
)

// portProcess holds information about a process listening on a port.
type portProcess struct {
	PID     int
	Name    string
	CmdLine string
	Side    string // "Linux", "macOS", "Windows"
}

// portDiagnoseAllowSudo holds the value of the --allow-sudo flag.
var portDiagnoseAllowSudo bool

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

Some port conflicts are only visible with elevated privileges (e.g. docker-proxy
run by rootful Docker CE). Use --allow-sudo to permit sudo use without being
asked interactively, or run in a terminal to be prompted.

Use this command when DDEV reports port conflicts on startup.`,
	Example: `ddev utility port-diagnose
ddev utility port-diagnose --allow-sudo
ddev ut port-diagnose`,
	Run: func(cmd *cobra.Command, args []string) {
		exitCode := runPortDiagnose()
		os.Exit(exitCode)
	},
}

func init() {
	PortDiagnoseCmd.Flags().BoolVar(&portDiagnoseAllowSudo, "allow-sudo", false, "permit sudo use for elevated port detection without interactive prompt")
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
		// Clear the cached rendered compose YAML so GetPrimaryRouterHTTP*Port reads
		// the current project config rather than stale values from a previous start.
		app.ComposeYaml = nil
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
		httpPort := globalconfig.DdevGlobalConfig.RouterHTTPPort
		httpsPort := globalconfig.DdevGlobalConfig.RouterHTTPSPort
		output.UserOut.Printf("Not in a DDEV project directory — checking default ports %s and %s.\n", httpPort, httpsPort)
		ports = []namedPort{
			{httpPort, "HTTP"},
			{httpsPort, "HTTPS"},
		}
	}

	// On non-Windows systems, some listeners (e.g. docker-proxy under rootful
	// Docker CE) are owned by root and invisible without elevated privileges.
	// Ask the user once upfront whether we may use sudo, showing the exact
	// commands we would run. We only ask when running on an interactive terminal.
	sudoPermitted := askSudoPermission()

	hasConflict := false

	for _, np := range ports {
		// Collect Linux-side processes separately so that finding something on
		// the Windows side (e.g. wslrelay) does not suppress the sudo escalation
		// that is needed to identify a root-owned Linux listener.
		linuxProcs := findPortProcesses(np.port)
		allProcs := append([]portProcess(nil), linuxProcs...)

		// On WSL2, also check the Windows side.
		if nodeps.IsWSL2() {
			allProcs = append(allProcs, findWindowsPortProcesses(np.port)...)
		}

		// On Windows, when PowerShell finds nothing, verify the port is actually
		// free before reporting a conflict — PowerShell misses kernel-level
		// listeners such as HTTP.sys.
		if nodeps.IsWindows() && len(allProcs) == 0 && isPortFree(np.port) {
			output.UserOut.Printf("Port %s (%s): Available\n", np.port, np.label)
			continue
		}

		if len(linuxProcs) == 0 && !nodeps.IsWindows() {
			// No Linux-side process found without elevated privileges.
			// Call isPortFree once and use the result to decide what to do next:
			//   - free on Linux + nothing on Windows → port is available
			//   - free on Linux + Windows found something → port held on Windows only; no sudo needed
			//   - in use on Linux → a root-owned process holds it; try sudo to identify it
			portFreeOnLinux := isPortFree(np.port)

			if portFreeOnLinux {
				if len(allProcs) == 0 {
					output.UserOut.Printf("Port %s (%s): Available\n", np.port, np.label)
					continue
				}
				// Port is free on Linux but Windows found something. Nothing to escalate.
			} else if sudoPermitted {
				var elevatedProcs []portProcess
				if hasLsof() {
					output.UserOut.Printf("Running: %s %s -iTCP:%s -sTCP:LISTEN -n -P\n", sudoFullPath(), lsofPath(), np.port)
					elevatedProcs, _ = findPortProcessesSudoLsof(np.port)
				} else if runtime.GOOS == "linux" && ssFullPath() != "" {
					output.UserOut.Printf("Running: %s %s -tlnp sport = :%s\n", sudoFullPath(), ssFullPath(), np.port)
					elevatedProcs = findPortProcessesSudoSS(np.port)
				}
				for _, p := range elevatedProcs {
					allProcs = appendUniquePID(allProcs, p)
				}
			}
		}

		if len(allProcs) == 0 {
			// Port responds but we still can't identify who owns it.
			hasConflict = true
			hint := "try installing lsof: sudo apt-get install lsof"
			if hasLsof() {
				hint = fmt.Sprintf("try 'sudo lsof -iTCP:%s -sTCP:LISTEN'", np.port)
			}
			if nodeps.IsWindows() {
				hint = fmt.Sprintf("try 'Get-NetTCPConnection -LocalPort %s -State Listen' in PowerShell", np.port)
			}
			output.UserOut.Printf("Port %s (%s): IN USE (unable to identify process — %s)\n", np.port, np.label, hint)
			continue
		}

		hasConflict = true

		// Deduplicate by process name (e.g. apache2 parent + workers all listen on same port),
		// then drop wslrelay when a more specific Docker provider process is also present.
		for _, p := range suppressWSLRelayIfRedundant(deduplicateByName(allProcs)) {
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

// askSudoPermission explains the exact sudo commands that may be run and asks
// the user whether to proceed. Returns true immediately if --allow-sudo was
// passed. Returns false if sudo is unavailable, no elevation tool exists, or
// the user declines. When not on an interactive terminal and --allow-sudo was
// not passed, returns false without prompting.
func askSudoPermission() bool {
	if nodeps.IsWindows() || sudoFullPath() == "" {
		return false
	}
	lsofAvail := hasLsof()
	hasSS := runtime.GOOS == "linux" && ssFullPath() != ""
	if !lsofAvail && !hasSS {
		return false
	}

	if portDiagnoseAllowSudo {
		return true
	}

	output.UserOut.Println("Some port conflicts are only visible with elevated privileges (e.g. root-owned docker-proxy).")
	output.UserOut.Println("If needed, this tool will run one of:")
	if lsofAvail {
		output.UserOut.Printf("  %s %s -iTCP:<port> -sTCP:LISTEN -n -P\n", sudoFullPath(), lsofPath())
	} else {
		output.UserOut.Printf("  %s %s -tlnp sport = :<port>\n", sudoFullPath(), ssFullPath())
	}
	return util.ConfirmTo("Allow sudo use?", false)
}

// sudoFullPath returns the absolute path to sudo.
// sudoFullPath returns the absolute path to sudo from a canonical location,
// or empty string if not found. Only canonical paths are accepted.
func sudoFullPath() string {
	for _, p := range []string{"/usr/bin/sudo", "/bin/sudo"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// ssFullPath returns the absolute path to ss from a canonical location,
// or empty string if not found. Only canonical paths are accepted.
func ssFullPath() string {
	for _, p := range []string{"/usr/sbin/ss", "/sbin/ss", "/bin/ss"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// findPortProcesses returns processes listening on port without elevated privileges.
// On Windows-native (not WSL2), delegates entirely to findWindowsPortProcesses.
// It tries lsof (macOS and most Linux distros), then ss and /proc/net/tcp (Linux only).
// Elevated (sudo) detection is handled by the caller after an IsPortActive confirmation.
func findPortProcesses(port string) []portProcess {
	if nodeps.IsWindows() {
		return findWindowsPortProcesses(port)
	}

	// Try lsof first (available on macOS and most Linux distros).
	if hasLsof() {
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
	cmd := exec.Command(sudoFullPath(), lsofPath(), "-iTCP:"+port, "-sTCP:LISTEN", "-n", "-P", "-F", "pcnT")
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

// isPortFree returns true if nothing is listening on the port.
// It first tries to bind 0.0.0.0:<port>. On Linux, ports below 1024 require
// root; a bind returning EACCES means we lack permission, not that the port
// is occupied. In that case we fall through to a dial-only check.
// If the bind succeeds we also dial 127.0.0.1:<port> to catch providers
// (e.g. Rancher Desktop) that use SO_REUSEPORT, which allows a second bind
// to succeed even when a listener is already present.
func isPortFree(port string) bool {
	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		// EACCES means we can't bind due to permissions (e.g. port < 1024 on
		// Linux without CAP_NET_BIND_SERVICE). Fall through to the dial check
		// rather than assuming the port is occupied.
		var syscallErr *net.OpError
		if errors.As(err, &syscallErr) {
			if errors.Is(syscallErr.Err, syscall.EACCES) {
				// Dial to see if anything actually answers.
				conn, dialErr := net.DialTimeout("tcp", "127.0.0.1:"+port, 250*time.Millisecond)
				if dialErr != nil {
					return true
				}
				conn.Close()
				return false
			}
		}
		// Any other bind error (EADDRINUSE, etc.) — port is in use.
		return false
	}
	l.Close()

	// Bind succeeded. Verify with a dial in case SO_REUSEPORT is in effect.
	conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 250*time.Millisecond)
	if err != nil {
		// Nothing answered — port is free.
		return true
	}
	conn.Close()
	// Something answered the dial even though we could bind — port is in use.
	return false
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

// lsofPath returns the absolute path to lsof from a canonical location,
// or empty string if not found. Only canonical paths are accepted.
func lsofPath() string {
	for _, p := range []string{"/usr/sbin/lsof", "/usr/bin/lsof"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// hasLsof returns true when lsof is available at a canonical path.
func hasLsof() bool {
	return lsofPath() != ""
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

// findPortProcessesSudoSS runs ss with sudo to see root-owned listeners (e.g.
// docker-proxy). Used as a fallback on Linux when lsof is not installed.
func findPortProcessesSudoSS(port string) []portProcess {
	cmd := exec.Command(sudoFullPath(), ssFullPath(), "-tlnp", "sport", "=", ":"+port)
	if isTerminal(os.Stdin) {
		cmd.Stdin = os.Stdin
	}
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	// Reuse the same ss output parser.
	var results []portProcess
	for rawLine := range strings.SplitSeq(string(out), "\n") {
		line := strings.TrimSpace(rawLine)
		if !strings.Contains(line, "users:") {
			continue
		}
		usersStart := strings.Index(line, "users:(")
		if usersStart < 0 {
			continue
		}
		usersSection := line[usersStart:]
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

// suppressWSLRelayIfRedundant removes wslrelay entries when a more specific Docker
// provider process (e.g. com.docker.backend) is also present. wslrelay is purely
// a WSL2→Windows forwarding relay; the provider process carries the actionable hint.
// When wslrelay is the only entry it is kept, since it is the only clue available.
func suppressWSLRelayIfRedundant(procs []portProcess) []portProcess {
	if len(procs) <= 1 {
		return procs
	}
	var result []portProcess
	for _, p := range procs {
		if strings.ToLower(p.Name) == "wslrelay" || strings.ToLower(p.Name) == "wslrelay.exe" {
			continue
		}
		result = append(result, p)
	}
	if len(result) == 0 {
		return procs
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

	case lower == "docker-proxy" || lower == "docker-pr":
		// docker-proxy is used by Docker CE (rootful). Route through dockerProviderHints
		// so that cross-provider cases (e.g. active=Podman, port held by Docker CE) are
		// explained instead of silently falling back to the generic container message.
		return dockerProviderHints("Docker", port)

	case lower == "rootlesskit" || lower == "rootlessk":
		// rootlesskit is Docker rootless's port-forwarding process.
		return dockerProviderHints("Docker (rootless)", port)

	case lower == "rootlessport" || lower == "rootlessp":
		// rootlessport is Podman rootless's port-forwarding process.
		return dockerProviderHints("Podman", port)

	case (lower == "ssh" || lower == "limactl") && strings.Contains(cmdLine, ".colima"):
		return dockerProviderHints("Colima", port)

	case (lower == "ssh" || lower == "limactl") && strings.Contains(cmdLine, "rancher-desktop"):
		return dockerProviderHints("Rancher Desktop", port)

	case (lower == "ssh" || lower == "limactl") && strings.Contains(cmdLine, "/.lima/"):
		return dockerProviderHints("Lima", port)

	case strings.HasPrefix(lower, "com.docker") || lower == "docker desktop" || lower == "dockerd":
		return dockerProviderHints("Docker Desktop", port)

	case strings.HasPrefix(lower, "com.orbstack") || strings.HasPrefix(lower, "orbstack"):
		return dockerProviderHints("OrbStack", port)

	case lower == "wslrelay" || lower == "wslrelay.exe":
		// wslrelay forwards WSL2 ports to the Windows host.
		// It may be a Docker/Rancher Desktop container, or a service in another WSL2 distro.
		if cname := findContainerForPort(port); cname != "" {
			return []string{
				fmt.Sprintf("Container '%s' is forwarded to Windows via WSL2.", cname),
				fmt.Sprintf("Run: docker stop %s", cname),
			}
		}
		return []string{
			"A WSL2 distro is forwarding this port to Windows.",
			"If it is a DDEV container, run 'ddev poweroff' inside WSL2.",
			"Otherwise check which distro holds it — in PowerShell:",
			"  wsl --list",
			"  wsl -d <distro> -- ss -tlnp",
		}

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
	case dockerutil.IsPodman():
		return "Podman"
	case dockerutil.IsDockerRootless():
		return "Docker (rootless)"
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
		return "Docker"
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
	return containerNameForPort(portNum, containers)
}

// containerNameForPort searches a container list for the first container that
// publishes hostPort and returns its name. Extracted for unit testing.
// Podman's container list API omits the IP field from port entries, so we match
// on PublicPort alone — a zero PublicPort (unexposed port) won't match a real port.
func containerNameForPort(hostPort int, containers []container.Summary) string {
	for _, c := range containers {
		for _, p := range c.Ports {
			if int(p.PublicPort) == hostPort {
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
	if cname := findContainerForPort(port); cname != "" {
		return []string{
			fmt.Sprintf("Container '%s' is holding this port.", cname),
			fmt.Sprintf("Run: docker stop %s", cname),
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
	case "Docker":
		// Rancher Desktop in dockerd mode uses docker-proxy internally, so when
		// Rancher Desktop is the active provider and docker-proxy is seen, treat
		// it as a Rancher Desktop container rather than a stray Docker CE instance.
		if active == "Rancher Desktop" {
			return dockerContainerHints(port)
		}
		return []string{
			"Docker CE (rootful) has a container holding this port (but is not your active Docker provider).",
			"Check: sudo docker ps",
			"Stop the container: sudo docker stop <name>",
		}
	case "Docker (rootless)":
		return []string{
			"Docker rootless has a container holding this port (but is not your active Docker provider).",
			fmt.Sprintf("Check: DOCKER_CONTEXT=rootless docker ps  (or DOCKER_HOST=unix:///run/user/%d/docker.sock docker ps)", os.Getuid()),
			"Stop the container: docker stop <name>",
		}
	case "Podman":
		return []string{
			"Podman has a container holding this port (but is not your active Docker provider).",
			"Check: podman ps",
			"Stop the container: podman stop <name>",
		}
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
