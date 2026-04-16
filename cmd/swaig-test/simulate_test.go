package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/lambda"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// ---------------------------------------------------------------------------
// Helpers shared across simulate tests
// ---------------------------------------------------------------------------

// simulateManagedKeys mirrors managedEnvKeys() — duplicated here so the
// tests don't depend on the private list becoming a test-visible export.
// Kept in sync by the test "TestManagedEnvKeysCoversAllKnownVars".
func simulateManagedKeys() []string {
	return []string{
		"AWS_LAMBDA_FUNCTION_NAME",
		"LAMBDA_TASK_ROOT",
		"AWS_REGION",
		"AWS_LAMBDA_FUNCTION_URL",
		"_HANDLER",
		"GATEWAY_INTERFACE",
		"HTTP_HOST",
		"SCRIPT_NAME",
		"FUNCTION_TARGET",
		"K_SERVICE",
		"GOOGLE_CLOUD_PROJECT",
		"AZURE_FUNCTIONS_ENVIRONMENT",
		"FUNCTIONS_WORKER_RUNTIME",
		"AzureWebJobsStorage",
		"SWML_PROXY_URL_BASE",
	}
}

// clearSimulatorEnv zeroes every env var the simulator manages. Use
// this at the top of any test that asserts on env state so an outer
// shell or leaked state from a previous test doesn't poison the
// result. t.Setenv is preferred over os.Setenv because Go's testing
// framework auto-restores those on t.Cleanup.
func clearSimulatorEnv(t *testing.T) {
	t.Helper()
	for _, k := range simulateManagedKeys() {
		t.Setenv(k, "")
		os.Unsetenv(k) // t.Setenv uses Setenv("", "") which keeps the key, so follow up with Unsetenv
	}
}

// newTestAgent returns a small agent configured for simulator tests.
// The agent has a single tool ("ping") that echoes back so tests can
// verify SWAIG dispatch too.
func newTestAgent(route, user, pass string) *agent.AgentBase {
	a := agent.NewAgentBase(
		agent.WithName("SimTestAgent"),
		agent.WithRoute(route),
		agent.WithBasicAuth(user, pass),
	)
	a.DefineTool(agent.ToolDefinition{
		Name:        "ping",
		Description: "ping",
		Parameters:  map[string]any{"msg": map[string]any{"type": "string"}},
		Handler: func(args, raw map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult(fmt.Sprintf("pong: %v", args["msg"]))
		},
	})
	return a
}

// ---------------------------------------------------------------------------
// Platform validation
// ---------------------------------------------------------------------------

func TestValidateSimulatePlatform_LambdaAccepted(t *testing.T) {
	if err := validateSimulatePlatform("lambda"); err != nil {
		t.Fatalf("expected lambda to be accepted, got error: %v", err)
	}
}

func TestValidateSimulatePlatform_UnimplementedPlatformsRejected(t *testing.T) {
	for _, platform := range []string{"gcf", "cloud_function", "azure", "azure_function", "cgi"} {
		t.Run(platform, func(t *testing.T) {
			err := validateSimulatePlatform(platform)
			if err == nil {
				t.Fatalf("expected error for unimplemented platform %q", platform)
			}
			if !strings.Contains(err.Error(), "not implemented") {
				t.Errorf("error for %q should say 'not implemented'; got: %v", platform, err)
			}
			if !strings.Contains(err.Error(), "Phase 9") {
				t.Errorf("error for %q should reference Phase 9; got: %v", platform, err)
			}
		})
	}
}

func TestValidateSimulatePlatform_UnknownPlatformRejected(t *testing.T) {
	err := validateSimulatePlatform("not-a-real-platform")
	if err == nil {
		t.Fatal("expected error for unknown platform")
	}
	if !strings.Contains(err.Error(), "unknown platform") {
		t.Errorf("error should say 'unknown platform'; got: %v", err)
	}
}

func TestValidateSimulatePlatform_EmptyRejected(t *testing.T) {
	err := validateSimulatePlatform("")
	if err == nil {
		t.Fatal("expected error for empty platform")
	}
}

