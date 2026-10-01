// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// SECURITY: an AgentBase configured for TLS must never serve PLAINTEXT.
//
// The agent is how an operator actually exposes a listener in practice, and it
// builds its OWN http.Server rather than delegating to swml.Service.Serve — so
// the plain-HTTP refusal has to be pinned on this path independently. Same
// failure mode, same probe: bind a real listener, drive a real cleartext GET at
// it, and assert nothing answers.
//
// The swml-side twin is pkg/swml/tls_no_silent_downgrade_test.go.
package agent

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// agentServeAndProbePlaintext starts a.Serve() and drives a real PLAIN-HTTP GET
// at its /health route, reporting whether cleartext was actually served.
func agentServeAndProbePlaintext(t *testing.T, a *AgentBase, port int) (plaintextOK bool, serveErr error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	errCh := make(chan error, 1)
	go func() { errCh <- a.RunContext(ctx) }()

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

// TestAgentTLS_NoSilentPlaintextDowngrade_EnvSwitch: SWML_SSL_ENABLED=true with
// no cert/key must NOT quietly serve plain HTTP from an agent.
func TestAgentTLS_NoSilentPlaintextDowngrade_EnvSwitch(t *testing.T) {
	t.Setenv("SWML_SSL_ENABLED", "true")
	t.Setenv("SWML_SSL_CERT_PATH", "")
	t.Setenv("SWML_SSL_KEY_PATH", "")

	port := agentFreePort(t)
	a := NewAgentBase(
		WithName("tls-downgrade-agent-env"),
		WithHost("127.0.0.1"),
		WithPort(port),
	)

	plaintextOK, serveErr := agentServeAndProbePlaintext(t, a, port)
	if plaintextOK {
		t.Fatal("SILENT PLAINTEXT DOWNGRADE: an agent with ssl_enabled=true and no " +
			"cert/key served a working plain-HTTP endpoint; the operator asked for " +
			"TLS and got cleartext with no error")
	}
	if serveErr == nil {
		t.Fatal("RunContext() neither refused nor reported an error")
	}
	if !strings.Contains(serveErr.Error(), "ssl_enabled") {
		t.Errorf("RunContext() error %q does not name the ssl_enabled misconfiguration", serveErr)
	}
}

// TestAgentTLS_NoSilentPlaintextDowngrade_ConfigFile: the same through a config
// file's security.ssl_enabled.
func TestAgentTLS_NoSilentPlaintextDowngrade_ConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "agent.json")
	if err := os.WriteFile(cfg, []byte(`{"security":{"ssl_enabled":true}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	port := agentFreePort(t)
	a := NewAgentBase(
		WithName("tls-downgrade-agent-cfg"),
		WithHost("127.0.0.1"),
		WithPort(port),
		WithConfigFile(cfg),
	)

	plaintextOK, serveErr := agentServeAndProbePlaintext(t, a, port)
	if plaintextOK {
		t.Fatal("SILENT PLAINTEXT DOWNGRADE: an agent with security.ssl_enabled=true " +
			"and no cert/key paths served a working plain-HTTP endpoint")
	}
	if serveErr == nil {
		t.Fatal("RunContext() neither refused nor reported an error")
	}
}

// TestAgentTLS_PlainHTTP_StillWorks is the negative control: with no TLS asked
// for, plain HTTP remains correct and must keep working, so the guard above
// cannot pass by refusing to serve at all.
func TestAgentTLS_PlainHTTP_StillWorks(t *testing.T) {
	t.Setenv("SWML_SSL_ENABLED", "")
	port := agentFreePort(t)
	a := NewAgentBase(
		WithName("plain-http-agent-control"),
		WithHost("127.0.0.1"),
		WithPort(port),
	)
	plaintextOK, serveErr := agentServeAndProbePlaintext(t, a, port)
	if !plaintextOK {
		t.Fatalf("plain HTTP was refused with no TLS requested (serveErr=%v)", serveErr)
	}
}
