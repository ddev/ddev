package mcp

// tools.go contains the MCP tool implementations for DDEV functionality

import (
	"context"
	"fmt"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListProjectsInput represents the input parameters for the ddev_list_projects tool
type ListProjectsInput struct {
	ActiveOnly bool   `json:"active_only,omitempty" jsonschema:"description:If set, only currently active projects will be returned"`
	TypeFilter string `json:"type_filter,omitempty" jsonschema:"description:Show only projects of this type (e.g. drupal11, wordpress, php)"`
}

// ListProjectsOutput represents the structured output of the ddev_list_projects tool
type ListProjectsOutput struct {
	Projects []ProjectInfo `json:"projects" jsonschema:"description:List of DDEV projects"`
	Count    int           `json:"count" jsonschema:"description:Number of projects returned"`
}

// registerDDEVTools registers all DDEV MCP tools with the provided server
func registerDDEVTools(server *mcp.Server) error {
	// Register the ddev_list_projects tool using the generic AddTool function
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ddev_list_projects",
		Description: "List all DDEV projects with their current status and configuration",
	}, handleListProjects)

	return nil
}

// handleListProjects handles the ddev_list_projects MCP tool
// The signature matches the ToolHandlerFor pattern from the SDK
func handleListProjects(ctx context.Context, req *mcp.CallToolRequest, input ListProjectsInput) (*mcp.CallToolResult, ListProjectsOutput, error) {
	// Use existing DDEV functionality to get projects
	settings := ddevapp.ListCommandSettings{
		ActiveOnly: input.ActiveOnly,
		TypeFilter: input.TypeFilter,
	}

	projects, err := ddevapp.GetProjects(settings.ActiveOnly)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Failed to get DDEV projects: %v", err),
				},
			},
		}, ListProjectsOutput{}, nil
	}

	// Build the response
	var projectList []ProjectInfo
	for _, app := range projects {
		// Apply type filter if specified
		if settings.TypeFilter != "" && settings.TypeFilter != app.Type {
			continue
		}

		// Get detailed project description
		desc, err := app.Describe(true)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: fmt.Sprintf("Failed to describe project %s: %v", app.GetName(), err),
					},
				},
			}, ListProjectsOutput{}, nil
		}

		// Convert to our ProjectInfo structure
		projectInfo := ProjectInfo{
			Name:       getStringFromInterface(desc, "name", ""),
			Status:     getStringFromInterface(desc, "status", ""),
			StatusDesc: getStringFromInterface(desc, "status_desc", ""),
			AppRoot:    getStringFromInterface(desc, "approot", ""),
			ShortRoot:  getStringFromInterface(desc, "shortroot", ""),
			Type:       getStringFromInterface(desc, "type", ""),
			PrimaryURL: getStringFromInterface(desc, "primary_url", ""),
			HTTPSUrl:   getStringFromInterface(desc, "httpsurl", ""),
			HTTPUrl:    getStringFromInterface(desc, "httpurl", ""),
		}

		// Add services info if available
		if services, ok := desc["services"]; ok {
			projectInfo.Services = map[string]any{"services": services}
		}

		// Add database info if available
		if dbInfo, ok := desc["dbinfo"]; ok {
			projectInfo.DatabaseInfo = map[string]any{"dbinfo": dbInfo}
		}

		projectList = append(projectList, projectInfo)
	}

	output := ListProjectsOutput{
		Projects: projectList,
		Count:    len(projectList),
	}

	// Return MCP-compatible result
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("Found %d DDEV projects", len(projectList)),
			},
		},
	}, output, nil
}

// handleDescribeProject handles the ddev_describe_project MCP tool
func handleDescribeProject(ctx context.Context, args map[string]any) (any, error) {
	// TODO: Implement in Task 4
	return nil, fmt.Errorf("ddev_describe_project not yet implemented")
}

// handleStartProject handles the ddev_start_project MCP tool
func handleStartProject(ctx context.Context, args map[string]any) (any, error) {
	// TODO: Implement in Task 5
	return nil, fmt.Errorf("ddev_start_project not yet implemented")
}

// handleStopProject handles the ddev_stop_project MCP tool
func handleStopProject(ctx context.Context, args map[string]any) (any, error) {
	// TODO: Implement in Task 5
	return nil, fmt.Errorf("ddev_stop_project not yet implemented")
}

// handleRestartProject handles the ddev_restart_project MCP tool
func handleRestartProject(ctx context.Context, args map[string]any) (any, error) {
	// TODO: Implement in Task 5
	return nil, fmt.Errorf("ddev_restart_project not yet implemented")
}

// handleExecCommand handles the ddev_exec_command MCP tool
func handleExecCommand(ctx context.Context, args map[string]any) (any, error) {
	// TODO: Implement in Task 6
	return nil, fmt.Errorf("ddev_exec_command not yet implemented")
}

// handleLogs handles the ddev_logs MCP tool
func handleLogs(ctx context.Context, args map[string]any) (any, error) {
	// TODO: Implement in Task 6
	return nil, fmt.Errorf("ddev_logs not yet implemented")
}

// Helper functions for tool implementations

// getBool safely extracts a boolean value from args map
func getBool(args map[string]any, key string, defaultValue bool) bool {
	if val, exists := args[key]; exists {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return defaultValue
}

// getString safely extracts a string value from args map
func getString(args map[string]any, key string, defaultValue string) string {
	if val, exists := args[key]; exists {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return defaultValue
}

// getStringFromInterface safely extracts a string value from map[string]interface{}
func getStringFromInterface(args map[string]interface{}, key string, defaultValue string) string {
	if val, exists := args[key]; exists {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return defaultValue
}

// getInt safely extracts an integer value from args map
func getInt(args map[string]any, key string, defaultValue int) int {
	if val, exists := args[key]; exists {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return defaultValue
}
