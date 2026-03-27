// Example: prefab_survey
//
// Survey prefab agent. The SurveyAgent conducts structured surveys with
// typed questions (rating, multiple choice, yes/no, open-ended). It
// includes built-in response validation and a post-prompt that generates
// a JSON summary of all responses.
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/prefabs"
)

func main() {
	// Create a SurveyAgent with mixed question types
	survey := prefabs.NewSurveyAgent(prefabs.SurveyOptions{
		Name:       "CustomerSurvey",
		Route:      "/survey",
		SurveyName: "Customer Satisfaction Survey",
		BrandName:  "Acme Corp",
		MaxRetries: 3,
		Intro:      "Welcome! We would love to hear your feedback about your recent experience with Acme Corp.",
		Conclusion: "Thank you for your valuable feedback! Your responses help us improve our services.",
		Questions: []prefabs.SurveyQuestion{
			{
				ID:    "overall_satisfaction",
				Text:  "On a scale of 1 to 5, how satisfied are you with our service overall?",
				Type:  "rating",
				Scale: 5,
			},
			{
				ID:   "contact_method",
				Text: "How did you first hear about Acme Corp?",
				Type: "multiple_choice",
				Choices: []string{
					"Social Media",
					"Friend or Family",
					"Online Search",
					"Advertisement",
					"Other",
				},
			},
			{
				ID:   "would_recommend",
				Text: "Would you recommend Acme Corp to a friend or colleague?",
				Type: "yes_no",
			},
			{
				ID:   "improvement_suggestions",
				Text: "What is one thing we could improve?",
				Type: "open_ended",
			},
		},
	})

	fmt.Println("Starting SurveyAgent (CustomerSurvey) on :3000/survey ...")
	fmt.Println("  Questions:")
	fmt.Println("    1. overall_satisfaction (rating 1-5)")
	fmt.Println("    2. contact_method (multiple choice)")
	fmt.Println("    3. would_recommend (yes/no)")
	fmt.Println("    4. improvement_suggestions (open ended)")
	fmt.Println("  Built-in tools: validate_response, log_response")

	if err := survey.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
