// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// TLS capability test: prove the RELAY client performs a *real* verified
// WSS handshake.
//
// This is one of the three cross-port "every SDK does verified HTTPS + WSS"
// capability quadrants. It spawns the shared mock_relay in --tls mode (so the
// WebSocket endpoint is wss:// backed by the shared self-signed test CA),
// points the real *relay.Client at wss://127.0.0.1:<port>, trusts the test CA,
// and drives the full connect + authenticate handshake.
//
// CA trust is wired idiomatically via SSL_CERT_FILE (set by trustTestCA before
// the first dial): Go's crypto/x509 system pool honors it on Linux, and
// gorilla/websocket's dialer uses that system pool when its TLSClientConfig is
// nil — which is exactly how pkg/relay/client.go constructs its dialer. No
// InsecureSkipVerify, no transport mock: the server-issued protocol string
// returned by Authenticate() can only come back over a genuinely-completed TLS
// session.
//
// A negative subtest dials the same wss:// endpoint with an *empty* root pool
// and asserts the handshake is rejected ("certificate signed by unknown
// authority"), proving the server presents a cert that must actually be
// verified — i.e. trust is real, not skipped.
package relay_test

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/signalwire/signalwire-go/v3/pkg/relay"
)

func TestTLS_RelayClient_WSS(t *testing.T) {
	trustTestCA(t) // SSL_CERT_FILE -> test CA, before any TLS dial
	mock := startTLSMockRelay(t)

	// Point the real RELAY client at the wss:// endpoint. The client reads
	// SIGNALWIRE_RELAY_HOST + SIGNALWIRE_RELAY_SCHEME (pkg/relay/client.go
	// connect()), so "wss" + host:port routes the gorilla dialer over TLS.
	t.Setenv("SIGNALWIRE_RELAY_HOST", fmt.Sprintf("127.0.0.1:%d", mock.wsPort))
	t.Setenv("SIGNALWIRE_RELAY_SCHEME", "wss")

	client := relay.NewRelayClient(
		relay.WithProject("test_proj"),
		relay.WithToken("test_tok"),
		relay.WithContexts("default"),
	)
	t.Cleanup(client.Stop)

	if err := client.Connect(); err != nil {
		t.Fatalf("Connect over wss:// failed: %v", err)
	}
	if err := client.Authenticate(); err != nil {
		t.Fatalf("Authenticate over wss:// failed: %v", err)
	}
	client.StartReadLoop()

	// Behavioral proof the TLS session carried a real RELAY handshake: the
	// mock only issues a protocol string in the signalwire.connect result on
	// a successful credential exchange (mock_relay auth.SessionStore.connect).
	// An empty value means the connect round-trip never completed over TLS.
	if proto := client.RelayProtocol(); proto == "" {
		t.Fatal("RelayProtocol() empty after WSS Authenticate; server-issued value missing")
	}

	// Wire proof: the mock journaled the inbound signalwire.connect frame on
	// the same (TLS) WebSocket. The journal is served over the plain-HTTP
	// control plane (mock_relay keeps the control plane HTTP even in --tls).
	if !mock.sawRecvMethod(t, "signalwire.connect") {
		t.Fatal("mock journal has no recv signalwire.connect frame over the WSS connection")
	}

	// Negative control: the same endpoint must reject a client that does NOT
	// trust the test CA, proving real certificate verification is in force.
	t.Run("untrusted_client_rejected", func(t *testing.T) {
		dialer := websocket.Dialer{
			HandshakeTimeout: 5 * time.Second,
			TLSClientConfig:  &tls.Config{RootCAs: x509.NewCertPool()}, // empty pool
		}
		url := fmt.Sprintf("wss://127.0.0.1:%d/api/relay/ws", mock.wsPort)
		//nolint:bodyclose // expect-failure test: the dial MUST fail (empty trust
		// pool), so conn/resp are nil on the error path — nothing to close.
		conn, _, err := dialer.Dial(url, http.Header{})
		if err == nil {
			_ = conn.Close()
			t.Fatal("WSS dial with empty trust store unexpectedly succeeded")
		}
		t.Logf("untrusted WSS dial correctly rejected: %v", err)
	})
}
