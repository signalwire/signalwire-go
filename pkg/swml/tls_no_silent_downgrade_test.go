// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// SECURITY: a service configured for TLS must never serve PLAINTEXT.
//
// tls_server_test.go proves the HAPPY path (WithTLS -> a real verified HTTPS
// listener). This file pins the FAILURE path, which is the one that actually
// hurts: an operator who asked for encryption and did not get it, and was never
// told. Serve()'s TLS branch is gated on Service.TLSEnabled(), which requires
// BOTH the ssl_enabled switch AND a non-empty cert/key pair — so every way of
// turning the switch on while the cert/key resolution comes back empty used to
// fall straight through to the plain-HTTP ListenAndServe with no error and no
// warning. Two reachable spellings of that:
//
//	SWML_SSL_ENABLED=true with no SWML_SSL_CERT_PATH/KEY_PATH
//	a config file's security.ssl_enabled: true with no ssl_cert_path/ssl_key_path
//
// Both are BEHAVIOURALLY probed here — a real listener is bound and a real
// plaintext HTTP request is driven at it — because the observable question is
// "what came out of the socket", not "what does the struct say". A grep for
// InsecureSkipVerify would answer neither.
package swml

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// serveAndProbePlaintext starts svc.Serve() and drives a real PLAIN-HTTP GET at
// its /health route. It returns whether the plaintext request SUCCEEDED (i.e.
// the listener was serving cleartext), plus any Serve() error.
func serveAndProbePlaintext(t *testing.T, svc *Service, port int) (plaintextOK bool, serveErr error) {
	t.Helper()
	errCh := make(chan error, 1)
	go func() { errCh <- svc.Serve() }()
	t.Cleanup(func() { _ = svc.Stop() })

	// Serve() either binds or fails fast; give it a moment either way.
	deadline := time.Now().Add(3 * time.Second)
	client := &http.Client{Timeout: 500 * time.Millisecond}
	for time.Now().Before(deadline) {
		select {
		case err := <-errCh:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				return false, err
			}
		default:
		}
		resp, err := client.Get("http://127.0.0.1:" + strconv.Itoa(port) + "/health")
		if err == nil {
			_ = resp.Body.Close()
			return true, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	select {
	case err := <-errCh:
		return false, err
	default:
		return false, nil
	}
}

// TestTLS_NoSilentPlaintextDowngrade_EnvSwitch: SWML_SSL_ENABLED=true with no
// cert/key must NOT quietly serve plain HTTP.
func TestTLS_NoSilentPlaintextDowngrade_EnvSwitch(t *testing.T) {
	t.Setenv("SWML_SSL_ENABLED", "true")
	t.Setenv("SWML_SSL_CERT_PATH", "")
	t.Setenv("SWML_SSL_KEY_PATH", "")

	port := freeTCPPort(t)
	svc := NewService(
		WithName("tls-downgrade-env"),
		WithHost("127.0.0.1"),
		WithPort(port),
	)

	plaintextOK, serveErr := serveAndProbePlaintext(t, svc, port)
	if plaintextOK {
		t.Fatal("SILENT PLAINTEXT DOWNGRADE: ssl_enabled=true with no cert/key " +
			"served a working plain-HTTP endpoint; an operator who asked for TLS " +
			"got cleartext and was never told")
	}
	if serveErr == nil {
		t.Fatal("Serve() neither refused nor reported an error; TLS was requested " +
			"and nothing said it could not be provided")
	}
	if !strings.Contains(serveErr.Error(), "ssl_enabled") {
		t.Errorf("Serve() error %q does not name the ssl_enabled misconfiguration", serveErr)
	}
}

// TestTLS_NoSilentPlaintextDowngrade_ConfigFile: the same switch via a config
// file's security.ssl_enabled, which is the documented operator path and the
// one security_config.go calls out as load-bearing.
func TestTLS_NoSilentPlaintextDowngrade_ConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "svc.json")
	if err := os.WriteFile(cfg, []byte(`{"security":{"ssl_enabled":true}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	port := freeTCPPort(t)
	svc := NewService(
		WithName("tls-downgrade-cfg"),
		WithHost("127.0.0.1"),
		WithPort(port),
		WithConfigFile(cfg),
	)

	plaintextOK, serveErr := serveAndProbePlaintext(t, svc, port)
	if plaintextOK {
		t.Fatal("SILENT PLAINTEXT DOWNGRADE: security.ssl_enabled=true with no " +
			"ssl_cert_path/ssl_key_path served a working plain-HTTP endpoint")
	}
	if serveErr == nil {
		t.Fatal("Serve() neither refused nor reported an error for an ssl_enabled " +
			"config file with no cert/key")
	}
}

// TestTLS_NoSilentPlaintextDowngrade_MissingCertFile: ssl_enabled with cert/key
// paths that do not EXIST. Go's ListenAndServeTLS already fails loud here, so
// this pins that it stays loud (and, critically, that nothing bound a cleartext
// listener on the way to failing).
func TestTLS_NoSilentPlaintextDowngrade_MissingCertFile(t *testing.T) {
	dir := t.TempDir()
	port := freeTCPPort(t)
	svc := NewService(
		WithName("tls-downgrade-missing"),
		WithHost("127.0.0.1"),
		WithPort(port),
		WithTLS(filepath.Join(dir, "absent.crt"), filepath.Join(dir, "absent.key")),
	)
	if !svc.TLSEnabled() {
		t.Fatal("TLSEnabled() false after an explicit WithTLS")
	}

	plaintextOK, serveErr := serveAndProbePlaintext(t, svc, port)
	if plaintextOK {
		t.Fatal("SILENT PLAINTEXT DOWNGRADE: a missing cert file produced a " +
			"working plain-HTTP endpoint")
	}
	if serveErr == nil {
		t.Fatal("Serve() reported no error for a nonexistent cert file")
	}
}

// TestTLS_PlainHTTP_StillWorks is the negative control for the three above: with
// NO ssl_enabled anywhere, plain HTTP is the correct, intended behaviour and must
// keep working. Without this, the downgrade guard could pass by refusing to serve
// at all.
func TestTLS_PlainHTTP_StillWorks(t *testing.T) {
	t.Setenv("SWML_SSL_ENABLED", "")
	port := freeTCPPort(t)
	svc := NewService(
		WithName("plain-http-control"),
		WithHost("127.0.0.1"),
		WithPort(port),
	)
	if svc.TLSEnabled() {
		t.Fatal("TLSEnabled() true with no TLS configuration")
	}
	plaintextOK, serveErr := serveAndProbePlaintext(t, svc, port)
	if !plaintextOK {
		t.Fatalf("plain HTTP was refused with no TLS requested (serveErr=%v)", serveErr)
	}
}
