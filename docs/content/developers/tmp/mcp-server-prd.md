# DDEV Model Context Protocol (MCP) Server - Product Requirements Document

## Executive Summary

This PRD defines the requirements for implementing a DDEV Model Context Protocol (MCP) server that enables AI assistants like Claude Code to interact programmatically with DDEV projects. The server will expose DDEV's project management capabilities through a standardized JSON-RPC interface, allowing AI to discover, monitor, and control DDEV environments.

## Background & Motivation

### Problem Statement

Currently, AI assistants working with DDEV projects must:

- Execute CLI commands through Bash and parse human-readable output
- Manually construct complex command sequences for project management
- Lack structured access to project status and configuration data
- Cannot efficiently monitor or respond to project state changes

### Opportunity

The Model Context Protocol (MCP) provides a standardized way for AI assistants to interact with external tools and data sources. By implementing an MCP server for DDEV, we can:

- Provide structured, typed access to DDEV functionality
- Enable more reliable AI-driven development workflows
- Reduce the cognitive load on developers using AI assistants
- Create a foundation for advanced DDEV automation and monitoring

### Success Metrics

- AI assistants can successfully manage DDEV project lifecycle (start/stop/restart)
- Project status and configuration information is accessible through structured API
- Command execution time reduced by 50% compared to CLI parsing
- Developer satisfaction improved through more reliable AI assistance

## Scope & Requirements

### In Scope - Phase 1 (MVP)

- Core MCP server implementation with stdio transport
- Project discovery and listing functionality  
- Project lifecycle management (start/stop/restart)
- Project status and configuration inspection
- Basic security and permission system
- Integration with existing DDEV CLI commands

### In Scope - Phase 2 (Enhanced)

- HTTP transport support for web clients
- Service-level operations (exec, logs)
- Configuration management capabilities
- Add-on management integration
- Advanced permission and approval workflows

### In Scope - Phase 3 (Advanced)  

- Real-time status updates via WebSocket
- Database operation support
- File management capabilities
- Multi-project batch operations
- Performance monitoring and metrics

### Out of Scope

- Web UI for MCP server management
- Authentication/authorization beyond basic permissions
- Plugin system for third-party extensions
- Integration with non-DDEV containerized environments

## Functional Requirements

### FR-1: MCP Server Foundation

**Priority:** P0 (Critical)

The system SHALL implement a standards-compliant MCP server that:

- Uses the official MCP Go SDK (github.com/modelcontextprotocol/go-sdk)
- Supports stdio transport for integration with AI assistants
- Implements proper JSON-RPC 2.0 protocol handling
- Provides server capabilities discovery
- Handles errors gracefully with appropriate error codes

**Acceptance Criteria:**

- MCP server starts via `ddev mcp start` command
- Server responds to MCP initialization handshake
- Tool discovery returns complete list of available operations
- Error responses include proper JSON-RPC error codes

### FR-2: Project Discovery

**Priority:** P0 (Critical)

The system SHALL provide project discovery capabilities that:

- List all DDEV projects with current status
- Filter projects by status (active/inactive) and type
- Return structured JSON data matching `ddev list -j` output
- Include project metadata (location, URLs, type, versions)

**Tool Specification:**

```json
{
  "name": "ddev_list_projects",
  "description": "List all DDEV projects with their current status",
  "inputSchema": {
    "type": "object",
    "properties": {
      "active_only": {"type": "boolean", "default": false},
      "type_filter": {"type": "string", "description": "Filter by project type"}
    }
  }
}
```

**Acceptance Criteria:**

- Returns array of project objects with all fields from `ddev list -j`
- Filtering works correctly for active_only and type_filter parameters
- Performance is equivalent to direct CLI command execution
- Handles cases with no projects gracefully

### FR-3: Project Status Inspection

**Priority:** P0 (Critical)

The system SHALL provide detailed project status capabilities that:

- Return comprehensive project information for specified project
- Include service status, URLs, container details, and configuration
- Provide data equivalent to `ddev describe -j` output
- Handle non-existent projects with appropriate errors

**Tool Specification:**

```json
{
  "name": "ddev_describe_project",
  "description": "Get detailed information about a specific DDEV project",
  "inputSchema": {
    "type": "object",
    "properties": {
      "project_name": {"type": "string", "description": "Name of project to describe"}
    },
    "required": ["project_name"]
  }
}
```

