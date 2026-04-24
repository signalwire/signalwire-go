//go:build ignore

// Example: bind an inbound phone number to an SWML webhook (the happy path).
//
// This is the simplest way to route a SignalWire phone number to a backend
// that returns an SWML document per inbound call. You set call_handler on
// the phone number; the server auto-materializes a swml_webhook Fabric
// resource pointing at your URL. You do NOT need to create the Fabric
// webhook resource manually; you do NOT call AssignPhoneRoute.
//
// Set these env vars (or pass them directly to NewRestClient):
//
//	SIGNALWIRE_PROJECT_ID   - your SignalWire project ID
//	SIGNALWIRE_API_TOKEN    - your SignalWire API token
//	SIGNALWIRE_SPACE        - your SignalWire space (e.g. example.signalwire.com)
//	PHONE_NUMBER_SID        - SID of a phone number you own (pn-...)
//	SWML_WEBHOOK_URL        - your backend's SWML endpoint
package main

import (
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/pkg/rest"
)

func main() {
	pnSID := os.Getenv("PHONE_NUMBER_SID")
	webhookURL := os.Getenv("SWML_WEBHOOK_URL")
	if pnSID == "" || webhookURL == "" {
		fmt.Println("PHONE_NUMBER_SID and SWML_WEBHOOK_URL must be set")
		os.Exit(1)
	}

	client, err := rest.NewRestClient("", "", "")
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		os.Exit(1)
	}

	// The typed helper — one line:
	fmt.Printf("Binding %s to %s ...\n", pnSID, webhookURL)
	if _, err := client.PhoneNumbers.SetSwmlWebhook(pnSID, webhookURL); err != nil {
		fmt.Printf("  Binding failed: %v\n", err)
		os.Exit(1)
	}

	// The equivalent wire-level form (use this if you need unusual fields):
	//
	//	import "github.com/signalwire/signalwire-go/pkg/rest/namespaces"
	//
	//	client.PhoneNumbers.Update(pnSID, map[string]any{
	//	    "call_handler":          string(namespaces.PhoneCallHandlerRelayScript),
	//	    "call_relay_script_url": webhookURL,
	//	})

	// Verify: the server auto-created a swml_webhook Fabric resource.
	pn, err := client.PhoneNumbers.Get(pnSID)
	if err != nil {
		fmt.Printf("  Verify failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  call_handler = %v\n", pn["call_handler"])
	fmt.Printf("  call_relay_script_url = %v\n", pn["call_relay_script_url"])
	fmt.Printf("  calling_handler_resource_id (server-derived) = %v\n",
		pn["calling_handler_resource_id"])

	// To route to something other than an SWML webhook, use:
	//
	//	client.PhoneNumbers.SetCxmlWebhook(sid, url, nil)            // LAML / Twilio-compat
	//	client.PhoneNumbers.SetAiAgent(sid, agentID)                 // AI Agent
	//	client.PhoneNumbers.SetCallFlow(sid, flowID, nil)            // Call Flow
	//	client.PhoneNumbers.SetRelayApplication(sid, name)           // Named RELAY app
	//	client.PhoneNumbers.SetRelayTopic(sid, topic, nil)           // RELAY topic
}
