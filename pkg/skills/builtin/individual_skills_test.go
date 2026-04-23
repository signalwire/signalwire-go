package builtin

import (
	"strings"
	"testing"

	"github.com/signalwire/signalwire-go/pkg/skills"
)

// ---------------------------------------------------------------------------
// Joke skill
// ---------------------------------------------------------------------------

func TestJokeSkill_CustomToolName(t *testing.T) {
	factory := skills.GetSkillFactory("joke")
	s := factory(map[string]any{"tool_name": "tell_funny"})
	s.Setup()
	tools := s.RegisterTools()
	if tools[0].Name != "tell_funny" {
		t.Errorf("tool name = %q, want tell_funny", tools[0].Name)
	}
}

func TestJokeSkill_HandlerReturnsJoke(t *testing.T) {
	factory := skills.GetSkillFactory("joke")
	s := factory(nil)
	s.Setup()
	tools := s.RegisterTools()
	result := tools[0].Handler(nil, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "joke") {
		t.Errorf("expected 'joke' in response, got %q", resp)
	}
}

func TestJokeSkill_HasHints(t *testing.T) {
	factory := skills.GetSkillFactory("joke")
	s := factory(nil)
	s.Setup()
	hints := s.GetHints()
	if len(hints) == 0 {
		t.Error("expected hints from joke skill")
	}
}

func TestJokeSkill_HasPromptSections(t *testing.T) {
	factory := skills.GetSkillFactory("joke")
	s := factory(nil)
	s.Setup()
	sections := s.GetPromptSections()
	if len(sections) == 0 {
		t.Error("expected prompt sections from joke skill")
	}
}

// ---------------------------------------------------------------------------
// Weather API skill
// ---------------------------------------------------------------------------

func TestWeatherAPI_ParameterSchema(t *testing.T) {
	factory := skills.GetSkillFactory("weather_api")
	s := factory(map[string]any{"api_key": "test"})
	s.Setup()
	schema := s.GetParameterSchema()
	if schema["api_key"] == nil {
		t.Error("expected api_key in parameter schema")
	}
	if schema["temperature_unit"] == nil {
		t.Error("expected temperature_unit in parameter schema")
	}
	enumVals, _ := schema["temperature_unit"]["enum"].([]string)
	if len(enumVals) != 2 {
		t.Errorf("expected 2 temperature units, got %d", len(enumVals))
	}
}

func TestWeatherAPI_CustomToolName(t *testing.T) {
	factory := skills.GetSkillFactory("weather_api")
	s := factory(map[string]any{"api_key": "test", "tool_name": "check_weather"})
	s.Setup()
	tools := s.RegisterTools()
	if tools[0].Name != "check_weather" {
		t.Errorf("tool name = %q, want check_weather", tools[0].Name)
	}
}

func TestWeatherAPI_CelsiusUnit(t *testing.T) {
	factory := skills.GetSkillFactory("weather_api")
	s := factory(map[string]any{"api_key": "test", "temperature_unit": "celsius"})
	if !s.Setup() {
		t.Fatal("Setup failed")
	}
}

func TestWeatherAPI_InvalidUnit(t *testing.T) {
	factory := skills.GetSkillFactory("weather_api")
	s := factory(map[string]any{"api_key": "test", "temperature_unit": "kelvin"})
	s.Setup()
	// Should default to fahrenheit
}

func TestWeatherAPI_HasHints(t *testing.T) {
	factory := skills.GetSkillFactory("weather_api")
	s := factory(map[string]any{"api_key": "test"})
	s.Setup()
	hints := s.GetHints()
	if len(hints) == 0 {
		t.Error("expected hints")
	}
}

// ---------------------------------------------------------------------------
// Web search skill
// ---------------------------------------------------------------------------

func TestWebSearch_CustomNumResults(t *testing.T) {
	factory := skills.GetSkillFactory("web_search")
	s := factory(map[string]any{
		"api_key":          "key",
		"search_engine_id": "eng",
		"num_results":      5,
	})
	if !s.Setup() {
		t.Fatal("Setup failed")
	}
}

func TestWebSearch_CustomToolName(t *testing.T) {
	factory := skills.GetSkillFactory("web_search")
	s := factory(map[string]any{
		"api_key":          "key",
		"search_engine_id": "eng",
		"tool_name":        "search_web",
	})
	s.Setup()
	tools := s.RegisterTools()
	if tools[0].Name != "search_web" {
		t.Errorf("tool name = %q, want search_web", tools[0].Name)
	}
}

