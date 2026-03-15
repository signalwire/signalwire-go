//go:build ignore

// Example: mcp_gateway
//
// MCP gateway skill integration. Connects a SignalWire AI agent to MCP
// (Model Context Protocol) servers through the mcp_gateway skill. The
// gateway bridges MCP tools so the agent can use them as SWAIG functions.
//
// Prerequisites:
//   - Start a gateway server: mcp-gateway -c config.json
//   - Set MCP_GATEWAY_URL, MCP_GATEWAY_AUTH_USER, MCP_GATEWAY_AUTH_PASSWORD
package main

import (
	"fmt"
	"os"

	"github.com/signalwire/signalwire-agents-go/pkg/agent"

	// Import builtin skills so their init() functions register them
	_ "github.com/signalwire/signalwire-agents-go/pkg/skills/builtin"
)

func main() {
	gatewayURL := os.Getenv("MCP_GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = "http://localhost:8080"
	}
	authUser := os.Getenv("MCP_GATEWAY_AUTH_USER")
	if authUser == "" {
		authUser = "admin"
	}
	authPassword := os.Getenv("MCP_GATEWAY_AUTH_PASSWORD")
	if authPassword == "" {
		authPassword = "changeme"
	}

	a := agent.NewAgentBase(
		agent.WithName("MCPGatewayAgent"),
		agent.WithRoute("/mcp-gateway"),
		agent.WithPort(3019),
	)

	a.AddLanguage(map[string]any{
		"name":  "English",
		"code":  "en-US",
		"voice": "rime.spore",
	})

	a.PromptAddSection("Role",
		"You are a helpful assistant with access to external tools provided "+
			"through MCP servers. Use the available tools to help users accomplish "+
			"their tasks.",
		nil,
	)

	// Connect to MCP gateway - tools are discovered automatically
	a.AddSkill("mcp_gateway", map[string]any{
		"gateway_url":   gatewayURL,
		"auth_user":     authUser,
		"auth_password": authPassword,
		"services":      []map[string]any{{"name": "todo"}},
	})

	fmt.Println("Starting MCPGatewayAgent on :3019/mcp-gateway ...")
	fmt.Printf("  Gateway URL: %s\n", gatewayURL)

	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
