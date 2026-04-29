package contexts

import (
	"testing"
)

// ---------------------------------------------------------------------------
// ContextBuilder creation and AddContext
// ---------------------------------------------------------------------------

func TestNewContextBuilder(t *testing.T) {
	cb := NewContextBuilder()
	if cb == nil {
		t.Fatal("NewContextBuilder returned nil")
	}
	if len(cb.contexts) != 0 {
		t.Fatalf("expected 0 contexts, got %d", len(cb.contexts))
	}
}

func TestAddContext(t *testing.T) {
	cb := NewContextBuilder()
	ctx := cb.AddContext("default")
	if ctx == nil {
		t.Fatal("AddContext returned nil")
	}
	if ctx.Name() != "default" {
		t.Fatalf("expected name 'default', got %q", ctx.Name())
	}
	if cb.GetContext("default") != ctx {
		t.Fatal("GetContext did not return the same context")
	}
	if cb.GetContext("nonexistent") != nil {
		t.Fatal("GetContext should return nil for unknown context")
	}
}

func TestAddMultipleContexts(t *testing.T) {
	cb := NewContextBuilder()
	cb.AddContext("sales")
	cb.AddContext("support")
	if len(cb.contexts) != 2 {
		t.Fatalf("expected 2 contexts, got %d", len(cb.contexts))
	}
	// Order preserved.
	if cb.contexts[0].name != "sales" || cb.contexts[1].name != "support" {
		t.Fatal("context order not preserved")
	}
}

// ---------------------------------------------------------------------------
// Step creation with sections, criteria, functions
// ---------------------------------------------------------------------------

func TestStepBasic(t *testing.T) {
	cb := NewContextBuilder()
	ctx := cb.AddContext("default")
	step := ctx.AddStep("greeting")
	step.SetText("Hello there!").
		SetStepCriteria("user greeted").
		SetFunctions("none")

	m := step.ToMap()
	if m["name"] != "greeting" {
		t.Fatalf("expected name 'greeting', got %v", m["name"])
	}
	if m["text"] != "Hello there!" {
		t.Fatalf("unexpected text: %v", m["text"])
	}
	if m["step_criteria"] != "user greeted" {
		t.Fatalf("unexpected step_criteria: %v", m["step_criteria"])
	}
	if m["functions"] != "none" {
		t.Fatalf("unexpected functions: %v", m["functions"])
	}
}

func TestStepWithSections(t *testing.T) {
	cb := NewContextBuilder()
	ctx := cb.AddContext("default")
	step := ctx.AddStep("info")
	step.AddSection("Task", "Collect user info").
		AddBullets("Process", []string{"Ask name", "Ask email"}).
		SetStepCriteria("info collected")

	m := step.ToMap()
	text, ok := m["text"].(string)
	if !ok {
		t.Fatal("text should be a string")
	}
	if text == "" {
		t.Fatal("rendered text should not be empty")
	}
	// Verify markdown rendering.
	if !contains(text, "## Task") {
		t.Fatal("expected '## Task' in rendered text")
	}
	if !contains(text, "- Ask name") {
		t.Fatal("expected '- Ask name' in rendered text")
	}
}

func TestStepFunctionsList(t *testing.T) {
	step := &Step{name: "test"}
	step.SetFunctions([]string{"get_weather", "get_time"})
	m := step.ToMap()
	fns, ok := m["functions"].([]string)
	if !ok {
		t.Fatal("functions should be []string")
	}
	if len(fns) != 2 || fns[0] != "get_weather" {
		t.Fatalf("unexpected functions: %v", fns)
	}
}

func TestStepBehaviorFlags(t *testing.T) {
	step := &Step{name: "final"}
	step.SetEnd(true).SetSkipUserTurn(true).SetSkipToNextStep(true)
	m := step.ToMap()
	if m["end"] != true {
		t.Fatal("expected end=true")
	}
	if m["skip_user_turn"] != true {
		t.Fatal("expected skip_user_turn=true")
	}
	if m["skip_to_next_step"] != true {
		t.Fatal("expected skip_to_next_step=true")
	}
}

