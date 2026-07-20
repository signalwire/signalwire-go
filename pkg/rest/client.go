// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Package rest provides a REST client for the SignalWire platform APIs.
//
// It includes an HTTP transport layer, generic CRUD resource abstractions,
// paginated iteration, and namespaced sub-clients for each API domain.
package rest

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/signalwire/signalwire-go/v3/pkg/logging"
)

// userAgent is the User-Agent header sent with every request.
const userAgent = "signalwire-go-rest/1.0"

// ---------- SignalWireRestError ----------

// SignalWireRestError is returned when a SignalWire REST request fails. It covers
// two failure modes with one typed family so a caller unwraps a single type:
//
//   - an HTTP error: the server responded with a non-2xx status. StatusCode is the
//     HTTP code (>= 400), Transport is false.
//   - a TRANSPORT failure: the request never reached a response (connection
//     refused, DNS failure, connection reset, TLS error). Go has no HTTP status in
//     that case, so StatusCode is 0 and Transport is true — the equivalent of the
//     Python reference's status_code=None. Body carries the underlying transport
//     error message.
//
// Because both modes are the SAME type, a caller catching *SignalWireRestError via
// errors.As handles HTTP and transport failures with one branch, instead of a bare
// net/url error leaking out.
type SignalWireRestError struct {
	StatusCode int
	Body       string
	URL        string
	Method     string
	// Transport is true when this error represents a transport-level failure (the
	// request never reached a response), in which case StatusCode is 0. It is false
	// for an HTTP-status error (a real >= 400 response).
	Transport bool
	// Headers is the response header map captured on an HTTP-status error (client-
	// side observability — plan 6.6; no wire change). nil for a transport failure
	// (no response was produced), matching the Python reference's headers=None.
	Headers http.Header
	// RequestID is the platform request-id extracted from the response headers
	// (x-request-id / x-signalwire-request-id / request-id / x-amzn-requestid, first
	// match wins — the Python reference's precedence). Empty for a transport failure
	// or when no such header is present. Appended to Error() for observability.
	RequestID string
	// cause is the underlying transport error (net/url error, context cancellation,
	// TLS error) for a Transport failure, preserved so errors.Is/errors.As still
	// unwrap to it — the Go equivalent of Python's ``raise ... from exc``. nil for
	// an HTTP-status error.
	cause error
}

// Unwrap returns the underlying transport error (or nil for an HTTP-status error),
// so errors.Is / errors.As see through a transport-wrapped *SignalWireRestError to
// the original cause (e.g. context.Canceled, a *net.OpError). This preserves the
// error chain the way Python's “raise SignalWireRestTransportError(...) from exc“
// does, while still presenting the typed REST error family at the top.
func (e *SignalWireRestError) Unwrap() error { return e.cause }

// Error implements the error interface. When a platform RequestID was captured it
// is appended for observability, matching the Python reference (plan 6.6).
func (e *SignalWireRestError) Error() string {
	var msg string
	if e.Transport {
		msg = fmt.Sprintf("%s %s failed to reach the server: %s", e.Method, e.URL, e.Body)
	} else {
		msg = fmt.Sprintf("%s %s returned %d: %s", e.Method, e.URL, e.StatusCode, e.Body)
	}
	if e.RequestID != "" {
		msg += fmt.Sprintf(" (request-id: %s)", e.RequestID)
	}
	return msg
}

// requestIDHeaders is the precedence-ordered set of response header names that
// carry the platform request-id (first match wins), mirroring the Python
// reference (_extract_request_id in signalwire/rest/_base.py).
var requestIDHeaders = []string{
	"X-Request-Id", "X-Signalwire-Request-Id", "Request-Id", "X-Amzn-Requestid",
}

// extractRequestID returns the platform request-id from a response header map, or
// "" when none of the known headers is present. http.Header.Get is
// case-insensitive (canonicalized), so the caller's header casing is irrelevant.
func extractRequestID(h http.Header) string {
	for _, name := range requestIDHeaders {
		if v := h.Get(name); v != "" {
			return v
		}
	}
	return ""
}

