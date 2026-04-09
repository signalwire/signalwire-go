// Package datamap provides a fluent builder for SWAIG data_map configurations.
//
// DataMap tools execute on SignalWire's servers without requiring webhook
// endpoints. They support API calls, expression-based pattern matching,
// variable expansion, and array processing.
//
// Example usage:
//
//	dm := datamap.New("get_weather").
//		Purpose("Get current weather information").
//		Parameter("location", "string", "City name", true, nil).
//		Webhook("GET", "https://api.weather.com/v1/current?q=${location}", nil, "", false, nil).
//		Output(swaig.NewFunctionResult("Weather: ${response.current.condition.text}"))
package datamap

import (
	"strings"

	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// paramDef describes a single function parameter.
type paramDef struct {
	name      string
	paramType string
	desc      string
	required  bool
	enum      []string
}

// expressionDef describes a pattern-matching expression.
type expressionDef struct {
	testValue     string
	pattern       string
	output        *swaig.FunctionResult
	nomatchOutput *swaig.FunctionResult
}

// webhookDef describes a webhook API call configuration.
type webhookDef struct {
	method            string
	url               string
	headers           map[string]string
	formParam         string
	inputArgsAsParams bool
	requireArgs       []string
}

// DataMap is a fluent builder for SWAIG data_map function definitions.
// Data map tools execute on SignalWire servers without needing a webhook endpoint.
type DataMap struct {
	functionName      string
	description       string
	parameters        []paramDef
	expressions       []expressionDef
	webhookConfig     *webhookDef
	webhookExprs      []map[string]any
	bodyData          map[string]any
	paramsData        map[string]any
	foreachConfig     map[string]any
	outputResult      *swaig.FunctionResult
	fallbackResult    *swaig.FunctionResult
	errorKeysList     []string
	globalErrorKeysList []string

	// Internal: accumulated webhooks for multi-webhook support.
	// Each entry is built from webhookConfig + body/params/output/foreach/error_keys
	// when a new Webhook() call is made.
	webhooks []map[string]any
}

// New creates a new DataMap builder for a function with the given name.
func New(functionName string) *DataMap {
	return &DataMap{
		functionName: functionName,
	}
}

// Purpose sets the LLM-facing tool description — this is an alias for
// Description.
//
// The description string is rendered into the OpenAI tool schema
// "description" field on every LLM turn. The model reads it to decide
// WHEN to call this tool. It is PROMPT ENGINEERING, not developer
// documentation.
//
// A vague Purpose is the #1 cause of "the model has the right tool but
// doesn't call it" failures with data-map tools.
//
// Bad vs good:
//
//	BAD:  dm.Purpose("weather api")
//	GOOD: dm.Purpose("Get the current weather conditions and forecast " +
//	          "for a specific city. Use this whenever the user asks about " +
//	          "weather, temperature, rain, or similar conditions in a " +
//	          "named location.")
func (dm *DataMap) Purpose(description string) *DataMap {
	dm.description = description
	return dm
}

// Description sets the LLM-facing tool description.
//
// This string is read by the model to decide WHEN to call this tool.
// See Purpose for bad-vs-good examples. Alias for Purpose.
func (dm *DataMap) Description(description string) *DataMap {
	dm.description = description
	return dm
}

// Parameter adds a parameter definition — the `desc` is LLM-FACING.
//
// Each parameter description is rendered into the OpenAI tool schema
// under parameters.properties.<name>.description and sent to the
// model. The model uses it to decide HOW to fill in the argument from
// user speech. It is prompt engineering, not developer FYI.
//
// Bad vs good:
//
//	BAD:  dm.Parameter("city", "string", "the city", true, nil)
//	GOOD: dm.Parameter("city", "string",
//	          "The name of the city to get weather for, e.g. 'San Francisco'. "+
//	          "Ask the user if they did not provide one. Include the state "+
//	          "or country if the city name is ambiguous.",
//	          true, nil)
//
// The enum parameter can be nil if no enumeration constraint is needed.
func (dm *DataMap) Parameter(name, paramType, desc string, required bool, enum []string) *DataMap {
	dm.parameters = append(dm.parameters, paramDef{
		name:      name,
		paramType: paramType,
		desc:      desc,
		required:  required,
		enum:      enum,
	})
	return dm
}

// Expression adds a pattern-matching expression for expression-based responses.
// testValue is the template string to test (e.g., "${args.command}").
// pattern is the regex pattern to match against.
// output is the FunctionResult returned when the pattern matches.
// nomatchOutput is an optional FunctionResult returned when the pattern does not match (can be nil).
func (dm *DataMap) Expression(testValue, pattern string, output *swaig.FunctionResult, nomatchOutput *swaig.FunctionResult) *DataMap {
	dm.expressions = append(dm.expressions, expressionDef{
		testValue:     testValue,
		pattern:       pattern,
		output:        output,
		nomatchOutput: nomatchOutput,
	})
	return dm
}

// flushCurrentWebhook saves the current webhook and its associated config
// into the accumulated webhooks slice, then resets per-webhook state.
func (dm *DataMap) flushCurrentWebhook() {
	if dm.webhookConfig == nil {
		return
	}

	wh := map[string]any{
		"url":    dm.webhookConfig.url,
		"method": dm.webhookConfig.method,
	}
	if len(dm.webhookConfig.headers) > 0 {
		wh["headers"] = dm.webhookConfig.headers
	}
	if dm.webhookConfig.formParam != "" {
		wh["form_param"] = dm.webhookConfig.formParam
	}
	if dm.webhookConfig.inputArgsAsParams {
		wh["input_args_as_params"] = true
	}
	if len(dm.webhookConfig.requireArgs) > 0 {
		wh["require_args"] = dm.webhookConfig.requireArgs
	}
	if dm.bodyData != nil {
		wh["body"] = dm.bodyData
	}
	if dm.paramsData != nil {
		wh["params"] = dm.paramsData
	}
	if dm.foreachConfig != nil {
		wh["foreach"] = dm.foreachConfig
	}
	if dm.outputResult != nil {
		wh["output"] = dm.outputResult.ToMap()
	}
	if len(dm.errorKeysList) > 0 {
		wh["error_keys"] = dm.errorKeysList
	}
	if len(dm.webhookExprs) > 0 {
		wh["expressions"] = dm.webhookExprs
	}

	dm.webhooks = append(dm.webhooks, wh)

	// Reset per-webhook state
	dm.webhookConfig = nil
	dm.bodyData = nil
	dm.paramsData = nil
	dm.foreachConfig = nil
	dm.outputResult = nil
	dm.errorKeysList = nil
	dm.webhookExprs = nil
}

// Webhook adds a webhook API call configuration.
// If a previous webhook was configured, it is finalized first.
// method is the HTTP method (GET, POST, etc.).
// url is the API endpoint URL (can include ${variable} substitutions).
// headers are optional HTTP headers (can be nil).
// formParam sends JSON body as a single form parameter with this name (empty string to skip).
// inputArgsAsParams merges function arguments into params.
// requireArgs lists arguments that must be present to execute (can be nil).
func (dm *DataMap) Webhook(method, url string, headers map[string]string, formParam string, inputArgsAsParams bool, requireArgs []string) *DataMap {
	// Flush any previous webhook
	dm.flushCurrentWebhook()

	dm.webhookConfig = &webhookDef{
		method:            strings.ToUpper(method),
		url:               url,
		headers:           headers,
		formParam:         formParam,
		inputArgsAsParams: inputArgsAsParams,
		requireArgs:       requireArgs,
	}
	return dm
}

// WebhookExpressions sets expressions to evaluate after the current webhook completes.
func (dm *DataMap) WebhookExpressions(expressions []map[string]any) *DataMap {
	dm.webhookExprs = expressions
	return dm
}

// Body sets the request body for the current webhook (for POST/PUT requests).
func (dm *DataMap) Body(data map[string]any) *DataMap {
	dm.bodyData = data
	return dm
}

// Params sets the request params for the current webhook.
func (dm *DataMap) Params(data map[string]any) *DataMap {
	dm.paramsData = data
	return dm
}

// Foreach configures array processing for the current webhook response.
func (dm *DataMap) Foreach(config map[string]any) *DataMap {
	dm.foreachConfig = config
	return dm
}

// Output sets the output result for the current webhook.
func (dm *DataMap) Output(result *swaig.FunctionResult) *DataMap {
	dm.outputResult = result
	return dm
}

// FallbackOutput sets the fallback output result used when all webhooks fail.
func (dm *DataMap) FallbackOutput(result *swaig.FunctionResult) *DataMap {
	dm.fallbackResult = result
	return dm
}

// ErrorKeys sets error indicator keys for the current webhook.
func (dm *DataMap) ErrorKeys(keys []string) *DataMap {
	dm.errorKeysList = keys
	return dm
}

// GlobalErrorKeys sets top-level error keys that apply to all webhooks.
func (dm *DataMap) GlobalErrorKeys(keys []string) *DataMap {
	dm.globalErrorKeysList = keys
	return dm
}

// ToSwaigFunction converts the DataMap to a complete SWAIG function definition map.
// The returned map contains "function", "purpose", "argument", and "data_map" keys.
func (dm *DataMap) ToSwaigFunction() map[string]any {
	// Build parameter schema
	properties := make(map[string]any)
	var requiredParams []string

	for _, p := range dm.parameters {
		propDef := map[string]any{
			"type":        p.paramType,
			"description": p.desc,
		}
		if len(p.enum) > 0 {
			propDef["enum"] = p.enum
		}
		properties[p.name] = propDef
		if p.required {
			requiredParams = append(requiredParams, p.name)
		}
	}

	argument := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(requiredParams) > 0 {
		argument["required"] = requiredParams
	}

	// Build data_map
	dataMap := make(map[string]any)

	// Add expressions
	if len(dm.expressions) > 0 {
		exprList := make([]map[string]any, 0, len(dm.expressions))
		for _, expr := range dm.expressions {
			entry := map[string]any{
				"string":  expr.testValue,
				"pattern": expr.pattern,
				"output":  expr.output.ToMap(),
			}
			if expr.nomatchOutput != nil {
				entry["nomatch-output"] = expr.nomatchOutput.ToMap()
			}
			exprList = append(exprList, entry)
		}
		dataMap["expressions"] = exprList
	}

	// Flush any pending webhook before serializing
	dm.flushCurrentWebhook()

	// Add webhooks
	if len(dm.webhooks) > 0 {
		dataMap["webhooks"] = dm.webhooks
	}

	// Add fallback output
	if dm.fallbackResult != nil {
		dataMap["output"] = dm.fallbackResult.ToMap()
	}

	// Add global error keys
	if len(dm.globalErrorKeysList) > 0 {
		dataMap["error_keys"] = dm.globalErrorKeysList
	}

	// Build the description, falling back to a generated one
	desc := dm.description
	if desc == "" {
		desc = "Execute " + dm.functionName
	}

	return map[string]any{
		"function": dm.functionName,
		"purpose":  desc,
		"argument": argument,
		"data_map": dataMap,
	}
}

// CreateSimpleApiTool creates a DataMap configured for a simple API call.
// name is the function name.
// url is the API endpoint URL.
// responseTemplate is the template for formatting the response.
// parameters maps parameter names to their definitions (each with "type", "description", "required" keys).
// method is the HTTP method (e.g., "GET", "POST").
// headers are optional HTTP headers (can be nil).
// body is an optional request body for POST/PUT (can be nil).
// errorKeys are optional error indicator keys (can be nil).
func CreateSimpleApiTool(name, url, responseTemplate string, parameters map[string]map[string]any, method string, headers map[string]string, body map[string]any, errorKeys []string) *DataMap {
	dm := New(name)

	// Add parameters
	for paramName, paramDef := range parameters {
		paramType, _ := paramDef["type"].(string)
		if paramType == "" {
			paramType = "string"
		}
		desc, _ := paramDef["description"].(string)
		if desc == "" {
			desc = paramName + " parameter"
		}
		required, _ := paramDef["required"].(bool)

		dm.Parameter(paramName, paramType, desc, required, nil)
	}

	// Add webhook
	dm.Webhook(method, url, headers, "", false, nil)

	// Add body if provided
	if body != nil {
		dm.Body(body)
	}

	// Add error keys if provided
	if len(errorKeys) > 0 {
		dm.ErrorKeys(errorKeys)
	}

	// Set output
	dm.Output(swaig.NewFunctionResult(responseTemplate))

	return dm
}

// CreateExpressionTool creates a DataMap configured for expression-based pattern matching.
// name is the function name.
// patterns maps test values to a two-element array where [0] is the pattern string
// and [1] is a *swaig.FunctionResult.
// parameters maps parameter names to their definitions (each with "type", "description", "required" keys).
func CreateExpressionTool(name string, patterns map[string][2]any, parameters map[string]map[string]any) *DataMap {
	dm := New(name)

	// Add parameters
	for paramName, paramDef := range parameters {
		paramType, _ := paramDef["type"].(string)
		if paramType == "" {
			paramType = "string"
		}
		desc, _ := paramDef["description"].(string)
		if desc == "" {
			desc = paramName + " parameter"
		}
		required, _ := paramDef["required"].(bool)

		dm.Parameter(paramName, paramType, desc, required, nil)
	}

	// Add expressions
	for testValue, patternPair := range patterns {
		patternStr, _ := patternPair[0].(string)
		result, _ := patternPair[1].(*swaig.FunctionResult)
		if result != nil {
			dm.Expression(testValue, patternStr, result, nil)
		}
	}

	return dm
}
