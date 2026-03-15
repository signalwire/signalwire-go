//go:build ignore

// Example: gather_info
//
// GatherInfo with typed questions in context steps. Demonstrates using
// the contexts system's gather_info mode for structured data collection
// with sequential question steps.
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-agents-go/pkg/agent"
	"github.com/signalwire/signalwire-agents-go/pkg/contexts"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("PatientIntake"),
		agent.WithRoute("/patient-intake"),
		agent.WithPort(3014),
	)

	a.AddLanguage(map[string]any{
		"name":  "English",
		"code":  "en-US",
		"voice": "rime.spore",
	})

	a.PromptAddSection("Role",
		"You are a friendly medical office intake assistant. "+
			"Collect patient information accurately and professionally.",
		nil,
	)

	// Define a context with gather info steps
	cb := a.DefineContexts()
	ctx := cb.AddContext("default")

	// Step 1: Gather patient demographics
	step1 := ctx.AddStep("demographics")
	step1.SetText("Collect the patient's basic information.")
	step1.SetGatherInfo("patient_demographics", "", "Please collect the following patient information.")
	step1.AddGatherQuestion("full_name", "What is your full name?")
	step1.AddGatherQuestion("date_of_birth", "What is your date of birth?")
	step1.AddGatherQuestion("phone_number", "What is your phone number?", contexts.WithConfirm(true))
	step1.AddGatherQuestion("email", "What is your email address?")
	step1.SetValidSteps([]string{"symptoms"})

	// Step 2: Gather symptoms
	step2 := ctx.AddStep("symptoms")
	step2.SetText("Ask about the patient's current symptoms and reason for visit.")
	step2.SetGatherInfo("patient_symptoms", "", "Now let's talk about why you're visiting today.")
	step2.AddGatherQuestion("reason_for_visit", "What is the main reason for your visit today?")
	step2.AddGatherQuestion("symptom_duration", "How long have you been experiencing these symptoms?")
	step2.AddGatherQuestion("pain_level", "On a scale of 1 to 10, how would you rate your discomfort?")
	step2.SetValidSteps([]string{"confirmation"})

	// Step 3: Confirmation (normal mode, not gather)
	step3 := ctx.AddStep("confirmation")
	step3.SetText(
		"Summarise all the information collected and confirm with the patient " +
			"that everything is correct. Thank them for their time.",
	)
	step3.SetStepCriteria("Patient has confirmed all information is correct")

	fmt.Println("Starting PatientIntake on :3014/patient-intake ...")
	fmt.Println("  Steps: demographics -> symptoms -> confirmation")

	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
