// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Command wait-liveness-dump is the Go port's WAIT-LIVENESS dump program for the
// cross-port behavioral differ (porting-sdk/scripts/diff_port_wait_liveness.py).
//
// The differ runs the wait_liveness_corpus against signalwire-python to build the
// golden LIVENESS classification, then runs THIS program (which embeds the same
// corpus) and structurally compares our per-case classification. The artifact is a
// CLASSIFICATION (not raw ms), so the golden is deterministic while the timing that
// produces it is real and unfakeable: a wait() that is a no-op returns at t~=0
// (blocked_until_event=false -> RED); a wait() that hangs blows the deadline
// (timed_out=true -> RED); a correct wait() blocks until the deferred completing
// event arrives, then returns with the finished state (the golden -> GREEN).
//
// Unlike wire-relay-dump (which RECORDS the send-side frame against a tiny
// in-process mock with no event delivery), this gate MUST exercise real liveness —
// so we drive the SDK against a REAL porting-sdk mock_relay server and arm the
// completing event as a DEFERRED (delay_ms) scenario. That delivers the event
// through the SAME socket-read -> event-dispatch path the real server drives; a
// wait() that never pumps the read loop cannot observe it. This is the exact
// mechanism pkg/relay/actions_mock_test.go (TestRelay_PlayResolvesOnFinishedEvent)
// uses, but driven without a *testing.T: we speak the mock's HTTP control plane
// (/__mock__/health, /__mock__/inbound_call, /__mock__/scenarios/<method>,
// /__mock__/journal/reset, /__mock__/scenarios/reset) directly with net/http.
//
// Protocol: stdout = ONE JSON object mapping case_id -> classification. Only stdout
// carries JSON; all setup/logging/diagnostics go to stderr. The corpus cases are
// embedded here (mirroring porting-sdk/scripts/wait_liveness_corpus.py) exactly, as
// wire-relay-dump embeds the wire_relay corpus — the differ does not feed the corpus
// in; it keys our output by case-id against the python oracle.
//
// Ports: honors MOCK_RELAY_PORT / MOCK_RELAY_HTTP_PORT when set (so a gate's
// pre-spawned mock is the one we hit); otherwise picks two INDEPENDENT free
// loopback ports, spawns `python -m mock_relay`, waits healthy (fails LOUD with the
// spawn log if it never comes up — never hangs), and KILLS it on exit.
//
// Run from the signalwire-go repo root:
//
//	go run ./cmd/wait-liveness-dump
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/signalwire/signalwire-go/v3/pkg/relay"
)

// ---------------------------------------------------------------------------
// Classification tolerances — MUST match porting-sdk/scripts/diff_port_wait_liveness.py.
// ---------------------------------------------------------------------------
const (
	deadlineS  = 5.0 // a wait() outliving this is HUNG (timed_out)
	blockTolMS = 40  // how much earlier than delay_ms a return may be and still count "blocked"
)

// The deferred-event delay — MUST match wait_liveness_corpus.DELAY_MS.
const delayMS = 150

const cid = "ctl-live-1"

// startupTimeout caps the wait for the mock-relay subprocess to answer
// /__mock__/health. `python -m mock_relay` includes import + schema loading which
// can stretch to a few seconds on a cold cache.
const startupTimeout = 30 * time.Second

// actionSpec is the verb + inputs + terminal event for a single Action-returning
// verb (one wait). A plain case is one spec; a nested case has an inner spec too.
type actionSpec struct {
	Verb     string
	Media    []map[string]any // play only
	Audio    map[string]any   // record only
	Terminal terminalEvent
}

// livenessCase mirrors one entry of wait_liveness_corpus.CORPUS.
type livenessCase struct {
	ID      string
	DelayMS int
	Spec    actionSpec
	// Nested, when non-nil, is the INNER action started + waited from inside the
	// outer action's completion (the re-entrant wait-inside-a-callback pattern).
	Nested *actionSpec
}

type terminalEvent struct {
	State string
}

// corpus mirrors porting-sdk/scripts/wait_liveness_corpus.py CORPUS exactly. Two
// distinct action types (play, record) so a port can't hardcode one surface.
var playMedia = []map[string]any{{"type": "audio", "params": map[string]any{"url": "https://x/a.mp3"}}}

