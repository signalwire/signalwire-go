package builtin

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/signalwire/signalwire-go/v3/pkg/skills"
	"github.com/signalwire/signalwire-go/v3/pkg/swaig"
)

// MCPGatewaySkill connects to MCP gateway servers and registers their tools.
type MCPGatewaySkill struct {
	skills.BaseSkill
	gatewayURL      string
	authToken       string
	authUser        string
	authPassword    string
	toolPrefix      string
	requestTimeout  int
	sessionTimeout  int
	retryAttempts   int
	verifySSL       bool
	services        []map[string]any
	registeredTools []skills.ToolRegistration
	lastSessionID   string
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

// Setup reads the gateway configuration and PROBES the gateway before accepting
// the load. It returns false — aborting the load — when "gateway_url" is empty,
// when the probe request cannot be built or sent, or when GET <gateway_url>/health
// answers with anything but 200.
//
// Other params: auth_token (bearer) or auth_user/auth_password (basic);
// tool_prefix (default "mcp_"); request_timeout 30s; session_timeout 300s;
// retry_attempts 3; verify_ssl true; and "services", a []any of service maps
// which, when left empty, means "discover every service the gateway offers" at
// RegisterTools time.
//
// Security: verify_ssl=false alone does NOT disable TLS peer verification — the
// deployment must ALSO set SIGNALWIRE_MCP_ALLOW_INSECURE_TLS, so a config
// toggle cannot silently weaken TLS on a connection that carries API keys.
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
	s.sessionTimeout = s.GetParamInt("session_timeout", 300)
	s.retryAttempts = s.GetParamInt("retry_attempts", 3)
	s.verifySSL = s.GetParamBool("verify_ssl", true)

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

	client := s.newHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// insecureTLSOptInEnv is the environment variable a deployment MUST set to
// actually disable TLS peer verification for the MCP gateway. Setting the
// skill's verify_ssl=false alone is NOT sufficient: turning verification off
// (a MITM-exposing footgun that carries API keys) requires this EXPLICIT,
// deliberate opt-in in ADDITION to the config toggle. This keeps the secure
// default honest — an off value in config cannot silently weaken TLS.
const insecureTLSOptInEnv = "SIGNALWIRE_MCP_ALLOW_INSECURE_TLS"

// insecureTLSOptedIn reports whether the deployment has explicitly opted in to
// disabling TLS verification via insecureTLSOptInEnv (truthy: "1"/"true"/"yes",
// case-insensitive).
func insecureTLSOptedIn() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(insecureTLSOptInEnv))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// newHTTPClient creates an HTTP client that respects the verifySSL setting.
//
// TLS verification is ON by default and stays on unless BOTH (a) the skill's
// verify_ssl param is false AND (b) the deployment explicitly opts in via the
// SIGNALWIRE_MCP_ALLOW_INSECURE_TLS env var. Requiring the explicit opt-in flag
// (not merely a config value) means an off toggle alone cannot disable
// verification — TLS-off is always a deliberate, auditable action.
func (s *MCPGatewaySkill) newHTTPClient() *http.Client {
	transport := http.DefaultTransport
	// insecure is a variable gated on the explicit opt-in; it is never a
	// hardcoded literal, so verification-off is impossible without the env flag.
	insecure := !s.verifySSL && insecureTLSOptedIn()
	if !s.verifySSL && !insecure {
		log.Printf("mcp_gateway: verify_ssl=false ignored — set %s to explicitly "+
			"opt in to disabling TLS verification; keeping verification ON",
			insecureTLSOptInEnv)
	}
	if insecure {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure}, //nolint:gosec // explicit opt-in via SIGNALWIRE_MCP_ALLOW_INSECURE_TLS
		}
	}
	return &http.Client{
		Timeout:   time.Duration(s.requestTimeout) * time.Second,
		Transport: transport,
	}
}

func (s *MCPGatewaySkill) applyAuth(req *http.Request) {
	if s.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.authToken)
	} else if s.authUser != "" && s.authPassword != "" {
		req.SetBasicAuth(s.authUser, s.authPassword)
	}
}