// ---------------------------------------------------------------------------
// Env save/restore lifecycle
// ---------------------------------------------------------------------------

// TestActivateLambdaEnv_SetsExpectedVars confirms the happy path:
// activation sets the preset variables so GetExecutionMode returns
// ModeLambda and the Lambda URL branch engages.
func TestActivateLambdaEnv_SetsExpectedVars(t *testing.T) {
	clearSimulatorEnv(t)

	snap := activateLambdaEnv(SimulateLambdaOptions{})
	defer snap.restore()

	if got := os.Getenv("AWS_LAMBDA_FUNCTION_NAME"); got != "test-agent-function" {
		t.Errorf("AWS_LAMBDA_FUNCTION_NAME = %q, want %q", got, "test-agent-function")
	}
	if got := os.Getenv("LAMBDA_TASK_ROOT"); got != "/var/task" {
		t.Errorf("LAMBDA_TASK_ROOT = %q, want %q", got, "/var/task")
	}
	if got := os.Getenv("AWS_REGION"); got != "us-east-1" {
		t.Errorf("AWS_REGION = %q, want %q", got, "us-east-1")
	}
}

// TestActivateLambdaEnv_RestoresOnHappyPath verifies that calling
// restore() rolls the environment back to its pre-activation state.
func TestActivateLambdaEnv_RestoresOnHappyPath(t *testing.T) {
	clearSimulatorEnv(t)
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "preexisting-fn")

	snap := activateLambdaEnv(SimulateLambdaOptions{})
	if got := os.Getenv("AWS_LAMBDA_FUNCTION_NAME"); got != "test-agent-function" {
		t.Fatalf("during activation AWS_LAMBDA_FUNCTION_NAME = %q, want preset value", got)
	}

	snap.restore()

	if got := os.Getenv("AWS_LAMBDA_FUNCTION_NAME"); got != "preexisting-fn" {
		t.Errorf("after restore AWS_LAMBDA_FUNCTION_NAME = %q, want %q", got, "preexisting-fn")
	}
}

// TestActivateLambdaEnv_RestoresUnsetState pins down that a var which
// was NOT set before activation is correctly unset on restore (rather
// than restored to empty string — those are meaningfully different to
// os.LookupEnv consumers).
func TestActivateLambdaEnv_RestoresUnsetState(t *testing.T) {
	clearSimulatorEnv(t)
	// Make sure this really is unset.
	os.Unsetenv("AWS_LAMBDA_FUNCTION_NAME")

	snap := activateLambdaEnv(SimulateLambdaOptions{})
	snap.restore()

	if _, ok := os.LookupEnv("AWS_LAMBDA_FUNCTION_NAME"); ok {
		t.Errorf("AWS_LAMBDA_FUNCTION_NAME should be unset after restore, but is present")
	}
}

// TestActivateLambdaEnv_ClearsProxyURLBase proves the simulator wipes
// SWML_PROXY_URL_BASE during activation — the load-bearing behaviour
// that matches mock_env.py and makes Lambda URL construction visible.
func TestActivateLambdaEnv_ClearsProxyURLBase(t *testing.T) {
	clearSimulatorEnv(t)
	t.Setenv("SWML_PROXY_URL_BASE", "https://proxy.outer-shell.example.com")

	snap := activateLambdaEnv(SimulateLambdaOptions{})
	defer snap.restore()

	if got := os.Getenv("SWML_PROXY_URL_BASE"); got != "" {
		t.Errorf("during activation SWML_PROXY_URL_BASE = %q, want cleared", got)
	}
}

// TestActivateLambdaEnv_RestoresProxyURLBase confirms the original
// value of SWML_PROXY_URL_BASE is put back after restore(). Leaking
// a cleared proxy base would break any subsequent test that relies
// on it.
func TestActivateLambdaEnv_RestoresProxyURLBase(t *testing.T) {
	clearSimulatorEnv(t)
	const original = "https://proxy.outer-shell.example.com"
	t.Setenv("SWML_PROXY_URL_BASE", original)

	snap := activateLambdaEnv(SimulateLambdaOptions{})
	snap.restore()

	if got := os.Getenv("SWML_PROXY_URL_BASE"); got != original {
		t.Errorf("after restore SWML_PROXY_URL_BASE = %q, want %q", got, original)
	}
}