var corpus = []livenessCase{
	// play -> PlayAction: wait() blocks until calling.call.play {state:finished}.
	{
		ID:      "live_play_wait",
		DelayMS: delayMS,
		Spec:    actionSpec{Verb: "play", Media: playMedia, Terminal: terminalEvent{State: "finished"}},
	},
	// record -> RecordAction: a SECOND action type so a port can't hardcode one surface.
	{
		ID:      "live_record_wait",
		DelayMS: delayMS,
		Spec:    actionSpec{Verb: "record", Audio: map[string]any{"format": "mp3"}, Terminal: terminalEvent{State: "finished"}},
	},
	// NESTED wait: an inner action started + waited from inside the outer action's
	// completion (re-entrant wait-inside-a-callback). Fold: timed_out if EITHER
	// hung, blocked_until_event only if BOTH blocked, completed_state from the
	// inner (last) completion.
	{
		ID:      "live_nested_wait",
		DelayMS: delayMS,
		Spec:    actionSpec{Verb: "play", Media: playMedia, Terminal: terminalEvent{State: "finished"}},
		Nested:  &actionSpec{Verb: "record", Audio: map[string]any{"format": "mp3"}, Terminal: terminalEvent{State: "finished"}},
	},
}

// verbMethod maps a corpus verb to the RELAY method whose scenario carries its
// terminal event (the key the mock arms under).
var verbMethod = map[string]string{
	"play":   "calling.play",
	"record": "calling.record",
}

// classification is the comparable artifact the differ byte-compares against the
// python golden. Field JSON tags MUST match classify_liveness() in the differ.
type classification struct {
	BlockedUntilEvent  bool   `json:"blocked_until_event"`
	ReturnedAfterEvent bool   `json:"returned_after_event"`
	CompletedState     string `json:"completed_state"`
	TimedOut           bool   `json:"timed_out"`
}

// classify derives the deterministic liveness classification from the measured
// instants. Mirrors classify_liveness() in the differ byte-for-byte.
func classify(delay int, tWaitStart, tReturn time.Time, completedState string, timedOut, returned bool) classification {
	if timedOut || !returned {
		return classification{
			BlockedUntilEvent:  false,
			ReturnedAfterEvent: false,
			CompletedState:     "",
			TimedOut:           true,
		}
	}
	elapsedMS := float64(tReturn.Sub(tWaitStart).Nanoseconds()) / 1e6
	blocked := elapsedMS >= float64(delay-blockTolMS)
	return classification{
		BlockedUntilEvent:  blocked,
		ReturnedAfterEvent: true,
		CompletedState:     completedState,
		TimedOut:           false,
	}
}

func main() {
	os.Exit(run())
}

func run() int {
	server, err := ensureServer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "wait-liveness-dump: %v\n", err)
		return 1
	}
	defer server.Close()

	out := map[string]classification{}
	for _, kase := range corpus {
		cls, err := runCase(server, kase)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wait-liveness-dump: case %s: %v\n", kase.ID, err)
			return 1
		}
		out[kase.ID] = cls
	}

	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "wait-liveness-dump: encode output: %v\n", err)
		return 1
	}
	return 0
}

// waitResult holds the measured instants of a single Action.Wait — the inputs to
// classify(). Mirrors the (t_wait_start, t_return, completed_state, timed_out)
// tuple _drive_action returns in the differ.
type waitResult struct {
	tWaitStart     time.Time
	tReturn        time.Time
	completedState string
	timedOut       bool
	returned       bool
}

