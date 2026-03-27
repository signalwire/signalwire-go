# SignalWire REST Client

Synchronous REST client for managing SignalWire resources, controlling live calls, and interacting with every SignalWire API surface from Go. No WebSocket required -- just standard HTTP requests with automatic connection pooling.

## Quick Start

```go
package main

import (
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/pkg/rest"
)

func main() {
	// Reads from SIGNALWIRE_PROJECT_ID, SIGNALWIRE_API_TOKEN, SIGNALWIRE_SPACE
	client, err := rest.NewRestClient("", "", "")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Create an AI agent
	agent, _ := client.Fabric.AIAgents.Create(map[string]any{
		"name":   "Support Bot",
		"prompt": map[string]any{"text": "You are a helpful support agent."},
	})

	// Search for a phone number
	results, _ := client.PhoneNumbers.Search(map[string]string{"area_code": "512"})

	// Place a call via REST
	client.Calling.Dial(map[string]any{
		"from": "+15559876543",
		"to":   "+15551234567",
		"url":  "https://example.com/call-handler",
	})
}
```

## Features

- Single `RestClient` with namespaced sub-objects for every API
- All 37 calling commands: dial, play, record, collect, detect, tap, stream, AI, transcribe, and more
- Full Fabric API: 13 resource types with CRUD + addresses, tokens, and generic resources
- Datasphere: document management and semantic search
- Video: rooms, sessions, recordings, conferences, tokens, streams
- Compatibility API: full Twilio-compatible LAML surface
- Phone number management, 10DLC registry, MFA, logs, and more
- Shared `http.Client` for connection pooling across all calls
- `map[string]any` returns -- raw JSON, no wrapper objects to learn

## Documentation

- [Getting Started](docs/getting-started.md) -- installation, configuration, first API call
- [Client Reference](docs/client-reference.md) -- RestClient constructor, namespaces, error handling
- [Fabric Resources](docs/fabric.md) -- managing AI agents, SWML scripts, subscribers, call flows, and more
- [Calling Commands](docs/calling.md) -- REST-based call control (dial, play, record, collect, AI, etc.)
- [Compatibility API](docs/compat.md) -- Twilio-compatible LAML endpoints
- [All Namespaces](docs/namespaces.md) -- phone numbers, video, datasphere, logs, registry, and more

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
- [rest_compat_laml.go](examples/rest_compat_laml.go) -- Twilio-compatible LAML migration
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
    client.go             // HttpClient -- HTTP transport, Basic Auth, JSON encoding
    signalwire_client.go  // RestClient -- namespace wiring, env var resolution
    namespaces/
        common.go         // HTTPClient interface, Resource, CrudResource
        fabric.go         // 13 resource types + generic resources + addresses + tokens
        calling.go        // 37 command dispatch methods via single POST
        phone_numbers.go  // Search, purchase, update, release
        compat.go         // Twilio-compatible LAML API
        video.go          // Rooms, sessions, recordings, conferences
        datasphere.go     // Documents, search, chunks
        ... and 15 more
```
