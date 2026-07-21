# SignalWire REST Client

Synchronous REST client for managing SignalWire resources, controlling live calls, and interacting with every SignalWire API surface from Go. No WebSocket required -- just standard HTTP requests with automatic connection pooling.

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/v3/pkg/rest"
	"github.com/signalwire/signalwire-go/v3/pkg/rest/namespaces"
)

func main() {
	// Reads from SIGNALWIRE_PROJECT_ID, SIGNALWIRE_API_TOKEN, SIGNALWIRE_SPACE
	client, err := rest.NewRestClient("", "", "")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Create an AI agent
	agent, _ := client.Fabric.AIAgents.Create(context.Background(), map[string]any{
		"name":   "Support Bot",
		"prompt": map[string]any{"text": "You are a helpful support agent."},
	})

	// Search for a phone number
	results, _ := client.PhoneNumbers.Search(context.Background(), map[string]string{"areacode": "512"})

	// Place a call via REST
	_, _ = client.Calling.Dial(context.Background(), namespaces.CallingNamespaceDialParams{
		From:   "+15559876543",
		To:     "+15551234567",
		Extras: map[string]any{"url": "https://example.com/call-handler"},
	})

	fmt.Println(agent, results)
}
```

## Features

- Single `RestClient` with namespaced sub-objects for every API
- All 37 calling commands: dial, play, record, collect, detect, tap, stream, AI, transcribe, and more
- Full Fabric API: 13 resource types with CRUD + addresses, tokens, and generic resources
- Datasphere: document management and semantic search
- Video: rooms, sessions, recordings, conferences, tokens, streams
- Phone number management, 10DLC registry, MFA, logs, and more
- Shared `http.Client` for connection pooling across all calls
- Typed params and responses -- generated `*Params` structs and `*Response` wrapper types per operation

## Documentation

- [Getting Started](docs/getting-started.md) -- installation, configuration, first API call
- [Client Reference](docs/client-reference.md) -- RestClient constructor, namespaces, error handling
- [Fabric Resources](docs/fabric.md) -- managing AI agents, SWML scripts, subscribers, call flows, and more
- [Calling Commands](docs/calling.md) -- REST-based call control (dial, play, record, collect, AI, etc.)
- [All Namespaces](docs/namespaces.md) -- phone numbers, video, datasphere, logs, registry, and more

## Pagination

<!-- snippet-setup -->
```go
import (
	"context"
	"fmt"
	"log"

	"github.com/signalwire/signalwire-go/v3/pkg/rest"
)

// Client established in the Quick Start above.
var client, _ = rest.NewRestClient("", "", "")

var (
	_ = context.Background
	_ = fmt.Println
	_ = log.Fatal
)
```

List endpoints return one page at a time with a `links.next` cursor. Every
list resource exposes a `Paginate(ctx, params)` method that returns a
`*namespaces.Paginator` — it follows that cursor for you, so you never hand-build
the `page_token` loop. `List(ctx, params)` still returns a single raw page when
that is all you want.

`Paginator` has two idioms. `Next` yields one page at a time and reports whether
more remain:

```go
it := client.Fabric.Addresses.Paginate(context.Background(), nil)
for {
    items, hasMore, err := it.Next()
    if err != nil {
        log.Fatal(err)
    }
    for _, item := range items {
        fmt.Println(item["id"])
    }
    if !hasMore {
        break
    }
}
```

`ForEach` walks every item across all pages, fetching pages lazily; return a
non-nil error from the callback to stop early:

```go
it := client.Fabric.Addresses.Paginate(context.Background(), nil)
err := it.ForEach(func(item map[string]any) error {
    fmt.Println(item["id"])
    return nil // return an error to stop paging early
})
if err != nil {
    log.Fatal(err)
}
```

Construction does not fetch — the first request happens on the first `Next`
(or `ForEach`). The `ctx` passed to `Paginate` is threaded onto every page fetch,
so cancelling it stops the walk. `Paginate` is available on both the CRUD
resources and the read-only list resources (logs, sessions).

## Examples

- [rest_manage_resources.go](examples/rest_manage_resources.go) -- create an AI agent, assign a phone number, and place a test call
- [rest_datasphere_search.go](examples/rest_datasphere_search.go) -- upload a document and run a semantic search
- [rest_calling_play_and_record.go](examples/rest_calling_play_and_record.go) -- play TTS, record, transcribe, and denoise on a call
- [rest_calling_ivr_and_ai.go](examples/rest_calling_ivr_and_ai.go) -- IVR collection, AI operations, and advanced call control
- [rest_fabric_swml_and_callflows.go](examples/rest_fabric_swml_and_callflows.go) -- deploy SWML scripts and call flows
- [rest_fabric_subscribers_and_sip.go](examples/rest_fabric_subscribers_and_sip.go) -- provision SIP-enabled users on Fabric
- [rest_fabric_conferences_and_routing.go](examples/rest_fabric_conferences_and_routing.go) -- conferences, cXML, routing, and tokens
- [rest_phone_number_management.go](examples/rest_phone_number_management.go) -- full phone number inventory lifecycle
- [rest_10dlc_registration.go](examples/rest_10dlc_registration.go) -- 10DLC brand and campaign registration
- [rest_queues_mfa_and_recordings.go](examples/rest_queues_mfa_and_recordings.go) -- queues, MFA verification, and recordings
- [rest_video_rooms.go](examples/rest_video_rooms.go) -- video rooms, sessions, conferences, and streams

## Environment Variables

| Variable | Description |
|----------|-------------|
| `SIGNALWIRE_PROJECT_ID` | Project ID for authentication |
| `SIGNALWIRE_API_TOKEN` | API token for authentication |
| `SIGNALWIRE_SPACE` | Space hostname (e.g. `example.signalwire.com`) |
| `SIGNALWIRE_LOG_LEVEL` | Log level (`debug` for HTTP request details) |

## Package Structure

```
pkg/rest/
    client.go                          // HTTPClient -- HTTP transport, Basic Auth, JSON encoding
    rest_client.go                     // RestClient -- namespace wiring, env var resolution
    rest_tree_generated.go             // generated namespace accessor tree on RestClient
    namespaces/
        common.go                      // HTTPClient interface, Resource, CrudResource
        paginator.go                   // list() pagination iterator
        call_handler.go                // calling command dispatch (single POST)
        client_tree_generated.go       // generated per-namespace resource containers
        fabric_resources_generated.go  // fabric resource types + addresses + tokens
        calling_resources_generated.go // calling command-dispatch resources
        video_resources_generated.go   // rooms, sessions, recordings, conferences
        datasphere_resources_generated.go // documents, search, chunks
        <ns>_resources_generated.go    // one per namespace (chat/fax/logs/message/
                                       //   messages/project/projects/pubsub/relay_rest/voice)
        <ns>_types_generated.go        // generated *Params / *Response types per namespace
```
