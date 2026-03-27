# LiveWire -- LiveKit-Compatible Agents on SignalWire

```
    __    _            _       ___
   / /   (_)   _____  | |     / (_)_______
  / /   / / | / / _ \ | | /| / / / ___/ _ \
 / /___/ /| |/ /  __/ | |/ |/ / / /  /  __/
/_____/_/ |___/\___/  |__/|__/_/_/   \___/

 LiveKit-compatible agents powered by SignalWire
```

LiveWire lets you run LiveKit-style voice agents on SignalWire's infrastructure with zero changes to your application logic. Just swap the import path -- SignalWire handles STT, TTS, VAD, LLM orchestration, and call control at scale.

## Quick Start

```go
package main

import "github.com/signalwire/signalwire-go/pkg/livewire"

func main() {
	server := livewire.NewAgentServer()

	server.RTCSession(func(ctx *livewire.JobContext) {
		ctx.Connect()

		session := livewire.NewAgentSession(
			livewire.WithSTT("deepgram"),
			livewire.WithLLM("openai/gpt-4"),
			livewire.WithTTS("elevenlabs"),
		)

		agent := livewire.NewAgent("You are a helpful weather assistant.")
		agent.FunctionTool("get_weather", func(ctx *livewire.RunContext, location string) string {
			return "The weather in " + location + " is sunny, 72F"
		}, livewire.WithDescription("Get weather for a location"))

		session.Start(ctx, agent)
		session.GenerateReply(livewire.WithReplyInstructions("Greet the user and ask how you can help."))
	})

	livewire.RunApp(server)
}
```

## Why LiveWire?

LiveKit agents require you to manage your own STT, TTS, VAD, and LLM infrastructure. Each component is a separate service you configure, deploy, and scale independently. LiveWire provides the same developer-facing API, but SignalWire's control plane handles the entire media pipeline:

- **STT** -- speech recognition runs in SignalWire's cloud at scale
- **TTS** -- text-to-speech runs in SignalWire's cloud at scale
- **VAD** -- voice activity detection is automatic, no configuration needed
- **LLM** -- model orchestration is handled by the platform
- **Call control** -- barge-in, hold, transfer, conferencing all built in

You write the same agent code. SignalWire runs it.

## Feature Mapping

| LiveKit Concept | SignalWire Equivalent | Notes |
|---|---|---|
| `AgentServer` | `LiveServer` / `NewAgentServer()` | Identical API |
| `RTCSession` | `RTCSession()` | Identical API |
| `JobContext.Connect()` | Noop | SignalWire connects automatically |
| `AgentSession` | `AgentSession` | Maps to `AgentBase` internally |
| `Agent` | `Agent` | Holds instructions and tools |
| `FunctionTool` | `FunctionTool()` | Registers SWAIG functions |
| `RunContext` | `RunContext` | Available in tool handlers |
| `WithSTT("deepgram")` | Noop (logged once) | Platform handles STT |
| `WithTTS("elevenlabs")` | Noop (logged once) | Platform handles TTS |
| `WithVAD(silero)` | Noop (logged once) | Platform handles VAD |
| `WithLLM("openai/gpt-4")` | Maps to model param | Model selection works |
| `WithAllowInterruptions` | Maps to barge config | Barge-in control works |
| `WithMinEndpointingDelay` | Maps to `end_of_speech_timeout` | Endpointing works |
| `WithMaxEndpointingDelay` | Maps to `attention_timeout` | Endpointing works |
| `WithMaxToolSteps` | Noop (logged once) | Platform manages depth |
| `SetupFunc` | Noop (logged once) | No warm pools needed |
| `WithServerType` | Noop (logged once) | Platform manages topology |
| `GenerateReply` | Appends to prompt | Triggers initial greeting |
| `Interrupt` | Noop | Barge-in is automatic |
| `UpdateInstructions` | Updates prompt text | Mid-session prompt changes |
| `AgentHandoff` | `AgentHandoff` | Multi-agent handoff |
| `StopResponse` | `StopResponse` | Suppress LLM reply |

## What's Noop'd and Why

Several LiveKit concepts are no-ops on SignalWire because the platform handles them automatically:

- **STT/TTS/VAD providers**: SignalWire's control plane runs the entire speech pipeline. Specifying `WithSTT("deepgram")` is accepted but ignored -- the platform selects optimal providers automatically.
- **JobContext.Connect()**: SignalWire agents connect when the platform invokes the SWML endpoint. There is no manual connection step.
- **SetupFunc / warm pools**: SignalWire manages media infrastructure scaling. No process prewarming is needed.
- **WithServerType**: Server topology (room vs. publisher) is a LiveKit deployment concern. SignalWire abstracts this away.
- **WithMaxToolSteps**: Tool execution depth is managed by the platform.
- **Interrupt()**: Barge-in (caller interrupting the agent) is automatic on SignalWire.

Each noop logs an informational message once so you know it was received but is not needed.

## Provider Stubs

LiveWire includes stub types for common LiveKit plugin providers:

- `NewDeepgramSTT()` -- STT stub
- `NewGoogleSTT()` -- STT stub
- `NewElevenLabsTTS()` -- TTS stub
- `NewCartesiaTTS()` -- TTS stub
- `NewOpenAITTS()` -- TTS stub
- `NewOpenAILLM()` -- LLM stub
- `NewSileroVAD()` -- VAD stub

These exist so that LiveKit code that creates provider instances continues to compile. They have no effect at runtime.

## Documentation

- [Migration Guide](docs/migration-guide.md) -- step-by-step guide for migrating a LiveKit agent to LiveWire
- [LiveWire source code](../pkg/livewire/) -- the full Go implementation

## Examples

- [livewire_basic_agent.go](examples/livewire_basic_agent.go) -- simple agent with a single tool
- [livewire_multi_tool.go](examples/livewire_multi_tool.go) -- agent with multiple function tools and RunContext
- [livewire_handoff.go](examples/livewire_handoff.go) -- multi-agent with AgentHandoff

## Environment Variables

LiveWire agents use the same environment variables as standard SignalWire agents:

| Variable | Description |
|----------|-------------|
| `SIGNALWIRE_PROJECT_ID` | Project ID (if using RELAY features) |
| `SIGNALWIRE_API_TOKEN` | API token (if using RELAY features) |
| `SWML_BASIC_AUTH_USER` | HTTP Basic Auth username (auto-generated if not set) |
| `SWML_BASIC_AUTH_PASSWORD` | HTTP Basic Auth password (auto-generated if not set) |
| `PORT` | HTTP server port (default: 3000) |
