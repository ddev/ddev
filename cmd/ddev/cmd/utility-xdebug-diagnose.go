package cmd

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/netutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// XdebugDiagnoseCmd implements the ddev utility xdebug-diagnose command
var XdebugDiagnoseCmd = &cobra.Command{
	Use:   "xdebug-diagnose",
	Short: "Diagnose Xdebug configuration and connectivity",
	Long: `Diagnose Xdebug setup and test connectivity between IDE and web container.

This command checks:
- Whether IDE is already listening on port 9003
- Network connectivity from web container to host
- Xdebug configuration and status
- host.docker.internal resolution
- Global xdebug_ide_location setting

Use this command when experiencing issues with Xdebug step debugging.`,
	Example: `ddev utility xdebug-diagnose
ddev ut xdebug-diagnose`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 0 {
			util.Failed("This command takes no additional arguments")
		}

		exitCode := runXdebugDiagnose()
		os.Exit(exitCode)
	},
}

func init() {
	DebugCmd.AddCommand(XdebugDiagnoseCmd)
}

// runXdebugDiagnose performs the diagnostic checks and outputs results
// Returns exit code: 0 if no issues, 1 if issues found
func runXdebugDiagnose() int {
	hasIssues := false

	// Try to load app from current directory
	app, err := ddevapp.GetActiveApp("")
	if err != nil {
		util.Warning("Not in a DDEV project directory.")
		util.Warning("Please run this command from within a DDEV project.")
		return 1
	}

	// Check if project is running
	status, _ := app.SiteStatus()
	if status != ddevapp.SiteRunning {
		output.UserOut.Printf("Project '%s' is not running. Starting project for diagnostics...", app.Name)
		output.UserOut.Println()
		err := app.Start()
		if err != nil {
			util.Failed("Failed to start project: %v", err)
			return 1
		}
	}

	output.UserOut.Printf("Xdebug Diagnostics for Project: %s", app.Name)
	output.UserOut.Println()

	// Check 1: Check if something is already listening on port 9003
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Port 9003 Pre-Check")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// In WSL2 NAT mode, the IDE runs on Windows, so check Windows port
	isWSL2NAT := nodeps.IsWSL2() && !nodeps.IsWSL2MirroredMode()
	var portInUse bool
	if isWSL2NAT {
		portInUse = isWindowsPortInUse(9003)
		if portInUse {
			output.UserOut.Println("  ✓ Port 9003 is in use on Windows")
			output.UserOut.Println("    This is likely your IDE listening for Xdebug connections (which is good!)")
			output.UserOut.Println()
			output.UserOut.Println("  To identify what's listening on port 9003, run in PowerShell:")
			output.UserOut.Println("    Get-NetTCPConnection -LocalPort 9003 | Select-Object OwningProcess")
		} else {
			output.UserOut.Println("  ℹ Port 9003 is not currently in use on Windows")
			output.UserOut.Println("    Your IDE should be listening on this port for Xdebug to work.")
			hasIssues = true
		}
	} else {
		portInUse = netutil.IsPortActive("9003")
		if portInUse {
			output.UserOut.Println("  ⚠ Port 9003 is already in use on the host")
			output.UserOut.Println("    This is likely your IDE listening for Xdebug connections (which is good!)")
			output.UserOut.Println("    Or it could be another process interfering with Xdebug.")
			output.UserOut.Println()
			output.UserOut.Println("  To identify what's listening on port 9003, run:")
			output.UserOut.Println("    • Linux/macOS: sudo lsof -i :9003 -sTCP:LISTEN")
			output.UserOut.Println("    • Windows: netstat -ano | findstr :9003")
		} else {
			output.UserOut.Println("  ℹ Port 9003 is not currently in use on the host")
			output.UserOut.Println("    Your IDE should be listening on this port for Xdebug to work.")
			hasIssues = true
		}
	}
	output.UserOut.Println()

	// Check 2: Get host.docker.internal information
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("host.docker.internal Configuration")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	hostDockerInternal := dockerutil.GetHostDockerInternal()
	output.UserOut.Printf("  IP address: %s\n", hostDockerInternal.IPAddress)
	output.UserOut.Printf("  Derivation: %s\n", hostDockerInternal.Message)
	output.UserOut.Println()

	// Check 3: Check xdebug_ide_location setting
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Global Configuration")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	xdebugIDELocation := globalconfig.DdevGlobalConfig.XdebugIDELocation
	if xdebugIDELocation == "" {
		output.UserOut.Println("  ✓ xdebug_ide_location is set to default (empty)")
		output.UserOut.Println("    This is correct for most users.")
	} else {
		output.UserOut.Printf("  ⚠ xdebug_ide_location is set to: '%s'\n", xdebugIDELocation)
		output.UserOut.Println("    This should only be set for special cases (WSL2 IDE, container IDE, etc.)")
		output.UserOut.Println("    If you're having issues, try: ddev config global --xdebug-ide-location=\"\"")
	}
	output.UserOut.Println()

	// Check 4: Start a test listener and test connection
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Connection Test")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Only run connection test if port 9003 is not already in use
	if !portInUse {
		// Detect if we're in WSL2 NAT mode
		isWSL2NAT := nodeps.IsWSL2() && !nodeps.IsWSL2MirroredMode()

		if isWSL2NAT {
			output.UserOut.Println("  Detected WSL2 NAT mode - testing connection to Windows host...")
			hasIssues = testWSL2NATConnection(app) || hasIssues
		} else {
			output.UserOut.Println("  Starting test listener on host port 9003...")
			hasIssues = testSimpleConnection(app) || hasIssues
		}
	} else {
		output.UserOut.Println("  ℹ Skipping connection test (port 9003 already in use)")
		output.UserOut.Println("    To test connectivity, stop your IDE's debug listener and run this command again.")
	}
	output.UserOut.Println()

	// Check 5: Check Xdebug status
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Xdebug Status")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	statusOut, _, _ := app.Exec(&ddevapp.ExecOpts{
		Cmd: "xdebug status",
	})
	output.UserOut.Printf("  Current status: %s\n", strings.TrimSpace(statusOut))

	if strings.Contains(statusOut, "disabled") {
		output.UserOut.Println("  ℹ Xdebug is currently disabled")
		output.UserOut.Println("    Enable with: ddev xdebug on")
	}
	output.UserOut.Println()

	// Check 6: Test Xdebug with PHP if enabled, or enable temporarily
	wasEnabled := !strings.Contains(statusOut, "disabled")
	if !wasEnabled {
		output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		output.UserOut.Println("Xdebug PHP Test")
		output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		output.UserOut.Println("  Enabling Xdebug temporarily for testing...")

		_, _, err := app.Exec(&ddevapp.ExecOpts{
			Cmd: "xdebug on",
		})
		if err != nil {
			output.UserOut.Printf("  ✗ Failed to enable Xdebug: %v\n", err)
			hasIssues = true
		} else {
			output.UserOut.Println("  ✓ Xdebug enabled")

			// Test with PHP
			phpOut, _, err := app.Exec(&ddevapp.ExecOpts{
				Cmd: "php -m",
			})
			if err == nil {
				if strings.Contains(phpOut, "xdebug") {
					output.UserOut.Println("  ✓ Xdebug is loaded in PHP")
					// Check for xdebug.mode
					if strings.Contains(phpOut, "xdebug.mode") {
						output.UserOut.Println("  ✓ xdebug.mode is configured")
					}
				} else {
					output.UserOut.Println("  ✗ Xdebug does not appear to be loaded in PHP")
					hasIssues = true
				}
			}

			// Disable Xdebug again if it wasn't enabled before
			output.UserOut.Println("  Disabling Xdebug...")
			_, _, _ = app.Exec(&ddevapp.ExecOpts{
				Cmd: "xdebug off",
			})
			output.UserOut.Println("  ✓ Xdebug disabled (restored previous state)")
		}
		output.UserOut.Println()
	}

	// Summary
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Summary")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if hasIssues {
		output.UserOut.Println("  ⚠ Issues detected. See recommendations below.")
		output.UserOut.Println()
		output.UserOut.Println("Recommendations:")
		output.UserOut.Println("  1. Ensure your IDE is listening for Xdebug on port 9003")
		output.UserOut.Println("  2. Check firewall settings - port 9003 must be open")
		output.UserOut.Println("  3. If using WSL2, VPN, or proxy, see:")
		output.UserOut.Println("     https://ddev.readthedocs.io/en/stable/users/debugging-profiling/step-debugging/#troubleshooting-xdebug")
		output.UserOut.Println("  4. Try: ddev config global --xdebug-ide-location=\"\"")
		output.UserOut.Println("  5. Enable Xdebug with: ddev xdebug on")
		output.UserOut.Println()
		return 1
	}

	output.UserOut.Println("  ✓ No major issues detected - Xdebug configuration looks good!")
	output.UserOut.Println()
	output.UserOut.Println("Next steps:")
	output.UserOut.Println("  1. Enable Xdebug: ddev xdebug on")
	output.UserOut.Println("  2. Start your IDE's debug listener")
	output.UserOut.Println("  3. Set a breakpoint in your code")
	output.UserOut.Println("  4. Visit your site in a browser")
	output.UserOut.Println()

	return 0
}

