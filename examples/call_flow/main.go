// Example: call_flow
//
// Call flow verbs and SWAIG actions. Demonstrates pre-answer verbs
// (ringback), answer configuration, post-answer verbs (welcome audio),
// post-AI verbs (hangup), debug events, and tools that return call
// control actions like connect and send_sms.
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-agents-go/pkg/agent"
	"github.com/signalwire/signalwire-agents-go/pkg/swaig"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("CallFlowDemo"),
		agent.WithRoute("/callflow"),
		agent.WithPort(3008),
	)

	a.SetPromptText(
		"You are a receptionist for Acme Corp. You can transfer callers to " +
			"departments, send confirmation texts, and help with general inquiries.",
	)

	// ---- Pre-answer verb: play ringback tone while the call connects ----
	a.AddPreAnswerVerb("play", map[string]any{
		"url":    "https://cdn.signalwire.com/default-music/welcome.mp3",
		"volume": 5,
	})

	// ---- Answer verb with max_duration override ----
	a.AddAnswerVerb(map[string]any{
		"max_duration": 7200, // 2 hours max
	})

	// ---- Post-answer verb: play a welcome message before AI takes over ----
	a.AddPostAnswerVerb("play", map[string]any{
		"url": "https://cdn.example.com/audio/welcome_acme.mp3",
	})

	// ---- Post-AI verb: hangup after the AI conversation ends ----
	a.AddPostAiVerb("hangup", map[string]any{})

	// ---- Enable debug events for monitoring ----
	a.EnableDebugEvents(1)
	a.OnDebugEvent(func(event map[string]any) {
		eventType, _ := event["event_type"].(string)
		fmt.Printf("[DEBUG] Event: %s\n", eventType)
	})

	// ---- Tool: Transfer to a department ----
	a.DefineTool(agent.ToolDefinition{
		Name:        "transfer_call",
		Description: "Transfer the caller to a specific department",
		Parameters: map[string]any{
			"department": map[string]any{
				"type":        "string",
				"description": "Department name (sales, support, billing)",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			department, _ := args["department"].(string)

			// Map departments to phone numbers
			numbers := map[string]string{
				"sales":   "+15551001001",
				"support": "+15551001002",
				"billing": "+15551001003",
			}

			number, ok := numbers[department]
			if !ok {
				return swaig.NewFunctionResult(
					"I don't have a number for that department. Available: sales, support, billing.",
				)
			}

			// Use the Connect helper to build a transfer action
			result := swaig.NewFunctionResult(
				fmt.Sprintf("Transferring you to %s now. Please hold.", department),
			)
			result.Connect(number, false, "+15559990000")
			return result
		},
	})

	// ---- Tool: Send a confirmation SMS ----
	a.DefineTool(agent.ToolDefinition{
		Name:        "send_confirmation",
		Description: "Send a confirmation text message to the caller",
		Parameters: map[string]any{
			"phone_number": map[string]any{
				"type":        "string",
				"description": "Phone number to send the text to",
			},
			"message": map[string]any{
				"type":        "string",
				"description": "Confirmation message text",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			phone, _ := args["phone_number"].(string)
			message, _ := args["message"].(string)

			result := swaig.NewFunctionResult(
				fmt.Sprintf("Sending confirmation text to %s.", phone),
			)
			// Use the SendSms helper for the SWML action
			result.SendSms(phone, "+15559990000", message, nil, nil)
			return result
		},
	})

	// ---- Tool: End the call ----
	a.DefineTool(agent.ToolDefinition{
		Name:        "end_call",
		Description: "End the current call after saying goodbye",
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			result := swaig.NewFunctionResult("Thank you for calling Acme Corp. Goodbye!")
			result.Hangup()
			return result
		},
	})

	fmt.Println("Starting CallFlowDemo on :3008/callflow ...")
	fmt.Println("  Pre-answer:  ringback music")
	fmt.Println("  Post-answer: welcome audio")
	fmt.Println("  Post-AI:     hangup")
	fmt.Println("  Tools:       transfer_call, send_confirmation, end_call")

	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
