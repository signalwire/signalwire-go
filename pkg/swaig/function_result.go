// Package swaig provides SWAIG (SignalWire AI Gateway) function result handling
// for building AI agent tool responses with actions and call control.
package swaig

import (
	"fmt"
	"strings"

	"github.com/signalwire/signalwire-go/pkg/logging"
)

var log = logging.New("swaig")

// FunctionResult represents the response from a SWAIG tool handler.
// It contains a text response, optional actions, and post-processing control.
// All mutating methods return *FunctionResult for method chaining.
type FunctionResult struct {
	response    string
	postProcess bool
	actions     []map[string]any
}

// NewFunctionResult creates a new FunctionResult with the given response text.
func NewFunctionResult(response string) *FunctionResult {
	return &FunctionResult{
		response: response,
		actions:  []map[string]any{},
	}
}

// --- Getters ---

// Response returns the natural language response text.
func (fr *FunctionResult) Response() string {
	return fr.response
}

// Actions returns the list of actions added to this result.
func (fr *FunctionResult) Actions() []map[string]any {
	return fr.actions
}

// PostProcess returns whether post-processing is enabled.
func (fr *FunctionResult) PostProcess() bool {
	return fr.postProcess
}

// --- Core methods ---

// SetResponse sets the natural language response text.
func (fr *FunctionResult) SetResponse(response string) *FunctionResult {
	fr.response = response
	return fr
}

// SetPostProcess controls whether the AI takes another turn before executing actions.
func (fr *FunctionResult) SetPostProcess(postProcess bool) *FunctionResult {
	fr.postProcess = postProcess
	return fr
}

// AddAction appends a single named action to the result.
func (fr *FunctionResult) AddAction(name string, data any) *FunctionResult {
	fr.actions = append(fr.actions, map[string]any{name: data})
	return fr
}

// AddActions appends multiple actions to the result.
func (fr *FunctionResult) AddActions(actions []map[string]any) *FunctionResult {
	fr.actions = append(fr.actions, actions...)
	return fr
}

// ToMap serializes the FunctionResult to a map suitable for JSON encoding.
// The "action" key is only included if there are actions.
// The "post_process" key is only included if true.
func (fr *FunctionResult) ToMap() map[string]any {
	result := map[string]any{
		"response": fr.response,
	}
	if len(fr.actions) > 0 {
		result["action"] = fr.actions
	}
	if fr.postProcess {
		result["post_process"] = true
	}
	return result
}

// --- Call Control Actions ---

// Connect adds a connect action to transfer/connect the call to another destination.
// If final is true, the call permanently transfers (exits the agent).
// If final is false, the call returns to the agent when the far end hangs up.
// The from parameter sets the caller ID; pass empty string to use the call's default.
func (fr *FunctionResult) Connect(destination string, final bool, from string) *FunctionResult {
	connectParams := map[string]any{"to": destination}
	if from != "" {
		connectParams["from"] = from
	}

	swmlAction := map[string]any{
		"SWML": map[string]any{
			"sections": map[string]any{
				"main": []any{
					map[string]any{"connect": connectParams},
				},
			},
			"version": "1.0.0",
		},
		"transfer": fmt.Sprintf("%t", final),
	}

	fr.actions = append(fr.actions, swmlAction)
	return fr
}

// SwmlTransfer adds a SWML transfer action with an AI response for when control returns.
func (fr *FunctionResult) SwmlTransfer(dest string, aiResponse string, final bool) *FunctionResult {
	swmlAction := map[string]any{
		"SWML": map[string]any{
			"version": "1.0.0",
			"sections": map[string]any{
				"main": []any{
					map[string]any{"set": map[string]any{"ai_response": aiResponse}},
					map[string]any{"transfer": map[string]any{"dest": dest}},
				},
			},
		},
		"transfer": fmt.Sprintf("%t", final),
	}

	fr.actions = append(fr.actions, swmlAction)
	return fr
}

// Hangup terminates the call.
func (fr *FunctionResult) Hangup() *FunctionResult {
	return fr.AddAction("hangup", true)
}

