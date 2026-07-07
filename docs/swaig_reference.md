# SwaigFunctionResult Methods Reference

SWAIG (SignalWire AI Gateway) is the platform's AI tool-calling system -- it connects the AI's decisions to actions like call transfers, SMS, recordings, and API calls, with native access to the media stack. This document provides a complete reference for all methods available on the `swaig.FunctionResult` type (package `github.com/signalwire/signalwire-go/pkg/swaig`). These methods provide convenient abstractions for SWAIG actions, eliminating the need to manually construct action JSON objects.

Construct a result with `swaig.NewFunctionResult(response string)`. Every action method returns the receiver (`*swaig.FunctionResult`) so calls can be chained.

<!-- snippet-setup -->
```go
import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// Shared context the fragments below assume.
var result = swaig.NewFunctionResult("")
var a = agent.NewAgentBase()

var (
	_ = fmt.Sprint
	_ = a
	_ = result
)
```

## Core Methods

### Basic Construction & Control

#### `NewFunctionResult(response string) *FunctionResult`
Creates a new result object with optional response text. Post-processing is off by default; call `SetPostProcess(true)` to enable it.

```go
result = swaig.NewFunctionResult("Hello, I'll help you with that")
result = swaig.NewFunctionResult("Processing request...").SetPostProcess(true)
```

#### `SetResponse(response string) *FunctionResult`
Sets or updates the response text that the AI will speak.

```go
result.SetResponse("I've updated your information")
```

#### `SetPostProcess(postProcess bool) *FunctionResult`
Controls whether AI gets one more turn before executing actions.

```go
result.SetPostProcess(true)  // AI speaks response before executing actions
result.SetPostProcess(false) // Actions execute immediately
```

---

## Action Methods

### Call Control Actions

#### `ExecuteSwml(swmlContent any, transfer bool) *FunctionResult`
Execute SWML content with flexible input support and optional transfer behavior. `swmlContent` accepts a raw JSON string or a `map[string]any` document.

```go
// Raw SWML string
result.ExecuteSwml(`{"version":"1.0.0","sections":{"main":[{"say":"Hello"}]}}`, false)

// SWML document as a map
swmlDoc := map[string]any{
	"version": "1.0.0",
	"sections": map[string]any{
		"main": []any{map[string]any{"say": "Hello"}},
	},
}
result.ExecuteSwml(swmlDoc, true)
```

#### `Connect(destination string, final bool, from string) *FunctionResult`
Transfer/connect call to another destination using SWML. Pass `from = ""` to omit the from-address.

```go
result.Connect("+15551234567", true, "")                        // Permanent transfer
result.Connect("support@company.com", false, "+15559876543")    // Temporary transfer
```

#### `SendSms(toNumber, fromNumber, body string, media []string, tags []string, region string) *FunctionResult`
**[HELPER METHOD]** - Send SMS message to PSTN phone number using SWML. Pass `""` / `nil` for optional arguments you want to omit.

```go
// Simple text message
result.SendSms(
	"+15551234567", // toNumber
	"+15559876543", // fromNumber
	"Your order has been confirmed!", // body
	nil, nil, "",
)

// Media message with images (no body)
result.SendSms(
	"+15551234567",
	"+15559876543",
	"",
	[]string{"https://example.com/receipt.jpg", "https://example.com/map.png"},
	nil, "",
)

// Full featured message with tags and region
result.SendSms(
	"+15551234567",
	"+15559876543",
	"Order update with receipt attached",
	[]string{"https://example.com/receipt.pdf"},
	[]string{"order", "confirmation", "customer"},
	"us",
)
```

**Parameters:**
- `to_number` (required): Phone number in E.164 format to send to
- `from_number` (required): Phone number in E.164 format to send from
- `body` (optional): Message text (required if no media)
- `media` (optional): Array of URLs to send (required if no body)
- `tags` (optional): Array of tags for UI searching
- `region` (optional): Region to originate message from

**Variables Set:**
- `send_sms_result`: "success" or "failed"

