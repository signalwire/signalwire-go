// Example: contexts_demo
//
// Multi-step conversation workflows using contexts and steps.
// Demonstrates creating multiple contexts with sequential steps,
// step criteria, navigation rules, function restrictions, and
// per-context language/voice configuration.
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("ContextsDemo"),
		agent.WithRoute("/contexts"),
		agent.WithPort(3004),
	)

	// Base prompt that applies across all contexts
	a.SetPromptText(
		"You are a versatile assistant that can handle both sales inquiries " +
			"and technical support. Follow the structured workflow for each context.",
	)

	// Define tools used across contexts
	a.DefineTool(agent.ToolDefinition{
		Name:        "check_inventory",
		Description: "Check product inventory levels",
		Parameters: map[string]any{
			"product": map[string]any{
				"type":        "string",
				"description": "Product name to check",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			product, _ := args["product"].(string)
			return swaig.NewFunctionResult(
				fmt.Sprintf("%s: 142 units in stock, ready to ship.", product),
			)
		},
	})

	a.DefineTool(agent.ToolDefinition{
		Name:        "create_order",
		Description: "Create a new order for a product",
		Parameters: map[string]any{
			"product": map[string]any{
				"type":        "string",
				"description": "Product to order",
			},
			"quantity": map[string]any{
				"type":        "integer",
				"description": "Number of units",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			product, _ := args["product"].(string)
			return swaig.NewFunctionResult(
				fmt.Sprintf("Order created for %s. Order #ORD-7823 confirmed.", product),
			)
		},
	})

	a.DefineTool(agent.ToolDefinition{
		Name:        "run_diagnostic",
		Description: "Run a diagnostic check on a device",
		Parameters: map[string]any{
			"device_id": map[string]any{
				"type":        "string",
				"description": "Device serial number",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			deviceID, _ := args["device_id"].(string)
			return swaig.NewFunctionResult(
				fmt.Sprintf("Diagnostic for device %s: All systems nominal. Firmware v2.1.4 is up to date.", deviceID),
			)
		},
	})

	a.DefineTool(agent.ToolDefinition{
		Name:        "escalate_ticket",
		Description: "Escalate the support case to a specialist",
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("Case escalated to a Level 2 specialist. They will contact you within 1 hour.")
		},
	})

	// ---- Build context structure ----
	cb := a.DefineContexts()

	// ---- Sales Context: 3 steps ----
	sales := cb.AddContext("sales")
	sales.SetValidContexts([]string{"support"})

	// Step 1: Qualification
	step1 := sales.AddStep("qualification")
	step1.SetText("Greet the customer and understand their needs. Ask what products they are interested in.")
	step1.SetStepCriteria("The customer has identified at least one product they are interested in.")
	step1.SetValidSteps([]string{"product_review"})
	step1.SetFunctions("none") // No tools needed for greeting

	// Step 2: Product Review
	step2 := sales.AddStep("product_review")
	step2.SetText("Present product details and check availability. Use check_inventory to verify stock levels.")
	step2.SetStepCriteria("The customer has reviewed the product details and confirmed interest.")
	step2.SetValidSteps([]string{"close_sale"})
	step2.SetFunctions([]string{"check_inventory"})

	// Step 3: Close Sale
	step3 := sales.AddStep("close_sale")
	step3.SetText("Finalize the order. Confirm the product and quantity, then create the order.")
	step3.SetFunctions([]string{"check_inventory", "create_order"})
	step3.SetEnd(true)

	// ---- Support Context: 2 steps ----
	support := cb.AddContext("support")
	support.SetValidContexts([]string{"sales"})

	// Step 1: Triage
	triageStep := support.AddStep("triage")
	triageStep.SetText("Identify the customer's issue. Ask about symptoms and when the problem started.")
	triageStep.SetStepCriteria("The issue has been clearly identified and described.")
	triageStep.SetValidSteps([]string{"resolution"})
	triageStep.SetFunctions([]string{"run_diagnostic"})

	// Step 2: Resolution
	resStep := support.AddStep("resolution")
	resStep.SetText("Attempt to resolve the issue. Run diagnostics if needed. Escalate if the problem cannot be resolved.")
	resStep.SetFunctions([]string{"run_diagnostic", "escalate_ticket"})
	resStep.SetEnd(true)

	// ---- Language configuration ----
	a.AddLanguage(map[string]any{
		"name":     "English",
		"code":     "en-US",
		"voice":    "rime.spore",
		"function": "auto",
	})
	a.AddLanguage(map[string]any{
		"name":     "Spanish",
		"code":     "es-ES",
		"voice":    "rime.luna",
		"function": "auto",
	})

	fmt.Println("Starting ContextsDemo on :3004/contexts ...")
	fmt.Println("  Sales context:   qualification -> product_review -> close_sale")
	fmt.Println("  Support context: triage -> resolution")

	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
