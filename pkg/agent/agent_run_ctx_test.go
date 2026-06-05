// Copyright (c) 2025 SignalWire
//
// Tests for AgentBase.RunContext — the context.Context-aware form of Run that
// triggers the existing graceful HTTP shutdown when the caller's context is
// cancelled. A Go-port addition (PORT_ADDITIONS.md): the Python reference's
// run()/serve loop has no caller-supplied cancellation token.
//
// Drives a REAL HTTP server on a real ephemeral port — no transport mocks.

package agent

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"
)

func agentFreePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("agentFreePort: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func portToStr(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func agentWaitHealthy(t *testing.T, base string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("agent server never became healthy")
}

// TestAgent_RunContextCancelStops proves RunContext serves real HTTP and then
// performs a graceful shutdown — returning nil — when the context is cancelled.
func TestAgent_RunContextCancelStops(t *testing.T) {
	port := agentFreePort(t)
	a := NewAgentBase(
		WithName("ctx-agent"),
		WithHost("127.0.0.1"),
		WithPort(port),
	)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() { errCh <- a.RunContext(ctx) }()

	base := "http://127.0.0.1:" + portToStr(port)
	agentWaitHealthy(t, base)

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("RunContext returned %v, want nil on ctx cancel", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("RunContext did not return after context cancellation")
	}

	// The server must have stopped listening.
	if resp, err := http.Get(base + "/health"); err == nil {
		resp.Body.Close()
		t.Fatal("agent server still serving after ctx-cancel shutdown")
	}
}

// TestAgent_RunContextDeadlineStops proves a context deadline also drives the
// graceful shutdown.
func TestAgent_RunContextDeadlineStops(t *testing.T) {
	port := agentFreePort(t)
	a := NewAgentBase(
		WithName("ctx-agent-deadline"),
		WithHost("127.0.0.1"),
		WithPort(port),
	)

	// Long enough for the server to come up, short enough to keep the test fast.
	ctx, cancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- a.RunContext(ctx) }()

	base := "http://127.0.0.1:" + portToStr(port)
	agentWaitHealthy(t, base)

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("RunContext returned %v, want nil on deadline shutdown", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("RunContext did not return after context deadline")
	}
}

// TestAgent_RunContextPreCancelled proves a pre-cancelled context makes
// RunContext bail before binding the listener.
func TestAgent_RunContextPreCancelled(t *testing.T) {
	port := agentFreePort(t)
	a := NewAgentBase(
		WithName("ctx-agent-precancel"),
		WithHost("127.0.0.1"),
		WithPort(port),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := a.RunContext(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %T: %v", err, err)
	}
	if resp, err := http.Get("http://127.0.0.1:" + portToStr(port) + "/health"); err == nil {
		resp.Body.Close()
		t.Fatal("RunContext bound the listener despite pre-cancelled ctx")
	}
}