// TestActivateLambdaEnv_RestoresOnErrorPath runs the simulated
// invocation through a handler that returns an error, and verifies
// the environment is still restored afterwards. Leaked env state
// would cause downstream tests in the same process to behave
// nondeterministically.
func TestActivateLambdaEnv_RestoresOnErrorPath(t *testing.T) {
	clearSimulatorEnv(t)
	t.Setenv("SWML_PROXY_URL_BASE", "https://original.example.com")

	// The handler panics to simulate a buggy agent. The test uses a
	// dedicated recovering wrapper so the panic doesn't tear the
	// test binary down — we only want to prove env state rolls back.
	func() {
		defer func() {
			_ = recover()
		}()
		snap := activateLambdaEnv(SimulateLambdaOptions{})
		defer snap.restore()
		panic("simulated agent crash")
	}()

	if got := os.Getenv("SWML_PROXY_URL_BASE"); got != "https://original.example.com" {
		t.Errorf("after panic+restore SWML_PROXY_URL_BASE = %q, want original", got)
	}
	if got := os.Getenv("AWS_LAMBDA_FUNCTION_NAME"); got != "" {
		t.Errorf("after panic+restore AWS_LAMBDA_FUNCTION_NAME = %q, want empty", got)
	}
}

// TestActivateLambdaEnv_WithOptions exercises the override fields on
// SimulateLambdaOptions: custom function name, region, and explicit
// function URL all propagate into the environment.
func TestActivateLambdaEnv_WithOptions(t *testing.T) {
	clearSimulatorEnv(t)

	snap := activateLambdaEnv(SimulateLambdaOptions{
		FunctionName:        "custom-fn",
		Region:              "eu-west-1",
		FunctionURLOverride: "https://override.example.com/",
	})
	defer snap.restore()

	if got := os.Getenv("AWS_LAMBDA_FUNCTION_NAME"); got != "custom-fn" {
		t.Errorf("FUNCTION_NAME = %q, want %q", got, "custom-fn")
	}
	if got := os.Getenv("AWS_REGION"); got != "eu-west-1" {
		t.Errorf("AWS_REGION = %q, want %q", got, "eu-west-1")
	}
	if got := os.Getenv("AWS_LAMBDA_FUNCTION_URL"); got != "https://override.example.com/" {
		t.Errorf("AWS_LAMBDA_FUNCTION_URL = %q, want override", got)
	}
}

// ---------------------------------------------------------------------------
// Adapter dispatch (dump-swml + exec) through the Lambda handler
// ---------------------------------------------------------------------------

// extractWebhookURL digs into a rendered SWML document for the first
// SWAIG function's web_hook_url. Mirrors the helper inside
// pkg/lambda/integration_test.go (duplicated here to keep this test
// file self-contained).
func extractWebhookURL(t *testing.T, body []byte) string {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatalf("response is not JSON: %v\nbody=%s", err, string(body))
	}
	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)
	for _, verb := range main {
		v, _ := verb.(map[string]any)
		ai, _ := v["ai"].(map[string]any)
		sw, _ := ai["SWAIG"].(map[string]any)
		if sw == nil {
			continue
		}
		fns, _ := sw["functions"].([]any)
		for _, fn := range fns {
			fm, _ := fn.(map[string]any)
			if url, ok := fm["web_hook_url"].(string); ok && url != "" {
				return url
			}
		}
		if url, ok := sw["web_hook_url"].(string); ok && url != "" {
			return url
		}
		if defs, _ := sw["defaults"].(map[string]any); defs != nil {
			if url, ok := defs["web_hook_url"].(string); ok && url != "" {
				return url
			}
		}
	}
	t.Fatalf("no web_hook_url found in SWML document: %s", string(body))
	return ""
}