// runCase drives ONE liveness case against the real mock and returns its
// classification. A FRESH client/session per case so a scoped scenario/journal
// never crosses between cases (mirrors the per-test client isolation in
// actions_mock_test.go).
func runCase(server *mockServer, kase livenessCase) (classification, error) {
	// Point the SDK at our mock (the same host-override env vars mocktest uses).
	_ = os.Setenv("SIGNALWIRE_RELAY_HOST", server.relayHost())
	_ = os.Setenv("SIGNALWIRE_RELAY_SCHEME", "ws")

	client := relay.NewRelayClient(
		relay.WithProject("test_proj"),
		relay.WithToken("test_tok"),
		relay.WithContexts("default"),
	)
	if err := client.Connect(); err != nil {
		return classification{}, fmt.Errorf("relay Connect: %w", err)
	}
	defer client.Stop()
	if err := client.Authenticate(); err != nil {
		return classification{}, fmt.Errorf("relay Authenticate: %w", err)
	}
	client.StartReadLoop()
	if err := client.SubscribeContexts(); err != nil {
		return classification{}, fmt.Errorf("relay SubscribeContexts: %w", err)
	}

	// Scope every control-plane call to THIS client's session so the inbound-call
	// sequence + armed scenario are delivered only to this client's read loop
	// (mirrors mocktest's scoped harness). A brand-new session starts with an
	// empty scoped journal, so no reset is needed up front.
	sessionID := client.SessionID()
	defer server.resetSession(sessionID) // best-effort per-case cleanup

	if kase.Nested == nil {
		outer, err := server.driveAction(client, sessionID, kase.DelayMS, kase.Spec, kase.ID+"-outer")
		if err != nil {
			return classification{}, err
		}
		return classify(kase.DelayMS, outer.tWaitStart, outer.tReturn, outer.completedState, outer.timedOut, outer.returned), nil
	}

	// Nested: drive the outer action; only if it did NOT hang, drive the inner
	// action (the re-entrant wait-inside-a-callback pattern — in Go the read loop
	// is a background goroutine, so the inner wait pumps the same live receive path
	// while the outer's completion is still being handled). FOLD the two: timed_out
	// if EITHER hung, blocked only if BOTH blocked, completed_state from the inner.
	outer, err := server.driveAction(client, sessionID, kase.DelayMS, kase.Spec, kase.ID+"-outer")
	if err != nil {
		return classification{}, err
	}
	outerCls := classify(kase.DelayMS, outer.tWaitStart, outer.tReturn, outer.completedState, outer.timedOut, outer.returned)

	var innerCls classification
	if outerCls.TimedOut {
		innerCls = classification{TimedOut: true}
	} else {
		inner, err := server.driveAction(client, sessionID, kase.DelayMS, *kase.Nested, kase.ID+"-inner")
		if err != nil {
			return classification{}, err
		}
		innerCls = classify(kase.DelayMS, inner.tWaitStart, inner.tReturn, inner.completedState, inner.timedOut, inner.returned)
	}

	if outerCls.TimedOut || innerCls.TimedOut {
		return classification{TimedOut: true}, nil
	}
	return classification{
		BlockedUntilEvent:  outerCls.BlockedUntilEvent && innerCls.BlockedUntilEvent,
		ReturnedAfterEvent: true,
		CompletedState:     innerCls.CompletedState,
		TimedOut:           false,
	}, nil
}

// driveAction establishes an answered inbound call, arms the spec's deferred
// completing event, starts the Action-returning verb, and waits — returning the
// measured instants. Mirrors _drive_action in the differ. callTag distinguishes
// the inbound call_id so the outer and inner actions of a nested case never
// collide on the same session.
func (s *mockServer) driveAction(client *relay.Client, sessionID string, delay int, spec actionSpec, callTag string) (waitResult, error) {
	method := verbMethod[spec.Verb]
	if method == "" {
		return waitResult{}, fmt.Errorf("unsupported verb %q", spec.Verb)
	}

	call, err := s.answeredInboundCall(client, sessionID, "live-"+callTag)
	if err != nil {
		return waitResult{}, err
	}

	// Arm the DEFERRED completing event: the mock emits {state: terminal.state}
	// for this method after delay_ms, delivered through the SDK's real socket-read
	// event-dispatch path. A no-op / non-pumping wait() cannot observe it.
	if err := s.armMethod(sessionID, method, []map[string]any{
		{"emit": map[string]any{"state": spec.Terminal.State}, "delay_ms": delay},
	}); err != nil {
		return waitResult{}, err
	}

	action := startAction(call, spec)
	if action == nil {
		return waitResult{}, fmt.Errorf("startAction returned nil for verb %q", spec.Verb)
	}

	res := waitResult{tWaitStart: time.Now()}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(deadlineS*float64(time.Second)))
	defer cancel()
	event, waitErr := action.Wait(ctx)
	switch {
	case waitErr != nil:
		// ctx deadline (or cancel) => hung wait.
		res.timedOut = true
	case event == nil:
		// A wait() that returns nil (no event) within the deadline is a no-op /
		// hung equivalent: treat as timed_out so the classification is honest.
		res.timedOut = true
	default:
		res.tReturn = time.Now()
		res.returned = true
		res.completedState = event.GetString("state")
	}
	return res, nil
}