func TestStepBehaviorFlagsOmittedWhenFalse(t *testing.T) {
	step := &Step{name: "normal"}
	step.SetText("hello")
	m := step.ToMap()
	if _, ok := m["end"]; ok {
		t.Fatal("end should not be present when false")
	}
	if _, ok := m["skip_user_turn"]; ok {
		t.Fatal("skip_user_turn should not be present when false")
	}
}

func TestStepResetObject(t *testing.T) {
	step := &Step{name: "switch"}
	step.SetText("switching").
		SetResetSystemPrompt("new system").
		SetResetUserPrompt("new user").
		SetResetConsolidate(true).
		SetResetFullReset(true)

	m := step.ToMap()
	reset, ok := m["reset"].(map[string]any)
	if !ok {
		t.Fatal("expected reset map")
	}
	if reset["system_prompt"] != "new system" {
		t.Fatalf("unexpected reset system_prompt: %v", reset["system_prompt"])
	}
	if reset["user_prompt"] != "new user" {
		t.Fatalf("unexpected reset user_prompt: %v", reset["user_prompt"])
	}
	if reset["consolidate"] != true {
		t.Fatal("expected consolidate=true")
	}
	if reset["full_reset"] != true {
		t.Fatal("expected full_reset=true")
	}
}

func TestStepResetOmittedWhenEmpty(t *testing.T) {
	step := &Step{name: "normal"}
	step.SetText("hello")
	m := step.ToMap()
	if _, ok := m["reset"]; ok {
		t.Fatal("reset should not be present when no reset fields set")
	}
}

func TestStepClearSections(t *testing.T) {
	step := &Step{name: "test"}
	step.AddSection("Title", "Body")
	step.ClearSections()
	if len(step.sections) != 0 {
		t.Fatal("sections should be empty after ClearSections")
	}
	if step.text != "" {
		t.Fatal("text should be empty after ClearSections")
	}
}

// ---------------------------------------------------------------------------
// Context with multiple steps
// ---------------------------------------------------------------------------

func TestContextMultipleSteps(t *testing.T) {
	ctx := newContext("default")
	ctx.AddStep("greet").SetText("Hello")
	ctx.AddStep("gather").SetText("Tell me more")
	ctx.AddStep("goodbye").SetText("Bye").SetEnd(true)

	if len(ctx.steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(ctx.steps))
	}

	m := ctx.ToMap()
	steps, ok := m["steps"].([]map[string]any)
	if !ok {
		t.Fatal("steps should be a slice of maps")
	}
	if len(steps) != 3 {
		t.Fatalf("expected 3 serialised steps, got %d", len(steps))
	}
	// Order preserved.
	if steps[0]["name"] != "greet" || steps[1]["name"] != "gather" || steps[2]["name"] != "goodbye" {
		t.Fatal("step order not preserved in serialisation")
	}
}

func TestContextGetStep(t *testing.T) {
	ctx := newContext("default")
	s := ctx.AddStep("first")
	if ctx.GetStep("first") != s {
		t.Fatal("GetStep should return the step")
	}
	if ctx.GetStep("nope") != nil {
		t.Fatal("GetStep should return nil for missing step")
	}
}

func TestContextRemoveStep(t *testing.T) {
	ctx := newContext("default")
	ctx.AddStep("a").SetText("A")
	ctx.AddStep("b").SetText("B")
	ctx.AddStep("c").SetText("C")

	ctx.RemoveStep("b")
	if len(ctx.steps) != 2 {
		t.Fatalf("expected 2 steps after remove, got %d", len(ctx.steps))
	}
	if ctx.GetStep("b") != nil {
		t.Fatal("removed step should not be found")
	}
	// Order preserved for remaining.
	if ctx.steps[0].name != "a" || ctx.steps[1].name != "c" {
		t.Fatal("remaining step order wrong after remove")
	}
}

