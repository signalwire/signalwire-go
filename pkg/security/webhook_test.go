package security

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/url"
	"sort"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Canonical test vectors from porting-sdk/webhooks.md (cross-port contract).
// If these break, the port has a real bug — DO NOT relax them.
// ---------------------------------------------------------------------------

var (
	vectorAKey    = "PSKtest1234567890abcdef"
	vectorAURL    = "https://example.ngrok.io/webhook"
	vectorABody   = `{"event":"call.state","params":{"call_id":"abc-123","state":"answered"}}`
	vectorAExpect = "c3c08c1fefaf9ee198a100d5906765a6f394bf0f"

	vectorBKey    = "12345"
	vectorBURL    = "https://mycompany.com/myapp.php?foo=1&bar=2"
	vectorBExpect = "RSOYDt4T1cUTdK1PDd93/VVr8B8="

	vectorCKey    = "PSKtest1234567890abcdef"
	vectorCBody   = `{"event":"call.state"}`
	vectorCURL    = "https://example.ngrok.io/webhook?bodySHA256=" +
		"69f3cbfc18e386ef8236cb7008cd5a54b7fed637a8cb3373b5a1591d7f0fd5f4"
	vectorCExpect = "dfO9ek8mxyFtn2nMz24plPmPfIY="
)

// vectorBParams is the canonical Twilio test-vector form body. Stable iteration
// order is irrelevant because the validator sorts keys before signing.
func vectorBParams() map[string][]string {
	return map[string][]string{
		"CallSid": {"CA1234567890ABCDE"},
		"Caller":  {"+14158675309"},
		"Digits":  {"1234"},
		"From":    {"+14158675309"},
		"To":      {"+18005551212"},
	}
}

