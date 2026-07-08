// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Shared test helpers for the namespaces mock-backed test suite.

package namespaces_test

import (
	"encoding/json"
	"testing"
)

// keys returns the keys of a map[string]any as a slice (order unspecified).
// Used by the mock-backed tests to assert on the set of fields a request body
// carried without depending on map iteration order.
func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// respMap normalises a generated REST method's response into a map[string]any so
// the mock-backed tests can assert on the decoded response fields the same way
// they did against the old loose map[string]any return. It accepts BOTH shapes a
// generated method now returns: the base CRUD / untyped-response methods still
// return map[string]any (passed through as-is), while the operation +
// command-dispatch methods with a typed ($ref) 200/201 response return a typed
// pointer (e.g. *CallResponse, *Document, *SearchResponse) which is JSON round-
// tripped back to a map. A nil typed pointer (204/empty body) yields an empty map.
func respMap(t *testing.T, v any) map[string]any {
	t.Helper()
	if v == nil {
		return map[string]any{}
	}
	// Already a loose map (base CRUD / untyped-response method) — pass through.
	if m, ok := v.(map[string]any); ok {
		return m
	}
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("respMap marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		// A response whose top-level JSON is not an object (rare) round-trips to a
		// non-map; the tests that use respMap only assert object responses.
		return map[string]any{}
	}
	return m
}
