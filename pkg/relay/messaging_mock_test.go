// Copyright (c) 2025 SignalWire
//
// Mock-relay-backed tests for messaging (send_message + inbound).
// Mirrors signalwire-python's tests/unit/relay/test_messaging_mock.py.

package relay_test

import (
	"context"
	"testing"
	"time"

	"github.com/signalwire/signalwire-go/pkg/relay"
	"github.com/signalwire/signalwire-go/pkg/relay/internal/mocktest"
)

// ---------------------------------------------------------------------------
// send_message — outbound
// ---------------------------------------------------------------------------

// TestRelay_SendMessageJournalsMessagingSend — Python:
// test_send_message_journals_messaging_send.
func TestRelay_SendMessageJournalsMessagingSend(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	msg, err := client.SendMessage(
		"+15551112222",
		"+15553334444",
		"hello",
		relay.WithMessageTags([]string{"t1", "t2"}),
	)
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if msg == nil {
		t.Fatal("SendMessage returned nil")
	}
	if msg.MessageID() == "" {
		t.Error("MessageID is empty")
	}
	if msg.Body() != "hello" {
		t.Errorf("Body = %q, want hello", msg.Body())
	}
	entry := h.JournalLast(t, "messaging.send")
	params, _ := entry.FrameParams()
	if params["to_number"] != "+15551112222" {
		t.Errorf("to_number = %v", params["to_number"])
	}
	if params["from_number"] != "+15553334444" {
		t.Errorf("from_number = %v", params["from_number"])
	}
	if params["body"] != "hello" {
		t.Errorf("body = %v", params["body"])
	}
	tags, _ := params["tags"].([]any)
	if len(tags) != 2 || tags[0] != "t1" || tags[1] != "t2" {
		t.Errorf("tags = %v, want [t1 t2]", params["tags"])
	}
}

// TestRelay_SendMessageWithMediaOnly — Python:
// test_send_message_with_media_only.
func TestRelay_SendMessageWithMediaOnly(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	// Use empty body; pass media via option.
	msg, err := client.SendMessage(
		"+15551112222",
		"+15553334444",
		"",
		relay.WithMessageMedia([]string{"https://media.example/cat.jpg"}),
	)
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if msg == nil {
		t.Fatal("SendMessage returned nil")
	}
	entry := h.JournalLast(t, "messaging.send")
	params, _ := entry.FrameParams()
	media, _ := params["media"].([]any)
	if len(media) != 1 || media[0] != "https://media.example/cat.jpg" {
		t.Errorf("media = %v, want [https://media.example/cat.jpg]", params["media"])
	}
	// Empty body should not be on wire (mirrors Python's "if body").
	if v, has := params["body"]; has && v != nil && v != "" {
		t.Errorf("body should be absent or empty for media-only, got %v", v)
	}
}

// TestRelay_SendMessageIncludesContext — Python:
// test_send_message_includes_context.
func TestRelay_SendMessageIncludesContext(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	_, err := client.SendMessage(
		"+15551112222",
		"+15553334444",
		"hi",
		relay.WithMessageContext("custom-ctx"),
	)
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	entry := h.JournalLast(t, "messaging.send")
	params, _ := entry.FrameParams()
	if params["context"] != "custom-ctx" {
		t.Errorf("context = %v, want custom-ctx", params["context"])
	}
}

// TestRelay_SendMessageReturnsInitialStateQueued — Python:
// test_send_message_returns_initial_state_queued.
func TestRelay_SendMessageReturnsInitialStateQueued(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	_ = h
	msg, err := client.SendMessage(
		"+15551112222",
		"+15553334444",
		"hi",
	)
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if msg.State() != relay.MessageStateQueued {
		t.Errorf("State = %q, want queued", msg.State())
	}
	if msg.IsDone() {
		t.Error("IsDone() should be false right after send")
	}
}

// TestRelay_SendMessageResolvesOnDelivered — Python:
// test_send_message_resolves_on_delivered.
func TestRelay_SendMessageResolvesOnDelivered(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	msg, err := client.SendMessage(
		"+15551112222",
		"+15553334444",
		"hi",
	)
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	h.Push(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      "evt-msg-delivered",
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	event, err := msg.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if msg.State() != "delivered" {
		t.Errorf("State = %q, want delivered", msg.State())
	}
	if !msg.IsDone() {
		t.Error("IsDone() should be true after delivered")
	}
	if event.GetString("message_state") != "delivered" {
		t.Errorf("event.message_state = %q, want delivered", event.GetString("message_state"))
	}
}

// TestRelay_SendMessageResolvesOnUndelivered — Python:
// test_send_message_resolves_on_undelivered.
func TestRelay_SendMessageResolvesOnUndelivered(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	msg, err := client.SendMessage(
		"+15551112222",
		"+15553334444",
		"hi",
	)
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	h.Push(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      "evt-msg-undeliv",
		"method":  "signalwire.event",
		"params": map[string]any{
			"event_type": "messaging.state",
			"params": map[string]any{
				"message_id":    msg.MessageID(),
				"message_state": "undelivered",
				"reason":        "carrier_blocked",
			},
		},
	}, "")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := msg.Wait(ctx); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if msg.State() != "undelivered" {
		t.Errorf("State = %q, want undelivered", msg.State())
	}
	if msg.Reason() != "carrier_blocked" {
		t.Errorf("Reason = %q, want carrier_blocked", msg.Reason())
	}
}

