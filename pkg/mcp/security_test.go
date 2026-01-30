package mcp

import (
	"fmt"
	"testing"
)

func TestSecurityManagerPermissions(t *testing.T) {
	tests := []struct {
		name        string
		settings    ServerSettings
		toolName    string
		expectError bool
		description string
	}{
		{
			name: "ReadOnly operations always allowed",
			settings: ServerSettings{
				AllowWrites: false,
				AutoApprove: []string{},
			},
			toolName:    "ddev_list_projects",
			expectError: false,
			description: "Read-only operations should always be allowed",
		},
		{
			name: "SafeOperations allowed without AllowWrites",
			settings: ServerSettings{
				AllowWrites: false,
				AutoApprove: []string{},
			},
			toolName:    "ddev_start_project",
			expectError: false,
			description: "Safe operations should be allowed even without AllowWrites",
		},
		{
			name: "DestructiveOperations blocked without AllowWrites",
			settings: ServerSettings{
				AllowWrites: false,
				AutoApprove: []string{},
			},
			toolName:    "ddev_exec_command",
			expectError: true,
			description: "Destructive operations should be blocked without AllowWrites",
		},
		{
			name: "DestructiveOperations allowed with AllowWrites",
			settings: ServerSettings{
				AllowWrites: true,
				AutoApprove: []string{},
			},
			toolName:    "ddev_exec_command",
			expectError: false,
			description: "Destructive operations should be allowed with AllowWrites",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			securityManager := NewSecurityManager(tt.settings)
			args := map[string]any{"test": "value"}

			err := securityManager.CheckPermission(tt.toolName, args)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s but got none. %s", tt.toolName, tt.description)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s but got: %v. %s", tt.toolName, err, tt.description)
			}
		})
	}
}

func TestSecurityManagerApproval(t *testing.T) {
	tests := []struct {
		name           string
		settings       ServerSettings
		toolName       string
		expectApproval bool
		description    string
	}{
		{
			name: "ReadOnly operations don't require approval",
			settings: ServerSettings{
				AllowWrites: false,
				AutoApprove: []string{},
			},
			toolName:       "ddev_list_projects",
			expectApproval: false,
			description:    "Read-only operations should not require approval",
		},
		{
			name: "SafeOperations don't require approval",
			settings: ServerSettings{
				AllowWrites: false,
				AutoApprove: []string{},
			},
			toolName:       "ddev_start_project",
			expectApproval: false,
			description:    "Safe operations should not require approval",
		},
		{
			name: "DestructiveOperations require approval by default",
			settings: ServerSettings{
				AllowWrites: true,
				AutoApprove: []string{},
			},
			toolName:       "ddev_exec_command",
			expectApproval: true,
			description:    "Destructive operations should require approval by default",
		},
		{
			name: "AutoApprove bypasses approval requirement",
			settings: ServerSettings{
				AllowWrites: true,
				AutoApprove: []string{"ddev_exec_command"},
			},
			toolName:       "ddev_exec_command",
			expectApproval: false,
			description:    "Operations in AutoApprove list should not require approval",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			securityManager := NewSecurityManager(tt.settings)
			args := map[string]any{"test": "value"}

			requiresApproval := securityManager.RequiresApproval(tt.toolName, args)

			if tt.expectApproval && !requiresApproval {
				t.Errorf("Expected approval required for %s but got false. %s", tt.toolName, tt.description)
			}

			if !tt.expectApproval && requiresApproval {
				t.Errorf("Expected no approval required for %s but got true. %s", tt.toolName, tt.description)
			}
		})
	}
}

func TestSecurityManagerApprovalRequest(t *testing.T) {
	securityManager := NewSecurityManager(ServerSettings{
		AllowWrites: true,
		AutoApprove: []string{},
	})

	args := map[string]any{"command": "test"}
	description := "Test command execution"

	// RequestApproval should always return an error in MCP context (no interactive prompts)
	err := securityManager.RequestApproval("ddev_exec_command", args, description)

	if err == nil {
		t.Error("Expected error from RequestApproval in MCP context, but got none")
	}

	expectedMessage := "operation ddev_exec_command requires approval"
	if err != nil && err.Error()[:len(expectedMessage)] != expectedMessage {
		t.Errorf("Expected approval error to start with '%s', got: %v", expectedMessage, err)
	}
}

func TestSecurityManagerToolPermissionLevels(t *testing.T) {
	securityManager := &BasicSecurityManager{
		settings: ServerSettings{},
	}

	tests := []struct {
		toolName string
		expected PermissionLevel
	}{
		{"ddev_list_projects", SafeOperations},
		{"ddev_describe_project", ReadOnly},
		{"ddev_start_project", SafeOperations},
		{"ddev_stop_project", SafeOperations},
		{"ddev_restart_project", SafeOperations},
		{"ddev_logs", ReadOnly},
		{"ddev_exec_command", DestructiveOperations},
		{"unknown_tool", ReadOnly},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			level := securityManager.getToolPermissionLevel(tt.toolName)
			if level != tt.expected {
				t.Errorf("Expected permission level %v for %s, got %v", tt.expected, tt.toolName, level)
			}
		})
	}
}

func TestSecurityManagerOperationLogging(t *testing.T) {
	settings := ServerSettings{
		AllowWrites: true,
	}
	securityManager := NewSecurityManager(settings)

	// Cast to concrete type to access GetOperationLog method
	basicSecManager, ok := securityManager.(*BasicSecurityManager)
	if !ok {
		t.Fatal("Expected BasicSecurityManager type")
	}

	toolName := "ddev_exec_command"
	args := map[string]any{"command": "test"}
	result := map[string]any{"success": true}

	// Test successful operation logging
	securityManager.LogOperation(toolName, args, result, nil)

	logs := basicSecManager.GetOperationLog()
	if len(logs) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(logs))
	}

	if len(logs) > 0 {
		log := logs[0]
		if log.ToolName != toolName {
			t.Errorf("Expected tool name %s, got %s", toolName, log.ToolName)
		}
		if !log.Success {
			t.Error("Expected successful operation to be logged as success=true")
		}
	}

	// Test failed operation logging
	testErr := fmt.Errorf("test error")
	securityManager.LogOperation(toolName, args, nil, testErr)

	logs = basicSecManager.GetOperationLog()
	if len(logs) != 2 {
		t.Errorf("Expected 2 log entries, got %d", len(logs))
	}

	if len(logs) > 1 {
		log := logs[1]
		if log.Success {
			t.Error("Expected failed operation to be logged as success=false")
		}
		if log.Error != testErr.Error() {
			t.Errorf("Expected error message '%s', got '%s'", testErr.Error(), log.Error)
		}
	}
}