// TestSimulateDumpSWMLViaLambda_NonRootRoute is the load-bearing
// regression test called out in the task: a Lambda-simulated agent
// at route /my-agent produces SWML whose webhook URL contains
// /my-agent/swaig.
func TestSimulateDumpSWMLViaLambda_NonRootRoute(t *testing.T) {
	clearSimulatorEnv(t)

	factory := func() http.Handler {
		return newTestAgent("/my-agent", "u", "p").AsRouter()
	}
	body, err := SimulateDumpSWMLViaLambda(
		factory,
		"/my-agent",
		SimulateLambdaOptions{},
		BasicAuth{User: "u", Password: "p"},
	)
	if err != nil {
		t.Fatalf("SimulateDumpSWMLViaLambda: %v", err)
	}

	url := extractWebhookURL(t, body)
	if !strings.Contains(url, "/my-agent/swaig") {
		t.Fatalf(
			"webhook URL = %q, want substring %q. Non-root route must be "+
				"preserved through the Lambda adapter.",
			url, "/my-agent/swaig",
		)
	}
	if !strings.Contains(url, "lambda-url.") || !strings.Contains(url, ".on.aws") {
		t.Errorf("webhook URL = %q, want Lambda-style host", url)
	}
}

// TestSimulateDumpSWMLViaLambda_ClearsProxyBaseBeforeRender is the
// other load-bearing behaviour: even if the outer shell has
// SWML_PROXY_URL_BASE set, the simulator clears it before the agent
// is constructed, so the webhook URLs come out Lambda-shaped rather
// than pointing at the outer proxy. This is what the factory-based
// simulator API buys us: the agent captures SWML_PROXY_URL_BASE at
// construction, so it has to be built post-activation.
func TestSimulateDumpSWMLViaLambda_ClearsProxyBaseBeforeRender(t *testing.T) {
	clearSimulatorEnv(t)
	t.Setenv("SWML_PROXY_URL_BASE", "https://outer.example.com")

	factory := func() http.Handler {
		return newTestAgent("/my-agent", "u", "p").AsRouter()
	}
	body, err := SimulateDumpSWMLViaLambda(
		factory,
		"/my-agent",
		SimulateLambdaOptions{},
		BasicAuth{User: "u", Password: "p"},
	)
	if err != nil {
		t.Fatalf("SimulateDumpSWMLViaLambda: %v", err)
	}

	url := extractWebhookURL(t, body)
	if strings.Contains(url, "outer.example.com") {
		t.Fatalf(
			"webhook URL = %q still contains outer proxy host; simulator "+
				"failed to clear SWML_PROXY_URL_BASE before agent load",
			url,
		)
	}
	if !strings.Contains(url, ".lambda-url.") || !strings.Contains(url, ".on.aws") {
		t.Fatalf(
			"webhook URL = %q is not Lambda-style; simulator didn't engage "+
				"Lambda URL generation",
			url,
		)
	}
	if !strings.Contains(url, "/my-agent/swaig") {
		t.Errorf("webhook URL = %q, want /my-agent/swaig substring", url)
	}

	// And the outer proxy env var must be restored on exit.
	if got := os.Getenv("SWML_PROXY_URL_BASE"); got != "https://outer.example.com" {
		t.Errorf("SWML_PROXY_URL_BASE after simulator exit = %q, want restored", got)
	}
}

// TestSimulateDumpSWMLViaLambda_RootRoute pins down the root-route
// case — a less common deployment but worth guarding against trailing-
// slash bugs.
func TestSimulateDumpSWMLViaLambda_RootRoute(t *testing.T) {
	clearSimulatorEnv(t)

	factory := func() http.Handler {
		return newTestAgent("/", "u", "p").AsRouter()
	}
	body, err := SimulateDumpSWMLViaLambda(
		factory,
		"/",
		SimulateLambdaOptions{},
		BasicAuth{User: "u", Password: "p"},
	)
	if err != nil {
		t.Fatalf("SimulateDumpSWMLViaLambda: %v", err)
	}

	url := extractWebhookURL(t, body)
	if !strings.Contains(url, "/swaig") {
		t.Errorf("webhook URL = %q, want /swaig suffix", url)
	}
	if !strings.Contains(url, ".lambda-url.") || !strings.Contains(url, ".on.aws") {
		t.Errorf("webhook URL = %q, want Lambda-style host", url)
	}
}

