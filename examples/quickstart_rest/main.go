//go:build ignore

// Example: quickstart_rest
//
// Minimal REST client used as the README quickstart. Creates a client from
// environment credentials, creates a Fabric AI agent, searches phone numbers,
// and dials a call. The `quickstart` region below is included byte-identically
// into README.md via the readme-include gate.
// region: quickstart
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/v3/pkg/rest"
	"github.com/signalwire/signalwire-go/v3/pkg/rest/namespaces"
)

func main() {
	// Reads from SIGNALWIRE_PROJECT_ID, SIGNALWIRE_API_TOKEN, SIGNALWIRE_SPACE
	client, err := rest.NewRestClient("", "", "")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	client.Fabric.AIAgents.Create(context.Background(), map[string]any{
		"name":   "Support Bot",
		"prompt": map[string]any{"text": "You are helpful."},
	})

	client.Calling.Dial(context.Background(), namespaces.CallingNamespaceDialParams{
		From: "+15559876543",
		To:   "+15551234567",
		URL:  ptr("https://example.com/call-handler"),
	})

	results, _ := client.PhoneNumbers.Search(context.Background(), map[string]string{"areacode": "512"})
	fmt.Println(results)
}

// ptr returns a pointer to v, for setting optional pointer-typed params.
func ptr[T any](v T) *T { return &v }

// endregion: quickstart
