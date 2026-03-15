//go:build ignore

// Example: datasphere
//
// Datasphere skill integration. Demonstrates connecting an AI agent
// to SignalWire Datasphere for document search and retrieval.
// Requires SIGNALWIRE_PROJECT_ID, SIGNALWIRE_API_TOKEN, and
// SIGNALWIRE_SPACE environment variables plus a Datasphere document ID.
package main

import (
	"fmt"
	"os"

	"github.com/signalwire/signalwire-agents-go/pkg/agent"

	// Import builtin skills so their init() functions register them
	_ "github.com/signalwire/signalwire-agents-go/pkg/skills/builtin"
)

func main() {
	documentID := os.Getenv("DATASPHERE_DOCUMENT_ID")
	if documentID == "" {
		fmt.Println("Error: DATASPHERE_DOCUMENT_ID environment variable is required.")
		fmt.Println("Also set SIGNALWIRE_PROJECT_ID, SIGNALWIRE_API_TOKEN, SIGNALWIRE_SPACE.")
		os.Exit(1)
	}

	a := agent.NewAgentBase(
		agent.WithName("DatasphereAgent"),
		agent.WithRoute("/datasphere"),
		agent.WithPort(3023),
	)

	a.AddLanguage(map[string]any{
		"name":  "English",
		"code":  "en-US",
		"voice": "rime.spore",
	})

	a.PromptAddSection("Role",
		"You are a knowledgeable assistant with access to a document knowledge base "+
			"through SignalWire Datasphere. Search for relevant information to answer questions.",
		nil,
	)

	// Add datasphere skill
	a.AddSkill("datasphere", map[string]any{
		"document_id": documentID,
	})

	fmt.Println("Starting DatasphereAgent on :3023/datasphere ...")
	fmt.Printf("  Document ID: %s\n", documentID)
	fmt.Println("  Skill: datasphere")

	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
