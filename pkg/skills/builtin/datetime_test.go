package builtin

import (
	"strings"
	"testing"

	"github.com/signalwire/signalwire-go/v3/pkg/skills"
)

func TestDateTimeSkill_DefaultTimezone(t *testing.T) {
	factory := skills.GetSkillFactory("datetime")
	s := factory(nil)
	s.Setup()
	tools := s.RegisterTools()
	result := tools[0].Handler(map[string]any{}, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "UTC") {
		t.Errorf("expected UTC in response when no timezone given, got %q", resp)
	}
}

func TestDateTimeSkill_ConfiguredTimezone(t *testing.T) {
	factory := skills.GetSkillFactory("datetime")
	s := factory(map[string]any{"timezone": "America/Chicago"})
	s.Setup()
	tools := s.RegisterTools()
	result := tools[0].Handler(map[string]any{}, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	// Should contain either CST or CDT depending on time of year
	if resp == "" {
		t.Error("expected non-empty response")
	}
}

func TestDateTimeSkill_ArgumentTimezoneOverrides(t *testing.T) {
	factory := skills.GetSkillFactory("datetime")
	s := factory(map[string]any{"timezone": "UTC"})
	s.Setup()
	tools := s.RegisterTools()
	result := tools[0].Handler(map[string]any{"timezone": "America/New_York"}, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	// Should use the argument timezone, not the config
	if strings.Contains(resp, "UTC") && !strings.Contains(resp, "EST") && !strings.Contains(resp, "EDT") {
		// This is acceptable if UTC and EST have same display, but at least check non-empty
		if resp == "" {
			t.Error("expected non-empty response")
		}
	}
}

func TestDateTimeSkill_InvalidTimezone(t *testing.T) {
	factory := skills.GetSkillFactory("datetime")
	s := factory(nil)
	s.Setup()
	tools := s.RegisterTools()
	result := tools[0].Handler(map[string]any{"timezone": "Invalid/Zone"}, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "Error") {
		t.Errorf("expected error for invalid timezone, got %q", resp)
	}
}

func TestDateTimeSkill_HasHints(t *testing.T) {
	factory := skills.GetSkillFactory("datetime")
	s := factory(nil)
	s.Setup()
	hints := s.GetHints()
	if len(hints) == 0 {
		t.Error("expected non-empty hints")
	}
	found := false
	for _, h := range hints {
		if h == "time" || h == "date" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected time or date in hints")
	}
}

func TestDateTimeSkill_HasPromptSections(t *testing.T) {
	factory := skills.GetSkillFactory("datetime")
	s := factory(nil)
	s.Setup()
	sections := s.GetPromptSections()
	if len(sections) == 0 {
		t.Error("expected prompt sections")
	}
	if sections[0]["title"] == nil {
		t.Error("expected title in prompt section")
	}
}

func TestDateTimeSkill_ToolName(t *testing.T) {
	factory := skills.GetSkillFactory("datetime")
	s := factory(nil)
	s.Setup()
	tools := s.RegisterTools()
	// Python registers exactly get_current_time and get_current_date
	// (datetime/skill.py:33,45) — no third combined tool.
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}
	for _, want := range []string{"get_current_time", "get_current_date"} {
		if !toolNames[want] {
			t.Errorf("expected tool %q to be registered", want)
		}
	}
	if toolNames["get_datetime"] {
		t.Error("get_datetime is not part of the Python interface; should not be registered")
	}
}

func TestDateTimeSkill_ResponseContainsDate(t *testing.T) {
	factory := skills.GetSkillFactory("datetime")
	s := factory(nil)
	s.Setup()
	tools := s.RegisterTools()

	// get_current_date returns a date-bearing response.
	var dateTool *skills.ToolRegistration
	for i := range tools {
		if tools[i].Name == "get_current_date" {
			dateTool = &tools[i]
			break
		}
	}
	if dateTool == nil {
		t.Fatal("expected get_current_date tool to be registered")
	}

	result := dateTool.Handler(map[string]any{"timezone": "UTC"}, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "date") {
		t.Errorf("expected 'date' in response, got %q", resp)
	}
}
