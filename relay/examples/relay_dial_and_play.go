//go:build ignore

// Example: Dial a number and play "Welcome to SignalWire" using the RELAY client.
//
// Requires env vars:
//
//	SIGNALWIRE_PROJECT_ID
//	SIGNALWIRE_API_TOKEN
//	SIGNALWIRE_SPACE
//	RELAY_FROM_NUMBER — a number on your SignalWire project
//	RELAY_TO_NUMBER   — destination to call
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/signalwire/signalwire-go/pkg/relay"
)

func main() {
	fromNumber := os.Getenv("RELAY_FROM_NUMBER")
	toNumber := os.Getenv("RELAY_TO_NUMBER")
	if fromNumber == "" || toNumber == "" {
		fmt.Println("Set RELAY_FROM_NUMBER and RELAY_TO_NUMBER env vars")
		os.Exit(1)
	}

	client := relay.NewRelayClient(
		relay.WithProject(os.Getenv("SIGNALWIRE_PROJECT_ID")),
		relay.WithToken(os.Getenv("SIGNALWIRE_API_TOKEN")),
		relay.WithSpace(os.Getenv("SIGNALWIRE_SPACE")),
	)

	// Connect to SignalWire (Run blocks, so we use a goroutine)
	go func() {
		if err := client.Run(); err != nil {
			fmt.Printf("RELAY client error: %v\n", err)
			os.Exit(1)
		}
	}()

	// Give the client a moment to connect and authenticate
	time.Sleep(2 * time.Second)

	// Dial the number
	devices := [][]map[string]any{
		{{"type": "phone", "params": map[string]any{"to_number": toNumber, "from_number": fromNumber}}},
	}
	call, err := client.Dial(devices)
	if err != nil {
		fmt.Printf("Dial failed: %v\n", err)
		client.Stop()
		return
	}
	fmt.Printf("Dialing %s from %s — call_id: %s\n", toNumber, fromNumber, call.CallID())

	// Wait for the call to be answered
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = call.WaitFor(ctx, "calling.call.state", func(e *relay.RelayEvent) bool {
		return e.GetString("call_state") == relay.CallStateAnswered
	})
	if err != nil {
		fmt.Println("No answer — timed out")
		client.Stop()
		return
	}
	fmt.Println("Call answered — playing TTS")

	// Play TTS
	playAction := call.Play([]map[string]any{
		{"type": "tts", "params": map[string]any{"text": "Welcome to SignalWire"}},
	})

	// Wait for playback to finish
	playCtx, playCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer playCancel()
	playAction.Wait(playCtx)
	fmt.Println("Playback finished — hanging up")

	call.Hangup("")

	// Wait for the call to end
	endCtx, endCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer endCancel()
	call.WaitFor(endCtx, "calling.call.state", func(e *relay.RelayEvent) bool {
		return e.GetString("call_state") == relay.CallStateEnded
	})
	fmt.Println("Call ended")

	client.Stop()
	fmt.Println("Disconnected")
}
