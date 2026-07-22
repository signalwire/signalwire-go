# Getting Started with the REST Client

The REST client provides synchronous access to all SignalWire APIs using standard HTTP requests. No WebSocket connection required.

## Installation

```bash
go get github.com/signalwire/signalwire-go/v3/pkg/rest
```

The only dependency beyond the Go standard library is the `net/http` package (built-in).

## Configuration

You need three things to connect:

| Parameter | Env Var | Description |
|-----------|---------|-------------|
| `project` | `SIGNALWIRE_PROJECT_ID` | Your SignalWire project ID |
| `token` | `SIGNALWIRE_API_TOKEN` | Your SignalWire API token |
| `space` | `SIGNALWIRE_SPACE` | Your space hostname (e.g. `example.signalwire.com`) |

### Overriding the API endpoint

By default the client targets `https://<space>`. Two environment variables let you
point it elsewhere without changing code:

| Env Var | Description |
|---------|-------------|
| `SIGNALWIRE_REST_BASE_URL` | Overrides the base URL entirely — set it to reach a staging endpoint, a proxy, or a loopback test fixture (e.g. `http://127.0.0.1:8933`). When unset the base URL is `https://<space>`. |
| `SIGNALWIRE_REST_CA_FILE` | Path to a PEM CA bundle to trust for HTTPS. Use it when the endpoint presents a certificate signed by a private CA (required on macOS, where Go's system trust store ignores `SSL_CERT_FILE`). Standard `HTTP(S)_PROXY` / `NO_PROXY` proxy support is preserved when this is set. |

```bash
# Point the client at a local mock or staging endpoint:
export SIGNALWIRE_REST_BASE_URL=http://127.0.0.1:8933
```

<!-- snippet-setup -->
```go
import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/v3/pkg/rest"
)

// Shared context assumed by the fragments below: a constructed REST client.
var client, err = rest.NewRestClient("project", "token", "space")

var (
	_ = client
	_ = err
	_ = errors.New
	_ = fmt.Sprint
	_ = os.Getenv
	_ = context.Background
)
```

## Minimal Example

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/v3/pkg/rest"
)

func main() {
	client, err := rest.NewRestClient(
		os.Getenv("SIGNALWIRE_PROJECT_ID"),
		os.Getenv("SIGNALWIRE_API_TOKEN"),
		os.Getenv("SIGNALWIRE_SPACE"),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// List your AI agents
	agents, err := client.Fabric.AIAgents.List(context.Background(), nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(agents)
}
```

Or use environment variables and pass empty strings:

```bash
export SIGNALWIRE_PROJECT_ID=your-project-id
export SIGNALWIRE_API_TOKEN=your-api-token
export SIGNALWIRE_SPACE=example.signalwire.com
```

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/v3/pkg/rest"
)

func main() {
	client, err := rest.NewRestClient("", "", "")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	agents, _ := client.Fabric.AIAgents.List(context.Background(), nil)
	fmt.Println(agents)
}
```

## CRUD Pattern

Most resources follow the same CRUD pattern:

```go
// List
items, err := client.Fabric.AIAgents.List(context.Background(), nil)

// Create
agent, err := client.Fabric.AIAgents.Create(context.Background(), map[string]any{
	"name":   "Support",
	"prompt": map[string]any{"text": "Be helpful"},
})

// Get by ID
agent, err = client.Fabric.AIAgents.Get(context.Background(), "agent-uuid")

// Update
_, err = client.Fabric.AIAgents.Update(context.Background(), "agent-uuid", map[string]any{"name": "Updated Name"})

// Delete
_, err = client.Fabric.AIAgents.Delete(context.Background(), "agent-uuid")

_, _ = items, agent
```

Fabric resources also support listing addresses:

```go
addresses, err := client.Fabric.AIAgents.ListAddresses(context.Background(), "agent-uuid", nil)
_ = addresses
```

## Error Handling

```go
client, _ = rest.NewRestClient("", "", "")

agent, err := client.Fabric.AIAgents.Get(context.Background(), "nonexistent-id")
if err != nil {
	var restErr *rest.SignalWireRestError
	if errors.As(err, &restErr) {
		fmt.Printf("HTTP %d: %s\n", restErr.StatusCode, restErr.Body)
		// HTTP 404: {"error":"not found"}
	}
}
_ = agent
```

## Debug Logging

Set the log level to see HTTP request details:

```bash
export SIGNALWIRE_LOG_LEVEL=debug
```

## Next Steps

- [Client Reference](client-reference.md) -- all namespaces and constructor options
- [Fabric Resources](fabric.md) -- managing AI agents, SWML scripts, and more
- [Calling Commands](calling.md) -- REST-based call control
- [All Namespaces](namespaces.md) -- phone numbers, video, datasphere, and more
