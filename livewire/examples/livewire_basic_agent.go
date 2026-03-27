//go:build ignore

// Example: Basic LiveWire agent with a single tool.
//
// Demonstrates the simplest possible LiveWire agent: an AI assistant
// with one function tool. This mirrors the standard LiveKit agent
// pattern -- just with a different import path.
//
// Run:
//
//	go run livewire_basic_agent.go
//
// Then point a SignalWire phone number at http://your-host:3000/
package main

import "github.com/signalwire/signalwire-go/pkg/livewire"

func main() {
	server := livewire.NewAgentServer()

	server.RTCSession(func(ctx *livewire.JobContext) {
		// Connect is a noop on SignalWire -- the platform handles
		// connection lifecycle automatically.
		ctx.Connect()

		// Configure the session. STT and TTS are noops -- SignalWire's
		// control plane handles the media pipeline. LLM model selection
		// is honored.
		session := livewire.NewAgentSession(
			livewire.WithSTT("deepgram"),
			livewire.WithLLM("openai/gpt-4"),
			livewire.WithTTS("elevenlabs"),
		)

		// Create an agent with instructions and a tool
		agent := livewire.NewAgent("You are a helpful weather assistant. When asked about weather, use the get_weather tool.")

		agent.FunctionTool("get_weather", func(ctx *livewire.RunContext, location string) string {
			// In production, call a real weather API here
			return "The weather in " + location + " is sunny, 72F with clear skies."
		}, livewire.WithDescription("Get current weather for a location"))

		// Start the session -- this binds the agent to SignalWire's
		// AgentBase and registers all tools.
		session.Start(ctx, agent)

		// Generate an initial greeting
		session.GenerateReply(
			livewire.WithReplyInstructions("Greet the user and ask how you can help with weather information."),
		)
	})

	livewire.RunApp(server)
}
