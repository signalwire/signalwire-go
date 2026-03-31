//go:build ignore

// Example: web_search_multi_instance
//
// Web search skill loaded multiple times with different configurations
// and tool names (general, news, quick). Also includes Wikipedia search.
//
// Required: GOOGLE_SEARCH_API_KEY, GOOGLE_SEARCH_ENGINE_ID
package main

import (
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/pkg/agent"

	_ "github.com/signalwire/signalwire-go/pkg/skills/builtin"
)

func main() {
	apiKey := os.Getenv("GOOGLE_SEARCH_API_KEY")
	engineID := os.Getenv("GOOGLE_SEARCH_ENGINE_ID")

	a := agent.NewAgentBase(
		agent.WithName("MultiSearchAgent"),
		agent.WithRoute("/multi-search"),
		agent.WithPort(3036),
	)

	a.AddLanguage(map[string]any{
		"name":  "English",
		"code":  "en-US",
		"voice": "rime.spore",
	})

	a.PromptAddSection("Role",
		"You are a research assistant with access to multiple search tools. "+
			"Use the most appropriate tool for each query.",
		nil,
	)

	a.AddSkill("datetime", nil)
	a.AddSkill("math", nil)

	// Wikipedia search
	a.AddSkill("wikipedia_search", map[string]any{
		"num_results": 2,
	})

	if apiKey == "" || engineID == "" {
		fmt.Println("Warning: Missing GOOGLE_SEARCH_API_KEY or GOOGLE_SEARCH_ENGINE_ID.")
		fmt.Println("Web search instances will not be available. Wikipedia search is still active.")
	} else {
		// Instance 1: General web search (default tool name)
		a.AddSkill("web_search", map[string]any{
			"api_key":          apiKey,
			"search_engine_id": engineID,
			"num_results":      3,
		})

		// Instance 2: News search
		a.AddSkill("web_search", map[string]any{
			"api_key":          apiKey,
			"search_engine_id": engineID,
			"tool_name":        "search_news",
			"num_results":      5,
			"delay":            0.5,
		})

		// Instance 3: Quick search (single result)
		a.AddSkill("web_search", map[string]any{
			"api_key":          apiKey,
			"search_engine_id": engineID,
			"tool_name":        "quick_search",
			"num_results":      1,
			"delay":            0,
		})
	}

	fmt.Println("Starting MultiSearchAgent on :3036/multi-search ...")
	fmt.Println("  Tools: web_search, search_news, quick_search, search_wiki")

	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
