package datamap

import (
	"testing"

	"github.com/signalwire/signalwire-go/pkg/swaig"
)

func TestNew(t *testing.T) {
	dm := New("test_function")
	if dm == nil {
		t.Fatal("New returned nil")
	}
	if dm.functionName != "test_function" {
		t.Errorf("expected functionName %q, got %q", "test_function", dm.functionName)
	}
	if dm.description != "" {
		t.Errorf("expected empty description, got %q", dm.description)
	}
	if len(dm.parameters) != 0 {
		t.Errorf("expected 0 parameters, got %d", len(dm.parameters))
	}
}

func TestFluentBuilderChain(t *testing.T) {
	dm := New("weather").
		Purpose("Get weather information").
		Parameter("city", "string", "City name", true, nil).
		Parameter("units", "string", "Temperature units", false, []string{"celsius", "fahrenheit"}).
		Webhook("GET", "https://api.weather.com/v1?q=${city}&units=${units}", map[string]string{
			"Authorization": "Bearer token123",
		}, "", false, nil).
		Output(swaig.NewFunctionResult("Weather in ${city}: ${response.temp}"))

	if dm.functionName != "weather" {
		t.Errorf("expected functionName %q, got %q", "weather", dm.functionName)
	}
	if dm.description != "Get weather information" {
		t.Errorf("expected description %q, got %q", "Get weather information", dm.description)
	}
	if len(dm.parameters) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(dm.parameters))
	}
	if dm.webhookConfig == nil {
		t.Fatal("expected webhookConfig to be set")
	}
	if dm.webhookConfig.method != "GET" {
		t.Errorf("expected method GET, got %q", dm.webhookConfig.method)
	}
	if dm.outputResult == nil {
		t.Fatal("expected outputResult to be set")
	}
}

func TestDescriptionAlias(t *testing.T) {
	dm1 := New("f1").Purpose("desc1")
	dm2 := New("f2").Description("desc2")

	if dm1.description != "desc1" {
		t.Errorf("Purpose did not set description: got %q", dm1.description)
	}
	if dm2.description != "desc2" {
		t.Errorf("Description did not set description: got %q", dm2.description)
	}
}

func TestToSwaigFunctionBasic(t *testing.T) {
	dm := New("greet").
		Purpose("Greet user").
		Parameter("name", "string", "User name", true, nil).
		Webhook("GET", "https://example.com/greet?name=${name}", nil, "", false, nil).
		Output(swaig.NewFunctionResult("Hello ${name}!"))

	result := dm.ToSwaigFunction()

	// Check top-level keys
	if result["function"] != "greet" {
		t.Errorf("expected function %q, got %v", "greet", result["function"])
	}
	if result["description"] != "Greet user" {
		t.Errorf("expected description %q, got %v", "Greet user", result["description"])
	}

	// Check parameters schema
	argument, ok := result["parameters"].(map[string]any)
	if !ok {
		t.Fatal("expected parameters to be map[string]any")
	}
	if argument["type"] != "object" {
		t.Errorf("expected parameters type %q, got %v", "object", argument["type"])
	}

	properties, ok := argument["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties to be map[string]any")
	}
	nameProp, ok := properties["name"].(map[string]any)
	if !ok {
		t.Fatal("expected name property to be map[string]any")
	}
	if nameProp["type"] != "string" {
		t.Errorf("expected name type %q, got %v", "string", nameProp["type"])
	}

	// Check required
	required, ok := argument["required"].([]string)
	if !ok {
		t.Fatal("expected required to be []string")
	}
	if len(required) != 1 || required[0] != "name" {
		t.Errorf("expected required [name], got %v", required)
	}

	// Check data_map has webhooks
	dataMap, ok := result["data_map"].(map[string]any)
	if !ok {
		t.Fatal("expected data_map to be map[string]any")
	}
	webhooks, ok := dataMap["webhooks"].([]map[string]any)
	if !ok {
		t.Fatal("expected webhooks to be []map[string]any")
	}
	if len(webhooks) != 1 {
		t.Errorf("expected 1 webhook, got %d", len(webhooks))
	}
	if webhooks[0]["method"] != "GET" {
		t.Errorf("expected webhook method GET, got %v", webhooks[0]["method"])
	}
	if webhooks[0]["url"] != "https://example.com/greet?name=${name}" {
		t.Errorf("unexpected webhook url: %v", webhooks[0]["url"])
	}

	// Check output inside webhook
	output, ok := webhooks[0]["output"].(map[string]any)
	if !ok {
		t.Fatal("expected webhook output to be map[string]any")
	}
	if output["response"] != "Hello ${name}!" {
		t.Errorf("expected output response %q, got %v", "Hello ${name}!", output["response"])
	}
}