// Hold puts the call on hold with the given timeout in seconds.
// Timeout is clamped to the range [0, 900].
func (fr *FunctionResult) Hold(timeout int) *FunctionResult {
	if timeout < 0 {
		timeout = 0
	}
	if timeout > 900 {
		timeout = 900
	}
	return fr.AddAction("hold", timeout)
}

// WaitForUser controls how the agent waits for user input.
// Pass nil for enabled/timeout to omit those fields. If answerFirst is true,
// the value is set to "answer_first" regardless of other parameters.
func (fr *FunctionResult) WaitForUser(enabled *bool, timeout *int, answerFirst bool) *FunctionResult {
	if answerFirst {
		return fr.AddAction("wait_for_user", "answer_first")
	}
	if timeout != nil {
		return fr.AddAction("wait_for_user", *timeout)
	}
	if enabled != nil {
		return fr.AddAction("wait_for_user", *enabled)
	}
	return fr.AddAction("wait_for_user", true)
}

// Stop stops the agent execution.
func (fr *FunctionResult) Stop() *FunctionResult {
	return fr.AddAction("stop", true)
}

// --- State & Data Management ---

// UpdateGlobalData sets or updates global agent data variables.
func (fr *FunctionResult) UpdateGlobalData(data map[string]any) *FunctionResult {
	return fr.AddAction("set_global_data", data)
}

// RemoveGlobalData removes global agent data variables by key slice.
func (fr *FunctionResult) RemoveGlobalData(keys []string) *FunctionResult {
	return fr.AddAction("unset_global_data", keys)
}

// RemoveGlobalDataKey removes a single global agent data variable by key.
// This matches the Python SDK's Union[str, List[str]] behavior for a bare string argument,
// which emits the key as a string (not a one-element array) in the action payload.
func (fr *FunctionResult) RemoveGlobalDataKey(key string) *FunctionResult {
	return fr.AddAction("unset_global_data", key)
}

// SetMetadata sets metadata scoped to the current function's meta_data_token.
func (fr *FunctionResult) SetMetadata(data map[string]any) *FunctionResult {
	return fr.AddAction("set_meta_data", data)
}

// RemoveMetadata removes metadata keys from the current function's scope.
func (fr *FunctionResult) RemoveMetadata(keys []string) *FunctionResult {
	return fr.AddAction("unset_meta_data", keys)
}

// RemoveMetadataKey removes a single metadata key from the current function's scope.
// This matches the Python SDK's Union[str, List[str]] behavior for a bare string argument,
// which emits the key as a string (not a one-element array) in the action payload.
func (fr *FunctionResult) RemoveMetadataKey(key string) *FunctionResult {
	return fr.AddAction("unset_meta_data", key)
}

// SwmlUserEvent sends a user event through SWML for real-time UI updates.
func (fr *FunctionResult) SwmlUserEvent(eventData map[string]any) *FunctionResult {
	swmlAction := map[string]any{
		"version": "1.0.0",
		"sections": map[string]any{
			"main": []any{
				map[string]any{
					"user_event": map[string]any{
						"event": eventData,
					},
				},
			},
		},
	}
	return fr.AddAction("SWML", swmlAction)
}

// SwmlChangeStep transitions to a different conversation step.
// Emits action key "change_step" with the step name as a plain string value,
// matching the Python SDK's add_action("change_step", step_name).
func (fr *FunctionResult) SwmlChangeStep(stepName string) *FunctionResult {
	return fr.AddAction("change_step", stepName)
}

// SwmlChangeContext transitions to a different conversation context.
// Emits action key "change_context" with the context name as a plain string value,
// matching the Python SDK's add_action("change_context", context_name).
func (fr *FunctionResult) SwmlChangeContext(contextName string) *FunctionResult {
	return fr.AddAction("change_context", contextName)
}

// SwitchContext changes the agent context/prompt during conversation.
// Only non-empty/true fields are included in the action.
func (fr *FunctionResult) SwitchContext(systemPrompt, userPrompt string, consolidate, fullReset, isolated bool) *FunctionResult {
	// Simple case: only system prompt provided
	if systemPrompt != "" && userPrompt == "" && !consolidate && !fullReset && !isolated {
		return fr.AddAction("context_switch", systemPrompt)
	}

	contextData := map[string]any{}
	if systemPrompt != "" {
		contextData["system_prompt"] = systemPrompt
	}
	if userPrompt != "" {
		contextData["user_prompt"] = userPrompt
	}
	if consolidate {
		contextData["consolidate"] = true
	}
	if fullReset {
		contextData["full_reset"] = true
	}
	if isolated {
		contextData["isolated"] = true
	}
	return fr.AddAction("context_switch", contextData)
}

