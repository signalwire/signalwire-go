package builtin

import (
	"os"
	"testing"

	"github.com/signalwire/signalwire-go/pkg/skills"
)

// allSkillNames contains the 18 built-in skill names that must be registered.
var allSkillNames = []string{
	"datetime",
	"math",
	"joke",
	"weather_api",
	"web_search",
	"wikipedia_search",
	"google_maps",
	"spider",
	"datasphere",
	"datasphere_serverless",
	"swml_transfer",
	"play_background_file",
	"api_ninjas_trivia",
	"native_vector_search",
	"info_gatherer",
	"claude_skills",
	"mcp_gateway",
	"custom_skills",
}

// TestRegistryHasAllSkills verifies all 18 skills are registered.
func TestRegistryHasAllSkills(t *testing.T) {
	registered := skills.ListSkills()
	registeredMap := make(map[string]bool)
	for _, name := range registered {
		registeredMap[name] = true
	}

	for _, name := range allSkillNames {
		if !registeredMap[name] {
			t.Errorf("skill %q not found in registry", name)
		}
	}

	if len(registered) < 18 {
		t.Errorf("expected at least 18 registered skills, got %d", len(registered))
	}
}

// TestDateTimeInstantiationAndSetup tests DateTimeSkill.
func TestDateTimeInstantiationAndSetup(t *testing.T) {
	factory := skills.GetSkillFactory("datetime")
	if factory == nil {
		t.Fatal("datetime factory not found")
	}
	s := factory(map[string]any{"timezone": "America/New_York"})
	if s.Name() != "datetime" {
		t.Errorf("expected name 'datetime', got %q", s.Name())
	}
	if !s.Setup() {
		t.Error("datetime Setup() returned false")
	}
	tools := s.RegisterTools()
	if len(tools) == 0 {
		t.Error("datetime RegisterTools() returned empty")
	}
	// Skills now register get_current_time, get_current_date, and get_datetime.
	// Verify all three are present.
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}
	for _, want := range []string{"get_current_time", "get_current_date", "get_datetime"} {
		if !toolNames[want] {
			t.Errorf("expected tool %q to be registered", want)
		}
	}
	if tools[0].Handler == nil {
		t.Error("datetime tool handler is nil")
	}
}

// TestMathInstantiationAndSetup tests MathSkill.
func TestMathInstantiationAndSetup(t *testing.T) {
	factory := skills.GetSkillFactory("math")
	if factory == nil {
		t.Fatal("math factory not found")
	}
	s := factory(nil)
	if !s.Setup() {
		t.Error("math Setup() returned false")
	}
	tools := s.RegisterTools()
	if len(tools) == 0 {
		t.Error("math RegisterTools() returned empty")
	}
	if tools[0].Name != "calculate" {
		t.Errorf("expected tool name 'calculate', got %q", tools[0].Name)
	}
	// Test the math handler
	result := tools[0].Handler(map[string]any{"expression": "2 + 3"}, nil)
	if result == nil {
		t.Error("math handler returned nil")
	}
}

// TestJokeInstantiationAndSetup tests JokeSkill.
func TestJokeInstantiationAndSetup(t *testing.T) {
	factory := skills.GetSkillFactory("joke")
	if factory == nil {
		t.Fatal("joke factory not found")
	}
	s := factory(nil)
	if !s.Setup() {
		t.Error("joke Setup() returned false")
	}
	tools := s.RegisterTools()
	if len(tools) == 0 {
		t.Error("joke RegisterTools() returned empty")
	}
	// Test handler
	result := tools[0].Handler(nil, nil)
	if result == nil {
		t.Error("joke handler returned nil")
	}
}

