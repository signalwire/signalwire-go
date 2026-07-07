# Call Methods Reference

A `*relay.Call` represents a live phone call. You get one from `client.OnCall` (inbound) or `client.Dial()` (outbound).

## Properties

Call properties are exposed via accessor methods:

| Accessor | Type | Description |
|----------|------|-------------|
| `CallID()` | `string` | Unique call identifier |
| `NodeID()` | `string` | Server node handling the call |
| `State()` | `string` | Current state: `created`, `ringing`, `answered`, `ending`, `ended` |
| `Direction()` | `string` | `inbound` or `outbound` |
| `Tag()` | `string` | Correlation tag |
| `Device()` | `map[string]any` | Device info (type, params) |
| `SegmentID()` | `string` | Segment identifier |

## Actions: Blocking vs Fire-and-Forget

Methods like `Play()`, `Record()`, `Detect()`, etc. return **Action** objects. The call itself only dispatches the command — the actual operation runs asynchronously on the server. You choose how to handle completion:

### Wait inline (blocking)

```go
action := call.Play([]map[string]any{{"type": "tts", "params": map[string]any{"text": "Hello"}}})
action.Wait(context.Background()) // blocks until playback finishes
// execution continues only after play is done
```

`Wait` takes a `context.Context`; use `context.WithTimeout` to bound how long you block.

### Fire and forget (background)

```go
action := call.Play([]map[string]any{{"type": "tts", "params": map[string]any{"text": "Hello"}}})
// don't call action.Wait() — continue immediately while audio plays
call.SendDigits("1234")

// check later if needed
if action.IsDone() {
	fmt.Printf("Play result: %v\n", action.Result())
}
```

### Fire with callback

```go
// Via the option
action := call.Play(
	[]map[string]any{{"type": "tts", "params": map[string]any{"text": "Hello"}}},
	relay.WithPlayOnCompleted(func(event *relay.RelayEvent) {
		fmt.Printf("Done: %v\n", event.Params)
	}),
)
_ = action

// Or register after creation
rec := call.Record(relay.WithRecordOnCompleted(func(event *relay.RelayEvent) {
	fmt.Printf("Recording URL: %s\n", event.GetString("url"))
	call.Hangup("")
}))
_ = rec
```

The completion callback is available on all action-based methods via their `With<Verb>OnCompleted` option (`Play`, `Record`, `Collect`, etc.). Errors in callbacks are caught and logged, never crash the client. The callback also fires when the call is gone (404/410).

### Action methods summary

| Method | Returns |
|--------|---------|
| `action.Wait(ctx)` | Blocks until the action completes, returns the terminal `*RelayEvent` |
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

End the call. Pass `""` for the default reason.

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
action := call.Play([]map[string]any{{"type": "tts", "params": map[string]any{"text": "Hello!"}}})
action.Wait(context.Background())

// Audio file
action = call.Play([]map[string]any{{"type": "audio", "params": map[string]any{"url": "https://example.com/sound.mp3"}}})

// Silence
action = call.Play([]map[string]any{{"type": "silence", "params": map[string]any{"duration": 2}}})

// Ringtone
action = call.Play([]map[string]any{{"type": "ringtone", "params": map[string]any{"name": "us"}}})

// Control playback
action.Pause()
action.Resume()
action.Volume(-3.0)
action.Stop()
```

Convenience helpers avoid building the media slice by hand: `call.PlayTTS(text, opts...)`, `call.PlayAudio(url, opts...)`, `call.PlaySilence(duration)`, and `call.PlayRingtone(name, opts...)`.

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
fmt.Printf("Recording URL: %s\n", event.GetString("url"))
```

## Input Collection

### `PlayAndCollect(media []map[string]any, collect map[string]any, opts ...PlayOption) *CollectAction`

Play audio and collect DTMF or speech input. Returns a `*CollectAction`.

```go
action := call.PlayAndCollect(
	[]map[string]any{{"type": "tts", "params": map[string]any{"text": "Press 1 for sales, 2 for support."}}},
	map[string]any{"digits": map[string]any{"max": 1, "digit_timeout": 5.0}},
)
event, _ := action.Wait(context.Background())
_ = event
```

### `Collect(params *CollectParams) *StandaloneCollectAction`

Collect input without playing audio. `CollectParams` exposes the named fields (`Digits`, `Speech`, `PartialResults`, etc.); pass `nil` for an empty collect body.

