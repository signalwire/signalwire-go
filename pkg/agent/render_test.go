package agent

import (
	"strings"
	"testing"

	"github.com/signalwire/signalwire-go/pkg/swaig"

	_ "github.com/signalwire/signalwire-go/pkg/skills/all"
)

// ---------------------------------------------------------------------------
// Rendering phase order: pre-answer -> answer -> record -> post-answer -> ai -> post-ai
// ---------------------------------------------------------------------------

func TestRenderSWML_AllPhases(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"), WithRecordCall(true))
	a.AddPreAnswerVerb("play", map[string]any{"url": "ring.mp3"})
	a.AddPostAnswerVerb("play", map[string]any{"url": "welcome.mp3"})
	a.AddPostAiVerb("hangup", map[string]any{})
	a.PromptAddSection("Role", "test", nil)

	doc := a.RenderSWML(nil, nil)
	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)

	// Expected order: pre-play, answer, record_call, post-play, ai, hangup
	if len(main) < 6 {
		t.Fatalf("expected at least 6 verbs, got %d", len(main))
	}

	verbNames := make([]string, 0, len(main))
	for _, v := range main {
		vm, _ := v.(map[string]any)
		for key := range vm {
			verbNames = append(verbNames, key)
		}
	}

	expected := []string{"play", "answer", "record_call", "play", "ai", "hangup"}
	for i, exp := range expected {
		if i >= len(verbNames) {
			t.Fatalf("missing verb at index %d, expected %q", i, exp)
		}
		if verbNames[i] != exp {
			t.Errorf("verb[%d] = %q, want %q", i, verbNames[i], exp)
		}
	}
}

// ---------------------------------------------------------------------------
// SWML document structure
// ---------------------------------------------------------------------------

func TestRenderSWML_HasVersionAndSections(t *testing.T) {
	a := NewAgentBase()
	doc := a.RenderSWML(nil, nil)

	if doc["version"] != "1.0.0" {
		t.Errorf("version = %v, want %q", doc["version"], "1.0.0")
	}
	sections, ok := doc["sections"].(map[string]any)
	if !ok {
		t.Fatal("expected sections map")
	}
	if _, ok := sections["main"]; !ok {
		t.Error("expected main section")
	}
}

// ---------------------------------------------------------------------------
// Empty prompt
// ---------------------------------------------------------------------------

func TestRenderSWML_EmptyPrompt(t *testing.T) {
	a := NewAgentBase()
	// No prompt set, POM mode with empty sections
	doc := a.RenderSWML(nil, nil)

	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)

	for _, v := range main {
		vm, _ := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			if _, ok := aiCfg["prompt"]; ok {
				t.Error("empty POM should not produce a prompt key")
			}
			return
		}
	}
	t.Fatal("AI verb not found")
}

// ---------------------------------------------------------------------------
// Full rendering with all AI config options
// ---------------------------------------------------------------------------

func TestRenderSWML_AllAIConfigOptions(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	a.SetPromptText("You are a bot")
	a.SetPostPrompt("Summarize the call")
	a.SetParams(map[string]any{"temperature": 0.5})
	a.AddHints([]string{"SignalWire", "API"})
	a.SetLanguages([]map[string]any{{"code": "en-US"}})
	a.AddPronunciation("API", "A P I")
	a.SetGlobalData(map[string]any{"company": "SignalWire"})
	a.SetNativeFunctions([]string{"stop"})
	a.AddPatternHint("numbers", "\\d+", "NUM")
	a.EnableDebugEvents(1)
	a.SetPromptLlmParams(map[string]any{"top_p": 0.9})
	a.SetPostPromptLlmParams(map[string]any{"max_tokens": 200})

	a.DefineTool(ToolDefinition{
		Name:        "greet",
		Description: "Greet the user",
		Handler: func(args map[string]any, raw map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("Hello!")
		},
	})

	doc := a.RenderSWML(nil, nil)
	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)

	for _, v := range main {
		vm, _ := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			// Check prompt
			if aiCfg["prompt"] == nil {
				t.Error("expected prompt")
			}
			// Check post_prompt
			if aiCfg["post_prompt"] == nil {
				t.Error("expected post_prompt")
			}
			// Check post_prompt_url
			if aiCfg["post_prompt_url"] == nil {
				t.Error("expected post_prompt_url")
			}
			// Check params
			params, _ := aiCfg["params"].(map[string]any)
			if params["temperature"] != 0.5 {
				t.Errorf("temperature = %v", params["temperature"])
			}
			// Check hints — a single mixed array: 2 plain-string hints plus the
			// 1 structured pattern hint (Python appends pattern hints into hints).
			hints, _ := aiCfg["hints"].([]any)
			if len(hints) != 3 {
				t.Errorf("expected 3 hints (2 string + 1 pattern), got %d", len(hints))
			}
			var patternFound bool
			for _, h := range hints {
				if hm, ok := h.(map[string]any); ok {
					if hm["hint"] == "numbers" && hm["pattern"] == "\\d+" && hm["replace"] == "NUM" {
						patternFound = true
					}
				}
			}
			if !patternFound {
				t.Error("expected the structured pattern hint to be merged into hints")
			}
			// Check languages
			langs, _ := aiCfg["languages"].([]map[string]any)
			if len(langs) != 1 {
				t.Errorf("expected 1 language, got %d", len(langs))
			}
			// Check pronunciations
			if aiCfg["pronounce"] == nil {
				t.Error("expected pronounce")
			}
			// Check global_data
			if aiCfg["global_data"] == nil {
				t.Error("expected global_data")
			}
			// Check native_functions
			nf, _ := aiCfg["native_functions"].([]string)
			if len(nf) != 1 || nf[0] != "stop" {
				t.Errorf("native_functions = %v", nf)
			}
			// Pattern hints render inside the hints array (checked above), not
			// under a separate pattern_hints key (Python parity).
			if aiCfg["pattern_hints"] != nil {
				t.Errorf("unexpected pattern_hints key: %v", aiCfg["pattern_hints"])
			}
			// Check debug events wiring (Python parity: params.debug_webhook_url
			// + params.debug_webhook_level; no separate ai.debug_events key).
			if aiCfg["debug_events"] != nil {
				t.Errorf("unexpected ai.debug_events key: %v", aiCfg["debug_events"])
			}
			if params["debug_webhook_level"] != 1 {
				t.Errorf("debug_webhook_level = %v", params["debug_webhook_level"])
			}
			if u, _ := params["debug_webhook_url"].(string); !strings.Contains(u, "/debug_events") {
				t.Errorf("debug_webhook_url = %v", params["debug_webhook_url"])
			}
			// Check SWAIG
			if aiCfg["SWAIG"] == nil {
				t.Error("expected SWAIG")
			}
			return
		}
	}
	t.Fatal("AI verb not found")
}