func TestToSwaigFunctionDefaultDescription(t *testing.T) {
	dm := New("my_tool")
	result := dm.ToSwaigFunction()

	if result["description"] != "Execute my_tool" {
		t.Errorf("expected default description %q, got %v", "Execute my_tool", result["description"])
	}
}

func TestParametersWithEnum(t *testing.T) {
	dm := New("set_mode").
		Purpose("Set operating mode").
		Parameter("mode", "string", "Operating mode", true, []string{"fast", "slow", "auto"})

	result := dm.ToSwaigFunction()

	argument := result["parameters"].(map[string]any)
	properties := argument["properties"].(map[string]any)
	modeProp := properties["mode"].(map[string]any)

	enumVal, ok := modeProp["enum"].([]string)
	if !ok {
		t.Fatal("expected enum to be []string")
	}
	if len(enumVal) != 3 {
		t.Errorf("expected 3 enum values, got %d", len(enumVal))
	}
	expected := []string{"fast", "slow", "auto"}
	for i, v := range expected {
		if enumVal[i] != v {
			t.Errorf("expected enum[%d] = %q, got %q", i, v, enumVal[i])
		}
	}
}

func TestParametersNoEnum(t *testing.T) {
	dm := New("lookup").
		Parameter("query", "string", "Search query", false, nil)

	result := dm.ToSwaigFunction()

	argument := result["parameters"].(map[string]any)
	properties := argument["properties"].(map[string]any)
	queryProp := properties["query"].(map[string]any)

	if _, exists := queryProp["enum"]; exists {
		t.Error("expected no enum key when enum is nil")
	}

	// Should not have required key when no params are required
	if _, exists := argument["required"]; exists {
		t.Error("expected no required key when no params are required")
	}
}

func TestWebhookConfiguration(t *testing.T) {
	dm := New("api_call").
		Webhook("POST", "https://api.example.com/data",
			map[string]string{"Content-Type": "application/json", "Authorization": "Bearer tok"},
			"json_data",
			true,
			[]string{"id", "name"},
		).
		Body(map[string]any{
			"id":   "${args.id}",
			"name": "${args.name}",
		}).
		Output(swaig.NewFunctionResult("Done: ${response.status}"))

	result := dm.ToSwaigFunction()
	dataMap := result["data_map"].(map[string]any)
	webhooks := dataMap["webhooks"].([]map[string]any)

	if len(webhooks) != 1 {
		t.Fatalf("expected 1 webhook, got %d", len(webhooks))
	}

	wh := webhooks[0]

	if wh["method"] != "POST" {
		t.Errorf("expected method POST, got %v", wh["method"])
	}
	if wh["url"] != "https://api.example.com/data" {
		t.Errorf("unexpected url: %v", wh["url"])
	}

	headers := wh["headers"].(map[string]string)
	if headers["Authorization"] != "Bearer tok" {
		t.Errorf("unexpected Authorization header: %v", headers["Authorization"])
	}

	if wh["form_param"] != "json_data" {
		t.Errorf("expected form_param %q, got %v", "json_data", wh["form_param"])
	}
	if wh["input_args_as_params"] != true {
		t.Errorf("expected input_args_as_params true, got %v", wh["input_args_as_params"])
	}

	requireArgs := wh["require_args"].([]string)
	if len(requireArgs) != 2 {
		t.Errorf("expected 2 require_args, got %d", len(requireArgs))
	}

	body := wh["body"].(map[string]any)
	if body["id"] != "${args.id}" {
		t.Errorf("unexpected body id: %v", body["id"])
	}
}

