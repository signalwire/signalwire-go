// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Command relay-liveness-dump is the Go port's RELAY-LIVENESS dump program for
// the cross-port behavioral differ (porting-sdk/scripts/diff_port_relay_liveness.py,
// corpus porting-sdk/scripts/relay_liveness_corpus.py).
//
// The BROADER sibling of wait-liveness-dump: where WAIT-LIVENESS pins
// Action.wait() blocks-until-event, this pins the RELAY *client's* connection +
// error contract (A6 creds, A2 relay-contract, F2.1 dead-peer, F2.2 black-hole,
// F3 reconnect, max-active-calls). The differ builds the golden by driving the
// python RelayClient, then runs THIS program and structurally compares the
// per-fixture CLASSIFICATION (booleans: raised/bounded/detected/enforced — never
// raw ms), so the golden is deterministic while the behavior is real.
//
// Unlike wait-liveness-dump (which drives the REAL mock_relay), most fixtures
// here need CONTROLLABLE transport misbehavior (auth-reject, half-open, silent,
// drop-after-auth) that the python differ gets by monkeypatching
// websockets.connect. Go cannot monkeypatch, so this program stands up its OWN
// in-process gorilla/websocket server (fakeWS) speaking the connect/auth/ping
// handshake, scriptable per fixture.
//
// Protocol: stdout = ONE JSON object mapping fixture_id -> classification. Only
// stdout carries JSON; all logging goes to stderr.
//
// Run from the signalwire-go repo root:
//
//	go run ./cmd/relay-liveness-dump
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/signalwire/signalwire-go/v3/pkg/relay"
)

// boundedWindow mirrors diff_port_relay_liveness.BOUNDED_WINDOW_S: a behavior
// that outlives this is HUNG/UNBOUNDED.
const boundedWindow = 5 * time.Second

const (
	nodeID = "node-relay-live"
	callID = "call-relay-live"
	ctlID  = "ctl-relay-live-1"
)

func main() { os.Exit(run()) }

func run() int {
	out := map[string]any{}
	out["cred_missing_project"] = driveCredMissing("project")
	out["cred_missing_token"] = driveCredMissing("token")
	out["cred_auth_reject"] = driveCredAuthReject()
	out["relay_contract_500"] = driveRelayContract("500")
	out["relay_contract_404"] = driveRelayContract("404")
	out["relay_contract_410"] = driveRelayContract("410")
	out["dead_peer_half_open"] = driveDeadPeer()
	out["black_hole_silent_peer"] = driveBlackHole()
	out["reconnect_after_drop"] = driveReconnect()
	out["max_active_calls_cap"] = driveMaxActiveCalls(2)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "relay-liveness-dump: encode: %v\n", err)
		return 1
	}
	return 0
}

// ---------------------------------------------------------------------------
// fakeWS — an in-process gorilla/websocket server that speaks the RELAY
// connect/auth/ping handshake and can be scripted to misbehave per fixture.
// ---------------------------------------------------------------------------

type fakeConfig struct {
	authError  string // non-empty => reject signalwire.connect with this message
	silent     bool   // accept connect, then never answer any request (black hole)
	answerPing bool   // answer signalwire.ping (false => half-open peer)
	dropAfter  bool   // close the socket right after a successful auth (F3 first conn)
	rpcCode    string // result `code` for calling.* verbs (e.g. "500"); "" => "200"
}

type fakeWS struct {
	srv       *http.Server
	ln        net.Listener
	port      int
	mu        sync.Mutex
	connects  int
	liveConn  *websocket.Conn // most-recent upgraded conn (for server-push fixtures)
	cfg       func(connN int) fakeConfig
	upgrader  websocket.Upgrader
	closeOnce sync.Once
}

func newFakeWS(cfg func(connN int) fakeConfig) (*fakeWS, error) {
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
		ln:       ln,
		port:     addr.Port,
		cfg:      cfg,
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

func (f *fakeWS) connectCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.connects
}

func (f *fakeWS) currentConn() *websocket.Conn {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.liveConn
}

