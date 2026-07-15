//go:build ignore

// Example: quickstart_agent
//
// Minimal AI agent used as the README quickstart. Creates an AgentBase, sets a
// prompt, defines a single SWAIG tool, and runs. The `quickstart` region below
// is included byte-identically into README.md via the readme-include gate.
// region: quickstart
package main

import (
	"fmt"
	"time"

	"github.com/signalwire/signalwire-go/v3/pkg/agent"
	"github.com/signalwire/signalwire-go/v3/pkg/swaig"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("my-agent"),
		agent.WithRoute("/agent"),
	)

	a.AddLanguage(map[string]any{
		"name": "English", "code": "en-US", "voice": "rime.spore",
	})
	a.SetPromptText("You are a helpful assistant.")

	a.DefineTool(agent.ToolDefinition{
		Name:        "get_time",
		Description: "Get the current time",
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			now := time.Now().Format("03:04 PM MST")
			return swaig.NewFunctionResult(fmt.Sprintf("The time is %s", now))
		},
	})

	a.Run()
}

// endregion: quickstart
