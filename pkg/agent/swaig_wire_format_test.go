// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package agent

import (
	"strings"
	"testing"

	_ "github.com/signalwire/signalwire-go/v3/pkg/skills/all"
	"github.com/signalwire/signalwire-go/v3/pkg/swaig"
)

// Tests guarding the SWAIG wire-format key renames this PR adopted
// (purpose → description, argument → parameters). Python emits the new
// keys; the old ones are schema-deprecated. Equivalent Python coverage
// lives in tests/unit/core/test_swml_renderer.py.

func TestBuildSwaigFunctionsUsesDescriptionNotPurpose(t *testing.T) {
	a := NewAgentBase(WithName("wire-test"))
	a.SetPromptText("hi")
	a.DefineTool(ToolDefinition{
		Name:        "lookup",
		Description: "Look up account info",
		Parameters: map[string]any{
			"id": map[string]any{"type": "string"},
		},
	})

	fns := a.buildSwaigFunctions("http://example.com/swaig", "")
	if len(fns) != 1 {
		t.Fatalf("expected 1 SWAIG function, got %d", len(fns))
	}
	fn := fns[0]

	if fn["description"] != "Look up account info" {
		t.Errorf(`expected description="Look up account info", got %v`, fn["description"])
	}
	if _, hasPurpose := fn["purpose"]; hasPurpose {
		t.Errorf(`emitted SWAIG function should not include the deprecated "purpose" key, got %#v`, fn)
	}
}

func TestBuildSwaigFunctionsUsesParametersNotArgument(t *testing.T) {
	a := NewAgentBase(WithName("wire-test"))
	a.SetPromptText("hi")
	a.DefineTool(ToolDefinition{
		Name:        "lookup",
		Description: "d",
		Parameters: map[string]any{
			"id": map[string]any{"type": "string"},
		},
	})

	fn := a.buildSwaigFunctions("http://example.com/swaig", "")[0]
	params, ok := fn["parameters"].(map[string]any)
	if !ok {
		t.Fatalf(`expected "parameters" key with object value, got %#v`, fn["parameters"])
	}
	if params["type"] != "object" {
		t.Errorf(`parameters.type = %v, want "object"`, params["type"])
	}
	if _, hasArgument := fn["argument"]; hasArgument {
		t.Errorf(`emitted SWAIG function should not include the deprecated "argument" key, got %#v`, fn)
	}
}

func TestBuildSwaigFunctionsOmitsParametersWhenNone(t *testing.T) {
	// A tool with no declared parameters should leave "parameters" absent
	// (not emit an empty object under the deprecated "argument" key either).
	a := NewAgentBase(WithName("wire-test"))
	a.SetPromptText("hi")
	a.DefineTool(ToolDefinition{Name: "no_args", Description: "d"})

	fn := a.buildSwaigFunctions("http://example.com/swaig", "")[0]
	if _, present := fn["parameters"]; present {
		t.Errorf(`no-param tool should omit "parameters", got %#v`, fn)
	}
	if _, present := fn["argument"]; present {
		t.Errorf(`no-param tool must not emit deprecated "argument", got %#v`, fn)
	}
}

// renderSecureFixtureSWAIG renders one agent carrying BOTH a default (secure)
// tool and an explicit secure=false tool with a live call_id, and returns the
// rendered SWAIG object.
func renderSecureFixtureSWAIG(t *testing.T) map[string]any {
	t.Helper()

	a := NewAgentBase(
		WithName("secure-webhook-test"),
		WithRoute("/sd"),
		WithBasicAuth("u", "p"),
	)
	a.SetPromptText("hi")

	handler := func(map[string]any, map[string]any) *swaig.FunctionResult {
		return swaig.NewFunctionResult("ok")
	}
	// No explicit Secure -> defaults to SECURE.
	a.DefineTool(ToolDefinition{
		Name: "sd_default_secure", Description: "d",
		Parameters: map[string]any{}, Handler: handler,
	})
	insecure := false
	a.DefineTool(ToolDefinition{
		Name: "sd_explicit_insecure", Description: "d",
		Parameters: map[string]any{}, Handler: handler, Secure: &insecure,
	})

	doc := a.RenderSWMLForCall(nil, nil, "call-secure-default-fixture")
	sections, ok := doc["sections"].(map[string]any)
	if !ok {
		t.Fatalf("rendered SWML has no sections: %#v", doc)
	}
	main, ok := sections["main"].([]any)
	if !ok {
		t.Fatalf("rendered SWML has no sections.main array: %#v", sections)
	}
	for _, raw := range main {
		verb, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		ai, ok := verb["ai"].(map[string]any)
		if !ok {
			continue
		}
		sw, ok := ai["SWAIG"].(map[string]any)
		if !ok {
			t.Fatalf("ai verb has no SWAIG object: %#v", ai)
		}
		return sw
	}
	t.Fatalf("no ai verb in rendered SWML: %#v", main)
	return nil
}

