package agent

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/signalwire/signalwire-go/v3/pkg/swaig"
)

// Integration tests for the typed swaig.Params builder used through a REAL
// agent: build the schema with the builder, register it via the real
// DefineTool, then (a) render real SWML and assert the parameters appear
// correctly in the generated SWAIG JSON, and (b) invoke the function through the
// real OnFunctionCall dispatch and assert behavior. No transport mocks.

// findSwaigFunctions walks a rendered SWML doc down to the AI verb's
// SWAIG.functions array, failing the test if the path is absent.
func findSwaigFunctions(t *testing.T, doc map[string]any) []map[string]any {
	t.Helper()
	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)
	for _, v := range main {
		vm, _ := v.(map[string]any)
		aiCfg, ok := vm["ai"].(map[string]any)
		if !ok {
			continue
		}
		swaigCfg, _ := aiCfg["SWAIG"].(map[string]any)
		if swaigCfg == nil {
			t.Fatal("AI verb present but SWAIG config missing")
		}
		fns, _ := swaigCfg["functions"].([]map[string]any)
		return fns
	}
	t.Fatal("AI verb not found in rendered SWML")
	return nil
}

// TestBuilderParams_RenderedIntoRealSWAIGJSON drives a real agent: a tool whose
// Parameters/Required come from swaig.Params is rendered through RenderSWML, and
// the emitted SWAIG function's "parameters" block must match the JSON-Schema
// envelope {type:object, properties:{...}, required:[...]} byte-for-byte.
func TestBuilderParams_RenderedIntoRealSWAIGJSON(t *testing.T) {
	params, required := swaig.NewParams().
		String("service", "The service to check (e.g., spa, restaurant)").
		String("date", "The date to check (YYYY-MM-DD format)").
		Enum("fmt", swaig.RecordFormatValues(), "Recording format").
		Required("service", "date").
		Build()

	a := NewAgentBase(WithName("builder-it"), WithBasicAuth("u", "p"))
	a.SetPromptText("hello")
	a.SetWebHookURL("https://example.com/swaig")
	a.DefineTool(ToolDefinition{
		Name:        "check_availability",
		Description: "Check availability for a service",
		Parameters:  params,
		Required:    required,
		Handler: func(args map[string]any, raw map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("checked " + as[string](t, args["service"]))
		},
	})

	fns := findSwaigFunctions(t, a.RenderSWML(nil, nil))
	if len(fns) != 1 {
		t.Fatalf("expected exactly 1 SWAIG function, got %d", len(fns))
	}
	fn := fns[0]

	if fn["function"] != "check_availability" {
		t.Errorf("function name = %v, want check_availability", fn["function"])
	}

	gotParams, ok := fn["parameters"].(map[string]any)
	if !ok {
		t.Fatalf("expected parameters object, got %#v", fn["parameters"])
	}

	// The full JSON-Schema envelope the AI verb must carry.
	wantParams := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"service": map[string]any{
				"type":        "string",
				"description": "The service to check (e.g., spa, restaurant)",
			},
			"date": map[string]any{
				"type":        "string",
				"description": "The date to check (YYYY-MM-DD format)",
			},
			"fmt": map[string]any{
				"type":        "string",
				"description": "Recording format",
				"enum":        []any{"mp3", "wav", "mp4"},
			},
		},
		"required": []string{"service", "date"},
	}

	// JSON-equal comparison (normalizes object key order, preserves array order).
	gb, err := json.Marshal(gotParams)
	if err != nil {
		t.Fatal(err)
	}
	wb, err := json.Marshal(wantParams)
	if err != nil {
		t.Fatal(err)
	}
	var gv, wv any
	_ = json.Unmarshal(gb, &gv)
	_ = json.Unmarshal(wb, &wv)
	if !reflect.DeepEqual(gv, wv) {
		t.Errorf("rendered SWAIG parameters mismatch:\n got = %s\nwant = %s", gb, wb)
	}

	// Spot-check the enum array actually rode through into the generated JSON.
	props := as[map[string]any](t, gotParams["properties"])
	fmtProp := as[map[string]any](t, props["fmt"])
	enum, ok := fmtProp["enum"].([]any)
	if !ok || len(enum) != 3 || enum[0] != "mp3" || enum[2] != "mp4" {
		t.Errorf("fmt enum not rendered correctly: %#v", fmtProp["enum"])
	}
}