// testSimpleConnection tests connection using a simple TCP listener (for non-WSL2-NAT scenarios)
func testSimpleConnection(app *ddevapp.DdevApp) bool {
	hasIssues := false

	// Start listener in background
	listener, err := net.Listen("tcp", "0.0.0.0:9003")
	if err != nil {
		output.UserOut.Printf("  ✗ Failed to start test listener: %v\n", err)
		return true
	}
	defer listener.Close()
	output.UserOut.Println("  ✓ Test listener started on 0.0.0.0:9003")

	// Accept connections in background
	connChan := make(chan net.Conn, 1)
	errChan := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			errChan <- err
			return
		}
		connChan <- conn
	}()

	// Test connection from container
	output.UserOut.Println("  Testing connection from web container...")
	ncOut, _, ncErr := app.Exec(&ddevapp.ExecOpts{
		Cmd: "bash -c 'echo test from container | nc -w 2 host.docker.internal 9003'",
	})

	// Wait for connection with timeout
	select {
	case conn := <-connChan:
		output.UserOut.Println("  ✓ Successfully connected from web container to host port 9003")
		buf := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, err := conn.Read(buf)
		if err == nil && n > 0 {
			message := strings.TrimSpace(string(buf[:n]))
			output.UserOut.Printf("  ✓ Received message: '%s'\n", message)
		}
		conn.Close()
	case err := <-errChan:
		output.UserOut.Printf("  ✗ Failed to accept connection: %v\n", err)
		hasIssues = true
	case <-time.After(5 * time.Second):
		output.UserOut.Println("  ✗ Timeout waiting for connection from web container")
		if ncErr != nil {
			output.UserOut.Printf("    nc error: %v\n", ncErr)
			output.UserOut.Printf("    nc output: %s\n", ncOut)
		}
		hasIssues = true
	}

	return hasIssues
}

