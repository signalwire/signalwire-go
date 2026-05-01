// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Package mocktest is the Go test helper for the porting-sdk mock_relay
// WebSocket server. It mirrors the Python conftest fixtures
// (signalwire_relay_client + mock_relay) so unit tests can drive the real
// RELAY client over a real WebSocket connection against a schema-driven
// mock that journals every frame.
//
// The mock server's lifetime is per-process: the first New call probes
// http://127.0.0.1:<HTTP_PORT>/__mock__/health and either confirms a
// running server or starts one as a subprocess. Each test gets a freshly
// reset journal/scenario state via t.Cleanup. Tests do not share journal
// entries.
//
// Defaults:
//   - WebSocket port: 8775 (Go's relay slot in the per-port matrix)
//   - HTTP control port: WebSocket port + 1000 (so 9775)
//
// Override with MOCK_RELAY_PORT in the test environment if a different
// mock instance is already running.
package mocktest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/signalwire/signalwire-go/pkg/relay"
)

// setProcessGroup configures the spawned mock-server process to run in
// its own process group on Unix (Setpgid: true). This prevents signals
// to the test binary from cascading to the child and keeps the child
// detached so the testing framework's pipe-drain logic doesn't block
// on it.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// JournalEntry mirrors mock_relay.journal.JournalEntry over the wire.
//
// Each entry records a single WebSocket frame in either direction. The
// "direction" field is "recv" for SDK → server frames (what we
// JSON-RPC-execute against the server) and "send" for server → SDK
// frames (events the server pushes to us).
type JournalEntry struct {
	Timestamp    float64        `json:"timestamp"`
	Direction    string         `json:"direction"`
	Method       string         `json:"method"`
	RequestID    string         `json:"request_id"`
	Frame        map[string]any `json:"frame"`
	ConnectionID string         `json:"connection_id"`
	SessionID    string         `json:"session_id"`
}

// FrameParams returns frame["params"] as a map, or (nil, false).
func (e JournalEntry) FrameParams() (map[string]any, bool) {
	if e.Frame == nil {
		return nil, false
	}
	p, ok := e.Frame["params"].(map[string]any)
	return p, ok
}

// EventParams returns the inner params dict on a signalwire.event frame.
// For an event frame the on-wire shape is:
//
//	frame.params = {"event_type": "...", "params": {...inner...}}
//
// This helper returns that inner dict.
func (e JournalEntry) EventParams() (map[string]any, bool) {
	outer, ok := e.FrameParams()
	if !ok {
		return nil, false
	}
	inner, ok := outer["params"].(map[string]any)
	return inner, ok
}

// EventType returns the inner event_type for signalwire.event frames.
func (e JournalEntry) EventType() string {
	outer, ok := e.FrameParams()
	if !ok {
		return ""
	}
	if et, ok := outer["event_type"].(string); ok {
		return et
	}
	return ""
}

// Harness wraps the running mock server. It exposes journal accessors,
// helpers to push scenarios / inbound calls, and a reset hook tests
// register via t.Cleanup.
type Harness struct {
	HTTPURL  string // http://127.0.0.1:<httpPort>
	WSURL    string // ws://127.0.0.1:<wsPort>
	WSPort   int
	HTTPPort int

	httpClient *http.Client
}

// RelayHost returns "127.0.0.1:<wsPort>" — the host:port the SDK should
// be pointed at via the SIGNALWIRE_RELAY_HOST env var.
func (h *Harness) RelayHost() string {
	return fmt.Sprintf("127.0.0.1:%d", h.WSPort)
}

// Journal returns every journaled frame, in arrival order.
func (h *Harness) Journal(t *testing.T) []JournalEntry {
	t.Helper()
	resp, err := h.httpClient.Get(h.HTTPURL + "/__mock__/journal")
	if err != nil {
		t.Fatalf("mocktest: GET /__mock__/journal: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("mocktest: read journal body: %v", err)
	}
	var entries []JournalEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		t.Fatalf("mocktest: decode journal: %v (body=%q)", err, body)
	}
	return entries
}

