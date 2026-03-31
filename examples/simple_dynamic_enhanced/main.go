//go:build ignore

// Example: simple_dynamic_enhanced
//
// Enhanced dynamic configuration that adapts based on request parameters:
// - vip=true/false (premium voice, faster response)
// - department=sales/support/billing (specialized expertise)
// - customer_id=<string> (personalized experience)
// - language=en/es (language and voice selection)
//
// Test:
//   curl "http://localhost:3035/dynamic-enhanced?vip=true&department=sales"
//   curl "http://localhost:3035/dynamic-enhanced?department=billing&language=es"
package main

import (
	"fmt"
	"strings"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("DynamicEnhanced"),
		agent.WithRoute("/dynamic-enhanced"),
		agent.WithPort(3035),
	)

	a.SetDynamicConfigCallback(func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ephemeral *agent.AgentBase) {
		isVIP := strings.ToLower(queryParams["vip"]) == "true"
		department := queryParams["department"]
		if department == "" {
			department = "general"
		}
		customerID := queryParams["customer_id"]
		lang := queryParams["language"]
		if lang == "" {
			lang = "en"
		}

		// Voice and language
		voice := "rime.spore"
		if isVIP {
			voice = "rime.alloy"
		}
		langName := "English"
		langCode := "en-US"
		if lang == "es" {
			langName = "Spanish"
			langCode = "es-ES"
		}
		ephemeral.AddLanguage(map[string]any{
			"name": langName, "code": langCode, "voice": voice,
		})

		// AI parameters
		endOfSpeech := 500
		attention := 15000
		if isVIP {
			endOfSpeech = 300
			attention = 20000
		}
		ephemeral.SetParam("end_of_speech_timeout", endOfSpeech)
		ephemeral.SetParam("attention_timeout", attention)

		// Global data
		globalData := map[string]any{
			"department":    department,
			"service_level": "standard",
		}
		if isVIP {
			globalData["service_level"] = "vip"
		}
		if customerID != "" {
			globalData["customer_id"] = customerID
		}
		ephemeral.SetGlobalData(globalData)

		// Role prompt
		role := "You are a professional customer service representative."
		if customerID != "" {
			role = fmt.Sprintf("You are a customer service rep helping customer %s.", customerID)
		}
		if isVIP {
			role += " This is a VIP customer who receives priority service."
		}
		ephemeral.PromptAddSection("Role", role, nil)

		// Department expertise
		switch department {
		case "sales":
			ephemeral.PromptAddSection("Sales Expertise", "You specialize in sales:", []string{
				"Present product features and benefits",
				"Handle pricing questions and offers",
				"Process orders and upgrades",
			})
			ephemeral.AddHints([]string{"pricing", "enterprise", "upgrade"})
		case "billing":
			ephemeral.PromptAddSection("Billing Expertise", "You specialize in billing:", []string{
				"Explain statements and charges",
				"Process payment arrangements",
				"Handle dispute resolution",
			})
			ephemeral.AddHints([]string{"invoice", "payment", "charges"})
		default:
			ephemeral.PromptAddSection("Support Guidelines", "Follow these principles:", []string{
				"Listen carefully to customer needs",
				"Provide accurate information",
				"Escalate complex issues when appropriate",
			})
			ephemeral.AddHints([]string{"support", "troubleshoot", "help"})
		}

		// VIP-specific service
		if isVIP {
			ephemeral.PromptAddSection("VIP Standards", "Premium service:", []string{
				"Provide immediate attention",
				"Offer exclusive options",
				"Ensure complete satisfaction",
			})

			ephemeral.DefineTool(agent.ToolDefinition{
				Name:        "schedule_priority_callback",
				Description: "Schedule a priority callback for VIP customer",
				Parameters: map[string]any{
					"time_slot": map[string]any{
						"type":        "string",
						"description": "Preferred callback time",
					},
				},
				Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
					slot, _ := args["time_slot"].(string)
					return swaig.NewFunctionResult(
						fmt.Sprintf("Priority callback scheduled for %s.", slot),
					)
				},
			})
		}

		// Common tool
		ephemeral.DefineTool(agent.ToolDefinition{
			Name:        "check_order",
			Description: "Check order status",
			Parameters: map[string]any{
				"order_number": map[string]any{
					"type": "string", "description": "Order number",
				},
			},
			Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
				num, _ := args["order_number"].(string)
				return swaig.NewFunctionResult(
					fmt.Sprintf("Order %s is being processed. Ships in 2 business days.", num),
				)
			},
		})
	})

	fmt.Println("Starting DynamicEnhanced on :3035/dynamic-enhanced ...")
	fmt.Println("  ?vip=true          Premium voice + faster response")
	fmt.Println("  ?department=sales  Sales-specific expertise")
	fmt.Println("  ?customer_id=X    Personalized experience")
	fmt.Println("  ?language=es       Spanish language")

	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
