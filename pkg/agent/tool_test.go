package agent

import (
	"testing"

	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// ---------------------------------------------------------------------------
// Tool registration
// ---------------------------------------------------------------------------

func TestDefineTool_InsertionOrder(t *testing.T) {
	a := NewAgentBase()
	a.DefineTool(ToolDefinition{Name: "alpha", Description: "a"})
	a.DefineTool(ToolDefinition{Name: "bravo", Description: "b"})
	a.DefineTool(ToolDefinition{Name: "charlie", Description: "c"})

	tools := a.DefineTools()
	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}
	order := []string{"alpha", "bravo", "charlie"}
	for i, name := range order {
		if tools[i].Name != name {
			t.Errorf("tool[%d] = %q, want %q", i, tools[i].Name, name)
		}
	}
}

func TestDefineTool_OverwritePreservesOrder(t *testing.T) {
	a := NewAgentBase()
	a.DefineTool(ToolDefinition{Name: "first", Description: "v1"})
	a.DefineTool(ToolDefinition{Name: "second", Description: "v1"})
	a.DefineTool(ToolDefinition{Name: "first", Description: "v2"})

	tools := a.DefineTools()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	if tools[0].Name != "first" {
		t.Errorf("first tool should still be 'first', got %q", tools[0].Name)
	}
	if tools[0].Description != "v2" {
		t.Errorf("first tool description should be updated to 'v2', got %q", tools[0].Description)
	}
}

func TestDefineTool_WithHandler(t *testing.T) {
	a := NewAgentBase()
	handler := func(args map[string]any, raw map[string]any) *swaig.FunctionResult {
		return swaig.NewFunctionResult("handled")
	}
	a.DefineTool(ToolDefinition{
		Name:        "greet",
		Description: "Greet user",
		Handler:     handler,
	})

	result, err := a.OnFunctionCall("greet", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, _ := result.(map[string]any)
	resp, _ := m["response"].(string)
	if resp != "handled" {
		t.Errorf("response = %q, want %q", resp, "handled")
	}
}

func TestDefineTool_WithParameters(t *testing.T) {
	a := NewAgentBase()
	a.DefineTool(ToolDefinition{
		Name:        "lookup",
		Description: "Look up data",
		Parameters: map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query",
			},
		},
	})

	tools := a.DefineTools()
	if tools[0].Parameters == nil {
		t.Error("expected non-nil parameters")
	}
	if _, ok := tools[0].Parameters["query"]; !ok {
		t.Error("expected 'query' parameter")
	}
}

func TestDefineTool_SecureTool(t *testing.T) {
	a := NewAgentBase()
	a.DefineTool(ToolDefinition{
		Name:    "secret",
		Secure:  true,
		Handler: func(args map[string]any, raw map[string]any) *swaig.FunctionResult { return nil },
	})

	tools := a.DefineTools()
	if !tools[0].Secure {
		t.Error("expected Secure=true")
	}
}

func TestDefineTool_WithFillers(t *testing.T) {
	a := NewAgentBase()
	a.DefineTool(ToolDefinition{
		Name: "slow_op",
		Fillers: map[string][]string{
			"en-US": {"Please wait...", "One moment..."},
		},
		Handler: func(args map[string]any, raw map[string]any) *swaig.FunctionResult { return nil },
	})

	tools := a.DefineTools()
	if tools[0].Fillers == nil {
		t.Error("expected non-nil fillers")
	}
	if len(tools[0].Fillers["en-US"]) != 2 {
		t.Errorf("expected 2 fillers for en-US, got %d", len(tools[0].Fillers["en-US"]))
	}
}

func TestDefineTool_WithMetaData(t *testing.T) {
	a := NewAgentBase()
	a.DefineTool(ToolDefinition{
		Name:     "meta_tool",
		MetaData: map[string]any{"version": "1.0"},
		Handler:  func(args map[string]any, raw map[string]any) *swaig.FunctionResult { return nil },
	})

	tools := a.DefineTools()
	if tools[0].MetaData == nil {
		t.Error("expected non-nil MetaData")
	}
	if tools[0].MetaData["version"] != "1.0" {
		t.Errorf("MetaData[version] = %v, want %q", tools[0].MetaData["version"], "1.0")
	}
}

// ---------------------------------------------------------------------------
// RegisterSwaigFunction (DataMap-style)
// ---------------------------------------------------------------------------

func TestRegisterSwaigFunction_Basic(t *testing.T) {
	a := NewAgentBase()
	a.RegisterSwaigFunction(map[string]any{
		"function": "dm_tool",
		"purpose":  "DataMap tool",
		"data_map": map[string]any{
			"expressions": []map[string]any{},
		},
	})

	tools := a.DefineTools()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Name != "dm_tool" {
		t.Errorf("tool name = %q, want %q", tools[0].Name, "dm_tool")
	}
	if tools[0].SwaigFields == nil {
		t.Error("expected non-nil SwaigFields")
	}
}

