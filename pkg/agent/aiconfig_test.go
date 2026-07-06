package agent

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Hints
// ---------------------------------------------------------------------------

func TestAddHint_Single(t *testing.T) {
	a := NewAgentBase()
	a.AddHint("SignalWire")
	if len(a.hints) != 1 || a.hints[0] != "SignalWire" {
		t.Errorf("hints = %v, want [SignalWire]", a.hints)
	}
}

func TestAddHints_Multiple(t *testing.T) {
	a := NewAgentBase()
	a.AddHints([]string{"hello", "world"})
	a.AddHints([]string{"foo"})
	if len(a.hints) != 3 {
		t.Errorf("expected 3 hints, got %d", len(a.hints))
	}
}

func TestAddPatternHint_WithLanguage(t *testing.T) {
	a := NewAgentBase()
	// Python-aligned signature: AddPatternHint(hint, pattern, replace, ignoreCase...)
	a.AddPatternHint("digits", "\\d{3}", "NUM")
	if len(a.patternHints) != 1 {
		t.Fatalf("expected 1 pattern hint, got %d", len(a.patternHints))
	}
	ph := a.patternHints[0]
	if ph["pattern"] != "\\d{3}" {
		t.Errorf("pattern = %v", ph["pattern"])
	}
	if ph["hint"] != "digits" {
		t.Errorf("hint = %v", ph["hint"])
	}
	if ph["replace"] != "NUM" {
		t.Errorf("replace = %v", ph["replace"])
	}
}

func TestAddPatternHint_WithoutLanguage(t *testing.T) {
	a := NewAgentBase()
	// Python-aligned: add_pattern_hint always stores ignore_case (default False),
	// so the structured hint carries ignore_case=false when not explicitly set.
	a.AddPatternHint("letters", "[A-Z]+", "WORD")
	ph := a.patternHints[0]
	v, ok := ph["ignore_case"]
	if !ok {
		t.Error("ignore_case should always be stored (Python parity)")
	}
	if v != false {
		t.Errorf("expected ignore_case=false, got %v", v)
	}
}

func TestAddPatternHint_IgnoreCase(t *testing.T) {
	a := NewAgentBase()
	a.AddPatternHint("digits", "\\d+", "NUM", true)
	ph := a.patternHints[0]
	if ph["ignore_case"] != true {
		t.Error("expected ignore_case=true")
	}
}

// ---------------------------------------------------------------------------
// Languages
// ---------------------------------------------------------------------------

func TestAddLanguage_Single(t *testing.T) {
	a := NewAgentBase()
	a.AddLanguage(map[string]any{"code": "en-US", "name": "English"})
	if len(a.languages) != 1 {
		t.Fatalf("expected 1 language, got %d", len(a.languages))
	}
	if a.languages[0]["code"] != "en-US" {
		t.Errorf("language code = %v", a.languages[0]["code"])
	}
}

func TestSetLanguages_ReplaceAll(t *testing.T) {
	a := NewAgentBase()
	a.AddLanguage(map[string]any{"code": "en-US"})
	a.SetLanguages([]map[string]any{
		{"code": "fr-FR"},
		{"code": "es-ES"},
	})
	if len(a.languages) != 2 {
		t.Errorf("expected 2 languages, got %d", len(a.languages))
	}
	if a.languages[0]["code"] != "fr-FR" {
		t.Errorf("first language = %v", a.languages[0]["code"])
	}
}

// SetMultilingual mirrors Python AIConfigMixin.set_multilingual: it stores the
// config and the AI verb renders a top-level "multilingual" object.
func TestSetMultilingual_Stores(t *testing.T) {
	a := NewAgentBase()
	got := a.SetMultilingual(map[string]any{"start_language": "en-US", "min_switch_words": 2})
	if got != a {
		t.Error("SetMultilingual should return receiver for chaining")
	}
	if a.multilingual["start_language"] != "en-US" || a.multilingual["min_switch_words"] != 2 {
		t.Errorf("multilingual = %#v", a.multilingual)
	}
}

func TestSetMultilingual_EmptyIsNoop(t *testing.T) {
	a := NewAgentBase()
	a.SetMultilingual(map[string]any{})
	if a.multilingual != nil {
		t.Errorf("empty config should be a no-op; got %#v", a.multilingual)
	}
}

