// Example: session_state
//
// Global data management and lifecycle callbacks. Demonstrates setting
// initial global data, configuring a post-prompt for conversation
// summaries, registering an OnSummary callback, and defining a tool
// that uses UpdateGlobalData to track state across interactions.
package main

import (
	"encoding/json"
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("SessionStateDemo"),
		agent.WithRoute("/session"),
		agent.WithPort(3007),
	)

	a.SetPromptText(
		"You are a shopping assistant. Help users add items to their cart " +
			"and track their selections. Use the add_to_cart tool when a user " +
			"wants to add an item.",
	)

	// Set initial global data representing the session state
	a.SetGlobalData(map[string]any{
		"cart_items":  []any{},
		"cart_total":  0.0,
		"customer_id": "",
		"currency":   "USD",
	})

	// Set a post-prompt for conversation summary
	a.SetPostPrompt(
		"Summarize the conversation as JSON with the following fields: " +
			"customer_intent (what the customer wanted), items_discussed (list of products), " +
			"cart_contents (final cart state), satisfaction (high/medium/low).",
	)

	// Set LLM parameters for the post-prompt (can differ from main prompt)
	a.SetPostPromptLlmParams(map[string]any{
		"temperature": 0.1,
		"top_p":       0.5,
	})

	// Register the OnSummary callback to log summaries
	a.OnSummary(func(summary map[string]any, rawData map[string]any) {
		summaryJSON, _ := json.MarshalIndent(summary, "", "  ")
		fmt.Printf("\n=== Conversation Summary ===\n%s\n============================\n\n", string(summaryJSON))
	})

	// Define a tool that modifies global data to track cart state
	a.DefineTool(agent.ToolDefinition{
		Name:        "add_to_cart",
		Description: "Add an item to the shopping cart",
		Parameters: map[string]any{
			"item_name": map[string]any{
				"type":        "string",
				"description": "Name of the item to add",
			},
			"price": map[string]any{
				"type":        "number",
				"description": "Price of the item in dollars",
			},
			"quantity": map[string]any{
				"type":        "integer",
				"description": "Number of units (default 1)",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			itemName, _ := args["item_name"].(string)
			price, _ := args["price"].(float64)
			quantity := 1
			if q, ok := args["quantity"].(float64); ok && q > 0 {
				quantity = int(q)
			}

			lineTotal := price * float64(quantity)

			// Read current cart state from global_data
			globalData, _ := rawData["global_data"].(map[string]any)
			currentTotal := 0.0
			if ct, ok := globalData["cart_total"].(float64); ok {
				currentTotal = ct
			}
			newTotal := currentTotal + lineTotal

			// Build the new cart item
			newItem := map[string]any{
				"name":     itemName,
				"price":    price,
				"quantity": quantity,
				"total":    lineTotal,
			}

			// Return a result that updates global data with the new cart state
			result := swaig.NewFunctionResult(
				fmt.Sprintf("Added %d x %s ($%.2f each) to cart. Line total: $%.2f. Cart total: $%.2f.",
					quantity, itemName, price, lineTotal, newTotal),
			)

			// Use UpdateGlobalData action to persist the state change
			// The existing cart_items array in global_data gets the new entry
			var items []any
			if existingItems, ok := globalData["cart_items"].([]any); ok {
				items = append(existingItems, newItem)
			} else {
				items = []any{newItem}
			}

			result.UpdateGlobalData(map[string]any{
				"cart_items": items,
				"cart_total": newTotal,
			})

			return result
		},
	})

	// Define a tool to view the cart
	a.DefineTool(agent.ToolDefinition{
		Name:        "view_cart",
		Description: "View the current contents of the shopping cart",
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			globalData, _ := rawData["global_data"].(map[string]any)
			items, _ := globalData["cart_items"].([]any)
			total, _ := globalData["cart_total"].(float64)

			if len(items) == 0 {
				return swaig.NewFunctionResult("The cart is empty.")
			}

			summary := fmt.Sprintf("Cart contains %d item(s). Total: $%.2f.", len(items), total)
			return swaig.NewFunctionResult(summary)
		},
	})

	fmt.Println("Starting SessionStateDemo on :3007/session ...")
	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
