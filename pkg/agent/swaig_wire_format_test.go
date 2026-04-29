// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package agent

import (
	"testing"

	_ "github.com/signalwire/signalwire-go/pkg/skills/builtin"
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

	fns := a.buildSwaigFunctions("http://example.com/swaig")
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

	fn := a.buildSwaigFunctions("http://example.com/swaig")[0]
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

	fn := a.buildSwaigFunctions("http://example.com/swaig")[0]
	if _, present := fn["parameters"]; present {
		t.Errorf(`no-param tool should omit "parameters", got %#v`, fn)
	}
	if _, present := fn["argument"]; present {
		t.Errorf(`no-param tool must not emit deprecated "argument", got %#v`, fn)
	}
}
