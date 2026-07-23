package swml

import "testing"

// SWML STRICT-RENDER (Wave-2 P#5): building an SWML document with a misshapen
// config, an unknown verb, or a misspelled/unknown key must RAISE (return an
// error) — not silently drop or accept it. A valid build must still render.
// These tests mirror signalwire-python
// tests/unit/core/test_swml_strict_render.py, ported to the Go SWMLService
// surface (Service.ExecuteVerb == python add_verb) with schema validation ON.

func strictService(t *testing.T) *Service {
	t.Helper()
	return NewService(
		WithName("s"),
		WithRoute("/s"),
		WithSchemaValidation(true),
	)
}

func TestStrictUnknownVerbRaises(t *testing.T) {
	if err := strictService(t).ExecuteVerb("foobar", map[string]any{}); err == nil {
		t.Fatal("unknown verb 'foobar' must raise, got nil error (silent-accept)")
	}
}

func TestStrictGoodVerbRenders(t *testing.T) {
	if err := strictService(t).ExecuteVerb("answer", map[string]any{"max_duration": 5}); err != nil {
		t.Fatalf("valid answer verb must render, got error: %v", err)
	}
}

// Misspelled/unknown keys on closed verbs must raise (the r5 silent-drop
// family). Table mirrors the parametrized python test.
func TestStrictMisspelledOrUnknownKeyRaises(t *testing.T) {
	cases := []struct {
		name   string
		verb   string
		config map[string]any
	}{
		{"answer_misspelled", "answer", map[string]any{"maxduration": 5}},
		{"answer_unknown", "answer", map[string]any{"wibble": 1}},
		{"play_misspelled", "play", map[string]any{"urlz": []any{"say:hi"}}},
		{"play_valid_plus_unknown", "play", map[string]any{"url": "say:hi", "foo": 1}},
		{"record_misspelled", "record", map[string]any{"formatt": "wav"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := strictService(t).ExecuteVerb(tc.verb, tc.config); err == nil {
				t.Fatalf("verb %q with config %v must raise (misspelled/unknown key), got nil error",
					tc.verb, tc.config)
			}
		})
	}
}

func TestStrictWrongTypedConfigRaises(t *testing.T) {
	// max_duration must be numeric; a string must raise.
	if err := strictService(t).ExecuteVerb("answer", map[string]any{"max_duration": "notanumber"}); err == nil {
		t.Fatal("answer.max_duration must be numeric; a string must raise, got nil error")
	}
}

func TestStrictGoodVerbsRender(t *testing.T) {
	if err := strictService(t).ExecuteVerb("play", map[string]any{"url": "say:hi"}); err != nil {
		t.Fatalf("valid play verb must render, got error: %v", err)
	}
}

// ---- the ai verb: closed at the top level, open under params ----

func TestStrictAIGoodConfigRenders(t *testing.T) {
	if err := strictService(t).ExecuteVerb("ai", map[string]any{"prompt": map[string]any{"text": "hi"}}); err != nil {
		t.Fatalf("valid ai verb must render, got error: %v", err)
	}
}

func TestStrictAIMisspelledTopLevelKeyRaises(t *testing.T) {
	// GAP1: a misspelled top-level ai key ('temperatur') must raise even though
	// the specialized handler previously accepted it silently.
	err := strictService(t).ExecuteVerb("ai", map[string]any{
		"prompt": map[string]any{"text": "hi"}, "temperatur": 0.5,
	})
	if err == nil {
		t.Fatal("misspelled top-level ai key 'temperatur' must raise, got nil error")
	}
}

func TestStrictAIUnknownTopLevelKeyRaises(t *testing.T) {
	// GAP1: an unknown top-level ai key ('zzz') must raise.
	err := strictService(t).ExecuteVerb("ai", map[string]any{
		"prompt": map[string]any{"text": "hi"}, "zzz": 1,
	})
	if err == nil {
		t.Fatal("unknown top-level ai key 'zzz' must raise, got nil error")
	}
}

func TestStrictAIMissingPromptRaises(t *testing.T) {
	// The ai verb requires a prompt; omitting it must raise.
	err := strictService(t).ExecuteVerb("ai", map[string]any{"post_prompt": map[string]any{"text": "bye"}})
	if err == nil {
		t.Fatal("ai verb without a prompt must raise, got nil error")
	}
}

func TestStrictAIParamsSubobjectStaysOpen(t *testing.T) {
	// ai.params is the DELIBERATE open door for LLM tuning: an arbitrary key
	// inside it is not a misspelling and must render.
	err := strictService(t).ExecuteVerb("ai", map[string]any{
		"prompt": map[string]any{"text": "hi"},
		"params": map[string]any{"some_future_param": 1},
	})
	if err != nil {
		t.Fatalf("ai.params extras must stay open (render), got error: %v", err)
	}
}

// The ai verb gets a SHALLOW top-level-key check (its deep shape is owned by
// the handler), NOT full deep schema validation. Full-deep-validating the ai
// verb would FALSE-REJECT legitimate deep emissions the bundled JSON schema
// does not fully accept — an empty prompt.pom, SWAIG.defaults, and
// functions[].web_hook_url/__token. These must all still render.
func TestStrictAIDeepShapesStillRender(t *testing.T) {
	cases := []struct {
		name   string
		config map[string]any
	}{
		{"empty_pom", map[string]any{"prompt": map[string]any{"pom": []any{}}}},
		{"swaig_defaults", map[string]any{
			"prompt": map[string]any{"text": "hi"},
			"SWAIG":  map[string]any{"defaults": map[string]any{"web_hook_url": "http://x"}},
		}},
		{"swaig_fn_webhook_and_token", map[string]any{
			"prompt": map[string]any{"text": "hi"},
			"SWAIG": map[string]any{"functions": []any{map[string]any{
				"function": "f", "description": "d", "parameters": map[string]any{},
				"web_hook_url": "http://x", "__token": "t",
			}}},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := strictService(t).ExecuteVerb("ai", tc.config); err != nil {
				t.Fatalf("legitimate deep ai shape %q must render (top-level-key check only), got error: %v",
					tc.name, err)
			}
		})
	}
}