// ReplaceInHistory replaces the tool call and result pair in conversation history.
// If text is a string, the tool call is replaced with an assistant message containing that text.
// If text is a bool and true, the pair is removed from history entirely.
func (fr *FunctionResult) ReplaceInHistory(text any) *FunctionResult {
	switch v := text.(type) {
	case bool:
		return fr.AddAction("replace_in_history", v)
	case string:
		return fr.AddAction("replace_in_history", v)
	default:
		log.Warn("ReplaceInHistory: unsupported type %T, using value as-is", text)
		return fr.AddAction("replace_in_history", text)
	}
}

// --- Media Control ---

// Say makes the agent speak specific text.
func (fr *FunctionResult) Say(text string) *FunctionResult {
	return fr.AddAction("say", text)
}

// PlayBackgroundFile plays an audio or video file in the background.
// If wait is true, attention-getting behavior is suppressed during playback.
func (fr *FunctionResult) PlayBackgroundFile(filename string, wait bool) *FunctionResult {
	if wait {
		return fr.AddAction("playback_bg", map[string]any{"file": filename, "wait": true})
	}
	return fr.AddAction("playback_bg", filename)
}

// StopBackgroundFile stops the currently playing background file.
func (fr *FunctionResult) StopBackgroundFile() *FunctionResult {
	return fr.AddAction("stop_playback_bg", true)
}

// RecordCallOptions holds optional parameters for RecordCall beyond the required fields.
type RecordCallOptions struct {
	// Terminators specifies digits that stop recording when pressed.
	Terminators string
	// Beep plays a beep before recording starts when true.
	Beep bool
	// InputSensitivity sets the input sensitivity for recording (default 44.0 in Python).
	// Zero value is omitted from the SWML payload.
	InputSensitivity float64
	// InitialTimeout is the time in seconds to wait for speech to start (voicemail-style).
	// Negative value is omitted.
	InitialTimeout float64
	// InitialTimeoutSet must be true for InitialTimeout of 0.0 to be included.
	InitialTimeoutSet bool
	// EndSilenceTimeout is seconds of silence before ending (voicemail-style).
	// Negative value is omitted.
	EndSilenceTimeout float64
	// EndSilenceTimeoutSet must be true for EndSilenceTimeout of 0.0 to be included.
	EndSilenceTimeoutSet bool
	// MaxLength is the maximum recording length in seconds. Negative value is omitted.
	MaxLength float64
	// MaxLengthSet must be true for MaxLength of 0.0 to be included.
	MaxLengthSet bool
	// StatusURL is the URL to send recording status events to.
	StatusURL string
}

// RecordCall starts background call recording using SWML.
// controlID, stereo, format, and direction are the primary parameters.
// Use opts to specify additional optional parameters (pass nil to use defaults).
func (fr *FunctionResult) RecordCall(controlID string, stereo bool, format string, direction string, opts *RecordCallOptions) *FunctionResult {
	recordParams := map[string]any{
		"stereo":    stereo,
		"format":    format,
		"direction": direction,
	}
	if controlID != "" {
		recordParams["control_id"] = controlID
	}

	if opts != nil {
		if opts.Terminators != "" {
			recordParams["terminators"] = opts.Terminators
		}
		if opts.Beep {
			recordParams["beep"] = opts.Beep
		}
		if opts.InputSensitivity != 0 {
			recordParams["input_sensitivity"] = opts.InputSensitivity
		}
		if opts.InitialTimeoutSet || opts.InitialTimeout > 0 {
			recordParams["initial_timeout"] = opts.InitialTimeout
		}
		if opts.EndSilenceTimeoutSet || opts.EndSilenceTimeout > 0 {
			recordParams["end_silence_timeout"] = opts.EndSilenceTimeout
		}
		if opts.MaxLengthSet || opts.MaxLength > 0 {
			recordParams["max_length"] = opts.MaxLength
		}
		if opts.StatusURL != "" {
			recordParams["status_url"] = opts.StatusURL
		}
	}

	swmlDoc := map[string]any{
		"version": "1.0.0",
		"sections": map[string]any{
			"main": []any{
				map[string]any{"record_call": recordParams},
			},
		},
	}
	return fr.AddAction("SWML", swmlDoc)
}

