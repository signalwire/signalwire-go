package util

import (
	"testing"
)

// envKeys lists every environment variable inspected by
// GetExecutionMode. Tests must clear all of them before asserting a
// specific mode so other CI / shell-leaked variables don't poison the
// result.
var envKeys = []string{
	"GATEWAY_INTERFACE",
	"AWS_LAMBDA_FUNCTION_NAME", "LAMBDA_TASK_ROOT",
	"FUNCTION_TARGET", "K_SERVICE", "GOOGLE_CLOUD_PROJECT",
	"AZURE_FUNCTIONS_ENVIRONMENT", "FUNCTIONS_WORKER_RUNTIME",
	"AzureWebJobsStorage",
}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range envKeys {
		t.Setenv(k, "")
		// Use Unsetenv so the function's "" check returns false.
		// t.Setenv with empty value still records as set on some
		// shells; explicit unset is safer for our os.Getenv check.
		// Cleanup is automatic via t.Setenv harness.
	}
}

func TestGetExecutionMode_DefaultIsServer(t *testing.T) {
	clearEnv(t)
	if got := GetExecutionMode(); got != "server" {
		t.Errorf("GetExecutionMode() = %q, want %q", got, "server")
	}
}

func TestGetExecutionMode_CGI(t *testing.T) {
	clearEnv(t)
	t.Setenv("GATEWAY_INTERFACE", "CGI/1.1")
	if got := GetExecutionMode(); got != "cgi" {
		t.Errorf("GetExecutionMode() = %q, want %q", got, "cgi")
	}
}

func TestGetExecutionMode_LambdaViaFunctionName(t *testing.T) {
	clearEnv(t)
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "my-fn")
	if got := GetExecutionMode(); got != "lambda" {
		t.Errorf("GetExecutionMode() = %q, want %q", got, "lambda")
	}
}

func TestGetExecutionMode_LambdaViaTaskRoot(t *testing.T) {
	clearEnv(t)
	t.Setenv("LAMBDA_TASK_ROOT", "/var/task")
	if got := GetExecutionMode(); got != "lambda" {
		t.Errorf("GetExecutionMode() = %q, want %q", got, "lambda")
	}
}

func TestGetExecutionMode_GoogleCloudFunction(t *testing.T) {
	clearEnv(t)
	t.Setenv("FUNCTION_TARGET", "my_handler")
	if got := GetExecutionMode(); got != "google_cloud_function" {
		t.Errorf("GetExecutionMode() = %q, want %q", got, "google_cloud_function")
	}
}

func TestGetExecutionMode_AzureFunction(t *testing.T) {
	clearEnv(t)
	t.Setenv("AZURE_FUNCTIONS_ENVIRONMENT", "Production")
	if got := GetExecutionMode(); got != "azure_function" {
		t.Errorf("GetExecutionMode() = %q, want %q", got, "azure_function")
	}
}

// Cross-language precedence contract: CGI beats Lambda when both are
// set. All ports must agree.
func TestGetExecutionMode_CGIBeatsLambda(t *testing.T) {
	clearEnv(t)
	t.Setenv("GATEWAY_INTERFACE", "CGI/1.1")
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "my-fn")
	if got := GetExecutionMode(); got != "cgi" {
		t.Errorf("GetExecutionMode() = %q, want %q (CGI must win over Lambda)", got, "cgi")
	}
}

func TestIsServerlessMode_ServerIsFalse(t *testing.T) {
	clearEnv(t)
	if IsServerlessMode() {
		t.Errorf("IsServerlessMode() = true, want false in default 'server' mode")
	}
}

func TestIsServerlessMode_LambdaIsTrue(t *testing.T) {
	clearEnv(t)
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "my-fn")
	if !IsServerlessMode() {
		t.Errorf("IsServerlessMode() = false, want true in lambda mode")
	}
}

func TestIsServerlessMode_CGIIsTrue(t *testing.T) {
	// CGI is short-lived per request — counts as serverless.
	clearEnv(t)
	t.Setenv("GATEWAY_INTERFACE", "CGI/1.1")
	if !IsServerlessMode() {
		t.Errorf("IsServerlessMode() = false, want true in cgi mode")
	}
}
