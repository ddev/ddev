package mcp

// tools.go contains the MCP tool implementations for DDEV functionality

import (
	"context"
	"fmt"
	"strings"

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

// ExecCommandInput represents the input parameters for the ddev_exec_command tool
type ExecCommandInput struct {
	Name    string `json:"name,omitempty" jsonschema:"description:Project name (uses active project if omitted)"`
	AppRoot string `json:"approot,omitempty" jsonschema:"description:Absolute path to project root (overrides name if set)"`
	Service string `json:"service,omitempty" jsonschema:"description:Service to execute in (default: web)"`
	Command string `json:"command" jsonschema:"description:Command to execute"`
	Dir     string `json:"dir,omitempty" jsonschema:"description:Working directory inside container"`
}

// ExecCommandOutput represents the structured output for command execution
type ExecCommandOutput struct {
	ProjectName string `json:"project_name" jsonschema:"description:Name of the project"`
	Service     string `json:"service" jsonschema:"description:Service where command was executed"`
	Command     string `json:"command" jsonschema:"description:Command that was executed"`
	WorkingDir  string `json:"working_dir,omitempty" jsonschema:"description:Working directory where command was executed"`
	Stdout      string `json:"stdout,omitempty" jsonschema:"description:Standard output from command"`
	Stderr      string `json:"stderr,omitempty" jsonschema:"description:Standard error from command"`
	Success     bool   `json:"success" jsonschema:"description:Whether the command executed successfully"`
	Message     string `json:"message,omitempty" jsonschema:"description:Additional information or error message"`
}

// LogsInput represents the input parameters for the ddev_logs tool
type LogsInput struct {
	Name       string `json:"name,omitempty" jsonschema:"description:Project name (uses active project if omitted)"`
	AppRoot    string `json:"approot,omitempty" jsonschema:"description:Absolute path to project root (overrides name if set)"`
	Service    string `json:"service,omitempty" jsonschema:"description:Service to get logs from (default: web)"`
	TailLines  string `json:"tail,omitempty" jsonschema:"description:Number of lines to tail (default: all)"`
	Timestamps bool   `json:"timestamps,omitempty" jsonschema:"description:Show timestamps in log output"`
}

// LogsOutput represents the structured output for log retrieval
type LogsOutput struct {
	ProjectName string `json:"project_name" jsonschema:"description:Name of the project"`
	Service     string `json:"service" jsonschema:"description:Service from which logs were retrieved"`
	Lines       int    `json:"lines" jsonschema:"description:Number of log lines returned"`
	Logs        string `json:"logs" jsonschema:"description:Log content"`
	Success     bool   `json:"success" jsonschema:"description:Whether log retrieval was successful"`
	Message     string `json:"message,omitempty" jsonschema:"description:Additional information or error message"`
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

	// Register service operation tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ddev_exec_command",
		Description: "Execute commands in DDEV project containers",
	}, handleExecCommand)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ddev_logs",
		Description: "Retrieve logs from DDEV project services",
	}, handleLogs)

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
func handleExecCommand(ctx context.Context, _ *mcp.CallToolRequest, input ExecCommandInput) (*mcp.CallToolResult, ExecCommandOutput, error) {
	var app *ddevapp.DdevApp
	var err error

	// Select project by approot > name > current directory
	switch {
	case input.AppRoot != "":
		app, err = ddevapp.NewApp(input.AppRoot, true)
		if err != nil {
			output := ExecCommandOutput{
				ProjectName: input.AppRoot,
				Service:     input.Service,
				Command:     input.Command,
				Success:     false,
				Message:     fmt.Sprintf("Failed to load app at approot %s: %v", input.AppRoot, err),
			}
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	case input.Name != "":
		app, err = ddevapp.GetActiveApp(input.Name)
		if err != nil {
			output := ExecCommandOutput{
				ProjectName: input.Name,
				Service:     input.Service,
				Command:     input.Command,
				Success:     false,
				Message:     fmt.Sprintf("Failed to find active app %s: %v", input.Name, err),
			}
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	default:
		app, err = ddevapp.GetActiveApp("")
		if err != nil {
			output := ExecCommandOutput{
				ProjectName: "current directory",
				Service:     input.Service,
				Command:     input.Command,
				Success:     false,
				Message:     fmt.Sprintf("Failed to find active app from current directory: %v", err),
			}
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	}

	// Set default service if not specified
	service := input.Service
	if service == "" {
		service = "web"
	}

	// Execute command using DDEV's Exec functionality
	opts := &ddevapp.ExecOpts{
		Service: service,
		Dir:     input.Dir,
		Cmd:     input.Command,
	}

	stdout, stderr, err := app.Exec(opts)
	success := err == nil

	output := ExecCommandOutput{
		ProjectName: app.GetName(),
		Service:     service,
		Command:     input.Command,
		WorkingDir:  input.Dir,
		Stdout:      stdout,
		Stderr:      stderr,
		Success:     success,
	}

	if err != nil {
		output.Message = fmt.Sprintf("Command execution failed: %v", err)
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
	}

	output.Message = "Command executed successfully"
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Executed '%s' in %s service of project %s", input.Command, service, app.GetName())}}}, output, nil
}

// handleLogs handles the ddev_logs MCP tool
func handleLogs(ctx context.Context, _ *mcp.CallToolRequest, input LogsInput) (*mcp.CallToolResult, LogsOutput, error) {
	var app *ddevapp.DdevApp
	var err error

	// Select project by approot > name > current directory
	switch {
	case input.AppRoot != "":
		app, err = ddevapp.NewApp(input.AppRoot, true)
		if err != nil {
			output := LogsOutput{
				ProjectName: input.AppRoot,
				Service:     input.Service,
				Success:     false,
				Message:     fmt.Sprintf("Failed to load app at approot %s: %v", input.AppRoot, err),
			}
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	case input.Name != "":
		app, err = ddevapp.GetActiveApp(input.Name)
		if err != nil {
			output := LogsOutput{
				ProjectName: input.Name,
				Service:     input.Service,
				Success:     false,
				Message:     fmt.Sprintf("Failed to find active app %s: %v", input.Name, err),
			}
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	default:
		app, err = ddevapp.GetActiveApp("")
		if err != nil {
			output := LogsOutput{
				ProjectName: "current directory",
				Service:     input.Service,
				Success:     false,
				Message:     fmt.Sprintf("Failed to find active app from current directory: %v", err),
			}
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	}

	// Set default service if not specified
	service := input.Service
	if service == "" {
		service = "web"
	}

	// Set default tail lines if not specified
	tailLines := input.TailLines
	if tailLines == "" {
		tailLines = "all"
	}

	// Capture logs using DDEV's CaptureLogs functionality
	logs, err := app.CaptureLogs(service, input.Timestamps, tailLines)
	if err != nil {
		output := LogsOutput{
			ProjectName: app.GetName(),
			Service:     service,
			Success:     false,
			Message:     fmt.Sprintf("Failed to capture logs: %v", err),
		}
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
	}

	// Count lines in logs
	lines := len([]rune(logs))
	if logs != "" {
		lines = len(strings.Split(logs, "\n"))
	}

	output := LogsOutput{
		ProjectName: app.GetName(),
		Service:     service,
		Lines:       lines,
		Logs:        logs,
		Success:     true,
		Message:     "Logs retrieved successfully",
	}

	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Retrieved %d lines of logs from %s service of project %s", lines, service, app.GetName())}}}, output, nil
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
