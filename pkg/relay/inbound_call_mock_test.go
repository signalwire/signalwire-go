// Copyright (c) 2025 SignalWire
//
// Mock-relay-backed tests for inbound calls (server-initiated). Mirrors
// signalwire-python's tests/unit/relay/test_inbound_call_mock.py.

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

// statePushFrame builds a signalwire.event(calling.call.state) frame
// matching Python's _state_push_frame helper.
func statePushFrame(callID, callState, tag, direction string) map[string]any {
	if direction == "" {
		direction = "inbound"
	}
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      "evt-" + callID + "-" + callState,
		"method":  "signalwire.event",
		"params": map[string]any{
			"event_type": "calling.call.state",
			"params": map[string]any{
				"call_id":    callID,
				"node_id":    "mock-relay-node-1",
				"tag":        tag,
				"call_state": callState,
				"direction":  direction,
				"device": map[string]any{
					"type": "phone",
					"params": map[string]any{
						"from_number": "+15551110000",
						"to_number":   "+15552220000",
					},
				},
			},
		},
	}
}

// waitFor polls predicate every 10ms up to timeout. Returns true on
// success, false on timeout.
func waitFor(timeout time.Duration, predicate func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if predicate() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return predicate()
}

// ---------------------------------------------------------------------------
// Basic inbound-call handler dispatch
// ---------------------------------------------------------------------------

// TestRelay_OnCallHandlerFiresWithCallObject — Python:
// test_on_call_handler_fires_with_call_object.
func TestRelay_OnCallHandlerFiresWithCallObject(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}

	var mu sync.Mutex
	var seen []*relay.Call
	done := make(chan struct{}, 1)

	client.OnCall(func(call *relay.Call) {
		mu.Lock()
		seen = append(seen, call)
		mu.Unlock()
		select {
		case done <- struct{}{}:
		default:
		}
	})

	h.InboundCall(t, mocktest.InboundCallOpts{
		CallID:     "c-handler",
		FromNumber: "+15551110000",
		ToNumber:   "+15552220000",
		AutoStates: []string{"created"},
	})

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("on_call handler did not fire within 5s")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 1 {
		t.Fatalf("expected 1 call, got %d", len(seen))
	}
	if seen[0].CallID() != "c-handler" {
		t.Errorf("CallID = %q, want %q", seen[0].CallID(), "c-handler")
	}
}