func (f *fakeWS) handle(w http.ResponseWriter, r *http.Request) {
	conn, err := f.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	f.mu.Lock()
	f.connects++
	n := f.connects
	f.liveConn = conn
	f.mu.Unlock()
	cfg := f.cfg(n)

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
			if cfg.authError != "" {
				writeJSON(conn, map[string]any{
					"jsonrpc": "2.0", "id": msg.ID,
					"error": map[string]any{
						"code": -32401, "message": cfg.authError,
						"data": map[string]any{"signalwire_error_code": "AUTH_REQUIRED"},
					},
				})
				continue
			}
			writeJSON(conn, map[string]any{
				"jsonrpc": "2.0", "id": msg.ID,
				"result": map[string]any{
					"protocol": "signalwire_fake", "identity": "id", "sessionid": "sess-fake",
				},
			})
			if cfg.dropAfter {
				time.Sleep(20 * time.Millisecond)
				return
			}
		case relay.MethodSignalWirePing:
			if cfg.silent || !cfg.answerPing {
				continue
			}
			writeJSON(conn, map[string]any{
				"jsonrpc": "2.0", "id": msg.ID,
				"result": map[string]any{"timestamp": time.Now().Unix()},
			})
		default:
			// A calling.*/signalwire.receive request.
			if cfg.silent {
				continue // black hole: accept, never respond
			}
			code := cfg.rpcCode
			if code == "" {
				code = "200"
			}
			writeJSON(conn, map[string]any{
				"jsonrpc": "2.0", "id": msg.ID,
				"result": map[string]any{"code": code, "message": "OK"},
			})
		}
	}
}

var writeMu sync.Mutex

func writeJSON(conn *websocket.Conn, v any) {
	writeMu.Lock()
	defer writeMu.Unlock()
	_ = conn.WriteJSON(v)
}

// pointAt sets the env vars so a relay.Client dials the fake server over ws.
func pointAt(f *fakeWS) {
	_ = os.Setenv("SIGNALWIRE_RELAY_HOST", f.host())
	_ = os.Setenv("SIGNALWIRE_RELAY_SCHEME", "ws")
}

// playParams is the calling.play RPC payload used by the verb-driving fixtures.
func playParams() map[string]any {
	return map[string]any{
		"node_id": nodeID, "call_id": callID, "control_id": ctlID,
		"play": []map[string]any{{"type": "tts", "params": map[string]any{"text": "hi"}}},
	}
}

// ---------------------------------------------------------------------------
// Fixture drivers — each returns the fixture's classification.
// ---------------------------------------------------------------------------

func driveCredMissing(omit string) map[string]any {
	for _, e := range []string{"SIGNALWIRE_PROJECT_ID", "SIGNALWIRE_API_TOKEN", "SIGNALWIRE_JWT_TOKEN"} {
		_ = os.Unsetenv(e)
	}
	project, token := "p", "t"
	var wants []string
	if omit == "project" {
		project = ""
		wants = []string{"project", "SIGNALWIRE_PROJECT_ID"}
	} else {
		token = ""
		wants = []string{"token", "SIGNALWIRE_API_TOKEN"}
	}
	c := relay.NewRelayClient(relay.WithProject(project), relay.WithToken(token))
	err := c.Connect()
	failed := err != nil
	actionable := failed && containsAll(err.Error(), wants)
	return map[string]any{"failed_preconnect_on_missing": failed && actionable}
}

func driveCredAuthReject() map[string]any {
	out := map[string]any{
		"raised_after_bounded_retry": false,
		"infinite_reconnect":         false,
		"server_message_surfaced":    false,
	}
	msg := "auth rejected: bad token"
	f, err := newFakeWS(func(int) fakeConfig { return fakeConfig{authError: msg, answerPing: true} })
	if err != nil {
		return out
	}
	defer f.close()
	pointAt(f)

	done := make(chan error, 1)
	go func() {
		c := relay.NewRelayClient(relay.WithProject("p"), relay.WithToken("t"))
		if e := c.Connect(); e != nil {
			done <- e
			return
		}
		done <- c.Authenticate() // surfaces the reject message
	}()
	select {
	case e := <-done:
		if e != nil {
			out["raised_after_bounded_retry"] = true
			out["server_message_surfaced"] = strings.Contains(e.Error(), msg)
		}
	case <-time.After(boundedWindow + 3*time.Second):
		out["infinite_reconnect"] = true
	}
	return out
}

func driveRelayContract(code string) map[string]any {
	out := map[string]any{"raised": false, "swallowed": false}
	f, err := newFakeWS(func(int) fakeConfig { return fakeConfig{answerPing: true, rpcCode: code} })
	if err != nil {
		return out
	}
	defer f.close()
	pointAt(f)

	c := relay.NewRelayClient(relay.WithProject("p"), relay.WithToken("t"))
	if e := c.Connect(); e != nil {
		return out
	}
	defer c.Stop()
	if e := c.Authenticate(); e != nil {
		return out
	}
	c.StartReadLoop()

	// Execute applies the A2 result-code raise; the Call-gone (404/410) swallow
	// contract (Call.execVerb) is applied here to mirror a verb call.
	_, execErr := c.Execute("calling.play", playParams())
	if execErr != nil {
		var re *relay.RelayError
		if errors.As(execErr, &re) && (re.Code == 404 || re.Code == 410) {
			out["swallowed"] = true
		} else {
			out["raised"] = true
		}
	} else {
		out["swallowed"] = true
	}
	return out
}

