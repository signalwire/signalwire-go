// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// TLS capability test: prove the SDK's own webhook/SWML server serves a *real*
// verified HTTPS endpoint.
//
// The third cross-port "every SDK does verified HTTPS + WSS" quadrant — the
// server side. It starts a swml.Service via WithTLS(server.crt, server.key)
// (the shared porting-sdk self-signed leaf cert, SAN localhost/127.0.0.1) in a
// goroutine, then reaches its unauthenticated /health route from an in-test Go
// *http.Client that trusts the test CA over https://, asserting a real
// response. Running the server in a goroutine + an in-process client keeps the
// handshake entirely in-test (no shelling to curl).
//
// CA trust is wired idiomatically via SSL_CERT_FILE (TestMain): the client's
// default transport consults Go's system cert pool, which honors it on Linux.
// No InsecureSkipVerify. A negative subtest uses an empty root pool and asserts
// the handshake is rejected, proving the server's cert is genuinely verified.
package swml

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestTLS_Server_HTTPS(t *testing.T) {
	certs := tlsServerCertsDir(t)
	trustTestCA(t, certs) // SSL_CERT_FILE -> test CA, before any TLS request
	certFile := filepath.Join(certs, "server.crt")
	keyFile := filepath.Join(certs, "server.key")

	port := freeTCPPort(t)
	svc := NewService(
		WithName("tls-cap-test"),
		WithHost("127.0.0.1"),
		WithPort(port),
		WithTLS(certFile, keyFile),
	)
	if !svc.TLSEnabled() {
		t.Fatal("TLSEnabled() false after WithTLS; server would not serve https")
	}

	// Serve() blocks (ListenAndServeTLS), so run it in a goroutine and stop it
	// on cleanup. A clean shutdown returns http.ErrServerClosed.
	serveErr := make(chan error, 1)
	go func() { serveErr <- svc.Serve() }()
	t.Cleanup(func() {
		_ = svc.Stop()
		select {
		case err := <-serveErr:
			if err != nil && err != http.ErrServerClosed {
				t.Logf("Serve returned: %v", err)
			}
		case <-time.After(3 * time.Second):
		}
	})

	baseURL := fmt.Sprintf("https://127.0.0.1:%d", port)

	// Trust the test CA via an explicit RootCAs pool built from ca.crt. This
	// works on every OS — unlike SSL_CERT_FILE, which Go's system cert pool
	// honors on Linux but NOT on macOS (Darwin delegates to Security.framework
	// and ignores SSL_CERT_FILE, so the default client there gets "certificate
	// is not trusted"). Mirrors the explicit-pool approach the negative-control
	// subtest below already uses. trustTestCA still sets SSL_CERT_FILE for any
	// other consumers; this client doesn't depend on it.
	caPEM, err := os.ReadFile(filepath.Join(certs, "ca.crt"))
	if err != nil {
		t.Fatalf("read test CA: %v", err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		t.Fatal("failed to load test CA into pool")
	}
	client := &http.Client{
		Timeout:   3 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: caPool}},
	}

	// Poll /health until the TLS listener is up, then assert a real response.
	var resp *http.Response
	deadline := time.Now().Add(10 * time.Second)
	for {
		var err error
		resp, err = client.Get(baseURL + "/health")
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("server /health never became reachable over https: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("https /health status = %d, want 200", resp.StatusCode)
	}
	if resp.TLS == nil {
		t.Fatal("response has no TLS connection state; request did not go over TLS")
	}
	body, _ := io.ReadAll(resp.Body)
	var payload map[string]string
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode /health body %q: %v", body, err)
	}
	if payload["status"] != "healthy" {
		t.Fatalf("https /health body = %v, want status=healthy", payload)
	}

	// Negative control: a client that does not trust the test CA must be
	// rejected, proving the server presents a cert that is actually verified.
	t.Run("untrusted_client_rejected", func(t *testing.T) {
		untrusted := &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{RootCAs: x509.NewCertPool()}, // empty pool
			},
		}
		_, err := untrusted.Get(baseURL + "/health")
		if err == nil {
			t.Fatal("https /health with empty trust store unexpectedly succeeded")
		}
		t.Logf("untrusted client correctly rejected by SDK https server: %v", err)
	})
}

// freeTCPPort asks the OS for an unused loopback TCP port.
func freeTCPPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freeTCPPort: %v", err)
	}
	defer func() { _ = l.Close() }()
	addr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected *net.TCPAddr, got %T", l.Addr())
	}
	return addr.Port
}

// ---------------------------------------------------------------------------
// Test-only TLS cert discovery for the server test. Self-contained so nothing
// the run-ci gates depend on is perturbed.
// ---------------------------------------------------------------------------

// trustTestCA wires the throwaway CA into Go's system cert pool via
// SSL_CERT_FILE so the in-test client trusts the SDK server's CA-signed leaf
// cert. It must run before the first TLS handshake in the process (the system
// pool is loaded once and cached); this TLS test is the only TLS user in the
// package, so setting it at the top of the test is sufficient. The negative
// subtest's explicit empty pool is unaffected.
func trustTestCA(t *testing.T, certsDir string) {
	t.Helper()
	t.Setenv("SSL_CERT_FILE", filepath.Join(certsDir, "ca.crt"))
}

// tlsServerCertsDir walks up to porting-sdk/test_harness/tls, runs the
// idempotent gen_certs.sh, and returns the certs dir. Skips the test when
// porting-sdk is not adjacent.
func tlsServerCertsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("tls: cannot resolve caller path")
	}
	dir, _ := filepath.Abs(filepath.Dir(file))
	for {
		tlsDir := filepath.Join(filepath.Dir(dir), "porting-sdk", "test_harness", "tls")
		if _, err := os.Stat(filepath.Join(tlsDir, "gen_certs.sh")); err == nil {
			// Idempotent: regenerates only if the leaf cert is missing/near expiry.
			if err := exec.Command("bash", filepath.Join(tlsDir, "gen_certs.sh")).Run(); err != nil {
				t.Skipf("tls: gen_certs.sh failed: %v", err)
			}
			return filepath.Join(tlsDir, "certs")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("tls: porting-sdk/test_harness/tls not found adjacent to repo")
		}
		dir = parent
	}
}
