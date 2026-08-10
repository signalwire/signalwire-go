// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package swml

// x-sdk-widen — the marker that declares a schema field's const/enum union to
// be a documentation HINT rather than a closed set, because the platform
// accepts any value of the same base scalar type.
//
// Owner ruling: anything that can do something useful with that value should
// read it and use it.
//
// Enforcing such a union anyway invents a constraint the server does not have,
// so the SDK rejects documents that work on the wire. That is the failure
// direction nobody looks for: a validator gets audited for being too LOOSE, and
// this one was too STRICT. Go compiles the whole schema with a real Draft
// 2020-12 validator, so it DOES enforce const values — and cf62703 routed the
// render path through it, making the over-strictness live on the public API.
//
// Exactly one field carries the marker today: $defs/Hangup.reason, whose
// hangup|busy|decline anyOf-of-consts this validator was rejecting anything
// outside of.

import (
	"strings"
	"testing"
)

// widenTestSU builds a SchemaUtils with the embedded schema and full validation
// on — the default production configuration.
func widenTestSU(t *testing.T) *SchemaUtils {
	t.Helper()
	su := NewSchemaUtils()
	if !su.FullValidationAvailable() {
		t.Fatalf("full validator unavailable — this test asserts the behaviour OF that " +
			"validator, so without it the assertions would be vacuous")
	}
	return su
}

// TestWidenedFieldAcceptsUnlistedValue is the core contract: a value outside
// the marked union must VALIDATE, because the platform accepts it.
func TestWidenedFieldAcceptsUnlistedValue(t *testing.T) {
	su := widenTestSU(t)

	// "no_answer" is a real platform hangup reason that the bundled schema's
	// hangup|busy|decline union does not list.
	for _, reason := range []string{"no_answer", "cancel", "some_future_reason"} {
		res := su.ValidateVerb("hangup", map[string]any{"reason": reason})
		if !res.Valid {
			t.Errorf("hangup.reason=%q must validate — $defs/Hangup.reason carries "+
				"x-sdk-widen, so its const union is advisory and the platform accepts "+
				"any string. Rejecting it makes the SDK stricter than the server. errs=%v",
				reason, res.Errors)
		}
	}
}

// TestWidenedFieldStillAcceptsListedValues pins that widening did not break the
// documented values — the union members must keep validating.
func TestWidenedFieldStillAcceptsListedValues(t *testing.T) {
	su := widenTestSU(t)

	for _, reason := range []string{"hangup", "busy", "decline"} {
		res := su.ValidateVerb("hangup", map[string]any{"reason": reason})
		if !res.Valid {
			t.Errorf("hangup.reason=%q is a DOCUMENTED value and must still validate; errs=%v",
				reason, res.Errors)
		}
	}
}

// TestWidenedFieldStillEnforcesBaseType pins that widening drops the VALUE
// constraint only, never the TYPE. A widened string field must still reject a
// non-string — otherwise the marker would be a blanket opt-out of validation
// and any typo'd shape would sail through.
func TestWidenedFieldStillEnforcesBaseType(t *testing.T) {
	su := widenTestSU(t)

	for _, bad := range []any{42, true, []any{"busy"}, map[string]any{"x": 1}} {
		res := su.ValidateVerb("hangup", map[string]any{"reason": bad})
		if res.Valid {
			t.Errorf("hangup.reason=%#v (not a string) must still be REJECTED — "+
				"x-sdk-widen drops the value constraint, never the base type", bad)
		}
	}
}

// TestWidenDoesNotWidenUnmarkedUnions pins the blast radius: a const/enum union
// WITHOUT the marker keeps being enforced. A recursive transform that widened
// everything would silently disable value checking across the whole schema.
func TestWidenDoesNotWidenUnmarkedUnions(t *testing.T) {
	su := widenTestSU(t)

	// $defs/Record.format is a plain const union with NO x-sdk-widen.
	res := su.ValidateVerb("record", map[string]any{"format": "definitely-not-a-format"})
	if res.Valid {
		t.Errorf("record.format is NOT marked x-sdk-widen, so its union must stay " +
			"ENFORCED — widening must be opt-in per field, never global")
	}
}

// TestWidenSurvivesTheDocumentPath pins that the widening is applied to the
// compiled validator itself, so it holds on ValidateDocument too — not just on
// the per-verb wrapper. The two share one compiled schema; a fix applied only
// at the verb call site would leave whole-document renders still rejecting.
func TestWidenedFieldAcceptsUnlistedValueInFullDocument(t *testing.T) {
	su := widenTestSU(t)

	doc := map[string]any{
		"version": "1.0.0",
		"sections": map[string]any{
			"main": []any{map[string]any{"hangup": map[string]any{"reason": "no_answer"}}},
		},
	}
	res := su.ValidateDocument(doc)
	if !res.Valid {
		t.Errorf("a full document with a widened hangup.reason must validate; errs=%v", res.Errors)
	}
}

