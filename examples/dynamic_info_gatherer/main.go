//go:build ignore

// Example: dynamic_info_gatherer
//
// Builds an InfoGathererAgent question set dynamically from the
// INFO_GATHERER_SET env var, selecting between several predefined
// intake flows (default, support, medical, onboarding).
//
// Run with:
//
//	INFO_GATHERER_SET=support go run ./examples/dynamic_info_gatherer/
package main

import (
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/pkg/prefabs"
)

func main() {
	questionSets := map[string][]prefabs.Question{
		"default": {
			{KeyName: "name", QuestionText: "What is your full name?"},
			{KeyName: "phone", QuestionText: "What is the best phone number to reach you?"},
			{KeyName: "reason", QuestionText: "How can I help you today?"},
		},
		"support": {
			{KeyName: "customer_name", QuestionText: "What is your name?"},
			{KeyName: "account_number", QuestionText: "What is your account number?", Confirm: true},
			{KeyName: "issue", QuestionText: "Can you describe the issue you are experiencing?"},
			{KeyName: "priority", QuestionText: "What is the urgency: Low, Medium, or High?"},
		},
		"medical": {
			{KeyName: "patient_name", QuestionText: "What is the patient's full name?", Confirm: true},
			{KeyName: "symptoms", QuestionText: "What symptoms are you experiencing?"},
			{KeyName: "duration", QuestionText: "How long have you had these symptoms?"},
			{KeyName: "medications", QuestionText: "What medications are you currently taking?"},
		},
		"onboarding": {
			{KeyName: "full_name", QuestionText: "What is your full name?"},
			{KeyName: "email", QuestionText: "What is your email address?", Confirm: true},
			{KeyName: "company", QuestionText: "What company do you work for?"},
			{KeyName: "department", QuestionText: "Which department will you be joining?"},
			{KeyName: "start_date", QuestionText: "When will you be starting?"},
		},
	}

	set := os.Getenv("INFO_GATHERER_SET")
	if set == "" {
		set = "default"
	}
	questions, ok := questionSets[set]
	if !ok {
		fmt.Printf("Unknown INFO_GATHERER_SET=%q; falling back to default.\n", set)
		questions = questionSets["default"]
		set = "default"
	}

	ig := prefabs.NewInfoGathererAgent(prefabs.InfoGathererOptions{
		Name:      "DynamicInfoGatherer",
		Route:     "/contact",
		Questions: &questions,
	})

	fmt.Printf("Starting DynamicInfoGatherer (%s set) on :3000/contact ...\n", set)
	fmt.Println("  Select a different set with INFO_GATHERER_SET=support|medical|onboarding")

	if err := ig.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
