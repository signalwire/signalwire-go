package swml

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Execution mode detection
// ---------------------------------------------------------------------------
//
// Every subtest clears the known serverless-detection env vars so that a
// developer running these tests inside an actual cloud runtime does not get
// spurious failures (e.g. a CI pipeline running on GKE where
// GOOGLE_CLOUD_PROJECT happens to be set). t.Setenv restores the prior value
// at the end of the test automatically.

// clearExecutionEnv zeroes every env var inspected by GetExecutionMode so
// each subtest starts from a deterministic baseline.
func clearExecutionEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"GATEWAY_INTERFACE",
		"AWS_LAMBDA_FUNCTION_NAME",
		"LAMBDA_TASK_ROOT",
		"AWS_LAMBDA_FUNCTION_URL",
		"AWS_REGION",
		"FUNCTION_TARGET",
		"K_SERVICE",
		"GOOGLE_CLOUD_PROJECT",
		"AZURE_FUNCTIONS_ENVIRONMENT",
		"FUNCTIONS_WORKER_RUNTIME",
		"AzureWebJobsStorage",
		"SWML_PROXY_URL_BASE",
	} {
		t.Setenv(k, "")
	}
}

func TestGetExecutionMode_DefaultIsServer(t *testing.T) {
	clearExecutionEnv(t)
	if got := GetExecutionMode(); got != ModeServer {
		t.Errorf("GetExecutionMode() = %q, want %q", got, ModeServer)
	}
}

func TestGetExecutionMode_DetectsCGI(t *testing.T) {
	clearExecutionEnv(t)
	t.Setenv("GATEWAY_INTERFACE", "CGI/1.1")
	if got := GetExecutionMode(); got != ModeCGI {
		t.Errorf("GetExecutionMode() = %q, want %q", got, ModeCGI)
	}
}

func TestGetExecutionMode_DetectsLambda_FunctionName(t *testing.T) {
	clearExecutionEnv(t)
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "my-func")
	if got := GetExecutionMode(); got != ModeLambda {
		t.Errorf("GetExecutionMode() = %q, want %q", got, ModeLambda)
	}
}

func TestGetExecutionMode_DetectsLambda_TaskRoot(t *testing.T) {
	clearExecutionEnv(t)
	t.Setenv("LAMBDA_TASK_ROOT", "/var/task")
	if got := GetExecutionMode(); got != ModeLambda {
		t.Errorf("GetExecutionMode() = %q, want %q", got, ModeLambda)
	}
}

func TestGetExecutionMode_DetectsGCF(t *testing.T) {
	tests := []string{"FUNCTION_TARGET", "K_SERVICE", "GOOGLE_CLOUD_PROJECT"}
	for _, env := range tests {
		t.Run(env, func(t *testing.T) {
			clearExecutionEnv(t)
			t.Setenv(env, "something")
			if got := GetExecutionMode(); got != ModeGoogleCloudFunction {
				t.Errorf("GetExecutionMode() = %q, want %q", got, ModeGoogleCloudFunction)
			}
		})
	}
}

func TestGetExecutionMode_DetectsAzure(t *testing.T) {
	tests := []string{
		"AZURE_FUNCTIONS_ENVIRONMENT",
		"FUNCTIONS_WORKER_RUNTIME",
		"AzureWebJobsStorage",
	}
	for _, env := range tests {
		t.Run(env, func(t *testing.T) {
			clearExecutionEnv(t)
			t.Setenv(env, "something")
			if got := GetExecutionMode(); got != ModeAzureFunction {
				t.Errorf("GetExecutionMode() = %q, want %q", got, ModeAzureFunction)
			}
		})
	}
}

// CGI takes precedence over every other serverless marker because CGI can be
// layered on top of any host — matching the Python SDK ordering.
func TestGetExecutionMode_CGIWinsOverLambda(t *testing.T) {
	clearExecutionEnv(t)
	t.Setenv("GATEWAY_INTERFACE", "CGI/1.1")
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "my-func")
	if got := GetExecutionMode(); got != ModeCGI {
		t.Errorf("GetExecutionMode() = %q, want %q (CGI should win)", got, ModeCGI)
	}
}

// ---------------------------------------------------------------------------
// Lambda base URL construction
// ---------------------------------------------------------------------------

func TestLambdaBaseURL_UsesExplicitFunctionURL(t *testing.T) {
	clearExecutionEnv(t)
	t.Setenv("AWS_LAMBDA_FUNCTION_URL", "https://abc123.lambda-url.us-west-2.on.aws")
	// Function name + region should be IGNORED when the explicit URL is present,
	// matching the Python reference implementation.
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "other-func")
	t.Setenv("AWS_REGION", "eu-central-1")

	got := lambdaBaseURL()
	want := "https://abc123.lambda-url.us-west-2.on.aws"
	if got != want {
		t.Errorf("lambdaBaseURL() = %q, want %q", got, want)
	}
}

func TestLambdaBaseURL_StripsTrailingSlashFromExplicitURL(t *testing.T) {
	clearExecutionEnv(t)
	t.Setenv("AWS_LAMBDA_FUNCTION_URL", "https://abc123.lambda-url.us-west-2.on.aws/")

	got := lambdaBaseURL()
	want := "https://abc123.lambda-url.us-west-2.on.aws"
	if got != want {
		t.Errorf("lambdaBaseURL() = %q, want %q", got, want)
	}
}

func TestLambdaBaseURL_BuildsFromFunctionNameAndRegion(t *testing.T) {
	clearExecutionEnv(t)
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "my-agent")
	t.Setenv("AWS_REGION", "eu-west-1")

	got := lambdaBaseURL()
	want := "https://my-agent.lambda-url.eu-west-1.on.aws"
	if got != want {
		t.Errorf("lambdaBaseURL() = %q, want %q", got, want)
	}
}

