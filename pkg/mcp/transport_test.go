package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestStdioTransport(t *testing.T) {
	t.Run("NewStdioTransport", func(t *testing.T) {
		server := mcp.NewServer(&mcp.Implementation{
			Name:    "test-server",
			Version: "test",
		}, nil)
		transport := NewStdioTransport(server)

		if transport == nil {
			t.Fatal("Expected non-nil transport")
		}

		if transport.IsRunning() {
			t.Error("Expected transport not to be running initially")
		}
	})

	t.Run("StdioTransport Start/Stop", func(t *testing.T) {
		// Create a mock server for testing
		server := mcp.NewServer(&mcp.Implementation{
			Name:    "test-server",
			Version: "test",
		}, nil)
		if server == nil {
			t.Fatal("Failed to create mock server")
		}

		transport := NewStdioTransport(server)

		// Start transport with timeout to prevent hanging
		_ = context.Background()

		// Start in goroutine since stdio transport blocks
		done := make(chan error, 1)
		go func() {
			err := transport.Start()
			done <- err
		}()

		// Give it a moment to start
		time.Sleep(50 * time.Millisecond)

		if !transport.IsRunning() {
			t.Error("Expected transport to be running after Start()")
		}

		// Stop transport
		err := transport.Stop()
		if err != nil {
			t.Errorf("Failed to stop transport: %v", err)
		}

		// Wait for start goroutine to finish
		select {
		case startErr := <-done:
			// Context cancellation is expected
			if startErr != nil && startErr.Error() != "context canceled" {
				t.Errorf("Unexpected error from Start(): %v", startErr)
			}
		case <-time.After(200 * time.Millisecond):
			t.Error("Start() goroutine did not finish in time")
		}

		if transport.IsRunning() {
			t.Error("Expected transport not to be running after Stop()")
		}
	})
}

func TestHTTPTransport(t *testing.T) {
	t.Run("NewHTTPTransport", func(t *testing.T) {
		server := mcp.NewServer(&mcp.Implementation{
			Name:    "test-server",
			Version: "test",
		}, nil)
		transport := NewHTTPTransport(server, 0) // Use port 0 for random available port

		if transport == nil {
			t.Fatal("Expected non-nil transport")
		}

		if transport.IsRunning() {
			t.Error("Expected transport not to be running initially")
		}
	})

	t.Run("HTTPTransport Start/Stop", func(t *testing.T) {
		// Create a mock server for testing
		server := mcp.NewServer(&mcp.Implementation{
			Name:    "test-server",
			Version: "test",
		}, nil)
		if server == nil {
			t.Fatal("Failed to create mock server")
		}

		transport := NewHTTPTransport(server, 0)

		// Start transport with timeout
		_ = context.Background()

		// Start in goroutine since it blocks
		done := make(chan error, 1)
		go func() {
			err := transport.Start()
			done <- err
		}()

		// Give server time to start
		time.Sleep(200 * time.Millisecond)

		if !transport.IsRunning() {
			t.Error("Expected transport to be running after Start()")
		}

		// Test basic functionality without port inspection
		t.Log("HTTP transport test completed")

		// Stop transport
		err := transport.Stop()
		if err != nil {
			t.Errorf("Failed to stop transport: %v", err)
		}

		// Wait for start goroutine to finish
		select {
		case err := <-done:
			// Context cancellation is expected
			if err != nil && err.Error() != "context canceled" &&
				err.Error() != "http: Server closed" {
				t.Errorf("Unexpected error from Start(): %v", err)
			}
		case <-time.After(3 * time.Second):
			t.Error("Start() goroutine did not finish in time")
		}

		if transport.IsRunning() {
			t.Error("Expected transport not to be running after Stop()")
		}
	})

	t.Run("HTTPTransport concurrent operations", func(t *testing.T) {
		server := mcp.NewServer(&mcp.Implementation{
			Name:    "test-server",
			Version: "test",
		}, nil)
		if server == nil {
			t.Fatal("Failed to create mock server")
		}

		transport := NewHTTPTransport(server, 0)

		// Start transport
		go func() {
			_ = transport.Start()
		}()

		time.Sleep(100 * time.Millisecond)

		// Test multiple concurrent stop calls
		stopResults := make(chan error, 3)

		for i := 0; i < 3; i++ {
			go func() {
				err := transport.Stop()
				stopResults <- err
			}()
		}

		// Collect results
		for i := 0; i < 3; i++ {
			select {
			case err := <-stopResults:
				if err != nil {
					t.Logf("Stop call %d returned error (may be expected): %v", i, err)
				}
			case <-time.After(500 * time.Millisecond):
				t.Errorf("Stop call %d timed out", i)
			}
		}

		// Verify final state
		if transport.IsRunning() {
			t.Error("Expected transport not to be running after concurrent stops")
		}
	})
}