**Acceptance Criteria:**

- Returns complete project description matching `ddev describe -j`
- Includes all service information, URLs, and configuration details
- Provides appropriate error for non-existent projects
- Response time under 2 seconds for typical projects

### FR-4: Project Lifecycle Management

**Priority:** P0 (Critical)

The system SHALL provide project lifecycle management that:

- Start stopped DDEV projects
- Stop running DDEV projects  
- Restart DDEV projects
- Provide status updates during operations
- Handle concurrent operations safely

**Tool Specifications:**

```json
{
  "name": "ddev_start_project",
  "description": "Start a DDEV project",
  "inputSchema": {
    "type": "object",
    "properties": {
      "project_name": {"type": "string"},
      "skip_hooks": {"type": "boolean", "default": false}
    },
    "required": ["project_name"]
  }
}
```

**Acceptance Criteria:**

- Successfully starts stopped projects
- Returns success/failure status with details
- Respects skip_hooks parameter
- Handles projects that are already running
- Concurrent start operations are handled safely

### FR-5: Service Operations

**Priority:** P1 (High)

The system SHALL provide service-level operations that:

- Execute commands within project containers
- Retrieve logs from project services
- Support different services (web, db, etc.)
- Stream logs in real-time when requested

**Tool Specifications:**

```json
{
  "name": "ddev_exec_command",
  "description": "Execute a command in a project container",
  "inputSchema": {
    "type": "object",
    "properties": {
      "project_name": {"type": "string"},
      "service": {"type": "string", "default": "web"},
      "command": {"type": "string"},
      "working_dir": {"type": "string"}
    },
    "required": ["project_name", "command"]
  }
}
```

**Acceptance Criteria:**

- Commands execute in specified service containers
- Return stdout, stderr, and exit codes
- Support working directory specification
- Handle long-running commands appropriately

### FR-6: Configuration Management  

**Priority:** P1 (High)

The system SHALL provide configuration management that:

- Read project configuration from .ddev/config.yaml
- Validate configuration changes before applying
- Support safe configuration updates
- Backup configurations before modifications

**Acceptance Criteria:**

- Configuration reads return structured YAML data
- Updates validate against DDEV configuration schema
- Backup system prevents configuration loss
- Invalid configurations are rejected with clear errors

### FR-7: Security and Permissions

**Priority:** P0 (Critical)

The system SHALL implement security controls that:

- Default to read-only operations
- Require explicit approval for destructive operations
- Support configurable permission levels
- Log all operations for audit purposes

**Permission Levels:**

- **Read-Only (default):** List, describe, logs (view-only)
- **Safe Operations:** Start, stop, restart projects
- **Destructive Operations:** Exec commands, configuration changes

**Acceptance Criteria:**

- Destructive operations blocked by default
- `--allow-writes` flag enables destructive operations
- Approval prompts shown for high-risk operations
- All operations logged with timestamp and user context

## Non-Functional Requirements

### NFR-1: Performance

- Project list operations complete within 1 second
- Project describe operations complete within 2 seconds
- Server startup time under 5 seconds
- Memory usage under 50MB for idle server

### NFR-2: Reliability

- Server handles unexpected shutdowns gracefully
- No data corruption during concurrent operations
- Proper cleanup of resources on server termination
- Error recovery for failed operations

### NFR-3: Compatibility

- Compatible with existing DDEV installations (v1.22+)
- Works with all supported DDEV project types
- Maintains compatibility with DDEV CLI commands
- No conflicts with existing DDEV functionality

### NFR-4: Maintainability

- Code follows existing DDEV style guidelines
- Comprehensive test coverage (>80%)
- Clear documentation for all public APIs
- Minimal dependencies beyond MCP Go SDK

## Technical Specifications

### Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   AI Assistant  │    │   MCP Server     │    │  DDEV Core      │
│   (Claude Code) │◄──►│                  │◄──►│  (pkg/ddevapp)  │
└─────────────────┘    │  ┌─────────────┐ │    └─────────────────┘
                       │  │ Transport   │ │
                       │  │ (stdio/http)│ │
                       │  └─────────────┘ │
                       │  ┌─────────────┐ │
                       │  │ Tool        │ │
                       │  │ Handlers    │ │
                       │  └─────────────┘ │
                       │  ┌─────────────┐ │
                       │  │ Security &  │ │
                       │  │ Permissions │ │
                       │  └─────────────┘ │
                       └──────────────────┘
