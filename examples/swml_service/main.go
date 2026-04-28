//go:build ignore

// Example: swml_service
//
// Basic SWMLService usage (non-AI SWML, IVR-style). Demonstrates creating
// and serving SWML documents with verbs like answer, play, record, prompt,
// switch, connect, and hangup -- without any AI component.
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/swml"
)

func main() {
	// Create a basic SWML service for an IVR menu
	svc := swml.NewService(
		swml.WithName("IVR-Menu"),
		swml.WithRoute("/ivr"),
		swml.WithPort(3010),
	)

	// Build the SWML document: answer, greet, prompt, and route
	maxDuration := 7200
	svc.Answer(&maxDuration, nil)

	welcome := "say:Welcome to our service. Press 1 for sales, 2 for support, or 3 to leave a message."
	svc.Play(&welcome, nil, nil, nil, nil, nil, nil)

	svc.Prompt(map[string]any{
		"play":       "say:Please make your selection now.",
		"max_digits":  1,
		"terminators": "#",
	})

	svc.Switch(map[string]any{
		"variable": "prompt_digits",
		"case": map[string]any{
			"1": []any{
				map[string]any{"play": map[string]any{"url": "say:Connecting you to sales. Please hold."}},
				map[string]any{"connect": map[string]any{"to": "+15551234567"}},
			},
			"2": []any{
				map[string]any{"play": map[string]any{"url": "say:Connecting you to support. Please hold."}},
				map[string]any{"connect": map[string]any{"to": "+15557654321"}},
			},
			"3": []any{
				map[string]any{"play": map[string]any{"url": "say:Please leave a message after the beep."}},
				map[string]any{"sleep": 1000},
				map[string]any{"record": map[string]any{"format": "mp3", "max_length": 120, "terminators": "#"}},
				map[string]any{"play": map[string]any{"url": "say:Thank you for your message. Goodbye!"}},
			},
		},
		"default": []any{
			map[string]any{"play": map[string]any{"url": "say:Sorry, I did not understand your selection."}},
		},
	})

	svc.Hangup(nil)

	// Print the rendered SWML document
	pretty, err := svc.RenderPretty()
	if err != nil {
		fmt.Printf("Render error: %v\n", err)
		return
	}
	fmt.Println("SWML Document:")
	fmt.Println(pretty)

	fmt.Println("\nStarting IVR-Menu on :3010/ivr ...")
	if err := svc.Serve(); err != nil {
		fmt.Printf("Service error: %v\n", err)
	}
}
