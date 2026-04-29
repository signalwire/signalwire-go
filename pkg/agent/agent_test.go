package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/signalwire/signalwire-go/pkg/swaig"

	// Import builtin skills so their init() functions register them
	_ "github.com/signalwire/signalwire-go/pkg/skills/builtin"
)

// ---------------------------------------------------------------------------
// Constructor tests
// ---------------------------------------------------------------------------

func TestNewAgentBase_Defaults(t *testing.T) {
	a := NewAgentBase()
	if a == nil {
		t.Fatal("expected non-nil agent")
	}
	if a.autoAnswer != true {
		t.Error("expected autoAnswer=true")
	}
	if a.usePom != true {
		t.Error("expected usePom=true")
	}
	if a.tokenExpirySecs != 3600 {
		t.Errorf("expected tokenExpirySecs=3600, got %d", a.tokenExpirySecs)
	}
	if a.recordFormat != "mp4" {
		t.Errorf("expected recordFormat=mp4, got %q", a.recordFormat)
	}
	if a.recordStereo != true {
		t.Error("expected recordStereo=true")
	}
	if a.Logger == nil {
		t.Error("expected non-nil logger")
	}
	if a.Service == nil {
		t.Error("expected non-nil swmlService")
	}
	if a.sessionManager == nil {
		t.Error("expected non-nil sessionManager")
	}
}

func TestNewAgentBase_WithOptions(t *testing.T) {
	a := NewAgentBase(
		WithName("TestBot"),
		WithRoute("/bot"),
		WithHost("127.0.0.1"),
		WithPort(8080),
		WithAutoAnswer(false),
		WithRecordCall(true),
		WithRecordFormat("wav"),
		WithRecordStereo(false),
		WithTokenExpiry(7200),
	)

	if a.Name != "TestBot" {
		t.Errorf("expected name=TestBot, got %q", a.Name)
	}
	if a.autoAnswer {
		t.Error("expected autoAnswer=false")
	}
	if !a.recordCall {
		t.Error("expected recordCall=true")
	}
	if a.recordFormat != "wav" {
		t.Errorf("expected recordFormat=wav, got %q", a.recordFormat)
	}
	if a.recordStereo {
		t.Error("expected recordStereo=false")
	}
	if a.tokenExpirySecs != 7200 {
		t.Errorf("expected tokenExpirySecs=7200, got %d", a.tokenExpirySecs)
	}
}

func TestNewAgentBase_WithBasicAuth(t *testing.T) {
	a := NewAgentBase(
		WithBasicAuth("myuser", "mypass"),
	)
	user, pass := a.Service.GetBasicAuthCredentials()
	if user != "myuser" {
		t.Errorf("expected user=myuser, got %q", user)
	}
	if pass != "mypass" {
		t.Errorf("expected pass=mypass, got %q", pass)
	}
}

// ---------------------------------------------------------------------------
// Prompt tests
// ---------------------------------------------------------------------------

func TestSetPromptText_And_GetPrompt(t *testing.T) {
	a := NewAgentBase()
	a.SetPromptText("You are a helpful assistant.")

	prompt := a.GetPrompt()
	text, ok := prompt.(string)
	if !ok {
		t.Fatalf("expected string prompt, got %T", prompt)
	}
	if text != "You are a helpful assistant." {
		t.Errorf("unexpected prompt: %q", text)
	}
	if a.usePom {
		t.Error("expected usePom=false after SetPromptText")
	}
}

func TestPromptPOM_Default(t *testing.T) {
	a := NewAgentBase()
	a.PromptAddSection("Role", "You are a bot.", nil)

	prompt := a.GetPrompt()
	sections, ok := prompt.([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any, got %T", prompt)
	}
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}
	if sections[0]["title"] != "Role" {
		t.Errorf("expected title=Role, got %v", sections[0]["title"])
	}
	if sections[0]["body"] != "You are a bot." {
		t.Errorf("unexpected body: %v", sections[0]["body"])
	}
}

func TestPromptAddSection_WithBullets(t *testing.T) {
	a := NewAgentBase()
	a.PromptAddSection("Rules", "", []string{"Be polite", "Be concise"})

	sections := a.GetPrompt().([]map[string]any)
	bullets, ok := sections[0]["bullets"].([]string)
	if !ok {
		t.Fatal("expected bullets as []string")
	}
	if len(bullets) != 2 || bullets[0] != "Be polite" {
		t.Errorf("unexpected bullets: %v", bullets)
	}
}

func TestPromptHasSection(t *testing.T) {
	a := NewAgentBase()
	if a.PromptHasSection("Nope") {
		t.Error("expected false for non-existent section")
	}
	a.PromptAddSection("Exists", "body", nil)
	if !a.PromptHasSection("Exists") {
		t.Error("expected true for existing section")
	}
}

func TestPromptAddToSection(t *testing.T) {
	a := NewAgentBase()
	a.PromptAddSection("Info", "Line 1", nil)
	a.PromptAddToSection("Info", "Line 2")

	sections := a.GetPrompt().([]map[string]any)
	body := sections[0]["body"].(string)
	if body != "Line 1\nLine 2" {
		t.Errorf("expected appended body, got %q", body)
	}
}

func TestPromptAddSubsection(t *testing.T) {
	a := NewAgentBase()
	a.PromptAddSection("Parent", "parent body", nil)
	a.PromptAddSubsection("Parent", "Child", "child body", nil)

	sections := a.GetPrompt().([]map[string]any)
	subs, ok := sections[0]["subsections"].([]map[string]any)
	if !ok || len(subs) != 1 {
		t.Fatal("expected 1 subsection")
	}
	if subs[0]["title"] != "Child" {
		t.Errorf("expected subsection title=Child, got %v", subs[0]["title"])
	}
}

