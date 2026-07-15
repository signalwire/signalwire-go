// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// DOC-WIRE fixture (gate-enforcement plan §2.1).
//
// The cross-port DOC-WIRE gate (porting-sdk/scripts/doc_wire.py) spawns the
// mock, points this runner at it (MOCK_SIGNALWIRE_PORT / SIGNALWIRE_MOCK_URL),
// runs it, then reads the mock's wire_violations journal over HTTP. The gate
// fails if any request the fixture makes carries an unknown body key or query
// param — a mis-spelled wire field the SNIPPET-COMPILE type checker cannot catch
// because these bodies are untyped map[string]any / map[string]string.
//
// This fixture replays the wire-shape-sensitive calls that appear in go's REST
// docs + examples with UNTYPED bodies — the phone-number search `areacode`
// param and the flat-tts / `Extras` map bodies (play, record, transcribe,
// detect, collect, MFA) — verbatim to the doc examples. It is NOT an assertion
// test: it deliberately IGNORES every response error (the mock has no live
// resources, so many calls 4xx) and exits 0 — the gate reads wire correctness
// from the mock's journal, not from this program's stdout or per-call exit.
// What it proves is that every wire KEY these doc bodies emit is one the spec
// knows (go's are clean: `areacode`, not `area_code`).
//
// It reuses pkg/rest/internal/mocktest.New, which PROBES-then-reuses a mock
// already listening on MOCK_SIGNALWIRE_PORT (the one doc_wire.py pre-spawned),
// so the requests land in the journal the gate inspects.

package namespaces_test

import (
	"context"
	"testing"

	"github.com/signalwire/signalwire-go/v3/pkg/rest/internal/mocktest"
	"github.com/signalwire/signalwire-go/v3/pkg/rest/namespaces"
)

// TestDocWireFixtures fires the untyped-map wire shapes from go's REST docs and
// examples against the shared mock. Response errors are intentionally ignored:
// the DOC-WIRE gate reads wire_violations from the mock journal, so what matters
// is that each request REACHES the mock with spec-clean keys.
func TestDocWireFixtures(t *testing.T) {
	client, mock := mocktest.New(t)
	if client == nil {
		t.Skip("mock unavailable")
	}
	mock.Reset(t)

	ctx := context.Background()
	const callID = "call-docwire"

	// --- phone-number search: `areacode` (NOT `area_code`) --------------------
	// examples/quickstart_rest/main.go:40, rest/examples/rest_manage_resources.go:61,
	// rest/examples/rest_phone_number_management.go:35, README quickstart.
	_, _ = client.PhoneNumbers.Search(ctx, map[string]string{"areacode": "512"})
	_, _ = client.PhoneNumbers.Search(ctx, map[string]string{"areacode": "312", "max_results": "5"})

	// --- Calling.Play: flat-tts `{type, text}` play body ----------------------
	// rest/examples/rest_calling_play_and_record.go:63.
	_, _ = client.Calling.Play(ctx, callID, namespaces.CallingNamespacePlayParams{
		Play: []map[string]any{{"type": "tts", "text": "Welcome to SignalWire."}},
	})

	// --- Calling.Record: beep/format extras -----------------------------------
	// rest/examples/rest_calling_play_and_record.go:104.
	_, _ = client.Calling.Record(ctx, callID, namespaces.CallingNamespaceRecordParams{
		Extras: map[string]any{"beep": true, "format": "mp3"},
	})

	// --- Calling.Transcribe: language extra -----------------------------------
	// rest/examples/rest_calling_play_and_record.go:141.
	_, _ = client.Calling.Transcribe(ctx, callID, namespaces.CallingNamespaceTranscribeParams{
		Extras: map[string]any{"language": "en-US"},
	})

	// --- Calling.Detect / Collect ---------------------------------------------
	// rest/examples/rest_calling_ivr_and_ai.go:56,70.
	_, _ = client.Calling.Detect(ctx, callID, namespaces.CallingNamespaceDetectParams{
		Extras: map[string]any{"type": "machine"},
	})
	_, _ = client.Calling.Collect(ctx, callID, namespaces.CallingNamespaceCollectParams{
		Extras: map[string]any{
			"digits": map[string]any{"max": 4, "terminators": "#"},
			"play":   []map[string]any{{"type": "tts", "text": "Enter your PIN followed by pound."}},
		},
	})

	// --- MFA send: SMS/Call bodies (to/from/message/token_length) -------------
	// rest/examples/rest_queues_mfa_and_recordings.go:136,153.
	_, _ = client.MFA.SMS(ctx, namespaces.MFANamespaceSMSParams{
		Extras: map[string]any{
			"to": "+15551234567", "from": "+15559876543",
			"message": "Your code is {{code}}", "token_length": 6,
		},
	})
	_, _ = client.MFA.Call(ctx, namespaces.MFANamespaceCallParams{
		Extras: map[string]any{
			"to": "+15551234567", "from": "+15559876543",
			"message": "Your verification code is {{code}}", "token_length": 6,
		},
	})

	// Content-shaped assertions. The DOC-WIRE gate proves wire-KEY correctness via
	// the mock's wire_violations journal; these assertions prove the fixture
	// actually EXERCISED the wire (so an empty/short-circuited run can't pass the
	// gate vacuously) and that the load-bearing wire SHAPE — the phone-search
	// `areacode` query param — reached the server exactly as spelled. A regression
	// to `area_code` (the drift §2.1 guards against) makes this assertion fail here
	// AND the gate fail on the journaled unknown_query_param.
	journal := mock.Journal(t)
	if len(journal) < 9 {
		t.Fatalf("expected >=9 journaled doc-wire requests, got %d", len(journal))
	}
	var sawAreacodeSearch bool
	for _, e := range journal {
		if e.Path == "/api/relay/rest/phone_numbers/search" {
			if got := e.QueryParams["areacode"]; len(got) == 1 && got[0] == "512" {
				sawAreacodeSearch = true
			}
			if _, wrong := e.QueryParams["area_code"]; wrong {
				t.Errorf("phone search sent `area_code` (spec wire key is `areacode`): %v", e.QueryParams)
			}
		}
	}
	if !sawAreacodeSearch {
		t.Error("phone-number search with `areacode=512` did not reach the mock — fixture did not exercise the doc wire shape")
	}
}
