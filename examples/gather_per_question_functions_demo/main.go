// Example: gather_per_question_functions_demo
//
// This example exists to teach one specific gotcha: while a step's
// gather_info is asking questions, ALL of the step's other functions are
// forcibly deactivated. The only callable tools during a gather question are:
//
//   - `gather_submit` (the native answer-submission tool, always active)
//   - Whatever names you list in that question's WithFunctions option
//
// next_step and change_context are also filtered out — the model literally
// cannot navigate away until the gather completes. This is by design: it
// forces a tight ask → submit → next-question loop.
//
// If a question needs to call out to a tool — for example, to validate an
// email format, geocode a ZIP, or look up something from an external service
// — you must list that tool name with WithFunctions on that question. The
// function is active ONLY for that question.
//
// Below: a customer-onboarding gather flow where each question unlocks a
// different validation tool, and where the step's own non-gather tools
// (escalate_to_human, lookup_existing_account) are LOCKED OUT during gather
// because they aren't whitelisted on any question.
//
// Run this file to see the resulting SWML.
package main

import (
	"encoding/json"
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/contexts"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("gather_per_question_functions_demo"),
		agent.WithRoute("/"),
	)

	// Tools that the step would normally have available — but during
	// gather questioning, they're all locked out unless they appear in
	// a question's `functions` whitelist.
	a.DefineTool(agent.ToolDefinition{
		Name:        "validate_email",
		Description: "Validate that an email address is well-formed and deliverable",
		Parameters: map[string]any{
			"email": map[string]any{"type": "string"},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("valid")
		},
	})
	a.DefineTool(agent.ToolDefinition{
		Name:        "geocode_zip",
		Description: "Look up the city/state for a US ZIP code",
		Parameters: map[string]any{
			"zip": map[string]any{"type": "string"},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult(`{"city":"...","state":"..."}`)
		},
	})
	a.DefineTool(agent.ToolDefinition{
		Name:        "check_age_eligibility",
		Description: "Verify the customer is old enough for the product",
		Parameters: map[string]any{
			"age": map[string]any{"type": "integer"},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("eligible")
		},
	})
	// These tools are NOT whitelisted on any gather question. They are
	// registered on the agent and active outside the gather, but during
	// the gather they cannot be called — gather mode locks them out.
	a.DefineTool(agent.ToolDefinition{
		Name:        "escalate_to_human",
		Description: "Transfer the conversation to a live agent",
		Parameters:  map[string]any{},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("transferred")
		},
	})
	a.DefineTool(agent.ToolDefinition{
		Name:        "lookup_existing_account",
		Description: "Search for an existing account by email",
		Parameters: map[string]any{
			"email": map[string]any{"type": "string"},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("not found")
		},
	})

	// Build a single-context agent with one onboarding step.
	cb := a.DefineContexts()
	ctx := cb.AddContext("default")

	onboard := ctx.AddStep("onboard").
		SetText(
			"Onboard a new customer by collecting their details. Use " +
				"gather_info to ask one question at a time. Each question " +
				"may unlock a specific validation tool — only that tool " +
				"and gather_submit are callable while answering it.",
		).
		SetFunctions([]string{
			// Outside of the gather (which is the entire step here),
			// these would be available. During the gather they are
			// forcibly hidden in favor of the per-question whitelists.
			"escalate_to_human",
			"lookup_existing_account",
		})

	onboard.SetGatherInfo(
		"customer",   // outputKey
		"next_step",  // completionAction
		"I'll need to collect a few details to set up your "+
			"account. I'll ask one question at a time.",
	)

	// Question 1: email — only validate_email + gather_submit callable.
	onboard.AddGatherQuestion(
		"email",
		"What's your email address?",
		contexts.WithConfirm(true),
		contexts.WithFunctions([]string{"validate_email"}),
	)

	// Question 2: zip — only geocode_zip + gather_submit callable.
	onboard.AddGatherQuestion(
		"zip",
		"What's your ZIP code?",
		contexts.WithFunctions([]string{"geocode_zip"}),
	)

	// Question 3: age — only check_age_eligibility + gather_submit callable.
	onboard.AddGatherQuestion(
		"age",
		"How old are you?",
		contexts.WithType("integer"),
		contexts.WithFunctions([]string{"check_age_eligibility"}),
	)

	// Question 4: referral_source — no WithFunctions → only gather_submit
	// is callable. The model cannot validate, lookup, escalate — nothing.
	// This is the right pattern when a question needs no tools.
	onboard.AddGatherQuestion(
		"referral_source",
		"How did you hear about us?",
	)

	// A simple confirmation step the gather auto-advances into.
	ctx.AddStep("confirm").
		SetText(
			"Read the collected info back to the customer and " +
				"confirm everything is correct.",
		).
		SetFunctions([]string{}).
		SetEnd(true)

	doc := a.RenderSWML(nil, nil)
	b, _ := json.MarshalIndent(doc, "", "  ")
	fmt.Println(string(b))
}
