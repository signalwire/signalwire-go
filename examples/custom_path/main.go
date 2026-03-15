//go:build ignore

// Example: custom_path
//
// Agent with a custom HTTP path. Demonstrates configuring an agent at
// a non-default route ("/chat") with dynamic per-request personalisation
// based on query parameters.
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-agents-go/pkg/agent"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("ChatAssistant"),
		agent.WithRoute("/chat"),
		agent.WithPort(3012),
	)

	// Base prompt
	a.PromptAddSection("Role",
		"You are a friendly chat assistant ready to help with any questions or conversations.",
		nil,
	)

	// Dynamic per-request personalisation
	a.SetDynamicConfigCallback(func(qp map[string]string, bp map[string]any, headers map[string]string, ep *agent.AgentBase) {
		userName := qp["user_name"]
		if userName == "" {
			userName = "friend"
		}
		topic := qp["topic"]
		if topic == "" {
			topic = "general conversation"
		}
		mood := qp["mood"]
		if mood == "" {
			mood = "friendly"
		}

		ep.PromptAddSection("Personalisation",
			fmt.Sprintf("The user's name is %s. They are interested in discussing %s.", userName, topic),
			nil,
		)

		ep.AddLanguage(map[string]any{
			"name":  "English",
			"code":  "en-US",
			"voice": "rime.spore",
		})

		switch mood {
		case "professional":
			ep.PromptAddSection("Communication Style",
				"Maintain a professional, business-appropriate tone in all interactions.",
				nil,
			)
		case "casual":
			ep.PromptAddSection("Communication Style",
				"Use a casual, relaxed conversational style. Feel free to use informal language.",
				nil,
			)
		default:
			ep.PromptAddSection("Communication Style",
				"Be warm, friendly, and approachable in your responses.",
				nil,
			)
		}

		ep.SetGlobalData(map[string]any{
			"user_name":    userName,
			"topic":        topic,
			"mood":         mood,
			"session_type": "chat",
		})

		ep.AddHints([]string{"chat", "assistant", "help", "conversation"})
	})

	fmt.Println("Starting ChatAssistant on :3012/chat ...")
	fmt.Println("  Try: POST /chat?user_name=Alice&topic=AI&mood=professional")

	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
