package cmd

import (
	"encoding/base64"
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

// issue represents a detected problem with fix instructions
type issue struct {
	problem string
	fix     string
}

// runXdebugDiagnose performs the diagnostic checks and outputs results
// Returns exit code: 0 if no issues, 1 if issues found
func runXdebugDiagnose() int {
	var issues []issue

	// Try to load app from current directory
	app, err := ddevapp.GetActiveApp("")
	if err != nil {
		util.Warning("Not in a DDEV project directory.")
		return 1
	}

	// Check if project is running
	status, _ := app.SiteStatus()
	if status != ddevapp.SiteRunning {
		output.UserOut.Printf("Starting project '%s' for diagnostics...\n", app.Name)
		if err := app.Start(); err != nil {
			util.Failed("Failed to start project: %v", err)
			return 1
		}
	}

	output.UserOut.Printf("Xdebug Diagnostics: %s\n", app.Name)
	output.UserOut.Println()

	isWSL2 := nodeps.IsWSL2()
	xdebugIDELocation := globalconfig.DdevGlobalConfig.XdebugIDELocation
	hostDockerInternal := dockerutil.GetHostDockerInternal()

	// Check 1: Port 9003 status
	var portInUse bool
	if isWSL2 && xdebugIDELocation != "wsl2" {
		portInUse = isWindowsPortInUse(9003)
		if portInUse {
			output.UserOut.Println("✓ Port 9003: IDE listening (Windows)")
		} else {
			output.UserOut.Println("○ Port 9003: No listener detected (Windows)")
		}
	} else {
		portInUse = netutil.IsPortActive("9003")
		if portInUse {
			output.UserOut.Println("✓ Port 9003: IDE listening")
		} else {
			output.UserOut.Println("○ Port 9003: No listener detected")
		}
	}

	// Check 2: WSL2 Mirrored Mode - hostAddressLoopback setting
	if isWSL2 && nodeps.IsWSL2MirroredMode() {
		if nodeps.IsWSL2HostAddressLoopbackEnabled() {
			output.UserOut.Println("✓ WSL2 mirrored mode: hostAddressLoopback enabled")
		} else {
			output.UserOut.Println("✗ WSL2 mirrored mode: hostAddressLoopback NOT enabled")
			issues = append(issues, issue{
				problem: "WSL2 mirrored mode requires hostAddressLoopback=true",
				fix:     "Add to C:\\Users\\<username>\\.wslconfig:\n     [experimental]\n     hostAddressLoopback=true\n   Then run: wsl --shutdown",
			})
		}
	}

	// Check 3: host.docker.internal
	output.UserOut.Printf("✓ host.docker.internal: %s\n", hostDockerInternal.IPAddress)

	// Check 4: xdebug_ide_location setting
	ideLocDisplay := "(default)"
	if xdebugIDELocation != "" {
		ideLocDisplay = "\"" + xdebugIDELocation + "\""
	}
	output.UserOut.Printf("○ xdebug_ide_location: %s\n", ideLocDisplay)

	// WSL2-specific configuration warning
	if isWSL2 && xdebugIDELocation == "" {
		output.UserOut.Println("  ⚠ If using VS Code with WSL extension, run:")
		output.UserOut.Println("    ddev config global --xdebug-ide-location=wsl2")
	}

	// Check 5: Connection test
	if !portInUse {
		output.UserOut.Print("○ Connection test: ")
		var connFailed bool
		if isWSL2 && xdebugIDELocation != "wsl2" {
			connFailed = testWSL2NATConnectionQuiet(app)
		} else {
			connFailed = testSimpleConnectionQuiet(app)
		}
		if connFailed {
			output.UserOut.Println("FAILED")
			issues = append(issues, issue{
				problem: "Container cannot connect to host on port 9003",
				fix:     "Check firewall settings - port 9003 must be accessible",
			})
		} else {
			output.UserOut.Println("OK")
		}
	} else {
		output.UserOut.Println("○ Connection test: Skipped (port in use)")
	}

	// Check 6: Xdebug status
	statusOut, _, _ := app.Exec(&ddevapp.ExecOpts{
		Cmd: "xdebug status",
	})
	xdebugEnabled := !strings.Contains(statusOut, "disabled")
	if xdebugEnabled {
		output.UserOut.Println("✓ Xdebug: Enabled")
	} else {
		output.UserOut.Println("○ Xdebug: Disabled (enable with: ddev xdebug on)")
	}

	// Check 7: PHP module test (only if xdebug was disabled)
	if !xdebugEnabled {
		_, _, err := app.Exec(&ddevapp.ExecOpts{Cmd: "xdebug on"})
		if err != nil {
			output.UserOut.Printf("✗ Xdebug enable: Failed (%v)\n", err)
			issues = append(issues, issue{
				problem: "Failed to enable Xdebug",
				fix:     "Check container logs with: ddev logs",
			})
		} else {
			phpOut, _, _ := app.Exec(&ddevapp.ExecOpts{Cmd: "php -m"})
			if strings.Contains(phpOut, "xdebug") {
				output.UserOut.Println("✓ PHP module: Xdebug loaded")
			} else {
				output.UserOut.Println("✗ PHP module: Xdebug NOT loaded")
				issues = append(issues, issue{
					problem: "Xdebug PHP module not loading",
					fix:     "Try: ddev restart",
				})
			}
			// Restore disabled state
			_, _, _ = app.Exec(&ddevapp.ExecOpts{Cmd: "xdebug off"})
		}
	}

	output.UserOut.Println()

	// Summary
	if len(issues) > 0 {
		output.UserOut.Printf("Found %d issue(s):\n", len(issues))
		output.UserOut.Println()
		for i, iss := range issues {
			output.UserOut.Printf("%d. %s\n", i+1, iss.problem)
			output.UserOut.Printf("   Fix: %s\n", iss.fix)
			output.UserOut.Println()
		}
		output.UserOut.Println("Docs: https://ddev.readthedocs.io/en/stable/users/debugging-profiling/step-debugging/")
		return 1
	}

	output.UserOut.Println("No issues detected. Ready to debug:")
	output.UserOut.Println("  1. ddev xdebug on")
	output.UserOut.Println("  2. Start IDE debug listener")
	output.UserOut.Println("  3. Set breakpoint and visit site")

	return 0
}

// testContainerToHostConnectivity tests if the container can connect to a host:port
// Returns true if connection succeeds, false otherwise
func testContainerToHostConnectivity(app *ddevapp.DdevApp, host string, port int) (bool, string) {
	// Use PHP fsockopen for reliable connectivity testing
	connectTestScript := fmt.Sprintf(`<?php
$sock = @fsockopen('%s', %d, $errno, $errstr, 3);
if (!$sock) {
    echo "FAILED: $errstr ($errno)\n";
    exit(1);
}
fclose($sock);
echo "SUCCESS\n";
?>`, host, port)

	connectTestB64 := base64.StdEncoding.EncodeToString([]byte(connectTestScript))
	connectCmd := fmt.Sprintf("echo %s | base64 -d | php", connectTestB64)

	connectOut, _, connectErr := app.Exec(&ddevapp.ExecOpts{
		Cmd: connectCmd,
	})

	if connectErr != nil || strings.Contains(connectOut, "FAILED:") {
		errMsg := "Connection failed"
		if strings.Contains(connectOut, "FAILED:") {
			errMsg = strings.TrimPrefix(connectOut, "FAILED: ")
			errMsg = strings.TrimSpace(errMsg)
		}
		return false, errMsg
	}

	return true, ""
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
		output.UserOut.Println("    This may indicate firewall or networking issues in WSL2")
		output.UserOut.Println("    Check Windows Firewall settings for port 9003")
		if nodeps.IsWSL2MirroredMode() && !nodeps.IsWSL2HostAddressLoopbackEnabled() {
			output.UserOut.Println()
			output.UserOut.Println("  ⚠ You are using WSL2 mirrored mode but hostAddressLoopback is not enabled!")
			output.UserOut.Println("    Add to your .wslconfig under [experimental]: hostAddressLoopback=true")
			output.UserOut.Println("    Then restart WSL with: wsl --shutdown")
		}
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

// testSimpleConnectionQuiet tests connection without verbose output
// Returns true if connection failed, false if successful
func testSimpleConnectionQuiet(app *ddevapp.DdevApp) bool {
	listener, err := net.Listen("tcp", "0.0.0.0:9003")
	if err != nil {
		return true
	}
	defer listener.Close()

	connChan := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		connChan <- conn
	}()

	_, _, _ = app.Exec(&ddevapp.ExecOpts{
		Cmd: "bash -c 'echo test | nc -w 2 host.docker.internal 9003'",
	})

	select {
	case conn := <-connChan:
		conn.Close()
		return false
	case <-time.After(5 * time.Second):
		return true
	}
}

// testWSL2NATConnectionQuiet tests WSL2 NAT connection without verbose output
// Returns true if connection failed, false if successful
func testWSL2NATConnectionQuiet(app *ddevapp.DdevApp) bool {
	psScript := `
$listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Any, 9003)
try { $listener.Start() } catch { Write-Output "INUSE"; exit 0 }
Write-Output "LISTENING"
$listener.Server.ReceiveTimeout = 10000
try {
    $client = $listener.AcceptTcpClient()
    Write-Output "RECEIVED"
    $client.Close()
} catch { Write-Output "ERROR" }
$listener.Stop()
`
	cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", psScript)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return true
	}
	if err := cmd.Start(); err != nil {
		return true
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	resultChan := make(chan string, 1)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if err != nil {
				return
			}
			output := strings.TrimSpace(string(buf[:n]))
			if strings.Contains(output, "INUSE") {
				resultChan <- "inuse"
				return
			}
			if strings.Contains(output, "LISTENING") {
				continue
			}
			if strings.Contains(output, "RECEIVED") {
				resultChan <- "ok"
				return
			}
			if strings.Contains(output, "ERROR") {
				resultChan <- "error"
				return
			}
		}
	}()

	// Wait for listener to be ready
	time.Sleep(500 * time.Millisecond)

	_, _, _ = app.Exec(&ddevapp.ExecOpts{
		Cmd: "bash -c 'echo test | nc -w 5 host.docker.internal 9003'",
	})

	select {
	case result := <-resultChan:
		return result != "ok" && result != "inuse"
	case <-time.After(10 * time.Second):
		return true
	}
}

