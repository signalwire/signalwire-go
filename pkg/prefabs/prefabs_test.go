package prefabs

import (
	"encoding/json"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// InfoGathererAgent tests
// ---------------------------------------------------------------------------

func TestNewInfoGathererAgent_MinimalOptions(t *testing.T) {
	qs := []Question{{KeyName: "name", QuestionText: "What is your name?"}}
	ig := NewInfoGathererAgent(InfoGathererOptions{
		Questions: &qs,
	})
	if ig == nil {
		t.Fatal("expected non-nil agent")
	}
	if ig.AgentBase == nil {
		t.Fatal("expected non-nil AgentBase")
	}
}

func TestInfoGatherer_HasTools(t *testing.T) {
	qs := []Question{{KeyName: "name", QuestionText: "What is your name?"}}
	ig := NewInfoGathererAgent(InfoGathererOptions{
		Questions: &qs,
	})

	tools := ig.DefineTools()
	if len(tools) < 2 {
		t.Fatalf("expected at least 2 tools, got %d", len(tools))
	}

	names := map[string]bool{}
	for _, td := range tools {
		names[td.Name] = true
	}
	if !names["start_questions"] {
		t.Error("expected start_questions tool")
	}
	if !names["submit_answer"] {
		t.Error("expected submit_answer tool")
	}
}

// TestInfoGatherer_SubmitAnswerAdvancesState is Tier-2 behavioral contract #3:
// submit_answer is a STATE MACHINE, not a "recorded" echo. Starting at
// question_index 0 with two questions, submitting an answer must (a) record the
// answer in global_data.answers keyed by the current question's key_name, (b)
// advance question_index to 1, and (c) present the SECOND question's text. A
// stub that just echoes "Answer recorded" with no set_global_data action would
// fail (a) and (b); one that doesn't advance the index would fail (c).
func TestInfoGatherer_SubmitAnswerAdvancesState(t *testing.T) {
	qs := []Question{
		{KeyName: "name", QuestionText: "What is your name?"},
		{KeyName: "email", QuestionText: "What is your email?"},
	}
	ig := NewInfoGathererAgent(InfoGathererOptions{Questions: &qs})

	// Simulate the request-body shape the platform delivers: global_data with
	// JSON-typed values (question_index is a float64, questions/answers are []any).
	rawData := map[string]any{
		"global_data": map[string]any{
			"questions": []any{
				map[string]any{"key_name": "name", "question_text": "What is your name?", "confirm": false},
				map[string]any{"key_name": "email", "question_text": "What is your email?", "confirm": false},
			},
			"question_index": float64(0),
			"answers":        []any{},
		},
	}

	res := ig.SubmitAnswer(map[string]any{"answer": "Ada Lovelace"}, rawData)
	if res == nil {
		t.Fatal("SubmitAnswer returned nil")
	}

	// The result must carry a set_global_data action recording the new state.
	var gd map[string]any
	for _, action := range res.Actions() {
		if v, ok := action["set_global_data"].(map[string]any); ok {
			gd = v
			break
		}
	}
	if gd == nil {
		t.Fatalf("no set_global_data action — SubmitAnswer is a stateless echo stub; actions=%v", res.Actions())
	}

	// (b) question_index advanced 0 → 1.
	if idx, ok := gd["question_index"].(int); !ok || idx != 1 {
		t.Errorf("question_index = %v (ok=%v), want 1 (state did not advance)", gd["question_index"], ok)
	}

	// (a) answer recorded under the first question's key_name.
	answers, _ := gd["answers"].([]any)
	if len(answers) != 1 {
		t.Fatalf("answers len = %d, want 1 (answer not recorded); answers=%v", len(answers), answers)
	}
	rec, _ := answers[0].(map[string]any)
	if rec["key_name"] != "name" || rec["answer"] != "Ada Lovelace" {
		t.Errorf("recorded answer = %v, want {key_name:name, answer:Ada Lovelace}", rec)
	}

	// (c) the result presents the SECOND question.
	if !strings.Contains(res.Response(), "What is your email?") {
		t.Errorf("response does not present the 2nd question; got %q", res.Response())
	}
}

func TestInfoGatherer_QuestionsInGlobalData(t *testing.T) {
	qs := []Question{
		{KeyName: "name", QuestionText: "What is your name?", Confirm: true},
		{KeyName: "email", QuestionText: "What is your email?"},
	}
	ig := NewInfoGathererAgent(InfoGathererOptions{
		Name:      "test_gatherer",
		Route:     "/gather",
		Questions: &qs,
	})

	// Render SWML and check global data
	doc := ig.RenderSWML(nil, nil)
	aiConfig := findAIConfig(t, doc)

	gd, ok := aiConfig["global_data"].(map[string]any)
	if !ok {
		t.Fatal("expected global_data in AI config")
	}
	questions, ok := gd["questions"].([]map[string]any)
	if !ok {
		// Try []any (JSON marshal/unmarshal uses this)
		questionsAny, ok2 := gd["questions"].([]any)
		if !ok2 {
			t.Fatalf("expected questions list in global_data, got %T", gd["questions"])
		}
		if len(questionsAny) != 2 {
			t.Fatalf("expected 2 questions, got %d", len(questionsAny))
		}
		// Check first question
		q0, ok := questionsAny[0].(map[string]any)
		if !ok {
			t.Fatalf("expected map for question, got %T", questionsAny[0])
		}
		if q0["key_name"] != "name" {
			t.Errorf("expected key_name=name, got %v", q0["key_name"])
		}
		return
	}
	if len(questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(questions))
	}
}

