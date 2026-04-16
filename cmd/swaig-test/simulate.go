// Simulator support for `--simulate-serverless` mode.
//
// In the Python SDK, `swaig-test` can load an agent from a source file
// and dispatch invocations through the chosen serverless adapter
// in-process. Go has no equivalent dynamic-loader for compiled binaries,
// so the simulator is split in two:
//
//   1. A library API (this file) that sets/clears the mode-detection
//      env vars, dispatches a synthetic Lambda Function URL event
//      through pkg/lambda, and restores the outer environment on exit.
//      Tests and in-process callers (e.g. users who embed `swaig-test`
//      in their own test suites) drive the simulator through this API.
//
//   2. A flag on the `swaig-test` CLI (see main.go) that validates the
//      requested platform against what the port actually implements and
//      surfaces a clear error for unsupported platforms (Phase 9 of the
//      porting guide). The flag also works with --url: it sets the
//      mode-detection env vars for the duration of the invocation so
//      the server-side URL generation goes through the platform branch.
//
// The simulator mirrors the behaviour of Python's
// `signalwire/cli/simulation/mock_env.py`:
//   - Platform preset env vars are applied (AWS_LAMBDA_FUNCTION_NAME etc).
//   - Conflicting env vars — most importantly SWML_PROXY_URL_BASE — are
//     cleared so platform-specific URL generation is actually exercised.
//   - The original env is restored on exit, whether the simulated call
//     succeeded, errored, or panicked. Leaking env across simulations
//     would corrupt later tests in the same process.
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"

	"github.com/signalwire/signalwire-go/pkg/lambda"
)

// supportedSimulatePlatforms is the set of platforms that `swaig-test
// --simulate-serverless` accepts. The list matches what the Go port
// implemented in Phase 9 of the porting guide. Adding an entry here
// without a corresponding adapter in pkg/ is a lie that will
// misleadingly pass `--simulate-serverless <new-platform>` calls.
var supportedSimulatePlatforms = map[string]bool{
	"lambda": true,
}

// notYetImplementedSimulatePlatforms enumerates platforms the porting
// guide mentions but the Go port has not yet shipped. Listing them
// explicitly lets the CLI give a targeted error ("not yet implemented")
// instead of a generic "unknown platform" — matches the guide's
// requirement that unimplemented platforms surface a clear error rather
// than silently falling back to the server path.
var notYetImplementedSimulatePlatforms = map[string]bool{
	"gcf":             true, // Google Cloud Functions
	"cloud_function":  true, // alternate Python name
	"azure":           true,
	"azure_function":  true,
	"cgi":             true,
}

// validateSimulatePlatform checks that the given platform name is
// supported by this port. It returns nil if the platform is usable,
// and a descriptive error otherwise. The error message points at
// Phase 9 of the porting guide so users can tell at a glance whether
// they're looking at a typo or a missing implementation.
func validateSimulatePlatform(platform string) error {
	if platform == "" {
		return fmt.Errorf("--simulate-serverless requires a platform name (e.g. \"lambda\")")
	}
	if supportedSimulatePlatforms[platform] {
		return nil
	}
	if notYetImplementedSimulatePlatforms[platform] {
		return fmt.Errorf(
			"--simulate-serverless %s: platform not implemented in this port. "+
				"Phase 9 of the porting guide has only been completed for: %s. "+
				"To use --simulate-serverless %s, implement the corresponding "+
				"adapter under pkg/ first.",
			platform, supportedPlatformList(), platform,
		)
	}
	return fmt.Errorf(
		"--simulate-serverless %s: unknown platform. Supported: %s",
		platform, supportedPlatformList(),
	)
}

// supportedPlatformList returns a human-readable comma-separated list
// of platforms the port supports, for inclusion in error messages.
func supportedPlatformList() string {
	names := make([]string, 0, len(supportedSimulatePlatforms))
	for name := range supportedSimulatePlatforms {
		names = append(names, name)
	}
	// Tiny list — skip sort to avoid pulling in sort package; the
	// iteration order is fine for the one-element case.
	return strings.Join(names, ", ")
}

// envSnapshot captures the values (and presence) of a fixed set of
// env vars so they can be restored verbatim, including distinguishing
// "unset" from "set to empty string".
type envSnapshot struct {
	values map[string]*string // nil pointer == unset, empty string == set empty
}

