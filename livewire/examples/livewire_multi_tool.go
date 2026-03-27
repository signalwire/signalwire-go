//go:build ignore

// Example: LiveWire agent with multiple function tools and RunContext.
//
// Demonstrates registering several tools on an agent, using RunContext
// to access agent state, and configuring session parameters like
// interruption control and endpointing delays.
//
// Run:
//
//	go run livewire_multi_tool.go
//
// Then point a SignalWire phone number at http://your-host:3000/
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/signalwire/signalwire-go/pkg/livewire"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

func main() {
	server := livewire.NewAgentServer()

	server.RTCSession(func(ctx *livewire.JobContext) {
		ctx.Connect()

		// Configure session with interruption and endpointing settings.
		// AllowInterruptions maps to barge configuration on SignalWire.
		// EndpointingDelay maps to end_of_speech_timeout.
		session := livewire.NewAgentSession(
			livewire.WithLLM("openai/gpt-4"),
			livewire.WithAllowInterruptions(true),
			livewire.WithMinEndpointingDelay(0.5),
			livewire.WithMaxEndpointingDelay(5.0),
		)

		agent := livewire.NewAgent(
			"You are a helpful customer support agent for Acme Corp. " +
				"You can check order status, look up product information, " +
				"and schedule callbacks. Be concise and helpful.",
		)

		// Tool 1: Check order status (LiveKit-style string handler)
		agent.FunctionTool("check_order", func(ctx *livewire.RunContext, orderID string) string {
			// Access userdata through RunContext if needed
			fmt.Printf("[tool] check_order called with: %s\n", orderID)
			return fmt.Sprintf("Order %s is currently in transit. Expected delivery: %s",
				orderID, time.Now().Add(48*time.Hour).Format("Jan 2, 2006"))
		}, livewire.WithDescription("Check the status of a customer order"),
			livewire.WithParameters(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"order_id": map[string]any{
						"type":        "string",
						"description": "The order ID to check, e.g. ORD-12345",
					},
				},
				"required": []string{"order_id"},
			}),
		)

		// Tool 2: Product lookup (SignalWire-native handler for full control)
		agent.FunctionTool("lookup_product", func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			query, _ := args["query"].(string)
			fmt.Printf("[tool] lookup_product called with: %s\n", query)

			// Simulate product search
			result := swaig.NewFunctionResult(fmt.Sprintf(
				"Found 3 products matching '%s': "+
					"1) Acme Widget Pro ($29.99) "+
					"2) Acme Widget Lite ($14.99) "+
					"3) Acme Widget Bundle ($39.99)",
				query,
			))
			return result
		}, livewire.WithDescription("Search for product information"),
			livewire.WithParameters(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Product name or keyword to search for",
					},
				},
				"required": []string{"query"},
			}),
		)

		// Tool 3: Schedule callback (LiveKit-style, no args)
		agent.FunctionTool("schedule_callback", func(ctx *livewire.RunContext) string {
			callbackTime := time.Now().Add(2 * time.Hour).Format("3:04 PM")
			return fmt.Sprintf("Callback scheduled for %s. An agent will call you back.", callbackTime)
		}, livewire.WithDescription("Schedule a callback from a human agent"))

		// Tool 4: Escalate to human (demonstrates map-based handler)
		agent.FunctionTool("escalate", func(args map[string]any) string {
			reason, _ := args["reason"].(string)
			department := "general support"
			if strings.Contains(strings.ToLower(reason), "billing") {
				department = "billing"
			} else if strings.Contains(strings.ToLower(reason), "technical") {
				department = "technical support"
			}
			return fmt.Sprintf("Transferring you to %s. Please hold.", department)
		}, livewire.WithDescription("Escalate the call to a human agent"),
			livewire.WithParameters(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"reason": map[string]any{
						"type":        "string",
						"description": "Reason for escalation",
					},
				},
				"required": []string{"reason"},
			}),
		)

		session.Start(ctx, agent)
		session.GenerateReply(
			livewire.WithReplyInstructions(
				"Welcome the caller to Acme Corp support. Ask how you can help them today.",
			),
		)
	})

	livewire.RunApp(server)
}