func TestOutputAndFallbackOutput(t *testing.T) {
	dm := New("multi_api").
		Purpose("Call multiple APIs").
		Webhook("GET", "https://primary.com/api", nil, "", false, nil).
		Output(swaig.NewFunctionResult("Primary: ${response.data}")).
		Webhook("GET", "https://fallback.com/api", nil, "", false, nil).
		Output(swaig.NewFunctionResult("Fallback: ${response.data}")).
		FallbackOutput(swaig.NewFunctionResult("All APIs failed"))

	result := dm.ToSwaigFunction()
	dataMap := result["data_map"].(map[string]any)

	webhooks := dataMap["webhooks"].([]map[string]any)
	if len(webhooks) != 2 {
		t.Fatalf("expected 2 webhooks, got %d", len(webhooks))
	}

	// Check first webhook output
	output1 := webhooks[0]["output"].(map[string]any)
	if output1["response"] != "Primary: ${response.data}" {
		t.Errorf("unexpected first webhook output: %v", output1["response"])
	}

	// Check second webhook output
	output2 := webhooks[1]["output"].(map[string]any)
	if output2["response"] != "Fallback: ${response.data}" {
		t.Errorf("unexpected second webhook output: %v", output2["response"])
	}

	// Check fallback output at data_map level
	fallback := dataMap["output"].(map[string]any)
	if fallback["response"] != "All APIs failed" {
		t.Errorf("unexpected fallback output: %v", fallback["response"])
	}
}

func TestExpressions(t *testing.T) {
	dm := New("command_handler").
		Purpose("Handle commands").
		Parameter("command", "string", "The command", true, nil).
		Expression("${args.command}", "start.*",
			swaig.NewFunctionResult("Starting..."),
			nil,
		).
		Expression("${args.command}", "stop.*",
			swaig.NewFunctionResult("Stopping..."),
			swaig.NewFunctionResult("Unknown stop command"),
		)

	result := dm.ToSwaigFunction()
	dataMap := result["data_map"].(map[string]any)

	expressions, ok := dataMap["expressions"].([]map[string]any)
	if !ok {
		t.Fatal("expected expressions to be []map[string]any")
	}
	if len(expressions) != 2 {
		t.Fatalf("expected 2 expressions, got %d", len(expressions))
	}

	// First expression - no nomatch
	expr1 := expressions[0]
	if expr1["string"] != "${args.command}" {
		t.Errorf("unexpected expression string: %v", expr1["string"])
	}
	if expr1["pattern"] != "start.*" {
		t.Errorf("unexpected expression pattern: %v", expr1["pattern"])
	}
	output1 := expr1["output"].(map[string]any)
	if output1["response"] != "Starting..." {
		t.Errorf("unexpected expression output: %v", output1["response"])
	}
	if _, exists := expr1["nomatch-output"]; exists {
		t.Error("expected no nomatch-output for first expression")
	}

	// Second expression - with nomatch
	expr2 := expressions[1]
	if _, exists := expr2["nomatch-output"]; !exists {
		t.Error("expected nomatch-output for second expression")
	}
	nomatch := expr2["nomatch-output"].(map[string]any)
	if nomatch["response"] != "Unknown stop command" {
		t.Errorf("unexpected nomatch output: %v", nomatch["response"])
	}
}

func TestCreateSimpleApiTool(t *testing.T) {
	dm := CreateSimpleApiTool(
		"get_stock",
		"https://api.stocks.com/v1/quote?symbol=${symbol}",
		"${response.name}: $${response.price}",
		map[string]map[string]any{
			"symbol": {
				"type":        "string",
				"description": "Stock ticker symbol",
				"required":    true,
			},
		},
		"GET",
		map[string]string{"X-Api-Key": "key123"},
		nil,
		[]string{"error", "message"},
	)

	result := dm.ToSwaigFunction()

	if result["function"] != "get_stock" {
		t.Errorf("expected function %q, got %v", "get_stock", result["function"])
	}

	// Should have default description since none was set
	if result["description"] != "Execute get_stock" {
		t.Errorf("expected default description, got %v", result["description"])
	}

	dataMap := result["data_map"].(map[string]any)
	webhooks := dataMap["webhooks"].([]map[string]any)
	if len(webhooks) != 1 {
		t.Fatalf("expected 1 webhook, got %d", len(webhooks))
	}

	wh := webhooks[0]
	if wh["method"] != "GET" {
		t.Errorf("expected method GET, got %v", wh["method"])
	}

	// Check error keys are on the webhook
	errorKeys := wh["error_keys"].([]string)
	if len(errorKeys) != 2 {
		t.Errorf("expected 2 error keys, got %d", len(errorKeys))
	}

	// Check output
	output := wh["output"].(map[string]any)
	if output["response"] != "${response.name}: $${response.price}" {
		t.Errorf("unexpected output response: %v", output["response"])
	}

	// Check headers
	headers := wh["headers"].(map[string]string)
	if headers["X-Api-Key"] != "key123" {
		t.Errorf("unexpected header: %v", headers["X-Api-Key"])
	}
}