// =============================================================================
// Interactive Mode Functions
// =============================================================================

// runInteractiveXdebugDiagnose runs the interactive guided diagnostic
func runInteractiveXdebugDiagnose() int {
	output.UserOut.Println("Interactive Xdebug Diagnostics")
	output.UserOut.Println()

	app, err := ddevapp.GetActiveApp("")
	if err != nil {
		util.Warning("Not in a DDEV project directory.")
		return 1
	}

	status, _ := app.SiteStatus()
	if status != ddevapp.SiteRunning {
		output.UserOut.Printf("Starting project '%s'...\n", app.Name)
		if err := app.Start(); err != nil {
			util.Failed("Failed to start project: %v", err)
			return 1
		}
	}

	// Step 1: Environment Detection
	output.UserOut.Println("[1/5] Environment")
	envType := detectAndDisplayEnvironment(app)
	_, _, _ = app.Exec(&ddevapp.ExecOpts{Cmd: "xdebug off"})
	output.UserOut.Println()

	// Step 2: Connectivity test
	output.UserOut.Println("[2/5] Connectivity Test")
	output.UserOut.Println("Stop your IDE's debug listener temporarily for this test.")
	if !util.Confirm("Press Enter when ready") {
		return 1
	}
	connectivityOK := runConnectivityTest(app, envType)
	if !connectivityOK {
		output.UserOut.Println("✗ Network connectivity failed")
		output.UserOut.Println("  Check firewall settings for port 9003")
		return 1
	}
	output.UserOut.Println()

	// Step 3: IDE Information
	output.UserOut.Println("[3/5] IDE Setup")
	ideType, ideLocation := promptIDEInfo(envType)
	output.UserOut.Println()

	// Step 3.5: Validate xdebug_ide_location for WSL2 scenarios
	needsWSL2Setting := false
	var setupDescription string
	if envType == "wsl2-nat" || envType == "wsl2-mirrored" {
		switch ideLocation {
		case "windows":
			if ideType == "vscode" {
				needsWSL2Setting = true
				setupDescription = "VS Code with WSL extension"
			}
		case "gateway-wsl2":
			needsWSL2Setting = true
			setupDescription = "PhpStorm with Gateway"
		case "wslg", "wsl2", "container":
			needsWSL2Setting = true
			setupDescription = "IDE in WSL2/container"
		}
	}

	if needsWSL2Setting {
		xdebugIDELocation := globalconfig.DdevGlobalConfig.XdebugIDELocation
		if xdebugIDELocation != "wsl2" {
			output.UserOut.Printf("✗ %s requires xdebug_ide_location=wsl2\n", setupDescription)
			if util.Confirm("Set it now?") {
				globalconfig.DdevGlobalConfig.XdebugIDELocation = "wsl2"
				if err := globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig); err != nil {
					output.UserOut.Printf("Failed: %v\n", err)
					output.UserOut.Println("Run: ddev config global --xdebug-ide-location=wsl2")
				} else {
					output.UserOut.Println("✓ Set to 'wsl2', restarting...")
					_ = app.Restart()
				}
			} else {
				output.UserOut.Println("Run: ddev config global --xdebug-ide-location=wsl2")
			}
			output.UserOut.Println()
		}
	}

	// Step 4: Enable listening
	output.UserOut.Println("[4/5] Start IDE Listener")
	promptEnableListening(app.Name, ideType, ideLocation, envType)
	output.UserOut.Println()

	// Step 5: Protocol test
	output.UserOut.Println("[5/5] Protocol Test")
	protocolOK := testDBGpProtocol(app, ideLocation, envType)
	output.UserOut.Println()

	// Summary
	if connectivityOK && protocolOK {
		output.UserOut.Println("✓ All tests passed!")
		output.UserOut.Println("  1. ddev xdebug on")
		output.UserOut.Println("  2. Set breakpoint and visit site")
		return 0
	}

	output.UserOut.Println("✗ Some tests failed. See output above.")
	return 1
}

