//go:build ignore

// Example: wikipedia
//
// Wikipedia skill demo. Demonstrates the wikipedia_search skill for
// factual information retrieval from Wikipedia articles.
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/agent"

	// Import builtin skills so their init() functions register them
	_ "github.com/signalwire/signalwire-go/pkg/skills/builtin"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("WikipediaAssistant"),
		agent.WithRoute("/wiki"),
		agent.WithPort(3022),
	)

	a.AddLanguage(map[string]any{
		"name":  "English",
		"code":  "en-US",
		"voice": "rime.spore",
	})

	a.PromptAddSection("Role",
		"You are a knowledgeable assistant that specialises in factual "+
			"information retrieval from Wikipedia.",
		nil,
	)
	a.PromptAddSection("Instructions", "", []string{
		"Use the search_wiki tool to look up information",
		"Provide clear, factual summaries from Wikipedia articles",
		"Cite the article title when referencing information",
	})

	// Add datetime skill for convenience
	a.AddSkill("datetime", map[string]any{
		"timezone": "America/New_York",
	})

	// Add Wikipedia search skill
	a.AddSkill("wikipedia_search", map[string]any{
		"num_results": 2,
	})

	fmt.Println("Starting Wikipedia Assistant on :3022/wiki ...")
	fmt.Println("  Skills: datetime, wikipedia_search")
	fmt.Println("  Try: 'Tell me about Albert Einstein'")
	fmt.Println("       'What is quantum physics?'")

	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