func TestSetMultilingual_RendersTopLevelObject(t *testing.T) {
	a := NewAgentBase()
	a.SetMultilingual(map[string]any{"start_language": "en-US"})

	doc := a.RenderSWML(nil, nil)
	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)
	for _, v := range main {
		vm, _ := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			ml, _ := aiCfg["multilingual"].(map[string]any)
			if ml == nil {
				t.Fatalf("expected top-level multilingual object on AI verb; got %#v", aiCfg["multilingual"])
			}
			if ml["start_language"] != "en-US" {
				t.Errorf("multilingual.start_language = %v", ml["start_language"])
			}
			return
		}
	}
	t.Fatal("no ai verb found in rendered SWML")
}

// ---------------------------------------------------------------------------
// Per-language params: AddLanguageTyped(...params), SetLanguageParams,
// GetLanguageParams. Mirrors Python TestPerLanguageParams (11 cases) — see
// signalwire-python tests/unit/core/mixins/test_ai_config_mixin.py.
// ---------------------------------------------------------------------------

func TestAddLanguageTyped_WithParams_AttachesParams(t *testing.T) {
	a := NewAgentBase()
	a.AddLanguageTyped("English", "en-US", "josh", nil, nil, "elevenlabs", "",
		map[string]any{"stability": 0.5, "similarity_boost": 0.75})
	got, _ := a.languages[0]["params"].(map[string]any)
	if got == nil {
		t.Fatalf("expected params map on language; got %#v", a.languages[0])
	}
	if got["stability"] != 0.5 || got["similarity_boost"] != 0.75 {
		t.Errorf("params = %v; want stability=0.5, similarity_boost=0.75", got)
	}
}

func TestAddLanguageTyped_WithoutParams_OmitsKey(t *testing.T) {
	a := NewAgentBase()
	a.AddLanguageTyped("French", "fr-FR", "fr-FR-Neural2-A", nil, nil, "", "")
	if _, ok := a.languages[0]["params"]; ok {
		t.Errorf("expected no params key when none passed; got %#v", a.languages[0])
	}
}

func TestAddLanguageTyped_WithEmptyParams_OmitsKey(t *testing.T) {
	a := NewAgentBase()
	a.AddLanguageTyped("French", "fr-FR", "v", nil, nil, "", "", map[string]any{})
	if _, ok := a.languages[0]["params"]; ok {
		t.Errorf("expected no params key for empty map; got %#v", a.languages[0])
	}
}

func TestGetLanguageParams_ReturnsSetDict(t *testing.T) {
	a := NewAgentBase()
	a.AddLanguageTyped("English", "en-US", "v", nil, nil, "", "", map[string]any{"a": 1})
	got := a.LanguageParams("en-US")
	if got == nil {
		t.Fatalf("expected params map; got nil")
	}
	if got["a"] != 1 {
		t.Errorf("params[a] = %v; want 1", got["a"])
	}
}

func TestGetLanguageParams_ReturnsNilWhenUnset(t *testing.T) {
	a := NewAgentBase()
	a.AddLanguageTyped("English", "en-US", "v", nil, nil, "", "")
	if got := a.LanguageParams("en-US"); got != nil {
		t.Errorf("expected nil for unset params; got %v", got)
	}
}

func TestGetLanguageParams_ReturnsNilForUnknownCode(t *testing.T) {
	a := NewAgentBase()
	if got := a.LanguageParams("zh-CN"); got != nil {
		t.Errorf("expected nil for unknown code; got %v", got)
	}
}

func TestSetLanguageParams_ReplacesExisting(t *testing.T) {
	a := NewAgentBase()
	a.AddLanguageTyped("English", "en-US", "v", nil, nil, "", "", map[string]any{"a": 1})
	a.SetLanguageParams("en-US", map[string]any{"b": 2})
	got := a.LanguageParams("en-US")
	if got == nil || got["b"] != 2 {
		t.Errorf("expected params = {b:2}; got %v", got)
	}
	if _, hasA := got["a"]; hasA {
		t.Errorf("replacement should drop old keys; got %v", got)
	}
}

func TestSetLanguageParams_AddsWhenUnset(t *testing.T) {
	a := NewAgentBase()
	a.AddLanguageTyped("English", "en-US", "v", nil, nil, "", "")
	a.SetLanguageParams("en-US", map[string]any{"c": 3})
	got := a.LanguageParams("en-US")
	if got == nil || got["c"] != 3 {
		t.Errorf("expected params = {c:3}; got %v", got)
	}
}

