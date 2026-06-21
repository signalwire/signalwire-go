// Package security — standalone security hygiene utilities.
//
// These mirror the TypeScript SDK's SecurityUtils (filterSensitiveHeaders,
// redactUrl, isValidHostname) and the Python reference's
// signalwire.core.security.security_utils free functions, so the same
// protections — keeping credentials out of user callbacks and logs, reusable
// hostname validation — are available in the Go port.
//
// They are package-level free functions (the idiomatic Go shape for stateless
// helpers, matching the Python reference's module-level functions) and project
// onto the Python canonical names via internal/surface/tables.go:
//
//	FilterSensitiveHeaders -> filter_sensitive_headers
//	RedactURL              -> redact_url
//	IsValidHostname        -> is_valid_hostname
//
// PascalCase + the all-caps URL initialism follow the Go naming idiom
// (Effective Go; staticcheck ST1003); the adapter maps them to the snake_case
// canonical names the DRIFT gate compares against.
package security

import (
	"regexp"
	"strings"
)

// sensitiveHeaders is the set of header names whose values are
// credentials/secrets and must never be handed to user callbacks or written to
// logs. Membership is tested case-insensitively (keys here are lowercase).
//
// Kept unexported — it is an internal implementation detail, not part of the
// public surface (matching the Python reference, where SENSITIVE_HEADERS is a
// module constant excluded from the public surface).
var sensitiveHeaders = map[string]struct{}{
	"authorization":       {},
	"cookie":              {},
	"x-api-key":           {},
	"proxy-authorization": {},
	"set-cookie":          {},
}

// urlCredentialsRE matches userinfo credentials in a URL: ://user:secret@host
// -> ://user:****@host. Mirrors the Python reference regex
// `://([^:@/]+):([^@/]+)@`.
var urlCredentialsRE = regexp.MustCompile(`://([^:@/]+):([^@/]+)@`)

// hostnameRejectRE matches any character a valid hostname must not contain:
// whitespace, slashes, backslashes, or control characters. Mirrors the Python
// reference regex `[\s/\\\x00-\x1f\x7f]`.
var hostnameRejectRE = regexp.MustCompile(`[\s/\\\x00-\x1f\x7f]`)

// FilterSensitiveHeaders returns a copy of headers with sensitive
// (credential-bearing) headers removed, so request headers can be safely passed
// to user callbacks or written to logs.
//
// The sensitivity check is case-insensitive; non-sensitive keys are preserved
// with their original casing. A nil or empty input yields a non-nil empty map.
func FilterSensitiveHeaders(headers map[string]string) map[string]string {
	out := make(map[string]string, len(headers))
	for k, v := range headers {
		if _, sensitive := sensitiveHeaders[strings.ToLower(k)]; sensitive {
			continue
		}
		out[k] = v
	}
	return out
}

// RedactURL masks the password in a URL's userinfo before logging:
//
//	https://user:secret@host/path -> https://user:****@host/path
//
// A URL with no embedded credentials is returned unchanged.
func RedactURL(url string) string {
	return urlCredentialsRE.ReplaceAllString(url, "://$1:****@")
}

// IsValidHostname is a standalone hostname sanity check: it rejects empty hosts
// and any host containing whitespace, slashes, backslashes, or control
// characters.
//
// This is the reusable character-level check, independent of the fuller
// util.ValidateURL (which also does scheme checks, DNS resolution, and
// private-IP blocking). Callers that only need to validate a hostname string
// use this.
func IsValidHostname(host string) bool {
	if host == "" {
		return false
	}
	return !hostnameRejectRE.MatchString(host)
}
