// Example: rest_demo
//
// REST API usage with the SignalWireClient. Demonstrates creating a client,
// listing phone numbers, and shows other namespace usage patterns.
// Requires SIGNALWIRE_PROJECT_ID, SIGNALWIRE_API_TOKEN, and
// SIGNALWIRE_SPACE environment variables.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/signalwire/signalwire-agents-go/pkg/rest"
)

func main() {
	// Create the REST client (reads from env vars if arguments are empty)
	client, err := rest.NewSignalWireClient("", "", "")
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		fmt.Println("Set SIGNALWIRE_PROJECT_ID, SIGNALWIRE_API_TOKEN, and SIGNALWIRE_SPACE environment variables.")
		os.Exit(1)
	}

	fmt.Println("SignalWire REST client created successfully.")

	// ---- List phone numbers ----
	fmt.Println("\n--- Phone Numbers ---")
	result, err := client.PhoneNumbers.List(nil)
	if err != nil {
		fmt.Printf("Error listing phone numbers: %v\n", err)
	} else {
		prettyPrint("Phone Numbers", result)
	}

	// ---- Other namespace usage patterns (commented for reference) ----

	// Search for available phone numbers:
	//   available, err := client.PhoneNumbers.Search(map[string]string{
	//       "area_code": "312",
	//   })

	// List recordings:
	//   recordings, err := client.Recordings.List(nil)

	// Get a specific recording:
	//   recording, err := client.Recordings.Get("recording-id")

	// Fabric AI Agents:
	//   agents, err := client.Fabric.AIAgents.List(nil)
	//   newAgent, err := client.Fabric.AIAgents.Create(map[string]any{
	//       "name": "My Agent",
	//   })

	// Datasphere documents:
	//   docs, err := client.Datasphere.Documents.List(nil)

	// SIP profiles:
	//   profiles, err := client.SipProfile.List(nil)

	// Verified callers:
	//   callers, err := client.VerifiedCallers.List(nil)

	// Video rooms:
	//   rooms, err := client.Video.Rooms.List(nil)

	// Logs:
	//   logs, err := client.Logs.List(nil)

	// PubSub:
	//   client.PubSub.Publish("channel", map[string]any{"message": "hello"})

	fmt.Println("\nREST demo complete.")
}

// prettyPrint formats and prints a labeled JSON result.
func prettyPrint(label string, data map[string]any) {
	formatted, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("%s: %v\n", label, data)
		return
	}
	fmt.Printf("%s:\n%s\n", label, string(formatted))
}
