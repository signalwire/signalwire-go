package builtin

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// DataSphereServerlessSkill provides DataSphere search using DataMap (serverless execution).
type DataSphereServerlessSkill struct {
	skills.BaseSkill
	spaceName  string
	projectID  string
	token      string
	documentID string
	count      int
	distance   float64
	toolName   string
	apiURL     string
	authHeader string
}

// NewDataSphereServerless creates a new DataSphereServerlessSkill.
func NewDataSphereServerless(params map[string]any) skills.SkillBase {
	return &DataSphereServerlessSkill{
		BaseSkill: skills.BaseSkill{
			SkillName: "datasphere_serverless",
			SkillDesc: "Search knowledge using DataSphere with serverless DataMap execution",
			SkillVer:  "1.0.0",
			Params:    params,
		},
	}
}

func (s *DataSphereServerlessSkill) RequiredEnvVars() []string {
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

func (s *DataSphereServerlessSkill) SupportsMultipleInstances() bool { return true }

func (s *DataSphereServerlessSkill) GetInstanceKey() string {
	toolName := s.GetParamString("tool_name", "search_datasphere_serverless")
	return "datasphere_serverless_" + toolName
}

func (s *DataSphereServerlessSkill) Setup() bool {
	s.spaceName = s.GetParamString("space_name", os.Getenv("SIGNALWIRE_SPACE_NAME"))
	s.projectID = s.GetParamString("project_id", os.Getenv("SIGNALWIRE_PROJECT_ID"))
	s.token = s.GetParamString("token", os.Getenv("SIGNALWIRE_TOKEN"))
	s.documentID = s.GetParamString("document_id", "")

	if s.spaceName == "" || s.projectID == "" || s.token == "" || s.documentID == "" {
		return false
	}

	s.count = s.GetParamInt("count", 1)
	s.distance = s.GetParamFloat("distance", 3.0)
	s.toolName = s.GetParamString("tool_name", "search_datasphere_serverless")
	s.apiURL = fmt.Sprintf("https://%s.signalwire.com/api/datasphere/documents/search", s.spaceName)
	s.authHeader = base64.StdEncoding.EncodeToString([]byte(s.projectID + ":" + s.token))
	return true
}

// RegisterTools returns DataMap-style tool registration for serverless execution.
// The actual tool is registered as a DataMap function that runs on SignalWire servers.
func (s *DataSphereServerlessSkill) RegisterTools() []skills.ToolRegistration {
	return []skills.ToolRegistration{
		{
			Name:        s.toolName,
			Description: "Search the knowledge base for information (serverless)",
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
			SwaigFields: map[string]any{
				"data_map": map[string]any{
					"webhooks": []map[string]any{
						{
							"url":    s.apiURL,
							"method": "POST",
							"headers": map[string]string{
								"Content-Type":  "application/json",
								"Authorization": "Basic " + s.authHeader,
							},
							"params": map[string]any{
								"document_id":  s.documentID,
								"query_string": "${args.query}",
								"count":        s.count,
								"distance":     s.distance,
							},
							"foreach": map[string]any{
								"input_key":  "chunks",
								"output_key": "formatted_results",
								"max":        s.count,
								"append":     "=== RESULT ===\n${this.text}\n" + strings.Repeat("=", 50) + "\n\n",
							},
							"output": map[string]any{
								"response": `I found results for "${args.query}":` + "\n\n${formatted_results}",
							},
						},
					},
					"error_keys": []string{"error"},
					"output": map[string]any{
						"response": "No results found. Try rephrasing your question.",
					},
				},
			},
		},
	}
}

// handleSearch is a fallback handler for non-DataMap execution.
func (s *DataSphereServerlessSkill) handleSearch(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	return swaig.NewFunctionResult("This tool is designed for serverless DataMap execution on SignalWire servers.")
}

func (s *DataSphereServerlessSkill) GetPromptSections() []map[string]any {
	return []map[string]any{
		{
			"title": "Knowledge Search Capability (Serverless)",
			"body":  "You can search a knowledge base using the " + s.toolName + " tool.",
			"bullets": []string{
				"Use " + s.toolName + " for information queries",
				"This tool executes on SignalWire servers for optimal performance",
			},
		},
	}
}

func (s *DataSphereServerlessSkill) GetParameterSchema() map[string]map[string]any {
	schema := s.BaseSkill.GetParameterSchema()
	schema["space_name"] = map[string]any{"type": "string", "description": "SignalWire space name", "required": true}
	schema["project_id"] = map[string]any{"type": "string", "description": "SignalWire project ID", "required": true, "env_var": "SIGNALWIRE_PROJECT_ID"}
	schema["token"] = map[string]any{"type": "string", "description": "SignalWire API token", "required": true, "hidden": true, "env_var": "SIGNALWIRE_TOKEN"}
	schema["document_id"] = map[string]any{"type": "string", "description": "DataSphere document ID", "required": true}
	schema["count"] = map[string]any{"type": "integer", "description": "Number of results", "default": 1, "required": false}
	return schema
}

func init() {
	skills.RegisterSkill("datasphere_serverless", NewDataSphereServerless)
}