// startAction starts the Action-returning verb for a spec against an answered call.
func startAction(call *relay.Call, spec actionSpec) *relay.Action {
	switch spec.Verb {
	case "play":
		return call.Play(spec.Media, relay.WithPlayControlID(cid)).Action
	case "record":
		return call.Record(relay.WithRecordAudio(spec.Audio), relay.WithRecordControlID(cid)).Action
	default:
		return nil
	}
}

// ---------------------------------------------------------------------------
// Mock server lifecycle + HTTP control plane (a testing.T-free port of the parts
// of pkg/relay/internal/mocktest/mocktest.go this cmd needs).
// ---------------------------------------------------------------------------

type mockServer struct {
	httpURL    string
	wsPort     int
	httpClient *http.Client
	cmd        *exec.Cmd // nil when we reused a pre-running server (env-provided)
}

func (s *mockServer) relayHost() string {
	return fmt.Sprintf("127.0.0.1:%d", s.wsPort)
}

// Close kills the spawned mock (if we spawned one). Idempotent; a nil cmd (reused
// pre-running server) is a no-op.
func (s *mockServer) Close() {
	if s == nil || s.cmd == nil || s.cmd.Process == nil {
		return
	}
	_ = s.cmd.Process.Kill()
	_, _ = s.cmd.Process.Wait()
}

// ensureServer resolves the mock ports (env override or free-picked), reuses a
// pre-running server when one answers /__mock__/health, else spawns
// `python -m mock_relay` and waits for health. Fails LOUD (never hangs) if the
// mock never becomes ready.
func ensureServer() (*mockServer, error) {
	httpClient := &http.Client{Timeout: 5 * time.Second}

	envWS := os.Getenv("MOCK_RELAY_PORT")
	envHTTP := os.Getenv("MOCK_RELAY_HTTP_PORT")

	// Case 1: both ports provided by the environment (a gate pre-spawned the mock).
	if envWS != "" {
		wsPort, err := strconv.Atoi(envWS)
		if err != nil || wsPort <= 0 {
			return nil, fmt.Errorf("invalid MOCK_RELAY_PORT %q", envWS)
		}
		httpPort := wsPort + 1000
		if envHTTP != "" {
			if p, e := strconv.Atoi(envHTTP); e == nil && p > 0 {
				httpPort = p
			}
		}
		httpURL := fmt.Sprintf("http://127.0.0.1:%d", httpPort)
		if probeHealth(httpClient, httpURL) {
			return &mockServer{httpURL: httpURL, wsPort: wsPort, httpClient: httpClient}, nil
		}
		// Env said to use this port but nothing is there — spawn on it (honor the port).
		return spawnServer(httpClient, wsPort, httpPort)
	}

	// Case 2: pick two INDEPENDENT free loopback ports and spawn.
	wsPort, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("pick free ws port: %w", err)
	}
	httpPort, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("pick free http port: %w", err)
	}
	// Guard against the (astronomically unlikely) collision of the two ephemeral picks.
	for httpPort == wsPort {
		if httpPort, err = freePort(); err != nil {
			return nil, fmt.Errorf("pick free http port: %w", err)
		}
	}
	return spawnServer(httpClient, wsPort, httpPort)
}

// freePort binds a loopback ephemeral port, reads the OS-assigned port, releases
// it, and returns it. (There is an inherent TOCTOU window between release and the
// child bind, but the window is tiny and the child fails loud on bind error.)
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = l.Close() }()
	addr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected listener address type %T", l.Addr())
	}
	return addr.Port, nil
}

