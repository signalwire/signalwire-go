package builtin

import (
	"strings"
	"testing"

	"github.com/signalwire/signalwire-go/pkg/skills"
)

func TestMathSkill_Addition(t *testing.T) {
	factory := skills.GetSkillFactory("math")
	s := factory(nil)
	s.Setup()
	tools := s.RegisterTools()
	result := tools[0].Handler(map[string]any{"expression": "2 + 3"}, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "5") {
		t.Errorf("expected 5 in response, got %q", resp)
	}
}

func TestMathSkill_Subtraction(t *testing.T) {
	factory := skills.GetSkillFactory("math")
	s := factory(nil)
	s.Setup()
	tools := s.RegisterTools()
	result := tools[0].Handler(map[string]any{"expression": "10 - 3"}, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "7") {
		t.Errorf("expected 7 in response, got %q", resp)
	}
}

func TestMathSkill_Multiplication(t *testing.T) {
	factory := skills.GetSkillFactory("math")
	s := factory(nil)
	s.Setup()
	tools := s.RegisterTools()
	result := tools[0].Handler(map[string]any{"expression": "4 * 5"}, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "20") {
		t.Errorf("expected 20 in response, got %q", resp)
	}
}

func TestMathSkill_Division(t *testing.T) {
	factory := skills.GetSkillFactory("math")
	s := factory(nil)
	s.Setup()
	tools := s.RegisterTools()
	result := tools[0].Handler(map[string]any{"expression": "10 / 2"}, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "5") {
		t.Errorf("expected 5 in response, got %q", resp)
	}
}

func TestMathSkill_DivisionByZero(t *testing.T) {
	factory := skills.GetSkillFactory("math")
	s := factory(nil)
	s.Setup()
	tools := s.RegisterTools()
	result := tools[0].Handler(map[string]any{"expression": "5 / 0"}, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "Error") && !strings.Contains(resp, "Invalid") {
		t.Errorf("expected error for division by zero, got %q", resp)
	}
}

func TestMathSkill_Parentheses(t *testing.T) {
	factory := skills.GetSkillFactory("math")
	s := factory(nil)
	s.Setup()
	tools := s.RegisterTools()
	result := tools[0].Handler(map[string]any{"expression": "(4 + 6) * 2"}, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "20") {
		t.Errorf("expected 20 in response, got %q", resp)
	}
}

func TestMathSkill_Modulo(t *testing.T) {
	factory := skills.GetSkillFactory("math")
	s := factory(nil)
	s.Setup()
	tools := s.RegisterTools()
	result := tools[0].Handler(map[string]any{"expression": "17 % 5"}, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "2") {
		t.Errorf("expected 2 in response, got %q", resp)
	}
}

func TestMathSkill_EmptyExpression(t *testing.T) {
	factory := skills.GetSkillFactory("math")
	s := factory(nil)
	s.Setup()
	tools := s.RegisterTools()
	result := tools[0].Handler(map[string]any{"expression": ""}, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if resp == "" {
		t.Error("expected non-empty response for empty expression")
	}
}

func TestMathSkill_InvalidExpression(t *testing.T) {
	factory := skills.GetSkillFactory("math")
	s := factory(nil)
	s.Setup()
	tools := s.RegisterTools()
	result := tools[0].Handler(map[string]any{"expression": "hello world"}, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "Error") && !strings.Contains(resp, "Invalid") {
		t.Errorf("expected error for invalid expression, got %q", resp)
	}
}

func TestMathSkill_FloatResult(t *testing.T) {
	factory := skills.GetSkillFactory("math")
	s := factory(nil)
	s.Setup()
	tools := s.RegisterTools()
	result := tools[0].Handler(map[string]any{"expression": "7 / 2"}, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "3.5") {
		t.Errorf("expected 3.5 in response, got %q", resp)
	}
}

func TestMathSkill_NegativeNumbers(t *testing.T) {
	factory := skills.GetSkillFactory("math")
	s := factory(nil)
	s.Setup()
	tools := s.RegisterTools()
	result := tools[0].Handler(map[string]any{"expression": "-5 + 3"}, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "-2") {
		t.Errorf("expected -2 in response, got %q", resp)
	}
}

func TestMathSkill_HasPromptSections(t *testing.T) {
	factory := skills.GetSkillFactory("math")
	s := factory(nil)
	s.Setup()
	sections := s.GetPromptSections()
	if len(sections) == 0 {
		t.Error("expected prompt sections")
	}
}

func TestSafeEval_DirectUnit(t *testing.T) {
	tests := []struct {
		expr   string
		expect float64
		hasErr bool
	}{
		{"2 + 3", 5, false},
		{"10 - 4", 6, false},
		{"3 * 7", 21, false},
		{"15 / 3", 5, false},
		{"10 % 3", 1, false},
		{"(2 + 3) * 4", 20, false},
		{"5 / 0", 0, true},
		{"abc", 0, true},
	}

	for _, tc := range tests {
		result, err := safeEval(tc.expr)
		if tc.hasErr {
			if err == nil {
				t.Errorf("safeEval(%q) expected error, got %v", tc.expr, result)
			}
		} else {
			if err != nil {
				t.Errorf("safeEval(%q) unexpected error: %v", tc.expr, err)
			}
			if result != tc.expect {
				t.Errorf("safeEval(%q) = %v, want %v", tc.expr, result, tc.expect)
			}
		}
	}
}
