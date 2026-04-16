package swml

import "os"

// ExecutionMode identifies the runtime environment the service is executing in.
// The value is used to adjust URL construction, request parsing, and auth
// handling for platforms that do not provide a traditional TCP listener.
type ExecutionMode string

const (
	// ModeServer is the default long-running HTTP server mode.
	ModeServer ExecutionMode = "server"
	// ModeCGI indicates the process was invoked by a CGI host
	// (detected via GATEWAY_INTERFACE).
	ModeCGI ExecutionMode = "cgi"
	// ModeLambda indicates the process is running inside AWS Lambda
	// (detected via AWS_LAMBDA_FUNCTION_NAME or LAMBDA_TASK_ROOT).
	ModeLambda ExecutionMode = "lambda"
	// ModeGoogleCloudFunction indicates a Google Cloud Functions
	// runtime (detected via FUNCTION_TARGET, K_SERVICE, or
	// GOOGLE_CLOUD_PROJECT).
	ModeGoogleCloudFunction ExecutionMode = "google_cloud_function"
	// ModeAzureFunction indicates an Azure Functions runtime
	// (detected via AZURE_FUNCTIONS_ENVIRONMENT, FUNCTIONS_WORKER_RUNTIME,
	// or AzureWebJobsStorage).
	ModeAzureFunction ExecutionMode = "azure_function"
)

// GetExecutionMode inspects the process environment and returns the detected
// runtime mode. The detection order matches the Python and TypeScript SDKs so
// that the same env vars resolve to the same mode across languages.
//
// Detection order:
//  1. CGI         (GATEWAY_INTERFACE)
//  2. AWS Lambda  (AWS_LAMBDA_FUNCTION_NAME or LAMBDA_TASK_ROOT)
//  3. GCF         (FUNCTION_TARGET, K_SERVICE, or GOOGLE_CLOUD_PROJECT)
//  4. Azure       (AZURE_FUNCTIONS_ENVIRONMENT, FUNCTIONS_WORKER_RUNTIME,
//     or AzureWebJobsStorage)
//  5. Server      (default fallback)
func GetExecutionMode() ExecutionMode {
	if os.Getenv("GATEWAY_INTERFACE") != "" {
		return ModeCGI
	}
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" || os.Getenv("LAMBDA_TASK_ROOT") != "" {
		return ModeLambda
	}
	if os.Getenv("FUNCTION_TARGET") != "" ||
		os.Getenv("K_SERVICE") != "" ||
		os.Getenv("GOOGLE_CLOUD_PROJECT") != "" {
		return ModeGoogleCloudFunction
	}
	if os.Getenv("AZURE_FUNCTIONS_ENVIRONMENT") != "" ||
		os.Getenv("FUNCTIONS_WORKER_RUNTIME") != "" ||
		os.Getenv("AzureWebJobsStorage") != "" {
		return ModeAzureFunction
	}
	return ModeServer
}

// lambdaBaseURL returns the Lambda-specific base URL (scheme+host, no route)
// using AWS_LAMBDA_FUNCTION_URL if present, otherwise reconstructing the
// canonical Function URL from AWS_REGION and AWS_LAMBDA_FUNCTION_NAME.
//
// The returned string has any trailing slash stripped so that the caller can
// safely concatenate the agent's route onto it. An empty string is returned
// only if the environment does not supply enough information to build a URL
// (no function name AND no explicit URL), which should never happen in a real
// Lambda runtime.
func lambdaBaseURL() string {
	if explicit := os.Getenv("AWS_LAMBDA_FUNCTION_URL"); explicit != "" {
		return trimTrailingSlash(explicit)
	}
	functionName := os.Getenv("AWS_LAMBDA_FUNCTION_NAME")
	if functionName == "" {
		return ""
	}
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}
	return "https://" + functionName + ".lambda-url." + region + ".on.aws"
}

// trimTrailingSlash removes a single trailing "/" if present. Used rather
// than strings.TrimRight because we only want to strip one slash, not all
// trailing slashes (though in practice that distinction does not matter for
// AWS Lambda Function URL values).
func trimTrailingSlash(s string) string {
	if len(s) > 0 && s[len(s)-1] == '/' {
		return s[:len(s)-1]
	}
	return s
}
