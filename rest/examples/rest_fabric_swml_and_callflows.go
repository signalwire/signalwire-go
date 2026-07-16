//go:build ignore

// Example: Deploy a voice application end-to-end with SWML and call flows.
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
	"context"
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/v3/pkg/rest"
)

func main() {
	client, err := rest.NewRestClient("", "", "")
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		os.Exit(1)
	}

	// 1. Create a SWML script
	fmt.Println("Creating SWML script...")
	swml, err := client.Fabric.SWMLScripts.Create(context.Background(), map[string]any{
		"name": "Greeting Script",
		"contents": map[string]any{
			"sections": map[string]any{
				"main": []map[string]any{
					{"play": map[string]any{"url": "say:Hello from SignalWire"}},
				},
			},
		},
	})
	if err != nil {
		fmt.Printf("  Create SWML script failed: %v\n", err)
		return
	}
	swmlID := swml["id"].(string)
	fmt.Printf("  Created SWML script: %s\n", swmlID)

	// 2. List SWML scripts to confirm
	fmt.Println("\nListing SWML scripts...")
	scripts, err := client.Fabric.SWMLScripts.List(context.Background(), nil)
	if err == nil {
		if data, ok := scripts["data"].([]any); ok {
			for _, s := range data {
				if m, ok := s.(map[string]any); ok {
					fmt.Printf("  - %s: %v\n", m["id"], m["display_name"])
				}
			}
		}
	}

	// 3. Create a call flow
	fmt.Println("\nCreating call flow...")
	flow, err := client.Fabric.CallFlows.Create(context.Background(), map[string]any{"title": "Main IVR Flow"})
	if err != nil {
		fmt.Printf("  Create call flow failed: %v\n", err)
		return
	}
	flowID := flow["id"].(string)
	fmt.Printf("  Created call flow: %s\n", flowID)

	// 4. Deploy a version of the call flow
	fmt.Println("\nDeploying call flow version...")
	version, err := client.Fabric.CallFlows.DeployVersion(context.Background(), flowID, map[string]any{"label": "v1"})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Deploy failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		fmt.Printf("  Deployed version: %v\n", version)
	}

	// 5. List call flow versions
	fmt.Println("\nListing call flow versions...")
	versions, err := client.Fabric.CallFlows.ListVersions(context.Background(), flowID, nil)
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  List versions failed: %d\n", restErr.StatusCode)
		}
	} else {
		for _, v := range versions.Data {
			fmt.Printf("  - Version: %v\n", v.Version)
		}
	}

	// 6. List addresses for the call flow
	fmt.Println("\nListing call flow addresses...")
	cfAddrs, err := client.Fabric.CallFlows.ListAddresses(context.Background(), flowID, nil)
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  List addresses failed: %d\n", restErr.StatusCode)
		}
	} else {
		for _, a := range cfAddrs.Data {
			fmt.Printf("  - %v\n", a.DisplayName)
		}
	}

	// 7. Create a SWML webhook as an alternative approach
	fmt.Println("\nCreating SWML webhook...")
	webhook, err := client.Fabric.SWMLWebhooks.Create(context.Background(), map[string]any{
		"name":                "External Handler",
		"primary_request_url": "https://example.com/swml-handler",
	})
	if err != nil {
		fmt.Printf("  Create webhook failed: %v\n", err)
		return
	}
	webhookID := webhook["id"].(string)
	fmt.Printf("  Created webhook: %s\n", webhookID)

	// 8. Clean up
	fmt.Println("\nCleaning up...")
	client.Fabric.SWMLWebhooks.Delete(context.Background(), webhookID)
	fmt.Printf("  Deleted webhook %s\n", webhookID)
	client.Fabric.CallFlows.Delete(context.Background(), flowID)
	fmt.Printf("  Deleted call flow %s\n", flowID)
	client.Fabric.SWMLScripts.Delete(context.Background(), swmlID)
	fmt.Printf("  Deleted SWML script %s\n", swmlID)
}
