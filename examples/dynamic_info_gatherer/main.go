//go:build ignore

// Example: dynamic_info_gatherer
//
// InfoGathererAgent with a callback function that dynamically selects
// questions based on request parameters (?set=support, ?set=medical, etc.).
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/prefabs"
)

func main() {
	questionSets := map[string][]prefabs.GatherField{
		"default": {
			{Name: "name", Description: "Full name", Required: true},
			{Name: "phone", Description: "Phone number", Required: true},
			{Name: "reason", Description: "How can I help you today?"},
		},
		"support": {
			{Name: "customer_name", Description: "Your name", Required: true},
			{Name: "account_number", Description: "Account number", Required: true},
			{Name: "issue", Description: "Describe the issue"},
			{Name: "priority", Description: "Urgency: Low, Medium, or High"},
		},
		"medical": {
			{Name: "patient_name", Description: "Patient full name", Required: true},
			{Name: "symptoms", Description: "Current symptoms", Required: true},
			{Name: "duration", Description: "Symptom duration"},
			{Name: "medications", Description: "Current medications"},
		},
		"onboarding": {
			{Name: "full_name", Description: "Your full name", Required: true},
			{Name: "email", Description: "Email address", Required: true},
			{Name: "company", Description: "Company name"},
			{Name: "department", Description: "Department"},
			{Name: "start_date", Description: "Start date"},
		},
	}

	ig := prefabs.NewInfoGathererAgent(
		agent.WithName("DynamicInfoGatherer"),
		agent.WithRoute("/contact"),
		agent.WithPort(3033),
	)

	ig.SetQuestionCallback(func(queryParams map[string]string) []prefabs.GatherField {
		set := queryParams["set"]
		if set == "" {
			set = "default"
		}
		fmt.Printf("Dynamic question set: %s\n", set)
		if fields, ok := questionSets[set]; ok {
			return fields
		}
		return questionSets["default"]
	})

	ig.SetOnComplete(func(data map[string]any) {
		fmt.Printf("All fields collected: %v\n", data)
	})

	fmt.Println("Starting DynamicInfoGatherer on :3033/contact ...")
	fmt.Println("  /contact            (default: name, phone, reason)")
	fmt.Println("  /contact?set=support (customer support intake)")
	fmt.Println("  /contact?set=medical (medical intake)")
	fmt.Println("  /contact?set=onboarding (employee onboarding)")

	if err := ig.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
