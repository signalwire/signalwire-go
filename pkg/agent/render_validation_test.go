package agent

import (
	"testing"

	"github.com/signalwire/signalwire-go/v3/pkg/swml"
)

// ---------------------------------------------------------------------------
// RenderSWMLForCall must emit ONLY schema-valid verbs.
//
// The defect this file pins: RenderSWMLForCall used to build a bare
// `swml.NewDocument()` that was never bound to a Service, then call the RAW
// `(*Document).AddVerb` for all six render phases. Document.AddVerb checks only
// that the verb NAME is non-empty — it never consults the schema. The agent
// embeds a *swml.Service whose ExecuteVerb DOES validate (service.go:686), so
// the validating path existed the whole time and the renderer simply did not use
// it. That is the same bypass the python reference does NOT have: the reference's
// _render_swml calls `agent_to_use.add_verb(...)` — the validating
// SWMLService.add_verb — for every one of its six phases
// (agent_base.py:1194,1199,1206,1216,1367,1372).
//
// The consequence was worst for the USER-SUPPLIED verbs (AddPreAnswerVerb /
// AddPostAnswerVerb / AddPostAiVerb): an arbitrary caller-provided verb name and
// config reached the rendered wire document with no schema consulted at all.
//
// These tests assert THROUGH the validator rather than against a literal blob, so
// they catch the next wrong key too.
// ---------------------------------------------------------------------------

// validateRenderedDoc pushes every verb of every section of a rendered document
// back through the SAME schema-validating entry point the renderer is required to
// use. Any verb the renderer emitted that the validator rejects is a wire-shape
// defect that was shipping.
func validateRenderedDoc(t *testing.T, doc map[string]any) {
	t.Helper()

	svc := swml.NewService()
	sections, ok := doc["sections"].(map[string]any)
	if !ok {
		t.Fatalf("rendered document has no sections map: %#v", doc)
	}
	for sectionName, raw := range sections {
		verbs, ok := raw.([]any)
		if !ok {
			t.Fatalf("section %q is not a verb list: %#v", sectionName, raw)
		}
		for i, v := range verbs {
			verb, ok := v.(map[string]any)
			if !ok {
				t.Fatalf("section %q verb %d is not a map: %#v", sectionName, i, v)
			}
			for name, cfg := range verb {
				if err := svc.ExecuteVerbToSection(sectionName, name, cfg); err != nil {
					t.Errorf("section %q verb %d (%q) is NOT schema-valid: %v\nconfig: %#v",
						sectionName, i, name, err, cfg)
				}
			}
		}
	}
}

// A fully-populated agent must render a document in which every emitted verb
// survives the validating entry point.
func TestRenderSWML_EveryVerbSurvivesTheValidator(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"), WithRecordCall(true))
	a.AddPreAnswerVerb("play", map[string]any{"url": "https://example.com/ring.mp3"})
	a.AddPostAnswerVerb("play", map[string]any{"url": "https://example.com/welcome.mp3"})
	a.AddPostAiVerb("hangup", map[string]any{})
	a.PromptAddSection("Role", "test", nil)

	doc := a.RenderSWML(nil, nil)
	validateRenderedDoc(t, doc)

	// The document must actually carry the six rendered phases in order — a
	// validator that silently dropped every verb would otherwise "pass".
	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)
	got := make([]string, 0, len(main))
	for _, v := range main {
		vm, ok := v.(map[string]any)
		if !ok {
			t.Fatalf("rendered verb is not a map: %#v", v)
		}
		for name := range vm {
			got = append(got, name)
		}
	}
	want := []string{"play", "answer", "record_call", "play", "ai", "hangup"}
	if len(got) != len(want) {
		t.Fatalf("rendered verbs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("verb[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// The minimal agent — no user verbs at all — must still render a valid document.
// This covers the internally-assembled answer / record_call / ai configs.
func TestRenderSWML_InternalVerbsSurviveTheValidator(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"), WithRecordCall(true))
	a.PromptAddSection("Role", "test", nil)

	doc := a.RenderSWML(nil, nil)
	validateRenderedDoc(t, doc)

	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)
	got := make([]string, 0, len(main))
	for _, v := range main {
		vm, ok := v.(map[string]any)
		if !ok {
			t.Fatalf("rendered verb is not a map: %#v", v)
		}
		for name := range vm {
			got = append(got, name)
		}
	}
	want := []string{"answer", "record_call", "ai"}
	if len(got) != len(want) {
		t.Fatalf("rendered verbs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("verb[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// A user-supplied verb with a config the schema REJECTS must not reach the
// rendered document. This is the severe shape: before the fix, an arbitrary
// caller config was written straight to the wire with no schema consulted.
func TestRenderSWML_DropsSchemaInvalidUserVerb(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	a.PromptAddSection("Role", "test", nil)
	// `play` has no `text` key — its config is PlayWithURL/PlayWithURLS and
	// spoken text goes through the `say:` URL scheme.
	a.AddPreAnswerVerb("play", map[string]any{"text": "hello"})

	doc := a.RenderSWML(nil, nil)
	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)

	for i, v := range main {
		verb, _ := v.(map[string]any)
		cfg, isPlay := verb["play"]
		if !isPlay {
			continue
		}
		cfgMap, _ := cfg.(map[string]any)
		if _, hasText := cfgMap["text"]; hasText {
			t.Errorf("verb %d: the renderer emitted a schema-forbidden play{text} config; "+
				"the render path must go through the validating entry point, got %#v", i, cfgMap)
		}
	}

	validateRenderedDoc(t, doc)
}

// An unknown verb NAME supplied by the caller must not reach the wire either.
func TestRenderSWML_DropsUnknownUserVerbName(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	a.PromptAddSection("Role", "test", nil)
	a.AddPostAnswerVerb("not_a_swml_verb", map[string]any{"whatever": 1})

	doc := a.RenderSWML(nil, nil)
	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)

	for i, v := range main {
		verb, _ := v.(map[string]any)
		if _, has := verb["not_a_swml_verb"]; has {
			t.Errorf("verb %d: the renderer emitted an unknown verb name; the render "+
				"path must go through the validating entry point", i)
		}
	}
}
