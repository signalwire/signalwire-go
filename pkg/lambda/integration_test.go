package lambda_test

// This file houses the cross-package integration tests that wire a real
// agent.AgentBase + swml.Service stack through the Lambda adapter. It
// lives in a separate _test package to avoid a dependency cycle with
// pkg/agent (which would pull the agent package into pkg/lambda at
// compile time if declared in the same `package lambda` block).

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/lambda"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// clearAWSEnv wipes the AWS env vars that would otherwise leak between
// tests running in the same process. GetExecutionMode inspects these
// directly and there is no dependency-injected seam at the moment.
func clearAWSEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"AWS_LAMBDA_FUNCTION_NAME",
		"LAMBDA_TASK_ROOT",
		"AWS_LAMBDA_FUNCTION_URL",
		"AWS_REGION",
		"SWML_PROXY_URL_BASE",
		"GATEWAY_INTERFACE",
		"FUNCTION_TARGET",
		"K_SERVICE",
		"GOOGLE_CLOUD_PROJECT",
		"AZURE_FUNCTIONS_ENVIRONMENT",
		"FUNCTIONS_WORKER_RUNTIME",
		"AzureWebJobsStorage",
	} {
		t.Setenv(k, "")
	}
}

// basicAuthHeader returns the HTTP basic-auth header value for the given
// credentials.
func basicAuthHeader(user, pass string) string {
	cred := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
	return "Basic " + cred
}

// ---------------------------------------------------------------------------
// End-to-end Lambda dispatch
// ---------------------------------------------------------------------------

// TestLambdaHandler_ServesSWMLDocument drives a complete agent through the
// Lambda adapter and asserts the SWML document comes out intact. This is
// the main smoke test for the "wrap agent as Lambda handler" use case.
func TestLambdaHandler_ServesSWMLDocument(t *testing.T) {
	clearAWSEnv(t)

	const user, pass = "user", "pass"
	a := agent.NewAgentBase(
		agent.WithName("IntegrationAgent"),
		agent.WithRoute("/my-agent"),
		agent.WithBasicAuth(user, pass),
	)
	a.PromptAddSection("Role", "You are helpful.", nil)

	h := lambda.NewHandler(a.AsRouter())

	req := events.LambdaFunctionURLRequest{
		RawPath: "/my-agent",
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: "POST"},
		},
		Headers: map[string]string{
			"authorization": basicAuthHeader(user, pass),
			"content-type":  "application/json",
		},
		Body: "{}",
	}

	resp, err := h.HandleFunctionURL(context.Background(), req)
	if err != nil {
		t.Fatalf("HandleFunctionURL: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%q", resp.StatusCode, resp.Body)
	}
	if ct := resp.Headers["Content-Type"]; !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var doc map[string]any
	if err := json.Unmarshal([]byte(resp.Body), &doc); err != nil {
		t.Fatalf("response body is not JSON: %v", err)
	}
	if v, _ := doc["version"].(string); v == "" {
		t.Errorf("SWML document missing version field: %v", doc)
	}
}

