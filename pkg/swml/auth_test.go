// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package swml

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Tests for the bearer / api-key / WithSecurityConfig / GetAuthInfo / TLS
// surface added in this PR. Equivalent Python coverage lives in
// tests/unit/core/test_auth_handler.py.

// --- VerifyBearerToken ---

func TestVerifyBearerTokenAcceptsMatch(t *testing.T) {
	svc := NewService(WithBearerToken("secret123"))
	if !svc.VerifyBearerToken("secret123") {
		t.Fatalf("VerifyBearerToken should accept the configured token")
	}
}

func TestVerifyBearerTokenRejectsMismatch(t *testing.T) {
	svc := NewService(WithBearerToken("secret123"))
	if svc.VerifyBearerToken("wrong") {
		t.Fatalf("VerifyBearerToken should reject a wrong token")
	}
}

func TestVerifyBearerTokenFalseWhenNoneConfigured(t *testing.T) {
	svc := NewService(WithName("no-bearer"))
	if svc.VerifyBearerToken("anything") {
		t.Fatalf("VerifyBearerToken should return false when no token is configured")
	}
}

// --- VerifyAPIKey ---

func TestVerifyAPIKeyAcceptsMatch(t *testing.T) {
	svc := NewService(WithAPIKey("k123", ""))
	if !svc.VerifyAPIKey("k123") {
		t.Fatalf("VerifyAPIKey should accept the configured key")
	}
}

func TestVerifyAPIKeyRejectsMismatch(t *testing.T) {
	svc := NewService(WithAPIKey("k123", ""))
	if svc.VerifyAPIKey("wrong") {
		t.Fatalf("VerifyAPIKey should reject a wrong key")
	}
}

func TestVerifyAPIKeyFalseWhenNoneConfigured(t *testing.T) {
	svc := NewService(WithName("no-api-key"))
	if svc.VerifyAPIKey("anything") {
		t.Fatalf("VerifyAPIKey should return false when no key is configured")
	}
}

// --- GetAuthInfo ---

func TestGetAuthInfoAlwaysIncludesBasic(t *testing.T) {
	svc := NewService(WithBasicAuth("alice", "pw"))
	info := svc.GetAuthInfo()
	basic, ok := info["basic"].(map[string]any)
	if !ok {
		t.Fatalf("expected basic entry, got %#v", info)
	}
	if basic["enabled"] != true {
		t.Fatalf(`basic["enabled"] = %v, want true`, basic["enabled"])
	}
	if basic["username"] != "alice" {
		t.Fatalf(`basic["username"] = %v, want "alice"`, basic["username"])
	}
}

func TestGetAuthInfoOmitsUnconfiguredMethods(t *testing.T) {
	svc := NewService(WithBasicAuth("u", "p"))
	info := svc.GetAuthInfo()
	if _, present := info["bearer"]; present {
		t.Errorf("bearer should be omitted when not configured")
	}
	if _, present := info["api_key"]; present {
		t.Errorf("api_key should be omitted when not configured")
	}
}

func TestGetAuthInfoIncludesBearerWhenConfigured(t *testing.T) {
	svc := NewService(WithBearerToken("tok"))
	info := svc.GetAuthInfo()
	bearer, ok := info["bearer"].(map[string]any)
	if !ok {
		t.Fatalf("expected bearer entry, got %#v", info)
	}
	if bearer["enabled"] != true {
		t.Errorf(`bearer["enabled"] = %v, want true`, bearer["enabled"])
	}
	if _, ok := bearer["hint"].(string); !ok {
		t.Errorf(`bearer["hint"] should be a string`)
	}
}

func TestGetAuthInfoIncludesAPIKeyWithCustomHeader(t *testing.T) {
	svc := NewService(WithAPIKey("k123", "X-Company-Key"))
	info := svc.GetAuthInfo()
	apiKey, ok := info["api_key"].(map[string]any)
	if !ok {
		t.Fatalf("expected api_key entry, got %#v", info)
	}
	if apiKey["header"] != "X-Company-Key" {
		t.Errorf(`api_key["header"] = %v, want "X-Company-Key"`, apiKey["header"])
	}
}

func TestGetAuthInfoAPIKeyDefaultsHeader(t *testing.T) {
	svc := NewService(WithAPIKey("k123", ""))
	info := svc.GetAuthInfo()
	apiKey, _ := info["api_key"].(map[string]any)
	if apiKey["header"] != "X-API-Key" {
		t.Errorf(`default api_key header = %v, want "X-API-Key"`, apiKey["header"])
	}
}

// --- WithSecurityConfig bundle ---

