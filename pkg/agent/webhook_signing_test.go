package agent

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// hexSig is the Scheme-A signing helper duplicated here so the agent test
// doesn't reach across packages for unexported helpers — keeps this file
// independent and easy to read.
func hexSigA(t *testing.T, key, url, body string) string {
	t.Helper()
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(url + body))
	return hex.EncodeToString(mac.Sum(nil))
}

// signedRequest builds a POST httptest.Request with valid X-SignalWire-Signature
// for the given key/path/body — the URL the validator will reconstruct is
// http://<r.Host><path> (no proxy headers, no TLS).
func signedRequest(t *testing.T, key, host, path, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "http://"+host+path, strings.NewReader(body))
	url := "http://" + host + path
	req.Header.Set("X-SignalWire-Signature", hexSigA(t, key, url, body))
	req.Host = host
	return req
}

// ---------------------------------------------------------------------------
// SigningKey integration with AgentBase mux
// ---------------------------------------------------------------------------

func TestAgentSigning_SignedRequestAccepted(t *testing.T) {
	const key = "PSKtestintegration"
	a := NewAgentBase(
		WithBasicAuth("u", "p"),
		WithSigningKey(key),
	)
	mux := a.buildMux()

	body := `{"event":"swml.request"}`
	req := signedRequest(t, key, "agent.test", "/", body)
	req.SetBasicAuth("u", "p")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	// SWML endpoint serves a JSON document on POST — any 2xx is fine.
	if rec.Code >= 400 {
		t.Errorf("signed POST rejected: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestAgentSigning_UnsignedRequestRejected(t *testing.T) {
	a := NewAgentBase(
		WithBasicAuth("u", "p"),
		WithSigningKey("PSKtestintegration"),
	)
	mux := a.buildMux()

	req := httptest.NewRequest(http.MethodPost, "http://agent.test/", strings.NewReader("{}"))
	req.SetBasicAuth("u", "p")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("unsigned POST: status=%d, want 403", rec.Code)
	}
}

func TestAgentSigning_WrongSignatureRejected(t *testing.T) {
	a := NewAgentBase(
		WithBasicAuth("u", "p"),
		WithSigningKey("PSKtestintegration"),
	)
	mux := a.buildMux()

	req := httptest.NewRequest(http.MethodPost, "http://agent.test/", strings.NewReader(`{"x":1}`))
	req.SetBasicAuth("u", "p")
	req.Header.Set("X-SignalWire-Signature", "deadbeef")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("wrong sig POST: status=%d, want 403", rec.Code)
	}
}

func TestAgentSigning_GETPassthroughWhenSigned(t *testing.T) {
	// GET on the SWML route serves the document for browser-style fetches;
	// per the spec only POST is signed by the platform. GET should not be
	// gated by the signature middleware.
	a := NewAgentBase(
		WithBasicAuth("u", "p"),
		WithSigningKey("PSKtestintegration"),
	)
	mux := a.buildMux()

	req := httptest.NewRequest(http.MethodGet, "http://agent.test/", nil)
	req.SetBasicAuth("u", "p")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code >= 400 {
		t.Errorf("GET should not be signature-gated: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestAgentSigning_SwaigEndpointGated(t *testing.T) {
	a := NewAgentBase(
		WithBasicAuth("u", "p"),
		WithSigningKey("PSKtestintegration"),
	)
	mux := a.buildMux()

	// Unsigned POST to /swaig
	req := httptest.NewRequest(http.MethodPost, "http://agent.test/swaig", strings.NewReader(`{"function":"x"}`))
	req.SetBasicAuth("u", "p")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("unsigned /swaig: status=%d, want 403", rec.Code)
	}
}

func TestAgentSigning_PostPromptEndpointGated(t *testing.T) {
	a := NewAgentBase(
		WithBasicAuth("u", "p"),
		WithSigningKey("PSKtestintegration"),
	)
	mux := a.buildMux()

	req := httptest.NewRequest(http.MethodPost, "http://agent.test/post_prompt", strings.NewReader(`{}`))
	req.SetBasicAuth("u", "p")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("unsigned /post_prompt: status=%d, want 403", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Env fallback
// ---------------------------------------------------------------------------

func TestAgentSigning_EnvFallback(t *testing.T) {
	const envKey = "PSKfromenv0123"
	t.Setenv("SIGNALWIRE_SIGNING_KEY", envKey)

	a := NewAgentBase(WithBasicAuth("u", "p"))
	if a.signingKey != envKey {
		t.Fatalf("env fallback: signingKey = %q, want %q", a.signingKey, envKey)
	}
	mux := a.buildMux()

	req := signedRequest(t, envKey, "agent.test", "/", `{"a":1}`)
	req.SetBasicAuth("u", "p")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code >= 400 {
		t.Errorf("env-key signed POST: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestAgentSigning_ExplicitOverridesEnv(t *testing.T) {
	t.Setenv("SIGNALWIRE_SIGNING_KEY", "envkey")
	a := NewAgentBase(WithSigningKey("explicit-wins"))
	if a.signingKey != "explicit-wins" {
		t.Errorf("explicit signingKey should win over env, got %q", a.signingKey)
	}
}

// ---------------------------------------------------------------------------
// Disabled-validation warning
// ---------------------------------------------------------------------------

func TestAgentSigning_DisabledWarningLogged(t *testing.T) {
	// Make sure neither explicit nor env key is present.
	os.Unsetenv("SIGNALWIRE_SIGNING_KEY")

	captured := captureStderr(t, func() {
		a := NewAgentBase()
		if a.signingKey != "" {
			t.Fatalf("expected unset signingKey, got %q", a.signingKey)
		}
	})

	if !strings.Contains(captured, "webhook signature validation is disabled") {
		t.Errorf("expected disabled-validation warning in stderr, got: %q", captured)
	}
}

func TestAgentSigning_NoWarningWhenKeyPresent(t *testing.T) {
	os.Unsetenv("SIGNALWIRE_SIGNING_KEY")

	captured := captureStderr(t, func() {
		_ = NewAgentBase(WithSigningKey("present"))
	})

	if strings.Contains(captured, "webhook signature validation is disabled") {
		t.Errorf("did not expect warning when key configured, got: %q", captured)
	}
}

// ---------------------------------------------------------------------------
// Disabled => no signature gate (passthrough)
// ---------------------------------------------------------------------------

func TestAgentSigning_DisabledAllowsUnsigned(t *testing.T) {
	os.Unsetenv("SIGNALWIRE_SIGNING_KEY")

	a := NewAgentBase(WithBasicAuth("u", "p"))
	if a.signingKey != "" {
		t.Fatalf("expected disabled signingKey, got %q", a.signingKey)
	}
	mux := a.buildMux()

	req := httptest.NewRequest(http.MethodPost, "http://agent.test/", strings.NewReader(`{}`))
	req.SetBasicAuth("u", "p")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	// No 403 — without a signing key, the gate is a passthrough.
	if rec.Code == http.StatusForbidden {
		t.Errorf("disabled signature gate should not 403, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Log-capture helper
//
// pkg/logging writes via log.New(os.Stderr, ...) at logger-construction
// time, so we redirect os.Stderr through an os.Pipe BEFORE running fn and
// restore the original Stderr afterwards. The reader-goroutine drains the
// pipe so writers don't block; we collect the output once fn returns.
// ---------------------------------------------------------------------------

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	origStderr := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = origStderr }()

	done := make(chan string)
	go func() {
		buf, _ := io.ReadAll(r)
		done <- string(buf)
	}()

	fn()
	w.Close()
	captured := <-done
	r.Close()
	return captured
}