// NewSignalWireRestError constructs a SignalWireRestError for an HTTP-status
// failure, substituting "GET" as the method when method is empty — matches
// Python's default. headers is the response header map (may be nil — e.g. a
// hand-built error); when present the platform request-id is extracted from it
// (plan 6.6 error-observability, mirroring the reference's optional headers param).
func NewSignalWireRestError(statusCode int, body, url, method string, headers http.Header) *SignalWireRestError {
	if method == "" {
		method = "GET"
	}
	return &SignalWireRestError{
		StatusCode: statusCode,
		Body:       body,
		URL:        url,
		Method:     method,
		Headers:    headers,
		RequestID:  extractRequestID(headers),
	}
}

// NewSignalWireRestTransportError constructs a SignalWireRestError for a
// TRANSPORT-level failure — the request never reached a response (connection
// refused, DNS failure, connection reset, TLS error, context cancellation).
// StatusCode is 0 (Go's idiom for "no HTTP status", the equivalent of the Python
// reference's status_code=None) and Transport is true, so the same
// *SignalWireRestError family a caller already unwraps for HTTP errors also carries
// transport failures — no bare net/url error escapes the REST client. cause is the
// underlying error, preserved for errors.Is/errors.As (may be nil); Body defaults
// to cause.Error() when body is empty.
func NewSignalWireRestTransportError(cause error, body, url, method string) *SignalWireRestError {
	if method == "" {
		method = "GET"
	}
	if body == "" && cause != nil {
		body = cause.Error()
	}
	return &SignalWireRestError{StatusCode: 0, Body: body, URL: url, Method: method, Transport: true, cause: cause}
}

// ---------- HTTPClient ----------

// HTTPClient is a thin wrapper around net/http that provides Basic Auth,
// JSON encoding/decoding, and standard headers for SignalWire API calls.
type HTTPClient struct {
	baseURL    string
	projectID  string
	token      string
	httpClient *http.Client
	logger     *logging.Logger
	// requestOptions is the CLIENT-DEFAULT request-options envelope (plan 4.2):
	// timeout / retries / backoff / abort applied to every request unless a
	// per-request *RequestOptions overrides it. nil => the built-in defaults
	// (30s timeout, 0 retries). Set via the trailing variadic option on
	// NewHTTPClient / NewRestClient.
	requestOptions *RequestOptions
}

// NewHTTPClient creates a new HTTPClient configured for the given SignalWire
// space. The baseURL is normally constructed as "https://<space>", but the
// SIGNALWIRE_REST_BASE_URL environment variable overrides it when set —
// pointing the client at a loopback fixture for the the shared test harness
// audit_rest_transport.py harness, or at any non-default endpoint.
func NewHTTPClient(projectID, token, space string, opts ...*RequestOptions) *HTTPClient {
	baseURL := os.Getenv("SIGNALWIRE_REST_BASE_URL")
	if baseURL == "" {
		baseURL = "https://" + space
	}
	// The per-attempt deadline lives in the request-options retry loop
	// (resolveOptions => effectiveOptions.timeout), applied per attempt via a
	// context.WithTimeout. The underlying http.Client carries NO fixed Timeout
	// so the loop is the sole owner of the deadline (a client-level Timeout
	// would double-cap and defeat a per-request override).
	httpClient := &http.Client{}
	// When SIGNALWIRE_REST_CA_FILE names a PEM CA bundle, trust it for HTTPS by
	// building a RootCAs pool and a custom transport. This matches the
	// cross-port convention (rust honors SIGNALWIRE_REST_CA_FILE, ruby a
	// ca_file arg) and lets callers behind a private CA verify the server
	// without relying on the OS trust store — which, via SSL_CERT_FILE, Go's
	// system cert pool honors on Linux but NOT on macOS. Unset/empty => the
	// default transport + system roots (unchanged behavior).
	if pool := caPoolFromEnv("SIGNALWIRE_REST_CA_FILE"); pool != nil {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool},
		}
	}
	var reqOpts *RequestOptions
	if len(opts) > 0 {
		reqOpts = opts[len(opts)-1] // last non-variadic option wins
	}
	return &HTTPClient{
		baseURL:        baseURL,
		projectID:      projectID,
		token:          token,
		httpClient:     httpClient,
		logger:         logging.New("rest_client"),
		requestOptions: reqOpts,
	}
}

