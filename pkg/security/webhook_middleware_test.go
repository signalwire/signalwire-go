package security

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Middleware tests — POST /webhook with X-SignalWire-Signature header
// ---------------------------------------------------------------------------

func TestMiddleware_AcceptsValidSignature(t *testing.T) {
	signingKey := "PSKtest1234567890abcdef"
	body := vectorABody
	sigURL := vectorAURL
	expected := vectorAExpect

	// Sanity check: vector matches our validator.
	if !ValidateWebhookSignature(signingKey, expected, sigURL, body) {
		t.Fatalf("setup error: vector A doesn't validate")
	}

	called := false
	var bodyOnHandler []byte

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		// Confirm raw body is forwarded — handler can re-read it.
		read, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("handler couldn't read body: %v", err)
		}
		bodyOnHandler = read

		// Confirm context-stashed body is also accessible.
		if ctxBody, ok := RawBodyFromContext(r.Context()); !ok {
			t.Errorf("RawBodyFromContext: expected ok=true")
		} else if string(ctxBody) != body {
			t.Errorf("RawBodyFromContext: got %q, want %q", ctxBody, body)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mw := WebhookMiddleware(signingKey, nil)
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodPost, sigURL, strings.NewReader(body))
	req.Header.Set("X-SignalWire-Signature", expected)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if !called {
		t.Errorf("handler not called on valid signature")
	}
	if string(bodyOnHandler) != body {
		t.Errorf("body forwarded to handler = %q, want %q", bodyOnHandler, body)
	}
}

func TestMiddleware_RejectsInvalidSignature(t *testing.T) {
	signingKey := "PSKtest1234567890abcdef"

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	mw := WebhookMiddleware(signingKey, nil)
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodPost, vectorAURL, strings.NewReader(vectorABody))
	req.Header.Set("X-SignalWire-Signature", "deadbeef")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
	if called {
		t.Errorf("handler unexpectedly called on invalid signature")
	}
}

func TestMiddleware_RejectsMissingSignatureHeader(t *testing.T) {
	signingKey := "PSKtest1234567890abcdef"
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := WebhookMiddleware(signingKey, nil)
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodPost, vectorAURL, strings.NewReader(vectorABody))
	// no X-SignalWire-Signature
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
	if called {
		t.Errorf("handler unexpectedly called when signature header missing")
	}
}

func TestMiddleware_AcceptsTwilioAliasHeader(t *testing.T) {
	// X-Twilio-Signature alias should be honored for cXML compat.
	signingKey := vectorBKey
	body := formEncode(vectorBParams())
	sigURL := vectorBURL
	expected := vectorBExpect

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := WebhookMiddleware(signingKey, nil)
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodPost, sigURL, strings.NewReader(body))
	req.Header.Set("X-Twilio-Signature", expected)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if !called {
		t.Errorf("handler not called when X-Twilio-Signature is valid")
	}
}

func TestMiddleware_TrustProxyReconstructsURL(t *testing.T) {
	// Behind a reverse proxy: r.Host=internal:8080, r.TLS=nil; X-Forwarded-*
	// gives the public URL the platform actually signed.
	signingKey := vectorAKey
	body := vectorABody
	// Reference URL is vectorAURL ("https://example.ngrok.io/webhook") —
	// what the platform signed; the X-Forwarded-* headers below should
	// rehydrate it from the internal request.
	expected := vectorAExpect

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := WebhookMiddleware(signingKey, &WebhookOpts{TrustProxy: true})
	wrapped := mw(handler)

	// Internal request hits us as plain HTTP on a different hostname.
	req := httptest.NewRequest(http.MethodPost, "http://internal-host:8080/webhook", strings.NewReader(body))
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "example.ngrok.io")
	req.Header.Set("X-SignalWire-Signature", expected)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("trust-proxy URL reconstruction: status = %d, want 200", rec.Code)
	}
}

func TestMiddleware_TrustProxyDisabledIgnoresHeaders(t *testing.T) {
	// Same forwarded headers as above — without TrustProxy they're ignored
	// and the validator sees the internal URL ⇒ signature mismatch ⇒ 403.
	signingKey := vectorAKey
	expected := vectorAExpect

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := WebhookMiddleware(signingKey, nil)
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodPost, "http://internal-host:8080/webhook", strings.NewReader(vectorABody))
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "example.ngrok.io")
	req.Header.Set("X-SignalWire-Signature", expected)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("trust-proxy disabled: status = %d, want 403", rec.Code)
	}
}

func TestMiddleware_ProxyURLBaseOverride(t *testing.T) {
	signingKey := vectorAKey
	expected := vectorAExpect

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := WebhookMiddleware(signingKey, &WebhookOpts{
		ProxyURLBase: "https://example.ngrok.io",
	})
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodPost, "http://anything-else/webhook", strings.NewReader(vectorABody))
	req.Header.Set("X-SignalWire-Signature", expected)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("ProxyURLBase: status = %d, want 200", rec.Code)
	}
}

func TestMiddleware_PanicsOnEmptySigningKey(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on empty signing key")
		}
	}()
	_ = WebhookMiddleware("", nil)
}