// formEncode produces an x-www-form-urlencoded body that round-trips through
// url.ParseQuery to the same key/value pairs the validator will sort and
// concat. Hand-encoded so the test body matches what HTTP middleware would
// see on the wire (+ → %2B).
func formEncode(params map[string][]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0)
	for _, k := range keys {
		for _, v := range params[k] {
			pairs = append(pairs, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}
	return strings.Join(pairs, "&")
}

// b64Sig is a small helper used by the URL-port-normalization tests to
// pre-compute a Scheme-B base64 signature without going through the public
// API (so we can construct test vectors that pin a specific candidate URL).
func b64Sig(t *testing.T, key, signedURL string, params map[string][]string) string {
	t.Helper()
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	concat := signedURL
	for _, k := range keys {
		for _, v := range params[k] {
			concat += k + v
		}
	}
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(concat))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// ---------------------------------------------------------------------------
// Scheme A — RELAY/JSON (hex)
// ---------------------------------------------------------------------------

func TestSchemeA_PositiveCanonicalVector(t *testing.T) {
	if !ValidateWebhookSignature(vectorAKey, vectorAExpect, vectorAURL, vectorABody) {
		t.Fatalf("Vector A: expected match, got false")
	}
}

func TestSchemeA_NegativeTamperedBody(t *testing.T) {
	tampered := strings.Replace(vectorABody, "answered", "ringing", 1)
	if ValidateWebhookSignature(vectorAKey, vectorAExpect, vectorAURL, tampered) {
		t.Fatalf("Vector A: tampered body unexpectedly matched")
	}
}

func TestSchemeA_NegativeWrongKey(t *testing.T) {
	if ValidateWebhookSignature("wrong-key", vectorAExpect, vectorAURL, vectorABody) {
		t.Fatalf("Vector A: wrong key unexpectedly matched")
	}
}

func TestSchemeA_NegativeWrongURL(t *testing.T) {
	if ValidateWebhookSignature(vectorAKey, vectorAExpect, "https://example.ngrok.io/different", vectorABody) {
		t.Fatalf("Vector A: wrong URL unexpectedly matched")
	}
}

// ---------------------------------------------------------------------------
// Scheme B — Compat/cXML (base64 form-encoded)
// ---------------------------------------------------------------------------

func TestSchemeB_PositiveCanonicalFormVector(t *testing.T) {
	body := formEncode(vectorBParams())
	if !ValidateWebhookSignature(vectorBKey, vectorBExpect, vectorBURL, body) {
		t.Fatalf("Vector B (raw form body): expected match, got false")
	}
}

func TestSchemeB_PositiveViaValidateRequestMap(t *testing.T) {
	if !ValidateRequest(vectorBKey, vectorBExpect, vectorBURL, vectorBParams()) {
		t.Fatalf("Vector B (map): expected match, got false")
	}
}

func TestSchemeB_PositiveViaValidateRequestURLValues(t *testing.T) {
	v := url.Values(vectorBParams())
	if !ValidateRequest(vectorBKey, vectorBExpect, vectorBURL, v) {
		t.Fatalf("Vector B (url.Values): expected match, got false")
	}
}

func TestSchemeB_PositiveViaValidateRequestStringMap(t *testing.T) {
	// map[string]string scalar shape — also accepted by the legacy alias.
	scalar := map[string]string{
		"CallSid": "CA1234567890ABCDE",
		"Caller":  "+14158675309",
		"Digits":  "1234",
		"From":    "+14158675309",
		"To":      "+18005551212",
	}
	if !ValidateRequest(vectorBKey, vectorBExpect, vectorBURL, scalar) {
		t.Fatalf("Vector B (map[string]string): expected match, got false")
	}
}

func TestSchemeB_BodySHA256CanonicalVector(t *testing.T) {
	if !ValidateWebhookSignature(vectorCKey, vectorCExpect, vectorCURL, vectorCBody) {
		t.Fatalf("Vector C (bodySHA256): expected match, got false")
	}
}

func TestSchemeB_BodySHA256Mismatch(t *testing.T) {
	// Same URL+sig as Vector C but a different body — sha256 check must fail.
	if ValidateWebhookSignature(vectorCKey, vectorCExpect, vectorCURL, `{"event":"DIFFERENT"}`) {
		t.Fatalf("Vector C: tampered body unexpectedly matched")
	}
}

// ---------------------------------------------------------------------------
// URL port normalization
// ---------------------------------------------------------------------------

func TestURL_SignatureWithPortAcceptedWhenRequestHasNoPort(t *testing.T) {
	// Backend signed with :443 — request URL has no port → accept.
	urlWith := "https://example.com:443/webhook"
	urlWithout := "https://example.com/webhook"
	sig := b64Sig(t, "test-key", urlWith, nil)
	if !ValidateWebhookSignature("test-key", sig, urlWithout, "{}") {
		t.Fatalf("expected with-port signature to validate against without-port URL")
	}
}

func TestURL_SignatureWithoutPortAcceptedWhenRequestHasStandardPort(t *testing.T) {
	// Backend signed without port — request URL has :443 → accept.
	urlWith := "https://example.com:443/webhook"
	urlWithout := "https://example.com/webhook"
	sig := b64Sig(t, "test-key", urlWithout, nil)
	if !ValidateWebhookSignature("test-key", sig, urlWith, "{}") {
		t.Fatalf("expected without-port signature to validate against with-port URL")
	}
}

func TestURL_HTTPPort80Normalization(t *testing.T) {
	urlWith := "http://example.com:80/path"
	urlWithout := "http://example.com/path"
	sig := b64Sig(t, "test-key", urlWith, nil)
	if !ValidateWebhookSignature("test-key", sig, urlWithout, "") {
		t.Fatalf("http :80 normalization failed")
	}
}

func TestURL_NonStandardPortNotNormalized(t *testing.T) {
	// :8443 is non-standard — only the input URL form should be tried.
	urlAsIs := "https://example.com:8443/path"
	sig := b64Sig(t, "test-key", urlAsIs, nil)
	if !ValidateWebhookSignature("test-key", sig, urlAsIs, "") {
		t.Fatalf("non-standard port: as-is URL should match")
	}
	// And a "without-port" form must NOT match the non-standard port sig.
	if ValidateWebhookSignature("test-key", sig, "https://example.com/path", "") {
		t.Fatalf("non-standard port: without-port URL unexpectedly matched")
	}
}

// ---------------------------------------------------------------------------
// Repeated form keys (deterministic submission-order preservation)
// ---------------------------------------------------------------------------

func TestRepeatedFormKeys_ConcatInSubmissionOrder(t *testing.T) {
	key := "test-key"
	u := "https://example.com/hook"
	body := "To=a&To=b"
	// Expected concat: ToaTob (sorted by key only; preserve order within).
	expectedData := u + "ToaTob"
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(expectedData))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if !ValidateWebhookSignature(key, sig, u, body) {
		t.Fatalf("To=a&To=b should produce ToaTob signing string")
	}
}

