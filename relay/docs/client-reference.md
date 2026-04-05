# RelayClient Reference

## Constructor

```go
client := relay.NewRelayClient(opts ...ClientOption)
```

### Functional Options

| Option | Env Var | Description |
|--------|---------|-------------|
| `WithProject(id)` | `SIGNALWIRE_PROJECT_ID` | Project ID for authentication |
| `WithToken(token)` | `SIGNALWIRE_API_TOKEN` | API token for authentication |
| `WithJWTToken(jwt)` | `SIGNALWIRE_JWT_TOKEN` | JWT token (alternative to project/token) |
| `WithSpace(host)` | `SIGNALWIRE_SPACE` | Space hostname (default: `relay.signalwire.com`) |
| `WithContexts(ctx...)` | -- | Topics to subscribe to |
| `WithMaxActiveCalls(n)` | `RELAY_MAX_ACTIVE_CALLS` | Max concurrent calls (default: 1000) |

Authentication requires either `WithProject()` + `WithToken()` (legacy) or `WithJWTToken()` (faster, no server roundtrip). All parameters fall back to their corresponding environment variables.

## Methods

### `Run()`

Blocking entry point. Connects, authenticates, and runs the event loop with auto-reconnect until interrupted (SIGINT/SIGTERM).

```go
client.Run()
```

### `Connect(ctx) error` / `Disconnect() error`

Manual lifecycle control for use within an existing application.

```go
ctx := context.Background()
if err := client.Connect(ctx); err != nil {
	log.Fatalf("Connect failed: %v", err)
}
defer client.Disconnect()
// ... use client ...
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

```go
call, err := client.Dial(
	[][]map[string]any{
		{{"type": "phone", "params": map[string]any{
			"to_number": "+15551234567", "from_number": "+15559876543",
		}}},
	},
	relay.WithDialTag("my-tag"),
	relay.WithDialMaxDuration(30),
	relay.WithDialTimeout(120 * time.Second),
)
if err != nil {
	fmt.Printf("Dial failed: %v\n", err)
	return
}
```

### `OnMessage(handler func(*relay.Message))`

Register the inbound message handler. The handler receives a `*Message` object.

```go
client.OnMessage(func(message *relay.Message) {
	fmt.Printf("SMS from %s: %s\n", message.FromNumber, message.Body)
})
```

### `SendMessage(opts ...MessageOption) (*Message, error)`

Send an outbound SMS/MMS. Returns a `*Message` that tracks delivery state.

```go
message, err := client.SendMessage(
	relay.WithMessageTo("+15552222222"),
	relay.WithMessageFrom("+15551111111"),
	relay.WithMessageBody("Hello!"),
)
if err != nil {
	fmt.Printf("Send failed: %v\n", err)
	return
}
event := message.Wait(context.Background()) // block until delivered/failed
```

See [Messaging](messaging.md) for full details.

### `Execute(method string, params map[string]any) (map[string]any, error)`

Send a raw JSON-RPC request. Used internally by Call methods, but available for custom commands.

### `Receive(contexts []string) error` / `Unreceive(contexts []string) error`

Dynamically subscribe to or unsubscribe from contexts after connecting.

```go
client.Receive([]string{"new-context"})
client.Unreceive([]string{"old-context"})
```

## Properties

| Property | Type | Description |
|----------|------|-------------|
| `RelayProtocol()` | `string` | Server-assigned protocol string from connect response |
| `Project()` | `string` | Project ID |
| `Host()` | `string` | Relay host |
| `Contexts()` | `[]string` | Initial contexts |

## Connection Behavior

- **Auto-reconnect**: On connection loss, the client reconnects with exponential backoff (1s to 30s).
- **Ping/pong**: Client sends periodic pings and monitors server pings. After 3 consecutive failures, the connection is force-closed and reconnected.
- **Request queueing**: Requests made while disconnected are queued and sent after re-authentication.
- **Authorization state**: The server sends encrypted auth state via events. On reconnect, this is sent back for fast re-authentication without a full auth roundtrip.
- **Server disconnect**: The server can request a graceful disconnect (e.g. during deployment). The client auto-reconnects afterward.

## Concurrency

Each inbound call handler runs in its own goroutine, so multiple calls are handled concurrently. The `WithMaxActiveCalls()` option (default: 1000) caps concurrent calls to prevent unbounded goroutine growth.

For multiple WebSocket connections in one process, set `RELAY_MAX_CONNECTIONS` (default: 1).

## Error Handling

```go
import "github.com/signalwire/signalwire-go/pkg/relay"

result, err := call.Play([]map[string]any{...}).Wait(context.Background())
if err != nil {
	var relayErr *relay.RelayError
	if errors.As(err, &relayErr) {
		fmt.Printf("Error %d: %s\n", relayErr.Code, relayErr.Message)
	}
}
```

`RelayError` is returned when the server returns a non-2xx response code. Errors 404 and 410 (call gone) are silently swallowed by Call methods since the call no longer exists.

## Graceful Shutdown

Use a context with cancellation or handle OS signals to shut down cleanly:

```go
ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer cancel()

client := relay.NewRelayClient(relay.WithContexts("default"))

client.OnCall(func(call *relay.Call) {
	call.Answer()
	call.Hangup("")
})

if err := client.Connect(ctx); err != nil {
	log.Fatal(err)
}
defer client.Disconnect()

<-ctx.Done()
fmt.Println("Shutting down...")
```
