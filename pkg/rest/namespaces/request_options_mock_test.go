// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// RequestOptions envelope — behavioral contract over the real mock (plan 4.2).
// Translated from signalwire-python/tests/unit/rest/test_request_options.py.
//
// These drive a real REST HTTPClient through the real net/http transport into
// the shared mock_signalwire and assert on the recorded journal — the same
// journal the REST-COVERAGE gate reads. Retry / timeout are wire-observable: the
// mock sees N attempts and honors the backoff/delay ordering, so the contract is
// proven over the real mock, NOT a transport mock/patch.
//
// Contract pinned here (the oracle):
//   - Retries: a retryable failure is retried up to Retries extra times; the mock
//     sees Retries+1 attempts; the final success is returned.
//   - idempotency asymmetry: GET/PUT/DELETE retry on the full retry_on_status set;
//     POST/PATCH retry only on 429/503 (throttles), never 500/502/504.
//   - Timeout: a server-side delay exceeding the timeout raises the transport
//     error family.
//   - AbortSignal (a cancelled context.Context): raises the transport error family
//     before the request goes out.
//   - per-request options shallow-override the client default.
package namespaces_test

import (
	"context"
	"errors"
	"testing"

	rest "github.com/signalwire/signalwire-go/v3/pkg/rest"
	"github.com/signalwire/signalwire-go/v3/pkg/rest/internal/mocktest"
)

const (
	roAddressesPath       = "/api/fabric/addresses"
	roAddressesEndpointID = "fabric.list_fabric_addresses"
	roCreatePath          = "/api/relay/rest/addresses"
	roCreateEndpointID    = "relay-rest.create_address"
)

func roIntPtr(i int) *int           { return &i }
func roFloatPtr(f float64) *float64 { return &f }

// roGetCount counts GET journal entries for the addresses path.
func roGetCount(t *testing.T, m *mocktest.Harness) int {
	t.Helper()
	n := 0
	for _, e := range m.Journal(t) {
		if e.Path == roAddressesPath && e.Method == "GET" {
			n++
		}
	}
	return n
}

func roPostCount(t *testing.T, m *mocktest.Harness) int {
	t.Helper()
	n := 0
	for _, e := range m.Journal(t) {
		if e.Path == roCreatePath && e.Method == "POST" {
			n++
		}
	}
	return n
}

// ---------- Retry contract ----------

func TestRequestOptions_GetRetries503ThenSucceeds(t *testing.T) {
	client, m := mocktest.New(t)
	if client == nil {
		return
	}
	m.Reset(t)
	// Arm a single 503; the mock's default synthesized 200 follows it. With
	// Retries=1 the client retries the 503 into the 200 => 2 attempts.
	m.PushScenario(t, roAddressesEndpointID, 503, map[string]any{"errors": []any{map[string]any{"code": "X"}}})

	resp, err := client.HTTPClient().Get(roAddressesPath, nil,
		&rest.RequestOptions{Retries: roIntPtr(1), RetryBackoff: roFloatPtr(0)})
	if err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected a non-nil body")
	}
	if got := roGetCount(t, m); got != 2 {
		t.Errorf("expected 2 attempts (503 then 200), got %d", got)
	}
}

func TestRequestOptions_NoRetriesByDefault(t *testing.T) {
	client, m := mocktest.New(t)
	if client == nil {
		return
	}
	m.Reset(t)
	m.PushScenario(t, roAddressesEndpointID, 503, map[string]any{"errors": []any{map[string]any{"code": "X"}}})

	_, err := client.HTTPClient().Get(roAddressesPath, nil, nil)
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("expected *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 503 {
		t.Errorf("expected status 503, got %d", restErr.StatusCode)
	}
	if got := roGetCount(t, m); got != 1 {
		t.Errorf("default must not retry: expected 1 attempt, got %d", got)
	}
}

func TestRequestOptions_RetriesExhaustedRaisesLastError(t *testing.T) {
	client, m := mocktest.New(t)
	if client == nil {
		return
	}
	m.Reset(t)
	// Two 503s + Retries=1 => attempts = 2, both 503 => raise the 503.
	m.PushScenario(t, roAddressesEndpointID, 503, map[string]any{"errors": []any{map[string]any{"code": "X"}}})
	m.PushScenario(t, roAddressesEndpointID, 503, map[string]any{"errors": []any{map[string]any{"code": "X"}}})

	_, err := client.HTTPClient().Get(roAddressesPath, nil,
		&rest.RequestOptions{Retries: roIntPtr(1), RetryBackoff: roFloatPtr(0)})
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("expected *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 503 {
		t.Errorf("expected status 503, got %d", restErr.StatusCode)
	}
	if got := roGetCount(t, m); got != 2 {
		t.Errorf("Retries=1 => exactly 2 attempts, got %d", got)
	}
}

