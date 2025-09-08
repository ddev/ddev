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

// ProjectLifecycleInput represents the input parameters for project lifecycle tools (start/stop/restart)
type ProjectLifecycleInput struct {
	Name      string `json:"name,omitempty" jsonschema:"description:Project name (uses active project if omitted)"`
	AppRoot   string `json:"approot,omitempty" jsonschema:"description:Absolute path to project root (overrides name if set)"`
	SkipHooks bool   `json:"skip_hooks,omitempty" jsonschema:"description:Skip hooks during start operations"`
}

// ProjectLifecycleOutput represents the structured output for project lifecycle operations
type ProjectLifecycleOutput struct {
	ProjectName string `json:"project_name" jsonschema:"description:Name of the project"`
	Operation   string `json:"operation" jsonschema:"description:Operation performed (start/stop/restart)"`
	Success     bool   `json:"success" jsonschema:"description:Whether the operation was successful"`
	Message     string `json:"message,omitempty" jsonschema:"description:Additional information or error message"`
	Status      string `json:"status,omitempty" jsonschema:"description:Current project status after operation"`
}

// registerDDEVTools registers all DDEV MCP tools with the provided server
func registerDDEVTools(server *mcp.Server) error {
	// Register the ddev_list_projects tool using the generic AddTool function
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ddev_list_projects",
		Description: "List all DDEV projects with their current status and configuration",
	}, handleListProjects)

	// Register the ddev_describe_project tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ddev_describe_project",
		Description: "Describe a DDEV project by name or approot (full details)",
	}, handleDescribeProject)

	// Register project lifecycle management tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ddev_start_project",
		Description: "Start a DDEV project with optional hooks skipping",
	}, handleStartProject)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ddev_stop_project",
		Description: "Stop a running DDEV project",
	}, handleStopProject)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ddev_restart_project",
		Description: "Restart a DDEV project (stop then start)",
	}, handleRestartProject)

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

// DescribeProjectInput represents input for describing a single project
type DescribeProjectInput struct {
	// Name: project name. If empty and AppRoot not set, uses current directory.
	Name string `json:"name,omitempty" jsonschema:"description:Project name (uses active project if omitted)"`
	// AppRoot: optional absolute path to project root (directory containing .ddev)
	AppRoot string `json:"approot,omitempty" jsonschema:"description:Absolute path to project root (overrides name if set)"`
	// Short: if true, return brief description
	Short bool `json:"short,omitempty" jsonschema:"description:Return a short summary instead of full details"`
}

// DescribeProjectOutput represents output of the describe tool
type DescribeProjectOutput struct {
	Project map[string]any `json:"project" jsonschema:"description:Full project description"`
}

// handleDescribeProject handles the ddev_describe_project MCP tool
func handleDescribeProject(ctx context.Context, _ *mcp.CallToolRequest, input DescribeProjectInput) (*mcp.CallToolResult, DescribeProjectOutput, error) {
	var app *ddevapp.DdevApp
	var err error

	// Select project by approot > name > current directory
	switch {
	case input.AppRoot != "":
		app, err = ddevapp.NewApp(input.AppRoot, true)
		if err != nil {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to load app at approot %s: %v", input.AppRoot, err)}}}, DescribeProjectOutput{}, nil
		}
	case input.Name != "":
		app, err = ddevapp.GetActiveApp(input.Name)
		if err != nil {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to find active app %s: %v", input.Name, err)}}}, DescribeProjectOutput{}, nil
		}
	default:
		app, err = ddevapp.GetActiveApp("")
		if err != nil {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to find active app from current directory: %v", err)}}}, DescribeProjectOutput{}, nil
		}
	}

	desc, err := app.Describe(input.Short)
	if err != nil {
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to describe project %s: %v", app.GetName(), err)}}}, DescribeProjectOutput{}, nil
	}

	result := &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Described project %s (status=%v)", desc["name"], desc["status"])}}}
	return result, DescribeProjectOutput{Project: desc}, nil
}

