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

	isWSL2 := nodeps.IsWSL2()
	xdebugIDELocation := globalconfig.DdevGlobalConfig.XdebugIDELocation
	var portInUse bool

	// Determine where to check for the port based on IDE location
	// If xdebug_ide_location=wsl2, the IDE listener runs in WSL2
	// Otherwise in WSL2, the IDE typically runs on Windows
	if isWSL2 && xdebugIDELocation != "wsl2" {
		portInUse = isWindowsPortInUse(9003)
		if portInUse {
			output.UserOut.Println("  ✓ Port 9003 is in use on Windows")
			output.UserOut.Println("    This is likely your IDE listening for Xdebug connections (which is good!)")
			output.UserOut.Println()
			output.UserOut.Println("  To identify what's listening on port 9003, run in PowerShell:")
			output.UserOut.Println("    Get-NetTCPConnection -LocalPort 9003 | Select-Object OwningProcess")
		} else {
			output.UserOut.Println("  Port 9003 is not currently in use on Windows")
			output.UserOut.Println("  When you're ready to debug, start your IDE's debug listener on port 9003.")
		}
	} else {
		portInUse = netutil.IsPortActive("9003")
		if portInUse {
			output.UserOut.Println("  ✓ Port 9003 is in use on the host")
			output.UserOut.Println("    This is likely your IDE listening for Xdebug connections (which is good!)")
			output.UserOut.Println()
			output.UserOut.Println("  To identify what's listening on port 9003, run:")
			if isWSL2 {
				output.UserOut.Println("    • WSL2: sudo lsof -i :9003 -sTCP:LISTEN")
			} else {
				output.UserOut.Println("    • Linux/macOS: sudo lsof -i :9003 -sTCP:LISTEN")
				output.UserOut.Println("    • Windows: netstat -ano | findstr :9003")
			}
		} else {
			output.UserOut.Println("  Port 9003 is not currently in use on the host")
			output.UserOut.Println("  When you're ready to debug, start your IDE's debug listener on port 9003.")
		}
	}
	output.UserOut.Println()

	// Check 2: WSL2 Mirrored Mode - hostAddressLoopback setting
	if isWSL2 && nodeps.IsWSL2MirroredMode() {
		output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		output.UserOut.Println("WSL2 Mirrored Mode Configuration")
		output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		if nodeps.IsWSL2HostAddressLoopbackEnabled() {
			output.UserOut.Println("  ✓ hostAddressLoopback=true is enabled in .wslconfig")
			output.UserOut.Println("    This is required for Xdebug in WSL2 mirrored mode.")
		} else {
			output.UserOut.Println("  ✗ hostAddressLoopback=true is NOT set in .wslconfig")
			output.UserOut.Println("    This setting is REQUIRED for Xdebug to work in WSL2 mirrored mode.")
			output.UserOut.Println()
			output.UserOut.Println("  To fix this, add the following to your Windows .wslconfig file")
			output.UserOut.Println("  (located at C:\\Users\\<username>\\.wslconfig):")
			output.UserOut.Println()
			output.UserOut.Println("    [experimental]")
			output.UserOut.Println("    hostAddressLoopback=true")
			output.UserOut.Println()
			output.UserOut.Println("  Then restart WSL with: wsl --shutdown")
			output.UserOut.Println()
			output.UserOut.Println("  See: https://ddev.readthedocs.io/en/stable/users/debugging-profiling/step-debugging/")
			hasIssues = true
		}
		output.UserOut.Println()
	}

	// Check 3: Get host.docker.internal information
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("host.docker.internal Configuration")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	hostDockerInternal := dockerutil.GetHostDockerInternal()
	output.UserOut.Printf("  IP address: %s\n", hostDockerInternal.IPAddress)
	output.UserOut.Printf("  Derivation: %s\n", hostDockerInternal.Message)
	output.UserOut.Println()

	// Check 4: Check xdebug_ide_location setting
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Global Configuration")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	output.UserOut.Printf("  xdebug_ide_location: %s\n", func() string {
		if xdebugIDELocation == "" {
			return "(empty/default)"
		}
		return "\"" + xdebugIDELocation + "\""
	}())

	// Special check for WSL2 + VS Code with WSL extension
	if isWSL2 {
		output.UserOut.Println()
		output.UserOut.Println("  WSL2 + VS Code Setup:")
		output.UserOut.Println()
		output.UserOut.Println("  If you are using VS Code on Windows with the WSL extension:")
		output.UserOut.Println("    • VS Code file explorer should show [WSL: <distro>] (e.g., [WSL: Ubuntu])")
		output.UserOut.Println("    • PHP Debug extension must be installed IN WSL (not Windows)")
		output.UserOut.Println("    • xdebug_ide_location MUST be set to 'wsl2'")
		output.UserOut.Println()

		if xdebugIDELocation == "wsl2" {
			output.UserOut.Println("  ✓ xdebug_ide_location is correctly set to 'wsl2'")
		} else if xdebugIDELocation == "" {
			output.UserOut.Println("  ✗ xdebug_ide_location is not set")
			output.UserOut.Println("    For VS Code with WSL extension, you MUST run:")
			output.UserOut.Println("      ddev config global --xdebug-ide-location=wsl2")
			hasIssues = true
		} else {
			output.UserOut.Printf("  ⚠ xdebug_ide_location is set to '%s'\n", xdebugIDELocation)
			output.UserOut.Println("    For VS Code with WSL extension, it should be 'wsl2'")
			output.UserOut.Println("    If using VS Code with WSL extension, run:")
			output.UserOut.Println("      ddev config global --xdebug-ide-location=wsl2")
		}
		output.UserOut.Println()
		output.UserOut.Println("  If your IDE runs in WSLg, WSL2, or a container:")
		if xdebugIDELocation == "wsl2" {
			output.UserOut.Println("  ✓ xdebug_ide_location is correctly set to 'wsl2'")
		} else {
			output.UserOut.Println("  ✗ xdebug_ide_location should be set to 'wsl2'")
			output.UserOut.Println("    Run: ddev config global --xdebug-ide-location=wsl2")
		}
		output.UserOut.Println()
		output.UserOut.Println("  If your IDE runs on Windows (e.g., PHPStorm on Windows):")
		if xdebugIDELocation == "" {
			output.UserOut.Println("  ✓ xdebug_ide_location is correctly set to default (empty)")
		} else if xdebugIDELocation != "wsl2" {
			output.UserOut.Println("  ✓ xdebug_ide_location is set for a special configuration")
		} else {
			output.UserOut.Println("  ⚠ xdebug_ide_location should be empty for IDEs on Windows")
			output.UserOut.Println("    Run: ddev config global --xdebug-ide-location=\"\"")
		}
	} else {
		// Non-WSL2 environments
		if xdebugIDELocation == "" {
			output.UserOut.Println("  ✓ This is correct for most users.")
		} else {
			output.UserOut.Println()
			output.UserOut.Printf("  ⚠ xdebug_ide_location is set to: '%s'\n", xdebugIDELocation)
			output.UserOut.Println("    This should only be set for special cases (container IDE, remote IDE, etc.)")
			output.UserOut.Println("    If you're having issues, try: ddev config global --xdebug-ide-location=\"\"")
		}
	}
	output.UserOut.Println()

	// Check 5: Start a test listener and test connection
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Connection Test")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Only run connection test if port 9003 is not already in use
	if !portInUse {
		// Determine which connection test to use based on IDE location
		// If xdebug_ide_location=wsl2, the listener runs in WSL2, use simple connection test
		// Otherwise in WSL2, the listener runs on Windows, use WSL2 NAT connection test
		if isWSL2 && xdebugIDELocation != "wsl2" {
			output.UserOut.Println("  Detected WSL2 - testing connection to Windows host...")
			hasIssues = testWSL2NATConnection(app) || hasIssues
		} else {
			if isWSL2 && xdebugIDELocation == "wsl2" {
				output.UserOut.Println("  Starting test listener in WSL2 on port 9003...")
			} else {
				output.UserOut.Println("  Starting test listener on host port 9003...")
			}
			hasIssues = testSimpleConnection(app) || hasIssues
		}
	} else {
		output.UserOut.Println("  ℹ Skipping connection test (port 9003 already in use)")
		output.UserOut.Println("    To test connectivity, stop your IDE's debug listener and run this command again.")
	}
	output.UserOut.Println()

	// Check 6: Check Xdebug status
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Xdebug Status")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	statusOut, _, _ := app.Exec(&ddevapp.ExecOpts{
		Cmd: "xdebug status",
	})
	output.UserOut.Printf("  Current status: %s\n", strings.TrimSpace(statusOut))

	if strings.Contains(statusOut, "disabled") {
		output.UserOut.Println("  Xdebug is currently disabled")
		output.UserOut.Println("  When you're ready to debug, enable it with: ddev xdebug on")
	}
	output.UserOut.Println()

	// Check 7: Test Xdebug with PHP if enabled, or enable temporarily
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

	// Turn off Xdebug to prevent it from interfering with the test
	output.UserOut.Println("  Ensuring Xdebug is disabled for testing...")
	_, _, _ = app.Exec(&ddevapp.ExecOpts{
		Cmd: "xdebug off",
	})
	output.UserOut.Println()

	// Step 2: Prepare for connectivity test
	output.UserOut.Println("Step 2: Prepare for Connectivity Test")
	output.UserOut.Println("─────────────────────────────────────────────────────────────")
	output.UserOut.Println("We need to test if the network path from your container to your")
	output.UserOut.Println("host is working. To do this, we need to temporarily use port 9003.")
	output.UserOut.Println()
	output.UserOut.Println("If your IDE is currently listening for Xdebug connections, please")
	output.UserOut.Println("stop it temporarily. You'll be asked to start it again in a moment.")
	output.UserOut.Println()
	if !util.Confirm("Press Enter when your IDE's debug listener is stopped") {
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

	// Step 4.5: Validate xdebug_ide_location for WSL2 scenarios that require it
	// This includes: VS Code with WSL extension, IDEs in WSLg, IDEs in containers
	needsWSL2Setting := false
	var setupDescription string

	if envType == "wsl2-nat" || envType == "wsl2-mirrored" {
		switch ideLocation {
		case "windows":
			// VS Code on Windows with WSL extension
			if ideType == "vscode" {
				needsWSL2Setting = true
				setupDescription = "VS Code on Windows with WSL extension"
			}
		case "wslg":
			// Any IDE running in WSLg (graphical Linux app in WSL2)
			needsWSL2Setting = true
			setupDescription = "IDE running in WSLg"
		case "wsl2":
			// PHPStorm or other IDE running directly in WSL2
			needsWSL2Setting = true
			setupDescription = "IDE running in WSL2"
		case "container":
			// VS Code Remote Containers or similar
			needsWSL2Setting = true
			setupDescription = "IDE running in container"
		}
	}

	if needsWSL2Setting {
		output.UserOut.Println("Step 4.5: Validate xdebug_ide_location Setting")
		output.UserOut.Println("─────────────────────────────────────────────────────────────")
		xdebugIDELocation := globalconfig.DdevGlobalConfig.XdebugIDELocation
		output.UserOut.Printf("  Current xdebug_ide_location: %s\n", func() string {
			if xdebugIDELocation == "" {
				return "(empty/default)"
			}
			return "\"" + xdebugIDELocation + "\""
		}())
		output.UserOut.Println()

		if xdebugIDELocation != "wsl2" {
			output.UserOut.Printf("  ✗ For %s, xdebug_ide_location MUST be 'wsl2'\n", setupDescription)
			output.UserOut.Println()
			output.UserOut.Println("  This setting tells DDEV where your IDE's debug listener is located.")
			output.UserOut.Println("  Without it, Xdebug connections will fail.")
			output.UserOut.Println()

			if util.Confirm("Would you like to set xdebug_ide_location=wsl2 now?") {
				globalconfig.DdevGlobalConfig.XdebugIDELocation = "wsl2"
				err := globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
				if err != nil {
					output.UserOut.Printf("  ✗ Failed to update global config: %v\n", err)
					output.UserOut.Println("  Please run manually: ddev config global --xdebug-ide-location=wsl2")
				} else {
					output.UserOut.Println("  ✓ xdebug_ide_location has been set to 'wsl2'")
					output.UserOut.Println()
					output.UserOut.Println("  Restarting project to apply changes...")
					if err := app.Restart(); err != nil {
						output.UserOut.Printf("  ⚠ Failed to restart project: %v\n", err)
						output.UserOut.Println("    Please restart manually: ddev restart")
					} else {
						output.UserOut.Println("  ✓ Project restarted successfully")
					}
				}
			} else {
				output.UserOut.Println("  Please run manually: ddev config global --xdebug-ide-location=wsl2")
				output.UserOut.Println("  Then restart your project: ddev restart")
			}
		} else {
			output.UserOut.Println("  ✓ xdebug_ide_location is correctly set to 'wsl2'")
		}
		output.UserOut.Println()
	}

	// Step 5: Guide user to enable listening
	output.UserOut.Println("Step 5: Enable Debug Listening")
	output.UserOut.Println("─────────────────────────────────────────────────────────────")
	promptEnableListening(app.Name, ideType, ideLocation, envType)
	output.UserOut.Println()

	// Step 6: Test DBGp Protocol
	output.UserOut.Println("Step 6: IDE Protocol Test")
	output.UserOut.Println("─────────────────────────────────────────────────────────────")
	protocolOK := testDBGpProtocol(app, ideLocation, envType)
	output.UserOut.Println()

	// Summary
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Summary")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if connectivityOK && protocolOK {
		output.UserOut.Println("  ✓ All tests passed! Your Xdebug setup is working correctly.")
		output.UserOut.Println()
		output.UserOut.Println("Next steps to start debugging:")
		output.UserOut.Println("  1. Enable Xdebug: ddev xdebug on")
		output.UserOut.Println("  2. Ensure your IDE is listening for debug connections")
		output.UserOut.Println("  3. Set a breakpoint in your code")
		output.UserOut.Println("  4. Visit your site in a browser")
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
			// Check for hostAddressLoopback setting
			if nodeps.IsWSL2HostAddressLoopbackEnabled() {
				output.UserOut.Println("  ✓ hostAddressLoopback=true is enabled")
			} else {
				output.UserOut.Println("  ✗ hostAddressLoopback=true is NOT set in .wslconfig")
				output.UserOut.Println("    This is required for Xdebug in mirrored mode!")
				output.UserOut.Println("    Add to .wslconfig under [experimental]: hostAddressLoopback=true")
			}
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
	var ideName string
	switch ideType {
	case "phpstorm":
		ideName = "PHPStorm"
	case "vscode":
		ideName = "VS Code"
	default:
		ideName = "your IDE"
	}

	if envType == "wsl2-nat" || envType == "wsl2-mirrored" {
		if ideType == "vscode" {
			locationItems = []string{
				"VS Code on Windows using WSL extension (recommended)",
				"VS Code running in WSLg",
				"VS Code using Remote Containers",
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
				// VS Code has 4 options: WSL extension, WSLg, Remote Containers, Remote/Other
				switch idx {
				case 0:
					ideLocation = "windows" // VS Code on Windows using WSL extension
				case 1:
					ideLocation = "wslg" // VS Code running in WSLg
				case 2:
					ideLocation = "container" // VS Code using Remote Containers
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
		// WSL2-specific VS Code instructions
		isWSL2 := envType == "wsl2-nat" || envType == "wsl2-mirrored"
		if isWSL2 && ideLocation == "windows" {
			output.UserOut.Println("  VS Code on Windows with WSL extension setup:")
			output.UserOut.Println()
			output.UserOut.Println("  IMPORTANT: This is the recommended WSL2 configuration!")
			output.UserOut.Println()
			output.UserOut.Println("  1. Install the 'WSL' extension in VS Code (if not already installed)")
			output.UserOut.Println("     Extension ID: ms-vscode-remote.remote-wsl")
			output.UserOut.Println()
			output.UserOut.Println("  2. Open your WSL2 project folder:")
			output.UserOut.Println("     • Press F1 in VS Code")
			output.UserOut.Println("     • Run command: 'WSL: Open Folder in WSL'")
			output.UserOut.Println("     • Navigate to your project directory")
			output.UserOut.Println()
			output.UserOut.Println("  3. Install 'PHP Debug' extension IN WSL (not Windows):")
			output.UserOut.Println("     • Open Extensions view (Ctrl+Shift+X)")
			output.UserOut.Println("     • Search for 'PHP Debug' by Xdebug")
			output.UserOut.Println("     • Click 'Install in WSL' (NOT 'Install')")
			output.UserOut.Println("     • You should see the extension listed under 'WSL: <distro>' section")
			output.UserOut.Println()
			output.UserOut.Println("  4. Configure launch.json (if needed):")

			// Check for existing launch.json
			launchPath := ".vscode/launch.json"
			launchOK := checkVSCodeLaunchJSON(launchPath)

			if launchOK {
				output.UserOut.Println("     ✓ Found compliant .vscode/launch.json")
			} else {
				output.UserOut.Println("     Create .vscode/launch.json with:")
				output.UserOut.Println(`       {
         "version": "0.2.0",
         "configurations": [{
           "name": "Listen for Xdebug",
           "type": "php",
           "request": "launch",
           "port": 9003,
           "hostname": "0.0.0.0",
           "pathMappings": {
             "/var/www/html": "${workspaceFolder}"
           }
         }]
       }`)
			}
			output.UserOut.Println()
			output.UserOut.Println("  5. Start debugging:")
			output.UserOut.Println("     Press F5 or go to Run -> Start Debugging")
		} else if isWSL2 && (ideLocation == "wslg" || ideLocation == "container") {
			output.UserOut.Println("  VS Code in WSLg/Container setup:")
			output.UserOut.Println()
			output.UserOut.Println("  ℹ  Note: The recommended setup is VS Code on Windows with WSL extension,")
			output.UserOut.Println("      but this configuration works if you prefer running VS Code in WSLg.")
			output.UserOut.Println()

			// Check for existing launch.json
			launchPath := ".vscode/launch.json"
			launchOK := checkVSCodeLaunchJSON(launchPath)

			if launchOK {
				output.UserOut.Println("  ✓ Found compliant .vscode/launch.json")
				output.UserOut.Println()
				output.UserOut.Println("  1. Ensure 'PHP Debug' extension by Xdebug is installed")
				output.UserOut.Println("  2. Press F5 or go to Run -> Start Debugging")
			} else {
				output.UserOut.Println("  1. Install 'PHP Debug' extension by Xdebug")
				output.UserOut.Println("  2. Create .vscode/launch.json with:")
				output.UserOut.Println(`       {
         "version": "0.2.0",
         "configurations": [{
           "name": "Listen for Xdebug",
           "type": "php",
           "request": "launch",
           "port": 9003,
           "hostname": "0.0.0.0",
           "pathMappings": {
             "/var/www/html": "${workspaceFolder}"
           }
         }]
       }`)
				output.UserOut.Println("  3. Press F5 or go to Run -> Start Debugging")
			}
		} else {
			// Non-WSL2 VS Code setup
			launchPath := ".vscode/launch.json"
			launchOK := checkVSCodeLaunchJSON(launchPath)

			if launchOK {
				output.UserOut.Println("  ✓ Found compliant .vscode/launch.json")
				output.UserOut.Println("  VS Code setup:")
				output.UserOut.Println("    1. Ensure 'PHP Debug' extension by Xdebug is installed")
				output.UserOut.Println("    2. Press F5 or go to Run -> Start Debugging")
			} else {
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
           "hostname": "0.0.0.0",
           "pathMappings": {
             "/var/www/html": "${workspaceFolder}"
           }
         }]
       }`)
				output.UserOut.Println("    3. Press F5 or go to Run -> Start Debugging")
			}
		}
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
	// Determine the target host based on IDE location and environment
	// The connection must be made from inside the web container
	var targetHost string
	switch ideLocation {
	case "windows", "wsl2", "wslg", "container", "local", "macos", "linux":
		// From inside container, always use host.docker.internal to reach host
		targetHost = "host.docker.internal"
	case "remote":
		output.UserOut.Println("  ℹ For remote IDE setups, please ensure port forwarding is configured.")
		output.UserOut.Println("    Skipping protocol test for remote IDE.")
		return true
	default:
		targetHost = "host.docker.internal"
	}

	// Step 1: Simple connectivity check first
	output.UserOut.Printf("  Testing basic connectivity to %s:9003...\n", targetHost)

	connected, errMsg := testContainerToHostConnectivity(app, targetHost, 9003)
	if !connected {
		output.UserOut.Println("  ✗ Cannot connect to port 9003")
		output.UserOut.Println("    Your IDE is not listening on port 9003.")
		output.UserOut.Println()
		output.UserOut.Printf("    Error: %s\n", errMsg)
		output.UserOut.Println()
		output.UserOut.Println("  Please ensure:")
		output.UserOut.Println("    • Your IDE's debug listener is started")
		output.UserOut.Println("    • The listener is configured for port 9003")
		output.UserOut.Println("    • No firewall is blocking the connection")
		return false
	}

	output.UserOut.Println("  ✓ Successfully connected to port 9003")
	output.UserOut.Println()

	// Step 2: Now test the DBGp protocol
	output.UserOut.Println("  Testing DBGp protocol communication...")

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

	output.UserOut.Println("  Sending DBGp init packet to IDE...")
	testOut, testErr, execErr := app.Exec(&ddevapp.ExecOpts{
		Cmd: cmd,
	})

	// Handle exec errors gracefully (timeout, command failure, etc.)
	if execErr != nil {
		// Don't show the ugly raw error, provide a friendly message
		output.UserOut.Println("  ✗ Failed to complete protocol test")
		output.UserOut.Println()
		output.UserOut.Println("  This could mean:")
		output.UserOut.Println("    • Your IDE is listening but not responding to Xdebug packets")
		output.UserOut.Println("    • The IDE's debug configuration is incorrect")
		output.UserOut.Println("    • A network issue is preventing communication")
		output.UserOut.Println()
		output.UserOut.Println("  Try enabling Xdebug and testing with a real breakpoint:")
		output.UserOut.Println("    ddev xdebug on")
		return false
	}

	// Check for connection errors from PHP script
	if strings.Contains(testOut, "ERROR:") {
		errMsg := strings.TrimPrefix(testOut, "ERROR: ")
		errMsg = strings.TrimSpace(errMsg)
		output.UserOut.Println("  ✗ Connection failed")
		if strings.Contains(errMsg, "Connection refused") {
			output.UserOut.Println("    Your IDE is not listening on port 9003.")
			output.UserOut.Println("    Please start your IDE's debug listener and try again.")
		} else {
			output.UserOut.Printf("    Error: %s\n", errMsg)
		}
		return false
	}

	// Check stderr for errors
	if testErr != "" && !strings.Contains(testErr, "Deprecated") {
		output.UserOut.Printf("  ⚠ Warning: %s\n", testErr)
	}

	output.UserOut.Println("  ✓ Successfully sent DBGp init packet to IDE")

	// Check for IDE response
	if strings.Contains(testOut, "SUCCESS:") {
		// Extract response data (everything after the SUCCESS line)
		lines := strings.SplitN(testOut, "\n", 2)
		if len(lines) > 1 {
			response := lines[1]
			// Parse the byte count
			if strings.HasPrefix(lines[0], "SUCCESS: ") {
				byteCountStr := strings.TrimPrefix(lines[0], "SUCCESS: ")
				byteCountStr = strings.TrimSuffix(byteCountStr, " bytes")
				output.UserOut.Printf("  ✓ IDE responded with %s bytes\n", byteCountStr)
			}

			// Parse and display the DBGp commands
			output.UserOut.Println("    IDE commands received:")

			// Split by common command keywords to separate multiple commands
			// The response contains space-separated commands like: eval -i 1 -- base64 feature_set -i 2 -n name -v val
			commands := []string{}
			if strings.Contains(response, "eval") {
				commands = append(commands, "eval")
			}
			if strings.Contains(response, "feature_set") {
				commands = append(commands, "feature_set")
			}
			if strings.Contains(response, "feature_get") {
				commands = append(commands, "feature_get")
			}
			if strings.Contains(response, "status") {
				commands = append(commands, "status")
			}
			if strings.Contains(response, "breakpoint") {
				commands = append(commands, "breakpoint")
			}
			if strings.Contains(response, "run") {
				commands = append(commands, "run")
			}
			if strings.Contains(response, "step") {
				commands = append(commands, "step")
			}

			if len(commands) > 0 {
				for _, cmd := range commands {
					output.UserOut.Printf("      • %s\n", cmd)
				}
			} else {
				output.UserOut.Println("      • (unknown commands)")
			}

			// Look for common DBGp commands in the response
			if strings.Contains(response, "feature_get") ||
				strings.Contains(response, "feature_set") ||
				strings.Contains(response, "eval") ||
				strings.Contains(response, "status") ||
				strings.Contains(response, "breakpoint") ||
				strings.Contains(response, "run") ||
				strings.Contains(response, "step") {
				output.UserOut.Println("  ✓ IDE responded with valid DBGp commands")
				output.UserOut.Println("  ✓ Your IDE is correctly configured for Xdebug!")
			} else {
				output.UserOut.Println("  ✓ IDE is responding to Xdebug connections")
			}
			return true
		}
	}

	// Handle timeout or no response
	if strings.Contains(testOut, "TIMEOUT") {
		output.UserOut.Println("  ⚠ IDE connection timed out waiting for response")
		output.UserOut.Println("    The IDE accepted the connection but did not send commands.")
		output.UserOut.Println("    This could indicate:")
		output.UserOut.Println("    - Your IDE is listening but may not be properly configured for Xdebug")
		output.UserOut.Println("    - Some IDEs don't respond until Xdebug is actually enabled")
		output.UserOut.Println()
		output.UserOut.Println("  Try enabling Xdebug and setting a breakpoint to test fully:")
		output.UserOut.Println("    ddev xdebug on")
		return false
	}

	if strings.Contains(testOut, "NO_RESPONSE") {
		output.UserOut.Println("  ⚠ IDE accepted connection but did not send any response")
		output.UserOut.Println("    This could indicate:")
		output.UserOut.Println("    - Your IDE is listening but may not be properly configured for Xdebug")
		output.UserOut.Println("    - Some IDEs don't respond until Xdebug is actually enabled")
		output.UserOut.Println()
		output.UserOut.Println("  Try enabling Xdebug and setting a breakpoint to test fully:")
		output.UserOut.Println("    ddev xdebug on")
		return false
	}

	// Unexpected output
	output.UserOut.Printf("  ⚠ Unexpected test output: %s\n", testOut)
	return false
}
