# Messaging

Send and receive SMS/MMS messages through the RELAY client.

## Sending Messages

Use `client.SendMessage()` to send an outbound SMS or MMS. The three required
parameters (to, from, body) are positional; everything else is a
`MessageOption`.

<!-- snippet-setup -->
```go
import (
	"context"
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/relay"
)

// Shared context the fragments below assume.
var client = relay.NewRelayClient()
var message *relay.Message
var err error

var (
	_ = client
	_ = message
	_ = err
	_ = context.Background
	_ = fmt.Sprint
)
```

```go
message, err = client.SendMessage(
	"+15552222222",                 // to (E.164)
	"+15551111111",                 // from (E.164)
	"Hello from SignalWire!",       // body
)
if err != nil {
	fmt.Printf("Send failed: %v\n", err)
	return
}
_ = message
```

### Wait for delivery

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
message.Wait(context.Background()) // blocks until delivered/failed
fmt.Printf("Final state: %s\n", message.State())
if message.Reason() != "" {
	fmt.Printf("Reason: %s\n", message.Reason())
}
```

### Fire and forget

```go
message, err = client.SendMessage(
	"+15552222222",
	"+15551111111",
	"Hello!",
)
// don't call message.Wait() -- continue immediately
_, _ = message, err
```

### Callback via goroutine

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

go func() {
	event, _ := message.Wait(context.Background())
	state, _ := event.Params["message_state"].(string)
	fmt.Printf("Delivery: %s\n", state)
}()
```

### MMS (media messages)

```go
message, err = client.SendMessage(
	"+15552222222",
	"+15551111111",
	"Check this out!",
	relay.WithMessageMedia([]string{"https://example.com/image.jpg"}),
)
_, _ = message, err
```

### All options

```go
message, err = client.SendMessage(
	"+15552222222",                                       // to   (required -- E.164)
	"+15551111111",                                       // from (required -- E.164)
	"Message text",                                       // body (required if no media)
	relay.WithMessageMedia([]string{"https://..."}),      // required if no body
	relay.WithMessageContext("my_context"),               // context for state events (default: relay protocol)
	relay.WithMessageTags([]string{"vip", "support"}),    // optional tags
	relay.WithMessageRegion("us"),                        // optional origination region
)
_, _ = message, err
```

## Receiving Messages

Register a handler with `client.OnMessage()` to receive inbound SMS/MMS.

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

	client.OnMessage(func(message *relay.Message) {
		fmt.Printf("From: %s\n", message.FromNumber())
		fmt.Printf("To: %s\n", message.ToNumber())
		fmt.Printf("Body: %s\n", message.Body())
		if len(message.Media()) > 0 {
			fmt.Printf("Media: %v\n", message.Media())
		}

		// Reply back
		client.SendMessage(
			message.FromNumber(),
			message.ToNumber(),
			fmt.Sprintf("You said: %s", message.Body()),
		)
	})

	client.Run()
}
```

## Message Object

### Accessors

These are methods (not fields) -- call them with `()`:

| Accessor | Type | Description |
|----------|------|-------------|
| `MessageID()` | `string` | Unique message identifier |
| `Context()` | `string` | Context the message belongs to |
| `Direction()` | `string` | `inbound` or `outbound` |
| `FromNumber()` | `string` | Sender phone number (E.164) |
| `ToNumber()` | `string` | Recipient phone number (E.164) |
| `Body()` | `string` | Text body of the message |
| `Media()` | `[]string` | Media URLs (MMS) |
| `Segments()` | `int` | Number of message segments |
| `Tags()` | `[]string` | Tags attached to the message |
| `MessageState()` | `MessageState` | State as a typed `MessageState` value |

### Methods

| Method | Description |
|--------|-------------|
| `message.Wait(ctx)` | Block until terminal state. Returns `(*RelayEvent, error)`. |
| `message.State()` | Current message state as a string. |
| `message.Reason()` | Failure reason (on `undelivered` or `failed`). |
| `message.IsDone()` | `true` if message reached a terminal state. |
| `message.Result()` | The terminal `*RelayEvent` (or `nil` if not done). |
| `message.On(handler)` | Register a listener for state change events. |

### Message States

Outbound messages progress through these states:

| State | Description |
|-------|-------------|
| `queued` | Message accepted and queued for sending |
| `initiated` | Sending has started |
| `sent` | Message sent to carrier |
| `delivered` | Message delivered to recipient (terminal) |
| `undelivered` | Delivery failed (terminal) -- check `Reason()` |
| `failed` | Message failed to send (terminal) -- check `Reason()` |

Inbound messages always arrive with state `received`.

## Event Types

| Event | Description |
|-------|-------------|
| `MessageReceiveEvent` | Inbound message received |
| `MessageStateEvent` | Outbound message state change |

```go
// Use the event type constants
_ = relay.EventMessagingReceive // "messaging.receive"
_ = relay.EventMessagingState   // "messaging.state"
```

## Combining Calls and Messages

The same `RelayClient` handles both calls and messages:

```go
client = relay.NewRelayClient(
	relay.WithProject("..."),
	relay.WithToken("..."),
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

client.OnMessage(func(message *relay.Message) {
	fmt.Printf("SMS from %s: %s\n", message.FromNumber(), message.Body())
})

client.Run()
```
