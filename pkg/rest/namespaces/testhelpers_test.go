// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Shared test helpers for the namespaces mock-backed test suite.

package namespaces_test

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
