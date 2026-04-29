//go:build ignore

// Example: Twilio-compatible LAML migration -- phone numbers, messaging, calls,
// conferences, queues, recordings, project tokens, PubSub/Chat, and logs.
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

func safeCompat(label string, fn func() (map[string]any, error)) (map[string]any, string) {
	result, err := fn()
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  %s: failed (%d)\n", label, restErr.StatusCode)
		} else {
			fmt.Printf("  %s: failed (%v)\n", label, err)
		}
		return nil, ""
	}
	fmt.Printf("  %s: OK\n", label)
	sid, _ := result["sid"].(string)
	if sid == "" {
		sid, _ = result["id"].(string)
	}
	return result, sid
}

func main() {
	client, err := rest.NewRestClient("", "", "")
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		os.Exit(1)
	}

	// --- Compat Phone Numbers ---

	// 1. Search available numbers
	fmt.Println("Searching compat phone numbers...")
	safeCompat("Search local", func() (map[string]any, error) {
		return client.Compat.PhoneNumbers.SearchLocal("US", map[string]string{"AreaCode": "512"})
	})
	safeCompat("Search toll-free", func() (map[string]any, error) {
		return client.Compat.PhoneNumbers.SearchTollFree("US", nil)
	})
	safeCompat("List countries", func() (map[string]any, error) {
		return client.Compat.PhoneNumbers.ListAvailableCountries(nil)
	})

	// 2. Purchase a number (demo -- will fail without valid number)
	fmt.Println("\nPurchasing compat number...")
	_, numSID := safeCompat("Purchase", func() (map[string]any, error) {
		return client.Compat.PhoneNumbers.Purchase(map[string]any{"PhoneNumber": "+15125551234"})
	})

	// --- LaML Bin & Application ---

	// 3. Create a LaML bin and application
	fmt.Println("\nCreating LaML resources...")
	_, lamlSID := safeCompat("LaML bin", func() (map[string]any, error) {
		return client.Compat.LamlBins.Create(map[string]any{
			"Name":     "Hold Music",
			"Contents": "<Response><Say>Please hold.</Say></Response>",
		})
	})
	_, appSID := safeCompat("Application", func() (map[string]any, error) {
		return client.Compat.Applications.Create(map[string]any{
			"FriendlyName": "Demo App",
			"VoiceUrl":     "https://example.com/voice",
		})
	})

	// --- Messaging ---

	// 4. Send an SMS (demo -- requires valid numbers)
	fmt.Println("\nMessaging operations...")
	_, msgSID := safeCompat("Send SMS", func() (map[string]any, error) {
		return client.Compat.Messages.Create(map[string]any{
			"From": "+15559876543",
			"To":   "+15551234567",
			"Body": "Hello from SignalWire!",
		})
	})

	// 5. List and get messages
	safeCompat("List messages", func() (map[string]any, error) {
		return client.Compat.Messages.List(nil)
	})
	if msgSID != "" {
		safeCompat("Get message", func() (map[string]any, error) {
			return client.Compat.Messages.Get(msgSID)
		})
	}

	// --- Calls ---

	// 6. Outbound call with recording and streaming
	fmt.Println("\nCall operations...")
	_, callSID := safeCompat("Create call", func() (map[string]any, error) {
		return client.Compat.Calls.Create(map[string]any{
			"From": "+15559876543",
			"To":   "+15551234567",
			"Url":  "https://example.com/voice-handler",
		})
	})
	if callSID != "" {
		safeCompat("Start recording", func() (map[string]any, error) {
			return client.Compat.Calls.StartRecording(callSID, nil)
		})
	}

	// --- Conferences ---

	// 7. Conference operations
	fmt.Println("\nConference operations...")
	confs, _ := safeCompat("List conferences", func() (map[string]any, error) {
		return client.Compat.Conferences.List(nil)
	})
	var confSID string
	if confs != nil {
		if data, ok := confs["data"].([]any); ok && len(data) > 0 {
			if m, ok := data[0].(map[string]any); ok {
				confSID, _ = m["sid"].(string)
			}
		}
	}
	if confSID != "" {
		safeCompat("Get conference", func() (map[string]any, error) {
			return client.Compat.Conferences.Get(confSID)
		})
	}

	// --- Queues ---

	// 8. Queue operations
	fmt.Println("\nQueue operations...")
	_, qSID := safeCompat("Create queue", func() (map[string]any, error) {
		return client.Compat.Queues.Create(map[string]any{"FriendlyName": "compat-support-queue"})
	})

	// --- Recordings & Transcriptions ---

	// 9. Recordings and transcriptions
	fmt.Println("\nRecordings and transcriptions...")
	safeCompat("List recordings", func() (map[string]any, error) {
		return client.Compat.Recordings.List(nil)
	})
	safeCompat("List transcriptions", func() (map[string]any, error) {
		return client.Compat.Transcriptions.List(nil)
	})

	// --- Compat Accounts & Tokens ---

	// 10. Accounts and compat tokens
	fmt.Println("\nAccounts and compat tokens...")
	safeCompat("List accounts", func() (map[string]any, error) {
		return client.Compat.Accounts.List(nil)
	})

	// --- Project Tokens ---

	// 11. Project token management
	fmt.Println("\nProject tokens...")
	projToken, projTokenID := safeCompat("Create project token", func() (map[string]any, error) {
		return client.Project.Tokens.Create(map[string]any{
			"name":        "CI Token",
			"permissions": []string{"calling", "messaging", "video"},
		})
	})
	if projToken != nil && projTokenID != "" {
		safeCompat("Update project token", func() (map[string]any, error) {
			return client.Project.Tokens.Update(projTokenID, map[string]any{
				"name": "CI Token (updated)",
			})
		})
		if _, err := client.Project.Tokens.Delete(projTokenID); err == nil {
			fmt.Println("  Delete project token: OK")
		}
	}

	// --- PubSub & Chat Tokens ---

	// 12. PubSub and Chat tokens
	fmt.Println("\nPubSub and Chat tokens...")
	safeCompat("PubSub token", func() (map[string]any, error) {
		return client.PubSub.CreateToken(map[string]any{
			"channels": map[string]any{"notifications": map[string]any{"read": true, "write": true}},
			"ttl":      3600,
		})
	})
	safeCompat("Chat token", func() (map[string]any, error) {
		return client.Chat.CreateToken(map[string]any{
			"member_id": "user-alice",
			"channels":  map[string]any{"general": map[string]any{"read": true, "write": true}},
			"ttl":       3600,
		})
	})

	// --- Logs ---

	// 13. Log queries
	fmt.Println("\nQuerying logs...")
	safeCompat("Message logs", func() (map[string]any, error) {
		return client.Logs.Messages.List(nil)
	})
	safeCompat("Voice logs", func() (map[string]any, error) {
		return client.Logs.Voice.List(nil)
	})
	safeCompat("Fax logs", func() (map[string]any, error) {
		return client.Logs.Fax.List(nil)
	})
	safeCompat("Conference logs", func() (map[string]any, error) {
		return client.Logs.Conferences.List(nil)
	})

	// Get specific log entries with events
	voiceLogs, err := client.Logs.Voice.List(nil)
	if err == nil {
		if data, ok := voiceLogs["data"].([]any); ok && len(data) > 0 {
			if first, ok := data[0].(map[string]any); ok {
				if id, ok := first["id"].(string); ok && id != "" {
					safeCompat("Voice log detail", func() (map[string]any, error) {
						return client.Logs.Voice.Get(id)
					})
					safeCompat("Voice log events", func() (map[string]any, error) {
						return client.Logs.Voice.ListEvents(id, nil)
					})
				}
			}
		}
	}

	// --- Clean up ---

	fmt.Println("\nCleaning up...")
	if qSID != "" {
		if _, err := client.Compat.Queues.Delete(qSID); err == nil {
			fmt.Printf("  Deleted queue %s\n", qSID)
		}
	}
	if appSID != "" {
		if _, err := client.Compat.Applications.Delete(appSID); err == nil {
			fmt.Printf("  Deleted application %s\n", appSID)
		}
	}
	if lamlSID != "" {
		if _, err := client.Compat.LamlBins.Delete(lamlSID); err == nil {
			fmt.Printf("  Deleted LaML bin %s\n", lamlSID)
		}
	}
	if numSID != "" {
		if _, err := client.Compat.PhoneNumbers.Delete(numSID); err == nil {
			fmt.Printf("  Deleted number %s\n", numSID)
		}
	}
}
