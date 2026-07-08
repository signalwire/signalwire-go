//go:build ignore

// Example: quickstart_relay
//
// Minimal RELAY client used as the README quickstart. Connects over WebSocket,
// answers inbound calls, plays a TTS greeting, and hangs up. The `quickstart`
// region below is included byte-identically into README.md via the
// readme-include gate.
// region: quickstart
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/signalwire/signalwire-go/pkg/relay"
)

func main() {
	client := relay.NewRelayClient(
		relay.WithProject(os.Getenv("SIGNALWIRE_PROJECT_ID")),
		relay.WithToken(os.Getenv("SIGNALWIRE_API_TOKEN")),
		relay.WithSpace(os.Getenv("SIGNALWIRE_SPACE")),
		relay.WithContexts("default"),
	)

	client.OnCall(func(call *relay.Call) {
		call.Answer()
		action := call.Play([]map[string]any{
			{"type": "tts", "params": map[string]any{"text": "Welcome to SignalWire!"}},
		})
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		action.Wait(ctx)
		call.Hangup("")
	})

	fmt.Println("Waiting for inbound calls ...")
	client.Run()
}

// endregion: quickstart