// TestRelay_InboundCallObjectHasCorrectCallIDAndDirection — Python:
// test_inbound_call_object_has_correct_call_id_and_direction.
func TestRelay_InboundCallObjectHasCorrectCallIDAndDirection(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	done := make(chan *relay.Call, 1)
	client.OnCall(func(call *relay.Call) {
		done <- call
	})
	h.InboundCall(t, mocktest.InboundCallOpts{
		CallID:     "c-dir",
		AutoStates: []string{"created"},
	})
	select {
	case call := <-done:
		if call.CallID() != "c-dir" {
			t.Errorf("CallID = %q, want %q", call.CallID(), "c-dir")
		}
		if call.Direction() != "inbound" {
			t.Errorf("Direction = %q, want %q", call.Direction(), "inbound")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("on_call handler did not fire within 5s")
	}
}

// TestRelay_InboundCallCarriesFromToInDevice — Python:
// test_inbound_call_carries_from_to_in_device.
func TestRelay_InboundCallCarriesFromToInDevice(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	done := make(chan *relay.Call, 1)
	client.OnCall(func(call *relay.Call) {
		done <- call
	})
	h.InboundCall(t, mocktest.InboundCallOpts{
		CallID:     "c-from-to",
		FromNumber: "+15551112233",
		ToNumber:   "+15554445566",
		AutoStates: []string{"created"},
	})
	select {
	case call := <-done:
		dev := call.Device()
		if dev == nil {
			t.Fatal("Device is nil")
		}
		params, _ := dev["params"].(map[string]any)
		if params["from_number"] != "+15551112233" {
			t.Errorf("from_number = %v, want +15551112233", params["from_number"])
		}
		if params["to_number"] != "+15554445566" {
			t.Errorf("to_number = %v, want +15554445566", params["to_number"])
		}
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not fire")
	}
}

// TestRelay_InboundCallInitialStateIsCreated — Python:
// test_inbound_call_initial_state_is_created.
func TestRelay_InboundCallInitialStateIsCreated(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	done := make(chan *relay.Call, 1)
	client.OnCall(func(call *relay.Call) {
		done <- call
	})
	h.InboundCall(t, mocktest.InboundCallOpts{
		CallID:     "c-state",
		AutoStates: []string{"created"},
	})
	select {
	case call := <-done:
		if call.State() != "created" {
			t.Errorf("State = %q, want created", call.State())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not fire")
	}
}

// ---------------------------------------------------------------------------
// Handler answers — calling.answer journaled
// ---------------------------------------------------------------------------

// TestRelay_AnswerInHandlerJournalsCallingAnswer — Python:
// test_answer_in_handler_journals_calling_answer.
func TestRelay_AnswerInHandlerJournalsCallingAnswer(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	done := make(chan struct{}, 1)
	client.OnCall(func(call *relay.Call) {
		_ = call.Answer()
		done <- struct{}{}
	})
	h.InboundCall(t, mocktest.InboundCallOpts{
		CallID:     "c-ans",
		AutoStates: []string{"created"},
	})
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not fire")
	}
	// Allow the answer round-trip to land.
	time.Sleep(100 * time.Millisecond)

	answers := h.JournalRecv(t, "calling.answer")
	if len(answers) == 0 {
		t.Fatal("no calling.answer frame in journal")
	}
	last := answers[len(answers)-1]
	params, _ := last.FrameParams()
	if params["call_id"] != "c-ans" {
		t.Errorf("calling.answer.call_id = %v, want c-ans", params["call_id"])
	}
}

// TestRelay_AnswerThenStateEventAdvancesCallState — Python:
// test_answer_then_state_event_advances_call_state.
func TestRelay_AnswerThenStateEventAdvancesCallState(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	captured := make(chan *relay.Call, 1)
	client.OnCall(func(call *relay.Call) {
		_ = call.Answer()
		captured <- call
	})
	h.InboundCall(t, mocktest.InboundCallOpts{
		CallID:     "c-ans-state",
		AutoStates: []string{"created"},
	})
	var call *relay.Call
	select {
	case call = <-captured:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not fire")
	}

	h.Push(t, statePushFrame("c-ans-state", "answered", "", ""), "")
	if !waitFor(2*time.Second, func() bool {
		return call.State() == "answered"
	}) {
		t.Errorf("State = %q, want answered", call.State())
	}
}

// ---------------------------------------------------------------------------
// Handler hangs up / passes
// ---------------------------------------------------------------------------

// TestRelay_HangupInHandlerJournalsCallingEnd — Python:
// test_hangup_in_handler_journals_calling_end.
func TestRelay_HangupInHandlerJournalsCallingEnd(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	done := make(chan struct{}, 1)
	client.OnCall(func(call *relay.Call) {
		_ = call.Hangup("busy")
		done <- struct{}{}
	})
	h.InboundCall(t, mocktest.InboundCallOpts{
		CallID:     "c-hangup",
		AutoStates: []string{"created"},
	})
	<-done
	time.Sleep(100 * time.Millisecond)

	ends := h.JournalRecv(t, "calling.end")
	if len(ends) == 0 {
		t.Fatal("no calling.end frame in journal")
	}
	params, _ := ends[len(ends)-1].FrameParams()
	if params["call_id"] != "c-hangup" {
		t.Errorf("call_id = %v, want c-hangup", params["call_id"])
	}
	if params["reason"] != "busy" {
		t.Errorf("reason = %v, want busy", params["reason"])
	}
}

// TestRelay_PassInHandlerJournalsCallingPass — Python:
// test_pass_in_handler_journals_calling_pass.
func TestRelay_PassInHandlerJournalsCallingPass(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	done := make(chan struct{}, 1)
	client.OnCall(func(call *relay.Call) {
		_ = call.Pass()
		done <- struct{}{}
	})
	h.InboundCall(t, mocktest.InboundCallOpts{
		CallID:     "c-pass",
		AutoStates: []string{"created"},
	})
	<-done
	time.Sleep(100 * time.Millisecond)

	passes := h.JournalRecv(t, "calling.pass")
	if len(passes) == 0 {
		t.Fatal("no calling.pass frame in journal")
	}
	params, _ := passes[len(passes)-1].FrameParams()
	if params["call_id"] != "c-pass" {
		t.Errorf("call_id = %v, want c-pass", params["call_id"])
	}
}

// ---------------------------------------------------------------------------
// Multiple inbound calls — independent state
// ---------------------------------------------------------------------------

// TestRelay_MultipleInboundCallsInSequenceEachUniqueObject — Python:
// test_multiple_inbound_calls_in_sequence_each_unique_object.
func TestRelay_MultipleInboundCallsInSequenceEachUniqueObject(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	var mu sync.Mutex
	var seen []*relay.Call
	allDone := make(chan struct{}, 1)
	client.OnCall(func(call *relay.Call) {
		mu.Lock()
		seen = append(seen, call)
		complete := len(seen) == 2
		mu.Unlock()
		if complete {
			select {
			case allDone <- struct{}{}:
			default:
			}
		}
	})
	h.InboundCall(t, mocktest.InboundCallOpts{
		CallID:     "c-seq-1",
		AutoStates: []string{"created"},
	})
	time.Sleep(100 * time.Millisecond)
	h.InboundCall(t, mocktest.InboundCallOpts{
		CallID:     "c-seq-2",
		AutoStates: []string{"created"},
	})
	select {
	case <-allDone:
	case <-time.After(5 * time.Second):
		t.Fatal("not both handlers fired in time")
	}
	mu.Lock()
	defer mu.Unlock()
	if seen[0].CallID() != "c-seq-1" {
		t.Errorf("first CallID = %q, want c-seq-1", seen[0].CallID())
	}
	if seen[1].CallID() != "c-seq-2" {
		t.Errorf("second CallID = %q, want c-seq-2", seen[1].CallID())
	}
	if seen[0] == seen[1] {
		t.Error("expected distinct Call objects")
	}
}

// TestRelay_MultipleInboundCallsNoStateBleed — Python:
// test_multiple_inbound_calls_no_state_bleed.
func TestRelay_MultipleInboundCallsNoStateBleed(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	var mu sync.Mutex
	calls := make(map[string]*relay.Call)
	bothReceived := make(chan struct{}, 1)
	client.OnCall(func(call *relay.Call) {
		mu.Lock()
		calls[call.CallID()] = call
		complete := len(calls) == 2
		mu.Unlock()
		_ = call.Answer()
		if complete {
			select {
			case bothReceived <- struct{}{}:
			default:
			}
		}
	})
	h.InboundCall(t, mocktest.InboundCallOpts{
		CallID:     "cb-1",
		AutoStates: []string{"created"},
	})
	time.Sleep(50 * time.Millisecond)
	h.InboundCall(t, mocktest.InboundCallOpts{
		CallID:     "cb-2",
		AutoStates: []string{"created"},
	})
	select {
	case <-bothReceived:
	case <-time.After(5 * time.Second):
		t.Fatal("did not receive both inbound calls")
	}
	// Push answered to only cb-1.
	h.Push(t, statePushFrame("cb-1", "answered", "", ""), "")
	if !waitFor(2*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		c1, ok := calls["cb-1"]
		return ok && c1.State() == "answered"
	}) {
		t.Error("cb-1 never reached answered")
	}
	mu.Lock()
	defer mu.Unlock()
	if calls["cb-2"].State() == "answered" {
		t.Error("cb-2 unexpectedly reached answered (state bled)")
	}
}