func TestSetLanguageParams_EmptyDictRemovesKey(t *testing.T) {
	a := NewAgentBase()
	a.AddLanguageTyped("English", "en-US", "v", nil, nil, "", "", map[string]any{"a": 1})
	a.SetLanguageParams("en-US", map[string]any{})
	if got := a.LanguageParams("en-US"); got != nil {
		t.Errorf("expected nil after empty-dict set; got %v", got)
	}
	if _, ok := a.languages[0]["params"]; ok {
		t.Errorf("params key should be removed from language map; got %v", a.languages[0])
	}
}

func TestSetLanguageParams_UnknownCodeIsNoop(t *testing.T) {
	a := NewAgentBase()
	a.AddLanguageTyped("English", "en-US", "v", nil, nil, "", "")
	a.SetLanguageParams("zh-CN", map[string]any{"a": 1})
	// The known language remains untouched.
	if _, ok := a.languages[0]["params"]; ok {
		t.Errorf("untouched language should still have no params; got %v", a.languages[0])
	}
}

func TestSetLanguageParams_ReturnsSelfForChaining(t *testing.T) {
	a := NewAgentBase()
	a.AddLanguageTyped("English", "en-US", "v", nil, nil, "", "")
	if got := a.SetLanguageParams("en-US", map[string]any{"a": 1}); got != a {
		t.Errorf("SetLanguageParams should return self for chaining")
	}
}

// ---------------------------------------------------------------------------
// Pronunciations
// ---------------------------------------------------------------------------

func TestAddPronunciation_Basic(t *testing.T) {
	a := NewAgentBase()
	// Python-aligned signature: AddPronunciation(replace, withText, ignoreCase...)
	a.AddPronunciation("API", "A P I")
	if len(a.pronunciations) != 1 {
		t.Fatalf("expected 1 pronunciation, got %d", len(a.pronunciations))
	}
	p := a.pronunciations[0]
	if p["replace"] != "API" {
		t.Errorf("replace = %v", p["replace"])
	}
	if p["with"] != "A P I" {
		t.Errorf("with = %v", p["with"])
	}
}

func TestAddPronunciation_IgnoreCase(t *testing.T) {
	a := NewAgentBase()
	a.AddPronunciation("API", "A P I", true)
	if a.pronunciations[0]["ignore_case"] != true {
		t.Error("expected ignore_case=true")
	}
}

func TestSetPronunciations_ReplaceAll(t *testing.T) {
	a := NewAgentBase()
	a.AddPronunciation("X", "Y")
	a.SetPronunciations([]map[string]any{
		{"replace": "A", "with": "B"},
	})
	if len(a.pronunciations) != 1 {
		t.Errorf("expected 1 pronunciation after SetPronunciations, got %d", len(a.pronunciations))
	}
}

// ---------------------------------------------------------------------------
// Params
// ---------------------------------------------------------------------------

func TestSetParam_Single(t *testing.T) {
	a := NewAgentBase()
	a.SetParam("temperature", 0.7)
	if a.params["temperature"] != 0.7 {
		t.Errorf("params[temperature] = %v, want 0.7", a.params["temperature"])
	}
}

func TestSetParams_ReplaceAll(t *testing.T) {
	a := NewAgentBase()
	a.SetParam("temperature", 0.5)
	a.SetParams(map[string]any{"top_p": 0.9})
	if _, ok := a.params["temperature"]; ok {
		t.Error("SetParams should replace all params")
	}
	if a.params["top_p"] != 0.9 {
		t.Errorf("params[top_p] = %v, want 0.9", a.params["top_p"])
	}
}

// ---------------------------------------------------------------------------
// Global data
// ---------------------------------------------------------------------------

func TestSetGlobalData_Merge(t *testing.T) {
	a := NewAgentBase()
	a.UpdateGlobalData(map[string]any{"old": "data"})
	// SetGlobalData MERGES (Python parity: set_global_data is a .update()), so
	// the prior "old" key survives alongside the newly added "new" key.
	a.SetGlobalData(map[string]any{"new": "data"})
	if a.globalData["old"] != "data" {
		t.Error("SetGlobalData should merge (keep existing keys)")
	}
	if a.globalData["new"] != "data" {
		t.Errorf("globalData[new] = %v", a.globalData["new"])
	}
}

