//go:build ignore

// Example: llm_params
//
// LLM parameter tuning demo. Shows how to use SetPromptLlmParams and
// SetPostPromptLlmParams to create agents with different response
// characteristics: precise, creative, and customer-service profiles.
package main

import (
	"fmt"
	"math/rand"

	"github.com/signalwire/signalwire-agents-go/pkg/agent"
	"github.com/signalwire/signalwire-agents-go/pkg/swaig"
)

func main() {
	// ---- Precise Assistant (low temperature, hard to interrupt) ----
	precise := agent.NewAgentBase(
		agent.WithName("PreciseAssistant"),
		agent.WithRoute("/precise"),
		agent.WithPort(3017),
	)
	precise.PromptAddSection("Role", "You are a precise technical assistant.", nil)
	precise.PromptAddSection("Instructions", "", []string{
		"Provide accurate, factual information",
		"Be concise and direct",
		"Avoid speculation or guessing",
	})
	precise.SetPromptLlmParams(map[string]any{
		"temperature":      0.2,
		"top_p":            0.85,
		"barge_confidence": 0.8,
		"frequency_penalty": 0.1,
	})
	precise.SetPostPrompt("Provide a brief technical summary of the key points discussed.")
	precise.SetPostPromptLlmParams(map[string]any{
		"temperature": 0.1,
	})

	precise.DefineTool(agent.ToolDefinition{
		Name:        "get_system_info",
		Description: "Get technical system information",
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult(
				fmt.Sprintf("System Status: CPU %d%%, Memory %dGB, Uptime %d days",
					rand.Intn(80)+10, rand.Intn(15)+1, rand.Intn(30)+1),
			)
		},
	})

	// ---- Creative Assistant (high temperature, easy to interrupt) ----
	creative := agent.NewAgentBase(
		agent.WithName("CreativeAssistant"),
		agent.WithRoute("/creative"),
		agent.WithPort(3018),
	)
	creative.PromptAddSection("Role", "You are a creative writing assistant.", nil)
	creative.PromptAddSection("Instructions", "", []string{
		"Be imaginative and creative",
		"Use varied vocabulary and expressions",
		"Encourage creative thinking",
	})
	creative.SetPromptLlmParams(map[string]any{
		"temperature":       0.8,
		"top_p":             0.95,
		"barge_confidence":  0.5,
		"presence_penalty":  0.2,
		"frequency_penalty": 0.3,
	})
	creative.SetPostPrompt("Create an artistic summary of our conversation.")
	creative.SetPostPromptLlmParams(map[string]any{
		"temperature": 0.7,
	})

	creative.DefineTool(agent.ToolDefinition{
		Name:        "generate_story_prompt",
		Description: "Generate a creative story prompt",
		Parameters: map[string]any{
			"theme": map[string]any{
				"type":        "string",
				"description": "Story theme (adventure, mystery, etc.)",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			theme, _ := args["theme"].(string)
			if theme == "" {
				theme = "adventure"
			}
			prompts := []string{
				"A map that only appears during thunderstorms",
				"A compass that points to what you need most",
				"A door that leads somewhere different each time",
			}
			return swaig.NewFunctionResult(
				fmt.Sprintf("Story prompt for %s: %s", theme, prompts[rand.Intn(len(prompts))]),
			)
		},
	})

	fmt.Println("LLM Parameter Tuning Demo")
	fmt.Println("  Precise:  :3017/precise  (temperature=0.2, barge_confidence=0.8)")
	fmt.Println("  Creative: :3018/creative  (temperature=0.8, barge_confidence=0.5)")
	fmt.Println("\nStarting Precise Assistant on :3017 ...")

	// Run the precise agent (use a multi-agent server for both in production)
	if err := precise.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
