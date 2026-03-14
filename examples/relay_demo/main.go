// Example: relay_demo
//
// RELAY WebSocket call control. Demonstrates creating a RELAY client,
// setting an OnCall handler that answers inbound calls, plays TTS audio,
// and then hangs up. Requires SIGNALWIRE_PROJECT_ID, SIGNALWIRE_API_TOKEN,
// and SIGNALWIRE_SPACE environment variables.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/signalwire/signalwire-agents-go/pkg/relay"
)

func main() {
	projectID := os.Getenv("SIGNALWIRE_PROJECT_ID")
	token := os.Getenv("SIGNALWIRE_API_TOKEN")
	space := os.Getenv("SIGNALWIRE_SPACE")

	if projectID == "" || token == "" || space == "" {
		fmt.Println("Set SIGNALWIRE_PROJECT_ID, SIGNALWIRE_API_TOKEN, and SIGNALWIRE_SPACE to run this example.")
		os.Exit(1)
	}

	// Create the RELAY client
	client := relay.NewRelayClient(
		relay.WithProject(projectID),
		relay.WithToken(token),
		relay.WithSpace(space),
		relay.WithContexts("default"),        // Subscribe to the "default" inbound context
		relay.WithMaxActiveCalls(10),          // Limit concurrent calls
	)

	// Set up the inbound call handler
	client.OnCall(func(call *relay.Call) {
		fmt.Printf("Inbound call received: %s (state: %s)\n", call.CallID(), call.State())

		// Answer the call
		if err := call.Answer(); err != nil {
			fmt.Printf("Failed to answer call %s: %v\n", call.CallID(), err)
			return
		}
		fmt.Printf("Call %s answered\n", call.CallID())

		// Play a TTS greeting
		playAction := call.Play([]map[string]any{
			{
				"type": "tts",
				"text": "Hello! Thank you for calling. This is a demo of the SignalWire RELAY SDK for Go. Goodbye!",
			},
		})

		// Wait for the audio to finish (with a timeout)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := playAction.Wait(ctx)
		if err != nil {
			fmt.Printf("Play wait error on call %s: %v\n", call.CallID(), err)
		} else {
			fmt.Printf("Play completed on call %s: %v\n", call.CallID(), result)
		}

		// Hang up the call
		if err := call.Hangup(""); err != nil {
			fmt.Printf("Hangup error on call %s: %v\n", call.CallID(), err)
		} else {
			fmt.Printf("Call %s hung up\n", call.CallID())
		}
	})

	// Run the client (blocking)
	fmt.Printf("RELAY client connecting to %s ...\n", space)
	fmt.Println("Waiting for inbound calls on context 'default'.")

	if err := client.Run(); err != nil {
		fmt.Printf("RELAY client error: %v\n", err)
	}
}