// RegisterTools queries the gateway and returns one SWAIG tool per MCP tool
// exposed by each configured service. When no services were configured it first
// discovers every service the gateway advertises. Tool names carry the
// configured prefix (default "mcp_") plus the service name, so tools from
// different services cannot collide.
//
// A final internal "_mcp_gateway_hangup" tool is always appended: the agent
// calls it at end of call so the gateway session for each service is torn down
// rather than left to expire on session_timeout.
//
// Because it performs network calls, this returns an empty list (plus the
// hangup hook) if the gateway is unreachable at registration time.
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

	// Register hangup hook for session cleanup
	registrations = append(registrations, skills.ToolRegistration{
		Name:        "_mcp_gateway_hangup",
		Description: "Internal cleanup function for MCP sessions",
		Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		Handler:     s.hangupHandler,
	})

	s.registeredTools = registrations
	return registrations
}

func (s *MCPGatewaySkill) discoverServices() []map[string]any {
	req, err := http.NewRequest("GET", s.gatewayURL+"/services", nil)
	if err != nil {
		return nil
	}
	s.applyAuth(req)

	client := s.newHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer func() { _ = resp.Body.Close() }()

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

	client := s.newHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer func() { _ = resp.Body.Close() }()

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

// hangupHandler handles call hangup by cleaning up the MCP session via DELETE.
func (s *MCPGatewaySkill) hangupHandler(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
	// Resolve session ID: check global_data.mcp_call_id first, fall back to call_id
	sessionID := s.resolveSessionID(rawData)

	client := s.newHTTPClient()
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/sessions/%s", s.gatewayURL, sessionID), nil)
	if err == nil {
		s.applyAuth(req)
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
		}
	}

	return swaig.NewFunctionResult("Session cleanup complete")
}

// resolveSessionID checks global_data["mcp_call_id"] first, then falls back to call_id.
func (s *MCPGatewaySkill) resolveSessionID(rawData map[string]any) string {
	if globalData, ok := rawData["global_data"].(map[string]any); ok {
		if mcpCallID, ok := globalData["mcp_call_id"].(string); ok && mcpCallID != "" {
			return mcpCallID
		}
	}
	if callID, ok := rawData["call_id"].(string); ok && callID != "" {
		return callID
	}
	return "unknown"
}

func (s *MCPGatewaySkill) callMCPTool(serviceName, toolName string, args map[string]any, rawData map[string]any) *swaig.FunctionResult {
	// Check global_data["mcp_call_id"] first, fall back to call_id
	sessionID := s.resolveSessionID(rawData)
	s.lastSessionID = sessionID

	// Build request payload matching Python: tool, arguments, session_id, timeout, metadata
	callID, _ := rawData["call_id"].(string)
	reqData := map[string]any{
		"tool":       toolName,
		"arguments":  args,
		"session_id": sessionID,
		"timeout":    s.sessionTimeout,
		"metadata": map[string]any{
			"call_id": callID,
		},
	}

	var lastError string
	retries := s.retryAttempts
	if retries <= 0 {
		retries = 1
	}

	bodyBytes, err := json.Marshal(reqData)
	if err != nil {
		return swaig.NewFunctionResult(fmt.Sprintf("Error calling %s.%s", serviceName, toolName))
	}

	for range retries {
		req, err := http.NewRequest("POST",
			fmt.Sprintf("%s/services/%s/call", s.gatewayURL, serviceName),
			bytes.NewReader(bodyBytes),
		)
		if err != nil {
			return swaig.NewFunctionResult(fmt.Sprintf("Error calling %s.%s", serviceName, toolName))
		}
		req.Header.Set("Content-Type", "application/json")
		s.applyAuth(req)

		client := s.newHTTPClient()
		resp, err := client.Do(req)
		if err != nil {
			lastError = "connection error"
			// Retry on connection errors
			continue
		}

		if resp.StatusCode == http.StatusOK {
			var data map[string]any
			if decErr := json.NewDecoder(resp.Body).Decode(&data); decErr != nil {
				_ = resp.Body.Close()
				return swaig.NewFunctionResult("Error processing MCP response.")
			}
			_ = resp.Body.Close()
			resultText, _ := data["result"].(string)
			if resultText == "" {
				resultText = "No response from MCP tool."
			}
			return swaig.NewFunctionResult(resultText)
		}

		// Non-200 response
		statusCode := resp.StatusCode
		var errData map[string]any
		if decErr := json.NewDecoder(resp.Body).Decode(&errData); decErr == nil {
			if msg, ok := errData["error"].(string); ok {
				lastError = msg
			} else {
				lastError = fmt.Sprintf("HTTP %d", statusCode)
			}
		} else {
			lastError = fmt.Sprintf("HTTP %d", statusCode)
		}
		_ = resp.Body.Close()

		if statusCode >= 500 {
			// Server error — retry
			continue
		}
		// Client error — do not retry
		break
	}

	return swaig.NewFunctionResult(fmt.Sprintf("Failed to call %s.%s: %s", serviceName, toolName, lastError))
}

