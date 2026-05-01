// Copyright (c) 2025 SignalWire
//
// Mock-relay-backed tests for Action classes. Mirrors signalwire-python's
// tests/unit/relay/test_actions_mock.py.

package relay_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/signalwire/signalwire-go/pkg/relay"
	"github.com/signalwire/signalwire-go/pkg/relay/internal/mocktest"
)

// answeredInboundCall sets up the typical inbound call → answer
// sequence used by every action test. Returns the captured Call.
func answeredInboundCall(t *testing.T, client *relay.Client, h *mocktest.Harness, callID string) *relay.Call {
	t.Helper()
	captured := make(chan *relay.Call, 1)
	client.OnCall(func(c *relay.Call) {
		_ = c.Answer()
		captured <- c
	})
	h.InboundCall(t, mocktest.InboundCallOpts{
		CallID:     callID,
		AutoStates: []string{"created"},
	})
	select {
	case call := <-captured:
		return call
	case <-time.After(5 * time.Second):
		t.Fatal("answeredInboundCall: handler did not fire")
		return nil
	}
}

// ---------------------------------------------------------------------------
// PlayAction
// ---------------------------------------------------------------------------

// TestRelay_PlayJournalsCallingPlay — Python:
// test_play_journals_calling_play.
func TestRelay_PlayJournalsCallingPlay(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-play")
	_ = call.Play(
		[]map[string]any{{"type": "tts", "params": map[string]any{"text": "hi"}}},
		relay.WithPlayControlID("play-ctl-1"),
	)
	time.Sleep(150 * time.Millisecond)
	entry := h.JournalLast(t, "calling.play")
	params, _ := entry.FrameParams()
	if params["call_id"] != "call-play" {
		t.Errorf("call_id = %v, want call-play", params["call_id"])
	}
	if params["control_id"] != "play-ctl-1" {
		t.Errorf("control_id = %v, want play-ctl-1", params["control_id"])
	}
	playList, _ := params["play"].([]any)
	if len(playList) == 0 {
		t.Fatal("play list is empty")
	}
	first, _ := playList[0].(map[string]any)
	if first["type"] != "tts" {
		t.Errorf("play[0].type = %v, want tts", first["type"])
	}
}

