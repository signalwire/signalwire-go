# MCP to SWAIG Gateway

## Overview

The MCP-SWAIG Gateway bridges Model Context Protocol (MCP) servers with SignalWire AI Gateway (SWAIG) functions, letting SignalWire AI agents call MCP-based tools.

There are two pieces:

1. **The gateway service** — a standalone HTTP server that manages MCP server processes,
   sessions, and protocol translation. It is a separate component (not part of the Go SDK);
   deploy it once and point your agents at it.
2. **The `mcp_gateway` skill** — a built-in skill in the Go SDK
   (`pkg/skills/builtin/mcp_gateway.go`) that connects an agent to a running gateway,
   discovers each service's tools, and registers them as SWAIG functions.

This document covers the SDK-side skill (what you configure in Go) and the wire protocol it
speaks to the gateway service.

## Installation (SDK side)

The `mcp_gateway` skill ships with the SignalWire AI Agents Go SDK. Add the SDK to your
module:

```bash
go get github.com/signalwire/signalwire-go
```

Skills self-register in their `init()` function, so blank-import the built-in skill set to
make `mcp_gateway` available to `AgentBase.AddSkill` by name:

```go
package main

import (
	"github.com/signalwire/signalwire-go/pkg/agent"

	// Import built-in skills so their init() functions register them.
	_ "github.com/signalwire/signalwire-go/pkg/skills/all"
)

func main() {
	a := agent.NewAgentBase(agent.WithName("mcp-agent"))

	a.AddSkill("mcp_gateway", map[string]any{
		"gateway_url":   "https://localhost:8080",
		"auth_user":     "admin",
		"auth_password": "changeme",
		"services":      []map[string]any{{"name": "todo"}},
	})

	_ = a.Run()
}
```

## Architecture

### Components

1. **MCP Gateway Service** (standalone server)
   - HTTP/HTTPS server with Basic or Bearer-token authentication
   - Manages multiple MCP server instances
   - Handles session lifecycle per SignalWire call
   - Translates between SWAIG and MCP protocols

2. **`mcp_gateway` skill** (`pkg/skills/builtin/mcp_gateway.go`)
   - SignalWire skill that connects an agent to the gateway
   - Discovers MCP services, then registers each MCP tool as a SWAIG function named
     `<tool_prefix><service>_<tool>` (default prefix `mcp_`)
   - Registers an internal hangup-hook tool that closes the MCP session when the call ends

## Protocol Flow

```
SignalWire Agent                 Gateway Service              MCP Server
      |                                |                          |
      |---(1) Add Skill--------------->|                          |
      |<--(2) Query Tools--------------|                          |
      |                                |---(3) List Tools-------->|
      |                                |<--(4) Tool List----------|
      |---(5) Call SWAIG Function----->|                          |
      |                                |---(6) Spawn Session----->|
      |                                |---(7) Call MCP Tool----->|
      |                                |<--(8) MCP Response-------|
      |<--(9) SWAIG Response-----------|                          |
      |                                |                          |
      |---(10) Hangup Hook------------>|                          |
      |                                |---(11) Close Session---->|
```

## Message Envelope Format

When the skill calls a tool, it POSTs an envelope to the gateway's
`/services/<name>/call` endpoint:

```json
{
    "tool": "add_todo",
    "arguments": { "text": "Buy milk" },
    "session_id": "call_xyz123",
    "timeout": 300,
    "metadata": {
        "call_id": "call_xyz123"
    }
}
```

The `session_id` is derived from `global_data.mcp_call_id` when present, otherwise from the
SWAIG `call_id`. The `timeout` field carries the configured `session_timeout`.

## Skill Configuration

The skill's `Setup()` reads the following parameters from the `map[string]any` passed to
`AddSkill`:

