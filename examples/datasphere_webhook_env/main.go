//go:build ignore

// Example: datasphere_webhook_env
//
// Traditional webhook-based DataSphere skill configured from environment
// variables. Compare with datasphere_serverless_env for the serverless approach.
//
// Required: SIGNALWIRE_PROJECT_ID, SIGNALWIRE_API_TOKEN, SIGNALWIRE_SPACE,
//           DATASPHERE_DOCUMENT_ID
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/signalwire/signalwire-go/pkg/agent"

	_ "github.com/signalwire/signalwire-go/pkg/skills/builtin"
)

func requireEnv(name string) string {
	val := os.Getenv(name)
	if val == "" {
		fmt.Printf("Error: Required environment variable %s is not set.\n", name)
		os.Exit(1)
	}
	return val
}

func main() {
	documentID := requireEnv("DATASPHERE_DOCUMENT_ID")

	count := 3
	if v := os.Getenv("DATASPHERE_COUNT"); v != "" {
		count, _ = strconv.Atoi(v)
	}

	distance := 4.0
	if v := os.Getenv("DATASPHERE_DISTANCE"); v != "" {
		distance, _ = strconv.ParseFloat(v, 64)
	}

	a := agent.NewAgentBase(
		agent.WithName("DatasphereWebhookEnv"),
		agent.WithRoute("/datasphere-webhook"),
		agent.WithPort(3032),
	)

	a.AddLanguage(map[string]any{
		"name":  "English",
		"code":  "en-US",
		"voice": "rime.spore",
	})

	a.PromptAddSection("Role",
		"You are a knowledge assistant using webhook-based DataSphere "+
			"for document search and retrieval.",
		nil,
	)

	a.AddSkill("datetime", nil)
	a.AddSkill("math", nil)

	a.AddSkill("datasphere", map[string]any{
		"document_id": documentID,
		"count":       count,
		"distance":    distance,
		"mode":        "webhook",
	})

	fmt.Println("Starting DatasphereWebhookEnv on :3032/datasphere-webhook ...")
	fmt.Printf("  Document: %s\n", documentID)
	fmt.Println("  Execution: Webhook-based (traditional)")
	fmt.Println()
	fmt.Println("  Webhook: Full control, custom error handling, additional latency")
	fmt.Println("  Serverless: No webhooks needed, lower latency, executes on SignalWire")

	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
