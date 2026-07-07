# Messaging

Send and receive SMS/MMS messages through the RELAY client.

## Sending Messages

Use `client.SendMessage(to, from, body, opts...)` to send an outbound SMS or MMS.

```go
message, err := client.SendMessage("+15552222222", "+15551111111", "Hello from SignalWire!")
if err != nil {
	log.Fatal(err)
}
_ = message
```

### Wait for delivery

```go
message, err := client.SendMessage("+15552222222", "+15551111111", "Hello!")
if err != nil {
	log.Fatal(err)
}
message.Wait(context.Background()) // blocks until delivered/failed
fmt.Printf("Final state: %s\n", message.State())
if message.Reason() != "" {
	fmt.Printf("Reason: %s\n", message.Reason())
}
```

### Fire and forget

```go
message, err := client.SendMessage("+15552222222", "+15551111111", "Hello!")
if err != nil {
	log.Fatal(err)
}
// don't call message.Wait() — continue immediately
_ = message
```

### Callback on completion

```go
message, err := client.SendMessage("+15552222222", "+15551111111", "Hello!",
	relay.WithMessageOnCompleted(func(m *relay.Message, event *relay.RelayEvent) {
		fmt.Printf("Delivery: %s\n", event.GetString("message_state"))
	}),
)
_ = message
_ = err
```

### MMS (media messages)

```go
message, err := client.SendMessage("+15552222222", "+15551111111", "Check this out!",
	relay.WithMessageMedia([]string{"https://example.com/image.jpg"}),
)
_ = message
_ = err
```

### All parameters

```go
message, err := client.SendMessage(
	"+15552222222", // to    — required, E.164 format
	"+15551111111", // from  — required, E.164 format
	"Message text", // body  — required if no media
	relay.WithMessageMedia([]string{"https://..."}), // required if no body
	relay.WithMessageContext("my_context"),          // context for state events (default: relay protocol)
	relay.WithMessageTags([]string{"vip", "support"}), // optional tags for searching in UI
	relay.WithMessageRegion("us"),                     // optional origination region
	relay.WithMessageOnCompleted(callbackFn),          // optional completion callback
)
_ = message
_ = err
```

## Receiving Messages

Register a handler with `client.OnMessage` to receive inbound SMS/MMS.

```go
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/relay"
)

func main() {
	client := relay.NewRelayClient(
		relay.WithProject("your-project-id"),
		relay.WithToken("your-api-token"),
		relay.WithSpace("example.signalwire.com"),
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
		client.SendMessage(message.ToNumber(), message.FromNumber(),
			fmt.Sprintf("You said: %s", message.Body()))
	})

	client.Run()
}
```

## Message Object

### Accessors

| Method | Type | Description |
|--------|------|-------------|
| `MessageID()` | `string` | Unique message identifier |
| `Context()` | `string` | Context the message belongs to |
| `Direction()` | `string` | `inbound` or `outbound` |
| `FromNumber()` | `string` | Sender phone number (E.164) |
| `ToNumber()` | `string` | Recipient phone number (E.164) |
| `Body()` | `string` | Text body of the message |
| `Media()` | `[]string` | Media URLs (MMS) |
| `Segments()` | `int` | Number of message segments |
| `State()` | `string` | Current message state |
| `Reason()` | `string` | Failure reason (on `undelivered` or `failed`) |
| `Tags()` | `[]string` | Tags attached to the message |
| `IsDone()` | `bool` | `true` if message reached a terminal state |
| `Result()` | `*RelayEvent` | Terminal event (or `nil` if not done) |

### Methods

| Method | Description |
|--------|-------------|
| `Wait(ctx context.Context)` | Block until terminal state (or `ctx` cancellation). Returns the terminal `*RelayEvent`. |
| `On(handler func(*RelayEvent))` | Register a listener for state change events. |

### Message States

Outbound messages progress through these states:

| State | Description |
|-------|-------------|
| `queued` | Message accepted and queued for sending |
| `initiated` | Sending has started |
| `sent` | Message sent to carrier |
| `delivered` | Message delivered to recipient (terminal) |
| `undelivered` | Delivery failed (terminal) — check `reason` |
| `failed` | Message failed to send (terminal) — check `reason` |

Inbound messages always arrive with state `received`.

## Event Types

| Event | Description |
|-------|-------------|
| `MessageReceiveEvent` | Inbound message received |
| `MessageStateEvent` | Outbound message state change |

Both `relay.MessageReceiveEvent` and `relay.MessageStateEvent` are typed event
structs in the `relay` package.

## Combining Calls and Messages

The same `Client` handles both calls and messages:

```go
client := relay.NewRelayClient(
	relay.WithProject("..."),
	relay.WithToken("..."),
	relay.WithContexts("default"),
)

client.OnCall(func(call *relay.Call) {
	call.Answer()
	call.PlayTTS("Hello!")
	call.Hangup("")
})

client.OnMessage(func(message *relay.Message) {
	fmt.Printf("SMS from %s: %s\n", message.FromNumber(), message.Body())
})

client.Run()
```