func TestWebSearch_ParameterSchema(t *testing.T) {
	factory := skills.GetSkillFactory("web_search")
	s := factory(nil)
	schema := s.GetParameterSchema()
	if schema["api_key"] == nil {
		t.Error("expected api_key in schema")
	}
	if schema["search_engine_id"] == nil {
		t.Error("expected search_engine_id in schema")
	}
}

// ---------------------------------------------------------------------------
// Wikipedia skill
// ---------------------------------------------------------------------------

func TestWikipedia_CustomNumResults(t *testing.T) {
	factory := skills.GetSkillFactory("wikipedia_search")
	s := factory(map[string]any{"num_results": 3})
	if !s.Setup() {
		t.Fatal("Setup failed")
	}
}

func TestWikipedia_MinNumResults(t *testing.T) {
	factory := skills.GetSkillFactory("wikipedia_search")
	s := factory(map[string]any{"num_results": -1})
	s.Setup()
	// Should clamp to 1 minimum
}

func TestWikipedia_HasPromptSections(t *testing.T) {
	factory := skills.GetSkillFactory("wikipedia_search")
	s := factory(nil)
	s.Setup()
	sections := s.GetPromptSections()
	if len(sections) == 0 {
		t.Error("expected prompt sections")
	}
}

// ---------------------------------------------------------------------------
// Spider skill
// ---------------------------------------------------------------------------

func TestSpider_CustomToolName(t *testing.T) {
	factory := skills.GetSkillFactory("spider")
	s := factory(map[string]any{"tool_name": "my_spider"})
	s.Setup()
	tools := s.RegisterTools()
	if tools[0].Name != "my_spider_scrape_url" {
		t.Errorf("tool name = %q, want my_spider_scrape_url", tools[0].Name)
	}
}

func TestSpider_DefaultToolName(t *testing.T) {
	factory := skills.GetSkillFactory("spider")
	s := factory(nil)
	s.Setup()
	tools := s.RegisterTools()
	if tools[0].Name != "scrape_url" {
		t.Errorf("tool name = %q, want scrape_url", tools[0].Name)
	}
}

func TestSpider_HasHints(t *testing.T) {
	factory := skills.GetSkillFactory("spider")
	s := factory(nil)
	s.Setup()
	hints := s.GetHints()
	if len(hints) == 0 {
		t.Error("expected hints")
	}
}

func TestSpider_InstanceKey(t *testing.T) {
	factory := skills.GetSkillFactory("spider")
	s := factory(map[string]any{"tool_name": "custom"})
	key := s.GetInstanceKey()
	if key != "spider_custom" {
		t.Errorf("instance key = %q, want spider_custom", key)
	}
}

// ---------------------------------------------------------------------------
// Datasphere skill
// ---------------------------------------------------------------------------

func TestDatasphere_ParameterSchema(t *testing.T) {
	factory := skills.GetSkillFactory("datasphere")
	s := factory(nil)
	schema := s.GetParameterSchema()
	if schema["space_name"] == nil {
		t.Error("expected space_name in schema")
	}
	if schema["document_id"] == nil {
		t.Error("expected document_id in schema")
	}
}

func TestDatasphere_InstanceKey(t *testing.T) {
	factory := skills.GetSkillFactory("datasphere")
	s := factory(map[string]any{"tool_name": "kb_search"})
	key := s.GetInstanceKey()
	if key != "datasphere_kb_search" {
		t.Errorf("instance key = %q", key)
	}
}

// ---------------------------------------------------------------------------
// SWML transfer skill
// ---------------------------------------------------------------------------

func TestSWMLTransfer_TransferExecution(t *testing.T) {
	factory := skills.GetSkillFactory("swml_transfer")
	s := factory(map[string]any{
		"transfers": map[string]any{
			"sales": map[string]any{
				"url":     "https://example.com/sales",
				"message": "Connecting to sales...",
			},
		},
	})
	if !s.Setup() {
		t.Fatal("Setup failed")
	}
	tools := s.RegisterTools()
	handler := tools[0].Handler

	// Valid transfer
	result := handler(map[string]any{"transfer_type": "sales"}, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "sales") {
		t.Errorf("expected transfer message, got %q", resp)
	}

	// Invalid transfer
	result = handler(map[string]any{"transfer_type": "unknown"}, nil)
	m = result.ToMap()
	resp, _ = m["response"].(string)
	if !strings.Contains(resp, "valid") {
		t.Errorf("expected error for invalid transfer, got %q", resp)
	}
}

