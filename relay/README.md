# SignalWire RELAY Client

Real-time call control and messaging over WebSocket using Go's concurrency primitives. The RELAY client connects to SignalWire via the Blade protocol (JSON-RPC 2.0 over WebSocket) and gives you imperative control over live phone calls and SMS/MMS messaging.

## Quick Start

```go
package main

import (
	"fmt"
	"os"

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
			{"type": "tts", "params": map[string]any{"text": "Welcome to SignalWire!"}},
		})
		action.Wait(context.Background())
		call.Hangup("")
	})

	client.Run()
}
```

## Features

- Goroutine-safe with auto-reconnect and exponential backoff
- All 57+ calling methods: play, record, collect, connect, detect, fax, tap, stream, AI, conferencing, queues, and more
- SMS/MMS messaging: send outbound messages, receive inbound messages, track delivery state
- Action objects with `Wait()`, `Stop()`, `Pause()`, `Resume()` for controllable operations
- Typed event structs for all call events
- JWT and legacy authentication
- Dynamic context subscription/unsubscription
- Configurable concurrency limits via functional options

## Documentation

- [Getting Started](docs/getting-started.md) -- installation, configuration, first call
- [Call Methods Reference](docs/call-methods.md) -- every method available on a Call object
- [Events](docs/events.md) -- event types, typed event structs, call states
- [Messaging](docs/messaging.md) -- sending and receiving SMS/MMS messages
- [Client Reference](docs/client-reference.md) -- Client configuration, methods, connection behavior

## Examples

- [relay_answer_and_welcome.go](examples/relay_answer_and_welcome.go) -- answer an inbound call and play a TTS greeting
- [relay_dial_and_play.go](examples/relay_dial_and_play.go) -- dial an outbound call, play audio, and hang up
- [relay_ivr_connect.go](examples/relay_ivr_connect.go) -- IVR with DTMF collection, playback, and call connect

## Environment Variables

| Variable | Description |
|----------|-------------|
| `SIGNALWIRE_PROJECT_ID` | Project ID for authentication |
| `SIGNALWIRE_API_TOKEN` | API token for authentication |
| `SIGNALWIRE_JWT_TOKEN` | JWT token (alternative to project/token) |
| `SIGNALWIRE_SPACE` | Space hostname (default: `relay.signalwire.com`) |
| `RELAY_MAX_ACTIVE_CALLS` | Max concurrent calls per client (default: 1000) |
| `RELAY_MAX_CONNECTIONS` | Max WebSocket connections per process (default: 1) |
| `SIGNALWIRE_LOG_LEVEL` | Log level (`debug` for WebSocket traffic) |

## Package Structure

```
pkg/relay/
    client.go      // Client -- WebSocket connection, auth, event dispatch
    call.go        // Call object -- all calling methods and Action types
    action.go      // Action, PlayAction, RecordAction, etc.
    message.go     // Message object -- SMS/MMS message tracking
    event.go       // Typed event structs
    constants.go   // Protocol constants, call states, event types
    options.go     // Functional options for Client, Call, and Dial
```
