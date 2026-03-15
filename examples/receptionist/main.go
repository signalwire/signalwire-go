//go:build ignore

// Example: receptionist
//
// ReceptionistAgent prefab. Creates a receptionist that greets callers,
// collects their information, and transfers them to the appropriate
// department.
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-agents-go/pkg/prefabs"
)

func main() {
	receptionist := prefabs.NewReceptionistAgent(prefabs.ReceptionistOptions{
		Name:  "AcmeReceptionist",
		Route: "/reception",
		Greeting: "Hello, thank you for calling ACME Corporation. How may I direct your call today?",
		Departments: []prefabs.Department{
			{
				Name:        "sales",
				Description: "Product inquiries, pricing, and purchasing",
				Number:      "+15551235555",
			},
			{
				Name:        "support",
				Description: "Technical assistance, troubleshooting, and bug reports",
				Number:      "+15551236666",
			},
			{
				Name:        "billing",
				Description: "Payment questions, invoices, and subscription changes",
				Number:      "+15551237777",
			},
			{
				Name:        "general",
				Description: "All other inquiries",
				Number:      "+15551238888",
			},
		},
	})

	fmt.Println("Starting ACME Receptionist on :3000/reception ...")
	fmt.Println("  Departments: sales, support, billing, general")
	fmt.Println("  Tools: collect_caller_info, transfer_call")

	if err := receptionist.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
