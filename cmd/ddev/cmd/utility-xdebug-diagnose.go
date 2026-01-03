package cmd

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/netutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var interactiveFlag bool

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

Use --interactive for a guided step-by-step diagnostic that tests your actual IDE.

Use this command when experiencing issues with Xdebug step debugging.`,
	Example: `ddev utility xdebug-diagnose
ddev utility xdebug-diagnose --interactive
ddev ut xdebug-diagnose -i`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 0 {
			util.Failed("This command takes no additional arguments")
		}

		var exitCode int
		if interactiveFlag {
			if !globalconfig.IsInteractive() {
				util.Warning("Interactive mode requested but DDEV_NONINTERACTIVE is set.")
				util.Warning("Running standard diagnostics instead.")
				exitCode = runXdebugDiagnose()
			} else {
				exitCode = runInteractiveXdebugDiagnose()
			}
		} else {
			exitCode = runXdebugDiagnose()
		}
		os.Exit(exitCode)
	},
}

func init() {
	DebugCmd.AddCommand(XdebugDiagnoseCmd)
	XdebugDiagnoseCmd.Flags().BoolVarP(&interactiveFlag, "interactive", "i", false,
		"Run interactive guided diagnostics with step-by-step prompts")
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
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
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
	psOutput, err := cmd.Output()
	if err != nil {
		// If PowerShell fails, assume port is free
		return false
	}
	return strings.TrimSpace(string(psOutput)) == "INUSE"
}

// =============================================================================
// Interactive Mode Functions
// =============================================================================

// runInteractiveXdebugDiagnose runs the interactive guided diagnostic
func runInteractiveXdebugDiagnose() int {
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Interactive Xdebug Diagnostics")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println()
	output.UserOut.Println("This wizard will guide you through testing your Xdebug setup.")
	output.UserOut.Println()

	// Load the app
	app, err := ddevapp.GetActiveApp("")
	if err != nil {
		util.Warning("Not in a DDEV project directory.")
		util.Warning("Please run this command from within a DDEV project.")
		return 1
	}

	// Ensure project is running
	status, _ := app.SiteStatus()
	if status != ddevapp.SiteRunning {
		output.UserOut.Printf("Project '%s' is not running. Starting project...", app.Name)
		output.UserOut.Println()
		if err := app.Start(); err != nil {
			util.Failed("Failed to start project: %v", err)
			return 1
		}
	}

	// Step 1: Environment Detection
	output.UserOut.Println("Step 1: Environment Detection")
	output.UserOut.Println("─────────────────────────────────────────────────────────────")
	envType := detectAndDisplayEnvironment(app)
	output.UserOut.Println()

	// Step 2: Ask user to close IDE
	output.UserOut.Println("Step 2: Prepare for Connectivity Test")
	output.UserOut.Println("─────────────────────────────────────────────────────────────")
	output.UserOut.Println("We need to test if the network path from your container to your")
	output.UserOut.Println("host is working. To do this, we need to temporarily use port 9003.")
	output.UserOut.Println()
	if !util.Confirm("Please close or stop your IDE's debug listener, then press Enter to continue") {
		output.UserOut.Println("Cancelled by user.")
		return 1
	}
	output.UserOut.Println()

	// Step 3: Basic Connectivity Test
	output.UserOut.Println("Step 3: Network Connectivity Test")
	output.UserOut.Println("─────────────────────────────────────────────────────────────")
	connectivityOK := runConnectivityTest(app, envType)
	output.UserOut.Println()

	if !connectivityOK {
		output.UserOut.Println("  ✗ Network connectivity test failed.")
		output.UserOut.Println("    Please resolve the connectivity issues before continuing.")
		output.UserOut.Println()
		output.UserOut.Println("Common fixes:")
		output.UserOut.Println("  - Check firewall settings for port 9003")
		output.UserOut.Println("  - If using WSL2, ensure proper network configuration")
		output.UserOut.Println("  - See: https://ddev.readthedocs.io/en/stable/users/debugging-profiling/step-debugging/")
		return 1
	}

	// Step 4: Ask about IDE
	output.UserOut.Println("Step 4: IDE Information")
	output.UserOut.Println("─────────────────────────────────────────────────────────────")
	ideType, ideLocation := promptIDEInfo(envType)
	output.UserOut.Println()

	// Step 5: Guide user to enable listening
	output.UserOut.Println("Step 5: Enable Debug Listening")
	output.UserOut.Println("─────────────────────────────────────────────────────────────")
	promptEnableListening(app.Name, ideType)
	output.UserOut.Println()

	// Step 6: Test DBGp Protocol
	output.UserOut.Println("Step 6: IDE Protocol Test")
	output.UserOut.Println("─────────────────────────────────────────────────────────────")
	protocolOK := testDBGpProtocol(ideLocation, envType)
	output.UserOut.Println()

	// Summary
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Summary")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if connectivityOK && protocolOK {
		output.UserOut.Println("  ✓ All tests passed! Your Xdebug setup is working correctly.")
		output.UserOut.Println()
		output.UserOut.Println("Next steps:")
		output.UserOut.Println("  1. Enable Xdebug: ddev xdebug on")
		output.UserOut.Println("  2. Set a breakpoint in your code")
		output.UserOut.Println("  3. Visit your site in a browser")
		output.UserOut.Println()
		return 0
	}

	output.UserOut.Println("  ⚠ Some tests failed. See the output above for details.")
	return 1
}

// detectAndDisplayEnvironment detects and displays the current environment
// Returns the environment type string for later use
func detectAndDisplayEnvironment(app *ddevapp.DdevApp) string {
	output.UserOut.Printf("  Project: %s\n", app.Name)
	output.UserOut.Printf("  Project root: %s\n", app.AppRoot)

	var envType string

	// Detect environment
	if nodeps.IsWSL2() {
		if nodeps.IsWSL2MirroredMode() {
			envType = "wsl2-mirrored"
			output.UserOut.Println("  Environment: WSL2 (Mirrored networking mode)")
		} else {
			envType = "wsl2-nat"
			output.UserOut.Println("  Environment: WSL2 (NAT networking mode)")
			output.UserOut.Println("    Note: Your IDE likely runs on Windows, not in WSL2")
		}
	} else if runtime.GOOS == "darwin" {
		envType = "macos"
		output.UserOut.Println("  Environment: macOS")
	} else if runtime.GOOS == "windows" {
		envType = "windows"
		output.UserOut.Println("  Environment: Windows")
	} else {
		envType = "linux"
		output.UserOut.Println("  Environment: Linux")
	}

	// Show Docker info
	hostDockerInternal := dockerutil.GetHostDockerInternal()
	output.UserOut.Printf("  host.docker.internal: %s\n", hostDockerInternal.IPAddress)

	return envType
}

// runConnectivityTest runs the appropriate connectivity test based on environment
func runConnectivityTest(app *ddevapp.DdevApp, envType string) bool {
	if envType == "wsl2-nat" {
		output.UserOut.Println("  Testing connection to Windows host...")
		return !testWSL2NATConnection(app)
	}
	output.UserOut.Println("  Testing connection to host...")
	return !testSimpleConnection(app)
}

// promptIDEInfo asks the user about their IDE setup
func promptIDEInfo(envType string) (ideType string, ideLocation string) {
	// Ask which IDE
	idePrompt := promptui.Select{
		Label: "Which IDE are you using",
		Items: []string{"PHPStorm / IntelliJ", "VS Code", "Other"},
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}?",
			Active:   "▸ {{ . | cyan }}",
			Inactive: "  {{ . }}",
			Selected: "  IDE: {{ . | green }}",
		},
	}
	idx, _, err := idePrompt.Run()
	if err != nil {
		ideType = "other"
	} else {
		switch idx {
		case 0:
			ideType = "phpstorm"
		case 1:
			ideType = "vscode"
		default:
			ideType = "other"
		}
	}

	// Ask where IDE is running
	var locationItems []string
	if envType == "wsl2-nat" {
		locationItems = []string{
			"Windows (recommended for WSL2)",
			"Inside WSL2",
			"Remote/Other",
		}
	} else {
		locationItems = []string{
			"Same machine as DDEV",
			"Remote/Other",
		}
	}

	locationPrompt := promptui.Select{
		Label: "Where is your IDE running",
		Items: locationItems,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}?",
			Active:   "▸ {{ . | cyan }}",
			Inactive: "  {{ . }}",
			Selected: "  Location: {{ . | green }}",
		},
	}
	idx, _, err = locationPrompt.Run()
	if err != nil {
		ideLocation = "local"
	} else {
		if envType == "wsl2-nat" {
			switch idx {
			case 0:
				ideLocation = "windows"
			case 1:
				ideLocation = "wsl2"
			default:
				ideLocation = "remote"
			}
		} else {
			switch idx {
			case 0:
				ideLocation = "local"
			default:
				ideLocation = "remote"
			}
		}
	}

	return ideType, ideLocation
}

// promptEnableListening guides the user to enable debug listening in their IDE
func promptEnableListening(projectName string, ideType string) {
	output.UserOut.Printf("  Please open project '%s' in your IDE.\n", projectName)
	output.UserOut.Println()

	switch ideType {
	case "phpstorm":
		output.UserOut.Println("  PHPStorm setup:")
		output.UserOut.Println("    1. Go to Run -> Start Listening for PHP Debug Connections")
		output.UserOut.Println("       (or click the phone/bug icon in the toolbar)")
		output.UserOut.Println("    2. Ensure the icon shows a green bug or 'listening' state")
		output.UserOut.Println()
		output.UserOut.Println("  Settings to verify (File -> Settings -> PHP -> Debug):")
		output.UserOut.Println("    - Debug port: 9003")
		output.UserOut.Println("    - 'Can accept external connections' is checked")
	case "vscode":
		output.UserOut.Println("  VS Code setup:")
		output.UserOut.Println("    1. Install the 'PHP Debug' extension by Xdebug")
		output.UserOut.Println("    2. Create/update .vscode/launch.json with:")
		output.UserOut.Println(`       {
         "version": "0.2.0",
         "configurations": [{
           "name": "Listen for Xdebug",
           "type": "php",
           "request": "launch",
           "port": 9003,
           "pathMappings": {
             "/var/www/html": "${workspaceFolder}"
           }
         }]
       }`)
		output.UserOut.Println("    3. Press F5 or go to Run -> Start Debugging")
	default:
		output.UserOut.Println("  IDE setup:")
		output.UserOut.Println("    1. Configure your IDE to listen on port 9003")
		output.UserOut.Println("    2. Enable debug listening mode")
		output.UserOut.Println("    3. Ensure path mappings are set correctly:")
		output.UserOut.Println("       Container: /var/www/html -> Your project root")
	}

	output.UserOut.Println()
	util.Confirm("Press Enter when your IDE is listening for debug connections")
}

// testDBGpProtocol tests the DBGp protocol by connecting to the IDE and sending an init packet
func testDBGpProtocol(ideLocation string, envType string) bool {
	output.UserOut.Println("  Connecting to IDE on port 9003...")

	// Determine the target address based on IDE location and environment
	var targetAddr string
	switch ideLocation {
	case "windows":
		// For WSL2 NAT mode, connect to Windows host
		// Get the Windows IP from the default gateway
		targetAddr = getWindowsHostIP() + ":9003"
	case "wsl2":
		targetAddr = "127.0.0.1:9003"
	case "remote":
		output.UserOut.Println("  ℹ For remote IDE setups, please ensure port forwarding is configured.")
		output.UserOut.Println("    Skipping protocol test for remote IDE.")
		return true
	default:
		targetAddr = "127.0.0.1:9003"
	}

	output.UserOut.Printf("  Target: %s\n", targetAddr)

	// Connect to the IDE
	conn, err := net.DialTimeout("tcp", targetAddr, 5*time.Second)
	if err != nil {
		output.UserOut.Printf("  ✗ Cannot connect to IDE: %v\n", err)
		output.UserOut.Println("    Make sure your IDE is listening for debug connections.")
		return false
	}
	defer conn.Close()
	output.UserOut.Println("  ✓ Connected to IDE")

	// Send a simulated DBGp init packet
	// Format: length\0xml\0
	initXML := `<?xml version="1.0" encoding="UTF-8"?>
