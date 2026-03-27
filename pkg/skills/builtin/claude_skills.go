package builtin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// ClaudeSkillsSkill calls the Claude API for reasoning tasks.
type ClaudeSkillsSkill struct {
	skills.BaseSkill
	apiKey   string
	toolName string
	model    string
}

// NewClaudeSkills creates a new ClaudeSkillsSkill.
func NewClaudeSkills(params map[string]any) skills.SkillBase {
	return &ClaudeSkillsSkill{
		BaseSkill: skills.BaseSkill{
			SkillName: "claude_skills",
			SkillDesc: "Call Claude API for reasoning and analysis tasks",
			SkillVer:  "1.0.0",
			Params:    params,
		},
	}
}

func (s *ClaudeSkillsSkill) RequiredEnvVars() []string {
	if s.Params != nil {
		if _, ok := s.Params["api_key"]; ok {
			return nil
		}
	}
	return []string{"ANTHROPIC_API_KEY"}
}

func (s *ClaudeSkillsSkill) Setup() bool {
	s.apiKey = s.GetParamString("api_key", os.Getenv("ANTHROPIC_API_KEY"))
	if s.apiKey == "" {
		return false
	}
	s.toolName = s.GetParamString("tool_name", "ask_claude")
	s.model = s.GetParamString("model", "claude-sonnet-4-20250514")
	return true
}

func (s *ClaudeSkillsSkill) RegisterTools() []skills.ToolRegistration {
	return []skills.ToolRegistration{
		{
			Name:        s.toolName,
			Description: "Ask Claude for help with reasoning, analysis, or complex questions",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"prompt": map[string]any{
						"type":        "string",
						"description": "The question or task to send to Claude",
					},
				},
				"required": []string{"prompt"},
			},
			Handler: s.handleAskClaude,
		},
	}
}

func (s *ClaudeSkillsSkill) handleAskClaude(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	prompt, _ := args["prompt"].(string)
	if prompt == "" {
		return swaig.NewFunctionResult("Please provide a question or task for Claude.")
	}

	reqBody := map[string]any{
		"model":      s.model,
		"max_tokens": 1024,
		"messages": []map[string]any{
			{"role": "user", "content": prompt},
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return swaig.NewFunctionResult("Error creating request to Claude.")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return swaig.NewFunctionResult("Error connecting to Claude API.")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return swaig.NewFunctionResult(fmt.Sprintf("Claude API returned status %d. Please try again later.", resp.StatusCode))
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return swaig.NewFunctionResult("Error processing Claude's response.")
	}

	// Extract text from content blocks
	content, _ := data["content"].([]any)
	var textParts []string
	for _, block := range content {
		m, _ := block.(map[string]any)
		if m == nil {
			continue
		}
		if t, ok := m["text"].(string); ok {
			textParts = append(textParts, t)
		}
	}

	if len(textParts) == 0 {
		return swaig.NewFunctionResult("Claude returned an empty response.")
	}

	return swaig.NewFunctionResult(strings.Join(textParts, "\n"))
}

func (s *ClaudeSkillsSkill) GetPromptSections() []map[string]any {
	return []map[string]any{
		{
			"title": "Claude AI Reasoning",
			"body":  "You can delegate complex reasoning tasks to Claude.",
			"bullets": []string{
				"Use " + s.toolName + " for complex analysis, reasoning, or tasks requiring deep thought",
				"Provide clear, specific prompts for best results",
			},
		},
	}
}

func (s *ClaudeSkillsSkill) GetParameterSchema() map[string]map[string]any {
	schema := s.BaseSkill.GetParameterSchema()
	schema["api_key"] = map[string]any{
		"type":        "string",
		"description": "Anthropic API key",
		"required":    true,
		"hidden":      true,
		"env_var":     "ANTHROPIC_API_KEY",
	}
	return schema
}

func init() {
	skills.RegisterSkill("claude_skills", NewClaudeSkills)
}