// TestWidenReachesThePublicAPI is the end-to-end row. Service.Hangup is the
// user-facing entry point, and cf62703 routed the render path through the
// validating one — so an over-strict validator is a live rejection of a valid
// call, not a latent one.
func TestWidenedReasonAcceptedByServiceHangup(t *testing.T) {
	for _, reason := range []string{"hangup", "busy", "decline", "no_answer", "cancel"} {
		svc := NewService(WithName("widen-probe"), WithBasicAuth("u", "p"))
		r := reason
		if err := svc.Hangup(&r); err != nil {
			t.Errorf("Service.Hangup(%q) returned %v — the platform accepts any string "+
				"reason ($defs/Hangup.reason is x-sdk-widen), so the SDK must not refuse it",
				reason, err)
		}
	}
}

// TestSchemaItselfIsUnchanged pins that widening happens on the copy fed to the
// validator compiler, NOT on the schema map callers read. GetVerbParameters and
// the codegen tooling must still see the DOCUMENTED values — the union is a
// hint for validation, but it is still the documentation.
func TestWidenLeavesTheReadableSchemaIntact(t *testing.T) {
	su := widenTestSU(t)

	params := su.GetVerbParameters("hangup")
	reason, ok := params["reason"].(map[string]any)
	if !ok {
		t.Fatalf("hangup.reason not found in verb parameters: %#v", params)
	}
	if _, has := reason["anyOf"]; !has {
		t.Errorf("the schema map callers read must KEEP hangup.reason's anyOf union — "+
			"widening applies to the compiled validator only, so docs/codegen still "+
			"see the documented values. got %#v", reason)
	}
	if w, _ := reason["x-sdk-widen"].(bool); !w {
		t.Errorf("hangup.reason must still carry x-sdk-widen in the readable schema")
	}
}

// TestApplySDKWidenStripsOnlyMarkedNodes exercises the transform directly, so a
// regression is reported at the transform rather than only as a downstream
// validation surprise.
func TestApplySDKWidenStripsOnlyMarkedNodes(t *testing.T) {
	in := map[string]any{
		"marked": map[string]any{
			"x-sdk-widen": true,
			"anyOf": []any{
				map[string]any{"type": "string", "const": "a"},
				map[string]any{"type": "string", "const": "b"},
			},
			"description": "keep me",
		},
		"unmarked": map[string]any{
			"anyOf": []any{map[string]any{"type": "string", "const": "a"}},
		},
		"nested": []any{
			map[string]any{"x-sdk-widen": true, "enum": []any{1, 2}, "type": "integer"},
		},
	}

	out, ok := applySDKWiden(in).(map[string]any)
	if !ok {
		t.Fatalf("applySDKWiden returned %T, want map", applySDKWiden(in))
	}

	marked, ok := out["marked"].(map[string]any)
	if !ok {
		t.Fatalf("marked node is %T, want map", out["marked"])
	}
	if _, has := marked["anyOf"]; has {
		t.Errorf("marked node must lose anyOf; got %#v", marked)
	}
	if marked["type"] != "string" {
		t.Errorf("marked node must recover the base type from its union branches; got %#v", marked)
	}
	if marked["description"] != "keep me" {
		t.Errorf("widening must preserve non-constraint keywords; got %#v", marked)
	}

	unmarked, ok := out["unmarked"].(map[string]any)
	if !ok {
		t.Fatalf("unmarked node is %T, want map", out["unmarked"])
	}
	if _, has := unmarked["anyOf"]; !has {
		t.Errorf("unmarked node must keep anyOf; got %#v", unmarked)
	}

	nestedArr, ok := out["nested"].([]any)
	if !ok || len(nestedArr) == 0 {
		t.Fatalf("nested is %T, want a non-empty array", out["nested"])
	}
	nested, ok := nestedArr[0].(map[string]any)
	if !ok {
		t.Fatalf("nested[0] is %T, want map", nestedArr[0])
	}
	if _, has := nested["enum"]; has {
		t.Errorf("widening must recurse into arrays; nested node kept enum: %#v", nested)
	}
	if nested["type"] != "integer" {
		t.Errorf("a marked node's own declared type must be preserved; got %#v", nested)
	}

	// The INPUT must be untouched — the transform returns a copy.
	origMarked, _ := in["marked"].(map[string]any)
	if _, has := origMarked["anyOf"]; !has {
		t.Errorf("applySDKWiden must not mutate its input")
	}
}

// TestWidenErrorTextNamesTheField guards the diagnostic: when a widened field
// IS rejected (wrong base type), the message must still point at it.
func TestWidenTypeErrorStillNamesTheVerb(t *testing.T) {
	su := widenTestSU(t)

	res := su.ValidateVerb("hangup", map[string]any{"reason": 42})
	if res.Valid {
		t.Fatalf("a non-string reason must be rejected")
	}
	if !strings.Contains(strings.Join(res.Errors, " "), "hangup") {
		t.Errorf("the rejection must name the verb; got %v", res.Errors)
	}
}
