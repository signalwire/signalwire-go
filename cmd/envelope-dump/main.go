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
// The differ builds the golden reference by running the same corpus against the
// Python reference client, then byte-compares each artifact this program emits
// against Python's. See the differ's module docstring for the contract.
//
// Each case is exercised against an in-process httptest mock that honors the
// case's scenario (status / response body / Retry-After header / delay), and — for
// the RequestOptions retry cases (plan 4.2) — the FIFO scenario_repeat queue and
// the case's RequestOptions (retries / retry_backoff / timeout). A case flagged
// transport=true instead points the client at a DEAD port (a free port we bind
// then immediately release, so nothing is listening) — the connection-refused
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
	"context"
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

// scenario is one armed mock override (nil => a synthesized 200 list / 201).
type scenario struct {
	status   int
	response any    // a JSON-encodable value
	rawBody  string // set instead of response for a deliberately non-JSON body
	retry    string // Retry-After header value (empty => not set)
	delayMs  int
}

// requestOptionsSpec mirrors a corpus case's request_options block. A nil field
// means "unset" (the port's default applies).
type requestOptionsSpec struct {
	retries      *int
	retryBackoff *float64
	timeout      *float64
}

// envCase is the Go-native mirror of one porting-sdk envelope_corpus._case entry.
type envCase struct {
	id string
	// scenario is the armed override; scenarioRepeat arms it N times (FIFO) so a
	// retry-armed case sees the failure on every attempt before the default 200.
	scenario       *scenario
	scenarioRepeat int
	transport      bool
	// method + path + body drive the request. Defaults to GET callPath; the POST
	// idempotency cases set method=POST + createPath + a body.
	method string
	path   string
	body   map[string]any
	// requestOptions, when set, is passed as the port's RequestOptions for the call.
	requestOptions *requestOptionsSpec
	// composeLeg, when non-empty ("ctx"|"signal"|"both"), marks a ctx/abort_signal
	// COMPOSITION case (PSDK-4c / GO-5) whose artifact is the compose classification
	// {ctx_cancel_honored, signal_cancel_honored, both_compose} rather than the
	// error-envelope artifact. These cases drive a SLOW (3s) armed response and
	// assert the request is cancelled within composeWindow by the intended source.
	composeLeg string
}

// compose timing knobs — MUST match diff_port_envelope.py's _drive_compose.
const (
	composeWindow       = 1500 * time.Millisecond // bounded iff cancelled within this
	composeShortTimeout = 500 * time.Millisecond  // < the 3s delay: fires
	composeLongTimeout  = 10 * time.Second        // > the 3s delay: the SIGNAL fires, not this
	composeSlowDelayMs  = 3000                    // the armed response delay
)

// The GET list route every default case targets (envelope_corpus.CALL).
const callPath = "/api/fabric/addresses"

// The POST route the idempotency-asymmetry cases target (envelope_corpus.CREATE_CALL).
const createPath = "/api/relay/rest/addresses"

func intPtr(i int) *int           { return &i }
func floatPtr(f float64) *float64 { return &f }

