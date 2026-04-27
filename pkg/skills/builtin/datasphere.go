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

// DataSphereSkill searches knowledge using SignalWire DataSphere RAG stack.
type DataSphereSkill struct {
	skills.BaseSkill
	spaceName       string
	projectID       string
	token           string
	documentID      string
	count           int
	distance        float64
	toolName        string
	apiURL          string
	tags            []string
	language        string
	posToExpand     []string
	maxSynonyms     int
	noResultsMessage string
}

// NewDataSphere creates a new DataSphereSkill.
func NewDataSphere(params map[string]any) skills.SkillBase {
	return &DataSphereSkill{
		BaseSkill: skills.BaseSkill{
			SkillName: "datasphere",
			SkillDesc: "Search knowledge using SignalWire DataSphere RAG stack",
			SkillVer:  "1.0.0",
			Params:    params,
		},
		maxSynonyms: -1,
	}
}

func (s *DataSphereSkill) RequiredEnvVars() []string {
	if s.Params != nil {
		_, hasSpace := s.Params["space_name"]
		_, hasProject := s.Params["project_id"]
		_, hasToken := s.Params["token"]
		if hasSpace && hasProject && hasToken {
			return nil
		}
	}
	return []string{"SIGNALWIRE_PROJECT_ID", "SIGNALWIRE_TOKEN", "SIGNALWIRE_SPACE_NAME"}
}

func (s *DataSphereSkill) SupportsMultipleInstances() bool { return true }

func (s *DataSphereSkill) GetInstanceKey() string {
	toolName := s.GetParamString("tool_name", "search_knowledge")
	return "datasphere_" + toolName
}

func (s *DataSphereSkill) Setup() bool {
	s.spaceName = s.GetParamString("space_name", os.Getenv("SIGNALWIRE_SPACE_NAME"))
	s.projectID = s.GetParamString("project_id", os.Getenv("SIGNALWIRE_PROJECT_ID"))
	s.token = s.GetParamString("token", os.Getenv("SIGNALWIRE_TOKEN"))
	s.documentID = s.GetParamString("document_id", "")

	if s.spaceName == "" || s.projectID == "" || s.token == "" || s.documentID == "" {
		return false
	}

	s.count = s.GetParamInt("count", 1)
	s.distance = s.GetParamFloat("distance", 3.0)
	s.toolName = s.GetParamString("tool_name", "search_knowledge")
	s.apiURL = fmt.Sprintf("https://%s.signalwire.com/api/datasphere/documents/search", s.spaceName)

	// Optional: tags
	if v, ok := s.GetParam("tags"); ok {
		if arr, ok := v.([]any); ok {
			s.tags = nil
			for _, e := range arr {
				if str, ok := e.(string); ok {
					s.tags = append(s.tags, str)
				}
			}
		}
	}

	// Optional: language
	s.language = s.GetParamString("language", "")

	// Optional: pos_to_expand
	if v, ok := s.GetParam("pos_to_expand"); ok {
		if arr, ok := v.([]any); ok {
			s.posToExpand = nil
			for _, e := range arr {
				if str, ok := e.(string); ok {
					s.posToExpand = append(s.posToExpand, str)
				}
			}
		}
	}

	// Optional: max_synonyms (-1 means not set)
	s.maxSynonyms = s.GetParamInt("max_synonyms", -1)

	// Optional: no_results_message
	s.noResultsMessage = s.GetParamString("no_results_message",
		"I couldn't find any relevant information for '%s' in the knowledge base. Try rephrasing your question or asking about a different topic.")

	return true
}

func (s *DataSphereSkill) RegisterTools() []skills.ToolRegistration {
	return []skills.ToolRegistration{
		{
			Name:        s.toolName,
			Description: "Search the knowledge base for information on any topic",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "The search query",
					},
				},
				"required": []string{"query"},
			},
			Handler: s.handleSearch,
		},
	}
}

