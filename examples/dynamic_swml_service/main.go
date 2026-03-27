//go:build ignore

// Example: dynamic_swml_service
//
// SWML service with dynamic routing. Demonstrates creating a SWML service
// that generates different SWML documents based on the incoming request
// data (caller type, department, VIP status).
package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/signalwire/signalwire-go/pkg/swml"
)

func main() {
	svc := swml.NewService(
		swml.WithName("DynamicGreeting"),
		swml.WithRoute("/greeting"),
		swml.WithPort(3024),
	)

	// Build a default SWML document
	svc.Answer(map[string]any{})
	svc.Play(map[string]any{
		"url": "say:Hello, thank you for calling our service.",
	})
	svc.Prompt(map[string]any{
		"play":       "say:Press 1 for sales, 2 for support, or 3 to leave a message.",
		"max_digits":  1,
		"terminators": "#",
	})
	svc.Hangup(map[string]any{})

	// Register a routing callback that customises the response
	svc.RegisterRoutingCallback("/greeting", func(r *http.Request, body map[string]any) map[string]any {
		if body == nil {
			return nil // Use default document
		}

		callerType, _ := body["caller_type"].(string)
		callerType = strings.ToLower(callerType)
		callerName, _ := body["caller_name"].(string)

		// Build a dynamic document
		doc := swml.NewDocument()

		doc.AddVerb("answer", map[string]any{})

		// Personalised greeting
		if callerName != "" {
			doc.AddVerb("play", map[string]any{
				"url": fmt.Sprintf("say:Hello %s, welcome back to our service!", callerName),
			})
		} else {
			doc.AddVerb("play", map[string]any{
				"url": "say:Hello, thank you for calling our service.",
			})
		}

		// Route based on caller type
		switch callerType {
		case "vip":
			doc.AddVerb("play", map[string]any{
				"url": "say:As a VIP customer, you will be connected to priority support.",
			})
			doc.AddVerb("connect", map[string]any{
				"to":      "+15551234567",
				"timeout": 30,
			})
		case "existing":
			doc.AddVerb("prompt", map[string]any{
				"play":       "say:Press 1 for account management, 2 for support, or 3 for billing.",
				"max_digits":  1,
				"terminators": "#",
			})
		default:
			doc.AddVerb("prompt", map[string]any{
				"play":       "say:Press 1 for sales, 2 for support, or 3 to leave a message.",
				"max_digits":  1,
				"terminators": "#",
			})
		}

		doc.AddVerb("hangup", map[string]any{})
		return doc.ToMap()
	})

	fmt.Println("Starting DynamicGreeting on :3024/greeting ...")
	fmt.Println("  Default: generic menu")
	fmt.Println("  VIP:     POST with {\"caller_type\": \"vip\", \"caller_name\": \"John\"}")

	if err := svc.Serve(); err != nil {
		fmt.Printf("Service error: %v\n", err)
	}
}