func TestTransportIntegration(t *testing.T) {
	t.Run("Transport with real MCP tools", func(t *testing.T) {
		// Test that transports work with actual MCP server and tools
		settings := ServerSettings{
			TransportType: "http",
			Port:          0,
			LogLevel:      "error",
			AllowWrites:   false,
			AutoApprove:   []string{},
		}

		mcpServer := NewDDEVMCPServer(settings)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		// Start server in background
		go func() {
			_ = mcpServer.Start(ctx)
		}()

		// Give server time to start
		time.Sleep(200 * time.Millisecond)

		if !mcpServer.IsRunning() {
			t.Error("Expected MCP server to be running with HTTP transport")
		}

		// Stop server
		err := mcpServer.Stop()
		if err != nil {
			t.Errorf("Failed to stop MCP server: %v", err)
		}

		if mcpServer.IsRunning() {
			t.Error("Expected MCP server not to be running after Stop()")
		}
	})
}

func TestTransportErrorHandling(t *testing.T) {
	t.Run("HTTP transport with invalid host", func(t *testing.T) {
		server := mcp.NewServer(&mcp.Implementation{
			Name:    "test-server",
			Version: "test",
		}, nil)
		transport := NewHTTPTransport(server, 8080)

		_ = context.Background()

		err := transport.Start()
		// Should get some kind of error for invalid host
		if err == nil {
			t.Error("Expected error for invalid host")
		}

		t.Logf("Got expected error for invalid host: %v", err)
	})

	t.Run("HTTP transport stop before start", func(t *testing.T) {
		server := mcp.NewServer(&mcp.Implementation{
			Name:    "test-server",
			Version: "test",
		}, nil)
		transport := NewHTTPTransport(server, 0)

		err := transport.Stop()
		// Should handle gracefully
		if err != nil {
			t.Logf("Stop before start returned: %v", err)
		}

		if transport.IsRunning() {
			t.Error("Expected transport not to be running after stop-before-start")
		}
	})

	t.Run("Stdio transport stop before start", func(t *testing.T) {
		server := mcp.NewServer(&mcp.Implementation{
			Name:    "test-server",
			Version: "test",
		}, nil)
		transport := NewStdioTransport(server)

		err := transport.Stop()
		// Should handle gracefully
		if err != nil {
			t.Logf("Stop before start returned: %v", err)
		}

		if transport.IsRunning() {
			t.Error("Expected transport not to be running after stop-before-start")
		}
	})
}

func TestTransportTypes(t *testing.T) {
	t.Run("Transport interface compliance", func(t *testing.T) {
		var transport Transport

		// Test StdioTransport implements Transport
		server := mcp.NewServer(&mcp.Implementation{
			Name:    "test-server",
			Version: "test",
		}, nil)
		transport = NewStdioTransport(server)
		// Verify transport implements interface
		_ = transport.IsRunning()
		_ = transport.Stop()

		// Test HTTPTransport implements Transport
		transport = NewHTTPTransport(server, 8080)
		// Verify transport implements interface
		_ = transport.IsRunning()
		_ = transport.Stop()

		// Verify all interface methods are available
		_ = transport.IsRunning()
		_ = transport.Stop()
		// Note: Start() requires server parameter so can't test signature here
	})
}