func TestSetPromptPom(t *testing.T) {
	a := NewAgentBase()
	a.SetPromptText("raw text") // switch to text mode
	pom := []map[string]any{
		{"title": "Section1", "body": "Body1"},
	}
	a.SetPromptPom(pom)

	if !a.usePom {
		t.Error("expected usePom=true after SetPromptPom")
	}
	sections := a.GetPrompt().([]map[string]any)
	if len(sections) != 1 || sections[0]["title"] != "Section1" {
		t.Error("unexpected POM content")
	}
}

func TestSetPostPrompt(t *testing.T) {
	a := NewAgentBase()
	a.SetPostPrompt("Summarize the conversation.")
	if a.postPrompt != "Summarize the conversation." {
		t.Errorf("unexpected postPrompt: %q", a.postPrompt)
	}
}

// ---------------------------------------------------------------------------
// Tool tests
// ---------------------------------------------------------------------------

func TestDefineTool_And_DefineTools(t *testing.T) {
	a := NewAgentBase()
	a.DefineTool(ToolDefinition{
		Name:        "get_weather",
		Description: "Get the weather",
		Parameters: map[string]any{
			"city": map[string]any{"type": "string", "description": "City name"},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("sunny")
		},
	})
	a.DefineTool(ToolDefinition{
		Name:        "get_time",
		Description: "Get the time",
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("12:00")
		},
	})

	tools := a.DefineTools()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	if tools[0].Name != "get_weather" {
		t.Errorf("expected first tool=get_weather, got %q", tools[0].Name)
	}
	if tools[1].Name != "get_time" {
		t.Errorf("expected second tool=get_time, got %q", tools[1].Name)
	}
}

func TestDefineTool_Overwrite(t *testing.T) {
	a := NewAgentBase()
	a.DefineTool(ToolDefinition{Name: "tool1", Description: "v1"})
	a.DefineTool(ToolDefinition{Name: "tool1", Description: "v2"})

	tools := a.DefineTools()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool (overwritten), got %d", len(tools))
	}
	if tools[0].Description != "v2" {
		t.Errorf("expected description=v2, got %q", tools[0].Description)
	}
}

