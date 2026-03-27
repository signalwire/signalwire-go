//go:build ignore

// Example: Provision a SIP-enabled user on Fabric.
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

	// 1. Create a subscriber
	fmt.Println("Creating subscriber...")
	subscriber, err := client.Fabric.Subscribers.Create(map[string]any{
		"name":  "Alice Johnson",
		"email": "alice@example.com",
	})
	if err != nil {
		fmt.Printf("  Create subscriber failed: %v\n", err)
		return
	}
	subID := subscriber["id"].(string)
	innerSubID := subID
	if sub, ok := subscriber["subscriber"].(map[string]any); ok {
		if id, ok := sub["id"].(string); ok {
			innerSubID = id
		}
	}
	fmt.Printf("  Created subscriber: %s\n", subID)

	// 2. Add a SIP endpoint to the subscriber
	fmt.Println("\nCreating SIP endpoint on subscriber...")
	endpoint, err := client.Fabric.Subscribers.CreateSIPEndpoint(subID, map[string]any{
		"username": "alice_sip",
		"password": "SecurePass123!",
	})
	if err != nil {
		fmt.Printf("  Create SIP endpoint failed: %v\n", err)
		return
	}
	epID := endpoint["id"].(string)
	fmt.Printf("  Created SIP endpoint: %s\n", epID)

	// 3. List SIP endpoints on the subscriber
	fmt.Println("\nListing subscriber SIP endpoints...")
	endpoints, err := client.Fabric.Subscribers.ListSIPEndpoints(subID, nil)
	if err == nil {
		if data, ok := endpoints["data"].([]any); ok {
			for _, ep := range data {
				if m, ok := ep.(map[string]any); ok {
					fmt.Printf("  - %s: %v\n", m["id"], m["username"])
				}
			}
		}
	}

	// 4. Get specific SIP endpoint details
	fmt.Printf("\nGetting SIP endpoint %s...\n", epID)
	epDetail, err := client.Fabric.Subscribers.GetSIPEndpoint(subID, epID)
	if err == nil {
		fmt.Printf("  Username: %v\n", epDetail["username"])
	}

	// 5. Create a standalone SIP gateway
	fmt.Println("\nCreating SIP gateway...")
	gateway, err := client.Fabric.SIPGateways.Create(map[string]any{
		"name":       "Office PBX Gateway",
		"uri":        "sip:pbx.example.com",
		"encryption": "required",
		"ciphers":    []string{"AES_256_CM_HMAC_SHA1_80"},
		"codecs":     []string{"PCMU", "PCMA"},
	})
	if err != nil {
		fmt.Printf("  Create SIP gateway failed: %v\n", err)
		return
	}
	gwID := gateway["id"].(string)
	fmt.Printf("  Created SIP gateway: %s\n", gwID)

	// 6. List fabric addresses
	fmt.Println("\nListing fabric addresses...")
	addresses, err := client.Fabric.Addresses.List(nil)
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Fabric addresses failed: %d\n", restErr.StatusCode)
		}
	} else if data, ok := addresses["data"].([]any); ok {
		limit := 5
		if len(data) < limit {
			limit = len(data)
		}
		for _, addr := range data[:limit] {
			if m, ok := addr.(map[string]any); ok {
				fmt.Printf("  - %v\n", m["display_name"])
			}
		}

		// 7. Get a specific fabric address
		if len(data) > 0 {
			if first, ok := data[0].(map[string]any); ok {
				if id, ok := first["id"].(string); ok {
					addrDetail, err := client.Fabric.Addresses.Get(id)
					if err == nil {
						fmt.Printf("  Address detail: %v\n", addrDetail["display_name"])
					}
				}
			}
		}
	}

	// 8. Generate a subscriber token
	fmt.Println("\nGenerating subscriber token...")
	token, err := client.Fabric.Tokens.CreateSubscriberToken(map[string]any{
		"subscriber_id": innerSubID,
		"reference":     innerSubID,
	})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Token generation failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		tokenStr, _ := token["token"].(string)
		if len(tokenStr) > 40 {
			tokenStr = tokenStr[:40]
		}
		fmt.Printf("  Token: %s...\n", tokenStr)
	}

	// 9. Clean up
	fmt.Println("\nCleaning up...")
	client.Fabric.Subscribers.DeleteSIPEndpoint(subID, epID)
	fmt.Printf("  Deleted SIP endpoint %s\n", epID)
	client.Fabric.Subscribers.Delete(subID)
	fmt.Printf("  Deleted subscriber %s\n", subID)
	client.Fabric.SIPGateways.Delete(gwID)
	fmt.Printf("  Deleted SIP gateway %s\n", gwID)
}
