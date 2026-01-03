package cmd

import (
	"net"
	"os"
	"strings"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/netutil"
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

	if netutil.IsPortActive("9003") {
		output.UserOut.Println("  ⚠ Port 9003 is already in use on the host")
		output.UserOut.Println("    This is likely your IDE listening for Xdebug connections (which is good!)")
		output.UserOut.Println("    Or it could be another process interfering with Xdebug.")
		portInUseWarning := true
		// We don't fail here because IDE listening is expected behavior
		_ = portInUseWarning
	} else {
		output.UserOut.Println("  ℹ Port 9003 is not currently in use on the host")
		output.UserOut.Println("    Your IDE should be listening on this port for Xdebug to work.")
		hasIssues = true
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

	// Check 3: Test ping to host.docker.internal from container
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Network Connectivity Test")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Test ping
	pingOut, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: "ping -c 1 -W 2 host.docker.internal",
	})
	if err != nil {
		output.UserOut.Println("  ✗ Cannot ping host.docker.internal from web container")
		output.UserOut.Printf("    Error: %v\n", err)
		hasIssues = true
	} else {
		output.UserOut.Println("  ✓ Can ping host.docker.internal from web container")
		util.Verbose("ping output: %s", pingOut)
	}
	output.UserOut.Println()

	// Check 4: Check xdebug_ide_location setting
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

	// Check 5: Start a test listener and test connection
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	output.UserOut.Println("Connection Test")
	output.UserOut.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Only run connection test if port 9003 is not already in use
	if !netutil.IsPortActive("9003") {
		output.UserOut.Println("  Starting test listener on host port 9003...")

		// Start listener in background
		listener, err := net.Listen("tcp", "0.0.0.0:9003")
		if err != nil {
			output.UserOut.Printf("  ✗ Failed to start test listener: %v\n", err)
			hasIssues = true
		} else {
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
				Cmd: "bash -c \"echo 'test from container' | nc -w 2 host.docker.internal 9003\"",
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
		output.UserOut.Println("  ℹ Xdebug is currently disabled")
		output.UserOut.Println("    Enable with: ddev xdebug on")
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
				Cmd: "php -i",
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