// swaigFunctionsByName indexes a rendered SWAIG object's functions[] by name.
func swaigFunctionsByName(t *testing.T, sw map[string]any) map[string]map[string]any {
	t.Helper()
	fns, ok := sw["functions"].([]map[string]any)
	if !ok {
		t.Fatalf("SWAIG.functions is not a function array: %#v", sw["functions"])
	}
	out := make(map[string]map[string]any, len(fns))
	for _, fn := range fns {
		name, _ := fn["function"].(string)
		out[name] = fn
	}
	return out
}

// TestSecureToolCarriesTokenizedWebhook pins the SECURE half of the reference's
// three-way webhook branch (agent_base.py:1089-1099): a secure tool rendered
// inside a live call gets its OWN web_hook_url carrying the reserved __token
// query parameter.
func TestSecureToolCarriesTokenizedWebhook(t *testing.T) {
	byName := swaigFunctionsByName(t, renderSecureFixtureSWAIG(t))

	fn, ok := byName["sd_default_secure"]
	if !ok {
		t.Fatalf("secure tool absent from rendered functions: %#v", byName)
	}
	url, ok := fn["web_hook_url"].(string)
	if !ok || url == "" {
		t.Fatalf("secure tool must have its own web_hook_url, got %#v", fn["web_hook_url"])
	}
	if !strings.Contains(url, "__token=") {
		t.Errorf("secure tool's web_hook_url must carry __token, got %q", url)
	}
}

// TestInsecureToolHasNoOwnWebhookURL is the SECURITY assertion: a secure=false
// tool has NO token, so per the reference it must get NO per-tool web_hook_url
// KEY AT ALL — not an empty string, not a null, and above all not a tokenless
// URL, which would be an unauthenticated function-specific callback on the wire.
// It falls back to the shared SWAIG.defaults.web_hook_url instead.
func TestInsecureToolHasNoOwnWebhookURL(t *testing.T) {
	sw := renderSecureFixtureSWAIG(t)
	byName := swaigFunctionsByName(t, sw)

	fn, ok := byName["sd_explicit_insecure"]
	if !ok {
		t.Fatalf("insecure tool absent from rendered functions: %#v", byName)
	}
	if v, present := fn["web_hook_url"]; present {
		t.Errorf("insecure tool must have NO web_hook_url key at all, got %#v "+
			"(an unauthenticated per-function callback)", v)
	}

	// The fallback the insecure tool depends on must actually exist.
	defaults, ok := sw["defaults"].(map[string]any)
	if !ok {
		t.Fatalf("SWAIG.defaults must be emitted as the shared fallback endpoint, got %#v", sw["defaults"])
	}
	if url, _ := defaults["web_hook_url"].(string); url == "" {
		t.Errorf("SWAIG.defaults.web_hook_url must be a non-empty URL, got %#v", defaults["web_hook_url"])
	}
}

// TestInsecureToolGetsLocalWebhookWhenQueryParamsSet pins the middle branch:
// with SWAIG query params configured, even a tokenless tool DOES get the local
// URL (reference: “elif token or agent._swaig_query_params“).
func TestInsecureToolGetsLocalWebhookWhenQueryParamsSet(t *testing.T) {
	a := NewAgentBase(WithName("qp-test"), WithRoute("/sd"), WithBasicAuth("u", "p"))
	a.SetPromptText("hi")
	a.AddSwaigQueryParams(map[string]string{"tenant": "acme"})

	insecure := false
	a.DefineTool(ToolDefinition{
		Name: "qp_insecure", Description: "d", Secure: &insecure,
		Handler: func(map[string]any, map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("ok")
		},
	})

	fn := a.buildSwaigFunctions(a.buildWebhookURL(), "call-x")[0]
	url, ok := fn["web_hook_url"].(string)
	if !ok || !strings.Contains(url, "tenant=acme") {
		t.Errorf("insecure tool with swaig query params must get the local URL carrying them, got %#v",
			fn["web_hook_url"])
	}
	if strings.Contains(url, "__token=") {
		t.Errorf("insecure tool must never carry a __token, got %q", url)
	}
}

// TestExternalWebhookURLWinsOverToken pins the first branch: an explicit
// per-tool webhook URL is used verbatim and never has our token appended (it is
// the platform's own endpoint).
func TestExternalWebhookURLWinsOverToken(t *testing.T) {
	a := NewAgentBase(WithName("ext-test"), WithRoute("/sd"), WithBasicAuth("u", "p"))
	a.SetPromptText("hi")
	a.DefineTool(ToolDefinition{
		Name: "ext", Description: "d", WebhookURL: "https://external.example/hook",
		Handler: func(map[string]any, map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("ok")
		},
	})

	fn := a.buildSwaigFunctions(a.buildWebhookURL(), "call-x")[0]
	if got := fn["web_hook_url"]; got != "https://external.example/hook" {
		t.Errorf("external webhook must be used verbatim, got %#v", got)
	}
}
