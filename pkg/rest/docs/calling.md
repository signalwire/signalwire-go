# Calling Commands

The Calling API provides REST-based call control. All commands are dispatched via a single `POST /api/calling/calls` endpoint with a `command` field. No WebSocket connection is needed.

## How It Works

Every method on `client.Calling` sends a POST request with this structure:

```json
{
    "command": "calling.play",
    "id": "<call-uuid>",
    "params": { ... }
}
```

For `Dial` and `Update`, the call details are inside `params` (no top-level `id`). For all other commands, `id` is the UUID of the call to control.

Each method takes a typed `namespaces.CallingNamespace<Op>Params` struct. Optional
fields are pointers; use a small helper to take their address:

```go
func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
func floatPtr(f float64) *float64 { return &f }
```

Any parameter not modeled as a typed field can be passed through the
`Extras map[string]any` field present on every params struct.

## Call Lifecycle

### `Dial(params) (*CallResponse, error)`

Initiate an outbound call.

```go
result, err := client.Calling.Dial(namespaces.CallingNamespaceDialParams{
	From: "+15559876543",
	To:   "+15551234567",
	Url:  strPtr("https://example.com/call-handler"),
})
// CallResponse is decoded JSON; assert to a map to read fields.
call, _ := (*result).(map[string]any)
callID, _ := call["id"].(string)
```

### `Update(params) (*CallResponse, error)`

Update an active call's dialplan mid-call.

```go
_, err := client.Calling.Update(namespaces.CallingNamespaceUpdateParams{
	Id:  callID,
	Url: strPtr("https://example.com/new-handler"),
})
```

### `End(callID, params) (*CallResponse, error)`

Terminate a call.

```go
_, err := client.Calling.End(callID, namespaces.CallingNamespaceEndParams{
	Extras: map[string]any{"reason": "hangup"},
})
```

### `Transfer(callID, params) (*CallResponse, error)`

Transfer a call to a new destination.

```go
_, err := client.Calling.Transfer(callID, namespaces.CallingNamespaceTransferParams{
	Dest: "sip:agent@example.com",
})
```

### `Disconnect(callID, params) (*CallResponse, error)`

Disconnect bridged calls without hanging up either leg.

```go
_, err := client.Calling.Disconnect(callID, namespaces.CallingNamespaceDisconnectParams{})
```

## Audio Playback

### `Play(callID, params) (*CallResponse, error)`

Play audio, TTS, silence, or ringtone.

```go
_, err := client.Calling.Play(callID, namespaces.CallingNamespacePlayParams{
	Play:   []map[string]any{{"type": "tts", "text": "Hello!"}},
	Volume: floatPtr(5.0),
})
```

### `PlayPause(callID, params)` / `PlayResume(callID, params)`

Pause or resume active playback.

```go
_, err := client.Calling.PlayPause(callID, namespaces.CallingNamespacePlayPauseParams{ControlId: "ctrl-1"})
_, err = client.Calling.PlayResume(callID, namespaces.CallingNamespacePlayResumeParams{ControlId: "ctrl-1"})
```

### `PlayStop(callID, params)`

Stop active playback.

```go
_, err := client.Calling.PlayStop(callID, namespaces.CallingNamespacePlayStopParams{ControlId: "ctrl-1"})
```

### `PlayVolume(callID, params)`

Adjust playback volume.

```go
_, err := client.Calling.PlayVolume(callID, namespaces.CallingNamespacePlayVolumeParams{
	ControlId: "ctrl-1",
	Volume:    -3.0,
})
```

## Recording

### `Record(callID, params)` / `RecordPause` / `RecordResume` / `RecordStop`

```go
_, err := client.Calling.Record(callID, namespaces.CallingNamespaceRecordParams{
	ControlId: strPtr("rec-1"),
	Audio:     map[string]any{"beep": true, "format": "wav", "stereo": true},
})
_, err = client.Calling.RecordPause(callID, namespaces.CallingNamespaceRecordPauseParams{ControlId: "rec-1"})
_, err = client.Calling.RecordResume(callID, namespaces.CallingNamespaceRecordResumeParams{ControlId: "rec-1"})
_, err = client.Calling.RecordStop(callID, namespaces.CallingNamespaceRecordStopParams{ControlId: "rec-1"})
```

