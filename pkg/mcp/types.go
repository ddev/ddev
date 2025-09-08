package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DDEVMCPServer represents the DDEV MCP server instance
type DDEVMCPServer struct {
	server    *mcp.Server
	transport Transport
	security  SecurityManager
	settings  ServerSettings
}

// ServerSettings contains configuration for the MCP server
type ServerSettings struct {
	Port          int
	AllowWrites   bool
	AutoApprove   []string // Commands that don't need approval
	LogLevel      string
	TransportType string // "stdio", "http", "websocket"
}

// Transport interface for different MCP transport methods
type Transport interface {
	Start(ctx context.Context) error
	Stop() error
	IsRunning() bool
}

// SecurityManager handles permissions and approval workflows
type SecurityManager interface {
	CheckPermission(toolName string, args map[string]any) error
	RequiresApproval(toolName string, args map[string]any) bool
	RequestApproval(toolName string, args map[string]any, description string) error
	LogOperation(toolName string, args map[string]any, result any, err error)
}

// ProjectInfo represents DDEV project information structure
type ProjectInfo struct {
	Name         string         `json:"name"`
	Status       string         `json:"status"`
	StatusDesc   string         `json:"status_desc"`
	AppRoot      string         `json:"approot"`
	ShortRoot    string         `json:"shortroot"`
	Type         string         `json:"type"`
	PrimaryURL   string         `json:"primary_url"`
	HTTPSUrl     string         `json:"httpsurl"`
	HTTPUrl      string         `json:"httpurl"`
	Services     map[string]any `json:"services,omitempty"`
	DatabaseInfo map[string]any `json:"dbinfo,omitempty"`
}

// OperationResult represents the result of MCP operations
type OperationResult struct {
	Success  bool           `json:"success"`
	Message  string         `json:"message"`
	Data     map[string]any `json:"data,omitempty"`
	Errors   []string       `json:"errors,omitempty"`
	Warnings []string       `json:"warnings,omitempty"`
}

// PermissionLevel defines different security levels
type PermissionLevel int

const (
	ReadOnly PermissionLevel = iota
	SafeOperations
	DestructiveOperations
)

// String returns string representation of permission level
func (p PermissionLevel) String() string {
	switch p {
	case ReadOnly:
		return "read-only"
	case SafeOperations:
		return "safe-operations"
	case DestructiveOperations:
		return "destructive-operations"
	default:
		return "unknown"
	}
}