// ---------------------------------------------------------------------------
// Contexts rendering
// ---------------------------------------------------------------------------

func TestRenderSWML_WithContexts_Render(t *testing.T) {
	a := NewAgentBase()
	cb := a.DefineContexts()
	ctx := cb.AddContext("default")
	step := ctx.AddStep("greeting")
	step.SetStepCriteria("User has been greeted")

	doc := a.RenderSWML(nil, nil)
	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)

	for _, v := range main {
		vm, _ := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			if aiCfg["contexts"] == nil {
				t.Error("expected contexts in AI config")
			}
			return
		}
	}
	t.Fatal("AI verb not found")
}

// ---------------------------------------------------------------------------
// Function includes rendering
// ---------------------------------------------------------------------------

func TestRenderSWML_WithFunctionIncludes_Render(t *testing.T) {
	a := NewAgentBase()
	a.AddFunctionInclude("https://example.com/swaig", []string{"fn1", "fn2"}, map[string]any{"token": "abc"})

	doc := a.RenderSWML(nil, nil)
	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)

	for _, v := range main {
		vm, _ := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			swaigCfg, _ := aiCfg["SWAIG"].(map[string]any)
			if swaigCfg == nil {
				t.Fatal("expected SWAIG config")
			}
			includes, _ := swaigCfg["includes"].([]map[string]any)
			if len(includes) != 1 {
				t.Fatalf("expected 1 include, got %d", len(includes))
			}
			if includes[0]["url"] != "https://example.com/swaig" {
				t.Errorf("include url = %v", includes[0]["url"])
			}
			return
		}
	}
	t.Fatal("AI verb not found")
}

// ---------------------------------------------------------------------------
// Clone does not mutate original
// ---------------------------------------------------------------------------

func TestClone_IndependentCopy(t *testing.T) {
	a := NewAgentBase()
	a.PromptAddSection("Original", "text", nil)
	a.AddHints([]string{"hint1"})
	a.SetParam("temperature", 0.5)

	c := a.clone()

	// Modify clone
	c.PromptAddSection("Clone Section", "", nil)
	c.AddHints([]string{"hint2"})
	c.SetParam("temperature", 0.9)

	// Original should be unmodified
	if len(a.pomSections) != 1 {
		t.Errorf("original should have 1 section, got %d", len(a.pomSections))
	}
	if len(a.hints) != 1 {
		t.Errorf("original should have 1 hint, got %d", len(a.hints))
	}
	if a.params["temperature"] != 0.5 {
		t.Errorf("original temperature = %v, want 0.5", a.params["temperature"])
	}
}

// ---------------------------------------------------------------------------
// Skill integration rendering
// ---------------------------------------------------------------------------

func TestRenderSWML_WithSkill(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	a.AddSkill("datetime", nil)

	doc := a.RenderSWML(nil, nil)
	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)

	for _, v := range main {
		vm, _ := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			swaigCfg, _ := aiCfg["SWAIG"].(map[string]any)
			if swaigCfg == nil {
				t.Fatal("expected SWAIG config with skill tools")
			}
			fns, _ := swaigCfg["functions"].([]map[string]any)
			if len(fns) == 0 {
				t.Error("expected at least 1 function from datetime skill")
			}
			// Should also have hints (rendered as a mixed []any array).
			hints, _ := aiCfg["hints"].([]any)
			if len(hints) == 0 {
				t.Error("expected hints from datetime skill")
			}
			return
		}
	}
	t.Fatal("AI verb not found")
}