```go
partial := true
action := call.Collect(&relay.CollectParams{
	Digits:         map[string]any{"max": 4, "terminators": "#"},
	Speech:         map[string]any{"language": "en-US"},
	PartialResults: &partial,
})
event, _ := action.Wait(context.Background())
_ = event
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

Detect machine, fax, or digits.

```go
timeout := 30.0
action := call.Detect(map[string]any{"type": "machine"}, &timeout)
event, _ := action.Wait(context.Background())
_ = event
```

Typed convenience helpers: `call.DetectDigit(opts...)`, `call.DetectAnsweringMachine(opts...)`, and `call.DetectFax(opts...)`.

## SIP Refer

### `Refer(device map[string]any, statusURL string) error`

Transfer via SIP REFER.

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
action := call.SendFax("https://example.com/document.pdf", "+15551234567")
event, _ := action.Wait(context.Background())
_ = event
```

### `ReceiveFax(opts ...FaxOption) *FaxAction`

```go
action := call.ReceiveFax()
event, _ := action.Wait(context.Background())
_ = event
```

## Tap (Media Interception)

### `Tap(tap, device map[string]any, controlID ...string) *TapAction`

Intercept call media and stream to an RTP endpoint.

```go
action := call.Tap(
	map[string]any{"type": "audio", "params": map[string]any{"direction": "both"}},
	map[string]any{"type": "rtp", "params": map[string]any{"addr": "192.168.1.100", "port": 5000}},
)
_ = action
```

## Streaming

### `Stream(url string, opts ...StreamOption) *StreamAction`

Stream call audio to a WebSocket endpoint.

```go
action := call.Stream(
	"wss://example.com/audio",
	relay.WithStreamName("my_stream"),
	relay.WithStreamCodec("PCMU"),
	relay.WithStreamTrack("inbound_track"),
)
// Stop streaming
action.Stop()
```

## Payment

### `Pay(connectorURL string, opts ...PayOption) *PayAction`

Collect a payment via DTMF.

```go
action := call.Pay(
	"https://pay.example.com",
	relay.WithPayChargeAmount("25.99"),
	relay.WithPayCurrency("usd"),
	relay.WithPayInputMethod("dtmf"),
)
event, _ := action.Wait(context.Background())
_ = event
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

### `Transcribe(statusURL string, controlID ...string) *TranscribeAction`

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

Echo audio back to the caller (useful for testing).

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
	relay.WithAISWAIG(map[string]any{"functions": []any{}}),
	relay.WithAIParams(map[string]any{"end_of_speech_timeout": 3000}),
)
event, _ := action.Wait(context.Background())
_ = event
```

### `AmazonBedrock(opts ...AIOption) *AIAction`

Connect to an Amazon Bedrock AI agent. Takes the same `AIOption` values as `AI`.

### `AIMessage(controlID, text, role string, reset, globalData map[string]any) error`

Send a message to an active AI session.

### `AIHold(controlID, timeout, prompt string) error` / `AIUnhold(controlID, prompt string) error`

Put an AI session on/off hold.

## Rooms

### `JoinRoom(name, statusURL string) error`

```go
call.JoinRoom("my_room", "")
```

### `LeaveRoom() error`

```go
call.LeaveRoom()
```

## Queue

### `QueueEnter(name, statusURL string) error`

```go
call.QueueEnter("support", "")
```

### `QueueLeave(name, queueID, statusURL string) error`

```go
call.QueueLeave("support", "q-123", "")
```

## Digit Bindings

### `BindDigit(digits, method string, bindParams map[string]any, realm string, maxTriggers int) error`

Bind a DTMF sequence to trigger a RELAY method.

```go
call.BindDigit(
	"*1",
	"calling.play",
	map[string]any{"play": []map[string]any{{"type": "tts", "params": map[string]any{"text": "You pressed star-1"}}}},
	"",
	0,
)
```

### `ClearDigitBindings(realm string) error`

```go
call.ClearDigitBindings("")
```

## User Events

### `UserEvent(eventName string, extra ...map[string]any) error`

Send a custom event.

```go
call.UserEvent("order_placed", map[string]any{"order_id": "12345"})
```

## Event Handling

### `On(eventType string, handler func(*RelayEvent))`

Register an event listener on this call.

```go
call.On(relay.EventCallingCallPlay, func(event *relay.RelayEvent) {
	fmt.Printf("Play state: %s\n", event.GetString("state"))
})
```

### `WaitFor(ctx context.Context, eventType string, predicate func(*RelayEvent) bool) (*RelayEvent, error)`

Wait for a specific event. Pass a `nil` predicate to match the first event of that type.

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
event, err := call.WaitFor(ctx, relay.EventCallingCallPlay, nil)
_ = event
_ = err
```

### `WaitForEnded(ctx context.Context) (*RelayEvent, error)`

Wait for the call to end.

```go
event, _ := call.WaitForEnded(context.Background())
fmt.Printf("End reason: %s\n", event.GetString("end_reason"))
```
