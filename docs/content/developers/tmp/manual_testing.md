# Manual Testing: DDEV MCP Server

This page documents quick, reproducible ways to manually test the DDEV MCP server with various MCP clients, including IDE integrations and cloud environments. It covers a tiny Go client included in this repo, the generic MCP Inspector, Claude Code integration, GitHub Codespaces, and VS Code.

## Prerequisites

- DDEV built locally (`make`), which outputs `./.gotmp/bin/darwin_arm64/ddev` (on macOS ARM) and `ddev-hostname`.
- Docker running locally (DDEV validates Docker on startup).
- Optional: Node.js (for MCP Inspector).
- For cloud testing: GitHub account with Codespaces access.

## Option 1: Tiny Go MCP Client

A minimal MCP client is included to exercise the server via stdio and call a tool.

- Location: `testing/mcpclient/main.go`
- Build: `make .gotmp/bin/darwin_arm64/mcpclient` (or appropriate platform)
    - Uses vendored modules; no network access required.
    - Also built automatically with `make` as part of TARGETS.
- Run: `./.gotmp/bin/darwin_arm64/mcpclient`
    - Spawns `ddev mcp start` via stdio and calls the tool `ddev_list_projects` with `{ "active_only": true }`.
    - Prints a JSON response containing the tool result.

**Test Results:**

- ✅ **ddev_list_projects**: Successfully returns list of DDEV projects with status, URLs, and configuration
- ✅ **ddev_describe_project**: Returns detailed project information equivalent to `ddev describe -j`
- ✅ **Error handling**: Proper MCP-compatible error responses for invalid projects
- ✅ **JSON-RPC**: Clean communication over stdio transport

Notes:

- The client is intentionally simple and currently calls only `ddev_list_projects` by default.
- To test `ddev_describe_project`, modify the client to call with `{ "name": "project-name", "short": true }`.

## Option 2: MCP Inspector

MCP Inspector is a generic MCP client you can use to connect and call tools.

1. Start Inspector: `npx @modelcontextprotocol/inspector@latest`
2. Add Server → Command:
    - Command: `./.gotmp/bin/darwin_arm64/ddev`
    - Args: `mcp start`
    - Transport: stdio (default)
    - Env: leave empty
3. Connect. The Tools list should include:
    - `ddev_list_projects`
    - `ddev_describe_project`
4. Example calls:
    - List active: `{ "active_only": true }`
    - Describe by name: `{ "name": "myproject", "short": true }`
    - Describe by path: `{ "approot": "/absolute/path/to/project", "short": false }`

### Testing Destructive Operations (ddev_exec_command)

The `ddev_exec_command` tool requires elevated permissions because it can execute arbitrary commands in containers.

**Enable destructive operations:**

1. Update server Args to: `mcp start --allow-writes --auto-approve ddev_exec_command`
2. Reconnect to the server
3. The Tools list will now include `ddev_exec_command`

**Example exec command calls:**

- List files: `{ "name": "myproject", "command": "ls -la", "service": "web" }`
- Check PHP version: `{ "name": "myproject", "command": "php --version", "service": "web" }`
- Database query: `{ "name": "myproject", "command": "mysql -e 'SHOW DATABASES;'", "service": "db" }`

**Configuration options:**

- **Read-only (default)**: `mcp start`
- **With writes but approval required**: `mcp start --allow-writes`
- **With automatic approval**: `mcp start --allow-writes --auto-approve ddev_exec_command`

## Option 3: Claude Code Integration

Claude Code provides native MCP support for AI-powered DDEV project management.

### Setup

1. **Create `.mcp.json` in project root:**

   ```json
   {
     "mcpServers": {
       "ddev": {
         "command": "./.gotmp/bin/darwin_arm64/ddev",
         "args": ["mcp", "start"],
         "env": {}
       }
     }
   }
   ```

2. **Configure Claude Code settings** (`.claude/settings.local.json`):

   ```json
   {
     "enabledMcpjsonServers": ["ddev"]
   }
   ```

3. **Launch Claude Code** in the DDEV repository directory:

   ```bash
   cd /path/to/ddev && claude
   ```

### Testing with Claude Code

1. **Project Discovery:**
   - Ask: "What DDEV projects do I have?"
   - Expected: Claude uses `ddev_list_projects` tool and summarizes project status

