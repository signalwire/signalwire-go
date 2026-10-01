// Copyright (c) 2026 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Package capabilities reads what a client says it can render.
//
// A browser client — the SignalWire address widget, or anything speaking the
// same convention — declares its rendering capabilities in the user variables
// it sends at dial time:
//
//	{"vars": {"userVariables": {
//	    "capabilities": {"display_content": true, "transcript": true, ...},
//	    "metadata": {...}}}}
//
// These are declarations of what the client can RENDER, not grants of
// authority: treat them as hints for deciding what to offer (whether to push
// content to a screen, whether to advertise a text-handoff tool), never as
// permission to do anything privileged — a caller controls its own user
// variables.
//
// Absence means no. Every function here resolves errors and missing data to
// "not declared": offering a caller something they cannot reach (a PSTN caller
// has no browser) is worse than never mentioning it. There is deliberately no
// list of known capability names — a client may declare a capability this SDK
// has never heard of and an application can act on it today.
package capabilities

import "sort"

// UserVariables returns the user variables from a SWML request body
// (vars.userVariables), or an empty map. They are nested two levels down, which
// is easy to get silently wrong: a missing level yields an empty map and every
// downstream check reports "not declared".
func UserVariables(bodyParams any) map[string]any {
	body, ok := bodyParams.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	vars, ok := body["vars"].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	uv, ok := vars["userVariables"].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return uv
}

// DeclaredCapabilities returns the capability names the client declared as
// truthy, sorted. It accepts a full SWML request body or an already-extracted
// user-variables map, so it works from a dynamic-config callback and from a
// SWAIG handler alike. Empty when nothing was declared, the payload was
// malformed, or the client is not a browser at all.
//
//	if slices.Contains(capabilities.DeclaredCapabilities(body), "display_content") { ... }
func DeclaredCapabilities(bodyParams any) []string {
	variables := UserVariables(bodyParams)
	if len(variables) == 0 {
		if m, ok := bodyParams.(map[string]any); ok {
			variables = m // already-extracted user variables
		}
	}
	caps, ok := variables["capabilities"].(map[string]any)
	if !ok {
		return []string{}
	}
	out := make([]string, 0, len(caps))
	for name, value := range caps {
		if truthy(value) {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// HasCapability reports whether the client declared name, true only when it
// was explicitly declared truthy.
func HasCapability(bodyParams any, name string) bool {
	for _, c := range DeclaredCapabilities(bodyParams) {
		if c == name {
			return true
		}
	}
	return false
}

// truthy mirrors the reference's truthiness test on a decoded JSON value.
func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0
	case int:
		return t != 0
	case []any:
		return len(t) > 0
	case map[string]any:
		return len(t) > 0
	}
	return true
}
