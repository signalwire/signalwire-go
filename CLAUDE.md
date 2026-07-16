# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Project Overview

This is the SignalWire AI Agents Go SDK - a Go port of the Python/TypeScript SignalWire AI Agents framework. It provides tools for building, deploying, and managing AI agents as microservices that expose HTTP endpoints to interact with the SignalWire platform.

## Development Commands

Format, lint, and test go through **three canonical scripts** under `scripts/`.
They self-bootstrap their tool environment (resolve `go`, install the pinned
`golangci-lint` if absent) and run from the module root **regardless of your
current directory**, so they work identically for run-ci, an agent, or you from
any CWD. Do not invoke `gofmt` / `go vet` / `golangci-lint` / `go test` directly —
go through these scripts (shared env lives in `scripts/_env.sh`).

### Formatting — `scripts/run-format.sh`
```bash
bash scripts/run-format.sh          # APPLY: gofmt -w (reformat the tree in place)
bash scripts/run-format.sh --check  # VERIFY-ONLY (CI): gofmt -l, non-zero if unformatted
```

### Linting — `scripts/run-lint.sh`
```bash
bash scripts/run-lint.sh            # go vet ./... + golangci-lint (.golangci.yml)
bash scripts/run-lint.sh --fix      # + golangci-lint autofix where supported
```

### Testing — `scripts/run-tests.sh`
```bash
bash scripts/run-tests.sh                                 # go test ./... (full suite)
bash scripts/run-tests.sh ./pkg/swml/...                  # a subset (package path)
bash scripts/run-tests.sh -run TestFoo ./pkg/agent/...    # a filter passthrough
```

### Building
```bash
go build ./...
```

Raw `go test` flags still work when you need them (`-v`, `-cover`,
`-coverprofile=…`, `-race`) — pass them through `scripts/run-tests.sh`, e.g.
`bash scripts/run-tests.sh -race ./...`.

### Full gate set
`bash scripts/run-ci.sh` runs every gate (TEST/FMT/LINT wired to the three
scripts above, plus the drift/surface/coverage gates).

## Architecture Overview

### Module Layout
```
github.com/signalwire/signalwire-go/v3/
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