<init xmlns="urn:debugger_protocol_v1"
      xmlns:xdebug="https://xdebug.org/dbgp/xdebug"
      appid="ddev-test"
      idekey="DDEV_TEST"
      session=""
      thread="1"
      language="PHP"
      protocol_version="1.0"
      fileuri="file:///var/www/html/index.php">
  <engine version="3.0.0"><![CDATA[Xdebug]]></engine>
</init>`

	// DBGp format: length + null + xml + null
	packet := fmt.Sprintf("%d\x00%s\x00", len(initXML), initXML)

	output.UserOut.Println("  Sending DBGp init packet...")
	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err = conn.Write([]byte(packet))
	if err != nil {
		output.UserOut.Printf("  ✗ Failed to send init packet: %v\n", err)
		return false
	}
	output.UserOut.Println("  ✓ Init packet sent")

	// Wait for IDE response
	output.UserOut.Println("  Waiting for IDE response...")
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			output.UserOut.Println("  ✗ Timeout waiting for IDE response")
			output.UserOut.Println("    The IDE connected but did not respond to the init packet.")
			output.UserOut.Println("    This might indicate the IDE is not properly configured for Xdebug.")
			return false
		}
		output.UserOut.Printf("  ✗ Error reading IDE response: %v\n", err)
		return false
	}

	if n > 0 {
		// Parse the response - DBGp responses start with length + null byte
		response := string(buf[:n])
		output.UserOut.Println("  ✓ Received response from IDE!")

		// Look for common DBGp commands in the response
		if strings.Contains(response, "feature_get") ||
			strings.Contains(response, "feature_set") ||
			strings.Contains(response, "status") ||
			strings.Contains(response, "breakpoint") {
			output.UserOut.Println("  ✓ IDE responded with valid DBGp commands")
			output.UserOut.Println("  ✓ Your IDE is correctly configured for Xdebug!")
			return true
		}

		// Any response is actually good - it means the IDE is listening and responding
		output.UserOut.Println("  ✓ IDE is responding to Xdebug connections")
		util.Debug("DBGp response: %s", response)
		return true
	}

	output.UserOut.Println("  ⚠ IDE connected but sent empty response")
	return false
}

// getWindowsHostIP gets the Windows host IP from WSL2
func getWindowsHostIP() string {
	// In WSL2, the Windows host is typically accessible via the default gateway
	cmd := exec.Command("ip", "route", "show", "default")
	out, err := cmd.Output()
	if err != nil {
		return "127.0.0.1"
	}
	// Parse "default via 172.x.x.x dev eth0"
	fields := strings.Fields(string(out))
	for i, field := range fields {
		if field == "via" && i+1 < len(fields) {
			return fields[i+1]
		}
	}
	return "127.0.0.1"
}