```

### Implementation Structure

```
cmd/ddev/cmd/
├── mcp.go                   # MCP command definitions and CLI interface
pkg/
├── mcp/
│   ├── server.go           # Core MCP server implementation
│   ├── tools.go            # Tool handler implementations
│   ├── transport.go        # Transport layer (stdio/http)
│   ├── security.go         # Permission and approval system
│   └── types.go            # Type definitions and schemas
```

### Key Components

#### MCP Command Interface

```go
// cmd/ddev/cmd/mcp.go
var MCPCmd = &cobra.Command{
    Use:   "mcp",
    Short: "DDEV Model Context Protocol server",
    Long:  "Start and manage DDEV MCP server for AI assistant integration",
}

var MCPStartCmd = &cobra.Command{
    Use:   "start",
    Short: "Start DDEV MCP server",
    Run:   mcpStartHandler,
}
```

#### Server Implementation

```go
// pkg/mcp/server.go
type DDEVMCPServer struct {
    server     *mcp.Server
    transport  Transport
    security   SecurityManager
    settings   ServerSettings
}

func NewDDEVMCPServer(settings ServerSettings) *DDEVMCPServer {
    server := mcp.NewServer(&mcp.Implementation{
        Name:    "ddev-server",
        Version: versionconstrants.DdevVersion,
    }, nil)
    
    return &DDEVMCPServer{
        server:   server,
        settings: settings,
        security: NewSecurityManager(settings),
    }
}
```

#### Tool Handler Pattern

```go
// pkg/mcp/tools.go
func (s *DDEVMCPServer) handleListProjects(args map[string]any) (any, error) {
    settings := ddevapp.ListCommandSettings{
        ActiveOnly: getBool(args, "active_only", false),
        TypeFilter: getString(args, "type_filter", ""),
    }
    
    // Delegate to existing DDEV functionality
    return ddevapp.ListProjectsJSON(settings)
}
```

### Data Models

#### Project Information

```go
type ProjectInfo struct {
    Name            string            `json:"name"`
    Status          string            `json:"status"`
    StatusDesc      string            `json:"status_desc"`
    AppRoot         string            `json:"approot"`
    ShortRoot       string            `json:"shortroot"`
    Type            string            `json:"type"`
    PrimaryURL      string            `json:"primary_url"`
    HTTPSUrl        string            `json:"httpsurl"`
    HTTPUrl         string            `json:"httpurl"`
    Services        map[string]any    `json:"services,omitempty"`
    DatabaseInfo    map[string]any    `json:"dbinfo,omitempty"`
}
```

#### Operation Result

```go
type OperationResult struct {
    Success   bool              `json:"success"`
    Message   string            `json:"message"`
    Data      map[string]any    `json:"data,omitempty"`
    Errors    []string          `json:"errors,omitempty"`
    Warnings  []string          `json:"warnings,omitempty"`
}
```

## Documentation Requirements

### User Documentation Updates

#### 1. Create MCP Server Usage Guide

- New document: `users/usage/mcp-server.md`
- Target audience: Developers using AI assistants with DDEV
- Content requirements:
    - Introduction to MCP and AI assistant integration
    - Installation and setup instructions
    - Basic usage examples with Claude Code and other AI assistants
    - Configuration options and security considerations
    - Troubleshooting common issues
    - Example workflows and use cases

#### 2. Update Commands Reference

- File: `users/usage/commands.md`
- Add complete documentation for new `ddev mcp` command group:
    - `ddev mcp start` - Start MCP server
    - `ddev mcp stop` - Stop MCP server
    - `ddev mcp status` - Show server status
    - Command flags and options
    - Usage examples and common patterns

#### 3. Update Configuration Guide

- File: `users/configuration/config.md`
- Add new configuration section for MCP server settings:
    - Global config options in `~/.ddev/global_config.yaml`
    - Project-level MCP settings if applicable
    - Security and permission configuration
    - Transport configuration (stdio, HTTP, WebSocket)

#### 4. Update Architecture Documentation

- File: `users/usage/architecture.md`
- Add section describing MCP server integration
- Explain how MCP server fits into DDEV's architecture
- Document security model and permission boundaries

### Developer Documentation Updates

#### 1. Create Developer Integration Guide

- New document: `developers/mcp-integration.md`
- Content requirements:
    - Technical architecture overview
    - MCP tool implementation patterns
    - Security and permission system
    - Testing strategies for MCP tools
    - Contributing new MCP tools
    - Integration with existing DDEV functionality

#### 2. Update Developer Index

- File: `developers/index.md`
- Add reference to MCP server development
- Link to integration guide and PRD

### Navigation Updates

**mkdocs.yml changes required:**

```yaml
# Under Usage section
- users/usage/mcp-server.md