func TestSWMLTransfer_CustomParamName(t *testing.T) {
	factory := skills.GetSkillFactory("swml_transfer")
	s := factory(map[string]any{
		"parameter_name": "dest",
		"transfers":      map[string]any{"support": map[string]any{"url": "https://example.com"}},
	})
	s.Setup()
	tools := s.RegisterTools()
	params, _ := tools[0].Parameters["properties"].(map[string]any)
	if params["dest"] == nil {
		t.Error("expected custom parameter name 'dest'")
	}
}

func TestSWMLTransfer_HasHints(t *testing.T) {
	factory := skills.GetSkillFactory("swml_transfer")
	s := factory(map[string]any{
		"transfers": map[string]any{"sales": map[string]any{"url": "https://example.com"}},
	})
	s.Setup()
	hints := s.GetHints()
	if len(hints) == 0 {
		t.Error("expected hints")
	}
}

// ---------------------------------------------------------------------------
// Play background file skill
// ---------------------------------------------------------------------------

func TestPlayBgFile_ActionEnum(t *testing.T) {
	factory := skills.GetSkillFactory("play_background_file")
	s := factory(map[string]any{
		"files": []any{
			map[string]any{"key": "music", "description": "Music", "url": "https://example.com/music.mp3"},
			map[string]any{"key": "rain", "description": "Rain", "url": "https://example.com/rain.mp3"},
		},
	})
	s.Setup()
	tools := s.RegisterTools()
	params, _ := tools[0].Parameters["properties"].(map[string]any)
	action, _ := params["action"].(map[string]any)
	enumVals, _ := action["enum"].([]string)
	// Should have start_music, start_rain, stop
	if len(enumVals) != 3 {
		t.Errorf("expected 3 enum values, got %d: %v", len(enumVals), enumVals)
	}
}

func TestPlayBgFile_StopAction(t *testing.T) {
	factory := skills.GetSkillFactory("play_background_file")
	s := factory(map[string]any{
		"files": []any{
			map[string]any{"key": "music", "description": "Music", "url": "https://example.com/music.mp3"},
		},
	})
	s.Setup()
	tools := s.RegisterTools()
	result := tools[0].Handler(map[string]any{"action": "stop"}, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "Stop") {
		t.Errorf("expected stop message, got %q", resp)
	}
}

func TestPlayBgFile_InvalidFile(t *testing.T) {
	factory := skills.GetSkillFactory("play_background_file")
	s := factory(map[string]any{
		"files": []any{
			map[string]any{"key": "music", "description": "Music", "url": "https://example.com/music.mp3"},
		},
	})
	s.Setup()
	tools := s.RegisterTools()
	result := tools[0].Handler(map[string]any{"action": "start_nonexistent"}, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "Unknown") {
		t.Errorf("expected unknown action message, got %q", resp)
	}
}

// ---------------------------------------------------------------------------
// API Ninjas Trivia skill
// ---------------------------------------------------------------------------

func TestTrivia_ParameterSchema(t *testing.T) {
	factory := skills.GetSkillFactory("api_ninjas_trivia")
	s := factory(map[string]any{"api_key": "test"})
	schema := s.GetParameterSchema()
	if schema["api_key"] == nil {
		t.Error("expected api_key in schema")
	}
}

func TestTrivia_HasHints(t *testing.T) {
	factory := skills.GetSkillFactory("api_ninjas_trivia")
	s := factory(map[string]any{"api_key": "test"})
	s.Setup()
	hints := s.GetHints()
	if len(hints) == 0 {
		t.Error("expected hints")
	}
}

func TestTrivia_InstanceKey(t *testing.T) {
	factory := skills.GetSkillFactory("api_ninjas_trivia")
	s := factory(map[string]any{"api_key": "test", "tool_name": "quiz"})
	key := s.GetInstanceKey()
	if key != "api_ninjas_trivia_quiz" {
		t.Errorf("instance key = %q", key)
	}
}

// ---------------------------------------------------------------------------
// Info gatherer skill
// ---------------------------------------------------------------------------