// snapshotEnv captures the current values of the given env vars. A key
// that is unset is represented by a nil pointer; a key that is set
// (even to the empty string) is represented by a non-nil pointer.
func snapshotEnv(keys []string) envSnapshot {
	snap := envSnapshot{values: make(map[string]*string, len(keys))}
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok {
			vCopy := v
			snap.values[k] = &vCopy
		} else {
			snap.values[k] = nil
		}
	}
	return snap
}

// restore puts every captured key back to its pre-activation state.
// Called from a deferred closure so it runs on both the happy path
// and on error paths (including panics).
func (s envSnapshot) restore() {
	for k, v := range s.values {
		if v == nil {
			os.Unsetenv(k)
		} else {
			os.Setenv(k, *v)
		}
	}
}

// lambdaPresetEnv is the set of env vars set when activating a Lambda
// simulation. Values are tame defaults — they're only used for URL
// construction and mode detection, not for any real AWS API call.
//
// AWS_LAMBDA_FUNCTION_URL is deliberately omitted so GetFullURL's
// fallback (construct from function name + region) is the path
// exercised by default. Callers that need a specific URL can override
// it via SimulateLambdaOptions.
func lambdaPresetEnv() map[string]string {
	return map[string]string{
		"AWS_LAMBDA_FUNCTION_NAME": "test-agent-function",
		"LAMBDA_TASK_ROOT":         "/var/task",
		"AWS_REGION":               "us-east-1",
		"_HANDLER":                 "bootstrap",
	}
}

// managedEnvKeys returns every env var the simulator touches. These
// are the keys captured by snapshotEnv so their original values — or
// unset state — can be faithfully restored on exit.
//
// We include all of the AWS preset keys, the keys for other platforms
// (so SimulateLambda can clear them defensively), and SWML_PROXY_URL_BASE
// which Python's mock_env.py explicitly clears during any simulation.
func managedEnvKeys() []string {
	return []string{
		// Lambda
		"AWS_LAMBDA_FUNCTION_NAME",
		"LAMBDA_TASK_ROOT",
		"AWS_REGION",
		"AWS_LAMBDA_FUNCTION_URL",
		"_HANDLER",
		// CGI (cleared defensively)
		"GATEWAY_INTERFACE",
		"HTTP_HOST",
		"SCRIPT_NAME",
		// Google Cloud Functions (cleared defensively)
		"FUNCTION_TARGET",
		"K_SERVICE",
		"GOOGLE_CLOUD_PROJECT",
		// Azure Functions (cleared defensively)
		"AZURE_FUNCTIONS_ENVIRONMENT",
		"FUNCTIONS_WORKER_RUNTIME",
		"AzureWebJobsStorage",
		// Proxy override — cleared during simulation (matches mock_env.py)
		"SWML_PROXY_URL_BASE",
	}
}

// SimulateLambdaOptions tunes an in-process Lambda simulation.
type SimulateLambdaOptions struct {
	// FunctionURLOverride, if non-empty, is assigned to
	// AWS_LAMBDA_FUNCTION_URL during the simulation. The default
	// (empty) lets GetFullURL fall back to constructing the URL from
	// AWS_LAMBDA_FUNCTION_NAME + AWS_REGION, which is the more
	// interesting code path to exercise in tests.
	FunctionURLOverride string

	// FunctionName overrides AWS_LAMBDA_FUNCTION_NAME. Empty means
	// use the preset default ("test-agent-function").
	FunctionName string

	// Region overrides AWS_REGION. Empty means use the preset default
	// ("us-east-1").
	Region string

	// Logger receives warnings about env state — notably if
	// SWML_PROXY_URL_BASE is still set after the clear attempt. A nil
	// Logger sends warnings to os.Stderr.
	Logger func(format string, args ...any)
}

// activateLambdaEnv applies the Lambda-preset env vars, clears the
// conflicting ones, and returns the snapshot of the pre-simulation
// environment. Callers MUST call snapshot.restore() (typically via
// defer) to put the environment back once the simulation is done.
//
// The function deliberately returns a value type so it can't be
// accidentally shared between goroutines — simulations are one-at-a-
// time in a single process.
func activateLambdaEnv(opts SimulateLambdaOptions) envSnapshot {
	keys := managedEnvKeys()
	snap := snapshotEnv(keys)

	// Clear everything first so leftover state from a previous
	// simulation (or the outer shell, for SWML_PROXY_URL_BASE) can't
	// leak into the new simulation.
	for _, k := range keys {
		os.Unsetenv(k)
	}

	// Apply presets.
	preset := lambdaPresetEnv()
	if opts.FunctionName != "" {
		preset["AWS_LAMBDA_FUNCTION_NAME"] = opts.FunctionName
	}
	if opts.Region != "" {
		preset["AWS_REGION"] = opts.Region
	}
	if opts.FunctionURLOverride != "" {
		preset["AWS_LAMBDA_FUNCTION_URL"] = opts.FunctionURLOverride
	}
	for k, v := range preset {
		os.Setenv(k, v)
	}

	// Belt-and-suspenders: if the outer environment or a cooperating
	// process re-set SWML_PROXY_URL_BASE between our Unsetenv and
	// this check, warn so the user understands why their URLs don't
	// look Lambda-style. Matches Python's mock_env.py warning.
	if proxy := os.Getenv("SWML_PROXY_URL_BASE"); proxy != "" {
		warn := opts.Logger
		if warn == nil {
			warn = func(format string, args ...any) {
				fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
			}
		}
		warn("SWML_PROXY_URL_BASE is still set (%q) after simulator clear; Lambda URL generation may be bypassed", proxy)
	}

	return snap
}