// StopRecordCall stops an active background call recording.
func (fr *FunctionResult) StopRecordCall(controlID string) *FunctionResult {
	stopParams := map[string]any{}
	if controlID != "" {
		stopParams["control_id"] = controlID
	}

	swmlDoc := map[string]any{
		"version": "1.0.0",
		"sections": map[string]any{
			"main": []any{
				map[string]any{"stop_record_call": stopParams},
			},
		},
	}
	return fr.AddAction("SWML", swmlDoc)
}

// --- Speech & AI Config ---

// AddDynamicHints adds dynamic speech recognition hints during a call.
func (fr *FunctionResult) AddDynamicHints(hints []any) *FunctionResult {
	return fr.AddAction("add_dynamic_hints", hints)
}

// ClearDynamicHints removes all dynamic speech recognition hints.
func (fr *FunctionResult) ClearDynamicHints() *FunctionResult {
	fr.actions = append(fr.actions, map[string]any{"clear_dynamic_hints": map[string]any{}})
	return fr
}

// SetEndOfSpeechTimeout adjusts the end-of-speech timeout in milliseconds.
func (fr *FunctionResult) SetEndOfSpeechTimeout(ms int) *FunctionResult {
	return fr.AddAction("end_of_speech_timeout", ms)
}

// SetSpeechEventTimeout adjusts the speech event timeout in milliseconds.
func (fr *FunctionResult) SetSpeechEventTimeout(ms int) *FunctionResult {
	return fr.AddAction("speech_event_timeout", ms)
}

// ToggleFunctions enables or disables specific SWAIG functions.
// Each toggle should have "function" and "active" keys.
func (fr *FunctionResult) ToggleFunctions(toggles []map[string]any) *FunctionResult {
	return fr.AddAction("toggle_functions", toggles)
}

// EnableFunctionsOnTimeout enables or disables function calls on speaker timeout.
func (fr *FunctionResult) EnableFunctionsOnTimeout(enabled bool) *FunctionResult {
	return fr.AddAction("functions_on_speaker_timeout", enabled)
}

// EnableExtensiveData sends full data to LLM for this turn only.
func (fr *FunctionResult) EnableExtensiveData(enabled bool) *FunctionResult {
	return fr.AddAction("extensive_data", enabled)
}

// UpdateSettings updates agent runtime settings such as temperature, top_p, etc.
func (fr *FunctionResult) UpdateSettings(settings map[string]any) *FunctionResult {
	return fr.AddAction("settings", settings)
}

// --- Advanced Features ---

// ExecuteSwml executes SWML content. If transfer is true, the call exits the agent after execution.
// swmlContent can be a map[string]any or a string (raw SWML JSON).
func (fr *FunctionResult) ExecuteSwml(swmlContent any, transfer bool) *FunctionResult {
	var action map[string]any

	switch v := swmlContent.(type) {
	case map[string]any:
		// Make a shallow copy to avoid mutating the caller's data
		action = make(map[string]any, len(v)+1)
		for k, val := range v {
			action[k] = val
		}
	case string:
		action = map[string]any{"raw_swml": v}
	default:
		log.Warn("ExecuteSwml: unsupported content type %T", swmlContent)
		action = map[string]any{"raw_swml": fmt.Sprintf("%v", swmlContent)}
	}

	if transfer {
		action["transfer"] = "true"
	}

	return fr.AddAction("SWML", action)
}

