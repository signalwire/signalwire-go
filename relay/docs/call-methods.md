# Call Methods Reference

A `Call` object represents a live phone call. You get one from the `OnCall` handler (inbound) or `client.Dial()` (outbound).

## Accessors

These are methods (not fields) -- call them with `()`:

| Accessor | Type | Description |
|----------|------|-------------|
| `CallID()` | `string` | Unique call identifier |
| `NodeID()` | `string` | Server node handling the call |
| `State()` | `string` | Current state: `created`, `ringing`, `answered`, `ending`, `ended` |
| `CallState()` | `CallState` | Current state as a typed `CallState` value |
| `Direction()` | `string` | `inbound` or `outbound` |
| `Tag()` | `string` | Correlation tag |
| `Device()` | `map[string]any` | Device info (type, params) |
| `SegmentID()` | `string` | Segment identifier |
| `ProjectID()` | `string` | Project ID owning the call |
| `Context()` | `string` | Context the call arrived on |

## Actions: Blocking vs Fire-and-Forget

Methods like `Play()`, `Record()`, `Detect()`, etc. return **Action** objects. The `call.Play(...)` itself only waits for the server to accept the command -- the actual operation runs asynchronously on the server. You choose how to handle completion:

### Wait inline (blocking)

```go
action := call.Play([]map[string]any{
	{"type": "tts", "params": map[string]any{"text": "Hello"}},
})
event, _ := action.Wait(context.Background()) // blocks until playback finishes
// execution continues only after play is done
```

### Fire and forget (background)

```go
action := call.Play([]map[string]any{
	{"type": "tts", "params": map[string]any{"text": "Hello"}},
})
// don't call action.Wait() -- continue immediately while audio plays
call.SendDigits("1234")

// check later if needed
if action.IsDone() {
	fmt.Printf("Play result: %v\n", action.Result())
}
```

### Fire with goroutine callback

```go
action := call.Play([]map[string]any{
	{"type": "tts", "params": map[string]any{"text": "Hello"}},
})

go func() {
	event, _ := action.Wait(context.Background())
	fmt.Printf("Done: %v\n", event.Params)
}()
// continues immediately; goroutine fires when playback finishes
```

### Action methods summary

| Method | Returns |
|--------|---------|
| `action.Wait(ctx)` | Blocks until the action completes, returns `(*RelayEvent, error)` |
| `action.IsDone()` | `true` if the action has completed |
| `action.Result()` | The terminal `*RelayEvent` (or `nil` if not done) |
| `action.Completed()` | `true` if the action reached a terminal state |
| `action.Stop()` | Stop the operation on the server |

Some actions also have `Pause()`, `Resume()`, and `Volume()`.

## Lifecycle

### `Answer() error`

Answer an inbound call.

```go
call.Answer()
```

### `Hangup(reason string) error`

End the call.

```go
call.Hangup("")
call.Hangup("busy")
```

### `Pass() error`

Decline control, returning the call to routing.

```go
call.Pass()
```

## Audio Playback

### `Play(media []map[string]any, opts ...PlayOption) *PlayAction`

Play audio. Returns a `*PlayAction` with `Stop()`, `Pause()`, `Resume()`, `Volume()`, and `Wait()`.

```go
// TTS
action := call.Play([]map[string]any{
	{"type": "tts", "params": map[string]any{"text": "Hello!"}},
})
action.Wait(context.Background())

// Audio file
action = call.Play([]map[string]any{
	{"type": "audio", "params": map[string]any{"url": "https://example.com/sound.mp3"}},
})

// Silence
action = call.Play([]map[string]any{
	{"type": "silence", "params": map[string]any{"duration": 2}},
})

// Ringtone
action = call.Play([]map[string]any{
	{"type": "ringtone", "params": map[string]any{"name": "us"}},
})

// Control playback
action.Pause()
action.Resume()
action.Volume(-3.0)
action.Stop()
```

## Recording

### `Record(opts ...RecordOption) *RecordAction`

Record the call. Returns a `*RecordAction` with `Stop()`, `Pause()`, `Resume()`, and `Wait()`.

```go
action := call.Record(
	relay.WithRecordFormat("wav"),
	relay.WithRecordStereo(true),
	relay.WithRecordDirection("both"),
)
// ... later ...
action.Stop()
event, _ := action.Wait(context.Background())
fmt.Printf("Recording URL: %v\n", event.Params["url"])
```

## Input Collection

### `PlayAndCollect(media []map[string]any, collect map[string]any, opts ...CollectOption) *CollectAction`

Play audio and collect DTMF or speech input. Returns a `*CollectAction`.

