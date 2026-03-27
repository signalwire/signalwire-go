//go:build ignore

// Example: Call queues, recording review, and MFA verification.
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

	// --- Queues ---

	// 1. Create a queue
	fmt.Println("Creating call queue...")
	var queueID string
	queue, err := client.Queues.Create(map[string]any{"name": "Support Queue", "max_size": 50})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Queue creation failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		queueID = queue["id"].(string)
		fmt.Printf("  Created queue: %s\n", queueID)
	}

	// 2. List queues
	fmt.Println("\nListing queues...")
	queues, err := client.Queues.List(nil)
	if err == nil {
		if data, ok := queues["data"].([]any); ok {
			for _, q := range data {
				if m, ok := q.(map[string]any); ok {
					name := m["friendly_name"]
					if name == nil {
						name = m["name"]
					}
					fmt.Printf("  - %s: %v\n", m["id"], name)
				}
			}
		}
	}

	// 3. Get and update queue
	if queueID != "" {
		detail, err := client.Queues.Get(queueID)
		if err == nil {
			name := detail["friendly_name"]
			if name == nil {
				name = detail["name"]
			}
			fmt.Printf("\nQueue detail: %v (max: %v)\n", name, detail["max_size"])
		}

		_, err = client.Queues.Update(queueID, map[string]any{"name": "Priority Support Queue"})
		if err == nil {
			fmt.Println("  Updated queue name")
		}
	}

	// 4. Queue members
	if queueID != "" {
		fmt.Println("\nListing queue members...")
		members, err := client.Queues.ListMembers(queueID, nil)
		if err != nil {
			if restErr, ok := err.(*rest.SignalWireRestError); ok {
				fmt.Printf("  Member ops failed (expected if queue empty): %d\n", restErr.StatusCode)
			}
		} else if data, ok := members["data"].([]any); ok {
			for _, m := range data {
				if member, ok := m.(map[string]any); ok {
					fmt.Printf("  - Member: %v\n", member["call_id"])
				}
			}
		}

		next, err := client.Queues.GetNextMember(queueID)
		if err == nil {
			fmt.Printf("  Next member: %v\n", next)
		}
	}

	// --- Recordings ---

	// 5. List recordings
	fmt.Println("\nListing recordings...")
	recordings, err := client.Recordings.List(nil)
	if err == nil {
		if data, ok := recordings["data"].([]any); ok {
			limit := 5
			if len(data) < limit {
				limit = len(data)
			}
			for _, r := range data[:limit] {
				if m, ok := r.(map[string]any); ok {
					fmt.Printf("  - %s: %vs\n", m["id"], m["duration"])
				}
			}
		}
	}

	// 6. Get recording details
	if recordings != nil {
		if data, ok := recordings["data"].([]any); ok && len(data) > 0 {
			if first, ok := data[0].(map[string]any); ok {
				if id, ok := first["id"].(string); ok {
					recDetail, err := client.Recordings.Get(id)
					if err == nil {
						fmt.Printf("  Recording: %vs, %v\n", recDetail["duration"], recDetail["format"])
					}
				}
			}
		}
	}

	// --- MFA ---

	// 7. Send MFA via SMS
	fmt.Println("\nSending MFA SMS code...")
	var requestID string
	smsResult, err := client.MFA.SMS(map[string]any{
		"to":           "+15551234567",
		"from":         "+15559876543",
		"message":      "Your code is {{code}}",
		"token_length": 6,
	})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  MFA SMS failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		requestID, _ = smsResult["id"].(string)
		if requestID == "" {
			requestID, _ = smsResult["request_id"].(string)
		}
		fmt.Printf("  MFA SMS sent: %s\n", requestID)
	}

	// 8. Send MFA via voice call
	fmt.Println("\nSending MFA voice code...")
	voiceResult, err := client.MFA.Call(map[string]any{
		"to":           "+15551234567",
		"from":         "+15559876543",
		"message":      "Your verification code is {{code}}",
		"token_length": 6,
	})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  MFA call failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		vID, _ := voiceResult["id"].(string)
		if vID == "" {
			vID, _ = voiceResult["request_id"].(string)
		}
		fmt.Printf("  MFA call sent: %s\n", vID)
	}

	// 9. Verify MFA token
	if requestID != "" {
		fmt.Println("\nVerifying MFA token...")
		verify, err := client.MFA.Verify(requestID, map[string]any{"token": "123456"})
		if err != nil {
			if restErr, ok := err.(*rest.SignalWireRestError); ok {
				fmt.Printf("  Verify failed (expected in demo): %d\n", restErr.StatusCode)
			}
		} else {
			fmt.Printf("  Verification result: %v\n", verify)
		}
	}

	// 10. Clean up
	fmt.Println("\nCleaning up...")
	if queueID != "" {
		client.Queues.Delete(queueID)
		fmt.Printf("  Deleted queue %s\n", queueID)
	}
}