// TestBuilderParams_EquivalentToHandWrittenInRender proves equivalence at the
// rendered-SWML layer: two agents — one tool built with the typed builder, one
// with the hand-written nested-map literal — emit identical "parameters" blocks.
// This is the end-to-end "same wire output" guarantee through the real
// renderer, not just at the builder boundary.
func TestBuilderParams_EquivalentToHandWrittenInRender(t *testing.T) {
	builderParams, builderReq := swaig.NewParams().
		String("question_id", "The ID of the question").
		String("response", "The user's response to validate").
		Required("question_id").
		Build()

	makeAgent := func(p map[string]any, req []string) *AgentBase {
		a := NewAgentBase(WithName("eq"), WithBasicAuth("u", "p"))
		a.SetPromptText("hi")
		a.SetWebHookURL("https://example.com/swaig")
		a.DefineTool(ToolDefinition{
			Name:        "validate_response",
			Description: "Validate a response",
			Parameters:  p,
			Required:    req,
			Handler: func(args map[string]any, raw map[string]any) *swaig.FunctionResult {
				return swaig.NewFunctionResult("ok")
			},
		})
		return a
	}

	// Hand-written form lifted from pkg/prefabs/survey.go (validate_response).
	handParams := map[string]any{
		"question_id": map[string]any{
			"type":        "string",
			"description": "The ID of the question",
		},
		"response": map[string]any{
			"type":        "string",
			"description": "The user's response to validate",
		},
	}
	handReq := []string{"question_id"}

	builtFn := findSwaigFunctions(t, makeAgent(builderParams, builderReq).RenderSWML(nil, nil))[0]
	handFn := findSwaigFunctions(t, makeAgent(handParams, handReq).RenderSWML(nil, nil))[0]

	bb, err := json.Marshal(builtFn["parameters"])
	if err != nil {
		t.Fatal(err)
	}
	hb, err := json.Marshal(handFn["parameters"])
	if err != nil {
		t.Fatal(err)
	}
	var bv, hv any
	_ = json.Unmarshal(bb, &bv)
	_ = json.Unmarshal(hb, &hv)
	if !reflect.DeepEqual(bv, hv) {
		t.Errorf("builder vs hand-written rendered parameters differ:\nbuilder = %s\n   hand = %s", bb, hb)
	}
}

// TestBuilderParams_RealInvoke proves a builder-built tool is fully wired: the
// real OnFunctionCall dispatch reaches the handler and the declared argument
// flows through. This pairs the wire-shape assertions above with a behavioral
// one (no-cheat: real behavior, content-shaped result).
func TestBuilderParams_RealInvoke(t *testing.T) {
	params, required := swaig.NewParams().
		String("service", "The service to check").
		Required("service").
		Build()

	a := NewAgentBase(WithName("invoke"))
	a.DefineTool(ToolDefinition{
		Name:        "check_availability",
		Description: "Check availability",
		Parameters:  params,
		Required:    required,
		Handler: func(args map[string]any, raw map[string]any) *swaig.FunctionResult {
			svc, _ := args["service"].(string)
			return swaig.NewFunctionResult("Yes, " + svc + " is available")
		},
	})

	// The declared-required property is registered on the tool.
	td := a.Function("check_availability")
	if td == nil {
		t.Fatal("tool not registered")
	}
	if !reflect.DeepEqual(td.Required, []string{"service"}) {
		t.Errorf("tool.Required = %#v, want [service]", td.Required)
	}
	if _, ok := td.Parameters["service"]; !ok {
		t.Errorf("tool.Parameters missing 'service': %#v", td.Parameters)
	}

	res, err := a.OnFunctionCall("check_availability", map[string]any{"service": "spa"}, nil)
	if err != nil {
		t.Fatalf("OnFunctionCall error: %v", err)
	}
	m, _ := res.(map[string]any)
	if got, _ := m["response"].(string); got != "Yes, spa is available" {
		t.Errorf("handler response = %q, want %q", got, "Yes, spa is available")
	}
}