func TestUpdateGlobalData_Merge(t *testing.T) {
	a := NewAgentBase()
	a.UpdateGlobalData(map[string]any{"key1": "val1"})
	a.UpdateGlobalData(map[string]any{"key2": "val2"})
	if a.globalData["key1"] != "val1" {
		t.Errorf("globalData[key1] = %v", a.globalData["key1"])
	}
	if a.globalData["key2"] != "val2" {
		t.Errorf("globalData[key2] = %v", a.globalData["key2"])
	}
}

func TestUpdateGlobalData_OverwriteExisting(t *testing.T) {
	a := NewAgentBase()
	a.UpdateGlobalData(map[string]any{"key": "v1"})
	a.UpdateGlobalData(map[string]any{"key": "v2"})
	if a.globalData["key"] != "v2" {
		t.Errorf("globalData[key] = %v, want %q", a.globalData["key"], "v2")
	}
}

// ---------------------------------------------------------------------------
// Native functions
// ---------------------------------------------------------------------------

func TestSetNativeFunctions_Basic(t *testing.T) {
	a := NewAgentBase()
	a.SetNativeFunctions([]string{"func1", "func2"})
	if len(a.nativeFunctions) != 2 {
		t.Errorf("expected 2 native functions, got %d", len(a.nativeFunctions))
	}
}

func TestSetNativeFunctions_Replace(t *testing.T) {
	a := NewAgentBase()
	a.SetNativeFunctions([]string{"old"})
	a.SetNativeFunctions([]string{"new"})
	if len(a.nativeFunctions) != 1 || a.nativeFunctions[0] != "new" {
		t.Errorf("nativeFunctions = %v", a.nativeFunctions)
	}
}

// ---------------------------------------------------------------------------
// Internal fillers
// ---------------------------------------------------------------------------

func TestSetInternalFillers_ReplaceAll(t *testing.T) {
	a := NewAgentBase()
	a.AddInternalFiller("fn1", "en-US", []string{"wait..."})
	a.SetInternalFillers(map[string]map[string][]string{
		"fn2": {"es-ES": {"espere..."}},
	})
	if _, ok := a.internalFillers["fn1"]; ok {
		t.Error("SetInternalFillers should replace old data")
	}
	if len(a.internalFillers["fn2"]["es-ES"]) != 1 {
		t.Error("expected fn2 fillers")
	}
}

func TestAddInternalFiller_CreatesNested(t *testing.T) {
	a := NewAgentBase()
	a.AddInternalFiller("myFunc", "en-US", []string{"Hold on..."})
	a.AddInternalFiller("myFunc", "fr-FR", []string{"Un moment..."})
	if len(a.internalFillers["myFunc"]) != 2 {
		t.Errorf("expected 2 languages for myFunc, got %d", len(a.internalFillers["myFunc"]))
	}
}

// ---------------------------------------------------------------------------
// Debug events
// ---------------------------------------------------------------------------

func TestEnableDebugEvents_Level(t *testing.T) {
	a := NewAgentBase()
	if a.debugEventsLevel != 0 {
		t.Errorf("default debugEventsLevel = %d, want 0", a.debugEventsLevel)
	}
	a.EnableDebugEvents(2)
	if a.debugEventsLevel != 2 {
		t.Errorf("debugEventsLevel = %d, want 2", a.debugEventsLevel)
	}
}

func TestEnableDebugEvents_RendersInSWML(t *testing.T) {
	a := NewAgentBase()
	a.EnableDebugEvents(1)
	doc := a.RenderSWML(nil, nil)

	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)
	for _, v := range main {
		vm, _ := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			// Python parity (agent_base.py:1232-1246): debug events are wired
			// as params.debug_webhook_url + params.debug_webhook_level. There
			// is no separate ai.debug_events key.
			if aiCfg["debug_events"] != nil {
				t.Errorf("unexpected ai.debug_events key: %v", aiCfg["debug_events"])
			}
			params, ok := aiCfg["params"].(map[string]any)
			if !ok {
				t.Fatal("expected params in AI config")
			}
			level, ok := params["debug_webhook_level"].(int)
			if !ok || level != 1 {
				t.Errorf("params.debug_webhook_level = %v, want 1", params["debug_webhook_level"])
			}
			url, _ := params["debug_webhook_url"].(string)
			if !strings.Contains(url, "/debug_events") {
				t.Errorf("params.debug_webhook_url = %q, want it to contain /debug_events", url)
			}
			return
		}
	}
	t.Fatal("AI verb not found")
}

