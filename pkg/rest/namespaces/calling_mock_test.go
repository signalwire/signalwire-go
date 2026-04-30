// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Mock-backed unit tests translated from
// signalwire-python/tests/unit/rest/test_calling_mock.py.
//
// Every command in CallingNamespace is exercised against the mock server so
// we know the SDK sends the right wire request — method, path, command
// field, and (where applicable) the id and params.

package namespaces_test

import (
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

const callsPath = "/api/calling/calls"

// commandAssert checks a calling-command dispatch journal entry. expectedID
// may be "" to mean "no id field at body root" (only true for Dial / Update,
// which carry id inside params).
func commandAssert(t *testing.T, j mocktest.JournalEntry, command, expectedID string) map[string]any {
	t.Helper()
	if j.Method != "POST" {
		t.Errorf("method = %q, want POST", j.Method)
	}
	if j.Path != callsPath {
		t.Errorf("path = %q, want %q", j.Path, callsPath)
	}
	body, ok := j.BodyMap()
	if !ok {
		t.Fatalf("body type = %T", j.Body)
	}
	if body["command"] != command {
		t.Errorf("command = %v, want %q", body["command"], command)
	}
	if expectedID == "" {
		if _, present := body["id"]; present {
			t.Errorf("expected no id at body root, got %v", body["id"])
		}
	} else if body["id"] != expectedID {
		t.Errorf("id = %v, want %q", body["id"], expectedID)
	}
	params, _ := body["params"].(map[string]any)
	return params
}

// ----------------- Lifecycle -----------------

func TestCalling_Update(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Calling.Update(map[string]any{"id": "call-1", "state": "hold"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if _, ok := body["id"]; !ok {
		t.Errorf("response missing 'id', got keys %v", keys(body))
	}
	params := commandAssert(t, mock.Last(t), "update", "")
	if params["id"] != "call-1" {
		t.Errorf("params[id] = %v, want call-1", params["id"])
	}
	if params["state"] != "hold" {
		t.Errorf("params[state] = %v, want hold", params["state"])
	}
}

func TestCalling_Transfer(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Calling.Transfer("call-123", map[string]any{
		"destination": "+15551234567",
		"from_number": "+15559876543",
	})
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	if _, ok := body["id"]; !ok {
		t.Errorf("response missing 'id'")
	}
	params := commandAssert(t, mock.Last(t), "calling.transfer", "call-123")
	if params["destination"] != "+15551234567" {
		t.Errorf("params[destination] = %v", params["destination"])
	}
	if params["from_number"] != "+15559876543" {
		t.Errorf("params[from_number] = %v", params["from_number"])
	}
}

func TestCalling_Disconnect(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	body, err := client.Calling.Disconnect("call-456", map[string]any{"reason": "busy"})
	if err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	if _, ok := body["id"]; !ok {
		t.Errorf("response missing 'id'")
	}
	params := commandAssert(t, mock.Last(t), "calling.disconnect", "call-456")
	if params["reason"] != "busy" {
		t.Errorf("params[reason] = %v", params["reason"])
	}
}

// ----------------- Play -----------------

func TestCalling_PlayPause(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.PlayPause("call-1", map[string]any{"control_id": "ctrl-1"})
	if err != nil {
		t.Fatalf("PlayPause: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.play.pause", "call-1")
	if params["control_id"] != "ctrl-1" {
		t.Errorf("control_id = %v", params["control_id"])
	}
}

func TestCalling_PlayResume(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.PlayResume("call-1", map[string]any{"control_id": "ctrl-1"})
	if err != nil {
		t.Fatalf("PlayResume: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.play.resume", "call-1")
	if params["control_id"] != "ctrl-1" {
		t.Errorf("control_id = %v", params["control_id"])
	}
}

func TestCalling_PlayStop(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.PlayStop("call-1", map[string]any{"control_id": "ctrl-1"})
	if err != nil {
		t.Fatalf("PlayStop: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.play.stop", "call-1")
	if params["control_id"] != "ctrl-1" {
		t.Errorf("control_id = %v", params["control_id"])
	}
}

func TestCalling_PlayVolume(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.PlayVolume("call-1", map[string]any{
		"control_id": "ctrl-1",
		"volume":     2.5,
	})
	if err != nil {
		t.Fatalf("PlayVolume: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.play.volume", "call-1")
	if params["volume"] != 2.5 {
		t.Errorf("volume = %v, want 2.5", params["volume"])
	}
}

// ----------------- Record -----------------

func TestCalling_Record(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.Record("call-1", map[string]any{"record": map[string]any{"format": "mp3"}})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.record", "call-1")
	rec, _ := params["record"].(map[string]any)
	if rec["format"] != "mp3" {
		t.Errorf("record.format = %v", rec["format"])
	}
}

func TestCalling_RecordPause(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.RecordPause("call-1", map[string]any{"control_id": "rec-1"})
	if err != nil {
		t.Fatalf("RecordPause: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.record.pause", "call-1")
	if params["control_id"] != "rec-1" {
		t.Errorf("control_id = %v", params["control_id"])
	}
}

func TestCalling_RecordResume(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.RecordResume("call-1", map[string]any{"control_id": "rec-1"})
	if err != nil {
		t.Fatalf("RecordResume: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.record.resume", "call-1")
	if params["control_id"] != "rec-1" {
		t.Errorf("control_id = %v", params["control_id"])
	}
}

// ----------------- Collect -----------------

func TestCalling_Collect(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.Collect("call-1", map[string]any{
		"initial_timeout": 5,
		"digits":          map[string]any{"max": 4},
	})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.collect", "call-1")
	// JSON decoding turns numbers into float64.
	if v, ok := params["initial_timeout"].(float64); !ok || v != 5 {
		t.Errorf("initial_timeout = %v (%T), want 5", params["initial_timeout"], params["initial_timeout"])
	}
}

func TestCalling_CollectStop(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.CollectStop("call-1", map[string]any{"control_id": "col-1"})
	if err != nil {
		t.Fatalf("CollectStop: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.collect.stop", "call-1")
	if params["control_id"] != "col-1" {
		t.Errorf("control_id = %v", params["control_id"])
	}
}

func TestCalling_CollectStartInputTimers(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.CollectStartInputTimers("call-1", map[string]any{"control_id": "col-1"})
	if err != nil {
		t.Fatalf("CollectStartInputTimers: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.collect.start_input_timers", "call-1")
	if params["control_id"] != "col-1" {
		t.Errorf("control_id = %v", params["control_id"])
	}
}

// ----------------- Detect / tap / stream / denoise / transcribe -----------------

func TestCalling_Detect(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.Detect("call-1", map[string]any{
		"detect": map[string]any{"type": "machine", "params": map[string]any{}},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.detect", "call-1")
	det, _ := params["detect"].(map[string]any)
	if det["type"] != "machine" {
		t.Errorf("detect.type = %v", det["type"])
	}
}

func TestCalling_DetectStop(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.DetectStop("call-1", map[string]any{"control_id": "det-1"})
	if err != nil {
		t.Fatalf("DetectStop: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.detect.stop", "call-1")
	if params["control_id"] != "det-1" {
		t.Errorf("control_id = %v", params["control_id"])
	}
}

func TestCalling_Tap(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.Tap("call-1", map[string]any{
		"tap":    map[string]any{"type": "audio"},
		"device": map[string]any{"type": "rtp"},
	})
	if err != nil {
		t.Fatalf("Tap: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.tap", "call-1")
	tap, _ := params["tap"].(map[string]any)
	if tap["type"] != "audio" {
		t.Errorf("tap.type = %v", tap["type"])
	}
}

func TestCalling_TapStop(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.TapStop("call-1", map[string]any{"control_id": "tap-1"})
	if err != nil {
		t.Fatalf("TapStop: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.tap.stop", "call-1")
	if params["control_id"] != "tap-1" {
		t.Errorf("control_id = %v", params["control_id"])
	}
}

func TestCalling_Stream(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.Stream("call-1", map[string]any{"url": "wss://example.com/audio"})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.stream", "call-1")
	if params["url"] != "wss://example.com/audio" {
		t.Errorf("url = %v", params["url"])
	}
}

func TestCalling_StreamStop(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.StreamStop("call-1", map[string]any{"control_id": "stream-1"})
	if err != nil {
		t.Fatalf("StreamStop: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.stream.stop", "call-1")
	if params["control_id"] != "stream-1" {
		t.Errorf("control_id = %v", params["control_id"])
	}
}

func TestCalling_Denoise(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.Denoise("call-1", nil)
	if err != nil {
		t.Fatalf("Denoise: %v", err)
	}
	commandAssert(t, mock.Last(t), "calling.denoise", "call-1")
}

func TestCalling_DenoiseStop(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.DenoiseStop("call-1", map[string]any{"control_id": "dn-1"})
	if err != nil {
		t.Fatalf("DenoiseStop: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.denoise.stop", "call-1")
	if params["control_id"] != "dn-1" {
		t.Errorf("control_id = %v", params["control_id"])
	}
}

func TestCalling_Transcribe(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.Transcribe("call-1", map[string]any{
		"language":   "en-US",
		"transcribe": map[string]any{"engine": "google"},
	})
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.transcribe", "call-1")
	if params["language"] != "en-US" {
		t.Errorf("language = %v", params["language"])
	}
}

func TestCalling_TranscribeStop(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.TranscribeStop("call-1", map[string]any{"control_id": "tr-1"})
	if err != nil {
		t.Fatalf("TranscribeStop: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.transcribe.stop", "call-1")
	if params["control_id"] != "tr-1" {
		t.Errorf("control_id = %v", params["control_id"])
	}
}

// ----------------- AI -----------------

func TestCalling_AIHold(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.AIHold("call-1", nil)
	if err != nil {
		t.Fatalf("AIHold: %v", err)
	}
	commandAssert(t, mock.Last(t), "calling.ai_hold", "call-1")
}

func TestCalling_AIUnhold(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.AIUnhold("call-1", nil)
	if err != nil {
		t.Fatalf("AIUnhold: %v", err)
	}
	commandAssert(t, mock.Last(t), "calling.ai_unhold", "call-1")
}

func TestCalling_AIStop(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.AIStop("call-1", nil)
	if err != nil {
		t.Fatalf("AIStop: %v", err)
	}
	commandAssert(t, mock.Last(t), "calling.ai.stop", "call-1")
}

// ----------------- Live transcribe / translate -----------------

func TestCalling_LiveTranscribe(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.LiveTranscribe("call-1", map[string]any{"language": "en-US"})
	if err != nil {
		t.Fatalf("LiveTranscribe: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.live_transcribe", "call-1")
	if params["language"] != "en-US" {
		t.Errorf("language = %v", params["language"])
	}
}

func TestCalling_LiveTranslate(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.LiveTranslate("call-1", map[string]any{
		"source_language": "en",
		"target_language": "es",
	})
	if err != nil {
		t.Fatalf("LiveTranslate: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.live_translate", "call-1")
	if params["source_language"] != "en" {
		t.Errorf("source_language = %v", params["source_language"])
	}
	if params["target_language"] != "es" {
		t.Errorf("target_language = %v", params["target_language"])
	}
}

// ----------------- Fax -----------------

func TestCalling_SendFaxStop(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.SendFaxStop("call-1", nil)
	if err != nil {
		t.Fatalf("SendFaxStop: %v", err)
	}
	commandAssert(t, mock.Last(t), "calling.send_fax.stop", "call-1")
}

func TestCalling_ReceiveFaxStop(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.ReceiveFaxStop("call-1", nil)
	if err != nil {
		t.Fatalf("ReceiveFaxStop: %v", err)
	}
	commandAssert(t, mock.Last(t), "calling.receive_fax.stop", "call-1")
}

// ----------------- SIP refer + custom user_event -----------------

func TestCalling_Refer(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.Refer("call-1", map[string]any{"to": "sip:other@example.com"})
	if err != nil {
		t.Fatalf("Refer: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.refer", "call-1")
	if params["to"] != "sip:other@example.com" {
		t.Errorf("to = %v", params["to"])
	}
}

func TestCalling_UserEvent(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	_, err := client.Calling.UserEvent("call-1", map[string]any{
		"event_name": "my-event",
		"payload":    map[string]any{"foo": "bar"},
	})
	if err != nil {
		t.Fatalf("UserEvent: %v", err)
	}
	params := commandAssert(t, mock.Last(t), "calling.user_event", "call-1")
	if params["event_name"] != "my-event" {
		t.Errorf("event_name = %v", params["event_name"])
	}
	pl, _ := params["payload"].(map[string]any)
	if pl["foo"] != "bar" {
		t.Errorf("payload.foo = %v", pl["foo"])
	}
}
