// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package swaig

import (
	"reflect"
	"testing"
)

// TestInferSchemaFromTypedHandler builds a SWAIG parameter schema from a typed
// Params declaration (the Go analog of Python infer_schema reflecting a typed
// handler) and checks the full 5-tuple: parameters, required, description,
// isTyped, hasRawData.
func TestInferSchemaFromTypedHandler(t *testing.T) {
	p := NewParams().
		Describe("Check the weather for a city on a date").
		WithRawData().
		String("city", "The city to check").
		Enum("fmt", RecordFormatValues(), "Recording format").
		Integer("days", "How many days ahead").
		Required("city", "fmt")

	params, required, description, isTyped, hasRawData := InferSchema(p)

	// parameters: one JSON-Schema property object per declared param.
	if got := params["city"]["type"]; got != "string" {
		t.Errorf("city.type = %v, want string", got)
	}
	if got := params["days"]["type"]; got != "integer" {
		t.Errorf("days.type = %v, want integer", got)
	}
	enum, ok := params["fmt"]["enum"].([]any)
	if !ok || len(enum) != 3 {
		t.Errorf("fmt.enum = %v, want the 3 record-format values", params["fmt"]["enum"])
	}

	if want := []string{"city", "fmt"}; !reflect.DeepEqual(required, want) {
		t.Errorf("required = %v, want %v", required, want)
	}
	if description == nil || *description != "Check the weather for a city on a date" {
		t.Errorf("description = %v, want the declared summary", description)
	}
	if !isTyped {
		t.Error("isTyped = false, want true for a Params declaration")
	}
	if !hasRawData {
		t.Error("hasRawData = false, want true (declared via WithRawData)")
	}
}

// TestInferSchemaZeroParam covers the zero-param typed-tool path: nil / empty
// builder yields an empty parameter set but is still a typed tool.
func TestInferSchemaZeroParam(t *testing.T) {
	params, required, description, isTyped, hasRawData := InferSchema(nil)
	if len(params) != 0 {
		t.Errorf("params = %v, want empty", params)
	}
	if required != nil {
		t.Errorf("required = %v, want nil", required)
	}
	if description != nil {
		t.Errorf("description = %v, want nil", description)
	}
	if !isTyped {
		t.Error("isTyped = false, want true (zero-param typed tool)")
	}
	if hasRawData {
		t.Error("hasRawData = true, want false")
	}
}

// TestCreateTypedHandlerWrapper adapts a typed handler to the standard
// ToolHandler calling convention, threading raw_data only when declared.
func TestCreateTypedHandlerWrapper(t *testing.T) {
	var sawRaw map[string]any
	typed := func(args map[string]any, rawData map[string]any) *FunctionResult {
		sawRaw = rawData
		return NewFunctionResult("ok")
	}

	// hasRawData=true: rawData threaded through.
	raw := map[string]any{"call_id": "x"}
	w := CreateTypedHandlerWrapper(typed, true)
	_ = w(map[string]any{"a": 1}, raw)
	if sawRaw == nil || sawRaw["call_id"] != "x" {
		t.Errorf("raw_data not threaded through when hasRawData=true: %v", sawRaw)
	}

	// hasRawData=false: handler receives nil raw_data.
	sawRaw = map[string]any{"stale": true}
	w = CreateTypedHandlerWrapper(typed, false)
	_ = w(map[string]any{"a": 1}, raw)
	if sawRaw != nil {
		t.Errorf("raw_data should be nil when hasRawData=false, got %v", sawRaw)
	}
}