// JoinConferenceOptions holds optional parameters for JoinConference beyond the required name.
type JoinConferenceOptions struct {
	// Muted joins the conference muted when true.
	Muted bool
	// Beep controls beep behavior: "true" (default), "false", "onEnter", "onExit".
	Beep string
	// StartOnEnter controls whether the conference starts when this participant enters (default true in Python).
	StartOnEnter *bool
	// EndOnExit controls whether the conference ends when this participant exits (default false).
	EndOnExit bool
	// WaitURL is the SWML URL for hold music (replaces the old holdAudio parameter).
	WaitURL string
	// MaxParticipants sets the maximum number of participants (<= 250). 0 uses server default.
	MaxParticipants int
	// Record sets the recording mode: "do-not-record" (default) or "record-from-start".
	Record string
	// Region sets the conference region.
	Region string
	// Trim controls silence trimming: "trim-silence" (default) or "do-not-trim".
	Trim string
	// Coach sets the SWML Call ID or CXML CallSid for coaching.
	Coach string
	// StatusCallbackEvent specifies events to report (space-separated).
	StatusCallbackEvent string
	// StatusCallback is the URL for status callbacks.
	StatusCallback string
	// StatusCallbackMethod sets the HTTP method for status callbacks ("GET" or "POST").
	StatusCallbackMethod string
	// RecordingStatusCallback is the URL for recording status callbacks.
	RecordingStatusCallback string
	// RecordingStatusCallbackMethod sets the HTTP method for recording callbacks ("GET" or "POST").
	RecordingStatusCallbackMethod string
	// RecordingStatusCallbackEvent sets recording events to report.
	RecordingStatusCallbackEvent string
	// Result sets switch-on-return-value behavior (object or array).
	Result any
}

// JoinConference joins an ad-hoc audio conference.
// Pass nil for opts to use default behavior (muted=false, beep="true", no holdAudio).
func (fr *FunctionResult) JoinConference(name string, opts *JoinConferenceOptions) *FunctionResult {
	if opts == nil {
		opts = &JoinConferenceOptions{}
	}

	joinParams := map[string]any{"name": name}
	if opts.Muted {
		joinParams["muted"] = true
	}
	if opts.Beep != "" && opts.Beep != "true" {
		joinParams["beep"] = opts.Beep
	}
	if opts.StartOnEnter != nil && !*opts.StartOnEnter {
		joinParams["start_on_enter"] = false
	}
	if opts.EndOnExit {
		joinParams["end_on_exit"] = true
	}
	if opts.WaitURL != "" {
		joinParams["wait_url"] = opts.WaitURL
	}
	if opts.MaxParticipants != 0 && opts.MaxParticipants != 250 {
		joinParams["max_participants"] = opts.MaxParticipants
	}
	if opts.Record != "" && opts.Record != "do-not-record" {
		joinParams["record"] = opts.Record
	}
	if opts.Region != "" {
		joinParams["region"] = opts.Region
	}
	if opts.Trim != "" && opts.Trim != "trim-silence" {
		joinParams["trim"] = opts.Trim
	}
	if opts.Coach != "" {
		joinParams["coach"] = opts.Coach
	}
	if opts.StatusCallbackEvent != "" {
		joinParams["status_callback_event"] = opts.StatusCallbackEvent
	}
	if opts.StatusCallback != "" {
		joinParams["status_callback"] = opts.StatusCallback
	}
	if opts.StatusCallbackMethod != "" && opts.StatusCallbackMethod != "POST" {
		joinParams["status_callback_method"] = opts.StatusCallbackMethod
	}
	if opts.RecordingStatusCallback != "" {
		joinParams["recording_status_callback"] = opts.RecordingStatusCallback
	}
	if opts.RecordingStatusCallbackMethod != "" && opts.RecordingStatusCallbackMethod != "POST" {
		joinParams["recording_status_callback_method"] = opts.RecordingStatusCallbackMethod
	}
	if opts.RecordingStatusCallbackEvent != "" && opts.RecordingStatusCallbackEvent != "completed" {
		joinParams["recording_status_callback_event"] = opts.RecordingStatusCallbackEvent
	}
	if opts.Result != nil {
		joinParams["result"] = opts.Result
	}

	swmlDoc := map[string]any{
		"version": "1.0.0",
		"sections": map[string]any{
			"main": []any{
				map[string]any{"join_conference": joinParams},
			},
		},
	}
	return fr.AddAction("SWML", swmlDoc)
}

// JoinRoom joins a RELAY room for multi-party communication.
func (fr *FunctionResult) JoinRoom(name string) *FunctionResult {
	swmlDoc := map[string]any{
		"version": "1.0.0",
		"sections": map[string]any{
			"main": []any{
				map[string]any{"join_room": map[string]any{"name": name}},
			},
		},
	}
	return fr.AddAction("SWML", swmlDoc)
}

