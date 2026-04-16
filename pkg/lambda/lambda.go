// Package lambda adapts a net/http handler produced by the SignalWire
// Agents SDK (typically agent.AsRouter()) so it can run inside AWS Lambda.
//
// The adapter translates a Lambda invocation event into a synthetic
// *http.Request, runs the underlying handler against an in-memory response
// recorder, and marshals the recorder's result back into the Lambda
// response type that matches the invoking event.
//
// Two event shapes are supported, because they are the two ways an AI
// agent is realistically exposed over HTTP on Lambda:
//
//   - Lambda Function URLs (events.LambdaFunctionURLRequest /
//     LambdaFunctionURLResponse). This is the simplest deployment and
//     the one we recommend; no API Gateway is required.
//
//   - API Gateway HTTP API v2 (events.APIGatewayV2HTTPRequest /
//     APIGatewayV2HTTPResponse). Identical payload shape in practice,
//     but consumers with an API Gateway layer want the exact response
//     type rather than an assignable alias.
//
// The classic REST API (v1) payload is intentionally not supported as a
// first-class path to keep this package small. Users who need v1 can wrap
// it via github.com/awslabs/aws-lambda-go-api-proxy.
//
// # Usage
//
//	package main
//
//	import (
//	    "github.com/aws/aws-lambda-go/lambda"
//	    swlambda "github.com/signalwire/signalwire-go/pkg/lambda"
//	    "github.com/signalwire/signalwire-go/pkg/agent"
//	)
//
//	var a = agent.NewAgentBase(agent.WithName("MyAgent"), agent.WithRoute("/my-agent"))
//	var handler = swlambda.NewHandler(a.AsRouter())
//
//	func main() {
//	    lambda.Start(handler.HandleFunctionURL)
//	}
//
// Because agent.AsRouter() installs routes relative to the agent's Route
// (e.g. /my-agent, /my-agent/swaig), the Lambda event's RawPath must line
// up with that Route. Lambda Function URLs preserve the full request path
// unchanged, so no rewriting is needed in the common case.
package lambda

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/aws/aws-lambda-go/events"
)

// Handler wraps an http.Handler so it can service AWS Lambda invocations.
// Construct one with NewHandler.
type Handler struct {
	h http.Handler
}

// NewHandler returns a Handler that dispatches Lambda events to the given
// http.Handler. The handler is usually agent.AsRouter() but any
// http.Handler works.
//
// Panics if h is nil to fail loudly at cold-start rather than silently
// returning 500 on every invocation.
func NewHandler(h http.Handler) *Handler {
	if h == nil {
		panic("lambda.NewHandler: http.Handler must not be nil")
	}
	return &Handler{h: h}
}

// HandleFunctionURL processes a Lambda Function URL invocation. It is
// intended to be passed to github.com/aws/aws-lambda-go/lambda.Start.
func (l *Handler) HandleFunctionURL(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	r, err := l.buildRequest(ctx, req.RequestContext.HTTP.Method, req.RawPath, req.RawQueryString, req.Headers, req.Cookies, req.Body, req.IsBase64Encoded)
	if err != nil {
		return events.LambdaFunctionURLResponse{}, err
	}

	rec := httptest.NewRecorder()
	l.h.ServeHTTP(rec, r)

	body, isBase64 := encodeBody(rec.Body.Bytes())
	headers, cookies := splitCookies(rec.Header())
	return events.LambdaFunctionURLResponse{
		StatusCode:      rec.Code,
		Headers:         headers,
		Body:            body,
		IsBase64Encoded: isBase64,
		Cookies:         cookies,
	}, nil
}

// HandleAPIGatewayV2 processes an API Gateway HTTP API v2 invocation.
// The payload shape is virtually identical to Function URLs, but the
// response type differs, so we provide a dedicated entry point.
func (l *Handler) HandleAPIGatewayV2(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	r, err := l.buildRequest(ctx, req.RequestContext.HTTP.Method, req.RawPath, req.RawQueryString, req.Headers, req.Cookies, req.Body, req.IsBase64Encoded)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	rec := httptest.NewRecorder()
	l.h.ServeHTTP(rec, r)

	body, isBase64 := encodeBody(rec.Body.Bytes())
	headers, cookies := splitCookies(rec.Header())
	return events.APIGatewayV2HTTPResponse{
		StatusCode:      rec.Code,
		Headers:         headers,
		Body:            body,
		IsBase64Encoded: isBase64,
		Cookies:         cookies,
	}, nil
}

