// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Shared test-only TLS support for the RELAY package's WSS capability test:
// a TestMain that wires CA trust via SSL_CERT_FILE, a gen_certs.sh runner, and
// a spawner for the shared mock_relay in --tls (WSS) mode on dedicated ports.
package relay_test

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
// first). gorilla/websocket's dialer and net/http both consult that pool when
// their TLSClientConfig.RootCAs is nil — which is how the RELAY client builds
// its dialer — so this is the idiomatic, no-SDK-change way to trust the CA.
//
// It must run before the first TLS handshake in the process (the system pool
// is loaded once and cached). The TLS capability test is the only TLS user in
// this package — every other mock test uses plain ws:// / http:// — so calling
// it at the top of the test, before any dial, is sufficient. The negative
// subtest supplies an explicit empty pool and is therefore unaffected.
//
// Skips the test when porting-sdk is not adjacent (matching the mocktest
// adjacency contract).
func trustTestCA(t *testing.T) {
	t.Helper()
	dir := findTLSCertsDir()
	if dir == "" {
		t.Skip("tls: porting-sdk/test_harness/tls not found adjacent to repo")
	}
	t.Setenv("SSL_CERT_FILE", filepath.Join(dir, "ca.crt"))
}

// findTLSCertsDir walks up to porting-sdk/test_harness/tls, runs the idempotent
// gen_certs.sh, and returns the certs dir, or "" when not adjacent / on error.
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

// runGenCerts invokes the idempotent gen_certs.sh so server.crt/key + ca.crt
// exist. gen_certs.sh is a no-op when the leaf cert still has >30 days left.
func runGenCerts(tlsDir string) error {
	cmd := exec.Command("bash", filepath.Join(tlsDir, "gen_certs.sh"))
	return cmd.Run()
}

// tlsMockRelay is a single --tls mock_relay instance on its own ports.
type tlsMockRelay struct {
	wsPort   int
	httpPort int
	httpURL  string // plain http:// control plane (TLS mode keeps it HTTP)
}

// sawRecvMethod reports whether the mock journaled an inbound (SDK->server)
// frame with the given JSON-RPC method, proving traffic crossed the WSS link.
func (m *tlsMockRelay) sawRecvMethod(t *testing.T, method string) bool {
	t.Helper()
	resp, err := http.Get(m.httpURL + "/__mock__/journal")
	if err != nil {
		t.Fatalf("tls mock_relay journal GET: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var entries []struct {
		Direction string `json:"direction"`
		Method    string `json:"method"`
	}
	if err := json.Unmarshal(body, &entries); err != nil {
		t.Fatalf("tls mock_relay journal decode: %v (body=%q)", err, body)
	}
	for _, e := range entries {
		if e.Direction == "recv" && e.Method == method {
			return true
		}
	}
	return false
}

// startTLSMockRelay spawns `python -m mock_relay --tls` on a dedicated WS+HTTP
// port pair, injecting porting-sdk/test_harness/mock_relay into PYTHONPATH via
// the adjacency walk. It waits for the plain-HTTP control plane to answer
// /__mock__/health, registers a Kill cleanup, and skips the test when the
// harness is unavailable.
func startTLSMockRelay(t *testing.T) *tlsMockRelay {
	t.Helper()
	pkgDir := discoverHarnessPkg("mock_relay")
	if pkgDir == "" {
		t.Skip("tls: porting-sdk/test_harness/mock_relay not adjacent")
	}
	wsPort := 18775
	httpPort := 19775
	httpURL := fmt.Sprintf("http://127.0.0.1:%d", httpPort)

	cmd := exec.Command("python", "-m", "mock_relay",
		"--host", "127.0.0.1",
		"--ws-port", strconv.Itoa(wsPort),
		"--http-port", strconv.Itoa(httpPort),
		"--tls",
		"--log-level", "error",
	)
	cmd.Env = harnessEnv(pkgDir, map[string]string{
		"SIGNALWIRE_MOCK_TLS": "1",
		"MOCK_RELAY_PORT":     strconv.Itoa(wsPort),
	})
	devnull, _ := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if devnull != nil {
		cmd.Stdout, cmd.Stderr, cmd.Stdin = devnull, devnull, devnull
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Skipf("tls: spawn mock_relay --tls: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		go func() { _ = cmd.Wait() }()
	})

	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(httpURL + "/__mock__/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return &tlsMockRelay{wsPort: wsPort, httpPort: httpPort, httpURL: httpURL}
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	t.Fatalf("tls: mock_relay --tls not ready on ws=%d http=%d", wsPort, httpPort)
	return nil
}

// discoverHarnessPkg walks up to porting-sdk/test_harness/<name>, returning the
// dir to prepend to PYTHONPATH (so `python -m <name>` resolves), or "".
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

// harnessEnv returns os.Environ() with PYTHONPATH prepended with pkgDir and any
// extra key=value overrides applied (replacing existing keys).
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
