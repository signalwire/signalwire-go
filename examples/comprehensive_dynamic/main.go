//go:build ignore

// Example: comprehensive_dynamic
//
// Tier-based dynamic agent configuration (standard/premium/enterprise).
// Demonstrates per-request voice selection, industry-specific prompts,
// LLM parameter tuning, A/B testing, and global data customization.
package main

import (
	"fmt"
	"strings"

	"github.com/signalwire/signalwire-agents-go/pkg/agent"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("ComprehensiveDynamicAgent"),
		agent.WithRoute("/dynamic"),
		agent.WithPort(3011),
	)

	// Base prompt (overridden per-request)
	a.SetPromptText("You are a configurable support agent.")

	a.SetDynamicConfigCallback(func(qp map[string]string, bp map[string]any, headers map[string]string, ep *agent.AgentBase) {
		tier := strings.ToLower(qp["tier"])
		if tier == "" {
			tier = "standard"
		}
		industry := strings.ToLower(qp["industry"])
		if industry == "" {
			industry = "general"
		}
		testGroup := strings.ToUpper(qp["test_group"])
		if testGroup == "" {
			testGroup = "A"
		}

		// ---- Voice & Language ----
		voice := "rime.spore"
		if tier == "enterprise" || tier == "premium" {
			voice = "rime.spore"
		}
		ep.AddLanguage(map[string]any{
			"name":  "English",
			"code":  "en-US",
			"voice": voice,
		})

		// ---- Tier Parameters ----
		switch tier {
		case "enterprise":
			ep.SetParams(map[string]any{
				"end_of_speech_timeout": 800,
				"attention_timeout":     25000,
				"temperature":           0.3,
			})
		case "premium":
			ep.SetParams(map[string]any{
				"end_of_speech_timeout": 600,
				"attention_timeout":     20000,
				"temperature":           0.4,
			})
		default:
			ep.SetParams(map[string]any{
				"end_of_speech_timeout": 400,
				"attention_timeout":     15000,
				"temperature":           0.3,
			})
		}

		// ---- Industry Prompts ----
		ep.PromptAddSection("Role and Purpose",
			fmt.Sprintf("You are a professional AI assistant specialised in %s services.", industry),
			nil,
		)

		switch industry {
		case "healthcare":
			ep.PromptAddSection("Healthcare Guidelines",
				"Follow HIPAA compliance standards. Never provide medical diagnoses.",
				[]string{
					"Protect patient privacy at all times",
					"Direct medical questions to qualified healthcare providers",
					"Maintain professional bedside manner",
				},
			)
		case "finance":
			ep.PromptAddSection("Financial Guidelines",
				"Adhere to financial industry regulations.",
				[]string{
					"Never provide specific investment advice",
					"Protect sensitive financial information",
					"Refer complex matters to qualified advisors",
				},
			)
		case "retail":
			ep.PromptAddSection("Customer Service Excellence",
				"Focus on customer satisfaction and sales support.",
				[]string{
					"Maintain friendly, helpful demeanour",
					"Handle complaints with empathy",
					"Look for opportunities to enhance customer experience",
				},
			)
		}

		// Enhanced capabilities for premium/enterprise
		if tier == "premium" || tier == "enterprise" {
			ep.PromptAddSection("Enhanced Capabilities",
				fmt.Sprintf("As a %s service, you have access to advanced features:", tier),
				[]string{
					"Extended conversation memory",
					"Priority processing and faster responses",
					"Access to specialised knowledge bases",
				},
			)
		}

		// ---- A/B Testing ----
		if testGroup == "B" {
			ep.AddHints([]string{"enhanced", "personalised", "proactive"})
			ep.PromptAddSection("Enhanced Interaction Style",
				"You are using an enhanced conversation style for this session:",
				[]string{
					"Ask clarifying questions more frequently",
					"Provide more detailed explanations",
					"Offer proactive suggestions when appropriate",
				},
			)
		}

		// ---- Global Data ----
		features := []string{"basic_conversation", "function_calling"}
		if tier == "premium" || tier == "enterprise" {
			features = append(features, "extended_memory", "priority_processing")
		}
		if tier == "enterprise" {
			features = append(features, "custom_integration", "dedicated_support")
		}

		ep.SetGlobalData(map[string]any{
			"service_tier":    tier,
			"industry_focus":  industry,
			"test_group":      testGroup,
			"features_enabled": features,
		})
	})

	fmt.Println("Starting ComprehensiveDynamicAgent on :3011/dynamic ...")
	fmt.Println("  Standard:   POST /dynamic")
	fmt.Println("  Premium:    POST /dynamic?tier=premium&industry=healthcare")
	fmt.Println("  Enterprise: POST /dynamic?tier=enterprise&industry=finance&test_group=B")

	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