2. **Project Details:**
   - Ask: "Show me details for project [name]"
   - Expected: Claude calls `ddev_describe_project` with project name

3. **Project Management:**
   - Ask: "Start the [project-name] project"
   - Expected: Claude identifies the project needs starting and attempts to use appropriate tools

### Configuration Examples

**Basic Configuration:**

```json
{
  "mcpServers": {
    "ddev": {
      "command": "./.gotmp/bin/darwin_arm64/ddev",
      "args": ["mcp", "start"]
    }
  }
}
```

**Debug Configuration:**

```json
{
  "mcpServers": {
    "ddev": {
      "command": "./.gotmp/bin/darwin_arm64/ddev", 
      "args": ["mcp", "start", "--verbose"],
      "env": {
        "DDEV_DEBUG": "true"
      }
    }
  }
}
```

## Option 4: GitHub Codespaces Integration

GitHub Codespaces provides a cloud-based development environment where DDEV MCP server can be tested.

### Setup in Codespaces

1. **Open DDEV repository in Codespaces:**
   - Navigate to <https://github.com/ddev/ddev>
   - Click "Code" → "Codespaces" → "Create codespace on main"

2. **Build DDEV in Codespaces:**

   ```bash
   make
   ```

3. **Install Docker in Codespace** (if not pre-installed):

   ```bash
   sudo apt-get update
   sudo apt-get install docker.io
   sudo systemctl start docker
   sudo usermod -aG docker $USER
   # Restart terminal session
   ```

4. **Test MCP Server:**

   ```bash
   # Basic functionality test
   ./.gotmp/bin/linux_amd64/mcpclient
   
   # Manual server test
   ./.gotmp/bin/linux_amd64/ddev mcp start --test
   ```

### Claude Code in Codespaces

1. **Configure `.mcp.json` for Linux:**

   ```json
   {
     "mcpServers": {
       "ddev": {
         "command": "./.gotmp/bin/linux_amd64/ddev",
         "args": ["mcp", "start"],
         "env": {}
       }
     }
   }
   ```

2. **Launch Claude Code:**

   ```bash
   # Install Claude Code if not available
   curl -fsSL https://claude.ai/install.sh | sh
   
   # Launch in repository
   claude
   ```

3. **Test MCP Integration:**
   - Ask: "List my DDEV projects"
   - Verify tool calls work correctly in cloud environment

### Codespaces-Specific Testing

**Network Testing:**

```bash
# Test if Docker networking works
docker run hello-world

# Test DDEV project creation
mkdir test-project && cd test-project
../.gotmp/bin/linux_amd64/ddev config --project-type=php
../.gotmp/bin/linux_amd64/ddev start
```

**MCP Server with Projects:**

```bash
# Create test project for MCP testing  
mkdir ~/codespace-test-project && cd ~/codespace-test-project
../.gotmp/bin/linux_amd64/ddev config --project-type=drupal --php-version=8.2
cd ~/workspace/ddev

# Test MCP server can discover the project
./.gotmp/bin/linux_amd64/mcpclient
```

## Option 5: VS Code Integration

VS Code can integrate with MCP servers through extensions and configuration.

### Setup with MCP Extension

1. **Install MCP Extension** (if available):
   - Search for "Model Context Protocol" in VS Code Extensions
   - Or use generic MCP client extensions

2. **Configure settings.json:**

   ```json
   {
     "mcp.servers": {
       "ddev": {
         "command": "./.gotmp/bin/darwin_arm64/ddev",
         "args": ["mcp", "start"],
         "transport": "stdio"
       }
     }
   }
   ```

### VS Code with Terminal Integration

**Manual Testing in VS Code Terminal:**

