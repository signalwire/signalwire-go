// Tests proving swml.Service can host SWAIG functions and serve a non-agent
// SWML doc (e.g. ai_sidecar) without depending on AgentBase. This is the
// contract that lets sidecar / non-agent verbs reuse the SWAIG dispatch
// surface that previously lived only on AgentBase.

package swml

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func makeSwaigSvc(t *testing.T) *Service {
	t.Helper()
	return NewService(
		WithName("svc"),
		WithBasicAuth("u", "p"),
	)
}

func authHeader() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte("u:p"))
}

func TestServiceDefineToolDispatchesViaOnFunctionCall(t *testing.T) {
	svc := makeSwaigSvc(t)
	captured := map[string]any{}
	svc.DefineTool(&ToolDefinition{
		Name:        "lookup",
		Description: "Look it up",
		Handler: func(args map[string]any, raw map[string]any) any {
			for k, v := range args {
				captured[k] = v
			}
			return map[string]any{"response": "ok"}
		},
	})
	result := svc.OnFunctionCall("lookup", map[string]any{"x": "y"}, map[string]any{})
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["response"] != "ok" {
		t.Errorf("response = %v, want ok", m["response"])
	}
	if captured["x"] != "y" {
		t.Errorf("captured[x] = %v, want y", captured["x"])
	}
}

func TestServiceOnFunctionCallReturnsNotFoundForUnknown(t *testing.T) {
	svc := makeSwaigSvc(t)
	result := svc.OnFunctionCall("no_such_fn", map[string]any{}, map[string]any{})
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "not found") {
		t.Errorf("expected 'not found' in response, got %q", resp)
	}
}

func TestServiceListToolNamesReturnsRegisteredOrder(t *testing.T) {
	svc := makeSwaigSvc(t)
	svc.DefineTool(&ToolDefinition{Name: "first", Handler: func(_, _ map[string]any) any { return nil }})
	svc.DefineTool(&ToolDefinition{Name: "second", Handler: func(_, _ map[string]any) any { return nil }})
	names := svc.ListToolNames()
	if len(names) != 2 || names[0] != "first" || names[1] != "second" {
		t.Errorf("ListToolNames = %v, want [first second]", names)
	}
}

func TestServiceRegisterSwaigFunctionTracksInOrder(t *testing.T) {
	svc := makeSwaigSvc(t)
	svc.RegisterSwaigFunction(map[string]any{
		"function":    "datamap_tool",
		"description": "from data map",
	})
	if !svc.HasTool("datamap_tool") {
		t.Errorf("HasTool(datamap_tool) = false, want true")
	}
}

// ---- /swaig HTTP endpoint tests --------------------------------------

func makeRequest(method, path, body string) *http.Request {
	var rdr *bytes.Reader
	if body != "" {
		rdr = bytes.NewReader([]byte(body))
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Authorization", authHeader())
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func TestServiceSwaigGETReturnsSwml(t *testing.T) {
	svc := makeSwaigSvc(t)
	svc.Hangup(nil)
	mux := svc.buildMux()
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, makeRequest(http.MethodGet, "/swaig", ""))
	if rr.Code != 200 {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var doc map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := doc["sections"]; !ok {
		t.Errorf("response missing sections key: %v", doc)
	}
}

func TestServiceSwaigPOSTDispatchesRegisteredHandler(t *testing.T) {
	svc := makeSwaigSvc(t)
	svc.DefineTool(&ToolDefinition{
		Name:        "lookup_competitor",
		Description: "Look up competitor pricing.",
		Parameters:  map[string]any{"competitor": map[string]any{"type": "string"}},
		Handler: func(args, _ map[string]any) any {
			c, _ := args["competitor"].(string)
			return map[string]any{"response": c + " is $99/seat; we're $79."}
		},
	})
	payload := `{"function":"lookup_competitor","argument":{"parsed":[{"competitor":"ACME"}]},"call_id":"c-1"}`
	mux := svc.buildMux()
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, makeRequest(http.MethodPost, "/swaig", payload))
	if rr.Code != 200 {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "ACME") || !strings.Contains(rr.Body.String(), "$79") {
		t.Errorf("response body missing expected text: %s", rr.Body.String())
	}
}

func TestServiceSwaigPOSTMissingFunctionReturns400(t *testing.T) {
	svc := makeSwaigSvc(t)
	mux := svc.buildMux()
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, makeRequest(http.MethodPost, "/swaig", "{}"))
	if rr.Code != 400 {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestServiceSwaigPOSTInvalidFunctionNameReturns400(t *testing.T) {
	svc := makeSwaigSvc(t)
	mux := svc.buildMux()
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, makeRequest(http.MethodPost, "/swaig", `{"function":"../etc/passwd"}`))
	if rr.Code != 400 {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestServiceSwaigUnauthorizedReturns401(t *testing.T) {
	svc := makeSwaigSvc(t)
	mux := svc.buildMux()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/swaig", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != 401 {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

// ---- Sidecar pattern: non-agent SWML + tool registration -----------

func TestServiceSidecarPatternEmitsVerbAndRegistersTool(t *testing.T) {
	svc := NewService(WithName("sidecar"), WithRoute("/sidecar"), WithBasicAuth("u", "p"))

	// 1. Build the SWML — answer + ai_sidecar verb config. ai_sidecar isn't
	// in the schema yet, so add it directly to the document.
	svc.Answer(nil, nil)
	svc.GetDocument().AddVerbToSection("main", "ai_sidecar", map[string]any{
		"prompt":    "real-time copilot",
		"lang":      "en-US",
		"direction": []string{"remote-caller", "local-caller"},
	})

	doc := svc.GetDocument().ToMap()
	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)
	hasAnswer, hasSidecar := false, false
	for _, v := range main {
		if vm, ok := v.(map[string]any); ok {
			if _, found := vm["answer"]; found {
				hasAnswer = true
			}
			if _, found := vm["ai_sidecar"]; found {
				hasSidecar = true
			}
		}
	}
	if !hasAnswer || !hasSidecar {
		t.Errorf("missing verbs: answer=%v sidecar=%v in %v", hasAnswer, hasSidecar, main)
	}

	// 2. Register a SWAIG tool.
	svc.DefineTool(&ToolDefinition{
		Name:        "lookup_competitor",
		Description: "Look up competitor pricing.",
		Parameters:  map[string]any{"competitor": map[string]any{"type": "string"}},
		Handler: func(args, _ map[string]any) any {
			c, _ := args["competitor"].(string)
			return map[string]any{"response": "Pricing for " + c + ": $99"}
		},
	})

	// 3. Dispatch end-to-end.
	result := svc.OnFunctionCall(
		"lookup_competitor",
		map[string]any{"competitor": "ACME"},
		map[string]any{},
	)
	m, _ := result.(map[string]any)
	resp, _ := m["response"].(string)
	if !strings.Contains(resp, "ACME") {
		t.Errorf("expected ACME in response, got %q", resp)
	}
}
