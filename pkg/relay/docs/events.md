# Events

RELAY events are server-pushed notifications about call state changes and operation results. Events arrive over the WebSocket as `signalwire.event` JSON-RPC messages and are automatically routed to the correct `Call` object.

## Listening for Events

### On a Call

```go
client.OnCall(func(call *relay.Call) {
	// Register a listener
	call.On(relay.EventCallingCallPlay, func(event *relay.RelayEvent) {
		fmt.Printf("Play: %v\n", event.Params)
	})

	// Or wait for a specific event
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	event, err := call.WaitFor(ctx, relay.EventCallingCallState, func(e *relay.RelayEvent) bool {
		return e.GetString("call_state") == relay.CallStateEnded
	})
	_ = event
	_ = err
})
```

### Via Actions

Actions returned by `Play()`, `Record()`, etc. have a `Wait()` method that resolves when the operation completes:

```go
action := call.PlayTTS("Hello")
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
event, err := action.Wait(ctx)
// event is a *relay.RelayEvent with the terminal state
_ = event
_ = err
```

## Event Types

All event type constants live in the `relay` package (from `constants.go`):

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
| `EventCallingCallConference` | `calling.conference` | Conference state changes |
| `EventCallingCallError` | `calling.error` | Error events |
| `EventMessagingReceive` | `messaging.receive` | Inbound message received |
| `EventMessagingState` | `messaging.state` | Outbound message state change |

## Typed Event Structs

Raw events are always `*relay.RelayEvent` with a `Params` map. For convenience, typed event structs embed `*RelayEvent` and add named fields. `relay.ParseEvent` returns the concrete typed struct (as `any`) for a raw payload:

```go
// Automatic parsing — returns the concrete typed event as `any`.
event := relay.ParseEvent(rawPayload)

// Type-switch to the concrete struct
switch e := event.(type) {
case *relay.CallStateEvent:
	fmt.Println(e.CallState) // "answered"
	fmt.Println(e.EndReason) // "hangup" (only on ended)
case *relay.PlayEvent:
	fmt.Println(e.State)
}
```

### Available Typed Events

| Struct | Key Fields |
|--------|------------|
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

Constants: `CALL_STATE_CREATED`, `CALL_STATE_RINGING`, `CALL_STATE_ANSWERED`, `CALL_STATE_ENDING`, `CALL_STATE_ENDED`

## End Reasons

When a call reaches the `ended` state, the `end_reason` field indicates why:

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

Outbound messages progress through: `queued` → `initiated` → `sent` → `delivered` (or `undelivered`/`failed`).

Constants: `MESSAGE_STATE_QUEUED`, `MESSAGE_STATE_INITIATED`, `MESSAGE_STATE_SENT`, `MESSAGE_STATE_DELIVERED`, `MESSAGE_STATE_UNDELIVERED`, `MESSAGE_STATE_FAILED`, `MESSAGE_STATE_RECEIVED`
