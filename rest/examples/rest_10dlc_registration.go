//go:build ignore

// Example: 10DLC brand and campaign compliance registration.
//
// WARNING: This example interacts with the real 10DLC registration system.
// Brand and campaign registrations may have side effects and costs.
// Use with caution in production environments.
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

func safe(label string, fn func() (map[string]any, error)) (map[string]any, string) {
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
	id, _ := result["id"].(string)
	return result, id
}

func main() {
	client, err := rest.NewRestClient("", "", "")
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		os.Exit(1)
	}

	// 1. Register a brand
	fmt.Println("Registering 10DLC brand...")
	_, brandID := safe("Register brand", func() (map[string]any, error) {
		return client.Registry.Brands.Create(map[string]any{
			"company_name": "Acme Corp",
			"ein":          "12-3456789",
			"entity_type":  "PRIVATE_PROFIT",
			"vertical":     "TECHNOLOGY",
			"website":      "https://acme.example.com",
			"country":      "US",
		})
	})

	// 2. List brands
	fmt.Println("\nListing brands...")
	brands, err := client.Registry.Brands.List(nil)
	if err == nil {
		if data, ok := brands["data"].([]any); ok {
			for _, b := range data {
				if m, ok := b.(map[string]any); ok {
					fmt.Printf("  - %s: %s\n", m["id"], m["name"])
					if brandID == "" {
						brandID, _ = m["id"].(string)
					}
				}
			}
		}
	}

	// 3. Get brand details
	if brandID != "" {
		detail, err := client.Registry.Brands.Get(brandID)
		if err == nil {
			fmt.Printf("\nBrand detail: %v (%v)\n", detail["name"], detail["state"])
		}
	}

	// 4. Create a campaign under the brand
	var campaignID string
	if brandID != "" {
		fmt.Println("\nCreating campaign...")
		_, campaignID = safe("Create campaign", func() (map[string]any, error) {
			return client.Registry.Brands.CreateCampaign(brandID, map[string]any{
				"use_case":       "MIXED",
				"description":    "Customer notifications and support messages",
				"sample_message": "Your order #12345 has shipped.",
			})
		})
	}

	// 5. List campaigns for the brand
	if brandID != "" {
		fmt.Println("\nListing brand campaigns...")
		campaigns, err := client.Registry.Brands.ListCampaigns(brandID, nil)
		if err == nil {
			if data, ok := campaigns["data"].([]any); ok {
				for _, c := range data {
					if m, ok := c.(map[string]any); ok {
						fmt.Printf("  - %s: %s\n", m["id"], m["name"])
						if campaignID == "" {
							campaignID, _ = m["id"].(string)
						}
					}
				}
			}
		}
	}

	// 6. Get and update campaign
	if campaignID != "" {
		campDetail, err := client.Registry.Campaigns.Get(campaignID)
		if err == nil {
			fmt.Printf("\nCampaign: %v (%v)\n", campDetail["name"], campDetail["state"])
		}

		_, err = client.Registry.Campaigns.Update(campaignID, map[string]any{
			"description": "Updated: customer notifications",
		})
		if err == nil {
			fmt.Println("  Campaign description updated")
		} else if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Campaign update failed: %d\n", restErr.StatusCode)
		}
	}

	// 7. Create an order to assign numbers
	var orderID string
	if campaignID != "" {
		fmt.Println("\nCreating number assignment order...")
		_, orderID = safe("Create order", func() (map[string]any, error) {
			return client.Registry.Campaigns.CreateOrder(campaignID, map[string]any{
				"phone_numbers": []string{"+15125551234"},
			})
		})
	}

	// 8. Get order status
	if orderID != "" {
		orderDetail, err := client.Registry.Orders.Get(orderID)
		if err == nil {
			fmt.Printf("  Order status: %v\n", orderDetail["status"])
		}
	}

	// 9. List campaign numbers and orders
	if campaignID != "" {
		fmt.Println("\nListing campaign numbers...")
		numbers, err := client.Registry.Campaigns.ListNumbers(campaignID, nil)
		if err == nil {
			if data, ok := numbers["data"].([]any); ok {
				for _, n := range data {
					if m, ok := n.(map[string]any); ok {
						fmt.Printf("  - %v\n", m["phone_number"])
					}
				}
			}
		}

		orders, err := client.Registry.Campaigns.ListOrders(campaignID, nil)
		if err == nil {
			if data, ok := orders["data"].([]any); ok {
				for _, o := range data {
					if m, ok := o.(map[string]any); ok {
						fmt.Printf("  - Order %s: %v\n", m["id"], m["status"])
					}
				}
			}
		}
	}

	// 10. Unassign numbers (clean up)
	if campaignID != "" {
		fmt.Println("\nUnassigning numbers...")
		nums, err := client.Registry.Campaigns.ListNumbers(campaignID, nil)
		if err == nil {
			if data, ok := nums["data"].([]any); ok {
				for _, n := range data {
					if m, ok := n.(map[string]any); ok {
						nID, _ := m["id"].(string)
						if _, delErr := client.Registry.Numbers.Delete(nID); delErr == nil {
							fmt.Printf("  Unassigned number %s\n", nID)
						} else if restErr, ok := delErr.(*rest.SignalWireRestError); ok {
							fmt.Printf("  Unassign failed: %d\n", restErr.StatusCode)
						}
					}
				}
			}
		}
	}
}
