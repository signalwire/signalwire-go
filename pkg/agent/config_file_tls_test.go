// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// BEHAVIORAL test for the config-file TLS layer.
//
// The reference resolves security settings defaults -> env -> CONFIG FILE, with
// the config file at HIGHEST priority (security_config.py __init__:
// _set_defaults(); load_from_env(); _load_config_file(...)), and SWMLService
// then serves off self.ssl_enabled / self.ssl_cert_path / self.ssl_key_path.
//
// So an operator who supplies ssl_enabled + cert + key ONLY in a config file
// must get a real HTTPS listener. A construction-only assertion cannot catch a
// regression here — the failure mode is "the process comes up plain HTTP with
// no error while the operator believes it is serving HTTPS" — so these tests
// start the real server and speak real TLS to it.
//
// Nothing is put in the environment: every SWML_SSL_* var is explicitly cleared
// so the ONLY source of the TLS settings is the config file.

package agent

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// clearSSLEnv unsets every SWML_SSL_* variable for the duration of the test, so
// a green result cannot be explained by an ambient environment value. t.Setenv
// restores the prior values on cleanup.
func clearSSLEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"SWML_SSL_ENABLED",
		"SWML_SSL_CERT_PATH",
		"SWML_SSL_KEY_PATH",
		"SWML_SSL_DOMAIN",
		"SWML_DOMAIN",
	} {
		t.Setenv(k, "")
		if err := os.Unsetenv(k); err != nil {
			t.Fatalf("unset %s: %v", k, err)
		}
	}
}

// agentTLSCertsDir walks up to the shared test harness's tls directory, runs the
// idempotent gen_certs.sh, and returns the certs dir. Skips when porting-sdk is
// not adjacent (mirrors pkg/swml's tlsServerCertsDir).
func agentTLSCertsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("tls: cannot resolve caller path")
	}
	dir, _ := filepath.Abs(filepath.Dir(file))
	for {
		tlsDir := filepath.Join(filepath.Dir(dir), "porting-sdk", "test_harness", "tls")
		if _, err := os.Stat(filepath.Join(tlsDir, "gen_certs.sh")); err == nil {
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

// agentCAClient builds an http.Client that trusts ONLY the shared test CA, so a
// successful request proves the server presented a genuinely verified cert (no
// InsecureSkipVerify).
func agentCAClient(t *testing.T, certsDir string) *http.Client {
	t.Helper()
	caPEM, err := os.ReadFile(filepath.Join(certsDir, "ca.crt"))
	if err != nil {
		t.Fatalf("read test CA: %v", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		t.Fatal("failed to load test CA into pool")
	}
	return &http.Client{
		Timeout:   3 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}},
	}
}

// serveAgentInBackground starts a.Run() in a goroutine and stops it on cleanup.
func serveAgentInBackground(t *testing.T, a *AgentBase) {
	t.Helper()
	runErr := make(chan error, 1)
	go func() { runErr <- a.Run() }()
	t.Cleanup(func() {
		_ = a.Stop()
		select {
		case err := <-runErr:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				t.Logf("Run returned: %v", err)
			}
		case <-time.After(3 * time.Second):
		}
	})
}

