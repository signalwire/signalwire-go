// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Package mocktest is the Go test helper for the shared mock_relay server
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
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/signalwire/signalwire-go/pkg/relay"
)

// setProcessGroup configures the spawned mock-server process to run in its own
// process group. Its implementation is platform-specific (Unix sets
// Setpgid:true; Windows is a no-op) and lives in build-constrained files
// (setprocessgroup_unix.go / setprocessgroup_windows.go) because the
// syscall.SysProcAttr.Setpgid field does not exist on Windows — referencing it
// unconditionally fails to COMPILE there (not a runtime branch).

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

	// SessionID scopes journal reads, scenario arming, pushes, and reset to a
	// single RELAY session (the server-assigned `sessionid` from the connect
	// handshake). When set, control-plane calls carry `?session_id=<id>` (and
	// scenario_play stamps each push/expect_recv op), so a test only ever sees
	// and disturbs its own frames — making the shared singleton mock safe under
	// parallel (t.Parallel) execution. New() sets this automatically to the
	// connected client's session. Empty => global (legacy, single-threaded).
	//
	// Mirrors signalwire-typescript's MockRelayHarness.sessionId.
	SessionID string

	httpClient *http.Client
}

// RelayHost returns "127.0.0.1:<wsPort>" — the host:port the SDK should
// be pointed at via the SIGNALWIRE_RELAY_HOST env var.
func (h *Harness) RelayHost() string {
	return fmt.Sprintf("127.0.0.1:%d", h.WSPort)
}

// scoped returns a shallow copy of this harness with SessionID set to the given
// session id. The copy shares the same underlying HTTP client + server address;
// only the session scope differs. Used by New()/NewClientOnly() to hand each
// test (or each client built mid-test) a view scoped to its own session, while
// leaving the package-singleton harness untouched. Mirrors the TS harness's
// per-call `new MockRelayHarness(...)` + `mock.sessionId = ...`.
func (h *Harness) scoped(sessionID string) *Harness {
	cp := *h
	cp.SessionID = sessionID
	return &cp
}

// ScopeToClient returns a copy of this harness scoped to the given client's
// captured connect-handshake session id. Tests that build their own client
// mid-test (e.g. a second client via NewClientOnly already returns a scoped
// harness, but a hand-rolled relay.NewRelayClient does not) call this to
// re-scope the harness to that client's session. Mirrors the TS harness's
// `mock.sessionId = sessionIdOf(client)`.
func (h *Harness) ScopeToClient(client *relay.Client) *Harness {
	return h.scoped(client.SessionID())
}

// sessionQuery returns the "?session_id=<id>" suffix when this harness is
// session-scoped, or "" otherwise. The id is URL-escaped so a session id with
// reserved characters stays a single query value.
func (h *Harness) sessionQuery() string {
	if h.SessionID == "" {
		return ""
	}
	return "?session_id=" + url.QueryEscape(h.SessionID)
}

