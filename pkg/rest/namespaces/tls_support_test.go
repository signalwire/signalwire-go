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
	"bytes"
	"crypto/tls"
	"crypto/x509"
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
	"testing"
	"time"
)

// tlsSyncBuffer is a goroutine-safe bytes.Buffer: the spawned mock writes to it
// from the os/exec writer goroutine while the test reads it on the readiness
// timeout. Without the mutex that is a data race (caught by `go test -race`).
type tlsSyncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *tlsSyncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *tlsSyncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// trustTestCA wires the shared throwaway CA into Go's system cert pool by
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
	caPath := filepath.Join(dir, "ca.crt")
	// SIGNALWIRE_REST_CA_FILE makes the SDK's REST client trust the test CA via
	// an explicit RootCAs pool — works on every OS, including macOS where Go's
	// system cert pool ignores SSL_CERT_FILE (Darwin delegates to
	// Security.framework). SSL_CERT_FILE is still set for any Linux consumer
	// that relies on the system pool, but the SDK no longer depends on it.
	t.Setenv("SIGNALWIRE_REST_CA_FILE", caPath)
	t.Setenv("SSL_CERT_FILE", caPath)
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

// testCAPool builds an x509 pool from the test CA (certs/ca.crt), for the
// in-test https:// health-probe client. Explicit RootCAs work on every OS,
// unlike SSL_CERT_FILE which Go's system pool ignores on macOS.
func testCAPool(t *testing.T) *x509.CertPool {
	t.Helper()
	dir := findTLSCertsDir()
	if dir == "" {
		t.Skip("tls: porting-sdk/test_harness/tls not found adjacent to repo")
	}
	pem, err := os.ReadFile(filepath.Join(dir, "ca.crt"))
	if err != nil {
		t.Fatalf("read test CA: %v", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		t.Fatal("failed to load test CA into pool")
	}
	return pool
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
	// The journal control plane is a separate HTTPS request to the mock sidecar;
	// under CI load its TLS session can be dropped mid-handshake (a transient
	// `EOF`/connection reset), which a single-shot GET would surface as a hard
	// test failure. Retry a few times on a transport-level error before giving
	// up — the journal is idempotent, so a retry is safe. (A non-empty successful
	// response, or a decode/assert failure, returns/fails immediately.)
	var body []byte
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		resp, err := m.client.Get(m.baseURL + "/__mock__/journal")
		if err != nil {
			lastErr = err
			time.Sleep(100 * time.Millisecond)
			continue
		}
		body, _ = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		lastErr = nil
		break
	}
	if lastErr != nil {
		t.Fatalf("tls mock_signalwire journal GET (after retries): %v", lastErr)
	}
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
// dedicated port, injecting the shared test harness's mock_signalwire package into
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
	// Capture the mock's stdout+stderr so that if it dies on startup (e.g. its
	// starlette/uvicorn/pyyaml deps aren't installed in this job), the readiness
	// timeout below can surface *why* instead of a bare "not ready". stdin is
	// detached to /dev/null.
	var mockOut tlsSyncBuffer
	cmd.Stdout, cmd.Stderr = &mockOut, &mockOut
	if devnull, _ := os.OpenFile(os.DevNull, os.O_RDWR, 0); devnull != nil {
		cmd.Stdin = devnull
	}
	// Run the mock in its own process group (Unix) so signals to the test
	// binary don't cascade to it. Build-constrained (Setpgid is Unix-only; it
	// does not exist on Windows). See setprocessgroup_*_test.go.
	tlsSetProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		t.Skipf("tls: spawn mock_signalwire --tls: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		go func() { _ = cmd.Wait() }()
	})

	// The health probe hits the mock over https://, so it needs to trust the
	// test CA. Use an explicit RootCAs pool built from ca.crt rather than
	// SSL_CERT_FILE — the latter is ignored by Go's system pool on macOS, which
	// is exactly what made this probe time out into a bogus "not ready" there.
	client := &http.Client{
		Timeout:   3 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: testCAPool(t)}},
	}
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
	t.Fatalf("tls: mock_signalwire --tls not ready on port %d after 30s; mock output:\n%s", port, mockOut.String())
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
