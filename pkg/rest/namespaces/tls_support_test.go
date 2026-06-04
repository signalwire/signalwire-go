// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Shared test-only TLS support for the REST namespaces HTTPS capability test:
// a TestMain that wires CA trust via SSL_CERT_FILE, a gen_certs.sh runner, and
// a spawner for the shared mock_signalwire in --tls (HTTPS) mode on a
// dedicated port. Kept self-contained so the run-ci gates' plain-HTTP shared
// mock on the default port is untouched.
package namespaces_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"testing"
	"time"
)

// trustTestCA wires the porting-sdk throwaway CA into Go's system cert pool by
// setting SSL_CERT_FILE to certs/ca.crt (running the idempotent gen_certs.sh
// first). The REST client's *http.Client uses the default transport (nil
// TLSClientConfig), which consults that pool — so this is the idiomatic,
// no-SDK-change way to trust the CA for HTTPS.
//
// It must run before the first TLS handshake in the process (the system pool
// is loaded once and cached). The TLS capability test is the only TLS user in
// this package — every other mock test uses plain http:// — so calling it at
// the top of the test, before any request, is sufficient. The negative subtest
// supplies an explicit empty pool and is therefore unaffected.
func trustTestCA(t *testing.T) {
	t.Helper()
	dir := findTLSCertsDir()
	if dir == "" {
		t.Skip("tls: porting-sdk/test_harness/tls not found adjacent to repo")
	}
	t.Setenv("SSL_CERT_FILE", filepath.Join(dir, "ca.crt"))
}

func findTLSCertsDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	dir, _ := filepath.Abs(filepath.Dir(file))
	for {
		tlsDir := filepath.Join(filepath.Dir(dir), "porting-sdk", "test_harness", "tls")
		if _, err := os.Stat(filepath.Join(tlsDir, "gen_certs.sh")); err == nil {
			if runGenCerts(tlsDir) != nil {
				return ""
			}
			return filepath.Join(tlsDir, "certs")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func runGenCerts(tlsDir string) error {
	return exec.Command("bash", filepath.Join(tlsDir, "gen_certs.sh")).Run()
}

// tlsMockSignalwire is a single --tls mock_signalwire instance on its own port.
// In TLS mode uvicorn serves the entire app (including the /__mock__/ control
// plane) over HTTPS, so journal reads go over https:// + the trusted CA.
type tlsMockSignalwire struct {
	port    int
	baseURL string // https://127.0.0.1:<port>
	client  *http.Client
}

type tlsJournalEntry struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

func (m *tlsMockSignalwire) lastJournal(t *testing.T) tlsJournalEntry {
	t.Helper()
	resp, err := m.client.Get(m.baseURL + "/__mock__/journal")
	if err != nil {
		t.Fatalf("tls mock_signalwire journal GET: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var entries []tlsJournalEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		t.Fatalf("tls mock_signalwire journal decode: %v (body=%q)", err, body)
	}
	if len(entries) == 0 {
		t.Fatal("tls mock_signalwire journal empty - HTTPS request did not reach the mock")
	}
	return entries[len(entries)-1]
}

// startTLSMockSignalwire spawns `python -m mock_signalwire --tls` on a
// dedicated port, injecting porting-sdk/test_harness/mock_signalwire into
// PYTHONPATH. It waits for the HTTPS /__mock__/health (using an SSL_CERT_FILE-
// trusting client), registers a Kill cleanup, and skips when unavailable.
func startTLSMockSignalwire(t *testing.T) *tlsMockSignalwire {
	t.Helper()
	pkgDir := discoverHarnessPkg("mock_signalwire")
	if pkgDir == "" {
		t.Skip("tls: porting-sdk/test_harness/mock_signalwire not adjacent")
	}
	port := 18766
	baseURL := fmt.Sprintf("https://127.0.0.1:%d", port)

	cmd := exec.Command("python", "-m", "mock_signalwire",
		"--host", "127.0.0.1",
		"--port", strconv.Itoa(port),
		"--tls",
		"--log-level", "error",
	)
	cmd.Env = harnessEnv(pkgDir, map[string]string{"SIGNALWIRE_MOCK_TLS": "1"})
	devnull, _ := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if devnull != nil {
		cmd.Stdout, cmd.Stderr, cmd.Stdin = devnull, devnull, devnull
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Skipf("tls: spawn mock_signalwire --tls: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		go func() { _ = cmd.Wait() }()
	})

	// SSL_CERT_FILE (set in TestMain) makes this default client trust the CA.
	client := &http.Client{Timeout: 3 * time.Second}
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(baseURL + "/__mock__/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return &tlsMockSignalwire{port: port, baseURL: baseURL, client: client}
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	t.Fatalf("tls: mock_signalwire --tls not ready on port %d", port)
	return nil
}

func discoverHarnessPkg(name string) string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	dir, _ := filepath.Abs(filepath.Dir(file))
	for {
		candidate := filepath.Join(filepath.Dir(dir), "porting-sdk", "test_harness", name)
		if info, err := os.Stat(filepath.Join(candidate, name, "__init__.py")); err == nil && !info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func harnessEnv(pkgDir string, extra map[string]string) []string {
	env := os.Environ()
	set := func(key, val string) {
		prefix := key + "="
		for i, kv := range env {
			if len(kv) >= len(prefix) && kv[:len(prefix)] == prefix {
				env[i] = prefix + val
				return
			}
		}
		env = append(env, prefix+val)
	}
	pp := pkgDir
	if existing := os.Getenv("PYTHONPATH"); existing != "" {
		pp = pkgDir + string(os.PathListSeparator) + existing
	}
	set("PYTHONPATH", pp)
	for k, v := range extra {
		set(k, v)
	}
	return env
}