#### `Pay(connectorURL string, opts *PayOptions) *FunctionResult`
**[HELPER METHOD]** - Process payments using SWML pay action with extensive customization. `connectorURL` is required; pass a `*swaig.PayOptions` (or `nil` for all defaults) for the rest.

```go
// Simple payment setup
result.Pay("https://api.example.com/accept-payment", &swaig.PayOptions{
	ChargeAmount: "10.99",
	Description:  "Monthly subscription",
})

// Advanced payment with custom prompts.
// Build prompt actions with the package helper functions.
welcomeActions := []map[string]string{
	swaig.CreatePaymentAction("Say", "Welcome to our payment system"),
	swaig.CreatePaymentAction("Say", "Please enter your credit card number"),
}
cardPrompt := swaig.CreatePaymentPrompt("payment-card-number", welcomeActions, "", "")

errorActions := []map[string]string{
	swaig.CreatePaymentAction("Say", "Invalid card number, please try again"),
}
errorPrompt := swaig.CreatePaymentPrompt(
	"payment-card-number",
	errorActions,
	"",                          // cardType
	"invalid-card-number timeout", // errorType
)

// Create payment parameters.
params := []map[string]string{
	swaig.CreatePaymentParameter("customer_id", "12345"),
	swaig.CreatePaymentParameter("order_id", "ORD-789"),
}

// Full payment configuration.
falseVal := false
result.Pay("https://api.example.com/accept-payment", &swaig.PayOptions{
	StatusURL:       "https://api.example.com/payment-status",
	Timeout:         10,
	MaxAttempts:     3,
	SecurityCode:    true,
	SecurityCodeSet: true,
	PostalCode:      falseVal,
	TokenType:       "one-time",
	ChargeAmount:    "25.50",
	Currency:        "usd",
	Language:        "en-US",
	Voice:           "polly.Sally",
	Description:     "Premium service upgrade",
	ValidCardTypes:  "visa mastercard amex",
	Parameters:      params,
	Prompts:         []map[string]any{cardPrompt, errorPrompt},
})
```

**Core Parameters:**
- `payment_connector_url` (required): URL to process payment requests
- `input_method`: "dtmf" or "voice" (default: "dtmf")
- `payment_method`: "credit-card" (default: "credit-card")
- `timeout`: Seconds to wait for input (default: 5)
- `max_attempts`: Number of retry attempts (default: 1)

**Security & Validation:**
- `security_code`: Prompt for CVV (default: True)
- `postal_code`: Prompt for postal code or provide known code (default: True)
- `min_postal_code_length`: Minimum postal code digits (default: 0)
- `valid_card_types`: Space-separated card types (default: "visa mastercard amex")

**Payment Configuration:**
- `token_type`: "one-time" or "reusable" (default: "reusable")
- `charge_amount`: Amount as decimal string
- `currency`: Currency code (default: "usd")
- `description`: Payment description

**Customization:**
- `language`: Prompt language (default: "en-US")
- `voice`: TTS voice (default: "woman")
- `status_url`: URL for status notifications
- `parameters`: Additional name/value pairs for connector
- `prompts`: Custom prompt configurations

**Helper Functions for Payment Setup:**
```go
// Create payment action
action := swaig.CreatePaymentAction("Say", "Enter card number")

// Create payment prompt
prompt := swaig.CreatePaymentPrompt(
	"payment-card-number",
	[]map[string]string{action},
	"",                     // cardType
	"invalid-card-number", // errorType
)

// Create payment parameter
param := swaig.CreatePaymentParameter("customer_id", "12345")
_, _, _ = action, prompt, param
```

**Variables Set:**
- `pay_result`: "success", "too-many-failed-attempts", "payment-connector-error", etc.
- `pay_payment_results`: JSON with payment details including tokens and card info

#### `RecordCall(controlID string, stereo bool, format RecordFormat, direction RecordDirection, opts *RecordCallOptions) *FunctionResult`
**[HELPER METHOD]** - Start background call recording using SWML.

Unlike foreground recording, the script continues executing while recording happens in the background. `format` and `direction` are defined string types (`swaig.FormatWAV`/`swaig.FormatMP3`; `swaig.RecordDirectionBoth`/`Speak`/`Listen`), but bare string literals also compile.

