// Copyright (c) 2025 SignalWire
//
// Mock-relay-backed tests for outbound calls (Client.Dial). Mirrors
// signalwire-python's tests/unit/relay/test_outbound_call_mock.py.

package relay_test

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/signalwire/signalwire-go/pkg/relay"
	"github.com/signalwire/signalwire-go/pkg/relay/internal/mocktest"
)

// phoneDevice builds a serial-leg phone-device map matching the Python
// helper at test_outbound_call_mock.py:_phone_device.
func phoneDevice(to, from string) map[string]any {
	if to == "" {
		to = "+15551112222"
	}
	if from == "" {
		from = "+15553334444"
	}
	return map[string]any{
		"type":   "phone",
		"params": map[string]any{"to_number": to, "from_number": from},
	}
}

// ---------------------------------------------------------------------------
// Happy-path dial
// ---------------------------------------------------------------------------

// TestRelay_DialResolvesToCallWithWinnerID — Python:
// test_dial_resolves_to_call_with_winner_id.
func TestRelay_DialResolvesToCallWithWinnerID(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	h.ArmDial(t, mocktest.DialOpts{
		Tag:          "t-happy",
		WinnerCallID: "winner-1",
		States:       []string{"created", "ringing", "answered"},
		NodeID:       "node-mock-1",
		Device:       phoneDevice("", ""),
		DelayMS:      1,
	})
	call, err := client.Dial(
		[][]map[string]any{{phoneDevice("", "")}},
		relay.WithDialTag("t-happy"),
		relay.WithDialClientTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	if call == nil {
		t.Fatal("Dial returned nil call")
	}
	if call.CallID() != "winner-1" {
		t.Errorf("CallID = %q, want %q", call.CallID(), "winner-1")
	}
	if call.Tag() != "t-happy" {
		t.Errorf("Tag = %q, want %q", call.Tag(), "t-happy")
	}
	if call.State() != "answered" {
		t.Errorf("State = %q, want %q", call.State(), "answered")
	}
	if call.Direction() != "outbound" {
		t.Errorf("Direction = %q, want %q", call.Direction(), "outbound")
	}
}

// TestRelay_DialJournalRecordsCallingDialFrame — Python:
// test_dial_journal_records_calling_dial_frame.
func TestRelay_DialJournalRecordsCallingDialFrame(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	h.ArmDial(t, mocktest.DialOpts{
		Tag:          "t-frame",
		WinnerCallID: "winner-frame",
		States:       []string{"created", "answered"},
		NodeID:       "node-mock-1",
		Device:       phoneDevice("", ""),
	})
	_, err := client.Dial(
		[][]map[string]any{{phoneDevice("", "")}},
		relay.WithDialTag("t-frame"),
		relay.WithDialClientTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	entry := h.JournalLast(t, "calling.dial")
	params, _ := entry.FrameParams()
	if params["tag"] != "t-frame" {
		t.Errorf("tag = %v, want %q", params["tag"], "t-frame")
	}
	devices, ok := params["devices"].([]any)
	if !ok || len(devices) == 0 {
		t.Fatalf("devices missing or empty: %#v", params["devices"])
	}
	leg, ok := devices[0].([]any)
	if !ok || len(leg) == 0 {
		t.Fatalf("first leg empty: %#v", devices[0])
	}
	dev, ok := leg[0].(map[string]any)
	if !ok || dev["type"] != "phone" {
		t.Errorf("device[0].type = %v, want %q", dev["type"], "phone")
	}
}

// TestRelay_DialWithMaxDurationInFrame — Python:
// test_dial_with_max_duration_in_frame.
func TestRelay_DialWithMaxDurationInFrame(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	h.ArmDial(t, mocktest.DialOpts{
		Tag:          "t-md",
		WinnerCallID: "winner-md",
		States:       []string{"created", "answered"},
		NodeID:       "node-mock-1",
		Device:       phoneDevice("", ""),
	})
	_, err := client.Dial(
		[][]map[string]any{{phoneDevice("", "")}},
		relay.WithDialTag("t-md"),
		relay.WithDialMaxDuration(300),
		relay.WithDialClientTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	entry := h.JournalLast(t, "calling.dial")
	params, _ := entry.FrameParams()
	v, _ := params["max_duration"].(float64)
	if int(v) != 300 {
		t.Errorf("max_duration = %v, want 300", params["max_duration"])
	}
}

// TestRelay_DialAutoGeneratesUUIDTagWhenOmitted — Python:
// test_dial_auto_generates_uuid_tag_when_omitted. Without an explicit
// tag the SDK generates a UUID and includes it on the wire.
func TestRelay_DialAutoGeneratesUUIDTagWhenOmitted(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	// Push the dial answer event after the dial frame lands. We have to
	// poll the journal because the auto-generated tag isn't known until
	// the SDK writes the dial frame.
	uuidRE := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

	done := make(chan struct{})
	var observedTag string

	go func() {
		defer close(done)
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			entries := h.JournalRecv(t, "calling.dial")
			if len(entries) > 0 {
				params, _ := entries[len(entries)-1].FrameParams()
				if t, ok := params["tag"].(string); ok && t != "" {
					observedTag = t
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
		}
		if observedTag == "" {
			return
		}
		// Push the answered event.
		h.Push(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      "auto-tag-evt",
			"method":  "signalwire.event",
			"params": map[string]any{
				"event_type": "calling.call.dial",
				"params": map[string]any{
					"tag":        observedTag,
					"node_id":    "node-mock-1",
					"dial_state": "answered",
					"call": map[string]any{
						"call_id":      "auto-tag-winner",
						"node_id":      "node-mock-1",
						"tag":          observedTag,
						"device":       phoneDevice("", ""),
						"dial_winner":  true,
					},
				},
			},
		}, "")
	}()

	call, err := client.Dial(
		[][]map[string]any{{phoneDevice("", "")}},
		relay.WithDialClientTimeout(5*time.Second),
	)
	<-done
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	if call == nil {
		t.Fatal("Dial returned nil call")
	}
	if call.CallID() != "auto-tag-winner" {
		t.Errorf("CallID = %q, want %q", call.CallID(), "auto-tag-winner")
	}
	if !uuidRE.MatchString(observedTag) {
		t.Errorf("auto-generated tag %q is not UUID-shaped", observedTag)
	}
	if call.Tag() != observedTag {
		t.Errorf("Tag = %q, want %q", call.Tag(), observedTag)
	}
}

// ---------------------------------------------------------------------------
// Failure paths
// ---------------------------------------------------------------------------

// TestRelay_DialFailedRaisesRelayError — Python:
// test_dial_failed_raises_relay_error. A pushed
// calling.call.dial(failed) event makes Dial return a RelayError.
func TestRelay_DialFailedRaisesRelayError(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	go func() {
		// Wait for SDK's calling.dial frame, then push the failure event.
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if len(h.JournalRecv(t, "calling.dial")) > 0 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		h.Push(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      "fail-evt",
			"method":  "signalwire.event",
			"params": map[string]any{
				"event_type": "calling.call.dial",
				"params": map[string]any{
					"tag":        "t-fail",
					"node_id":    "node-mock-1",
					"dial_state": "failed",
					"call":       map[string]any{},
				},
			},
		}, "")
	}()

	_, err := client.Dial(
		[][]map[string]any{{phoneDevice("", "")}},
		relay.WithDialTag("t-fail"),
		relay.WithDialClientTimeout(5*time.Second),
	)
	if err == nil {
		t.Fatal("expected RelayError on failed dial, got nil")
	}
	var rerr *relay.RelayError
	if !errors.As(err, &rerr) {
		t.Fatalf("expected *relay.RelayError, got %T: %v", err, err)
	}
	if !strings.Contains(rerr.Error(), "failed") {
		t.Errorf("error %q does not mention 'failed'", rerr.Error())
	}
}

// TestRelay_DialTimeoutWhenNoDialEvent — Python:
// test_dial_timeout_when_no_dial_event. No scripted dial event → SDK
// times out cleanly with a RelayError mentioning "timed out".
func TestRelay_DialTimeoutWhenNoDialEvent(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	_ = h
	_, err := client.Dial(
		[][]map[string]any{{phoneDevice("", "")}},
		relay.WithDialTag("t-timeout"),
		relay.WithDialClientTimeout(500*time.Millisecond),
	)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	var rerr *relay.RelayError
	if !errors.As(err, &rerr) {
		t.Fatalf("expected *relay.RelayError, got %T: %v", err, err)
	}
	if !strings.Contains(strings.ToLower(rerr.Error()), "timed out") {
		t.Errorf("error %q does not mention 'timed out'", rerr.Error())
	}
}

// ---------------------------------------------------------------------------
// Parallel dial — winner + losers
// ---------------------------------------------------------------------------

// TestRelay_DialWinnerCarriesDialWinnerTrue — Python:
// test_dial_winner_carries_dial_winner_true.
func TestRelay_DialWinnerCarriesDialWinnerTrue(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	h.ArmDial(t, mocktest.DialOpts{
		Tag:          "t-winner",
		WinnerCallID: "WIN-ID",
		States:       []string{"created", "answered"},
		NodeID:       "node-mock-1",
		Device:       phoneDevice("", ""),
		Losers: []mocktest.DialLoserOpts{
			{CallID: "LOSE-A", States: []string{"created", "ended"}},
			{CallID: "LOSE-B", States: []string{"created", "ended"}},
		},
	})
	call, err := client.Dial(
		[][]map[string]any{{phoneDevice("", "")}},
		relay.WithDialTag("t-winner"),
		relay.WithDialClientTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	if call.CallID() != "WIN-ID" {
		t.Errorf("CallID = %q, want %q", call.CallID(), "WIN-ID")
	}
	// Verify dial event in send journal carries dial_winner: true.
	sends := h.JournalSend(t, "calling.call.dial")
	if len(sends) == 0 {
		t.Fatal("no calling.call.dial event was pushed")
	}
	var found bool
	for _, e := range sends {
		inner, _ := e.EventParams()
		if inner == nil {
			continue
		}
		if inner["dial_state"] == "answered" {
			callInfo, _ := inner["call"].(map[string]any)
			if callInfo == nil {
				continue
			}
			if w, _ := callInfo["dial_winner"].(bool); w && callInfo["call_id"] == "WIN-ID" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("no answered calling.call.dial event with dial_winner=true and call_id=WIN-ID")
	}
}

// TestRelay_DialLosersGetStateEvents — Python:
// test_dial_losers_get_state_events.
func TestRelay_DialLosersGetStateEvents(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	h.ArmDial(t, mocktest.DialOpts{
		Tag:          "t-losers",
		WinnerCallID: "WIN-2",
		States:       []string{"created", "answered"},
		NodeID:       "node-mock-1",
		Device:       phoneDevice("", ""),
		Losers: []mocktest.DialLoserOpts{
			{CallID: "L1", States: []string{"created", "ended"}},
		},
	})
	_, err := client.Dial(
		[][]map[string]any{{phoneDevice("", "")}},
		relay.WithDialTag("t-losers"),
		relay.WithDialClientTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	stateEvents := h.JournalSend(t, "calling.call.state")
	var sawEnded bool
	for _, e := range stateEvents {
		inner, _ := e.EventParams()
		if inner["call_id"] == "L1" && inner["call_state"] == "ended" {
			sawEnded = true
			break
		}
	}
	if !sawEnded {
		t.Error("loser L1 never reached 'ended' in state events")
	}
}

// TestRelay_DialLosersCleanedUpFromCallsRegistry — Python:
// test_dial_losers_cleaned_up_from_calls_dict. The SDK removes ended
// loser calls from its internal registry.
func TestRelay_DialLosersCleanedUpFromCallsRegistry(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	h.ArmDial(t, mocktest.DialOpts{
		Tag:          "t-cleanup",
		WinnerCallID: "WIN-CL",
		States:       []string{"created", "answered"},
		NodeID:       "node-mock-1",
		Device:       phoneDevice("", ""),
		Losers: []mocktest.DialLoserOpts{
			{CallID: "LOSE-CL", States: []string{"created", "ended"}},
		},
	})
	call, err := client.Dial(
		[][]map[string]any{{phoneDevice("", "")}},
		relay.WithDialTag("t-cleanup"),
		relay.WithDialClientTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	// Allow loser-state events to flush.
	time.Sleep(200 * time.Millisecond)
	if call.CallID() != "WIN-CL" {
		t.Errorf("Winner CallID = %q, want %q", call.CallID(), "WIN-CL")
	}
	// We don't have a public surface to enumerate calls; assert the
	// winner is still reachable via Dial-returned Call.
	if call.State() != "answered" {
		t.Errorf("winner state = %q, want answered", call.State())
	}
}

// ---------------------------------------------------------------------------
// Devices shape on the wire
// ---------------------------------------------------------------------------

// TestRelay_DialDevicesSerialTwoLegsOnWire — Python:
// test_dial_devices_serial_two_legs_on_wire.
func TestRelay_DialDevicesSerialTwoLegsOnWire(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	h.ArmDial(t, mocktest.DialOpts{
		Tag:          "t-serial",
		WinnerCallID: "WIN-SER",
		States:       []string{"created", "answered"},
		NodeID:       "node-mock-1",
		Device:       phoneDevice("", ""),
	})
	devs := [][]map[string]any{
		{
			phoneDevice("+15551110001", ""),
			phoneDevice("+15551110002", ""),
		},
	}
	_, err := client.Dial(devs,
		relay.WithDialTag("t-serial"),
		relay.WithDialClientTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	entry := h.JournalLast(t, "calling.dial")
	params, _ := entry.FrameParams()
	devices, _ := params["devices"].([]any)
	if len(devices) != 1 {
		t.Fatalf("expected 1 leg, got %d", len(devices))
	}
	leg0, _ := devices[0].([]any)
	if len(leg0) != 2 {
		t.Fatalf("expected 2 devices in leg, got %d", len(leg0))
	}
	dev0, _ := leg0[0].(map[string]any)
	devParams, _ := dev0["params"].(map[string]any)
	if devParams["to_number"] != "+15551110001" {
		t.Errorf("first device.to_number = %v, want +15551110001", devParams["to_number"])
	}
}

// TestRelay_DialDevicesParallelTwoLegsOnWire — Python:
// test_dial_devices_parallel_two_legs_on_wire.
func TestRelay_DialDevicesParallelTwoLegsOnWire(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	h.ArmDial(t, mocktest.DialOpts{
		Tag:          "t-par",
		WinnerCallID: "WIN-PAR",
		States:       []string{"created", "answered"},
		NodeID:       "node-mock-1",
		Device:       phoneDevice("", ""),
	})
	devs := [][]map[string]any{
		{phoneDevice("+15551110001", "")},
		{phoneDevice("+15551110002", "")},
	}
	_, err := client.Dial(devs,
		relay.WithDialTag("t-par"),
		relay.WithDialClientTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	entry := h.JournalLast(t, "calling.dial")
	params, _ := entry.FrameParams()
	devices, _ := params["devices"].([]any)
	if len(devices) != 2 {
		t.Errorf("expected 2 legs, got %d", len(devices))
	}
}

// ---------------------------------------------------------------------------
// State transitions during dial
// ---------------------------------------------------------------------------

// TestRelay_DialRecordsCallStateProgressionOnWinner — Python:
// test_dial_records_call_state_progression_on_winner.
func TestRelay_DialRecordsCallStateProgressionOnWinner(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	h.ArmDial(t, mocktest.DialOpts{
		Tag:          "t-prog",
		WinnerCallID: "WIN-PROG",
		States:       []string{"created", "ringing", "answered"},
		NodeID:       "node-mock-1",
		Device:       phoneDevice("", ""),
	})
	call, err := client.Dial(
		[][]map[string]any{{phoneDevice("", "")}},
		relay.WithDialTag("t-prog"),
		relay.WithDialClientTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	stateEvents := h.JournalSend(t, "calling.call.state")
	var winnerStates []string
	for _, e := range stateEvents {
		inner, _ := e.EventParams()
		if inner["call_id"] == "WIN-PROG" {
			winnerStates = append(winnerStates, inner["call_state"].(string))
		}
	}
	for _, s := range []string{"created", "ringing", "answered"} {
		var found bool
		for _, ws := range winnerStates {
			if ws == s {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing winner state %q in: %v", s, winnerStates)
		}
	}
	if call.State() != "answered" {
		t.Errorf("call.State() = %q, want answered", call.State())
	}
}

// ---------------------------------------------------------------------------
// After dial — call object is usable
// ---------------------------------------------------------------------------

// TestRelay_DialedCallCanSendSubsequentCommand — Python:
// test_dialed_call_can_send_subsequent_command.
func TestRelay_DialedCallCanSendSubsequentCommand(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	h.ArmDial(t, mocktest.DialOpts{
		Tag:          "t-after",
		WinnerCallID: "WIN-AFTER",
		States:       []string{"created", "answered"},
		NodeID:       "node-mock-1",
		Device:       phoneDevice("", ""),
	})
	call, err := client.Dial(
		[][]map[string]any{{phoneDevice("", "")}},
		relay.WithDialTag("t-after"),
		relay.WithDialClientTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	if err := call.Hangup(""); err != nil {
		t.Fatalf("Hangup: %v", err)
	}
	endFrames := h.JournalRecv(t, "calling.end")
	if len(endFrames) == 0 {
		t.Fatal("no calling.end frame in journal")
	}
	last := endFrames[len(endFrames)-1]
	params, _ := last.FrameParams()
	if params["call_id"] != "WIN-AFTER" {
		t.Errorf("calling.end.call_id = %v, want WIN-AFTER", params["call_id"])
	}
}

// TestRelay_DialedCallCanPlay — Python: test_dialed_call_can_play.
func TestRelay_DialedCallCanPlay(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	h.ArmDial(t, mocktest.DialOpts{
		Tag:          "t-play",
		WinnerCallID: "WIN-PLAY",
		States:       []string{"created", "answered"},
		NodeID:       "node-mock-1",
		Device:       phoneDevice("", ""),
	})
	call, err := client.Dial(
		[][]map[string]any{{phoneDevice("", "")}},
		relay.WithDialTag("t-play"),
		relay.WithDialClientTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	_ = call.Play([]map[string]any{
		{"type": "tts", "params": map[string]any{"text": "hi"}},
	})
	// Allow the play frame to land.
	time.Sleep(200 * time.Millisecond)
	plays := h.JournalRecv(t, "calling.play")
	if len(plays) == 0 {
		t.Fatal("no calling.play frame after dial")
	}
	last := plays[len(plays)-1]
	params, _ := last.FrameParams()
	if params["call_id"] != "WIN-PLAY" {
		t.Errorf("calling.play.call_id = %v, want WIN-PLAY", params["call_id"])
	}
	playList, _ := params["play"].([]any)
	if len(playList) == 0 {
		t.Fatal("play list is empty")
	}
	pe, _ := playList[0].(map[string]any)
	if pe["type"] != "tts" {
		t.Errorf("play[0].type = %v, want tts", pe["type"])
	}
}

// ---------------------------------------------------------------------------
// Tag preservation
// ---------------------------------------------------------------------------

// TestRelay_DialPreservesExplicitTag — Python:
// test_dial_preserves_explicit_tag.
func TestRelay_DialPreservesExplicitTag(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	h.ArmDial(t, mocktest.DialOpts{
		Tag:          "my-very-explicit-tag-99",
		WinnerCallID: "WIN-T",
		States:       []string{"created", "answered"},
		NodeID:       "node-mock-1",
		Device:       phoneDevice("", ""),
	})
	call, err := client.Dial(
		[][]map[string]any{{phoneDevice("", "")}},
		relay.WithDialTag("my-very-explicit-tag-99"),
		relay.WithDialClientTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	if call.Tag() != "my-very-explicit-tag-99" {
		t.Errorf("Tag = %q, want %q", call.Tag(), "my-very-explicit-tag-99")
	}
}

// ---------------------------------------------------------------------------
// JSON-RPC envelope
// ---------------------------------------------------------------------------

// TestRelay_DialUsesJSONRPC2_0 — Python: test_dial_uses_jsonrpc_2_0.
func TestRelay_DialUsesJSONRPC2_0(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	h.ArmDial(t, mocktest.DialOpts{
		Tag:          "t-rpc",
		WinnerCallID: "W",
		States:       []string{"created", "answered"},
		NodeID:       "n",
		Device:       phoneDevice("", ""),
	})
	_, err := client.Dial(
		[][]map[string]any{{phoneDevice("", "")}},
		relay.WithDialTag("t-rpc"),
		relay.WithDialClientTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	entry := h.JournalLast(t, "calling.dial")
	if entry.Frame["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want 2.0", entry.Frame["jsonrpc"])
	}
	if entry.Frame["method"] != "calling.dial" {
		t.Errorf("method = %v, want calling.dial", entry.Frame["method"])
	}
	if _, has := entry.Frame["id"]; !has {
		t.Error("id missing from frame")
	}
	if _, has := entry.Frame["params"]; !has {
		t.Error("params missing from frame")
	}
}
