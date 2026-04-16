//go:build ignore

// Example: lambda
//
// AWS Lambda deployment of a SignalWire AI agent, using the built-in
// pkg/lambda adapter. The same main.go can be compiled as a Lambda
// binary (GOOS=linux GOARCH=amd64 go build -o bootstrap) and deployed as
// a Lambda Function URL or behind an API Gateway v2 HTTP API.
//
// When running in Lambda, the SDK automatically detects the environment
// (via AWS_LAMBDA_FUNCTION_NAME or LAMBDA_TASK_ROOT) and generates
// webhook URLs that resolve to the running function — no extra proxy
// configuration is required. Set AWS_LAMBDA_FUNCTION_URL if your
// function URL does not match the default
// https://{function}.lambda-url.{region}.on.aws pattern.
package main

import (
	"fmt"
	"os"
	"time"

	awslambda "github.com/aws/aws-lambda-go/lambda"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/lambda"
	"github.com/signalwire/signalwire-go/pkg/swaig"
	"github.com/signalwire/signalwire-go/pkg/swml"
)

// Create the agent once at package load so it survives across Lambda
// invocations (warm starts). Holding any per-invocation state on the
// agent would defeat that caching — use request-scoped structures
// inside tool handlers instead.
var a = newAgent()

func newAgent() *agent.AgentBase {
	ag := agent.NewAgentBase(
		agent.WithName("LambdaAgent"),
		// Non-root route is the recommended default: it lets you host
		// multiple agents behind a single Function URL if you grow into
		// that deployment. The SDK appends /swaig and /post_prompt to
		// this route automatically.
		agent.WithRoute("/my-agent"),
		agent.WithPort(3016), // only used when running the agent locally
	)

	ag.AddLanguage(map[string]any{
		"name":  "English",
		"code":  "en-US",
		"voice": "rime.spore",
	})

	ag.PromptAddSection("Role",
		"You are a helpful AI assistant running in a serverless environment.",
		nil,
	)
	ag.PromptAddSection("Instructions", "", []string{
		"Greet users warmly and offer help",
		"Use the greet_user function when asked to greet someone",
		"Use the get_time function when asked about the current time",
	})

	ag.DefineTool(agent.ToolDefinition{
		Name:        "greet_user",
		Description: "Greet a user by name",
		Parameters: map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "The user's name",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			name, _ := args["name"].(string)
			if name == "" {
				name = "friend"
			}
			return swaig.NewFunctionResult(
				fmt.Sprintf("Hello %s! I'm running in a serverless environment.", name),
			)
		},
	})

	ag.DefineTool(agent.ToolDefinition{
		Name:        "get_time",
		Description: "Get the current time",
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult(
				fmt.Sprintf("Current time: %s", time.Now().Format(time.RFC3339)),
			)
		},
	})

	return ag
}

// handler is the Lambda-facing entry point. lambda.NewHandler wraps the
// agent's http.Router so every SWML and SWAIG request arriving as a
// Lambda Function URL invocation is handled by the same code that serves
// the local HTTP listener.
var handler = lambda.NewHandler(a.AsRouter())

func main() {
	// If we're running under the Lambda runtime, hand control to it.
	// Otherwise, start the agent locally so the same binary can be used
	// for iterative development.
	if swml.GetExecutionMode() == swml.ModeLambda {
		awslambda.Start(handler.HandleFunctionURL)
		return
	}

	fmt.Fprintln(os.Stderr, "Not running in Lambda; starting local HTTP server on :3016/my-agent ...")
	if err := a.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Agent error: %v\n", err)
		os.Exit(1)
	}
}