// TestLambdaHandler_RejectsMissingAuth verifies that the adapter correctly
// surfaces the agent's auth middleware — 401 comes back through the Lambda
// response envelope when credentials are absent.
func TestLambdaHandler_RejectsMissingAuth(t *testing.T) {
	clearAWSEnv(t)

	a := agent.NewAgentBase(
		agent.WithName("AuthAgent"),
		agent.WithRoute("/secure"),
		agent.WithBasicAuth("u", "p"),
	)
	h := lambda.NewHandler(a.AsRouter())

	req := events.LambdaFunctionURLRequest{
		RawPath: "/secure",
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: "GET"},
		},
		// Intentionally no Authorization header.
	}
	resp, err := h.HandleFunctionURL(context.Background(), req)
	if err != nil {
		t.Fatalf("HandleFunctionURL: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
	if resp.Headers["Www-Authenticate"] == "" && resp.Headers["WWW-Authenticate"] == "" {
		t.Errorf("expected WWW-Authenticate header in 401 response, got headers=%v", resp.Headers)
	}
}

// TestLambdaHandler_DispatchesSwaigFunctionCall verifies that a SWAIG
// function call delivered via the Lambda adapter is routed to the correct
// tool handler and the handler's FunctionResult is returned unmodified.
func TestLambdaHandler_DispatchesSwaigFunctionCall(t *testing.T) {
	clearAWSEnv(t)

	const user, pass = "swaig-u", "swaig-p"
	a := agent.NewAgentBase(
		agent.WithName("SwaigAgent"),
		agent.WithRoute("/bot"),
		agent.WithBasicAuth(user, pass),
	)
	a.DefineTool(agent.ToolDefinition{
		Name:        "greet",
		Description: "Greet",
		Parameters:  map[string]any{"name": map[string]any{"type": "string"}},
		Handler: func(args map[string]any, raw map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult(fmt.Sprintf("hello %v", args["name"]))
		},
	})

	h := lambda.NewHandler(a.AsRouter())

	// The Go SDK's handleSwaig treats `argument` as the flat args map
	// (matching how the existing pkg/agent tests exercise it). The nested
	// parsed/raw shape seen in Python is not accepted here.
	body := `{"function":"greet","argument":{"name":"Ada"}}`
	req := events.LambdaFunctionURLRequest{
		RawPath: "/bot/swaig",
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: "POST"},
		},
		Headers: map[string]string{
			"authorization": basicAuthHeader(user, pass),
			"content-type":  "application/json",
		},
		Body: body,
	}
	resp, err := h.HandleFunctionURL(context.Background(), req)
	if err != nil {
		t.Fatalf("HandleFunctionURL: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%q", resp.StatusCode, resp.Body)
	}
	if !strings.Contains(resp.Body, "hello Ada") {
		t.Errorf("response body missing greeting: %q", resp.Body)
	}
}