// Journal returns every journaled frame, in arrival order. Scoped to this
// harness's SessionID when set (so a parallel test never sees another test's
// frames); unscoped harnesses see the whole journal.
func (h *Harness) Journal(t *testing.T) []JournalEntry {
	t.Helper()
	resp, err := h.httpClient.Get(h.HTTPURL + "/__mock__/journal" + h.sessionQuery())
	if err != nil {
		t.Fatalf("mocktest: GET /__mock__/journal: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
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

// JournalReset clears the mock journal. Scoped to this harness's SessionID
// when set (clears only this session's entries), else clears everything.
func (h *Harness) JournalReset(t *testing.T) {
	t.Helper()
	h.post(t, "/__mock__/journal/reset"+h.sessionQuery(), nil)
}

// Reset clears journal + scenarios on the mock server. Scoped to this harness's
// SessionID when set (so a parallel test only ever clears its own session and
// never races a concurrent test's global state), else clears everything. Tests
// do not call this directly — New registers it as a t.Cleanup hook.
func (h *Harness) Reset(t *testing.T) {
	t.Helper()
	h.post(t, "/__mock__/journal/reset"+h.sessionQuery(), nil)
	h.post(t, "/__mock__/scenarios/reset"+h.sessionQuery(), nil)
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
	// Most control-plane calls are fast (the shared 2s-timeout client is fine).
	// `/__mock__/scenario_play` is the exception: it plays a scripted timeline
	// SYNCHRONOUSLY (sleep_ms steps + event round-trips) and only responds when
	// the whole timeline completes, so under CI load it routinely exceeds 2s.
	// Use a generous per-call timeout for it so the POST doesn't spuriously
	// "context deadline exceeded" while the server is legitimately mid-timeline.
	client := h.httpClient
	if strings.Contains(path, "/scenario_play") {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("mocktest: %s: %v", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("mocktest: read body for %s: %v", path, err)
	}
	if resp.StatusCode >= 400 {
		t.Fatalf("mocktest: %s -> %d: %s", path, resp.StatusCode, respBody)
	}
	return respBody
}

// Push delivers a single frame to this harness's session (when scoped) so a
// parallel test's client never receives it. An explicit sessionID arg overrides
// the harness scope; an unscoped harness with sessionID=="" broadcasts to every
// connected session (legacy single-threaded behavior). Mirrors the TS harness's
// push(frame, sessionId?).
func (h *Harness) Push(t *testing.T, frame map[string]any, sessionID string) {
	t.Helper()
	target := sessionID
	if target == "" {
		target = h.SessionID
	}
	path := "/__mock__/push"
	if target != "" {
		path = path + "?session_id=" + url.QueryEscape(target)
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
	// Target this harness's session by default so the inbound-call sequence is
	// delivered only to this test's client (an unscoped harness broadcasts, as
	// before). An explicit opts.SessionID overrides. Mirrors the TS harness's
	// inboundCall default-to-this-session behavior.
	sid := opts.SessionID
	if sid == "" {
		sid = h.SessionID
	}
	if sid != "" {
		body["session_id"] = sid
	}
	h.post(t, "/__mock__/inbound_call", body)
}

// ScenarioPlay runs a scripted timeline (mix of sleep_ms / push /
// expect_recv ops). The op shape matches the JSON the mock expects;
// callers can pass map literals.
//
// When this harness is session-scoped, each push/expect_recv op is stamped with
// this session id (unless it already carries a session_id), so the timeline
// targets only this test's client and expect_recv matches only this session's
// frames — making it parallel-safe. (The eviction-bug fix that makes the
// session-filtered scan correct already lives in the mock server; the harness
// only has to stamp.) Mirrors the TS harness's scenarioPlay scopeOp().
func (h *Harness) ScenarioPlay(t *testing.T, ops []map[string]any) map[string]any {
	t.Helper()
	scoped := ops
	if h.SessionID != "" {
		scoped = make([]map[string]any, len(ops))
		for i, op := range ops {
			scoped[i] = h.scopeOp(op)
		}
	}
	body := h.post(t, "/__mock__/scenario_play", scoped)
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("mocktest: decode scenario_play response: %v", err)
	}
	return out
}

// scopeOp injects this harness's SessionID into a timeline op's push/expect_recv
// spec when the op doesn't already specify a session_id. Leaves sleep ops
// untouched. Returns a shallow copy so the caller's map literals are unmodified.
func (h *Harness) scopeOp(op map[string]any) map[string]any {
	out := make(map[string]any, len(op))
	for k, v := range op {
		out[k] = v
	}
	for _, key := range []string{"push", "expect_recv"} {
		spec, ok := out[key].(map[string]any)
		if !ok {
			continue
		}
		if _, has := spec["session_id"]; has {
			continue
		}
		newSpec := make(map[string]any, len(spec)+1)
		for k, v := range spec {
			newSpec[k] = v
		}
		newSpec["session_id"] = h.SessionID
		out[key] = newSpec
	}
	return out
}

// ArmMethod queues scripted post-RPC events for `method`. Each entry is
// a map of {emit: {state, ...}, delay_ms?: int, event_type?: string}.
// Scoped to this harness's SessionID when set, so a parallel test's
// calling.<method> execute won't consume another test's armed events.
func (h *Harness) ArmMethod(t *testing.T, method string, events []map[string]any) {
	t.Helper()
	h.post(t, "/__mock__/scenarios/"+method+h.sessionQuery(), events)
}

// DialOpts mirrors the dial-scenario JSON body.
type DialOpts struct {
	Tag          string          `json:"tag"`
	WinnerCallID string          `json:"winner_call_id"`
	States       []string        `json:"states"`
	NodeID       string          `json:"node_id"`
	Device       map[string]any  `json:"device,omitempty"`
	Losers       []DialLoserOpts `json:"losers,omitempty"`
	DelayMS      int             `json:"delay_ms,omitempty"`
}

// DialLoserOpts mirrors the loser leg shape inside arm_dial.
type DialLoserOpts struct {
	CallID string   `json:"call_id"`
	States []string `json:"states"`
}

// ArmDial queues a full dial dance for the next calling.dial whose params
// carry the given Tag. Scoped to this harness's SessionID when set, so a
// parallel dial test won't consume another test's queued dial dance.
func (h *Harness) ArmDial(t *testing.T, opts DialOpts) {
	t.Helper()
	h.post(t, "/__mock__/scenarios/dial"+h.sessionQuery(), opts)
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
	defer func() { _ = resp.Body.Close() }()
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

// getJSON is a non-fatal GET+decode used by DumpDiagnostics so that dumping
// state on a failure path never itself calls t.Fatalf (which would mask the
// original failure). Returns the raw body on decode error.
func (h *Harness) getJSON(path string, out any) (string, error) {
	resp, err := h.httpClient.Get(h.HTTPURL + path)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return string(body), err
		}
	}
	return string(body), nil
}

// DumpDiagnostics writes the full mock-server journal and live WS sessions to
// the test log. Call it right before failing an event-driven wait so that an
// intermittent CI failure ("handler never fired") is self-diagnosing: the dump
// shows whether the server sent the event at all, over which session, and
// whether the SDK's session was still connected. It never fails the test
// itself (best-effort, swallows transport errors) so it can run on the failure
// path without masking the original cause.
//
// `context` is a short label (usually the wait that timed out) included at the
// top of the dump.
func (h *Harness) DumpDiagnostics(t *testing.T, context string) {
	t.Helper()
	t.Logf("=== mocktest diagnostics: %s ===", context)

	// Live sessions: the most decisive signal for "event never fired" — if the
	// SDK's session is absent/closed here, the push had nowhere to go.
	var sess struct {
		Sessions []map[string]any `json:"sessions"`
	}
	if raw, err := h.getJSON("/__mock__/sessions", &sess); err != nil {
		t.Logf("  sessions: <unavailable: %v> raw=%q", err, raw)
	} else if len(sess.Sessions) == 0 {
		t.Logf("  sessions: NONE (no live WS session — a push would be dropped)")
	} else {
		for i, s := range sess.Sessions {
			t.Logf("  session[%d]: %v", i, s)
		}
	}

	// Full journal in arrival order, both directions, with session attribution
	// so a send-to-wrong-session race is visible.
	var entries []JournalEntry
	if raw, err := h.getJSON("/__mock__/journal", &entries); err != nil {
		t.Logf("  journal: <unavailable: %v> raw=%q", err, raw)
		return
	}
	if len(entries) == 0 {
		t.Logf("  journal: EMPTY (server saw no frames in either direction)")
		return
	}
	t.Logf("  journal: %d frame(s) in arrival order:", len(entries))
	for i, e := range entries {
		desc := e.Method
		if e.Direction == "send" && e.Method == "signalwire.event" {
			desc = "signalwire.event:" + e.EventType()
		}
		t.Logf("    [%d] t=%.3f %-4s %-28s sess=%s conn=%s id=%s",
			i, e.Timestamp, e.Direction, desc,
			valOrDefault(e.SessionID, "-"), valOrDefault(e.ConnectionID, "-"),
			valOrDefault(e.RequestID, "-"))
	}
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
// an adjacent “the shared test harness package's __init__.py“.
// The adjacency contract is "the shared test harness lives next to signalwire-go in
// ~/src/", so a fresh clone of either repo can find the mock harness
// with no prior pip install. Returns the absolute path to the
// directory containing the Python package (i.e. the path that should
// be added to PYTHONPATH so that “python -m <name>“ resolves), or
// "" when no adjacent test harness is reachable.
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
	defer func() { _ = resp.Body.Close() }()
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
	shared := ensureServer(t)
	if shared == nil {
		return nil, nil
	}

	// Point the SDK at our mock. The relay client honors
	// SIGNALWIRE_RELAY_HOST + SIGNALWIRE_RELAY_SCHEME for exactly this
	// override scenario.
	t.Setenv("SIGNALWIRE_RELAY_HOST", shared.RelayHost())
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

	// Return a per-test harness view scoped to THIS client's session id (the
	// server-assigned `sessionid` from the connect handshake), so the test's
	// journal reads/resets/pushes see only its own frames — making the shared
	// singleton mock safe under parallel (t.Parallel) execution. No global
	// reset is needed: a brand-new session starts with an empty (scoped)
	// journal. The cleanup reset is scoped to this session too, so it never
	// races a concurrent test's state. Mirrors the TS newRelayClient().
	h := shared.scoped(client.SessionID())
	t.Cleanup(func() { h.Reset(t) })
	return client, h
}

// NewClientOnly returns a fresh *relay.Client connected to the running mock,
// with no contexts subscribed, plus a Harness view scoped to THAT client's
// session id. Useful for tests that drive multiple clients in one test (e.g.
// reconnect-with-protocol): each client gets its own session, and the returned
// harness reads/pushes only that client's frames, so the two clients don't
// step on each other and the test is parallel-safe.
//
// The passed-in `h` is only used for its server address; the returned harness
// is a fresh scoped copy (the original `h` keeps its own scope). t.Cleanup is
// registered to disconnect this client and reset its session when the test
// ends. Mirrors the TS pattern of re-scoping a harness via sessionIdOf(client).
func NewClientOnly(t *testing.T, h *Harness, opts ...relay.ClientOption) (*relay.Client, *Harness) {
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
	scoped := h.scoped(client.SessionID())
	t.Cleanup(func() {
		client.Stop()
		scoped.Reset(t)
	})
	return client, scoped
}