```go
action := call.PlayAndCollect(
	[]map[string]any{
		{"type": "tts", "params": map[string]any{"text": "Press 1 for sales, 2 for support."}},
	},
	map[string]any{"digits": map[string]any{"max": 1, "digit_timeout": 5.0}},
)
event, _ := action.Wait(context.Background())
result, _ := event.Params["result"].(map[string]any)
params, _ := result["params"].(map[string]any)
digit, _ := params["digits"].(string)
```

### `Collect(params *CollectParams) *StandaloneCollectAction`

Collect input without playing audio. Pass a `*CollectParams` describing
the digit and/or speech recognition to perform (pass `nil` for an empty
collect body).

```go
partial := true
action := call.Collect(&relay.CollectParams{
	Digits:         map[string]any{"max": 4, "terminators": "#"},
	Speech:         map[string]any{"language": "en-US"},
	PartialResults: &partial,
})
event, _ := action.Wait(context.Background())
```

## Bridging

### `Connect(devices [][]map[string]any, opts ...ConnectOption) error`

Bridge the call to another destination.

```go
call.Connect(
	[][]map[string]any{
		{{"type": "phone", "params": map[string]any{"to_number": "+15551234567", "from_number": "+15559876543"}}},
	},
	relay.WithConnectRingback([]map[string]any{{"type": "ringtone", "params": map[string]any{"name": "us"}}}),
)
```

### `Disconnect() error`

Unbridge a connected call.

```go
call.Disconnect()
```

## DTMF

### `SendDigits(digits string) error`

Send DTMF tones.

```go
call.SendDigits("1234#")
```

## Detection

### `Detect(detect map[string]any, timeout *float64, controlID ...string) *DetectAction`

Detect machine, fax, or digits. `timeout` is a `*float64` (seconds); pass
`nil` for the server default. The optional `controlID` overrides the
auto-generated control identifier.

```go
timeout := 30.0
action := call.Detect(
	map[string]any{"type": "machine"},
	&timeout,
)
event, _ := action.Wait(context.Background())
```

Typed helpers cover the common cases with functional options:

```go
call.DetectAnsweringMachine(relay.WithAMDInitialTimeout(4.5)) // *DetectAction
call.DetectDigit()                                            // *DetectAction
call.DetectFax()                                              // *DetectAction
```

## SIP Refer

### `Refer(device map[string]any, statusURL string) error`

Transfer via SIP REFER. Pass an empty `statusURL` for no status callback.

```go
call.Refer(map[string]any{"type": "sip", "params": map[string]any{"to": "sip:user@example.com"}}, "")
```

## Transfer

### `Transfer(dest string) error`

Transfer call control to another RELAY app or SWML script.

```go
call.Transfer("https://example.com/swml-endpoint")
```

## Fax

### `SendFax(document, identity string, opts ...FaxOption) *FaxAction`

```go
action := call.SendFax(
	"https://example.com/document.pdf",
	"+15551234567", // caller identity
)
event, _ := action.Wait(context.Background())
```

### `ReceiveFax(opts ...FaxOption) *FaxAction`

```go
action := call.ReceiveFax()
event, _ := action.Wait(context.Background())
```

## Tap (Media Interception)

### `Tap(tap, device map[string]any) *TapAction`

Intercept call media and stream to an RTP endpoint.

```go
action := call.Tap(
	map[string]any{"type": "audio", "params": map[string]any{"direction": "both"}},
	map[string]any{"type": "rtp", "params": map[string]any{"addr": "192.168.1.100", "port": 5000}},
)
```

## Streaming

### `Stream(url string, opts ...StreamOption) *StreamAction`

Stream call audio to a WebSocket endpoint.

```go
action := call.Stream("wss://example.com/audio",
	relay.WithStreamCodec("PCMU"),
	relay.WithStreamDirection("inbound"),
)
// Stop streaming
action.Stop()
```

## Payment

### `Pay(connectorURL string, opts ...PayOption) *PayAction`

Collect a payment via DTMF. Amount, currency, prompts, etc. are supplied
through functional options.

```go
action := call.Pay(
	"https://pay.example.com",
	relay.WithPayChargeAmount("25.99"),
	relay.WithPayCurrency("usd"),
)
event, _ := action.Wait(context.Background())
```

## Conference

### `JoinConference(name string, opts ...ConferenceOption) error`

```go
call.JoinConference("my_conference",
	relay.WithConferenceMuted(false),
	relay.WithConferenceBeep("onEnter"),
)
```

### `LeaveConference(confID string) error`

```go
call.LeaveConference("conf-123")
```

## Hold

### `Hold() error` / `Unhold() error`

```go
call.Hold()
// ... later ...
call.Unhold()
```

## Denoise

### `Denoise() error` / `DenoiseStop() error`

```go
call.Denoise()
// ... later ...
call.DenoiseStop()
```

