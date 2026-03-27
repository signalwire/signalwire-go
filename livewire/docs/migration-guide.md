# Migrating a LiveKit Agent to LiveWire

This guide walks through converting an existing LiveKit voice agent to run on SignalWire's platform using LiveWire. The process is mechanical -- mostly import path changes -- because LiveWire mirrors LiveKit's API surface.

## Step 1: Change the Import Path

Replace all LiveKit agent imports with the LiveWire package:

```go
// Before (LiveKit)
import (
    "github.com/livekit/agents"
    "github.com/livekit/agents/llm"
    "github.com/livekit/plugins-go/deepgram"
    "github.com/livekit/plugins-go/elevenlabs"
    "github.com/livekit/plugins-go/openai"
    "github.com/livekit/plugins-go/silero"
)

// After (LiveWire)
import "github.com/signalwire/signalwire-go/pkg/livewire"
```

All types are in a single package. No separate plugin packages needed.

## Step 2: Update Type References

Replace LiveKit type names with their LiveWire equivalents. In most cases the names are identical, just with a different package prefix:

```go
// Before (LiveKit)
server := agents.NewAgentServer()
session := agents.NewAgentSession(...)
agent := llm.NewAgent("instructions")

// After (LiveWire)
server := livewire.NewAgentServer()
session := livewire.NewAgentSession(...)
agent := livewire.NewAgent("instructions")
```

## Step 3: Update Option Functions

LiveWire provides the same option functions. Update the package prefix:

```go
// Before (LiveKit)
session := agents.NewAgentSession(
    agents.WithSTT(deepgram.NewSTT()),
    agents.WithTTS(elevenlabs.NewTTS()),
    agents.WithLLM(openai.NewLLM()),
    agents.WithVAD(silero.NewVAD()),
)

// After (LiveWire)
session := livewire.NewAgentSession(
    livewire.WithSTT("deepgram"),
    livewire.WithTTS("elevenlabs"),
    livewire.WithLLM("openai/gpt-4"),
    livewire.WithVAD(livewire.NewSileroVAD()),
)
```

Note: STT, TTS, and VAD providers are accepted but are noops on SignalWire. The platform handles the media pipeline. LLM model selection is honored.

## Step 4: Update Tool Definitions

LiveWire supports two handler signatures:

```go
// LiveKit-style handler (string args, string return)
agent.FunctionTool("get_weather", func(ctx *livewire.RunContext, location string) string {
    return "Sunny, 72F in " + location
}, livewire.WithDescription("Get weather for a location"))

// SignalWire-native handler (full control)
agent.FunctionTool("lookup_order", func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
    orderID, _ := args["order_id"].(string)
    return swaig.NewFunctionResult("Order " + orderID + " is shipped")
}, livewire.WithDescription("Look up an order by ID"),
   livewire.WithParameters(map[string]any{
       "type": "object",
       "properties": map[string]any{
           "order_id": map[string]any{"type": "string", "description": "The order ID"},
       },
   }),
)
```

The LiveKit-style handler is a convenience wrapper. For production agents, the SignalWire-native handler gives you access to the full `FunctionResult` API (actions, data payloads, etc.).

## Step 5: Update the Entrypoint

```go
// Before (LiveKit)
func main() {
    server := agents.NewAgentServer()
    server.SetSetupFunc(func(proc *agents.JobProcess) { /* warmup */ })
    server.RTCSession(entrypoint, agents.WithAgentName("my-agent"))
    agents.RunApp(server)
}

// After (LiveWire)
func main() {
    server := livewire.NewAgentServer()
    server.SetSetupFunc(func(proc *livewire.JobProcess) { /* warmup (noop) */ })
    server.RTCSession(entrypoint, livewire.WithAgentName("my-agent"))
    livewire.RunApp(server)
}
```

## Step 6: Remove Infrastructure Configuration

LiveKit agents typically have configuration for:

- STT API keys and endpoints
- TTS API keys and endpoints
- VAD model paths
- LLM API keys
- WebRTC TURN/STUN servers
- Room service URLs

With LiveWire, none of this is needed. SignalWire's platform manages the entire media pipeline. You can delete all infrastructure configuration.

The only configuration you need:

```bash
# For the agent HTTP server
export PORT=3000  # optional, defaults to 3000

# If using RELAY or REST features
export SIGNALWIRE_PROJECT_ID=your-project-id
export SIGNALWIRE_API_TOKEN=your-api-token
```

## Step 7: Deploy

LiveWire agents are standard HTTP servers. Deploy them anywhere:

```bash
# Build
go build -o myagent .

# Run
./myagent
```

Point your SignalWire phone number at the agent's URL and calls will flow through automatically.

## Complete Before/After Example

### Before (LiveKit)

```go
package main

import (
    "github.com/livekit/agents"
    "github.com/livekit/agents/llm"
    "github.com/livekit/plugins-go/deepgram"
    "github.com/livekit/plugins-go/elevenlabs"
    "github.com/livekit/plugins-go/silero"
)

func main() {
    server := agents.NewAgentServer()
    server.RTCSession(func(ctx *agents.JobContext) {
        ctx.Connect()
        session := agents.NewAgentSession(
            agents.WithSTT(deepgram.NewSTT()),
            agents.WithTTS(elevenlabs.NewTTS()),
            agents.WithVAD(silero.NewVAD().Load()),
        )
        agent := llm.NewAgent("You are a helpful assistant.")
        agent.FunctionTool("greet", func(ctx *llm.RunContext, name string) string {
            return "Hello, " + name + "!"
        })
        session.Start(ctx, agent)
    })
    agents.RunApp(server)
}
```

### After (LiveWire)

```go
package main

import "github.com/signalwire/signalwire-go/pkg/livewire"

func main() {
    server := livewire.NewAgentServer()
    server.RTCSession(func(ctx *livewire.JobContext) {
        ctx.Connect()
        session := livewire.NewAgentSession(
            livewire.WithSTT("deepgram"),
            livewire.WithTTS("elevenlabs"),
            livewire.WithVAD(livewire.NewSileroVAD()),
        )
        agent := livewire.NewAgent("You are a helpful assistant.")
        agent.FunctionTool("greet", func(ctx *livewire.RunContext, name string) string {
            return "Hello, " + name + "!"
        }, livewire.WithDescription("Greet someone by name"))
        session.Start(ctx, agent)
    })
    livewire.RunApp(server)
}
```

The code is nearly identical. The differences are:

1. Single import path instead of multiple plugin packages
2. Provider names are strings instead of struct instances
3. Everything runs on SignalWire's infrastructure -- no STT/TTS/VAD services to manage
