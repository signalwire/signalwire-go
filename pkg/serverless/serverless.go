// Package serverless adapts a net/http handler produced by the SignalWire
// Agents SDK (typically agent.AsRouter()) so an agent can run under the
// non-Lambda serverless platforms: CGI and Google Cloud Functions.
//
// It mirrors the design of pkg/lambda: a platform invocation is translated
// into a synthetic *http.Request, served against the underlying http.Handler
// with an in-memory recorder, and the recorder's result is translated back
// into the platform's response shape. Because the agent's own AsRouter()
// already handles auth, SWML rendering at the root path, SWAIG dispatch at
// /swaig, and routing callbacks (/sip), a thin request adapter is all each
// platform needs — matching Python's ServerlessMixin, which dispatches CGI /
// Lambda / Cloud Functions / Azure through the same request-handling core.
//
// Platform detection lives in pkg/swml (swml.DetectRunMode); this package is
// the DISPATCH layer that produces a real response for a detected platform,
// so a serverless deployment no longer falls back to ErrServerlessUnsupported.
//
//	CGI: main() { serverless.NewHandler(a.AsRouter()).ServeCGI(context.Background()) }
//	GCF: functions.HTTP("Agent", serverless.NewHandler(a.AsRouter()).ServeHTTP)
package serverless

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// MaxCGIBodySize caps the request body read from stdin in CGI mode, matching
// Python's MAX_CGI_BODY_SIZE guard in serverless_mixin.py against unbounded
// CONTENT_LENGTH.
const MaxCGIBodySize = 10 * 1024 * 1024 // 10 MiB

// Handler wraps an http.Handler so it can service non-Lambda serverless
// invocations (CGI, Google Cloud Functions). Construct one with NewHandler.
type Handler struct {
	h http.Handler
}

// NewHandler returns a Handler that dispatches serverless invocations to the
// given http.Handler (usually agent.AsRouter()).
//
// Panics if h is nil to fail loudly at cold-start rather than silently
// returning 500 on every invocation — same contract as pkg/lambda.
func NewHandler(h http.Handler) *Handler {
	if h == nil {
		panic("serverless.NewHandler: http.Handler must not be nil")
	}
	return &Handler{h: h}
}

// ServeHTTP handles a Google Cloud Functions (2nd-gen / Cloud Run functions)
// invocation. A GCF HTTP function receives a standard *http.Request and writes
// to a standard http.ResponseWriter, so dispatch is a direct pass-through to
// the wrapped agent handler — the agent's router already renders SWML at the
// root path, dispatches SWAIG at /swaig, and enforces auth. This is the GCF
// analog of pkg/lambda's HandleFunctionURL.
//
// This signature (func(http.ResponseWriter, *http.Request)) is exactly what
// GCP's functions-framework functions.HTTP registration expects.
func (s *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.h.ServeHTTP(w, r)
}

// CGIResult is the outcome of a CGI dispatch: the HTTP status, response
// headers, and body the CGI host should emit. WriteCGI serializes it to a
// writer in the CGI response format (status + headers + blank line + body).
type CGIResult struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// ServeCGI reads a CGI invocation from the process environment and stdin,
// dispatches it through the wrapped agent handler, and writes the CGI-format
// response to stdout. It is the entry point a CGI-deployed agent's main()
// calls. Returns an error only for an unrecoverable I/O failure; an agent-level
// error (401, 500, …) is a normal response, not a Go error.
//
// Mirrors Python's ServerlessMixin CGI branch: PATH_INFO selects the route
// (empty → SWML at root), REQUEST_METHOD / CONTENT_LENGTH / QUERY_STRING
// reconstruct the request, and the body is read from stdin up to
// CONTENT_LENGTH (bounded by MaxCGIBodySize).
func (s *Handler) ServeCGI(ctx context.Context) error {
	res, err := s.DispatchCGI(ctx, cgiEnv(), os.Stdin)
	if err != nil {
		return err
	}
	return WriteCGI(os.Stdout, res)
}

