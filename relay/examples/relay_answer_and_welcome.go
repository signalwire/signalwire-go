//go:build ignore

// Example: Answer an inbound call and say "Welcome to SignalWire!"
//
// Set these env vars (or pass them directly to NewRelayClient):
//
//	SIGNALWIRE_PROJECT_ID   - your SignalWire project ID
//	SIGNALWIRE_API_TOKEN    - your SignalWire API token
//	SIGNALWIRE_SPACE        - your SignalWire space (e.g. example.signalwire.com)
//
// For full WebSocket / JSON-RPC debug output:
//
//	SIGNALWIRE_LOG_LEVEL=debug
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
		fmt.Printf("Incoming call: %s\n", call)

		if err := call.Answer(); err != nil {
			fmt.Printf("Answer failed: %v\n", err)
			return
		}

		action := call.Play([]map[string]any{
			{"type": "tts", "params": map[string]any{"text": "Welcome to SignalWire!"}},
		})

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		action.Wait(ctx)

		call.Hangup("")
		fmt.Printf("Call ended: %s\n", call.CallID())
	})

	fmt.Println("Waiting for inbound calls on context 'default' ...")
	if err := client.Run(); err != nil {
		fmt.Printf("RELAY client error: %v\n", err)
		os.Exit(1)
	}
}
