//go:build ignore

// Example: IVR menu with DTMF collection, playback, and call connect.
//
// Answers an inbound call, plays a greeting, collects a digit, and
// routes the caller based on their choice:
//
//	1 - Hear a sales message
//	2 - Hear a support message
//	0 - Connect to a live agent at +19184238080
//
// Set these env vars (or pass them directly to NewRelayClient):
//
//	SIGNALWIRE_PROJECT_ID   - your SignalWire project ID
//	SIGNALWIRE_API_TOKEN    - your SignalWire API token
//	SIGNALWIRE_SPACE        - your SignalWire space (optional, defaults to relay.signalwire.com)
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

const agentNumber = "+19184238080"

// tts builds a TTS play element.
func tts(text string) map[string]any {
	return map[string]any{"type": "tts", "params": map[string]any{"text": text}}
}

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

		// Play greeting and collect a single digit
		collectAction := call.PlayAndCollect(
			[]map[string]any{
				tts("Welcome to SignalWire!"),
				tts("Press 1 for sales. Press 2 for support. Press 0 to speak with an agent."),
			},
			map[string]any{
				"digits": map[string]any{
					"max":           1,
					"digit_timeout": 5.0,
				},
				"initial_timeout": 10.0,
			},
		)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		resultEvent, err := collectAction.Wait(ctx)
		if err != nil {
			fmt.Printf("Collect failed: %v\n", err)
			call.Hangup("")
			return
		}

		// Extract the collected digit from the result event
		resultType := ""
		digits := ""
		if result, ok := resultEvent.Params["result"].(map[string]any); ok {
			resultType, _ = result["type"].(string)
			if params, ok := result["params"].(map[string]any); ok {
				digits, _ = params["digits"].(string)
			}
		}
		fmt.Printf("Collect result: type=%s digits=%s\n", resultType, digits)

		playCtx, playCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer playCancel()

		switch {
		case resultType == "digit" && digits == "1":
			// Sales
			action := call.Play([]map[string]any{
				tts("Thank you for your interest! A sales representative will be with you shortly."),
			})
			action.Wait(playCtx)

		case resultType == "digit" && digits == "2":
			// Support
			action := call.Play([]map[string]any{
				tts("Please hold while we connect you to our support team."),
			})
			action.Wait(playCtx)

		case resultType == "digit" && digits == "0":
			// Connect to live agent
			action := call.Play([]map[string]any{
				tts("Connecting you to an agent now. Please hold."),
			})
			action.Wait(playCtx)

			fmt.Printf("Connecting to %s\n", agentNumber)

			err := call.Connect(
				[][]map[string]any{
					{
						{
							"type": "phone",
							"params": map[string]any{
								"to_number": agentNumber,
								"timeout":   30,
							},
						},
					},
				},
				relay.WithConnectRingback([]map[string]any{
					tts("Please wait while we connect your call."),
				}),
			)
			if err != nil {
				fmt.Printf("Connect failed: %v\n", err)
				call.Hangup("")
				return
			}

			// Stay on the call until the bridge ends
			endCtx, endCancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer endCancel()
			call.WaitFor(endCtx, "calling.call.state", func(e *relay.RelayEvent) bool {
				return e.GetString("call_state") == relay.CallStateEnded
			})
			fmt.Printf("Connected call ended: %s\n", call.CallID())
			return

		default:
			// No input or invalid
			action := call.Play([]map[string]any{
				tts("We didn't receive a valid selection."),
			})
			action.Wait(playCtx)
		}

		call.Hangup("")
		fmt.Printf("Call ended: %s\n", call.CallID())
	})

	fmt.Println("Waiting for inbound calls on context 'default' ...")
	if err := client.Run(); err != nil {
		fmt.Printf("RELAY client error: %v\n", err)
		os.Exit(1)
	}
}
