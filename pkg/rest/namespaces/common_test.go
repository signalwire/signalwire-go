// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

import "testing"

// mergeExtra is the funnel the generated Create/Update/Set* wrappers use to fold
// their Extras escape-hatch map into the request body. The typed params are
// written into body FIRST (each unconditionally, including its Go zero value),
// then Extras is merged. The teaching (see the Extras doc on each generated
// params struct + PORT_ADDITIONS.md) is that typed params are the intended
// surface and Extras is the escape hatch for genuinely-open / forward-compat
// fields. The collision contract is LAST-writer-wins with Extras applied last:
// an Extras key overrides a typed param of the same name — which is how a caller
// supplies a wire value for a field they left on the typed zero value.

func TestMergeExtra_ExtraOverridesTypedKey(t *testing.T) {
	// body already carries the typed params (here "to" left at its zero value "").
	body := map[string]any{"to": "", "from": "+15550000000"}
	mergeExtra(body, []map[string]any{{"to": "+15551234567"}})

	if body["to"] != "+15551234567" {
		t.Errorf("Extras must override a typed key set to its zero value; to = %v want +15551234567", body["to"])
	}
	if body["from"] != "+15550000000" {
		t.Errorf("a typed key with no Extras collision is unchanged; from = %v", body["from"])
	}
}

func TestMergeExtra_FillsOpenKey(t *testing.T) {
	body := map[string]any{"ttl": 42}
	mergeExtra(body, []map[string]any{{"member_id": "m-1"}})
	if body["member_id"] != "m-1" {
		t.Errorf("Extras must fill a key the typed surface omitted; member_id = %v want m-1", body["member_id"])
	}
	if body["ttl"] != 42 {
		t.Errorf("untouched typed key changed; ttl = %v", body["ttl"])
	}
}

func TestMergeExtra_EmptyIsNoOp(t *testing.T) {
	body := map[string]any{"ttl": 1}
	mergeExtra(body, nil)
	mergeExtra(body, []map[string]any{})
	mergeExtra(body, []map[string]any{nil})
	if len(body) != 1 || body["ttl"] != 1 {
		t.Errorf("empty/nil Extras must be a no-op; body = %v", body)
	}
}

func TestMergeExtra_LastWriterWinsAcrossMaps(t *testing.T) {
	body := map[string]any{}
	mergeExtra(body, []map[string]any{{"k": "first"}, {"k": "second"}})
	if body["k"] != "second" {
		t.Errorf("last Extras map should win on collision; got %v want second", body["k"])
	}
}
