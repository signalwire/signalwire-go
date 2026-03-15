//go:build ignore

// Example: simple_static
//
// Minimal static agent. All configuration is set once during
// initialisation and remains the same for every request. Demonstrates
// voice, language, AI parameters, hints, global data, and POM prompts.
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-agents-go/pkg/agent"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("SimpleStaticAgent"),
		agent.WithRoute("/"),
		agent.WithPort(3025),
		agent.WithAutoAnswer(true),
		agent.WithRecordCall(true),
	)

	// Voice and language
	a.AddLanguage(map[string]any{
		"name":  "English",
		"code":  "en-US",
		"voice": "rime.spore",
	})

	// AI parameters
	a.SetParams(map[string]any{
		"end_of_speech_timeout": 500,
		"attention_timeout":     15000,
	})

	// Speech recognition hints
	a.AddHints([]string{"SignalWire", "SWML", "API", "webhook", "SIP"})

	// Global data
	a.SetGlobalData(map[string]any{
		"agent_type":    "customer_service",
		"service_level": "standard",
		"features_enabled": []string{"basic_conversation", "help_desk"},
	})

	// Prompt sections
	a.PromptAddSection("Role and Purpose",
		"You are a professional customer service representative. Your goal is to help "+
			"customers with their questions and provide excellent service.",
		nil,
	)
	a.PromptAddSection("Guidelines",
		"Follow these customer service principles:",
		[]string{
			"Listen carefully to customer needs",
			"Provide accurate and helpful information",
			"Maintain a professional and friendly tone",
			"Escalate complex issues when appropriate",
		},
	)
	a.PromptAddSection("Available Services",
		"You can help customers with:",
		[]string{
			"General product information",
			"Account questions and support",
			"Technical troubleshooting guidance",
			"Billing and payment inquiries",
		},
	)

	fmt.Println("Starting SimpleStaticAgent on :3025/ ...")
	fmt.Println("  Configuration: STATIC (set once at startup)")
	fmt.Println("  Voice: rime.spore")
	fmt.Println("  Speech Timeout: 500ms")

	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