```go
package main

import (
	"github.com/signalwire/signalwire-go/pkg/agent"

	_ "github.com/signalwire/signalwire-go/pkg/skills/all"
)

func main() {
	a := agent.NewAgentBase(agent.WithName("mcp-agent"))

	a.AddSkill("mcp_gateway", map[string]any{
		"gateway_url":   "https://localhost:8080", // required
		"auth_user":     "admin",                  // basic auth (or use auth_token)
		"auth_password": "changeme",
		// "auth_token": "bearer-token",           // alternative to basic auth
		"services": []map[string]any{
			{"name": "todo"},
			{"name": "calculator"},
		},
		"tool_prefix":     "mcp_", // prefix for SWAIG function names
		"session_timeout": 300,    // session timeout in seconds
		"request_timeout": 30,     // per-request timeout in seconds
		"retry_attempts":  3,      // gateway connection retries
		"verify_ssl":      true,   // SSL certificate verification
	})

	_ = a.Run()
}
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `gateway_url` | string | (required) | URL of the MCP Gateway service. `Setup()` returns `false` if empty. |
| `auth_token` | string | `""` | Bearer token (alternative to basic auth) |
| `auth_user` | string | `""` | Basic auth username |
| `auth_password` | string | `""` | Basic auth password |
| `services` | `[]map[string]any` | `[]` (all) | Services to connect to; each entry is `{"name": "<service>"}`. If empty, the skill queries `/services` and connects to all available services. |
| `tool_prefix` | string | `mcp_` | Prefix for registered SWAIG function names |
| `session_timeout` | integer | 300 | Session timeout in seconds (sent as the envelope `timeout`) |
| `request_timeout` | integer | 30 | Per-request HTTP timeout in seconds |
| `retry_attempts` | integer | 3 | Retry attempts for failed requests |
| `verify_ssl` | boolean | true | Verify SSL certificates (set `false` for self-signed certs) |

Each `services` entry is matched by its `name`; the skill fetches that service's full tool
list from the gateway and registers every tool. If `services` is empty, the skill discovers
all services the gateway exposes.

The example agent in `examples/mcp_gateway/main.go` reads the gateway URL and credentials
from the `MCP_GATEWAY_URL`, `MCP_GATEWAY_AUTH_USER`, and `MCP_GATEWAY_AUTH_PASSWORD`
environment variables before passing them into the skill config.

## API Endpoints (Gateway Service)

These are the endpoints the `mcp_gateway` skill calls on the gateway service.

#### GET /health
Health check endpoint (the skill calls this during `Setup()`; a non-200 makes setup fail).
```bash
curl http://localhost:8080/health
```

#### GET /services
List available MCP services (used when `services` is empty).
```bash
curl -u admin:changeme http://localhost:8080/services
```

#### GET /services/{service_name}/tools
Get the tools for a specific service.
```bash
curl -u admin:changeme http://localhost:8080/services/todo/tools
```

#### POST /services/{service_name}/call
Call a tool on a service.

Using Basic Auth:
```bash
curl -u admin:changeme -X POST http://localhost:8080/services/todo/call \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "add_todo",
    "arguments": {"text": "Test item"},
    "session_id": "test-123",
    "timeout": 300
  }'
```

Using Bearer Token:
```bash
curl -X POST http://localhost:8080/services/todo/call \
  -H "Authorization: Bearer your-token-here" \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "add_todo",
    "arguments": {"text": "Test item"},
    "session_id": "test-123"
  }'
```

#### DELETE /sessions/{session_id}
Close a specific session (the skill calls this from its hangup hook).
```bash
curl -u admin:changeme -X DELETE http://localhost:8080/sessions/test-123
```

## Security

### Authentication
The skill authenticates to the gateway with either:
- **Basic Auth**: `auth_user` + `auth_password`, or
- **Bearer Token**: `auth_token` (sent as `Authorization: Bearer <token>`).

If `auth_token` is set it takes precedence; otherwise basic auth is used when both
`auth_user` and `auth_password` are present.

### SSL Verification
SSL certificate verification is on by default. Set `verify_ssl` to `false` to accept
self-signed certificates (development only).

## Testing with the swaig-test CLI

The Go SDK ships a `swaig-test` CLI for exercising an agent's SWAIG functions without a live
SignalWire call. Run your agent, then point the CLI at it:

```bash
# List the registered MCP tools
go run ./cmd/swaig-test --url http://admin:changeme@localhost:3019/mcp-gateway --list-tools

# Call a tool
go run ./cmd/swaig-test --url http://admin:changeme@localhost:3019/mcp-gateway \
  --exec mcp_todo_add_todo --param text="Buy milk"

# Dump the SWML document
go run ./cmd/swaig-test --url http://admin:changeme@localhost:3019/mcp-gateway --dump-swml
```

Registered function names follow the `<tool_prefix><service>_<tool>` pattern — with the
default `mcp_` prefix, the `add_todo` tool on the `todo` service becomes `mcp_todo_add_todo`.

## Implementation Details

### Session Management

1. **Session Creation**: The first tool call creates a session keyed by `session_id`.
2. **Session Persistence**: Sessions are maintained across multiple tool calls within a call.
3. **Session Cleanup**: The skill's hangup-hook tool issues `DELETE /sessions/{id}` when the call ends.
4. **State Isolation**: Each session gets a separate MCP server instance on the gateway.

### Error Handling

1. **Server errors (5xx)**: The skill retries up to `retry_attempts` times.
2. **Client errors (4xx)**: Returned immediately without retry.
3. **Connection errors**: Retried within the `retry_attempts` budget.
4. **Failures**: Returned to the AI as a `FunctionResult` describing the error.

## Running a Gateway Server

The gateway service itself is a **separate component** — the Go SDK ships only the client
skill, not a server. Stand up a gateway (for example, the reference gateway distributed with
the Python SignalWire AI Agents SDK, or your own implementation of the endpoints above),
then point the skill at it via `gateway_url`. The skill only requires that the server
implement the HTTP contract documented in the API Endpoints and Message Envelope Format
sections.

## Troubleshooting

1. **Gateway health check fails** — Verify `gateway_url` is reachable and that credentials match. The skill's `Setup()` returns `false` (the skill fails to load) if the health check does not return 200.
2. **Authentication failures** — Confirm `auth_user`/`auth_password` (or `auth_token`) match the gateway configuration.
3. **SSL certificate errors** — For self-signed certs, set `verify_ssl` to `false`.
4. **Session persistence issues** — Ensure the gateway keeps the MCP process alive between calls and that the same `session_id` (call_id) is used.

## Examples

- `examples/mcp_gateway/main.go` — an agent that connects to MCP servers through the `mcp_gateway` skill.
