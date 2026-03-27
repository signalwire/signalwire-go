//go:build ignore

// Example: Create an AI agent, assign a phone number, and place a test call.
//
// Set these env vars (or pass them directly to NewRestClient):
//
//	SIGNALWIRE_PROJECT_ID   - your SignalWire project ID
//	SIGNALWIRE_API_TOKEN    - your SignalWire API token
//	SIGNALWIRE_SPACE        - your SignalWire space (e.g. example.signalwire.com)
//
// For full HTTP debug output:
//
//	SIGNALWIRE_LOG_LEVEL=debug
package main

import (
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/pkg/rest"
)

func main() {
	client, err := rest.NewRestClient("", "", "")
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		os.Exit(1)
	}

	// 1. Create an AI agent
	fmt.Println("Creating AI agent...")
	agent, err := client.Fabric.AIAgents.Create(map[string]any{
		"name":   "Demo Support Bot",
		"prompt": map[string]any{"text": "You are a friendly support agent for Acme Corp."},
	})
	if err != nil {
		fmt.Printf("  Create agent failed: %v\n", err)
		return
	}
	agentID := agent["id"].(string)
	fmt.Printf("  Created agent: %s\n", agentID)

	// 2. List all AI agents
	fmt.Println("\nListing AI agents...")
	agents, err := client.Fabric.AIAgents.List(nil)
	if err != nil {
		fmt.Printf("  List agents failed: %v\n", err)
	} else if data, ok := agents["data"].([]any); ok {
		for _, a := range data {
			if m, ok := a.(map[string]any); ok {
				fmt.Printf("  - %s: %s\n", m["id"], m["name"])
			}
		}
	}

	// 3. Search for a phone number
	fmt.Println("\nSearching for available phone numbers...")
	available, err := client.PhoneNumbers.Search(map[string]string{
		"area_code":   "512",
		"max_results": "3",
	})
	if err != nil {
		fmt.Printf("  Search failed: %v\n", err)
	} else if data, ok := available["data"].([]any); ok {
		for _, n := range data {
			if m, ok := n.(map[string]any); ok {
				fmt.Printf("  - %v\n", m["e164"])
			}
		}
	}

	// 4. Place a test call (requires valid numbers)
	fmt.Println("\nPlacing a test call...")
	result, err := client.Calling.Dial(map[string]any{
		"from": "+15559876543",
		"to":   "+15551234567",
		"url":  "https://example.com/call-handler",
	})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Call failed (expected in demo): %d\n", restErr.StatusCode)
		} else {
			fmt.Printf("  Call failed: %v\n", err)
		}
	} else {
		fmt.Printf("  Call initiated: %v\n", result)
	}

	// 5. Clean up: delete the agent
	fmt.Printf("\nDeleting agent %s...\n", agentID)
	if err := client.Fabric.AIAgents.Delete(agentID); err != nil {
		fmt.Printf("  Delete failed: %v\n", err)
	} else {
		fmt.Println("  Deleted.")
	}
}
