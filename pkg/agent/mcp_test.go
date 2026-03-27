package agent

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func newMcpAgent() *AgentBase {
	a := NewAgentBase(WithName("test-mcp"), WithRoute("/test"))
	a.EnableMcpServer()
	a.DefineTool(ToolDefinition{
		Name:        "get_weather",
		Description: "Get the weather for a location",
		Parameters: map[string]any{
			"location": map[string]any{
				"type":        "string",
				"description": "City name",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			loc, _ := args["location"].(string)
			if loc == "" {
				loc = "unknown"
			}
			return swaig.NewFunctionResult("72F sunny in " + loc)
		},
	})
	return a
}

// ---------------------------------------------------------------------------
// MCP Server tests
// ---------------------------------------------------------------------------

func TestBuildMcpToolList(t *testing.T) {
	a := newMcpAgent()
	tools := a.buildMcpToolList()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0]["name"] != "get_weather" {
		t.Errorf("expected name=get_weather, got %v", tools[0]["name"])
	}
	if tools[0]["description"] != "Get the weather for a location" {
		t.Errorf("unexpected description: %v", tools[0]["description"])
	}
	schema, ok := tools[0]["inputSchema"].(map[string]any)
	if !ok {
		t.Fatal("expected inputSchema to be a map")
	}
	if schema["type"] != "object" {
		t.Errorf("expected type=object, got %v", schema["type"])
	}
	props, _ := schema["properties"].(map[string]any)
	if _, ok := props["location"]; !ok {
		t.Error("expected location in properties")
	}
}

func TestInitializeHandshake(t *testing.T) {
	a := newMcpAgent()
	resp := a.handleMcpRequest(map[string]any{
		"jsonrpc": "2.0",
		"id":      float64(1),
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities":   map[string]any{},
			"clientInfo":     map[string]any{"name": "test", "version": "1.0"},
		},
	})

	if resp["jsonrpc"] != "2.0" {
		t.Errorf("expected jsonrpc=2.0, got %v", resp["jsonrpc"])
	}
	if resp["id"] != float64(1) {
		t.Errorf("expected id=1, got %v", resp["id"])
	}
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatal("expected result map")
	}
	if result["protocolVersion"] != "2025-06-18" {
		t.Errorf("unexpected protocolVersion: %v", result["protocolVersion"])
	}
	caps, _ := result["capabilities"].(map[string]any)
	if _, ok := caps["tools"]; !ok {
		t.Error("expected tools in capabilities")
	}
}

func TestInitializedNotification(t *testing.T) {
	a := newMcpAgent()
	resp := a.handleMcpRequest(map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	})
	if _, ok := resp["result"]; !ok {
		t.Error("expected result in response")
	}
}

func TestToolsList(t *testing.T) {
	a := newMcpAgent()
	resp := a.handleMcpRequest(map[string]any{
		"jsonrpc": "2.0",
		"id":      float64(2),
		"method":  "tools/list",
		"params":  map[string]any{},
	})

	if resp["id"] != float64(2) {
		t.Errorf("expected id=2, got %v", resp["id"])
	}
	result, _ := resp["result"].(map[string]any)
	tools, _ := result["tools"].([]map[string]any)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0]["name"] != "get_weather" {
		t.Errorf("expected get_weather, got %v", tools[0]["name"])
	}
}

func TestToolsCall(t *testing.T) {
	a := newMcpAgent()
	resp := a.handleMcpRequest(map[string]any{
		"jsonrpc": "2.0",
		"id":      float64(3),
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "get_weather",
			"arguments": map[string]any{"location": "Orlando"},
		},
	})

	if resp["id"] != float64(3) {
		t.Errorf("expected id=3, got %v", resp["id"])
	}
	result, _ := resp["result"].(map[string]any)
	if result["isError"] != false {
		t.Error("expected isError=false")
	}
	content, _ := result["content"].([]map[string]any)
	if len(content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(content))
	}
	if content[0]["type"] != "text" {
		t.Errorf("expected type=text, got %v", content[0]["type"])
	}
	text, _ := content[0]["text"].(string)
	if text == "" || !contains(text, "Orlando") {
		t.Errorf("expected text containing 'Orlando', got %q", text)
	}
}

func TestToolsCallUnknown(t *testing.T) {
	a := newMcpAgent()
	resp := a.handleMcpRequest(map[string]any{
		"jsonrpc": "2.0",
		"id":      float64(4),
		"method":  "tools/call",
		"params":  map[string]any{"name": "nonexistent", "arguments": map[string]any{}},
	})

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error in response")
	}
	if errObj["code"] != -32602 {
		t.Errorf("expected code=-32602, got %v", errObj["code"])
	}
	msg, _ := errObj["message"].(string)
	if !contains(msg, "nonexistent") {
		t.Errorf("expected message to contain 'nonexistent', got %q", msg)
	}
}

func TestUnknownMethod(t *testing.T) {
	a := newMcpAgent()
	resp := a.handleMcpRequest(map[string]any{
		"jsonrpc": "2.0",
		"id":      float64(5),
		"method":  "resources/list",
		"params":  map[string]any{},
	})

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error in response")
	}
	if errObj["code"] != -32601 {
		t.Errorf("expected code=-32601, got %v", errObj["code"])
	}
}