func TestWithSecurityConfigAppliesAllMethods(t *testing.T) {
	svc := NewService(WithSecurityConfig(SecurityConfig{
		BasicAuthUser:     "u",
		BasicAuthPassword: "p",
		BearerToken:       "tok",
		APIKey:            "k",
		APIKeyHeader:      "X-Key",
	}))
	if !svc.VerifyBasicAuth("u", "p") {
		t.Errorf("basic auth should be configured")
	}
	if !svc.VerifyBearerToken("tok") {
		t.Errorf("bearer token should be configured")
	}
	if !svc.VerifyAPIKey("k") {
		t.Errorf("api key should be configured")
	}
}

// --- TLSEnabled ---

func TestTLSEnabledDefaultFalse(t *testing.T) {
	svc := NewService(WithName("no-tls"))
	if svc.TLSEnabled() {
		t.Fatalf("TLSEnabled should be false by default")
	}
}

func TestTLSEnabledTrueAfterWithTLS(t *testing.T) {
	svc := NewService(WithTLS("cert.pem", "key.pem"))
	if !svc.TLSEnabled() {
		t.Fatalf("TLSEnabled should be true after WithTLS")
	}
}

func TestTLSEnabledFalseIfOnlyOnePathSet(t *testing.T) {
	// Both paths are required for TLS to be considered enabled.
	svc := NewService(WithTLS("cert.pem", ""))
	if svc.TLSEnabled() {
		t.Fatalf("TLSEnabled should require both cert and key paths")
	}
}

// --- Middleware end-to-end: withSecurity priority + 401 shape ---

func TestWithSecurityBearerTokenSucceeds(t *testing.T) {
	svc := NewService(WithBasicAuth("u", "p"), WithBearerToken("tok"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer tok")

	called := false
	handler := svc.withSecurity(func(http.ResponseWriter, *http.Request) { called = true })
	handler(rec, req)
	if !called {
		t.Fatalf("next handler should have been invoked on valid bearer token, got status %d", rec.Code)
	}
}

func TestWithSecurityBearerTokenWrongValueRejected(t *testing.T) {
	svc := NewService(WithBasicAuth("u", "p"), WithBearerToken("tok"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer WRONG")

	called := false
	handler := svc.withSecurity(func(http.ResponseWriter, *http.Request) { called = true })
	handler(rec, req)
	if called {
		t.Fatalf("next handler should NOT have been invoked on wrong bearer token")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if got := rec.Header().Get("WWW-Authenticate"); got == "" {
		t.Fatalf("expected WWW-Authenticate header on 401, got empty")
	}
}

func TestWithSecurityAPIKeyDefaultHeaderSucceeds(t *testing.T) {
	svc := NewService(WithBasicAuth("u", "p"), WithAPIKey("k", ""))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "k")

	called := false
	handler := svc.withSecurity(func(http.ResponseWriter, *http.Request) { called = true })
	handler(rec, req)
	if !called {
		t.Fatalf("next handler should have been invoked on valid api key, got status %d", rec.Code)
	}
}

func TestWithSecurityAPIKeyCustomHeaderSucceeds(t *testing.T) {
	svc := NewService(WithBasicAuth("u", "p"), WithAPIKey("k", "X-Custom"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Custom", "k")

	called := false
	handler := svc.withSecurity(func(http.ResponseWriter, *http.Request) { called = true })
	handler(rec, req)
	if !called {
		t.Fatalf("next handler should have been invoked on valid api key (custom header), got status %d", rec.Code)
	}
}

func TestWithSecurityFallsBackToBasicAuth(t *testing.T) {
	// No bearer/api-key configured — only basic.
	svc := NewService(WithBasicAuth("u", "p"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("u", "p")

	called := false
	handler := svc.withSecurity(func(http.ResponseWriter, *http.Request) { called = true })
	handler(rec, req)
	if !called {
		t.Fatalf("next handler should have been invoked on basic auth, got status %d", rec.Code)
	}
}

func TestWithSecurityRejectsUnauthenticated(t *testing.T) {
	svc := NewService(WithBasicAuth("u", "p"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler := svc.withSecurity(func(http.ResponseWriter, *http.Request) {
		t.Fatalf("handler should not be invoked without credentials")
	})
	handler(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestWithSecurityAnyMethodSufficesWhenMultipleConfigured(t *testing.T) {
	// All three methods configured; bearer is first in priority but a
	// request using only the API key should still succeed.
	svc := NewService(
		WithBasicAuth("u", "p"),
		WithBearerToken("tok"),
		WithAPIKey("k", ""),
	)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "k")

	called := false
	handler := svc.withSecurity(func(http.ResponseWriter, *http.Request) { called = true })
	handler(rec, req)
	if !called {
		t.Fatalf("any configured auth method should suffice; got status %d", rec.Code)
	}
}