// TestSimulateExecToolViaLambda verifies SWAIG dispatch through the
// simulated adapter hits the right tool handler.
func TestSimulateExecToolViaLambda(t *testing.T) {
	clearSimulatorEnv(t)

	factory := func() http.Handler {
		return newTestAgent("/my-agent", "u", "p").AsRouter()
	}
	body, err := SimulateExecToolViaLambda(
		factory,
		"/my-agent",
		"ping",
		map[string]any{"msg": "hi"},
		SimulateLambdaOptions{},
		BasicAuth{User: "u", Password: "p"},
	)
	if err != nil {
		t.Fatalf("SimulateExecToolViaLambda: %v", err)
	}

	if !strings.Contains(string(body), "pong: hi") {
		t.Errorf("tool response body = %q, want substring %q", string(body), "pong: hi")
	}
}

// TestSimulateExecToolViaLambda_RestoresOnError proves env restore
// runs even when the tool call fails (here: we return HTTP 401 by
// passing wrong credentials, so the adapter yields an error).
func TestSimulateExecToolViaLambda_RestoresOnError(t *testing.T) {
	clearSimulatorEnv(t)
	t.Setenv("SWML_PROXY_URL_BASE", "https://original.example.com")

	factory := func() http.Handler {
		return newTestAgent("/my-agent", "u", "p").AsRouter()
	}
	_, err := SimulateExecToolViaLambda(
		factory,
		"/my-agent",
		"ping",
		map[string]any{},
		SimulateLambdaOptions{},
		BasicAuth{User: "wrong", Password: "credentials"},
	)
	if err == nil {
		t.Fatal("expected error for wrong auth credentials")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error = %v, want HTTP 401", err)
	}

	// Even though the simulated call errored, env should be restored.
	if got := os.Getenv("SWML_PROXY_URL_BASE"); got != "https://original.example.com" {
		t.Errorf("after error SWML_PROXY_URL_BASE = %q, want restored", got)
	}
	if _, ok := os.LookupEnv("AWS_LAMBDA_FUNCTION_NAME"); ok {
		t.Errorf("AWS_LAMBDA_FUNCTION_NAME leaked past restore")
	}
}

// ---------------------------------------------------------------------------
// Library API boundary: the lambda adapter does actually get used
// ---------------------------------------------------------------------------

// TestSimulateLambdaInvocation_NilHandlerRejected ensures we bail out
// cleanly rather than panicking on a programming error.
func TestSimulateLambdaInvocation_NilHandlerRejected(t *testing.T) {
	_, err := SimulateLambdaInvocation(nil, http.MethodGet, "/", nil, nil)
	if err == nil {
		t.Fatal("expected error for nil handler")
	}
}

// TestSimulateLambdaInvocation_PassThroughMethodAndPath sanity-checks
// that method/path flow into the synthetic event correctly by wiring
// a handler that echoes them back through pkg/lambda.
func TestSimulateLambdaInvocation_PassThroughMethodAndPath(t *testing.T) {
	var seenMethod, seenPath string
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenMethod = r.Method
		seenPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	_, err := SimulateLambdaInvocation(h, http.MethodPost, "/agent-a/swaig", nil, strings.NewReader(""))
	if err != nil {
		t.Fatalf("SimulateLambdaInvocation: %v", err)
	}
	if seenMethod != "POST" {
		t.Errorf("handler saw method %q, want POST", seenMethod)
	}
	if seenPath != "/agent-a/swaig" {
		t.Errorf("handler saw path %q, want /agent-a/swaig", seenPath)
	}
}

