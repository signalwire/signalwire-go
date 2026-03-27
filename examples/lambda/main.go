//go:build ignore

// Example: lambda
//
// Serverless Lambda handler pattern. Demonstrates how to structure an AI
// agent for deployment on AWS Lambda or similar serverless platforms.
// The agent is created at package level and exposes an HTTP handler
// that can be wrapped with an API Gateway adapter.
package main

import (
	"fmt"
	"time"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// Create the agent at package level so it is initialised once per Lambda
// cold start. In a real Lambda deployment you would wrap agent.AsRouter()
// with an API Gateway adapter (e.g. github.com/awslabs/aws-lambda-go-api-proxy).
var a = newAgent()

func newAgent() *agent.AgentBase {
	ag := agent.NewAgentBase(
		agent.WithName("LambdaAgent"),
		agent.WithRoute("/"),
		agent.WithPort(3016),
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
				fmt.Sprintf("Hello %s! I'm running in a serverless environment!", name),
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

// In a real Lambda deployment, you would expose the handler like:
//
//   import "github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
//   var handler = httpadapter.New(a.AsRouter()).ProxyWithContext
//
// For local testing, just run the agent directly.
func main() {
	fmt.Println("Starting LambdaAgent on :3016/ ...")
	fmt.Println("  In production, wrap a.AsRouter() with a Lambda adapter.")

	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
