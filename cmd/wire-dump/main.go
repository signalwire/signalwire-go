// Command wire-dump is the Go port's WIRE-CRYPTO dump program for the cross-port
// wire differ (porting-sdk/scripts/diff_port_wire.py).
//
// It runs the shared wire_crypto corpus against the Go SDK's native security
// package (SessionManager tokens, webhook-signature validation, redact/filter
// helpers) and prints ONE JSON object mapping
//
//	case-id -> observable-artifact
//
// to stdout. The differ runs this program, canonicalizes both sides, and
// byte-compares each entry against the python oracle. Only stdout carries JSON;
// nothing else is printed there.
//
// The corpus sentinels (__ORACLE_FORMAT_TOKEN__, __TAMPERED_TOKEN__,
// __ORACLE_SIG__) are materialized here from the fixed per-case SECRET exactly
// as the oracle materializes them, so the interop/tamper cases are reproducible.
//
// Run from the signalwire-go repo root:
//
//	go run ./cmd/wire-dump
package main

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // webhook Scheme A is defined as HMAC-SHA1; matches the wire spec.
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/v3/pkg/security"
)

// SECRET mirrors wire_crypto_corpus.SECRET ("a" * 64).
const secret = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

const (
	oracleExpiry = int64(9999999999)  // fixed far-future expiry (deterministic)
	oracleNonce  = "0123456789abcdef" // fixed 16-hex nonce (deterministic)
)

// oracleToken builds a token in the SDK wire format (call_id.fn.expiry.nonce.sig,
// base64url) from the fixed SECRET — the Go mirror of diff_port_wire._oracle_token.
func oracleToken(callID, fn string) string {
	msg := fmt.Sprintf("%s:%s:%d:%s", callID, fn, oracleExpiry, oracleNonce)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	sig := hex.EncodeToString(mac.Sum(nil))
	raw := fmt.Sprintf("%s.%s.%d.%s.%s", callID, fn, oracleExpiry, oracleNonce, sig)
	return base64.URLEncoding.EncodeToString([]byte(raw))
}

// tamperedToken flips one signature character — the Go mirror of _tampered_token.
func tamperedToken() string {
	tok := oracleToken("c", "f")
	raw, _ := base64.URLEncoding.DecodeString(tok)
	s := raw
	// flip the last char (first char of the sig, which is the tail after the last '.')
	// mirror python: parts[-1][0] flip, but flipping the very last byte is equivalent
	// for a "signature no longer matches" guarantee. Match python exactly instead:
	// find last '.' and flip the byte right after it.
	last := -1
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			last = i
			break
		}
	}
	idx := last + 1
	if s[idx] == 'f' {
		s[idx] = 'e'
	} else {
		s[idx] = 'f'
	}
	return base64.URLEncoding.EncodeToString(s)
}

// oracleSig computes the correct webhook signature: hex(HMAC-SHA1(key, url+body)).
func oracleSig(url, body, key string) string {
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(url + body))
	return hex.EncodeToString(mac.Sum(nil))
}

// observeTokenFields decodes a token and returns its wire-format shape.
func observeTokenFields(token string) map[string]any {
	raw, _ := base64.URLEncoding.DecodeString(token)
	parts := splitDots(string(raw))
	nonce := ""
	if len(parts) > 3 {
		nonce = parts[3]
	}
	isHex := len(parts) > 3
	for _, c := range nonce {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			isHex = false
			break
		}
	}
	var callID, fn any
	if len(parts) > 0 {
		callID = parts[0]
	}
	if len(parts) > 1 {
		fn = parts[1]
	}
	return map[string]any{
		"n_fields":      len(parts),
		"call_id":       callID,
		"function_name": fn,
		"nonce_len":     len(nonce),
		"nonce_is_hex":  isHex,
	}
}

func splitDots(s string) []string {
	var out []string
	start := 0
	for i := range len(s) {
		if s[i] == '.' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func main() {
	out := map[string]any{}

	// token_format: generate a token via the SDK, decode its fields.
	sm := security.NewSessionManager(9999999999, security.WithSecret([]byte(secret)))
	out["token_format"] = observeTokenFields(sm.GenerateToken("my_func", "call_1"))

	// token_nonce_distinct: two generations must differ (random nonce).
	n1 := sm.GenerateToken("f", "c")
	n2 := sm.GenerateToken("f", "c")
	out["token_nonce_distinct"] = map[string]any{"distinct": n1 != n2}

	// token_interop: validate an oracle-format token built from SECRET.
	out["token_interop"] = map[string]any{
		"valid": sm.ValidateToken("oracle_fn", oracleToken("oracle_call", "oracle_fn"), "oracle_call"),
	}

	// token_tamper_rejected: a one-byte-flipped signature must fail.
	out["token_tamper_rejected"] = map[string]any{
		"valid": sm.ValidateToken("f", tamperedToken(), "c"),
	}

	// wire_validate_webhook_signature: correct HMAC-SHA1 -> valid.
	whURL := "https://example.com/hook"
	whBody := `{"event":"call.created"}`
	out["wire_validate_webhook_signature"] = map[string]any{
		"valid": security.ValidateWebhookSignature(secret, oracleSig(whURL, whBody, secret), whURL, whBody),
	}
	// wire_validate_webhook_signature_bad: wrong sig -> invalid.
	badSig := ""
	for range 8 {
		badSig += "deadbeef"
	}
	out["wire_validate_webhook_signature_bad"] = map[string]any{
		"valid": security.ValidateWebhookSignature(secret, badSig, whURL, whBody),
	}

	// redact_url: credentials + token redacted, structure preserved.
	out["wire_redact_url"] = map[string]any{
		"redacted": security.RedactURL("https://user:s3cr3t@api.signalwire.com/path?token=abc"),
	}

	// filter_sensitive_headers: authorization + x-api-key dropped, content-type kept.
	filtered := security.FilterSensitiveHeaders(map[string]string{
		"Authorization": "Bearer x",
		"X-Api-Key":     "y",
		"Content-Type":  "application/json",
	})
	out["wire_filter_sensitive_headers"] = map[string]any{"filtered": filtered}

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "wire-dump: encode failed: %v\n", err)
		os.Exit(1)
	}
}