// TestWeatherAPISetupWithoutKey tests WeatherAPISkill behavior without key param.
func TestWeatherAPISetupWithoutKey(t *testing.T) {
	factory := skills.GetSkillFactory("weather_api")
	if factory == nil {
		t.Fatal("weather_api factory not found")
	}
	s := factory(nil)
	// Setup succeeds if WEATHER_API_KEY env var is set, otherwise fails
	result := s.Setup()
	if os.Getenv("WEATHER_API_KEY") == "" && result {
		t.Error("weather_api Setup() should return false without api_key and env var")
	}
	if os.Getenv("WEATHER_API_KEY") != "" && !result {
		t.Error("weather_api Setup() should return true when env var is set")
	}
}

// TestWeatherAPISetupWithKey tests WeatherAPISkill succeeds with key.
func TestWeatherAPISetupWithKey(t *testing.T) {
	factory := skills.GetSkillFactory("weather_api")
	s := factory(map[string]any{"api_key": "test-key"})
	if !s.Setup() {
		t.Error("weather_api Setup() returned false with api_key")
	}
	tools := s.RegisterTools()
	if len(tools) == 0 {
		t.Error("weather_api RegisterTools() returned empty")
	}
}

// TestWebSearchSetup tests WebSearchSkill.
func TestWebSearchSetup(t *testing.T) {
	factory := skills.GetSkillFactory("web_search")
	if factory == nil {
		t.Fatal("web_search factory not found")
	}
	// Without params: depends on env vars
	s := factory(nil)
	result := s.Setup()
	hasEnv := os.Getenv("GOOGLE_SEARCH_API_KEY") != "" && os.Getenv("GOOGLE_SEARCH_ENGINE_ID") != ""
	if !hasEnv && result {
		t.Error("web_search Setup() should return false without params and env vars")
	}

	// With params should succeed
	s2 := factory(map[string]any{
		"api_key":          "test-key",
		"search_engine_id": "test-engine",
	})
	if !s2.Setup() {
		t.Error("web_search Setup() returned false with params")
	}
	if !s2.SupportsMultipleInstances() {
		t.Error("web_search should support multiple instances")
	}
}

// TestWikipediaSearchSetup tests WikipediaSearchSkill.
func TestWikipediaSearchSetup(t *testing.T) {
	factory := skills.GetSkillFactory("wikipedia_search")
	if factory == nil {
		t.Fatal("wikipedia_search factory not found")
	}
	s := factory(nil)
	if !s.Setup() {
		t.Error("wikipedia_search Setup() returned false")
	}
	tools := s.RegisterTools()
	if len(tools) == 0 {
		t.Error("wikipedia_search RegisterTools() returned empty")
	}
	if tools[0].Name != "search_wikipedia" {
		t.Errorf("expected tool name 'search_wikipedia', got %q", tools[0].Name)
	}
}

// TestGoogleMapsSetup tests GoogleMapsSkill.
func TestGoogleMapsSetup(t *testing.T) {
	factory := skills.GetSkillFactory("google_maps")
	if factory == nil {
		t.Fatal("google_maps factory not found")
	}
	s := factory(nil)
	if s.Setup() {
		t.Error("google_maps Setup() should return false without api_key")
	}

	s2 := factory(map[string]any{"api_key": "test-key"})
	if !s2.Setup() {
		t.Error("google_maps Setup() returned false with api_key")
	}
}

// TestSpiderSetup tests SpiderSkill.
func TestSpiderSetup(t *testing.T) {
	factory := skills.GetSkillFactory("spider")
	if factory == nil {
		t.Fatal("spider factory not found")
	}
	s := factory(nil)
	if !s.Setup() {
		t.Error("spider Setup() returned false")
	}
	if !s.SupportsMultipleInstances() {
		t.Error("spider should support multiple instances")
	}
	tools := s.RegisterTools()
	if len(tools) == 0 {
		t.Error("spider RegisterTools() returned empty")
	}
}