func TestOnFunctionCall_Dispatch(t *testing.T) {
	a := NewAgentBase()
	a.DefineTool(ToolDefinition{
		Name: "echo",
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			msg, _ := args["msg"].(string)
			return swaig.NewFunctionResult("Echo: " + msg)
		},
	})

	result, err := a.OnFunctionCall("echo", map[string]any{"msg": "hello"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["response"] != "Echo: hello" {
		t.Errorf("unexpected response: %v", m["response"])
	}
}

func TestOnFunctionCall_UnknownTool(t *testing.T) {
	a := NewAgentBase()
	_, err := a.OnFunctionCall("nonexistent", nil, nil)
	if err == nil {
		t.Error("expected error for unknown tool")
	}
	if !strings.Contains(err.Error(), "unknown tool") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestOnFunctionCall_NoHandler(t *testing.T) {
	a := NewAgentBase()
	a.DefineTool(ToolDefinition{Name: "datamap_tool"})

	_, err := a.OnFunctionCall("datamap_tool", nil, nil)
	if err == nil {
		t.Error("expected error for tool with no handler")
	}
	if !strings.Contains(err.Error(), "no handler") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRegisterSwaigFunction(t *testing.T) {
	a := NewAgentBase()
	a.RegisterSwaigFunction(map[string]any{
		"function": "remote_tool",
		"purpose":  "Remote tool",
		"data_map": map[string]any{"url": "https://example.com"},
	})

	tools := a.DefineTools()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Name != "remote_tool" {
		t.Errorf("expected tool name=remote_tool, got %q", tools[0].Name)
	}
	if tools[0].SwaigFields == nil {
		t.Error("expected SwaigFields to be set")
	}
}

// ---------------------------------------------------------------------------
// AI Config tests
// ---------------------------------------------------------------------------

func TestAddHints(t *testing.T) {
	a := NewAgentBase()
	a.AddHint("SignalWire").AddHints([]string{"AI", "agents"})
	if len(a.hints) != 3 {
		t.Errorf("expected 3 hints, got %d", len(a.hints))
	}
}

func TestAddPatternHint(t *testing.T) {
	a := NewAgentBase()
	// Python-aligned signature: AddPatternHint(hint, pattern, replace, ignoreCase...)
	a.AddPatternHint("Three-letter code", "[A-Z]{3}", "CODE")
	if len(a.patternHints) != 1 {
		t.Fatal("expected 1 pattern hint")
	}
	if a.patternHints[0]["hint"] != "Three-letter code" {
		t.Error("expected hint='Three-letter code'")
	}
	if a.patternHints[0]["pattern"] != "[A-Z]{3}" {
		t.Error("expected pattern='[A-Z]{3}'")
	}
}

func TestLanguages(t *testing.T) {
	a := NewAgentBase()
	a.AddLanguage(map[string]any{"code": "en", "name": "English"})
	a.AddLanguage(map[string]any{"code": "es", "name": "Spanish"})

	if len(a.languages) != 2 {
		t.Errorf("expected 2 languages, got %d", len(a.languages))
	}

	a.SetLanguages([]map[string]any{{"code": "fr", "name": "French"}})
	if len(a.languages) != 1 || a.languages[0]["code"] != "fr" {
		t.Error("SetLanguages did not replace correctly")
	}
}

func TestPronunciations(t *testing.T) {
	a := NewAgentBase()
	// Python-aligned signature: AddPronunciation(replace, withText, ignoreCase...)
	a.AddPronunciation("SWML", "swimmel")
	if len(a.pronunciations) != 1 {
		t.Fatal("expected 1 pronunciation")
	}
	if a.pronunciations[0]["replace"] != "SWML" {
		t.Error("unexpected pronunciation entry")
	}

	a.SetPronunciations([]map[string]any{{"replace": "API", "with": "A P I"}})
	if len(a.pronunciations) != 1 || a.pronunciations[0]["replace"] != "API" {
		t.Error("SetPronunciations did not replace correctly")
	}
}

func TestSetParam_And_SetParams(t *testing.T) {
	a := NewAgentBase()
	a.SetParam("temperature", 0.7)
	if a.params["temperature"] != 0.7 {
		t.Error("expected temperature=0.7")
	}

	a.SetParams(map[string]any{"top_p": 0.9})
	if _, exists := a.params["temperature"]; exists {
		t.Error("SetParams should replace all params")
	}
	if a.params["top_p"] != 0.9 {
		t.Error("expected top_p=0.9")
	}
}

func TestGlobalData(t *testing.T) {
	a := NewAgentBase()
	a.SetGlobalData(map[string]any{"key1": "val1"})
	if a.globalData["key1"] != "val1" {
		t.Error("expected key1=val1")
	}

	a.UpdateGlobalData(map[string]any{"key2": "val2"})
	if a.globalData["key1"] != "val1" || a.globalData["key2"] != "val2" {
		t.Error("UpdateGlobalData should merge")
	}

	a.SetGlobalData(map[string]any{"key3": "val3"})
	if _, exists := a.globalData["key1"]; exists {
		t.Error("SetGlobalData should replace all")
	}
}

func TestSetNativeFunctions(t *testing.T) {
	a := NewAgentBase()
	a.SetNativeFunctions([]string{"transfer", "hangup"})
	if len(a.nativeFunctions) != 2 {
		t.Errorf("expected 2 native functions, got %d", len(a.nativeFunctions))
	}
}

func TestInternalFillers(t *testing.T) {
	a := NewAgentBase()
	a.AddInternalFiller("get_weather", "en", []string{"Checking the weather..."})
	if a.internalFillers["get_weather"]["en"][0] != "Checking the weather..." {
		t.Error("unexpected filler")
	}

	a.SetInternalFillers(map[string]map[string][]string{
		"get_time": {"en": {"Let me check..."}},
	})
	if _, exists := a.internalFillers["get_weather"]; exists {
		t.Error("SetInternalFillers should replace all")
	}
}

func TestEnableDebugEvents(t *testing.T) {
	a := NewAgentBase()
	a.EnableDebugEvents(2)
	if a.debugEventsLevel != 2 {
		t.Errorf("expected debugEventsLevel=2, got %d", a.debugEventsLevel)
	}
}

func TestFunctionIncludes(t *testing.T) {
	a := NewAgentBase()
	a.AddFunctionInclude("https://remote.com/swaig", []string{"tool1"}, map[string]any{"key": "val"})
	if len(a.functionIncludes) != 1 {
		t.Fatal("expected 1 function include")
	}
	if a.functionIncludes[0]["url"] != "https://remote.com/swaig" {
		t.Error("unexpected include URL")
	}

	a.SetFunctionIncludes([]map[string]any{{"url": "https://other.com"}})
	if len(a.functionIncludes) != 1 || a.functionIncludes[0]["url"] != "https://other.com" {
		t.Error("SetFunctionIncludes did not replace correctly")
	}
}

func TestPromptLlmParams(t *testing.T) {
	a := NewAgentBase()
	a.SetPromptLlmParams(map[string]any{"temperature": 0.5})
	if a.promptLlmParams["temperature"] != 0.5 {
		t.Error("expected prompt LLM params")
	}

	a.SetPostPromptLlmParams(map[string]any{"temperature": 0.3})
	if a.postPromptLlmParams["temperature"] != 0.3 {
		t.Error("expected post-prompt LLM params")
	}
}

// ---------------------------------------------------------------------------
// Verb management tests
// ---------------------------------------------------------------------------

func TestPreAnswerVerbs(t *testing.T) {
	a := NewAgentBase()
	a.AddPreAnswerVerb("play", map[string]any{"url": "https://example.com/intro.mp3"})
	if len(a.preAnswerVerbs) != 1 {
		t.Fatalf("expected 1 pre-answer verb, got %d", len(a.preAnswerVerbs))
	}
	if a.preAnswerVerbs[0].Name != "play" {
		t.Error("expected verb name=play")
	}

	a.ClearPreAnswerVerbs()
	if len(a.preAnswerVerbs) != 0 {
		t.Error("expected 0 pre-answer verbs after clear")
	}
}

func TestPostAnswerVerbs(t *testing.T) {
	a := NewAgentBase()
	a.AddPostAnswerVerb("record_call", map[string]any{"format": "wav"})
	if len(a.postAnswerVerbs) != 1 {
		t.Fatal("expected 1 post-answer verb")
	}

	a.ClearPostAnswerVerbs()
	if len(a.postAnswerVerbs) != 0 {
		t.Error("expected 0 post-answer verbs after clear")
	}
}

func TestPostAiVerbs(t *testing.T) {
	a := NewAgentBase()
	a.AddPostAiVerb("hangup", map[string]any{})
	if len(a.postAiVerbs) != 1 {
		t.Fatal("expected 1 post-ai verb")
	}

	a.ClearPostAiVerbs()
	if len(a.postAiVerbs) != 0 {
		t.Error("expected 0 post-ai verbs after clear")
	}
}

func TestAddAnswerVerb(t *testing.T) {
	a := NewAgentBase()
	a.AddAnswerVerb(map[string]any{"max_duration": 1800})
	if a.answerConfig["max_duration"] != 1800 {
		t.Error("expected max_duration=1800 in answer config")
	}
}

// ---------------------------------------------------------------------------
// Context tests
// ---------------------------------------------------------------------------

func TestDefineContexts(t *testing.T) {
	a := NewAgentBase()
	cb := a.DefineContexts()
	if cb == nil {
		t.Fatal("expected non-nil context builder")
	}
	// Same instance on second call
	cb2 := a.Contexts()
	if cb != cb2 {
		t.Error("expected same context builder on second call")
	}
}

func TestDefineContexts_WithSteps(t *testing.T) {
	a := NewAgentBase()
	ctx := a.DefineContexts().AddContext("default")
	ctx.AddStep("greeting").SetText("Hello! How can I help?")
	ctx.AddStep("goodbye").SetText("Goodbye!").SetEnd(true)

	ctxMap, err := a.contextBuilder.ToMap()
	if err != nil {
		t.Fatalf("context validation failed: %v", err)
	}
	if _, ok := ctxMap["default"]; !ok {
		t.Error("expected 'default' context in map")
	}
}

// ---------------------------------------------------------------------------
// RenderSWML tests
// ---------------------------------------------------------------------------

func TestRenderSWML_BasicStructure(t *testing.T) {
	a := NewAgentBase(
		WithName("TestBot"),
		WithBasicAuth("user", "pass"),
	)
	a.SetPromptText("You are a test bot.")

	doc := a.RenderSWML(nil, nil)

	// Check version
	version, ok := doc["version"].(string)
	if !ok || version != "1.0.0" {
		t.Errorf("expected version=1.0.0, got %v", doc["version"])
	}

	// Check sections exist
	sections, ok := doc["sections"].(map[string]any)
	if !ok {
		t.Fatal("expected sections map")
	}

	// Check main section exists
	main, ok := sections["main"].([]any)
	if !ok {
		t.Fatal("expected main section as slice")
	}

	// Should have answer + ai verb minimum
	if len(main) < 2 {
		t.Fatalf("expected at least 2 verbs in main, got %d", len(main))
	}

	// Check first verb is answer
	firstVerb, ok := main[0].(map[string]any)
	if !ok {
		t.Fatal("expected map for first verb")
	}
	if _, hasAnswer := firstVerb["answer"]; !hasAnswer {
		t.Error("expected first verb to be 'answer'")
	}

	// Check AI verb is present
	foundAI := false
	for _, v := range main {
		vm, ok := v.(map[string]any)
		if !ok {
			continue
		}
		if _, hasAI := vm["ai"]; hasAI {
			foundAI = true
			aiConfig := vm["ai"].(map[string]any)

			// Check prompt
			promptCfg, ok := aiConfig["prompt"].(map[string]any)
			if !ok {
				t.Fatal("expected prompt config as map")
			}
			if promptCfg["text"] != "You are a test bot." {
				t.Errorf("unexpected prompt text: %v", promptCfg["text"])
			}
		}
	}
	if !foundAI {
		t.Error("expected AI verb in document")
	}
}

func TestRenderSWML_WithPOM(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	a.PromptAddSection("Role", "You are a bot.", nil)
	a.PromptAddSection("Rules", "", []string{"Be polite"})

	doc := a.RenderSWML(nil, nil)
	sections := doc["sections"].(map[string]any)
	main := sections["main"].([]any)

	for _, v := range main {
		vm := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			promptCfg, ok := aiCfg["prompt"].(map[string]any)
			if !ok {
				t.Fatal("expected prompt config")
			}
			pom, ok := promptCfg["pom"].([]map[string]any)
			if !ok {
				t.Fatal("expected pom in prompt config")
			}
			if len(pom) != 2 {
				t.Errorf("expected 2 POM sections, got %d", len(pom))
			}
			return
		}
	}
	t.Error("AI verb not found")
}

func TestRenderSWML_AutoAnswerFalse(t *testing.T) {
	a := NewAgentBase(WithAutoAnswer(false), WithBasicAuth("u", "p"))
	a.SetPromptText("Hello")

	doc := a.RenderSWML(nil, nil)
	sections := doc["sections"].(map[string]any)
	main := sections["main"].([]any)

	for _, v := range main {
		vm := v.(map[string]any)
		if _, hasAnswer := vm["answer"]; hasAnswer {
			t.Error("answer verb should not be present when autoAnswer=false")
		}
	}
}

func TestRenderSWML_WithTools(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	a.SetPromptText("Bot")
	a.DefineTool(ToolDefinition{
		Name:        "get_weather",
		Description: "Get weather",
		Parameters: map[string]any{
			"city": map[string]any{"type": "string"},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("sunny")
		},
	})

	doc := a.RenderSWML(nil, nil)
	sections := doc["sections"].(map[string]any)
	main := sections["main"].([]any)

	for _, v := range main {
		vm := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			swaigCfg, ok := aiCfg["SWAIG"].(map[string]any)
			if !ok {
				t.Fatal("expected SWAIG config")
			}
			functions, ok := swaigCfg["functions"].([]map[string]any)
			if !ok {
				t.Fatal("expected functions array")
			}
			if len(functions) != 1 {
				t.Fatalf("expected 1 function, got %d", len(functions))
			}
			fn := functions[0]
			if fn["function"] != "get_weather" {
				t.Errorf("expected function=get_weather, got %v", fn["function"])
			}
			if fn["description"] != "Get weather" {
				t.Errorf("unexpected description: %v", fn["description"])
			}
			webhookURL, _ := fn["web_hook_url"].(string)
			if !strings.Contains(webhookURL, "/swaig") {
				t.Errorf("expected webhook URL to contain /swaig, got %q", webhookURL)
			}
			params, ok := fn["parameters"].(map[string]any)
			if !ok {
				t.Fatal("expected parameters schema")
			}
			if params["type"] != "object" {
				t.Errorf("expected parameters type=object, got %v", params["type"])
			}
			return
		}
	}
	t.Error("AI verb not found")
}

func TestRenderSWML_WithParams(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	a.SetPromptText("Bot").
		SetParam("temperature", 0.5).
		AddHints([]string{"SignalWire", "SWAIG"}).
		SetGlobalData(map[string]any{"company": "SW"})

	doc := a.RenderSWML(nil, nil)
	sections := doc["sections"].(map[string]any)
	main := sections["main"].([]any)

	for _, v := range main {
		vm := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			// Check params
			params, ok := aiCfg["params"].(map[string]any)
			if !ok {
				t.Fatal("expected params")
			}
			if params["temperature"] != 0.5 {
				t.Errorf("expected temperature=0.5, got %v", params["temperature"])
			}

			// Check hints
			hints, ok := aiCfg["hints"].([]string)
			if !ok {
				t.Fatal("expected hints")
			}
			if len(hints) != 2 {
				t.Errorf("expected 2 hints, got %d", len(hints))
			}

			// Check global data
			gd, ok := aiCfg["global_data"].(map[string]any)
			if !ok {
				t.Fatal("expected global_data")
			}
			if gd["company"] != "SW" {
				t.Errorf("unexpected global_data: %v", gd)
			}
			return
		}
	}
	t.Error("AI verb not found")
}