// LambdaSimResult is the decoded result of one synthetic Lambda
// invocation. Status is the HTTP status returned by the handler,
// Body is the response body (already base64-decoded if the
// adapter marked it so), and Headers mirrors the Lambda response
// envelope's flattened single-value map.
type LambdaSimResult struct {
	Status  int
	Body    []byte
	Headers map[string]string
}

// SimulateLambdaInvocation runs a single synthetic Lambda Function
// URL event against the given handler with the Lambda environment
// active. It does NOT touch env vars itself — call activateLambdaEnv
// first and defer restore() so the env change has the right scope.
//
// The split exists so one activation can host multiple invocations
// (dump-SWML then exec-tool, say) without paying the env save/restore
// cost per call.
func SimulateLambdaInvocation(
	handler http.Handler,
	method, path string,
	headers map[string]string,
	body io.Reader,
) (LambdaSimResult, error) {
	if handler == nil {
		return LambdaSimResult{}, fmt.Errorf("simulate-lambda: handler must not be nil")
	}
	if method == "" {
		method = http.MethodGet
	}
	if path == "" {
		path = "/"
	}

	var bodyStr string
	if body != nil {
		raw, err := io.ReadAll(body)
		if err != nil {
			return LambdaSimResult{}, fmt.Errorf("simulate-lambda: reading body: %w", err)
		}
		bodyStr = string(raw)
	}

	// Construct a synthetic Function URL event. We only populate the
	// fields the adapter actually reads; Lambda injects many more
	// in real invocations, none of which the SDK looks at.
	evt := events.LambdaFunctionURLRequest{
		RawPath: path,
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: method},
		},
		Headers: headers,
		Body:    bodyStr,
	}

	adapter := lambda.NewHandler(handler)
	resp, err := adapter.HandleFunctionURL(context.Background(), evt)
	if err != nil {
		return LambdaSimResult{}, fmt.Errorf("simulate-lambda: adapter returned error: %w", err)
	}

	// The adapter base64-encodes non-UTF-8 bodies; in this simulator
	// we don't care about the distinction for display, so decode into
	// raw bytes unconditionally.
	bodyBytes := []byte(resp.Body)
	if resp.IsBase64Encoded {
		// Fail loudly on malformed base64 — if the adapter emitted
		// base64 but it doesn't decode, something is broken in the
		// handler's response pipeline and hiding the error would
		// waste user time.
		decoded, err := base64Decode(resp.Body)
		if err != nil {
			return LambdaSimResult{}, fmt.Errorf("simulate-lambda: base64-decode response body: %w", err)
		}
		bodyBytes = decoded
	}

	return LambdaSimResult{
		Status:  resp.StatusCode,
		Body:    bodyBytes,
		Headers: resp.Headers,
	}, nil
}

// base64Decode is a thin wrapper around encoding/base64.StdEncoding
// kept exclusively to centralise error wrapping in one place.
func base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// basicAuthHeader constructs a Basic auth header value for the given
// credentials. Centralised so tests and CLI code agree on the format.
func basicAuthHeader(user, pass string) string {
	cred := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
	return "Basic " + cred
}

// HandlerFactory is a zero-arg function that constructs the
// http.Handler under test. The factory is called AFTER the simulator
// has activated the platform env vars, so any env-var-driven state
// the agent captures at construction (notably SWML_PROXY_URL_BASE,
// which pkg/swml.Service reads in its constructor) reflects the
// simulated environment rather than the outer shell.
//
// This mirrors the Python SDK's mock_env.py flow: env vars are set
// first, then the agent module is imported/loaded, then invocations
// run against that freshly-loaded agent.
type HandlerFactory func() http.Handler