// GetGlobalData returns MCP gateway state for DataMap variable expansion.
// The keys are mcp_gateway_url, mcp_session_id and mcp_services.
func (s *MCPGatewaySkill) GetGlobalData() map[string]any {
	serviceNames := make([]string, 0, len(s.services))
	for _, svc := range s.services {
		if name, ok := svc["name"].(string); ok {
			serviceNames = append(serviceNames, name)
		}
	}
	return map[string]any{
		"mcp_gateway_url": s.gatewayURL,
		"mcp_session_id":  s.lastSessionID,
		"mcp_services":    serviceNames,
	}
}

// GetHints returns "MCP" and "gateway" plus the name of each configured
// service, biasing speech recognition toward the service names a caller may
// speak.
func (s *MCPGatewaySkill) GetHints() []string {
	hints := []string{"MCP", "gateway"}
	for _, svc := range s.services {
		if name, ok := svc["name"].(string); ok {
			hints = append(hints, name)
		}
	}
	return hints
}

// GetPromptSections returns one POM section naming the gateway URL, the
// available services, the tool-name prefix, and the fact that each service keeps
// its own session state for the duration of the call.
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
				"Each service maintains its own session state throughout the call",
			},
		},
	}
}

// GetParameterSchema extends the common skill parameters with the gateway
// configuration: the required gateway_url; the credentials auth_token and
// auth_password (both hidden) and auth_user; the optional services list; and
// session_timeout (300), tool_prefix ("mcp_"), retry_attempts (3),
// request_timeout (30) and verify_ssl (true). The verify_ssl description states
// the SIGNALWIRE_MCP_ALLOW_INSECURE_TLS second opt-in, so a caller reading the
// schema cannot mistake the toggle for sufficient.
func (s *MCPGatewaySkill) GetParameterSchema() map[string]map[string]any {
	schema := s.BaseSkill.GetParameterSchema()
	schema["gateway_url"] = map[string]any{
		"type":        "string",
		"description": "URL of the MCP Gateway service",
		"required":    true,
	}
	schema["auth_token"] = map[string]any{
		"type":        "string",
		"description": "Bearer token for authentication (alternative to basic auth)",
		"required":    false,
		"hidden":      true,
	}
	schema["auth_user"] = map[string]any{
		"type":        "string",
		"description": "Username for basic authentication",
		"required":    false,
	}
	schema["auth_password"] = map[string]any{
		"type":        "string",
		"description": "Password for basic authentication",
		"required":    false,
		"hidden":      true,
	}
	schema["services"] = map[string]any{
		"type":        "array",
		"description": "List of MCP services to connect to (empty for all available)",
		"default":     []any{},
		"required":    false,
	}
	schema["session_timeout"] = map[string]any{
		"type":        "integer",
		"description": "Session timeout in seconds",
		"default":     300,
		"required":    false,
	}
	schema["tool_prefix"] = map[string]any{
		"type":        "string",
		"description": "Prefix for registered SWAIG function names",
		"default":     "mcp_",
		"required":    false,
	}
	schema["retry_attempts"] = map[string]any{
		"type":        "integer",
		"description": "Number of retry attempts for failed requests",
		"default":     3,
		"required":    false,
	}
	schema["request_timeout"] = map[string]any{
		"type":        "integer",
		"description": "Request timeout in seconds",
		"default":     30,
		"required":    false,
	}
	schema["verify_ssl"] = map[string]any{
		"type": "boolean",
		"description": "Verify SSL certificates (default true). Setting this false " +
			"ALSO requires the SIGNALWIRE_MCP_ALLOW_INSECURE_TLS env var to be set " +
			"to actually disable verification; without that explicit opt-in, " +
			"verification stays ON.",
		"default":  true,
		"required": false,
	}
	return schema
}

func init() {
	skills.RegisterSkill("mcp_gateway", NewMCPGateway)
}
