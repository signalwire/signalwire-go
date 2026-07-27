package security

import (
	"crypto/subtle"
	"net/http"
)

// AuthHandler verifies inbound HTTP credentials against a SecurityConfig.
//
// It is the Go form of the reference's signalwire.core.auth_handler.AuthHandler
// (auth_handler.py:47). The reference bundles three things into that class:
//
//  1. the SecurityConfig it verifies against, plus constant-time verification of
//     the credentials in it — plain, framework-free capability, which is what this
//     type provides;
//  2. Basic/Bearer/API-key credential OBJECTS from FastAPI's security helpers;
//  3. framework adapters (`get_fastapi_dependency`, `flask_decorator`).
//
// (2) and (3) have no Go analogue — Go composes `net/http` middleware instead of
// registering framework dependencies — and remain recorded as omissions. Go's own
// request path uses `AgentBase.withAuth`, which performs the same constant-time
// comparison inline; this type makes the verification reusable and lets a caller
// read back the configuration it is verifying against, exactly as the reference's
// `AuthHandler.security_config` does.
type AuthHandler struct {
	securityConfig *SecurityConfig
}

// NewAuthHandler returns an AuthHandler that verifies credentials against cfg.
// A nil cfg yields a handler that rejects every credential.
func NewAuthHandler(cfg *SecurityConfig) *AuthHandler {
	return &AuthHandler{securityConfig: cfg}
}

// SecurityConfig returns the configuration this handler verifies against
// (Python: AuthHandler.security_config, auth_handler.py:62).
func (h *AuthHandler) SecurityConfig() *SecurityConfig { return h.securityConfig }

// VerifyBasicAuth reports whether the request carries basic credentials matching
// the configured ones (Python: verify_basic_auth).
//
// The reference takes a single `credentials: HTTPBasicCredentials` — FastAPI's
// parsed Authorization header. Go's carrier for the same thing is the
// *http.Request the middleware already holds, so this takes the request and reads
// the header off it via r.BasicAuth(). Same one-value shape, no invented
// credentials struct; use VerifyBasicAuthPair when the two strings are already in
// hand.
func (h *AuthHandler) VerifyBasicAuth(r *http.Request) bool {
	if r == nil {
		return false
	}
	username, password, ok := r.BasicAuth()
	if !ok {
		return false
	}
	return h.VerifyBasicAuthPair(username, password)
}

// VerifyBasicAuthPair reports whether username/password match the configured
// basic credentials. Both comparisons run in constant time and BOTH always run,
// so the result does not leak which half was wrong — the same property
// `secrets.compare_digest` gives the reference.
func (h *AuthHandler) VerifyBasicAuthPair(username, password string) bool {
	if h.securityConfig == nil {
		return false
	}
	wantUser, wantPass := h.securityConfig.GetBasicAuth()
	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(wantUser)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(password), []byte(wantPass)) == 1
	return userOK && passOK
}
