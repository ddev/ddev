# Manual Testing: DDEV MCP Server

This page documents quick, reproducible ways to manually test the DDEV MCP server without relying on an IDE assistant. It covers a tiny Go client included in this repo and the generic MCP Inspector.

## Prerequisites

- DDEV built locally (`make`), which outputs `./.gotmp/bin/darwin_arm64/ddev` (on macOS ARM) and `ddev-hostname`.
- Docker running locally (DDEV validates Docker on startup).
- Optional: Node.js (for MCP Inspector).

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

## IDE Integration (Claude Code)

- `.mcp.json` is configured to launch the local binary: `./.gotmp/bin/darwin_arm64/ddev mcp start`.
- `.claude/settings.local.json` has `"ddev"` enabled in `enabledMcpjsonServers`.
- In Claude Code, select the `ddev` MCP server and call the tools listed above.

## Implementation Notes

- Transport: stdio is implemented; HTTP transport is a stub and not for testing.
- Stdout vs stderr: The MCP server writes human-readable status to stderr to keep stdout as clean JSON-RPC for the transport.
- Docker: DDEV probes Docker during startup; ensure Docker Desktop or Docker Engine is running.

## Troubleshooting

- Inspector doesn’t list tools: verify the Command path and args, and that Docker is running.
- EOF on connect with the tiny client: usually indicates the server exited early (often Docker not available).
- No projects returned: confirm DDEV projects exist on this machine and are active when using `active_only: true`.