// ---------------------------------------------------------------------------
// Function includes
// ---------------------------------------------------------------------------

func TestAddFunctionInclude_Basic(t *testing.T) {
	a := NewAgentBase()
	a.AddFunctionInclude("https://example.com/swaig", []string{"fn1"}, nil)
	if len(a.functionIncludes) != 1 {
		t.Fatalf("expected 1 include, got %d", len(a.functionIncludes))
	}
	inc := a.functionIncludes[0]
	if inc["url"] != "https://example.com/swaig" {
		t.Errorf("url = %v", inc["url"])
	}
}

func TestAddFunctionInclude_WithMetaData(t *testing.T) {
	a := NewAgentBase()
	a.AddFunctionInclude("https://example.com", nil, map[string]any{"token": "xyz"})
	inc := a.functionIncludes[0]
	md, _ := inc["meta_data"].(map[string]any)
	if md["token"] != "xyz" {
		t.Errorf("meta_data[token] = %v", md["token"])
	}
}

func TestSetFunctionIncludes_ReplaceAll(t *testing.T) {
	a := NewAgentBase()
	a.AddFunctionInclude("https://old.com", nil, nil)
	a.SetFunctionIncludes([]map[string]any{
		{"url": "https://new.com"},
	})
	if len(a.functionIncludes) != 1 || a.functionIncludes[0]["url"] != "https://new.com" {
		t.Errorf("includes not replaced properly")
	}
}

// ---------------------------------------------------------------------------
// LLM params
// ---------------------------------------------------------------------------

func TestSetPromptLlmParams_Basic(t *testing.T) {
	a := NewAgentBase()
	a.SetPromptLlmParams(map[string]any{"temperature": 0.5})
	if a.promptLlmParams["temperature"] != 0.5 {
		t.Errorf("promptLlmParams[temperature] = %v", a.promptLlmParams["temperature"])
	}
}

func TestSetPostPromptLlmParams_Basic(t *testing.T) {
	a := NewAgentBase()
	a.SetPostPromptLlmParams(map[string]any{"max_tokens": 100})
	if a.postPromptLlmParams["max_tokens"] != 100 {
		t.Errorf("postPromptLlmParams[max_tokens] = %v", a.postPromptLlmParams["max_tokens"])
	}
}

func TestPromptLlmParams_RenderInSWML(t *testing.T) {
	a := NewAgentBase()
	a.SetPromptText("test prompt")
	a.SetPromptLlmParams(map[string]any{"temperature": 0.3})
	doc := a.RenderSWML(nil, nil)

	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)
	for _, v := range main {
		vm, _ := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			prompt, _ := aiCfg["prompt"].(map[string]any)
			if prompt == nil {
				t.Fatal("expected prompt config")
			}
			temp, ok := prompt["temperature"].(float64)
			if !ok {
				t.Fatalf("expected temperature in prompt, got keys: %v", keysOf(prompt))
			}
			if temp != 0.3 {
				t.Errorf("temperature = %v, want 0.3", temp)
			}
			return
		}
	}
	t.Fatal("AI verb not found")
}

// TestSetPromptLlmParams_MergesAcrossCalls is Tier-2 behavioral contract #2:
// set_prompt_llm_params MERGES (Python ai_config_mixin.py:669 does
// self._prompt_llm_params.update(params)). Two calls with distinct keys must
// BOTH survive into the rendered SWML — a replace stub (the old
// `a.promptLlmParams = params`) would drop temperature, keeping only top_p.
func TestSetPromptLlmParams_MergesAcrossCalls(t *testing.T) {
	a := NewAgentBase()
	a.SetPromptText("test prompt")
	a.SetPromptLlmParams(map[string]any{"temperature": 0.5})
	a.SetPromptLlmParams(map[string]any{"top_p": 0.9})

	doc := a.RenderSWML(nil, nil)
	prompt := findAIVerbSubMap(t, doc, "prompt")

	if got, ok := prompt["temperature"].(float64); !ok || got != 0.5 {
		t.Errorf("temperature = %v (ok=%v), want 0.5 — first call was dropped (replace, not merge)", prompt["temperature"], ok)
	}
	if got, ok := prompt["top_p"].(float64); !ok || got != 0.9 {
		t.Errorf("top_p = %v (ok=%v), want 0.9", prompt["top_p"], ok)
	}
}

