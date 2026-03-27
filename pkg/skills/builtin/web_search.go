package builtin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// WebSearchSkill searches the web using Google Custom Search API.
type WebSearchSkill struct {
	skills.BaseSkill
	apiKey         string
	searchEngineID string
	numResults     int
	toolName       string
}

// NewWebSearch creates a new WebSearchSkill.
func NewWebSearch(params map[string]any) skills.SkillBase {
	return &WebSearchSkill{
		BaseSkill: skills.BaseSkill{
			SkillName: "web_search",
			SkillDesc: "Search the web for information using Google Custom Search API",
			SkillVer:  "2.0.0",
			Params:    params,
		},
	}
}

func (s *WebSearchSkill) RequiredEnvVars() []string {
	if s.Params != nil {
		_, hasKey := s.Params["api_key"]
		_, hasEngine := s.Params["search_engine_id"]
		if hasKey && hasEngine {
			return nil
		}
	}
	return []string{"GOOGLE_SEARCH_API_KEY", "GOOGLE_SEARCH_ENGINE_ID"}
}

func (s *WebSearchSkill) SupportsMultipleInstances() bool { return true }

func (s *WebSearchSkill) GetInstanceKey() string {
	toolName := s.GetParamString("tool_name", "web_search")
	return "web_search_" + toolName
}

func (s *WebSearchSkill) Setup() bool {
	s.apiKey = s.GetParamString("api_key", os.Getenv("GOOGLE_SEARCH_API_KEY"))
	s.searchEngineID = s.GetParamString("search_engine_id", os.Getenv("GOOGLE_SEARCH_ENGINE_ID"))
	if s.apiKey == "" || s.searchEngineID == "" {
		return false
	}
	s.numResults = s.GetParamInt("num_results", 3)
	s.toolName = s.GetParamString("tool_name", "web_search")
	return true
}

func (s *WebSearchSkill) RegisterTools() []skills.ToolRegistration {
	return []skills.ToolRegistration{
		{
			Name:        s.toolName,
			Description: "Search the web for information",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "The search query - what you want to find information about",
					},
				},
				"required": []string{"query"},
			},
			Handler: s.handleWebSearch,
		},
	}
}

func (s *WebSearchSkill) handleWebSearch(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	query, _ := args["query"].(string)
	if query == "" {
		return swaig.NewFunctionResult("Please provide a search query.")
	}

	apiURL := fmt.Sprintf(
		"https://www.googleapis.com/customsearch/v1?key=%s&cx=%s&q=%s&num=%d",
		s.apiKey,
		s.searchEngineID,
		url.QueryEscape(query),
		s.numResults,
	)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return swaig.NewFunctionResult("Sorry, I encountered an error while searching. Please try again later.")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return swaig.NewFunctionResult("Sorry, the search service is unavailable. Please try again later.")
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return swaig.NewFunctionResult("Error processing search results.")
	}

	items, _ := data["items"].([]any)
	if len(items) == 0 {
		return swaig.NewFunctionResult(fmt.Sprintf("No search results found for: %s", query))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Search results for '%s':\n\n", query))

	for i, item := range items {
		if i >= s.numResults {
			break
		}
		m, _ := item.(map[string]any)
		if m == nil {
			continue
		}
		title, _ := m["title"].(string)
		link, _ := m["link"].(string)
		snippet, _ := m["snippet"].(string)

		sb.WriteString(fmt.Sprintf("=== RESULT %d ===\n", i+1))
		sb.WriteString(fmt.Sprintf("Title: %s\n", title))
		sb.WriteString(fmt.Sprintf("URL: %s\n", link))
		sb.WriteString(fmt.Sprintf("Snippet: %s\n\n", snippet))
	}

	return swaig.NewFunctionResult(sb.String())
}

func (s *WebSearchSkill) GetPromptSections() []map[string]any {
	return []map[string]any{
		{
			"title": "Web Search Capability",
			"body":  "You can search the internet for information using the " + s.toolName + " tool.",
			"bullets": []string{
				"Use " + s.toolName + " when users ask for information you need to look up",
				"Summarize results in a clear, helpful way",
			},
		},
	}
}

func (s *WebSearchSkill) GetParameterSchema() map[string]map[string]any {
	schema := s.BaseSkill.GetParameterSchema()
	schema["api_key"] = map[string]any{
		"type":        "string",
		"description": "Google Custom Search API key",
		"required":    true,
		"hidden":      true,
		"env_var":     "GOOGLE_SEARCH_API_KEY",
	}
	schema["search_engine_id"] = map[string]any{
		"type":        "string",
		"description": "Google Custom Search Engine ID",
		"required":    true,
		"hidden":      true,
		"env_var":     "GOOGLE_SEARCH_ENGINE_ID",
	}
	schema["num_results"] = map[string]any{
		"type":        "integer",
		"description": "Number of results to return",
		"default":     3,
		"required":    false,
	}
	return schema
}

func init() {
	skills.RegisterSkill("web_search", NewWebSearch)
}
