//go:build ignore

// Example: joke_skill
//
// Joke skill demo using the modular skills system with DataMap.
// Compare with joke_agent (raw data_map) to see the benefits of the skills system.
//
// Required: API_NINJAS_KEY environment variable.
package main

import (
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/pkg/agent"

	_ "github.com/signalwire/signalwire-go/pkg/skills/builtin"
)

func main() {
	apiKey := os.Getenv("API_NINJAS_KEY")
	if apiKey == "" {
		fmt.Println("Error: API_NINJAS_KEY environment variable is required.")
		fmt.Println("Get your free API key from https://api.api-ninjas.com/")
		os.Exit(1)
	}

	a := agent.NewAgentBase(
		agent.WithName("JokeSkillDemo"),
		agent.WithRoute("/joke-skill"),
		agent.WithPort(3034),
	)

	a.AddLanguage(map[string]any{
		"name":  "English",
		"code":  "en-US",
		"voice": "rime.spore",
	})

	a.PromptAddSection("Personality",
		"You are a cheerful comedian who loves sharing jokes and making people laugh.",
		nil,
	)
	a.PromptAddSection("Instructions", "", []string{
		"When users ask for jokes, use your joke functions to provide them",
		"Be enthusiastic and fun in your responses",
		"You can tell both regular jokes and dad jokes",
	})

	// Add joke skill (uses DataMap for serverless execution)
	a.AddSkill("joke", map[string]any{
		"api_key": apiKey,
	})

	fmt.Println("Starting JokeSkillDemo on :3034/joke-skill ...")
	fmt.Println("  Skill: joke (via DataMap, serverless)")
	fmt.Println("  Benefits over raw DataMap:")
	fmt.Println("    - One-liner integration via skills system")
	fmt.Println("    - Automatic validation and error handling")
	fmt.Println("    - Reusable across agents")

	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