// ---------------------------------------------------------------------------
// Scripted state sequences
// ---------------------------------------------------------------------------

// TestRelay_ScriptedStateSequenceAdvancesCall — Python:
// test_scripted_state_sequence_advances_call.
func TestRelay_ScriptedStateSequenceAdvancesCall(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	captured := make(chan *relay.Call, 1)
	client.OnCall(func(call *relay.Call) {
		_ = call.Answer()
		captured <- call
	})
	h.InboundCall(t, mocktest.InboundCallOpts{
		CallID:     "c-scripted",
		AutoStates: []string{"created"},
	})
	var call *relay.Call
	select {
	case call = <-captured:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not fire")
	}

	h.Push(t, statePushFrame("c-scripted", "answered", "", ""), "")
	h.Push(t, statePushFrame("c-scripted", "ended", "", ""), "")
	if !waitFor(2*time.Second, func() bool {
		return call.State() == "ended"
	}) {
		t.Errorf("State = %q, want ended", call.State())
	}
}

// ---------------------------------------------------------------------------
// Handler patterns: async/sync, raise
// ---------------------------------------------------------------------------

// TestRelay_AsyncHandlerCompletesNormally — Python:
// test_async_handler_completes_normally. (Go handlers are always
// goroutines from the SDK side; this test verifies a handler that does
// some asynchronous work observes the right call_id.)
func TestRelay_AsyncHandlerCompletesNormally(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	fired := make(chan string, 1)
	client.OnCall(func(call *relay.Call) {
		time.Sleep(10 * time.Millisecond)
		fired <- call.CallID()
	})
	h.InboundCall(t, mocktest.InboundCallOpts{
		CallID:     "c-async",
		AutoStates: []string{"created"},
	})
	select {
	case id := <-fired:
		if id != "c-async" {
			t.Errorf("CallID = %q, want c-async", id)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not fire")
	}
}

// TestRelay_HandlerExceptionDoesNotCrashClient — Python:
// test_handler_exception_does_not_crash_client. A panicking handler
// must not break the client.
func TestRelay_HandlerExceptionDoesNotCrashClient(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	fired := make(chan struct{}, 1)
	client.OnCall(func(call *relay.Call) {
		select {
		case fired <- struct{}{}:
		default:
		}
		panic("intentional from handler")
	})
	h.InboundCall(t, mocktest.InboundCallOpts{
		CallID:     "c-raise",
		AutoStates: []string{"created"},
	})
	select {
	case <-fired:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not fire")
	}
	// Allow goroutine cleanup.
	time.Sleep(100 * time.Millisecond)
	// Client is still alive: a follow-up Execute must succeed.
	if _, err := client.Execute("signalwire.ping", map[string]any{}); err != nil {
		t.Fatalf("client died after handler panic: %v", err)
	}
}

// ---------------------------------------------------------------------------
// scenario_play — full inbound flow
// ---------------------------------------------------------------------------

// TestRelay_ScenarioPlayFullInboundFlow — Python:
// test_scenario_play_full_inbound_flow.
func TestRelay_ScenarioPlayFullInboundFlow(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	captured := make(chan *relay.Call, 1)
	var fired atomic.Bool
	client.OnCall(func(call *relay.Call) {
		fired.Store(true)
		_ = call.Answer()
		captured <- call
	})

	timeline := []map[string]any{
		{
			"push": map[string]any{
				"frame": map[string]any{
					"jsonrpc": "2.0",
					"id":      "scen-receive",
					"method":  "signalwire.event",
					"params": map[string]any{
						"event_type": "calling.call.receive",
						"params": map[string]any{
							"call_id":    "c-scen",
							"node_id":    "mock-relay-node-1",
							"tag":        "",
							"call_state": "created",
							"direction":  "inbound",
							"device": map[string]any{
								"type": "phone",
								"params": map[string]any{
									"from_number": "+15551110000",
									"to_number":   "+15552220000",
								},
							},
							"context": "default",
						},
					},
				},
			},
		},
		{"expect_recv": map[string]any{"method": "calling.answer", "timeout_ms": 5000}},
		{"push": map[string]any{"frame": statePushFrame("c-scen", "answered", "", "")}},
		{"sleep_ms": 50},
		{"push": map[string]any{"frame": statePushFrame("c-scen", "ended", "", "")}},
	}

	resultCh := make(chan map[string]any, 1)
	go func() {
		resultCh <- h.ScenarioPlay(t, timeline)
	}()

	var call *relay.Call
	select {
	case call = <-captured:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not fire in scenario_play timeline")
	}
	result := <-resultCh
	if result["status"] != "completed" {
		t.Fatalf("scenario didn't complete: %v", result)
	}
	if !fired.Load() {
		t.Fatal("on_call handler did not fire")
	}
	if !waitFor(2*time.Second, func() bool {
		return call.State() == "ended"
	}) {
		t.Errorf("call.State() = %q, want ended", call.State())
	}
}

// ---------------------------------------------------------------------------
// Wire shape — calling.call.receive
// ---------------------------------------------------------------------------

// TestRelay_InboundCallJournalSendRecordsCallingCallReceive — Python:
// test_inbound_call_journal_send_records_calling_call_receive.
func TestRelay_InboundCallJournalSendRecordsCallingCallReceive(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	done := make(chan struct{}, 1)
	client.OnCall(func(call *relay.Call) {
		done <- struct{}{}
	})
	h.InboundCall(t, mocktest.InboundCallOpts{
		CallID:     "c-wire",
		AutoStates: []string{"created"},
	})
	<-done

	sends := h.JournalSend(t, "calling.call.receive")
	if len(sends) == 0 {
		t.Fatal("no calling.call.receive frame in journal")
	}
	inner, _ := sends[len(sends)-1].EventParams()
	if inner["call_id"] != "c-wire" {
		t.Errorf("call_id = %v, want c-wire", inner["call_id"])
	}
	if inner["direction"] != "inbound" {
		t.Errorf("direction = %v, want inbound", inner["direction"])
	}
}

// ---------------------------------------------------------------------------
// Inbound without a registered handler — does not crash
// ---------------------------------------------------------------------------

// TestRelay_InboundWithoutHandlerDoesNotCrash — Python:
// test_inbound_without_handler_does_not_crash.
func TestRelay_InboundWithoutHandlerDoesNotCrash(t *testing.T) {
	_, h := mocktest.New(t)
	if h == nil {
		return
	}
	// Build a fresh client with no on_call registered.
	client := mocktest.NewClientOnly(t, h,
		relay.WithProject("p"),
		relay.WithToken("t"),
		relay.WithContexts("default"),
	)
	h.InboundCall(t, mocktest.InboundCallOpts{
		CallID:     "c-nohandler",
		AutoStates: []string{"created"},
	})
	// Give the recv loop time to process.
	time.Sleep(200 * time.Millisecond)
	// Client is still alive — a ping should succeed.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := executeWithCtx(ctx, client, "signalwire.ping", map[string]any{}); err != nil {
		t.Fatalf("client died after inbound-without-handler: %v", err)
	}
}

// executeWithCtx runs client.Execute under a context for tests that need
// a bounded wait — the SDK itself uses a 30s timeout internally, but
// inside tests we want explicit failure timing.
func executeWithCtx(ctx context.Context, c *relay.Client, method string, params map[string]any) (any, error) {
	type out struct {
		v   any
		err error
	}
	ch := make(chan out, 1)
	go func() {
		v, err := c.Execute(method, params)
		ch <- out{v, err}
	}()
	select {
	case r := <-ch:
		return r.v, r.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
