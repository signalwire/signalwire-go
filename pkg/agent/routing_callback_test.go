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

	"github.com/signalwire/signalwire-go/pkg/swml"
)

// Tests for the routing-callback dispatch added in this PR.
// Mirrors Python web_mixin._handle_request (line 620) which dispatches
// requests matching a registered callback path through the callback.

const routingMarker = "__routing_cb_fired__"

func swmlResponseWithMarker() map[string]any {
	return map[string]any{
		"version":  routingMarker,
		"sections": map[string]any{"main": []any{}},
	}
}

// dispatchMarkerAt sends a POST to the agent's HTTP router at path and
// reports whether the response body's "version" field is the routing
// callback marker (i.e., the callback fired).
func dispatchMarkerAt(t *testing.T, a *AgentBase, path string) bool {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{}"))
	req.SetBasicAuth("u", "p")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	a.AsRouter().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Logf("dispatch at %q returned status %d, body=%s", path, rec.Code, rec.Body.String())
		return false
	}
	var doc map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		return false
	}
	return doc["version"] == routingMarker
}

func TestRoutingCallbackDispatchedAtExactPath(t *testing.T) {
	a := NewAgentBase(WithName("t"), WithRoute("/svc"), WithBasicAuth("u", "p"))
	a.RegisterRoutingCallback(func(r *http.Request, body map[string]any) map[string]any {
		return swmlResponseWithMarker()
	}, "/agents")

	if !dispatchMarkerAt(t, a, "/agents") {
		t.Fatal("routing callback should fire when request matches the registered path")
	}
}

func TestRoutingCallbackDispatchedAtTrailingSlashVariant(t *testing.T) {
	a := NewAgentBase(WithName("t"), WithRoute("/svc"), WithBasicAuth("u", "p"))
	a.RegisterRoutingCallback(func(r *http.Request, body map[string]any) map[string]any {
		return swmlResponseWithMarker()
	}, "/agents")

	// Python registers both "/agents" and "/agents/" — Go mirrors that.
	if !dispatchMarkerAt(t, a, "/agents/") {
		t.Fatal("routing callback should also fire at the trailing-slash path variant")
	}
}

func TestRoutingCallbackNotFiredForUnmatchedPath(t *testing.T) {
	a := NewAgentBase(WithName("t"), WithRoute("/svc"), WithBasicAuth("u", "p"))
	a.RegisterRoutingCallback(func(r *http.Request, body map[string]any) map[string]any {
		return swmlResponseWithMarker()
	}, "/agents")

	// A request to the main SWML route must use the default doc, not the
	// routing callback.
	if dispatchMarkerAt(t, a, "/svc") {
		t.Fatal("routing callback must not fire for requests at the main SWML route")
	}
}

func TestRoutingCallbackNilFallsBackToDefault(t *testing.T) {
	// When the callback returns nil, swml.Service.OnRequest falls back to
	// the service's default document — matches Python's
	// "no modifications returned" path.
	a := NewAgentBase(WithName("t"), WithRoute("/svc"), WithBasicAuth("u", "p"))
	a.SetPromptText("hi")
	called := false
	a.RegisterRoutingCallback(func(r *http.Request, body map[string]any) map[string]any {
		called = true
		return nil
	}, "/agents")

	if dispatchMarkerAt(t, a, "/agents") {
		t.Fatal("nil callback result should not produce the marker")
	}
	if !called {
		t.Fatal("callback should have been invoked even when it returns nil")
	}
}

// Guard that the swml.Service exposes the list of registered callback
// paths — needed by the agent mux to know which HTTP handlers to register.
func TestRoutingCallbackPathsExposedFromSwmlService(t *testing.T) {
	svc := swml.NewService(swml.WithName("paths"))
	svc.RegisterRoutingCallback("/b", func(r *http.Request, body map[string]any) map[string]any { return nil })
	svc.RegisterRoutingCallback("/a", func(r *http.Request, body map[string]any) map[string]any { return nil })

	got := svc.RoutingCallbackPaths()
	if len(got) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(got), got)
	}
	// Sorted order for determinism.
	if got[0] != "/a" || got[1] != "/b" {
		t.Fatalf("expected sorted paths [/a /b], got %v", got)
	}
}
