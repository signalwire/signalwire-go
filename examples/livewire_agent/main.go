// Example: livewire_agent
//
// Demonstrates a LiveKit-style agent running on SignalWire's platform.
// Uses familiar LiveKit API symbols — just change your import path.
package main

import (
	"log"

	"github.com/signalwire/signalwire-go/pkg/livewire"
)

func main() {
	server := livewire.NewAgentServer()

	server.RTCSession(func(ctx *livewire.JobContext) {
		if err := ctx.Connect(); err != nil {
			log.Printf("connect failed: %v", err)
			return
		}

		session := livewire.NewAgentSession(
			livewire.WithSTT("deepgram"),
			livewire.WithLLM("openai/gpt-4"),
			livewire.WithTTS("elevenlabs"),
		)

		agent := livewire.NewAgent("You are a helpful weather assistant.")
		agent.FunctionTool("get_weather", func(ctx *livewire.RunContext, location string) string {
			return "The weather in " + location + " is sunny, 72°F"
		}, livewire.WithDescription("Get weather for a location"))

		if err := session.Start(ctx, agent); err != nil {
			log.Printf("session start failed: %v", err)
			return
		}
		session.GenerateReply(livewire.WithReplyInstructions("Greet the user and ask how you can help."))
	})

	livewire.RunApp(server)
}