```go
// Simple background recording (all defaults)
result.RecordCall("", false, swaig.FormatWAV, swaig.RecordDirectionBoth, nil)

// Recording with custom settings
result.RecordCall("support_call_001", true, swaig.FormatMP3, swaig.RecordDirectionBoth,
	&swaig.RecordCallOptions{
		MaxLength:    300, // 5 minutes max
		MaxLengthSet: true,
	})

// Recording with terminator and status webhook
result.RecordCall("customer_voicemail", false, swaig.FormatWAV, swaig.RecordDirectionSpeak, // Only record customer voice
	&swaig.RecordCallOptions{
		Terminators:       "#",   // Stop on '#' press
		Beep:              true,  // Play beep before recording
		InitialTimeout:    4.0,   // Wait 4 seconds for speech
		InitialTimeoutSet: true,
		EndSilenceTimeout:    3.0, // Stop after 3 seconds of silence
		EndSilenceTimeoutSet: true,
		StatusURL:            "https://api.example.com/recording-status",
	})
```

**Core Parameters:**
- `control_id` (optional): Identifier for this recording (for use with stop_record_call)
- `stereo`: Record in stereo (default: False)
- `format`: "wav" or "mp3" (default: "wav")
- `direction`: "speak", "listen", or "both" (default: "both")

**Control Options:**
- `terminators`: Digits that stop recording when pressed
- `beep`: Play beep before recording (default: False)
- `max_length`: Maximum recording length in seconds

**Timing Options:**
- `input_sensitivity`: Input sensitivity (default: 44.0)
- `initial_timeout`: Time to wait for speech start (default: 0.0)
- `end_silence_timeout`: Time to wait in silence before ending (default: 0.0)

**Webhook Options:**
- `status_url`: URL to send recording status events to

**Variables Set:**
- `record_call_result`: "success" or "failed"
- `record_call_url`: URL of recorded file (when recording completes)

#### `StopRecordCall(controlID string) *FunctionResult`
**[HELPER METHOD]** - Stop an active background call recording using SWML. Pass `""` to stop the most recent recording.

```go
// Stop the most recent recording
result.StopRecordCall("")

// Stop specific recording by ID
result.StopRecordCall("support_call_001")

// Chain to stop recording and provide feedback
result.StopRecordCall("customer_voicemail").
	Say("Thank you, your message has been recorded")
```

**Parameters:**
- `control_id` (optional): Identifier for recording to stop. If not provided, stops the most recent recording.

**Variables Set:**
- `stop_record_call_result`: "success" or "failed"

#### `JoinRoom(name string) *FunctionResult`
**[HELPER METHOD]** - Join a RELAY room using SWML.

RELAY rooms enable multi-party communication and collaboration features.

```go
// Join a conference room
result.JoinRoom("support_team_room")

// Join customer meeting room
result.JoinRoom("customer_meeting_001").
	Say("Welcome to the customer meeting room")

// Join room and set metadata
result.JoinRoom("sales_conference").
	SetMetadata(map[string]any{"participant_role": "moderator", "join_time": "2024-01-01T12:00:00Z"})
```

**Parameters:**
- `name` (required): The name of the room to join

**Variables Set:**
- `join_room_result`: "success" or "failed"

#### `SIPRefer(toURI string) *FunctionResult`
**[HELPER METHOD]** - Send SIP REFER for call transfer using SWML.

SIP REFER is used for call transfer in SIP environments, allowing one endpoint to request another to initiate a new connection.

```go
// Basic SIP refer to transfer call
result.SIPRefer("sip:support@company.com")

// Transfer to specific SIP address with domain
result.SIPRefer("sip:agent123@pbx.company.com:5060")

// Chain with announcement
result.Say("Transferring your call to our specialist").
	SIPRefer("sip:specialist@company.com")
```

**Parameters:**
- `to_uri` (required): The SIP URI to send the REFER to

**Variables Set:**
- `sip_refer_result`: "success" or "failed"

#### `JoinConference(name string, opts *JoinConferenceOptions) *FunctionResult`
**[HELPER METHOD]** - Join an ad-hoc audio conference with RELAY and CXML calls using SWML.

