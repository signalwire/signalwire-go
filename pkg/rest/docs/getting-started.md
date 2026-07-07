# Getting Started with the REST Client

The REST client provides synchronous access to all SignalWire APIs using standard HTTP requests. No WebSocket connection required.

## Installation

The REST client is part of the `signalwire-go` module. Add it to your project with:

```bash
go get github.com/signalwire/signalwire-go
```

It depends only on the Go standard library for HTTP.

## Configuration

You need three things to connect:

| Constructor arg | Env Var | Description |
|-----------------|---------|-------------|
| `project` | `SIGNALWIRE_PROJECT_ID` | Your SignalWire project ID |
| `token` | `SIGNALWIRE_API_TOKEN` | Your SignalWire API token |
| `space` | `SIGNALWIRE_SPACE` | Your space hostname (e.g. `example.signalwire.com`) |

## Minimal Example

```go
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/rest"
)

func main() {
	client, err := rest.NewRestClient(
		"your-project-id",
		"your-api-token",
		"example.signalwire.com",
	)
	if err != nil {
		panic(err)
	}

	// List your AI agents
	agents, err := client.Fabric.AIAgents.List(nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(agents)
}
```

Or use environment variables and pass empty strings so the constructor reads them:

```bash
export SIGNALWIRE_PROJECT_ID=your-project-id
export SIGNALWIRE_API_TOKEN=your-api-token
export SIGNALWIRE_SPACE=example.signalwire.com
```

```go
client, err := rest.NewRestClient("", "", "")
if err != nil {
	panic(err)
}
agents, err := client.Fabric.AIAgents.List(nil)
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
agent, err = client.Fabric.AIAgents.Get("agent-uuid")

// Update
_, err = client.Fabric.AIAgents.Update("agent-uuid", map[string]any{"name": "Updated Name"})

// Delete
_, err = client.Fabric.AIAgents.Delete("agent-uuid")
```

`List` accepts a `map[string]string` of query params (or `nil` for none), and
`Create`/`Update` take a `map[string]any` request body.

Fabric resources also support listing addresses:

```go
addresses, err := client.Fabric.AIAgents.ListAddresses("agent-uuid", nil)
```

## Error Handling

A non-2xx HTTP response is returned as a `*rest.SignalWireRestError`:

```go
import (
	"errors"
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/rest"
)

_, err := client.Fabric.AIAgents.Get("nonexistent-id")
if err != nil {
	var restErr *rest.SignalWireRestError
	if errors.As(err, &restErr) {
		fmt.Printf("HTTP %d: %s\n", restErr.StatusCode, restErr.Body)
		// HTTP 404: {"error": "not found"}
	}
}
```

## Next Steps

- [Client Reference](client-reference.md) -- all namespaces and constructor options
- [Fabric Resources](fabric.md) -- managing AI agents, SWML scripts, and more
- [Calling Commands](calling.md) -- REST-based call control
- [All Namespaces](namespaces.md) -- phone numbers, video, datasphere, and more
