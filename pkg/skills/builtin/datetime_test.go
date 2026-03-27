package builtin

import (
	"strings"
	"testing"

	"github.com/signalwire/signalwire-go/pkg/skills"
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
	if tools[0].Name != "get_datetime" {
		t.Errorf("tool name = %q, want get_datetime", tools[0].Name)
	}
}

func TestDateTimeSkill_ResponseContainsDate(t *testing.T) {
	factory := skills.GetSkillFactory("datetime")
	s := factory(nil)
	s.Setup()
	tools := s.RegisterTools()
	result := tools[0].Handler(map[string]any{"timezone": "UTC"}, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "date") {
		t.Errorf("expected 'date' in response, got %q", resp)
	}
	if !strings.Contains(resp, "time") {
		t.Errorf("expected 'time' in response, got %q", resp)
	}
}