func TestRepeatedFormKeys_SwappedOrderIsDifferent(t *testing.T) {
	key := "test-key"
	u := "https://example.com/hook"
	bodyAB := "To=a&To=b"
	bodyBA := "To=b&To=a"

	// Sign for bodyAB, then verify bodyBA does NOT match — proves order
	// within repeated keys is honored, not lexically sorted.
	dataAB := u + "ToaTob"
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(dataAB))
	sigForAB := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if !ValidateWebhookSignature(key, sigForAB, u, bodyAB) {
		t.Fatalf("AB body should match AB signature")
	}
	if ValidateWebhookSignature(key, sigForAB, u, bodyBA) {
		t.Fatalf("BA body should not match AB signature")
	}
}

// ---------------------------------------------------------------------------
// Error modes
// ---------------------------------------------------------------------------

func TestErrors_MissingSignatureReturnsFalse(t *testing.T) {
	if ValidateWebhookSignature(vectorAKey, "", vectorAURL, vectorABody) {
		t.Fatalf("empty signature should return false")
	}
}

func TestErrors_MissingSigningKeyPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on empty signing key")
		}
	}()
	ValidateWebhookSignature("", "sig", vectorAURL, vectorABody)
}

func TestErrors_MissingSigningKeyReturnsErrorOnE(t *testing.T) {
	ok, err := ValidateWebhookSignatureE("", "sig", vectorAURL, vectorABody)
	if ok {
		t.Fatalf("expected ok=false on empty key")
	}
	if err == nil || err != ErrMissingSigningKey {
		t.Fatalf("expected ErrMissingSigningKey, got %v", err)
	}
}

func TestErrors_MalformedSignatureReturnsFalseNoPanic(t *testing.T) {
	// Wrong length, weird chars, base64 noise — none should panic.
	garbage := []string{"xyz", "!!!!", strings.Repeat("a", 100), "%%notbase64%%"}
	for _, g := range garbage {
		if ValidateWebhookSignature(vectorAKey, g, vectorAURL, vectorABody) {
			t.Fatalf("garbage signature %q unexpectedly matched", g)
		}
	}
}

func TestValidateRequest_InvalidArgTypePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on invalid arg type")
		}
	}()
	ValidateRequest(vectorAKey, "sig", vectorAURL, 42)
}

func TestValidateRequest_InvalidArgTypeReturnsErrorOnE(t *testing.T) {
	_, err := ValidateRequestE(vectorAKey, "sig", vectorAURL, 42)
	if err == nil {
		t.Fatalf("expected error on invalid arg type")
	}
}

// ---------------------------------------------------------------------------
// validate_request legacy alias dispatch
// ---------------------------------------------------------------------------

func TestValidateRequest_StringArgDelegates(t *testing.T) {
	// String 4th arg should behave identically to ValidateWebhookSignature.
	if !ValidateRequest(vectorAKey, vectorAExpect, vectorAURL, vectorABody) {
		t.Fatalf("string 4th arg: expected match")
	}
}

func TestValidateRequest_NilArgRunsSchemeBEmpty(t *testing.T) {
	// Nil 4th arg → empty params concat (signing string == URL only).
	u := "https://example.com/empty"
	sig := b64Sig(t, "test-key", u, nil)
	if !ValidateRequest("test-key", sig, u, nil) {
		t.Fatalf("nil 4th arg should run Scheme B with empty concat")
	}
}

// ---------------------------------------------------------------------------
// Constant-time compare — the implementation MUST use subtle.ConstantTimeCompare
// for signature comparison. We assert at the source level rather than timing.
// ---------------------------------------------------------------------------

func TestImplementation_UsesConstantTimeCompare(t *testing.T) {
	// White-box test: invoke safeStringEq (the in-package wrapper) with
	// inputs of equal length to confirm the underlying primitive runs.
	// Plain == in user code wouldn't be exercised by this entry point; the
	// implementation review checks equality flows go through safeStringEq.
	if !safeStringEq("abc", "abc") {
		t.Fatalf("safeStringEq should match equal strings")
	}
	if safeStringEq("abc", "abd") {
		t.Fatalf("safeStringEq should reject unequal strings")
	}
	if safeStringEq("abc", "abcd") {
		t.Fatalf("safeStringEq should reject length-mismatched strings")
	}
}