func TestInfoGatherer_StartQuestionsHandler(t *testing.T) {
	qs := []Question{{KeyName: "name", QuestionText: "What is your name?", Confirm: true}}
	ig := NewInfoGathererAgent(InfoGathererOptions{
		Questions: &qs,
	})

	rawData := map[string]any{
		"global_data": map[string]any{
			"questions": []any{
				map[string]any{
					"key_name":      "name",
					"question_text": "What is your name?",
					"confirm":       true,
				},
			},
			"question_index": float64(0),
			"answers":        []any{},
		},
	}

	result, err := ig.OnFunctionCall("start_questions", map[string]any{}, rawData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "What is your name?") {
		t.Errorf("expected response to contain question text, got %q", resp)
	}
	if !strings.Contains(resp, "confirms") {
		// Should mention confirmation
		if !strings.Contains(resp, "confirm") {
			t.Errorf("expected response to mention confirmation for confirm=true, got %q", resp)
		}
	}
}

func TestInfoGatherer_SubmitAnswerHandler(t *testing.T) {
	qs := []Question{
		{KeyName: "name", QuestionText: "What is your name?"},
		{KeyName: "email", QuestionText: "What is your email?"},
	}
	ig := NewInfoGathererAgent(InfoGathererOptions{
		Questions: &qs,
	})

	rawData := map[string]any{
		"global_data": map[string]any{
			"questions": []any{
				map[string]any{"key_name": "name", "question_text": "What is your name?", "confirm": false},
				map[string]any{"key_name": "email", "question_text": "What is your email?", "confirm": false},
			},
			"question_index": float64(0),
			"answers":        []any{},
		},
	}

	// key_name is now derived server-side from global_data; only answer is passed by the model
	result, err := ig.OnFunctionCall("submit_answer", map[string]any{
		"answer": "Alice",
	}, rawData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, _ := result.(map[string]any)
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "What is your email?") {
		t.Errorf("expected next question in response, got %q", resp)
	}
}

// ---------------------------------------------------------------------------
// SurveyAgent tests
// ---------------------------------------------------------------------------

func TestNewSurveyAgent_MinimalOptions(t *testing.T) {
	sa := NewSurveyAgent(SurveyOptions{
		SurveyName: "Test Survey",
		Questions: []SurveyQuestion{
			{ID: "q1", Text: "How satisfied are you?", Type: "rating", Scale: 5},
		},
	})
	if sa == nil {
		t.Fatal("expected non-nil agent")
	}
}

func TestSurvey_HasTools(t *testing.T) {
	sa := NewSurveyAgent(SurveyOptions{
		SurveyName: "Test Survey",
		Questions: []SurveyQuestion{
			{ID: "q1", Text: "How satisfied are you?", Type: "rating", Scale: 5},
		},
	})

	tools := sa.DefineTools()
	names := map[string]bool{}
	for _, td := range tools {
		names[td.Name] = true
	}
	if !names["validate_response"] {
		t.Error("expected validate_response tool")
	}
	if !names["log_response"] {
		t.Error("expected log_response tool")
	}
}

