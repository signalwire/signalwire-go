// Example: prefab_info_gatherer
//
// Prefab agent usage. The InfoGathererAgent is a pre-built agent pattern
// that collects answers to a series of questions sequentially. It comes
// with built-in tools (start_questions, submit_answer) and prompt sections.
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-agents-go/pkg/prefabs"
)

func main() {
	// Create an InfoGathererAgent with 3 questions
	ig := prefabs.NewInfoGathererAgent(prefabs.InfoGathererOptions{
		Name:  "PatientIntake",
		Route: "/intake",
		Questions: []prefabs.Question{
			{
				KeyName:      "full_name",
				QuestionText: "What is your full legal name?",
				Confirm:      true, // Requires user confirmation
			},
			{
				KeyName:      "date_of_birth",
				QuestionText: "What is your date of birth?",
				Confirm:      true,
			},
			{
				KeyName:      "reason_for_visit",
				QuestionText: "What is the reason for your visit today?",
				Confirm:      false, // No confirmation needed
			},
		},
	})

	fmt.Println("Starting InfoGathererAgent (PatientIntake) on :3000/intake ...")
	fmt.Println("  Questions: full_name, date_of_birth, reason_for_visit")
	fmt.Println("  Built-in tools: start_questions, submit_answer")

	if err := ig.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
