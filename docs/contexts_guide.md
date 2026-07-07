# Contexts and Steps Guide

## Table of Contents

- [Overview](#overview)
- [Core Concepts](#core-concepts)
- [Getting Started](#getting-started)
- [API Reference](#api-reference)
- [Navigation and Flow Control](#navigation-and-flow-control)
- [Function Restrictions](#function-restrictions)
- [Real-World Examples](#real-world-examples)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)
- [Migration from POM](#migration-from-pom)

## Overview

The **Contexts and Steps** system enhances traditional Prompt Object Model (POM) prompts in SignalWire AI agents by adding structured workflows on top of your base prompt. Instead of just defining a single prompt, you create workflows with explicit steps, navigation rules, and completion criteria. Steps can restrict which SWAIG (SignalWire AI Gateway) functions are available at each stage of the conversation.

### Key Benefits

- **Structured Workflows**: Define clear, step-by-step processes
- **Explicit Navigation**: Control exactly where users can go next
- **Function Restrictions**: Limit AI tool access per step
- **Completion Criteria**: Define clear progression requirements
- **Context Isolation**: Separate different conversation flows
- **Debugging**: Easier to trace and debug complex interactions

### When to Use Contexts vs Traditional Prompts

**Use Contexts and Steps when:**
- Building multi-step workflows (onboarding, support tickets, applications)
- Need explicit navigation control between conversation states
- Want to restrict function access based on conversation stage
- Building complex customer service or troubleshooting flows
- Creating guided experiences with clear progression

**Use Traditional Prompts when:**
- Building simple, freeform conversational agents
- Want maximum flexibility in conversation flow
- Creating general-purpose assistants
- Prototyping or building simple proof-of-concepts

## Core Concepts

### Contexts

A **Context** represents a conversation state or workflow area. Contexts can be:

- **Workflow Container**: Simple step organization without state changes
- **Context Switch**: Triggers conversation state changes when entered

Each context can define:

- **Steps**: Individual workflow stages within the context
- **Context Prompts**: Guidance that applies to all steps in the context  
- **Entry Parameters**: Control conversation state when context is entered
- **Navigation Rules**: Which other contexts can be accessed

### Context Entry Parameters

When entering a context, these parameters control conversation behavior:

- **`post_prompt`**: Override the agent's post prompt for this context
- **`system_prompt`**: Trigger conversation reset with new instructions
- **`consolidate`**: Summarize previous conversation in new prompt
- **`full_reset`**: Complete system prompt replacement vs injection
- **`user_prompt`**: Inject user message for context establishment

**Important**: If `system_prompt` is present, the context becomes a "Context Switch Context" that processes entry parameters like a `context_switch` SWAIG action. Without `system_prompt`, it's a "Workflow Container Context" that only organizes steps.

### Context Prompts

Contexts can have their own prompts (separate from entry parameters):

```go
// Simple string prompt
context.SetPrompt("Context-specific guidance")

// POM-style sections
context.AddSection("Department", "Billing Department")
context.AddBullets("Services", []string{"Payments", "Refunds", "Account inquiries"})
```

Context prompts provide guidance that applies to all steps within that context, creating a prompt hierarchy: Base Agent Prompt → Context Prompt → Step Prompt.

### Steps

A **Step** is a specific stage within a context. Each step defines:

- **Prompt Content**: What the AI says/does (text or POM sections)
- **Completion Criteria**: When the step is considered complete
- **Navigation Rules**: Where the user can go next
- **Function Access**: Which AI tools are available

### Navigation Control

The system provides fine-grained control over conversation flow:

- **Valid Steps**: Control movement within a context
- **Valid Contexts**: Control switching between contexts  
- **Implicit Navigation**: Automatic "next" step progression
- **Explicit Navigation**: User must explicitly choose next step

## Getting Started

### Basic Single-Context Workflow

```go
package main

import "github.com/signalwire/signalwire-go/pkg/agent"

func main() {
	a := agent.NewAgentBase(
		agent.WithName("Onboarding Assistant"),
		agent.WithRoute("/onboarding"),
	)

	// Define contexts (replaces traditional prompt setup)
	contexts := a.DefineContexts()

	// Single context must be named "default"
	workflow := contexts.AddContext("default")

	// Step 1: Welcome
	workflow.AddStep("welcome").
		SetText("Welcome to our service! Let's get you set up. What's your name?").
		SetStepCriteria("User has provided their name").
		SetValidSteps([]string{"collect_email"})

	// Step 2: Collect Email
	workflow.AddStep("collect_email").
		SetText("Thanks! Now I need your email address to create your account.").
		SetStepCriteria("Valid email address has been provided").
		SetValidSteps([]string{"confirm_details"})

	// Step 3: Confirmation
	workflow.AddStep("confirm_details").
		SetText("Perfect! Let me confirm your details before we proceed.").
		SetStepCriteria("User has confirmed their information").
		SetValidSteps([]string{"complete"})

	// Step 4: Completion
	workflow.AddStep("complete").
		SetText("All set! Your account has been created successfully.")
		// No SetValidSteps = end of workflow

	if err := a.Run(); err != nil {
		panic(err)
	}
}
```

### Multi-Context Workflow

```go
package main

import "github.com/signalwire/signalwire-go/pkg/agent"

func main() {
	a := agent.NewAgentBase(
		agent.WithName("Customer Service"),
		agent.WithRoute("/service"),
	)

	// Add skills for enhanced capabilities
	a.AddSkill("datetime", nil)
	a.AddSkill("web_search", map[string]any{
		"api_key":          "your-api-key",
		"search_engine_id": "your-engine-id",
	})

	contexts := a.DefineContexts()

	// Main triage context
	triage := contexts.AddContext("triage")
	triage.AddStep("greeting").
		AddSection("Current Task", "Understand the customer's need and route appropriately").
		AddBullets("Required Information", []string{
			"Type of issue they're experiencing",
			"Urgency level of the problem",
			"Previous troubleshooting attempts",
		}).
		SetStepCriteria("Customer's need has been identified").
		SetValidContexts([]string{"technical", "billing", "general"})

	// Technical support context
	tech := contexts.AddContext("technical")
	tech.AddStep("technical_help").
		AddSection("Current Task", "Help diagnose and resolve technical issues").
		AddSection("Available Tools", "Use web search and datetime functions for technical solutions").
		SetFunctions([]string{"web_search", "datetime"}).
		SetStepCriteria("Issue is resolved or escalated").
		SetValidContexts([]string{"triage"})

	// Billing context (restricted functions for security)
	billing := contexts.AddContext("billing")
	billing.AddStep("billing_help").
		SetText("I'll help with your billing question. For security, please provide your account verification.").
		SetFunctions("none").
		SetStepCriteria("Billing issue is addressed").
		SetValidContexts([]string{"triage"})

	// General inquiries context
	general := contexts.AddContext("general")
	general.AddStep("general_help").
		SetText("I'm here to help with general questions. What can I assist you with?").
		SetFunctions([]string{"web_search", "datetime"}).
		SetStepCriteria("Question has been answered").
		SetValidContexts([]string{"triage"})

	if err := a.Run(); err != nil {
		panic(err)
	}
}
```

## API Reference

### ContextBuilder

The main entry point for defining contexts and steps.

```go
// Get the builder
contexts := a.DefineContexts()

// Create contexts
context := contexts.AddContext(name string) // returns *contexts.Context
```

### Context

Represents a conversation context or workflow state.

```go
// Context represents a conversation context or workflow state.
// All setters return *Context for fluent chaining.

func (c *Context) AddStep(name string) *Step
// Create a new step in this context

func (c *Context) SetValidContexts(ctxs []string) *Context
// Set which contexts can be accessed from this context

// Context entry parameters
func (c *Context) SetPostPrompt(prompt string) *Context
// Override post prompt for this context

func (c *Context) SetSystemPrompt(prompt string) *Context
// Trigger context switch with new system prompt

func (c *Context) SetConsolidate(consolidate bool) *Context
// Consolidate conversation history when entering

func (c *Context) SetFullReset(fullReset bool) *Context
// Full system prompt replacement vs injection

func (c *Context) SetUserPrompt(prompt string) *Context
// Inject user message for context

// Context prompts
func (c *Context) SetPrompt(prompt string) *Context
// Set simple string prompt for context

func (c *Context) AddSection(title, body string) *Context
// Add POM section to context prompt

func (c *Context) AddBullets(title string, bullets []string) *Context
// Add POM bullet section to context prompt

// Context isolation and fillers
func (c *Context) SetIsolated(isolated bool) *Context
// Mark context as isolated (independent conversation state)

func (c *Context) SetEnterFillers(fillers map[string][]string) *Context
// Set fillers spoken when entering this context

func (c *Context) SetExitFillers(fillers map[string][]string) *Context
// Set fillers spoken when exiting this context

func (c *Context) AddEnterFiller(langCode string, fillers []string) *Context
// Add enter fillers for a specific language

func (c *Context) AddExitFiller(langCode string, fillers []string) *Context
// Add exit fillers for a specific language
```

#### Methods

- `AddStep(name)`: Create and return a new Step
- `SetValidContexts(ctxs)`: Allow navigation to specified contexts
- `SetPostPrompt(prompt)`: Override agent's post prompt for this context
- `SetSystemPrompt(prompt)`: Trigger context switch behavior (makes this a Context Switch Context)
- `SetConsolidate(bool)`: Whether to consolidate conversation when entering
- `SetFullReset(bool)`: Complete vs partial context reset
- `SetUserPrompt(prompt)`: User message to inject when entering context
- `SetPrompt(text)`: Simple string prompt for context
- `AddSection(title, body)`: Add POM section to context prompt
- `AddBullets(title, list)`: Add POM bullet section to context prompt
- `SetIsolated(bool)`: Mark context as isolated (independent conversation state)
- `SetEnterFillers(map)`: Set all enter fillers by language code
- `SetExitFillers(map)`: Set all exit fillers by language code
- `AddEnterFiller(lang, list)`: Add enter fillers for a specific language
- `AddExitFiller(lang, list)`: Add exit fillers for a specific language

### Step

Represents a single step within a context workflow.

```go
// Step represents a single step within a context workflow.
// All setters return *Step for fluent chaining.

// Content definition (choose one approach)
func (s *Step) SetText(text string) *Step
// Set direct text prompt (mutually exclusive with POM sections)

func (s *Step) AddSection(title, body string) *Step
// Add a POM-style section (mutually exclusive with SetText)

func (s *Step) AddBullets(title string, bullets []string) *Step
// Add a POM bullet section to this step

// Flow control
func (s *Step) SetStepCriteria(criteria string) *Step
// Define completion criteria for this step

func (s *Step) SetValidSteps(steps []string) *Step
// Set which steps can be accessed next in same context

func (s *Step) SetValidContexts(contexts []string) *Step
// Set which contexts can be accessed from this step

// Function restrictions — accepts []string of names, []string{} to disable
// all, or the string "none" as a synonym for disable-all.
func (s *Step) SetFunctions(functions any) *Step
// Restrict available functions

// Reset behavior when entering step
func (s *Step) SetResetSystemPrompt(prompt string) *Step
// Reset system prompt when entering this step

func (s *Step) SetResetUserPrompt(prompt string) *Step
// Reset user prompt when entering this step

func (s *Step) SetResetConsolidate(consolidate bool) *Step
// Consolidate conversation when entering this step

func (s *Step) SetResetFullReset(fullReset bool) *Step
// Full conversation reset when entering this step
```

#### Content Methods

**Option 1: Direct Text**
```go
step.SetText("Direct prompt text for the AI")
```

**Option 2: POM-Style Sections**
```go
step.AddSection("Role", "You are a helpful assistant").
	AddSection("Instructions", "Help users with their questions").
	AddBullets("Guidelines", []string{"Be friendly", "Ask clarifying questions"})
```

**Note**: You cannot mix `SetText()` with `AddSection()` in the same step.

#### Navigation Methods

```go
// Control step progression within context
step.SetValidSteps([]string{"step1", "step2"}) // Can go to step1 or step2
step.SetValidSteps([]string{})                  // Cannot progress (dead end)
// No SetValidSteps() call = implicit "next" step

// Control context switching
step.SetValidContexts([]string{"context1", "context2"}) // Can switch contexts
step.SetValidContexts([]string{})                        // Trapped in current context
// No SetValidContexts() call = inherit from context level
```

#### Function Restriction Methods

```go
// Allow specific functions only
step.SetFunctions([]string{"datetime", "math"})

// Block all functions
step.SetFunctions("none")

// No restriction (default - all agent functions available)
// Don't call SetFunctions() at all
```

## Navigation and Flow Control

### Step Navigation Rules

The `set_valid_steps()` method controls movement within a context:

```go
// Explicit step list - can only go to these steps
step.SetValidSteps([]string{"review", "edit", "cancel"})

// Empty list - dead end, cannot progress
step.SetValidSteps([]string{})

// Not called - implicit "next" step progression
// (will go to the next step defined in the context)
```

### Context Navigation Rules

The `set_valid_contexts()` method controls switching between contexts:

```go
// Can switch to these contexts
step.SetValidContexts([]string{"billing", "technical", "general"})

// Trapped in current context
step.SetValidContexts([]string{})

// Not called - inherit from context-level settings
```

### Navigation Inheritance

Context-level navigation settings are inherited by steps:

```go
// Set at context level
context.SetValidContexts([]string{"main", "help"})

// All steps in this context can access main and help contexts
// unless overridden at step level
step.SetValidContexts([]string{"main"}) // Override - only main allowed
```

### Complete Navigation Example

```go
contexts := a.DefineContexts()

// Main context
main := contexts.AddContext("main")
main.SetValidContexts([]string{"help", "settings"}) // Context-level setting

main.AddStep("welcome").
	SetText("Welcome! How can I help you?").
	SetValidSteps([]string{"menu"}) // Must go to menu
	// Inherits context-level valid_contexts

main.AddStep("menu").
	SetText("Choose an option: 1) Help 2) Settings 3) Continue").
	SetValidContexts([]string{"help", "settings", "main"}) // Override context setting
	// No valid_steps = this is a branching point

// Help context
helpCtx := contexts.AddContext("help")
helpCtx.AddStep("help_info").
	SetText("Here's how to use the system...").
	SetValidContexts([]string{"main"}) // Can return to main

// Settings context
settings := contexts.AddContext("settings")
settings.AddStep("settings_menu").
	SetText("Choose a setting to modify...").
	SetValidContexts([]string{"main"}) // Can return to main
```

## Function Restrictions

Control which AI tools/functions are available in each step for enhanced security and user experience.

### Function Restriction Levels

```go
// No restrictions (default) - all agent functions available
step // Don't call SetFunctions()

// Allow specific functions only
step.SetFunctions([]string{"datetime", "math", "web_search"})

// Block all functions
step.SetFunctions("none")
```

### Security-Focused Example

```go
package main

import "github.com/signalwire/signalwire-go/pkg/agent"

func main() {
	a := agent.NewAgentBase(
		agent.WithName("Banking Assistant"),
		agent.WithRoute("/banking"),
	)

	// Add potentially sensitive functions
	a.AddSkill("web_search", map[string]any{"api_key": "key", "search_engine_id": "id"})
	a.AddSkill("datetime", nil)

	contexts := a.DefineContexts()

	// Public context - full access
	public := contexts.AddContext("public")
	public.AddStep("welcome").
		SetText("Welcome to banking support. Are you an existing customer?").
		SetFunctions([]string{"datetime", "web_search"}). // Safe functions only
		SetValidContexts([]string{"authenticated", "public"})

	// Authenticated context - restricted for security
	auth := contexts.AddContext("authenticated")
	auth.AddStep("account_access").
		SetText("I can help with your account. What do you need assistance with?").
		SetFunctions("none").                  // No external functions for account data
		SetValidContexts([]string{"public"}) // Can log out

	_ = a
}
```

### Function Access Patterns

```go
// Progressive function access based on trust level
contexts := a.DefineContexts()

// Low trust - limited functions
public := contexts.AddContext("public")
public.AddStep("initial_contact").
	SetFunctions([]string{"datetime"}) // Only safe functions

// Medium trust - more functions
verified := contexts.AddContext("verified")
verified.AddStep("verified_user").
	SetFunctions([]string{"datetime", "web_search"}) // Add search capability

// High trust - full access
authenticated := contexts.AddContext("authenticated")
authenticated.AddStep("full_access")
// No SetFunctions() call = all functions available
```

## Step Modes

Steps can operate in two modes:

- **Normal Mode**: The step's text is injected as instructions. The AI follows those instructions, and the step completes based on criteria you define or by navigating to the next step.
- **Gather Info Mode**: The step collects structured information from the caller one question at a time, with zero tool artifacts in the LLM conversation history. Once all questions are answered, the step either auto-advances or returns to normal mode.

### Normal Mode

In normal mode, the step's text is injected as a system message with this structure:

```
[context prompt if any]

## Instructions to complete the Current Step
[your step text]

Do not mention to the user that you are following steps, or the names of the steps.
Do not ask the user any questions not explicitly related to these instructions.
Do not end the conversation when this step is complete.
[step criteria if any]
```

The step text supports `${variable}` expansion from `global_data` and prompt variables.

Step criteria tell the AI when a step is done. The AI evaluates the criteria and calls `next_step` when they're met:

```go
ctx.AddStep("verify").
	SetText("Verify the caller's identity.").
	SetStepCriteria(
		"The caller has provided their account number " +
			"AND confirmed their date of birth.",
	).
	SetValidSteps([]string{"handle_request"})
```

### Gather Info Mode

When an AI agent needs to collect structured information (name, address, account number, etc.), the traditional approach uses SWAIG functions -- the AI calls a function for each piece of data, which creates `tool_call` and `tool_result` entries in the conversation history. These tool artifacts confuse some models (especially reasoning models at low effort settings), waste tokens, and can cause the model to lose track of where it is in the collection flow.

Gather info mode solves this by using **dynamic step instruction re-injection**. Questions are presented one at a time by swapping out the system instruction, and answers are recorded via an internal function that routes through the system-log path -- producing **zero** tool_call/tool_result entries in the LLM-visible conversation history.

#### How It Works Internally

1. **Step entry**: When the AI enters a step with `gather_info`, the system switches to gather questioning mode.
2. **Preamble injection** (first question only): If the gather has a `prompt`, it's injected as a **persistent** system message for the entire gather sequence.
3. **Question injection**: A minimal system instruction is injected as a **clearable** message containing the question text, type hint, confirmation instructions, and any per-question prompt text.
4. **Tool lockdown**: During gather mode, **all normal functions are hidden** -- only `gather_submit` (an internal function) and any per-question `functions` are visible.
5. **Answer submission**: When the AI calls `gather_submit`, the answer is written to `global_data` and the next question's instruction is re-injected. The `gather_submit` call routes through the system-log path, so the LLM never sees tool_call/tool_result for it.
6. **Completion**: When all questions are answered, either:
   - The step auto-advances to the next sequential step (`completion_action="next_step"`)
   - The step jumps to a specific named step (`completion_action="step_name"`)
   - The step returns to normal mode with the regular step text, plus a note that gathered data is available (when `completion_action` is None)

Here's what the LLM conversation history looks like during gather mode:

```
[system] You are a travel assistant. You need to collect some details.    <- persistent preamble
[system] Ask the user: "What is your first name?"                        <- clearable, changes per question
         When you have the answer, call the gather_submit function.
         Do not ask the user any other questions.

[assistant] Hi there! I'm your travel assistant. What's your first name?
[user] Tony.
                                                        <- gather_submit recorded via system-log (invisible)
[system] Ask the user: "What is your last name?"        <- previous question instruction replaced
         ...

[assistant] Great, Tony! And your last name?
[user] Smith.
```

No tool_call/tool_result entries anywhere. Clean conversation history.

#### Basic Gather Example

```go
ctx.AddStep("collect_info").
	SetText("Help the caller with their request.").
	SetGatherInfo("caller_info", "", ""). // outputKey, completionAction, prompt
	AddGatherQuestion("first_name", "What is your first name?").
	AddGatherQuestion("last_name", "What is your last name?").
	AddGatherQuestion("email", "What is your email address?")
```

This collects three pieces of information, stores them under `caller_info` in global_data, then returns to normal step mode with the step text "Help the caller with their request."

#### The Gather Prompt (Preamble)

The gather `prompt` is injected once as a persistent message when the first question begins:

```go
ctx.AddStep("collect_profile").
	SetText("Use the profile to recommend products.").
	SetGatherInfo(
		"profile", // outputKey
		"",        // completionAction
		"Welcome the caller and introduce yourself as a product specialist. "+
			"Explain that you need to ask a few quick questions to find the "+
			"best products for them. Be friendly and conversational.", // prompt
	).
	AddGatherQuestion("name", "What is your name?").
	AddGatherQuestion("budget", "What is your budget?", contexts.WithType("number"))
```

Without a gather `prompt`, the AI jumps straight into asking the first question with no introduction.

#### Question Types

Each question has a `type` that controls the JSON schema of the `answer` parameter in `gather_submit`:

```go
// String (default) - free text
AddGatherQuestion("name", "What is your name?", contexts.WithType("string"))

// Integer - whole numbers
AddGatherQuestion("age", "How old are you?", contexts.WithType("integer"))

// Number - decimal values
AddGatherQuestion("budget", "What is your budget in dollars?", contexts.WithType("number"))

// Boolean - yes/no questions
AddGatherQuestion("has_passport", "Do you have a valid passport?", contexts.WithType("boolean"))
```

#### Confirmation Flow

When `confirm=True`, the AI must read the answer back to the caller and get explicit confirmation before submitting:

```go
AddGatherQuestion(
	"last_name",
	"What is your last name?",
	contexts.WithType("string"),
	contexts.WithConfirm(true),
)
```

How it works:

1. The question instruction includes: "You MUST confirm the answer with the user before submitting."
2. The `gather_submit` function schema includes a required `confirmed_by_user` enum parameter.
3. If the AI calls `gather_submit` with `confirmed_by_user` set to `"false"`, the function rejects the submission and tells the AI to confirm with the user first.
4. The AI must read back the answer, get the user's "yes", then call `gather_submit` again with `confirmed_by_user: "true"`.

#### Per-Question Instructions and Functions

Each question can have additional instructions and specific functions made available:

```go
AddGatherQuestion(
	"home_airport",
	"What is your home airport or nearest major city for departure?",
	contexts.WithType("string"),
	contexts.WithConfirm(true),
	contexts.WithPrompt("Use the resolve_airport function to validate the airport code "+
		"before submitting. If the airport is ambiguous, clarify with the user."),
	contexts.WithFunctions([]string{"resolve_airport"}),
)
```

The `resolve_airport` function must already be registered on the agent. The `functions` array activates those functions for this question only, alongside `gather_submit`. When the next question begins, they're deactivated again.

#### Output Storage

Answers are stored in `global_data`, which is available in prompt variable expansion via `${key}`:

```go
// Store under a namespace
SetGatherInfo("profile", "", "")
// Results in: global_data.profile.first_name, global_data.profile.last_name, etc.
// Accessible in prompts as: ${profile}

// Store at top level (empty output key)
SetGatherInfo("", "", "")
// Results in: global_data.first_name, global_data.last_name, etc.
```

After gathering, `global_data` is refreshed so subsequent step prompts can reference the collected values:

```go
ctx.AddStep("plan_trip").
	SetText(
		"The caller's travel profile is: ${profile}. " +
			"Use their name, budget, and preferences to suggest destinations.",
	)
```

#### Auto-Advancing After Gather

With `completion_action`, the step automatically advances when the last question is answered. You can advance to the next sequential step or jump to a specific named step:

```go
// Advance to the next sequential step
ctx.AddStep("collect_profile").
	SetText("Collect the caller's profile.").
	SetGatherInfo(
		"profile",   // outputKey
		"next_step", // completionAction
		"Welcome the caller. You need to collect a few details.", // prompt
	).
	AddGatherQuestion("name", "What is your name?").
	AddGatherQuestion("email", "What is your email?")

// This step runs immediately after the last question is answered
ctx.AddStep("process").
	SetText("You have the caller's profile in ${profile}. Help them with their request.")
```

You can also jump to a specific step by name:

```go
ctx.AddStep("collect_info").
	SetText("Collect caller info.").
	SetGatherInfo(
		"info",   // outputKey
		"review", // completionAction — jump directly to "review" step
		"",       // prompt
	).
	AddGatherQuestion("name", "What is your name?").
	AddGatherQuestion("issue", "What is your issue?")

ctx.AddStep("other_step").
	SetText("This step is skipped when coming from collect_info.")

ctx.AddStep("review").
	SetText("Review the collected info in ${info} and help the caller.")
```

> **Note**: The target step is validated at build time (via `ContextBuilder.Validate()`). Using `"next_step"` on the last step in a context, or naming a step that doesn't exist, returns a validation error.

#### Combining Gather with Normal Step Mode

Without `completion_action` (or when set to None), the step returns to normal mode after all questions are answered:

```go
ctx.AddStep("intake").
	SetText(
		"Review the caller's information in ${intake_data}. " +
			"Confirm everything looks correct, then proceed to scheduling.",
	).
	SetGatherInfo("intake_data", "", ""). // outputKey, completionAction, prompt
	AddGatherQuestion("name", "What is your name?").
	AddGatherQuestion("dob", "What is your date of birth?").
	AddGatherQuestion("reason", "What is the reason for your visit?").
	SetValidSteps([]string{"schedule"})
```

Flow:
1. Gather mode: Questions are asked one at a time
2. All questions answered -> step switches to normal mode
3. Step text is injected with `valid_steps` and `step_criteria` restored
4. The AI follows the normal step instructions using the gathered data
5. Navigation to `schedule` becomes available

#### Gather Info API Reference

**`SetGatherInfo(outputKey, completionAction, prompt string)` Parameters** (positional):

| Parameter | Type | Empty value | Description |
|-----------|------|-------------|-------------|
| `outputKey` | string | `""` | Key in global_data to store answers under. If `""`, answers stored at top level. |
| `completionAction` | string | `""` | Where to go when all questions are answered: `"next_step"` to advance sequentially, or a specific step name (e.g. `"process_results"`) to jump to that step. If `""`, returns to normal step mode. The target is validated — `"next_step"` requires a following step, and named steps must exist in the context. |
| `prompt` | string | `""` | Preamble text injected once as a persistent message when entering the gather step. |

**`AddGatherQuestion(key, question string, opts ...GatherQuestionOption)` Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `key` | string | required | Key name for storing the answer in global_data |
| `question` | string | required | The question text presented to the AI |
| `WithType(t)` | option | `"string"` | JSON schema type: `"string"`, `"integer"`, `"number"`, `"boolean"` |
| `WithConfirm(b)` | option | `false` | If true, AI must confirm answer with user before submitting |
| `WithPrompt(p)` | option | `""` | Additional instruction text for this question |
| `WithFunctions(f)` | option | `nil` | Function names to make visible for this question only |

## Real-World Examples

### Example 1: Technical Support Troubleshooting

```go
package main

import "github.com/signalwire/signalwire-go/pkg/agent"

func main() {
	a := agent.NewAgentBase(
		agent.WithName("Tech Support"),
		agent.WithRoute("/tech-support"),
	)

	// Add diagnostic tools
	a.AddSkill("web_search", map[string]any{"api_key": "key", "search_engine_id": "id"})
	a.AddSkill("datetime", nil)

	contexts := a.DefineContexts()

	// Initial triage
	triage := contexts.AddContext("triage")
	triage.AddStep("problem_identification").
		AddSection("Current Task", "Identify the type of technical issue").
		AddBullets("Information to Gather", []string{
			"Description of the specific problem",
			"When did the issue start occurring?",
			"What steps has the customer already tried?",
			"Rate the severity level (critical/high/medium/low)",
		}).
		SetStepCriteria("Issue type and severity determined").
		SetValidContexts([]string{"hardware", "software", "network"})

	// Hardware troubleshooting
	hardware := contexts.AddContext("hardware")
	hardware.AddStep("hardware_diagnosis").
		AddSection("Current Task", "Guide user through hardware diagnostics").
		AddSection("Available Tools", "Use web search to find hardware specifications and troubleshooting guides").
		SetFunctions([]string{"web_search"}). // Can search for hardware info
		SetStepCriteria("Hardware issue diagnosed").
		SetValidSteps([]string{"hardware_solution"})

	hardware.AddStep("hardware_solution").
		SetText("Based on the diagnosis, here's how to resolve the hardware issue...").
		SetStepCriteria("Solution provided and tested").
		SetValidContexts([]string{"triage"}) // Can start over if needed

	// Software troubleshooting
	software := contexts.AddContext("software")
	software.AddStep("software_diagnosis").
		AddSection("Current Task", "Diagnose software-related issues").
		AddSection("Available Tools", "Use web search for software updates and datetime to check for recent changes").
		SetFunctions([]string{"web_search", "datetime"}). // Can check for updates
		SetStepCriteria("Software issue identified").
		SetValidSteps([]string{"software_fix", "escalation"})

	software.AddStep("software_fix").
		SetText("Let's try these software troubleshooting steps...").
		SetStepCriteria("Fix attempted and result confirmed").
		SetValidSteps([]string{"escalation", "resolution"})

	software.AddStep("escalation").
		SetText("I'll escalate this to our specialist team.").
		SetFunctions("none"). // No tools needed for escalation
		SetStepCriteria("Escalation ticket created")

	software.AddStep("resolution").
		SetText("Great! The issue has been resolved.").
		SetStepCriteria("Customer confirms resolution").
		SetValidContexts([]string{"triage"})

	// Network troubleshooting
	network := contexts.AddContext("network")
	network.AddStep("network_diagnosis").
		AddSection("Current Task", "Diagnose network and connectivity issues").
		AddSection("Available Tools", "Use web search to check service status and datetime for outage windows").
		SetFunctions([]string{"web_search", "datetime"}). // Check service status
		SetStepCriteria("Network issue diagnosed").
		SetValidSteps([]string{"network_fix"})

	network.AddStep("network_fix").
		SetText("Let's resolve your connectivity issue with these steps...").
		SetStepCriteria("Network connectivity restored").
		SetValidContexts([]string{"triage"})

	if err := a.Run(); err != nil {
		panic(err)
	}
}
```

### Example 2: Multi-Step Application Process

```go
package main

import "github.com/signalwire/signalwire-go/pkg/agent"

func main() {
	a := agent.NewAgentBase(
		agent.WithName("Loan Application"),
		agent.WithRoute("/loan-app"),
	)

	// Add verification tools
	a.AddSkill("datetime", nil) // For date validation

	contexts := a.DefineContexts()

	// Single workflow context
	application := contexts.AddContext("default")

	// Step 1: Introduction and eligibility
	application.AddStep("introduction").
		AddSection("Current Task", "Guide customers through the loan application process").
		AddBullets("Information to Provide", []string{
			"Explain the process clearly",
			"Outline what information will be needed",
			"Set expectations for timeline and next steps",
		}).
		SetStepCriteria("Customer understands process and wants to continue").
		SetValidSteps([]string{"personal_info"})

	// Step 2: Personal information
	application.AddStep("personal_info").
		AddSection("Instructions", "Collect personal information").
		AddBullets("Required Fields", []string{
			"Full legal name",
			"Date of birth",
			"Social Security Number",
			"Phone number and email",
		}).
		SetFunctions([]string{"datetime"}). // Can validate dates
		SetStepCriteria("All personal information collected and verified").
		SetValidSteps([]string{"employment_info", "personal_info"}) // Can review/edit

	// Step 3: Employment information
	application.AddStep("employment_info").
		SetText("Now I need information about your employment and income.").
		SetStepCriteria("Employment and income information complete").
		SetValidSteps([]string{"financial_info", "personal_info"}) // Can go back

	// Step 4: Financial information
	application.AddStep("financial_info").
		SetText("Let's review your financial situation including assets and debts.").
		SetStepCriteria("Financial information complete").
		SetValidSteps([]string{"review", "employment_info"}) // Can go back

	// Step 5: Review all information
	application.AddStep("review").
		AddSection("Instructions", "Review all collected information").
		AddBullets("Checklist", []string{
			"Confirm personal details",
			"Verify employment information",
			"Review financial data",
			"Ensure accuracy before submission",
		}).
		SetStepCriteria("Customer has reviewed and confirmed all information").
		SetValidSteps([]string{"submit", "personal_info", "employment_info", "financial_info"})

	// Step 6: Submission
	application.AddStep("submit").
		SetText("Thank you! Your loan application has been submitted successfully. You'll receive a decision within 2-3 business days.").
		SetFunctions("none"). // No tools needed for final message
		SetStepCriteria("Application submitted and confirmation provided")
		// No valid_steps = end of process

	if err := a.Run(); err != nil {
		panic(err)
	}
}
```

### Example 3: E-commerce Customer Service

```go
package main

import "github.com/signalwire/signalwire-go/pkg/agent"

func main() {
	a := agent.NewAgentBase(
		agent.WithName("E-commerce Support"),
		agent.WithRoute("/ecommerce"),
	)

	// Add tools for order management
	a.AddSkill("web_search", map[string]any{"api_key": "key", "search_engine_id": "id"})
	a.AddSkill("datetime", nil)

	contexts := a.DefineContexts()

	// Main service menu
	main := contexts.AddContext("main")
	main.AddStep("service_menu").
		AddSection("Current Task", "Help customers with their orders and questions").
		AddBullets("Service Areas Available", []string{
			"Order status, modifications, and tracking",
			"Returns and refunds",
			"Product information and specifications",
			"Account-related questions",
		}).
		SetStepCriteria("Customer's need has been identified").
		SetValidContexts([]string{"orders", "returns", "products", "account"})

	// Order management context
	orders := contexts.AddContext("orders")
	orders.AddStep("order_assistance").
		AddSection("Current Task", "Help with order status, modifications, and tracking").
		AddSection("Available Tools", "Use datetime to check delivery dates and processing times").
		SetFunctions([]string{"datetime"}). // Can check delivery dates
		SetStepCriteria("Order issue resolved or escalated").
		SetValidContexts([]string{"main"})

	// Returns and refunds context
	returns := contexts.AddContext("returns")
	returns.AddStep("return_process").
		AddSection("Current Task", "Guide customers through return process").
		AddBullets("Return Process Steps", []string{
			"Verify return eligibility",
			"Explain return policy",
			"Provide return instructions",
			"Process refund if applicable",
		}).
		SetFunctions("none"). // Sensitive financial operations
		SetStepCriteria("Return request processed").
		SetValidContexts([]string{"main"})

	// Product information context
	products := contexts.AddContext("products")
	products.AddStep("product_help").
		AddSection("Current Task", "Help customers with product questions").
		AddSection("Available Tools", "Use web search to find detailed product information and specifications").
		SetFunctions([]string{"web_search"}). // Can search for product info
		SetStepCriteria("Product question answered").
		SetValidContexts([]string{"main"})

	// Account management context
	account := contexts.AddContext("account")
	account.AddStep("account_help").
		SetText("I can help with account-related questions. Please verify your identity first.").
		SetFunctions("none"). // Security-sensitive context
		SetStepCriteria("Account issue resolved").
		SetValidContexts([]string{"main"})

	if err := a.Run(); err != nil {
		panic(err)
	}
}
```

## Best Practices

### 1. Clear Step Naming

Use descriptive step names that indicate purpose:

```go
// Good
AddStep("collect_shipping_address")
AddStep("verify_payment_method")
AddStep("confirm_order_details")

// Avoid
AddStep("step1")
AddStep("next")
AddStep("continue")
```

### 2. Meaningful Completion Criteria

Define clear, testable completion criteria:

```go
// Good - specific and measurable
SetStepCriteria("User has provided valid email address and confirmed subscription preferences")
SetStepCriteria("All required fields completed and payment method verified")

// Avoid - vague or subjective
SetStepCriteria("User is ready")
SetStepCriteria("Everything is good")
```

### 3. Logical Navigation Flow

Design intuitive navigation that matches user expectations:

```go
// Allow users to go back and review
SetValidSteps([]string{"review_info", "edit_details", "confirm_submission"})

// Provide escape routes
SetValidContexts([]string{"main_menu", "help"})

// Consider dead ends carefully
SetValidSteps([]string{}) // Only if this is truly the end
```

### 4. Progressive Function Access

Restrict functions based on security and context needs:

```go
// Public areas - limited functions
publicStep.SetFunctions([]string{"datetime", "web_search"})

// Authenticated areas - more functions allowed
authStep.SetFunctions([]string{"datetime", "web_search", "user_profile"})

// Sensitive operations - minimal functions
billingStep.SetFunctions("none")
```

### 5. Context Organization

Organize contexts by functional area or user journey:

```go
// By functional area
contexts := []string{"triage", "technical_support", "billing", "account_management"}

// By user journey stage
contexts := []string{"onboarding", "verification", "configuration", "completion"}

// By security level
contexts := []string{"public", "authenticated", "admin"}
```

### 6. Error Handling and Recovery

Provide recovery paths for common issues:

```go
// Allow users to retry failed steps
SetValidSteps([]string{"retry_payment", "choose_different_method", "contact_support"})

// Provide help context access
SetValidContexts([]string{"help", "main"})

// Include validation steps
verificationCtx.AddStep("validation").
	SetStepCriteria("Data validation passed").
	SetValidSteps([]string{"proceed", "edit_data"})
```

### 7. Content Strategy

Choose the right content approach for each step:

```go
// Use SetText() for simple, direct instructions
step.SetText("Please provide your email address")

// Use POM sections for complex, structured content
step.AddSection("Role", "You are a technical specialist").
	AddSection("Context", "Customer is experiencing network issues").
	AddSection("Instructions", "Follow diagnostic protocol").
	AddBullets("Steps", []string{"Check connectivity", "Test speed", "Verify settings"})
```

## Troubleshooting

### Common Issues

#### 1. "Single context must be named 'default'"

**Error**: When using a single context with a name other than "default"

```go
// Wrong
context := contexts.AddContext("main") // Error!

// Correct
context := contexts.AddContext("default")
```

#### 2. "Cannot mix set_text with add_section"

**Error**: Using both direct text and POM sections in the same step

```go
// Wrong
step.SetText("Welcome!").
	AddSection("Role", "Assistant") // Error!

// Correct - choose one approach
step.SetText("Welcome! I'm your assistant.")
// OR
step.AddSection("Role", "Assistant").
	AddSection("Message", "Welcome!")
```

#### 3. Navigation Issues

**Problem**: Users getting stuck or unable to navigate

```go
// Check your navigation rules
step.SetValidSteps([]string{})    // Dead end - is this intended?
step.SetValidContexts([]string{}) // Trapped in context - is this intended?

// Add appropriate navigation
step.SetValidSteps([]string{"next_step", "previous_step"})
step.SetValidContexts([]string{"main", "help"})
```

#### 4. Function Access Problems

**Problem**: Functions not available when expected

```go
// Check function restrictions
step.SetFunctions("none")               // All functions blocked
step.SetFunctions([]string{"datetime"}) // Only datetime allowed

// Verify function names match your agent's functions
a.AddSkill("web_search", nil)             // Function name is "web_search"
step.SetFunctions([]string{"web_search"}) // Must match exactly
```

### Debugging Tips

#### 1. Trace Navigation Flow

Add logging to understand flow:

```go
func createStepWithLogging(ctx *contexts.Context, name string) *contexts.Step {
	step := ctx.AddStep(name)
	fmt.Printf("Created step: %s\n", name)
	return step
}
```

#### 2. Validate Navigation Rules

Check that all referenced steps/contexts exist:

```go
// Ensure referenced steps exist
SetValidSteps([]string{"review", "edit"}) // Both "review" and "edit" steps must exist

// Ensure referenced contexts exist
SetValidContexts([]string{"main", "help"}) // Both "main" and "help" contexts must exist
```

#### 3. Test Function Restrictions

Verify functions are properly restricted:

```go
// Test with all functions
// step // No SetFunctions() call

// Test with restrictions
step.SetFunctions([]string{"datetime"})

// Test with no functions
step.SetFunctions("none")
```

## Migration from POM

### Converting Traditional Prompts

**Before (Traditional POM):**
```go
package main

import "github.com/signalwire/signalwire-go/pkg/agent"

func main() {
	a := agent.NewAgentBase(
		agent.WithName("assistant"),
		agent.WithRoute("/assistant"),
	)

	a.PromptAddSection("Role", "You are a helpful assistant", nil)
	a.PromptAddSection("Instructions", "Help users with questions", nil)
	a.PromptAddSection("Guidelines", "", []string{
		"Be friendly",
		"Ask clarifying questions",
		"Provide accurate information",
	})

	_ = a
}
```

**After (Contexts and Steps):**
```go
package main

import "github.com/signalwire/signalwire-go/pkg/agent"

func main() {
	a := agent.NewAgentBase(
		agent.WithName("assistant"),
		agent.WithRoute("/assistant"),
	)

	contexts := a.DefineContexts()
	main := contexts.AddContext("default")

	main.AddStep("assistance").
		AddSection("Role", "You are a helpful assistant").
		AddSection("Instructions", "Help users with questions").
		AddBullets("Guidelines", []string{
			"Be friendly",
			"Ask clarifying questions",
			"Provide accurate information",
		}).
		SetStepCriteria("User's question has been answered")

	_ = main
}
```

### Hybrid Approach

You can use both traditional prompts and contexts in the same agent:

```go
package main

import "github.com/signalwire/signalwire-go/pkg/agent"

func main() {
	a := agent.NewAgentBase(
		agent.WithName("hybrid"),
		agent.WithRoute("/hybrid"),
	)

	// Traditional prompt sections (from skills, global settings, etc.)
	// These will coexist with contexts

	// Define contexts for structured workflows
	contexts := a.DefineContexts()
	workflow := contexts.AddContext("default")

	workflow.AddStep("structured_process").
		SetText("Following the structured workflow...").
		SetStepCriteria("Workflow complete")

	_ = a
}
```

### Migration Strategy

1. **Start Simple**: Convert one workflow at a time
2. **Preserve Existing**: Keep traditional prompts for simple interactions
3. **Add Structure**: Use contexts for complex, multi-step processes
4. **Test Thoroughly**: Verify navigation and function access work as expected
5. **Iterate**: Refine step criteria and navigation based on testing

---

## Conclusion

The Contexts and Steps system provides structured workflow control for building sophisticated AI agents. By combining structured navigation, function restrictions, and clear completion criteria, you can create predictable, user-friendly agent experiences that guide users through complex processes while maintaining security and control.

Start with simple single-context workflows and gradually build more complex multi-context systems as your requirements grow. The system is designed to be flexible and scalable, supporting both simple linear workflows and complex branching conversation trees.

### Dynamic Context Switching

To switch contexts dynamically during a conversation, use a `swaig.FunctionResult` (from `swaig.NewFunctionResult`) with the `SwmlChangeContext()` method:

```go
package main

import (
	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("multi-context"),
		agent.WithRoute("/multi"),
	)

	// Define contexts using the ContextBuilder pattern
	contexts := a.DefineContexts()

	// Sales context
	sales := contexts.AddContext("sales")
	sales.AddSection("Role", "You are a helpful sales representative.")
	sales.AddStep("greeting").SetText("Welcome customers and understand their needs.")

	// Support context
	support := contexts.AddContext("support")
	support.AddSection("Role", "You are a technical support specialist.")
	support.AddStep("diagnose").SetText("Help diagnose and resolve technical issues.")

	// Tool: transfer to support — uses SwmlChangeContext to switch contexts
	a.DefineTool(agent.ToolDefinition{
		Name:        "transfer_to_support",
		Description: "Transfer the customer to technical support",
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("Transferring you to technical support...").
				SwmlChangeContext("support")
		},
	})

	// Tool: transfer to sales
	a.DefineTool(agent.ToolDefinition{
		Name:        "transfer_to_sales",
		Description: "Transfer the customer to sales",
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("Transferring you to sales...").
				SwmlChangeContext("sales")
		},
	})

	if err := a.Run(); err != nil {
		panic(err)
	}
}
```

For a complete example of multi-context agents with different personas, see `examples/contexts_demo/main.go`.

---

### Example 4: Travel Profile Agent (Gather Info Mode)

Collects a travel profile with typed questions and confirmation, then recommends destinations:

```go
package main

import (
	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/contexts"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("Travel Agent"),
		agent.WithRoute("/travel"),
	)

	a.PromptAddSection("Role", "You are a friendly travel booking assistant.", nil)

	cb := a.DefineContexts()
	ctx := cb.AddContext("default")

	// Step 1: Collect profile (gather mode, auto-advance)
	ctx.AddStep("collect_profile").
		SetText("Collect the caller's travel profile.").
		SetGatherInfo(
			"profile",   // outputKey
			"next_step", // completionAction
			"Welcome the caller and introduce yourself as a travel "+
				"booking assistant. You need to collect a few details "+
				"to build their travel profile. Be warm and conversational.", // prompt
		).
		AddGatherQuestion("first_name", "What is your first name?").
		AddGatherQuestion("last_name", "What is your last name?", contexts.WithConfirm(true)).
		AddGatherQuestion("party_size", "How many people are traveling?", contexts.WithType("integer")).
		AddGatherQuestion("budget_per_person", "What is your budget per person?", contexts.WithType("number")).
		AddGatherQuestion("has_passport", "Do you have a valid passport?", contexts.WithType("boolean")).
		AddGatherQuestion("home_airport", "What is your home airport?", contexts.WithConfirm(true))

	// Step 2: Recommend destinations (normal mode)
	ctx.AddStep("plan_trip").
		SetText(
			"You now have the caller's travel profile in ${profile}. " +
				"Use their name, party size, budget, passport status, and " +
				"home airport to suggest three vacation destinations. " +
				"If they don't have a passport, only suggest domestic destinations.",
		)

	a.AddLanguage(map[string]any{"name": "English", "code": "en-US", "voice": "rime.spore"})

	if err := a.Run(); err != nil {
		panic(err)
	}
}
```

### Example 5: Support Ticket Agent (Gather + Triage)

Gathers issue details, then routes to the right team using normal mode navigation:

```go
package main

import (
	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/contexts"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("Support Agent"),
		agent.WithRoute("/support"),
	)

	a.PromptAddSection("Role", "You are a technical support agent.", nil)

	cb := a.DefineContexts()
	ctx := cb.AddContext("default")

	// Collect ticket info, then return to normal mode for triage
	ctx.AddStep("intake").
		SetText(
			"You have the caller's issue details in ${ticket}. " +
				"Based on the category and description, route them to " +
				"the appropriate team.",
		).
		SetGatherInfo(
			"ticket", // outputKey
			"",       // completionAction (empty = return to normal mode)
			"Thank the caller for contacting support. "+
				"You need to collect some details about their issue.", // prompt
		).
		AddGatherQuestion("name", "What is your name?").
		AddGatherQuestion("account_id", "What is your account ID?", contexts.WithConfirm(true)).
		AddGatherQuestion("category", "Is this about billing, a technical issue, or something else?").
		AddGatherQuestion("description", "Please describe the issue in detail.").
		SetValidSteps([]string{"billing_support", "tech_support", "general_support"})

	ctx.AddStep("billing_support").
		SetText("Help the caller with their billing issue. Details: ${ticket}.")

	ctx.AddStep("tech_support").
		SetText("Help the caller with their technical issue. Details: ${ticket}.").
		SetFunctions([]string{"run_diagnostics", "check_service_status"})

	ctx.AddStep("general_support").
		SetText("Help the caller with their general inquiry. Details: ${ticket}.")

	a.AddLanguage(map[string]any{"name": "English", "code": "en-US", "voice": "rime.spore"})

	if err := a.Run(); err != nil {
		panic(err)
	}
}
```

Note: This example uses gather **without** `completion_action`. After all questions are answered, the step returns to normal mode with `valid_steps` restored. The AI uses the gathered data to decide which support step to route to.

## Related Documentation

- **[API Reference](api_reference.md)** - Complete AgentBase class reference
- **[SWAIG Reference](swaig_reference.md)** - All available result methods including `SwmlChangeContext()` and `SwmlChangeStep()`
- **[Agent Guide](agent_guide.md)** - General agent development guide
- **[DataMap Guide](datamap_guide.md)** - Serverless function integration

### Example Files

- `examples/contexts_demo/main.go` - Multi-context agent with sales and support workflows
- `examples/gather_info/main.go` - Structured data collection using `SetGatherInfo()` and `AddGatherQuestion()`
- `examples/gather_per_question_functions_demo/main.go` - Per-question function activation during gather
- `examples/prefab_survey/main.go` - Survey workflow with steps
- `examples/prefab_info_gatherer/main.go` - Information gathering workflow