func TestRegisterSwaigFunction_EmptyName(t *testing.T) {
	a := NewAgentBase()
	a.RegisterSwaigFunction(map[string]any{
		"purpose": "no name",
	})
	tools := a.DefineTools()
	if len(tools) != 0 {
		t.Error("should not register function with empty name")
	}
}

func TestRegisterSwaigFunction_MixedWithHandlerTools(t *testing.T) {
	a := NewAgentBase()
	a.DefineTool(ToolDefinition{
		Name:    "handler_tool",
		Handler: func(args map[string]any, raw map[string]any) *swaig.FunctionResult { return nil },
	})
	a.RegisterSwaigFunction(map[string]any{
		"function": "datamap_tool",
		"data_map": map[string]any{},
	})

	tools := a.DefineTools()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	if tools[0].Name != "handler_tool" {
		t.Errorf("first tool should be handler_tool, got %q", tools[0].Name)
	}
	if tools[1].Name != "datamap_tool" {
		t.Errorf("second tool should be datamap_tool, got %q", tools[1].Name)
	}
}

// ---------------------------------------------------------------------------
// OnFunctionCall dispatch
// ---------------------------------------------------------------------------

func TestOnFunctionCall_PassesArgs(t *testing.T) {
	a := NewAgentBase()
	var receivedArgs map[string]any
	a.DefineTool(ToolDefinition{
		Name: "echo",
		Handler: func(args map[string]any, raw map[string]any) *swaig.FunctionResult {
			receivedArgs = args
			return swaig.NewFunctionResult("ok")
		},
	})

	_, err := a.OnFunctionCall("echo", map[string]any{"key": "value"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedArgs["key"] != "value" {
		t.Errorf("args[key] = %v, want %q", receivedArgs["key"], "value")
	}
}

func TestOnFunctionCall_PassesRawData(t *testing.T) {
	a := NewAgentBase()
	var receivedRaw map[string]any
	a.DefineTool(ToolDefinition{
		Name: "raw_check",
		Handler: func(args map[string]any, raw map[string]any) *swaig.FunctionResult {
			receivedRaw = raw
			return swaig.NewFunctionResult("ok")
		},
	})

	rawData := map[string]any{"call_id": "call-123"}
	_, err := a.OnFunctionCall("raw_check", nil, rawData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedRaw["call_id"] != "call-123" {
		t.Errorf("rawData[call_id] = %v, want %q", receivedRaw["call_id"], "call-123")
	}
}

func TestOnFunctionCall_NilResult(t *testing.T) {
	a := NewAgentBase()
	a.DefineTool(ToolDefinition{
		Name: "nil_result",
		Handler: func(args map[string]any, raw map[string]any) *swaig.FunctionResult {
			return nil
		},
	})

	result, err := a.OnFunctionCall("nil_result", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

// ---------------------------------------------------------------------------
// SWML rendering of tools
// ---------------------------------------------------------------------------

func TestToolRendering_WebhookURL(t *testing.T) {
	a := NewAgentBase(WithBasicAuth("user", "pass"))
	a.SetWebHookUrl("https://example.com/swaig")
	a.DefineTool(ToolDefinition{
		Name:        "test_tool",
		Description: "A test tool",
		Handler: func(args map[string]any, raw map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("ok")
		},
	})

	doc := a.RenderSWML(nil, nil)
	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)

	for _, v := range main {
		vm, _ := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			swaigCfg, _ := aiCfg["SWAIG"].(map[string]any)
			if swaigCfg == nil {
				t.Fatal("expected SWAIG config")
			}
			functions, _ := swaigCfg["functions"].([]map[string]any)
			if len(functions) == 0 {
				t.Fatal("expected at least 1 function")
			}
			webhookURL, _ := functions[0]["web_hook_url"].(string)
			if webhookURL != "https://example.com/swaig" {
				t.Errorf("webhook URL = %q, want %q", webhookURL, "https://example.com/swaig")
			}
			return
		}
	}
	t.Fatal("AI verb not found")
}

func TestToolRendering_DataMapSkipsWebhook(t *testing.T) {
	a := NewAgentBase()
	a.RegisterSwaigFunction(map[string]any{
		"function": "dm_tool",
		"purpose":  "DataMap",
		"data_map": map[string]any{
			"expressions": []any{},
		},
	})

	doc := a.RenderSWML(nil, nil)
	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)

	for _, v := range main {
		vm, _ := v.(map[string]any)
		if aiCfg, ok := vm["ai"].(map[string]any); ok {
			swaigCfg, _ := aiCfg["SWAIG"].(map[string]any)
			if swaigCfg == nil {
				t.Fatal("expected SWAIG config")
			}
			functions, _ := swaigCfg["functions"].([]map[string]any)
			if len(functions) != 1 {
				t.Fatalf("expected 1 function, got %d", len(functions))
			}
			// DataMap tools should use SwaigFields directly, not have web_hook_url
			if functions[0]["function"] != "dm_tool" {
				t.Errorf("function name = %v", functions[0]["function"])
			}
			return
		}
	}
	t.Fatal("AI verb not found")
}