// handleStartProject handles the ddev_start_project MCP tool
func handleStartProject(ctx context.Context, _ *mcp.CallToolRequest, input ProjectLifecycleInput) (*mcp.CallToolResult, ProjectLifecycleOutput, error) {
	var app *ddevapp.DdevApp
	var err error

	// Select project by approot > name > current directory
	switch {
	case input.AppRoot != "":
		app, err = ddevapp.NewApp(input.AppRoot, true)
		if err != nil {
			output := ProjectLifecycleOutput{
				ProjectName: input.AppRoot,
				Operation:   "start",
				Success:     false,
				Message:     fmt.Sprintf("Failed to load app at approot %s: %v", input.AppRoot, err),
			}
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	case input.Name != "":
		app, err = ddevapp.GetActiveApp(input.Name)
		if err != nil {
			output := ProjectLifecycleOutput{
				ProjectName: input.Name,
				Operation:   "start",
				Success:     false,
				Message:     fmt.Sprintf("Failed to find active app %s: %v", input.Name, err),
			}
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	default:
		app, err = ddevapp.GetActiveApp("")
		if err != nil {
			output := ProjectLifecycleOutput{
				ProjectName: "current directory",
				Operation:   "start",
				Success:     false,
				Message:     fmt.Sprintf("Failed to find active app from current directory: %v", err),
			}
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	}

	// Start the project (Note: DDEV's Start() doesn't support skip_hooks parameter directly)
	err = app.Start()
	if err != nil {
		status, _ := app.SiteStatus()
		output := ProjectLifecycleOutput{
			ProjectName: app.GetName(),
			Operation:   "start",
			Success:     false,
			Message:     fmt.Sprintf("Failed to start project: %v", err),
			Status:      status,
		}
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
	}

	status, _ := app.SiteStatus()
	output := ProjectLifecycleOutput{
		ProjectName: app.GetName(),
		Operation:   "start",
		Success:     true,
		Message:     "Project started successfully",
		Status:      status,
	}

	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Started project %s successfully", app.GetName())}}}, output, nil
}

// handleStopProject handles the ddev_stop_project MCP tool
func handleStopProject(ctx context.Context, _ *mcp.CallToolRequest, input ProjectLifecycleInput) (*mcp.CallToolResult, ProjectLifecycleOutput, error) {
	var app *ddevapp.DdevApp
	var err error

	// Select project by approot > name > current directory
	switch {
	case input.AppRoot != "":
		app, err = ddevapp.NewApp(input.AppRoot, true)
		if err != nil {
			output := ProjectLifecycleOutput{
				ProjectName: input.AppRoot,
				Operation:   "stop",
				Success:     false,
				Message:     fmt.Sprintf("Failed to load app at approot %s: %v", input.AppRoot, err),
			}
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	case input.Name != "":
		app, err = ddevapp.GetActiveApp(input.Name)
		if err != nil {
			output := ProjectLifecycleOutput{
				ProjectName: input.Name,
				Operation:   "stop",
				Success:     false,
				Message:     fmt.Sprintf("Failed to find active app %s: %v", input.Name, err),
			}
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	default:
		app, err = ddevapp.GetActiveApp("")
		if err != nil {
			output := ProjectLifecycleOutput{
				ProjectName: "current directory",
				Operation:   "stop",
				Success:     false,
				Message:     fmt.Sprintf("Failed to find active app from current directory: %v", err),
			}
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	}

	// Stop the project
	err = app.Stop(false, false)
	if err != nil {
		status, _ := app.SiteStatus()
		output := ProjectLifecycleOutput{
			ProjectName: app.GetName(),
			Operation:   "stop",
			Success:     false,
			Message:     fmt.Sprintf("Failed to stop project: %v", err),
			Status:      status,
		}
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
	}

	status, _ := app.SiteStatus()
	output := ProjectLifecycleOutput{
		ProjectName: app.GetName(),
		Operation:   "stop",
		Success:     true,
		Message:     "Project stopped successfully",
		Status:      status,
	}

	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Stopped project %s successfully", app.GetName())}}}, output, nil
}

// handleRestartProject handles the ddev_restart_project MCP tool
func handleRestartProject(ctx context.Context, _ *mcp.CallToolRequest, input ProjectLifecycleInput) (*mcp.CallToolResult, ProjectLifecycleOutput, error) {
	var app *ddevapp.DdevApp
	var err error

	// Select project by approot > name > current directory
	switch {
	case input.AppRoot != "":
		app, err = ddevapp.NewApp(input.AppRoot, true)
		if err != nil {
			output := ProjectLifecycleOutput{
				ProjectName: input.AppRoot,
				Operation:   "restart",
				Success:     false,
				Message:     fmt.Sprintf("Failed to load app at approot %s: %v", input.AppRoot, err),
			}
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	case input.Name != "":
		app, err = ddevapp.GetActiveApp(input.Name)
		if err != nil {
			output := ProjectLifecycleOutput{
				ProjectName: input.Name,
				Operation:   "restart",
				Success:     false,
				Message:     fmt.Sprintf("Failed to find active app %s: %v", input.Name, err),
			}
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	default:
		app, err = ddevapp.GetActiveApp("")
		if err != nil {
			output := ProjectLifecycleOutput{
				ProjectName: "current directory",
				Operation:   "restart",
				Success:     false,
				Message:     fmt.Sprintf("Failed to find active app from current directory: %v", err),
			}
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	}

	// Restart the project (stop then start)
	err = app.Restart()
	if err != nil {
		status, _ := app.SiteStatus()
		output := ProjectLifecycleOutput{
			ProjectName: app.GetName(),
			Operation:   "restart",
			Success:     false,
			Message:     fmt.Sprintf("Failed to restart project: %v", err),
			Status:      status,
		}
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
	}

	status, _ := app.SiteStatus()
	output := ProjectLifecycleOutput{
		ProjectName: app.GetName(),
		Operation:   "restart",
		Success:     true,
		Message:     "Project restarted successfully",
		Status:      status,
	}

	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Restarted project %s successfully", app.GetName())}}}, output, nil
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
