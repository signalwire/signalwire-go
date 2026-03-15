//go:build ignore

// Example: kubernetes
//
// Kubernetes-ready agent with health and readiness endpoints. Demonstrates
// production deployment configuration with environment variable support,
// /health and /ready probes, and structured logging.
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/signalwire/signalwire-agents-go/pkg/agent"
	"github.com/signalwire/signalwire-agents-go/pkg/swaig"
)

func main() {
	port := 8080
	if envPort := os.Getenv("PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			port = p
		}
	}

	a := agent.NewAgentBase(
		agent.WithName("k8s-agent"),
		agent.WithRoute("/"),
		agent.WithHost("0.0.0.0"),
		agent.WithPort(port),
	)

	a.AddLanguage(map[string]any{
		"name":  "English",
		"code":  "en-US",
		"voice": "rime.spore",
	})

	a.PromptAddSection("Role",
		"You are a production-ready AI agent running in Kubernetes. "+
			"You can help users with general questions and demonstrate cloud-native deployment patterns.",
		nil,
	)

	// A tool that reports health status
	a.DefineTool(agent.ToolDefinition{
		Name:        "health_status",
		Description: "Get the health status of this agent",
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult(
				fmt.Sprintf("Agent %s is healthy, running on port %d in Kubernetes.", a.GetName(), port),
			)
		},
	})

	// Note: /health and /ready endpoints are automatically provided by the
	// underlying swml.Service HTTP server.

	fmt.Printf("Kubernetes-ready agent starting on port %d\n", port)
	fmt.Printf("  Health check: http://localhost:%d/health\n", port)
	fmt.Printf("  Readiness check: http://localhost:%d/ready\n", port)

	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
