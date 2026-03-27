//go:build ignore

// Example: joke_agent
//
// Joke skill demo. Demonstrates the joke skill integration using
// the built-in skills system. Requires API_NINJAS_KEY environment variable.
package main

import (
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/pkg/agent"

	// Import builtin skills so their init() functions register them
	_ "github.com/signalwire/signalwire-go/pkg/skills/builtin"
)

func main() {
	apiKey := os.Getenv("API_NINJAS_KEY")
	if apiKey == "" {
		fmt.Println("Error: API_NINJAS_KEY environment variable is required.")
		fmt.Println("Get your free API key from https://api.api-ninjas.com/")
		fmt.Println("Then run: API_NINJAS_KEY=your_key go run ./examples/joke_agent/")
		os.Exit(1)
	}

	a := agent.NewAgentBase(
		agent.WithName("JokeAgent"),
		agent.WithRoute("/joke"),
		agent.WithPort(3015),
	)

	a.PromptAddSection("Personality",
		"You are a cheerful comedian who loves sharing jokes and making people laugh.",
		nil,
	)
	a.PromptAddSection("Goal",
		"Entertain users with great jokes and spread joy.",
		nil,
	)
	a.PromptAddSection("Instructions", "", []string{
		"When users ask for jokes, use your joke functions to provide them",
		"Be enthusiastic and fun in your responses",
		"You can tell both regular jokes and dad jokes",
	})

	// Add joke skill using the skills registry
	a.AddSkill("joke", map[string]any{
		"api_key": apiKey,
	})

	fmt.Println("Starting JokeAgent on :3015/joke ...")
	fmt.Println("  Skill: joke (API Ninjas)")
	fmt.Println("  Ask for jokes like: 'Tell me a joke' or 'I want a dad joke'")

	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
