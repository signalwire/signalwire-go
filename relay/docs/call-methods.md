# Call Methods Reference

A `Call` object represents a live phone call. You get one from the `OnCall` handler (inbound) or `client.Dial()` (outbound).

## Properties

| Property | Type | Description |
|----------|------|-------------|
| `CallID` | `string` | Unique call identifier |
| `NodeID` | `string` | Server node handling the call |
| `State` | `string` | Current state: `created`, `ringing`, `answered`, `ending`, `ended` |
| `Direction` | `string` | `inbound` or `outbound` |
| `Tag` | `string` | Correlation tag |
| `Device` | `map[string]any` | Device info (type, params) |
| `SegmentID` | `string` | Segment identifier |

## Actions: Blocking vs Fire-and-Forget

Methods like `Play()`, `Record()`, `Detect()`, etc. return **Action** objects. The `call.Play(...)` itself only waits for the server to accept the command -- the actual operation runs asynchronously on the server. You choose how to handle completion:

### Wait inline (blocking)

```go
action := call.Play([]map[string]any{
	{"type": "tts", "params": map[string]any{"text": "Hello"}},
})
event := action.Wait(context.Background()) // blocks until playback finishes
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
	event := action.Wait(context.Background())
	fmt.Printf("Done: %v\n", event.Params)
}()
// continues immediately; goroutine fires when playback finishes
```

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

### `Answer(opts ...CallOption) (map[string]any, error)`

Answer an inbound call.

```go
call.Answer()
```

### `Hangup(reason string) (map[string]any, error)`

End the call.

```go
call.Hangup("")
call.Hangup("busy")
```

### `Pass() (map[string]any, error)`

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
	relay.WithRecordAudio(map[string]any{"format": "wav", "stereo": true, "direction": "both"}),
)
// ... later ...
action.Stop()
event := action.Wait(context.Background())
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
event := action.Wait(context.Background())
result, _ := event.Params["result"].(map[string]any)
params, _ := result["params"].(map[string]any)
digit, _ := params["digits"].(string)
```

### `Collect(opts ...CollectOption) *StandaloneCollectAction`

Collect input without playing audio.

```go
action := call.Collect(
	relay.WithCollectDigits(map[string]any{"max": 4, "terminators": "#"}),
	relay.WithCollectSpeech(map[string]any{"language": "en-US"}),
	relay.WithCollectPartialResults(true),
)
event := action.Wait(context.Background())
```

## Bridging

### `Connect(devices [][]map[string]any, opts ...ConnectOption) (map[string]any, error)`

Bridge the call to another destination.

```go
call.Connect(
	[][]map[string]any{
		{{"type": "phone", "params": map[string]any{"to_number": "+15551234567", "from_number": "+15559876543"}}},
	},
	relay.WithConnectRingback([]map[string]any{{"type": "ringtone", "params": map[string]any{"name": "us"}}}),
)
```

### `Disconnect() (map[string]any, error)`

Unbridge a connected call.

```go
call.Disconnect()
```

## DTMF

### `SendDigits(digits string, opts ...DigitOption) (map[string]any, error)`

Send DTMF tones.

```go
call.SendDigits("1234#")
```

## Detection

### `Detect(detect map[string]any, opts ...DetectOption) *DetectAction`

Detect machine, fax, or digits.

```go
action := call.Detect(
	map[string]any{"type": "machine"},
	relay.WithDetectTimeout(30.0),
)
event := action.Wait(context.Background())
```

## SIP Refer

### `Refer(device map[string]any, opts ...ReferOption) (map[string]any, error)`

Transfer via SIP REFER.

```go
call.Refer(map[string]any{"type": "sip", "params": map[string]any{"to": "sip:user@example.com"}})
```

## Transfer

### `Transfer(dest string) (map[string]any, error)`

Transfer call control to another RELAY app or SWML script.

```go
call.Transfer("https://example.com/swml-endpoint")
```

## Fax

### `SendFax(document string, opts ...FaxOption) *FaxAction`

```go
action := call.SendFax("https://example.com/document.pdf",
	relay.WithFaxIdentity("+15551234567"),
)
event := action.Wait(context.Background())
```

### `ReceiveFax(opts ...FaxOption) *FaxAction`

```go
action := call.ReceiveFax()
event := action.Wait(context.Background())
```

## Tap (Media Interception)

### `Tap(tap, device map[string]any, opts ...TapOption) *TapAction`

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
action := call.Pay("https://pay.example.com",
	relay.WithPayChargeAmount("25.99"),
	relay.WithPayCurrency("usd"),
	relay.WithPayInputMethod("dtmf"),
)
event := action.Wait(context.Background())
```

