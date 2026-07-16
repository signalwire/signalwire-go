// Copyright (c) 2025 SignalWire
//
// Tests for the context.Context-aware AgentServer.RunContext, the graceful
// AgentServer.Shutdown(ctx), and the ErrServerNotRunning sentinel. These are
// Go-port additions (PORT_ADDITIONS.md); the Python reference's AgentServer
// has no graceful-shutdown surface and no caller-cancellation token.
//
// Every test drives a REAL HTTP server bound to a real ephemeral port and
// makes REAL HTTP requests — no mocks of the transport.

package server

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/signalwire/signalwire-go/v3/pkg/agent"
)

// freePort returns an OS-assigned free TCP port on 127.0.0.1.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	defer func() { _ = ln.Close() }()
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected *net.TCPAddr, got %T", ln.Addr())
	}
	return addr.Port
}

// waitHealthy blocks until GET /health on the given base URL returns 200, or
// fails after the deadline. Proves the real server is actually listening.
func waitHealthy(t *testing.T, base string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("server never became healthy")
}

// TestServer_RunContextCancelStops proves RunContext serves real HTTP and then
// returns nil when the caller's context is cancelled (a graceful shutdown).
func TestServer_RunContextCancelStops(t *testing.T) {
	port := freePort(t)
	srv := NewAgentServer(WithServerHost("127.0.0.1"), WithServerPort(port))

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() { errCh <- srv.RunContext(ctx) }()

	base := "http://127.0.0.1:" + itoa(port)
	waitHealthy(t, base)

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("RunContext returned %v, want nil on ctx cancel", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("RunContext did not return after context cancellation")
	}

	// After shutdown, new requests must fail (server stopped listening).
	if resp, err := http.Get(base + "/health"); err == nil {
		_ = resp.Body.Close()
		t.Fatal("server still serving after ctx-cancel shutdown")
	}
}

// TestServer_ShutdownStopsRunningServer proves Shutdown(ctx) on a live server
// returns nil, unblocks RunContext (returns nil), and stops the listener.
func TestServer_ShutdownStopsRunningServer(t *testing.T) {
	port := freePort(t)
	srv := NewAgentServer(WithServerHost("127.0.0.1"), WithServerPort(port))

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run() }()

	base := "http://127.0.0.1:" + itoa(port)
	waitHealthy(t, base)

	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown returned %v, want nil", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run returned %v after Shutdown, want nil", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after Shutdown")
	}

	if resp, err := http.Get(base + "/health"); err == nil {
		_ = resp.Body.Close()
		t.Fatal("server still serving after Shutdown")
	}
}

// TestServer_ShutdownDrainsInFlight proves the core graceful-shutdown
// guarantee: an in-flight request that is mid-handler when Shutdown is called
// is DRAINED to completion (200), not dropped, and Shutdown blocks until it
// finishes. The slowness is real — the agent's SWML request hook sleeps,
// holding the connection open inside the handler.
func TestServer_ShutdownDrainsInFlight(t *testing.T) {
	const handlerHold = 600 * time.Millisecond

	var (
		handlerEntered = make(chan struct{})
		enteredOnce    sync.Once
		handlerDone    atomic.Bool
	)

	a := agent.NewAgentBase(
		agent.WithName("drainer"),
		agent.WithRoute("/"),
		agent.WithBasicAuth("u", "p"),
	)
	// A real, slow in-handler hook: it fires inside handleSWML on every SWML
	// request (GET included) and blocks, so the request is genuinely in-flight
	// across the Shutdown call.
	a.SetOnSwmlRequestHook(func(_ map[string]any, _ string, _ *http.Request) map[string]any {
		enteredOnce.Do(func() { close(handlerEntered) })
		time.Sleep(handlerHold)
		handlerDone.Store(true)
		return nil
	})

	port := freePort(t)
	srv := NewAgentServer(WithServerHost("127.0.0.1"), WithServerPort(port))
	srv.Register(a, "/")

	runErr := make(chan error, 1)
	go func() { runErr <- srv.Run() }()

	base := "http://127.0.0.1:" + itoa(port)
	waitHealthy(t, base)

	// Fire the slow request. Root agents are mounted under /_root/ by the
	// server mux (the bare /_root resolves to the index handler; the trailing
	// slash routes into the agent's own mux, stripped to "/"). An
	// authenticated GET there reaches handleSWML → our sleeping hook.
	reqResult := make(chan int, 1)
	go func() {
		req, _ := http.NewRequest(http.MethodGet, base+"/_root/", nil)
		req.SetBasicAuth("u", "p")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			reqResult <- -1
			return
		}
		defer func() { _ = resp.Body.Close() }()
		_, _ = io.Copy(io.Discard, resp.Body)
		reqResult <- resp.StatusCode
	}()

	// Wait until the handler is actually executing, then shut down.
	select {
	case <-handlerEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("slow handler never entered; cannot test drain")
	}
	if handlerDone.Load() {
		t.Fatal("handler already finished before Shutdown; timing too loose to prove drain")
	}

	shutStart := time.Now()
	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown returned %v, want nil", err)
	}
	shutElapsed := time.Since(shutStart)

	// Shutdown must have waited for the in-flight handler (drain), so it
	// blocked for roughly the remaining hold.
	if !handlerDone.Load() {
		t.Fatal("Shutdown returned before the in-flight handler completed (no drain)")
	}
	if shutElapsed < handlerHold/2 {
		t.Fatalf("Shutdown returned in %v; expected it to block ~%v draining", shutElapsed, handlerHold)
	}

	// The drained request must have completed successfully — not dropped.
	select {
	case code := <-reqResult:
		if code != http.StatusOK {
			t.Fatalf("in-flight request status = %d, want 200 (drained, not dropped)", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("in-flight request never completed")
	}

	if err := <-runErr; err != nil {
		t.Fatalf("Run returned %v after drain, want nil", err)
	}
}

// TestServer_ShutdownNotRunning proves Shutdown before Run (or after stop)
// returns ErrServerNotRunning, matchable via errors.Is.
func TestServer_ShutdownNotRunning(t *testing.T) {
	srv := NewAgentServer(WithServerHost("127.0.0.1"), WithServerPort(freePort(t)))
	err := srv.Shutdown(context.Background())
	if !errors.Is(err, ErrServerNotRunning) {
		t.Fatalf("expected errors.Is(err, ErrServerNotRunning), got %T: %v", err, err)
	}
}

// TestServer_RunContextPreCancelled proves a pre-cancelled context makes
// RunContext bail before binding the listener.
func TestServer_RunContextPreCancelled(t *testing.T) {
	port := freePort(t)
	srv := NewAgentServer(WithServerHost("127.0.0.1"), WithServerPort(port))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := srv.RunContext(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %T: %v", err, err)
	}
	// The port must still be free — nothing was bound.
	if resp, err := http.Get("http://127.0.0.1:" + itoa(port) + "/health"); err == nil {
		_ = resp.Body.Close()
		t.Fatal("RunContext bound the listener despite pre-cancelled ctx")
	}
}

// itoa is a tiny dependency-free int->string for building URLs.
func itoa(n int) string {
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
