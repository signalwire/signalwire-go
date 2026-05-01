// Copyright (c) 2025 SignalWire
//
// Mock-relay-backed tests for SDK event dispatch / routing edge cases.
// Mirrors signalwire-python's tests/unit/relay/test_event_dispatch_mock.py.

package relay_test

import (
	"context"
	"testing"
	"time"

	"github.com/signalwire/signalwire-go/pkg/relay"
	"github.com/signalwire/signalwire-go/pkg/relay/internal/mocktest"
)

// bareEventFrame builds a signalwire.event frame.
func bareEventFrame(eventType string, params map[string]any) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      "evt-" + eventType,
		"method":  "signalwire.event",
		"params": map[string]any{
			"event_type": eventType,
			"params":     params,
		},
	}
}

// ---------------------------------------------------------------------------
// Sub-command journaling
// ---------------------------------------------------------------------------

// TestRelay_RecordPauseJournalsRecordPause — Python:
// test_record_pause_journals_record_pause.
func TestRelay_RecordPauseJournalsRecordPause(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "ec-rec-pa")
	action := call.Record(
		relay.WithRecordAudio(map[string]any{"format": "wav"}),
		relay.WithRecordControlID("ec-rec-pa-1"),
	)
	time.Sleep(50 * time.Millisecond)
	if err := action.Pause("continuous"); err != nil {
		t.Fatalf("Pause: %v", err)
	}
	pauses := h.JournalRecv(t, "calling.record.pause")
	if len(pauses) == 0 {
		t.Fatal("no calling.record.pause frame")
	}
	last := pauses[len(pauses)-1]
	params, _ := last.FrameParams()
	if params["control_id"] != "ec-rec-pa-1" {
		t.Errorf("control_id = %v, want ec-rec-pa-1", params["control_id"])
	}
	if params["behavior"] != "continuous" {
		t.Errorf("behavior = %v, want continuous", params["behavior"])
	}
}

// TestRelay_RecordResumeJournalsRecordResume — Python:
// test_record_resume_journals_record_resume.
func TestRelay_RecordResumeJournalsRecordResume(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "ec-rec-re")
	action := call.Record(
		relay.WithRecordAudio(map[string]any{"format": "wav"}),
		relay.WithRecordControlID("ec-rec-re-1"),
	)
	time.Sleep(50 * time.Millisecond)
	if err := action.Resume(); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	resumes := h.JournalRecv(t, "calling.record.resume")
	if len(resumes) == 0 {
		t.Fatal("no calling.record.resume frame")
	}
	params, _ := resumes[len(resumes)-1].FrameParams()
	if params["control_id"] != "ec-rec-re-1" {
		t.Errorf("control_id = %v, want ec-rec-re-1", params["control_id"])
	}
}

// TestRelay_CollectStartInputTimersJournalsCorrectly — Python:
// test_collect_start_input_timers_journals_correctly.
func TestRelay_CollectStartInputTimersJournalsCorrectly(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "ec-col-sit")
	startInputTimers := false
	action := call.Collect(&relay.CollectParams{
		Digits:           map[string]any{"max": 4},
		StartInputTimers: &startInputTimers,
		ControlID:        "ec-col-sit-1",
	})
	time.Sleep(50 * time.Millisecond)
	if err := action.StartInputTimers(); err != nil {
		t.Fatalf("StartInputTimers: %v", err)
	}
	starts := h.JournalRecv(t, "calling.collect.start_input_timers")
	if len(starts) == 0 {
		t.Fatal("no calling.collect.start_input_timers frame")
	}
	params, _ := starts[len(starts)-1].FrameParams()
	if params["control_id"] != "ec-col-sit-1" {
		t.Errorf("control_id = %v, want ec-col-sit-1", params["control_id"])
	}
}

// TestRelay_PlayVolumeCarriesNegativeValue — Python:
// test_play_volume_carries_negative_value.
func TestRelay_PlayVolumeCarriesNegativeValue(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "ec-pvol")
	action := call.Play(
		[]map[string]any{{"type": "silence", "params": map[string]any{"duration": 60}}},
		relay.WithPlayControlID("ec-pvol-1"),
	)
	time.Sleep(50 * time.Millisecond)
	if err := action.Volume(-5.5); err != nil {
		t.Fatalf("Volume: %v", err)
	}
	vol := h.JournalRecv(t, "calling.play.volume")
	if len(vol) == 0 {
		t.Fatal("no calling.play.volume frame")
	}
	params, _ := vol[len(vol)-1].FrameParams()
	if v, _ := params["volume"].(float64); v != -5.5 {
		t.Errorf("volume = %v, want -5.5", params["volume"])
	}
}

// ---------------------------------------------------------------------------
// Unknown event types — recv loop survives
// ---------------------------------------------------------------------------