// JournalRecv returns every recv (SDK → server) journal entry, optionally
// filtered by method.
func (h *Harness) JournalRecv(t *testing.T, method string) []JournalEntry {
	t.Helper()
	out := []JournalEntry{}
	for _, e := range h.Journal(t) {
		if e.Direction != "recv" {
			continue
		}
		if method != "" && e.Method != method {
			continue
		}
		out = append(out, e)
	}
	return out
}

// JournalSend returns every send (server → SDK) journal entry,
// optionally filtered by inner event_type for signalwire.event frames.
func (h *Harness) JournalSend(t *testing.T, eventType string) []JournalEntry {
	t.Helper()
	out := []JournalEntry{}
	for _, e := range h.Journal(t) {
		if e.Direction != "send" {
			continue
		}
		if eventType == "" {
			out = append(out, e)
			continue
		}
		if e.Frame != nil && e.Frame["method"] == "signalwire.event" {
			if e.EventType() == eventType {
				out = append(out, e)
			}
		}
	}
	return out
}

// JournalLast returns the most recent recv entry, optionally filtered by
// method. Fails the test if none is found.
func (h *Harness) JournalLast(t *testing.T, method string) JournalEntry {
	t.Helper()
	entries := h.JournalRecv(t, method)
	if len(entries) == 0 {
		t.Fatalf("mocktest: no recv journal entry for method=%q", method)
	}
	return entries[len(entries)-1]
}

// JournalReset clears the mock journal.
func (h *Harness) JournalReset(t *testing.T) {
	t.Helper()
	h.post(t, "/__mock__/journal/reset", nil)
}

// Reset clears journal + scenarios on the mock server. Tests do not call
// this directly — New registers it as a t.Cleanup hook.
func (h *Harness) Reset(t *testing.T) {
	t.Helper()
	h.post(t, "/__mock__/journal/reset", nil)
	h.post(t, "/__mock__/scenarios/reset", nil)
}

