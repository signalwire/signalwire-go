// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Command secret-scrub-dump is the Go port's SECRET-SCRUB-LIVE dump program for
// the cross-port behavioral differ
// (porting-sdk/scripts/diff_port_secret_scrub.py, corpus
// porting-sdk/scripts/secret_scrub_corpus.py).
//
// The contract: driving the RELAY client through a real connect + an inbound
// `signalwire.authorization.state` re-auth frame AT DEBUG LEVEL with known
// fixture credentials must leak NONE of them into the log. The differ builds the
// golden by doing exactly that against the python reference, which scrubs at both
// frame-log sites, so every golden is `{leaked: false}`. This program does the
// identical drive against the Go SDK, captures the SDK's OWN log output, and
// emits per-sentinel `{"<id>": {"leaked": <bool>}}` on stdout.
//
// # Two-phase design (why)
//
// The Go relay client logs through a `*log.Logger` that writes to the process's
// REAL stderr file descriptor, and there is no logger-injection option to
// redirect it in-process. An in-process buffer swap would therefore capture
// nothing the SDK actually emits — a vacuous `leaked: false`. So this program
// re-executes ITSELF as a child (`SW_SS_CHILD=1`) with the child's stderr wired
// to a pipe. The child performs the real drive; the PARENT reads that pipe (the
// bytes the SDK genuinely wrote), classifies each sentinel against it, forwards
// the child's output to the real stderr so the differ's own subprocess capture
// sees the identical bytes, and prints ONLY the JSON classification on stdout.
//
// Because the classification is derived from captured OUTPUT rather than from the
// source, it is unfakeable: a port cannot report `leaked: false` while its log
// literally contains `PT-TESTLEAK`.
//
// The inbound re-auth frame is delivered by an in-process gorilla/websocket
// server (the same fakeWS approach as cmd/relay-liveness-dump) rather than the
// shared mock_relay, because the fixture needs to push one specific
// `signalwire.authorization.state` event carrying the sentinel blob at a
// controlled moment.
//
// Protocol: stdout = ONE JSON object mapping corpus id -> {leaked: bool}.
//
// Run from the signalwire-go repo root:
//
//	go run ./cmd/secret-scrub-dump
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/signalwire/signalwire-go/v3/pkg/relay"
)

// Fixture sentinels — byte-identical to porting-sdk/scripts/secret_scrub_corpus.py.
const (
	sentinelProject            = "PJ-TESTLEAK"
	sentinelToken              = "PT-TESTLEAK"
	sentinelAuthorizationState = "AENC-TESTLEAK"
)

// corpus maps each corpus id to the sentinel string that must NOT appear in the
// captured log. Mirrors secret_scrub_corpus.CORPUS.
var corpus = map[string]string{
	"project":             sentinelProject,
	"token":               sentinelToken,
	"authorization_state": sentinelAuthorizationState,
}

// childEnv marks the re-executed drive phase.
const childEnv = "SW_SS_CHILD"

// boundedWindow caps the drive so a relay path that never returns fails loudly
// rather than hanging out the differ's deadline.
const boundedWindow = 8 * time.Second

func main() {
	if os.Getenv(childEnv) == "1" {
		os.Exit(childDrive())
	}
	os.Exit(parent())
}

// parent re-executes this program as a child with stderr captured, classifies
// each sentinel against the bytes the SDK actually wrote, forwards them to the
// real stderr, and prints the JSON classification to stdout.
func parent() int {
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "secret-scrub-dump: locate self: %v\n", err)
		return 1
	}

	var captured bytes.Buffer
	cmd := exec.Command(self) //nolint:gosec // G204: re-execs THIS binary (os.Executable) to capture its own scrubbed output; no external input.
	cmd.Env = append(os.Environ(),
		childEnv+"=1",
		// Drive at DEBUG — the level at which a raw-frame log site would fire.
		"SIGNALWIRE_LOG_LEVEL=debug",
		"SIGNALWIRE_LOG_MODE=default",
	)
	// Both streams are candidate leak channels: capture each, keep stdout out of
	// our own stdout so the JSON stays pure.
	cmd.Stdout = &captured
	cmd.Stderr = &captured
	runErr := cmd.Run()

	log := captured.String()
	// Forward what the SDK emitted so the differ's own subprocess-stderr capture
	// sees the identical bytes.
	if log != "" {
		fmt.Fprint(os.Stderr, log)
	}
	if runErr != nil {
		fmt.Fprintf(os.Stderr, "secret-scrub-dump: child drive failed: %v\n", runErr)
		return 1
	}

	out := map[string]map[string]bool{}
	for id, sentinel := range corpus {
		out[id] = map[string]bool{"leaked": strings.Contains(log, sentinel)}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "secret-scrub-dump: encode: %v\n", err)
		return 1
	}
	return 0
}

