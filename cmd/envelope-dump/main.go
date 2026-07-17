// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Command envelope-dump is the Go port's ENVELOPE-DUMP program for the cross-port
// REST error-envelope differ (porting-sdk/scripts/diff_port_envelope.py).
//
// It runs the shared error-envelope corpus (porting-sdk/scripts/envelope_corpus.py
// — the single source of truth, mirrored natively below) through the Go SDK's REST
// HTTPClient and prints ONE JSON object mapping
//
//	corpus-id -> artifact
//
// to stdout, where each artifact is the shared cross-port reduction:
//
//	{ "raised": bool, "error_kind": "typed"|"bare:<Type>"|null,
//	  "status_code": int|null, "body_error_code": string|null,
//	  "request_count": int }
//
// The differ builds the golden oracle by running the same corpus against the
// Python reference client, then byte-compares each artifact this program emits
// against Python's. See the differ's module docstring for the contract.
//
// Each case is exercised against an in-process httptest mock that honors the
// case's scenario (status / response body / Retry-After header / delay). A case
// flagged transport=true instead points the client at a DEAD port (a free port we
// bind then immediately release, so nothing is listening) — the connection-refused
// path. A correct client raises its TYPED transport error (the *SignalWireRestError
// family with StatusCode 0), which this program reports as error_kind "typed" with
// status_code null and request_count 0; a client leaking a bare net/url error would
// report "bare:<Type>" and fail the differ.
//
// Run from the signalwire-go repo root:
//
//	go run ./cmd/envelope-dump
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"time"

	"github.com/signalwire/signalwire-go/v3/pkg/rest"
)

// scenario is the armed mock override for a case (nil => a synthesized 200 list).
type scenario struct {
	status   int
	response any    // a JSON-encodable value
	rawBody  string // set instead of response for a deliberately non-JSON body
	retry    string // Retry-After header value (empty => not set)
	delayMs  int
}

// envCase is the Go-native mirror of one porting-sdk envelope_corpus._case entry.
type envCase struct {
	id        string
	scenario  *scenario
	transport bool
}

// The path every case targets (envelope_corpus.CALL — a GET list route).
const callPath = "/api/fabric/addresses"

// corpus mirrors porting-sdk/scripts/envelope_corpus.CORPUS. Keep the id set and
// the armed scenarios in lockstep with the Python source — the differ compares
// each artifact against Python's oracle for the same id.
var corpus = []envCase{
	// 200 success baseline: no scenario -> a synthesized 200 list body.
	{id: "envelope_200_success"},
	// 404 with a well-formed errors[] envelope.
	{id: "envelope_404_typed", scenario: &scenario{
		status:   404,
		response: map[string]any{"errors": []any{map[string]any{"code": "NOT_FOUND", "message": "no such address"}}},
	}},
	// 429 + Retry-After: pinned NO retry.
	{id: "envelope_429_retry_after", scenario: &scenario{
		status:   429,
		response: map[string]any{"errors": []any{map[string]any{"code": "RATE_LIMITED", "message": "slow down"}}},
		retry:    "2",
	}},
	// 503 service-unavailable: no retry.
	{id: "envelope_503_unavailable", scenario: &scenario{
		status:   503,
		response: map[string]any{"errors": []any{map[string]any{"code": "UNAVAILABLE", "message": "maintenance"}}},
	}},
	// 500 with a NON-JSON body: still typed, body_error_code null.
	{id: "envelope_500_malformed_body", scenario: &scenario{
		status:  500,
		rawBody: "not-json-at-all <garbage",
	}},
	// 200 whose body carries errors[]: 2xx == success, nothing raised.
	{id: "envelope_200_with_error_body", scenario: &scenario{
		status:   200,
		response: map[string]any{"errors": []any{map[string]any{"code": "SOFT_FAIL", "message": "ignored on 2xx"}}},
	}},
	// 200ms-delayed 503: the delay path still yields one typed 503.
	{id: "envelope_503_delayed", scenario: &scenario{
		status:   503,
		response: map[string]any{"errors": []any{map[string]any{"code": "UNAVAILABLE", "message": "slow-fail"}}},
		delayMs:  200,
	}},
	// connection refused (dead port): typed transport error, status null, count 0.
	{id: "envelope_transport_refused", transport: true},
}

