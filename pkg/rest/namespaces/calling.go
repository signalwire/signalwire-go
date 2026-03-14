// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

// CallingNamespace provides REST-based call control. All commands are
// dispatched as POST /api/calling/calls with a "command" field.
type CallingNamespace struct {
	Resource
}

// NewCallingNamespace creates a new CallingNamespace.
func NewCallingNamespace(client HTTPClient) *CallingNamespace {
	return &CallingNamespace{
		Resource: Resource{HTTP: client, Base: "/api/calling/calls"},
	}
}

// execute sends a command to the calling endpoint.
func (c *CallingNamespace) execute(command string, callID string, params map[string]any) (map[string]any, error) {
	body := map[string]any{
		"command": command,
		"params":  params,
	}
	if callID != "" {
		body["id"] = callID
	}
	return c.HTTP.Post(c.Base, body)
}

// --- Call lifecycle ---

// Dial initiates a new call.
func (c *CallingNamespace) Dial(params map[string]any) (map[string]any, error) {
	return c.execute("dial", "", params)
}

// Update updates call parameters.
func (c *CallingNamespace) Update(params map[string]any) (map[string]any, error) {
	return c.execute("update", "", params)
}

// End terminates a call.
func (c *CallingNamespace) End(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.end", callID, params)
}

// Transfer transfers a call.
func (c *CallingNamespace) Transfer(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.transfer", callID, params)
}

// Disconnect disconnects a call.
func (c *CallingNamespace) Disconnect(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.disconnect", callID, params)
}

// --- Play ---

// Play starts playback on a call.
func (c *CallingNamespace) Play(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.play", callID, params)
}

// PlayPause pauses playback.
func (c *CallingNamespace) PlayPause(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.play.pause", callID, params)
}

// PlayResume resumes playback.
func (c *CallingNamespace) PlayResume(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.play.resume", callID, params)
}

// PlayStop stops playback.
func (c *CallingNamespace) PlayStop(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.play.stop", callID, params)
}

// PlayVolume adjusts playback volume.
func (c *CallingNamespace) PlayVolume(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.play.volume", callID, params)
}

// --- Record ---

// Record starts recording on a call.
func (c *CallingNamespace) Record(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.record", callID, params)
}

// RecordPause pauses recording.
func (c *CallingNamespace) RecordPause(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.record.pause", callID, params)
}

// RecordResume resumes recording.
func (c *CallingNamespace) RecordResume(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.record.resume", callID, params)
}

// RecordStop stops recording.
func (c *CallingNamespace) RecordStop(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.record.stop", callID, params)
}

// --- Collect ---

// Collect starts input collection on a call.
func (c *CallingNamespace) Collect(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.collect", callID, params)
}

// CollectStop stops input collection.
func (c *CallingNamespace) CollectStop(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.collect.stop", callID, params)
}

// CollectStartInputTimers starts input timers for collection.
func (c *CallingNamespace) CollectStartInputTimers(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.collect.start_input_timers", callID, params)
}

// --- Detect ---

// Detect starts detection (e.g., answering machine) on a call.
func (c *CallingNamespace) Detect(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.detect", callID, params)
}

// DetectStop stops detection.
func (c *CallingNamespace) DetectStop(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.detect.stop", callID, params)
}

// --- Tap ---

// Tap starts tapping a call.
func (c *CallingNamespace) Tap(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.tap", callID, params)
}

// TapStop stops tapping.
func (c *CallingNamespace) TapStop(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.tap.stop", callID, params)
}

// --- Stream ---

// Stream starts streaming on a call.
func (c *CallingNamespace) Stream(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.stream", callID, params)
}

// StreamStop stops streaming.
func (c *CallingNamespace) StreamStop(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.stream.stop", callID, params)
}

// --- Denoise ---

// Denoise enables denoising on a call.
func (c *CallingNamespace) Denoise(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.denoise", callID, params)
}

// DenoiseStop disables denoising.
func (c *CallingNamespace) DenoiseStop(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.denoise.stop", callID, params)
}

// --- Transcribe ---

// Transcribe starts transcription on a call.
func (c *CallingNamespace) Transcribe(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.transcribe", callID, params)
}

// TranscribeStop stops transcription.
func (c *CallingNamespace) TranscribeStop(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.transcribe.stop", callID, params)
}

// --- AI ---

// AIMessage sends a message to the AI agent on a call.
func (c *CallingNamespace) AIMessage(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.ai_message", callID, params)
}

// AIHold puts the AI on hold.
func (c *CallingNamespace) AIHold(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.ai_hold", callID, params)
}

// AIUnhold takes the AI off hold.
func (c *CallingNamespace) AIUnhold(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.ai_unhold", callID, params)
}

// AIStop stops the AI session.
func (c *CallingNamespace) AIStop(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.ai.stop", callID, params)
}

// --- Live transcribe / translate ---

// LiveTranscribe starts live transcription.
func (c *CallingNamespace) LiveTranscribe(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.live_transcribe", callID, params)
}

// LiveTranslate starts live translation.
func (c *CallingNamespace) LiveTranslate(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.live_translate", callID, params)
}

// --- Fax ---

// SendFaxStop stops sending a fax.
func (c *CallingNamespace) SendFaxStop(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.send_fax.stop", callID, params)
}

// ReceiveFaxStop stops receiving a fax.
func (c *CallingNamespace) ReceiveFaxStop(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.receive_fax.stop", callID, params)
}

// --- SIP ---

// Refer sends a SIP REFER on a call.
func (c *CallingNamespace) Refer(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.refer", callID, params)
}

// --- Custom events ---

// UserEvent sends a custom user event on a call.
func (c *CallingNamespace) UserEvent(callID string, params map[string]any) (map[string]any, error) {
	return c.execute("calling.user_event", callID, params)
}
