//go:build ignore

// Example: datasphere_multi_instance
//
// Loads the datasphere skill multiple times with different knowledge bases
// and custom tool names. Each instance searches a different document.
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/agent"

	_ "github.com/signalwire/signalwire-go/pkg/skills/builtin"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("MultiDatasphere"),
		agent.WithRoute("/datasphere-multi"),
		agent.WithPort(3030),
	)

	a.AddLanguage(map[string]any{
		"name":  "English",
		"code":  "en-US",
		"voice": "rime.spore",
	})

	a.PromptAddSection("Role",
		"You are an assistant with access to multiple knowledge bases. "+
			"Use the appropriate search tool depending on the topic.",
		nil,
	)

	// Utility skills
	a.AddSkill("datetime", nil)
	a.AddSkill("math", nil)

	// Instance 1: Drinks knowledge base
	a.AddSkill("datasphere", map[string]any{
		"document_id": "drinks-doc-123",
		"tool_name":   "search_drinks_knowledge",
		"count":       2,
		"distance":    5.0,
	})

	// Instance 2: Food knowledge base
	a.AddSkill("datasphere", map[string]any{
		"document_id": "food-doc-456",
		"tool_name":   "search_food_knowledge",
		"count":       3,
		"distance":    4.0,
	})

	// Instance 3: General knowledge (default tool name)
	a.AddSkill("datasphere", map[string]any{
		"document_id": "general-doc-789",
		"count":       1,
		"distance":    3.0,
	})

	fmt.Println("Starting MultiDatasphere on :3030/datasphere-multi ...")
	fmt.Println("  Tools: search_drinks_knowledge, search_food_knowledge, search_knowledge")
	fmt.Println("  Note: Replace document IDs with your actual DataSphere documents.")

	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