func TestRenderSWML_WithRecordCall(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"), WithRecordCall(true))
	a.SetPromptText("Bot")

	doc := a.RenderSWML(nil, nil)
	sections := doc["sections"].(map[string]any)
	main := sections["main"].([]any)

	foundRecord := false
	for _, v := range main {
		vm := v.(map[string]any)
		if _, has := vm["record_call"]; has {
			foundRecord = true
		}
	}
	if !foundRecord {
		t.Error("expected record_call verb when recordCall=true")
	}
}

func TestRenderSWML_PreAndPostVerbs(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	a.SetPromptText("Bot")
	a.AddPreAnswerVerb("play", map[string]any{"url": "https://example.com/intro.mp3"})
	a.AddPostAnswerVerb("sleep", map[string]any{"duration": 500})
	a.AddPostAiVerb("hangup", map[string]any{})

	doc := a.RenderSWML(nil, nil)
	sections := doc["sections"].(map[string]any)
	main := sections["main"].([]any)

	// First verb should be play (pre-answer)
	first := main[0].(map[string]any)
	if _, hasPlay := first["play"]; !hasPlay {
		t.Error("expected first verb to be play (pre-answer)")
	}

	// Last verb should be hangup (post-ai)
	last := main[len(main)-1].(map[string]any)
	if _, hasHangup := last["hangup"]; !hasHangup {
		t.Error("expected last verb to be hangup (post-ai)")
	}
}

