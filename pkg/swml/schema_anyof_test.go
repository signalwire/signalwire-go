// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package swml

// The shallow closed-key check and anyOf/oneOf-shaped verb configs.
//
// verbTopLevelPropertyNames used to test `body["type"] == "object"` on the verb's
// config node and bail otherwise. A union node (`{"anyOf": [...]}`) carries no
// `type` of its own, so that test failed and the resolver returned
// (nil, false) — which ValidateVerbTopLevelKeys reads as "no key-set to
// enforce" and answers Valid for ANY key. The check did not report a problem; it
// stopped checking and reported success, which is the worse of the two.
//
// This was not hypothetical or contingent on a future schema: five verbs in the
// SHIPPED schema.json are union-shaped — connect and play (oneOf of $refs),
// send_sms (anyOf of $refs), sleep (anyOf of an object / integer / SWMLVar), and
// unset (anyOf of string / array). Four of the five have object branches whose
// keys are perfectly enumerable.
//
// The semantic: a config satisfying a union satisfies SOME branch, so the known
// keys are the UNION of the object branches' keys, and a key belonging to no
// branch belongs to no valid document. Non-object branches contribute nothing
// (they constrain the config to not be an object at all — a different question).
// unset has no object branch, so it correctly stays disengaged.

import (
	"sort"
	"strings"
	"testing"
)

// unionShapedVerbs are the verb configs the shipped schema expresses as an
// anyOf/oneOf, with the key set the union must resolve to and a legitimate
// config that must keep passing.
var unionShapedVerbs = []struct {
	verb      string
	wantKey   string // one key that must be in the resolved set
	legit     map[string]any
	wantCount int
}{
	{"sleep", "duration", map[string]any{"duration": 5000}, 1},
	{"play", "url", map[string]any{"url": "https://example.test/a.mp3"}, 8},
	{"send_sms", "body", map[string]any{
		"to_number": "+15551110000", "from_number": "+15552220000", "body": "hi",
	}, 6},
	{"connect", "to", map[string]any{"to": "sip:alice@example.test"}, 22},
}

// TestUnionShapedVerbsResolveAKeySet is the direct negative control: before the
// fix every one of these resolved to ok=false, i.e. the closed-key check was
// disengaged on them.
func TestUnionShapedVerbsResolveAKeySet(t *testing.T) {
	su := NewSchemaUtils()
	for _, tc := range unionShapedVerbs {
		known, ok := su.verbTopLevelPropertyNames(tc.verb)
		if !ok {
			t.Errorf("%s: closed-key check DISENGAGED on a union-shaped config; "+
				"it must resolve to the union of the object branches' keys", tc.verb)
			continue
		}
		if _, has := known[tc.wantKey]; !has {
			keys := make([]string, 0, len(known))
			for k := range known {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			t.Errorf("%s: resolved key set is missing %q; got %v", tc.verb, tc.wantKey, keys)
		}
		if len(known) != tc.wantCount {
			t.Errorf("%s: resolved %d keys, want %d", tc.verb, len(known), tc.wantCount)
		}
	}
}

// TestUnionShapedVerbsRejectUnknownKeys is the forbidden-key direction: a key
// present in no branch must be rejected. Every one of these was ACCEPTED before
// the fix.
func TestUnionShapedVerbsRejectUnknownKeys(t *testing.T) {
	su := NewSchemaUtils()
	for _, tc := range unionShapedVerbs {
		cfg := map[string]any{"zzz_not_a_real_key": 1}
		for k, v := range tc.legit {
			cfg[k] = v
		}
		res := su.ValidateVerbTopLevelKeys(tc.verb, cfg)
		if res.Valid {
			t.Errorf("%s: a key present in no branch was ACCEPTED — the closed-key "+
				"check is disengaged on this union-shaped config", tc.verb)
			continue
		}
		if !strings.Contains(strings.Join(res.Errors, " "), "zzz_not_a_real_key") {
			t.Errorf("%s: rejection must name the offending key; got %v", tc.verb, res.Errors)
		}
	}
}

// TestUnionShapedVerbsAcceptLegitimateConfigs is the other direction — the fix
// must not start rejecting valid documents. A branch-union that were computed as
// an INTERSECTION would fail here, since a key valid in one branch is absent
// from the others.
func TestUnionShapedVerbsAcceptLegitimateConfigs(t *testing.T) {
	su := NewSchemaUtils()
	for _, tc := range unionShapedVerbs {
		res := su.ValidateVerbTopLevelKeys(tc.verb, tc.legit)
		if !res.Valid {
			t.Errorf("%s: legitimate config rejected: %v", tc.verb, res.Errors)
		}
	}
}

// TestNonEnumerableConfigsStayDisengaged pins the shapes that genuinely have no
// closed key set, so the fix is not read as "always enforce something".
//
//   - set   — an OPEN object (unevaluatedProperties:{} with no `not`, zero
//     declared properties): a free-form variable bag by design.
//   - unset — a union with no object branch (string | array of string).
//   - cond / label / return — array / string / untyped, not objects at all.
func TestNonEnumerableConfigsStayDisengaged(t *testing.T) {
	su := NewSchemaUtils()
	for _, verb := range []string{"set", "unset", "cond", "label", "return"} {
		if _, ok := su.verbTopLevelPropertyNames(verb); ok {
			t.Errorf("%s has no closed key-set in the schema; the shallow check "+
				"must stay disengaged rather than invent one", verb)
		}
		// And the check itself must be a no-op, not a rejection.
		res := su.ValidateVerbTopLevelKeys(verb, map[string]any{"anything": 1})
		if !res.Valid {
			t.Errorf("%s: disengaged check must pass, got %v", verb, res.Errors)
		}
	}
}

// TestRefFollowingStillWorks guards the shape the resolver already handled — a
// single $ref (ai -> AIObject) — since the fix rewrote that path into the shared
// recursive resolver.
func TestRefFollowingStillWorks(t *testing.T) {
	su := NewSchemaUtils()
	known, ok := su.verbTopLevelPropertyNames("ai")
	if !ok {
		t.Fatal("ai: $ref to AIObject must still resolve to a closed key set")
	}
	for _, want := range []string{"prompt", "params", "SWAIG"} {
		if _, has := known[want]; !has {
			t.Errorf("ai: resolved key set is missing %q", want)
		}
	}
}
