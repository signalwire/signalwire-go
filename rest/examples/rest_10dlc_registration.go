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
	"github.com/signalwire/signalwire-go/pkg/rest/namespaces"
)

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func safe(label string, fn func() (string, error)) string {
	id, err := fn()
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  %s: failed (%d)\n", label, restErr.StatusCode)
		} else {
			fmt.Printf("  %s: failed (%v)\n", label, err)
		}
		return ""
	}
	fmt.Printf("  %s: OK\n", label)
	return id
}

func main() {
	client, err := rest.NewRestClient("", "", "")
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		os.Exit(1)
	}

	// 1. Register a brand
	fmt.Println("Registering 10DLC brand...")
	brandID := safe("Register brand", func() (string, error) {
		resp, err := client.Registry.Brands.Create(map[string]any{
			"company_name": "Acme Corp",
			"ein":          "12-3456789",
			"entity_type":  "PRIVATE_PROFIT",
			"vertical":     "TECHNOLOGY",
			"website":      "https://acme.example.com",
			"country":      "US",
		})
		if err != nil {
			return "", err
		}
		return string(resp.Id), nil
	})

	// 2. List brands
	fmt.Println("\nListing brands...")
	brands, err := client.Registry.Brands.List(nil)
	if err == nil {
		for _, b := range brands.Data {
			fmt.Printf("  - %s: %s\n", b.Id, deref(b.Name))
			if brandID == "" {
				brandID = string(b.Id)
			}
		}
	}

	// 3. Get brand details
	if brandID != "" {
		detail, err := client.Registry.Brands.Get(brandID, nil)
		if err == nil {
			fmt.Printf("\nBrand detail: %v (%v)\n", deref(detail.Name), deref(detail.State))
		}
	}

	// 4. Create a campaign under the brand
	var campaignID string
	if brandID != "" {
		fmt.Println("\nCreating campaign...")
		campaignID = safe("Create campaign", func() (string, error) {
			resp, err := client.Registry.Brands.CreateCampaign(brandID, map[string]any{
				"use_case":       "MIXED",
				"description":    "Customer notifications and support messages",
				"sample_message": "Your order #12345 has shipped.",
			})
			if err != nil {
				return "", err
			}
			return string(resp.Id), nil
		})
	}

	// 5. List campaigns for the brand
	if brandID != "" {
		fmt.Println("\nListing brand campaigns...")
		campaigns, err := client.Registry.Brands.ListCampaigns(brandID, nil)
		if err == nil {
			for _, c := range campaigns.Data {
				fmt.Printf("  - %s: %s\n", c.Id, deref(c.Name))
				if campaignID == "" {
					campaignID = string(c.Id)
				}
			}
		}
	}

	// 6. Get and update campaign
	if campaignID != "" {
		campDetail, err := client.Registry.Campaigns.Get(campaignID, nil)
		if err == nil {
			fmt.Printf("\nCampaign: %v (%v)\n", deref(campDetail.Name), deref(campDetail.State))
		}

		_, err = client.Registry.Campaigns.Update(campaignID, namespaces.RegistryCampaignsUpdateParams{Extras: map[string]any{
			"description": "Updated: customer notifications",
		}})
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
		orderID = safe("Create order", func() (string, error) {
			resp, err := client.Registry.Campaigns.CreateOrder(campaignID, namespaces.RegistryCampaignsCreateOrderParams{Extras: map[string]any{
				"phone_numbers": []string{"+15125551234"},
			}})
			if err != nil {
				return "", err
			}
			return string(resp.Id), nil
		})
	}

	// 8. Get order status
	if orderID != "" {
		orderDetail, err := client.Registry.Orders.Get(orderID, nil)
		if err == nil {
			fmt.Printf("  Order status: %v\n", deref(orderDetail.State))
		}
	}

	// 9. List campaign numbers and orders
	if campaignID != "" {
		fmt.Println("\nListing campaign numbers...")
		numbers, err := client.Registry.Campaigns.ListNumbers(campaignID, nil)
		if err == nil {
			for _, n := range numbers.Data {
				if n.PhoneNumber != nil {
					fmt.Printf("  - %v\n", deref(n.PhoneNumber.Number))
				}
			}
		}

		orders, err := client.Registry.Campaigns.ListOrders(campaignID, nil)
		if err == nil {
			for _, o := range orders.Data {
				fmt.Printf("  - Order %s: %v\n", o.Id, deref(o.State))
			}
		}
	}

	// 10. Unassign numbers (clean up)
	if campaignID != "" {
		fmt.Println("\nUnassigning numbers...")
		nums, err := client.Registry.Campaigns.ListNumbers(campaignID, nil)
		if err == nil {
			for _, n := range nums.Data {
				nID := string(n.Id)
				if _, delErr := client.Registry.Numbers.Delete(nID); delErr == nil {
					fmt.Printf("  Unassigned number %s\n", nID)
				} else if restErr, ok := delErr.(*rest.SignalWireRestError); ok {
					fmt.Printf("  Unassign failed: %d\n", restErr.StatusCode)
				}
			}
		}
	}
}