func TestRenderSWML_WithPostPrompt(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	a.SetPromptText("Bot").
		SetPostPrompt("Summarize the call.")

	doc := a.RenderSWML(nil, nil)
	sections := doc["sections"].(map[string]any)
	main := sections["main"].([]any)

	for _, v := range main {
		vm := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			ppCfg, ok := aiCfg["post_prompt"].(map[string]any)
			if !ok {
				t.Fatal("expected post_prompt config")
			}
			if ppCfg["text"] != "Summarize the call." {
				t.Errorf("unexpected post_prompt text: %v", ppCfg["text"])
			}
			ppURL, ok := aiCfg["post_prompt_url"].(string)
			if !ok || ppURL == "" {
				t.Error("expected post_prompt_url to be set")
			}
			return
		}
	}
	t.Error("AI verb not found")
}

func TestRenderSWML_WithNativeFunctions(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	a.SetPromptText("Bot").
		SetNativeFunctions([]string{"transfer", "hangup"})

	doc := a.RenderSWML(nil, nil)
	sections := doc["sections"].(map[string]any)
	main := sections["main"].([]any)

	for _, v := range main {
		vm := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			nf, ok := aiCfg["native_functions"].([]string)
			if !ok {
				t.Fatal("expected native_functions")
			}
			if len(nf) != 2 {
				t.Errorf("expected 2 native functions, got %d", len(nf))
			}
			return
		}
	}
	t.Error("AI verb not found")
}

func TestRenderSWML_WithContexts(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	a.SetPromptText("Bot")
	ctx := a.DefineContexts().AddContext("default")
	ctx.AddStep("greeting").SetText("Hello!")

	doc := a.RenderSWML(nil, nil)
	sections := doc["sections"].(map[string]any)
	main := sections["main"].([]any)

	for _, v := range main {
		vm := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			ctxs, ok := aiCfg["contexts"].(map[string]any)
			if !ok {
				t.Fatal("expected contexts in AI config")
			}
			if _, ok := ctxs["default"]; !ok {
				t.Error("expected 'default' context")
			}
			return
		}
	}
	t.Error("AI verb not found")
}

func TestRenderSWML_WithDebugEvents(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	a.SetPromptText("Bot").EnableDebugEvents(2)

	doc := a.RenderSWML(nil, nil)
	sections := doc["sections"].(map[string]any)
	main := sections["main"].([]any)

	for _, v := range main {
		vm := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			if aiCfg["debug_events"] != 2 {
				t.Errorf("expected debug_events=2, got %v", aiCfg["debug_events"])
			}
			return
		}
	}
	t.Error("AI verb not found")
}