func TestCreateSimpleApiToolWithBody(t *testing.T) {
	dm := CreateSimpleApiTool(
		"search",
		"https://api.search.com/query",
		"Found: ${response.results[0].title}",
		map[string]map[string]any{
			"query": {
				"type":        "string",
				"description": "Search query",
				"required":    true,
			},
		},
		"POST",
		nil,
		map[string]any{"q": "${args.query}", "limit": 5},
		nil,
	)

	result := dm.ToSwaigFunction()
	dataMap := result["data_map"].(map[string]any)
	webhooks := dataMap["webhooks"].([]map[string]any)
	wh := webhooks[0]

	if wh["method"] != "POST" {
		t.Errorf("expected POST, got %v", wh["method"])
	}

	body := wh["body"].(map[string]any)
	if body["q"] != "${args.query}" {
		t.Errorf("unexpected body q: %v", body["q"])
	}
	if body["limit"] != 5 {
		t.Errorf("unexpected body limit: %v", body["limit"])
	}
}

func TestCreateExpressionTool(t *testing.T) {
	dm := CreateExpressionTool(
		"playback_control",
		map[string]ExpressionPattern{
			"${args.command}": {
				Pattern: "play.*",
				Result:  swaig.NewFunctionResult("Playing now"),
			},
		},
		map[string]map[string]any{
			"command": {
				"type":        "string",
				"description": "Playback command",
				"required":    true,
			},
		},
	)

	result := dm.ToSwaigFunction()

	if result["function"] != "playback_control" {
		t.Errorf("expected function %q, got %v", "playback_control", result["function"])
	}

	dataMap := result["data_map"].(map[string]any)
	expressions := dataMap["expressions"].([]map[string]any)
	if len(expressions) != 1 {
		t.Fatalf("expected 1 expression, got %d", len(expressions))
	}

	expr := expressions[0]
	if expr["string"] != "${args.command}" {
		t.Errorf("unexpected expression string: %v", expr["string"])
	}
	if expr["pattern"] != "play.*" {
		t.Errorf("unexpected expression pattern: %v", expr["pattern"])
	}
}

func TestWebhookExpressions(t *testing.T) {
	exprs := []map[string]any{
		{
			"string":  "${response.status}",
			"pattern": "error",
			"output": map[string]any{
				"response": "An error occurred",
			},
		},
	}

	dm := New("check_status").
		Webhook("GET", "https://api.example.com/status", nil, "", false, nil).
		WebhookExpressions(exprs).
		Output(swaig.NewFunctionResult("Status: ${response.status}"))

	result := dm.ToSwaigFunction()
	dataMap := result["data_map"].(map[string]any)
	webhooks := dataMap["webhooks"].([]map[string]any)

	wh := webhooks[0]
	whExprs, ok := wh["expressions"].([]map[string]any)
	if !ok {
		t.Fatal("expected expressions on webhook")
	}
	if len(whExprs) != 1 {
		t.Errorf("expected 1 webhook expression, got %d", len(whExprs))
	}
}

func TestGlobalErrorKeys(t *testing.T) {
	dm := New("api_call").
		Webhook("GET", "https://example.com/api", nil, "", false, nil).
		Output(swaig.NewFunctionResult("Result: ${response.data}")).
		GlobalErrorKeys([]string{"error", "err_msg"})

	result := dm.ToSwaigFunction()
	dataMap := result["data_map"].(map[string]any)

	errorKeys, ok := dataMap["error_keys"].([]string)
	if !ok {
		t.Fatal("expected error_keys in data_map")
	}
	if len(errorKeys) != 2 {
		t.Errorf("expected 2 global error keys, got %d", len(errorKeys))
	}
	if errorKeys[0] != "error" || errorKeys[1] != "err_msg" {
		t.Errorf("unexpected global error keys: %v", errorKeys)
	}
}

