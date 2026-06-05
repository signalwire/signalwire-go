// Copyright (c) 2025 SignalWire
//
// Mock-relay-backed tests for the context.Context-aware entry points
// (DialContext, RunContext) and the errors.Is-able sentinel errors
// (ErrDialTimeout, ErrDialFailed, ErrNotConnected). These are Go-port
// additions (PORT_ADDITIONS.md); the Python reference has neither
// caller-cancellation nor a sentinel-error set.
//
// Every test drives the REAL WebSocket flow against the shared mock_relay
// server via the mocktest harness — no mocks of the transport.

package relay_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/signalwire/signalwire-go/pkg/relay"
	"github.com/signalwire/signalwire-go/pkg/relay/internal/mocktest"
)

// TestRelay_DialContextCancelAborts verifies that cancelling the caller's
// context aborts an in-flight DialContext promptly (well before its
// dial-timeout would fire) and surfaces context.Canceled. No dial event is
// armed, so absent cancellation the call would block for the full timeout.
func TestRelay_DialContextCancelAborts(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	_ = h

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel shortly after the dial frame is on the wire — proving the cancel
	// (not a timeout) ended the call.
	go func() {
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if len(h.JournalRecv(t, "calling.dial")) > 0 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		cancel()
	}()

	start := time.Now()
	_, err := client.DialContext(
		ctx,
		[][]map[string]any{{phoneDevice("", "")}},
		relay.WithDialTag("t-ctx-cancel"),
		relay.WithDialClientTimeout(60*time.Second), // long: cancel must win
	)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from cancelled DialContext, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected errors.Is(err, context.Canceled), got %T: %v", err, err)
	}
	// Must have returned because of the cancel, not the 60s dial timeout.
	if elapsed > 10*time.Second {
		t.Fatalf("DialContext took %v; cancellation did not abort it promptly", elapsed)
	}
}

// TestRelay_DialContextDeadlineAborts verifies that a context deadline shorter
// than the dial-timeout aborts DialContext with context.DeadlineExceeded.
func TestRelay_DialContextDeadlineAborts(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	_ = h

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := client.DialContext(
		ctx,
		[][]map[string]any{{phoneDevice("", "")}},
		relay.WithDialTag("t-ctx-deadline"),
		relay.WithDialClientTimeout(60*time.Second), // long: deadline must win
	)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from deadline'd DialContext, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected errors.Is(err, context.DeadlineExceeded), got %T: %v", err, err)
	}
	if elapsed > 10*time.Second {
		t.Fatalf("DialContext took %v; deadline did not abort it promptly", elapsed)
	}
}

// TestRelay_DialContextPreCancelledReturnsImmediately verifies that passing an
// already-cancelled context returns without ever sending a dial frame.
func TestRelay_DialContextPreCancelledReturnsImmediately(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the call

	_, err := client.DialContext(
		ctx,
		[][]map[string]any{{phoneDevice("", "")}},
		relay.WithDialTag("t-ctx-precancel"),
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %T: %v", err, err)
	}
	// No calling.dial frame should have been emitted — we bailed pre-flight.
	if got := h.JournalRecv(t, "calling.dial"); len(got) != 0 {
		t.Fatalf("expected no calling.dial frame on pre-cancelled ctx, got %d", len(got))
	}
}

// TestRelay_DialTimeoutIsErrDialTimeout verifies that a real dial timeout (no
// scripted dial event) is matchable BOTH via errors.Is(ErrDialTimeout) — the
// new sentinel — AND via errors.As(*RelayError) — the pre-existing typed
// error. The two coexist because RelayError.Unwrap() exposes the sentinel.
func TestRelay_DialTimeoutIsErrDialTimeout(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	_ = h

	_, err := client.Dial(
		[][]map[string]any{{phoneDevice("", "")}},
		relay.WithDialTag("t-sentinel-timeout"),
		relay.WithDialClientTimeout(400*time.Millisecond),
	)
	if err == nil {
		t.Fatal("expected dial timeout error, got nil")
	}
	if !errors.Is(err, relay.ErrDialTimeout) {
		t.Fatalf("expected errors.Is(err, relay.ErrDialTimeout), got %T: %v", err, err)
	}
	// The sentinel must NOT leak into an unrelated bucket.
	if errors.Is(err, relay.ErrDialFailed) {
		t.Fatal("timeout error wrongly matches ErrDialFailed")
	}
	var rerr *relay.RelayError
	if !errors.As(err, &rerr) {
		t.Fatalf("expected the timeout to remain a *relay.RelayError, got %T", err)
	}
	if rerr.Code != -1 {
		t.Errorf("RelayError.Code = %d, want -1", rerr.Code)
	}
}