func TestLambdaBaseURL_DefaultsRegionToUSEast1(t *testing.T) {
	clearExecutionEnv(t)
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "my-agent")
	// AWS_REGION intentionally unset.

	got := lambdaBaseURL()
	want := "https://my-agent.lambda-url.us-east-1.on.aws"
	if got != want {
		t.Errorf("lambdaBaseURL() = %q, want %q", got, want)
	}
}

func TestLambdaBaseURL_EmptyIfNothingConfigured(t *testing.T) {
	clearExecutionEnv(t)
	if got := lambdaBaseURL(); got != "" {
		t.Errorf("lambdaBaseURL() = %q, want empty string", got)
	}
}

// ---------------------------------------------------------------------------
// GetFullURL integration with Lambda env vars
// ---------------------------------------------------------------------------
//
// These tests are the main contract. The Python and TS SDKs had a bug
// where the proxy short-circuit returned the bare proxy URL WITHOUT
// appending the agent's route, resulting in broken SWAIG callbacks. The Go
// SDK already has the correct behaviour and these tests pin it down so a
// future refactor cannot regress.

func TestGetFullURL_LambdaWithExplicitFunctionURL_RootRoute(t *testing.T) {
	clearExecutionEnv(t)
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "demo") // trip mode detection
	t.Setenv("AWS_LAMBDA_FUNCTION_URL", "https://xyz.lambda-url.us-east-1.on.aws")

	// WithRoute("/") is canonicalised to "" internally via
	// strings.TrimRight, matching the existing TestServiceGetFullURL
	// expectation for server mode. The result therefore has no trailing
	// slash. This is the intentional SDK contract.
	svc := NewService(WithName("demo"), WithRoute("/"))
	got := svc.GetFullURL(false)
	want := "https://xyz.lambda-url.us-east-1.on.aws"
	if got != want {
		t.Errorf("GetFullURL() = %q, want %q", got, want)
	}
}

func TestGetFullURL_LambdaWithExplicitFunctionURL_NonRootRoute(t *testing.T) {
	clearExecutionEnv(t)
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "demo")
	t.Setenv("AWS_LAMBDA_FUNCTION_URL", "https://xyz.lambda-url.us-east-1.on.aws")

	svc := NewService(WithName("demo"), WithRoute("/my-agent"))
	got := svc.GetFullURL(false)
	want := "https://xyz.lambda-url.us-east-1.on.aws/my-agent"
	if got != want {
		t.Errorf("GetFullURL() = %q, want %q", got, want)
	}
}

func TestGetFullURL_LambdaFallbackFromFunctionName(t *testing.T) {
	clearExecutionEnv(t)
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "demo-func")
	t.Setenv("AWS_REGION", "us-east-2")
	// AWS_LAMBDA_FUNCTION_URL intentionally unset: we want to exercise the
	// region+name fallback path.

	svc := NewService(WithName("demo"), WithRoute("/my-agent"))
	got := svc.GetFullURL(false)
	want := "https://demo-func.lambda-url.us-east-2.on.aws/my-agent"
	if got != want {
		t.Errorf("GetFullURL() = %q, want %q", got, want)
	}
}

// ROUTE-PRESERVATION REGRESSION GUARD. This is the bug that bit the Python
// and TypeScript SDKs: when SWML_PROXY_URL_BASE is set in a Lambda env, the
// returned URL MUST still include the agent's route. Without this test, a
// future refactor could reintroduce the "bare proxy URL" bug.
func TestGetFullURL_Regression_LambdaNonRootRouteWithProxyBase(t *testing.T) {
	clearExecutionEnv(t)
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "demo")
	t.Setenv("AWS_LAMBDA_FUNCTION_URL", "https://ignored.lambda-url.us-east-1.on.aws")
	t.Setenv("SWML_PROXY_URL_BASE", "https://proxy.example.com")

	svc := NewService(WithName("demo"), WithRoute("/my-agent"))
	got := svc.GetFullURL(false)
	want := "https://proxy.example.com/my-agent"
	if got != want {
		t.Fatalf(
			"route-preservation regression: GetFullURL() = %q, want %q. "+
				"The proxy base URL MUST have the agent's route appended.",
			got, want,
		)
	}
}

// Pairs with the regression test above: without a proxy base set, Lambda
// detection must STILL append the agent's route.
func TestGetFullURL_Regression_LambdaNonRootRouteWithoutProxyBase(t *testing.T) {
	clearExecutionEnv(t)
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "demo")
	t.Setenv("AWS_LAMBDA_FUNCTION_URL", "https://xxx.lambda-url.us-east-1.on.aws")

	svc := NewService(WithName("demo"), WithRoute("/my-agent"))
	got := svc.GetFullURL(false)
	want := "https://xxx.lambda-url.us-east-1.on.aws/my-agent"
	if got != want {
		t.Fatalf(
			"route-preservation regression: GetFullURL() = %q, want %q. "+
				"The Lambda function URL MUST have the agent's route appended.",
			got, want,
		)
	}
}

// Server mode (no Lambda env vars, no proxy) keeps the existing local-URL
// behaviour. Guards against the Lambda code path leaking into plain
// development use.
func TestGetFullURL_ServerModeUnchanged(t *testing.T) {
	clearExecutionEnv(t)

	svc := NewService(WithName("demo"), WithPort(3000), WithRoute("/agent"))
	got := svc.GetFullURL(false)
	want := "http://localhost:3000/agent"
	if got != want {
		t.Errorf("GetFullURL() = %q, want %q", got, want)
	}
}