// ---------------------------------------------------------------------------
// Dynamic config callback tests
// ---------------------------------------------------------------------------

func TestDynamicConfigCallback(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	a.SetPromptText("Default prompt")
	a.SetDynamicConfigCallback(func(qp map[string]string, bp map[string]any, headers map[string]string, agent *AgentBase) {
		if name, ok := qp["name"]; ok {
			agent.SetPromptText("Hello, " + name + "!")
		}
	})

	// Simulate an HTTP request with query params
	req := httptest.NewRequest("GET", "/bot?name=Alice", nil)
	req.SetBasicAuth("u", "p")

	body := map[string]any{}
	doc := a.handleDynamicConfig(body, req)

	sections := doc["sections"].(map[string]any)
	main := sections["main"].([]any)

	for _, v := range main {
		vm := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			promptCfg := aiCfg["prompt"].(map[string]any)
			if promptCfg["text"] != "Hello, Alice!" {
				t.Errorf("expected dynamic prompt, got %v", promptCfg["text"])
			}
			return
		}
	}
	t.Error("AI verb not found in dynamic config result")
}

func TestDynamicConfig_DoesNotMutateOriginal(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	a.SetPromptText("Original")
	a.SetDynamicConfigCallback(func(qp map[string]string, bp map[string]any, headers map[string]string, agent *AgentBase) {
		agent.SetPromptText("Modified")
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("u", "p")
	a.handleDynamicConfig(nil, req)

	// Original agent should be unchanged
	prompt := a.GetPrompt().(string)
	if prompt != "Original" {
		t.Errorf("original agent was mutated: %q", prompt)
	}
}

// ---------------------------------------------------------------------------
// SWAIG webhook URL tests
// ---------------------------------------------------------------------------

func TestBuildWebhookURL_Default(t *testing.T) {
	a := NewAgentBase(
		WithBasicAuth("u", "p"),
		WithRoute("/mybot"),
		WithPort(3000),
	)

	url := a.buildWebhookURL()
	if !strings.Contains(url, "u:p@") {
		t.Error("expected basic auth in webhook URL")
	}
	if !strings.HasSuffix(url, "/mybot/swaig") {
		t.Errorf("expected URL to end with /mybot/swaig, got %q", url)
	}
}

func TestBuildWebhookURL_ExplicitOverride(t *testing.T) {
	a := NewAgentBase()
	a.SetWebHookUrl("https://custom.example.com/swaig")

	url := a.buildWebhookURL()
	if url != "https://custom.example.com/swaig" {
		t.Errorf("expected explicit webhook URL, got %q", url)
	}
}

func TestBuildWebhookURL_WithQueryParams(t *testing.T) {
	a := NewAgentBase(
		WithBasicAuth("u", "p"),
		WithRoute("/bot"),
	)
	a.AddSwaigQueryParams(map[string]string{"tenant": "123"})

	url := a.buildWebhookURL()
	if !strings.Contains(url, "tenant=123") {
		t.Errorf("expected query param in URL, got %q", url)
	}
	if !strings.Contains(url, "?") {
		t.Error("expected ? before query params")
	}
}

func TestClearSwaigQueryParams(t *testing.T) {
	a := NewAgentBase()
	a.AddSwaigQueryParams(map[string]string{"key": "val"})
	a.ClearSwaigQueryParams()
	if len(a.swaigQueryParams) != 0 {
		t.Error("expected empty query params after clear")
	}
}

// ---------------------------------------------------------------------------
// Web/HTTP method tests
// ---------------------------------------------------------------------------

func TestSetPostPromptUrl(t *testing.T) {
	a := NewAgentBase()
	a.SetPostPromptUrl("https://example.com/summary")
	if a.postPromptURL != "https://example.com/summary" {
		t.Error("unexpected postPromptURL")
	}
}

func TestManualSetProxyUrl(t *testing.T) {
	a := NewAgentBase()
	a.ManualSetProxyUrl("https://proxy.example.com")
	if a.proxyURLBase != "https://proxy.example.com" {
		t.Error("unexpected proxyURLBase")
	}
}

// ---------------------------------------------------------------------------
// SIP tests
// ---------------------------------------------------------------------------

func TestEnableSipRouting(t *testing.T) {
	a := NewAgentBase()
	a.EnableSipRouting(true, "/sip")
	if !a.sipRoutingEnabled {
		t.Error("expected sipRoutingEnabled=true")
	}
}

func TestRegisterSipUsername(t *testing.T) {
	a := NewAgentBase()
	a.RegisterSipUsername("alice")
	if !a.sipUsernames["alice"] {
		t.Error("expected alice in SIP usernames")
	}
}

// ---------------------------------------------------------------------------
// Lifecycle callback tests
// ---------------------------------------------------------------------------

func TestOnSummary(t *testing.T) {
	a := NewAgentBase()
	called := false
	a.OnSummary(func(summary map[string]any, rawData map[string]any) {
		called = true
	})
	if a.summaryCallback == nil {
		t.Error("expected summaryCallback to be set")
	}
	a.summaryCallback(nil, nil)
	if !called {
		t.Error("summary callback was not called")
	}
}

func TestOnDebugEvent(t *testing.T) {
	a := NewAgentBase()
	called := false
	a.OnDebugEvent(func(event map[string]any) {
		called = true
	})
	if a.debugEventHandler == nil {
		t.Error("expected debugEventHandler to be set")
	}
	a.debugEventHandler(nil)
	if !called {
		t.Error("debug event handler was not called")
	}
}

// ---------------------------------------------------------------------------
// HTTP handler tests
// ---------------------------------------------------------------------------

func TestHTTP_HealthEndpoint(t *testing.T) {
	a := NewAgentBase()
	mux := a.buildMux()

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var body map[string]string
	json.NewDecoder(rr.Body).Decode(&body)
	if body["status"] != "healthy" {
		t.Errorf("unexpected health status: %v", body)
	}
}

func TestHTTP_ReadyEndpoint(t *testing.T) {
	a := NewAgentBase()
	mux := a.buildMux()

	req := httptest.NewRequest("GET", "/ready", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestHTTP_SWMLEndpoint_RequiresAuth(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("testuser", "testpass"))
	mux := a.buildMux()

	// Without auth
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d", rr.Code)
	}

	// With auth
	req = httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("testuser", "testpass")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 with auth, got %d", rr.Code)
	}
}

func TestHTTP_SWMLEndpoint_ReturnsSWML(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	a.SetPromptText("Hello")
	mux := a.buildMux()

	req := httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("u", "p")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	var doc map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&doc); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if doc["version"] != "1.0.0" {
		t.Errorf("expected version=1.0.0, got %v", doc["version"])
	}
}