func TestInfoGatherer_WithPrefix(t *testing.T) {
	factory := skills.GetSkillFactory("info_gatherer")
	s := factory(map[string]any{
		"prefix": "billing",
		"questions": []any{
			map[string]any{"key_name": "card", "question_text": "Card number?"},
		},
	})
	if !s.Setup() {
		t.Fatal("Setup failed")
	}
	tools := s.RegisterTools()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	if tools[0].Name != "billing_start_questions" {
		t.Errorf("start tool = %q", tools[0].Name)
	}
	if tools[1].Name != "billing_submit_answer" {
		t.Errorf("submit tool = %q", tools[1].Name)
	}
}

func TestInfoGatherer_StartQuestions(t *testing.T) {
	factory := skills.GetSkillFactory("info_gatherer")
	s := factory(map[string]any{
		"questions": []any{
			map[string]any{"key_name": "name", "question_text": "What is your name?"},
		},
	})
	s.Setup()
	tools := s.RegisterTools()
	result := tools[0].Handler(nil, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "What is your name?") {
		t.Errorf("expected question in response, got %q", resp)
	}
}

func TestInfoGatherer_SubmitAnswerMovesToNext(t *testing.T) {
	factory := skills.GetSkillFactory("info_gatherer")
	s := factory(map[string]any{
		"questions": []any{
			map[string]any{"key_name": "name", "question_text": "What is your name?"},
			map[string]any{"key_name": "age", "question_text": "How old are you?"},
		},
	})
	s.Setup()
	tools := s.RegisterTools()
	submitHandler := tools[1].Handler

	rawData := map[string]any{
		"global_data": map[string]any{
			"skill:info_gatherer": map[string]any{
				"question_index": 0,
				"answers":        []any{},
			},
		},
	}

	result := submitHandler(map[string]any{"answer": "Alice"}, rawData)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "How old are you?") {
		t.Errorf("expected next question, got %q", resp)
	}
}

func TestInfoGatherer_GlobalData(t *testing.T) {
	factory := skills.GetSkillFactory("info_gatherer")
	s := factory(map[string]any{
		"questions": []any{
			map[string]any{"key_name": "name", "question_text": "Name?"},
		},
	})
	s.Setup()
	gd := s.GetGlobalData()
	if gd == nil {
		t.Fatal("expected global data")
	}
}

func TestInfoGatherer_ParameterSchema(t *testing.T) {
	factory := skills.GetSkillFactory("info_gatherer")
	s := factory(nil)
	schema := s.GetParameterSchema()
	if schema["questions"] == nil {
		t.Error("expected questions in schema")
	}
	if schema["prefix"] == nil {
		t.Error("expected prefix in schema")
	}
}

// ---------------------------------------------------------------------------
// Native vector search skill
// ---------------------------------------------------------------------------

func TestVectorSearch_ParameterSchema(t *testing.T) {
	factory := skills.GetSkillFactory("native_vector_search")
	s := factory(nil)
	schema := s.GetParameterSchema()
	if schema["remote_url"] == nil {
		t.Error("expected remote_url in schema")
	}
	if schema["index_name"] == nil {
		t.Error("expected index_name in schema")
	}
}

func TestVectorSearch_InstanceKey(t *testing.T) {
	factory := skills.GetSkillFactory("native_vector_search")
	s := factory(map[string]any{"tool_name": "kb"})
	key := s.GetInstanceKey()
	// Key formula: "native_vector_search_{tool_name}_{index_name}" (3-part, mirrors Python)
	if key != "native_vector_search_kb_default" {
		t.Errorf("instance key = %q, want %q", key, "native_vector_search_kb_default")
	}
}

func TestVectorSearch_InstanceKeyWithIndex(t *testing.T) {
	factory := skills.GetSkillFactory("native_vector_search")
	s := factory(map[string]any{"tool_name": "kb", "index_name": "docs"})
	key := s.GetInstanceKey()
	if key != "native_vector_search_kb_docs" {
		t.Errorf("instance key = %q, want %q", key, "native_vector_search_kb_docs")
	}
}

// ---------------------------------------------------------------------------
// Claude skills
// ---------------------------------------------------------------------------

func TestClaude_ParameterSchema(t *testing.T) {
	factory := skills.GetSkillFactory("claude_skills")
	s := factory(map[string]any{"api_key": "test"})
	schema := s.GetParameterSchema()
	if schema["api_key"] == nil {
		t.Error("expected api_key in schema")
	}
}

func TestClaude_CustomToolName(t *testing.T) {
	factory := skills.GetSkillFactory("claude_skills")
	s := factory(map[string]any{"api_key": "test", "tool_name": "claude_think"})
	s.Setup()
	tools := s.RegisterTools()
	if tools[0].Name != "claude_think" {
		t.Errorf("tool name = %q, want claude_think", tools[0].Name)
	}
}

