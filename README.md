<!-- Header -->
<div align="center">
    <a href="https://signalwire.com" target="_blank">
        <img src="https://github.com/user-attachments/assets/0c8ed3b9-8c50-4dc6-9cc4-cc6cd137fd50" width="500" />
    </a>

# SignalWire SDK for Go

_Build AI voice agents, control live calls over WebSocket, and manage every SignalWire resource over REST -- all from one package._

<p align="center">
  <a href="https://developer.signalwire.com/sdks/agents-sdk" target="_blank">Documentation</a> &middot;
  <a href="https://github.com/signalwire/signalwire-docs/issues/new/choose" target="_blank">Report an Issue</a> &middot;
  <a href="https://pkg.go.dev/github.com/signalwire/signalwire-go" target="_blank">pkg.go.dev</a>
</p>

<a href="https://discord.com/invite/F2WNYTNjuF" target="_blank"><img src="https://img.shields.io/badge/Discord%20Community-5865F2" alt="Discord" /></a>
<a href="LICENSE"><img src="https://img.shields.io/badge/MIT-License-blue" alt="MIT License" /></a>
<a href="https://github.com/signalwire/signalwire-go" target="_blank"><img src="https://img.shields.io/github/stars/signalwire/signalwire-go" alt="GitHub Stars" /></a>

</div>

---

## What's in this SDK

