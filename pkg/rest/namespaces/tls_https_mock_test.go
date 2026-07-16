// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// TLS capability test: prove the REST client performs a *real* verified HTTPS
// request.
//
// One of the three cross-port "every SDK does verified HTTPS + WSS" quadrants.
// It spawns the shared mock_signalwire in --tls mode (HTTPS, backed by the
// shared self-signed test CA), points a real *rest.RestClient at
// https://127.0.0.1:<port> via SetBaseURL, trusts the test CA, and performs a
// GET against a spec-backed endpoint, asserting a real JSON response.
//
// CA trust is wired idiomatically via SSL_CERT_FILE (TestMain): the REST
// client's underlying *http.Client uses the default transport, whose nil
// TLSClientConfig consults Go's system cert pool — which honors SSL_CERT_FILE
// on Linux. No InsecureSkipVerify, no transport mock.
//
// A negative subtest issues the same GET with an *empty* root pool and asserts
// the handshake is rejected, proving the cert is genuinely verified.
package namespaces_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"testing"
	"time"

	rest "github.com/signalwire/signalwire-go/v3/pkg/rest"
)

func TestTLS_RestClient_HTTPS(t *testing.T) {
	trustTestCA(t) // SSL_CERT_FILE -> test CA, before any TLS request
	mock := startTLSMockSignalwire(t)
	baseURL := fmt.Sprintf("https://127.0.0.1:%d", mock.port)

	// Build a real REST client and repoint its transport at the https://
	// mock. The mock accepts any non-empty Basic Auth.
	client, err := rest.NewRestClient("test_proj", "test_tok", fmt.Sprintf("127.0.0.1:%d", mock.port))
	if err != nil {
		t.Fatalf("NewRestClient: %v", err)
	}
	client.SetBaseURL(baseURL)

	// GET a spec-backed collection endpoint over HTTPS. A real JSON response
	// with a "data" array can only come back over a completed, CA-verified
	// TLS session (SSL_CERT_FILE was set in TestMain).
	bodyResp, err := client.Addresses.List(context.Background(), map[string]string{"page_size": "5"})
	if err != nil {
		t.Fatalf("Addresses.List over https:// failed: %v", err)
	}
	body := respMap(t, bodyResp)
	if _, ok := body["data"]; !ok {
		t.Fatalf("https response missing 'data' key; got keys %v", keys(body))
	}

	// Wire proof: the mock journaled the GET on its (HTTPS) control plane.
	if last := mock.lastJournal(t); last.Method != "GET" || last.Path != "/api/relay/rest/addresses" {
		t.Fatalf("mock journal did not record the HTTPS GET; got %s %s", last.Method, last.Path)
	}

	// Negative control: the same endpoint must reject a client that does not
	// trust the test CA, proving real certificate verification.
	t.Run("untrusted_client_rejected", func(t *testing.T) {
		untrusted := &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{RootCAs: x509.NewCertPool()}, // empty pool
			},
		}
		//nolint:bodyclose // expect-failure test: the request MUST fail (empty
		// trust pool), so the response is nil on the error path — nothing to close.
		_, err := untrusted.Get(baseURL + "/__mock__/health")
		if err == nil {
			t.Fatal("HTTPS GET with empty trust store unexpectedly succeeded")
		}
		t.Logf("untrusted HTTPS GET correctly rejected: %v", err)
	})
}
