# SignalWire AI Agents Go SDK

A Go framework for building, deploying, and managing AI agents as microservices that interact with the [SignalWire](https://signalwire.com) platform.

## Features

- **Agent Framework** — Build AI agents with structured prompts, tools, and skills
- **SWML Generation** — Automatic SWML document creation for the SignalWire AI platform
- **SWAIG Functions** — Define tools the AI can call during conversations
- **DataMap Tools** — Server-side API integrations without webhook infrastructure
- **Contexts & Steps** — Structured multi-step conversation workflows
- **Skills System** — Modular, reusable capabilities (datetime, math, web search, etc.)
- **Prefab Agents** — Ready-to-use agent patterns (surveys, reception, FAQ, etc.)
- **Multi-Agent Hosting** — Run multiple agents on a single server
- **RELAY Client** — Real-time WebSocket-based call control and messaging
- **REST Client** — Full SignalWire REST API access with typed resources
- **Serverless Support** — Deploy to Lambda, Cloud Functions, Azure Functions

## Quick Start

```go
package main

import (
    "github.com/signalwire/signalwire-agents-go/pkg/agent"
    "github.com/signalwire/signalwire-agents-go/pkg/swaig"
)

func main() {
    a := agent.NewAgentBase(agent.WithName("my-agent"))

    a.SetPromptText("You are a helpful assistant.")

    a.DefineTool(swaig.ToolDefinition{
        Name:        "get_time",
        Description: "Get the current time",
        Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
            return swaig.NewFunctionResult("The current time is 3:00 PM")
        },
    })

    a.Run()
}
```

## Installation

```bash
go get github.com/signalwire/signalwire-agents-go
```

## Documentation

See the [docs/](docs/) directory for comprehensive guides:

- [Architecture](docs/architecture.md) — System design and component relationships
- [Agent Guide](docs/agent_guide.md) — Building agents, prompts, tools, and skills
- [SWAIG Reference](docs/swaig_reference.md) — SwaigFunctionResult actions
- [DataMap Guide](docs/datamap_guide.md) — Server-side tools
- [Contexts Guide](docs/contexts_guide.md) — Multi-step workflows
- [Skills System](docs/skills_system.md) — Modular capabilities
- [Security](docs/security.md) — Authentication, tokens, HTTPS

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `3000` |
| `SWML_BASIC_AUTH_USER` | Basic auth username | auto-generated |
| `SWML_BASIC_AUTH_PASSWORD` | Basic auth password | auto-generated |
| `SWML_PROXY_URL_BASE` | Proxy/tunnel base URL | auto-detected |
| `SIGNALWIRE_PROJECT_ID` | Project ID for RELAY/REST | — |
| `SIGNALWIRE_API_TOKEN` | API token for RELAY/REST | — |
| `SIGNALWIRE_SPACE` | Space hostname | — |

## License

Copyright (c) SignalWire. All rights reserved.