// detectAndDisplayEnvironment detects and displays the current environment
// Returns the environment type string for later use
func detectAndDisplayEnvironment(app *ddevapp.DdevApp) string {
	output.UserOut.Printf("Project: %s\n", app.Name)

	var envType string
	if nodeps.IsWSL2() {
		if nodeps.IsWSL2MirroredMode() {
			envType = "wsl2-mirrored"
			output.UserOut.Print("Platform: WSL2 (mirrored) ")
			if nodeps.IsWSL2HostAddressLoopbackEnabled() {
				output.UserOut.Println("✓")
			} else {
				output.UserOut.Println("✗ hostAddressLoopback not set")
			}
		} else {
			envType = "wsl2-nat"
			output.UserOut.Println("Platform: WSL2 (NAT)")
		}
	} else if runtime.GOOS == "darwin" {
		envType = "macos"
		output.UserOut.Println("Platform: macOS")
	} else if runtime.GOOS == "windows" {
		envType = "windows"
		output.UserOut.Println("Platform: Windows")
	} else {
		envType = "linux"
		output.UserOut.Println("Platform: Linux")
	}

	hostDockerInternal := dockerutil.GetHostDockerInternal()
	output.UserOut.Printf("host.docker.internal: %s\n", hostDockerInternal.IPAddress)

	return envType
}