## Input Collection

### `Collect(callID, params)` / `CollectStop` / `CollectStartInputTimers`

```go
_, err := client.Calling.Collect(callID, namespaces.CallingNamespaceCollectParams{
	ControlId: strPtr("coll-1"),
	Digits:    map[string]any{"max": 4, "terminators": "#"},
	Speech:    map[string]any{"end_silence_timeout": 2.0},
})
_, err = client.Calling.CollectStop(callID, namespaces.CallingNamespaceCollectStopParams{ControlId: "coll-1"})
_, err = client.Calling.CollectStartInputTimers(callID, namespaces.CallingNamespaceCollectStartInputTimersParams{ControlId: "coll-1"})
```

## Detection

### `Detect(callID, params)` / `DetectStop`

```go
_, err := client.Calling.Detect(callID, namespaces.CallingNamespaceDetectParams{
	ControlId: strPtr("det-1"),
	Detect:    map[string]any{"type": "machine", "params": map[string]any{"initial_timeout": 4.5}},
})
_, err = client.Calling.DetectStop(callID, namespaces.CallingNamespaceDetectStopParams{ControlId: "det-1"})
```

## Tap & Stream

### `Tap(callID, params)` / `TapStop`

```go
_, err := client.Calling.Tap(callID, namespaces.CallingNamespaceTapParams{
	ControlId: strPtr("tap-1"),
	Tap:       map[string]any{"type": "audio", "params": map[string]any{"direction": "both"}},
	Device:    map[string]any{"type": "rtp", "params": map[string]any{"addr": "192.168.1.1", "port": 1234}},
})
_, err = client.Calling.TapStop(callID, namespaces.CallingNamespaceTapStopParams{ControlId: "tap-1"})
```

### `Stream(callID, params)` / `StreamStop`

```go
_, err := client.Calling.Stream(callID, namespaces.CallingNamespaceStreamParams{
	Url:       "wss://example.com/audio-stream",
	ControlId: strPtr("str-1"),
	Codec:     strPtr("PCMU"),
})
_, err = client.Calling.StreamStop(callID, namespaces.CallingNamespaceStreamStopParams{ControlId: "str-1"})
```

## Denoise

### `Denoise(callID, params)` / `DenoiseStop(callID, params)`

```go
_, err := client.Calling.Denoise(callID, namespaces.CallingNamespaceDenoiseParams{})
_, err = client.Calling.DenoiseStop(callID, namespaces.CallingNamespaceDenoiseStopParams{})
```

## Transcription

### `Transcribe(callID, params)` / `TranscribeStop`

```go
_, err := client.Calling.Transcribe(callID, namespaces.CallingNamespaceTranscribeParams{
	ControlId: strPtr("tx-1"),
	StatusUrl: strPtr("https://example.com/hook"),
})
_, err = client.Calling.TranscribeStop(callID, namespaces.CallingNamespaceTranscribeStopParams{ControlId: "tx-1"})
```

## AI

### `AIMessage(callID, params)`

Inject a message into an active AI session.

```go
_, err := client.Calling.AIMessage(callID, namespaces.CallingNamespaceAIMessageParams{
	Role:        strPtr("user"),
	MessageText: strPtr("Transfer me to billing"),
})
```

### `AIHold(callID, params)` / `AIUnhold(callID, params)`

```go
_, err := client.Calling.AIHold(callID, namespaces.CallingNamespaceAIHoldParams{
	Timeout: intPtr(60),
	Prompt:  strPtr("Please wait while I transfer you."),
})
_, err = client.Calling.AIUnhold(callID, namespaces.CallingNamespaceAIUnholdParams{
	Prompt: strPtr("I'm back, how can I help?"),
})
```

### `AIStop(callID, params)`

```go
_, err := client.Calling.AIStop(callID, namespaces.CallingNamespaceAIStopParams{ControlId: "ai-1"})
```