// caPoolFromEnv reads the PEM file named by the given env var and returns a
// CertPool containing its certificates, or nil when the env var is unset/empty.
// A non-empty env var pointing at an unreadable/invalid PEM is fatal: a
// configured-but-broken CA must not silently fall back to system roots (that
// would mask a misconfiguration as a different, confusing TLS error later).
func caPoolFromEnv(envVar string) *x509.CertPool {
	path := os.Getenv(envVar)
	if path == "" {
		return nil
	}
	//nolint:gosec // G304: path is the operator-supplied CA-file env var, not
	// attacker input — reading the configured CA bundle is the intended behavior.
	pem, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("read %s %q: %v", envVar, path, err))
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		panic(fmt.Sprintf("parse %s %q: no certificates found in PEM", envVar, path))
	}
	return pool
}

// BaseURL returns the base URL used by this client.
func (c *HTTPClient) BaseURL() string {
	return c.baseURL
}

// SetBaseURL overrides the base URL used by this client. Useful for
// pointing the client at a non-default endpoint (audit fixtures, mock
// servers, etc.) without re-running the constructor with a synthetic
// space name.
func (c *HTTPClient) SetBaseURL(url string) {
	c.baseURL = url
}

// Get performs an HTTP GET request. params are added as query-string
// parameters.
func (c *HTTPClient) Get(path string, params map[string]string, opts *RequestOptions) (map[string]any, error) {
	return c.doRequestContextOpts(context.Background(), "GET", path, nil, params, opts)
}

// Post performs an HTTP POST request with a JSON body. Optional params are
// appended to the URL as query-string parameters.
func (c *HTTPClient) Post(path string, body map[string]any, params map[string]string, opts *RequestOptions) (map[string]any, error) {
	return c.doRequestContextOpts(context.Background(), "POST", path, body, params, opts)
}

// Put performs an HTTP PUT request with a JSON body.
func (c *HTTPClient) Put(path string, body map[string]any, opts *RequestOptions) (map[string]any, error) {
	return c.doRequestContextOpts(context.Background(), "PUT", path, body, nil, opts)
}

// Patch performs an HTTP PATCH request with a JSON body.
func (c *HTTPClient) Patch(path string, body map[string]any, opts *RequestOptions) (map[string]any, error) {
	return c.doRequestContextOpts(context.Background(), "PATCH", path, body, nil, opts)
}

// Delete performs an HTTP DELETE request. It returns the parsed response body
// (or an empty map for 204 No Content) and any error.
func (c *HTTPClient) Delete(path string, opts *RequestOptions) (map[string]any, error) {
	return c.doRequestContextOpts(context.Background(), "DELETE", path, nil, nil, opts)
}

// GetContext is the context-aware variant of Get: the request is cancelled when
// ctx is cancelled or its deadline passes.
func (c *HTTPClient) GetContext(ctx context.Context, path string, params map[string]string) (map[string]any, error) {
	return c.doRequestContextOpts(ctx, "GET", path, nil, params, nil)
}

// PostContext is the context-aware variant of Post.
func (c *HTTPClient) PostContext(ctx context.Context, path string, body map[string]any, params map[string]string) (map[string]any, error) {
	return c.doRequestContextOpts(ctx, "POST", path, body, params, nil)
}

// PutContext is the context-aware variant of Put.
func (c *HTTPClient) PutContext(ctx context.Context, path string, body map[string]any) (map[string]any, error) {
	return c.doRequestContextOpts(ctx, "PUT", path, body, nil, nil)
}

// PatchContext is the context-aware variant of Patch.
func (c *HTTPClient) PatchContext(ctx context.Context, path string, body map[string]any) (map[string]any, error) {
	return c.doRequestContextOpts(ctx, "PATCH", path, body, nil, nil)
}

// DeleteContext is the context-aware variant of Delete.
func (c *HTTPClient) DeleteContext(ctx context.Context, path string) (map[string]any, error) {
	return c.doRequestContextOpts(ctx, "DELETE", path, nil, nil, nil)
}

