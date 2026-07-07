# MCP Integration

The SDK supports the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) in two ways:

1. **MCP Client** — Connect to external MCP servers and use their tools in your agent
2. **MCP Server** — Expose your agent's tools as an MCP endpoint for other clients

These features are independent and can be used separately or together.

## Adding External MCP Servers

Use `AddMcpServer()` to connect your agent to remote MCP servers. Tools are discovered at call start via the MCP protocol and added to the AI's tool list alongside your `DefineTool` functions.

```go
package main

import "github.com/signalwire/signalwire-go/pkg/agent"

func main() {
    a := agent.NewAgentBase(
        agent.WithName("my-agent"),
        agent.WithRoute("/agent"),
    )

    a.AddMcpServer(agent.MCPServerConfig{
        URL:     "https://mcp.example.com/tools",
        Headers: map[string]string{"Authorization": "Bearer sk-xxx"},
    })

    a.Run()
}
```

### Fields (`agent.MCPServerConfig`)

| Field | Type | Description |
|---|---|---|
| `URL` | string | MCP server HTTP endpoint URL |
| `Headers` | map[string]string | Optional HTTP headers for authentication |
| `Resources` | bool | Fetch resources into `global_data` (default: false) |
| `ResourceVars` | map[string]string | Variables for URI template substitution |

### With Resources

MCP servers can expose read-only data as resources. When enabled, resources are fetched at session start and merged into `global_data`:

```go
a.AddMcpServer(agent.MCPServerConfig{
    URL:          "https://mcp.example.com/crm",
    Headers:      map[string]string{"Authorization": "Bearer sk-xxx"},
    Resources:    true,
    ResourceVars: map[string]string{"caller_id": "${caller_id_number}"},
})
```

Resource data is available in prompts via `${global_data.key}` and included in every webhook call.

### Multiple Servers

```go
a.AddMcpServer(agent.MCPServerConfig{
    URL:     "https://mcp-search.example.com/tools",
    Headers: map[string]string{"Authorization": "Bearer search-key"},
})
a.AddMcpServer(agent.MCPServerConfig{
    URL:     "https://mcp-crm.example.com/tools",
    Headers: map[string]string{"Authorization": "Bearer crm-key"},
})
```

Tools from all servers are merged into one list. If an MCP tool has the same name as a `DefineTool` function, your local function's description is used but execution routes through MCP.

## Exposing Tools as MCP Server

Use `EnableMcpServer()` to add an MCP endpoint at `/mcp` on your agent's server. Any MCP client can connect and use your `DefineTool` functions.

```go
package main

import (
    "github.com/signalwire/signalwire-go/pkg/agent"
    "github.com/signalwire/signalwire-go/pkg/swaig"
)

func main() {
    a := agent.NewAgentBase(
        agent.WithName("my-agent"),
        agent.WithRoute("/agent"),
    )
    a.EnableMcpServer()

    a.DefineTool(agent.ToolDefinition{
        Name:        "get_weather",
        Description: "Get weather for a location",
        Parameters: map[string]any{
            "location": map[string]any{"type": "string", "description": "City name or zip code"},
        },
        Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
            location, _ := args["location"].(string)
            if location == "" {
                location = "unknown"
            }
            return swaig.NewFunctionResult("72F sunny in " + location)
        },
    })

    a.Run()
}
```

The `/mcp` endpoint handles the full MCP protocol:
- `initialize` — protocol version and capability negotiation
- `notifications/initialized` — ready signal
- `tools/list` — returns all `DefineTool` functions in MCP format
- `tools/call` — invokes the handler and returns the result
- `ping` — keepalive

### Connecting from Claude Desktop

Add your agent as an MCP server in Claude Desktop's config:

```json
{
    "mcpServers": {
        "my-agent": {
            "url": "https://your-server.com/agent/mcp"
        }
    }
}
```

Your `DefineTool` functions are now available in Claude Desktop conversations.

## Using Both Together

The two features are independent:

```go
a := agent.NewAgentBase(
    agent.WithName("my-agent"),
    agent.WithRoute("/agent"),
)

// Expose my tools as MCP (for Claude Desktop, other agents)
a.EnableMcpServer()

// Pull in tools from external MCP servers (for voice calls)
a.AddMcpServer(agent.MCPServerConfig{
    URL:       "https://mcp.example.com/crm",
    Headers:   map[string]string{"Authorization": "Bearer sk-xxx"},
    Resources: true,
})

a.DefineTool(agent.ToolDefinition{
    Name:        "transfer_call",
    Description: "Transfer the caller",
    Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
        // This tool is available both as MCP AND as SWAIG webhook
        return swaig.NewFunctionResult("Transferring now.")
    },
})
```

In this setup:
- Voice calls use `transfer_call` via SWAIG webhook plus CRM tools via MCP
- Claude Desktop uses `transfer_call` via MCP endpoint
- The same tool code serves both protocols

### Self-Referencing

If you want your agent's voice calls to also discover tools via MCP instead of webhooks:

```go
a.EnableMcpServer()
a.AddMcpServer(agent.MCPServerConfig{URL: "https://your-server.com/agent/mcp"})
```

This is optional — by default, `EnableMcpServer()` only adds the endpoint without affecting the agent's own SWML output.

## MCP vs SWAIG Webhooks

| | SWAIG Webhooks | MCP Tools |
|---|---|---|
| Response format | JSON with `response`, `action`, `SWML` | Text content only |
| Call control | Can trigger hold, transfer, SWML | Response only |
| Discovery | Defined in SWML config | Auto-discovered via protocol |
| Auth | `web_hook_auth_user/password` | `Headers` map |

MCP tools are best for data retrieval. Use `DefineTool` functions with SWAIG webhooks when you need call control actions.