// buildRequest constructs a synthetic *http.Request from Lambda event
// fields that the underlying http.Handler can serve against.
//
// The reconstruction is lossy in one intentional way: Method is taken
// from the event's HTTP block, not from headers, because that is where
// AWS puts it. Everything else is forwarded verbatim.
func (l *Handler) buildRequest(
	ctx context.Context,
	method, rawPath, rawQuery string,
	headers map[string]string,
	cookies []string,
	body string,
	isBase64 bool,
) (*http.Request, error) {
	if method == "" {
		method = http.MethodGet
	}
	if rawPath == "" {
		rawPath = "/"
	}

	// Reconstruct the target URL. We deliberately use a fake host because
	// the handler doesn't use r.Host for routing — ServeMux uses r.URL.Path.
	target := rawPath
	if rawQuery != "" {
		target = rawPath + "?" + rawQuery
	}
	u, err := url.ParseRequestURI(target)
	if err != nil {
		return nil, fmt.Errorf("lambda adapter: invalid request URI %q: %w", target, err)
	}

	// Decode body. Lambda delivers binary data (and sometimes text) with
	// isBase64Encoded=true; raw JSON comes through as a plain string.
	// We always pass a non-nil reader so handlers that unconditionally
	// call io.ReadAll(r.Body) do not nil-deref on empty payloads.
	var bodyReader io.Reader = bytes.NewReader(nil)
	if body != "" {
		if isBase64 {
			raw, decErr := base64.StdEncoding.DecodeString(body)
			if decErr != nil {
				return nil, fmt.Errorf("lambda adapter: base64 body decode failed: %w", decErr)
			}
			bodyReader = bytes.NewReader(raw)
		} else {
			bodyReader = strings.NewReader(body)
		}
	}

	r, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("lambda adapter: NewRequest failed: %w", err)
	}

	// Populate headers. Lambda normalises header names to lowercase; Go's
	// http package canonicalises on Set, so the resulting r.Header is
	// indistinguishable from a real HTTP request.
	for k, v := range headers {
		r.Header.Set(k, v)
	}

	// Function URLs put cookies in a separate "cookies" array rather than
	// a combined Cookie header. Fold them back into the header so handlers
	// that read r.Cookies() see what they expect.
	if len(cookies) > 0 {
		if existing := r.Header.Get("Cookie"); existing != "" {
			r.Header.Set("Cookie", existing+"; "+strings.Join(cookies, "; "))
		} else {
			r.Header.Set("Cookie", strings.Join(cookies, "; "))
		}
	}

	// Ensure RequestURI is empty for client-style requests, which is
	// what ServeHTTP expects on synthetic requests. http.NewRequest
	// already leaves it empty; this is a belt-and-braces reminder.
	r.RequestURI = ""

	return r, nil
}

// splitCookies extracts Set-Cookie header values into the dedicated
// cookies slice that Lambda Function URL / API Gateway V2 responses
// expect, and returns the remaining headers flattened into the
// single-value map the Lambda response schema requires.
//
// Combining multiple Set-Cookie headers into a single comma-joined
// string would break browsers; the Lambda response types expose a
// separate Cookies field precisely so the proxy can emit them as
// multiple Set-Cookie lines.
func splitCookies(h http.Header) (map[string]string, []string) {
	out := make(map[string]string, len(h))
	var cookies []string
	for k, v := range h {
		if len(v) == 0 {
			continue
		}
		if http.CanonicalHeaderKey(k) == "Set-Cookie" {
			cookies = append(cookies, v...)
			continue
		}
		// For all other headers, joining multiple values with ", " is
		// the canonical HTTP combining rule.
		out[k] = strings.Join(v, ", ")
	}
	return out, cookies
}

// encodeBody returns the response body in a format Lambda accepts. If the
// body is valid UTF-8 it is returned verbatim; otherwise it is base64
// encoded and the second return value is set to true. Binary responses
// (audio, images, etc.) are rare for this SDK — SWML is always JSON — but
// handling them correctly is cheap.
func encodeBody(b []byte) (string, bool) {
	if utf8.Valid(b) {
		return string(b), false
	}
	return base64.StdEncoding.EncodeToString(b), true
}