// TestLambdaHandler_HealthEndpointNoAuth verifies that health checks don't
// require basic auth — something typical Lambda deployments rely on.
func TestLambdaHandler_HealthEndpointNoAuth(t *testing.T) {
	clearAWSEnv(t)

	a := agent.NewAgentBase(
		agent.WithName("HealthAgent"),
		agent.WithRoute("/app"),
		agent.WithBasicAuth("u", "p"),
	)
	h := lambda.NewHandler(a.AsRouter())

	req := events.LambdaFunctionURLRequest{
		RawPath: "/health",
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: "GET"},
		},
	}
	resp, err := h.HandleFunctionURL(context.Background(), req)
	if err != nil {
		t.Fatalf("HandleFunctionURL: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("health status = %d, want 200", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Regression guard: route preservation under Lambda env
// ---------------------------------------------------------------------------
//
// This is the critical bug documented in the port brief: the webhook URL
// baked into the rendered SWML document must include both the agent's
// route AND the /swaig suffix. Earlier Python/TS fixes missed the Route
// append when a proxy base was set.
//
// The test renders the SWML document through the Lambda adapter with a
// non-root Route and both Lambda + proxy env vars set, then walks the
// document down to the AI verb's SWAIG.functions[].web_hook_url and
// asserts the full expected URL string. Relying on the rendered document
// rather than an internal helper means we catch regressions even if the
// relevant code paths get refactored.

func TestLambdaHandler_Regression_WebhookURLIncludesRoute(t *testing.T) {
	clearAWSEnv(t)
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "demo-func")
	t.Setenv("AWS_REGION", "us-east-1")
	// Proxy base overrides Lambda URL construction; the regression bug
	// was specifically that this override dropped the agent's route.
	t.Setenv("SWML_PROXY_URL_BASE", "https://xyz.lambda-url.us-east-1.on.aws")

	const user, pass = "u", "p"
	a := agent.NewAgentBase(
		agent.WithName("RouteRegression"),
		agent.WithRoute("/my-agent"),
		agent.WithBasicAuth(user, pass),
	)
	a.DefineTool(agent.ToolDefinition{
		Name:        "ping",
		Description: "ping",
		Handler: func(args, raw map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("pong")
		},
	})

	h := lambda.NewHandler(a.AsRouter())
	req := events.LambdaFunctionURLRequest{
		RawPath: "/my-agent",
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: "POST"},
		},
		Headers: map[string]string{"authorization": basicAuthHeader(user, pass)},
		Body:    "{}",
	}
	resp, err := h.HandleFunctionURL(context.Background(), req)
	if err != nil {
		t.Fatalf("HandleFunctionURL: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	webhookURL := extractFirstSwaigWebhook(t, resp.Body)
	wantSuffix := "/my-agent/swaig"
	if !strings.Contains(webhookURL, wantSuffix) {
		t.Fatalf(
			"route-preservation regression: webhook URL = %q, want substring %q. "+
				"The proxy base + Lambda combo must still include the agent's route.",
			webhookURL, wantSuffix,
		)
	}
	// Also verify the scheme/host block came from the proxy env var, not
	// from the localhost fallback.
	if !strings.Contains(webhookURL, "xyz.lambda-url.us-east-1.on.aws") {
		t.Fatalf("webhook URL = %q, want proxy host substring", webhookURL)
	}
	// And verify it is NOT the buggy shape that omits the route.
	if strings.HasSuffix(webhookURL, "xyz.lambda-url.us-east-1.on.aws/swaig") ||
		strings.Contains(webhookURL, ".on.aws@") && strings.HasSuffix(webhookURL, "/swaig") && !strings.Contains(webhookURL, "/my-agent/swaig") {
		t.Fatalf("webhook URL matches the forbidden bare-proxy shape: %q", webhookURL)
	}
}

// TestLambdaHandler_Regression_WebhookURLLambdaOnly exercises the other
// half of the same invariant: even when NO proxy base is configured, a
// Lambda-mode agent with a non-root Route must still produce a webhook
// URL that includes that route.
func TestLambdaHandler_Regression_WebhookURLLambdaOnly(t *testing.T) {
	clearAWSEnv(t)
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "demo-func")
	t.Setenv("AWS_LAMBDA_FUNCTION_URL", "https://demo-func.lambda-url.us-east-1.on.aws")

	const user, pass = "u", "p"
	a := agent.NewAgentBase(
		agent.WithName("LambdaOnly"),
		agent.WithRoute("/bot"),
		agent.WithBasicAuth(user, pass),
	)
	a.DefineTool(agent.ToolDefinition{
		Name:        "noop",
		Description: "noop",
		Handler: func(args, raw map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("ok")
		},
	})

	h := lambda.NewHandler(a.AsRouter())
	req := events.LambdaFunctionURLRequest{
		RawPath: "/bot",
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: "POST"},
		},
		Headers: map[string]string{"authorization": basicAuthHeader(user, pass)},
		Body:    "{}",
	}
	resp, err := h.HandleFunctionURL(context.Background(), req)
	if err != nil {
		t.Fatalf("HandleFunctionURL: %v", err)
	}

	webhookURL := extractFirstSwaigWebhook(t, resp.Body)
	const wantSuffix = "/bot/swaig"
	if !strings.Contains(webhookURL, wantSuffix) {
		t.Fatalf("webhook URL = %q, want substring %q", webhookURL, wantSuffix)
	}
	if !strings.Contains(webhookURL, "demo-func.lambda-url.us-east-1.on.aws") {
		t.Fatalf("webhook URL = %q, want Lambda function URL host", webhookURL)
	}
}

// extractFirstSwaigWebhook walks a rendered SWML document (as a JSON
// string) and returns the web_hook_url of the first SWAIG function it
// finds. t.Fatal is called if the structure is missing — this is always
// a test bug rather than an expected condition, so a hard fail is fine.
func extractFirstSwaigWebhook(t *testing.T, body string) string {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("SWML body is not JSON: %v; body=%q", err, body)
	}
	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)
	for _, v := range main {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		aiCfg, ok := m["ai"].(map[string]any)
		if !ok {
			continue
		}
		swaigCfg, ok := aiCfg["SWAIG"].(map[string]any)
		if !ok {
			continue
		}
		fns, _ := swaigCfg["functions"].([]any)
		if len(fns) == 0 {
			t.Fatalf("SWAIG functions array is empty; document=%v", doc)
		}
		first, _ := fns[0].(map[string]any)
		url, _ := first["web_hook_url"].(string)
		return url
	}
	t.Fatalf("could not find ai verb with SWAIG functions in document; body=%s", body)
	return ""
}

