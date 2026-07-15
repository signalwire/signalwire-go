# RelayClient Reference

<!-- snippet-setup -->
```go
import (
	"context"
	"fmt"

	"github.com/signalwire/signalwire-go/v3/pkg/relay"
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

<!-- snippet: no-compile illustrative API signature (reference only) -->
```go
client := relay.NewRelayClient(opts ...ClientOption)
```

### Functional Options

| Option | Env Var | Description |
|--------|---------|-------------|
| `WithProject(id)` | `SIGNALWIRE_PROJECT_ID` | Project ID for authentication |
| `WithToken(token)` | `SIGNALWIRE_API_TOKEN` | API token for authentication |
| `WithJWT(jwt)` | `SIGNALWIRE_JWT_TOKEN` | JWT token (alternative to project/token) |
| `WithSpace(host)` | `SIGNALWIRE_SPACE` | Space hostname (e.g. `example.signalwire.com`; a bare name has `.signalwire.com` appended) |
| `WithContexts(ctx...)` | -- | Topics to subscribe to |
| `WithMaxActiveCalls(n)` | `RELAY_MAX_ACTIVE_CALLS` | Max concurrent calls (default: 1000) |

Authentication requires either `WithProject()` + `WithToken()` (legacy) or `WithJWT()` (faster, no server roundtrip). All parameters fall back to their corresponding environment variables.

## Methods

### `Run()`

Blocking entry point. Connects, authenticates, and runs the event loop with auto-reconnect until interrupted (SIGINT/SIGTERM).

```go
client.Run()
```

### `Stop()`

Signal a running client to tear down its WebSocket and exit `Run()`. Safe to
call from another goroutine.

```go
import "time"

go func() {
	time.Sleep(5 * time.Minute)
	client.Stop()
}()
client.Run() // blocks until Stop() is called
```

### `OnCall(handler func(*relay.Call))`

Register the inbound call handler. The handler receives a `*Call` object. Each call runs in its own goroutine.

```go
client.OnCall(func(call *relay.Call) {
	call.Answer()
	// ...
})
```

### `Dial(devices [][]map[string]any, opts ...DialOption) (*Call, error)`

Place an outbound call. Returns a `*Call` once the remote party answers.

- `devices` -- nested slice of device objects (serial/parallel dial)
- `relay.WithDialFromNumber(from)` -- caller ID
- `relay.WithDialTag(tag)` -- optional correlation tag (auto-generated if omitted)
- `relay.WithDialMaxDuration(minutes)` -- max call duration in minutes
- `relay.WithDialClientTimeout(d)` -- how long to wait before returning `ErrDialTimeout` (default: 120s)

```go
call, err = client.Dial(
	[][]map[string]any{
		{{"type": "phone", "params": map[string]any{
			"to_number": "+15551234567", "from_number": "+15559876543",
		}}},
	},
	relay.WithDialFromNumber("+15559876543"),
	relay.WithDialTimeout(30), // seconds
)
if err != nil {
	fmt.Printf("Dial failed: %v\n", err)
	return
}
_ = call
```

`DialContext(ctx, devices, opts...)` adds caller cancellation via a
`context.Context`.

### `OnMessage(handler func(*relay.Message))`

Register the inbound message handler. The handler receives a `*Message` object.

```go
client.OnMessage(func(message *relay.Message) {
	fmt.Printf("SMS from %s: %s\n", message.FromNumber(), message.Body())
})
```

### `SendMessage(to, from, body string, opts ...MessageOption) (*Message, error)`

Send an outbound SMS/MMS. Returns a `*Message` that tracks delivery state.
The three required parameters are positional; additional `MessageOption`
values configure media, region, and tags.

```go
message, err = client.SendMessage(
	"+15552222222",
	"+15551111111",
	"Hello!",
)
if err != nil {
	fmt.Printf("Send failed: %v\n", err)
	return
}
event, _ := message.Wait(context.Background()) // block until delivered/failed
_ = event
```

See [Messaging](messaging.md) for full details.

### `Connect() error` / `Stop()`

Manual lifecycle control. `Connect()` connects and authenticates without
blocking (unlike `Run()`), returning an error if the client fails to start;
`Stop()` tears the connection down. Use these when you drive the client from
your own loop instead of `Run()`.

```go
import "log"

if err := client.Connect(); err != nil {
	log.Fatal(err)
}
// ... use client ...
client.Stop()
```

### `RunContext(ctx context.Context) error`

Cancellable form of `Run()`: it runs the event loop until `ctx` is cancelled
(or the client is stopped), then returns.

### `Execute(method string, params map[string]any) (json.RawMessage, error)`

Send a raw JSON-RPC request. Used internally by the `Call` methods, but
exposed for custom RELAY commands.

## Context Subscriptions

The client subscribes to the RELAY contexts (topics) passed at construction
time via `WithContexts(...)`. You can also change the subscription set
dynamically after connecting:

### `Receive(contexts ...string) error` / `Unreceive(contexts ...string) error`

Dynamically subscribe to or unsubscribe from contexts on a live client.

```go
client.Receive("new-context")
client.Unreceive("old-context")
```

## Accessors

| Method | Type | Description |
|--------|------|-------------|
| `RelayProtocol()` | `string` | Server-assigned protocol string from the connect response |
| `ProjectID()` | `string` | Project ID |
| `Space()` | `string` | Relay host |
| `Contexts()` | `[]string` | Initial contexts passed at construction |

## Connection Behavior

- **Auto-reconnect**: On connection loss, the client reconnects with exponential backoff (1s to 30s).
- **Ping/pong**: Client sends periodic pings and monitors server pings. After 3 consecutive failures, the connection is force-closed and reconnected.
- **Request queueing**: Requests made while disconnected are queued and sent after re-authentication.
- **Authorization state**: The server sends encrypted auth state via events. On reconnect, this is sent back for fast re-authentication without a full auth roundtrip.
- **Server disconnect**: The server can request a graceful disconnect (e.g. during deployment). The client auto-reconnects afterward.

## Concurrency

Each inbound call handler runs in its own goroutine, so multiple calls are handled concurrently. The `WithMaxActiveCalls()` option (default: 1000, or the `RELAY_MAX_ACTIVE_CALLS` env var) caps concurrent calls to prevent unbounded goroutine growth.

## Error Handling

```go
import "errors"

result, err := call.Play([]map[string]any{
	{"type": "tts", "params": map[string]any{"text": "Hello"}},
}).Wait(context.Background())
if err != nil {
	var relayErr *relay.RelayError
	if errors.As(err, &relayErr) {
		fmt.Printf("Error %d: %s\n", relayErr.Code, relayErr.Message)
	}
}
_ = result
```

`RelayError` carries the numeric `Code` and `Message` the RELAY server
returned for a failed command (its `Error()` string is formatted as
`RELAY error {code}: {message}`). A `RelayError` may also wrap a sentinel
(such as `relay.ErrDialTimeout`) so the same value satisfies both
`errors.As(err, &relayErr)` and `errors.Is(err, relay.ErrDialTimeout)`.

## Graceful Shutdown

Use a cancellable context or OS signal handler to call `client.Stop()` when
the process should exit:

```go
import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer cancel()

client = relay.NewRelayClient(relay.WithContexts("default"))

client.OnCall(func(call *relay.Call) {
	call.Answer()
	call.Hangup("")
})

go func() {
	<-ctx.Done()
	fmt.Println("Shutting down...")
	client.Stop()
}()

if err := client.Run(); err != nil {
	log.Fatal(err)
}
```
