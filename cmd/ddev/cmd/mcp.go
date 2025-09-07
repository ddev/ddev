package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ddev/ddev/pkg/mcp"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var mcpSettings = mcp.ServerSettings{
	Port:          8080,
	AllowWrites:   false,
	AutoApprove:   []string{},
	LogLevel:      "info",
	TransportType: "stdio",
}

// MCPCmd represents the mcp command group
var MCPCmd = &cobra.Command{
	Use:   "mcp",
	Short: "DDEV Model Context Protocol server",
	Long: `Start and manage DDEV MCP server for AI assistant integration.

The MCP server provides structured access to DDEV functionality for AI assistants
like Claude Code. It exposes project discovery, lifecycle management, and service
operations through a standardized JSON-RPC interface.`,
	Example: `ddev mcp start
ddev mcp start --transport=http --port=8080
ddev mcp start --allow-writes
ddev mcp status
ddev mcp stop`,
}

// MCPStartCmd represents the mcp start command
var MCPStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start DDEV MCP server",
	Long: `Start the DDEV MCP server to enable AI assistant integration.

The server provides structured access to DDEV projects and operations through
the Model Context Protocol (MCP). By default, uses stdio transport and runs
with read-only permissions for security.`,
	Example: `ddev mcp start
ddev mcp start --transport=http --port=8080 --allow-writes
ddev mcp start --transport=stdio`,
	Run: func(_ *cobra.Command, _ []string) {
		// Create MCP server with current settings
		server := mcp.NewDDEVMCPServer(mcpSettings)

		// Set up signal handling for graceful shutdown
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		// Start server in goroutine
		serverErr := make(chan error, 1)
		go func() {
			if err := server.Start(ctx); err != nil {
				serverErr <- err
			}
		}()

		// Log server startup
		transportInfo := fmt.Sprintf("transport=%s", mcpSettings.TransportType)
		if mcpSettings.TransportType == "http" {
			transportInfo += fmt.Sprintf(" port=%d", mcpSettings.Port)
		}

		permissions := "read-only"
		if mcpSettings.AllowWrites {
			permissions = "read-write"
		}

		output.UserOut.Printf("DDEV MCP server starting (%s, %s)", transportInfo, permissions)

		// Wait for shutdown signal or server error
		select {
		case err := <-serverErr:
			util.Failed("MCP server failed to start: %v", err)
		case <-sigChan:
			output.UserOut.Println("Shutting down DDEV MCP server...")
			cancel()

			// Graceful shutdown with timeout
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer shutdownCancel()

			done := make(chan error, 1)
			go func() {
				done <- server.Stop()
			}()

			select {
			case err := <-done:
				if err != nil {
					output.UserOut.Printf("Error during shutdown: %v", err)
				} else {
					output.UserOut.Println("DDEV MCP server stopped")
				}
			case <-shutdownCtx.Done():
				output.UserOut.Println("Shutdown timeout exceeded, forcing exit")
			}
		}
	},
}

// MCPStopCmd represents the mcp stop command
var MCPStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop DDEV MCP server",
	Long:  `Stop the running DDEV MCP server if it exists.`,
	Run: func(_ *cobra.Command, _ []string) {
		// TODO: Implement server stop logic
		// This will require tracking running servers, likely via PID files
		// or other IPC mechanisms for Phase 2 implementation
		output.UserOut.Println("MCP server stop functionality not yet implemented")
		output.UserOut.Println("For stdio transport, use Ctrl+C to stop the server")
	},
}

// MCPStatusCmd represents the mcp status command
var MCPStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show DDEV MCP server status",
	Long:  `Display the current status of the DDEV MCP server.`,
	Run: func(_ *cobra.Command, _ []string) {
		// TODO: Implement server status checking
		// This will require tracking running servers for Phase 2 implementation
		output.UserOut.Println("MCP server status functionality not yet implemented")
		output.UserOut.Println("MCP server provides structured access to DDEV projects for AI assistants")
	},
}

func init() {
	// Configure flags for MCPStartCmd
	MCPStartCmd.Flags().BoolVar(&mcpSettings.AllowWrites, "allow-writes", false, "Enable destructive operations (exec commands, config changes)")
	MCPStartCmd.Flags().StringVar(&mcpSettings.TransportType, "transport", "stdio", "Transport type: stdio, http")
	MCPStartCmd.Flags().IntVar(&mcpSettings.Port, "port", 8080, "HTTP port (used with --transport=http)")
	MCPStartCmd.Flags().StringVar(&mcpSettings.LogLevel, "log-level", "info", "Log level: debug, info, warn, error")
	MCPStartCmd.Flags().StringSliceVar(&mcpSettings.AutoApprove, "auto-approve", []string{}, "Commands that don't require approval")

	// Add subcommands to MCPCmd
	MCPCmd.AddCommand(MCPStartCmd)
	MCPCmd.AddCommand(MCPStopCmd)
	MCPCmd.AddCommand(MCPStatusCmd)

	// Register MCPCmd with RootCmd
	RootCmd.AddCommand(MCPCmd)
}
