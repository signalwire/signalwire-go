// Package util — execution_mode helpers.
//
// GetExecutionMode and IsServerlessMode mirror the Python reference at
// signalwire.core.logging_config.get_execution_mode and
// signalwire.utils.is_serverless_mode. They detect the deployment
// environment (CGI, AWS Lambda, GCP Cloud Functions, Azure Functions,
// or long-lived server) by checking the same precedence-ordered
// environment variables every SDK port must observe.
//
// Cross-language SDK contract: order of precedence (FIRST match wins):
//
//  1. GATEWAY_INTERFACE                                          → "cgi"
//  2. AWS_LAMBDA_FUNCTION_NAME or LAMBDA_TASK_ROOT               → "lambda"
//  3. FUNCTION_TARGET, K_SERVICE, or GOOGLE_CLOUD_PROJECT        → "google_cloud_function"
//  4. AZURE_FUNCTIONS_ENVIRONMENT, FUNCTIONS_WORKER_RUNTIME, or
//     AzureWebJobsStorage                                        → "azure_function"
//  5. otherwise                                                   → "server"

package util

import "os"

// GetExecutionMode reports the SDK's deployment environment based on
// well-known environment variables. Returned values are:
// "cgi", "lambda", "google_cloud_function", "azure_function", "server".
func GetExecutionMode() string {
	if os.Getenv("GATEWAY_INTERFACE") != "" {
		return "cgi"
	}
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" || os.Getenv("LAMBDA_TASK_ROOT") != "" {
		return "lambda"
	}
	if os.Getenv("FUNCTION_TARGET") != "" ||
		os.Getenv("K_SERVICE") != "" ||
		os.Getenv("GOOGLE_CLOUD_PROJECT") != "" {
		return "google_cloud_function"
	}
	if os.Getenv("AZURE_FUNCTIONS_ENVIRONMENT") != "" ||
		os.Getenv("FUNCTIONS_WORKER_RUNTIME") != "" ||
		os.Getenv("AzureWebJobsStorage") != "" {
		return "azure_function"
	}
	return "server"
}

// IsServerlessMode reports whether the SDK is running in any
// serverless invocation environment (i.e. not "server").
func IsServerlessMode() bool {
	return GetExecutionMode() != "server"
}