// childDrive performs the real drive: connect with the sentinel credentials (so
// the outbound signalwire.connect frame genuinely carries them), then receive an
// inbound signalwire.authorization.state event carrying the sentinel blob (so the
// inbound frame is genuinely processed). Everything the SDK logs goes to our
// stderr, which the parent classifies. We print NOTHING to stdout.
func childDrive() int {
	// Route this program's own diagnostics to stderr; the SDK's logger already
	// writes there. stdout must stay empty in the child.
	log.SetOutput(os.Stderr)

	fake, err := newFakeWS()
	if err != nil {
		fmt.Fprintf(os.Stderr, "secret-scrub-dump: fake relay server: %v\n", err)
		return 1
	}
	defer fake.close()

	if err := os.Setenv("SIGNALWIRE_RELAY_HOST", fake.host()); err != nil {
		fmt.Fprintf(os.Stderr, "secret-scrub-dump: set relay host: %v\n", err)
		return 1
	}
	if err := os.Setenv("SIGNALWIRE_RELAY_SCHEME", "ws"); err != nil {
		fmt.Fprintf(os.Stderr, "secret-scrub-dump: set relay scheme: %v\n", err)
		return 1
	}

	done := make(chan error, 1)
	go func() { done <- drive(fake) }()
	select {
	case err := <-done:
		if err != nil {
			fmt.Fprintf(os.Stderr, "secret-scrub-dump: drive: %v\n", err)
			return 1
		}
	case <-time.After(boundedWindow):
		fmt.Fprintf(os.Stderr, "secret-scrub-dump: drive exceeded %v\n", boundedWindow)
		return 1
	}
	return 0
}

// drive runs the connect + authenticate + read-loop + inbound-re-auth sequence
// with the sentinel credentials.
func drive(fake *fakeWS) error {
	c := relay.NewRelayClient(
		relay.WithProject(sentinelProject),
		relay.WithToken(sentinelToken),
	)
	if err := c.Connect(); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer c.Stop()
	// Authenticate SENDS the outbound signalwire.connect frame carrying
	// authentication{project, token} — the >> log site's payload.
	if err := c.Authenticate(); err != nil {
		return fmt.Errorf("authenticate: %w", err)
	}
	c.StartReadLoop()

	// Push the inbound re-auth blob — the << log site's payload.
	conn := fake.currentConn()
	if conn == nil {
		return fmt.Errorf("no upgraded websocket connection to push the re-auth frame on")
	}
	fake.writeJSON(conn, map[string]any{
		"jsonrpc": "2.0",
		"method":  relay.MethodSignalWireEvent,
		"params": map[string]any{
			"event_type": relay.EventAuthorizationState,
			"params":     map[string]any{"authorization_state": sentinelAuthorizationState},
		},
	})

	// Let the read loop receive + process (and thus potentially log) the frame.
	time.Sleep(500 * time.Millisecond)

	// Assert the drive was REAL: the SDK must have actually taken the blob in. A
	// fixture that reports leaked=false without having received the frame would be
	// vacuous.
	if got := c.AuthorizationState(); got != sentinelAuthorizationState {
		return fmt.Errorf(
			"re-auth frame was not processed: AuthorizationState()=%q, want %q",
			got, sentinelAuthorizationState,
		)
	}
	return nil
}

// ---------------------------------------------------------------------------
// fakeWS — a minimal in-process RELAY websocket server that answers the
// connect/auth handshake and lets the fixture push a server-initiated event.
// ---------------------------------------------------------------------------

type fakeWS struct {
	srv       *http.Server
	port      int
	mu        sync.Mutex
	writeMu   sync.Mutex
	liveConn  *websocket.Conn
	upgrader  websocket.Upgrader
	closeOnce sync.Once
}

func newFakeWS() (*fakeWS, error) {
	// Bind an EPHEMERAL port — never a hardcoded one (a leftover listener on a
	// fixed port is a self-inflicted failure under the concurrent matrix).
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		_ = ln.Close()
		return nil, fmt.Errorf("listener addr is %T, want *net.TCPAddr", ln.Addr())
	}
	f := &fakeWS{
		port:     addr.Port,
		upgrader: websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/relay/ws", f.handle)
	f.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = f.srv.Serve(ln) }()
	return f, nil
}

func (f *fakeWS) host() string { return fmt.Sprintf("127.0.0.1:%d", f.port) }

func (f *fakeWS) close() { f.closeOnce.Do(func() { _ = f.srv.Close() }) }

func (f *fakeWS) currentConn() *websocket.Conn {
	// The client connects asynchronously; wait briefly for the upgrade.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		conn := f.liveConn
		f.mu.Unlock()
		if conn != nil {
			return conn
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

func (f *fakeWS) writeJSON(conn *websocket.Conn, v any) {
	f.writeMu.Lock()
	defer f.writeMu.Unlock()
	_ = conn.WriteJSON(v)
}

func (f *fakeWS) handle(w http.ResponseWriter, r *http.Request) {
	conn, err := f.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	f.mu.Lock()
	f.liveConn = conn
	f.mu.Unlock()

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var msg struct {
			ID     string `json:"id"`
			Method string `json:"method"`
		}
		if json.Unmarshal(raw, &msg) != nil {
			continue
		}
		switch msg.Method {
		case relay.MethodSignalWireConnect:
			f.writeJSON(conn, map[string]any{
				"jsonrpc": "2.0", "id": msg.ID,
				"result": map[string]any{
					"protocol": "signalwire_fake", "identity": "id", "sessionid": "sess-scrub",
				},
			})
		case relay.MethodSignalWirePing:
			f.writeJSON(conn, map[string]any{
				"jsonrpc": "2.0", "id": msg.ID,
				"result": map[string]any{"timestamp": time.Now().Unix()},
			})
		default:
			f.writeJSON(conn, map[string]any{
				"jsonrpc": "2.0", "id": msg.ID,
				"result": map[string]any{"code": "200", "message": "OK"},
			})
		}
	}
}
