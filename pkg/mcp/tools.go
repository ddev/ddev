package mcp

// tools.go contains the MCP tool implementations for DDEV functionality

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

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

// GetConfigInput represents the input parameters for the ddev_get_config tool
type GetConfigInput struct {
	Name    string `json:"name,omitempty" jsonschema:"description:Project name (uses active project if omitted)"`
	AppRoot string `json:"approot,omitempty" jsonschema:"description:Absolute path to project root (overrides name if set)"`
}

// GetConfigOutput represents the structured output for configuration retrieval
type GetConfigOutput struct {
	ProjectName string         `json:"project_name" jsonschema:"description:Name of the project"`
	ConfigPath  string         `json:"config_path" jsonschema:"description:Path to the configuration file"`
	Config      map[string]any `json:"config" jsonschema:"description:Complete project configuration"`
	Success     bool           `json:"success" jsonschema:"description:Whether configuration retrieval was successful"`
	Message     string         `json:"message,omitempty" jsonschema:"description:Additional information or error message"`
}

// UpdateConfigInput represents the input parameters for the ddev_update_config tool
type UpdateConfigInput struct {
	Name         string         `json:"name,omitempty" jsonschema:"description:Project name (uses active project if omitted)"`
	AppRoot      string         `json:"approot,omitempty" jsonschema:"description:Absolute path to project root (overrides name if set)"`
	Config       map[string]any `json:"config" jsonschema:"description:Configuration changes to apply"`
	CreateBackup bool           `json:"create_backup,omitempty" jsonschema:"description:Create backup before updating (default: true)"`
	ValidateOnly bool           `json:"validate_only,omitempty" jsonschema:"description:Only validate changes without applying them"`
}

// UpdateConfigOutput represents the structured output for configuration updates
type UpdateConfigOutput struct {
	ProjectName string   `json:"project_name" jsonschema:"description:Name of the project"`
	ConfigPath  string   `json:"config_path" jsonschema:"description:Path to the configuration file"`
	BackupPath  string   `json:"backup_path,omitempty" jsonschema:"description:Path to the configuration backup file"`
	Applied     bool     `json:"applied" jsonschema:"description:Whether configuration changes were applied"`
	Validated   bool     `json:"validated" jsonschema:"description:Whether configuration passed validation"`
	Success     bool     `json:"success" jsonschema:"description:Whether the operation was successful"`
	Message     string   `json:"message,omitempty" jsonschema:"description:Additional information or error message"`
	Errors      []string `json:"errors,omitempty" jsonschema:"description:Validation or application errors"`
	Warnings    []string `json:"warnings,omitempty" jsonschema:"description:Configuration warnings"`
}

// Global security manager reference for tool handlers
var globalSecurityManager SecurityManager

// registerDDEVTools registers all DDEV MCP tools with the provided server
func registerDDEVTools(server *mcp.Server, security SecurityManager) error {
	// Store security manager globally so handlers can access it
	globalSecurityManager = security
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

	// Register configuration management tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ddev_get_config",
		Description: "Read DDEV project configuration",
	}, handleGetConfig)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ddev_update_config",
		Description: "Update DDEV project configuration with validation and backup",
	}, handleUpdateConfig)

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

	// Build detailed text content for MCP clients
	var textContent strings.Builder
	textContent.WriteString(fmt.Sprintf("Found %d DDEV projects:\n\n", len(projectList)))

	for i, project := range projectList {
		textContent.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, project.Name, project.Type))
		textContent.WriteString(fmt.Sprintf("   Status: %s\n", project.Status))
		textContent.WriteString(fmt.Sprintf("   Location: %s\n", project.ShortRoot))
		if project.PrimaryURL != "" {
			textContent.WriteString(fmt.Sprintf("   URL: %s\n", project.PrimaryURL))
		}
		if i < len(projectList)-1 {
			textContent.WriteString("\n")
		}
	}

	// Return MCP-compatible result with detailed content
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: textContent.String(),
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
	// Security check
	toolName := "ddev_start_project"
	args := map[string]any{
		"name":       input.Name,
		"approot":    input.AppRoot,
		"skip_hooks": input.SkipHooks,
	}

	// Check permissions before proceeding
	if globalSecurityManager != nil {
		if permErr := globalSecurityManager.CheckPermission(toolName, args); permErr != nil {
			output := ProjectLifecycleOutput{
				ProjectName: input.Name,
				Operation:   "start",
				Success:     false,
				Message:     fmt.Sprintf("Permission denied: %v", permErr),
			}
			globalSecurityManager.LogOperation(toolName, args, output, permErr)
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	}

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

	// Log successful operation
	if globalSecurityManager != nil {
		globalSecurityManager.LogOperation(toolName, args, output, nil)
	}

	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Started project %s successfully", app.GetName())}}}, output, nil
}

