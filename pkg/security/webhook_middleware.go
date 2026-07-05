// Package security — webhook signature middleware.
//
// WebhookMiddleware wraps an http.Handler with the signature-validation
// gate from porting-sdk/webhooks.md, section "Framework adapter":
//
//  1. Reads the raw body and caches it on the request context (so the
//     downstream handler can re-parse without re-reading the stream).
//  2. Pulls X-SignalWire-Signature (or the X-Twilio-Signature alias).
//  3. Reconstructs the public URL the platform POSTed to, honoring
//     X-Forwarded-* headers when TrustProxy is enabled and falling back to
//     the raw r.URL otherwise.
//  4. Calls ValidateWebhookSignature.
//  5. On failure: 403 Forbidden, no body detail, downstream not invoked.
//  6. On success: stash the raw body bytes on r.Context() under the unique
//     key returned by RawBodyContextKey so the downstream handler can
//     access them via RawBodyFromContext without re-reading.

package security

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strings"
)

// WebhookRejection is the framework-free rejection triple the decomposed
// Validate core returns when an inbound signed request fails validation.
// It mirrors the cross-port decomposed contract
// signalwire.core.security.webhook_middleware.validate, whose return is
// optional<tuple<int,dict<string,string>,string>> — a nil *WebhookRejection
// means "pass" (the request is authentic), a non-nil value carries the
// (status, headers, body) an HTTP layer should send back to reject.
//
// This is the SAME shape dotnet ships as WebhookValidationMiddleware.Validate
// and that Rack/PSGI middleware responses literally are — the decision core
// every port's framework wrapper is built on. The http.Handler middleware
// (WebhookMiddleware, below) is Go's framework wrapper idiom over this core.
type WebhookRejection struct {
	// Status is the HTTP status code to return (403 for a failed signature).
	Status int
	// Headers are the response headers to send with the rejection.
	Headers map[string]string
	// Body is the response body (kept detail-free — never leaks which branch
	// failed, which scheme was tried, or the expected signature).
	Body string
}

// Validate is the framework-free decomposed webhook-validation core
// (cross-port contract signalwire.core.security.webhook_middleware.validate):
// given the primitives of an inbound HTTP request — method, the full public
// url, the request headers, and the raw body — it returns nil when the
// request carries a valid SignalWire signature ("pass"), or a
// *WebhookRejection carrying the (status, headers, body) an HTTP layer should
// send to reject the request.
//
// It pulls X-SignalWire-Signature (falling back to the legacy
// X-Twilio-Signature alias for cXML compatibility, per webhooks.md §"The
// Header"), then delegates the actual HMAC check to ValidateWebhookSignatureE.
// A missing header or a bad signature both reject with 403 and no detail; a
// missing signing key is a programmer error and panics (matching the
// bool-returning validator entry points).
//
// The framework middleware WebhookMiddleware is a thin http.Handler wrapper
// over this core; other embeddings (a serverless handler, a manual dispatch)
// can call Validate directly with the four primitives.
func Validate(method, url string, headers map[string]string, body string, signingKey string) *WebhookRejection {
	if signingKey == "" {
		panic(ErrMissingSigningKey)
	}

	sig := headerLookup(headers, "X-SignalWire-Signature")
	if sig == "" {
		sig = headerLookup(headers, "X-Twilio-Signature")
	}
	if sig == "" {
		return forbidden()
	}

	ok, err := ValidateWebhookSignatureE(signingKey, sig, url, body)
	if err != nil || !ok {
		return forbidden()
	}
	return nil
}

// forbidden builds the canonical 403 rejection triple. Body carries no branch
// detail, per webhooks.md §"Required SDK Behaviors / Error modes".
func forbidden() *WebhookRejection {
	return &WebhookRejection{
		Status:  http.StatusForbidden,
		Headers: map[string]string{"Content-Type": "text/plain; charset=utf-8"},
		Body:    "Forbidden",
	}
}

// headerLookup does a case-insensitive header fetch over a plain string map
// (the decomposed core takes primitives, not an http.Header). HTTP header
// names are case-insensitive, so a caller passing "x-signalwire-signature"
// still matches.
func headerLookup(headers map[string]string, name string) string {
	if v, ok := headers[name]; ok {
		return v
	}
	for k, v := range headers {
		if strings.EqualFold(k, name) {
			return v
		}
	}
	return ""
}

// rawBodyKey is an unexported type for the context key, preventing
// cross-package collisions per the http.Server docs guidance.
type rawBodyKey struct{}

// RawBodyContextKey is the context key under which WebhookMiddleware stashes
// the raw request body. Downstream handlers should retrieve via
// RawBodyFromContext rather than reaching for this key directly.
var RawBodyContextKey = rawBodyKey{}

// RawBodyFromContext returns the raw request body bytes that the webhook
// middleware captured before signature validation. Returns nil and false
// when called from a handler that wasn't wrapped by WebhookMiddleware (or
// when the request had no body).
func RawBodyFromContext(ctx context.Context) ([]byte, bool) {
	v := ctx.Value(RawBodyContextKey)
	if v == nil {
		return nil, false
	}
	b, ok := v.([]byte)
	return b, ok
}