// TestDataSphereSetup tests DataSphereSkill.
func TestDataSphereSetup(t *testing.T) {
	factory := skills.GetSkillFactory("datasphere")
	if factory == nil {
		t.Fatal("datasphere factory not found")
	}
	s := factory(nil)
	if s.Setup() {
		t.Error("datasphere Setup() should return false without params")
	}

	s2 := factory(map[string]any{
		"space_name":  "test",
		"project_id":  "proj123",
		"token":       "tok123",
		"document_id": "doc123",
	})
	if !s2.Setup() {
		t.Error("datasphere Setup() returned false with all params")
	}
}

// TestDataSphereServerlessSetup tests DataSphereServerlessSkill.
func TestDataSphereServerlessSetup(t *testing.T) {
	factory := skills.GetSkillFactory("datasphere_serverless")
	if factory == nil {
		t.Fatal("datasphere_serverless factory not found")
	}
	s := factory(map[string]any{
		"space_name":  "test",
		"project_id":  "proj123",
		"token":       "tok123",
		"document_id": "doc123",
	})
	if !s.Setup() {
		t.Error("datasphere_serverless Setup() returned false with all params")
	}
	tools := s.RegisterTools()
	if len(tools) == 0 {
		t.Error("datasphere_serverless RegisterTools() returned empty")
	}
}

// TestSWMLTransferSetup tests SWMLTransferSkill.
func TestSWMLTransferSetup(t *testing.T) {
	factory := skills.GetSkillFactory("swml_transfer")
	if factory == nil {
		t.Fatal("swml_transfer factory not found")
	}
	s := factory(nil)
	if s.Setup() {
		t.Error("swml_transfer Setup() should return false without transfers")
	}

	s2 := factory(map[string]any{
		"transfers": map[string]any{
			"sales": map[string]any{
				"url":     "https://example.com/sales",
				"message": "Transferring to sales...",
			},
		},
	})
	if !s2.Setup() {
		t.Error("swml_transfer Setup() returned false with transfers")
	}
	if !s2.SupportsMultipleInstances() {
		t.Error("swml_transfer should support multiple instances")
	}
}

// TestPlayBackgroundFileSetup tests PlayBackgroundFileSkill.
func TestPlayBackgroundFileSetup(t *testing.T) {
	factory := skills.GetSkillFactory("play_background_file")
	if factory == nil {
		t.Fatal("play_background_file factory not found")
	}
	s := factory(nil)
	if s.Setup() {
		t.Error("play_background_file Setup() should return false without files")
	}

	s2 := factory(map[string]any{
		"files": []any{
			map[string]any{
				"key":         "test",
				"description": "Test file",
				"url":         "https://example.com/test.mp3",
			},
		},
	})
	if !s2.Setup() {
		t.Error("play_background_file Setup() returned false with files")
	}
	if !s2.SupportsMultipleInstances() {
		t.Error("play_background_file should support multiple instances")
	}
}

// TestAPINinjasTriviaSetup tests APINinjasTriviaSkill.
func TestAPINinjasTriviaSetup(t *testing.T) {
	factory := skills.GetSkillFactory("api_ninjas_trivia")
	if factory == nil {
		t.Fatal("api_ninjas_trivia factory not found")
	}
	s := factory(nil)
	result := s.Setup()
	if os.Getenv("API_NINJAS_KEY") == "" && result {
		t.Error("api_ninjas_trivia Setup() should return false without api_key and env var")
	}

	s2 := factory(map[string]any{"api_key": "test-key"})
	if !s2.Setup() {
		t.Error("api_ninjas_trivia Setup() returned false with api_key")
	}
}

// TestNativeVectorSearchSetup tests NativeVectorSearchSkill.
func TestNativeVectorSearchSetup(t *testing.T) {
	factory := skills.GetSkillFactory("native_vector_search")
	if factory == nil {
		t.Fatal("native_vector_search factory not found")
	}
	// Without remote_url should fail
	s := factory(nil)
	if s.Setup() {
		t.Error("native_vector_search Setup() should return false without remote_url")
	}
}

