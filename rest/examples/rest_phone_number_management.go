//go:build ignore

// Example: Full phone number inventory lifecycle.
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

	// 1. Search for available phone numbers
	fmt.Println("Searching available numbers...")
	available, err := client.PhoneNumbers.Search(map[string]any{
		"area_code":   "512",
		"max_results": 3,
	})
	if err != nil {
		fmt.Printf("  Search failed: %v\n", err)
	} else if data, ok := available["data"].([]any); ok {
		for _, num := range data {
			if m, ok := num.(map[string]any); ok {
				fmt.Printf("  - %v\n", m["e164"])
			}
		}
	}

	// 2. Purchase a number
	fmt.Println("\nPurchasing a phone number...")
	var numID string
	numberE164 := "+15125551234"
	if available != nil {
		if data, ok := available["data"].([]any); ok && len(data) > 0 {
			if first, ok := data[0].(map[string]any); ok {
				if e164, ok := first["e164"].(string); ok {
					numberE164 = e164
				}
			}
		}
	}
	number, err := client.PhoneNumbers.Create(map[string]any{"number": numberE164})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Purchase failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		numID = number["id"].(string)
		fmt.Printf("  Purchased: %s\n", numID)
	}

	// 3. List and get owned numbers
	fmt.Println("\nListing owned numbers...")
	owned, err := client.PhoneNumbers.List(nil)
	if err == nil {
		if data, ok := owned["data"].([]any); ok {
			limit := 5
			if len(data) < limit {
				limit = len(data)
			}
			for _, n := range data[:limit] {
				if m, ok := n.(map[string]any); ok {
					fmt.Printf("  - %v (%s)\n", m["number"], m["id"])
				}
			}
		}
	}

	if numID != "" {
		detail, err := client.PhoneNumbers.Get(numID)
		if err == nil {
			fmt.Printf("  Detail: %v\n", detail["number"])
		}
	}

	// 4. Update a number
	if numID != "" {
		fmt.Printf("\nUpdating number %s...\n", numID)
		_, err := client.PhoneNumbers.Update(numID, map[string]any{"name": "Main Line"})
		if err == nil {
			fmt.Println("  Updated name to 'Main Line'")
		}
	}

	// 5. Create a number group
	fmt.Println("\nCreating number group...")
	var groupID string
	group, err := client.NumberGroups.Create(map[string]any{"name": "Sales Pool"})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Group creation failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		groupID = group["id"].(string)
		fmt.Printf("  Created group: %s\n", groupID)
	}

	// 6. Lookup carrier info
	fmt.Println("\nLooking up carrier info...")
	info, err := client.Lookup.PhoneNumber("+15125551234", nil)
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Lookup failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		if carrier, ok := info["carrier"].(map[string]any); ok {
			fmt.Printf("  Carrier: %v\n", carrier["name"])
		}
	}

	// 7. Create a verified caller
	fmt.Println("\nCreating verified caller...")
	var callerID string
	caller, err := client.VerifiedCallers.Create(map[string]any{"phone_number": "+15125559999"})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Verified caller failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		callerID = caller["id"].(string)
		fmt.Printf("  Created verified caller: %s\n", callerID)
	}

	// 8. Get SIP profile
	fmt.Println("\nGetting SIP profile...")
	profile, err := client.SipProfile.Get()
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  SIP profile failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		fmt.Printf("  SIP profile: %v\n", profile)
	}

	// 9. List short codes
	fmt.Println("\nListing short codes...")
	codes, err := client.ShortCodes.List(nil)
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Short codes failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else if data, ok := codes["data"].([]any); ok {
		for _, sc := range data {
			if m, ok := sc.(map[string]any); ok {
				fmt.Printf("  - %v\n", m["short_code"])
			}
		}
	}

	// 10. Create an address
	fmt.Println("\nCreating address...")
	var addrID string
	addr, err := client.Addresses.Create(map[string]any{
		"friendly_name": "HQ Address",
		"street":        "123 Main St",
		"city":          "Austin",
		"region":        "TX",
		"postal_code":   "78701",
		"iso_country":   "US",
	})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Address creation failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		addrID = addr["id"].(string)
		fmt.Printf("  Created address: %s\n", addrID)
	}

	// 11. Clean up
	fmt.Println("\nCleaning up...")
	if addrID != "" {
		client.Addresses.Delete(addrID)
		fmt.Printf("  Deleted address %s\n", addrID)
	}
	if callerID != "" {
		if _, err := client.VerifiedCallers.Delete(callerID); err != nil {
			if restErr, ok := err.(*rest.SignalWireRestError); ok {
				fmt.Printf("  Verified caller delete failed: %d\n", restErr.StatusCode)
			}
		} else {
			fmt.Printf("  Deleted verified caller %s\n", callerID)
		}
	}
	if groupID != "" {
		client.NumberGroups.Delete(groupID)
		fmt.Printf("  Deleted number group %s\n", groupID)
	}
	if numID != "" {
		if _, err := client.PhoneNumbers.Delete(numID); err != nil {
			if restErr, ok := err.(*rest.SignalWireRestError); ok {
				fmt.Printf("  Release number failed (recently purchased): %d\n", restErr.StatusCode)
			}
		} else {
			fmt.Printf("  Released number %s\n", numID)
		}
	}
}