func TestPing(t *testing.T) {
	a := newMcpAgent()
	resp := a.handleMcpRequest(map[string]any{
		"jsonrpc": "2.0",
		"id":      float64(6),
		"method":  "ping",
	})
	if _, ok := resp["result"]; !ok {
		t.Error("expected result in response")
	}
}

func TestInvalidJsonrpcVersion(t *testing.T) {
	a := newMcpAgent()
	resp := a.handleMcpRequest(map[string]any{
		"jsonrpc": "1.0",
		"id":      float64(7),
		"method":  "initialize",
	})

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error in response")
	}
	if errObj["code"] != -32600 {
		t.Errorf("expected code=-32600, got %v", errObj["code"])
	}
}

// ---------------------------------------------------------------------------
// MCP Client tests (add_mcp_server)
// ---------------------------------------------------------------------------

func TestAddMcpServerBasic(t *testing.T) {
	a := NewAgentBase()
	a.AddMcpServer(MCPServerConfig{URL: "https://mcp.example.com/tools"})

	if len(a.mcpServers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(a.mcpServers))
	}
	if a.mcpServers[0]["url"] != "https://mcp.example.com/tools" {
		t.Errorf("unexpected url: %v", a.mcpServers[0]["url"])
	}
}

func TestAddMcpServerWithHeaders(t *testing.T) {
	a := NewAgentBase()
	a.AddMcpServer(MCPServerConfig{
		URL:     "https://mcp.example.com/tools",
		Headers: map[string]string{"Authorization": "Bearer sk-xxx"},
	})

	headers, _ := a.mcpServers[0]["headers"].(map[string]string)
	if headers["Authorization"] != "Bearer sk-xxx" {
		t.Errorf("unexpected auth header: %v", headers["Authorization"])
	}
}

func TestAddMcpServerWithResources(t *testing.T) {
	a := NewAgentBase()
	a.AddMcpServer(MCPServerConfig{
		URL:          "https://mcp.example.com/crm",
		Resources:    true,
		ResourceVars: map[string]string{"caller_id": "${caller_id_number}"},
	})

	if a.mcpServers[0]["resources"] != true {
		t.Error("expected resources=true")
	}
	rv, _ := a.mcpServers[0]["resource_vars"].(map[string]string)
	if rv["caller_id"] != "${caller_id_number}" {
		t.Errorf("unexpected resource_vars: %v", rv)
	}
}

func TestAddMultipleMcpServers(t *testing.T) {
	a := NewAgentBase()
	a.AddMcpServer(MCPServerConfig{URL: "https://mcp1.example.com"})
	a.AddMcpServer(MCPServerConfig{URL: "https://mcp2.example.com"})

	if len(a.mcpServers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(a.mcpServers))
	}
}

func TestMcpMethodChaining(t *testing.T) {
	a := NewAgentBase()
	result := a.AddMcpServer(MCPServerConfig{URL: "https://mcp.example.com"})
	if result != a {
		t.Error("AddMcpServer should return self")
	}
}

func TestEnableMcpServer(t *testing.T) {
	a := NewAgentBase()
	if a.mcpServerEnabled {
		t.Error("expected mcpServerEnabled=false by default")
	}
	result := a.EnableMcpServer()
	if !a.mcpServerEnabled {
		t.Error("expected mcpServerEnabled=true")
	}
	if result != a {
		t.Error("EnableMcpServer should return self")
	}
}

func TestMcpServersInSwml(t *testing.T) {
	a := NewAgentBase(WithName("test"))
	a.AddMcpServer(MCPServerConfig{
		URL:     "https://mcp.example.com/tools",
		Headers: map[string]string{"Authorization": "Bearer key"},
	})

	swml := a.RenderSWML(nil, nil)
	sections, _ := swml["sections"].(map[string]any)
	main, _ := sections["main"].([]any)

	// Find the AI verb
	var aiConfig map[string]any
	for _, v := range main {
		if verbMap, ok := v.(map[string]any); ok {
			if ai, ok := verbMap["ai"]; ok {
				aiConfig, _ = ai.(map[string]any)
			}
		}
	}
	if aiConfig == nil {
		t.Fatal("expected ai verb in SWML")
	}

	servers, ok := aiConfig["mcp_servers"]
	if !ok {
		t.Fatal("expected mcp_servers in AI config")
	}
	serverList, _ := servers.([]map[string]any)
	if len(serverList) != 1 {
		t.Fatalf("expected 1 mcp server, got %d", len(serverList))
	}
	if serverList[0]["url"] != "https://mcp.example.com/tools" {
		t.Errorf("unexpected url: %v", serverList[0]["url"])
	}
}

// ---------------------------------------------------------------------------
// HTTP endpoint test
// ---------------------------------------------------------------------------

func TestMcpHttpEndpoint(t *testing.T) {
	a := newMcpAgent()
	mux := a.buildMux()
	ts := httptest.NewServer(mux)
	defer ts.Close()

	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params":  map[string]any{},
	})

	resp, err := http.Post(ts.URL+"/test/mcp", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	resultMap, _ := result["result"].(map[string]any)
	tools, _ := resultMap["tools"].([]any)
	if len(tools) != 1 {
		t.Errorf("expected 1 tool from HTTP, got %d", len(tools))
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