## Live Transcribe & Translate

```go
_, err := client.Calling.LiveTranscribe(callID, namespaces.CallingNamespaceLiveTranscribeParams{
	Action: "start",
	Extras: map[string]any{"lang": "en"},
})
_, err = client.Calling.LiveTranslate(callID, namespaces.CallingNamespaceLiveTranslateParams{
	Action: "start",
	Extras: map[string]any{"from_lang": "en", "to_lang": "es"},
})
```

## Fax

```go
_, err := client.Calling.SendFaxStop(callID, namespaces.CallingNamespaceSendFaxStopParams{ControlId: "fax-1"})
_, err = client.Calling.ReceiveFaxStop(callID, namespaces.CallingNamespaceReceiveFaxStopParams{ControlId: "fax-1"})
```

## SIP & Custom Events

```go
// SIP REFER transfer
_, err := client.Calling.Refer(callID, namespaces.CallingNamespaceReferParams{
	Device: map[string]any{"to": "sip:agent@example.com"},
})

// Custom event
_, err = client.Calling.UserEvent(callID, namespaces.CallingNamespaceUserEventParams{
	Event: map[string]any{"type": "custom", "data": map[string]any{"key": "value"}},
})
```

## Complete Method List

| Method | Command | Requires callID |
|--------|---------|:-:|
| `Dial(params)` | `dial` | No |
| `Update(params)` | `update` | No |
| `End(callID, params)` | `calling.end` | Yes |
| `Transfer(callID, params)` | `calling.transfer` | Yes |
| `Disconnect(callID, params)` | `calling.disconnect` | Yes |
| `Play(callID, params)` | `calling.play` | Yes |
| `PlayPause(callID, params)` | `calling.play.pause` | Yes |
| `PlayResume(callID, params)` | `calling.play.resume` | Yes |
| `PlayStop(callID, params)` | `calling.play.stop` | Yes |
| `PlayVolume(callID, params)` | `calling.play.volume` | Yes |
| `Record(callID, params)` | `calling.record` | Yes |
| `RecordPause(callID, params)` | `calling.record.pause` | Yes |
| `RecordResume(callID, params)` | `calling.record.resume` | Yes |
| `RecordStop(callID, params)` | `calling.record.stop` | Yes |
| `Collect(callID, params)` | `calling.collect` | Yes |
| `CollectStop(callID, params)` | `calling.collect.stop` | Yes |
| `CollectStartInputTimers(callID, params)` | `calling.collect.start_input_timers` | Yes |
| `Detect(callID, params)` | `calling.detect` | Yes |
| `DetectStop(callID, params)` | `calling.detect.stop` | Yes |
| `Tap(callID, params)` | `calling.tap` | Yes |
| `TapStop(callID, params)` | `calling.tap.stop` | Yes |
| `Stream(callID, params)` | `calling.stream` | Yes |
| `StreamStop(callID, params)` | `calling.stream.stop` | Yes |
| `Denoise(callID, params)` | `calling.denoise` | Yes |
| `DenoiseStop(callID, params)` | `calling.denoise.stop` | Yes |
| `Transcribe(callID, params)` | `calling.transcribe` | Yes |
| `TranscribeStop(callID, params)` | `calling.transcribe.stop` | Yes |
| `AIMessage(callID, params)` | `calling.ai_message` | Yes |
| `AIHold(callID, params)` | `calling.ai_hold` | Yes |
| `AIUnhold(callID, params)` | `calling.ai_unhold` | Yes |
| `AIStop(callID, params)` | `calling.ai.stop` | Yes |
| `LiveTranscribe(callID, params)` | `calling.live_transcribe` | Yes |
| `LiveTranslate(callID, params)` | `calling.live_translate` | Yes |
| `SendFaxStop(callID, params)` | `calling.send_fax.stop` | Yes |
| `ReceiveFaxStop(callID, params)` | `calling.receive_fax.stop` | Yes |
| `Refer(callID, params)` | `calling.refer` | Yes |
| `UserEvent(callID, params)` | `calling.user_event` | Yes |