// TestSetPostPromptLlmParams_MergesAcrossCalls is the post_prompt half of
// contract #2 (Python ai_config_mixin.py:703).
func TestSetPostPromptLlmParams_MergesAcrossCalls(t *testing.T) {
	a := NewAgentBase()
	a.SetPostPrompt("summarize")
	a.SetPostPromptLlmParams(map[string]any{"temperature": 0.2})
	a.SetPostPromptLlmParams(map[string]any{"max_tokens": 150})

	doc := a.RenderSWML(nil, nil)
	pp := findAIVerbSubMap(t, doc, "post_prompt")

	if got, ok := pp["temperature"].(float64); !ok || got != 0.2 {
		t.Errorf("post_prompt temperature = %v (ok=%v), want 0.2 — first call dropped (replace, not merge)", pp["temperature"], ok)
	}
	// max_tokens was passed as an int literal; the render copies params
	// verbatim (no JSON round-trip), so assert on the original int type.
	if got, ok := pp["max_tokens"].(int); !ok || got != 150 {
		t.Errorf("post_prompt max_tokens = %v (ok=%v), want 150", pp["max_tokens"], ok)
	}
}

// findAIVerbSubMap locates the AI verb in a rendered SWML doc and returns the
// named sub-config map (e.g. "prompt" or "post_prompt"), failing the test if
// the verb or sub-map is absent.
func findAIVerbSubMap(t *testing.T, doc map[string]any, key string) map[string]any {
	t.Helper()
	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)
	for _, v := range main {
		vm, _ := v.(map[string]any)
		aiCfg, ok := vm["ai"].(map[string]any)
		if !ok {
			continue
		}
		sub, ok := aiCfg[key].(map[string]any)
		if !ok {
			t.Fatalf("AI verb has no %q sub-config; ai keys: %v", key, keysOf(aiCfg))
		}
		return sub
	}
	t.Fatalf("AI verb not found in rendered SWML")
	return nil
}

func TestPostPromptLlmParams_RenderInSWML(t *testing.T) {
	a := NewAgentBase()
	a.SetPostPrompt("summarize")
	a.SetPostPromptLlmParams(map[string]any{"top_p": 0.8})
	doc := a.RenderSWML(nil, nil)

	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)
	for _, v := range main {
		vm, _ := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			pp, _ := aiCfg["post_prompt"].(map[string]any)
			if pp == nil {
				t.Fatal("expected post_prompt config")
			}
			topP, ok := pp["top_p"].(float64)
			if !ok {
				t.Fatalf("expected top_p in post_prompt, keys: %v", keysOf(pp))
			}
			if topP != 0.8 {
				t.Errorf("top_p = %v, want 0.8", topP)
			}
			return
		}
	}
	t.Fatal("AI verb not found")
}

// ---------------------------------------------------------------------------
// Method chaining
// ---------------------------------------------------------------------------

func TestAIConfigMethods_ReturnSelf(t *testing.T) {
	a := NewAgentBase()
	if a.AddHint("x") != a {
		t.Error("AddHint should return self")
	}
	if a.AddHints(nil) != a {
		t.Error("AddHints should return self")
	}
	if a.AddPatternHint("", "", "") != a { // hint, pattern, replace (all empty)
		t.Error("AddPatternHint should return self")
	}
	if a.AddLanguage(nil) != a {
		t.Error("AddLanguage should return self")
	}
	if a.SetLanguages(nil) != a {
		t.Error("SetLanguages should return self")
	}
	if a.AddPronunciation("", "") != a {
		t.Error("AddPronunciation should return self")
	}
	if a.SetPronunciations(nil) != a {
		t.Error("SetPronunciations should return self")
	}
	if a.SetParam("k", "v") != a {
		t.Error("SetParam should return self")
	}
	if a.SetParams(nil) != a {
		t.Error("SetParams should return self")
	}
	if a.SetGlobalData(nil) != a {
		t.Error("SetGlobalData should return self")
	}
	if a.UpdateGlobalData(nil) != a {
		t.Error("UpdateGlobalData should return self")
	}
	if a.SetNativeFunctions(nil) != a {
		t.Error("SetNativeFunctions should return self")
	}
	if a.SetInternalFillers(map[string]map[string][]string{}) != a {
		t.Error("SetInternalFillers should return self")
	}
	if a.AddInternalFiller("f", "l", nil) != a {
		t.Error("AddInternalFiller should return self")
	}
	if a.EnableDebugEvents(0) != a {
		t.Error("EnableDebugEvents should return self")
	}
	if a.AddFunctionInclude("", nil, nil) != a {
		t.Error("AddFunctionInclude should return self")
	}
	if a.SetFunctionIncludes(nil) != a {
		t.Error("SetFunctionIncludes should return self")
	}
	if a.SetPromptLlmParams(nil) != a {
		t.Error("SetPromptLlmParams should return self")
	}
	if a.SetPostPromptLlmParams(nil) != a {
		t.Error("SetPostPromptLlmParams should return self")
	}
}