// SipRefer sends a SIP REFER for call transfer in SIP environments.
func (fr *FunctionResult) SipRefer(toURI string) *FunctionResult {
	swmlDoc := map[string]any{
		"version": "1.0.0",
		"sections": map[string]any{
			"main": []any{
				map[string]any{"sip_refer": map[string]any{"to_uri": toURI}},
			},
		},
	}
	return fr.AddAction("SWML", swmlDoc)
}

// Tap starts background call tapping, streaming media to the given URI.
// rtpPtime sets the packetization time in milliseconds for RTP streams (0 = use default of 20ms).
// Pass empty string for statusURL to omit it.
func (fr *FunctionResult) Tap(uri string, controlID string, direction string, codec string, rtpPtime int, statusURL string) *FunctionResult {
	tapParams := map[string]any{"uri": uri}
	if controlID != "" {
		tapParams["control_id"] = controlID
	}
	if direction != "" && direction != "both" {
		tapParams["direction"] = direction
	}
	if codec != "" && codec != "PCMU" {
		tapParams["codec"] = codec
	}
	if rtpPtime != 0 && rtpPtime != 20 {
		tapParams["rtp_ptime"] = rtpPtime
	}
	if statusURL != "" {
		tapParams["status_url"] = statusURL
	}

	swmlDoc := map[string]any{
		"version": "1.0.0",
		"sections": map[string]any{
			"main": []any{
				map[string]any{"tap": tapParams},
			},
		},
	}
	return fr.AddAction("SWML", swmlDoc)
}

// StopTap stops an active tap stream.
func (fr *FunctionResult) StopTap(controlID string) *FunctionResult {
	stopParams := map[string]any{}
	if controlID != "" {
		stopParams["control_id"] = controlID
	}

	swmlDoc := map[string]any{
		"version": "1.0.0",
		"sections": map[string]any{
			"main": []any{
				map[string]any{"stop_tap": stopParams},
			},
		},
	}
	return fr.AddAction("SWML", swmlDoc)
}

// SendSms sends a text message to a PSTN phone number.
// Pass empty string for body if only sending media, nil for optional slices,
// and empty string for region to omit it.
func (fr *FunctionResult) SendSms(toNumber, fromNumber, body string, media []string, tags []string, region string) *FunctionResult {
	smsParams := map[string]any{
		"to_number":   toNumber,
		"from_number": fromNumber,
	}
	if body != "" {
		smsParams["body"] = body
	}
	if len(media) > 0 {
		smsParams["media"] = media
	}
	if len(tags) > 0 {
		smsParams["tags"] = tags
	}
	if region != "" {
		smsParams["region"] = region
	}

	swmlDoc := map[string]any{
		"version": "1.0.0",
		"sections": map[string]any{
			"main": []any{
				map[string]any{"send_sms": smsParams},
			},
		},
	}
	return fr.AddAction("SWML", swmlDoc)
}

// PayOptions holds all optional parameters for the Pay method.
type PayOptions struct {
	// InputMethod is the method to collect payment details ("dtmf" or "voice"). Defaults to "dtmf".
	InputMethod string
	// StatusURL is the URL for payment status change notifications.
	StatusURL string
	// PaymentMethod is the payment method type. Defaults to "credit-card".
	PaymentMethod string
	// Timeout is the seconds to wait for the next digit. Defaults to 5.
	Timeout int
	// MaxAttempts is the number of retry attempts. Defaults to 1.
	MaxAttempts int
	// SecurityCode controls whether to prompt for security code. Defaults to true.
	// Use SecurityCodeSet to override; zero value (false) is treated as "not set".
	SecurityCode bool
	// SecurityCodeSet must be true to explicitly set SecurityCode=false.
	SecurityCodeSet bool
	// PostalCode controls whether to prompt for postal code, or supplies the actual code.
	// String value is used as-is; bool true/false becomes "true"/"false".
	PostalCode any
	// MinPostalCodeLength sets the minimum number of postal code digits. Defaults to 0.
	MinPostalCodeLength int
	// TokenType is the payment token type: "one-time" or "reusable". Defaults to "reusable".
	TokenType string
	// ChargeAmount is the amount to charge as a decimal string (e.g. "9.99").
	ChargeAmount string
	// Currency is the currency code. Defaults to "usd".
	Currency string
	// Language is the language for prompts. Defaults to "en-US".
	Language string
	// Voice is the TTS voice to use. Defaults to "woman".
	Voice string
	// Description is a custom payment description.
	Description string
	// ValidCardTypes is a space-separated list of card types. Defaults to "visa mastercard amex".
	ValidCardTypes string
	// Parameters is an array of name/value pairs for the payment connector.
	Parameters []map[string]string
	// Prompts is an array of custom prompt configurations.
	Prompts []map[string]any
	// AIResponse is the message set via "set" verb before pay; empty string uses Python default.
	// Set to "-" to suppress the set verb entirely (no ai_response).
	AIResponse string
}

