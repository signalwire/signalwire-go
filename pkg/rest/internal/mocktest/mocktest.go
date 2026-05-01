// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Package mocktest is the Go test helper for the porting-sdk mock_signalwire
// HTTP server. It mirrors the Python conftest fixtures (signalwire_client +
// mock) so unit tests can exercise the real SDK code path against a real
// HTTP server backed by SignalWire's 13 OpenAPI specs.
//
// The mock server's lifetime is per-process: the first New call probes
// http://127.0.0.1:<port>/__mock__/health and either confirms a running
// server or starts one as a subprocess. Each test gets a freshly reset
// journal/scenario state via t.Cleanup. Tests do not share journal entries.
//
// The default port is 8765 (matching the Python harness default). Override
// with MOCK_SIGNALWIRE_PORT in the test environment if a different mock
// instance is already running.
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

	rest "github.com/signalwire/signalwire-go/pkg/rest"
)

// setProcessGroup configures the spawned mock-server process to run in its
// own process group on Unix (Setpgid: true). This prevents signals to the
// test binary from cascading to the child and keeps the child detached so
// the testing framework's pipe-drain logic doesn't block on it.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// JournalEntry mirrors mock_signalwire.journal.JournalEntry over the wire.
//
// Body is decoded as a generic interface{}: the mock returns a JSON object
// for application/json bodies and a string for everything else, so callers
// will typically need a type assertion via journal.BodyMap() or similar.
type JournalEntry struct {
	Timestamp      float64                  `json:"timestamp"`
	Method         string                   `json:"method"`
	Path           string                   `json:"path"`
	QueryParams    map[string][]string      `json:"query_params"`
	Headers        map[string]string        `json:"headers"`
	Body           any                      `json:"body"`
	MatchedRoute   *string                  `json:"matched_route"`
	ResponseStatus *int                     `json:"response_status"`
}

// BodyMap returns the request body coerced to map[string]any. It returns
// (nil, false) if the body is not a JSON object — typical for empty bodies
// or non-JSON content.
func (e JournalEntry) BodyMap() (map[string]any, bool) {
	m, ok := e.Body.(map[string]any)
	return m, ok
}

// Harness wraps the running mock server. It exposes journal accessors,
// a helper to push scenario overrides, and a reset hook tests register via
// t.Cleanup.
type Harness struct {
	URL  string
	Port int

	httpClient *http.Client
}

// Last returns the most recent journal entry. It fails the test if the
// journal is empty — every test that calls a mock-backed SDK method should
// produce at least one entry.
func (h *Harness) Last(t *testing.T) JournalEntry {
	t.Helper()
	entries := h.Journal(t)
	if len(entries) == 0 {
		t.Fatal("mocktest: journal is empty - SDK call did not reach the mock server")
	}
	return entries[len(entries)-1]
}

// Journal returns every entry recorded since the last reset, in arrival order.
func (h *Harness) Journal(t *testing.T) []JournalEntry {
	t.Helper()
	resp, err := h.httpClient.Get(h.URL + "/__mock__/journal")
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

// Reset clears journal + scenarios on the mock server. Tests do not call
// this directly — New registers it as a t.Cleanup hook.
func (h *Harness) Reset(t *testing.T) {
	t.Helper()
	post := func(path string) {
		req, err := http.NewRequest("POST", h.URL+path, nil)
		if err != nil {
			t.Fatalf("mocktest: build %s: %v", path, err)
		}
		resp, err := h.httpClient.Do(req)
		if err != nil {
			t.Fatalf("mocktest: %s: %v", path, err)
		}
		_ = resp.Body.Close()
	}
	post("/__mock__/journal/reset")
	post("/__mock__/scenarios/reset")
}

// PushScenario stages a one-shot response override for the route identified
// by endpointID. The status + body returned here will be served the next
// time the route is hit; subsequent hits fall back to spec synthesis.
//
// endpointID is the Spectral-style "OperationId" from the OpenAPI spec —
// see /__mock__/scenarios for the active list.
func (h *Harness) PushScenario(t *testing.T, endpointID string, status int, body any) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"status": status, "response": body})
	if err != nil {
		t.Fatalf("mocktest: marshal scenario: %v", err)
	}
	resp, err := h.httpClient.Post(
		h.URL+"/__mock__/scenarios/"+endpointID,
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("mocktest: push scenario: %v", err)
	}
	_ = resp.Body.Close()
}

// ---------------------------------------------------------------------------
// Server lifecycle
// ---------------------------------------------------------------------------

// serverState holds the singleton harness so subsequent New calls reuse the
// same backing server.
type serverState struct {
	once    sync.Once
	harness *Harness
	cmd     *exec.Cmd
	startErr error
}

var state serverState

// defaultPort matches mock_signalwire.server.DEFAULT_PORT.
const defaultPort = 8765

// startupTimeout caps the time we wait for an externally-launched server to
// answer /__mock__/health. The Python in-process harness boots in ~1s, but
// `python -m mock_signalwire` includes module import + spec loading which
// can stretch to ~5s on a cold cache.
const startupTimeout = 30 * time.Second

// discoverPortingSDKPackage walks up from this source file looking for an
// adjacent ``porting-sdk/test_harness/<name>/<name>/__init__.py``. The
// adjacency contract is "porting-sdk lives next to signalwire-go in ~/src/",
// so a fresh clone of either repo can find the mock harness with no prior
// pip install. Returns the absolute path to the directory containing the
// Python package (i.e. the path that should be added to PYTHONPATH so that
// ``python -m <name>`` resolves), or "" when no adjacent porting-sdk is
// reachable.
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

