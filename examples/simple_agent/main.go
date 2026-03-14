// Example: simple_agent
//
// Basic AI agent with prompt and tools. Demonstrates creating an AgentBase,
// setting prompt text, adding hints and language, defining SWAIG tools,
// setting global data, and running the agent.
package main

import (
	"fmt"
	"time"

	"github.com/signalwire/signalwire-agents-go/pkg/agent"
	"github.com/signalwire/signalwire-agents-go/pkg/swaig"
)

func main() {
	// Create an agent with functional options
	a := agent.NewAgentBase(
		agent.WithName("SimpleAgent"),
		agent.WithRoute("/simple"),
		agent.WithPort(3001),
	)

	// Set the prompt using raw text mode
	a.SetPromptText("You are a helpful assistant. You can tell the user the current time and provide weather information for any city.")

	// Add speech-recognition hints so the ASR engine better recognises these words
	a.AddHints([]string{"weather", "temperature", "time", "clock"})

	// Add a language configuration (English with a specific voice)
	a.AddLanguage(map[string]any{
		"name":     "English",
		"code":     "en-US",
		"voice":    "rime.spore",
		"function": "auto",
	})

	// Set AI parameters for consistent, professional responses
	a.SetParam("temperature", 0.3)
	a.SetParam("top_p", 0.9)

	// Set global data available to all tool handlers
	a.SetGlobalData(map[string]any{
		"company_name":   "Acme Corp",
		"support_number": "+15551234567",
	})

	// Define a tool that returns the current time
	a.DefineTool(agent.ToolDefinition{
		Name:        "get_time",
		Description: "Get the current date and time",
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			now := time.Now()
			return swaig.NewFunctionResult(
				fmt.Sprintf("The current date and time is %s", now.Format("Monday, January 02, 2006 at 03:04 PM MST")),
			)
		},
	})

	// Define a tool that returns mock weather data
	a.DefineTool(agent.ToolDefinition{
		Name:        "get_weather",
		Description: "Get the current weather for a given city",
		Parameters: map[string]any{
			"city": map[string]any{
				"type":        "string",
				"description": "The city to get weather for",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			city, _ := args["city"].(string)
			if city == "" {
				city = "Unknown"
			}
			return swaig.NewFunctionResult(
				fmt.Sprintf("The weather in %s is 72°F (22°C) and sunny with light winds.", city),
			)
		},
	})

	// Run the agent (blocking)
	fmt.Println("Starting SimpleAgent on :3001/simple ...")
	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