// runConnectivityTest runs the appropriate connectivity test based on environment
func runConnectivityTest(app *ddevapp.DdevApp, envType string) bool {
	xdebugIDELocation := globalconfig.DdevGlobalConfig.XdebugIDELocation

	// Determine which connection test to use based on IDE location
	// If xdebug_ide_location=wsl2, the listener runs in WSL2, use simple connection test
	// Otherwise in WSL2, the listener runs on Windows, use WSL2 NAT connection test
	if (envType == "wsl2-nat" || envType == "wsl2-mirrored") && xdebugIDELocation != "wsl2" {
		output.UserOut.Println("  Testing connection to Windows host...")
		return !testWSL2NATConnection(app)
	}

	if envType == "wsl2-nat" || envType == "wsl2-mirrored" {
		output.UserOut.Println("  Testing connection to WSL2 host...")
	} else {
		output.UserOut.Println("  Testing connection to host...")
	}
	return !testSimpleConnection(app)
}

// promptIDEInfo asks the user about their IDE setup
func promptIDEInfo(envType string) (ideType string, ideLocation string) {
	// Ask which IDE
	idePrompt := promptui.Select{
		Label: "Which IDE are you using",
		Items: []string{"PhpStorm / IntelliJ", "VS Code", "Other"},
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
	var ideName string
	switch ideType {
	case "phpstorm":
		ideName = "PhpStorm"
	case "vscode":
		ideName = "VS Code"
	default:
		ideName = "your IDE"
	}

	if envType == "wsl2-nat" || envType == "wsl2-mirrored" {
		if ideType == "vscode" {
			locationItems = []string{
				"VS Code on Windows using WSL extension (recommended)",
				"VS Code on Windows with project on Windows filesystem",
				"VS Code running in WSLg",
				"VS Code using Remote Containers",
				"Remote/Other",
			}
		} else if ideType == "phpstorm" {
			locationItems = []string{
				"PhpStorm on Windows with project on WSL2 distro (recommended)",
				"PhpStorm on Windows with project on Windows filesystem",
				"PhpStorm with JetBrains Gateway WSL2 backend",
				"PhpStorm running in WSL2/WSLg",
				"Remote/Other",
			}
		} else {
			locationItems = []string{
				fmt.Sprintf("%s installed on Windows (recommended for WSL2)", ideName),
				fmt.Sprintf("%s running in WSL2/WSLg", ideName),
				"Remote/Other",
			}
		}
	} else {
		locationItems = []string{
			"Same machine as DDEV",
			"Remote/Other",
		}
	}

	locationPrompt := promptui.Select{
		Label: fmt.Sprintf("Where is %s installed and running", ideName),
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
		if envType == "wsl2-nat" || envType == "wsl2-mirrored" {
			if ideType == "vscode" {
				// VS Code has 5 options
				switch idx {
				case 0:
					ideLocation = "windows" // VS Code on Windows using WSL extension
				case 1:
					ideLocation = "windows-native" // VS Code on Windows with project on Windows filesystem
				case 2:
					ideLocation = "wslg" // VS Code running in WSLg
				case 3:
					ideLocation = "container" // VS Code using Remote Containers
				default:
					ideLocation = "remote"
				}
			} else if ideType == "phpstorm" {
				// PhpStorm has 5 options
				switch idx {
				case 0:
					ideLocation = "windows" // PhpStorm on Windows with project on WSL2 distro - recommended
				case 1:
					ideLocation = "windows-native" // PhpStorm on Windows with project on Windows filesystem
				case 2:
					ideLocation = "gateway-wsl2" // PhpStorm with JetBrains Gateway WSL2 backend
				case 3:
					ideLocation = "wsl2" // PhpStorm running in WSL2/WSLg
				default:
					ideLocation = "remote"
				}
			} else {
				// Other IDEs have 3 options
				switch idx {
				case 0:
					ideLocation = "windows"
				case 1:
					ideLocation = "wsl2"
				default:
					ideLocation = "remote"
				}
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
func promptEnableListening(projectName string, ideType string, ideLocation string, envType string) {
	switch ideType {
	case "phpstorm":
		output.UserOut.Println("PhpStorm: Run -> Start Listening for PHP Debug Connections")
		output.UserOut.Println("Verify: Settings -> PHP -> Debug -> Port 9003, Accept external connections")
	case "vscode":
		isWSL2 := envType == "wsl2-nat" || envType == "wsl2-mirrored"
		launchOK := checkVSCodeLaunchJSON(".vscode/launch.json")

		if launchOK {
			output.UserOut.Println("✓ .vscode/launch.json found")
		} else {
			output.UserOut.Println("Create .vscode/launch.json:")
			output.UserOut.Println(`  {"version":"0.2.0","configurations":[{"name":"Xdebug","type":"php",`)
			output.UserOut.Println(`   "request":"launch","port":9003,"hostname":"0.0.0.0",`)
			output.UserOut.Println(`   "pathMappings":{"/var/www/html":"${workspaceFolder}"}}]}`)
		}
		if isWSL2 && ideLocation == "windows" {
			output.UserOut.Println("Note: Install PHP Debug extension IN WSL, not Windows")
		}
		output.UserOut.Println("Press F5 to start debugging")
	default:
		output.UserOut.Println("Configure IDE: listen on port 9003, map /var/www/html to project root")
	}

	output.UserOut.Println()
	util.Confirm("Press Enter when IDE is listening")
}

// checkVSCodeLaunchJSON checks if .vscode/launch.json exists and has proper Xdebug configuration
func checkVSCodeLaunchJSON(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	content := string(data)
	// Check for key requirements
	hasPort9003 := strings.Contains(content, `"port": 9003`) || strings.Contains(content, `"port":9003`)
	hasHostname := strings.Contains(content, `"hostname"`) && strings.Contains(content, `"0.0.0.0"`)
	hasPathMapping := strings.Contains(content, `"/var/www/html"`)

	return hasPort9003 && hasHostname && hasPathMapping
}

// testDBGpProtocol tests the DBGp protocol by connecting to the IDE from inside the web container
func testDBGpProtocol(app *ddevapp.DdevApp, ideLocation string, envType string) bool {
	targetHost := "host.docker.internal"
	if ideLocation == "remote" {
		output.UserOut.Println("Skipping protocol test for remote IDE")
		return true
	}

	connected, errMsg := testContainerToHostConnectivity(app, targetHost, 9003)
	if !connected {
		output.UserOut.Printf("✗ Cannot connect to port 9003: %s\n", errMsg)
		output.UserOut.Println("  Ensure IDE listener is started on port 9003")
		return false
	}
	output.UserOut.Println("✓ Connected to port 9003")

	// Create a PHP script to test the DBGp protocol using base64 encoding to avoid quoting issues
	// PHP provides better control over socket communication than nc
	phpScript := `<?php
$sock = @fsockopen('HOST_PLACEHOLDER', 9003, $errno, $errstr, 5);
if (!$sock) {
    echo "ERROR: $errstr ($errno)\n";
    exit(1);
}

$initXML = base64_decode('PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0iVVRGLTgiPz4KPGluaXQgeG1sbnM9InVybjpkZWJ1Z2dlcl9wcm90b2NvbF92MSIKICAgICAgeG1sbnM6eGRlYnVnPSJodHRwczovL3hkZWJ1Zy5vcmcvZGJncC94ZGVidWciCiAgICAgIGZpbGV1cmk9ImZpbGU6Ly8vdmFyL3d3dy9odG1sL2luZGV4LnBocCIKICAgICAgbGFuZ3VhZ2U9IlBIUCIKICAgICAgcHJvdG9jb2xfdmVyc2lvbj0iMS4wIgogICAgICBhcHBpZD0iZGRldi10ZXN0IgogICAgICBpZGVrZXk9IlBIUFNUT1JNIj4KICA8ZW5naW5lIHZlcnNpb249IjMuMy4wIj48IVtDREFUQVtYZGVidWddXT48L2VuZ2luZT4KPC9pbml0Pg==');

$packet = strlen($initXML) . "\0" . $initXML . "\0";
fwrite($sock, $packet);
fflush($sock);

stream_set_timeout($sock, 3);
$response = fread($sock, 8192);

if ($response) {
    echo "SUCCESS: " . strlen($response) . " bytes\n";
    echo $response;
} else {
    $info = stream_get_meta_data($sock);
    if ($info['timed_out']) {
        echo "TIMEOUT\n";
    } else {
        echo "NO_RESPONSE\n";
    }
}

fclose($sock);
?>`
	phpScript = strings.Replace(phpScript, "HOST_PLACEHOLDER", targetHost, 1)

	// Use base64 to avoid all shell interpretation issues
	phpScriptB64 := base64.StdEncoding.EncodeToString([]byte(phpScript))
	cmd := fmt.Sprintf("echo %s | base64 -d > /tmp/ddev_xdebug_test.php && php /tmp/ddev_xdebug_test.php && rm -f /tmp/ddev_xdebug_test.php", phpScriptB64)

	testOut, _, execErr := app.Exec(&ddevapp.ExecOpts{Cmd: cmd})

	if execErr != nil || strings.Contains(testOut, "ERROR:") {
		output.UserOut.Println("✗ Protocol test failed")
		output.UserOut.Println("  Try: ddev xdebug on, then set a breakpoint")
		return false
	}

	if strings.Contains(testOut, "SUCCESS:") {
		output.UserOut.Println("✓ IDE responding to DBGp protocol")
		return true
	}

	if strings.Contains(testOut, "TIMEOUT") || strings.Contains(testOut, "NO_RESPONSE") {
		output.UserOut.Println("⚠ IDE accepted connection but no response")
		output.UserOut.Println("  Some IDEs only respond when Xdebug is enabled")
		output.UserOut.Println("  Try: ddev xdebug on, then set a breakpoint")
		return false
	}

	return false
}
