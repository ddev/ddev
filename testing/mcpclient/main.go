package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Spawn the DDEV MCP server via stdio
	cmd := exec.Command("ddev", "mcp", "start")

	client := mcp.NewClient(&mcp.Implementation{Name: "ddev-mcp-client", Version: "v0.0.1"}, &mcp.ClientOptions{KeepAlive: 10 * time.Second})
	transport := &mcp.CommandTransport{Command: cmd}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		fmt.Printf("connect error: %v\n", err)
		return
	}
	defer session.Close()

	// Optional: list tools to confirm availability
	if tools, err := session.ListTools(ctx, nil); err == nil {
		// no-op, just confirms handshake
		_ = tools
	}

	// Call ddev_list_projects with active_only=true to see only running projects
	params := &mcp.CallToolParams{
		Name:      "ddev_list_projects",
		Arguments: map[string]any{"active_only": true},
	}
	res, err := session.CallTool(ctx, params)
	if err != nil {
		fmt.Printf("call error: %v\n", err)
		return
	}

	// Pretty print the result
	b, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(b))
}
