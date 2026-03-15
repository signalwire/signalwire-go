package agent

import (
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
	a.AddPatternHint("\\d{3}", "digits", "en-US")
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
	if ph["language"] != "en-US" {
		t.Errorf("language = %v", ph["language"])
	}
}

func TestAddPatternHint_WithoutLanguage(t *testing.T) {
	a := NewAgentBase()
	a.AddPatternHint("[A-Z]+", "letters", "")
	ph := a.patternHints[0]
	if _, ok := ph["language"]; ok {
		t.Error("empty language should not be stored")
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

// ---------------------------------------------------------------------------
// Pronunciations
// ---------------------------------------------------------------------------

func TestAddPronunciation_Basic(t *testing.T) {
	a := NewAgentBase()
	a.AddPronunciation("API", "A P I", "en-US")
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
	if p["lang"] != "en-US" {
		t.Errorf("lang = %v", p["lang"])
	}
}

func TestSetPronunciations_ReplaceAll(t *testing.T) {
	a := NewAgentBase()
	a.AddPronunciation("X", "Y", "en")
	a.SetPronunciations([]map[string]any{
		{"replace": "A", "with": "B", "lang": "en"},
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

func TestSetGlobalData_ReplaceAll(t *testing.T) {
	a := NewAgentBase()
	a.UpdateGlobalData(map[string]any{"old": "data"})
	a.SetGlobalData(map[string]any{"new": "data"})
	if _, ok := a.globalData["old"]; ok {
		t.Error("SetGlobalData should replace old data")
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
			level, ok := aiCfg["debug_events"].(int)
			if !ok {
				t.Fatal("expected debug_events in AI config")
			}
			if level != 1 {
				t.Errorf("debug_events = %d, want 1", level)
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
	if a.AddPatternHint("", "", "") != a {
		t.Error("AddPatternHint should return self")
	}
	if a.AddLanguage(nil) != a {
		t.Error("AddLanguage should return self")
	}
	if a.SetLanguages(nil) != a {
		t.Error("SetLanguages should return self")
	}
	if a.AddPronunciation("", "", "") != a {
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
// Helpers
// ---------------------------------------------------------------------------

func keysOf(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
