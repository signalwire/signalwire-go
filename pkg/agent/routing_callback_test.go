// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package agent

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/signalwire/signalwire-go/pkg/swml"
)

// Tests for the routing-callback dispatch. Mirrors Python
// web_mixin._handle_request (line 620-635): a request matching a registered
// callback path is dispatched through the callback as callback_fn(body, headers),
// and a non-nil route string triggers an HTTP 307 redirect.

func strptr(s string) *string { return &s }

// dispatchRedirectAt sends a POST to the agent's HTTP router at path and reports
// the response status code and Location header (for a 307 redirect).
func dispatchRedirectAt(t *testing.T, a *AgentBase, path string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"k":"v"}`))
	req.SetBasicAuth("u", "p")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	a.AsRouter().ServeHTTP(rec, req)
	return rec.Code, rec.Header().Get("Location")
}

func TestRoutingCallbackRedirectsAtExactPath(t *testing.T) {
	a := NewAgentBase(WithName("t"), WithRoute("/svc"), WithBasicAuth("u", "p"))
	a.RegisterRoutingCallback(func(body map[string]any, headers map[string]any) *string {
		return strptr("/redirected")
	}, "/agents")

	code, loc := dispatchRedirectAt(t, a, "/agents")
	if code != http.StatusTemporaryRedirect {
		t.Fatalf("routing callback should 307-redirect at the registered path, got %d", code)
	}
	if loc != "/redirected" {
		t.Fatalf("want Location=/redirected, got %q", loc)
	}
}

func TestRoutingCallbackRedirectsAtTrailingSlashVariant(t *testing.T) {
	a := NewAgentBase(WithName("t"), WithRoute("/svc"), WithBasicAuth("u", "p"))
	a.RegisterRoutingCallback(func(body map[string]any, headers map[string]any) *string {
		return strptr("/redirected")
	}, "/agents")

	// Python registers both "/agents" and "/agents/" — Go mirrors that.
	code, loc := dispatchRedirectAt(t, a, "/agents/")
	if code != http.StatusTemporaryRedirect || loc != "/redirected" {
		t.Fatalf("routing callback should also redirect at the trailing-slash variant, got %d %q", code, loc)
	}
}

func TestRoutingCallbackNotFiredForUnmatchedPath(t *testing.T) {
	a := NewAgentBase(WithName("t"), WithRoute("/svc"), WithBasicAuth("u", "p"))
	a.RegisterRoutingCallback(func(body map[string]any, headers map[string]any) *string {
		return strptr("/redirected")
	}, "/agents")

	// A request to the main SWML route must render the default doc, not redirect.
	code, _ := dispatchRedirectAt(t, a, "/svc")
	if code == http.StatusTemporaryRedirect {
		t.Fatal("routing callback must not fire for requests at the main SWML route")
	}
}

func TestRoutingCallbackNilFallsBackToDefault(t *testing.T) {
	// When the callback returns nil, no redirect happens and the agent renders
	// its default document — matches Python's "no route returned" path.
	a := NewAgentBase(WithName("t"), WithRoute("/svc"), WithBasicAuth("u", "p"))
	a.SetPromptText("hi")
	called := false
	a.RegisterRoutingCallback(func(body map[string]any, headers map[string]any) *string {
		called = true
		return nil
	}, "/agents")

	code, _ := dispatchRedirectAt(t, a, "/agents")
	if code == http.StatusTemporaryRedirect {
		t.Fatal("nil callback result should not produce a redirect")
	}
	if !called {
		t.Fatal("callback should have been invoked even when it returns nil")
	}
}

// Guard that the swml.Service exposes the list of registered callback paths —
// needed by the agent mux to know which HTTP handlers to register.
func TestRoutingCallbackPathsExposedFromSwmlService(t *testing.T) {
	svc := swml.NewService(swml.WithName("paths"))
	svc.RegisterRoutingCallback("/b", func(body map[string]any, headers map[string]any) *string { return nil })
	svc.RegisterRoutingCallback("/a", func(body map[string]any, headers map[string]any) *string { return nil })

	got := svc.RoutingCallbackPaths()
	if len(got) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(got), got)
	}
	if got[0] != "/a" || got[1] != "/b" {
		t.Fatalf("expected sorted paths [/a /b], got %v", got)
	}
}

// TestHandleRequestPrimitiveDispatch exercises the framework-free HandleRequest
// core: auth failure, routing redirect, and default document rendering.
func TestHandleRequestPrimitiveDispatch(t *testing.T) {
	a := NewAgentBase(WithName("t"), WithRoute("/svc"), WithBasicAuth("u", "p"))
	a.SetPromptText("hi")
	a.RegisterRoutingCallback(func(body map[string]any, headers map[string]any) *string {
		if body["go"] == true {
			return strptr("/elsewhere")
		}
		return nil
	}, "/agents")

	authHdr := map[string]string{"Authorization": "Basic dTpw"} // base64("u:p")

	// Missing auth → 401.
	if status, _, _ := a.HandleRequest("POST", "/svc", nil, nil); status != 401 {
		t.Fatalf("missing auth: want 401, got %d", status)
	}

	// Routing redirect → 307 + Location.
	status, hdrs, _ := a.HandleRequest("POST", "/agents", authHdr, map[string]any{"go": true})
	if status != 307 || hdrs["Location"] != "/elsewhere" {
		t.Fatalf("routing redirect: want 307 /elsewhere, got %d %q", status, hdrs["Location"])
	}

	// Normal request → 200 document.
	if status, _, _ := a.HandleRequest("POST", "/svc", authHdr, map[string]any{}); status != 200 {
		t.Fatalf("normal request: want 200, got %d", status)
	}
}