// ---------------------------------------------------------------------------
// Contract 8 — AI/LLM structured fillers
// ---------------------------------------------------------------------------

// aiConfigFrom extracts the rendered ai verb config from a SWML document.
func aiConfigFrom(t *testing.T, doc map[string]any) map[string]any {
	t.Helper()
	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)
	for _, v := range main {
		vm, _ := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			return aiCfg
		}
	}
	t.Fatal("no ai verb found in rendered SWML")
	return nil
}

// TestContract8_StructuredFillersAndPatternHint is the contract-8 lock-in:
// (a) add_pattern_hint attaches a STRUCTURED hint (pattern + replacements) that
// survives into the rendered SWML ai.hints array (NOT a bare string, and NOT a
// separate pattern_hints key); (b) add_language carries engine + model +
// fillers into the rendered ai.languages entry.
func TestContract8_StructuredFillersAndPatternHint(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	a.SetPromptText("You are a bot")

	// A structured pattern hint (hint + pattern + replace + ignore_case).
	a.AddPatternHint("SignalWire", "(?i)signal ?wire", "SignalWire", true)

	// A language carrying engine + model + BOTH filler lists.
	a.AddLanguageTyped(
		"English", "en-US", "josh",
		[]string{"um", "uh"},          // speech_fillers
		[]string{"one moment", "hmm"}, // function_fillers
		"elevenlabs",                  // engine
		"eleven_turbo_v2_5",           // model
	)

	aiCfg := aiConfigFrom(t, a.RenderSWML(nil, nil))

	// (a) The structured pattern hint survives into ai.hints.
	if aiCfg["pattern_hints"] != nil {
		t.Errorf("pattern_hints must not be a separate key (Python parity); got %v", aiCfg["pattern_hints"])
	}
	hints, ok := aiCfg["hints"].([]any)
	if !ok {
		t.Fatalf("ai.hints should be a mixed []any array, got %T", aiCfg["hints"])
	}
	var structured map[string]any
	for _, h := range hints {
		if hm, ok := h.(map[string]any); ok && hm["hint"] == "SignalWire" {
			structured = hm
		}
	}
	if structured == nil {
		t.Fatal("structured pattern hint did not survive into ai.hints")
	}
	if structured["pattern"] != "(?i)signal ?wire" {
		t.Errorf("pattern field lost: %v", structured["pattern"])
	}
	if structured["replace"] != "SignalWire" {
		t.Errorf("replace field lost: %v", structured["replace"])
	}
	if structured["ignore_case"] != true {
		t.Errorf("ignore_case field lost: %v", structured["ignore_case"])
	}

	// (b) The language carries engine + model + fillers into ai.languages.
	langs, ok := aiCfg["languages"].([]map[string]any)
	if !ok {
		t.Fatalf("ai.languages should be []map[string]any, got %T", aiCfg["languages"])
	}
	if len(langs) != 1 {
		t.Fatalf("expected 1 language, got %d", len(langs))
	}
	lang := langs[0]
	if lang["engine"] != "elevenlabs" {
		t.Errorf("engine lost: %v", lang["engine"])
	}
	if lang["model"] != "eleven_turbo_v2_5" {
		t.Errorf("model lost: %v", lang["model"])
	}
	sf, _ := lang["speech_fillers"].([]string)
	if len(sf) != 2 || sf[0] != "um" {
		t.Errorf("speech_fillers lost: %v", lang["speech_fillers"])
	}
	ff, _ := lang["function_fillers"].([]string)
	if len(ff) != 2 || ff[0] != "one moment" {
		t.Errorf("function_fillers lost: %v", lang["function_fillers"])
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func keysOf(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