Provides extensive configuration options (via `*swaig.JoinConferenceOptions`, or `nil` for defaults) for conference call management and recording.

```go
// Simple conference join
result.JoinConference("my_conference", nil)

// Basic conference with recording
result.JoinConference("daily_standup", &swaig.JoinConferenceOptions{
	Record:          "record-from-start",
	MaxParticipants: 10,
})

// Advanced conference with callbacks and coaching
startOnEnter := true
result.JoinConference("customer_support_conf", &swaig.JoinConferenceOptions{
	Muted:               false,
	Beep:                "onEnter",
	StartOnEnter:        &startOnEnter,
	EndOnExit:           false,
	MaxParticipants:     50,
	Record:              "record-from-start",
	Region:              "us-east",
	Trim:                "trim-silence",
	StatusCallback:      "https://api.company.com/conference-events",
	StatusCallbackEvent: "start end join leave",
	RecordingStatusCallback: "https://api.company.com/recording-events",
})

// Chain with other actions
result.Say("Joining you to the team conference").
	JoinConference("team_meeting", nil).
	SetMetadata(map[string]any{"meeting_type": "team_sync", "participant_role": "attendee"})
```

**Core Parameters:**
- `name` (required): Name of conference to join
- `muted`: Join muted (default: False)
- `beep`: Beep configuration - "true", "false", "onEnter", "onExit" (default: "true")
- `start_on_enter`: Conference starts when this participant enters (default: True)
- `end_on_exit`: Conference ends when this participant exits (default: False)

**Capacity & Region:**
- `max_participants`: Maximum participants <= 250 (default: 250)
- `region`: Conference region for optimization
- `wait_url`: SWML URL for custom hold music

**Recording Options:**
- `record`: "do-not-record" or "record-from-start" (default: "do-not-record")
- `trim`: "trim-silence" or "do-not-trim" (default: "trim-silence")
- `recording_status_callback`: URL for recording status events
- `recording_status_callback_method`: "GET" or "POST" (default: "POST")
- `recording_status_callback_event`: "in-progress completed absent" (default: "completed")

**Status & Coaching:**
- `coach`: SWML Call ID or CXML CallSid for coaching features
- `status_callback`: URL for conference status events
- `status_callback_method`: "GET" or "POST" (default: "POST")
- `status_callback_event`: Events to report - "start end join leave mute hold modify speaker announcement"

**Control Flow:**
- `result`: Switch on return_value (object {} or array [] for conditional logic)

**Variables Set:**
- `join_conference_result`: "completed", "answered", "no-answer", "failed", or "canceled"
- `return_value`: Same as `join_conference_result`

#### `Tap(uri string, controlID string, direction TapDirection, codec Codec, rtpPtime int, statusURL string) *FunctionResult`
**[HELPER METHOD]** - Start background call tap using SWML.

Media is streamed over Websocket or RTP to customer controlled URI for real-time monitoring and analysis. `direction` is `swaig.TapDirectionBoth`/`Speak`/`Hear`; `codec` is `swaig.CodecPCMU`/`CodecPCMA`. Pass `0` / `""` for arguments you want at their defaults.

```go
// Simple WebSocket tap
result.Tap("wss://example.com/tap", "", swaig.TapDirectionBoth, swaig.CodecPCMU, 0, "")

// RTP tap with custom settings
result.Tap("rtp://192.168.1.100:5004", "monitoring_tap_001", swaig.TapDirectionBoth, swaig.CodecPCMA, 30, "")

// Advanced tap with status callbacks
result.Tap(
	"wss://monitoring.company.com/audio-stream",
	"compliance_tap",
	swaig.TapDirectionSpeak, // Only what the party says
	swaig.CodecPCMU,
	0,
	"https://api.company.com/tap-status",
).SetMetadata(map[string]any{"tap_purpose": "compliance", "session_id": "sess_123"})
```

**Core Parameters:**
- `uri` (required): Destination of tap media stream
  - WebSocket: `ws://example.com` or `wss://example.com`
  - RTP: `rtp://IP:port`
- `control_id`: Identifier for this tap to use with stop_tap (optional, auto-generated if not provided)

