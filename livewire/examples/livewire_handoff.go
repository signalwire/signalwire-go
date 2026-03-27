//go:build ignore

// Example: Multi-agent with AgentHandoff.
//
// Demonstrates how to use AgentHandoff to switch between different
// agents within a single session. The triage agent determines the
// caller's intent and hands off to a specialized agent.
//
// Run:
//
//	go run livewire_handoff.go
//
// Then point a SignalWire phone number at http://your-host:3000/
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/livewire"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// createSalesAgent builds a sales-focused agent.
func createSalesAgent() *livewire.Agent {
	agent := livewire.NewAgent(
		"You are a sales specialist at Acme Corp. You help customers " +
			"find the right products and process orders. Be enthusiastic " +
			"but not pushy. Always mention our satisfaction guarantee.",
	)

	agent.FunctionTool("get_pricing", func(ctx *livewire.RunContext, product string) string {
		fmt.Printf("[sales] get_pricing called for: %s\n", product)
		return fmt.Sprintf("The %s is available at $29.99/month or $299/year (save 17%%). "+
			"We also offer a 30-day free trial.", product)
	}, livewire.WithDescription("Get pricing for a product"))

	agent.FunctionTool("create_order", func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
		product, _ := args["product"].(string)
		plan, _ := args["plan"].(string)
		fmt.Printf("[sales] create_order: product=%s plan=%s\n", product, plan)
		return swaig.NewFunctionResult(fmt.Sprintf(
			"Order created! You're now subscribed to %s on the %s plan. "+
				"A confirmation email will arrive shortly.", product, plan,
		))
	}, livewire.WithDescription("Create an order for a product"),
		livewire.WithParameters(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"product": map[string]any{"type": "string", "description": "Product name"},
				"plan":    map[string]any{"type": "string", "description": "Pricing plan (monthly or annual)"},
			},
			"required": []string{"product", "plan"},
		}),
	)

	return agent
}

// createSupportAgent builds a technical support agent.
func createSupportAgent() *livewire.Agent {
	agent := livewire.NewAgent(
		"You are a technical support specialist at Acme Corp. You help " +
			"customers troubleshoot issues, reset passwords, and resolve " +
			"technical problems. Be patient and thorough.",
	)

	agent.FunctionTool("check_system_status", func(ctx *livewire.RunContext) string {
		fmt.Println("[support] check_system_status called")
		return "All systems operational. API uptime: 99.99%. No known issues."
	}, livewire.WithDescription("Check current system status"))

	agent.FunctionTool("reset_password", func(ctx *livewire.RunContext, email string) string {
		fmt.Printf("[support] reset_password called for: %s\n", email)
		return fmt.Sprintf("Password reset email sent to %s. "+
			"The link expires in 30 minutes.", email)
	}, livewire.WithDescription("Send a password reset email"))

	agent.FunctionTool("create_ticket", func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
		issue, _ := args["issue"].(string)
		priority, _ := args["priority"].(string)
		fmt.Printf("[support] create_ticket: issue=%s priority=%s\n", issue, priority)
		return swaig.NewFunctionResult(fmt.Sprintf(
			"Support ticket #TKT-%d created with %s priority. "+
				"Our team will respond within 4 hours.", 10042, priority,
		))
	}, livewire.WithDescription("Create a support ticket"),
		livewire.WithParameters(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"issue":    map[string]any{"type": "string", "description": "Description of the issue"},
				"priority": map[string]any{"type": "string", "description": "Priority level: low, medium, high"},
			},
			"required": []string{"issue", "priority"},
		}),
	)

	return agent
}

func main() {
	server := livewire.NewAgentServer()

	// Pre-build the specialized agents
	salesAgent := createSalesAgent()
	supportAgent := createSupportAgent()

	server.RTCSession(func(ctx *livewire.JobContext) {
		ctx.Connect()

		session := livewire.NewAgentSession(
			livewire.WithLLM("openai/gpt-4"),
			livewire.WithAllowInterruptions(true),
		)

		// Create the triage agent that routes to specialists
		triageAgent := livewire.NewAgent(
			"You are the front-desk receptionist at Acme Corp. Your job is to " +
				"understand what the caller needs and route them to the right department. " +
				"Ask clarifying questions if needed, then use the appropriate handoff tool.",
		)

		// Handoff to sales
		triageAgent.FunctionTool("handoff_to_sales", func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			fmt.Println("[triage] Handing off to sales agent")

			// Update the session to use the sales agent
			// In a production app, UpdateInstructions would switch the prompt
			// to the sales agent's instructions.
			session.Say("Let me connect you to our sales team.")

			result := swaig.NewFunctionResult("Connecting you to our sales team now.")
			// In a full implementation, AgentHandoff would trigger a session swap
			_ = livewire.AgentHandoff{Agent: salesAgent}
			return result
		}, livewire.WithDescription("Transfer the caller to the sales department"))

		// Handoff to support
		triageAgent.FunctionTool("handoff_to_support", func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			fmt.Println("[triage] Handing off to support agent")

			// Update the session to use the support agent
			// In a production app, UpdateInstructions would switch the prompt
			// to the support agent's instructions.
			session.Say("Let me connect you to our technical support team.")

			result := swaig.NewFunctionResult("Connecting you to our technical support team now.")
			_ = livewire.AgentHandoff{Agent: supportAgent}
			return result
		}, livewire.WithDescription("Transfer the caller to technical support"))

		session.Start(ctx, triageAgent)
		session.GenerateReply(
			livewire.WithReplyInstructions(
				"Welcome the caller to Acme Corp. Ask whether they need help " +
					"with sales, purchasing, or technical support.",
			),
		)
	})

	livewire.RunApp(server)
}