// SimulateDumpSWMLViaLambda activates the Lambda environment, calls
// the factory to construct the agent (so any env-captured state
// reflects the simulated Lambda environment), issues a POST to the
// agent's route through the Lambda adapter, and returns the response
// body (the SWML document JSON). It's the library-side equivalent of
// `swaig-test --simulate-serverless lambda --dump-swml`, usable from
// in-process tests.
//
// The factory-based API is load-bearing: constructing the agent
// BEFORE activation would let SWML_PROXY_URL_BASE from the outer
// shell leak into the agent's own proxyURLBase field, and the
// rendered webhook URLs would point at the outer proxy instead of
// the simulated Lambda function. Use SimulateDumpSWMLViaLambdaHandler
// only when you've already verified the handler doesn't capture
// env state — it skips the activation-ordering guarantee.
//
// basicAuth, if non-empty in both fields, adds a basic-auth header
// to the synthetic event so authed agents don't 401.
func SimulateDumpSWMLViaLambda(
	factory HandlerFactory,
	agentRoute string,
	opts SimulateLambdaOptions,
	basicAuth BasicAuth,
) ([]byte, error) {
	if factory == nil {
		return nil, fmt.Errorf("simulate-lambda dump-swml: factory must not be nil")
	}
	if agentRoute == "" {
		agentRoute = "/"
	}

	snap := activateLambdaEnv(opts)
	defer snap.restore()

	// Build the handler AFTER env activation so any env-var-driven
	// state (e.g. agent.proxyURLBase, which is captured from
	// SWML_PROXY_URL_BASE at construction time) reflects the
	// simulated environment.
	handler := factory()
	if handler == nil {
		return nil, fmt.Errorf("simulate-lambda dump-swml: factory returned nil handler")
	}

	headers := map[string]string{
		"content-type": "application/json",
	}
	if basicAuth.User != "" || basicAuth.Password != "" {
		headers["authorization"] = basicAuthHeader(basicAuth.User, basicAuth.Password)
	}

	result, err := SimulateLambdaInvocation(
		handler, http.MethodPost, agentRoute, headers, strings.NewReader("{}"),
	)
	if err != nil {
		return nil, err
	}
	if result.Status < 200 || result.Status >= 300 {
		return nil, fmt.Errorf("simulate-lambda dump-swml: HTTP %d: %s", result.Status, string(result.Body))
	}
	return result.Body, nil
}

// SimulateExecToolViaLambda activates the Lambda environment, calls
// the factory to construct the agent (so any env-captured state
// reflects the simulated Lambda environment), and dispatches a SWAIG
// tool invocation through the Lambda adapter at `<agentRoute>/swaig`.
// Returns the raw response body.
func SimulateExecToolViaLambda(
	factory HandlerFactory,
	agentRoute, toolName string,
	args map[string]any,
	opts SimulateLambdaOptions,
	basicAuth BasicAuth,
) ([]byte, error) {
	if factory == nil {
		return nil, fmt.Errorf("simulate-lambda exec: factory must not be nil")
	}
	if agentRoute == "" {
		agentRoute = "/"
	}
	if toolName == "" {
		return nil, fmt.Errorf("simulate-lambda exec: tool name is required")
	}

	snap := activateLambdaEnv(opts)
	defer snap.restore()

	// See SimulateDumpSWMLViaLambda for why the handler is built
	// after env activation.
	handler := factory()
	if handler == nil {
		return nil, fmt.Errorf("simulate-lambda exec: factory returned nil handler")
	}

	payload := map[string]any{
		"function": toolName,
		"argument": args,
		"call_id":  "simulate-call-id",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("simulate-lambda exec: marshal payload: %w", err)
	}

	headers := map[string]string{
		"content-type": "application/json",
	}
	if basicAuth.User != "" || basicAuth.Password != "" {
		headers["authorization"] = basicAuthHeader(basicAuth.User, basicAuth.Password)
	}

	swaigPath := strings.TrimRight(agentRoute, "/") + "/swaig"
	if agentRoute == "/" {
		swaigPath = "/swaig"
	}

	result, err := SimulateLambdaInvocation(
		handler, http.MethodPost, swaigPath, headers, bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	if result.Status < 200 || result.Status >= 300 {
		return nil, fmt.Errorf("simulate-lambda exec: HTTP %d: %s", result.Status, string(result.Body))
	}
	return result.Body, nil
}

// BasicAuth bundles a username/password pair for convenience.
type BasicAuth struct {
	User     string
	Password string
}