// ---------------------------------------------------------------------------
// GatherInfo with questions
// ---------------------------------------------------------------------------

func TestGatherInfo(t *testing.T) {
	step := &Step{name: "gather"}
	step.SetText("Collecting info")
	step.SetGatherInfo("user_data", "next_step", "Let me ask you some questions").
		AddGatherQuestion("name", "What is your name?").
		AddGatherQuestion("email", "What is your email?", WithType("string"), WithConfirm(true))

	m := step.ToMap()
	giMap, ok := m["gather_info"].(map[string]any)
	if !ok {
		t.Fatal("expected gather_info map")
	}
	if giMap["output_key"] != "user_data" {
		t.Fatalf("unexpected output_key: %v", giMap["output_key"])
	}
	if giMap["completion_action"] != "next_step" {
		t.Fatalf("unexpected completion_action: %v", giMap["completion_action"])
	}
	if giMap["prompt"] != "Let me ask you some questions" {
		t.Fatalf("unexpected prompt: %v", giMap["prompt"])
	}

	questions, ok := giMap["questions"].([]map[string]any)
	if !ok {
		t.Fatal("expected questions slice")
	}
	if len(questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(questions))
	}
	if questions[0]["key"] != "name" {
		t.Fatalf("unexpected first question key: %v", questions[0]["key"])
	}
	if questions[1]["confirm"] != true {
		t.Fatal("expected confirm=true on second question")
	}
}

func TestGatherQuestionOptions(t *testing.T) {
	q := GatherQuestion{Key: "k", Question: "q?", Type: "string"}
	WithType("integer")(&q)
	WithConfirm(true)(&q)
	WithPrompt("extra")(&q)
	WithFunctions([]string{"fn1"})(&q)

	m := q.ToMap()
	if m["type"] != "integer" {
		t.Fatalf("expected type 'integer', got %v", m["type"])
	}
	if m["confirm"] != true {
		t.Fatal("expected confirm=true")
	}
	if m["prompt"] != "extra" {
		t.Fatalf("expected prompt 'extra', got %v", m["prompt"])
	}
	fns, ok := m["functions"].([]string)
	if !ok || len(fns) != 1 || fns[0] != "fn1" {
		t.Fatalf("unexpected functions: %v", m["functions"])
	}
}

func TestGatherQuestionDefaultsOmitted(t *testing.T) {
	q := GatherQuestion{Key: "k", Question: "q?", Type: "string"}
	m := q.ToMap()
	if _, ok := m["type"]; ok {
		t.Fatal("default type 'string' should be omitted")
	}
	if _, ok := m["confirm"]; ok {
		t.Fatal("confirm=false should be omitted")
	}
	if _, ok := m["prompt"]; ok {
		t.Fatal("empty prompt should be omitted")
	}
	if _, ok := m["functions"]; ok {
		t.Fatal("nil functions should be omitted")
	}
}

func TestAddGatherQuestionWithoutSetGatherInfo(t *testing.T) {
	step := &Step{name: "test"}
	step.SetText("test")
	step.AddGatherQuestion("name", "What is your name?")
	if step.gatherInfo == nil {
		t.Fatal("gatherInfo should be auto-initialised")
	}
	if len(step.gatherInfo.Questions) != 1 {
		t.Fatal("expected 1 question")
	}
}

