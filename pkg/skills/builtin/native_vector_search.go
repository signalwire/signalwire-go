package builtin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// NativeVectorSearchSkill searches knowledge using a remote search server.
type NativeVectorSearchSkill struct {
	skills.BaseSkill
	toolName   string
	remoteURL  string
	indexName  string
	count      int
	noResults  string
}

// NewNativeVectorSearch creates a new NativeVectorSearchSkill.
func NewNativeVectorSearch(params map[string]any) skills.SkillBase {
	return &NativeVectorSearchSkill{
		BaseSkill: skills.BaseSkill{
			SkillName: "native_vector_search",
			SkillDesc: "Search document indexes using vector similarity (network mode)",
			SkillVer:  "1.0.0",
			Params:    params,
		},
	}
}

func (s *NativeVectorSearchSkill) SupportsMultipleInstances() bool { return true }

func (s *NativeVectorSearchSkill) GetInstanceKey() string {
	toolName := s.GetParamString("tool_name", "search_knowledge")
	return "native_vector_search_" + toolName
}

func (s *NativeVectorSearchSkill) Setup() bool {
	s.toolName = s.GetParamString("tool_name", "search_knowledge")
	s.remoteURL = s.GetParamString("remote_url", "")
	s.indexName = s.GetParamString("index_name", "default")
	s.count = s.GetParamInt("count", 5)
	s.noResults = s.GetParamString("no_results_message", "No information found for '{query}'")

	if s.remoteURL == "" {
		return false
	}

	// Strip trailing slash
	s.remoteURL = strings.TrimRight(s.remoteURL, "/")

	// Test connection
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(s.remoteURL + "/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		return false
	}
	resp.Body.Close()
	return true
}

func (s *NativeVectorSearchSkill) RegisterTools() []skills.ToolRegistration {
	desc := s.GetParamString("description", "Search the knowledge base for information")
	return []skills.ToolRegistration{
		{
			Name:        s.toolName,
			Description: desc,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query or question",
					},
				},
				"required": []string{"query"},
			},
			Handler: s.handleSearch,
		},
	}
}

func (s *NativeVectorSearchSkill) handleSearch(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	query, _ := args["query"].(string)
	if query == "" {
		return swaig.NewFunctionResult("Please provide a search query.")
	}

	searchReq := map[string]any{
		"query":      query,
		"index_name": s.indexName,
		"count":      s.count,
	}

	bodyBytes, _ := json.Marshal(searchReq)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(s.remoteURL+"/search", "application/json", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return swaig.NewFunctionResult("Search service is temporarily unavailable. Please try again later.")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return swaig.NewFunctionResult("Search service returned an error. Please try again later.")
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return swaig.NewFunctionResult("Error processing search results.")
	}

	results, _ := data["results"].([]any)
	if len(results) == 0 {
		msg := strings.ReplaceAll(s.noResults, "{query}", query)
		return swaig.NewFunctionResult(msg)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d results for '%s':\n\n", len(results), query))
	for i, r := range results {
		m, _ := r.(map[string]any)
		if m == nil {
			continue
		}
		content, _ := m["content"].(string)
		score, _ := m["score"].(float64)
		metadata, _ := m["metadata"].(map[string]any)
		filename := ""
		if metadata != nil {
			filename, _ = metadata["filename"].(string)
		}

		sb.WriteString(fmt.Sprintf("**Result %d** (from %s, relevance: %.2f)\n%s\n\n", i+1, filename, score, content))
	}

	return swaig.NewFunctionResult(sb.String())
}

func (s *NativeVectorSearchSkill) GetHints() []string {
	hints := []string{"search", "find", "look up", "documentation", "knowledge base"}
	if customHints, ok := s.Params["hints"].([]any); ok {
		for _, h := range customHints {
			if hs, ok := h.(string); ok {
				hints = append(hints, hs)
			}
		}
	}
	return hints
}

func (s *NativeVectorSearchSkill) GetPromptSections() []map[string]any {
	return []map[string]any{
		{
			"title": "Knowledge Search",
			"body":  "You can search knowledge sources using " + s.toolName + ".",
			"bullets": []string{
				"Use " + s.toolName + " to search document indexes",
				"Search for relevant information using clear, specific queries",
				"If no results are found, suggest the user try rephrasing their question",
			},
		},
	}
}

func (s *NativeVectorSearchSkill) GetParameterSchema() map[string]map[string]any {
	schema := s.BaseSkill.GetParameterSchema()
	schema["remote_url"] = map[string]any{"type": "string", "description": "URL of remote search server", "required": true}
	schema["index_name"] = map[string]any{"type": "string", "description": "Name of index on remote server", "default": "default", "required": false}
	schema["count"] = map[string]any{"type": "integer", "description": "Number of results", "default": 5, "required": false}
	return schema
}

func init() {
	skills.RegisterSkill("native_vector_search", NewNativeVectorSearch)
}
