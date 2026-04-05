# Getting Started with the REST Client

The REST client provides synchronous access to all SignalWire APIs using standard HTTP requests. No WebSocket connection required.

## Installation

```bash
go get github.com/signalwire/signalwire-go/pkg/rest
```

The only dependency beyond the Go standard library is the `net/http` package (built-in).

## Configuration

You need three things to connect:

| Parameter | Env Var | Description |
|-----------|---------|-------------|
| `project` | `SIGNALWIRE_PROJECT_ID` | Your SignalWire project ID |
| `token` | `SIGNALWIRE_API_TOKEN` | Your SignalWire API token |
| `host` | `SIGNALWIRE_SPACE` | Your space hostname (e.g. `example.signalwire.com`) |

## Minimal Example

```go
package main

import (
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/pkg/rest"
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
	agents, err := client.Fabric.AIAgents.List(nil)
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
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/pkg/rest"
)

func main() {
	client, err := rest.NewRestClient("", "", "")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	agents, _ := client.Fabric.AIAgents.List(nil)
	fmt.Println(agents)
}
```

## CRUD Pattern

Most resources follow the same CRUD pattern:

```go
// List
items, err := client.Fabric.AIAgents.List(nil)

// Create
agent, err := client.Fabric.AIAgents.Create(map[string]any{
	"name":   "Support",
	"prompt": map[string]any{"text": "Be helpful"},
})

// Get by ID
agent, err := client.Fabric.AIAgents.Get("agent-uuid")

// Update
_, err := client.Fabric.AIAgents.Update("agent-uuid", map[string]any{"name": "Updated Name"})

// Delete
err := client.Fabric.AIAgents.Delete("agent-uuid")
```

Fabric resources also support listing addresses:

```go
addresses, err := client.Fabric.AIAgents.ListAddresses("agent-uuid")
```

## Error Handling

```go
import "github.com/signalwire/signalwire-go/pkg/rest"

client, _ := rest.NewRestClient("", "", "")

agent, err := client.Fabric.AIAgents.Get("nonexistent-id")
if err != nil {
	var restErr *rest.SignalWireRestError
	if errors.As(err, &restErr) {
		fmt.Printf("HTTP %d: %v\n", restErr.StatusCode, restErr.Body)
		// HTTP 404: map[error:not found]
	}
}
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
- [Compatibility API](compat.md) -- Twilio-compatible LAML endpoints
- [All Namespaces](namespaces.md) -- phone numbers, video, datasphere, and more
