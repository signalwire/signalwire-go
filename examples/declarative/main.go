//go:build ignore

// Example: declarative
//
// Agent configured declaratively using struct-level configuration.
// Demonstrates setting up the entire prompt, tools, and post-prompt
// via AgentBase methods in a struct-like pattern without subclassing.
package main

import (
	"fmt"
	"time"

	"github.com/signalwire/signalwire-agents-go/pkg/agent"
	"github.com/signalwire/signalwire-agents-go/pkg/swaig"
)

func main() {
	// Create the agent with all configuration declared up front
	a := agent.NewAgentBase(
		agent.WithName("DeclarativeAgent"),
		agent.WithRoute("/declarative"),
		agent.WithPort(3013),
	)

	// ---- Declarative prompt via POM sections ----
	a.PromptAddSection("Personality",
		"You are a friendly and helpful AI assistant who responds in a casual, conversational tone.",
		nil,
	)
	a.PromptAddSection("Goal",
		"Help users with their questions about time and weather.",
		nil,
	)
	a.PromptAddSection("Instructions", "", []string{
		"Be concise and direct in your responses.",
		"If you don't know something, say so clearly.",
		"Use the get_time function when asked about the current time.",
		"Use the get_weather function when asked about the weather.",
	})

	// ---- Post-prompt for summary ----
	a.SetPostPrompt(`Return a JSON summary of the conversation:
{
    "topic": "MAIN_TOPIC",
    "satisfied": true/false,
    "follow_up_needed": true/false
}`)

	// ---- Tools ----
	a.DefineTool(agent.ToolDefinition{
		Name:        "get_time",
		Description: "Get the current time",
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			now := time.Now().Format("03:04:05 PM MST")
			return swaig.NewFunctionResult(fmt.Sprintf("The current time is %s", now))
		},
	})

	a.DefineTool(agent.ToolDefinition{
		Name:        "get_weather",
		Description: "Get the current weather for a location",
		Parameters: map[string]any{
			"location": map[string]any{
				"type":        "string",
				"description": "The city or location to get weather for",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			location, _ := args["location"].(string)
			if location == "" {
				location = "Unknown location"
			}
			return swaig.NewFunctionResult(
				fmt.Sprintf("It's sunny and 72°F in %s.", location),
			)
		},
	})

	fmt.Println("Starting DeclarativeAgent on :3013/declarative ...")
	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