// TestRelay_PlayResolvesOnFinishedEvent — Python:
// test_play_resolves_on_finished_event.
func TestRelay_PlayResolvesOnFinishedEvent(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-play-fin")
	h.ArmMethod(t, "calling.play", []map[string]any{
		{"emit": map[string]any{"state": "playing"}, "delay_ms": 1},
		{"emit": map[string]any{"state": "finished"}, "delay_ms": 5},
	})
	action := call.Play(
		[]map[string]any{{"type": "silence", "params": map[string]any{"duration": 1}}},
		relay.WithPlayControlID("play-ctl-fin"),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	event, err := action.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if !action.IsDone() {
		t.Error("action not done after Wait")
	}
	if event.GetString("state") != "finished" {
		t.Errorf("state = %q, want finished", event.GetString("state"))
	}
}

// TestRelay_PlayStopJournalsPlayStop — Python:
// test_play_stop_journals_play_stop.
func TestRelay_PlayStopJournalsPlayStop(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-play-stop")
	action := call.Play(
		[]map[string]any{{"type": "silence", "params": map[string]any{"duration": 60}}},
		relay.WithPlayControlID("play-ctl-stop"),
	)
	time.Sleep(50 * time.Millisecond)
	if err := action.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	stops := h.JournalRecv(t, "calling.play.stop")
	if len(stops) == 0 {
		t.Fatal("no calling.play.stop frame")
	}
	params, _ := stops[len(stops)-1].FrameParams()
	if params["control_id"] != "play-ctl-stop" {
		t.Errorf("control_id = %v, want play-ctl-stop", params["control_id"])
	}
}

// TestRelay_PlayPauseResumeVolumeJournal — Python:
// test_play_pause_resume_volume_journal.
func TestRelay_PlayPauseResumeVolumeJournal(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-play-prv")
	action := call.Play(
		[]map[string]any{{"type": "silence", "params": map[string]any{"duration": 60}}},
		relay.WithPlayControlID("play-ctl-prv"),
	)
	time.Sleep(50 * time.Millisecond)
	if err := action.Pause(); err != nil {
		t.Fatalf("Pause: %v", err)
	}
	if err := action.Resume(); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if err := action.Volume(-3.0); err != nil {
		t.Fatalf("Volume: %v", err)
	}
	if len(h.JournalRecv(t, "calling.play.pause")) == 0 {
		t.Error("no calling.play.pause frame")
	}
	if len(h.JournalRecv(t, "calling.play.resume")) == 0 {
		t.Error("no calling.play.resume frame")
	}
	vol := h.JournalRecv(t, "calling.play.volume")
	if len(vol) == 0 {
		t.Fatal("no calling.play.volume frame")
	}
	last := vol[len(vol)-1]
	params, _ := last.FrameParams()
	if v, _ := params["volume"].(float64); v != -3.0 {
		t.Errorf("volume = %v, want -3.0", params["volume"])
	}
}

// TestRelay_PlayOnCompletedCallbackFires — Python:
// test_play_on_completed_callback_fires.
func TestRelay_PlayOnCompletedCallbackFires(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-play-cb")
	h.ArmMethod(t, "calling.play", []map[string]any{
		{"emit": map[string]any{"state": "finished"}, "delay_ms": 1},
	})
	fired := make(chan *relay.RelayEvent, 1)
	action := call.Play(
		[]map[string]any{{"type": "silence", "params": map[string]any{"duration": 1}}},
		relay.WithPlayControlID("play-ctl-cb"),
		relay.WithPlayOnCompleted(func(e *relay.RelayEvent) { fired <- e }),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := action.Wait(ctx); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	select {
	case e := <-fired:
		if e.GetString("state") != "finished" {
			t.Errorf("callback event state = %q, want finished", e.GetString("state"))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("on_completed callback did not fire within 2s")
	}
}

// ---------------------------------------------------------------------------
// RecordAction
// ---------------------------------------------------------------------------

// TestRelay_RecordJournalsCallingRecord — Python:
// test_record_journals_calling_record.
func TestRelay_RecordJournalsCallingRecord(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-rec")
	_ = call.Record(
		relay.WithRecordAudio(map[string]any{"format": "mp3"}),
		relay.WithRecordControlID("rec-ctl-1"),
	)
	time.Sleep(150 * time.Millisecond)
	entry := h.JournalLast(t, "calling.record")
	params, _ := entry.FrameParams()
	if params["call_id"] != "call-rec" {
		t.Errorf("call_id = %v, want call-rec", params["call_id"])
	}
	if params["control_id"] != "rec-ctl-1" {
		t.Errorf("control_id = %v, want rec-ctl-1", params["control_id"])
	}
	rec, _ := params["record"].(map[string]any)
	audio, _ := rec["audio"].(map[string]any)
	if audio["format"] != "mp3" {
		t.Errorf("record.audio.format = %v, want mp3", audio["format"])
	}
}

// TestRelay_RecordResolvesOnFinishedEvent — Python:
// test_record_resolves_on_finished_event.
func TestRelay_RecordResolvesOnFinishedEvent(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-rec-fin")
	h.ArmMethod(t, "calling.record", []map[string]any{
		{"emit": map[string]any{"state": "recording"}, "delay_ms": 1},
		{"emit": map[string]any{"state": "finished", "url": "http://r.wav"}, "delay_ms": 5},
	})
	action := call.Record(
		relay.WithRecordAudio(map[string]any{"format": "wav"}),
		relay.WithRecordControlID("rec-ctl-fin"),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	event, err := action.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if event.GetString("state") != "finished" {
		t.Errorf("state = %q, want finished", event.GetString("state"))
	}
}

// TestRelay_RecordStopJournalsRecordStop — Python:
// test_record_stop_journals_record_stop.
func TestRelay_RecordStopJournalsRecordStop(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-rec-stop")
	action := call.Record(
		relay.WithRecordAudio(map[string]any{"format": "wav"}),
		relay.WithRecordControlID("rec-ctl-stop"),
	)
	time.Sleep(50 * time.Millisecond)
	if err := action.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	stops := h.JournalRecv(t, "calling.record.stop")
	if len(stops) == 0 {
		t.Fatal("no calling.record.stop frame")
	}
	params, _ := stops[len(stops)-1].FrameParams()
	if params["control_id"] != "rec-ctl-stop" {
		t.Errorf("control_id = %v, want rec-ctl-stop", params["control_id"])
	}
}

// ---------------------------------------------------------------------------
// DetectAction — gotcha: resolves on first detect payload
// ---------------------------------------------------------------------------

// TestRelay_DetectResolvesOnFirstDetectPayload — Python:
// test_detect_resolves_on_first_detect_payload.
func TestRelay_DetectResolvesOnFirstDetectPayload(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-det")
	h.ArmMethod(t, "calling.detect", []map[string]any{
		{
			"emit": map[string]any{
				"detect": map[string]any{
					"type":   "machine",
					"params": map[string]any{"event": "MACHINE"},
				},
			},
			"delay_ms": 1,
		},
		{"emit": map[string]any{"state": "finished"}, "delay_ms": 10},
	})
	action := call.Detect(
		map[string]any{"type": "machine", "params": map[string]any{}},
		nil,
		"det-ctl-1",
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	event, err := action.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	d, _ := event.Params["detect"].(map[string]any)
	if d["type"] != "machine" {
		t.Errorf("detect.type = %v, want machine", d["type"])
	}
}

// TestRelay_DetectStopJournalsDetectStop — Python:
// test_detect_stop_journals_detect_stop.
func TestRelay_DetectStopJournalsDetectStop(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-det-stop")
	action := call.Detect(
		map[string]any{"type": "fax", "params": map[string]any{}},
		nil,
		"det-stop",
	)
	time.Sleep(50 * time.Millisecond)
	if err := action.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	stops := h.JournalRecv(t, "calling.detect.stop")
	if len(stops) == 0 {
		t.Fatal("no calling.detect.stop frame")
	}
	params, _ := stops[len(stops)-1].FrameParams()
	if params["control_id"] != "det-stop" {
		t.Errorf("control_id = %v, want det-stop", params["control_id"])
	}
}

// ---------------------------------------------------------------------------
// CollectAction (play_and_collect) — gotcha
// ---------------------------------------------------------------------------

// TestRelay_PlayAndCollectJournalsPlayAndCollect — Python:
// test_play_and_collect_journals_play_and_collect.
func TestRelay_PlayAndCollectJournalsPlayAndCollect(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-pac")
	_ = call.PlayAndCollect(
		[]map[string]any{{"type": "tts", "params": map[string]any{"text": "Press 1"}}},
		map[string]any{"digits": map[string]any{"max": 1}},
		relay.WithPlayControlID("pac-ctl-1"),
	)
	time.Sleep(150 * time.Millisecond)
	entry := h.JournalLast(t, "calling.play_and_collect")
	params, _ := entry.FrameParams()
	if params["call_id"] != "call-pac" {
		t.Errorf("call_id = %v, want call-pac", params["call_id"])
	}
	playList, _ := params["play"].([]any)
	first, _ := playList[0].(map[string]any)
	if first["type"] != "tts" {
		t.Errorf("play[0].type = %v, want tts", first["type"])
	}
	collect, _ := params["collect"].(map[string]any)
	digits, _ := collect["digits"].(map[string]any)
	if v, _ := digits["max"].(float64); int(v) != 1 {
		t.Errorf("collect.digits.max = %v, want 1", digits["max"])
	}
}

// TestRelay_PlayAndCollectResolvesOnCollectEventOnly — Python:
// test_play_and_collect_resolves_on_collect_event_only.
// Verifies the gotcha: a play(finished) MUST NOT resolve, only a
// calling.call.collect event resolves.
func TestRelay_PlayAndCollectResolvesOnCollectEventOnly(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-pac-go")
	action := call.PlayAndCollect(
		[]map[string]any{{"type": "silence", "params": map[string]any{"duration": 1}}},
		map[string]any{"digits": map[string]any{"max": 1}},
		relay.WithPlayControlID("pac-go"),
	)
	time.Sleep(100 * time.Millisecond)
	// Push a play(finished) — action MUST NOT resolve.
	h.Push(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      "evt-pac-play",
		"method":  "signalwire.event",
		"params": map[string]any{
			"event_type": "calling.call.play",
			"params": map[string]any{
				"call_id":    "call-pac-go",
				"control_id": "pac-go",
				"state":      "finished",
			},
		},
	}, "")
	time.Sleep(150 * time.Millisecond)
	if action.IsDone() {
		t.Fatal("play_and_collect resolved on play(finished); should wait for collect")
	}
	// Now push the collect event — action resolves.
	h.Push(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      "evt-pac-collect",
		"method":  "signalwire.event",
		"params": map[string]any{
			"event_type": "calling.call.collect",
			"params": map[string]any{
				"call_id":    "call-pac-go",
				"control_id": "pac-go",
				"result": map[string]any{
					"type":   "digit",
					"params": map[string]any{"digits": "1"},
				},
			},
		},
	}, "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	event, err := action.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if event.EventType != "calling.call.collect" {
		t.Errorf("event_type = %q, want calling.call.collect", event.EventType)
	}
	res, _ := event.Params["result"].(map[string]any)
	if res["type"] != "digit" {
		t.Errorf("result.type = %v, want digit", res["type"])
	}
}

// TestRelay_PlayAndCollectStopJournalsPACStop — Python:
// test_play_and_collect_stop_journals_pac_stop.
func TestRelay_PlayAndCollectStopJournalsPACStop(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-pac-stop")
	action := call.PlayAndCollect(
		[]map[string]any{{"type": "silence", "params": map[string]any{"duration": 1}}},
		map[string]any{"digits": map[string]any{"max": 1}},
		relay.WithPlayControlID("pac-stop"),
	)
	time.Sleep(50 * time.Millisecond)
	if err := action.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	stops := h.JournalRecv(t, "calling.play_and_collect.stop")
	if len(stops) == 0 {
		t.Fatal("no calling.play_and_collect.stop frame")
	}
	params, _ := stops[len(stops)-1].FrameParams()
	if params["control_id"] != "pac-stop" {
		t.Errorf("control_id = %v, want pac-stop", params["control_id"])
	}
}

// ---------------------------------------------------------------------------
// StandaloneCollectAction
// ---------------------------------------------------------------------------

// TestRelay_CollectJournalsCallingCollect — Python:
// test_collect_journals_calling_collect.
func TestRelay_CollectJournalsCallingCollect(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-col")
	_ = call.Collect(&relay.CollectParams{
		Digits:    map[string]any{"max": 4},
		ControlID: "col-ctl",
	})
	time.Sleep(150 * time.Millisecond)
	entry := h.JournalLast(t, "calling.collect")
	params, _ := entry.FrameParams()
	digits, _ := params["digits"].(map[string]any)
	if v, _ := digits["max"].(float64); int(v) != 4 {
		t.Errorf("digits.max = %v, want 4", digits["max"])
	}
	if params["control_id"] != "col-ctl" {
		t.Errorf("control_id = %v, want col-ctl", params["control_id"])
	}
}

// TestRelay_CollectStopJournalsCollectStop — Python:
// test_collect_stop_journals_collect_stop.
func TestRelay_CollectStopJournalsCollectStop(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-col-stop")
	action := call.Collect(&relay.CollectParams{
		Digits:    map[string]any{"max": 4},
		ControlID: "col-stop",
	})
	time.Sleep(50 * time.Millisecond)
	if err := action.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	stops := h.JournalRecv(t, "calling.collect.stop")
	if len(stops) == 0 {
		t.Fatal("no calling.collect.stop frame")
	}
	params, _ := stops[len(stops)-1].FrameParams()
	if params["control_id"] != "col-stop" {
		t.Errorf("control_id = %v, want col-stop", params["control_id"])
	}
}

// ---------------------------------------------------------------------------
// PayAction
// ---------------------------------------------------------------------------

// TestRelay_PayJournalsCallingPay — Python:
// test_pay_journals_calling_pay.
func TestRelay_PayJournalsCallingPay(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-pay")
	_ = call.Pay(
		"https://pay.example/connect",
		relay.WithPayControlID("pay-ctl"),
		relay.WithPayChargeAmount("9.99"),
	)
	time.Sleep(150 * time.Millisecond)
	entry := h.JournalLast(t, "calling.pay")
	params, _ := entry.FrameParams()
	if params["payment_connector_url"] != "https://pay.example/connect" {
		t.Errorf("payment_connector_url = %v", params["payment_connector_url"])
	}
	if params["control_id"] != "pay-ctl" {
		t.Errorf("control_id = %v, want pay-ctl", params["control_id"])
	}
	if params["charge_amount"] != "9.99" {
		t.Errorf("charge_amount = %v, want 9.99", params["charge_amount"])
	}
}

// TestRelay_PayReturnsPayAction — Python:
// test_pay_returns_pay_action.
func TestRelay_PayReturnsPayAction(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-pay-act")
	action := call.Pay(
		"https://pay.example/connect",
		relay.WithPayControlID("pay-act"),
	)
	if action == nil {
		t.Fatal("Pay returned nil action")
	}
	if action.ControlID() != "pay-act" {
		t.Errorf("ControlID = %q, want pay-act", action.ControlID())
	}
}

// TestRelay_PayStopJournalsPayStop — Python:
// test_pay_stop_journals_pay_stop.
func TestRelay_PayStopJournalsPayStop(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-pay-stop")
	action := call.Pay(
		"https://pay.example/connect",
		relay.WithPayControlID("pay-stop"),
	)
	time.Sleep(50 * time.Millisecond)
	if err := action.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	stops := h.JournalRecv(t, "calling.pay.stop")
	if len(stops) == 0 {
		t.Fatal("no calling.pay.stop frame")
	}
	params, _ := stops[len(stops)-1].FrameParams()
	if params["control_id"] != "pay-stop" {
		t.Errorf("control_id = %v, want pay-stop", params["control_id"])
	}
}

// ---------------------------------------------------------------------------
// FaxAction
// ---------------------------------------------------------------------------

// TestRelay_SendFaxJournalsCallingSendFax — Python:
// test_send_fax_journals_calling_send_fax.
func TestRelay_SendFaxJournalsCallingSendFax(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-sfax")
	_ = call.SendFax(
		"https://docs.example/test.pdf",
		"+15551112222",
		relay.WithFaxControlID("sfax-ctl"),
	)
	time.Sleep(150 * time.Millisecond)
	entry := h.JournalLast(t, "calling.send_fax")
	params, _ := entry.FrameParams()
	if params["document"] != "https://docs.example/test.pdf" {
		t.Errorf("document = %v", params["document"])
	}
	if params["identity"] != "+15551112222" {
		t.Errorf("identity = %v", params["identity"])
	}
	if params["control_id"] != "sfax-ctl" {
		t.Errorf("control_id = %v, want sfax-ctl", params["control_id"])
	}
}

// TestRelay_ReceiveFaxReturnsFaxAction — Python:
// test_receive_fax_returns_fax_action.
func TestRelay_ReceiveFaxReturnsFaxAction(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-rfax")
	action := call.ReceiveFax(relay.WithFaxControlID("rfax-ctl"))
	if action == nil {
		t.Fatal("ReceiveFax returned nil")
	}
	if action.ControlID() != "rfax-ctl" {
		t.Errorf("ControlID = %q, want rfax-ctl", action.ControlID())
	}
}

// ---------------------------------------------------------------------------
// TapAction
// ---------------------------------------------------------------------------

// TestRelay_TapJournalsCallingTap — Python:
// test_tap_journals_calling_tap.
func TestRelay_TapJournalsCallingTap(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-tap")
	_ = call.Tap(
		map[string]any{"type": "audio"},
		map[string]any{"type": "rtp", "params": map[string]any{"addr": "203.0.113.1", "port": 4000}},
		"tap-ctl",
	)
	time.Sleep(150 * time.Millisecond)
	entry := h.JournalLast(t, "calling.tap")
	params, _ := entry.FrameParams()
	tap, _ := params["tap"].(map[string]any)
	if tap["type"] != "audio" {
		t.Errorf("tap.type = %v, want audio", tap["type"])
	}
	dev, _ := params["device"].(map[string]any)
	devParams, _ := dev["params"].(map[string]any)
	if v, _ := devParams["port"].(float64); int(v) != 4000 {
		t.Errorf("device.params.port = %v, want 4000", devParams["port"])
	}
	if params["control_id"] != "tap-ctl" {
		t.Errorf("control_id = %v, want tap-ctl", params["control_id"])
	}
}

// TestRelay_TapStopJournalsTapStop — Python:
// test_tap_stop_journals_tap_stop.
func TestRelay_TapStopJournalsTapStop(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-tap-stop")
	action := call.Tap(
		map[string]any{"type": "audio"},
		map[string]any{"type": "rtp", "params": map[string]any{"addr": "203.0.113.1", "port": 4000}},
		"tap-stop",
	)
	time.Sleep(50 * time.Millisecond)
	if err := action.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	stops := h.JournalRecv(t, "calling.tap.stop")
	if len(stops) == 0 {
		t.Fatal("no calling.tap.stop frame")
	}
	params, _ := stops[len(stops)-1].FrameParams()
	if params["control_id"] != "tap-stop" {
		t.Errorf("control_id = %v, want tap-stop", params["control_id"])
	}
}

// ---------------------------------------------------------------------------
// StreamAction
// ---------------------------------------------------------------------------

// TestRelay_StreamJournalsCallingStream — Python:
// test_stream_journals_calling_stream.
func TestRelay_StreamJournalsCallingStream(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-strm")
	_ = call.Stream(
		"wss://stream.example/audio",
		relay.WithStreamCodec("OPUS@48000h"),
		relay.WithStreamControlID("strm-ctl"),
	)
	time.Sleep(150 * time.Millisecond)
	entry := h.JournalLast(t, "calling.stream")
	params, _ := entry.FrameParams()
	if params["url"] != "wss://stream.example/audio" {
		t.Errorf("url = %v", params["url"])
	}
	if params["codec"] != "OPUS@48000h" {
		t.Errorf("codec = %v", params["codec"])
	}
	if params["control_id"] != "strm-ctl" {
		t.Errorf("control_id = %v, want strm-ctl", params["control_id"])
	}
}

// TestRelay_StreamStopJournalsStreamStop — Python:
// test_stream_stop_journals_stream_stop.
func TestRelay_StreamStopJournalsStreamStop(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-strm-stop")
	action := call.Stream(
		"wss://stream.example/audio",
		relay.WithStreamControlID("strm-stop"),
	)
	time.Sleep(50 * time.Millisecond)
	if err := action.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	stops := h.JournalRecv(t, "calling.stream.stop")
	if len(stops) == 0 {
		t.Fatal("no calling.stream.stop frame")
	}
	params, _ := stops[len(stops)-1].FrameParams()
	if params["control_id"] != "strm-stop" {
		t.Errorf("control_id = %v, want strm-stop", params["control_id"])
	}
}

// ---------------------------------------------------------------------------
// TranscribeAction
// ---------------------------------------------------------------------------

// TestRelay_TranscribeJournalsCallingTranscribe — Python:
// test_transcribe_journals_calling_transcribe.
func TestRelay_TranscribeJournalsCallingTranscribe(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-tr")
	action := call.Transcribe("", "tr-ctl")
	if action == nil {
		t.Fatal("Transcribe returned nil")
	}
	time.Sleep(150 * time.Millisecond)
	entry := h.JournalLast(t, "calling.transcribe")
	params, _ := entry.FrameParams()
	if params["control_id"] != "tr-ctl" {
		t.Errorf("control_id = %v, want tr-ctl", params["control_id"])
	}
}

// TestRelay_TranscribeStopJournalsTranscribeStop — Python:
// test_transcribe_stop_journals_transcribe_stop.
func TestRelay_TranscribeStopJournalsTranscribeStop(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-tr-stop")
	action := call.Transcribe("", "tr-stop")
	time.Sleep(50 * time.Millisecond)
	if err := action.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	stops := h.JournalRecv(t, "calling.transcribe.stop")
	if len(stops) == 0 {
		t.Fatal("no calling.transcribe.stop frame")
	}
	params, _ := stops[len(stops)-1].FrameParams()
	if params["control_id"] != "tr-stop" {
		t.Errorf("control_id = %v, want tr-stop", params["control_id"])
	}
}

// ---------------------------------------------------------------------------
// AIAction
// ---------------------------------------------------------------------------

// TestRelay_AIJournalsCallingAI — Python:
// test_ai_journals_calling_ai.
func TestRelay_AIJournalsCallingAI(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-ai")
	action := call.AI(
		relay.WithAIPrompt(map[string]any{"text": "You are helpful."}),
		relay.WithAIControlID("ai-ctl"),
	)
	if action == nil {
		t.Fatal("AI returned nil")
	}
	time.Sleep(150 * time.Millisecond)
	entry := h.JournalLast(t, "calling.ai")
	params, _ := entry.FrameParams()
	prompt, _ := params["prompt"].(map[string]any)
	if prompt["text"] != "You are helpful." {
		t.Errorf("prompt.text = %v, want 'You are helpful.'", prompt["text"])
	}
	if params["control_id"] != "ai-ctl" {
		t.Errorf("control_id = %v, want ai-ctl", params["control_id"])
	}
}

// TestRelay_AIStopJournalsAIStop — Python:
// test_ai_stop_journals_ai_stop.
func TestRelay_AIStopJournalsAIStop(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-ai-stop")
	action := call.AI(
		relay.WithAIPrompt(map[string]any{"text": "You are helpful."}),
		relay.WithAIControlID("ai-stop"),
	)
	time.Sleep(50 * time.Millisecond)
	if err := action.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	stops := h.JournalRecv(t, "calling.ai.stop")
	if len(stops) == 0 {
		t.Fatal("no calling.ai.stop frame")
	}
	params, _ := stops[len(stops)-1].FrameParams()
	if params["control_id"] != "ai-stop" {
		t.Errorf("control_id = %v, want ai-stop", params["control_id"])
	}
}

// ---------------------------------------------------------------------------
// General — control_id correlation across multiple concurrent actions
// ---------------------------------------------------------------------------

// TestRelay_ConcurrentPlayAndRecordRouteIndependently — Python:
// test_concurrent_play_and_record_route_independently.
func TestRelay_ConcurrentPlayAndRecordRouteIndependently(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "call-multi")
	playAction := call.Play(
		[]map[string]any{{"type": "silence", "params": map[string]any{"duration": 60}}},
		relay.WithPlayControlID("ctl-play-x"),
	)
	recordAction := call.Record(
		relay.WithRecordAudio(map[string]any{"format": "wav"}),
		relay.WithRecordControlID("ctl-rec-y"),
	)
	if playAction.ControlID() != "ctl-play-x" {
		t.Errorf("play.ControlID = %q", playAction.ControlID())
	}
	if recordAction.ControlID() != "ctl-rec-y" {
		t.Errorf("rec.ControlID = %q", recordAction.ControlID())
	}
	time.Sleep(50 * time.Millisecond)

	// Push finished only for play.
	h.Push(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      "evt-multi-play",
		"method":  "signalwire.event",
		"params": map[string]any{
			"event_type": "calling.call.play",
			"params": map[string]any{
				"call_id":    "call-multi",
				"control_id": "ctl-play-x",
				"state":      "finished",
			},
		},
	}, "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := playAction.Wait(ctx); err != nil {
		t.Fatalf("playAction.Wait: %v", err)
	}
	if !playAction.IsDone() {
		t.Error("playAction not done")
	}
	if recordAction.IsDone() {
		t.Error("recordAction unexpectedly done")
	}
}

// Sentinel (avoid unused import warnings on partial builds).
var _ atomic.Bool
var _ sync.Mutex
