//go:build swexample

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

	"github.com/signalwire/signalwire-go/v3/pkg/relay"
)

func main() {
	client := relay.NewRelayClient(
		relay.WithProject(os.Getenv("SIGNALWIRE_PROJECT_ID")),
		relay.WithToken(os.Getenv("SIGNALWIRE_API_TOKEN")),
		relay.WithSpace(os.Getenv("SIGNALWIRE_SPACE")),
		relay.WithContexts("default"),
	)

	client.OnCall(func(call *relay.Call) {
		if err := call.Answer(); err != nil {
			fmt.Printf("answer failed: %v\n", err)
			return
		}
		action := call.Play([]map[string]any{
			{"type": "tts", "params": map[string]any{"text": "Welcome to SignalWire!"}},
		})
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := action.Wait(ctx); err != nil {
			fmt.Printf("play did not finish: %v\n", err)
		}
		if err := call.Hangup(""); err != nil {
			fmt.Printf("hangup failed: %v\n", err)
		}
	})

	fmt.Println("Waiting for inbound calls ...")
	if err := client.Run(); err != nil {
		fmt.Printf("relay client stopped: %v\n", err)
	}
}

// endregion: quickstart