| Capability | What it does | Quick link |
|-----------|-------------|------------|
| **AI Agents** | Build voice agents that handle calls autonomously -- the platform runs the AI pipeline, your code defines the persona, tools, and call flow | [Agent Guide](#ai-agents) |
| **RELAY Client** | Control live calls and SMS/MMS in real time over WebSocket -- answer, play, record, collect DTMF, conference, transfer, and more | [RELAY docs](relay/README.md) |
| **REST Client** | Manage SignalWire resources over HTTP -- phone numbers, SIP endpoints, Fabric AI agents, video rooms, messaging, and 18+ API namespaces | [REST docs](rest/README.md) |

```bash
go get github.com/signalwire/signalwire-go
```

---

## AI Agents

Each agent is a self-contained microservice that generates [SWML](docs/swml_service_guide.md) (SignalWire Markup Language) and handles [SWAIG](docs/swaig_reference.md) (SignalWire AI Gateway) tool calls. The SignalWire platform runs the entire AI pipeline (STT, LLM, TTS) -- your agent just defines the behavior.

<!-- include: examples/quickstart_agent/main.go#quickstart -->
```go
package main

import (
	"fmt"
	"time"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("my-agent"),
		agent.WithRoute("/agent"),
	)

	a.AddLanguage(map[string]any{
		"name": "English", "code": "en-US", "voice": "rime.spore",
	})
	a.SetPromptText("You are a helpful assistant.")

	a.DefineTool(agent.ToolDefinition{
		Name:        "get_time",
		Description: "Get the current time",
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			now := time.Now().Format("03:04 PM MST")
			return swaig.NewFunctionResult(fmt.Sprintf("The time is %s", now))
		},
	})

	a.Run()
}
```

Test locally without running a server:

```bash
go run ./cmd/swaig-test --list-tools ./examples/simple_agent/
go run ./cmd/swaig-test --dump-swml ./examples/simple_agent/
go run ./cmd/swaig-test --exec get_time ./examples/simple_agent/
```

### Agent Features

- **Prompt Object Model (POM)** -- structured prompt composition via `PromptAddSection()`
- **SWAIG tools** -- define functions with `DefineTool()` that the AI calls mid-conversation, with native access to the call's media stack
- **Skills system** -- add capabilities with one-liners: `agent.AddSkill("datetime")`
- **Contexts and steps** -- structured multi-step workflows with navigation control
- **DataMap tools** -- tools that execute on SignalWire's servers, calling REST APIs without your own webhook
- **Dynamic configuration** -- per-request agent customization for multi-tenant deployments
- **Call flow control** -- pre-answer, post-answer, and post-AI verb insertion
- **Prefab agents** -- ready-to-use archetypes (InfoGatherer, Survey, FAQ, Receptionist, Concierge)
- **Multi-agent hosting** -- serve multiple agents on a single server with `AgentServer`
- **SIP routing** -- route SIP calls to agents based on usernames
- **Session state** -- persistent conversation state with global data and post-prompt summaries
- **Security** -- auto-generated basic auth, function-specific HMAC tokens, SSL support
- **Serverless** -- deploy to Lambda, Cloud Functions, Azure Functions

### Agent Examples

The [`examples/`](examples/) directory contains 40+ working examples:

| Example | What it demonstrates |
|---------|---------------------|
| [simple_agent](examples/simple_agent/) | POM prompts, SWAIG tools, multilingual support, LLM tuning |
| [contexts_demo](examples/contexts_demo/) | Multi-persona workflow with context switching and step navigation |
| [datamap_demo](examples/datamap_demo/) | Server-side API tools without webhooks |
| [skills_demo](examples/skills_demo/) | Loading built-in skills (datetime, math) |
| [call_flow](examples/call_flow/) | Call flow verbs, debug events, FunctionResult actions |
| [session_state](examples/session_state/) | OnSummary, global data, post-prompt summaries |
| [multi_agent_server](examples/multi_agent_server/) | Multiple agents on one server |
| [lambda](examples/lambda/) | AWS Lambda deployment with AsRouter() |
| [comprehensive_dynamic](examples/comprehensive_dynamic/) | Per-request dynamic configuration, multi-tenant routing |

See [examples/README.md](examples/README.md) for the full list organized by category.

---

## RELAY Client

Real-time call control and messaging over WebSocket. The RELAY client connects to SignalWire via the Blade protocol and gives you goroutine-safe, imperative control over live phone calls and SMS/MMS.

<!-- include: examples/quickstart_relay/main.go#quickstart -->
```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/signalwire/signalwire-go/pkg/relay"
)

func main() {
	client := relay.NewRelayClient(
		relay.WithProject(os.Getenv("SIGNALWIRE_PROJECT_ID")),
		relay.WithToken(os.Getenv("SIGNALWIRE_API_TOKEN")),
		relay.WithSpace(os.Getenv("SIGNALWIRE_SPACE")),
		relay.WithContexts("default"),
	)

	client.OnCall(func(call *relay.Call) {
		call.Answer()
		action := call.Play([]map[string]any{
			{"type": "tts", "text": "Welcome to SignalWire!"},
		})
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		action.Wait(ctx)
		call.Hangup("")
	})

	fmt.Println("Waiting for inbound calls ...")
	client.Run()
}
```

- 57+ calling methods (play, record, collect, detect, tap, stream, AI, conferencing, and more)
- SMS/MMS messaging with delivery tracking
- Action objects with `Wait()`, `Stop()`, `Pause()`, `Resume()`
- Auto-reconnect with exponential backoff

See the **[RELAY documentation](relay/README.md)** for the full guide, API reference, and examples.

---

## REST Client

Synchronous REST client for managing SignalWire resources and controlling calls over HTTP. No WebSocket required.

<!-- include: examples/quickstart_rest/main.go#quickstart -->
```go
package main

import (
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/pkg/rest"
	"github.com/signalwire/signalwire-go/pkg/rest/namespaces"
)

func main() {
	// Reads from SIGNALWIRE_PROJECT_ID, SIGNALWIRE_API_TOKEN, SIGNALWIRE_SPACE
	client, err := rest.NewRestClient("", "", "")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	client.Fabric.AIAgents.Create(map[string]any{
		"name":   "Support Bot",
		"prompt": map[string]any{"text": "You are helpful."},
	})

	client.Calling.Dial(namespaces.CallingNamespaceDialParams{
		From: "+15559876543",
		To:   "+15551234567",
		Url:  ptr("https://example.com/call-handler"),
	})

	results, _ := client.PhoneNumbers.Search(map[string]string{"area_code": "512"})
	fmt.Println(results)
}

// ptr returns a pointer to v, for setting optional pointer-typed params.
func ptr[T any](v T) *T { return &v }
```

- 20 namespaced API surfaces: Fabric (13 resource types), Calling (37 commands), Video, Datasphere, Phone Numbers, SIP, Queues, Recordings, and more
- Shared `http.Client` for connection pooling
- `map[string]any` returns -- raw JSON, no wrapper objects

See the **[REST documentation](rest/README.md)** for the full guide, API reference, and examples.

---

## Installation

```bash
go get github.com/signalwire/signalwire-go
```

Requires Go 1.25 or later.

## Documentation

Full reference documentation is available at **[developer.signalwire.com/sdks/agents-sdk](https://developer.signalwire.com/sdks/agents-sdk)**.

Guides are also available in the [`docs/`](docs/) directory:

### Getting Started

- [Agent Guide](docs/agent_guide.md) -- creating agents, prompt configuration, dynamic setup
- [Architecture](docs/architecture.md) -- SDK architecture and core concepts
- [SDK Features](docs/sdk_features.md) -- feature overview, SDK vs raw SWML comparison

### Core Features

- [SWAIG Reference](docs/swaig_reference.md) -- function results, actions, post_data lifecycle
- [Contexts and Steps](docs/contexts_guide.md) -- structured workflows, navigation, gather mode
- [DataMap Guide](docs/datamap_guide.md) -- serverless API tools without webhooks
- [LLM Parameters](docs/llm_parameters.md) -- temperature, top_p, barge confidence tuning
- [SWML Service Guide](docs/swml_service_guide.md) -- low-level construction of SWML documents

### Skills and Extensions

- [Skills System](docs/skills_system.md) -- built-in skills and the modular framework
- [Third-Party Skills](docs/third_party_skills.md) -- creating and publishing custom skills
- [MCP Gateway](docs/mcp_gateway_reference.md) -- Model Context Protocol integration

### Deployment

- [CLI Guide](docs/cli_guide.md) -- `swaig-test` command reference
- [Cloud Functions](docs/cloud_functions_guide.md) -- Lambda, Cloud Functions, Azure deployment
- [Configuration](docs/configuration.md) -- environment variables, SSL, proxy setup
- [Security](docs/security.md) -- authentication and security model

### Reference

- [API Reference](docs/api_reference.md) -- complete type and method reference
- [Web Service](docs/web_service.md) -- HTTP server and endpoint details
- [Skills Parameter Schema](docs/skills_parameter_schema.md) -- skill parameter definitions

## Environment Variables

| Variable | Used by | Description |
|----------|---------|-------------|
| `SIGNALWIRE_PROJECT_ID` | RELAY, REST | Project identifier |
| `SIGNALWIRE_API_TOKEN` | RELAY, REST | API token |
| `SIGNALWIRE_SPACE` | RELAY, REST | Space hostname (e.g. `example.signalwire.com`) |
| `SWML_BASIC_AUTH_USER` | Agents | Basic auth username (default: auto-generated) |
| `SWML_BASIC_AUTH_PASSWORD` | Agents | Basic auth password (default: auto-generated) |
| `SWML_PROXY_URL_BASE` | Agents | Base URL when behind a reverse proxy |
| `SWML_SSL_ENABLED` | Agents | Enable HTTPS (`true`, `1`, `yes`) |
| `SWML_SSL_CERT_PATH` | Agents | Path to SSL certificate |
| `SWML_SSL_KEY_PATH` | Agents | Path to SSL private key |
| `SIGNALWIRE_LOG_LEVEL` | All | Logging level (`debug`, `info`, `warn`, `error`) |
| `SIGNALWIRE_LOG_MODE` | All | Set to `off` to suppress all logging |

## Testing, linting, formatting

Format, lint, and test go through three canonical scripts under `scripts/`. They
self-bootstrap their tool environment and run from the module root regardless of
your current directory.

```bash
# Test — go test ./... (optional filter/package passthrough)
bash scripts/run-tests.sh
bash scripts/run-tests.sh ./pkg/agent/...
bash scripts/run-tests.sh -run TestFoo ./pkg/relay/...
bash scripts/run-tests.sh -coverprofile=coverage.out ./...   # raw go test flags pass through

# Lint — go vet + golangci-lint
bash scripts/run-lint.sh

# Format — gofmt (apply) / --check (verify-only)
bash scripts/run-format.sh
bash scripts/run-format.sh --check

# Everything (full gate set)
bash scripts/run-ci.sh
```

## License

MIT -- see [LICENSE](LICENSE) for details.