// testWSL2NATConnection tests connection using PowerShell listener on Windows side (for WSL2 NAT mode)
// In WSL2 NAT mode, the Docker container connects to host.docker.internal which routes to Windows.
// The IDE typically runs on Windows and listens on port 9003, so we test that path directly.
func testWSL2NATConnection(app *ddevapp.DdevApp) bool {
	hasIssues := false

	// PowerShell script to create a simple TCP listener on Windows port 9003
	// This simulates what an IDE would do - listen for incoming Xdebug connections
	psScript := `
$listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Any, 9003)
try {
    $listener.Start()
} catch {
    Write-Output "INUSE:$($_.Exception.Message)"
    exit 0
}
Write-Output "LISTENING"
$listener.Server.ReceiveTimeout = 15000
try {
    $client = $listener.AcceptTcpClient()
    $stream = $client.GetStream()
    $buffer = New-Object byte[] 1024
    $count = $stream.Read($buffer, 0, $buffer.Length)
    if ($count -gt 0) {
        $msg = [System.Text.Encoding]::ASCII.GetString($buffer, 0, $count).Trim()
        Write-Output "RECEIVED:$msg"
    }
    $client.Close()
} catch {
    Write-Output "ERROR:$($_.Exception.Message)"
}
$listener.Stop()
`

	// Start PowerShell listener on Windows side
	output.UserOut.Println("  Starting test listener on Windows port 9003...")
	cmd := exec.Command("powershell.exe",
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-Command", psScript,
	)

	// Capture stdout to monitor listener status
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		output.UserOut.Printf("  ✗ Failed to create stdout pipe: %v\n", err)
		return true
	}

	// Start the PowerShell listener
	err = cmd.Start()
	if err != nil {
		output.UserOut.Printf("  ✗ Failed to start PowerShell listener: %v\n", err)
		output.UserOut.Println("    Make sure PowerShell is available")
		return true
	}

	// Ensure we clean up the PowerShell process
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	// Wait for listener to be ready by reading the LISTENING output
	readyChan := make(chan bool, 1)
	outputLines := make(chan string, 10)
	go func() {
		buf := make([]byte, 4096)
		listenerReady := false
		for {
			n, err := stdout.Read(buf)
			if err != nil {
				return
			}
			lines := strings.Split(string(buf[:n]), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				// Handle Windows CRLF line endings
				line = strings.TrimSuffix(line, "\r")
				if line == "" {
					continue
				}
				if line == "LISTENING" && !listenerReady {
					listenerReady = true
					readyChan <- true
					continue // Don't put LISTENING in outputLines
				}
				// Handle port already in use on Windows
				if strings.HasPrefix(line, "INUSE:") {
					outputLines <- line
					return // Exit goroutine, listener won't start
				}
				outputLines <- line
			}
		}
	}()

	// Wait for listener to be ready (or detect port already in use)
	select {
	case <-readyChan:
		output.UserOut.Println("  ✓ Test listener started on Windows 0.0.0.0:9003")
	case line := <-outputLines:
		if strings.HasPrefix(line, "INUSE:") {
			output.UserOut.Println("  ✓ Port 9003 is already in use on Windows")
			output.UserOut.Println("    This is likely your IDE listening for Xdebug connections (which is good!)")
			output.UserOut.Println("    Skipping connection test since we cannot bind to the port.")
			return false // Not an issue - port is in use, presumably by IDE
		}
		output.UserOut.Printf("  ✗ Unexpected output from Windows listener: %s\n", line)
		return true
	case <-time.After(5 * time.Second):
		output.UserOut.Println("  ✗ Timeout waiting for Windows listener to start")
		return true
	}

	// Test connection from container
	output.UserOut.Println("  Testing connection from web container to Windows...")
	_, _, ncErr := app.Exec(&ddevapp.ExecOpts{
		Cmd: "bash -c 'echo test-from-container | nc -w 5 host.docker.internal 9003'",
	})

	// Check if PowerShell received the connection
	select {
	case line := <-outputLines:
		if strings.HasPrefix(line, "RECEIVED:") {
			message := strings.TrimPrefix(line, "RECEIVED:")
			output.UserOut.Println("  ✓ Successfully connected from web container to Windows")
			output.UserOut.Printf("  ✓ Received message: '%s'\n", message)
		} else if strings.HasPrefix(line, "ERROR:") {
			errMsg := strings.TrimPrefix(line, "ERROR:")
			output.UserOut.Printf("  ✗ PowerShell listener error: %s\n", errMsg)
			hasIssues = true
		} else {
			output.UserOut.Printf("  ✓ Received response from listener: %s\n", line)
		}
	case <-time.After(10 * time.Second):
		output.UserOut.Println("  ✗ Timeout waiting for connection from web container")
		output.UserOut.Println("    This may indicate firewall or networking issues in WSL2 NAT mode")
		output.UserOut.Println("    Check Windows Firewall settings for port 9003")
		if ncErr != nil {
			output.UserOut.Printf("    Container connection error: %v\n", ncErr)
		}
		hasIssues = true
	}

	return hasIssues
}

// isWindowsPortInUse checks if a port is in use on Windows from WSL2
// by using PowerShell to query Windows networking
func isWindowsPortInUse(port int) bool {
	// Use PowerShell to check if the port is listening on Windows
	psScript := `
$connections = Get-NetTCPConnection -LocalPort ` + fmt.Sprintf("%d", port) + ` -State Listen -ErrorAction SilentlyContinue
if ($connections) { Write-Output "INUSE" } else { Write-Output "FREE" }
`
	cmd := exec.Command("powershell.exe", "-NoProfile", "-Command", psScript)
	output, err := cmd.Output()
	if err != nil {
		// If PowerShell fails, assume port is free
		return false
	}
	return strings.TrimSpace(string(output)) == "INUSE"
}
