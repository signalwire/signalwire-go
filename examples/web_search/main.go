//go:build ignore

// Example: web_search
//
// Web search skill demo. Demonstrates the web_search skill integration
// using Google Custom Search API. Requires GOOGLE_SEARCH_API_KEY and
// GOOGLE_SEARCH_ENGINE_ID environment variables.
package main

import (
	"fmt"
	"os"

	"github.com/signalwire/signalwire-agents-go/pkg/agent"

	// Import builtin skills so their init() functions register them
	_ "github.com/signalwire/signalwire-agents-go/pkg/skills/builtin"
)

func main() {
	apiKey := os.Getenv("GOOGLE_SEARCH_API_KEY")
	engineID := os.Getenv("GOOGLE_SEARCH_ENGINE_ID")

	if apiKey == "" || engineID == "" {
		fmt.Println("Error: Missing required environment variables:")
		fmt.Println("  GOOGLE_SEARCH_API_KEY")
		fmt.Println("  GOOGLE_SEARCH_ENGINE_ID")
		fmt.Println("\nGet these from: https://developers.google.com/custom-search/v1/introduction")
		os.Exit(1)
	}

	a := agent.NewAgentBase(
		agent.WithName("WebSearchAssistant"),
		agent.WithRoute("/search"),
		agent.WithPort(3021),
	)

	a.AddLanguage(map[string]any{
		"name":  "English",
		"code":  "en-US",
		"voice": "rime.spore",
	})

	a.PromptAddSection("Personality",
		"You are Franklin, a friendly and knowledgeable search bot. You are "+
			"enthusiastic about helping people find information on the internet.",
		nil,
	)
	a.PromptAddSection("Goal",
		"Help users find accurate, up-to-date information from the web.",
		nil,
	)
	a.PromptAddSection("Instructions", "", []string{
		"Always introduce yourself as Franklin when users first interact with you",
		"Use your web search capabilities to find current information",
		"Present search results in a well-organised format with source URLs",
	})

	a.AddSkill("web_search", map[string]any{
		"api_key":          apiKey,
		"search_engine_id": engineID,
		"num_results":      1,
	})

	fmt.Println("Starting Web Search Assistant on :3021/search ...")
	fmt.Println("  Skill: web_search (Google Custom Search)")

	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
