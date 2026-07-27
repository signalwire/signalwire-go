// Regression guard for task #58: swml.ToolDefinition deliberately has NO
// Secure flag.
//
// The reference's per-tool `secure` flag gates SWAIG *token validation*, and
// that machinery (the SessionManager that mints a per-(tool, call) HMAC token
// and the validator that checks it) lives entirely on AgentBase — reference
// signalwire/core/agent_base.py + core/mixins/state_mixin.py; Go pkg/agent,
// agent.ToolDefinition.Secure (a tri-state *bool defaulting to SECURE). A bare
// SWMLService owns no session manager and mints no tokens, so a secure flag at
// this level has nothing to gate. The reference's SWMLService has no tool
// registry at all.
//
// `secure` is likewise not a SWML/SWAIG wire field: it appears in neither the
// vendored mod_openai specs (porting-sdk/swaig-specs/) nor the SWML schema. A
// previous revision declared `Secure bool` on this struct with zero readers, and
// two shipped examples set it believing it did something. These tests assert the
// wire truth so the dead field cannot come back unnoticed: nothing named
// "secure" reaches either the list-tools introspection payload or the rendered
// SWML document.

package swml

import (
	"encoding/json"
	"strings"
	"testing"
)

// swmlToolSecureFixture registers a tool on a bare SWMLService with an <ai>
// verb, so both the introspection payload and the rendered document have a
// SWAIG function to describe.
func swmlToolSecureFixture(t *testing.T) *Service {
	t.Helper()
	svc := NewService(WithName("secure-probe"), WithBasicAuth("u", "p"))
	svc.DefineTool(&ToolDefinition{
		Name:        "lookup_competitor",
		Description: "Look up competitor pricing by company name.",
		Parameters: map[string]any{
			"competitor": map[string]any{
				"type":        "string",
				"description": "The competitor's company name.",
			},
		},
		Handler: func(_, _ map[string]any) any {
			return map[string]any{"response": "ok"}
		},
	})
	return svc
}

// TestSwmlToolDefinition_ListToolsPayloadHasNoSecureKey proves the SWAIG
// introspection payload (SWAIG_LIST_TOOLS / `swaig-test --example`) emits only
// function/description/parameters — no per-tool secure flag.
func TestSwmlToolDefinition_ListToolsPayloadHasNoSecureKey(t *testing.T) {
	svc := swmlToolSecureFixture(t)

	raw, err := svc.BuildSwaigListToolsPayload()
	if err != nil {
		t.Fatalf("BuildSwaigListToolsPayload err: %v", err)
	}
	if strings.Contains(strings.ToLower(string(raw)), "secure") {
		t.Errorf("list-tools payload leaked a secure flag; swml.ToolDefinition has no\n"+
			"Secure field and a bare SWMLService mints no tokens. payload=%s", raw)
	}

	var out struct {
		Tools []map[string]any `json:"tools"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(out.Tools))
	}
	for key := range out.Tools[0] {
		switch key {
		case "function", "description", "parameters":
			// The three keys BuildSwaigListToolsPayload documents.
		default:
			t.Errorf("unexpected key %q in list-tools entry; want only "+
				"function/description/parameters", key)
		}
	}
}

// TestSwmlToolDefinition_RenderedDocumentHasNoSecureKey proves the rendered SWML
// document carries no per-tool secure flag either — `secure` is an SDK-side
// token-gating concept, never a wire field.
func TestSwmlToolDefinition_RenderedDocumentHasNoSecureKey(t *testing.T) {
	svc := swmlToolSecureFixture(t)

	rendered, err := svc.Render()
	if err != nil {
		t.Fatalf("Render err: %v", err)
	}
	if strings.Contains(strings.ToLower(rendered), "secure") {
		t.Errorf("rendered SWML leaked a secure flag; `secure` is not a SWML/SWAIG\n"+
			"wire field (absent from swaig-specs/ and the SWML schema). doc=%s", rendered)
	}
}