// spawnServer runs `python -m mock_relay` on the given ports, injecting the
// adjacent porting-sdk/test_harness/mock_relay onto PYTHONPATH so it resolves with
// no prior pip install. Waits for /__mock__/health; on timeout it kills the child,
// dumps its captured log to stderr, and returns an error (fail LOUD, never hang).
func spawnServer(httpClient *http.Client, wsPort, httpPort int) (*mockServer, error) {
	httpURL := fmt.Sprintf("http://127.0.0.1:%d", httpPort)

	cmd := exec.Command("python", "-m", "mock_relay",
		"--host", "127.0.0.1",
		"--ws-port", strconv.Itoa(wsPort),
		"--http-port", strconv.Itoa(httpPort),
		"--log-level", "error",
	)
	env := os.Environ()
	if pkgDir := discoverPortingSDKPackage("mock_relay"); pkgDir != "" {
		newPP := pkgDir
		if existing := os.Getenv("PYTHONPATH"); existing != "" {
			newPP = pkgDir + string(os.PathListSeparator) + existing
		}
		env = upsertEnv(env, "PYTHONPATH", newPP)
	}
	env = upsertEnv(env, "MOCK_RELAY_PORT", strconv.Itoa(wsPort))
	cmd.Env = env

	// Capture the child's output so a startup failure is self-diagnosing.
	var logBuf bytes.Buffer
	cmd.Stdout = &logBuf
	cmd.Stderr = &logBuf
	setProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("spawn `python -m mock_relay`: %w (set MOCK_RELAY_PORT to use a pre-running instance)", err)
	}

	deadline := time.Now().Add(startupTimeout)
	for time.Now().Before(deadline) {
		// Detect the child dying mid-startup so we fail immediately instead of
		// polling health for the full deadline.
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			break
		}
		if probeHealth(httpClient, httpURL) {
			return &mockServer{httpURL: httpURL, wsPort: wsPort, httpClient: httpClient, cmd: cmd}, nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()
	return nil, fmt.Errorf(
		"`python -m mock_relay` did not become ready within %s on ws_port=%d http_port=%d\n--- mock_relay log ---\n%s",
		startupTimeout, wsPort, httpPort, logBuf.String(),
	)
}

// upsertEnv sets key=val in a KEY=VALUE slice, replacing an existing entry.
func upsertEnv(env []string, key, val string) []string {
	prefix := key + "="
	for i, kv := range env {
		if len(kv) >= len(prefix) && kv[:len(prefix)] == prefix {
			env[i] = prefix + val
			return env
		}
	}
	return append(env, prefix+val)
}

// discoverPortingSDKPackage walks up from this source file to an adjacent
// porting-sdk/test_harness/<name> (the same adjacency contract mocktest uses).
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

// probeHealth returns true when /__mock__/health responds 200 with a payload
// containing "schemas_loaded".
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
// HTTP control-plane operations (session-scoped, testing.T-free).
// ---------------------------------------------------------------------------

func (s *mockServer) sessionQuery(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	return "?session_id=" + url.QueryEscape(sessionID)
}

// post POSTs a JSON body (or nil) to path and returns the response body,
// erroring on transport failure or a >=400 status.
func (s *mockServer) post(path string, body any) error {
	var rdr io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body for %s: %w", path, err)
		}
		rdr = bytes.NewReader(buf)
	}
	req, err := http.NewRequest("POST", s.httpURL+path, rdr)
	if err != nil {
		return fmt.Errorf("build %s: %w", path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s -> %d: %s", path, resp.StatusCode, respBody)
	}
	return nil
}

// armMethod queues a scripted post-RPC event for `method`, scoped to sessionID.
func (s *mockServer) armMethod(sessionID, method string, events []map[string]any) error {
	return s.post("/__mock__/scenarios/"+method+s.sessionQuery(sessionID), events)
}

// resetSession clears this session's journal + scenarios (best-effort).
func (s *mockServer) resetSession(sessionID string) {
	q := s.sessionQuery(sessionID)
	_ = s.post("/__mock__/journal/reset"+q, nil)
	_ = s.post("/__mock__/scenarios/reset"+q, nil)
}

// answeredInboundCall establishes an answered inbound call and returns the
// captured *relay.Call (mirrors answeredInboundCall in actions_mock_test.go). The
// inbound-call sequence is delivered only to this client's session.
func (s *mockServer) answeredInboundCall(client *relay.Client, sessionID, callID string) (*relay.Call, error) {
	captured := make(chan *relay.Call, 1)
	client.OnCall(func(c *relay.Call) {
		_ = c.Answer()
		select {
		case captured <- c:
		default:
		}
	})

	body := map[string]any{
		"from_number": "+15551234567",
		"to_number":   "+15559876543",
		"context":     "default",
		"call_id":     callID,
		"auto_states": []string{"created"},
		"delay_ms":    50,
	}
	if sessionID != "" {
		body["session_id"] = sessionID
	}
	if err := s.post("/__mock__/inbound_call", body); err != nil {
		return nil, fmt.Errorf("inbound_call: %w", err)
	}

	select {
	case call := <-captured:
		return call, nil
	case <-time.After(time.Duration(deadlineS * float64(time.Second))):
		return nil, fmt.Errorf("inbound-call handler did not fire within %.0fs", deadlineS)
	}
}
