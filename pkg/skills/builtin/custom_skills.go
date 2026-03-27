package builtin

import (
	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// CustomSkillsSkill is a meta-skill that registers user-defined tools.
type CustomSkillsSkill struct {
	skills.BaseSkill
	tools []map[string]any
}

// NewCustomSkills creates a new CustomSkillsSkill.
func NewCustomSkills(params map[string]any) skills.SkillBase {
	return &CustomSkillsSkill{
		BaseSkill: skills.BaseSkill{
			SkillName: "custom_skills",
			SkillDesc: "Register user-defined tools from configuration",
			SkillVer:  "1.0.0",
			Params:    params,
		},
	}
}

func (s *CustomSkillsSkill) SupportsMultipleInstances() bool { return true }

func (s *CustomSkillsSkill) GetInstanceKey() string {
	name := s.GetParamString("tool_name", "custom_skills")
	return "custom_skills_" + name
}

func (s *CustomSkillsSkill) Setup() bool {
	toolsRaw, ok := s.Params["tools"]
	if !ok {
		return false
	}

	toolsSlice, ok := toolsRaw.([]any)
	if !ok || len(toolsSlice) == 0 {
		return false
	}

	s.tools = make([]map[string]any, 0, len(toolsSlice))
	for _, t := range toolsSlice {
		m, ok := t.(map[string]any)
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		if name == "" {
			continue
		}
		s.tools = append(s.tools, m)
	}

	return len(s.tools) > 0
}

func (s *CustomSkillsSkill) RegisterTools() []skills.ToolRegistration {
	var registrations []skills.ToolRegistration

	for _, toolDef := range s.tools {
		name, _ := toolDef["name"].(string)
		description, _ := toolDef["description"].(string)
		if description == "" {
			description = "Custom tool: " + name
		}

		params := map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
		if p, ok := toolDef["parameters"].(map[string]any); ok {
			params = p
		}

		// Build response from tool definition
		responseTemplate, _ := toolDef["response"].(string)
		if responseTemplate == "" {
			responseTemplate = "Tool " + name + " executed."
		}

		// Create handler
		tmpl := responseTemplate
		handler := func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult(tmpl)
		}

		// If there's a data_map, include it in SwaigFields
		var swaigFields map[string]any
		if dm, ok := toolDef["data_map"].(map[string]any); ok {
			swaigFields = map[string]any{
				"data_map": dm,
			}
		}

		registrations = append(registrations, skills.ToolRegistration{
			Name:        name,
			Description: description,
			Parameters:  params,
			Handler:     handler,
			SwaigFields: swaigFields,
		})
	}

	return registrations
}

func (s *CustomSkillsSkill) GetPromptSections() []map[string]any {
	if len(s.tools) == 0 {
		return nil
	}

	var bullets []string
	for _, t := range s.tools {
		name, _ := t["name"].(string)
		desc, _ := t["description"].(string)
		if desc == "" {
			desc = "Custom tool"
		}
		bullets = append(bullets, name+": "+desc)
	}

	return []map[string]any{
		{
			"title":   "Custom Tools",
			"body":    "You have access to the following custom tools:",
			"bullets": bullets,
		},
	}
}

func (s *CustomSkillsSkill) GetParameterSchema() map[string]map[string]any {
	schema := s.BaseSkill.GetParameterSchema()
	schema["tools"] = map[string]any{
		"type":        "array",
		"description": "List of tool definitions with name, description, parameters, and response",
		"required":    true,
	}
	return schema
}

func init() {
	skills.RegisterSkill("custom_skills", NewCustomSkills)
}