func driveDeadPeer() map[string]any {
	out := map[string]any{"detected_bounded": false, "hung": true}
	f, err := newFakeWS(func(int) fakeConfig { return fakeConfig{answerPing: false} })
	if err != nil {
		return out
	}
	defer f.close()
	pointAt(f)

	c := relay.NewRelayClient(
		relay.WithProject("p"), relay.WithToken("t"),
		relay.WithPingWatchdog(50*time.Millisecond, 3),
		relay.WithReconnectBackoff(20*time.Millisecond),
	)
	if e := c.Connect(); e != nil {
		return out
	}
	defer c.Stop()
	if e := c.Authenticate(); e != nil {
		return out
	}
	c.StartReadLoop()

	deadline := time.Now().Add(boundedWindow)
	for time.Now().Before(deadline) {
		if !c.IsConnected() {
			out["detected_bounded"] = true
			out["hung"] = false
			return out
		}
		time.Sleep(20 * time.Millisecond)
	}
	return out
}

func driveBlackHole() map[string]any {
	out := map[string]any{"bounded_error": false, "unbounded_hang": true}
	f, err := newFakeWS(func(int) fakeConfig { return fakeConfig{silent: true} })
	if err != nil {
		return out
	}
	defer f.close()
	pointAt(f)

	c := relay.NewRelayClient(
		relay.WithProject("p"), relay.WithToken("t"),
		relay.WithExecuteTimeout(400*time.Millisecond),
	)
	if e := c.Connect(); e != nil {
		return out
	}
	defer c.Stop()
	if e := c.Authenticate(); e != nil {
		return out
	}
	c.StartReadLoop()

	t0 := time.Now()
	_, execErr := c.Execute("calling.play", playParams())
	if execErr != nil && time.Since(t0) < boundedWindow {
		out["bounded_error"] = true
		out["unbounded_hang"] = false
	}
	return out
}

func driveReconnect() map[string]any {
	out := map[string]any{"reconnected": false, "pending_faulted_not_hung": false, "zombie": true}
	f, err := newFakeWS(func(n int) fakeConfig {
		return fakeConfig{answerPing: true, dropAfter: n == 1} // first conn drops after auth
	})
	if err != nil {
		return out
	}
	defer f.close()
	pointAt(f)

	c := relay.NewRelayClient(
		relay.WithProject("p"), relay.WithToken("t"),
		relay.WithReconnectBackoff(20*time.Millisecond),
	)
	if e := c.Connect(); e != nil {
		return out
	}
	if e := c.Authenticate(); e != nil {
		return out
	}
	c.StartReadLoop()

	deadline := time.Now().Add(boundedWindow)
	for time.Now().Before(deadline) {
		if f.connectCount() >= 2 {
			out["reconnected"] = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	// A caller after the drop must be bounded (reconnect re-drives, or execute
	// times out) — never an unbounded hang.
	t0 := time.Now()
	_, _ = c.Execute("calling.play", playParams())
	out["pending_faulted_not_hung"] = time.Since(t0) < boundedWindow

	// Tear down; assert the transport is fully released (no zombie).
	c.Stop()
	time.Sleep(50 * time.Millisecond)
	out["zombie"] = c.IsConnected()
	return out
}

func driveMaxActiveCalls(maxCalls int) map[string]any {
	out := map[string]any{"cap_enforced": false}
	f, err := newFakeWS(func(int) fakeConfig { return fakeConfig{answerPing: true} })
	if err != nil {
		return out
	}
	defer f.close()
	pointAt(f)

	var active int32
	c := relay.NewRelayClient(
		relay.WithProject("p"), relay.WithToken("t"),
		relay.WithContexts("default"),
		relay.WithMaxActiveCalls(maxCalls),
	)
	c.OnCall(func(*relay.Call) {
		atomic.AddInt32(&active, 1)
		go time.Sleep(boundedWindow) // keep it "active" for the window
	})
	if e := c.Connect(); e != nil {
		return out
	}
	defer c.Stop()
	if e := c.Authenticate(); e != nil {
		return out
	}
	c.StartReadLoop()
	if e := c.SubscribeContexts(); e != nil {
		return out
	}

	conn := f.currentConn()
	if conn == nil {
		return out
	}
	for i := range maxCalls + 1 {
		writeJSON(conn, map[string]any{
			"jsonrpc": "2.0", "method": relay.MethodSignalWireEvent,
			"params": map[string]any{
				"event_type": "calling.call.receive",
				"params": map[string]any{
					"call_id": fmt.Sprintf("c%d", i), "node_id": nodeID,
					"direction": "inbound", "call_state": "created", "context": "default",
				},
			},
		})
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(300 * time.Millisecond)
	out["cap_enforced"] = int(atomic.LoadInt32(&active)) == maxCalls
	return out
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func containsAll(s string, subs []string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