// corpus mirrors porting-sdk/scripts/envelope_corpus.CORPUS. Keep the id set and
// the armed scenarios in lockstep with the Python source — the differ compares
// each artifact against Python's reference for the same id.
var corpus = []envCase{
	// 200 success baseline: no scenario -> a synthesized 200 list body.
	{id: "envelope_200_success"},
	// 404 with a well-formed errors[] envelope.
	{id: "envelope_404_typed", scenario: &scenario{
		status:   404,
		response: map[string]any{"errors": []any{map[string]any{"code": "NOT_FOUND", "message": "no such address"}}},
	}},
	// 429 + Retry-After with DEFAULT options: pinned NO retry (request_count 1).
	{id: "envelope_429_retry_after", scenario: &scenario{
		status:   429,
		response: map[string]any{"errors": []any{map[string]any{"code": "RATE_LIMITED", "message": "slow down"}}},
		retry:    "2",
	}},
	// 503 service-unavailable with DEFAULT options: no retry.
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

	// ================= RequestOptions envelope (plan 4.2) =================
	// retry_backoff=0 so the differ never waits on wall-clock; the observable is
	// the attempt COUNT.

	// GET + retries=1: the single armed 503 is retried into the default 200 =>
	// raised=false, request_count=2.
	{
		id:             "envelope_get_retry_once_succeeds",
		scenario:       &scenario{status: 503, response: map[string]any{"errors": []any{map[string]any{"code": "UNAVAILABLE", "message": "transient"}}}},
		requestOptions: &requestOptionsSpec{retries: intPtr(1), retryBackoff: floatPtr(0)},
	},
	// GET + retries=1 with the 503 armed on BOTH attempts: retries exhausted =>
	// typed 503 raised, request_count=2.
	{
		id:             "envelope_get_retry_exhausted",
		scenario:       &scenario{status: 503, response: map[string]any{"errors": []any{map[string]any{"code": "UNAVAILABLE", "message": "down"}}}},
		scenarioRepeat: 2,
		requestOptions: &requestOptionsSpec{retries: intPtr(1), retryBackoff: floatPtr(0)},
	},
	// POST + retries=2 with a 500: non-idempotent must NOT retry 500 =>
	// request_count=1, typed 500 raised.
	{
		id:             "envelope_post_500_not_retried",
		method:         "POST",
		path:           createPath,
		body:           map[string]any{"label": "x"},
		scenario:       &scenario{status: 500, response: map[string]any{"errors": []any{map[string]any{"code": "SERVER_ERROR", "message": "boom"}}}},
		requestOptions: &requestOptionsSpec{retries: intPtr(2), retryBackoff: floatPtr(0)},
	},
	// POST + retries=1 with a 503: a throttle IS safe to retry => retried into the
	// default 201/200, request_count=2.
	{
		id:             "envelope_post_503_retried",
		method:         "POST",
		path:           createPath,
		body:           map[string]any{"label": "x"},
		scenario:       &scenario{status: 503, response: map[string]any{"errors": []any{map[string]any{"code": "UNAVAILABLE", "message": "throttled"}}}},
		requestOptions: &requestOptionsSpec{retries: intPtr(1), retryBackoff: floatPtr(0)},
	},

	// ============ ctx/abort_signal COMPOSITION (PSDK-4c / GO-5) ============
	// A per-request abort_signal and a caller ctx/deadline must COMPOSE: a cancel
	// from EITHER cancels the request (merge, never replace). Each leg arms a SLOW
	// (3s) response and asserts the request is cancelled within composeWindow by
	// the intended source. The "both" leg is the GO-5 red: a client-default
	// abort_signal that REPLACED the caller ctx (the old bug) dropped the ctx
	// deadline, letting the request run to ~3s (un-bounded).
	{id: "compose_ctx_timeout_alone", composeLeg: "ctx"},
	{id: "compose_abort_signal_alone", composeLeg: "signal"},
	{id: "compose_ctx_and_signal_both", composeLeg: "both"},
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

// caseMethod / casePath resolve a case's request verb + path, defaulting to the
// GET list route when unset.
func caseMethod(c envCase) string {
	if c.method != "" {
		return c.method
	}
	return "GET"
}

func casePath(c envCase) string {
	if c.path != "" {
		return c.path
	}
	return callPath
}

// buildRequestOptions maps a case's requestOptions spec to a *rest.RequestOptions
// (nil when the case pins no options => the client's default: retries 0).
func buildRequestOptions(spec *requestOptionsSpec) *rest.RequestOptions {
	if spec == nil {
		return nil
	}
	opts := &rest.RequestOptions{}
	if spec.retries != nil {
		opts.Retries = spec.retries
	}
	if spec.retryBackoff != nil {
		opts.RetryBackoff = spec.retryBackoff
	}
	if spec.timeout != nil {
		opts.Timeout = spec.timeout
	}
	return opts
}

// runCase exercises one corpus case and returns its artifact. It stands up an
// in-process httptest mock honoring the scenario queue (scenarioRepeat armed
// entries served FIFO, then the synthesized success), or — for a transport case —
// points the client at a dead port, makes the request with the case's
// RequestOptions, and reduces the outcome.
func runCase(c envCase) artifact {
	var art artifact
	client := rest.NewHTTPClient("envelope_proj", "envelope_tok", "mock.invalid")
	method := caseMethod(c)
	path := casePath(c)
	reqOpts := buildRequestOptions(c.requestOptions)

	if c.transport {
		// Dead port: nothing listening -> connection refused. request_count stays 0.
		dead, err := freeDeadPort()
		if err != nil {
			panic(fmt.Sprintf("reserve dead port: %v", err))
		}
		client.SetBaseURL(fmt.Sprintf("http://127.0.0.1:%d", dead))
		reduceError(&art, doCall(client, method, path, c.body, reqOpts))
		return art
	}

	// The FIFO scenario queue: the armed override repeated scenarioRepeat times,
	// then exhausted (nil) so subsequent attempts fall through to the synthesized
	// success. This mirrors the mock_signalwire ScenarioStore the Python reference
	// arms via scenario_repeat.
	repeat := c.scenarioRepeat
	if repeat < 1 {
		repeat = 1
	}
	var queue []*scenario
	if c.scenario != nil {
		for range repeat {
			queue = append(queue, c.scenario)
		}
	}
	var qi int32 = -1

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == path {
			atomic.AddInt32(&hits, 1)
		}
		// Pop the next armed scenario (FIFO); nil once the queue is exhausted.
		var sc *scenario
		idx := int(atomic.AddInt32(&qi, 1))
		if idx < len(queue) {
			sc = queue[idx]
		}
		if sc == nil {
			// Synthesized success (the no-scenario / exhausted-queue happy path):
			// 200 for GET, 201 for POST — a body every port decodes as success.
			w.Header().Set("Content-Type", "application/json")
			if method == "POST" {
				w.WriteHeader(201)
				_, _ = w.Write([]byte(`{"id":"addr_created"}`))
				return
			}
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
	reduceError(&art, doCall(client, method, path, c.body, reqOpts))
	art.RequestCount = int(atomic.LoadInt32(&hits))
	return art
}

// doCall issues the case's REST verb through the SDK's request-options-aware HTTP
// client, returning only the request error (the body is irrelevant to the
// artifact).
func doCall(client *rest.HTTPClient, method, path string, body map[string]any, opts *rest.RequestOptions) error {
	switch method {
	case "POST":
		_, err := client.Post(path, body, nil, opts)
		return err
	default:
		_, err := client.Get(path, nil, opts)
		return err
	}
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

// composeClassification is the compose-case artifact the differ byte-compares
// against the python golden {ctx_cancel_honored, signal_cancel_honored,
// both_compose}.
type composeClassification struct {
	CtxCancelHonored    bool `json:"ctx_cancel_honored"`
	SignalCancelHonored bool `json:"signal_cancel_honored"`
	BothCompose         bool `json:"both_compose"`
}

// runComposeCase drives one compose leg against an in-process mock that delays
// its 200 response by composeSlowDelayMs, and reports whether the request was
// cancelled within composeWindow by the intended source. It uses ONLY the
// exported client API: a client-default abort_signal (via NewHTTPClient's
// variadic RequestOptions) plus a per-call ctx deadline (GetContext) — exactly
// the natural GO-5 usage (a global-shutdown abort_signal must NOT swallow each
// call's ctx deadline).
func runComposeCase(c envCase) composeClassification {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Duration(composeSlowDelayMs) * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	// bounded runs one GET and reports whether it returned (was cancelled) within
	// composeWindow. clientSignal is the client-DEFAULT abort_signal (nil => none);
	// ctxTimeout is the per-call ctx deadline (0 => background, no deadline).
	bounded := func(clientSignal context.Context, ctxTimeout time.Duration) bool {
		var defaults *rest.RequestOptions
		if clientSignal != nil {
			defaults = &rest.RequestOptions{AbortSignal: clientSignal, Retries: intPtr(0)}
		}
		var client *rest.HTTPClient
		if defaults != nil {
			client = rest.NewHTTPClient("compose_proj", "compose_tok", "mock.invalid", defaults)
		} else {
			client = rest.NewHTTPClient("compose_proj", "compose_tok", "mock.invalid")
		}
		client.SetBaseURL(srv.URL)

		ctx := context.Background()
		if ctxTimeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, ctxTimeout)
			defer cancel()
		}
		t0 := time.Now()
		_, _ = client.GetContext(ctx, callPath, nil)
		return time.Since(t0) < composeWindow
	}

	var out composeClassification
	switch c.composeLeg {
	case "ctx":
		// SHORT ctx deadline vs slow response, NO signal: the ctx must cut it.
		out.CtxCancelHonored = bounded(nil, composeShortTimeout)
	case "signal":
		// A PRE-CANCELLED client-default abort_signal, GENEROUS ctx deadline: the
		// SIGNAL must cut it (not the deadline).
		sigCtx, sigCancel := context.WithCancel(context.Background())
		sigCancel() // pre-cancelled
		out.SignalCancelHonored = bounded(sigCtx, composeLongTimeout)
	case "both":
		// A LIVE (un-cancelled) client-default abort_signal AND a SHORT ctx
		// deadline: the ctx deadline must still fire (proving the signal did NOT
		// replace/drop the ctx — the GO-5 compose contract).
		sigCtx, sigCancel := context.WithCancel(context.Background())
		defer sigCancel()
		out.BothCompose = bounded(sigCtx, composeShortTimeout)
	}
	return out
}

func main() {
	out := map[string]any{}
	for _, c := range corpus {
		if c.composeLeg != "" {
			out[c.id] = runComposeCase(c)
			continue
		}
		out[c.id] = runCase(c)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
