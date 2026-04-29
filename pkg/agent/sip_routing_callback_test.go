// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Tests covering AgentBase.GetFullURL forwarding and
// AgentBase.RegisterSipRoutingCallback (Python web_mixin
// register_routing_callback semantics: string return → HTTP 307 redirect).

func TestGetFullURLForwardsToSwmlService(t *testing.T) {
	a := NewAgentBase(WithName("t"), WithRoute("/svc"), WithHost("example.com"), WithPort(8080))

	got := a.GetFullURL(false)
	want := a.Service.GetFullURL(false)
	if got != want {
		t.Errorf("GetFullURL(false) = %q, want %q (matching swml.Service.GetFullURL)", got, want)
	}

	gotAuth := a.GetFullURL(true)
	wantAuth := a.Service.GetFullURL(true)
	if gotAuth != wantAuth {
		t.Errorf("GetFullURL(true) = %q, want %q", gotAuth, wantAuth)
	}
}

func TestSipRoutingCallbackEmitsRedirectOnString(t *testing.T) {
	a := NewAgentBase(WithName("t"), WithRoute("/svc"), WithBasicAuth("u", "p"))

	const target = "https://elsewhere.example/handoff"
	a.RegisterSipRoutingCallback(func(r *http.Request, body map[string]any) string {
		return target
	}, "/sip")

	req := httptest.NewRequest(http.MethodPost, "/sip", strings.NewReader(`{"call":{"from":"+15555550000"}}`))
	req.SetBasicAuth("u", "p")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	a.AsRouter().ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d, want %d (307); body=%s", rec.Code, http.StatusTemporaryRedirect, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != target {
		t.Errorf("Location header = %q, want %q", loc, target)
	}
}

func TestSipRoutingCallbackFallsThroughOnEmptyString(t *testing.T) {
	a := NewAgentBase(WithName("t"), WithRoute("/svc"), WithBasicAuth("u", "p"))

	called := false
	a.RegisterSipRoutingCallback(func(r *http.Request, body map[string]any) string {
		called = true
		return "" // fall through to normal SWML pipeline
	}, "/sip")

	req := httptest.NewRequest(http.MethodPost, "/sip", strings.NewReader(`{"call":{"from":"+15555550000"}}`))
	req.SetBasicAuth("u", "p")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	a.AsRouter().ServeHTTP(rec, req)

	if !called {
		t.Fatal("callback was not invoked")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (fall-through to SWML); body=%s", rec.Code, rec.Body.String())
	}
	// Empty-string return must not surface as a redirect.
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Errorf("Location header = %q, want empty (no redirect on empty return)", loc)
	}
	// Body should be a JSON SWML document, not empty.
	var doc map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("response body is not JSON: %v\nbody=%s", err, rec.Body.String())
	}
	if _, ok := doc["sections"]; !ok {
		t.Errorf("response body lacks 'sections' key (not a SWML doc); got keys=%v", keys(doc))
	}
}

func TestSipRoutingCallbackPathNormalization(t *testing.T) {
	cases := []struct {
		name       string
		registered string
		want       string
	}{
		{"trailing-slash-stripped", "/sip/", "/sip"},
		{"leading-slash-added", "sip", "/sip"},
		{"empty-defaults-to-sip", "", "/sip"},
		{"already-canonical", "/sip", "/sip"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := NewAgentBase(WithName("t"), WithRoute("/svc"))
			a.RegisterSipRoutingCallback(func(r *http.Request, body map[string]any) string {
				return ""
			}, tc.registered)

			paths := a.sipRoutingCallbackPaths()
			if len(paths) != 1 || paths[0] != tc.want {
				t.Errorf("registered %q → paths=%v, want [%q]", tc.registered, paths, tc.want)
			}
		})
	}
}

func TestSipRoutingCallbackIgnoresGetRequests(t *testing.T) {
	a := NewAgentBase(WithName("t"), WithRoute("/svc"), WithBasicAuth("u", "p"))

	called := false
	a.RegisterSipRoutingCallback(func(r *http.Request, body map[string]any) string {
		called = true
		return "https://elsewhere.example/handoff"
	}, "/sip")

	// GET should not invoke the SIP callback (Python web_mixin.py:624 only
	// runs the callback when request.method == "POST" and body is set).
	req := httptest.NewRequest(http.MethodGet, "/sip", nil)
	req.SetBasicAuth("u", "p")

	rec := httptest.NewRecorder()
	a.AsRouter().ServeHTTP(rec, req)

	if called {
		t.Error("SIP callback was invoked on GET; should only fire on POST")
	}
	if rec.Code == http.StatusTemporaryRedirect {
		t.Error("GET produced a 307 redirect; should fall through to normal handling")
	}
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