func TestGatherInfoValidate(t *testing.T) {
	// Empty GatherInfo must fail validation — mirrors Python's ValueError.
	gi := &GatherInfo{OutputKey: "data", Prompt: "Tell me"}
	if err := gi.Validate(); err == nil {
		t.Fatal("expected error from Validate() on GatherInfo with no questions")
	}

	// After adding a question, Validate must pass.
	gi.AddQuestion("name", "What is your name?")
	if err := gi.Validate(); err != nil {
		t.Fatalf("unexpected error from Validate() after adding a question: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Serialization (ToMap)
// ---------------------------------------------------------------------------

func TestContextBuilderToMap(t *testing.T) {
	cb := NewContextBuilder()
	ctx := cb.AddContext("default")
	ctx.AddStep("start").SetText("Welcome!")

	m, err := cb.ToMap()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defaultCtx, ok := m["default"].(map[string]any)
	if !ok {
		t.Fatal("expected 'default' context in map")
	}
	steps, ok := defaultCtx["steps"].([]map[string]any)
	if !ok {
		t.Fatal("expected steps slice")
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0]["name"] != "start" {
		t.Fatalf("unexpected step name: %v", steps[0]["name"])
	}
}

func TestContextToMapWithPrompt(t *testing.T) {
	ctx := newContext("default")
	ctx.AddStep("s1").SetText("step text")
	ctx.SetPrompt("context prompt text")
	ctx.SetPostPrompt("post prompt text")
	ctx.SetSystemPrompt("system prompt text")
	ctx.SetUserPrompt("user prompt text")
	ctx.SetConsolidate(true)
	ctx.SetFullReset(true)
	ctx.SetIsolated(true)

	m := ctx.ToMap()
	if m["prompt"] != "context prompt text" {
		t.Fatalf("unexpected prompt: %v", m["prompt"])
	}
	if m["post_prompt"] != "post prompt text" {
		t.Fatalf("unexpected post_prompt: %v", m["post_prompt"])
	}
	if m["system_prompt"] != "system prompt text" {
		t.Fatalf("unexpected system_prompt: %v", m["system_prompt"])
	}
	if m["user_prompt"] != "user prompt text" {
		t.Fatalf("unexpected user_prompt: %v", m["user_prompt"])
	}
	if m["consolidate"] != true {
		t.Fatal("expected consolidate=true")
	}
	if m["full_reset"] != true {
		t.Fatal("expected full_reset=true")
	}
	if m["isolated"] != true {
		t.Fatal("expected isolated=true")
	}
}

func TestContextToMapWithPOMSections(t *testing.T) {
	ctx := newContext("default")
	ctx.AddStep("s1").SetText("hi")
	ctx.AddSection("Intro", "Welcome text")
	ctx.AddBullets("Rules", []string{"Be polite", "Be concise"})

	m := ctx.ToMap()
	pom, ok := m["pom"].([]map[string]any)
	if !ok {
		t.Fatal("expected 'pom' key with sections")
	}
	if len(pom) != 2 {
		t.Fatalf("expected 2 POM sections, got %d", len(pom))
	}
	if _, exists := m["prompt"]; exists {
		t.Fatal("prompt should not be set when POM sections exist")
	}
}

func TestContextToMapWithSystemSections(t *testing.T) {
	ctx := newContext("default")
	ctx.AddStep("s1").SetText("hi")
	ctx.AddSystemSection("Identity", "You are an assistant")
	ctx.AddSystemBullets("Constraints", []string{"No swearing"})

	m := ctx.ToMap()
	sp, ok := m["system_prompt"].(string)
	if !ok {
		t.Fatal("system_prompt should be a rendered string")
	}
	if !contains(sp, "## Identity") || !contains(sp, "- No swearing") {
		t.Fatalf("system prompt rendering unexpected: %s", sp)
	}
}

func TestContextOmitsEmptyFields(t *testing.T) {
	ctx := newContext("default")
	ctx.AddStep("s1").SetText("hi")
	m := ctx.ToMap()
	for _, key := range []string{
		"valid_contexts", "valid_steps", "post_prompt", "system_prompt",
		"consolidate", "full_reset", "user_prompt", "isolated",
		"prompt", "pom", "enter_fillers", "exit_fillers",
	} {
		if _, ok := m[key]; ok {
			t.Fatalf("key %q should not be present when not set", key)
		}
	}
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

func TestValidateEmpty(t *testing.T) {
	cb := NewContextBuilder()
	err := cb.Validate()
	if err == nil {
		t.Fatal("expected error for empty builder")
	}
}

func TestValidateSingleContextNotDefault(t *testing.T) {
	cb := NewContextBuilder()
	ctx := cb.AddContext("main")
	ctx.AddStep("s1").SetText("hi")

	err := cb.Validate()
	if err == nil {
		t.Fatal("expected error when single context is not named 'default'")
	}
}

func TestValidateSingleContextDefault(t *testing.T) {
	cb := NewContextBuilder()
	ctx := cb.AddContext("default")
	ctx.AddStep("s1").SetText("hi")

	err := cb.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMultipleContextsNoDefaultRequired(t *testing.T) {
	cb := NewContextBuilder()
	c1 := cb.AddContext("sales")
	c1.AddStep("greet").SetText("Hello")
	c2 := cb.AddContext("support")
	c2.AddStep("greet").SetText("How can I help?")

	err := cb.Validate()
	if err != nil {
		t.Fatalf("unexpected error for multi-context without 'default': %v", err)
	}
}

func TestValidateContextNoSteps(t *testing.T) {
	cb := NewContextBuilder()
	cb.AddContext("default") // no steps added

	err := cb.Validate()
	if err == nil {
		t.Fatal("expected error for context with no steps")
	}
}

func TestValidateStepNoName(t *testing.T) {
	cb := NewContextBuilder()
	ctx := cb.AddContext("default")
	// Manually add a step with empty name to test validation.
	ctx.steps = append(ctx.steps, &Step{})
	ctx.stepMap[""] = ctx.steps[0]

	err := cb.Validate()
	if err == nil {
		t.Fatal("expected error for step with empty name")
	}
}

// ---------------------------------------------------------------------------
// Navigation rules (valid steps, valid contexts)
// ---------------------------------------------------------------------------

func TestStepValidSteps(t *testing.T) {
	step := &Step{name: "test"}
	step.SetText("hi").SetValidSteps([]string{"next", "goodbye"})
	m := step.ToMap()
	vs, ok := m["valid_steps"].([]string)
	if !ok {
		t.Fatal("expected valid_steps slice")
	}
	if len(vs) != 2 || vs[0] != "next" || vs[1] != "goodbye" {
		t.Fatalf("unexpected valid_steps: %v", vs)
	}
}

func TestStepValidContexts(t *testing.T) {
	step := &Step{name: "test"}
	step.SetText("hi").SetValidContexts([]string{"support", "sales"})
	m := step.ToMap()
	vc, ok := m["valid_contexts"].([]string)
	if !ok {
		t.Fatal("expected valid_contexts slice")
	}
	if len(vc) != 2 || vc[0] != "support" {
		t.Fatalf("unexpected valid_contexts: %v", vc)
	}
}

func TestContextValidStepsAndContexts(t *testing.T) {
	ctx := newContext("default")
	ctx.AddStep("s1").SetText("hi")
	ctx.SetValidSteps([]string{"s1"}).SetValidContexts([]string{"other"})

	m := ctx.ToMap()
	if m["valid_steps"] == nil {
		t.Fatal("expected valid_steps in context map")
	}
	if m["valid_contexts"] == nil {
		t.Fatal("expected valid_contexts in context map")
	}
}

// ---------------------------------------------------------------------------
// Fillers
// ---------------------------------------------------------------------------

func TestEnterExitFillers(t *testing.T) {
	ctx := newContext("default")
	ctx.AddStep("s1").SetText("hi")
	ctx.SetEnterFillers(map[string][]string{
		"en-US":   {"Welcome!", "Hello!"},
		"default": {"Entering..."},
	})
	ctx.SetExitFillers(map[string][]string{
		"en-US": {"Goodbye!"},
	})

	m := ctx.ToMap()
	ef, ok := m["enter_fillers"].(map[string][]string)
	if !ok {
		t.Fatal("expected enter_fillers map")
	}
	if len(ef["en-US"]) != 2 {
		t.Fatalf("expected 2 en-US enter fillers, got %d", len(ef["en-US"]))
	}
	xf, ok := m["exit_fillers"].(map[string][]string)
	if !ok {
		t.Fatal("expected exit_fillers map")
	}
	if len(xf["en-US"]) != 1 {
		t.Fatalf("expected 1 en-US exit filler, got %d", len(xf["en-US"]))
	}
}

func TestAddEnterExitFiller(t *testing.T) {
	ctx := newContext("default")
	ctx.AddStep("s1").SetText("hi")
	ctx.AddEnterFiller("en-US", []string{"Hello"})
	ctx.AddEnterFiller("es", []string{"Hola"})
	ctx.AddExitFiller("en-US", []string{"Goodbye"})

	m := ctx.ToMap()
	ef := m["enter_fillers"].(map[string][]string)
	if len(ef) != 2 {
		t.Fatalf("expected 2 enter filler languages, got %d", len(ef))
	}
	if ef["es"][0] != "Hola" {
		t.Fatalf("unexpected es filler: %v", ef["es"])
	}
	xf := m["exit_fillers"].(map[string][]string)
	if xf["en-US"][0] != "Goodbye" {
		t.Fatalf("unexpected exit filler: %v", xf["en-US"])
	}
}

// ---------------------------------------------------------------------------
// Step ordering (MoveStep)
// ---------------------------------------------------------------------------

func TestMoveStep(t *testing.T) {
	ctx := newContext("default")
	ctx.AddStep("a").SetText("A")
	ctx.AddStep("b").SetText("B")
	ctx.AddStep("c").SetText("C")

	// Move "c" to position 0 (first).
	ctx.MoveStep("c", 0)
	if ctx.steps[0].name != "c" || ctx.steps[1].name != "a" || ctx.steps[2].name != "b" {
		names := []string{ctx.steps[0].name, ctx.steps[1].name, ctx.steps[2].name}
		t.Fatalf("unexpected order after MoveStep: %v", names)
	}
}

func TestMoveStepToEnd(t *testing.T) {
	ctx := newContext("default")
	ctx.AddStep("a").SetText("A")
	ctx.AddStep("b").SetText("B")
	ctx.AddStep("c").SetText("C")

	// Move "a" to position 2 (last).
	ctx.MoveStep("a", 2)
	if ctx.steps[0].name != "b" || ctx.steps[1].name != "c" || ctx.steps[2].name != "a" {
		names := []string{ctx.steps[0].name, ctx.steps[1].name, ctx.steps[2].name}
		t.Fatalf("unexpected order after MoveStep to end: %v", names)
	}
}

func TestMoveStepToMiddle(t *testing.T) {
	ctx := newContext("default")
	ctx.AddStep("a").SetText("A")
	ctx.AddStep("b").SetText("B")
	ctx.AddStep("c").SetText("C")
	ctx.AddStep("d").SetText("D")

	// Move "d" to position 1.
	ctx.MoveStep("d", 1)
	expected := []string{"a", "d", "b", "c"}
	for i, want := range expected {
		if ctx.steps[i].name != want {
			names := make([]string, len(ctx.steps))
			for j, s := range ctx.steps {
				names[j] = s.name
			}
			t.Fatalf("unexpected order: got %v, want %v", names, expected)
		}
	}
}

func TestMoveStepNonExistent(t *testing.T) {
	ctx := newContext("default")
	ctx.AddStep("a").SetText("A")
	// Should not panic.
	ctx.MoveStep("zzz", 0)
	if len(ctx.steps) != 1 || ctx.steps[0].name != "a" {
		t.Fatal("MoveStep on nonexistent step should be a no-op")
	}
}

func TestMoveStepClampPosition(t *testing.T) {
	ctx := newContext("default")
	ctx.AddStep("a").SetText("A")
	ctx.AddStep("b").SetText("B")

	// Move to position beyond the end; should clamp.
	ctx.MoveStep("a", 999)
	if ctx.steps[0].name != "b" || ctx.steps[1].name != "a" {
		t.Fatal("MoveStep should clamp to end when position is out of range")
	}
}

// ---------------------------------------------------------------------------
// CreateSimpleContext helper
// ---------------------------------------------------------------------------

func TestCreateSimpleContext(t *testing.T) {
	ctx := CreateSimpleContext("")
	if ctx.Name() != "default" {
		t.Fatalf("expected 'default', got %q", ctx.Name())
	}
}

func TestCreateSimpleContextWithName(t *testing.T) {
	ctx := CreateSimpleContext("sales")
	if ctx.Name() != "sales" {
		t.Fatalf("expected 'sales', got %q", ctx.Name())
	}
}

// ---------------------------------------------------------------------------
// Full round-trip: build, validate, serialise
// ---------------------------------------------------------------------------

func TestFullRoundTrip(t *testing.T) {
	cb := NewContextBuilder()

	// Build a multi-context configuration.
	sales := cb.AddContext("sales")
	sales.SetPostPrompt("Summarise the sales call")
	sales.AddEnterFiller("en-US", []string{"Welcome to sales!"})

	greet := sales.AddStep("greet")
	greet.SetText("Welcome to sales!").
		SetStepCriteria("customer greeted").
		SetFunctions([]string{"lookup_customer"}).
		SetValidSteps([]string{"next", "qualify"})

	qualify := sales.AddStep("qualify")
	qualify.AddSection("Task", "Qualify the lead").
		AddBullets("Process", []string{"Ask about budget", "Ask about timeline"}).
		SetStepCriteria("lead qualified")

	support := cb.AddContext("support")
	s1 := support.AddStep("triage")
	s1.SetText("What issue are you experiencing?").
		SetValidContexts([]string{"sales"})

	// Validate should pass.
	if err := cb.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	// Serialise.
	m, err := cb.ToMap()
	if err != nil {
		t.Fatalf("unexpected ToMap error: %v", err)
	}

	// Check top-level keys.
	if _, ok := m["sales"]; !ok {
		t.Fatal("expected 'sales' context in output")
	}
	if _, ok := m["support"]; !ok {
		t.Fatal("expected 'support' context in output")
	}

	salesMap := m["sales"].(map[string]any)
	if salesMap["post_prompt"] != "Summarise the sales call" {
		t.Fatal("unexpected post_prompt")
	}
	steps := salesMap["steps"].([]map[string]any)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps in sales, got %d", len(steps))
	}
	if steps[0]["name"] != "greet" {
		t.Fatal("first step should be 'greet'")
	}
}

// ---------------------------------------------------------------------------
// GatherInfo round-trip via ContextBuilder
// ---------------------------------------------------------------------------

func TestGatherInfoInBuilder(t *testing.T) {
	cb := NewContextBuilder()
	ctx := cb.AddContext("default")
	step := ctx.AddStep("collect")
	step.SetText("Collecting info")
	step.SetGatherInfo("answers", "process", "I need to ask a few things").
		AddGatherQuestion("age", "How old are you?", WithType("integer")).
		AddGatherQuestion("city", "Where do you live?")
	// A completion_action of "process" must target a real step for
	// validation to pass.
	ctx.AddStep("process").SetText("Process collected info")

	m, err := cb.ToMap()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defaultCtx := m["default"].(map[string]any)
	steps := defaultCtx["steps"].([]map[string]any)
	giMap := steps[0]["gather_info"].(map[string]any)
	qs := giMap["questions"].([]map[string]any)
	if len(qs) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(qs))
	}
	if qs[0]["type"] != "integer" {
		t.Fatal("first question type should be 'integer'")
	}
	// Second question should omit default type.
	if _, ok := qs[1]["type"]; ok {
		t.Fatal("second question should omit default type 'string'")
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
