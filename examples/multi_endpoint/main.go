//go:build ignore

// Example: multi_endpoint
//
// Single agent with multiple SWML routes. Demonstrates using an
// AgentServer to host one agent alongside additional custom endpoints
// like a health API and a simple JSON endpoint.
package main

import (
	"fmt"
	"time"

	"github.com/signalwire/signalwire-agents-go/pkg/agent"
	"github.com/signalwire/signalwire-agents-go/pkg/server"
	"github.com/signalwire/signalwire-agents-go/pkg/swaig"
)

func main() {
	// Create the voice AI agent at /swml
	a := agent.NewAgentBase(
		agent.WithName("MultiEndpoint"),
		agent.WithRoute("/swml"),
	)

	a.PromptAddSection("Role",
		"You are a helpful voice assistant with access to time information.",
		nil,
	)
	a.PromptAddSection("Instructions", "", []string{
		"Greet callers warmly",
		"Be concise in your responses",
		"Use the get_time tool when asked about the time",
	})

	a.AddLanguage(map[string]any{
		"name":  "English",
		"code":  "en-US",
		"voice": "rime.spore",
	})

	a.DefineTool(agent.ToolDefinition{
		Name:        "get_time",
		Description: "Get the current time",
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			now := time.Now().Format("03:04 PM MST")
			return swaig.NewFunctionResult(fmt.Sprintf("The current time is %s", now))
		},
	})

	// Use AgentServer to host the agent alongside other routes
	srv := server.NewAgentServer(
		server.WithServerPort(3020),
	)

	srv.Register(a, "/swml")

	fmt.Println("Starting MultiEndpoint server on :3020")
	fmt.Println("  Voice AI:  POST /swml")
	fmt.Println("  SWAIG:     POST /swml/swaig")
	fmt.Println("  Health:    GET  /health")
	fmt.Println("  Ready:     GET  /ready")
	fmt.Println("  Index:     GET  /")

	if err := srv.Run(); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
