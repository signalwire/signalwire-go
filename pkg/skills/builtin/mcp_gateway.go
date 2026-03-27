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

// MCPGatewaySkill connects to MCP gateway servers and registers their tools.
type MCPGatewaySkill struct {
	skills.BaseSkill
	gatewayURL     string
	authToken      string
	authUser       string
	authPassword   string
	toolPrefix     string
	requestTimeout int
	services       []map[string]any
	registeredTools []skills.ToolRegistration
}

// NewMCPGateway creates a new MCPGatewaySkill.
func NewMCPGateway(params map[string]any) skills.SkillBase {
	return &MCPGatewaySkill{
		BaseSkill: skills.BaseSkill{
			SkillName: "mcp_gateway",
			SkillDesc: "Bridge MCP servers with SWAIG functions",
			SkillVer:  "1.0.0",
			Params:    params,
		},
	}
}

func (s *MCPGatewaySkill) Setup() bool {
	s.gatewayURL = s.GetParamString("gateway_url", "")
	if s.gatewayURL == "" {
		return false
	}
	s.gatewayURL = strings.TrimRight(s.gatewayURL, "/")

	s.authToken = s.GetParamString("auth_token", "")
	s.authUser = s.GetParamString("auth_user", "")
	s.authPassword = s.GetParamString("auth_password", "")
	s.toolPrefix = s.GetParamString("tool_prefix", "mcp_")
	s.requestTimeout = s.GetParamInt("request_timeout", 30)

	// Parse services
	if servicesRaw, ok := s.Params["services"]; ok {
		if servSlice, ok := servicesRaw.([]any); ok {
			for _, svc := range servSlice {
				if m, ok := svc.(map[string]any); ok {
					s.services = append(s.services, m)
				}
			}
		}
	}

	// Test gateway connectivity
	req, err := http.NewRequest("GET", s.gatewayURL+"/health", nil)
	if err != nil {
		return false
	}
	s.applyAuth(req)

	client := &http.Client{Timeout: time.Duration(s.requestTimeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (s *MCPGatewaySkill) applyAuth(req *http.Request) {
	if s.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.authToken)
	} else if s.authUser != "" && s.authPassword != "" {
		req.SetBasicAuth(s.authUser, s.authPassword)
	}
}

func (s *MCPGatewaySkill) RegisterTools() []skills.ToolRegistration {
	// If no services specified, discover all
	if len(s.services) == 0 {
		s.services = s.discoverServices()
	}

	var registrations []skills.ToolRegistration
	for _, svc := range s.services {
		name, _ := svc["name"].(string)
		if name == "" {
			continue
		}
		tools := s.getServiceTools(name)
		for _, tool := range tools {
			reg := s.buildToolRegistration(name, tool)
			if reg != nil {
				registrations = append(registrations, *reg)
			}
		}
	}

	s.registeredTools = registrations
	return registrations
}

func (s *MCPGatewaySkill) discoverServices() []map[string]any {
	req, err := http.NewRequest("GET", s.gatewayURL+"/services", nil)
	if err != nil {
		return nil
	}
	s.applyAuth(req)

	client := &http.Client{Timeout: time.Duration(s.requestTimeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil
	}

	var services []map[string]any
	for name := range data {
		services = append(services, map[string]any{"name": name})
	}
	return services
}

func (s *MCPGatewaySkill) getServiceTools(serviceName string) []map[string]any {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/services/%s/tools", s.gatewayURL, serviceName), nil)
	if err != nil {
		return nil
	}
	s.applyAuth(req)

	client := &http.Client{Timeout: time.Duration(s.requestTimeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil
	}

	toolsRaw, _ := data["tools"].([]any)
	var tools []map[string]any
	for _, t := range toolsRaw {
		if m, ok := t.(map[string]any); ok {
			tools = append(tools, m)
		}
	}
	return tools
}

func (s *MCPGatewaySkill) buildToolRegistration(serviceName string, tool map[string]any) *skills.ToolRegistration {
	toolName, _ := tool["name"].(string)
	if toolName == "" {
		return nil
	}

	swaigName := fmt.Sprintf("%s%s_%s", s.toolPrefix, serviceName, toolName)
	desc, _ := tool["description"].(string)
	if desc == "" {
		desc = toolName
	}

	// Build parameters from inputSchema
	inputSchema, _ := tool["inputSchema"].(map[string]any)
	params := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
	if inputSchema != nil {
		if props, ok := inputSchema["properties"].(map[string]any); ok {
			params["properties"] = props
		}
		if req, ok := inputSchema["required"].([]any); ok {
			params["required"] = req
		}
	}

	// Create handler closure
	handler := func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
		return s.callMCPTool(serviceName, toolName, args, rawData)
	}

	return &skills.ToolRegistration{
		Name:        swaigName,
		Description: fmt.Sprintf("[%s] %s", serviceName, desc),
		Parameters:  params,
		Handler:     handler,
	}
}

func (s *MCPGatewaySkill) callMCPTool(serviceName, toolName string, args map[string]any, rawData map[string]any) *swaig.FunctionResult {
	callID, _ := rawData["call_id"].(string)

	reqData := map[string]any{
		"tool":       toolName,
		"arguments":  args,
		"session_id": callID,
	}

	bodyBytes, _ := json.Marshal(reqData)
	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/services/%s/call", s.gatewayURL, serviceName),
		strings.NewReader(string(bodyBytes)),
	)
	if err != nil {
		return swaig.NewFunctionResult(fmt.Sprintf("Error calling %s.%s", serviceName, toolName))
	}
	req.Header.Set("Content-Type", "application/json")
	s.applyAuth(req)

	client := &http.Client{Timeout: time.Duration(s.requestTimeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return swaig.NewFunctionResult(fmt.Sprintf("Failed to call %s.%s: connection error", serviceName, toolName))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return swaig.NewFunctionResult(fmt.Sprintf("Failed to call %s.%s: HTTP %d", serviceName, toolName, resp.StatusCode))
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return swaig.NewFunctionResult("Error processing MCP response.")
	}

	resultText, _ := data["result"].(string)
	if resultText == "" {
		resultText = "No response from MCP tool."
	}
	return swaig.NewFunctionResult(resultText)
}

func (s *MCPGatewaySkill) GetHints() []string {
	hints := []string{"MCP", "gateway"}
	for _, svc := range s.services {
		if name, ok := svc["name"].(string); ok {
			hints = append(hints, name)
		}
	}
	return hints
}

func (s *MCPGatewaySkill) GetPromptSections() []map[string]any {
	var serviceNames []string
	for _, svc := range s.services {
		if name, ok := svc["name"].(string); ok {
			serviceNames = append(serviceNames, name)
		}
	}
	return []map[string]any{
		{
			"title": "MCP Gateway Integration",
			"body":  "You have access to external MCP services through a gateway.",
			"bullets": []string{
				"Connected to gateway at " + s.gatewayURL,
				"Available services: " + strings.Join(serviceNames, ", "),
				"Functions are prefixed with '" + s.toolPrefix + "' followed by service name",
			},
		},
	}
}

func (s *MCPGatewaySkill) GetParameterSchema() map[string]map[string]any {
	schema := s.BaseSkill.GetParameterSchema()
	schema["gateway_url"] = map[string]any{
		"type":        "string",
		"description": "URL of the MCP Gateway service",
		"required":    true,
	}
	schema["server_name"] = map[string]any{
		"type":        "string",
		"description": "Name of the MCP server",
		"required":    false,
	}
	return schema
}

func init() {
	skills.RegisterSkill("mcp_gateway", NewMCPGateway)
}
