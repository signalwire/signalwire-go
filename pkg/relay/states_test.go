// Copyright (c) 2025 SignalWire
//
// Tests for the typed RELAY lifecycle kinds CallState / DialState /
// MessageState (Tier-3 idiom addition). Pure-function tests cover String /
// IsKnown / IsTerminal (real behavior — no mocks needed for value logic);
// mock-backed tests drive a REAL calling.call.state and messaging.state event
// through the shared mock_relay and assert the typed accessor returns the right
// kind AND agrees with the existing bare-string accessor.

package relay_test

import (
	"testing"
	"time"

	"github.com/signalwire/signalwire-go/v3/pkg/relay"
	"github.com/signalwire/signalwire-go/v3/pkg/relay/internal/mocktest"
)

// ---------------------------------------------------------------------------
// CallState — pure value logic
// ---------------------------------------------------------------------------

func TestCallState_IsTerminal(t *testing.T) {
	terminal := []relay.CallState{relay.CallEnded}
	nonTerminal := []relay.CallState{
		relay.CallCreated, relay.CallRinging, relay.CallAnswered, relay.CallEnding,
	}
	for _, s := range terminal {
		if !s.IsTerminal() {
			t.Errorf("CallState(%q).IsTerminal() = false, want true", s)
		}
	}
	for _, s := range nonTerminal {
		if s.IsTerminal() {
			t.Errorf("CallState(%q).IsTerminal() = true, want false", s)
		}
	}
	// An unknown/future server state is not terminal and not known.
	if relay.CallState("teleporting").IsTerminal() {
		t.Error("unknown CallState should not be terminal")
	}
}

func TestCallState_IsKnownAndString(t *testing.T) {
	known := []relay.CallState{
		relay.CallCreated, relay.CallRinging, relay.CallAnswered,
		relay.CallEnding, relay.CallEnded,
	}
	for _, s := range known {
		if !s.IsKnown() {
			t.Errorf("CallState(%q).IsKnown() = false, want true", s)
		}
	}
	if relay.CallState("teleporting").IsKnown() {
		t.Error("CallState(teleporting).IsKnown() = true, want false")
	}
	// String() and the underlying constant must equal the wire token.
	if relay.CallAnswered.String() != "answered" {
		t.Errorf("CallAnswered.String() = %q, want answered", relay.CallAnswered.String())
	}
	if string(relay.CallEnded) != relay.CallStateEnded {
		t.Errorf("CallEnded underlying = %q, want %q", string(relay.CallEnded), relay.CallStateEnded)
	}
}

// ---------------------------------------------------------------------------
// DialState — pure value logic (distinct vocabulary from CallState)
// ---------------------------------------------------------------------------

func TestDialState_IsTerminal(t *testing.T) {
	// answered + failed are terminal dial outcomes; dialing is progress.
	if !relay.DialAnswered.IsTerminal() {
		t.Error("DialAnswered.IsTerminal() = false, want true")
	}
	if !relay.DialFailed.IsTerminal() {
		t.Error("DialFailed.IsTerminal() = false, want true")
	}
	if relay.DialDialing.IsTerminal() {
		t.Error("DialDialing.IsTerminal() = true, want false (dialing is progress)")
	}
}

func TestDialState_IsKnownAndDistinctVocabulary(t *testing.T) {
	for _, s := range []relay.DialState{relay.DialDialing, relay.DialAnswered, relay.DialFailed} {
		if !s.IsKnown() {
			t.Errorf("DialState(%q).IsKnown() = false, want true", s)
		}
	}
	// "ended" belongs to CallState, NOT DialState — must be unknown here.
	if relay.DialState("ended").IsKnown() {
		t.Error("DialState(ended).IsKnown() = true; ended is a CallState, not a DialState")
	}
	// "created"/"ringing" are CallState-only — unknown as a DialState.
	if relay.DialState("created").IsKnown() {
		t.Error("DialState(created).IsKnown() = true; created is a CallState, not a DialState")
	}
}

// ---------------------------------------------------------------------------
// MessageState — pure value logic (distinct vocabulary)
// ---------------------------------------------------------------------------

func TestMessageState_IsTerminal(t *testing.T) {
	// Mirrors Python MESSAGE_TERMINAL_STATES: delivered, undelivered, failed.
	terminal := []relay.MessageState{
		relay.MsgDelivered, relay.MsgUndelivered, relay.MsgFailed,
	}
	nonTerminal := []relay.MessageState{
		relay.MsgQueued, relay.MsgInitiated, relay.MsgSent, relay.MsgReceived,
	}
	for _, s := range terminal {
		if !s.IsTerminal() {
			t.Errorf("MessageState(%q).IsTerminal() = false, want true", s)
		}
	}
	for _, s := range nonTerminal {
		if s.IsTerminal() {
			t.Errorf("MessageState(%q).IsTerminal() = true, want false", s)
		}
	}
}

func TestMessageState_IsKnownAndDistinctVocabulary(t *testing.T) {
	known := []relay.MessageState{
		relay.MsgQueued, relay.MsgInitiated, relay.MsgSent, relay.MsgDelivered,
		relay.MsgUndelivered, relay.MsgFailed, relay.MsgReceived,
	}
	for _, s := range known {
		if !s.IsKnown() {
			t.Errorf("MessageState(%q).IsKnown() = false, want true", s)
		}
	}
	// "answered"/"ringing" are call/dial vocab — unknown as a MessageState.
	if relay.MessageState("answered").IsKnown() {
		t.Error("MessageState(answered).IsKnown() = true; answered is not a message state")
	}
}