**Audio Configuration:**
- `direction`: Audio direction to tap (default: "both")
  - `"speak"`: What party says
  - `"hear"`: What party hears
  - `"both"`: What party hears and says
- `codec`: Codec for tap stream - "PCMU" or "PCMA" (default: "PCMU")
- `rtp_ptime`: RTP packetization time in milliseconds (default: 20)

**Status & Monitoring:**
- `status_url`: URL for tap status change requests

**Variables Set:**
- `tap_uri`: Destination URI of the newly started tap
- `tap_result`: "success" or "failed"
- `tap_control_id`: Control ID of this tap
- `tap_rtp_src_addr`: If RTP, source address of the tap stream
- `tap_rtp_src_port`: If RTP, source port of the tap stream
- `tap_ptime`: Packetization time of the tap stream
- `tap_codec`: Codec in the tap stream
- `tap_rate`: Sample rate in the tap stream

#### `StopTap(controlID string) *FunctionResult`
**[HELPER METHOD]** - Stop an active tap stream using SWML. Pass `""` to stop the most recent tap.

```go
// Stop the most recent tap
result.StopTap("")

// Stop specific tap by ID
result.StopTap("monitoring_tap_001")

// Chain to stop tap and provide feedback
result.StopTap("compliance_tap").
	Say("Audio monitoring has been stopped").
	UpdateGlobalData(map[string]any{"tap_active": false})
```

**Parameters:**
- `control_id` (optional): ID of the tap to stop. If not set, the last tap started will be stopped.

**Variables Set:**
- `stop_tap_result`: "success" or "failed"

#### `Hangup() *FunctionResult`
Terminate the call immediately.

```go
result.Hangup()
```

---

### Call Flow Control

#### `Hold(timeout int) *FunctionResult`
Put call on hold with timeout (max 900 seconds).

```go
result.Hold(60)    // Hold for 1 minute
result.Hold(600)   // Hold for 10 minutes
```

#### `WaitForUser(enabled *bool, timeout *int, answerFirst bool) *FunctionResult`
Control how agent waits for user input with flexible parameters. `enabled` and `timeout` are pointers so they can be omitted with `nil`.

```go
enabled := true
result.WaitForUser(&enabled, nil, false)      // Wait indefinitely

timeout := 30
result.WaitForUser(nil, &timeout, false)      // Wait 30 seconds

result.WaitForUser(nil, nil, true)            // Special answer-first mode

disabled := false
result.WaitForUser(&disabled, nil, false)     // Disable waiting
```

#### `Stop() *FunctionResult`
Stop agent execution completely.

```go
result.Stop()
```

---

### Speech & Audio Control

#### `Say(text string) *FunctionResult`
Make the agent speak specific text immediately.

```go
result.Say("Please hold while I look that up for you")
```

#### `PlayBackgroundFile(filename string, wait bool) *FunctionResult`
Play audio file in background with attention control.

```go
result.PlayBackgroundFile("hold_music.wav", false)       // AI tries to get attention
result.PlayBackgroundFile("announcement.mp3", true)      // AI suppresses attention
```

#### `StopBackgroundFile() *FunctionResult`
Stop currently playing background audio.

```go
result.StopBackgroundFile()
```

---

### Speech Recognition Settings

#### `SetEndOfSpeechTimeout(ms int) *FunctionResult`
Set silence timeout after speech detection for finalizing recognition.

```go
result.SetEndOfSpeechTimeout(2000)  // 2 seconds of silence
```

#### `SetSpeechEventTimeout(ms int) *FunctionResult`
Set timeout since last speech event - better for noisy environments.

```go
result.SetSpeechEventTimeout(3000)  // 3 seconds since last speech event
```

---

### Data Management

#### `UpdateGlobalData(data map[string]any) *FunctionResult`
Update global agent data variables.

```go
result.UpdateGlobalData(map[string]any{"user_name": "John", "step": 2})
```

#### `RemoveGlobalData(keys []string) *FunctionResult` / `RemoveGlobalDataKey(key string) *FunctionResult`
Remove global data variables by key(s).