// artifact is the shared cross-port reduction the differ byte-compares.
type artifact struct {
	Raised        bool    `json:"raised"`
	ErrorKind     *string `json:"error_kind"`
	StatusCode    *int    `json:"status_code"`
	BodyErrorCode *string `json:"body_error_code"`
	RequestCount  int     `json:"request_count"`
}

func strp(s string) *string { return &s }
func intp(i int) *int       { return &i }

// freeDeadPort binds a loopback port then releases it, so nothing listens there.
func freeDeadPort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		_ = ln.Close()
		return 0, fmt.Errorf("listener addr is %T, want *net.TCPAddr", ln.Addr())
	}
	port := addr.Port
	_ = ln.Close()
	return port, nil
}

// decodeBodyErrorCode mirrors the differ's _decode_body_error_code: parse a JSON
// body and pull errors[0].code, else null.
func decodeBodyErrorCode(body string) *string {
	var m map[string]any
	if json.Unmarshal([]byte(body), &m) != nil {
		return nil
	}
	errs, ok := m["errors"].([]any)
	if !ok || len(errs) == 0 {
		return nil
	}
	first, ok := errs[0].(map[string]any)
	if !ok {
		return nil
	}
	code, ok := first["code"].(string)
	if !ok {
		return nil
	}
	return &code
}

// runCase exercises one corpus case and returns its artifact. It stands up an
// in-process httptest mock honoring the scenario (or, for a transport case, points
// the client at a dead port), makes the request, and reduces the outcome.
func runCase(c envCase) artifact {
	var art artifact
	client := rest.NewHTTPClient("envelope_proj", "envelope_tok", "mock.invalid")

	if c.transport {
		// Dead port: nothing listening -> connection refused. request_count stays 0.
		dead, err := freeDeadPort()
		if err != nil {
			panic(fmt.Sprintf("reserve dead port: %v", err))
		}
		client.SetBaseURL(fmt.Sprintf("http://127.0.0.1:%d", dead))
		_, reqErr := client.Get(callPath, nil)
		reduceError(&art, reqErr)
		return art
	}

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == callPath {
			atomic.AddInt32(&hits, 1)
		}
		sc := c.scenario
		if sc == nil {
			// Synthesized 200 list body (the no-scenario happy path).
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"data":[]}`))
			return
		}
		if sc.delayMs > 0 {
			time.Sleep(time.Duration(sc.delayMs) * time.Millisecond)
		}
		if sc.retry != "" {
			w.Header().Set("Retry-After", sc.retry)
		}
		status := sc.status
		if status == 0 {
			status = 200
		}
		if sc.rawBody != "" {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(sc.rawBody))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		enc, _ := json.Marshal(sc.response)
		_, _ = w.Write(enc)
	}))
	defer srv.Close()

	client.SetBaseURL(srv.URL)
	_, reqErr := client.Get(callPath, nil)
	reduceError(&art, reqErr)
	art.RequestCount = int(atomic.LoadInt32(&hits))
	return art
}

// reduceError fills the raised/error_kind/status_code/body_error_code fields from
// the request error, distinguishing the typed *SignalWireRestError family (a
// transport failure OR an HTTP error) from a bare leaked error.
func reduceError(art *artifact, reqErr error) {
	if reqErr == nil {
		return
	}
	art.Raised = true
	var restErr *rest.SignalWireRestError
	if errors.As(reqErr, &restErr) {
		art.ErrorKind = strp("typed")
		if restErr.Transport {
			// Transport failure: no HTTP status -> status_code null (Go's 0 maps to
			// the Python reference's None); body carries the transport message with
			// no errors[] to decode -> body_error_code null.
			art.StatusCode = nil
			art.BodyErrorCode = nil
		} else {
			art.StatusCode = intp(restErr.StatusCode)
			art.BodyErrorCode = decodeBodyErrorCode(restErr.Body)
		}
		return
	}
	art.ErrorKind = strp("bare:" + fmt.Sprintf("%T", reqErr))
}

func main() {
	out := map[string]artifact{}
	for _, c := range corpus {
		out[c.id] = runCase(c)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
