# SignalWire REST Client

Synchronous REST client for managing SignalWire resources, controlling live calls, and interacting with every SignalWire API surface from Go. No WebSocket required -- just standard HTTP requests with automatic connection pooling.

## Quick Start

```go
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/rest"
	"github.com/signalwire/signalwire-go/pkg/rest/namespaces"
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

	// Create an AI agent
	agent, err := client.Fabric.AIAgents.Create(map[string]any{
		"name":   "Support Bot",
		"prompt": map[string]any{"text": "You are a helpful support agent."},
	})
	_ = agent

	// Search for a phone number
	results, err := client.PhoneNumbers.Search(map[string]string{"areacode": "512"})
	_ = results

	// Place a call via REST
	_, err = client.Calling.Dial(namespaces.CallingNamespaceDialParams{
		From: "+15559876543",
		To:   "+15551234567",
		Url:  ptr("https://example.com/call-handler"),
	})
	if err != nil {
		fmt.Println(err)
	}
}

// ptr returns a pointer to v, for setting optional pointer-typed params.
func ptr[T any](v T) *T { return &v }
```

## Features

- Single `RestClient` with namespaced sub-objects for every API
- All 37 calling commands: dial, play, record, collect, detect, tap, stream, AI, transcribe, and more
- Full Fabric API: 13 resource types with CRUD + addresses, tokens, and generic resources
- Datasphere: document management and semantic search
- Video: rooms, sessions, recordings, conferences, tokens, streams
- Phone number management, 10DLC registry, MFA, logs, and more
- Shared `net/http` client with connection pooling across all calls
- Typed generated response structs for typed endpoints; `map[string]any` (raw JSON) for the rest

## Documentation

- [Getting Started](../../rest/docs/getting-started.md) -- installation, configuration, first API call
- [Client Reference](../../rest/docs/client-reference.md) -- RestClient constructor, namespaces, error handling
- [Fabric Resources](../../rest/docs/fabric.md) -- managing AI agents, SWML scripts, subscribers, call flows, and more
- [Calling Commands](../../rest/docs/calling.md) -- REST-based call control (dial, play, record, collect, AI, etc.)
- [All Namespaces](../../rest/docs/namespaces.md) -- phone numbers, video, datasphere, logs, registry, and more

## Examples

- [rest_demo](../../examples/rest_demo/) -- create an AI agent, assign a phone number, and place a test call
- [datasphere](../../examples/datasphere/) -- upload a document and run a semantic search

## Environment Variables

| Variable | Description |
|----------|-------------|
| `SIGNALWIRE_PROJECT_ID` | Project ID for authentication |
| `SIGNALWIRE_API_TOKEN` | API token for authentication |
| `SIGNALWIRE_SPACE` | Space hostname (e.g. `example.signalwire.com`) |

## Package Structure

```
github.com/signalwire/signalwire-go/pkg/rest/
    client.go                 # HTTPClient, SignalWireRestError
    rest_client.go            # RestClient -- namespace wiring, env var resolution
    rest_tree_generated.go    # generated top-level namespace fields
    namespaces/
        common.go                       # Resource, CrudResource, CrudWithAddresses
        fabric_resources_generated.go   # 13 resource types + generic resources + addresses + tokens
        calling_resources_generated.go  # 37 command dispatch methods via single POST
        relay_rest_resources_generated.go # phone numbers, queues, MFA, registry, and more
        video_resources_generated.go    # rooms, sessions, recordings, conferences
        datasphere_resources_generated.go # documents, search, chunks
        ... and more
```