# Under Development section  
- developers/mcp-integration.md
```

### Content Standards

All documentation must:

- Follow DDEV writing style guide
- Include practical examples with code snippets
- Provide troubleshooting sections
- Use consistent terminology and formatting
- Include appropriate cross-references
- Pass all Markdown linting requirements

## Testing Strategy

### Unit Tests

- Tool handler logic validation
- Security permission checks
- Data transformation accuracy
- Error handling completeness

### Integration Tests  

- End-to-end MCP server functionality
- DDEV command integration
- Transport layer reliability
- Concurrent operation safety

### AI Assistant Tests

- Claude Code integration scenarios
- Common workflow validation
- Error recovery testing
- Performance under AI load

### Security Tests

- Permission boundary validation
- Approval workflow testing
- Audit logging verification
- Unauthorized access prevention

## Risk Assessment

### Technical Risks

- **MCP Go SDK Stability:** Mitigation - Pin to stable version, contribute fixes upstream
- **DDEV API Changes:** Mitigation - Use stable internal APIs, comprehensive test coverage
- **Performance Impact:** Mitigation - Benchmarking, profiling, resource limits

### Security Risks

- **Unauthorized Access:** Mitigation - Default read-only, explicit approval requirements
- **Resource Exhaustion:** Mitigation - Rate limiting, resource monitoring
- **Data Exposure:** Mitigation - Audit logging, minimal data exposure

### Adoption Risks

- **AI Assistant Compatibility:** Mitigation - Standard MCP compliance, extensive testing
- **User Experience Complexity:** Mitigation - Simple defaults, clear documentation
- **Maintenance Burden:** Mitigation - Clean architecture, comprehensive tests

## Success Criteria

### Functional Success

- [ ] AI assistants can discover and manage DDEV projects
- [ ] All core project operations work reliably
- [ ] Security model prevents unauthorized access
- [ ] Performance meets defined benchmarks

### Quality Success  

- [ ] Test coverage exceeds 80%
- [ ] No critical security vulnerabilities
- [ ] Documentation is complete and accurate
- [ ] Code passes all static analysis checks

### Adoption Success

- [ ] Positive feedback from early adopters
- [ ] Integration with major AI assistants
- [ ] Community contributions and extensions
- [ ] Inclusion in DDEV core distribution

## Appendices

### A. MCP Protocol Reference

- [MCP Specification](https://modelcontextprotocol.io/specification)
- [MCP Go SDK Documentation](https://github.com/modelcontextprotocol/go-sdk)

### B. DDEV Integration Points

- `pkg/ddevapp/ddevapp.go` - Core project management
- `cmd/ddev/cmd/list.go` - Project listing implementation
- `cmd/ddev/cmd/describe.go` - Project description implementation
- `cmd/ddev/cmd/start.go` - Project startup implementation

### C. Example Usage Scenarios

#### Scenario 1: Project Discovery

```
AI: "What DDEV projects do I have?"
→ ddev_list_projects()
→ Returns: List of all projects with status
→ AI presents summary to user
```

#### Scenario 2: Development Workflow

```
AI: "Start my API project and show me its status"
→ ddev_start_project(project_name: "api")
→ Wait for startup completion
→ ddev_describe_project(project_name: "api")  
→ Returns: Full project details with URLs
→ AI provides access URLs to user
```

#### Scenario 3: Troubleshooting

```
AI: "Check the logs for the failing project"
→ ddev_list_projects(active_only: true)
→ Identify projects with issues
→ ddev_logs(project_name: "problematic", service: "web")
→ Returns: Recent log entries
→ AI analyzes logs and suggests fixes
```

This PRD provides the complete specification for implementing a DDEV MCP server that will enable powerful AI-driven development workflows while maintaining security and reliability standards.