// TestInfoGathererSetup tests InfoGathererSkill.
func TestInfoGathererSetup(t *testing.T) {
	factory := skills.GetSkillFactory("info_gatherer")
	if factory == nil {
		t.Fatal("info_gatherer factory not found")
	}
	s := factory(nil)
	if s.Setup() {
		t.Error("info_gatherer Setup() should return false without questions")
	}

	s2 := factory(map[string]any{
		"questions": []any{
			map[string]any{
				"key_name":      "name",
				"question_text": "What is your name?",
			},
			map[string]any{
				"key_name":      "age",
				"question_text": "How old are you?",
			},
		},
	})
	if !s2.Setup() {
		t.Error("info_gatherer Setup() returned false with questions")
	}
	tools := s2.RegisterTools()
	if len(tools) != 2 {
		t.Errorf("expected 2 tools from info_gatherer, got %d", len(tools))
	}
	if !s2.SupportsMultipleInstances() {
		t.Error("info_gatherer should support multiple instances")
	}
}

// TestClaudeSkillsSetup tests ClaudeSkillsSkill.
func TestClaudeSkillsSetup(t *testing.T) {
	factory := skills.GetSkillFactory("claude_skills")
	if factory == nil {
		t.Fatal("claude_skills factory not found")
	}

	// Without skills_path: Setup() must return false.
	s := factory(nil)
	if s.Setup() {
		t.Error("claude_skills Setup() should return false without skills_path")
	}

	// With a non-existent path: Setup() must return false.
	s2 := factory(map[string]any{"skills_path": "/nonexistent/path/that/does/not/exist"})
	if s2.Setup() {
		t.Error("claude_skills Setup() should return false with non-existent skills_path")
	}

	// With a valid directory (using os.TempDir): Setup() returns true (0 skills is valid).
	tmpDir := t.TempDir()
	s3 := factory(map[string]any{"skills_path": tmpDir})
	if !s3.Setup() {
		t.Error("claude_skills Setup() returned false with valid (empty) skills_path")
	}
}

// TestMCPGatewaySetup tests MCPGatewaySkill.
func TestMCPGatewaySetup(t *testing.T) {
	factory := skills.GetSkillFactory("mcp_gateway")
	if factory == nil {
		t.Fatal("mcp_gateway factory not found")
	}
	s := factory(nil)
	if s.Setup() {
		t.Error("mcp_gateway Setup() should return false without gateway_url")
	}
}

// TestCustomSkillsSetup tests CustomSkillsSkill.
func TestCustomSkillsSetup(t *testing.T) {
	factory := skills.GetSkillFactory("custom_skills")
	if factory == nil {
		t.Fatal("custom_skills factory not found")
	}
	s := factory(nil)
	if s.Setup() {
		t.Error("custom_skills Setup() should return false without tools")
	}

	s2 := factory(map[string]any{
		"tools": []any{
			map[string]any{
				"name":        "greet",
				"description": "Greet the user",
				"response":    "Hello!",
			},
		},
	})
	if !s2.Setup() {
		t.Error("custom_skills Setup() returned false with tools")
	}
	tools := s2.RegisterTools()
	if len(tools) == 0 {
		t.Error("custom_skills RegisterTools() returned empty")
	}
	if tools[0].Name != "greet" {
		t.Errorf("expected tool name 'greet', got %q", tools[0].Name)
	}
}

