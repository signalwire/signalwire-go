// Package security — webhook signature validation.
//
// Implements the cross-language SDK contract from porting-sdk/webhooks.md:
//
//   - Scheme A (RELAY/SWML/JSON): hex(HMAC-SHA1(key, url + rawBody))
//   - Scheme B (Compat/cXML form): base64(HMAC-SHA1(key, url + sortedConcatParams))
//     with optional bodySHA256 query-param fallback for JSON-on-compat-surface
//     and URL port normalization (with-port / without-port try).
//
// Public API:
//
//	ValidateWebhookSignature(signingKey, signature, url, rawBody string) bool
//	ValidateRequest(signingKey, signature, url string, paramsOrRawBody any) bool
//
// All comparisons use crypto/subtle.ConstantTimeCompare so the secret never
// leaks through timing side-channels.

package security

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// ErrMissingSigningKey is returned by ValidateWebhookSignatureE /
// ValidateRequestE when an empty signing key is supplied. Per the spec, a
// missing signing key is a programming error rather than a validation
// failure, so the bool-returning entry points panic; the *E variants return
// this sentinel for callers that prefer error returns.
var ErrMissingSigningKey = errors.New("signalwire/security: signing key is required")

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// hexHMACSHA1 returns the lowercase hex digest of HMAC-SHA1(key, message),
// the Scheme-A wire form.
func hexHMACSHA1(key, message string) string {
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// b64HMACSHA1 returns the standard base64 digest of HMAC-SHA1(key, message),
// the Scheme-B wire form.
func b64HMACSHA1(key, message string) string {
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// safeStringEq compares two strings of any length in constant time. Returns
// false (without panicking) for any length mismatch — callers therefore must
// not treat a false here as a leak of the expected length, since the input
// is the attacker-supplied header.
func safeStringEq(a, b string) bool {
	// subtle.ConstantTimeCompare requires equal-length inputs to be useful;
	// when lengths differ the documented behavior is to return 0. Wrap so
	// callers don't need to guard.
	if len(a) != len(b) {
		// Still touch both lengths' bytes to keep timing roughly proportional
		// to the shorter input. A length-mismatch leak is unavoidable here
		// without padding to a fixed envelope; the caller's threat model
		// (header-vs-expected-digest) makes this acceptable.
		_ = subtle.ConstantTimeCompare([]byte(a), []byte(a))
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// sortedConcatParams concatenates form params per Scheme B rules:
//   - sort by key, ASCII ascending;
//   - for repeated keys, emit "key+value" once per occurrence in original
//     order;
//   - return the concatenation (caller prepends URL).
//
// Accepts the canonical map shape, where each key maps to an ordered slice
// of values (matching net/url.Values and the JS reference).
func sortedConcatParams(params map[string][]string) string {
	if len(params) == 0 {
		return ""
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		for _, v := range params[k] {
			b.WriteString(k)
			b.WriteString(v)
		}
	}
	return b.String()
}

// parseFormBody best-effort decodes an x-www-form-urlencoded body.
// Returns nil (signed against url+"") if the body doesn't look like form
// data. Mirrors Python's _parse_form_body.
func parseFormBody(rawBody string) map[string][]string {
	if rawBody == "" {
		return nil
	}
	// url.ParseQuery is lenient; on parse error we fall back to no params.
	parsed, err := url.ParseQuery(rawBody)
	if err != nil || len(parsed) == 0 {
		return nil
	}
	return parsed
}

// candidateURLs returns the URL variants to try for Scheme B port
// normalization, mirroring Python's _candidate_urls:
//
//   - no port + standard scheme  → [input, input-with-:443/:80]
//   - standard explicit port     → [input, input-with-port-stripped]
//   - non-standard explicit port → [input]
//   - missing/unparseable host   → [input]
func candidateURLs(rawURL string) []string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed == nil || parsed.Host == "" {
		return []string{rawURL}
	}

	host := parsed.Hostname()
	port := parsed.Port()
	scheme := strings.ToLower(parsed.Scheme)

	standardPort := ""
	switch scheme {
	case "https":
		standardPort = "443"
	case "http":
		standardPort = "80"
	}

	candidates := []string{rawURL}

	switch {
	case port == "" && standardPort != "":
		// Try with-standard-port variant.
		alt := withHostPort(parsed, host, standardPort)
		if alt != rawURL {
			candidates = append(candidates, alt)
		}
	case port != "" && port == standardPort:
		// Try without-port variant.
		alt := withHostPort(parsed, host, "")
		if alt != rawURL {
			candidates = append(candidates, alt)
		}
	}
	return candidates
}

// withHostPort rebuilds a URL string from a parsed copy with a different
// host:port specifier. Empty port means strip the port. IPv6 hosts are
// re-bracketed automatically by net/url when the Host field uses brackets.
func withHostPort(parsed *url.URL, host, port string) string {
	clone := *parsed
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	if port == "" {
		clone.Host = host
	} else {
		clone.Host = host + ":" + port
	}
	return clone.String()
}

// checkBodySHA256 honors the optional ?bodySHA256=<hex> query param on the
// compat surface: when present, it must equal sha256_hex(rawBody). When
// absent there's no constraint and we return true.
func checkBodySHA256(rawURL, rawBody string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed == nil {
		return true
	}
	expected := parsed.Query().Get("bodySHA256")
	if expected == "" {
		return true
	}
	sum := sha256.Sum256([]byte(rawBody))
	actual := hex.EncodeToString(sum[:])
	return safeStringEq(actual, expected)
}

// ---------------------------------------------------------------------------
// Public API — bool-returning entry points (panic on programmer error)
// ---------------------------------------------------------------------------

// ValidateWebhookSignature validates a SignalWire webhook signature against
// both schemes. Returns true if the signature matches Scheme A (hex JSON) or
// Scheme B (base64 form, with port-normalization variants and optional
// bodySHA256 fallback); false otherwise.
//
// signingKey: customer's Signing Key. Empty string panics — that's a
// programming error, not a validation failure (use ValidateWebhookSignatureE
// for the error-returning variant).
//
// signature: X-SignalWire-Signature (or X-Twilio-Signature) header value.
// Empty returns false without panicking.
//
// url: full URL SignalWire POSTed to (scheme, host, optional port, path,
// query). Must match what the platform saw — see the URL reconstruction
// section of porting-sdk/webhooks.md.
//
// rawBody: raw request body bytes as a UTF-8 string, BEFORE any JSON / form
// parsing.
//
// All comparisons use crypto/subtle.ConstantTimeCompare. The function does
// not log which scheme was tried, what the expected signature was, or any
// other branch information.
func ValidateWebhookSignature(signingKey, signature, url, rawBody string) bool {
	ok, err := ValidateWebhookSignatureE(signingKey, signature, url, rawBody)
	if err != nil {
		panic(err)
	}
	return ok
}

// ValidateRequest is the legacy @signalwire/compatibility-api drop-in entry
// point. The fourth argument is dynamically dispatched:
//
//   - string: delegate to ValidateWebhookSignature (Scheme A then Scheme B
//     with parsed form);
//   - map[string][]string / url.Values / map[string]string: pre-parsed form
//     params, run Scheme B directly with URL port normalization;
//   - nil: pre-parsed empty params, run Scheme B with empty concat string;
//   - anything else: panic with a clear message (programmer error).
func ValidateRequest(signingKey, signature, urlStr string, paramsOrRawBody any) bool {
	ok, err := ValidateRequestE(signingKey, signature, urlStr, paramsOrRawBody)
	if err != nil {
		panic(err)
	}
	return ok
}

// ---------------------------------------------------------------------------
// Public API — error-returning variants (idiomatic Go)
// ---------------------------------------------------------------------------

// ValidateWebhookSignatureE is the error-returning variant of
// ValidateWebhookSignature. Returns ErrMissingSigningKey when signingKey is
// empty; otherwise (matched, nil).
func ValidateWebhookSignatureE(signingKey, signature, urlStr, rawBody string) (bool, error) {
	if signingKey == "" {
		return false, ErrMissingSigningKey
	}
	if signature == "" {
		return false, nil
	}

	// Scheme A — RELAY/SWML/JSON
	expectedA := hexHMACSHA1(signingKey, urlStr+rawBody)
	if safeStringEq(expectedA, signature) {
		return true, nil
	}

	// Scheme B — Compat/cXML
	parsed := parseFormBody(rawBody)

	// Two param-shape attempts: parsed form (when body is form-encoded) and
	// empty params (JSON-on-compat-surface).
	shapes := []map[string][]string{parsed, nil}

	for _, candidate := range candidateURLs(urlStr) {
		for _, shape := range shapes {
			concat := sortedConcatParams(shape)
			expectedB := b64HMACSHA1(signingKey, candidate+concat)
			if !safeStringEq(expectedB, signature) {
				continue
			}
			// HMAC matches. If the URL carries bodySHA256, verify the body
			// hash too. On mismatch keep trying other shapes/candidates.
			if checkBodySHA256(candidate, rawBody) {
				return true, nil
			}
		}
	}
	return false, nil
}

// ValidateRequestE is the error-returning variant of ValidateRequest.
// Returns ErrMissingSigningKey when signingKey is empty, or a typed error
// when paramsOrRawBody is neither a string, nil, nor a recognized map shape.
func ValidateRequestE(signingKey, signature, urlStr string, paramsOrRawBody any) (bool, error) {
	if signingKey == "" {
		return false, ErrMissingSigningKey
	}
	if signature == "" {
		return false, nil
	}

	// String: delegate to combined validator.
	if s, ok := paramsOrRawBody.(string); ok {
		return ValidateWebhookSignatureE(signingKey, signature, urlStr, s)
	}

	// Coerce the parameter to map[string][]string (the canonical shape).
	var params map[string][]string
	switch v := paramsOrRawBody.(type) {
	case nil:
		params = nil
	case map[string][]string:
		params = v
	case url.Values:
		params = v
	case map[string]string:
		// Promote scalar map to the canonical shape — every value becomes a
		// single-element slice.
		params = make(map[string][]string, len(v))
		for k, val := range v {
			params[k] = []string{val}
		}
	case map[string]any:
		// Best-effort: accept stringly-coercible values; reject otherwise so
		// silent "%!s(...)" stringification doesn't sneak into the signing
		// string.
		params = make(map[string][]string, len(v))
		for k, raw := range v {
			coerced, err := coerceFormValue(raw)
			if err != nil {
				return false, fmt.Errorf("signalwire/security: param %q: %w", k, err)
			}
			params[k] = coerced
		}
	default:
		return false, fmt.Errorf(
			"signalwire/security: paramsOrRawBody must be a string, map[string][]string, "+
				"url.Values, map[string]string, map[string]any, or nil; got %T", paramsOrRawBody,
		)
	}

	concat := sortedConcatParams(params)
	for _, candidate := range candidateURLs(urlStr) {
		expectedB := b64HMACSHA1(signingKey, candidate+concat)
		if safeStringEq(expectedB, signature) {
			// No raw body to check bodySHA256 against here — skip that check.
			return true, nil
		}
	}
	return false, nil
}

// coerceFormValue stringifies a single param value, accepting scalars and
// []string / []any slices. Used by ValidateRequestE when the caller hands us
// a map[string]any (e.g. unmarshaled JSON).
func coerceFormValue(v any) ([]string, error) {
	switch x := v.(type) {
	case string:
		return []string{x}, nil
	case []string:
		out := make([]string, len(x))
		copy(out, x)
		return out, nil
	case []any:
		out := make([]string, 0, len(x))
		for _, raw := range x {
			s, err := stringifyScalar(raw)
			if err != nil {
				return nil, err
			}
			out = append(out, s)
		}
		return out, nil
	default:
		s, err := stringifyScalar(v)
		if err != nil {
			return nil, err
		}
		return []string{s}, nil
	}
}

// stringifyScalar coerces a JSON scalar to its on-the-wire form. Booleans,
// numbers and strings are accepted; nil maps to "". Composite types are
// rejected because they'd silently drift from the JS reference's behavior.
func stringifyScalar(v any) (string, error) {
	switch x := v.(type) {
	case nil:
		return "", nil
	case string:
		return x, nil
	case bool:
		if x {
			return "true", nil
		}
		return "false", nil
	case int:
		return fmt.Sprintf("%d", x), nil
	case int64:
		return fmt.Sprintf("%d", x), nil
	case float64:
		// Format without trailing zeros where possible. JSON numbers in Go
		// land as float64; mimic %v but force integer form for whole nums.
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x)), nil
		}
		return fmt.Sprintf("%g", x), nil
	default:
		return "", fmt.Errorf("unsupported scalar type %T", v)
	}
}
