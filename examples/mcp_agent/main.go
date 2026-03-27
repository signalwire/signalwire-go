// Example: mcp_agent
//
// Demonstrates both MCP features:
//
// 1. MCP Server: Exposes tools at /mcp so external MCP clients
//    (Claude Desktop, other agents) can discover and invoke them.
//
// 2. MCP Client: Connects to external MCP servers to pull in additional
//    tools for voice calls.
//
// Usage:
//
//	go run examples/mcp_agent/main.go
//
//	Then:
//	- Point a SignalWire phone number at http://your-server:3000/agent
//	- Connect Claude Desktop to http://your-server:3000/agent/mcp
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("MCPAgent"),
		agent.WithRoute("/agent"),
		agent.WithPort(3000),
	)

	// ── MCP Server ──────────────────────────────────────────────
	// Adds a /mcp endpoint that speaks JSON-RPC 2.0 (MCP protocol).
	// Any MCP client can connect and use our tools.
	a.EnableMcpServer()

	// ── MCP Client ──────────────────────────────────────────────
	// Connect to external MCP servers. Tools are discovered at
	// call start and added to the AI's tool list.
	a.AddMcpServer(agent.MCPServerConfig{
		URL:     "https://mcp.example.com/tools",
		Headers: map[string]string{"Authorization": "Bearer sk-your-mcp-api-key"},
	})

	// MCP Client with resources — data is fetched into global_data.
	a.AddMcpServer(agent.MCPServerConfig{
		URL:          "https://mcp.example.com/crm",
		Headers:      map[string]string{"Authorization": "Bearer sk-your-crm-key"},
		Resources:    true,
		ResourceVars: map[string]string{"caller_id": "${caller_id_number}", "tenant": "acme-corp"},
	})

	// ── Agent Configuration ─────────────────────────────────────
	a.PromptAddSection("Role", "You are a helpful customer support agent. "+
		"You have access to the customer's profile via global_data. "+
		"Use the available tools to look up information and assist the caller.", nil)

	a.PromptAddSection("Customer Context",
		"Customer name: ${global_data.customer_name}\n"+
			"Account status: ${global_data.account_status}\n"+
			"If customer data is not available, ask the caller for their name.", nil)

	a.SetParam("attention_timeout", 15000)

	// ── Local Tools ─────────────────────────────────────────────
	// Available both as SWAIG webhooks (voice calls) AND as MCP tools.

	a.DefineTool(agent.ToolDefinition{
		Name:        "get_weather",
		Description: "Get the current weather for a location",
		Parameters: map[string]any{
			"location": map[string]any{
				"type":        "string",
				"description": "City name or zip code",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			location, _ := args["location"].(string)
			if location == "" {
				location = "unknown"
			}
			return swaig.NewFunctionResult(
				fmt.Sprintf("Currently 72F and sunny in %s.", location),
			)
		},
	})

	a.DefineTool(agent.ToolDefinition{
		Name:        "create_ticket",
		Description: "Create a support ticket for the customer",
		Parameters: map[string]any{
			"subject": map[string]any{
				"type":        "string",
				"description": "Ticket subject",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Detailed description of the issue",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			subject, _ := args["subject"].(string)
			if subject == "" {
				subject = "No subject"
			}
			return swaig.NewFunctionResult(
				fmt.Sprintf("Ticket created: '%s'. Reference number: TK-12345.", subject),
			)
		},
	})

	fmt.Println("Starting MCPAgent on :3000/agent ...")
	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