func TestClaude_HasPromptSections(t *testing.T) {
	factory := skills.GetSkillFactory("claude_skills")
	s := factory(map[string]any{"api_key": "test"})
	s.Setup()
	sections := s.GetPromptSections()
	if len(sections) == 0 {
		t.Error("expected prompt sections")
	}
}

// ---------------------------------------------------------------------------
// MCP Gateway skill
// ---------------------------------------------------------------------------

func TestMCPGateway_ParameterSchema(t *testing.T) {
	factory := skills.GetSkillFactory("mcp_gateway")
	s := factory(nil)
	schema := s.GetParameterSchema()
	if schema["gateway_url"] == nil {
		t.Error("expected gateway_url in schema")
	}
}

// ---------------------------------------------------------------------------
// Custom skills
// ---------------------------------------------------------------------------

func TestCustomSkills_MultipleTools(t *testing.T) {
	factory := skills.GetSkillFactory("custom_skills")
	s := factory(map[string]any{
		"tools": []any{
			map[string]any{"name": "tool1", "description": "First", "response": "Response1"},
			map[string]any{"name": "tool2", "description": "Second", "response": "Response2"},
		},
	})
	if !s.Setup() {
		t.Fatal("Setup failed")
	}
	tools := s.RegisterTools()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	if tools[0].Name != "tool1" {
		t.Errorf("tool[0].Name = %q", tools[0].Name)
	}
	if tools[1].Name != "tool2" {
		t.Errorf("tool[1].Name = %q", tools[1].Name)
	}
}

func TestCustomSkills_ExecuteHandler(t *testing.T) {
	factory := skills.GetSkillFactory("custom_skills")
	s := factory(map[string]any{
		"tools": []any{
			map[string]any{"name": "greet", "response": "Hello there!"},
		},
	})
	s.Setup()
	tools := s.RegisterTools()
	result := tools[0].Handler(nil, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if resp != "Hello there!" {
		t.Errorf("response = %q, want %q", resp, "Hello there!")
	}
}

func TestCustomSkills_DefaultDescription(t *testing.T) {
	factory := skills.GetSkillFactory("custom_skills")
	s := factory(map[string]any{
		"tools": []any{
			map[string]any{"name": "nodesc"},
		},
	})
	s.Setup()
	tools := s.RegisterTools()
	if tools[0].Description == "" {
		t.Error("expected default description")
	}
}

func TestCustomSkills_HasPromptSections(t *testing.T) {
	factory := skills.GetSkillFactory("custom_skills")
	s := factory(map[string]any{
		"tools": []any{
			map[string]any{"name": "test", "description": "Test tool"},
		},
	})
	s.Setup()
	sections := s.GetPromptSections()
	if len(sections) == 0 {
		t.Error("expected prompt sections")
	}
}

func TestCustomSkills_WithDataMap(t *testing.T) {
	factory := skills.GetSkillFactory("custom_skills")
	s := factory(map[string]any{
		"tools": []any{
			map[string]any{
				"name":     "dm_tool",
				"data_map": map[string]any{"expressions": []any{}},
			},
		},
	})
	s.Setup()
	tools := s.RegisterTools()
	if tools[0].SwaigFields == nil {
		t.Error("expected SwaigFields for data_map tool")
	}
}

func TestCustomSkills_SkipsNameless(t *testing.T) {
	factory := skills.GetSkillFactory("custom_skills")
	s := factory(map[string]any{
		"tools": []any{
			map[string]any{"description": "no name"},
			map[string]any{"name": "valid"},
		},
	})
	s.Setup()
	tools := s.RegisterTools()
	if len(tools) != 1 {
		t.Errorf("expected 1 tool (skipping nameless), got %d", len(tools))
	}
}

func TestCustomSkills_ParameterSchema(t *testing.T) {
	factory := skills.GetSkillFactory("custom_skills")
	s := factory(nil)
	schema := s.GetParameterSchema()
	if schema["tools"] == nil {
		t.Error("expected tools in schema")
	}
}

func TestCustomSkills_InstanceKey(t *testing.T) {
	factory := skills.GetSkillFactory("custom_skills")
	s := factory(map[string]any{"tool_name": "my_customs"})
	key := s.GetInstanceKey()
	if key != "custom_skills_my_customs" {
		t.Errorf("instance key = %q", key)
	}
}