// TestSimulateLambdaInvocation_RealAdapter ties the simulator to the
// actual pkg/lambda adapter by reaching into the same API that a
// Lambda-deployed agent would use. This is what proves "route through
// the adapter, NOT the HTTP server" in this CI: the simulator
// composes lambda.NewHandler under the hood.
func TestSimulateLambdaInvocation_RealAdapter(t *testing.T) {
	clearSimulatorEnv(t)

	a := newTestAgent("/my-agent", "u", "p")

	// Call pkg/lambda directly as a sanity check that what the
	// simulator invokes is byte-identical to what a real Lambda
	// runtime would do.
	adapter := lambda.NewHandler(a.AsRouter())
	evt := events.LambdaFunctionURLRequest{
		RawPath: "/my-agent",
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: "POST"},
		},
		Headers: map[string]string{
			"authorization": basicAuthHeader("u", "p"),
			"content-type":  "application/json",
		},
		Body: "{}",
	}
	resp, err := adapter.HandleFunctionURL(context.Background(), evt)
	if err != nil {
		t.Fatalf("direct adapter call failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("direct adapter status = %d, want 200, body=%q", resp.StatusCode, resp.Body)
	}

	// Now invoke the same agent via the simulator; its response body
	// should parse as the same SWML shape.
	snap := activateLambdaEnv(SimulateLambdaOptions{})
	defer snap.restore()
	result, err := SimulateLambdaInvocation(
		a.AsRouter(),
		"POST",
		"/my-agent",
		map[string]string{
			"authorization": basicAuthHeader("u", "p"),
			"content-type":  "application/json",
		},
		strings.NewReader("{}"),
	)
	if err != nil {
		t.Fatalf("simulator call failed: %v", err)
	}
	if result.Status != http.StatusOK {
		t.Fatalf("simulator status = %d, want 200", result.Status)
	}

	var doc map[string]any
	if err := json.Unmarshal(result.Body, &doc); err != nil {
		t.Fatalf("simulator body is not JSON: %v\nbody=%s", err, string(result.Body))
	}
	if _, ok := doc["sections"].(map[string]any); !ok {
		t.Errorf("simulator body missing 'sections': %v", doc)
	}
}

// ---------------------------------------------------------------------------
// CLI-level integration: validate --simulate-serverless behaviour through run()
// ---------------------------------------------------------------------------

// TestRun_SimulateServerless_RejectsUnimplementedPlatform drives the
// CLI entry point (run) with an unimplemented platform and confirms
// it exits non-zero (returns a non-nil error) with a message that
// points at Phase 9.
func TestRun_SimulateServerless_RejectsUnimplementedPlatform(t *testing.T) {
	for _, platform := range []string{"gcf", "azure", "cgi"} {
		t.Run(platform, func(t *testing.T) {
			cfg := config{
				url:                "http://localhost:3000/",
				dumpSWML:           true,
				simulateServerless: platform,
			}
			err := run(cfg)
			if err == nil {
				t.Fatalf("expected error for --simulate-serverless %s", platform)
			}
			if !strings.Contains(err.Error(), "Phase 9") && !strings.Contains(err.Error(), "not implemented") {
				t.Errorf("error for %s should reference Phase 9 or 'not implemented'; got: %v", platform, err)
			}
		})
	}
}

// TestRun_SimulateServerless_RejectsUnknownPlatform ensures typos
// surface as "unknown platform" errors.
func TestRun_SimulateServerless_RejectsUnknownPlatform(t *testing.T) {
	cfg := config{
		url:                "http://localhost:3000/",
		dumpSWML:           true,
		simulateServerless: "lamda", // typo
	}
	err := run(cfg)
	if err == nil {
		t.Fatal("expected error for unknown platform")
	}
	if !strings.Contains(err.Error(), "unknown platform") {
		t.Errorf("error should say 'unknown platform'; got: %v", err)
	}
}