// TestRelay_SendMessageResolvesOnFailed — Python:
// test_send_message_resolves_on_failed.
func TestRelay_SendMessageResolvesOnFailed(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	msg, err := client.SendMessage(
		"+15551112222",
		"+15553334444",
		"hi",
	)
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	h.Push(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      "evt-msg-failed",
		"method":  "signalwire.event",
		"params": map[string]any{
			"event_type": "messaging.state",
			"params": map[string]any{
				"message_id":    msg.MessageID(),
				"message_state": "failed",
				"reason":        "spam",
			},
		},
	}, "")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := msg.Wait(ctx); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if msg.State() != "failed" {
		t.Errorf("State = %q, want failed", msg.State())
	}
}

// TestRelay_SendMessageIntermediateStateDoesNotResolve — Python:
// test_send_message_intermediate_state_does_not_resolve.
//
// Python: "intermediate" states like 'sent' update Message.state but
// don't resolve. The Go SDK's terminal-state set currently includes
// "sent" (matches MessageStateSent → terminal). Python's behavior
// differs: 'sent' is the carrier-handoff state that does NOT terminate
// the wait. We honor Python's behavior by adjusting Go's terminal set
// (see message.go's isTerminalMessageState).
func TestRelay_SendMessageIntermediateStateDoesNotResolve(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	msg, err := client.SendMessage(
		"+15551112222",
		"+15553334444",
		"hi",
	)
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	h.Push(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      "evt-msg-sent",
		"method":  "signalwire.event",
		"params": map[string]any{
			"event_type": "messaging.state",
			"params": map[string]any{
				"message_id":    msg.MessageID(),
				"message_state": "sent",
			},
		},
	}, "")
	if !waitFor(2*time.Second, func() bool {
		return msg.State() == "sent"
	}) {
		t.Errorf("State = %q, want sent", msg.State())
	}
	if msg.IsDone() {
		t.Error("IsDone() should be false for intermediate 'sent' state")
	}
}

// ---------------------------------------------------------------------------
// Inbound messages
// ---------------------------------------------------------------------------

// TestRelay_InboundMessageFiresOnMessageHandler — Python:
// test_inbound_message_fires_on_message_handler.
func TestRelay_InboundMessageFiresOnMessageHandler(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	received := make(chan *relay.Message, 1)
	client.OnMessage(func(m *relay.Message) {
		received <- m
	})
	h.Push(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      "evt-in-msg-1",
		"method":  "signalwire.event",
		"params": map[string]any{
			"event_type": "messaging.receive",
			"params": map[string]any{
				"message_id":    "in-msg-1",
				"context":       "default",
				"direction":     "inbound",
				"from_number":   "+15551110000",
				"to_number":     "+15552220000",
				"body":          "hello back",
				"media":         []any{},
				"segments":      1,
				"message_state": "received",
				"tags":          []any{"incoming"},
			},
		},
	}, "")
	select {
	case m := <-received:
		if m.MessageID() != "in-msg-1" {
			t.Errorf("MessageID = %q, want in-msg-1", m.MessageID())
		}
		if m.Direction() != "inbound" {
			t.Errorf("Direction = %q, want inbound", m.Direction())
		}
		if m.FromNumber() != "+15551110000" {
			t.Errorf("FromNumber = %q", m.FromNumber())
		}
		if m.ToNumber() != "+15552220000" {
			t.Errorf("ToNumber = %q", m.ToNumber())
		}
		if m.Body() != "hello back" {
			t.Errorf("Body = %q", m.Body())
		}
		tags := m.Tags()
		if len(tags) != 1 || tags[0] != "incoming" {
			t.Errorf("Tags = %v, want [incoming]", tags)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("on_message handler did not fire")
	}
}

// ---------------------------------------------------------------------------
// State progression — full pipeline
// ---------------------------------------------------------------------------

// TestRelay_FullMessageStateProgression — Python:
// test_full_message_state_progression. sent → delivered.
func TestRelay_FullMessageStateProgression(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	msg, err := client.SendMessage(
		"+15551112222",
		"+15553334444",
		"full pipeline",
	)
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	h.Push(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      "evt-msg-sent",
		"method":  "signalwire.event",
		"params": map[string]any{
			"event_type": "messaging.state",
			"params": map[string]any{
				"message_id":    msg.MessageID(),
				"message_state": "sent",
			},
		},
	}, "")
	if !waitFor(2*time.Second, func() bool {
		return msg.State() == "sent"
	}) {
		t.Errorf("State = %q, want sent", msg.State())
	}
	h.Push(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      "evt-msg-delivered",
		"method":  "signalwire.event",
		"params": map[string]any{
			"event_type": "messaging.state",
			"params": map[string]any{
				"message_id":    msg.MessageID(),
				"message_state": "delivered",
			},
		},
	}, "")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := msg.Wait(ctx); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if msg.State() != "delivered" {
		t.Errorf("State = %q, want delivered", msg.State())
	}
}
