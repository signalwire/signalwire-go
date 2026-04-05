# Getting Started with RELAY

The RELAY client connects to SignalWire via WebSocket and gives you real-time, imperative control over phone calls using Go's concurrency primitives.

## Installation

```bash
go get github.com/signalwire/signalwire-go/pkg/relay
```

No additional dependencies beyond the Go standard library and `gorilla/websocket`.

## Configuration

You need three things to connect:

| Parameter | Env Var | Description |
|-----------|---------|-------------|
| `WithProject()` | `SIGNALWIRE_PROJECT_ID` | Your SignalWire project ID |
| `WithToken()` | `SIGNALWIRE_API_TOKEN` | Your SignalWire API token |
| `WithSpace()` | `SIGNALWIRE_SPACE` | Your space hostname (e.g. `example.signalwire.com`) |

Alternatively, you can authenticate with a JWT token:

| Parameter | Env Var | Description |
|-----------|---------|-------------|
| `WithJWTToken()` | `SIGNALWIRE_JWT_TOKEN` | A SignalWire JWT auth token |

## Minimal Example

```go
package main

import (
	"context"
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
			{"type": "tts", "params": map[string]any{"text": "Hello!"}},
		})
		action.Wait(context.Background())
		call.Hangup("")
	})

	client.Run()
}
```

Or use environment variables and skip the constructor args:

```bash
export SIGNALWIRE_PROJECT_ID=your-project-id
export SIGNALWIRE_API_TOKEN=your-api-token
export SIGNALWIRE_SPACE=example.signalwire.com
```

```go
package main

import (
	"github.com/signalwire/signalwire-go/pkg/relay"
)

func main() {
	client := relay.NewRelayClient(
		relay.WithContexts("default"),
	)

	client.OnCall(func(call *relay.Call) {
		call.Answer()
		call.Hangup("")
	})

	client.Run()
}
```

## Contexts

Contexts are topics your client subscribes to for receiving inbound calls. When a call arrives on a context you're subscribed to, your `OnCall` handler is invoked.

```go
// Subscribe at connect time
client := relay.NewRelayClient(
	relay.WithContexts("sales", "support"),
)

// Or dynamically after connecting
client.Receive([]string{"billing"})
client.Unreceive([]string{"sales"})
```

## Making Outbound Calls

Use `client.Dial()` to place an outbound call:

```go
call, err := client.Dial([][]map[string]any{
	{{"type": "phone", "params": map[string]any{"to_number": "+15551234567", "from_number": "+15559876543"}}},
})
if err != nil {
	fmt.Printf("Dial failed: %v\n", err)
	return
}
// call is now a live Call object
action := call.Play([]map[string]any{
	{"type": "tts", "params": map[string]any{"text": "This is an outbound call."}},
})
action.Wait(context.Background())
call.Hangup("")
```

The outer slice represents serial attempts; the inner slice represents parallel attempts. For example, to try two numbers simultaneously:

```go
call, err := client.Dial([][]map[string]any{
	{
		{"type": "phone", "params": map[string]any{"to_number": "+15551111111", "from_number": "+15559876543"}},
		{"type": "phone", "params": map[string]any{"to_number": "+15552222222", "from_number": "+15559876543"}},
	},
})
```

## Debug Logging

Set the log level to see WebSocket traffic:

```bash
export SIGNALWIRE_LOG_LEVEL=debug
```

## Manual Lifecycle Control

For use within an existing application where you need explicit connect/disconnect:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

client := relay.NewRelayClient(relay.WithContexts("default"))

if err := client.Connect(ctx); err != nil {
	fmt.Printf("Connect failed: %v\n", err)
	os.Exit(1)
}
defer client.Disconnect()

call, err := client.Dial([][]map[string]any{...})
```

## Next Steps

- [Call Methods Reference](call-methods.md) -- all methods available on a Call object
- [Events](events.md) -- handling real-time call events
- [Messaging](messaging.md) -- sending and receiving SMS/MMS messages
- [Client Reference](client-reference.md) -- RelayClient configuration and methods
