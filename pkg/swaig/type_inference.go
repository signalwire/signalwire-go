// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package swaig

// type_inference.go — the Go analog of the Python reference module
// signalwire.core.agent.tools.type_inference (infer_schema +
// create_typed_handler_wrapper).
//
// IDIOM. Python's infer_schema reflects a handler function's signature and type
// hints at runtime (inspect.signature / typing.get_type_hints) to derive a
// JSON-Schema parameter object. Go erases parameter names and types at compile
// time and carries no runtime signature reflection over an arbitrary func value,
// so — like every statically-typed port — the typed declaration is supplied
// explicitly, here via the fluent Params builder. The builder IS the typed
// declaration; InferSchema renders it into the exact same 5-tuple Python's
// infer_schema returns: (parameters, required, description, isTyped, hasRawData).
//
// This is why the two symbols are NOT omitted as impossible: the reference role
// (turn a typed handler declaration into a SWAIG parameter schema + a wrapper
// that adapts it to the standard calling convention) is fully realized, just
// driven from a typed params-builder rather than runtime reflection. The
// enumerators project InferSchema / CreateTypedHandlerWrapper onto the reference
// module-level names via FreeFnTable.

// InferSchema renders a SWAIG tool parameter schema from a typed declaration
// (the Params builder), returning the same 5-tuple as the Python reference
// infer_schema:
//
//   - parameters:  the JSON-Schema properties map (name -> property object);
//   - required:    the top-level required property names, in first-seen order;
//   - description: the tool-level description (nil when none was declared);
//   - isTyped:     true — a Params declaration is always a typed tool;
//   - hasRawData:  whether the handler also receives the SWAIG raw payload
//     (declared via Params.WithRawData()).
//
// A nil builder yields the empty zero-param typed schema (isTyped true), matching
// the reference's zero-param typed-tool path.
func InferSchema(p *Params) (map[string]map[string]any, []string, *string, bool, bool) {
	if p == nil {
		return map[string]map[string]any{}, nil, nil, true, false
	}

	// The reference return type is dict[str, dict[str, Any]] — each property is a
	// JSON-Schema object. Params stores exactly that; re-shape Properties()'s
	// map[string]any (whose values are the property objects) to the typed form.
	props := p.Properties()
	parameters := make(map[string]map[string]any, len(props))
	for name, schema := range props {
		if obj, ok := schema.(map[string]any); ok {
			parameters[name] = obj
		} else {
			parameters[name] = map[string]any{"value": schema}
		}
	}

	var description *string
	if p.description != "" {
		d := p.description
		description = &d
	}

	return parameters, p.requiredNames(), description, true, p.hasRawData
}

// TypedHandler is a SWAIG tool handler expressed over already-parsed named
// arguments (the Go analog of the reference's typed handler that declares its
// parameters by name). CreateTypedHandlerWrapper adapts one to the standard
// ToolHandler calling convention.
type TypedHandler func(args map[string]any, rawData map[string]any) *FunctionResult

// CreateTypedHandlerWrapper wraps a typed handler so it can be invoked with the
// standard SWAIG calling convention (args, rawData), mirroring the reference
// create_typed_handler_wrapper. When hasRawData is false the raw payload is not
// threaded through (the wrapped handler receives a nil rawData), matching the
// reference wrapper which passes raw_data only when the handler declared it.
func CreateTypedHandlerWrapper(fn TypedHandler, hasRawData bool) ToolHandler {
	return func(args map[string]any, rawData map[string]any) *FunctionResult {
		if hasRawData {
			return fn(args, rawData)
		}
		return fn(args, nil)
	}
}