// TestRelay_DialFailedIsErrDialFailed verifies the failed-dial path is
// matchable via errors.Is(ErrDialFailed) and stays a *RelayError.
func TestRelay_DialFailedIsErrDialFailed(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}

	go func() {
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if len(h.JournalRecv(t, "calling.dial")) > 0 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		h.Push(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      "fail-evt-sentinel",
			"method":  "signalwire.event",
			"params": map[string]any{
				"event_type": "calling.call.dial",
				"params": map[string]any{
					"tag":        "t-sentinel-failed",
					"node_id":    "node-mock-1",
					"dial_state": "failed",
					"call":       map[string]any{},
				},
			},
		}, "")
	}()

	_, err := client.Dial(
		[][]map[string]any{{phoneDevice("", "")}},
		relay.WithDialTag("t-sentinel-failed"),
		relay.WithDialClientTimeout(5*time.Second),
	)
	if err == nil {
		t.Fatal("expected failed-dial error, got nil")
	}
	if !errors.Is(err, relay.ErrDialFailed) {
		t.Fatalf("expected errors.Is(err, relay.ErrDialFailed), got %T: %v", err, err)
	}
	if errors.Is(err, relay.ErrDialTimeout) {
		t.Fatal("failed error wrongly matches ErrDialTimeout")
	}
	var rerr *relay.RelayError
	if !errors.As(err, &rerr) {
		t.Fatalf("expected the failure to remain a *relay.RelayError, got %T", err)
	}
}

// TestRelay_NotConnectedIsErrNotConnected verifies that issuing an RPC on a
// client that was never connected surfaces ErrNotConnected via errors.Is.
// This drives the real writeJSON path (nil conn), no mocks.
func TestRelay_NotConnectedIsErrNotConnected(t *testing.T) {
	// A bare client — never Connect()'d, so c.conn is nil.
	client := relay.NewRelayClient(
		relay.WithProject("p"),
		relay.WithToken("t"),
	)
	err := client.Notify("signalwire.event", map[string]any{"x": 1})
	if err == nil {
		t.Fatal("expected ErrNotConnected from Notify on unconnected client, got nil")
	}
	if !errors.Is(err, relay.ErrNotConnected) {
		t.Fatalf("expected errors.Is(err, relay.ErrNotConnected), got %T: %v", err, err)
	}
}

// TestRelay_RunContextCancelStops verifies that RunContext drives the full
// connect/authenticate/subscribe/read-loop flow against the mock and then
// returns promptly when the caller's context is cancelled, surfacing
// context.Canceled. Run itself blocks forever; RunContext makes it
// cancellable.
func TestRelay_RunContextCancelStops(t *testing.T) {
	// mocktest.New sets SIGNALWIRE_RELAY_HOST/SCHEME for us and gives a live
	// harness; we build our OWN client so RunContext owns the lifecycle
	// (the harness's client is already read-looping).
	_, h := mocktest.New(t)
	if h == nil {
		return
	}

	client := relay.NewRelayClient(
		relay.WithProject("test_proj"),
		relay.WithToken("test_tok"),
		relay.WithContexts("default"),
	)
	t.Cleanup(client.Stop)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- client.RunContext(ctx)
	}()

	// Wait until RunContext has actually authenticated (proves the real flow
	// ran), then cancel.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if client.RelayProtocol() != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if client.RelayProtocol() == "" {
		t.Fatal("RunContext never completed the auth handshake against the mock")
	}

	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected RunContext to return context.Canceled, got %T: %v", err, err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("RunContext did not return after context cancellation")
	}
}

// TestRelay_RunContextPreCancelledReturnsImmediately verifies a pre-cancelled
// context makes RunContext bail before connecting.
func TestRelay_RunContextPreCancelledReturnsImmediately(t *testing.T) {
	_, h := mocktest.New(t)
	if h == nil {
		return
	}
	client := relay.NewRelayClient(
		relay.WithProject("test_proj"),
		relay.WithToken("test_tok"),
	)
	t.Cleanup(client.Stop)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.RunContext(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %T: %v", err, err)
	}
	if client.RelayProtocol() != "" {
		t.Fatal("RunContext connected despite a pre-cancelled context")
	}
}
