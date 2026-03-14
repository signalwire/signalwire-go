# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Project Overview

This is the SignalWire AI Agents Go SDK - a Go port of the Python/TypeScript SignalWire AI Agents framework. It provides tools for building, deploying, and managing AI agents as microservices that expose HTTP endpoints to interact with the SignalWire platform.

## Development Commands

### Building
```bash
go build ./...
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test -v ./pkg/swml/...
go test -v ./pkg/agent/...

# Run with coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run with race detection
go test -race ./...
```

### Linting
```bash
go vet ./...
```

## Architecture Overview

### Module Layout
```
github.com/signalwire/signalwire-agents-go/
├── pkg/
│   ├── swml/           # SWML document model, builder, schema validation
│   ├── agent/          # AgentBase, AI config, prompts, dynamic config
│   ├── swaig/          # SwaigFunctionResult, tool registry, SWAIG functions
│   ├── datamap/        # DataMap builder for server-side tools
│   ├── contexts/       # ContextBuilder, Context, Step workflows
│   ├── skills/         # SkillBase interface, SkillManager, built-in skills
│   ├── prefabs/        # Pre-built agents (InfoGatherer, Survey, etc.)
│   ├── server/         # AgentServer for multi-agent hosting
│   ├── relay/          # RELAY WebSocket client (Blade/JSON-RPC 2.0)
│   ├── rest/           # REST HTTP client with namespaced resources
│   ├── security/       # SessionManager, auth, tokens
│   └── logging/        # Structured logging system
├── cmd/
│   └── swaig-test/     # CLI tool for testing agents
├── examples/           # Example agents
├── docs/               # Documentation
├── internal/           # Internal packages (schema utils, helpers)
└── tests/              # Integration tests
```

### Key Design Patterns

- **Composition over inheritance**: Go structs compose manager objects (PromptManager, ToolRegistry, etc.)
- **Fluent builder API**: Methods return receiver pointer for chaining
- **Functional options**: Constructors use option functions for configuration
- **Interface-based skills**: SkillBase is an interface, not a base class
- **Context propagation**: `context.Context` threaded through HTTP handlers and async operations
- **Channel-based correlation**: RELAY uses Go channels for async request/response matching

### Core Components
1. **SWML** — Document model and rendering (generates JSON consumed by SignalWire platform)
2. **AgentBase** — Main agent struct with prompt, tool, skill, and config management
3. **SwaigFunctionResult** — Response builder with 40+ call-control actions
4. **DataMap** — Server-side tool definitions without webhook infrastructure
5. **Contexts/Steps** — Structured conversation workflows
6. **Skills** — Modular capability plugins
7. **RELAY** — WebSocket-based real-time call control
8. **REST** — HTTP client for SignalWire API

### Important Notes
- Use `encoding/json` for all serialization
- Thread safety via `sync.RWMutex` for shared state (global data, tool registry)
- RELAY correlation requires 4 mechanisms: JSON-RPC id, call_id, control_id, tag
- HMAC-SHA256 for secure tool tokens (crypto/hmac, crypto/sha256)
- Standard library `net/http` for HTTP server