// Pay processes a payment using SWML pay action.
// connectorURL is the only required parameter.
// opts may be nil to use Python SDK defaults for all optional parameters.
func (fr *FunctionResult) Pay(connectorURL string, opts *PayOptions) *FunctionResult {
	if opts == nil {
		opts = &PayOptions{}
	}

	inputMethod := opts.InputMethod
	if inputMethod == "" {
		inputMethod = "dtmf"
	}
	paymentMethod := opts.PaymentMethod
	if paymentMethod == "" {
		paymentMethod = "credit-card"
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 5
	}
	maxAttempts := opts.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = 1
	}
	tokenType := opts.TokenType
	if tokenType == "" {
		tokenType = "reusable"
	}
	currency := opts.Currency
	if currency == "" {
		currency = "usd"
	}
	language := opts.Language
	if language == "" {
		language = "en-US"
	}
	voice := opts.Voice
	if voice == "" {
		voice = "woman"
	}
	validCardTypes := opts.ValidCardTypes
	if validCardTypes == "" {
		validCardTypes = "visa mastercard amex"
	}

	// Handle security_code: default true
	securityCode := "true"
	if opts.SecurityCodeSet && !opts.SecurityCode {
		securityCode = "false"
	} else if !opts.SecurityCodeSet && opts.SecurityCode {
		securityCode = "true"
	}

	// Handle postal_code: default true
	var postalCodeStr string
	if opts.PostalCode == nil {
		postalCodeStr = "true"
	} else {
		switch v := opts.PostalCode.(type) {
		case bool:
			if v {
				postalCodeStr = "true"
			} else {
				postalCodeStr = "false"
			}
		case string:
			postalCodeStr = v
		default:
			postalCodeStr = fmt.Sprintf("%v", v)
		}
	}

	payParams := map[string]any{
		"payment_connector_url": connectorURL,
		"input":                 inputMethod,
		"payment_method":        paymentMethod,
		"timeout":               fmt.Sprintf("%d", timeout),
		"max_attempts":          fmt.Sprintf("%d", maxAttempts),
		"security_code":         securityCode,
		"postal_code":           postalCodeStr,
		"min_postal_code_length": fmt.Sprintf("%d", opts.MinPostalCodeLength),
		"token_type":            tokenType,
		"currency":              currency,
		"language":              language,
		"voice":                 voice,
		"valid_card_types":      validCardTypes,
	}

	if opts.StatusURL != "" {
		payParams["status_url"] = opts.StatusURL
	}
	if opts.ChargeAmount != "" {
		payParams["charge_amount"] = opts.ChargeAmount
	}
	if opts.Description != "" {
		payParams["description"] = opts.Description
	}
	if len(opts.Parameters) > 0 {
		payParams["parameters"] = opts.Parameters
	}
	if len(opts.Prompts) > 0 {
		payParams["prompts"] = opts.Prompts
	}

	// Determine ai_response: Python default is a status message
	aiResponse := opts.AIResponse
	if aiResponse == "" {
		aiResponse = "The payment status is ${pay_result}, do not mention anything else about collecting payment if successful."
	}

	mainVerbs := []any{
		map[string]any{"pay": payParams},
	}

	if aiResponse != "-" {
		mainVerbs = append([]any{
			map[string]any{"set": map[string]any{"ai_response": aiResponse}},
		}, mainVerbs...)
	}

	swmlDoc := map[string]any{
		"version": "1.0.0",
		"sections": map[string]any{
			"main": mainVerbs,
		},
	}
	return fr.AddAction("SWML", swmlDoc)
}