// doRequestContext is the shared request execution method. It threads the
// caller's context onto the HTTP request (so cancellation and deadlines are
// observed), sets Basic Auth, Content-Type, Accept, and User-Agent headers. A
// 204 No Content response returns an empty map. Non-2xx responses return a
// *SignalWireRestError.
// doRequestContextOpts is the request-options-aware entry point: it resolves
// the effective options (per-request over client-default over built-in),
// then runs the retry/timeout/abort loop over doAttempt.
//
// total attempts = retries + 1; a retryable failure (an idempotency-aware
// retryable status, or a transport error) is retried, honoring Retry-After
// then exponential backoff. AbortSignal (a context.Context) is checked BEFORE
// each attempt (a cancelled ctx raises the typed transport error without a
// send) and is also threaded onto the request so it cuts an in-flight send.
// A per-attempt Timeout is applied via context.WithTimeout.
func (c *HTTPClient) doRequestContextOpts(ctx context.Context, method, path string, body any, params map[string]string, perRequest *RequestOptions) (map[string]any, error) {
	opts := Resolve(c.requestOptions, perRequest)
	// AbortSignal (the RequestOptions cancellation primitive) IS a
	// context.Context. It must COMPOSE with the caller-supplied ctx (the
	// ...Context verbs' deadline/cancellation), NOT replace it: a cancel from
	// EITHER source cancels the request. Deriving `parent` from ctx and wiring
	// the abort_signal to cancel that derived context means the caller's ctx
	// deadline is still honored when an AbortSignal is armed (the GO-5 fix — the
	// old code overwrote ctx with the signal, silently dropping every caller's
	// timeout/cancellation whenever a client-default AbortSignal was set).
	parent := ctx
	if opts.abortSignal != nil {
		var cancel context.CancelFunc
		parent, cancel = context.WithCancel(ctx)
		defer cancel()
		// Cancel the derived context when the abort_signal fires (AfterFunc runs
		// immediately if it is already cancelled). ctx cancelling still cancels
		// `parent` directly (it is the parent), so BOTH sources compose.
		stop := context.AfterFunc(opts.abortSignal, cancel)
		defer stop()
	}

	attempt := 0
	for {
		attempt++
		// Cooperative cancellation BEFORE the attempt: a cancelled parent (the
		// AbortSignal or caller ctx) surfaces as the typed transport error with
		// NO send, mirroring the Python reference's pre-attempt abort check.
		if err := parent.Err(); err != nil {
			return nil, NewSignalWireRestTransportError(err, "request cancelled by abort_signal", c.buildURL(path, params), method)
		}

		result, status, retryAfter, err := c.doAttempt(parent, method, path, body, params, opts.timeout)
		if err == nil {
			return result, nil
		}

		var restErr *SignalWireRestError
		isREST := errorsAs(err, &restErr)
		retryable := false
		if isREST && restErr.Transport {
			// Transport failure (connection refused / DNS / reset / TLS / timeout):
			// no response was produced. Retry if attempts remain, regardless of
			// method (there is no server-side side effect to duplicate).
			retryable = true
		} else if isREST {
			retryable = StatusIsRetryable(method, status, opts)
		}

		if retryable && attempt <= opts.retries {
			delay := retryAfter
			if delay < 0 {
				delay = opts.retryBackoff * math.Pow(2, float64(attempt-1))
			}
			if !c.sleepOrCancel(parent, delay) {
				// Cancelled during backoff -> typed transport error.
				return nil, NewSignalWireRestTransportError(parent.Err(), "request cancelled by abort_signal", c.buildURL(path, params), method)
			}
			continue
		}
		return nil, err
	}
}

// buildURL composes the FULL request URL (scheme+host+path+query) from the
// client's base URL, the path, and the query params — the exact string sent on
// the wire and stored in error.URL (plan D1) so a caller can replay the request.
func (c *HTTPClient) buildURL(path string, params map[string]string) string {
	reqURL := c.baseURL + path
	if len(params) > 0 {
		q := url.Values{}
		for k, v := range params {
			q.Set(k, v)
		}
		reqURL += "?" + q.Encode()
	}
	return reqURL
}

