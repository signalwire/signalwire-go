# RelayClient Reference

<!-- snippet-setup -->
```go
import (
	"context"
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/relay"
)

// Shared context the fragments below assume.
var client = relay.NewRelayClient()
var call *relay.Call
var message *relay.Message
var err error

var (
	_ = client
	_ = call
	_ = message
	_ = err
	_ = context.Background
	_ = fmt.Sprint
)
```

## Constructor

`NewRelayClient` takes functional options:

```go
client = relay.NewRelayClient(
	relay.WithProject("..."),      // SIGNALWIRE_PROJECT_ID
	relay.WithToken("..."),        // SIGNALWIRE_API_TOKEN
	relay.WithJWT("..."),          // SIGNALWIRE_JWT_TOKEN
	relay.WithSpace("..."),        // SIGNALWIRE_SPACE (default: relay.signalwire.com)
	relay.WithContexts("default"), // Topics to subscribe to (variadic)
	relay.WithMaxActiveCalls(100), // RELAY_MAX_ACTIVE_CALLS (default: 1000)
)
```

Authentication requires either `WithProject` + `WithToken` (legacy) or `WithJWT` (faster, no server roundtrip). Any credential you don't set explicitly falls back to its corresponding environment variable.

## Methods

### `Run() error`

Blocking entry point. Connects, authenticates, and runs the event loop with auto-reconnect until interrupted. Returns an error if the client fails to start.

```go
import "log"

if err := client.Run(); err != nil {
	log.Fatal(err)
}
```

`RunContext(ctx)` is the cancellable form — it returns when `ctx` is cancelled.

### `Connect() error` / `Stop()`

Manual lifecycle control.

```go
import "log"

if err := client.Connect(); err != nil {
	log.Fatal(err)
}
// ... use client ...
client.Stop()
```

### `OnCall(handler func(*relay.Call))`

Register the inbound call handler. The handler receives a `*relay.Call`.

```go
client.OnCall(func(call *relay.Call) {
	call.Answer()
})
```

### `Dial(devices [][]map[string]any, opts ...DialOption) (*relay.Call, error)`

Place an outbound call. Returns a `*relay.Call` once the remote party answers.

- `devices` -- nested slice of device objects (serial/parallel dial)
- `relay.WithDialTag(tag)` -- optional correlation tag (auto-generated if omitted)
- `relay.WithDialMaxDuration(minutes)` -- max call duration in minutes
- `relay.WithDialClientTimeout(d)` -- how long to wait before returning `ErrDialTimeout` (default: 120s)

```go
call, err = client.Dial([][]map[string]any{
	{
		{"type": "phone", "params": map[string]any{"to_number": "+15551234567", "from_number": "+15559876543"}},
	},
})
```

`DialContext(ctx, devices, opts...)` adds caller cancellation via a `context.Context`.

### `OnMessage(handler func(*relay.Message))`

Register the inbound message handler. The handler receives a `*relay.Message`.

```go
client.OnMessage(func(message *relay.Message) {
	fmt.Printf("SMS from %s: %s\n", message.FromNumber(), message.Body())
})
```

### `SendMessage(to, from, body string, opts ...MessageOption) (*relay.Message, error)`

Send an outbound SMS/MMS. Returns a `*relay.Message` that tracks delivery state.

```go
import "log"

message, err = client.SendMessage("+15552222222", "+15551111111", "Hello!")
if err != nil {
	log.Fatal(err)
}
event, _ := message.Wait(context.Background()) // block until delivered/failed
_ = event
```

See [Messaging](messaging.md) for full details.

### `Execute(method string, params map[string]any) (json.RawMessage, error)`

Send a raw JSON-RPC request. Used internally by Call methods, but available for custom commands.

### `Receive(contexts ...string) error` / `Unreceive(contexts ...string) error`

Dynamically subscribe to or unsubscribe from contexts after connecting.

```go
client.Receive("new-context")
client.Unreceive("old-context")
```

## Accessors

| Method | Type | Description |
|--------|------|-------------|
| `RelayProtocol()` | `string` | Server-assigned protocol string from connect response |
| `ProjectID()` | `string` | Project ID |
| `Space()` | `string` | Relay host |
| `Contexts()` | `[]string` | Initial contexts |

## Connection Behavior

- **Auto-reconnect**: On connection loss, the client reconnects with exponential backoff (1s to 30s).
- **Ping/pong**: Client sends periodic pings and monitors server pings. After 3 consecutive failures, the connection is force-closed and reconnected.
- **Request queueing**: Requests made while disconnected are queued and sent after re-authentication.
- **Authorization state**: The server sends encrypted auth state via events. On reconnect, this is sent back for fast re-authentication without a full auth roundtrip.
- **Server disconnect**: The server can request a graceful disconnect (e.g. during deployment). The client auto-reconnects afterward.

## Concurrency

Each inbound call handler runs as an independent `asyncio.Task`, so multiple calls are handled concurrently. The `max_active_calls` parameter (default: 1000) caps concurrent calls to prevent unbounded memory growth.

For multiple WebSocket connections in one process, set `RELAY_MAX_CONNECTIONS` (default: 1).

## Error Handling

RELAY methods that return an `error` surface a `*relay.RelayError` when the
server returns a non-2xx response code. Use `errors.As` to inspect the code and
message:

```go
import "errors"

action := call.PlayTTS("Hello")
if _, err := action.Wait(context.Background()); err != nil {
	var re *relay.RelayError
	if errors.As(err, &re) {
		fmt.Printf("Error %d: %s\n", re.Code, re.Message)
	}
}
```

Errors 404 and 410 (call gone) are silently swallowed by Call methods since the
call no longer exists. Dial failures expose the `relay.ErrDialTimeout` and
`relay.ErrDialFailed` sentinels for `errors.Is` checks.