// TestRelay_UnknownEventTypeDoesNotCrash — Python:
// test_unknown_event_type_does_not_crash.
func TestRelay_UnknownEventTypeDoesNotCrash(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	h.Push(t, bareEventFrame("nonsense.unknown", map[string]any{"foo": "bar"}), "")
	time.Sleep(100 * time.Millisecond)
	// Verify client still alive via ping execute.
	if _, err := client.Execute("signalwire.ping", map[string]any{}); err != nil {
		t.Fatalf("client died: %v", err)
	}
}

// TestRelay_EventWithBadCallIDIsDropped — Python:
// test_event_with_bad_call_id_is_dropped.
func TestRelay_EventWithBadCallIDIsDropped(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	h.Push(t, bareEventFrame("calling.call.play", map[string]any{
		"call_id":    "no-such-call-bogus",
		"control_id": "stranger",
		"state":      "playing",
	}), "")
	time.Sleep(100 * time.Millisecond)
	if _, err := client.Execute("signalwire.ping", map[string]any{}); err != nil {
		t.Fatalf("client died: %v", err)
	}
}

// TestRelay_EventWithEmptyEventTypeIsDropped — Python:
// test_event_with_empty_event_type_is_dropped.
func TestRelay_EventWithEmptyEventTypeIsDropped(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	h.Push(t, bareEventFrame("", map[string]any{"call_id": "x"}), "")
	time.Sleep(100 * time.Millisecond)
	if _, err := client.Execute("signalwire.ping", map[string]any{}); err != nil {
		t.Fatalf("client died: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Multi-action concurrency: 3 actions on one call
// ---------------------------------------------------------------------------

// TestRelay_ThreeConcurrentActionsResolveIndependently — Python:
// test_three_concurrent_actions_resolve_independently.
func TestRelay_ThreeConcurrentActionsResolveIndependently(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "ec-3acts")
	play1 := call.Play(
		[]map[string]any{{"type": "silence", "params": map[string]any{"duration": 60}}},
		relay.WithPlayControlID("3a-p1"),
	)
	play2 := call.Play(
		[]map[string]any{{"type": "silence", "params": map[string]any{"duration": 60}}},
		relay.WithPlayControlID("3a-p2"),
	)
	rec := call.Record(
		relay.WithRecordAudio(map[string]any{"format": "wav"}),
		relay.WithRecordControlID("3a-r1"),
	)
	time.Sleep(50 * time.Millisecond)

	// Fire only play1's finished.
	h.Push(t, bareEventFrame("calling.call.play", map[string]any{
		"call_id":    "ec-3acts",
		"control_id": "3a-p1",
		"state":      "finished",
	}), "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := play1.Wait(ctx); err != nil {
		t.Fatalf("play1.Wait: %v", err)
	}
	if !play1.IsDone() {
		t.Error("play1 not done")
	}
	if play2.IsDone() {
		t.Error("play2 unexpectedly done")
	}
	if rec.IsDone() {
		t.Error("rec unexpectedly done")
	}
	// Fire play2's.
	h.Push(t, bareEventFrame("calling.call.play", map[string]any{
		"call_id":    "ec-3acts",
		"control_id": "3a-p2",
		"state":      "finished",
	}), "")
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()
	if _, err := play2.Wait(ctx2); err != nil {
		t.Fatalf("play2.Wait: %v", err)
	}
	if !play2.IsDone() {
		t.Error("play2 not done")
	}
	if rec.IsDone() {
		t.Error("rec unexpectedly done after play2")
	}
}

// ---------------------------------------------------------------------------
// Event ACK round-trip — server-pushed events get ack frames back
// ---------------------------------------------------------------------------

// TestRelay_EventAckSentBackToServer — Python:
// test_event_ack_sent_back_to_server.
func TestRelay_EventAckSentBackToServer(t *testing.T) {
	_, h := mocktest.New(t)
	if h == nil {
		return
	}
	evtID := "evt-ack-test-1"
	h.Push(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      evtID,
		"method":  "signalwire.event",
		"params": map[string]any{
			"event_type": "calling.call.play",
			"params": map[string]any{
				"call_id":    "anything",
				"control_id": "x",
				"state":      "playing",
			},
		},
	}, "")
	// Wait for the recv loop to ACK.
	time.Sleep(300 * time.Millisecond)

	j := h.Journal(t)
	var found bool
	for _, e := range j {
		if e.Direction != "recv" {
			continue
		}
		if e.Frame == nil {
			continue
		}
		if e.Frame["id"] != evtID {
			continue
		}
		if _, has := e.Frame["result"]; has {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("no event ACK with id=%q in journal", evtID)
	}
}

// ---------------------------------------------------------------------------
// Tag-based dial routing — call.call_id nested
// ---------------------------------------------------------------------------

// TestRelay_DialEventRoutesViaTagWhenNoTopLevelCallID — Python:
// test_dial_event_routes_via_tag_when_no_top_level_call_id.
func TestRelay_DialEventRoutesViaTagWhenNoTopLevelCallID(t *testing.T) {
	_, h := mocktest.New(t)
	if h == nil {
		return
	}
	client := mocktest.NewClientOnly(t, h,
		relay.WithProject("p"),
		relay.WithToken("t"),
		relay.WithContexts("default"),
	)
	h.ArmDial(t, mocktest.DialOpts{
		Tag:          "ec-tag-route",
		WinnerCallID: "WINTAG",
		States:       []string{"created", "answered"},
		NodeID:       "n",
		Device:       map[string]any{"type": "phone", "params": map[string]any{}},
	})
	call, err := client.Dial(
		[][]map[string]any{{phoneDevice("+1", "+2")}},
		relay.WithDialTag("ec-tag-route"),
		relay.WithDialClientTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	if call.CallID() != "WINTAG" {
		t.Errorf("CallID = %q, want WINTAG", call.CallID())
	}
	sends := h.JournalSend(t, "calling.call.dial")
	if len(sends) == 0 {
		t.Fatal("no calling.call.dial event in journal")
	}
	last := sends[len(sends)-1]
	inner, _ := last.EventParams()
	// Top-level params: tag, dial_state, call. NO call_id.
	if _, has := inner["call_id"]; has {
		t.Error("calling.call.dial event should not carry top-level call_id")
	}
	callInfo, _ := inner["call"].(map[string]any)
	if callInfo["call_id"] != "WINTAG" {
		t.Errorf("call.call_id = %v, want WINTAG", callInfo["call_id"])
	}
}

// ---------------------------------------------------------------------------
// Server ping handling
// ---------------------------------------------------------------------------

// TestRelay_ServerPingAckedBySDK — Python:
// test_server_ping_acked_by_sdk.
func TestRelay_ServerPingAckedBySDK(t *testing.T) {
	_, h := mocktest.New(t)
	if h == nil {
		return
	}
	pingID := "ping-test-1"
	h.Push(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      pingID,
		"method":  "signalwire.ping",
		"params":  map[string]any{},
	}, "")
	time.Sleep(300 * time.Millisecond)

	j := h.Journal(t)
	var found bool
	for _, e := range j {
		if e.Direction != "recv" {
			continue
		}
		if e.Frame == nil {
			continue
		}
		if e.Frame["id"] != pingID {
			continue
		}
		if _, has := e.Frame["result"]; has {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("SDK did not respond to ping id=%q", pingID)
	}
}

// ---------------------------------------------------------------------------
// Authorization state — captured for reconnect
// ---------------------------------------------------------------------------

// TestRelay_AuthorizationStateEventCaptured — Python:
// test_authorization_state_event_captured.
func TestRelay_AuthorizationStateEventCaptured(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	h.Push(t, bareEventFrame("signalwire.authorization.state", map[string]any{
		"authorization_state": "test-auth-state-blob",
	}), "")
	if !waitFor(2*time.Second, func() bool {
		return client.AuthorizationState() != ""
	}) {
		t.Fatalf("authorization state never updated; got %q", client.AuthorizationState())
	}
	if client.AuthorizationState() != "test-auth-state-blob" {
		t.Errorf("AuthorizationState() = %q, want %q", client.AuthorizationState(), "test-auth-state-blob")
	}
}

// ---------------------------------------------------------------------------
// Calling.error event — does not raise into the SDK
// ---------------------------------------------------------------------------

// TestRelay_CallingErrorEventDoesNotCrash — Python:
// test_calling_error_event_does_not_crash.
func TestRelay_CallingErrorEventDoesNotCrash(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	h.Push(t, bareEventFrame("calling.error", map[string]any{
		"code":    "5001",
		"message": "synthetic error",
	}), "")
	time.Sleep(100 * time.Millisecond)
	if _, err := client.Execute("signalwire.ping", map[string]any{}); err != nil {
		t.Fatalf("client died: %v", err)
	}
}

// ---------------------------------------------------------------------------
// State event for an answered call updates Call.state
// ---------------------------------------------------------------------------

// TestRelay_CallStateEventUpdatesState — Python:
// test_call_state_event_updates_state.
func TestRelay_CallStateEventUpdatesState(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "ec-stt")
	h.Push(t, bareEventFrame("calling.call.state", map[string]any{
		"call_id":    "ec-stt",
		"call_state": "ending",
		"direction":  "inbound",
	}), "")
	if !waitFor(2*time.Second, func() bool {
		return call.State() == "ending"
	}) {
		t.Errorf("State = %q, want ending", call.State())
	}
}

// TestRelay_CallListenerFiresOnEvent — Python:
// test_call_listener_fires_on_event.
func TestRelay_CallListenerFiresOnEvent(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "ec-list")
	fired := make(chan *relay.RelayEvent, 1)
	call.On("calling.call.play", func(e *relay.RelayEvent) {
		fired <- e
	})
	h.Push(t, bareEventFrame("calling.call.play", map[string]any{
		"call_id":    "ec-list",
		"control_id": "x",
		"state":      "playing",
	}), "")
	select {
	case e := <-fired:
		if e.EventType != "calling.call.play" {
			t.Errorf("event_type = %q, want calling.call.play", e.EventType)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("call listener did not fire")
	}
}
