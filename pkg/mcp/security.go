package mcp

import (
	"fmt"
	"slices"
	"time"

	"github.com/sirupsen/logrus"
)

// BasicSecurityManager implements the SecurityManager interface
type BasicSecurityManager struct {
	settings     ServerSettings
	logger       *logrus.Logger
	operationLog []OperationLogEntry
}

// OperationLogEntry represents a logged MCP operation
type OperationLogEntry struct {
	Timestamp time.Time      `json:"timestamp"`
	ToolName  string         `json:"tool_name"`
	Args      map[string]any `json:"args"`
	Result    any            `json:"result,omitempty"`
	Error     string         `json:"error,omitempty"`
	Success   bool           `json:"success"`
}

// NewSecurityManager creates a new security manager instance
func NewSecurityManager(settings ServerSettings) SecurityManager {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	return &BasicSecurityManager{
		settings:     settings,
		logger:       logger,
		operationLog: make([]OperationLogEntry, 0),
	}
}

// CheckPermission validates if an operation is allowed based on current settings
func (sm *BasicSecurityManager) CheckPermission(toolName string, args map[string]any) error {
	permLevel := sm.getToolPermissionLevel(toolName)

	switch permLevel {
	case ReadOnly:
		// Always allowed
		return nil
	case SafeOperations:
		// Allowed if not explicitly restricted
		return nil
	case DestructiveOperations:
		// Only allowed if AllowWrites is enabled
		if !sm.settings.AllowWrites {
			return fmt.Errorf("destructive operation %s blocked: requires --allow-writes flag", toolName)
		}
		return nil
	default:
		return fmt.Errorf("unknown permission level for tool %s", toolName)
	}
}

// RequiresApproval determines if an operation needs explicit user approval
func (sm *BasicSecurityManager) RequiresApproval(toolName string, args map[string]any) bool {
	// Check if tool is in auto-approve list
	if slices.Contains(sm.settings.AutoApprove, toolName) {
		return false
	}

	// Destructive operations require approval unless auto-approved
	permLevel := sm.getToolPermissionLevel(toolName)
	return permLevel == DestructiveOperations
}

// LogOperation records an MCP operation for audit purposes
func (sm *BasicSecurityManager) LogOperation(toolName string, args map[string]any, result any, err error) {
	entry := OperationLogEntry{
		Timestamp: time.Now(),
		ToolName:  toolName,
		Args:      args,
		Result:    result,
		Success:   err == nil,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	sm.operationLog = append(sm.operationLog, entry)

	// Log to structured logger
	logEntry := sm.logger.WithFields(logrus.Fields{
		"tool":      toolName,
		"success":   entry.Success,
		"timestamp": entry.Timestamp,
	})

	if err != nil {
		logEntry.WithError(err).Error("MCP operation failed")
	} else {
		logEntry.Info("MCP operation completed")
	}
}

// getToolPermissionLevel determines the permission level required for a specific tool
func (sm *BasicSecurityManager) getToolPermissionLevel(toolName string) PermissionLevel {
	destructiveOps := []string{
		"ddev_exec_command",
		"ddev_configure_project",
		"ddev_update_config",
	}

	safeOps := []string{
		"ddev_start_project",
		"ddev_stop_project",
		"ddev_restart_project",
	}

	if slices.Contains(destructiveOps, toolName) {
		return DestructiveOperations
	}

	if slices.Contains(safeOps, toolName) {
		return SafeOperations
	}

	// Default to read-only for unknown tools
	return ReadOnly
}

// GetOperationLog returns the current operation log (for debugging/auditing)
func (sm *BasicSecurityManager) GetOperationLog() []OperationLogEntry {
	return sm.operationLog
}