// TestSkillManagerLoadUnload tests SkillManager operations.
func TestSkillManagerLoadUnload(t *testing.T) {
	sm := skills.NewSkillManager()

	// Load a simple skill
	factory := skills.GetSkillFactory("joke")
	if factory == nil {
		t.Fatal("joke factory not found")
	}
	s := factory(nil)

	ok, errMsg := sm.LoadSkill(s)
	if !ok {
		t.Fatalf("LoadSkill failed: %s", errMsg)
	}

	// Verify loaded
	if !sm.HasSkill("joke") {
		t.Error("HasSkill returned false after loading")
	}

	loaded := sm.ListLoadedSkills()
	if len(loaded) != 1 {
		t.Errorf("expected 1 loaded skill, got %d", len(loaded))
	}

	// Get skill
	got := sm.GetSkill("joke")
	if got == nil {
		t.Error("GetSkill returned nil")
	}

	// Try loading again (should fail)
	ok, _ = sm.LoadSkill(s)
	if ok {
		t.Error("LoadSkill should fail for already loaded skill")
	}

	// Unload
	if !sm.UnloadSkill("joke") {
		t.Error("UnloadSkill returned false")
	}
	if sm.HasSkill("joke") {
		t.Error("HasSkill returned true after unloading")
	}

	// Unload non-existent
	if sm.UnloadSkill("nonexistent") {
		t.Error("UnloadSkill should return false for non-existent skill")
	}
}

// TestSkillManagerLoadWithEnvVarRequirement tests env var validation.
func TestSkillManagerLoadWithEnvVarRequirement(t *testing.T) {
	sm := skills.NewSkillManager()

	// weather_api without API key env var should fail.
	// If the env var is currently set, temporarily unset it.
	origKey := os.Getenv("WEATHER_API_KEY")
	os.Unsetenv("WEATHER_API_KEY")
	defer func() {
		if origKey != "" {
			os.Setenv("WEATHER_API_KEY", origKey)
		}
	}()

	factory := skills.GetSkillFactory("weather_api")
	s := factory(nil)
	ok, errMsg := sm.LoadSkill(s)
	if ok {
		t.Error("LoadSkill should fail without required env vars")
	}
	if errMsg == "" {
		t.Error("expected error message for missing env vars")
	}
}

// TestBaseSkillDefaults tests BaseSkill default method implementations.
func TestBaseSkillDefaults(t *testing.T) {
	b := &skills.BaseSkill{
		SkillName: "test",
		SkillDesc: "Test skill",
	}

	if b.Name() != "test" {
		t.Errorf("expected name 'test', got %q", b.Name())
	}
	if b.Description() != "Test skill" {
		t.Errorf("expected description 'Test skill', got %q", b.Description())
	}
	if b.Version() != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", b.Version())
	}
	if b.RequiredEnvVars() != nil {
		t.Error("expected nil RequiredEnvVars")
	}
	if b.SupportsMultipleInstances() {
		t.Error("expected SupportsMultipleInstances to be false")
	}
	if b.GetHints() != nil {
		t.Error("expected nil GetHints")
	}
	if b.GetGlobalData() != nil {
		t.Error("expected nil GetGlobalData")
	}
	if b.GetPromptSections() != nil {
		t.Error("expected nil GetPromptSections")
	}
	if b.GetInstanceKey() != "test" {
		t.Errorf("expected instance key 'test', got %q", b.GetInstanceKey())
	}

	schema := b.GetParameterSchema()
	if schema == nil {
		t.Error("expected non-nil GetParameterSchema")
	}
	if _, ok := schema["swaig_fields"]; !ok {
		t.Error("expected swaig_fields in parameter schema")
	}
	if _, ok := schema["skip_prompt"]; !ok {
		t.Error("expected skip_prompt in parameter schema")
	}

	// Cleanup should not panic
	b.Cleanup()
}

// TestBaseSkillParamHelpers tests parameter extraction helpers.
func TestBaseSkillParamHelpers(t *testing.T) {
	b := &skills.BaseSkill{
		SkillName: "test",
		SkillDesc: "Test",
		Params: map[string]any{
			"name":    "alice",
			"count":   42,
			"rate":    3.14,
			"enabled": true,
		},
	}

	if b.GetParamString("name", "default") != "alice" {
		t.Error("GetParamString failed")
	}
	if b.GetParamString("missing", "default") != "default" {
		t.Error("GetParamString default failed")
	}
	if b.GetParamInt("count", 0) != 42 {
		t.Error("GetParamInt failed")
	}
	if b.GetParamInt("missing", 99) != 99 {
		t.Error("GetParamInt default failed")
	}
	if b.GetParamFloat("rate", 0) != 3.14 {
		t.Error("GetParamFloat failed")
	}
	if b.GetParamBool("enabled", false) != true {
		t.Error("GetParamBool failed")
	}
	if b.GetParamBool("missing", false) != false {
		t.Error("GetParamBool default failed")
	}

	// Nil params
	b2 := &skills.BaseSkill{SkillName: "test2", SkillDesc: "Test2"}
	if b2.GetParamString("x", "def") != "def" {
		t.Error("GetParamString with nil params failed")
	}
}