// TestRun_SimulateServerless_SetsAndRestoresEnvForURLMode simulates
// the CLI flow with --url + --simulate-serverless lambda pointed at
// an httptest server. The server asserts the Lambda env vars were
// set when the request arrived; after run() returns the test
// verifies those vars are restored (i.e. unset) in the outer process.
func TestRun_SimulateServerless_SetsAndRestoresEnvForURLMode(t *testing.T) {
	clearSimulatorEnv(t)

	// The server checks the env the *caller's* process has — not the
	// request headers — because that's what the CLI mutates. When the
	// CLI issues a GET to this server, the caller's process still
	// sees AWS_LAMBDA_FUNCTION_NAME set.
	var seenDuringRequest string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenDuringRequest = os.Getenv("AWS_LAMBDA_FUNCTION_NAME")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"version":"1.0.0","sections":{"main":[]}}`))
	}))
	defer srv.Close()

	cfg := config{
		url:                srv.URL,
		dumpSWML:           true,
		simulateServerless: "lambda",
	}
	if err := run(cfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	if seenDuringRequest == "" {
		t.Errorf("during simulated run, AWS_LAMBDA_FUNCTION_NAME was empty; simulator didn't activate env")
	}

	// After run() returns, env must be restored.
	if got := os.Getenv("AWS_LAMBDA_FUNCTION_NAME"); got != "" {
		t.Errorf("after run AWS_LAMBDA_FUNCTION_NAME = %q, want cleared", got)
	}
}

// TestRun_SimulateServerless_RestoresEnvOnURLError drives the CLI
// against a URL that returns 500, confirms run() reports the error,
// and checks env vars were rolled back.
func TestRun_SimulateServerless_RestoresEnvOnURLError(t *testing.T) {
	clearSimulatorEnv(t)
	t.Setenv("SWML_PROXY_URL_BASE", "https://outer.example.com")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := config{
		url:                srv.URL,
		dumpSWML:           true,
		simulateServerless: "lambda",
	}
	if err := run(cfg); err == nil {
		t.Fatal("expected run() to return an error for HTTP 500")
	}

	// Env still restored despite error.
	if got := os.Getenv("SWML_PROXY_URL_BASE"); got != "https://outer.example.com" {
		t.Errorf("SWML_PROXY_URL_BASE after error = %q, want restored", got)
	}
	if got := os.Getenv("AWS_LAMBDA_FUNCTION_NAME"); got != "" {
		t.Errorf("AWS_LAMBDA_FUNCTION_NAME after error = %q, want empty", got)
	}
}

// TestRun_SimulateServerless_NoURLErrorsCleanly ensures that the
// "Go CLI without an agent source" case fails fast with a helpful
// pointer to the library API, rather than silently succeeding.
func TestRun_SimulateServerless_NoURLErrorsCleanly(t *testing.T) {
	cfg := config{
		simulateServerless: "lambda",
		dumpSWML:           true,
	}
	err := run(cfg)
	if err == nil {
		t.Fatal("expected error when --simulate-serverless is used without --url")
	}
	if !strings.Contains(err.Error(), "--url") {
		t.Errorf("error should mention --url; got: %v", err)
	}
	if !strings.Contains(err.Error(), "SimulateDumpSWMLViaLambda") {
		t.Errorf("error should point at the library API; got: %v", err)
	}
}

// TestRun_SimulateServerless_DefaultsToDumpSWML verifies that running
// `--simulate-serverless lambda --url ...` with no explicit sub-action
// falls back to --dump-swml mode (matches the Python CLI's "bare
// simulate" behaviour).
func TestRun_SimulateServerless_DefaultsToDumpSWML(t *testing.T) {
	clearSimulatorEnv(t)

	var served bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served = true
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"version":"1.0.0","sections":{"main":[]}}`))
	}))
	defer srv.Close()

	cfg := config{
		url:                srv.URL,
		simulateServerless: "lambda",
		// No --dump-swml / --list-tools / --exec on purpose.
	}
	if err := run(cfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !served {
		t.Error("expected the test server to be hit under default (dump-swml) mode")
	}
}

// ---------------------------------------------------------------------------
// Flag parser coverage for the new flag
// ---------------------------------------------------------------------------

func TestParseFlags_SimulateServerless(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "lambda flag",
			args: []string{"--url", "http://localhost/", "--simulate-serverless", "lambda", "--dump-swml"},
			want: "lambda",
		},
		{
			name: "not present",
			args: []string{"--url", "http://localhost/", "--dump-swml"},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := parseFlags(tt.args)
			if cfg.simulateServerless != tt.want {
				t.Errorf("simulateServerless = %q, want %q", cfg.simulateServerless, tt.want)
			}
		})
	}
}
