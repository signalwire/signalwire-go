// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package swml

// hangup.reason — the SDK validates the value set the ENGINE validates.
//
// The engine's contract is stated once, in C, at
// mod_infrastructure/relay_apis.c:1105:
//
//	JSON_CHECK_STRING_MATCHES_OPTIONAL(reason, "hangup,cancel,busy,noAnswer,decline,error")
//
// and a non-match is a hard reject (libks ks_json_check.h: sets *error_msg and
// `return 0`). The SWML layer types the field as a bare string
// (swml_schema.c:1571) and swml.c forwards it verbatim into the `end` RPC on
// the same call, so the contract a document must satisfy is the COMPOSITION of
// the two layers — which is exactly these six values.
//
// This file replaces schema_widen_test.go, which asserted the opposite: that
// "no_answer" and "some_future_reason" must validate. Both are refused by the
// engine, so that test pinned a bug. The bundled schema previously listed only
// hangup|busy|decline and carried x-sdk-widen, and the SDK stripped the value
// set entirely at compile time — which accepted the three engine values the
// schema omitted, but also accepted everything else.

import "testing"

func reasonTestSU(t *testing.T) *SchemaUtils {
	t.Helper()
	su := NewSchemaUtils()
	if !su.FullValidationAvailable() {
		t.Fatalf("full validator unavailable — this test asserts the behaviour OF that " +
			"validator, so without it the assertions would be vacuous")
	}
	return su
}

// engineReasons is the closed set from relay_apis.c:1105, in source order.
// Note the camelCase "noAnswer": "no_answer" is NOT an engine value in any
// spelling, which is why it appears in the rejection test below.
var engineReasons = []string{"hangup", "cancel", "busy", "noAnswer", "decline", "error"}

// TestEngineReasonsValidate is the positive half: every value the engine
// accepts must validate here. Three of these (cancel, noAnswer, error) were
// absent from the bundled schema's old three-const union and validated only
// because the widen transform removed the constraint altogether.
func TestEngineReasonsValidate(t *testing.T) {
	su := reasonTestSU(t)

	for _, reason := range engineReasons {
		res := su.ValidateVerb("hangup", map[string]any{"reason": reason})
		if !res.Valid {
			t.Errorf("hangup.reason=%q is accepted by relay_apis.c:1105 and must "+
				"validate; errs=%v", reason, res.Errors)
		}
	}
}

// TestNonEngineReasonsAreRejected is the negative half, and it is the
// behaviour change: these previously validated. Rejecting them client-side is
// STRICTER and correct — the caller now gets a clear local error instead of an
// opaque server-side call failure.
func TestNonEngineReasonsAreRejected(t *testing.T) {
	su := reasonTestSU(t)

	// "no_answer" is the snake_case near-miss the old widen test asserted MUST
	// validate; the engine spells it "noAnswer" and refuses this one.
	for _, reason := range []string{"no_answer", "some_future_reason", "", "HANGUP"} {
		res := su.ValidateVerb("hangup", map[string]any{"reason": reason})
		if res.Valid {
			t.Errorf("hangup.reason=%q is REFUSED by relay_apis.c:1105, so the SDK "+
				"must reject it rather than emit a document the server will fail",
				reason)
		}
	}
}

// TestReasonStillEnforcesBaseType pins that a non-string is refused, so the
// enum did not become the only check.
func TestReasonStillEnforcesBaseType(t *testing.T) {
	su := reasonTestSU(t)

	for _, bad := range []any{42, true, []any{"busy"}, map[string]any{"x": 1}} {
		res := su.ValidateVerb("hangup", map[string]any{"reason": bad})
		if res.Valid {
			t.Errorf("hangup.reason=%#v is not a string and must be rejected", bad)
		}
	}
}

// TestReasonAcceptedByServiceHangup is the end-to-end row: the public API must
// accept every engine value and refuse a non-engine one.
func TestReasonAcceptedByServiceHangup(t *testing.T) {
	for _, reason := range engineReasons {
		svc := NewService(WithName("reason-probe"), WithBasicAuth("u", "p"))
		r := reason
		if err := svc.Hangup(&r); err != nil {
			t.Errorf("Service.Hangup(%q) returned %v — the engine accepts this reason",
				reason, err)
		}
	}

	svc := NewService(WithName("reason-probe"), WithBasicAuth("u", "p"))
	bad := "no_answer"
	if err := svc.Hangup(&bad); err == nil {
		t.Errorf("Service.Hangup(%q) must fail — the engine refuses it, so failing "+
			"locally is what saves the caller a server-side rejection", bad)
	}
}

// TestSchemaPublishesTheEngineValues guards the bundled artifact itself, so a
// re-vendor that reintroduces the three-value union or the x-sdk-widen marker
// is caught at the source rather than only through validator behaviour.
func TestSchemaPublishesTheEngineValues(t *testing.T) {
	su := reasonTestSU(t)

	params := su.GetVerbParameters("hangup")
	reason, ok := params["reason"].(map[string]any)
	if !ok {
		t.Fatalf("hangup.reason not found in verb parameters: %#v", params)
	}
	if _, has := reason["x-sdk-widen"]; has {
		t.Errorf("hangup.reason must NOT carry x-sdk-widen — the marker told the SDK " +
			"to stop validating a field the engine DOES validate")
	}
	raw, ok := reason["enum"].([]any)
	if !ok {
		t.Fatalf("hangup.reason must publish an enum; got %#v", reason)
	}
	got := make([]string, 0, len(raw))
	for _, v := range raw {
		s, _ := v.(string)
		got = append(got, s)
	}
	if len(got) != len(engineReasons) {
		t.Fatalf("hangup.reason enum = %v, want the six engine values %v", got, engineReasons)
	}
	for i, want := range engineReasons {
		if got[i] != want {
			t.Errorf("hangup.reason enum[%d] = %q, want %q (relay_apis.c:1105)", i, got[i], want)
		}
	}
}
