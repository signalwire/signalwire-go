// Example: simple_dynamic_agent
//
// Per-request agent customization using a dynamic config callback.
// The callback inspects query parameters to determine the caller's tier
// (standard vs. premium) and adjusts the prompt, global data, and tools.
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-agents-go/pkg/agent"
	"github.com/signalwire/signalwire-agents-go/pkg/swaig"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("DynamicAgent"),
		agent.WithRoute("/dynamic"),
		agent.WithPort(3002),
	)

	// Base prompt (applies to all tiers)
	a.SetPromptText("You are a customer support agent.")

	// Register the dynamic config callback
	a.SetDynamicConfigCallback(func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ephemeral *agent.AgentBase) {
		tier := queryParams["tier"]
		if tier == "" {
			tier = "standard"
		}

		switch tier {
		case "premium":
			// Premium users get a more capable prompt and extra tools
			ephemeral.SetPromptText(
				"You are a premium-tier concierge agent. Provide white-glove service, " +
					"proactive suggestions, and priority issue resolution.",
			)
			ephemeral.SetGlobalData(map[string]any{
				"tier":              "premium",
				"priority_queue":    true,
				"discount_eligible": true,
			})
			ephemeral.SetParam("temperature", 0.5)

			// Premium-only tool: schedule a callback
			ephemeral.DefineTool(agent.ToolDefinition{
				Name:        "schedule_callback",
				Description: "Schedule a priority callback from a specialist",
				Parameters: map[string]any{
					"time_slot": map[string]any{
						"type":        "string",
						"description": "Preferred callback time (e.g., '2pm today')",
					},
				},
				Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
					slot, _ := args["time_slot"].(string)
					return swaig.NewFunctionResult(
						fmt.Sprintf("Priority callback scheduled for %s. A specialist will call you.", slot),
					)
				},
			})

		default:
			// Standard tier
			ephemeral.SetPromptText(
				"You are a standard support agent. Help the customer with common questions " +
					"and direct them to the right resources.",
			)
			ephemeral.SetGlobalData(map[string]any{
				"tier":              "standard",
				"priority_queue":    false,
				"discount_eligible": false,
			})
			ephemeral.SetParam("temperature", 0.3)
		}

		// Tool available to all tiers
		ephemeral.DefineTool(agent.ToolDefinition{
			Name:        "check_order_status",
			Description: "Check the status of an order by order number",
			Parameters: map[string]any{
				"order_number": map[string]any{
					"type":        "string",
					"description": "The order number to look up",
				},
			},
			Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
				orderNum, _ := args["order_number"].(string)
				return swaig.NewFunctionResult(
					fmt.Sprintf("Order %s is currently being processed and will ship within 2 business days.", orderNum),
				)
			},
		})
	})

	fmt.Println("Starting DynamicAgent on :3002/dynamic ...")
	fmt.Println("  Standard: POST /dynamic")
	fmt.Println("  Premium:  POST /dynamic?tier=premium")
	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