func TestSurvey_QuestionsInGlobalData(t *testing.T) {
	sa := NewSurveyAgent(SurveyOptions{
		SurveyName: "Customer Survey",
		BrandName:  "Acme Corp",
		Questions: []SurveyQuestion{
			{ID: "satisfaction", Text: "How satisfied are you?", Type: "rating", Scale: 5},
			{ID: "recommend", Text: "Would you recommend us?", Type: "yes_no"},
		},
	})

	doc := sa.RenderSWML(nil, nil)
	aiConfig := findAIConfig(t, doc)

	gd, ok := aiConfig["global_data"].(map[string]any)
	if !ok {
		t.Fatal("expected global_data")
	}
	if gd["survey_name"] != "Customer Survey" {
		t.Errorf("expected survey_name=Customer Survey, got %v", gd["survey_name"])
	}
	if gd["brand_name"] != "Acme Corp" {
		t.Errorf("expected brand_name=Acme Corp, got %v", gd["brand_name"])
	}
}

func TestSurvey_ValidateRatingResponse(t *testing.T) {
	sa := NewSurveyAgent(SurveyOptions{
		SurveyName: "Test",
		Questions: []SurveyQuestion{
			{ID: "q1", Text: "Rate us", Type: "rating", Scale: 5},
		},
	})

	// Valid rating
	result, err := sa.OnFunctionCall("validate_response", map[string]any{
		"question_id": "q1",
		"response":    "3",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, _ := result.(map[string]any)
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "valid") {
		t.Errorf("expected 'valid' in response, got %q", resp)
	}

	// Invalid rating
	result, err = sa.OnFunctionCall("validate_response", map[string]any{
		"question_id": "q1",
		"response":    "10",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, _ = result.(map[string]any)
	resp, _ = m["response"].(string)
	if !strings.Contains(resp, "Invalid") {
		t.Errorf("expected 'Invalid' in response for out-of-range rating, got %q", resp)
	}
}

func TestSurvey_ValidateMultipleChoice(t *testing.T) {
	sa := NewSurveyAgent(SurveyOptions{
		SurveyName: "Test",
		Questions: []SurveyQuestion{
			{ID: "q1", Text: "Pick one", Type: "multiple_choice", Choices: []string{"A", "B", "C"}},
		},
	})

	// Valid choice
	result, _ := sa.OnFunctionCall("validate_response", map[string]any{
		"question_id": "q1",
		"response":    "B",
	}, nil)
	m, _ := result.(map[string]any)
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "valid") {
		t.Errorf("expected valid response, got %q", resp)
	}

	// Invalid choice
	result, _ = sa.OnFunctionCall("validate_response", map[string]any{
		"question_id": "q1",
		"response":    "D",
	}, nil)
	m, _ = result.(map[string]any)
	resp, _ = m["response"].(string)
	if !strings.Contains(resp, "Invalid") {
		t.Errorf("expected Invalid response for bad choice, got %q", resp)
	}
}

// ---------------------------------------------------------------------------
// ReceptionistAgent tests
// ---------------------------------------------------------------------------

func TestNewReceptionistAgent_MinimalOptions(t *testing.T) {
	ra := NewReceptionistAgent(ReceptionistOptions{
		Departments: []Department{
			{Name: "sales", Description: "Sales inquiries", Number: "+15551234567"},
		},
	})
	if ra == nil {
		t.Fatal("expected non-nil agent")
	}
}

func TestReceptionist_HasTools(t *testing.T) {
	ra := NewReceptionistAgent(ReceptionistOptions{
		Departments: []Department{
			{Name: "sales", Description: "Sales inquiries", Number: "+15551234567"},
		},
	})

	tools := ra.DefineTools()
	names := map[string]bool{}
	for _, td := range tools {
		names[td.Name] = true
	}
	if !names["collect_caller_info"] {
		t.Error("expected collect_caller_info tool")
	}
	if !names["transfer_call"] {
		t.Error("expected transfer_call tool")
	}
}

func TestReceptionist_DepartmentsInGlobalData(t *testing.T) {
	ra := NewReceptionistAgent(ReceptionistOptions{
		Departments: []Department{
			{Name: "sales", Description: "Sales inquiries", Number: "+15551234567"},
			{Name: "support", Description: "Technical support", Number: "+15559876543"},
		},
	})

	doc := ra.RenderSWML(nil, nil)
	aiConfig := findAIConfig(t, doc)

	gd, ok := aiConfig["global_data"].(map[string]any)
	if !ok {
		t.Fatal("expected global_data")
	}

	depts, ok := gd["departments"].([]map[string]any)
	if !ok {
		deptsAny, ok2 := gd["departments"].([]any)
		if !ok2 {
			t.Fatalf("expected departments list, got %T", gd["departments"])
		}
		if len(deptsAny) != 2 {
			t.Fatalf("expected 2 departments, got %d", len(deptsAny))
		}
		return
	}
	if len(depts) != 2 {
		t.Fatalf("expected 2 departments, got %d", len(depts))
	}
}

func TestReceptionist_CollectCallerInfo(t *testing.T) {
	ra := NewReceptionistAgent(ReceptionistOptions{
		Departments: []Department{
			{Name: "sales", Description: "Sales", Number: "+15551234567"},
		},
	})

	result, err := ra.OnFunctionCall("collect_caller_info", map[string]any{
		"name":   "Bob",
		"reason": "pricing question",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, _ := result.(map[string]any)
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "Bob") {
		t.Errorf("expected response to mention caller name, got %q", resp)
	}
}

func TestReceptionist_TransferCall_Connect(t *testing.T) {
	ra := NewReceptionistAgent(ReceptionistOptions{
		Departments: []Department{
			{Name: "sales", Description: "Sales", Number: "+15551234567", TransferSWML: false},
		},
	})

	rawData := map[string]any{
		"global_data": map[string]any{
			"caller_info": map[string]any{"name": "Alice"},
		},
	}

	result, err := ra.OnFunctionCall("transfer_call", map[string]any{
		"department": "sales",
	}, rawData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, _ := result.(map[string]any)
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "sales") {
		t.Errorf("expected response to mention department, got %q", resp)
	}
	if !strings.Contains(resp, "Alice") {
		t.Errorf("expected response to mention caller name, got %q", resp)
	}
	// Should have actions (connect)
	if m["action"] == nil {
		t.Error("expected transfer actions in result")
	} else {
		actions, ok := m["action"].([]map[string]any)
		if !ok {
			t.Errorf("expected action as []map[string]any, got %T", m["action"])
		} else if len(actions) == 0 {
			t.Error("expected at least one transfer action")
		}
	}
}

func TestReceptionist_TransferCall_SwmlTransfer(t *testing.T) {
	ra := NewReceptionistAgent(ReceptionistOptions{
		Departments: []Department{
			{Name: "support", Description: "Support", Number: "swml://support-agent", TransferSWML: true},
		},
	})

	rawData := map[string]any{
		"global_data": map[string]any{
			"caller_info": map[string]any{"name": "Charlie"},
		},
	}

	result, err := ra.OnFunctionCall("transfer_call", map[string]any{
		"department": "support",
	}, rawData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, _ := result.(map[string]any)
	if m["action"] == nil {
		t.Error("expected SWML transfer actions in result")
	} else {
		actions, ok := m["action"].([]map[string]any)
		if !ok {
			t.Errorf("expected action as []map[string]any, got %T", m["action"])
		} else if len(actions) == 0 {
			t.Error("expected at least one SWML transfer action")
		}
	}
}

func TestReceptionist_TransferCall_UnknownDept(t *testing.T) {
	ra := NewReceptionistAgent(ReceptionistOptions{
		Departments: []Department{
			{Name: "sales", Description: "Sales", Number: "+15551234567"},
		},
	})

	result, err := ra.OnFunctionCall("transfer_call", map[string]any{
		"department": "unknown_dept",
	}, map[string]any{"global_data": map[string]any{"caller_info": map[string]any{}}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, _ := result.(map[string]any)
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "couldn't find") {
		t.Errorf("expected error message for unknown dept, got %q", resp)
	}
}

// ---------------------------------------------------------------------------
// FAQBotAgent tests
// ---------------------------------------------------------------------------

func TestNewFAQBotAgent_MinimalOptions(t *testing.T) {
	fb := NewFAQBotAgent(FAQBotOptions{
		FAQs: []FAQ{
			{Question: "What is Go?", Answer: "A programming language."},
		},
	})
	if fb == nil {
		t.Fatal("expected non-nil agent")
	}
}

func TestFAQBot_HasTools(t *testing.T) {
	fb := NewFAQBotAgent(FAQBotOptions{
		FAQs: []FAQ{
			{Question: "What is Go?", Answer: "A programming language."},
		},
	})

	tools := fb.DefineTools()
	names := map[string]bool{}
	for _, td := range tools {
		names[td.Name] = true
	}
	if !names["search_faqs"] {
		t.Error("expected search_faqs tool")
	}
}

func TestFAQBot_SearchMatch(t *testing.T) {
	fb := NewFAQBotAgent(FAQBotOptions{
		FAQs: []FAQ{
			{Question: "What is SignalWire?", Answer: "A cloud communications platform.", Categories: []string{"general"}},
			{Question: "How much does it cost?", Answer: "Pay-as-you-go pricing.", Categories: []string{"pricing"}},
			{Question: "What languages are supported?", Answer: "Many languages.", Categories: []string{"technical"}},
		},
	})

	result, err := fb.OnFunctionCall("search_faqs", map[string]any{
		"query": "signalwire",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, _ := result.(map[string]any)
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "SignalWire") {
		t.Errorf("expected matching FAQ in response, got %q", resp)
	}
}

func TestFAQBot_SearchNoMatch(t *testing.T) {
	fb := NewFAQBotAgent(FAQBotOptions{
		FAQs: []FAQ{
			{Question: "What is Go?", Answer: "A programming language."},
		},
	})

	result, err := fb.OnFunctionCall("search_faqs", map[string]any{
		"query": "quantum computing",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, _ := result.(map[string]any)
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "No matching") {
		t.Errorf("expected no-match message, got %q", resp)
	}
}

func TestFAQBot_PromptHasFAQSection(t *testing.T) {
	fb := NewFAQBotAgent(FAQBotOptions{
		FAQs: []FAQ{
			{Question: "What is Go?", Answer: "A programming language."},
		},
	})

	if !fb.PromptHasSection("FAQ Database") {
		t.Error("expected FAQ Database section in prompt")
	}
}

func TestFAQBot_SuggestRelated(t *testing.T) {
	boolTrue := true
	fb := NewFAQBotAgent(FAQBotOptions{
		FAQs: []FAQ{
			{Question: "What is Go?", Answer: "A programming language."},
		},
		SuggestRelated: &boolTrue,
	})

	if !fb.PromptHasSection("Related Questions") {
		t.Error("expected Related Questions section when SuggestRelated=true")
	}
}

// ---------------------------------------------------------------------------
// ConciergeAgent tests
// ---------------------------------------------------------------------------

func TestNewConciergeAgent_MinimalOptions(t *testing.T) {
	ca := NewConciergeAgent(ConciergeOptions{
		VenueName: "Test Venue",
		Services:  []string{"room service"},
		Amenities: map[string]Amenity{
			"pool": {Hours: "9 AM - 9 PM", Location: "2nd Floor"},
		},
	})
	if ca == nil {
		t.Fatal("expected non-nil agent")
	}
}

func TestConcierge_HasTools(t *testing.T) {
	ca := NewConciergeAgent(ConciergeOptions{
		VenueName: "Test Venue",
		Services:  []string{"room service"},
		Amenities: map[string]Amenity{
			"pool": {Hours: "9 AM - 9 PM", Location: "2nd Floor"},
		},
	})

	tools := ca.DefineTools()
	names := map[string]bool{}
	for _, td := range tools {
		names[td.Name] = true
	}
	if !names["check_availability"] {
		t.Error("expected check_availability tool")
	}
	if !names["get_directions"] {
		t.Error("expected get_directions tool")
	}
}

func TestConcierge_VenueInfoInGlobalData(t *testing.T) {
	ca := NewConciergeAgent(ConciergeOptions{
		VenueName: "Grand Hotel",
		Services:  []string{"spa", "restaurant"},
		Amenities: map[string]Amenity{
			"pool": {Hours: "7 AM - 10 PM", Location: "2nd Floor"},
			"gym":  {Hours: "24 hours", Location: "3rd Floor"},
		},
		Hours: "8 AM - 10 PM",
	})

	doc := ca.RenderSWML(nil, nil)
	aiConfig := findAIConfig(t, doc)

	gd, ok := aiConfig["global_data"].(map[string]any)
	if !ok {
		t.Fatal("expected global_data")
	}
	if gd["venue_name"] != "Grand Hotel" {
		t.Errorf("expected venue_name=Grand Hotel, got %v", gd["venue_name"])
	}
	// global_data["hours"] carries the hours_of_operation MAP, matching the
	// reference (`"hours": self.hours_of_operation`, concierge.py:168). The
	// single-line Hours option is the shorthand for {"default": Hours}, so it
	// lands under the "default" label. This test previously asserted a bare
	// string, i.e. it asserted go's own divergent wire shape: an agent
	// configured with per-day hours could not express them at all, and
	// global_data disagreed with every other port.
	hours, ok := gd["hours"].(map[string]string)
	if !ok {
		hoursAny, okAny := gd["hours"].(map[string]any)
		if !okAny {
			t.Fatalf("expected hours to be a map, got %T: %v", gd["hours"], gd["hours"])
		}
		hours = map[string]string{}
		for k, v := range hoursAny {
			sv, _ := v.(string)
			hours[k] = sv
		}
	}
	if hours["default"] != "8 AM - 10 PM" {
		t.Errorf("expected hours[default]=8 AM - 10 PM, got %v", hours)
	}
}

// TestConcierge_HoursOfOperationMap covers the reference's labelled
// hours_of_operation map (concierge.py:78): each label renders as its own
// "<Title>: <hours>" line and the whole map reaches global_data. Before this,
// go's ConciergeOptions carried a single `Hours string`, so per-day hours were
// unreachable through the port at all.
func TestConcierge_HoursOfOperationMap(t *testing.T) {
	ca := NewConciergeAgent(ConciergeOptions{
		VenueName: "Grand Hotel",
		Services:  []string{"spa"},
		Amenities: map[string]Amenity{},
		HoursOfOperation: map[string]string{
			"weekdays": "9 AM - 6 PM",
			"saturday": "10 AM - 2 PM",
		},
	})

	// (1) Readable back through the accessor.
	got := ca.HoursOfOperation()
	if got["weekdays"] != "9 AM - 6 PM" || got["saturday"] != "10 AM - 2 PM" {
		t.Errorf("HoursOfOperation() = %v, want both labels", got)
	}

	// (2) Both labels render into the prompt, one line each, sorted for
	// determinism, with no "General hours:" prefix (the reference has none).
	prompt := renderedPromptText(t, ca)
	for _, want := range []string{"Saturday: 10 AM - 2 PM", "Weekdays: 9 AM - 6 PM"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q\n%s", want, prompt)
		}
	}
	if strings.Contains(prompt, "General hours:") {
		t.Error("prompt must not carry a 'General hours:' prefix — the reference emits the bare label lines")
	}

	// (3) special_instructions round-trips through the accessor and reaches the
	// Instructions section (it was previously accepted and never stored).
	withInstr := NewConciergeAgent(ConciergeOptions{
		VenueName:           "Grand Hotel",
		Services:            []string{"spa"},
		Amenities:           map[string]Amenity{},
		SpecialInstructions: []string{"Always mention the rooftop bar."},
	})
	if si := withInstr.SpecialInstructions(); len(si) != 1 || si[0] != "Always mention the rooftop bar." {
		t.Errorf("SpecialInstructions() = %v, want the caller's instruction", si)
	}
	if !strings.Contains(renderedPromptText(t, withInstr), "Always mention the rooftop bar.") {
		t.Error("a special instruction must reach the Instructions prompt section")
	}
}

// renderedPromptText renders the agent's SWML and returns its whole AI-config
// prompt as one searchable string, so a test can assert on rendered prompt COPY
// without depending on the POM section structure.
func renderedPromptText(t *testing.T, ca *ConciergeAgent) string {
	t.Helper()
	aiConfig := findAIConfig(t, ca.RenderSWML(nil, nil))
	blob, err := json.Marshal(aiConfig["prompt"])
	if err != nil {
		t.Fatalf("marshal prompt: %v", err)
	}
	return string(blob)
}

func TestConcierge_CheckAvailability_Found(t *testing.T) {
	ca := NewConciergeAgent(ConciergeOptions{
		VenueName: "Grand Hotel",
		Services:  []string{"spa", "restaurant"},
		Amenities: map[string]Amenity{},
	})

	result, err := ca.OnFunctionCall("check_availability", map[string]any{
		"service": "spa",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, _ := result.(map[string]any)
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "available") {
		t.Errorf("expected availability confirmation, got %q", resp)
	}
}

func TestConcierge_CheckAvailability_NotFound(t *testing.T) {
	ca := NewConciergeAgent(ConciergeOptions{
		VenueName: "Grand Hotel",
		Services:  []string{"spa"},
		Amenities: map[string]Amenity{},
	})

	result, err := ca.OnFunctionCall("check_availability", map[string]any{
		"service": "helicopter",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, _ := result.(map[string]any)
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "don't offer") {
		t.Errorf("expected not-offered message, got %q", resp)
	}
}

func TestConcierge_GetDirections_Found(t *testing.T) {
	ca := NewConciergeAgent(ConciergeOptions{
		VenueName: "Grand Hotel",
		Services:  []string{"spa"},
		Amenities: map[string]Amenity{
			"pool": {Hours: "9 AM - 9 PM", Location: "2nd Floor"},
		},
	})

	result, err := ca.OnFunctionCall("get_directions", map[string]any{
		"location": "pool",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, _ := result.(map[string]any)
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "2nd Floor") {
		t.Errorf("expected location in directions, got %q", resp)
	}
}

func TestConcierge_GetDirections_NotFound(t *testing.T) {
	ca := NewConciergeAgent(ConciergeOptions{
		VenueName: "Grand Hotel",
		Services:  []string{"spa"},
		Amenities: map[string]Amenity{
			"pool": {Hours: "9 AM - 9 PM", Location: "2nd Floor"},
		},
	})

	result, err := ca.OnFunctionCall("get_directions", map[string]any{
		"location": "helipad",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, _ := result.(map[string]any)
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "don't have") {
		t.Errorf("expected not-found message, got %q", resp)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// findAIConfig extracts the AI verb configuration from a rendered SWML document.
func findAIConfig(t *testing.T, doc map[string]any) map[string]any {
	t.Helper()

	sections, ok := doc["sections"].(map[string]any)
	if !ok {
		t.Fatal("expected sections in SWML doc")
	}
	main, ok := sections["main"].([]any)
	if !ok {
		t.Fatal("expected main section as []any")
	}

	for _, v := range main {
		vm, ok := v.(map[string]any)
		if !ok {
			continue
		}
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			return aiCfg
		}
	}

	t.Fatal("AI verb not found in SWML document")
	return nil
}

// findVerbConfig extracts a named verb's configuration from a rendered SWML
// document.  Returns nil if the verb is not present.
func findVerbConfig(doc map[string]any, verbName string) map[string]any {
	sections, ok := doc["sections"].(map[string]any)
	if !ok {
		return nil
	}
	main, ok := sections["main"].([]any)
	if !ok {
		return nil
	}
	for _, v := range main {
		vm, ok := v.(map[string]any)
		if !ok {
			continue
		}
		if cfg, ok := vm[verbName].(map[string]any); ok {
			return cfg
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// BedrockAgent tests
// ---------------------------------------------------------------------------

func TestNewBedrockAgent_Defaults(t *testing.T) {
	ba := NewBedrockAgent(BedrockOptions{})
	if ba == nil {
		t.Fatal("expected non-nil BedrockAgent")
	}
	if ba.AgentBase == nil {
		t.Fatal("expected non-nil AgentBase")
	}
	if ba.GetName() != "bedrock_agent" {
		t.Errorf("expected default name bedrock_agent, got %q", ba.GetName())
	}
	if ba.GetRoute() != "/bedrock" {
		t.Errorf("expected default route /bedrock, got %q", ba.GetRoute())
	}
}

func TestNewBedrockAgent_CustomOptions(t *testing.T) {
	ba := NewBedrockAgent(BedrockOptions{
		Name:         "my_bedrock",
		Route:        "/aws",
		SystemPrompt: "Hello from Bedrock",
		VoiceID:      "joanna",
		Temperature:  0.5,
		TopP:         0.8,
		MaxTokens:    512,
	})
	if ba.GetName() != "my_bedrock" {
		t.Errorf("expected name my_bedrock, got %q", ba.GetName())
	}
	if ba.GetRoute() != "/aws" {
		t.Errorf("expected route /aws, got %q", ba.GetRoute())
	}
}

func TestBedrockAgent_RendersSWMLWithAmazonBedrockVerb(t *testing.T) {
	ba := NewBedrockAgent(BedrockOptions{
		SystemPrompt: "You are a helpful assistant.",
	})

	doc := ba.RenderSWML(nil, nil)

	// The SWML document must NOT contain an "ai" verb in the main section.
	aiCfg := findVerbConfig(doc, "ai")
	if aiCfg != nil {
		t.Error("SWML should not contain an 'ai' verb for BedrockAgent")
	}

	// It MUST contain an "amazon_bedrock" verb.
	bedrockCfg := findVerbConfig(doc, "amazon_bedrock")
	if bedrockCfg == nil {
		t.Fatal("SWML must contain an 'amazon_bedrock' verb for BedrockAgent")
	}
}

func TestBedrockAgent_PromptContainsVoiceAndInferenceParams(t *testing.T) {
	ba := NewBedrockAgent(BedrockOptions{
		SystemPrompt: "Test prompt.",
		VoiceID:      "joanna",
		Temperature:  0.5,
		TopP:         0.8,
	})

	doc := ba.RenderSWML(nil, nil)
	bedrockCfg := findVerbConfig(doc, "amazon_bedrock")
	if bedrockCfg == nil {
		t.Fatal("expected amazon_bedrock verb")
	}

	prompt, ok := bedrockCfg["prompt"].(map[string]any)
	if !ok {
		t.Fatal("expected prompt in amazon_bedrock config")
	}

	if prompt["voice_id"] != "joanna" {
		t.Errorf("expected voice_id=joanna, got %v", prompt["voice_id"])
	}
	if prompt["temperature"] != 0.5 {
		t.Errorf("expected temperature=0.5, got %v", prompt["temperature"])
	}
	if prompt["top_p"] != 0.8 {
		t.Errorf("expected top_p=0.8, got %v", prompt["top_p"])
	}
}

func TestBedrockAgent_TextModelKeysAreFiltered(t *testing.T) {
	ba := NewBedrockAgent(BedrockOptions{
		SystemPrompt: "Test.",
	})
	// Inject text-model-specific LLM params to verify they are stripped.
	ba.SetPromptLlmParams(map[string]any{
		"barge_confidence":  0.5,
		"presence_penalty":  0.1,
		"frequency_penalty": 0.2,
		"some_other_param":  "keep_me",
	})

	doc := ba.RenderSWML(nil, nil)
	bedrockCfg := findVerbConfig(doc, "amazon_bedrock")
	if bedrockCfg == nil {
		t.Fatal("expected amazon_bedrock verb")
	}

	prompt, ok := bedrockCfg["prompt"].(map[string]any)
	if !ok {
		t.Fatal("expected prompt map")
	}

	banned := []string{"barge_confidence", "presence_penalty", "frequency_penalty"}
	for _, k := range banned {
		if _, found := prompt[k]; found {
			t.Errorf("key %q must be filtered from Bedrock prompt config", k)
		}
	}

	// non-banned key must survive
	if prompt["some_other_param"] != "keep_me" {
		t.Errorf("expected some_other_param to be preserved, got %v", prompt["some_other_param"])
	}
}

func TestBedrockAgent_SetVoice(t *testing.T) {
	ba := NewBedrockAgent(BedrockOptions{SystemPrompt: "hi"})
	ba.SetVoice("salli")

	doc := ba.RenderSWML(nil, nil)
	bedrockCfg := findVerbConfig(doc, "amazon_bedrock")
	if bedrockCfg == nil {
		t.Fatal("expected amazon_bedrock verb")
	}
	prompt, ok := bedrockCfg["prompt"].(map[string]any)
	if !ok {
		t.Fatal("expected prompt map")
	}
	if prompt["voice_id"] != "salli" {
		t.Errorf("expected voice_id=salli after SetVoice, got %v", prompt["voice_id"])
	}
}

func TestBedrockAgent_SetInferenceParams(t *testing.T) {
	ba := NewBedrockAgent(BedrockOptions{SystemPrompt: "hi"})
	ba.SetInferenceParams(0.3, 0.6, 2048)

	doc := ba.RenderSWML(nil, nil)
	bedrockCfg := findVerbConfig(doc, "amazon_bedrock")
	if bedrockCfg == nil {
		t.Fatal("expected amazon_bedrock verb")
	}
	prompt, ok := bedrockCfg["prompt"].(map[string]any)
	if !ok {
		t.Fatal("expected prompt map")
	}
	if prompt["temperature"] != 0.3 {
		t.Errorf("expected temperature=0.3, got %v", prompt["temperature"])
	}
	if prompt["top_p"] != 0.6 {
		t.Errorf("expected top_p=0.6, got %v", prompt["top_p"])
	}
}

func TestBedrockAgent_SetLLMTemperature(t *testing.T) {
	ba := NewBedrockAgent(BedrockOptions{SystemPrompt: "hi"})
	ba.SetLLMTemperature(0.2)

	doc := ba.RenderSWML(nil, nil)
	bedrockCfg := findVerbConfig(doc, "amazon_bedrock")
	if bedrockCfg == nil {
		t.Fatal("expected amazon_bedrock verb")
	}
	prompt, ok := bedrockCfg["prompt"].(map[string]any)
	if !ok {
		t.Fatal("expected prompt map")
	}
	if prompt["temperature"] != 0.2 {
		t.Errorf("expected temperature=0.2 after SetLLMTemperature, got %v", prompt["temperature"])
	}
}