// TestMathHandler tests the math calculation handler.
func TestMathHandler(t *testing.T) {
	factory := skills.GetSkillFactory("math")
	s := factory(nil)
	s.Setup()
	tools := s.RegisterTools()
	handler := tools[0].Handler

	tests := []struct {
		expr   string
		expect bool // whether result should contain "="
	}{
		{"2 + 3", true},
		{"10 / 2", true},
		{"(4 + 6) * 2", true},
		{"", false}, // empty expression
	}

	for _, tc := range tests {
		result := handler(map[string]any{"expression": tc.expr}, nil)
		if result == nil {
			t.Errorf("handler returned nil for expr %q", tc.expr)
			continue
		}
		resultMap := result.ToMap()
		response, _ := resultMap["response"].(string)
		if tc.expect && response == "" {
			t.Errorf("expected non-empty response for expr %q", tc.expr)
		}
	}
}

// TestDateTimeHandler tests the datetime handler.
func TestDateTimeHandler(t *testing.T) {
	factory := skills.GetSkillFactory("datetime")
	s := factory(nil)
	s.Setup()
	tools := s.RegisterTools()
	handler := tools[0].Handler

	result := handler(map[string]any{"timezone": "UTC"}, nil)
	if result == nil {
		t.Error("datetime handler returned nil")
	}

	// Test invalid timezone
	result2 := handler(map[string]any{"timezone": "Invalid/Zone"}, nil)
	if result2 == nil {
		t.Error("datetime handler returned nil for invalid timezone")
	}
}

// TestAllSkillsHaveFactories verifies every registered skill has a working factory.
func TestAllSkillsHaveFactories(t *testing.T) {
	for _, name := range allSkillNames {
		factory := skills.GetSkillFactory(name)
		if factory == nil {
			t.Errorf("no factory registered for skill %q", name)
			continue
		}

		// Each factory should produce a valid SkillBase
		s := factory(nil)
		if s == nil {
			t.Errorf("factory for %q returned nil", name)
			continue
		}
		if s.Name() != name {
			t.Errorf("skill %q has Name() = %q", name, s.Name())
		}
		if s.Description() == "" {
			t.Errorf("skill %q has empty Description()", name)
		}
		if s.Version() == "" {
			t.Errorf("skill %q has empty Version()", name)
		}
	}
}

// TestSkillsWithNoEnvVarsCanSetup tests that skills without env var requirements
// can successfully call Setup().
func TestSkillsWithNoEnvVarsCanSetup(t *testing.T) {
	// Skills that should Setup() successfully without any env vars or params
	simpleSkills := map[string]map[string]any{
		"datetime":         nil,
		"math":             nil,
		"joke":             nil,
		"wikipedia_search": nil,
		"spider":           nil,
	}

	for name, params := range simpleSkills {
		factory := skills.GetSkillFactory(name)
		if factory == nil {
			t.Errorf("factory not found for %q", name)
			continue
		}
		s := factory(params)
		if !s.Setup() {
			t.Errorf("skill %q Setup() returned false (expected true)", name)
		}
		tools := s.RegisterTools()
		if len(tools) == 0 {
			t.Errorf("skill %q RegisterTools() returned empty", name)
		}
		for _, tool := range tools {
			if tool.Handler == nil {
				t.Errorf("skill %q tool %q has nil handler", name, tool.Name)
			}
		}
	}
}
