//go:build ignore

// Example: Conference infrastructure, cXML resources, generic routing, and tokens.
//
// Set these env vars (or pass them directly to NewRestClient):
//
//	SIGNALWIRE_PROJECT_ID   - your SignalWire project ID
//	SIGNALWIRE_API_TOKEN    - your SignalWire API token
//	SIGNALWIRE_SPACE        - your SignalWire space (e.g. example.signalwire.com)
//
// For full HTTP debug output:
//
//	SIGNALWIRE_LOG_LEVEL=debug
package main

import (
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/pkg/rest"
)

func main() {
	client, err := rest.NewRestClient("", "", "")
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		os.Exit(1)
	}

	// 1. Create a conference room
	fmt.Println("Creating conference room...")
	room, err := client.Fabric.ConferenceRooms.Create(map[string]any{"name": "team-standup"})
	if err != nil {
		fmt.Printf("  Create conference room failed: %v\n", err)
		return
	}
	roomID := room["id"].(string)
	fmt.Printf("  Created conference room: %s\n", roomID)

	// 2. List conference room addresses
	fmt.Println("\nListing conference room addresses...")
	addrs, err := client.Fabric.ConferenceRooms.ListAddresses(roomID, nil)
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  List addresses failed: %d\n", restErr.StatusCode)
		}
	} else if data, ok := addrs["data"].([]any); ok {
		for _, a := range data {
			if m, ok := a.(map[string]any); ok {
				fmt.Printf("  - %v\n", m["display_name"])
			}
		}
	}

	// 3. Create a cXML script
	fmt.Println("\nCreating cXML script...")
	cxml, err := client.Fabric.CXMLScripts.Create(map[string]any{
		"name":     "Hold Music Script",
		"contents": "<Response><Say>Please hold.</Say><Play>https://example.com/hold.mp3</Play></Response>",
	})
	if err != nil {
		fmt.Printf("  Create cXML script failed: %v\n", err)
		return
	}
	cxmlID := cxml["id"].(string)
	fmt.Printf("  Created cXML script: %s\n", cxmlID)

	// 4. Create a cXML webhook
	fmt.Println("\nCreating cXML webhook...")
	cxmlWH, err := client.Fabric.CXMLWebhooks.Create(map[string]any{
		"name":                "External cXML Handler",
		"primary_request_url": "https://example.com/cxml-handler",
	})
	if err != nil {
		fmt.Printf("  Create cXML webhook failed: %v\n", err)
		return
	}
	cxmlWHID := cxmlWH["id"].(string)
	fmt.Printf("  Created cXML webhook: %s\n", cxmlWHID)

	// 5. Create a relay application
	fmt.Println("\nCreating relay application...")
	relayApp, err := client.Fabric.RelayApplications.Create(map[string]any{
		"name":  "Inbound Handler",
		"topic": "office",
	})
	if err != nil {
		fmt.Printf("  Create relay application failed: %v\n", err)
		return
	}
	relayID := relayApp["id"].(string)
	fmt.Printf("  Created relay application: %s\n", relayID)

	// 6. Generic resources: list all
	fmt.Println("\nListing all fabric resources...")
	resources, err := client.Fabric.Resources.List(nil)
	if err == nil {
		if data, ok := resources["data"].([]any); ok {
			limit := 5
			if len(data) < limit {
				limit = len(data)
			}
			for _, r := range data[:limit] {
				if m, ok := r.(map[string]any); ok {
					fmt.Printf("  - %v: %v\n", m["type"], m["display_name"])
				}
			}
		}
	}

	// 7. Get a specific generic resource
	if resources != nil {
		if data, ok := resources["data"].([]any); ok && len(data) > 0 {
			if first, ok := data[0].(map[string]any); ok {
				if id, ok := first["id"].(string); ok {
					detail, err := client.Fabric.Resources.Get(id)
					if err == nil {
						fmt.Printf("  Resource detail: %v (%v)\n", detail["display_name"], detail["type"])
					}
				}
			}
		}
	}

	// 8. Assign a phone route to a resource (demo)
	fmt.Println("\nAssigning phone route (demo)...")
	_, err = client.Fabric.Resources.AssignPhoneRoute(relayID, map[string]any{
		"phone_number": "+15551234567",
	})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Route assignment failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		fmt.Println("  Phone route assigned")
	}

	// 9. Assign a domain application (demo)
	fmt.Println("\nAssigning domain application (demo)...")
	_, err = client.Fabric.Resources.AssignDomainApplication(relayID, map[string]any{
		"domain": "app.example.com",
	})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Domain assignment failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		fmt.Println("  Domain application assigned")
	}

	// 10. Generate tokens
	fmt.Println("\nGenerating tokens...")
	guest, err := client.Fabric.Tokens.CreateGuestToken(map[string]any{"resource_id": relayID})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Guest token failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		token, _ := guest["token"].(string)
		if len(token) > 40 {
			token = token[:40]
		}
		fmt.Printf("  Guest token: %s...\n", token)
	}

	invite, err := client.Fabric.Tokens.CreateInviteToken(map[string]any{"resource_id": relayID})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Invite token failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		token, _ := invite["token"].(string)
		if len(token) > 40 {
			token = token[:40]
		}
		fmt.Printf("  Invite token: %s...\n", token)
	}

	embed, err := client.Fabric.Tokens.CreateEmbedToken(map[string]any{"resource_id": relayID})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Embed token failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		token, _ := embed["token"].(string)
		if len(token) > 40 {
			token = token[:40]
		}
		fmt.Printf("  Embed token: %s...\n", token)
	}

	// 11. Clean up
	fmt.Println("\nCleaning up...")
	client.Fabric.RelayApplications.Delete(relayID)
	fmt.Printf("  Deleted relay application %s\n", relayID)
	client.Fabric.CXMLWebhooks.Delete(cxmlWHID)
	fmt.Printf("  Deleted cXML webhook %s\n", cxmlWHID)
	client.Fabric.CXMLScripts.Delete(cxmlID)
	fmt.Printf("  Deleted cXML script %s\n", cxmlID)
	client.Fabric.ConferenceRooms.Delete(roomID)
	fmt.Printf("  Deleted conference room %s\n", roomID)
}