## Conference

### `JoinConference(name string, opts ...ConferenceOption) (map[string]any, error)`

```go
call.JoinConference("my_conference",
	relay.WithConferenceMuted(false),
	relay.WithConferenceBeep("onEnter"),
)
```

### `LeaveConference(conferenceID string) (map[string]any, error)`

```go
call.LeaveConference("conf-123")
```

## Hold

### `Hold() (map[string]any, error)` / `Unhold() (map[string]any, error)`

```go
call.Hold()
// ... later ...
call.Unhold()
```

## Denoise

### `Denoise() (map[string]any, error)` / `DenoiseStop() (map[string]any, error)`

```go
call.Denoise()
// ... later ...
call.DenoiseStop()
```

## Transcription

### `Transcribe(opts ...TranscribeOption) *TranscribeAction`

```go
action := call.Transcribe(
	relay.WithTranscribeStatusURL("https://example.com/transcription"),
)
// ... later ...
action.Stop()
```

## Live Transcribe / Translate

### `LiveTranscribe(actionObj map[string]any) (map[string]any, error)`

```go
call.LiveTranscribe(map[string]any{"start": map[string]any{"language": "en-US"}})
```

### `LiveTranslate(actionObj map[string]any, opts ...TranslateOption) (map[string]any, error)`

```go
call.LiveTranslate(map[string]any{"start": map[string]any{"source": "en-US", "target": "es"}})
```

## Echo

### `Echo(opts ...EchoOption) (map[string]any, error)`

Echo audio back to the caller (useful for testing).

```go
call.Echo(relay.WithEchoTimeout(30.0))
```

## AI Agent

### `AI(opts ...AIOption) *AIAction`

Start an AI agent session on the call.

```go
action := call.AI(
	relay.WithAIPrompt(map[string]any{"text": "You are a helpful support agent."}),
	relay.WithAISWAIG(map[string]any{"functions": []any{...}}),
	relay.WithAIParams(map[string]any{"end_of_speech_timeout": 3000}),
)
event := action.Wait(context.Background())
```

### `AmazonBedrock(opts ...AIOption) (map[string]any, error)`

Connect to an Amazon Bedrock AI agent.

### `AIMessage(opts ...AIMessageOption) (map[string]any, error)`

Send a message to an active AI session.

### `AIHold(opts ...AIHoldOption) (map[string]any, error)` / `AIUnhold(opts ...AIUnholdOption) (map[string]any, error)`

Put an AI session on/off hold.

## Rooms

### `JoinRoom(name string, opts ...RoomOption) (map[string]any, error)`

```go
call.JoinRoom("my_room")
```

### `LeaveRoom() (map[string]any, error)`

```go
call.LeaveRoom()
```

## Queue

### `QueueEnter(queueName string, opts ...QueueOption) (map[string]any, error)`

```go
call.QueueEnter("support")
```

### `QueueLeave(queueName string, opts ...QueueOption) (map[string]any, error)`

```go
call.QueueLeave("support", relay.WithQueueID("q-123"))
```

## Digit Bindings

### `BindDigit(digits, bindMethod string, opts ...BindDigitOption) (map[string]any, error)`

Bind a DTMF sequence to trigger a RELAY method.

```go
call.BindDigit("*1", "calling.play",
	relay.WithBindParams(map[string]any{
		"play": []map[string]any{{"type": "tts", "params": map[string]any{"text": "You pressed star-1"}}},
	}),
)
```

### `ClearDigitBindings(opts ...BindDigitOption) (map[string]any, error)`

```go
call.ClearDigitBindings()
```

## User Events

### `UserEvent(opts ...UserEventOption) (map[string]any, error)`

Send a custom event.

```go
call.UserEvent(
	relay.WithUserEventName("order_placed"),
	relay.WithUserEventData(map[string]any{"order_id": "12345"}),
)
```

## Event Handling

### `On(eventType string, handler func(*relay.RelayEvent))`

Register an event listener on this call.

```go
call.On("calling.call.play", func(event *relay.RelayEvent) {
	fmt.Printf("Play state: %v\n", event.Params["state"])
})
```

### `WaitFor(eventType string, opts ...WaitOption) (*relay.RelayEvent, error)`

Wait for a specific event.

```go
event, err := call.WaitFor("calling.call.play",
	relay.WithWaitTimeout(30 * time.Second),
)
```

### `WaitForEnded(opts ...WaitOption) (*relay.RelayEvent, error)`

Wait for the call to end.

```go
event, err := call.WaitForEnded()
fmt.Printf("End reason: %v\n", event.Params["end_reason"])
```
