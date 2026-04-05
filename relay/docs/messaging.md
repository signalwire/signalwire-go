# Messaging

Send and receive SMS/MMS messages through the RELAY client.

## Sending Messages

Use `client.SendMessage()` to send an outbound SMS or MMS.

```go
message, err := client.SendMessage(
	relay.WithMessageTo("+15552222222"),
	relay.WithMessageFrom("+15551111111"),
	relay.WithMessageBody("Hello from SignalWire!"),
)
if err != nil {
	fmt.Printf("Send failed: %v\n", err)
	return
}
```

### Wait for delivery

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
event := message.Wait(context.Background()) // blocks until delivered/failed
fmt.Printf("Final state: %s\n", message.State())
if message.Reason() != "" {
	fmt.Printf("Reason: %s\n", message.Reason())
}
```

### Fire and forget

```go
message, err := client.SendMessage(
	relay.WithMessageTo("+15552222222"),
	relay.WithMessageFrom("+15551111111"),
	relay.WithMessageBody("Hello!"),
)
// don't call message.Wait() -- continue immediately
```

### Callback via goroutine

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

go func() {
	event := message.Wait(context.Background())
	state, _ := event.Params["message_state"].(string)
	fmt.Printf("Delivery: %s\n", state)
}()
```

### MMS (media messages)

```go
message, err := client.SendMessage(
	relay.WithMessageTo("+15552222222"),
	relay.WithMessageFrom("+15551111111"),
	relay.WithMessageBody("Check this out!"),
	relay.WithMessageMedia([]string{"https://example.com/image.jpg"}),
)
```

### All parameters

```go
message, err := client.SendMessage(
	relay.WithMessageTo("+15552222222"),         // required -- E.164 format
	relay.WithMessageFrom("+15551111111"),       // required -- E.164 format
	relay.WithMessageBody("Message text"),       // required if no media
	relay.WithMessageMedia([]string{"https://..."}), // required if no body
	relay.WithMessageContext("my_context"),       // context for state events
	relay.WithMessageTags([]string{"vip", "support"}), // optional tags
	relay.WithMessageRegion("us"),               // optional origination region
)
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
		fmt.Printf("From: %s\n", message.FromNumber)
		fmt.Printf("To: %s\n", message.ToNumber)
		fmt.Printf("Body: %s\n", message.Body)
		if len(message.Media) > 0 {
			fmt.Printf("Media: %v\n", message.Media)
		}

		// Reply back
		client.SendMessage(
			relay.WithMessageTo(message.FromNumber),
			relay.WithMessageFrom(message.ToNumber),
			relay.WithMessageBody(fmt.Sprintf("You said: %s", message.Body)),
		)
	})

	client.Run()
}
```

## Message Object

### Properties

| Property | Type | Description |
|----------|------|-------------|
| `MessageID` | `string` | Unique message identifier |
| `Context` | `string` | Context the message belongs to |
| `Direction` | `string` | `inbound` or `outbound` |
| `FromNumber` | `string` | Sender phone number (E.164) |
| `ToNumber` | `string` | Recipient phone number (E.164) |
| `Body` | `string` | Text body of the message |
| `Media` | `[]string` | Media URLs (MMS) |
| `Segments` | `int` | Number of message segments |
| `Tags` | `[]string` | Tags attached to the message |

### Methods

| Method | Description |
|--------|-------------|
| `message.Wait(ctx)` | Block until terminal state. Returns the terminal `*RelayEvent`. |
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
import "github.com/signalwire/signalwire-go/pkg/relay"

// Use the event type constants
relay.EventMessagingReceive // "messaging.receive"
relay.EventMessagingState   // "messaging.state"
```

## Combining Calls and Messages

The same `RelayClient` handles both calls and messages:

```go
client := relay.NewRelayClient(
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
	fmt.Printf("SMS from %s: %s\n", message.FromNumber, message.Body)
})

client.Run()
```