1. **Open DDEV repository in VS Code**
2. **Open integrated terminal** (Ctrl+`)
3. **Build and test:**

   ```bash
   make
   ./.gotmp/bin/darwin_arm64/mcpclient
   ```

4. **Test with MCP Inspector:**

   ```bash
   npx @modelcontextprotocol/inspector@latest
   # Configure with VS Code terminal path
   ```

### VS Code Tasks Configuration

**Create `.vscode/tasks.json`:**

```json
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "Build DDEV",
            "type": "shell",
            "command": "make",
            "group": "build",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "panel": "new"
            }
        },
        {
            "label": "Test MCP Server",
            "type": "shell", 
            "command": "./.gotmp/bin/darwin_arm64/mcpclient",
            "group": "test",
            "dependsOn": "Build DDEV"
        },
        {
            "label": "Start MCP Server",
            "type": "shell",
            "command": "./.gotmp/bin/darwin_arm64/ddev",
            "args": ["mcp", "start", "--test"],
            "group": "test",
            "dependsOn": "Build DDEV"
        }
    ]
}
```

**Usage:**

- Ctrl+Shift+P → "Tasks: Run Task" → Select task
- Quick testing of MCP server functionality within VS Code

## Implementation Notes

- Transport: stdio is implemented; HTTP transport is a stub and not for testing.
- Stdout vs stderr: The MCP server writes human-readable status to stderr to keep stdout as clean JSON-RPC for the transport.
- Docker: DDEV probes Docker during startup; ensure Docker Desktop or Docker Engine is running.

## Troubleshooting

### General Issues

- **Inspector doesn't list tools**: verify the Command path and args, and that Docker is running.
- **EOF on connect with the tiny client**: usually indicates the server exited early (often Docker not available).
- **No projects returned**: confirm DDEV projects exist on this machine and are active when using `active_only: true`.

### Claude Code Specific

- **MCP server not recognized**: Verify `.mcp.json` is in the project root and Claude Code was launched from the correct directory.
- **Tool calls fail**: Check that the binary path in `.mcp.json` matches your platform (darwin_arm64, linux_amd64, etc.).
- **Permission denied**: Ensure the DDEV binary is executable: `chmod +x ./.gotmp/bin/darwin_arm64/ddev`

### GitHub Codespaces Issues

- **Docker not available**: Codespaces may not have Docker pre-installed. Follow the Docker installation steps above.
- **Binary not found**: Use `linux_amd64` binaries in Codespaces, not `darwin_arm64`.
- **Network restrictions**: Some corporate Codespaces may block Docker operations. Test with `docker run hello-world` first.
- **Path issues**: Use absolute paths in Codespaces: `/workspaces/ddev/.gotmp/bin/linux_amd64/ddev`

### VS Code Integration Issues

- **Extension not found**: MCP extensions for VS Code are still emerging. Use terminal-based testing as fallback.
- **Terminal path issues**: VS Code terminal may use different PATH. Specify full binary paths in tasks.json.
- **Task execution fails**: Ensure tasks.json is valid JSON and dependencies are correctly specified.

### Platform-Specific Issues

**macOS:**

- Use `darwin_arm64` for Apple Silicon Macs
- Use `darwin_amd64` for Intel Macs
- Binary path: `./.gotmp/bin/darwin_arm64/ddev` or `./.gotmp/bin/darwin_amd64/ddev`

**Linux:**

- Use `linux_amd64` for most Linux systems
- Binary path: `./.gotmp/bin/linux_amd64/ddev`
- May need `chmod +x` on binaries after build

**Windows:**

- Use `windows_amd64` or `windows_arm64`
- Binary path: `.\.gotmp\bin\windows_amd64\ddev.exe`
- May need Windows Subsystem for Linux (WSL) for full Docker support

### Debug Mode

Enable verbose logging for troubleshooting:

```bash
# CLI debug mode
DDEV_DEBUG=true ./.gotmp/bin/darwin_arm64/ddev mcp start

# MCP server with verbose output  
./.gotmp/bin/darwin_arm64/ddev mcp start --verbose
```

**Debug configuration for Claude Code:**

```json
{
  "mcpServers": {
    "ddev": {
      "command": "./.gotmp/bin/darwin_arm64/ddev",
      "args": ["mcp", "start", "--verbose"],
      "env": {
        "DDEV_DEBUG": "true"
      }
    }
  }
}
```

### Common Error Messages

**"Transport not configured"**:

- Indicates MCP server startup failed
- Check Docker availability and binary permissions

**"Project not found"**:

- DDEV project doesn't exist or isn't initialized
- Run `ddev list` to verify project names and paths

**"Connection refused"**:

- Server failed to start, check Docker and binary path
- Try `ddev mcp start --test` for debugging

**"JSON-RPC parse error"**:

- Usually indicates version mismatch or corrupted communication
- Rebuild DDEV binary and retry