// TestConfigFileTLS_AgentServesHTTPS is the security regression test.
//
// 1. Write a config file supplying ssl_enabled / ssl_cert_path / ssl_key_path.
// 2. Put NOTHING in the environment.
// 3. Start the agent and prove it actually serves HTTPS.
//
// Before the fix, agent.WithConfigFile only stored the path on AgentBase — it
// was never handed to swml.NewService — so the file's TLS settings were
// silently discarded and the agent came up plain HTTP.
func TestConfigFileTLS_AgentServesHTTPS(t *testing.T) {
	clearSSLEnv(t)
	certs := agentTLSCertsDir(t)
	certFile := filepath.Join(certs, "server.crt")
	keyFile := filepath.Join(certs, "server.key")

	cfgPath := filepath.Join(t.TempDir(), "agent_config.json")
	cfg := fmt.Sprintf(`{
  "security": {
    "ssl_enabled": true,
    "ssl_cert_path": %q,
    "ssl_key_path": %q,
    "domain": "localhost"
  }
}
`, certFile, keyFile)
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	port := agentFreePort(t)
	a := NewAgentBase(
		WithName("cfgfile-tls"),
		WithRoute("/agent"),
		WithHost("127.0.0.1"),
		WithPort(port),
		WithConfigFile(cfgPath),
	)

	serveAgentInBackground(t, a)

	client := agentCAClient(t, certs)
	baseURL := "https://127.0.0.1:" + portToStr(port)

	// Poll /health over https until the TLS listener is up.
	var resp *http.Response
	deadline := time.Now().Add(10 * time.Second)
	for {
		var err error
		resp, err = client.Get(baseURL + "/health") //nolint:noctx // short-lived test probe with a client timeout
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("agent never became reachable over HTTPS with a config-file cert/key "+
				"(the config file's TLS settings were ignored and the server is plain HTTP): %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("https /health status = %d, want 200", resp.StatusCode)
	}
	if resp.TLS == nil {
		t.Fatal("response carries no TLS connection state; the request did not go over TLS")
	}
}

// TestConfigFileTLS_SSLEnabledFalseServesHTTP is the other half of the
// contract: ssl_enabled is honoured as a real switch, not ignored. With cert
// and key present but ssl_enabled false, the reference serves plain HTTP
// (swml_service.serve only takes the ssl branch when self.ssl_enabled).
//
// Without this, "always serve TLS when a cert path happens to be set" would
// pass the test above while diverging from the reference.
func TestConfigFileTLS_SSLEnabledFalseServesHTTP(t *testing.T) {
	clearSSLEnv(t)
	certs := agentTLSCertsDir(t)

	cfgPath := filepath.Join(t.TempDir(), "agent_config.json")
	cfg := fmt.Sprintf(`{
  "security": {
    "ssl_enabled": false,
    "ssl_cert_path": %q,
    "ssl_key_path": %q
  }
}
`, filepath.Join(certs, "server.crt"), filepath.Join(certs, "server.key"))
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	port := agentFreePort(t)
	a := NewAgentBase(
		WithName("cfgfile-notls"),
		WithRoute("/agent"),
		WithHost("127.0.0.1"),
		WithPort(port),
		WithConfigFile(cfgPath),
	)
	if a.Service.TLSEnabled() {
		t.Fatal("TLSEnabled() true with ssl_enabled:false in the config file")
	}

	serveAgentInBackground(t, a)
	agentWaitHealthy(t, "http://127.0.0.1:"+portToStr(port))
}

// TestConfigFileTLS_EnvOnlyServesHTTPS covers the middle layer: with no config
// file at all, SWML_SSL_ENABLED + SWML_SSL_CERT_PATH + SWML_SSL_KEY_PATH must
// put the agent on HTTPS. That is what docs/security.md tells operators to set,
// and the agent's serve path previously ignored it just as thoroughly as it
// ignored the config file.
func TestConfigFileTLS_EnvOnlyServesHTTPS(t *testing.T) {
	clearSSLEnv(t)
	certs := agentTLSCertsDir(t)
	t.Setenv("SWML_SSL_ENABLED", "true")
	t.Setenv("SWML_SSL_CERT_PATH", filepath.Join(certs, "server.crt"))
	t.Setenv("SWML_SSL_KEY_PATH", filepath.Join(certs, "server.key"))

	port := agentFreePort(t)
	a := NewAgentBase(
		WithName("env-tls"),
		WithRoute("/agent"),
		WithHost("127.0.0.1"),
		WithPort(port),
	)

	serveAgentInBackground(t, a)

	client := agentCAClient(t, certs)
	baseURL := "https://127.0.0.1:" + portToStr(port)

	var resp *http.Response
	deadline := time.Now().Add(10 * time.Second)
	for {
		var err error
		resp, err = client.Get(baseURL + "/health") //nolint:noctx // short-lived test probe with a client timeout
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("agent never served HTTPS from the SWML_SSL_* env vars: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.TLS == nil {
		t.Fatal("response carries no TLS connection state")
	}
}

// TestConfigFileTLS_EnvOverriddenByConfigFile pins the PRECEDENCE the reference
// defines: defaults -> env -> config file, config file HIGHEST. The env points
// at a nonexistent cert; the config file points at the real one. The config
// file must win, so the server serves verified HTTPS.
func TestConfigFileTLS_EnvOverriddenByConfigFile(t *testing.T) {
	clearSSLEnv(t)
	certs := agentTLSCertsDir(t)
	certFile := filepath.Join(certs, "server.crt")
	keyFile := filepath.Join(certs, "server.key")

	bogus := filepath.Join(t.TempDir(), "does-not-exist")
	t.Setenv("SWML_SSL_ENABLED", "true")
	t.Setenv("SWML_SSL_CERT_PATH", bogus+".crt")
	t.Setenv("SWML_SSL_KEY_PATH", bogus+".key")

	cfgPath := filepath.Join(t.TempDir(), "agent_config.json")
	cfg := fmt.Sprintf(`{
  "security": {
    "ssl_enabled": true,
    "ssl_cert_path": %q,
    "ssl_key_path": %q
  }
}
`, certFile, keyFile)
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	port := agentFreePort(t)
	a := NewAgentBase(
		WithName("cfgfile-precedence"),
		WithRoute("/agent"),
		WithHost("127.0.0.1"),
		WithPort(port),
		WithConfigFile(cfgPath),
	)

	serveAgentInBackground(t, a)

	client := agentCAClient(t, certs)
	baseURL := "https://127.0.0.1:" + portToStr(port)

	var resp *http.Response
	deadline := time.Now().Add(10 * time.Second)
	for {
		var err error
		resp, err = client.Get(baseURL + "/health") //nolint:noctx // short-lived test probe with a client timeout
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("config file did not override the env cert/key paths: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.TLS == nil {
		t.Fatal("response carries no TLS connection state")
	}
}