// ---------------------------------------------------------------------------
// Typed accessors driven by REAL dispatched events (mock_relay)
// ---------------------------------------------------------------------------

// TestCallState_TypedAccessorFromRealStateEvent drives a real
// calling.call.state event through the mock and asserts Call.CallState()
// returns the right typed kind AND agrees byte-for-byte with Call.State().
func TestCallState_TypedAccessorFromRealStateEvent(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "ec-cs-typed")

	// Push a real state event transitioning the call to "ending".
	h.Push(t, bareEventFrame("calling.call.state", map[string]any{
		"call_id":    "ec-cs-typed",
		"call_state": "ending",
		"direction":  "inbound",
	}), "")

	if !waitFor(2*time.Second, func() bool {
		return call.CallState() == relay.CallEnding
	}) {
		t.Fatalf("CallState() = %q, want ending", call.CallState())
	}

	// The typed accessor must agree with the bare-string accessor.
	if string(call.CallState()) != call.State() {
		t.Errorf("CallState() underlying %q != State() %q", string(call.CallState()), call.State())
	}
	// "ending" is a known, non-terminal call state.
	if !call.CallState().IsKnown() {
		t.Error("CallState().IsKnown() = false for ending")
	}
	if call.CallState().IsTerminal() {
		t.Error("CallState().IsTerminal() = true for ending, want false")
	}

	// Now drive it to terminal "ended" and confirm IsTerminal() flips.
	h.Push(t, bareEventFrame("calling.call.state", map[string]any{
		"call_id":    "ec-cs-typed",
		"call_state": "ended",
	}), "")
	if !waitFor(2*time.Second, func() bool {
		return call.CallState().IsTerminal()
	}) {
		t.Fatalf("CallState() = %q, want terminal (ended)", call.CallState())
	}
	if call.CallState() != relay.CallEnded {
		t.Errorf("CallState() = %q, want ended", call.CallState())
	}
}

// TestMessageState_TypedAccessorFromRealStateEvent drives a real
// messaging.state event and asserts Message.MessageState() returns the right
// typed kind, is terminal, and agrees with Message.State().
func TestMessageState_TypedAccessorFromRealStateEvent(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	msg, err := client.SendMessage("+15551112222", "+15553334444", "hi")
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	h.Push(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      "evt-msg-state-typed",
		"method":  "signalwire.event",
		"params": map[string]any{
			"event_type": "messaging.state",
			"params": map[string]any{
				"message_id":    msg.MessageID(),
				"message_state": "delivered",
				"from_number":   "+15553334444",
				"to_number":     "+15551112222",
				"body":          "hi",
			},
		},
	}, "")

	if !waitFor(5*time.Second, func() bool {
		return msg.MessageState() == relay.MsgDelivered
	}) {
		t.Fatalf("MessageState() = %q, want delivered", msg.MessageState())
	}
	// Typed accessor agrees with the bare-string accessor.
	if string(msg.MessageState()) != msg.State() {
		t.Errorf("MessageState() underlying %q != State() %q", string(msg.MessageState()), msg.State())
	}
	// "delivered" is a known terminal message state.
	if !msg.MessageState().IsKnown() {
		t.Error("MessageState().IsKnown() = false for delivered")
	}
	if !msg.MessageState().IsTerminal() {
		t.Error("MessageState().IsTerminal() = false for delivered, want true")
	}
}

// TestDialState_TypedAccessorFromRealDialEvent drives a real calling.call.dial
// event (via the armed dial flow) and asserts the DialEvent's typed accessor
// returns the right kind and agrees with the string field.
func TestDialState_TypedAccessorFromRealDialEvent(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}

	// Capture the dial event as the SDK dispatches it, via a generic hook.
	got := make(chan *relay.DialEvent, 4)
	client.OnEvent(func(eventType string, params map[string]any) {
		if eventType == relay.EventCallingCallDial {
			got <- relay.NewDialEvent(params)
		}
	})

	h.ArmDial(t, mocktest.DialOpts{
		Tag:          "t-dialstate",
		WinnerCallID: "winner-dialstate",
		States:       []string{"created", "answered"},
		NodeID:       "node-mock-1",
		Device:       phoneDevice("", ""),
	})
	_, err := client.Dial(
		[][]map[string]any{{phoneDevice("", "")}},
		relay.WithDialTag("t-dialstate"),
		relay.WithDialClientTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}

	// Wait for an "answered" dial event and assert the typed accessor.
	deadline := time.After(5 * time.Second)
	for {
		select {
		case ev := <-got:
			if ev.DialState != "answered" {
				continue // skip "dialing"/progress events
			}
			if ev.DialStateTyped() != relay.DialAnswered {
				t.Errorf("DialStateTyped() = %q, want answered", ev.DialStateTyped())
			}
			// Typed accessor agrees with the string field.
			if string(ev.DialStateTyped()) != ev.DialState {
				t.Errorf("DialStateTyped() %q != DialState %q", ev.DialStateTyped(), ev.DialState)
			}
			if !ev.DialStateTyped().IsTerminal() {
				t.Error("answered DialState should be terminal")
			}
			return
		case <-deadline:
			t.Fatal("no answered calling.call.dial event observed")
		}
	}
}