func TestHTTP_SwaigEndpoint(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	a.DefineTool(ToolDefinition{
		Name: "greet",
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			name, _ := args["name"].(string)
			return swaig.NewFunctionResult("Hi, " + name)
		},
	})
	mux := a.buildMux()

	payload := `{"function":"greet","argument":{"name":"World"}}`
	req := httptest.NewRequest("POST", "/swaig", strings.NewReader(payload))
	req.SetBasicAuth("u", "p")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var result map[string]any
	json.NewDecoder(rr.Body).Decode(&result)
	if result["response"] != "Hi, World" {
		t.Errorf("unexpected response: %v", result)
	}
}

func TestHTTP_SwaigEndpoint_UnknownFunction(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	mux := a.buildMux()

	payload := `{"function":"nonexistent","argument":{}}`
	req := httptest.NewRequest("POST", "/swaig", strings.NewReader(payload))
	req.SetBasicAuth("u", "p")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	// Should return 200 with an error response (not HTTP error)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var result map[string]any
	json.NewDecoder(rr.Body).Decode(&result)
	resp, _ := result["response"].(string)
	if !strings.Contains(resp, "Error") {
		t.Errorf("expected error in response, got %q", resp)
	}
}

func TestHTTP_PostPromptEndpoint(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	called := false
	a.OnSummary(func(summary map[string]any, rawData map[string]any) {
		called = true
	})
	mux := a.buildMux()

	payload := `{"summary":"Call completed"}`
	req := httptest.NewRequest("POST", "/post_prompt", strings.NewReader(payload))
	req.SetBasicAuth("u", "p")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !called {
		t.Error("summary callback was not invoked")
	}
}

// ---------------------------------------------------------------------------
// Chaining tests
// ---------------------------------------------------------------------------

func TestMethodChaining(t *testing.T) {
	// Verify that all setter methods return *AgentBase for chaining
	a := NewAgentBase()
	result := a.
		SetPromptText("test").
		SetPostPrompt("summary").
		AddHint("hint1").
		AddHints([]string{"hint2"}).
		SetParam("temperature", 0.5).
		SetGlobalData(map[string]any{"k": "v"}).
		UpdateGlobalData(map[string]any{"k2": "v2"}).
		SetNativeFunctions([]string{"transfer"}).
		EnableDebugEvents(1).
		AddPreAnswerVerb("play", map[string]any{}).
		AddPostAnswerVerb("sleep", map[string]any{}).
		AddPostAiVerb("hangup", map[string]any{}).
		SetWebHookUrl("https://example.com").
		SetPostPromptUrl("https://example.com/pp").
		ManualSetProxyUrl("https://proxy.example.com")

	if result != a {
		t.Error("chaining should return the same agent")
	}
}

func TestRenderSWML_WithFunctionIncludes(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	a.SetPromptText("Bot")
	a.AddFunctionInclude("https://remote.com/swaig", []string{"tool_a"}, nil)

	doc := a.RenderSWML(nil, nil)
	sections := doc["sections"].(map[string]any)
	main := sections["main"].([]any)

	for _, v := range main {
		vm := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			swaigCfg, ok := aiCfg["SWAIG"].(map[string]any)
			if !ok {
				t.Fatal("expected SWAIG config")
			}
			includes, ok := swaigCfg["includes"].([]map[string]any)
			if !ok {
				t.Fatal("expected includes array")
			}
			if len(includes) != 1 {
				t.Fatalf("expected 1 include, got %d", len(includes))
			}
			if includes[0]["url"] != "https://remote.com/swaig" {
				t.Errorf("unexpected include URL: %v", includes[0]["url"])
			}
			return
		}
	}
	t.Error("AI verb not found")
}

func TestRenderSWML_WithPronunciations(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	a.SetPromptText("Bot")
	a.AddPronunciation("SWML", "swimmel")

	doc := a.RenderSWML(nil, nil)
	sections := doc["sections"].(map[string]any)
	main := sections["main"].([]any)

	for _, v := range main {
		vm := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			pronounce, ok := aiCfg["pronounce"].([]map[string]any)
			if !ok {
				t.Fatal("expected pronounce config")
			}
			if len(pronounce) != 1 || pronounce[0]["replace"] != "SWML" {
				t.Error("unexpected pronunciation data")
			}
			return
		}
	}
	t.Error("AI verb not found")
}