// DispatchCGI builds a synthetic request from a CGI environment map + body
// reader, serves it through the wrapped handler, and returns the CGIResult.
// Exposed (rather than only ServeCGI) so it can be exercised in tests without
// touching the real process environment or stdout.
func (s *Handler) DispatchCGI(ctx context.Context, env map[string]string, stdin io.Reader) (*CGIResult, error) {
	method := env["REQUEST_METHOD"]
	if method == "" {
		method = http.MethodGet
	}

	// PATH_INFO selects the route; Python strips the surrounding slashes and
	// treats empty as "render SWML at root". Reconstruct a leading-slash path
	// the agent router matches.
	pathInfo := strings.Trim(env["PATH_INFO"], "/")
	target := "/" + pathInfo
	if q := env["QUERY_STRING"]; q != "" {
		target += "?" + q
	}

	u, err := url.ParseRequestURI(target)
	if err != nil {
		return nil, fmt.Errorf("serverless cgi: invalid request URI %q: %w", target, err)
	}

	// Read the body from stdin bounded by CONTENT_LENGTH (and the hard cap).
	var body io.Reader = bytes.NewReader(nil)
	if clStr := env["CONTENT_LENGTH"]; clStr != "" {
		cl, convErr := strconv.Atoi(clStr)
		if convErr != nil || cl < 0 {
			return nil, fmt.Errorf("serverless cgi: invalid CONTENT_LENGTH %q", clStr)
		}
		if cl > MaxCGIBodySize {
			return &CGIResult{
				StatusCode: http.StatusRequestEntityTooLarge,
				Header:     http.Header{"Content-Type": {"application/json"}},
				Body:       []byte(fmt.Sprintf(`{"error":"Request body too large","max_size":%d}`, MaxCGIBodySize)),
			}, nil
		}
		if cl > 0 && stdin != nil {
			buf := make([]byte, cl)
			n, readErr := io.ReadFull(stdin, buf)
			if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
				return nil, fmt.Errorf("serverless cgi: reading body: %w", readErr)
			}
			body = bytes.NewReader(buf[:n])
		}
	}

	r, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("serverless cgi: NewRequest failed: %w", err)
	}
	r.RequestURI = ""

	// Forward CGI HTTP_* headers and the well-known content headers so the
	// agent's auth + JSON handling see a real-looking request.
	if ct := env["CONTENT_TYPE"]; ct != "" {
		r.Header.Set("Content-Type", ct)
	}
	for k, v := range env {
		if name, ok := strings.CutPrefix(k, "HTTP_"); ok {
			r.Header.Set(cgiHeaderName(name), v)
		}
	}

	rec := httptest.NewRecorder()
	s.h.ServeHTTP(rec, r)

	return &CGIResult{
		StatusCode: rec.Code,
		Header:     rec.Header(),
		Body:       rec.Body.Bytes(),
	}, nil
}

// WriteCGI serializes a CGIResult to w in CGI response format: a Status header,
// each response header, a blank line, then the body.
func WriteCGI(w io.Writer, res *CGIResult) error {
	bw := bufio.NewWriter(w)
	if _, err := fmt.Fprintf(bw, "Status: %d %s\r\n", res.StatusCode, http.StatusText(res.StatusCode)); err != nil {
		return err
	}
	for k, vs := range res.Header {
		for _, v := range vs {
			if _, err := fmt.Fprintf(bw, "%s: %s\r\n", k, v); err != nil {
				return err
			}
		}
	}
	if _, err := io.WriteString(bw, "\r\n"); err != nil {
		return err
	}
	if _, err := bw.Write(res.Body); err != nil {
		return err
	}
	return bw.Flush()
}

// cgiEnv snapshots the process environment into a map for DispatchCGI.
func cgiEnv() map[string]string {
	env := os.Environ()
	m := make(map[string]string, len(env))
	for _, kv := range env {
		if k, v, ok := strings.Cut(kv, "="); ok {
			m[k] = v
		}
	}
	return m
}

// cgiHeaderName converts a CGI HTTP_ variable suffix (e.g. "X_CALL_ID") back
// into a canonical HTTP header name ("X-Call-Id").
func cgiHeaderName(cgiName string) string {
	return http.CanonicalHeaderKey(strings.ReplaceAll(cgiName, "_", "-"))
}