## Transcription

### `Transcribe(statusURL string) *TranscribeAction`

```go
action := call.Transcribe("https://example.com/transcription")
// ... later ...
action.Stop()
```

## Live Transcribe / Translate

### `LiveTranscribe(action map[string]any) error`

```go
call.LiveTranscribe(map[string]any{"start": map[string]any{"language": "en-US"}})
```

### `LiveTranslate(action map[string]any, statusURL string) error`

```go
call.LiveTranslate(map[string]any{"start": map[string]any{"source": "en-US", "target": "es"}}, "")
```

## Echo

### `Echo(timeout *float64, statusURL string) error`

Echo audio back to the caller (useful for testing). `timeout` is a
`*float64` (seconds); pass `nil` for the server default. Pass an empty
`statusURL` for no status callback.

```go
timeout := 30.0
call.Echo(&timeout, "")
```

## AI Agent

### `AI(opts ...AIOption) *AIAction`

Start an AI agent session on the call.

```go
action := call.AI(
	relay.WithAIPrompt(map[string]any{"text": "You are a helpful support agent."}),
	relay.WithAIParams(map[string]any{"end_of_speech_timeout": 3000}),
)
event, _ := action.Wait(context.Background())
```

### `AmazonBedrock(opts ...AIOption) *AIAction`

Connect to an Amazon Bedrock AI agent.

### `AIMessage(controlID, text, role string, reset, globalData map[string]any) error`

Send a message to an active AI session.

```go
call.AIMessage("ai-1", "Transfer me to billing", "user", nil, nil)
```

### `AIHold(controlID, timeout, prompt string) error` / `AIUnhold(controlID, prompt string) error`

Put an AI session on/off hold.

```go
call.AIHold("ai-1", "60", "Please wait while I transfer you.")
call.AIUnhold("ai-1", "I'm back, how can I help?")
```

## Rooms

### `JoinRoom(name string, statusURL string) error`

```go
call.JoinRoom("my_room", "")
```

### `LeaveRoom() error`

```go
call.LeaveRoom()
```

## Queue

### `QueueEnter(name string, statusURL string) error`

```go
call.QueueEnter("support", "")
```

### `QueueLeave(name string, queueID string, statusURL string) error`

```go
call.QueueLeave("support", "", "")
```

## Digit Bindings

### `BindDigit(digits, method string, bindParams map[string]any, realm string, maxTriggers int) error`

Bind a DTMF sequence to trigger a RELAY method. Pass an empty `realm` for
the default and `0` for unlimited triggers.

```go
call.BindDigit("*1", "calling.play", map[string]any{
	"play": []map[string]any{{"type": "tts", "params": map[string]any{"text": "You pressed star-1"}}},
}, "", 0)
```

### `ClearDigitBindings(realm string) error`

```go
call.ClearDigitBindings("")
```

## User Events

### `UserEvent(eventName string, extra ...map[string]any) error`

Send a custom event. The eventName is the event identifier (sent as the
`event` key on the wire). Any extra map(s) are merged into the top-level
wire params, matching Python's `user_event(*, event: Optional[str] = None,
**kwargs)`.

```go
call.UserEvent("order_placed", map[string]any{
    "order_id": "12345",
})
```

## Event Handling

### `On(eventType string, handler func(*relay.RelayEvent))`

Register an event listener on this call.

```go
call.On("calling.call.play", func(event *relay.RelayEvent) {
	fmt.Printf("Play state: %v\n", event.Params["state"])
})
```

### `WaitFor(ctx context.Context, eventType string, predicate func(*relay.RelayEvent) bool) (*relay.RelayEvent, error)`

Wait for a specific event. Pass a `nil` predicate to match any event of
`eventType`, or a custom function for additional filtering. Timeout is
handled via the context.

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
event, err := call.WaitFor(ctx, "calling.call.play", nil)
```

### State-change convenience helpers

`Call` provides typed helpers for the common state transitions, each
returning `(*relay.RelayEvent, error)`:

| Helper | Resolves when |
|--------|---------------|
| `WaitForRinging(ctx)` | the call starts ringing |
| `WaitForAnswered(ctx)` | the call is answered |
| `WaitForEnding(ctx)` | the call begins ending |
| `WaitForEnded(ctx)` | the call has ended |

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
defer cancel()
event, err := call.WaitForEnded(ctx)
if err == nil {
	fmt.Printf("End reason: %v\n", event.Params["end_reason"])
}
```

You can also filter the raw state-change event yourself with `WaitFor`:

```go
event, err := call.WaitFor(ctx, "calling.call.state", func(e *relay.RelayEvent) bool {
	state, _ := e.Params["call_state"].(string)
	return state == "ended"
})
```