// post is an internal HTTP-POST helper.
func (h *Harness) post(t *testing.T, path string, body any) []byte {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("mocktest: marshal body for %s: %v", path, err)
		}
		rdr = bytes.NewReader(buf)
	}
	req, err := http.NewRequest("POST", h.HTTPURL+path, rdr)
	if err != nil {
		t.Fatalf("mocktest: build %s: %v", path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := h.httpClient.Do(req)
	if err != nil {
		t.Fatalf("mocktest: %s: %v", path, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("mocktest: read body for %s: %v", path, err)
	}
	if resp.StatusCode >= 400 {
		t.Fatalf("mocktest: %s -> %d: %s", path, resp.StatusCode, respBody)
	}
	return respBody
}

// Push delivers a single frame to all connected sessions (or one
// session if sessionID != "").
func (h *Harness) Push(t *testing.T, frame map[string]any, sessionID string) {
	t.Helper()
	path := "/__mock__/push"
	if sessionID != "" {
		path = path + "?session_id=" + sessionID
	}
	h.post(t, path, map[string]any{"frame": frame})
}

// InboundCallOpts holds parameters for the inbound_call helper.
type InboundCallOpts struct {
	CallID     string
	FromNumber string
	ToNumber   string
	Context    string
	AutoStates []string
	DelayMS    int
	SessionID  string
}

// InboundCall invokes the mock's /__mock__/inbound_call endpoint, which
// scripts the typical inbound-call sequence (calling.call.receive plus
// state events for each entry in AutoStates).
func (h *Harness) InboundCall(t *testing.T, opts InboundCallOpts) {
	t.Helper()
	body := map[string]any{
		"from_number": valOrDefault(opts.FromNumber, "+15551234567"),
		"to_number":   valOrDefault(opts.ToNumber, "+15559876543"),
		"context":     valOrDefault(opts.Context, "default"),
	}
	if opts.CallID != "" {
		body["call_id"] = opts.CallID
	}
	if len(opts.AutoStates) == 0 {
		body["auto_states"] = []string{"created"}
	} else {
		body["auto_states"] = opts.AutoStates
	}
	if opts.DelayMS > 0 {
		body["delay_ms"] = opts.DelayMS
	} else {
		body["delay_ms"] = 50
	}
	if opts.SessionID != "" {
		body["session_id"] = opts.SessionID
	}
	h.post(t, "/__mock__/inbound_call", body)
}

// ScenarioPlay runs a scripted timeline (mix of sleep_ms / push /
// expect_recv ops). The op shape matches the JSON the mock expects;
// callers can pass map literals.
func (h *Harness) ScenarioPlay(t *testing.T, ops []map[string]any) map[string]any {
	t.Helper()
	body := h.post(t, "/__mock__/scenario_play", ops)
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("mocktest: decode scenario_play response: %v", err)
	}
	return out
}

// ArmMethod queues scripted post-RPC events for `method`. Each entry is
// a map of {emit: {state, ...}, delay_ms?: int, event_type?: string}.
func (h *Harness) ArmMethod(t *testing.T, method string, events []map[string]any) {
	t.Helper()
	h.post(t, "/__mock__/scenarios/"+method, events)
}

// DialOpts mirrors the dial-scenario JSON body.
type DialOpts struct {
	Tag           string         `json:"tag"`
	WinnerCallID  string         `json:"winner_call_id"`
	States        []string       `json:"states"`
	NodeID        string         `json:"node_id"`
	Device        map[string]any `json:"device,omitempty"`
	Losers        []DialLoserOpts `json:"losers,omitempty"`
	DelayMS       int            `json:"delay_ms,omitempty"`
}

// DialLoserOpts mirrors the loser leg shape inside arm_dial.
type DialLoserOpts struct {
	CallID string   `json:"call_id"`
	States []string `json:"states"`
}

// ArmDial queues a full dial dance for the next calling.dial whose params
// carry the given Tag.
func (h *Harness) ArmDial(t *testing.T, opts DialOpts) {
	t.Helper()
	h.post(t, "/__mock__/scenarios/dial", opts)
}

// Sessions returns the active WebSocket session list reported by the
// mock. Tests rarely need this directly but it's exposed for scripted
// scenarios that target a single session.
func (h *Harness) Sessions(t *testing.T) []map[string]any {
	t.Helper()
	resp, err := h.httpClient.Get(h.HTTPURL + "/__mock__/sessions")
	if err != nil {
		t.Fatalf("mocktest: GET /__mock__/sessions: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("mocktest: read sessions body: %v", err)
	}
	var payload struct {
		Sessions []map[string]any `json:"sessions"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("mocktest: decode sessions: %v (body=%q)", err, body)
	}
	return payload.Sessions
}

func valOrDefault(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

// ---------------------------------------------------------------------------
// Server lifecycle
// ---------------------------------------------------------------------------

// serverState holds the singleton harness so subsequent New calls reuse
// the same backing server.
type serverState struct {
	once     sync.Once
	harness  *Harness
	cmd      *exec.Cmd
	startErr error
}

var state serverState

// defaultWSPort is the Go relay slot in the per-port matrix. Override
// via MOCK_RELAY_PORT.
const defaultWSPort = 8775

// startupTimeout caps the time we wait for the mock-relay subprocess to
// answer /__mock__/health. The Python harness boots in ~1s, but
// `python -m mock_relay` includes module import + schema loading which
// can stretch to ~5s on a cold cache.
const startupTimeout = 30 * time.Second

// discoverPortingSDKPackage walks up from this source file looking for
// an adjacent ``porting-sdk/test_harness/<name>/<name>/__init__.py``.
// The adjacency contract is "porting-sdk lives next to signalwire-go in
// ~/src/", so a fresh clone of either repo can find the mock harness
// with no prior pip install. Returns the absolute path to the
// directory containing the Python package (i.e. the path that should
// be added to PYTHONPATH so that ``python -m <name>`` resolves), or
// "" when no adjacent porting-sdk is reachable.
func discoverPortingSDKPackage(name string) string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	here, err := filepath.Abs(file)
	if err != nil {
		return ""
	}
	dir := filepath.Dir(here)
	for {
		candidate := filepath.Join(filepath.Dir(dir), "porting-sdk", "test_harness", name)
		init := filepath.Join(candidate, name, "__init__.py")
		if info, err := os.Stat(init); err == nil && !info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// resolveWSPort returns the configured mock WebSocket port — either
// MOCK_RELAY_PORT or the default 8775.
func resolveWSPort() int {
	if raw := os.Getenv("MOCK_RELAY_PORT"); raw != "" {
		if p, err := strconv.Atoi(raw); err == nil && p > 0 {
			return p
		}
	}
	return defaultWSPort
}

// ensureServer probes the mock server's health endpoint and starts a
// subprocess if nothing's listening. The subprocess runs `python -m
// mock_relay --ws-port <wsPort> --http-port <httpPort> --log-level
// error` and is left running until process exit (test runs are short
// and the OS cleans up).
//
// Subsequent calls reuse the singleton; only the first New call pays
// the startup cost.
func ensureServer(t *testing.T) *Harness {
	t.Helper()
	state.once.Do(func() {
		wsPort := resolveWSPort()
		httpPort := wsPort + 1000
		client := &http.Client{Timeout: 2 * time.Second}
		httpURL := fmt.Sprintf("http://127.0.0.1:%d", httpPort)
		wsURL := fmt.Sprintf("ws://127.0.0.1:%d", wsPort)

		// Probe — if a server is already running we reuse it.
		if probeHealth(client, httpURL) {
			state.harness = &Harness{
				HTTPURL:    httpURL,
				WSURL:      wsURL,
				WSPort:     wsPort,
				HTTPPort:   httpPort,
				httpClient: client,
			}
			return
		}

		// Spawn a subprocess. We deliberately detach stdout/stderr
		// (point them at /dev/null) because Go's testing runner waits
		// on the child's pipes before exiting, which would hang the
		// test process for the full WaitDelay (60s by default) when
		// the subprocess stays alive across the test binary lifetime.
		cmd := exec.Command("python", "-m", "mock_relay",
			"--host", "127.0.0.1",
			"--ws-port", strconv.Itoa(wsPort),
			"--http-port", strconv.Itoa(httpPort),
			"--log-level", "error",
		)
		// Try to inject porting-sdk/test_harness/mock_relay/ into
		// PYTHONPATH so `python -m mock_relay` resolves without a
		// prior `pip install -e ...`. Adjacency contract: porting-sdk
		// next to signalwire-go in ~/src/. When the walk fails (e.g.
		// because porting-sdk is not adjacent), we still spawn — the
		// child will fall back to whatever is on the system Python's
		// sys.path, and surface a clear "module not found" error from
		// the spawn-readiness probe if neither mode is available.
		if pkgDir := discoverPortingSDKPackage("mock_relay"); pkgDir != "" {
			env := os.Environ()
			existingPP := os.Getenv("PYTHONPATH")
			newPP := pkgDir
			if existingPP != "" {
				newPP = pkgDir + string(os.PathListSeparator) + existingPP
			}
			replaced := false
			for i, kv := range env {
				if len(kv) >= 11 && kv[:11] == "PYTHONPATH=" {
					env[i] = "PYTHONPATH=" + newPP
					replaced = true
					break
				}
			}
			if !replaced {
				env = append(env, "PYTHONPATH="+newPP)
			}
			// Set MOCK_RELAY_PORT for the child so its own defaults
			// see the same port we chose.
			ppReplaced := false
			for i, kv := range env {
				if len(kv) >= 16 && kv[:16] == "MOCK_RELAY_PORT=" {
					env[i] = "MOCK_RELAY_PORT=" + strconv.Itoa(wsPort)
					ppReplaced = true
					break
				}
			}
			if !ppReplaced {
				env = append(env, "MOCK_RELAY_PORT="+strconv.Itoa(wsPort))
			}
			cmd.Env = env
		}
		devnull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
		if err == nil {
			cmd.Stdout = devnull
			cmd.Stderr = devnull
			cmd.Stdin = devnull
		}
		setProcessGroup(cmd)
		if err := cmd.Start(); err != nil {
			state.startErr = fmt.Errorf("mocktest: failed to spawn `python -m mock_relay`: %w (set MOCK_RELAY_PORT to use a pre-running instance)", err)
			return
		}

		deadline := time.Now().Add(startupTimeout)
		for time.Now().Before(deadline) {
			if probeHealth(client, httpURL) {
				state.harness = &Harness{
					HTTPURL:    httpURL,
					WSURL:      wsURL,
					WSPort:     wsPort,
					HTTPPort:   httpPort,
					httpClient: client,
				}
				state.cmd = cmd
				go func() { _ = cmd.Wait() }()
				return
			}
			time.Sleep(150 * time.Millisecond)
		}
		_ = cmd.Process.Kill()
		state.startErr = fmt.Errorf("mocktest: `python -m mock_relay` did not become ready within %s on ws_port=%d http_port=%d (clone porting-sdk next to signalwire-go so tests can find porting-sdk/test_harness/mock_relay/, or pip install the mock_relay package)", startupTimeout, wsPort, httpPort)
	})
	if state.startErr != nil {
		t.Skipf("mocktest: mock_relay unavailable: %v", state.startErr)
		return nil
	}
	return state.harness
}

// probeHealth returns true when the mock server's /__mock__/health
// responds with 200 OK and a payload containing "schemas_loaded".
func probeHealth(client *http.Client, base string) bool {
	resp, err := client.Get(base + "/__mock__/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}
	_, ok := payload["schemas_loaded"]
	return ok
}

// ---------------------------------------------------------------------------
// Test entry point
// ---------------------------------------------------------------------------

// New returns (client, harness) for a single test. The client is a real
// *relay.Client wired up to the mock's WebSocket via SIGNALWIRE_RELAY_HOST
// + SIGNALWIRE_RELAY_SCHEME=ws (the same host-override env vars the
// audit_relay_handshake.py fixture uses). The client is started with
// project=test_proj / token=test_tok / contexts=["default"] — matching
// the Python signalwire_relay_client fixture.
//
// The mock's journal + scenarios are reset before the test runs, and the
// reset is repeated via t.Cleanup so accidental leftover state from a
// panic doesn't leak into the next test.
//
// The returned client is connected and authenticated; tests may call
// any Client method directly. Disconnect happens automatically via
// t.Cleanup.
func New(t *testing.T) (*relay.Client, *Harness) {
	t.Helper()
	h := ensureServer(t)
	if h == nil {
		return nil, nil
	}
	h.Reset(t)
	t.Cleanup(func() { h.Reset(t) })

	// Point the SDK at our mock. The relay client honors
	// SIGNALWIRE_RELAY_HOST + SIGNALWIRE_RELAY_SCHEME for exactly this
	// override scenario.
	t.Setenv("SIGNALWIRE_RELAY_HOST", h.RelayHost())
	t.Setenv("SIGNALWIRE_RELAY_SCHEME", "ws")

	client := relay.NewRelayClient(
		relay.WithProject("test_proj"),
		relay.WithToken("test_tok"),
		relay.WithContexts("default"),
	)

	// Drive the connect/auth handshake without starting the read loop's
	// blocking Run() — tests want a connected client they can use
	// synchronously. We mirror what Run() does: connect, authenticate,
	// then start the read loop in a goroutine. Without the read loop
	// running, subsequent Execute() calls block on response delivery.
	if err := client.Connect(); err != nil {
		t.Fatalf("mocktest: relay Connect: %v", err)
	}
	if err := client.Authenticate(); err != nil {
		t.Fatalf("mocktest: relay Authenticate: %v", err)
	}
	client.StartReadLoop()
	if err := client.SubscribeContexts(); err != nil {
		t.Fatalf("mocktest: relay SubscribeContexts: %v", err)
	}

	t.Cleanup(func() {
		client.Stop()
	})
	return client, h
}

// NewClientOnly returns a fresh *relay.Client connected to the running
// mock, with no contexts subscribed. Useful for tests that need to
// drive multiple clients in one test (e.g. reconnect-with-protocol).
//
// The harness is shared with any previously created clients in the same
// test — t.Cleanup is registered to disconnect this client when the
// test ends.
func NewClientOnly(t *testing.T, h *Harness, opts ...relay.ClientOption) *relay.Client {
	t.Helper()
	t.Setenv("SIGNALWIRE_RELAY_HOST", h.RelayHost())
	t.Setenv("SIGNALWIRE_RELAY_SCHEME", "ws")

	client := relay.NewRelayClient(opts...)
	if err := client.Connect(); err != nil {
		t.Fatalf("mocktest: relay Connect: %v", err)
	}
	if err := client.Authenticate(); err != nil {
		t.Fatalf("mocktest: relay Authenticate: %v", err)
	}
	client.StartReadLoop()
	if err := client.SubscribeContexts(); err != nil {
		t.Fatalf("mocktest: relay SubscribeContexts: %v", err)
	}
	t.Cleanup(func() { client.Stop() })
	return client
}