func TestMiddleware_BodyForwardedReReadable(t *testing.T) {
	// The middleware reads r.Body once — the wrapped handler must still see
	// the bytes via either r.Body (we restore it via NopCloser) or via
	// RawBodyFromContext.
	signingKey := vectorAKey
	body := vectorABody
	expected := vectorAExpect

	var fromBody, fromContext []byte
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fromBody, _ = io.ReadAll(r.Body)
		if ctxBody, ok := RawBodyFromContext(r.Context()); ok {
			fromContext = ctxBody
		}
		w.WriteHeader(http.StatusOK)
	})

	mw := WebhookMiddleware(signingKey, nil)
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodPost, vectorAURL, bytes.NewBufferString(body))
	req.Header.Set("X-SignalWire-Signature", expected)
	wrapped.ServeHTTP(httptest.NewRecorder(), req)

	if string(fromBody) != body {
		t.Errorf("body from r.Body = %q, want %q", fromBody, body)
	}
	if string(fromContext) != body {
		t.Errorf("body from context = %q, want %q", fromContext, body)
	}
}

func TestMiddleware_MaxBodyBytesEnforced(t *testing.T) {
	signingKey := "key"
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := WebhookMiddleware(signingKey, &WebhookOpts{MaxBodyBytes: 16})
	wrapped := mw(handler)

	// 17 bytes > 16-byte cap.
	big := strings.Repeat("x", 17)
	req := httptest.NewRequest(http.MethodPost, "https://example.com/", strings.NewReader(big))
	req.Header.Set("X-SignalWire-Signature", "anything")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
	if called {
		t.Errorf("handler unexpectedly called when body exceeds MaxBodyBytes")
	}
}

// ---------------------------------------------------------------------------
// Decomposed validate-core tests — security.Validate(method,url,headers,body,
// signingKey) -> *WebhookRejection (nil=pass, triple=reject). This is the
// framework-free cross-port contract (signalwire.core.security.
// webhook_middleware.validate); the http.Handler middleware above is Go's
// framework wrapper over it.
// ---------------------------------------------------------------------------

// TestValidate_ValidSignaturePasses proves a valid Scheme-A signature makes the
// decomposed core return nil ("pass") — the canonical webhooks.md vector.
func TestValidate_ValidSignaturePasses(t *testing.T) {
	headers := map[string]string{"X-SignalWire-Signature": vectorAExpect}
	rej := Validate(http.MethodPost, vectorAURL, headers, vectorABody, vectorAKey)
	if rej != nil {
		t.Fatalf("Validate returned rejection %+v for a valid signature; want nil (pass)", rej)
	}
}

// TestValidate_BadSignatureRejects proves a tampered/wrong signature yields the
// 403 rejection triple (status, headers, body) rather than nil.
func TestValidate_BadSignatureRejects(t *testing.T) {
	headers := map[string]string{"X-SignalWire-Signature": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"}
	rej := Validate(http.MethodPost, vectorAURL, headers, vectorABody, vectorAKey)
	if rej == nil {
		t.Fatal("Validate returned nil (pass) for a bad signature; want a 403 rejection")
	}
	if rej.Status != http.StatusForbidden {
		t.Errorf("rejection status = %d, want 403", rej.Status)
	}
	if rej.Body == "" {
		t.Error("rejection body is empty; want a non-empty (detail-free) body")
	}
	// The rejection must never leak the expected signature or key.
	if strings.Contains(rej.Body, vectorAKey) || strings.Contains(rej.Body, vectorAExpect) {
		t.Errorf("rejection body leaks secret/expected signature: %q", rej.Body)
	}
}

// TestValidate_MissingSignatureRejects proves an absent signature header is a
// 403 rejection (never a panic, never a pass).
func TestValidate_MissingSignatureRejects(t *testing.T) {
	rej := Validate(http.MethodPost, vectorAURL, map[string]string{}, vectorABody, vectorAKey)
	if rej == nil {
		t.Fatal("Validate returned nil for a missing signature header; want a 403 rejection")
	}
	if rej.Status != http.StatusForbidden {
		t.Errorf("rejection status = %d, want 403", rej.Status)
	}
}

// TestValidate_TwilioSignatureAliasHonored proves the legacy X-Twilio-Signature
// header alias is accepted by the decomposed core (cXML compatibility,
// webhooks.md §"The Header").
func TestValidate_TwilioSignatureAliasHonored(t *testing.T) {
	headers := map[string]string{"X-Twilio-Signature": vectorAExpect}
	rej := Validate(http.MethodPost, vectorAURL, headers, vectorABody, vectorAKey)
	if rej != nil {
		t.Fatalf("Validate rejected a valid X-Twilio-Signature alias: %+v; want nil (pass)", rej)
	}
}

// TestValidate_HeaderLookupCaseInsensitive proves the primitive string-map
// header lookup is case-insensitive (HTTP header names are), so a lowercased
// signature key still validates.
func TestValidate_HeaderLookupCaseInsensitive(t *testing.T) {
	headers := map[string]string{"x-signalwire-signature": vectorAExpect}
	rej := Validate(http.MethodPost, vectorAURL, headers, vectorABody, vectorAKey)
	if rej != nil {
		t.Fatalf("Validate rejected a case-variant header key: %+v; want nil (pass)", rej)
	}
}

// TestValidate_MissingSigningKeyPanics proves a missing signing key is treated
// as a programmer error (panic), matching the bool-returning validators — a
// missing key is never a silent pass.
func TestValidate_MissingSigningKeyPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Validate did not panic on an empty signing key; want a panic")
		}
	}()
	headers := map[string]string{"X-SignalWire-Signature": vectorAExpect}
	Validate(http.MethodPost, vectorAURL, headers, vectorABody, "")
}