// ---------- Idempotency asymmetry ----------

func TestRequestOptions_PostDoesNotRetry500(t *testing.T) {
	client, m := mocktest.New(t)
	if client == nil {
		return
	}
	m.Reset(t)
	// A real POST route; 500 is NOT retryable for a non-idempotent method even
	// with retries armed => exactly one attempt, raise the 500.
	m.PushScenario(t, roCreateEndpointID, 500, map[string]any{"errors": []any{map[string]any{"code": "SERVER_ERROR"}}})

	_, err := client.HTTPClient().Post(roCreatePath, map[string]any{"label": "x"}, nil,
		&rest.RequestOptions{Retries: roIntPtr(2), RetryBackoff: roFloatPtr(0)})
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("expected *SignalWireRestError, got %v", err)
	}
	if restErr.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", restErr.StatusCode)
	}
	if got := roPostCount(t, m); got != 1 {
		t.Errorf("POST must not retry a 500 (side-effect safety): expected 1, got %d", got)
	}
}

func TestRequestOptions_PostDoesRetry503(t *testing.T) {
	client, m := mocktest.New(t)
	if client == nil {
		return
	}
	m.Reset(t)
	// 503 (throttle) IS retryable even for a non-idempotent method => the 503
	// retries into the mock's default success.
	m.PushScenario(t, roCreateEndpointID, 503, map[string]any{"errors": []any{map[string]any{"code": "UNAVAILABLE"}}})

	_, err := client.HTTPClient().Post(roCreatePath, map[string]any{"label": "x"}, nil,
		&rest.RequestOptions{Retries: roIntPtr(1), RetryBackoff: roFloatPtr(0)})
	if err != nil {
		t.Fatalf("expected success after 503 retry, got %v", err)
	}
	if got := roPostCount(t, m); got != 2 {
		t.Errorf("POST retries a 503 throttle (safe): expected 2, got %d", got)
	}
}

// ---------- Timeout ----------

func TestRequestOptions_SlowResponseTimesOut(t *testing.T) {
	client, m := mocktest.New(t)
	if client == nil {
		return
	}
	m.Reset(t)
	// Arm a 200 delayed 400ms; a 100ms timeout must fire => transport error.
	m.PushScenarioFull(t, roAddressesEndpointID, 200,
		map[string]any{"data": []any{}}, nil, 400)

	_, err := client.HTTPClient().Get(roAddressesPath, nil,
		&rest.RequestOptions{Timeout: roFloatPtr(0.1)})
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("expected *SignalWireRestError (transport), got %v", err)
	}
	if !restErr.Transport {
		t.Errorf("expected a transport error (timeout), got HTTP status %d", restErr.StatusCode)
	}
}

// ---------- Abort signal (context.Context) ----------

func TestRequestOptions_PresetAbortRaisesTransportError(t *testing.T) {
	client, m := mocktest.New(t)
	if client == nil {
		return
	}
	m.Reset(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before the request

	_, err := client.HTTPClient().Get(roAddressesPath, nil,
		&rest.RequestOptions{AbortSignal: ctx})
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("expected *SignalWireRestError (transport), got %v", err)
	}
	if !restErr.Transport {
		t.Errorf("expected a transport error (aborted), got HTTP status %d", restErr.StatusCode)
	}
	// errors.Is must see through to context.Canceled (the wrapped cause).
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected the error to wrap context.Canceled")
	}
	// Nothing reached the mock — cancelled before the send.
	if got := roGetCount(t, m); got != 0 {
		t.Errorf("aborted request must not reach the server, got %d entries", got)
	}
}

// ---------- Per-request override ----------

func TestRequestOptions_PerRequestOverridesClientDefault(t *testing.T) {
	// Client default = no retries; per-request opts in to 1 retry.
	client, m := mocktest.NewWithOptions(t, &rest.RequestOptions{Retries: roIntPtr(0)})
	if client == nil {
		return
	}
	m.Reset(t)
	m.PushScenario(t, roAddressesEndpointID, 503, map[string]any{"errors": []any{map[string]any{"code": "X"}}})

	resp, err := client.HTTPClient().Get(roAddressesPath, nil,
		&rest.RequestOptions{Retries: roIntPtr(1), RetryBackoff: roFloatPtr(0)})
	if err != nil {
		t.Fatalf("per-request retries=1 should override client default 0, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected a non-nil body")
	}
	if got := roGetCount(t, m); got != 2 {
		t.Errorf("expected 2 attempts (per-request override), got %d", got)
	}
}
