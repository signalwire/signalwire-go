//go:build ignore

// Example: concierge
//
// ConciergeAgent prefab. Creates a virtual concierge for a luxury resort
// with amenities, services, and built-in tools for availability checks
// and directions.
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-agents-go/pkg/prefabs"
)

func main() {
	concierge := prefabs.NewConciergeAgent(prefabs.ConciergeOptions{
		Name:      "OceanviewConcierge",
		Route:     "/concierge",
		VenueName: "Oceanview Resort",
		Services: []string{
			"room service",
			"spa bookings",
			"restaurant reservations",
			"activity bookings",
			"airport shuttle",
			"valet parking",
		},
		Amenities: map[string]prefabs.Amenity{
			"infinity pool": {
				Hours:    "7:00 AM - 10:00 PM",
				Location: "Main Level, Ocean View",
				Details:  "Heated infinity pool with poolside service and cabanas.",
			},
			"spa": {
				Hours:    "9:00 AM - 8:00 PM",
				Location: "Lower Level, East Wing",
				Details:  "Full-service luxury spa. Reservations required.",
			},
			"fitness center": {
				Hours:    "24 hours",
				Location: "2nd Floor, North Wing",
				Details:  "Cardio equipment, weights, and yoga studio.",
			},
			"beach access": {
				Hours:    "Dawn to Dusk",
				Location: "Southern Pathway",
				Details:  "Private beach with complimentary chairs and towels.",
			},
		},
		Hours: "Front desk 24 hours, Concierge 7 AM - 11 PM",
	})

	fmt.Println("Starting Oceanview Resort Concierge on :3000/concierge ...")
	fmt.Println("  Services: room service, spa, restaurant, activities, shuttle, valet")
	fmt.Println("  Tools: check_availability, get_directions")

	if err := concierge.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