// resolvePort returns the configured mock port — either MOCK_SIGNALWIRE_PORT
// or the default 8765.
func resolvePort() int {
	if raw := os.Getenv("MOCK_SIGNALWIRE_PORT"); raw != "" {
		if p, err := strconv.Atoi(raw); err == nil && p > 0 {
			return p
		}
	}
	return defaultPort
}

// ensureServer probes the mock server's health endpoint and starts a
// subprocess if nothing's listening. The subprocess runs `python -m
// mock_signalwire --port <port> --log-level error` and is left running
// until process exit (test runs are short and the OS cleans up).
//
// Subsequent calls reuse the singleton; only the first New call pays the
// startup cost.
func ensureServer(t *testing.T) *Harness {
	t.Helper()
	state.once.Do(func() {
		port := resolvePort()
		client := &http.Client{Timeout: 2 * time.Second}
		url := fmt.Sprintf("http://127.0.0.1:%d", port)

		// Probe — if a server is already running we reuse it.
		if probeHealth(client, url) {
			state.harness = &Harness{URL: url, Port: port, httpClient: client}
			return
		}

		// Spawn a subprocess. We deliberately detach stdout/stderr (point
		// them at /dev/null) because Go's testing runner waits on the
		// child's pipes before exiting, which would hang the test process
		// for the full WaitDelay (60s by default) when the subprocess
		// stays alive across the test binary lifetime.
		cmd := exec.Command("python", "-m", "mock_signalwire",
			"--host", "127.0.0.1",
			"--port", strconv.Itoa(port),
			"--log-level", "error",
		)
		// Try to inject porting-sdk/test_harness/mock_signalwire/ into
		// PYTHONPATH so `python -m mock_signalwire` resolves without a
		// prior `pip install -e ...`. Adjacency contract: porting-sdk
		// next to signalwire-go in ~/src/. When the walk fails (e.g.
		// because porting-sdk is not adjacent), we still spawn — the
		// child will fall back to whatever is on the system Python's
		// sys.path, and surface a clear "module not found" error from
		// the spawn-readiness probe if neither mode is available.
		if pkgDir := discoverPortingSDKPackage("mock_signalwire"); pkgDir != "" {
			env := os.Environ()
			existingPP := os.Getenv("PYTHONPATH")
			newPP := pkgDir
			if existingPP != "" {
				newPP = pkgDir + string(os.PathListSeparator) + existingPP
			}
			// Replace any existing PYTHONPATH entry, otherwise append.
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
			cmd.Env = env
		}
		devnull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
		if err == nil {
			cmd.Stdout = devnull
			cmd.Stderr = devnull
			cmd.Stdin = devnull
		}
		// Put the child in its own process group so a Ctrl-C to the test
		// binary doesn't propagate, and so we can clean up via Kill on
		// exit without the parent waiting on the pipe goroutines.
		setProcessGroup(cmd)
		if err := cmd.Start(); err != nil {
			state.startErr = fmt.Errorf("mocktest: failed to spawn `python -m mock_signalwire`: %w (set MOCK_SIGNALWIRE_PORT to use a pre-running instance)", err)
			return
		}

		// Wait for /__mock__/health.
		deadline := time.Now().Add(startupTimeout)
		for time.Now().Before(deadline) {
			if probeHealth(client, url) {
				state.harness = &Harness{URL: url, Port: port, httpClient: client}
				state.cmd = cmd
				// Detach and let the OS clean up on exit; the goroutine
				// reaps the process so we don't leave a zombie.
				go func() { _ = cmd.Wait() }()
				return
			}
			time.Sleep(150 * time.Millisecond)
		}
		_ = cmd.Process.Kill()
		state.startErr = fmt.Errorf("mocktest: `python -m mock_signalwire` did not become ready within %s on port %d (clone porting-sdk next to signalwire-go so tests can find porting-sdk/test_harness/mock_signalwire/, or pip install the mock_signalwire package)", startupTimeout, port)
	})
	if state.startErr != nil {
		t.Skipf("mocktest: mock_signalwire unavailable: %v", state.startErr)
		return nil
	}
	return state.harness
}

// probeHealth returns true when the mock server's /__mock__/health responds
// with 200 OK and a payload containing "specs_loaded".
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
	_, ok := payload["specs_loaded"]
	return ok
}

// ---------------------------------------------------------------------------
// Test entry point
// ---------------------------------------------------------------------------

// New returns (client, harness) for a single test. The client is a real
// rest.RestClient pointed at the local mock server with project=test_proj
// and token=test_tok — matching the Python signalwire_client fixture.
//
// The mock's journal + scenarios are reset before the test runs, and the
// reset is repeated via t.Cleanup so accidental leftover state from a panic
// doesn't leak into the next test.
func New(t *testing.T) (*rest.RestClient, *Harness) {
	t.Helper()
	h := ensureServer(t)
	if h == nil {
		// ensureServer already called t.Skipf.
		return nil, nil
	}
	h.Reset(t)
	t.Cleanup(func() { h.Reset(t) })

	// Build a real client with throwaway credentials. The mock accepts
	// any non-empty Basic Auth header.
	client, err := rest.NewRestClient("test_proj", "test_tok", fmt.Sprintf("127.0.0.1:%d", h.Port))
	if err != nil {
		t.Fatalf("mocktest: NewRestClient: %v", err)
	}
	// Repoint the underlying HttpClient at http:// (the constructor builds
	// https:// + space). SetBaseURL exists for exactly this purpose.
	client.SetBaseURL(h.URL)
	return client, h
}
