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

## Call Lifecycle

### `Dial(params map[string]any) (map[string]any, error)`

Initiate an outbound call.

```go
result, err := client.Calling.Dial(map[string]any{
	"from": "+15559876543",
	"to":   "+15551234567",
	"url":  "https://example.com/call-handler",
})
if err != nil {
	fmt.Printf("Dial failed: %v\n", err)
	return
}
callID, _ := result["id"].(string)
```

### `Update(params map[string]any) (map[string]any, error)`

Update an active call's dialplan mid-call.

```go
client.Calling.Update(map[string]any{"id": callID, "url": "https://example.com/new-handler"})
```

### `End(callID string, params map[string]any) (map[string]any, error)`

Terminate a call.

```go
client.Calling.End(callID, map[string]any{"reason": "hangup"})
```

### `Transfer(callID string, params map[string]any) (map[string]any, error)`

Transfer a call to a new destination.

```go
client.Calling.Transfer(callID, map[string]any{"dest": "sip:agent@example.com"})
```

### `Disconnect(callID string) (map[string]any, error)`

Disconnect bridged calls without hanging up either leg.

```go
client.Calling.Disconnect(callID)
```

## Audio Playback

### `Play(callID string, params map[string]any) (map[string]any, error)`

Play audio, TTS, silence, or ringtone.

```go
client.Calling.Play(callID, map[string]any{
	"play":   []map[string]any{{"type": "tts", "text": "Hello!"}},
	"volume": 5.0,
})
```

### `PlayPause` / `PlayResume`

Pause or resume active playback.

```go
client.Calling.PlayPause(callID, map[string]any{"control_id": "ctrl-1"})
client.Calling.PlayResume(callID, map[string]any{"control_id": "ctrl-1"})
```

### `PlayStop`

Stop active playback.

```go
client.Calling.PlayStop(callID, map[string]any{"control_id": "ctrl-1"})
```

### `PlayVolume`

Adjust playback volume.

```go
client.Calling.PlayVolume(callID, map[string]any{"control_id": "ctrl-1", "volume": -3.0})
```

## Recording

### `Record` / `RecordPause` / `RecordResume` / `RecordStop`

```go
client.Calling.Record(callID, map[string]any{
	"control_id": "rec-1",
	"audio":      map[string]any{"beep": true, "format": "wav", "stereo": true},
})
client.Calling.RecordPause(callID, map[string]any{"control_id": "rec-1"})
client.Calling.RecordResume(callID, map[string]any{"control_id": "rec-1"})
client.Calling.RecordStop(callID, map[string]any{"control_id": "rec-1"})
```

## Input Collection

### `Collect` / `CollectStop` / `CollectStartInputTimers`

```go
client.Calling.Collect(callID, map[string]any{
	"control_id": "coll-1",
	"digits":     map[string]any{"max": 4, "terminators": "#"},
	"speech":     map[string]any{"end_silence_timeout": 2.0},
})
client.Calling.CollectStop(callID, map[string]any{"control_id": "coll-1"})
client.Calling.CollectStartInputTimers(callID, map[string]any{"control_id": "coll-1"})
```

## Detection

### `Detect` / `DetectStop`

```go
client.Calling.Detect(callID, map[string]any{
	"control_id": "det-1",
	"detect":     map[string]any{"type": "machine", "params": map[string]any{"initial_timeout": 4.5}},
})
client.Calling.DetectStop(callID, map[string]any{"control_id": "det-1"})
```

## Tap & Stream

### `Tap` / `TapStop`

```go
client.Calling.Tap(callID, map[string]any{
	"control_id": "tap-1",
	"tap":        map[string]any{"type": "audio", "params": map[string]any{"direction": "both"}},
	"device":     map[string]any{"type": "rtp", "params": map[string]any{"addr": "192.168.1.1", "port": 1234}},
})
client.Calling.TapStop(callID, map[string]any{"control_id": "tap-1"})
```

### `Stream` / `StreamStop`

```go
client.Calling.Stream(callID, map[string]any{
	"control_id": "str-1",
	"url":        "wss://example.com/audio-stream",
	"codec":      "PCMU",
})
client.Calling.StreamStop(callID, map[string]any{"control_id": "str-1"})
```

## Denoise

### `Denoise` / `DenoiseStop`

```go
client.Calling.Denoise(callID)
client.Calling.DenoiseStop(callID)
```

## Transcription

### `Transcribe` / `TranscribeStop`

```go
client.Calling.Transcribe(callID, map[string]any{
	"control_id": "tx-1",
	"status_url": "https://example.com/hook",
})
client.Calling.TranscribeStop(callID, map[string]any{"control_id": "tx-1"})
```

## AI

### `AIMessage`

Inject a message into an active AI session.

```go
client.Calling.AIMessage(callID, map[string]any{
	"role": "user", "message_text": "Transfer me to billing",
})
```

### `AIHold` / `AIUnhold`

```go
client.Calling.AIHold(callID, map[string]any{
	"timeout": 60, "prompt": "Please wait while I transfer you.",
})
client.Calling.AIUnhold(callID, map[string]any{
	"prompt": "I'm back, how can I help?",
})
```

### `AIStop`

```go
client.Calling.AIStop(callID, map[string]any{"control_id": "ai-1"})
```

## Live Transcribe & Translate

```go
client.Calling.LiveTranscribe(callID, map[string]any{"action": "start", "lang": "en"})
client.Calling.LiveTranslate(callID, map[string]any{
	"action": "start", "from_lang": "en", "to_lang": "es",
})
```

## Fax

```go
client.Calling.SendFaxStop(callID, map[string]any{"control_id": "fax-1"})
client.Calling.ReceiveFaxStop(callID, map[string]any{"control_id": "fax-1"})
```

## SIP & Custom Events

```go
// SIP REFER transfer
client.Calling.Refer(callID, map[string]any{
	"device": map[string]any{"to": "sip:agent@example.com"},
})

// Custom event
client.Calling.UserEvent(callID, map[string]any{
	"event": map[string]any{"type": "custom", "data": map[string]any{"key": "value"}},
})
```

## Complete Method List

| Method | Command | Requires callID |
|--------|---------|:-:|
| `Dial(params)` | `dial` | No |
| `Update(params)` | `update` | No |
| `End(callID, params)` | `calling.end` | Yes |
| `Transfer(callID, params)` | `calling.transfer` | Yes |
| `Disconnect(callID)` | `calling.disconnect` | Yes |
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
| `Denoise(callID)` | `calling.denoise` | Yes |
| `DenoiseStop(callID)` | `calling.denoise.stop` | Yes |
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