func (s *DataSphereSkill) handleSearch(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	query, _ := args["query"].(string)
	if query == "" {
		return swaig.NewFunctionResult("Please provide a search query.")
	}

	payload := map[string]any{
		"document_id":  s.documentID,
		"query_string": query,
		"distance":     s.distance,
		"count":        s.count,
	}

	// Add optional parameters only if they were provided
	if len(s.tags) > 0 {
		payload["tags"] = s.tags
	}
	if s.language != "" {
		payload["language"] = s.language
	}
	if len(s.posToExpand) > 0 {
		payload["pos_to_expand"] = s.posToExpand
	}
	if s.maxSynonyms >= 0 {
		payload["max_synonyms"] = s.maxSynonyms
	}

	bodyBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", s.apiURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return swaig.NewFunctionResult("Error creating search request.")
	}
	req.SetBasicAuth(s.projectID, s.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return swaig.NewFunctionResult("Sorry, the knowledge search timed out. Please try again.")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return swaig.NewFunctionResult("Sorry, there was an error accessing the knowledge base.")
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return swaig.NewFunctionResult("Error processing search results.")
	}

	chunks, _ := data["chunks"].([]any)
	if len(chunks) == 0 {
		return swaig.NewFunctionResult(fmt.Sprintf(s.noResultsMessage, query))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d results for '%s':\n\n", len(chunks), query))
	for i, chunk := range chunks {
		m, _ := chunk.(map[string]any)
		if m == nil {
			continue
		}
		text := ""
		if t, ok := m["text"].(string); ok {
			text = t
		} else if t, ok := m["content"].(string); ok {
			text = t
		}
		sb.WriteString(fmt.Sprintf("=== RESULT %d ===\n%s\n%s\n\n", i+1, text, strings.Repeat("=", 50)))
	}

	return swaig.NewFunctionResult(sb.String())
}

func (s *DataSphereSkill) GetGlobalData() map[string]any {
	return map[string]any{
		"datasphere_enabled": true,
		"document_id":        s.documentID,
		"knowledge_provider": "SignalWire DataSphere",
	}
}

func (s *DataSphereSkill) GetPromptSections() []map[string]any {
	return []map[string]any{
		{
			"title": "Knowledge Search Capability",
			"body":  "You can search a knowledge base for information using the " + s.toolName + " tool.",
			"bullets": []string{
				"Use " + s.toolName + " when users ask for information that might be in the knowledge base",
				"Search for relevant information using clear, specific queries",
				"Summarize search results in a clear, helpful way",
				"If no results are found, suggest the user try rephrasing their question",
			},
		},
	}
}

func (s *DataSphereSkill) GetParameterSchema() map[string]map[string]any {
	schema := s.BaseSkill.GetParameterSchema()
	schema["space_name"] = map[string]any{"type": "string", "description": "SignalWire space name", "required": true}
	schema["project_id"] = map[string]any{"type": "string", "description": "SignalWire project ID", "required": true, "env_var": "SIGNALWIRE_PROJECT_ID"}
	schema["token"] = map[string]any{"type": "string", "description": "SignalWire API token", "required": true, "hidden": true, "env_var": "SIGNALWIRE_TOKEN"}
	schema["document_id"] = map[string]any{"type": "string", "description": "DataSphere document ID", "required": true}
	schema["count"] = map[string]any{"type": "integer", "description": "Number of results to return", "default": 1, "required": false, "minimum": 1, "maximum": 10}
	schema["distance"] = map[string]any{"type": "number", "description": "Maximum distance threshold for results (lower is more relevant)", "default": 3.0, "required": false, "minimum": 0.0, "maximum": 10.0}
	schema["tags"] = map[string]any{"type": "array", "description": "Tags to filter search results", "required": false, "items": map[string]any{"type": "string"}}
	schema["language"] = map[string]any{"type": "string", "description": "Language code for query expansion (e.g., 'en', 'es')", "required": false}
	schema["pos_to_expand"] = map[string]any{"type": "array", "description": "Parts of speech to expand with synonyms", "required": false, "items": map[string]any{"type": "string", "enum": []string{"NOUN", "VERB", "ADJ", "ADV"}}}
	schema["max_synonyms"] = map[string]any{"type": "integer", "description": "Maximum number of synonyms to use for query expansion", "required": false, "minimum": 1, "maximum": 10}
	schema["no_results_message"] = map[string]any{"type": "string", "description": "Message to return when no results are found", "default": "I couldn't find any relevant information for '%s' in the knowledge base. Try rephrasing your question or asking about a different topic.", "required": false}
	return schema
}

func init() {
	skills.RegisterSkill("datasphere", NewDataSphere)
}
