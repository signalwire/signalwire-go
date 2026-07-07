# Getting Started with RELAY

The RELAY client connects to SignalWire via WebSocket and gives you real-time, imperative control over phone calls.

## Installation

The RELAY client is part of the SignalWire Go SDK:

```bash
go get github.com/signalwire/signalwire-go
```

Then import the package:

<!-- snippet: no-compile illustrative import statement (reference only) -->
```go
import "github.com/signalwire/signalwire-go/pkg/relay"
```

## Configuration

You need three things to connect:

| Option | Env Var | Description |
|--------|---------|-------------|
| `relay.WithProject` | `SIGNALWIRE_PROJECT_ID` | Your SignalWire project ID |
| `relay.WithToken` | `SIGNALWIRE_API_TOKEN` | Your SignalWire API token |
| `relay.WithSpace` | `SIGNALWIRE_SPACE` | Your space hostname (e.g. `example.signalwire.com`) |

Alternatively, you can authenticate with a JWT token:

| Option | Env Var | Description |
|--------|---------|-------------|
| `relay.WithJWT` | `SIGNALWIRE_JWT_TOKEN` | A SignalWire JWT auth token |

## Minimal Example

```go
package main

import (
	"context"

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
		action := call.PlayTTS("Hello!")
		action.Wait(context.Background())
		call.Hangup("")
	})

	client.Run()
}
```

Or skip the credential options entirely — `NewRelayClient` automatically falls
back to `SIGNALWIRE_PROJECT_ID`, `SIGNALWIRE_API_TOKEN`, `SIGNALWIRE_JWT_TOKEN`,
and `SIGNALWIRE_SPACE` for any credential you don't set explicitly:

```bash
export SIGNALWIRE_PROJECT_ID=your-project-id
export SIGNALWIRE_API_TOKEN=your-api-token
export SIGNALWIRE_SPACE=example.signalwire.com
```

<!-- snippet-setup -->
```go
import (
	"github.com/signalwire/signalwire-go/pkg/relay"
)

// Shared context the fragments below assume.
var client = relay.NewRelayClient()
var call *relay.Call
var err error

var (
	_ = client
	_ = call
	_ = err
)
```

```go
client = relay.NewRelayClient(
	relay.WithContexts("default"),
)

client.OnCall(func(call *relay.Call) {
	call.Answer()
	call.Hangup("")
})

client.Run()
```

## Contexts

Contexts are topics your client subscribes to for receiving inbound calls. When a call arrives on a context you're subscribed to, your `OnCall` handler is invoked.

```go
// Subscribe at connect time
client = relay.NewRelayClient(relay.WithContexts("sales", "support"))

// Or dynamically after connecting
client.Receive("billing")
client.Unreceive("sales")
```

## Making Outbound Calls

Use `client.Dial()` to place an outbound call. Devices are `[][]map[string]any`:
the outer slice is serial attempts, the inner slice is parallel attempts.

```go
import "context"

call, err = client.Dial([][]map[string]any{
	{
		{"type": "phone", "params": map[string]any{"to_number": "+15551234567", "from_number": "+15559876543"}},
	},
})
if err != nil {
	// dial failed or timed out
}
// call is now a live *relay.Call
action := call.PlayTTS("This is an outbound call.")
action.Wait(context.Background())
call.Hangup("")
```

To try two numbers simultaneously, put both devices in the same inner slice:

```go
call, err = client.Dial([][]map[string]any{
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

## Cancellable Dial

For use within an existing application, `DialContext` accepts a `context.Context`
so a caller can cancel or time out the dial:

```go
import (
	"context"
	"time"
)

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

call, err = client.DialContext(ctx, [][]map[string]any{
	{
		{"type": "phone", "params": map[string]any{"to_number": "+15551234567", "from_number": "+15559876543"}},
	},
})
```

## Next Steps

- [Call Methods Reference](call-methods.md) -- all methods available on a Call object
- [Events](events.md) -- handling real-time call events
- [Client Reference](client-reference.md) -- RelayClient configuration and methods
