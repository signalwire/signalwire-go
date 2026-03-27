// Example: multi_agent_server
//
// Hosts multiple AI agents on a single HTTP server using AgentServer.
// Each agent gets its own route and has unique prompts and tools.
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/server"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

func main() {
	// ---- Support Agent ----
	support := agent.NewAgentBase(
		agent.WithName("SupportAgent"),
		agent.WithRoute("/support"),
	)
	support.SetPromptText(
		"You are a technical support agent. Help users troubleshoot issues " +
			"with their accounts and devices. Be patient and thorough.",
	)
	support.AddHints([]string{"reset", "password", "login", "error", "broken"})

	support.DefineTool(agent.ToolDefinition{
		Name:        "lookup_account",
		Description: "Look up a customer account by email address",
		Parameters: map[string]any{
			"email": map[string]any{
				"type":        "string",
				"description": "Customer email address",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			email, _ := args["email"].(string)
			return swaig.NewFunctionResult(
				fmt.Sprintf("Account found for %s: Active subscription, last login 2 days ago.", email),
			)
		},
	})

	support.DefineTool(agent.ToolDefinition{
		Name:        "create_ticket",
		Description: "Create a support ticket for the customer",
		Parameters: map[string]any{
			"subject": map[string]any{
				"type":        "string",
				"description": "Ticket subject",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Detailed issue description",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			subject, _ := args["subject"].(string)
			return swaig.NewFunctionResult(
				fmt.Sprintf("Support ticket created: '%s' (Ticket #TK-4821). A specialist will follow up within 24 hours.", subject),
			)
		},
	})

	// ---- Sales Agent ----
	sales := agent.NewAgentBase(
		agent.WithName("SalesAgent"),
		agent.WithRoute("/sales"),
	)
	sales.SetPromptText(
		"You are a friendly sales agent. Help customers understand our products, " +
			"provide pricing, and guide them through the purchase process.",
	)
	sales.AddHints([]string{"pricing", "discount", "purchase", "plan", "upgrade"})

	sales.DefineTool(agent.ToolDefinition{
		Name:        "get_pricing",
		Description: "Get pricing information for a product plan",
		Parameters: map[string]any{
			"plan": map[string]any{
				"type":        "string",
				"description": "Plan name (basic, pro, enterprise)",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			plan, _ := args["plan"].(string)
			prices := map[string]string{
				"basic":      "$9.99/month",
				"pro":        "$29.99/month",
				"enterprise": "$99.99/month",
			}
			price, ok := prices[plan]
			if !ok {
				return swaig.NewFunctionResult("Available plans: basic ($9.99/mo), pro ($29.99/mo), enterprise ($99.99/mo).")
			}
			return swaig.NewFunctionResult(
				fmt.Sprintf("The %s plan is %s. Would you like to proceed?", plan, price),
			)
		},
	})

	// ---- Billing Agent ----
	billing := agent.NewAgentBase(
		agent.WithName("BillingAgent"),
		agent.WithRoute("/billing"),
	)
	billing.SetPromptText(
		"You are a billing specialist. Help customers with invoices, payments, " +
			"refunds, and subscription management.",
	)
	billing.AddHints([]string{"invoice", "payment", "refund", "charge", "billing"})

	billing.DefineTool(agent.ToolDefinition{
		Name:        "check_balance",
		Description: "Check the account balance for a customer",
		Parameters: map[string]any{
			"account_id": map[string]any{
				"type":        "string",
				"description": "The customer account ID",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			accountID, _ := args["account_id"].(string)
			return swaig.NewFunctionResult(
				fmt.Sprintf("Account %s: Current balance is $47.98. Next payment of $29.99 due on the 15th.", accountID),
			)
		},
	})

	// ---- Create and run the server ----
	srv := server.NewAgentServer(
		server.WithServerPort(3003),
	)

	srv.Register(support, "/support")
	srv.Register(sales, "/sales")
	srv.Register(billing, "/billing")

	fmt.Println("Starting multi-agent server on :3003")
	fmt.Println("  Support: /support")
	fmt.Println("  Sales:   /sales")
	fmt.Println("  Billing: /billing")

	if err := srv.Run(); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
