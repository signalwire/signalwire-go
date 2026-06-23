# Events

RELAY events are server-pushed notifications about call state changes and operation results. Events arrive over the WebSocket as `signalwire.event` JSON-RPC messages and are automatically routed to the correct `Call` object.

## Listening for Events

### On a Call

```go
client.OnCall(func(call *relay.Call) {
	// Register a listener
	call.On("calling.call.play", func(event *relay.RelayEvent) {
		fmt.Printf("Play: %v\n", event.Params)
	})

	// Or wait for a specific event with a predicate + deadline
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	event, err := call.WaitFor(ctx, "calling.call.state",
		func(e *relay.RelayEvent) bool {
			state, _ := e.Params["call_state"].(string)
			return state == "ended"
		},
	)
	if err != nil {
		fmt.Printf("Wait error: %v\n", err)
	}
})
```

### Via Actions

Actions returned by `Play()`, `Record()`, etc. have a `Wait()` method that resolves when the operation completes:

```go
action := call.Play([]map[string]any{
	{"type": "tts", "params": map[string]any{"text": "Hello"}},
})
event, _ := action.Wait(context.Background())
// event is a *RelayEvent with the terminal state
```

## Event Types

All event type constants are defined in the `relay` package:

| Constant | Value | Description |
|----------|-------|-------------|
| `EventCallingCallState` | `calling.call.state` | Call state changes (created, ringing, answered, ending, ended) |
| `EventCallingCallReceive` | `calling.call.receive` | Inbound call notification |
| `EventCallingCallPlay` | `calling.call.play` | Play operation state changes |
| `EventCallingCallRecord` | `calling.call.record` | Record operation state changes |
| `EventCallingCallCollect` | `calling.call.collect` | Input collection results |
| `EventCallingCallConnect` | `calling.call.connect` | Bridge/connect state changes |
| `EventCallingCallDetect` | `calling.call.detect` | Detection results |
| `EventCallingCallFax` | `calling.call.fax` | Fax operation state changes |
| `EventCallingCallTap` | `calling.call.tap` | Tap operation state changes |
| `EventCallingCallStream` | `calling.call.stream` | Stream operation state changes |
| `EventCallingCallSendDigits` | `calling.call.send_digits` | DTMF send completion |
| `EventCallingCallDial` | `calling.call.dial` | Outbound dial progress |
| `EventCallingCallRefer` | `calling.call.refer` | SIP REFER results |
| `EventCallingCallDenoise` | `calling.call.denoise` | Denoise state changes |
| `EventCallingCallPay` | `calling.call.pay` | Payment state changes |
| `EventCallingCallQueue` | `calling.call.queue` | Queue state changes |
| `EventCallingCallEcho` | `calling.call.echo` | Echo state changes |
| `EventCallingCallTranscribe` | `calling.call.transcribe` | Transcription state changes |
| `EventCallingCallHold` | `calling.call.hold` | Hold/unhold state changes |
| `EventCallingCallAI` | `calling.call.ai` | AI agent session events |
| `EventCallingCallConference` | `calling.conference` | Conference state changes |
| `EventCallingCallError` | `calling.error` | Error events |
| `EventMessagingReceive` | `messaging.receive` | Inbound message received |
| `EventMessagingState` | `messaging.state` | Outbound message state change |

## Typed Event Structs

Raw events are always `*RelayEvent` with a `Params` map. For convenience, typed event structs provide named fields:

```go
import "github.com/signalwire/signalwire-go/pkg/relay"

// The event arrives as a *RelayEvent with an EventType and Params map.
if event.EventType == relay.EventCallingCallState {
	// Promote the generic event to its typed struct by passing the
	// Params map to the matching New<EventName> factory.
	stateEvent := relay.NewCallStateEvent(event.Params)
	fmt.Println(stateEvent.CallState) // "answered"
	fmt.Println(stateEvent.EndReason) // "hangup" (only on ended)
}
```

### Available Typed Events