```go
result.RemoveGlobalDataKey("temporary_data")               // Single key
result.RemoveGlobalData([]string{"step", "temp_value"})    // Multiple keys
```

#### `SetMetadata(data map[string]any) *FunctionResult`
Set metadata scoped to current function's meta_data_token.

```go
result.SetMetadata(map[string]any{"session_id": "abc123", "user_tier": "premium"})
```

#### `RemoveMetadata(keys []string) *FunctionResult` / `RemoveMetadataKey(key string) *FunctionResult`
Remove metadata from current function's scope.

```go
result.RemoveMetadataKey("temp_session_data")              // Single key
result.RemoveMetadata([]string{"cache_key", "temp_flag"})  // Multiple keys
```

---

### Function & Behavior Control

#### `ToggleFunctions(toggles []map[string]any) *FunctionResult`
Enable/disable specific SWAIG functions dynamically.

```go
result.ToggleFunctions([]map[string]any{
	{"function": "transfer_call", "active": false},
	{"function": "lookup_info", "active": true},
})
```

#### `EnableFunctionsOnTimeout(enabled bool) *FunctionResult`
Control whether functions can be called on speaker timeout.

```go
result.EnableFunctionsOnTimeout(true)
result.EnableFunctionsOnTimeout(false)
```

#### `EnableExtensiveData(enabled bool) *FunctionResult`
Send full data to LLM for this turn only, then use smaller replacement.

```go
result.EnableExtensiveData(true)   // Send extensive data this turn
result.EnableExtensiveData(false)  // Use normal data
```

#### `ReplaceInHistory(text any) *FunctionResult`
Remove or replace the tool_call + tool_result pair from the LLM's conversation history after the first send. This is useful when a function call is an implementation detail that would confuse the model if it remained visible in context.

When called with a string, the tool_call/tool_result pair is replaced with an assistant message containing that text. When called with `true`, the pair is removed entirely — the LLM will never see that the function was called.

```go
// Remove entirely — LLM won't see this function was called
result = swaig.NewFunctionResult("Done.")
result.ReplaceInHistory(true)

// Replace with a friendly assistant message instead of tool artifacts
result = swaig.NewFunctionResult("Profile saved.")
result.ReplaceInHistory("I've saved your profile information.")

// Practical example: data collection function that shouldn't clutter history
a.DefineTool(agent.ToolDefinition{
	Name:        "save_answer",
	Description: "Save the user's answer",
	Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
		answer, _ := args["answer"].(string)
		result := swaig.NewFunctionResult(fmt.Sprintf("Answer recorded: %s", answer))
		result.ReplaceInHistory(true) // Keep history clean
		return result
	},
})
```

**When to use:**
- Functions that are implementation details (saving data, logging, internal state changes)
- Functions called frequently that would bloat conversation history
- Situations where tool artifacts confuse the model's reasoning (especially with reasoning models at low effort settings)