// handleStopProject handles the ddev_stop_project MCP tool
func handleStopProject(ctx context.Context, _ *mcp.CallToolRequest, input ProjectLifecycleInput) (*mcp.CallToolResult, ProjectLifecycleOutput, error) {
	// Security check
	toolName := "ddev_stop_project"
	args := map[string]any{
		"name":       input.Name,
		"approot":    input.AppRoot,
		"skip_hooks": input.SkipHooks,
	}

	// Check permissions before proceeding
	if globalSecurityManager != nil {
		if permErr := globalSecurityManager.CheckPermission(toolName, args); permErr != nil {
			output := ProjectLifecycleOutput{
				ProjectName: input.Name,
				Operation:   "stop",
				Success:     false,
				Message:     fmt.Sprintf("Permission denied: %v", permErr),
			}
			globalSecurityManager.LogOperation(toolName, args, output, permErr)
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	}

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

	// Log successful operation
	if globalSecurityManager != nil {
		globalSecurityManager.LogOperation(toolName, args, output, nil)
	}

	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Stopped project %s successfully", app.GetName())}}}, output, nil
}

// handleRestartProject handles the ddev_restart_project MCP tool
func handleRestartProject(ctx context.Context, _ *mcp.CallToolRequest, input ProjectLifecycleInput) (*mcp.CallToolResult, ProjectLifecycleOutput, error) {
	// Security check
	toolName := "ddev_restart_project"
	args := map[string]any{
		"name":       input.Name,
		"approot":    input.AppRoot,
		"skip_hooks": input.SkipHooks,
	}

	// Check permissions before proceeding
	if globalSecurityManager != nil {
		if permErr := globalSecurityManager.CheckPermission(toolName, args); permErr != nil {
			output := ProjectLifecycleOutput{
				ProjectName: input.Name,
				Operation:   "restart",
				Success:     false,
				Message:     fmt.Sprintf("Permission denied: %v", permErr),
			}
			globalSecurityManager.LogOperation(toolName, args, output, permErr)
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	}

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

	// Log successful operation
	if globalSecurityManager != nil {
		globalSecurityManager.LogOperation(toolName, args, output, nil)
	}

	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Restarted project %s successfully", app.GetName())}}}, output, nil
}

// handleExecCommand handles the ddev_exec_command MCP tool
func handleExecCommand(ctx context.Context, _ *mcp.CallToolRequest, input ExecCommandInput) (*mcp.CallToolResult, ExecCommandOutput, error) {
	// Security check - this is a destructive operation
	toolName := "ddev_exec_command"
	args := map[string]any{
		"name":    input.Name,
		"approot": input.AppRoot,
		"service": input.Service,
		"command": input.Command,
		"dir":     input.Dir,
	}

	// Check permissions before proceeding
	if globalSecurityManager != nil {
		if permErr := globalSecurityManager.CheckPermission(toolName, args); permErr != nil {
			output := ExecCommandOutput{
				ProjectName: input.Name,
				Service:     input.Service,
				Command:     input.Command,
				Success:     false,
				Message:     fmt.Sprintf("Permission denied: %v", permErr),
			}
			globalSecurityManager.LogOperation(toolName, args, output, permErr)
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}

		// Check if approval is required for this operation
		if globalSecurityManager.RequiresApproval(toolName, args) {
			description := fmt.Sprintf("Execute command '%s' in %s service", input.Command, input.Service)
			if approvalErr := globalSecurityManager.RequestApproval(toolName, args, description); approvalErr != nil {
				output := ExecCommandOutput{
					ProjectName: input.Name,
					Service:     input.Service,
					Command:     input.Command,
					Success:     false,
					Message:     fmt.Sprintf("Approval required: %v", approvalErr),
				}
				globalSecurityManager.LogOperation(toolName, args, output, approvalErr)
				return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
			}
		}
	}

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

	// Log successful operation
	if globalSecurityManager != nil {
		globalSecurityManager.LogOperation(toolName, args, output, nil)
	}

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

// handleGetConfig handles the ddev_get_config MCP tool
func handleGetConfig(ctx context.Context, _ *mcp.CallToolRequest, input GetConfigInput) (*mcp.CallToolResult, GetConfigOutput, error) {
	var app *ddevapp.DdevApp
	var err error

	// Select project by approot > name > current directory
	switch {
	case input.AppRoot != "":
		app, err = ddevapp.NewApp(input.AppRoot, true)
		if err != nil {
			output := GetConfigOutput{
				ProjectName: input.AppRoot,
				Success:     false,
				Message:     fmt.Sprintf("Failed to load app at approot %s: %v", input.AppRoot, err),
			}
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	case input.Name != "":
		app, err = ddevapp.GetActiveApp(input.Name)
		if err != nil {
			output := GetConfigOutput{
				ProjectName: input.Name,
				Success:     false,
				Message:     fmt.Sprintf("Failed to find active app %s: %v", input.Name, err),
			}
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	default:
		app, err = ddevapp.GetActiveApp("")
		if err != nil {
			output := GetConfigOutput{
				ProjectName: "current directory",
				Success:     false,
				Message:     fmt.Sprintf("Failed to find active app from current directory: %v", err),
			}
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	}

	// Get the full configuration by describing the project
	desc, err := app.Describe(false)
	if err != nil {
		output := GetConfigOutput{
			ProjectName: app.GetName(),
			ConfigPath:  app.ConfigPath,
			Success:     false,
			Message:     fmt.Sprintf("Failed to get project configuration: %v", err),
		}
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
	}

	output := GetConfigOutput{
		ProjectName: app.GetName(),
		ConfigPath:  app.ConfigPath,
		Config:      desc,
		Success:     true,
		Message:     "Configuration retrieved successfully",
	}

	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Retrieved configuration for project %s from %s", app.GetName(), app.ConfigPath)}}}, output, nil
}

// handleUpdateConfig handles the ddev_update_config MCP tool
func handleUpdateConfig(ctx context.Context, _ *mcp.CallToolRequest, input UpdateConfigInput) (*mcp.CallToolResult, UpdateConfigOutput, error) {
	// Security check - this is a destructive operation
	toolName := "ddev_update_config"
	args := map[string]any{
		"name":          input.Name,
		"approot":       input.AppRoot,
		"config":        input.Config,
		"create_backup": input.CreateBackup,
		"validate_only": input.ValidateOnly,
	}

	// Check permissions before proceeding
	if globalSecurityManager != nil {
		if permErr := globalSecurityManager.CheckPermission(toolName, args); permErr != nil {
			output := UpdateConfigOutput{
				ProjectName: input.Name,
				Success:     false,
				Message:     fmt.Sprintf("Permission denied: %v", permErr),
			}
			globalSecurityManager.LogOperation(toolName, args, output, permErr)
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}

		// Check if approval is required for this operation
		if globalSecurityManager.RequiresApproval(toolName, args) {
			description := "Update project configuration with validation and backup"
			if approvalErr := globalSecurityManager.RequestApproval(toolName, args, description); approvalErr != nil {
				output := UpdateConfigOutput{
					ProjectName: input.Name,
					Success:     false,
					Message:     fmt.Sprintf("Approval required: %v", approvalErr),
				}
				globalSecurityManager.LogOperation(toolName, args, output, approvalErr)
				return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
			}
		}
	}

	var app *ddevapp.DdevApp
	var err error

	// Select project by approot > name > current directory
	switch {
	case input.AppRoot != "":
		app, err = ddevapp.NewApp(input.AppRoot, true)
		if err != nil {
			output := UpdateConfigOutput{
				ProjectName: input.AppRoot,
				Success:     false,
				Message:     fmt.Sprintf("Failed to load app at approot %s: %v", input.AppRoot, err),
			}
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	case input.Name != "":
		app, err = ddevapp.GetActiveApp(input.Name)
		if err != nil {
			output := UpdateConfigOutput{
				ProjectName: input.Name,
				Success:     false,
				Message:     fmt.Sprintf("Failed to find active app %s: %v", input.Name, err),
			}
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	default:
		app, err = ddevapp.GetActiveApp("")
		if err != nil {
			output := UpdateConfigOutput{
				ProjectName: "current directory",
				Success:     false,
				Message:     fmt.Sprintf("Failed to find active app from current directory: %v", err),
			}
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
	}

	output := UpdateConfigOutput{
		ProjectName: app.GetName(),
		ConfigPath:  app.ConfigPath,
		Applied:     false,
		Validated:   false,
		Success:     false,
	}

	// Create backup by default unless explicitly disabled
	createBackup := input.CreateBackup
	if !input.ValidateOnly && createBackup {
		backupPath, backupErr := createConfigBackup(app.ConfigPath)
		if backupErr != nil {
			output.Message = fmt.Sprintf("Failed to create configuration backup: %v", backupErr)
			output.Errors = []string{backupErr.Error()}
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
		}
		output.BackupPath = backupPath
	}

	// Apply configuration changes to a copy for validation
	appCopy := *app
	updateErr := applyConfigChanges(&appCopy, input.Config)
	if updateErr != nil {
		output.Message = fmt.Sprintf("Failed to apply configuration changes: %v", updateErr)
		output.Errors = []string{updateErr.Error()}
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
	}

	// Validate the updated configuration
	validationErr := appCopy.ValidateConfig()
	if validationErr != nil {
		output.Message = fmt.Sprintf("Configuration validation failed: %v", validationErr)
		output.Errors = []string{validationErr.Error()}
		output.Validated = false
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
	}

	output.Validated = true

	// If validate only, return success without applying
	if input.ValidateOnly {
		output.Success = true
		output.Message = "Configuration validation successful - no changes applied"
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
	}

	// Apply changes to the actual app and write config
	updateErr = applyConfigChanges(app, input.Config)
	if updateErr != nil {
		output.Message = fmt.Sprintf("Failed to apply configuration changes: %v", updateErr)
		output.Errors = []string{updateErr.Error()}
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
	}

	// Write the updated configuration
	writeErr := app.WriteConfig()
	if writeErr != nil {
		output.Message = fmt.Sprintf("Failed to write configuration: %v", writeErr)
		output.Errors = []string{writeErr.Error()}
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: output.Message}}}, output, nil
	}

	output.Applied = true
	output.Success = true
	output.Message = "Configuration updated successfully"

	// Log successful operation
	if globalSecurityManager != nil {
		globalSecurityManager.LogOperation(toolName, args, output, nil)
	}

	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Updated configuration for project %s", app.GetName())}}}, output, nil
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

// checkPermissionAndLog performs security checks and logging for MCP tool operations
func checkPermissionAndLog(toolName string, args map[string]any, result any, err error) error {
	if globalSecurityManager == nil {
		return fmt.Errorf("security manager not initialized")
	}

	// Check permission before operation
	if permErr := globalSecurityManager.CheckPermission(toolName, args); permErr != nil {
		globalSecurityManager.LogOperation(toolName, args, nil, permErr)
		return permErr
	}

	// Log operation after completion
	globalSecurityManager.LogOperation(toolName, args, result, err)
	return nil
}

// createConfigBackup creates a backup of the configuration file
func createConfigBackup(configPath string) (string, error) {
	if configPath == "" {
		return "", fmt.Errorf("config path is empty")
	}

	// Create backup with timestamp
	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.backup.%s", configPath, timestamp)

	// Read original config
	originalData, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read original config: %v", err)
	}

	// Write backup
	err = os.WriteFile(backupPath, originalData, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write backup: %v", err)
	}

	return backupPath, nil
}

// applyConfigChanges applies configuration changes to a DdevApp
func applyConfigChanges(app *ddevapp.DdevApp, changes map[string]any) error {
	for key, value := range changes {
		err := setConfigField(app, key, value)
		if err != nil {
			return fmt.Errorf("failed to set field %s: %v", key, err)
		}
	}
	return nil
}

// setConfigField sets a specific configuration field on the DdevApp
func setConfigField(app *ddevapp.DdevApp, fieldName string, value any) error {
	switch fieldName {
	case "name":
		if v, ok := value.(string); ok {
			app.Name = v
		} else {
			return fmt.Errorf("name must be a string")
		}
	case "type":
		if v, ok := value.(string); ok {
			app.Type = v
		} else {
			return fmt.Errorf("type must be a string")
		}
	case "docroot":
		if v, ok := value.(string); ok {
			app.Docroot = v
		} else {
			return fmt.Errorf("docroot must be a string")
		}
	case "php_version":
		if v, ok := value.(string); ok {
			app.PHPVersion = v
		} else {
			return fmt.Errorf("php_version must be a string")
		}
	case "webserver_type":
		if v, ok := value.(string); ok {
			app.WebserverType = v
		} else {
			return fmt.Errorf("webserver_type must be a string")
		}
	case "nodejs_version":
		if v, ok := value.(string); ok {
			app.NodeJSVersion = v
		} else {
			return fmt.Errorf("nodejs_version must be a string")
		}
	case "composer_version":
		if v, ok := value.(string); ok {
			app.ComposerVersion = v
		} else {
			return fmt.Errorf("composer_version must be a string")
		}
	case "mariadb_version":
		if v, ok := value.(string); ok {
			app.MariaDBVersion = v
		} else {
			return fmt.Errorf("mariadb_version must be a string")
		}
	case "mysql_version":
		if v, ok := value.(string); ok {
			app.MySQLVersion = v
		} else {
			return fmt.Errorf("mysql_version must be a string")
		}
	case "xdebug_enabled":
		if v, ok := value.(bool); ok {
			app.XdebugEnabled = v
		} else {
			return fmt.Errorf("xdebug_enabled must be a boolean")
		}
	case "bind_all_interfaces":
		if v, ok := value.(bool); ok {
			app.BindAllInterfaces = v
		} else {
			return fmt.Errorf("bind_all_interfaces must be a boolean")
		}
	case "additional_hostnames":
		if v, ok := value.([]string); ok {
			app.AdditionalHostnames = v
		} else if v, ok := value.([]interface{}); ok {
			// Convert []interface{} to []string
			hostnames := make([]string, len(v))
			for i, item := range v {
				if s, ok := item.(string); ok {
					hostnames[i] = s
				} else {
					return fmt.Errorf("additional_hostnames must be an array of strings")
				}
			}
			app.AdditionalHostnames = hostnames
		} else {
			return fmt.Errorf("additional_hostnames must be an array of strings")
		}
	case "additional_fqdns":
		if v, ok := value.([]string); ok {
			app.AdditionalFQDNs = v
		} else if v, ok := value.([]interface{}); ok {
			// Convert []interface{} to []string
			fqdns := make([]string, len(v))
			for i, item := range v {
				if s, ok := item.(string); ok {
					fqdns[i] = s
				} else {
					return fmt.Errorf("additional_fqdns must be an array of strings")
				}
			}
			app.AdditionalFQDNs = fqdns
		} else {
			return fmt.Errorf("additional_fqdns must be an array of strings")
		}
	default:
		return fmt.Errorf("unsupported configuration field: %s", fieldName)
	}
	return nil
}