| Struct | Key Fields |
|--------|-----------|
| `CallStateEvent` | `CallState`, `EndReason`, `Direction`, `Device` |
| `CallReceiveEvent` | `CallState`, `Direction`, `Device`, `NodeID`, `Context`, `Tag` |
| `PlayEvent` | `ControlID`, `State` |
| `RecordEvent` | `ControlID`, `State`, `URL`, `Duration`, `Size` |
| `CollectEvent` | `ControlID`, `State`, `Result`, `Final` |
| `ConnectEvent` | `ConnectState`, `Peer` |
| `DetectEvent` | `ControlID`, `Detect` |
| `FaxEvent` | `ControlID`, `Fax` |
| `TapEvent` | `ControlID`, `State`, `Tap`, `Device` |
| `StreamEvent` | `ControlID`, `State`, `URL`, `Name` |
| `SendDigitsEvent` | `ControlID`, `State` |
| `DialEvent` | `Tag`, `DialState`, `Call` |
| `ReferEvent` | `State`, `SIPReferTo`, `SIPReferResponseCode` |
| `DenoiseEvent` | `Denoised` |
| `PayEvent` | `ControlID`, `State` |
| `QueueEvent` | `ControlID`, `Status`, `QueueID`, `QueueName`, `Position`, `Size` |
| `EchoEvent` | `State` |
| `TranscribeEvent` | `ControlID`, `State`, `URL`, `Duration`, `Size` |
| `HoldEvent` | `State` |
| `ConferenceEvent` | `ConferenceID`, `Name`, `Status` |
| `CallingErrorEvent` | `Code`, `Message` |
| `MessageReceiveEvent` | `MessageID`, `Context`, `Direction`, `FromNumber`, `ToNumber`, `Body`, `Media`, `Segments`, `MessageState`, `Tags` |
| `MessageStateEvent` | `MessageID`, `Context`, `Direction`, `FromNumber`, `ToNumber`, `Body`, `Media`, `Segments`, `MessageState`, `Reason`, `Tags` |

## Call States

```
created -> ringing -> answered -> ending -> ended
```

Constants: `CallStateCreated`, `CallStateRinging`, `CallStateAnswered`, `CallStateEnding`, `CallStateEnded`

## End Reasons

When a call reaches the `ended` state, the `EndReason` field indicates why:

| Reason | Description |
|--------|-------------|
| `hangup` | Normal hangup |
| `cancel` | Caller cancelled |
| `busy` | Destination busy |
| `noAnswer` | No answer |
| `decline` | Call declined |
| `error` | Error occurred |
| `abandoned` | Call abandoned |
| `max_duration` | Max duration reached |
| `not_found` | Destination not found |

## Message States

Outbound messages progress through: `queued` -> `initiated` -> `sent` -> `delivered` (or `undelivered`/`failed`).

Constants: `MessageStateQueued`, `MessageStateInitiated`, `MessageStateSent`, `MessageStateDelivered`, `MessageStateUndelivered`, `MessageStateFailed`, `MessageStateReceived`

## Event Handling Patterns

### Goroutine-based listener

```go
client.OnCall(func(call *relay.Call) {
	call.Answer()

	// Listen for events in a separate goroutine
	go func() {
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			event, err := call.WaitFor(ctx, relay.EventCallingCallState, nil)
			cancel()
			if err != nil {
				return // timeout or call ended
			}
			state, _ := event.Params["call_state"].(string)
			fmt.Printf("Call state changed: %s\n", state)
			if state == "ended" {
				return
			}
		}
	}()

	action := call.Play([]map[string]any{
		{"type": "tts", "params": map[string]any{"text": "Hello!"}},
	})
	action.Wait(context.Background())
	call.Hangup("")
})
```

### Multiple concurrent listeners

```go
client.OnCall(func(call *relay.Call) {
	call.Answer()

	call.On(relay.EventCallingCallPlay, func(event *relay.RelayEvent) {
		fmt.Printf("Play state: %v\n", event.Params["state"])
	})

	call.On(relay.EventCallingCallRecord, func(event *relay.RelayEvent) {
		fmt.Printf("Record state: %v\n", event.Params["state"])
	})

	call.On(relay.EventCallingCallError, func(event *relay.RelayEvent) {
		fmt.Printf("Error: %v\n", event.Params["message"])
	})

	// Start playback and recording concurrently
	playAction := call.Play([]map[string]any{
		{"type": "tts", "params": map[string]any{"text": "Recording in progress."}},
	})
	recordAction := call.Record()

	playAction.Wait(context.Background())
	recordAction.Stop()
	call.Hangup("")
})
```