// WebhookOpts configures WebhookMiddleware.
type WebhookOpts struct {
	// TrustProxy makes the middleware honor X-Forwarded-Proto and
	// X-Forwarded-Host when reconstructing the URL passed to the validator.
	// Leave false (default) when the SDK terminates TLS itself; flip true
	// when running behind a reverse proxy / ngrok / load balancer.
	TrustProxy bool

	// MaxBodyBytes caps the body size the middleware will buffer before
	// signature validation. Zero (default) imposes no cap beyond Go's
	// own MaxBytesReader behavior. A small positive value protects against
	// memory exhaustion from oversized POSTs targeted at the gate.
	MaxBodyBytes int64

	// ProxyURLBase, when non-empty, overrides URL reconstruction entirely:
	// the validator sees ProxyURLBase + r.URL.RequestURI(). This matches
	// the SWML_PROXY_URL_BASE env-var override documented in the spec.
	// When empty, the env var is consulted at construction time.
	ProxyURLBase string
}

// WebhookMiddleware returns an http.Handler middleware that validates the
// X-SignalWire-Signature (or X-Twilio-Signature) header against signingKey
// before invoking the wrapped handler.
//
// Pass nil for opts to use defaults (no proxy trust, no size cap, env-var
// override consulted).
//
// The middleware never logs the signing key, the expected signature, or
// which validation branch matched — per porting-sdk/webhooks.md §"Required
// SDK Behaviors / Error modes".
func WebhookMiddleware(signingKey string, opts *WebhookOpts) func(http.Handler) http.Handler {
	if signingKey == "" {
		// Programmer error — surface immediately rather than silently
		// accepting all requests.
		panic(ErrMissingSigningKey)
	}
	resolved := WebhookOpts{}
	if opts != nil {
		resolved = *opts
	}
	if resolved.ProxyURLBase == "" {
		resolved.ProxyURLBase = os.Getenv("SWML_PROXY_URL_BASE")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Read full body up front — the validator needs the exact wire
			// bytes BEFORE any framework parser consumes them.
			var bodyReader io.Reader = r.Body
			if resolved.MaxBodyBytes > 0 {
				bodyReader = io.LimitReader(r.Body, resolved.MaxBodyBytes+1)
			}
			body, err := io.ReadAll(bodyReader)
			if err != nil {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			if resolved.MaxBodyBytes > 0 && int64(len(body)) > resolved.MaxBodyBytes {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			// Restore body for downstream handlers — Go consumes r.Body once.
			r.Body = io.NopCloser(bytes.NewReader(body))

			// Reconstruct the public URL the platform POSTed to.
			fullURL := reconstructURL(r, &resolved)

			// Delegate the decision to the framework-free decomposed core
			// (Validate). The middleware is Go's http.Handler wrapper idiom
			// over that core: it marshals the request into the four primitives
			// (method, url, headers, body), then either rejects with the
			// returned triple or forwards to the downstream handler.
			if rej := Validate(r.Method, fullURL, flattenHeaders(r.Header), string(body), signingKey); rej != nil {
				for k, v := range rej.Headers {
					w.Header().Set(k, v)
				}
				w.WriteHeader(rej.Status)
				_, _ = io.WriteString(w, rej.Body)
				return
			}

			// Stash raw body on context so downstream can re-parse without
			// touching r.Body again.
			ctx := context.WithValue(r.Context(), RawBodyContextKey, body)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// flattenHeaders collapses an http.Header (map[string][]string) into the
// plain map[string]string the decomposed Validate core consumes, taking the
// first value per key. The validator only reads the signature header, so
// first-value-wins is sufficient and matches how the platform sends a single
// signature.
func flattenHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, v := range h {
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	return out
}

// reconstructURL rebuilds the public URL the platform saw, honoring proxy
// headers when opts.TrustProxy is set. Mirrors the URL-reconstruction rules
// in porting-sdk/webhooks.md §"URL reconstruction behind proxies".
func reconstructURL(r *http.Request, opts *WebhookOpts) string {
	// Explicit override wins over everything.
	if opts != nil && opts.ProxyURLBase != "" {
		base := strings.TrimRight(opts.ProxyURLBase, "/")
		// r.URL on a server-side request doesn't carry scheme/host, only
		// path+query; RequestURI() does the right thing.
		return base + r.URL.RequestURI()
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host

	if opts != nil && opts.TrustProxy {
		if xfp := r.Header.Get("X-Forwarded-Proto"); xfp != "" {
			// Use the first value if comma-separated.
			scheme = strings.TrimSpace(strings.SplitN(xfp, ",", 2)[0])
		}
		if xfh := r.Header.Get("X-Forwarded-Host"); xfh != "" {
			host = strings.TrimSpace(strings.SplitN(xfh, ",", 2)[0])
		}
	}

	// r.URL on a server request has Path + RawQuery populated (no scheme/host).
	return scheme + "://" + host + r.URL.RequestURI()
}
