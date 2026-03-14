package swaig

// ToolHandler is the function signature for SWAIG tool handlers.
// args contains the parsed function arguments, rawData contains the full
// request payload including global_data, call_id, etc.
type ToolHandler func(args map[string]any, rawData map[string]any) *FunctionResult
