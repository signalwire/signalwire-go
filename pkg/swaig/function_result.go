// Package swaig provides SWAIG (SignalWire AI Gateway) function result handling
// for building AI agent tool responses with actions and call control.
package swaig

import (
	"fmt"
	"strings"

	"github.com/signalwire/signalwire-agents-go/pkg/logging"
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

// RemoveGlobalData removes global agent data variables by key.
func (fr *FunctionResult) RemoveGlobalData(keys []string) *FunctionResult {
	return fr.AddAction("unset_global_data", keys)
}

// SetMetadata sets metadata scoped to the current function's meta_data_token.
func (fr *FunctionResult) SetMetadata(data map[string]any) *FunctionResult {
	return fr.AddAction("set_meta_data", data)
}

// RemoveMetadata removes metadata keys from the current function's scope.
func (fr *FunctionResult) RemoveMetadata(keys []string) *FunctionResult {
	return fr.AddAction("unset_meta_data", keys)
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
func (fr *FunctionResult) SwmlChangeStep(stepName string) *FunctionResult {
	return fr.AddAction("context_switch", map[string]any{"step": stepName})
}

// SwmlChangeContext transitions to a different conversation context.
func (fr *FunctionResult) SwmlChangeContext(contextName string) *FunctionResult {
	return fr.AddAction("context_switch", map[string]any{"context": contextName})
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

// RecordCall starts background call recording using SWML.
func (fr *FunctionResult) RecordCall(controlID string, stereo bool, format string, direction string) *FunctionResult {
	recordParams := map[string]any{
		"stereo":    stereo,
		"format":    format,
		"direction": direction,
	}
	if controlID != "" {
		recordParams["control_id"] = controlID
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

// JoinConference joins an ad-hoc audio conference.
func (fr *FunctionResult) JoinConference(name string, muted bool, beep string, holdAudio string) *FunctionResult {
	joinParams := map[string]any{"name": name}
	if muted {
		joinParams["muted"] = true
	}
	if beep != "" && beep != "true" {
		joinParams["beep"] = beep
	}
	if holdAudio != "" {
		joinParams["wait_url"] = holdAudio
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
func (fr *FunctionResult) Tap(uri string, controlID string, direction string, codec string) *FunctionResult {
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
// Pass empty string for body if only sending media, and nil for optional slices.
func (fr *FunctionResult) SendSms(toNumber, fromNumber, body string, media []string, tags []string) *FunctionResult {
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

// Pay processes a payment using SWML pay action.
func (fr *FunctionResult) Pay(connectorURL string, inputMethod string, actionURL string, timeout int, maxAttempts int) *FunctionResult {
	payParams := map[string]any{
		"payment_connector_url": connectorURL,
		"input":                 inputMethod,
		"timeout":               fmt.Sprintf("%d", timeout),
		"max_attempts":          fmt.Sprintf("%d", maxAttempts),
	}

	mainVerbs := []any{
		map[string]any{"pay": payParams},
	}

	if actionURL != "" {
		mainVerbs = append([]any{
			map[string]any{"set": map[string]any{"ai_response": actionURL}},
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
func (fr *FunctionResult) ExecuteRpc(method string, params map[string]any) *FunctionResult {
	rpcParams := map[string]any{
		"method":  method,
		"jsonrpc": "2.0",
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
// Pass nil for callTimeout to omit it, and empty string for region to omit it.
func (fr *FunctionResult) RpcDial(toNumber, fromNumber, destSwml string, callTimeout *int, region string) *FunctionResult {
	dialParams := map[string]any{
		"devices": map[string]any{
			"type": "phone",
			"params": map[string]any{
				"to_number":   toNumber,
				"from_number": fromNumber,
			},
		},
		"dest_swml": destSwml,
	}
	if callTimeout != nil {
		dialParams["timeout"] = *callTimeout
	}
	if region != "" {
		dialParams["region"] = region
	}

	return fr.ExecuteRpc("calling.dial", dialParams)
}

// RpcAiMessage injects a message into an AI agent on another call.
func (fr *FunctionResult) RpcAiMessage(callID, messageText string) *FunctionResult {
	rpcParams := map[string]any{
		"method":  "calling.ai_message",
		"jsonrpc": "2.0",
		"call_id": callID,
		"params": map[string]any{
			"role":         "system",
			"message_text": messageText,
		},
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

// RpcAiUnhold unholds another call.
func (fr *FunctionResult) RpcAiUnhold(callID string) *FunctionResult {
	rpcParams := map[string]any{
		"method":  "calling.ai_unhold",
		"jsonrpc": "2.0",
		"call_id": callID,
		"params":  map[string]any{},
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

// SimulateUserInput queues simulated user input text.
func (fr *FunctionResult) SimulateUserInput(text string) *FunctionResult {
	return fr.AddAction("simulate_user_input", text)
}

// --- Payment Helpers ---

// CreatePaymentPrompt creates a payment prompt configuration.
func CreatePaymentPrompt(forSituation string, actions []map[string]string) map[string]any {
	return map[string]any{
		"for":     forSituation,
		"actions": actions,
	}
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