func TestRenderSWML_DataMapTool(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("u", "p"))
	a.SetPromptText("Bot")
	a.RegisterSwaigFunction(map[string]any{
		"function": "lookup",
		"purpose":  "Lookup data",
		"data_map": map[string]any{
			"webhooks": []map[string]any{
				{"url": "https://api.example.com/lookup"},
			},
		},
	})

	doc := a.RenderSWML(nil, nil)
	sections := doc["sections"].(map[string]any)
	main := sections["main"].([]any)

	for _, v := range main {
		vm := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			swaigCfg := aiCfg["SWAIG"].(map[string]any)
			functions := swaigCfg["functions"].([]map[string]any)
			if len(functions) != 1 {
				t.Fatalf("expected 1 function, got %d", len(functions))
			}
			fn := functions[0]
			if fn["function"] != "lookup" {
				t.Errorf("expected function=lookup, got %v", fn["function"])
			}
			// DataMap tools should include their raw fields
			if fn["data_map"] == nil {
				t.Error("expected data_map in DataMap tool")
			}
			return
		}
	}
	t.Error("AI verb not found")
}

func TestRenderSWML_WithRoute(t *testing.T) {
	a := NewAgentBase(
		WithBasicAuth("u", "p"),
		WithRoute("/myagent"),
	)
	a.SetPromptText("Bot")
	a.DefineTool(ToolDefinition{
		Name:        "test_tool",
		Description: "Test",
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("ok")
		},
	})

	doc := a.RenderSWML(nil, nil)
	sections := doc["sections"].(map[string]any)
	main := sections["main"].([]any)

	for _, v := range main {
		vm := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			swaigCfg := aiCfg["SWAIG"].(map[string]any)
			functions := swaigCfg["functions"].([]map[string]any)
			webhookURL := functions[0]["web_hook_url"].(string)
			if !strings.Contains(webhookURL, "/myagent/swaig") {
				t.Errorf("expected webhook URL to contain route /myagent/swaig, got %q", webhookURL)
			}
			return
		}
	}
	t.Error("AI verb not found")
}

// ---------------------------------------------------------------------------
// Skills integration tests
// ---------------------------------------------------------------------------

func TestAddSkill_DateTime(t *testing.T) {
	a := NewAgentBase()
	a.AddSkill("datetime", map[string]any{"timezone": "America/New_York"})

	// Verify the skill is loaded
	if !a.HasSkill("datetime") {
		t.Error("expected datetime skill to be loaded")
	}

	// Verify ListSkills includes datetime
	loaded := a.ListSkills()
	found := false
	for _, name := range loaded {
		if name == "datetime" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'datetime' in loaded skills, got %v", loaded)
	}

	// Verify the tool was registered
	tools := a.DefineTools()
	toolFound := false
	for _, tool := range tools {
		if tool.Name == "get_datetime" {
			toolFound = true
			if tool.Handler == nil {
				t.Error("expected handler to be set for get_datetime tool")
			}
			break
		}
	}
	if !toolFound {
		t.Error("expected get_datetime tool to be registered")
	}

	// Verify hints were added
	if len(a.hints) == 0 {
		t.Error("expected hints to be added from datetime skill")
	}

	// Verify prompt section was added (datetime adds a "Date and Time Information" section)
	if !a.PromptHasSection("Date and Time Information") {
		t.Error("expected 'Date and Time Information' prompt section from datetime skill")
	}
}

func TestAddSkill_SkipPrompt(t *testing.T) {
	a := NewAgentBase()
	a.AddSkill("datetime", map[string]any{"skip_prompt": true})

	if !a.HasSkill("datetime") {
		t.Error("expected datetime skill to be loaded")
	}

	// With skip_prompt, the prompt section should NOT be added
	if a.PromptHasSection("Date and Time Information") {
		t.Error("expected no prompt section when skip_prompt=true")
	}
}

func TestAddSkill_Unknown(t *testing.T) {
	a := NewAgentBase()
	a.AddSkill("nonexistent_skill_xyz", nil)

	// Should not panic, and no skills should be loaded
	if len(a.ListSkills()) != 0 {
		t.Error("expected no skills loaded for unknown skill name")
	}
}

func TestAddSkill_NilParams(t *testing.T) {
	a := NewAgentBase()
	a.AddSkill("datetime", nil)

	// Should work with nil params
	if !a.HasSkill("datetime") {
		t.Error("expected datetime skill to be loaded with nil params")
	}
}

func TestRemoveSkill(t *testing.T) {
	a := NewAgentBase()
	a.AddSkill("datetime", nil)
	if !a.HasSkill("datetime") {
		t.Fatal("expected datetime skill to be loaded")
	}

	a.RemoveSkill("datetime")
	if a.HasSkill("datetime") {
		t.Error("expected datetime skill to be unloaded after RemoveSkill")
	}
}

func TestAddSkill_ToolExecutes(t *testing.T) {
	a := NewAgentBase()
	a.AddSkill("datetime", nil)

	// Call the tool and verify it returns a valid result
	result, err := a.OnFunctionCall("get_datetime", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("unexpected error calling get_datetime: %v", err)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	resp, ok := m["response"].(string)
	if !ok || resp == "" {
		t.Errorf("expected non-empty response from get_datetime, got %v", m["response"])
	}
}

func TestAddSkill_Chaining(t *testing.T) {
	a := NewAgentBase()
	result := a.AddSkill("datetime", nil).
		AddSkill("math", nil)

	if result != a {
		t.Error("AddSkill should return the same agent for chaining")
	}
	if !a.HasSkill("datetime") {
		t.Error("expected datetime skill")
	}
	if !a.HasSkill("math") {
		t.Error("expected math skill")
	}
}
