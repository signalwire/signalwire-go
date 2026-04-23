package prefabs

import (
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
	if gd["hours"] != "8 AM - 10 PM" {
		t.Errorf("expected hours=8 AM - 10 PM, got %v", gd["hours"])
	}
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