func TestForeachConfig(t *testing.T) {
	dm := New("search_docs").
		Purpose("Search documentation").
		Parameter("query", "string", "Search query", true, nil).
		Webhook("POST", "https://api.docs.com/search", nil, "", false, nil).
		Body(map[string]any{"query": "${args.query}", "limit": 3}).
		Foreach(map[string]any{
			"input_key":  "results",
			"output_key": "formatted",
			"max":        3,
			"append":     "- ${this.title}: ${this.summary}\n",
		}).
		Output(swaig.NewFunctionResult("Results:\n${formatted}"))

	result := dm.ToSwaigFunction()
	dataMap := result["data_map"].(map[string]any)
	webhooks := dataMap["webhooks"].([]map[string]any)

	wh := webhooks[0]
	foreach, ok := wh["foreach"].(map[string]any)
	if !ok {
		t.Fatal("expected foreach on webhook")
	}
	if foreach["input_key"] != "results" {
		t.Errorf("unexpected foreach input_key: %v", foreach["input_key"])
	}
	if foreach["output_key"] != "formatted" {
		t.Errorf("unexpected foreach output_key: %v", foreach["output_key"])
	}
	if foreach["max"] != 3 {
		t.Errorf("unexpected foreach max: %v", foreach["max"])
	}
}

func TestParamsOnWebhook(t *testing.T) {
	dm := New("lookup").
		Webhook("GET", "https://example.com/lookup", nil, "", false, nil).
		Params(map[string]any{"q": "${args.query}"}).
		Output(swaig.NewFunctionResult("Found: ${response.result}"))

	result := dm.ToSwaigFunction()
	dataMap := result["data_map"].(map[string]any)
	webhooks := dataMap["webhooks"].([]map[string]any)

	wh := webhooks[0]
	params, ok := wh["params"].(map[string]any)
	if !ok {
		t.Fatal("expected params on webhook")
	}
	if params["q"] != "${args.query}" {
		t.Errorf("unexpected params q: %v", params["q"])
	}
}

func TestEmptyDataMap(t *testing.T) {
	dm := New("empty_tool")
	result := dm.ToSwaigFunction()

	if result["function"] != "empty_tool" {
		t.Errorf("expected function %q, got %v", "empty_tool", result["function"])
	}

	dataMap := result["data_map"].(map[string]any)
	if len(dataMap) != 0 {
		t.Errorf("expected empty data_map, got %d entries", len(dataMap))
	}

	argument := result["parameters"].(map[string]any)
	properties := argument["properties"].(map[string]any)
	if len(properties) != 0 {
		t.Errorf("expected empty properties, got %d entries", len(properties))
	}
}

func TestMethodUppercased(t *testing.T) {
	dm := New("test").
		Webhook("post", "https://example.com", nil, "", false, nil).
		Output(swaig.NewFunctionResult("ok"))

	result := dm.ToSwaigFunction()
	dataMap := result["data_map"].(map[string]any)
	webhooks := dataMap["webhooks"].([]map[string]any)

	if webhooks[0]["method"] != "POST" {
		t.Errorf("expected method to be uppercased to POST, got %v", webhooks[0]["method"])
	}
}

func TestMultipleWebhooksPreserveSeparateConfig(t *testing.T) {
	dm := New("multi").
		Webhook("GET", "https://primary.com", nil, "", false, nil).
		Body(map[string]any{"source": "primary"}).
		ErrorKeys([]string{"err"}).
		Output(swaig.NewFunctionResult("Primary result")).
		Webhook("GET", "https://secondary.com", nil, "", false, nil).
		Body(map[string]any{"source": "secondary"}).
		Output(swaig.NewFunctionResult("Secondary result"))

	result := dm.ToSwaigFunction()
	dataMap := result["data_map"].(map[string]any)
	webhooks := dataMap["webhooks"].([]map[string]any)

	if len(webhooks) != 2 {
		t.Fatalf("expected 2 webhooks, got %d", len(webhooks))
	}

	// First webhook should have its own body and error keys
	body1 := webhooks[0]["body"].(map[string]any)
	if body1["source"] != "primary" {
		t.Errorf("expected first webhook body source %q, got %v", "primary", body1["source"])
	}
	ek1 := webhooks[0]["error_keys"].([]string)
	if len(ek1) != 1 || ek1[0] != "err" {
		t.Errorf("expected first webhook error_keys [err], got %v", ek1)
	}

	// Second webhook should have its own body, no error keys
	body2 := webhooks[1]["body"].(map[string]any)
	if body2["source"] != "secondary" {
		t.Errorf("expected second webhook body source %q, got %v", "secondary", body2["source"])
	}
	if _, exists := webhooks[1]["error_keys"]; exists {
		t.Error("expected second webhook to have no error_keys")
	}
}