// sleepOrCancel sleeps for delay seconds, returning false if the context is
// cancelled first (so the loop can surface the typed transport error instead
// of finishing the backoff). A non-positive delay returns immediately.
func (c *HTTPClient) sleepOrCancel(ctx context.Context, delaySeconds float64) bool {
	if delaySeconds <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(time.Duration(delaySeconds * float64(time.Second)))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// doAttempt performs ONE HTTP attempt. It returns the decoded body on success,
// or a *SignalWireRestError (HTTP-status or transport) on failure, along with
// the HTTP status (0 for a transport failure) and the parsed Retry-After delta
// in seconds (-1 when absent) so the retry loop can honor it. timeoutSeconds
// caps this single attempt via context.WithTimeout.
func (c *HTTPClient) doAttempt(ctx context.Context, method, path string, body any, params map[string]string, timeoutSeconds float64) (map[string]any, int, float64, error) {
	reqURL := c.buildURL(path, params)

	// Encode body
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, -1, fmt.Errorf("json marshal: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	c.logger.Debug("REST request %s %s", method, path)

	// Per-attempt deadline: the resolved timeout caps THIS attempt. A
	// cancelled/expired ctx makes http.Client.Do fail, which we map to the
	// typed transport error below (status 0).
	attemptCtx := ctx
	var cancel context.CancelFunc
	if timeoutSeconds > 0 {
		attemptCtx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds*float64(time.Second)))
		defer cancel()
	}

	req, err := http.NewRequestWithContext(attemptCtx, method, reqURL, bodyReader)
	if err != nil {
		return nil, 0, -1, fmt.Errorf("new request: %w", err)
	}

	// Headers
	req.SetBasicAuth(c.projectID, c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Transport failure (connection refused / DNS / reset / TLS / context
		// cancellation / per-attempt timeout): the request never produced a
		// response. Wrap it in the typed error family so a caller unwrapping
		// *SignalWireRestError handles it too, instead of a bare net/url error
		// leaking out. The underlying error is kept as the cause so errors.Is
		// (e.g. context.Canceled / context.DeadlineExceeded) still sees through it.
		// url = the FULL request URL (scheme+host+path+query, plan D1), so a caller
		// logging error.URL can replay the exact request.
		return nil, 0, -1, NewSignalWireRestTransportError(err, "", reqURL, method)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, -1, fmt.Errorf("read body: %w", err)
	}

	// Non-2xx error
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// URL = the FULL request URL (scheme+host+path+query, plan D1), not the
		// bare path — a caller logging error.URL can replay the exact request.
		// Headers + RequestID capture the response headers + platform request-id
		// for client-side observability (plan 6.6; no wire change).
		return nil, resp.StatusCode, retryAfterSeconds(resp), &SignalWireRestError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
			URL:        reqURL,
			Method:     method,
			Headers:    resp.Header,
			RequestID:  extractRequestID(resp.Header),
		}
	}

	// 204 No Content or empty body
	if resp.StatusCode == 204 || len(respBody) == 0 {
		return map[string]any{}, resp.StatusCode, -1, nil
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		// Some endpoints (e.g. fabric list_subscriber_addresses) return a
		// top-level JSON array rather than an object. Wrap it under the
		// canonical "data" key so callers and the paginator see a uniform
		// map[string]any shape.
		var arr []any
		if arrErr := json.Unmarshal(respBody, &arr); arrErr == nil {
			return map[string]any{"data": arr}, resp.StatusCode, -1, nil
		}
		return nil, 0, -1, fmt.Errorf("json unmarshal: %w", err)
	}
	return result, resp.StatusCode, -1, nil
}

// retryAfterSeconds parses a Retry-After response header in delta-seconds form,
// returning the delay in seconds or -1 when the header is absent or in the
// HTTP-date form (which we do not honor; the loop falls back to exponential
// backoff). Mirrors the Python reference _retry_after_seconds.
func retryAfterSeconds(resp *http.Response) float64 {
	v := resp.Header.Get("Retry-After")
	if v == "" {
		return -1
	}
	secs, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return -1 // HTTP-date form: fall back to computed backoff
	}
	return secs
}

// errorsAs is a thin wrapper over errors.As kept local so the retry loop reads
// cleanly; it lets a *SignalWireRestError be recovered from a wrapped error.
func errorsAs(err error, target **SignalWireRestError) bool {
	return errors.As(err, target)
}