**Note:** For structured data collection, consider using [gather_info mode](contexts_guide.md#gather-info-mode) instead, which produces zero tool artifacts by design and doesn't require `replace_in_history`.

---

### Agent Settings & Configuration

#### `UpdateSettings(settings map[string]any) *FunctionResult`
Update agent runtime settings with validation.

```go
// AI model settings
result.UpdateSettings(map[string]any{
	"temperature":       0.7,
	"max-tokens":        2048,
	"frequency-penalty": -0.5,
})

// Speech recognition settings
result.UpdateSettings(map[string]any{
	"confidence":       0.8,
	"barge-confidence": 0.7,
})
```

**Supported Settings:**
- `frequency-penalty`: Float (-2.0 to 2.0)
- `presence-penalty`: Float (-2.0 to 2.0) 
- `max-tokens`: Integer (0 to 4096)
- `top-p`: Float (0.0 to 1.0)
- `confidence`: Float (0.0 to 1.0)
- `barge-confidence`: Float (0.0 to 1.0)
- `temperature`: Float (0.0 to 2.0, clamped to 1.5)

#### `SwitchContext(systemPrompt, userPrompt string, consolidate, fullReset, isolated bool) *FunctionResult`
Change agent context/prompt during conversation. Pass `""` for a prompt you want to omit.

```go
// Simple context switch
result.SwitchContext("You are now a technical support agent", "", false, false, false)

// Advanced context switch
result.SwitchContext(
	"You are a billing specialist",         // systemPrompt
	"The user needs help with their invoice", // userPrompt
	true,  // consolidate
	false, // fullReset
	false, // isolated
)
```

#### `SimulateUserInput(text string) *FunctionResult`
Queue simulated user input for testing or flow control.

```go
result.SimulateUserInput("Yes, I'd like to speak to billing")
```

---

## Low-Level Methods

### Manual Action Construction

#### `AddAction(name string, data any) *FunctionResult`
Add a single action manually (for custom actions not covered by helper methods).

```go
result.AddAction("custom_action", map[string]any{"param": "value"})
```

#### `AddActions(actions []map[string]any) *FunctionResult`
Add multiple actions at once.

```go
result.AddActions([]map[string]any{
	{"say": "Hello"},
	{"hold": 300},
})
```

### Output Generation

#### `ToMap() map[string]any`
Convert result to a map for JSON serialization.

```go
resultMap := result.ToMap()
// Returns: {"response": "...", "action": [...], "post_process": true/false}
_ = resultMap
```

---

## Method Chaining

All methods return the receiver (`*swaig.FunctionResult`) to enable fluent method chaining:

```go
result = swaig.NewFunctionResult("Processing your request").
	SetPostProcess(true).
	UpdateGlobalData(map[string]any{"status": "processing"}).
	PlayBackgroundFile("processing.wav", true).
	SetEndOfSpeechTimeout(2500)

// Complex chaining example
result = swaig.NewFunctionResult("Let me transfer you to billing").
	SetMetadata(map[string]any{"transfer_reason": "billing_inquiry"}).
	UpdateGlobalData(map[string]any{"last_action": "transfer_to_billing"}).
	Connect("+15551234567", true, "")
```

---

## Implementation Status

- **[IMPLEMENTED]**: `Connect()`, `UpdateGlobalData()`, and all methods listed above
- **[HELPER METHODS]**: `SendSms()`, `Pay()`, `RecordCall()`, `StopRecordCall()`, `JoinRoom()`, `SIPRefer()`, `JoinConference()`, `Tap()`, `StopTap()` - Additional convenience methods that generate SWML
- **[UTILITY FUNCTIONS]**: `CreatePaymentPrompt()`, `CreatePaymentAction()`, `CreatePaymentParameter()`
- **[EXTENSIBLE]**: Additional convenience methods for common SWML patterns

## Best Practices

1. **Use `SetPostProcess(true)`** when you want the AI to speak before executing actions
2. **Chain methods** for cleaner, more readable code
3. **Use specific methods** instead of manual action construction when available
4. **Handle inputs carefully** - pass zero values (`""`, `nil`, `0`) for optional arguments you want to omit
5. **Validate settings** - `UpdateSettings()` relies on server-side validation

### Final State
The framework now includes **10 virtual helpers total**:
1. `Connect()` - Call transfer/connect
2. `SendSms()` - SMS messaging
3. `Pay()` - Payment processing
4. `RecordCall()` - Start background recording
5. `StopRecordCall()` - Stop background recording
6. `JoinRoom()` - Join RELAY room
7. `SIPRefer()` - SIP REFER transfer
8. `JoinConference()` - Join audio conference with extensive options
9. `Tap()` - Start background call tap for monitoring
10. `StopTap()` - Stop background call tap

---

## Post Data Reference

The `post_data` object is the JSON payload sent to SWAIG function handlers. Its structure differs between webhook functions and DataMap functions.

### Base Keys (All Functions)

| Key | Type | Description |
|-----|------|-------------|
| `app_name` | string | Name of the AI application |
| `function` | string | Name of the SWAIG function being called |
| `call_id` | string | Unique UUID of the current call session |
| `ai_session_id` | string | Unique UUID of the AI session |
| `caller_id_name` | string | Caller ID name (if available) |
| `caller_id_num` | string | Caller ID number (if available) |
| `channel_active` | boolean | Whether the channel is currently up |
| `channel_offhook` | boolean | Whether the channel is off-hook |
| `channel_ready` | boolean | Whether the AI session is ready |
| `argument` | object | Parsed function arguments |
| `argument_desc` | object | Function argument schema/description |
| `purpose` | string | Description of what the function does |
| `content_type` | string | Always `"text/swaig"` |
| `version` | string | SWAIG protocol version |
| `global_data` | object | Application-level global data (when set) |
| `conversation_id` | string | Conversation identifier (when tracking enabled) |
| `project_id` | string | SignalWire project ID |
| `space_id` | string | SignalWire space ID |

### Webhook-Only Keys

These keys are only present for traditional webhook SWAIG functions:

| Key | Type | Description | Present When |
|-----|------|-------------|--------------|
| `meta_data_token` | string | Token for metadata access | Function has metadata token |
| `meta_data` | object | Function-level metadata | Function has metadata token |
| `SWMLVars` | object | SWML variables | `swaig_post_swml_vars` parameter set |
| `SWMLCall` | object | SWML call state | `swaig_post_swml_vars` parameter set |
| `call_log` | array | Processed conversation history | `swaig_post_conversation` is true |
| `raw_call_log` | array | Raw conversation history | `swaig_post_conversation` is true |

**Metadata scoping**: Functions sharing the same `meta_data_token` share access to the same metadata. If no token is specified, scope defaults to function name/URL.

**Conversation history**: `call_log` may shrink after conversation resets (consolidation), while `raw_call_log` preserves full history. Both include timing data (latency, utterance_latency, audio_latency).

### DataMap-Specific Keys

| Key | Type | Description |
|-----|------|-------------|
| `prompt_vars` | object | Template variables built from call context, SWML vars, and global_data |
| `args` | object | First parsed argument object for easy template access |
| `input` | object | Copy of entire post_data for variable expansion |

### prompt_vars Contents

| Key | Source | Description |
|-----|--------|-------------|
| `call_direction` | Call direction | `"inbound"` or `"outbound"` |
| `caller_id_name` | Channel variable | Caller's name |
| `caller_id_number` | Channel variable | Caller's number |
| `local_date` | System time | Current date in local timezone |
| `local_time` | System time | Current time with timezone |
| `time_of_day` | Derived from hour | `"morning"`, `"afternoon"`, or `"evening"` |
| `supported_languages` | App config | Available languages |
| `default_language` | App config | Primary language |

All keys from `global_data` are also merged into `prompt_vars`, with global_data taking precedence.

### SWML Parameters Controlling post_data

| Parameter | Type | Default | Purpose |
|-----------|------|---------|---------|
| `swaig_allow_swml` | boolean | true | Allow functions to execute SWML actions |
| `swaig_allow_settings` | boolean | true | Allow functions to modify AI settings |
| `swaig_post_conversation` | boolean | false | Include conversation history in post_data |
| `swaig_set_global_data` | boolean | true | Allow functions to modify global_data |
| `swaig_post_swml_vars` | boolean/array | false | Include SWML variables in post_data |

### Variable Expansion in DataMap

DataMap processing supports template expansion with access to:

- Nested object access via dot notation: `${user.name}`
- Array access: `${items[0].value}`
- Encoding functions: `${enc:url:variable}`
- Built-in functions: `@{strftime %Y-%m-%d}`, `@{expr 2+2}`

---

## Related Documentation

- **[API Reference](api_reference.md)** - Complete `AgentBase` and `swaig.FunctionResult` API reference
- **[Contexts Guide](contexts_guide.md)** - Using `SwmlChangeContext()` and `SwmlChangeStep()`
- **[DataMap Guide](datamap_guide.md)** - Using `swaig.FunctionResult` with DataMap outputs
- **[Agent Guide](agent_guide.md)** - General agent development guide

### Example Files

- `examples/simple_agent/main.go` - Basic SWAIG function usage
- `examples/swaig_features/main.go` - Advanced SWAIG features (FunctionResult actions)
- `examples/record_call/main.go` - Recording and tapping calls
- `examples/room_and_sip/main.go` - Room joining and SIP transfer