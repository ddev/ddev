package mcp

import (
	"context"

	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewDDEVMCPServer creates a new DDEV MCP server instance
func NewDDEVMCPServer(settings ServerSettings) *DDEVMCPServer {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "ddev-server",
		Version: versionconstants.DdevVersion,
	}, nil)

	return &DDEVMCPServer{
		server:   server,
		settings: settings,
		security: NewSecurityManager(settings),
	}
}

// Start initializes and starts the MCP server
func (s *DDEVMCPServer) Start(ctx context.Context) error {
	// Register MCP tools
	if err := s.registerTools(); err != nil {
		return err
	}

	// Initialize transport based on settings
	transport, err := s.createTransport()
	if err != nil {
		return err
	}

	// Safely set the transport
	s.mu.Lock()
	s.transport = transport
	s.mu.Unlock()

	// Start the transport
	return transport.Start(ctx)
}

// Stop gracefully shuts down the MCP server
func (s *DDEVMCPServer) Stop() error {
	s.mu.RLock()
	transport := s.transport
	s.mu.RUnlock()

	if transport != nil && transport.IsRunning() {
		return transport.Stop()
	}
	return nil
}

// IsRunning returns whether the MCP server is currently running
func (s *DDEVMCPServer) IsRunning() bool {
	s.mu.RLock()
	transport := s.transport
	s.mu.RUnlock()

	return transport != nil && transport.IsRunning()
}

// registerTools registers all available DDEV MCP tools with the server
func (s *DDEVMCPServer) registerTools() error {
	return registerDDEVTools(s.server, s.security)
}

// createTransport creates the appropriate transport based on server settings
func (s *DDEVMCPServer) createTransport() (Transport, error) {
	switch s.settings.TransportType {
	case "stdio":
		return NewStdioTransport(s.server), nil
	case "http":
		return NewHTTPTransport(s.server, s.settings.Port), nil
	default:
		// Default to stdio transport
		return NewStdioTransport(s.server), nil
	}
}
