# SignalWire RELAY Client

Real-time call control and messaging over WebSocket. The RELAY client connects to SignalWire via the Blade protocol (JSON-RPC 2.0 over WebSocket) and gives you imperative control over live phone calls and SMS/MMS messaging.

## Installation

```bash
go get github.com/signalwire/signalwire-go
```

## Quick Start

```go
package main

import (
	"context"
	"time"

	"github.com/signalwire/signalwire-go/pkg/relay"
)

func main() {
	client := relay.NewRelayClient(
		relay.WithProject("your-project-id"),
		relay.WithToken("your-api-token"),
		relay.WithSpace("example.signalwire.com"),
		relay.WithContexts("default"),
	)

	client.OnCall(func(call *relay.Call) {
		call.Answer()
		action := call.PlayTTS("Welcome to SignalWire!")
		action.Wait(context.Background())
		call.Hangup("")
	})

	client.Run() // blocking
	_ = time.Second
}
```

## Features

- Auto-reconnect with exponential backoff
- All 57+ calling methods: play, record, collect, connect, detect, fax, tap, stream, AI, conferencing, queues, and more
- SMS/MMS messaging: send outbound messages, receive inbound messages, track delivery state
- Action objects with `Wait()`, `Stop()`, `Pause()`, `Resume()` for controllable operations
- Typed event structs for all call events
- JWT and legacy authentication
- Dynamic context subscription/unsubscription
- Configurable concurrency limits

## Documentation

- [Getting Started](../../relay/docs/getting-started.md) -- installation, configuration, first call
- [Call Methods Reference](../../relay/docs/call-methods.md) -- every method available on a Call object
- [Events](../../relay/docs/events.md) -- event types, typed event classes, call states
- [Messaging](../../relay/docs/messaging.md) -- sending and receiving SMS/MMS messages
- [Client Reference](../../relay/docs/client-reference.md) -- RelayClient configuration, methods, connection behavior

## Examples

- [relay_demo](../../examples/relay_demo/) -- connect a RelayClient, answer an inbound call, and play a TTS greeting

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
    client.go        # Client (relay.NewRelayClient) -- WebSocket connection, auth, event dispatch
    call.go          # Call -- all calling methods
    action.go        # Action types -- Wait/Stop/Pause/Resume/Volume for controllable operations
    message.go       # Message -- SMS/MMS message tracking
    event.go         # Typed event structs (CallStateEvent, PlayEvent, ...)
    options.go       # Functional options (WithProject, WithToken, WithContexts, ...)
    constants.go     # Protocol constants, call states, event types
```