// --- RPC Actions ---

// ExecuteRpc executes an RPC method on a call.
// Pass empty strings for callID and nodeID to omit them from the payload.
func (fr *FunctionResult) ExecuteRpc(method string, params map[string]any, callID string, nodeID string) *FunctionResult {
	rpcParams := map[string]any{
		"method":  method,
		"jsonrpc": "2.0",
	}
	if callID != "" {
		rpcParams["call_id"] = callID
	}
	if nodeID != "" {
		rpcParams["node_id"] = nodeID
	}
	if len(params) > 0 {
		rpcParams["params"] = params
	}

	swmlDoc := map[string]any{
		"version": "1.0.0",
		"sections": map[string]any{
			"main": []any{
				map[string]any{"execute_rpc": rpcParams},
			},
		},
	}
	return fr.AddAction("SWML", swmlDoc)
}

// RpcDial dials out to a number with a destination SWML URL using execute_rpc.
// deviceType defaults to "phone" when empty.
// This matches the Python SDK's rpc_dial() which calls execute_rpc(method="dial", ...).
func (fr *FunctionResult) RpcDial(toNumber, fromNumber, destSwml string, deviceType string) *FunctionResult {
	if deviceType == "" {
		deviceType = "phone"
	}
	dialParams := map[string]any{
		"devices": map[string]any{
			"type": deviceType,
			"params": map[string]any{
				"to_number":   toNumber,
				"from_number": fromNumber,
			},
		},
		"dest_swml": destSwml,
	}

	return fr.ExecuteRpc("dial", dialParams, "", "")
}

// RpcAiMessage injects a message into an AI agent on another call.
// role defaults to "system" when empty, matching the Python SDK default.
// This matches the Python SDK's rpc_ai_message() which calls execute_rpc(method="ai_message", ...).
func (fr *FunctionResult) RpcAiMessage(callID, messageText, role string) *FunctionResult {
	if role == "" {
		role = "system"
	}
	return fr.ExecuteRpc("ai_message", map[string]any{
		"role":         role,
		"message_text": messageText,
	}, callID, "")
}

// RpcAiUnhold unholds another call.
// This matches the Python SDK's rpc_ai_unhold() which calls execute_rpc(method="ai_unhold", ...).
func (fr *FunctionResult) RpcAiUnhold(callID string) *FunctionResult {
	return fr.ExecuteRpc("ai_unhold", map[string]any{}, callID, "")
}

// SimulateUserInput queues simulated user input text.
// Emits action key "user_input" matching the Python SDK's add_action("user_input", text).
func (fr *FunctionResult) SimulateUserInput(text string) *FunctionResult {
	return fr.AddAction("user_input", text)
}

// --- Payment Helpers ---

// CreatePaymentPrompt creates a payment prompt configuration.
// cardType and errorType are optional; pass empty strings to omit them.
// This matches the Python SDK's create_payment_prompt() static method signature.
func CreatePaymentPrompt(forSituation string, actions []map[string]string, cardType string, errorType string) map[string]any {
	prompt := map[string]any{
		"for":     forSituation,
		"actions": actions,
	}
	if cardType != "" {
		prompt["card_type"] = cardType
	}
	if errorType != "" {
		prompt["error_type"] = errorType
	}
	return prompt
}

// CreatePaymentAction creates a single payment action entry.
func CreatePaymentAction(actionType string, phrase string) map[string]string {
	return map[string]string{
		"type":   actionType,
		"phrase": phrase,
	}
}

// CreatePaymentParameter creates a payment parameter entry.
func CreatePaymentParameter(name string, value string) map[string]string {
	return map[string]string{
		"name":  name,
		"value": value,
	}
}

// String returns a human-readable representation including the response and action count.
func (fr *FunctionResult) String() string {
	resp := fr.response
	if len(resp) > 50 {
		resp = resp[:50] + "..."
	}
	resp = strings.ReplaceAll(resp, "\n", "\\n")
	return fmt.Sprintf("FunctionResult{response=%q, actions=%d, post_process=%t}", resp, len(fr.actions), fr.postProcess)
}
