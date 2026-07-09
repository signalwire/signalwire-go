# SignalWire AI Agent Guide

## Table of Contents
- [Introduction](#introduction)
- [Architecture Overview](#architecture-overview)
- [Creating an Agent](#creating-an-agent)
- [Prompt Building](#prompt-building)
- [SWAIG Functions (SignalWire AI Gateway)](#swaig-functions)
- [Skills System](#skills-system)
- [Multilingual Support](#multilingual-support)
- [Agent Configuration](#agent-configuration)
- [Dynamic Agent Configuration](#dynamic-agent-configuration)
  - [Overview](#overview)
  - [Setting Up Dynamic Configuration](#setting-up-dynamic-configuration)
  - [Dynamic Configuration Methods](#dynamic-configuration-methods)
  - [Request Data Access](#request-data-access)
  - [Configuration Examples](#configuration-examples)
  - [Use Cases](#use-cases)
  - [Migration Guide](#migration-guide)
  - [Best Practices](#best-practices)
- [Advanced Features](#advanced-features)
  - [State Management](#state-management)
  - [SIP Routing](#sip-routing)
  - [Custom Routing](#custom-routing)
- [Prefab Agents](#prefab-agents)
- [API Reference](#api-reference)
- [Examples](#examples)

## Introduction

The `AgentBase` struct provides the foundation for creating AI-powered agents using the SignalWire AI Agent SDK for Go. It builds on the `SWMLService` layer, inheriting all its SWML (SignalWire Markup Language) document creation and serving capabilities, while adding AI-specific functionality. SWML is the JSON document format that tells the SignalWire platform how an agent should behave during a call.

Key features of `AgentBase` include:

- Structured prompt building with POM (Prompt Object Model)
- SWAIG (SignalWire AI Gateway) function definitions -- SWAIG is the platform's AI tool-calling system with native access to the media stack
- Multilingual support
- Agent configuration (hint handling, pronunciation rules, etc.)
- State management for conversations

This guide explains how to create and customize your own AI agents, with examples based on the SDK's sample implementations.

Add the SDK to your module with:

```bash
go get github.com/signalwire/signalwire-go
```

## Architecture Overview

The Agent SDK architecture consists of several layers:

1. **SWMLService**: The base layer for SWML document creation and serving
2. **AgentBase**: Composes SWMLService with AI agent functionality
3. **Your Agent**: Your specific agent, configured via functional options and fluent methods

Here's how these components relate to each other:

```text
┌─────────────┐
│ Your Agent  │ (Configures AgentBase with your specific functionality)
└─────▲───────┘
      │
┌─────┴───────┐
│  AgentBase  │ (Adds AI functionality to SWMLService)
└─────▲───────┘
      │
┌─────┴───────┐
│ SWMLService │ (Provides SWML document creation and web service)
└─────────────┘
```

## Creating an Agent

Unlike the Python SDK (which uses subclassing), the Go SDK composes an agent using functional options and fluent configuration methods. Create an agent with `agent.NewAgentBase` and configure it directly:

```go
package main

import (
	"github.com/signalwire/signalwire-go/pkg/agent"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("my-agent"),
		agent.WithRoute("/agent"),
		agent.WithHost("0.0.0.0"),
		agent.WithPort(3000),
		agent.WithUsePom(true), // Enable Prompt Object Model
	)

	// Define agent personality and behavior
	a.PromptAddSection("Personality", "You are a helpful and friendly assistant.", nil)
	a.PromptAddSection("Goal", "Help users with their questions and tasks.", nil)
	a.PromptAddSection("Instructions", "", []string{
		"Answer questions clearly and concisely",
		"If you don't know, say so",
		"Use the provided tools when appropriate",
	})

	// Add a post-prompt for summary
	a.SetPostPrompt("Please summarize the key points of this conversation.")

	if err := a.Run(); err != nil {
		panic(err)
	}
}
```

`PromptAddSection(title, body, bullets, opts...)` takes the section title, a body string (empty string for none), and a bullet slice (`nil` for none).

## Running Your Agent

The SignalWire AI Agent SDK provides a `Run()` method that automatically detects the execution environment and configures the agent appropriately. This method works across all deployment modes:

### Deployment with `Run()`

<!-- snippet-setup -->
```go
import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// Shared context the fragments below assume (established in prose above):
// `a`/`ep` are agent instances; newMyAgent is your agent factory. The remaining
// vars/closures stand in for your own helpers referenced illustratively below.
var a = agent.NewAgentBase()
var ep = agent.NewAgentBase()
var newMyAgent = func() *agent.AgentBase { return agent.NewAgentBase() }
var err error

var tierConfigs = map[string]map[string]any{}
var isValidCustomer = func(id string) bool { return true }
var getCustomerTier = func(id string) string { return "standard" }
var applyCustomConfig = func(qp map[string]string, ep *agent.AgentBase) error { return nil }
var applyDefaultConfig = func(ep *agent.AgentBase) {}
var alertOpsTeam = func(event map[string]any) {}

// sessions stands in for your external session store; loadUserPreferences for
// your own lookup. Referenced illustratively by the lifecycle-hook examples.
var sessions = struct {
	Update func(id string, data map[string]any)
	Get    func(id string) (map[string]any, bool)
	Delete func(id string)
}{
	Update: func(id string, data map[string]any) {},
	Get:    func(id string) (map[string]any, bool) { return nil, false },
	Delete: func(id string) {},
}
var loadUserPreferences = func(callerID string) map[string]any { return map[string]any{} }

// Additional agent instances referenced by the multi-agent routing examples.
var registrationAgent = agent.NewAgentBase()
var supportAgent = agent.NewAgentBase()
var sendToAnalytics = func(data map[string]any) {}

var (
	_ = a
	_ = ep
	_ = newMyAgent
	_ = err
	_ = skills.SkillWebSearch
	_ = swaig.NewFunctionResult
	_ = tierConfigs
	_ = isValidCustomer
	_ = getCustomerTier
	_ = applyCustomConfig
	_ = applyDefaultConfig
	_ = alertOpsTeam
	_ = sessions
	_ = loadUserPreferences
	_ = registrationAgent
	_ = supportAgent
	_ = sendToAnalytics
	_ = log.Printf
	_ = http.MethodGet
	_ = os.Getenv
	_ = strconv.Atoi
	_ = strings.ToLower
)
```

```go
import "fmt"

func main() {
	a := newMyAgent()

	fmt.Println("Starting agent server...")
	fmt.Println("Note: Works in any deployment mode (server/CGI/Lambda)")
	if err := a.Run(); err != nil { // Auto-detects environment
		fmt.Printf("Agent error: %v\n", err)
	}
}
```

The `Run()` method automatically detects and configures for:

- **HTTP Server**: When run directly, starts an HTTP server
- **CGI**: When CGI environment variables are detected, operates in CGI mode
- **AWS Lambda**: When Lambda environment is detected, configures for serverless execution

To force a specific mode, use `RunWithMode(mode)`; `DetectRunMode()` returns the mode that `Run()` would auto-select.

### Deployment Modes

#### HTTP Server Mode
When run directly (e.g., `go run ./cmd/my_agent`), the agent starts an HTTP server:

```go
// Automatically starts HTTP server when run directly
a.Run()
```

#### CGI Mode
When CGI environment variables are present, operates in CGI mode with clean HTTP output:

```go
// Same code - automatically detects CGI environment
a.Run()
```

#### AWS Lambda Mode
When AWS Lambda environment is detected, configures for serverless execution:

```go
// Same code - automatically detects Lambda environment
a.Run()
```

### Environment Detection

The SDK automatically detects the execution environment:

| Environment | Detection Method | Behavior |
|-------------|------------------|----------|
| **HTTP Server** | Default when no serverless environment detected | Starts HTTP server on specified host/port |
| **CGI** | `GATEWAY_INTERFACE` environment variable present | Processes single CGI request and exits |
| **AWS Lambda** | `AWS_LAMBDA_FUNCTION_NAME` environment variable | Handles Lambda event/context |
| **Google Cloud** | `FUNCTION_NAME` or `K_SERVICE` variables | Processes Cloud Function request |
| **Azure Functions** | `AZURE_FUNCTIONS_*` variables | Handles Azure Function request |

### Logging Configuration

The SDK includes a central logging system that automatically configures based on the deployment environment:

```text
# Logging is automatically configured based on environment.
# No manual setup required in most cases.

# Optional: Override logging mode via environment variable
# SIGNALWIRE_LOG_MODE=off      # Disable all logging
# SIGNALWIRE_LOG_MODE=stderr   # Log to stderr
# SIGNALWIRE_LOG_MODE=default  # Use default logging
# SIGNALWIRE_LOG_MODE=auto     # Auto-detect (default)
```

The logging system automatically:
- **CGI Mode**: Sets logging to 'off' to avoid interfering with HTTP headers
- **Lambda Mode**: Configures appropriate logging for serverless environment
- **Server Mode**: Uses structured logging with timestamps and levels
- **Debug Mode**: Enhanced logging when debug flags are set

You can also suppress structured logs at construction with `agent.WithSuppressLogs(true)`.

## Prompt Building

There are several ways to build prompts for your agent:

### 1. Using Prompt Sections (POM)

The Prompt Object Model (POM) provides a structured way to build prompts:

```go
// Add a section with just body text
a.PromptAddSection("Personality", "You are a friendly assistant.", nil)

// Add a section with bullet points
a.PromptAddSection("Instructions", "", []string{
	"Answer questions clearly",
	"Be helpful and polite",
	"Use functions when appropriate",
})

// Add a section with both body and bullets
a.PromptAddSection("Context",
	"The user is calling about technical support.",
	[]string{
		"They may need help with their account",
		"Check for existing tickets",
	})
```

To append to an existing section later, use `PromptAddToSection(title, body, opts...)` with the `agent.WithBullet` / `agent.WithBullets` options, or `PromptAddSubsection(parentTitle, title, body, bullets)` to nest a subsection.

### 2. Using Raw Text Prompts

For simpler agents, you can set the prompt directly as text:

```go
a.SetPromptText(`
You are a helpful assistant. Your goal is to provide clear and concise information
to the user. Answer their questions to the best of your ability.
`)
```

### 3. Setting a Post-Prompt

The post-prompt is sent to the AI after the conversation for summary or analysis:

```go
a.SetPostPrompt(`
Analyze the conversation and extract:
1. Main topics discussed
2. Action items or follow-ups needed
3. Whether the user's questions were answered satisfactorily
`)
```

## SWAIG Functions

SWAIG (SignalWire AI Gateway) functions allow the AI agent to perform actions and access external systems during a call. The AI decides when to call a function based on the conversation; SWAIG handles invocation, parameter passing, and delivering the result back to the AI. There are two types of SWAIG functions you can define:

### SWAIG functions ARE LLM tools — descriptions matter

Before writing your first SWAIG function, internalize this: a SWAIG function is **exactly the same concept** as a "tool" in native OpenAI / Anthropic tool calling. There is no separate "SWAIG layer" between your function and the model. Each SWAIG function is rendered into the OpenAI tool schema format on every turn:

```json
{
  "type": "function",
  "function": {
    "name":        "your_function_name",
    "description": "your description text",
    "parameters":  { "your": "JSON schema" }
  }
}
```

That schema is sent to the model as part of the same API call that produces the next assistant message. The model reads:

- the **function `description`** to decide WHEN to call this tool
- the **per-parameter `description` strings** inside `parameters` to decide HOW to fill in each argument

This means **descriptions are prompt engineering**, not developer documentation. They are not a comment for the next human reading the code — they are instructions to the LLM that directly determine whether the model picks your tool when the user's request matches it.

Compare:

| Bad (model often misses the tool) | Good (model picks it reliably) |
|---|---|
| `Description: "Lookup function"` | `Description: "Look up a customer's account details by their account number. Use this BEFORE quoting any account-specific information (balance, plan, status, billing date). Don't use it for general product questions."` |
| `"description": "the id"` (parameter) | `"description": "The customer's 8-digit account number, no dashes or spaces. Ask the user if they don't provide it."` |

A vague description is the #1 cause of "the model has the right tool but doesn't call it" failures. When you find yourself debugging why the model isn't picking a tool that obviously matches the user's request, the first thing to check is whether the description tells the model — in plain language — when to use it and what makes it the right choice over sibling tools.

**Tool count matters too.** LLM tool selection accuracy degrades noticeably past ~7-8 simultaneously-active tools per call. If you have many tools, partition them across steps using `Step.SetFunctions()` so only the relevant subset is active at any moment. See `contexts_guide.md` for the per-step whitelist mechanism.

### 1. Local Webhook Functions (Standard)

These are the traditional SWAIG functions that are handled locally by your agent. Register them with `DefineTool`, providing a handler that returns a `*swaig.FunctionResult`:

```go
import "fmt"

a.DefineTool(agent.ToolDefinition{
	Name:        "get_weather",
	Description: "Get the current weather for a location",
	Parameters: map[string]any{
		"location": map[string]any{
			"type":        "string",
			"description": "The city or location to get weather for",
		},
	},
	Secure: true, // Optional, defaults to true
	Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
		// Extract the location parameter
		location, _ := args["location"].(string)
		if location == "" {
			location = "Unknown location"
		}

		// Here you would typically call a weather API.
		// For this example, we'll return mock data.
		weatherData := fmt.Sprintf("It's sunny and 72°F in %s.", location)

		// Return a FunctionResult
		return swaig.NewFunctionResult(weatherData)
	},
})
```

### 2. External Webhook Functions

External webhook functions allow you to delegate function execution to external services instead of handling them locally. This is useful when you want to:
- Use existing web services or APIs directly
- Distribute function processing across multiple servers
- Integrate with third-party systems that provide their own endpoints

To create an external webhook function, set the `WebhookURL` field on the tool definition:

```go
a.DefineTool(agent.ToolDefinition{
	Name:        "get_weather_external",
	Description: "Get weather from external service",
	Parameters: map[string]any{
		"location": map[string]any{
			"type":        "string",
			"description": "The city or location to get weather for",
		},
	},
	WebhookURL: "https://your-service.com/weather-endpoint",
	Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
		// This handler will never be called locally when WebhookURL is set.
		// The external service at WebhookURL will receive the function call instead.
		return swaig.NewFunctionResult("This should not be reached for external webhooks")
	},
})
```

#### How External Webhooks Work

When you specify a `WebhookURL`:

1. **Function Registration**: The function is registered with your agent as usual
2. **SWML Generation**: The generated SWML includes the external webhook URL instead of your local endpoint
3. **SignalWire Processing**: When the AI calls the function, SignalWire makes an HTTP POST request directly to your external URL
4. **Payload Format**: The external service receives a JSON payload with the function call data:

```json
{
    "function": "get_weather_external",
    "argument": {
        "parsed": [{"location": "New York"}],
        "raw": "{\"location\": \"New York\"}"
    },
    "call_id": "abc123-def456-ghi789",
    "call": { "call": "information" },
    "vars": { "call": "variables" }
}
```

5. **Response Handling**: Your external service should return a JSON response that SignalWire will process.

#### Mixing Local and External Functions

You can mix both types of functions in the same agent:

```go
package main

import (
	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

func newHybridAgent() *agent.AgentBase {
	a := agent.NewAgentBase(
		agent.WithName("hybrid-agent"),
		agent.WithRoute("/hybrid"),
	)

	// Local function - handled by this agent
	a.DefineTool(agent.ToolDefinition{
		Name:        "get_help",
		Description: "Get help information",
		Handler: func(args, rawData map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("I can help you with weather and news!")
		},
	})

	// External function - handled by external service
	a.DefineTool(agent.ToolDefinition{
		Name:        "get_weather",
		Description: "Get current weather",
		Parameters: map[string]any{
			"location": map[string]any{"type": "string", "description": "City name"},
		},
		WebhookURL: "https://weather-service.com/api/weather",
		Handler: func(args, rawData map[string]any) *swaig.FunctionResult {
			return nil // never called for external webhooks
		},
	})

	// Another external function - different service
	a.DefineTool(agent.ToolDefinition{
		Name:        "get_news",
		Description: "Get latest news",
		Parameters: map[string]any{
			"topic": map[string]any{"type": "string", "description": "News topic"},
		},
		WebhookURL: "https://news-service.com/api/news",
		Handler: func(args, rawData map[string]any) *swaig.FunctionResult {
			return nil // never called for external webhooks
		},
	})

	return a
}

func main() {}
```

#### Testing External Webhooks

You can test external webhook functions using the CLI tool:

```bash
# Start the agent first, then drive it over HTTP via --url.
# Test local function
swaig-test --url http://localhost:3000/ --exec get_help

# Test external webhook function (pass args with --param key=value)
swaig-test --url http://localhost:3000/ --verbose --exec get_weather --param location="New York"

# List all functions with their types
swaig-test --url http://localhost:3000/ --list-tools
```

The CLI tool will automatically detect external webhook functions and make HTTP requests to the external services, simulating what SignalWire does in production.

### Function Parameters

The parameters for a SWAIG function are defined using JSON Schema, expressed in Go as a `map[string]any`:

<!-- snippet: no-compile illustrative struct-field literal (Parameters value fragment) -->
```go
Parameters: map[string]any{
	"parameter_name": map[string]any{
		"type":        "string", // Can be string, number, integer, boolean, array, object
		"description": "Description of the parameter",
		// Optional attributes:
		"enum":    []string{"option1", "option2"}, // For enumerated values
		"minimum": 0,                               // For numeric types
		"maximum": 100,                             // For numeric types
		"pattern": "^[A-Z]+$",                      // For string validation
	},
}
```

Parameters that must always be present are listed in the tool definition's `Required` field (a `[]string` of parameter names).

### Function Results

To return results from a SWAIG function, use the `swaig.FunctionResult` builder:

<!-- snippet: no-compile illustrative bare-return examples (multiple returns outside a function body) -->
```go
// Basic result with just text
return swaig.NewFunctionResult("Here's the result")

// Result with a single action
return swaig.NewFunctionResult("Here's the result with an action").
	AddAction("say", "I found the information you requested.")

// Result with multiple actions using AddActions
return swaig.NewFunctionResult("Multiple actions example").
	AddActions([]map[string]any{
		{"playback_bg": map[string]any{"file": "https://example.com/music.mp3"}},
		{"set_global_data": map[string]any{"key": "value"}},
	})

// Alternative way to add multiple actions sequentially (fluent chaining)
return swaig.NewFunctionResult("Sequential actions example").
	AddAction("say", "I found the information you requested.").
	AddAction("playback_bg", map[string]any{"file": "https://example.com/music.mp3"})
```

In the examples above:
- `AddAction(name, data)` adds a single action with the given name and data
- `AddActions(actions)` adds multiple actions at once from a slice of action objects

Many common actions also have dedicated helper methods that read more idiomatically, e.g. `.Say("...")`, `.Hangup()`, `.Hold(60)`, `.Connect(dest, final, from)`, `.UpdateGlobalData(...)`, and `.SetMetadata(...)`. See the `swaig_features` example for the full set.

### Native Functions

The agent can use SignalWire's built-in functions:

```go
// Enable native functions
a.SetNativeFunctions([]string{
	"check_time",
	"wait_seconds",
})
```

### Function Includes

You can include functions from remote sources:

```go
// Include remote functions
a.AddFunctionInclude(
	"https://api.example.com/functions",
	[]string{"get_weather", "get_news"},
	map[string]any{"session_id": "unique-session-123"}, // Use for session tracking, NOT credentials
)
```

### SWAIG Function Security

The SDK implements an automated security mechanism for SWAIG functions to ensure that only authorized calls can be made to your functions. This is important because SWAIG functions often provide access to sensitive operations or data.

#### Token-Based Security

By default, all SWAIG functions are marked as secure (`Secure: true`), which enables token-based security:

```go
a.DefineTool(agent.ToolDefinition{
	Name:        "get_account_details",
	Description: "Get customer account details",
	Parameters: map[string]any{
		"account_id": map[string]any{"type": "string"},
	},
	Secure: true, // This is the default, can be omitted
	Handler: func(args, rawData map[string]any) *swaig.FunctionResult {
		// Implementation
		return swaig.NewFunctionResult("...")
	},
})
```

When a function is marked as secure:

1. The SDK automatically generates a secure token for each function when rendering the SWML document
2. The token is added to the function's URL as a query parameter: `?token=X2FiY2RlZmcuZ2V0X3RpbWUuMTcxOTMxNDI1...`
3. When the function is called, the token is validated before executing the function

These security tokens have important properties:
- **Completely stateless**: The system doesn't need to store tokens or track sessions
- **Self-contained**: Each token contains all information needed for validation
- **Function-specific**: A token for one function can't be used for another
- **Session-bound**: Tokens are tied to a specific call/session ID
- **Time-limited**: Tokens expire after a configurable duration (default: 60 minutes)
- **Cryptographically signed**: Tokens can't be tampered with or forged

This stateless design provides several benefits:
- **Server resilience**: Tokens remain valid even if the server restarts
- **No memory consumption**: No need to track sessions or store tokens in memory
- **High scalability**: Multiple servers can validate tokens without shared state
- **Load balancing**: Requests can be distributed across multiple servers freely

The token system secures both SWAIG functions and post-prompt endpoints:
- SWAIG function calls for interactive AI capabilities
- Post-prompt requests for receiving conversation summaries

You can disable token security for specific functions when appropriate:

```go
a.DefineTool(agent.ToolDefinition{
	Name:        "get_public_information",
	Description: "Get public information that doesn't require security",
	Secure:      false, // Disable token security for this function
	Handler: func(args, rawData map[string]any) *swaig.FunctionResult {
		// Implementation
		return swaig.NewFunctionResult("...")
	},
})
```

#### Token Expiration

The default token expiration is 60 minutes (3600 seconds), but you can configure this when constructing your agent:

```go
a = agent.NewAgentBase(
	agent.WithName("my_agent"),
	agent.WithTokenExpiry(1800), // Set token expiration to 30 minutes
)
```

The expiration timer resets each time a function is successfully called, so as long as there is activity at least once within the expiration period, the tokens will remain valid throughout the entire conversation.

#### Token Validation

Token validation and issuance are handled by `ValidateToolToken(functionName, token, callID)` and `CreateToolToken(toolName, callID)` on the agent. These use HMAC-SHA256 signing keyed by the agent's signing key (set via `agent.WithSigningKey(key)`).

## Skills System

The Skills System allows you to extend your agents with reusable capabilities via one-liner calls. Skills are modular, reusable components that can be easily added to any agent and configured with parameters.

Skills are referenced by typed `skills.SkillName` constants (e.g. `skills.SkillWebSearch`, `skills.SkillDatetime`, `skills.SkillMath`), not raw strings.

### Quick Start

```go
package main

import (
	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/skills"
)

func main() {}

func newSkillfulAgent() *agent.AgentBase {
	a := agent.NewAgentBase(
		agent.WithName("skillful-agent"),
		agent.WithRoute("/skillful"),
	)

	// Add skills with one-liners (pass nil for default params)
	a.AddSkill(skills.SkillWebSearch, nil) // Web search capability
	a.AddSkill(skills.SkillDatetime, nil)  // Current date/time info
	a.AddSkill(skills.SkillMath, nil)      // Mathematical calculations

	// Configure skills with parameters
	a.AddSkill(skills.SkillWebSearch, map[string]any{
		"num_results": 3,   // Get 3 search results instead of default 1
		"delay":       0.5, // Add delay between requests
	})

	return a
}
```

### Available Built-in Skills

#### Web Search Skill (`skills.SkillWebSearch`)
Provides web search capabilities using Google Custom Search API with web scraping.

**Parameters:**
- `api_key` (required): Google Custom Search API key
- `search_engine_id` (required): Google Custom Search Engine ID
- `num_results` (default: 1): Number of search results to return
- `delay` (default: 0): Delay in seconds between requests
- `tool_name` (default: "web_search"): Custom name for the search tool
- `no_results_message` (default: "I couldn't find any results for '{query}'. This might be due to a very specific query or temporary issues. Try rephrasing your search or asking about a different topic."): Custom message to return when no search results are found. Use `{query}` as a placeholder for the search query.

**Multiple Instance Support:**
The web_search skill supports multiple instances with different search engines and tool names, allowing you to search different data sources:

**Example:**
```go
// Basic single instance
a.AddSkill(skills.SkillWebSearch, map[string]any{
	"api_key":          "your-google-api-key",
	"search_engine_id": "your-search-engine-id",
})
// Creates tool: web_search

// Fast single result (previous default)
a.AddSkill(skills.SkillWebSearch, map[string]any{
	"api_key":          "your-google-api-key",
	"search_engine_id": "your-search-engine-id",
	"num_results":      1,
	"delay":            0,
})

// Multiple results with delay
a.AddSkill(skills.SkillWebSearch, map[string]any{
	"api_key":          "your-google-api-key",
	"search_engine_id": "your-search-engine-id",
	"num_results":      5,
	"delay":            1.0,
})

// Multiple instances with different search engines
a.AddSkill(skills.SkillWebSearch, map[string]any{
	"api_key":          "your-google-api-key",
	"search_engine_id": "general-search-engine-id",
	"tool_name":        "search_general",
	"num_results":      1,
})
// Creates tool: search_general

a.AddSkill(skills.SkillWebSearch, map[string]any{
	"api_key":          "your-google-api-key",
	"search_engine_id": "news-search-engine-id",
	"tool_name":        "search_news",
	"num_results":      3,
	"delay":            0.5,
})
// Creates tool: search_news

// Custom no results message
a.AddSkill(skills.SkillWebSearch, map[string]any{
	"api_key":            "your-google-api-key",
	"search_engine_id":   "your-search-engine-id",
	"no_results_message": "Sorry, I couldn't find information about '{query}'. Please try a different search term.",
})
```

#### DateTime Skill (`skills.SkillDatetime`)
Provides current date and time information with timezone support.

**Tools Added:**
- `get_current_time`: Get current time with optional timezone
- `get_current_date`: Get current date with optional timezone

**Example:**
```go
a.AddSkill(skills.SkillDatetime, nil)
// Agent can now tell users the current time and date
```

#### Math Skill (`skills.SkillMath`)
Provides safe mathematical expression evaluation.

**Tools Added:**
- `calculate`: Evaluate mathematical expressions safely

**Example:**
```go
a.AddSkill(skills.SkillMath, nil)
// Agent can now perform calculations like "2 + 3 * 4"
```

#### DataSphere Skill (`skills.SkillDatasphere`)
Provides knowledge search capabilities using SignalWire DataSphere, a cloud-hosted document search and retrieval-augmented generation (RAG) service.

**Parameters:**
- `space_name` (required): SignalWire space name
- `project_id` (required): SignalWire project ID
- `token` (required): SignalWire authentication token
- `document_id` (required): DataSphere document ID to search
- `count` (default: 1): Number of search results to return
- `distance` (default: 3.0): Distance threshold for search matching
- `tags` (optional): List of tags to filter search results
- `language` (optional): Language code to limit search
- `pos_to_expand` (optional): List of parts of speech for synonym expansion (e.g., `["NOUN", "VERB"]`)
- `max_synonyms` (optional): Maximum number of synonyms to use for each word
- `tool_name` (default: "search_knowledge"): Custom name for the search tool
- `no_results_message` (default: "I couldn't find any relevant information for '{query}' in the knowledge base. Try rephrasing your question or asking about a different topic."): Custom message when no results found

**Multiple Instance Support:**
The DataSphere skill supports multiple instances with different tool names, allowing you to search multiple knowledge bases:

**Example:**
```go
// Basic single instance
a.AddSkill(skills.SkillDatasphere, map[string]any{
	"space_name":  "my-space",
	"project_id":  "my-project",
	"token":       "my-token",
	"document_id": "general-knowledge",
})
// Creates tool: search_knowledge

// Multiple instances for different knowledge bases
a.AddSkill(skills.SkillDatasphere, map[string]any{
	"space_name":  "my-space",
	"project_id":  "my-project",
	"token":       "my-token",
	"document_id": "product-docs",
	"tool_name":   "search_products",
	"tags":        []string{"Products", "Features"},
	"count":       3,
})
// Creates tool: search_products

a.AddSkill(skills.SkillDatasphere, map[string]any{
	"space_name":         "my-space",
	"project_id":         "my-project",
	"token":              "my-token",
	"document_id":        "support-kb",
	"tool_name":          "search_support",
	"no_results_message": "I couldn't find support information about '{query}'. Try contacting our support team.",
	"distance":           5.0,
})
// Creates tool: search_support
```

#### Native Vector Search Skill (`skills.SkillNativeVectorSearch`)
Provides knowledge-base search by querying a **remote search server** over HTTP. The Go skill is **remote-only**: it requires a `remote_url` and does not build or read local index files. (This differs from the Python SDK, which also supports local `.swsearch` index files.)

The skill connects to a search server that exposes `/health` and `/search` HTTP endpoints. On setup it validates `remote_url` (SSRF protection — http/https only, no private/loopback hosts unless `SWML_ALLOW_PRIVATE_URLS` is set) and checks the server's `/health` endpoint. Basic-auth credentials may be embedded in the URL (`http://user:pass@host:8001`).

**Requirements:**
- A reachable remote search server (no local packages or index files required).

**Parameters:**
- `remote_url` (**required**): URL of the remote search server (e.g., `http://localhost:8001`, or `http://user:pass@host:8001` for basic auth)
- `index_name` (default: `"default"`): Name of the index to query on the remote server
- `tool_name` (default: `"search_knowledge"`): Custom name for the search tool
- `description` (default: `"Search the knowledge base for information"`): Tool description shown to the AI
- `count` (default: 5): Number of search results to return
- `similarity_threshold` (default: 0.0): Minimum similarity score for results (0.0 = no limit, 1.0 = exact match)
- `tags` (optional): List of tags to filter search results
- `response_prefix` (optional): Text to prepend to search responses
- `response_postfix` (optional): Text to append to search responses
- `max_content_length` (default: 32768): Maximum total response size in characters (distributed across results)
- `no_results_message` (default: `"No information found for '{query}'"`): Message when no results are found; `{query}` is substituted
- `hints` (optional): Additional speech-recognition hints for this skill

**Multiple Instance Support:**
The native vector search skill supports multiple instances with different remote indexes and tool names:

**Example:**
```go
// Basic remote search
a.AddSkill(skills.SkillNativeVectorSearch, map[string]any{
	"remote_url":  "http://localhost:8001",
	"index_name":  "concepts",
	"tool_name":   "search_knowledge",
	"description": "Search the knowledge base",
	"count":       3,
})
// Creates tool: search_knowledge

// Second instance against a different index/server
a.AddSkill(skills.SkillNativeVectorSearch, map[string]any{
	"remote_url":      "http://search.internal:8001",
	"index_name":      "examples",
	"tool_name":       "search_examples",
	"description":     "Search code examples",
	"response_prefix": "From the examples:",
})
// Creates tool: search_examples

// Voice-optimized responses
a.AddSkill(skills.SkillNativeVectorSearch, map[string]any{
	"remote_url":         "http://localhost:8001",
	"index_name":         "concepts",
	"tool_name":          "search_docs",
	"response_prefix":    "Based on the comprehensive SDK guide:",
	"response_postfix":   "Would you like more specific information?",
	"no_results_message": "I couldn't find information about '{query}' in the concepts guide.",
})
```

The remote search server is a separate component that hosts the indexes; the Go SDK does not include a CLI for building indexes.

### Skill Management

```go
import "fmt"

// Check what skills are loaded
loadedSkills := a.ListSkills()
fmt.Printf("Loaded skills: %s\n", strings.Join(loadedSkills, ", "))

// Check if a specific skill is loaded
if a.HasSkill(skills.SkillWebSearch) {
	fmt.Println("Web search is available")
}

// Remove a skill (if needed)
a.RemoveSkill(skills.SkillMath)
```

### Advanced Skill Configuration with swaig_fields

Skills support a special `swaig_fields` parameter that allows you to customize how SWAIG functions are registered. When you pass `swaig_fields` to a skill, they are automatically merged into all tool definitions created by that skill.

```go
// Add a skill with swaig_fields to customize SWAIG function properties
a.AddSkill(skills.SkillMath, map[string]any{
	"precision": 2, // Regular skill parameter
	"swaig_fields": map[string]any{ // Special fields merged into SWAIG function automatically
		"secure": false, // Override default security requirement
		"fillers": map[string]any{
			"en-US": []string{"Let me calculate that...", "Computing the result..."},
			"es-ES": []string{"Déjame calcular eso...", "Calculando el resultado..."},
		},
	},
})

// Add web search with custom security and fillers
a.AddSkill(skills.SkillWebSearch, map[string]any{
	"num_results": 3,
	"delay":       0.5,
	"swaig_fields": map[string]any{
		"secure": true, // Require authentication
		"fillers": map[string]any{
			"en-US": []string{"Searching the web...", "Looking that up...", "Finding information..."},
		},
	},
})
```

The `swaig_fields` can include any field the SWAIG function system supports:
- `secure`: Boolean indicating if the function requires authentication
- `fillers`: Map of language codes to arrays of filler phrases
- Any other fields supported by the SWAIG function system

### Error Handling

`AddSkill` returns the agent for chaining; skills validate their required parameters/environment during load and surface failures through the agent's structured logger. Validate required configuration (API keys, environment variables) before adding a skill so a missing dependency is caught early:

```go
if os.Getenv("GOOGLE_SEARCH_API_KEY") == "" {
	log.Println("web_search unavailable: GOOGLE_SEARCH_API_KEY not set")
	// Continue without web search capability
} else {
	a.AddSkill(skills.SkillWebSearch, nil)
}
```

### Creating Custom Skills

You can create your own skills by implementing the `skills.SkillBase` interface (in Go the skill is an interface, not a base class to subclass). A skill registers its tools with the agent, contributes speech-recognition hints, and can add prompt sections. For example, a weather skill would:

- declare its `SkillName`, description, and version
- validate required parameters in its `Setup` step
- register a `get_weather` tool (with a `location` parameter and optional `units` enum) whose handler returns a `*swaig.FunctionResult`
- return speech-recognition hints such as `[]string{"weather", "temperature", "forecast", "conditions"}`
- return prompt sections describing when to use the tool

Once implemented and registered with the skill manager, use it in your agent like any built-in skill:

```go
a.AddSkill(skills.SkillName("weather"), map[string]any{
	"units":   "celsius",
	"timeout": 15,
})
```

For the exact interface methods to implement, see the built-in skills under `pkg/skills` as reference implementations.

### Skills with Dynamic Configuration

Skills work with dynamic configuration — add them inside your per-request callback:

```go
a.SetDynamicConfigCallback(func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
	// Add different skills based on request parameters
	tier := queryParams["tier"]
	if tier == "" {
		tier = "basic"
	}

	// Basic skills for all users
	ep.AddSkill(skills.SkillDatetime, nil)
	ep.AddSkill(skills.SkillMath, nil)

	// Premium skills for premium users
	switch tier {
	case "premium":
		ep.AddSkill(skills.SkillWebSearch, map[string]any{
			"num_results": 5,
			"delay":       0.5,
		})
	case "basic":
		ep.AddSkill(skills.SkillWebSearch, map[string]any{
			"num_results": 1,
			"delay":       0,
		})
	}
})
```

### Best Practices

1. **Choose appropriate parameters**: Configure skills for your use case
   ```go
   // For speed (customer service)
   a.AddSkill(skills.SkillWebSearch, map[string]any{"num_results": 1, "delay": 0})

   // For research (detailed analysis)
   a.AddSkill(skills.SkillWebSearch, map[string]any{"num_results": 5, "delay": 1.0})
   ```

2. **Handle missing dependencies gracefully**: Check required environment/config before adding a skill (see [Error Handling](#error-handling) above).

3. **Document your custom skills**: Include clear descriptions and parameter documentation

4. **Test skills in isolation**: Create simple test programs to verify skill functionality

For more detailed information about the skills system architecture and advanced customization, see the [Skills System Guide](skills_system.md).

## Multilingual Support

Agents can support multiple languages. Use `AddLanguage(config)` with a `map[string]any`, or the typed helper `AddLanguageTyped(name, code, voice, speechFillers, functionFillers, engine, model, params...)`:

```go
// Add English language (typed helper)
a.AddLanguageTyped(
	"English",
	"en-US",
	"en-US-Neural2-F",
	[]string{"Let me think...", "One moment please..."}, // speech fillers
	[]string{"I'm looking that up...", "Let me check that..."}, // function fillers
	"", "", // engine, model (empty = default)
)

// Add Spanish language (map form)
a.AddLanguage(map[string]any{
	"name":           "Spanish",
	"code":           "es",
	"voice":          "rime.spore:multilingual",
	"speech_fillers": []string{"Un momento por favor...", "Estoy pensando..."},
})
```

### Voice Formats

There are different ways to specify voices:

```go
// Simple format
a.AddLanguage(map[string]any{"name": "English", "code": "en-US", "voice": "en-US-Neural2-F"})

// Explicit parameters with engine and model
a.AddLanguageTyped(
	"British English",
	"en-GB",
	"spore",
	nil, nil,
	"rime",         // engine
	"multilingual", // model
)

// Combined string format
a.AddLanguage(map[string]any{
	"name":  "Spanish",
	"code":  "es",
	"voice": "rime.spore:multilingual",
})
```

## Agent Configuration

### Adding Hints

Hints help the AI understand certain terms better:

```go
// Simple hints (slice of words)
a.AddHints([]string{"SignalWire", "SWML", "SWAIG"})

// Pattern hint with replacement
a.AddPatternHint(
	"AI Agent",     // hint
	"AI\\s+Agent",  // pattern
	"A.I. Agent",   // replace
	true,           // ignoreCase
)
```

### Adding Pronunciation Rules

Pronunciation rules help the AI speak certain terms correctly:

```go
// Add pronunciation rule (replace, withText, ignoreCase...)
a.AddPronunciation("API", "A P I", false)
a.AddPronunciation("SIP", "sip", true)
```

### Setting AI Parameters

Configure various AI behavior parameters:

```go
// Set AI parameters
a.SetParams(map[string]any{
	"wait_for_user":         false,
	"end_of_speech_timeout": 1000,
	"ai_volume":             5,
	"languages_enabled":     true,
	"local_tz":              "America/Los_Angeles",
})
```

Use `SetParam(key, value)` to set a single parameter.

### Setting Global Data

Provide global data for the AI to reference:

```go
// Set global data
a.SetGlobalData(map[string]any{
	"company_name": "SignalWire",
	"product":      "AI Agent SDK",
	"supported_features": []string{
		"Voice AI",
		"Telephone integration",
		"SWAIG functions",
	},
})
```

### Customizing LLM Parameters

The SDK provides methods to fine-tune the Language Model parameters for both the main prompt and post-prompt, giving you precise control over the AI's behavior. The parameters are passed as a `map[string]any` and forwarded to the server, which validates them based on the model:

```go
// Set LLM parameters for the main prompt
a.SetPromptLlmParams(map[string]any{
	"temperature":       0.7, // Controls randomness
	"top_p":             0.9, // Nucleus sampling threshold
	"barge_confidence":  0.6, // ASR confidence to interrupt
	"presence_penalty":  0.0, // Penalizes token repetition
	"frequency_penalty": 0.0, // Penalizes frequent word usage
})

// Set different parameters for the post-prompt
a.SetPostPromptLlmParams(map[string]any{
	"temperature": 0.3,  // Lower temperature for consistent summaries
	"top_p":       0.95, // Slightly wider token selection
})
```

**Common Use Cases:**

- **Customer Service**: Low temperature (0.2-0.4) for consistent, professional responses
- **Creative Tasks**: Higher temperature (0.7-0.9) for varied, creative outputs
- **Technical Support**: Very low temperature (0.1-0.3) with high confidence for accuracy
- **General Assistant**: Medium temperature (0.5-0.7) for balanced interaction

For detailed information about each parameter and advanced tuning strategies, see [LLM Parameters Guide](llm_parameters.md).

## Dynamic Agent Configuration

Dynamic agent configuration allows you to configure agents per-request based on parameters from the HTTP request (query parameters, body data, headers). This enables patterns like multi-tenant applications, A/B testing, personalization, and localization.

### Overview

There are two main approaches to agent configuration:

#### Static Configuration (Traditional)
```go
package main

import (
	"github.com/signalwire/signalwire-go/pkg/agent"
)

func newStaticAgent() *agent.AgentBase {
	a := agent.NewAgentBase(agent.WithName("static-agent"))

	// Configuration happens once at startup
	a.AddLanguage(map[string]any{"name": "English", "code": "en-US", "voice": "rime.spore:mistv2"})
	a.SetParams(map[string]any{"end_of_speech_timeout": 500})
	a.PromptAddSection("Role", "You are a customer service agent.", nil)
	a.SetGlobalData(map[string]any{"service_level": "standard"})

	return a
}

func main() {}
```

**Pros**: Simple, fast, predictable
**Cons**: Same behavior for all users, requires separate agents for different configurations

#### Dynamic Configuration
```go
package main

import (
	"github.com/signalwire/signalwire-go/pkg/agent"
)

func newDynamicAgent() *agent.AgentBase {
	a := agent.NewAgentBase(agent.WithName("dynamic-agent"))

	// No static configuration - set up dynamic callback instead
	a.SetDynamicConfigCallback(func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
		// Configuration happens fresh for each request
		tier := queryParams["tier"]
		if tier == "" {
			tier = "standard"
		}

		if tier == "premium" {
			ep.AddLanguage(map[string]any{"name": "English", "code": "en-US", "voice": "rime.spore:mistv2"})
			ep.SetParams(map[string]any{"end_of_speech_timeout": 300}) // Faster
			ep.PromptAddSection("Role", "You are a premium customer service agent.", nil)
			ep.SetGlobalData(map[string]any{"service_level": "premium"})
		} else {
			ep.AddLanguage(map[string]any{"name": "English", "code": "en-US", "voice": "rime.spore:mistv2"})
			ep.SetParams(map[string]any{"end_of_speech_timeout": 500}) // Standard
			ep.PromptAddSection("Role", "You are a customer service agent.", nil)
			ep.SetGlobalData(map[string]any{"service_level": "standard"})
		}
	})

	return a
}

func main() {}
```

**Pros**: Highly flexible, single agent serves multiple configurations, enables advanced use cases
**Cons**: Slightly more complex, configuration overhead per request

### Setting Up Dynamic Configuration

Use the `SetDynamicConfigCallback()` method to register a callback function that will be called for each request:

```go
package main

import (
	"github.com/signalwire/signalwire-go/pkg/agent"
)

func newMyDynamicAgent() *agent.AgentBase {
	a := agent.NewAgentBase(
		agent.WithName("my-agent"),
		agent.WithRoute("/agent"),
	)

	// Register the dynamic configuration callback
	a.SetDynamicConfigCallback(configureAgentDynamically)

	return a
}

// configureAgentDynamically is called for every request to configure the agent.
//
//	queryParams: query string parameters from the URL
//	bodyParams:  parsed JSON body from POST requests
//	headers:     HTTP headers from the request
//	ep:          the (cloned) agent instance to configure
func configureAgentDynamically(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
	// Your dynamic configuration logic here
}

func main() {}
```

The callback function receives four parameters:
- **queryParams** (`map[string]string`): URL query parameters
- **bodyParams** (`map[string]any`): parsed JSON body (empty for GET requests)
- **headers** (`map[string]string`): HTTP headers
- **ep** (`*agent.AgentBase`): the agent instance to configure dynamically

### Dynamic Configuration Methods

The agent parameter in your callback is the actual agent instance, allowing you to use all the same configuration methods you would use during initialization:

#### Language Configuration
```go
// Add languages with voice configuration
ep.AddLanguage(map[string]any{"name": "English", "code": "en-US", "voice": "rime.spore:mistv2"})
ep.AddLanguage(map[string]any{"name": "Spanish", "code": "es-ES", "voice": "rime.spore:mistv2"})
```

#### Prompt Building
```go
// Add prompt sections
ep.PromptAddSection("Role", "You are a helpful assistant.", nil)
ep.PromptAddSection("Guidelines", "", []string{
	"Be professional and courteous",
	"Provide accurate information",
	"Ask clarifying questions when needed",
})

// Set raw prompt text
ep.SetPromptText("You are a specialized AI assistant...")

// Set post-prompt for summary
ep.SetPostPrompt("Summarize the key points of this conversation.")
```

#### AI Parameters
```go
// Configure AI behavior
ep.SetParams(map[string]any{
	"end_of_speech_timeout":  300,
	"attention_timeout":      20000,
	"background_file_volume": -30,
})
```

#### Global Data
```go
// Set data available to the AI
ep.SetGlobalData(map[string]any{
	"customer_tier":    "premium",
	"features_enabled": []string{"advanced_support", "priority_queue"},
	"session_info":     map[string]any{"start_time": "2024-01-01T00:00:00Z"},
})

// Update existing global data
ep.UpdateGlobalData(map[string]any{"additional_info": "value"})
```

#### Speech Recognition Hints
```go
// Add hints for better speech recognition
ep.AddHints([]string{"SignalWire", "SWML", "API", "technical"})
ep.AddPronunciation("API", "A P I", false)
```

#### Function Configuration
```go
// Set native functions
ep.SetNativeFunctions([]string{"transfer", "hangup"})

// Add function includes
ep.AddFunctionInclude(
	"https://api.example.com/functions",
	[]string{"get_account_info", "update_profile"},
	nil,
)
```

### Request Data Access

Your callback function receives detailed information about the incoming request:

#### Query Parameters
```go
package main

import (
	"strings"
	"github.com/signalwire/signalwire-go/pkg/agent"
)

func configureAgentDynamically(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
	// Extract query parameters
	tier := queryParams["tier"]
	if tier == "" {
		tier = "standard"
	}
	customerID := queryParams["customer_id"]
	debug := strings.ToLower(queryParams["debug"]) == "true"
	_ = debug

	// Use parameters for configuration
	if tier == "premium" {
		ep.SetParams(map[string]any{"end_of_speech_timeout": 300})
	}

	if customerID != "" {
		ep.SetGlobalData(map[string]any{"customer_id": customerID})
	}
}

// Request: GET /agent?tier=premium&language=es&customer_id=12345&debug=true

func main() {}
```

#### POST Body Parameters
```go
package main

import (
	"github.com/signalwire/signalwire-go/pkg/agent"
)

func configureAgentDynamically(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
	// Extract from POST body
	userProfile, _ := bodyParams["user_profile"].(map[string]any)
	preferences, _ := bodyParams["preferences"].(map[string]any)

	// Configure based on profile
	if lang, _ := userProfile["language"].(string); lang == "es" {
		ep.AddLanguage(map[string]any{"name": "Spanish", "code": "es-ES", "voice": "rime.spore:mistv2"})
	}

	if speed, _ := preferences["voice_speed"].(string); speed == "fast" {
		ep.SetParams(map[string]any{"end_of_speech_timeout": 200})
	}
}

// Request: POST /agent with JSON body:
// {
//   "user_profile": {"language": "es", "region": "mx"},
//   "preferences": {"voice_speed": "fast", "tone": "formal"}
// }

func main() {}
```

#### HTTP Headers
```go
package main

import (
	"strings"
	"github.com/signalwire/signalwire-go/pkg/agent"
)

func configureAgentDynamically(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
	// Extract headers
	userAgent := headers["user-agent"]
	locale := headers["accept-language"]
	if locale == "" {
		locale = "en-US"
	}

	// Configure based on headers
	if strings.Contains(strings.ToLower(userAgent), "mobile") {
		ep.SetParams(map[string]any{"end_of_speech_timeout": 400}) // Longer for mobile
	}

	if strings.HasPrefix(locale, "es") {
		ep.AddLanguage(map[string]any{"name": "Spanish", "code": "es-ES", "voice": "rime.spore:mistv2"})
	}
}

func main() {}
```

### Configuration Examples

#### Simple Multi-Tenant Configuration
```go
package main

import (
	"github.com/signalwire/signalwire-go/pkg/agent"
)

func configureAgentDynamically(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
	tenant := queryParams["tenant"]
	if tenant == "" {
		tenant = "default"
	}

	// Tenant-specific configuration
	switch tenant {
	case "healthcare":
		ep.AddLanguage(map[string]any{"name": "English", "code": "en-US", "voice": "rime.spore:mistv2"})
		ep.PromptAddSection("Compliance",
			"Follow HIPAA guidelines and maintain patient confidentiality.", nil)
		ep.SetGlobalData(map[string]any{
			"industry":         "healthcare",
			"compliance_level": "hipaa",
		})
	case "finance":
		ep.AddLanguage(map[string]any{"name": "English", "code": "en-US", "voice": "rime.spore:mistv2"})
		ep.PromptAddSection("Compliance",
			"Follow financial regulations and protect sensitive data.", nil)
		ep.SetGlobalData(map[string]any{
			"industry":         "finance",
			"compliance_level": "pci",
		})
	}
}

func main() {}
```

#### Language and Localization
```go
package main

import (
	"github.com/signalwire/signalwire-go/pkg/agent"
)

func configureAgentDynamically(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
	language := queryParams["language"]
	if language == "" {
		language = "en"
	}
	region := queryParams["region"]
	if region == "" {
		region = "us"
	}

	// Configure language and voice
	switch language {
	case "es":
		if region == "mx" {
			ep.AddLanguage(map[string]any{"name": "Spanish (Mexico)", "code": "es-MX", "voice": "rime.spore:mistv2"})
		} else {
			ep.AddLanguage(map[string]any{"name": "Spanish", "code": "es-ES", "voice": "rime.spore:mistv2"})
		}
		ep.PromptAddSection("Language", "Respond in Spanish.", nil)
	case "fr":
		ep.AddLanguage(map[string]any{"name": "French", "code": "fr-FR", "voice": "rime.alois"})
		ep.PromptAddSection("Language", "Respond in French.", nil)
	default:
		ep.AddLanguage(map[string]any{"name": "English", "code": "en-US", "voice": "rime.spore:mistv2"})
	}

	// Regional customization
	currency := "MXN"
	switch region {
	case "us":
		currency = "USD"
	case "eu":
		currency = "EUR"
	}
	ep.SetGlobalData(map[string]any{
		"language": language,
		"region":   region,
		"currency": currency,
	})
}

func main() {}
```

#### A/B Testing Configuration
```go
package main

import (
	"github.com/signalwire/signalwire-go/pkg/agent"
)

func configureAgentDynamically(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
	// Determine test group (could be from query param, user ID hash, etc.)
	testGroup := queryParams["test_group"]
	if testGroup == "" {
		testGroup = "A"
	}

	if testGroup == "A" {
		// Control group - standard configuration
		ep.SetParams(map[string]any{"end_of_speech_timeout": 500})
		ep.PromptAddSection("Style", "Use a standard conversational approach.", nil)
		ep.SetGlobalData(map[string]any{"test_group": "A", "features": []string{"basic"}})
	} else {
		// Test group B - experimental features
		ep.SetParams(map[string]any{"end_of_speech_timeout": 300})
		ep.PromptAddSection("Style",
			"Use an enhanced, more interactive conversational approach.", nil)
		ep.SetGlobalData(map[string]any{"test_group": "B", "features": []string{"basic", "enhanced"}})
	}
}

func main() {}
```

#### Customer Tier-Based Configuration
```go
package main

import (
	"github.com/signalwire/signalwire-go/pkg/agent"
)

func configureAgentDynamically(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
	customerID := queryParams["customer_id"]
	tier := queryParams["tier"]
	if tier == "" {
		tier = "standard"
	}

	// Base configuration
	ep.AddLanguage(map[string]any{"name": "English", "code": "en-US", "voice": "rime.spore:mistv2"})

	// Tier-specific configuration
	var features []string
	switch tier {
	case "enterprise":
		ep.SetParams(map[string]any{
			"end_of_speech_timeout": 200,   // Fastest response
			"attention_timeout":     30000, // Longest attention span
		})
		ep.PromptAddSection("Service Level",
			"You provide white-glove enterprise support with priority handling.", nil)
		features = []string{"all_features", "dedicated_support", "custom_integration"}
	case "premium":
		ep.SetParams(map[string]any{
			"end_of_speech_timeout": 300,
			"attention_timeout":     20000,
		})
		ep.PromptAddSection("Service Level",
			"You provide premium support with enhanced features.", nil)
		features = []string{"premium_features", "priority_support"}
	default:
		ep.SetParams(map[string]any{
			"end_of_speech_timeout": 500,
			"attention_timeout":     15000,
		})
		ep.PromptAddSection("Service Level",
			"You provide standard customer support.", nil)
		features = []string{"basic_features"}
	}

	// Set global data
	globalData := map[string]any{"tier": tier, "features": features}
	if customerID != "" {
		globalData["customer_id"] = customerID
	}
	ep.SetGlobalData(globalData)
}

func main() {}
```

### Use Cases

#### Multi-Tenant SaaS Applications
Perfect for SaaS platforms where each customer needs different agent behavior:

```text
Different tenants get different capabilities:
  /agent?tenant=acme&industry=healthcare
  /agent?tenant=globex&industry=finance
```

Benefits:
- Single agent deployment serves all customers
- Tenant-specific branding and behavior
- Industry-specific compliance and terminology
- Custom feature sets per subscription level

#### A/B Testing and Experimentation
Test different agent configurations with real users:

```text
Split traffic between different configurations:
  /agent?test_group=A  (control)
  /agent?test_group=B  (experimental)
```

Benefits:
- Compare agent performance metrics
- Test new features with subset of users
- Gradual rollout of improvements
- Data-driven optimization

#### Personalization and User Preferences
Adapt agent behavior to individual user preferences:

```text
Personalized based on user profile:
  /agent?user_id=123&voice_speed=fast&formality=casual
```

Benefits:
- Improved user experience
- Accessibility support (voice speed, etc.)
- Cultural and linguistic adaptation
- Learning from user interactions

#### Geographic and Cultural Localization
Adapt to different regions and cultures:

```text
Location-based configuration:
  /agent?country=mx&language=es&timezone=America/Mexico_City
```

Benefits:
- Local language and dialect support
- Cultural appropriateness
- Regional business practices
- Time zone aware responses

### Migration Guide

#### Converting Static Agents to Dynamic

**Step 1: Move Configuration to Callback**

Before (Static):
```go
package main

import (
	"github.com/signalwire/signalwire-go/pkg/agent"
)

func newMyAgent() *agent.AgentBase {
	a := agent.NewAgentBase(agent.WithName("my-agent"))

	// Static configuration
	a.AddLanguage(map[string]any{"name": "English", "code": "en-US", "voice": "rime.spore:mistv2"})
	a.SetParams(map[string]any{"end_of_speech_timeout": 500})
	a.PromptAddSection("Role", "You are a helpful assistant.", nil)
	a.SetGlobalData(map[string]any{"version": "1.0"})

	return a
}

func main() {}
```

After (Dynamic):
```go
package main

import (
	"github.com/signalwire/signalwire-go/pkg/agent"
)

func newMyAgent() *agent.AgentBase {
	a := agent.NewAgentBase(agent.WithName("my-agent"))

	// Set up dynamic configuration
	a.SetDynamicConfigCallback(func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
		// Same configuration, but now dynamic
		ep.AddLanguage(map[string]any{"name": "English", "code": "en-US", "voice": "rime.spore:mistv2"})
		ep.SetParams(map[string]any{"end_of_speech_timeout": 500})
		ep.PromptAddSection("Role", "You are a helpful assistant.", nil)
		ep.SetGlobalData(map[string]any{"version": "1.0"})
	})

	return a
}

func main() {}
```

**Step 2: Add Parameter-Based Logic**

```go
a.SetDynamicConfigCallback(func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
	// Start with base configuration
	ep.AddLanguage(map[string]any{"name": "English", "code": "en-US", "voice": "rime.spore:mistv2"})
	ep.PromptAddSection("Role", "You are a helpful assistant.", nil)

	// Add parameter-based customization
	timeout := 500
	if v, err := strconv.Atoi(queryParams["timeout"]); err == nil {
		timeout = v
	}
	ep.SetParams(map[string]any{"end_of_speech_timeout": timeout})

	version := queryParams["version"]
	if version == "" {
		version = "1.0"
	}
	ep.SetGlobalData(map[string]any{"version": version})
})
```

**Step 3: Support Both Approaches During Migration**

You can support both static and dynamic patterns during migration by branching at construction time:

```go
package main

import (
	"github.com/signalwire/signalwire-go/pkg/agent"
)

func newMyAgent(useDynamic bool) *agent.AgentBase {
	a := agent.NewAgentBase(agent.WithName("my-agent"))

	if useDynamic {
		a.SetDynamicConfigCallback(func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
			// New dynamic configuration
			// ... dynamic config logic
		})
	} else {
		// Keep static configuration for backward compatibility
		a.AddLanguage(map[string]any{"name": "English", "code": "en-US", "voice": "rime.spore:mistv2"})
		// ... rest of static config
	}

	return a
}

func main() {}
```

### Best Practices

#### Performance Considerations

1. **Keep Callbacks Lightweight**
```go
a.SetDynamicConfigCallback(func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
	// Good: Simple parameter extraction and configuration
	tier := queryParams["tier"]
	if tier == "" {
		tier = "standard"
	}
	ep.SetParams(tierConfigs[tier])

	// Avoid: Heavy computation or external API calls
	// customerData := expensiveAPICall(customerID) // Don't do this
})
```

2. **Cache Configuration Data**
```go
package main

import "github.com/signalwire/signalwire-go/pkg/agent"

func main() {}

// Pre-compute configuration templates once (package-level or captured in a closure)
var tierConfigs = map[string]map[string]any{
	"basic":      {"end_of_speech_timeout": 500},
	"premium":    {"end_of_speech_timeout": 300},
	"enterprise": {"end_of_speech_timeout": 200},
}

func newMyAgent() *agent.AgentBase {
	a := agent.NewAgentBase(agent.WithName("my-agent"))

	a.SetDynamicConfigCallback(func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
		tier := queryParams["tier"]
		if tier == "" {
			tier = "basic"
		}
		ep.SetParams(tierConfigs[tier])
	})

	return a
}
```

3. **Use Default Values**
```go
a.SetDynamicConfigCallback(func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
	// Always provide defaults
	language := queryParams["language"]
	if language == "" {
		language = "en"
	}
	tier := queryParams["tier"]
	if tier == "" {
		tier = "standard"
	}

	// Handle invalid values gracefully
	switch language {
	case "en", "es", "fr":
		// valid
	default:
		language = "en"
	}
	_ = tier
})
```

#### Security Considerations

1. **Validate Input Parameters**
```go
a.SetDynamicConfigCallback(func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
	// Validate and sanitize inputs
	tier := queryParams["tier"]
	switch tier {
	case "basic", "premium", "enterprise":
		// valid
	default:
		tier = "basic" // Safe default
	}

	// Validate numeric parameters
	timeout := 500
	if v, err := strconv.Atoi(queryParams["timeout"]); err == nil {
		timeout = v
	}
	if timeout < 100 {
		timeout = 100
	} else if timeout > 2000 {
		timeout = 2000 // Clamp to reasonable range
	}
	_ = tier
	_ = timeout
})
```

2. **Protect Sensitive Configuration**
```go
a.SetDynamicConfigCallback(func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
	// Don't expose internal configuration via parameters.
	// Bad: ep.SetGlobalData(map[string]any{"api_key": queryParams["api_key"]})

	// Good: Use internal mapping for call-related data only
	customerID := queryParams["customer_id"]
	if customerID != "" && isValidCustomer(customerID) {
		// Store call-related customer info, NOT sensitive credentials
		ep.SetGlobalData(map[string]any{
			"customer_id":   customerID,
			"customer_tier": getCustomerTier(customerID),
			"account_type":  "premium",
		})
	}
})
```

3. **Rate Limiting for Complex Configurations**
```go
package main

import "github.com/signalwire/signalwire-go/pkg/agent"

// customerConfigCache stands in for your own memoizing cache.
type customerConfigCache struct{}

func (c *customerConfigCache) Get(id string) map[string]any { return map[string]any{} }

func main() {}

// Cache expensive lookups (e.g. with an in-memory cache guarded by a mutex)
func newMyAgent(cache *customerConfigCache) *agent.AgentBase {
	a := agent.NewAgentBase(agent.WithName("my-agent"))

	a.SetDynamicConfigCallback(func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
		customerID := queryParams["customer_id"]
		if customerID != "" {
			config := cache.Get(customerID) // memoized customer settings
			ep.SetGlobalData(config)
		}
	})

	return a
}
```

#### Error Handling

1. **Graceful Degradation**
```go
a.SetDynamicConfigCallback(func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
	if err := applyCustomConfig(queryParams, ep); err != nil {
		// Log error but don't fail the request
		log.Printf("config_error: %v", err)

		// Fall back to default configuration
		applyDefaultConfig(ep)
	}
})
```

2. **Configuration Validation**
```go
a.SetDynamicConfigCallback(func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
	// Validate required parameters
	if queryParams["tenant"] == "" {
		ep.SetGlobalData(map[string]any{"error": "Missing tenant parameter"})
		return
	}

	// Validate configuration makes sense
	language := queryParams["language"]
	if language == "" {
		language = "en"
	}
	region := queryParams["region"]
	if region == "" {
		region = "us"
	}

	if language == "es" && region == "us" {
		// Adjust for Spanish speakers in US
		ep.AddLanguage(map[string]any{"name": "Spanish (US)", "code": "es-US", "voice": "rime.spore:mistv2"})
	}
})
```

Dynamic agent configuration enables sophisticated, multi-tenant AI applications while maintaining the familiar AgentBase API. Start with simple parameter-based configuration and gradually add more complex logic as your use cases evolve.

## Advanced Features

### Debug Events

The debug events system provides real-time visibility into what the AI module is doing during a call. When enabled, the module POSTs structured JSON events to your agent throughout the call lifecycle — session start/end, barge interruptions, LLM errors, step changes, and more.

#### Basic Setup

```go
a = agent.NewAgentBase(agent.WithName("my_agent"))
a.EnableDebugEvents(1) // Level 1 — events are auto-logged
a.Serve()
```

With `EnableDebugEvents(1)`, every debug event is logged through the agent's structured logger. No other configuration is needed — the SDK automatically:
- Registers a `/debug_events` endpoint on the agent
- Sets `debug_webhook_url` and `debug_webhook_level` in the SWML params
- Logs each incoming event with its type and payload

#### Custom Event Handler

To act on specific events (alerting, metrics, custom logging), register a handler with `OnDebugEvent`. The handler receives each event as a `map[string]any` (the event's `event_type` and `call_id` are keys in the map):

```go
import "fmt"

a = agent.NewAgentBase(agent.WithName("my_agent"))
a.EnableDebugEvents(1)

a.OnDebugEvent(func(event map[string]any) {
	callID, _ := event["call_id"].(string)
	eventType, _ := event["event_type"].(string)

	switch eventType {
	case "barge":
		fmt.Printf("[%s] Caller interrupted after %vms\n", callID, event["barge_elapsed_ms"])

	case "llm_error":
		fmt.Printf("[%s] LLM error: %v\n", callID, event["event"])
		alertOpsTeam(event)

	case "session_end":
		durationMs, _ := event["duration_ms"].(float64)
		fmt.Printf("[%s] Call ended after %.1fs — reason: %v\n", callID, durationMs/1000, event["reason"])
	}
})

a.Serve()
```

The handler is called for every event in addition to the default structured logging.

#### Verbosity Levels

- **Level 1** (default): High-level events — session start/end, barge, errors, step changes, hold, filler, gather flow, action processing
- **Level 2+**: Adds high-volume events — every LLM request/response, conversation history additions

```go
a.EnableDebugEvents(2) // Include LLM request/response events
```

For the complete list of event types and their payloads, see the [API Reference](api_reference.md#debug-events).

### Session Lifecycle Hooks

SignalWire provides special SWAIG functions that are automatically called at specific points during a voice session's lifecycle. These hooks enable you to perform initialization tasks when a call starts and cleanup tasks when a call ends.

#### Overview

Session lifecycle hooks are special SWAIG functions that SignalWire calls automatically:
- `startup_hook`: Called immediately when a new voice session begins
- `hangup_hook`: Called when a voice session ends (regardless of how it ended)

These hooks are particularly useful for:
- Initializing session state or resources
- Loading user preferences or history
- Logging session start/end events
- Cleaning up temporary resources
- Saving session data for analytics

#### Implementation

To implement lifecycle hooks, define them as regular SWAIG functions with these specific names. Because Go handlers are closures, you can capture any external session store (a database, Redis, an in-memory map) that they need:

```go
import (
	"fmt"
	"time"
)

// sessions is any external store you maintain (DB, Redis, guarded map, etc.).
a.DefineTool(agent.ToolDefinition{
	Name:        "startup_hook",
	Description: "Called when the voice session starts",
	Handler: func(args, rawData map[string]any) *swaig.FunctionResult {
		// Extract session information
		callID, _ := rawData["call_id"].(string)
		fromNumber, _ := rawData["from_number"].(string)
		toNumber, _ := rawData["to_number"].(string)

		// Initialize session state
		sessions.Update(callID, map[string]any{
			"session_start":     time.Now().Format(time.RFC3339),
			"from":              fromNumber,
			"to":                toNumber,
			"interaction_count": 0,
		})

		fmt.Printf("Session started: %s from %s\n", callID, fromNumber)

		// Return success (SignalWire expects a response)
		return swaig.NewFunctionResult("Session initialized successfully")
	},
})

a.DefineTool(agent.ToolDefinition{
	Name:        "hangup_hook",
	Description: "Called when the voice session ends",
	Handler: func(args, rawData map[string]any) *swaig.FunctionResult {
		callID, _ := rawData["call_id"].(string)

		// Retrieve session state
		state, ok := sessions.Get(callID)
		if ok {
			// Calculate session duration
			startStr, _ := state["session_start"].(string)
			start, _ := time.Parse(time.RFC3339, startStr)
			duration := time.Since(start).Seconds()

			fmt.Printf("Session ended: %s\n", callID)
			fmt.Printf("Duration: %.0f seconds\n", duration)
			fmt.Printf("Interactions: %v\n", state["interaction_count"])

			// Clean up state
			sessions.Delete(callID)
		}

		return swaig.NewFunctionResult("Session cleanup completed")
	},
})
```

#### Common Use Cases

##### 1. User Preference Loading
```go
a.DefineTool(agent.ToolDefinition{
	Name:        "startup_hook",
	Description: "Called when the voice session starts",
	Handler: func(args, rawData map[string]any) *swaig.FunctionResult {
		callerID, _ := rawData["from_number"].(string)

		// Load user preferences from database
		preferences := loadUserPreferences(callerID)

		// Store in session state for quick access
		callID, _ := rawData["call_id"].(string)
		lang, _ := preferences["language"].(string)
		if lang == "" {
			lang = "en-US"
		}
		sessions.Update(callID, map[string]any{
			"user_preferences": preferences,
			"language":         lang,
			"previous_orders":  preferences["recent_orders"],
		})

		return swaig.NewFunctionResult("User preferences loaded")
	},
})
```

##### 2. Analytics and Logging
```go
a.DefineTool(agent.ToolDefinition{
	Name:        "hangup_hook",
	Description: "Called when the voice session ends",
	Handler: func(args, rawData map[string]any) *swaig.FunctionResult {
		callID, _ := rawData["call_id"].(string)
		state, _ := sessions.Get(callID)

		// Send analytics data
		analyticsData := map[string]any{
			"call_id":          callID,
			"duration":         state["duration"],
			"functions_called": state["functions_called"],
			"outcome":          state["outcome"],
		}

		// Post to analytics service
		sendToAnalytics(analyticsData)

		return swaig.NewFunctionResult("Analytics data sent")
	},
})
```

#### Important Notes

1. **Function Names**: The hooks must be named exactly `startup_hook` and `hangup_hook` for SignalWire to call them
2. **Error Handling**: Always implement proper error handling in hooks - failures shouldn't crash the voice session
3. **Timing**: `startup_hook` is called before the AI starts speaking to the caller
4. **Session Data**: Any data you need to persist across the session should be stored in external storage (Redis, database, etc.)
5. **Return Values**: Both hooks must return a `*swaig.FunctionResult` value

### SIP Routing

SIP routing allows your agents to receive voice calls via SIP addresses. The SDK supports both individual agent-level routing and centralized server-level routing.

#### Individual Agent SIP Routing

Enable SIP routing on a single agent:

```go
// Enable SIP routing with automatic username mapping based on agent name
a.EnableSIPRouting(true, "/sip") // autoMap=true, path="/sip"

// Register additional SIP usernames for this agent
a.RegisterSIPUsername("support_agent")
a.RegisterSIPUsername("help_desk")
```

When `autoMap` is `true`, the agent automatically registers SIP usernames based on:
- The agent's name (e.g., `support@domain`)
- The agent's route path (e.g., `/support` becomes `support@domain`)
- Common variations (e.g., removing vowels for shorter dialing)

`AutoMapSIPUsernames()` performs this auto-mapping explicitly.

#### Server-Level SIP Routing (Multi-Agent)

For multi-agent setups, centralized routing is more efficient. Use `server.NewAgentServer`:

```go
import (
	"github.com/signalwire/signalwire-go/pkg/server"
)

// Create an AgentServer
srv := server.NewAgentServer(
	server.WithServerHost("0.0.0.0"),
	server.WithServerPort(3000),
)

// Register multiple agents
srv.Register(registrationAgent, "/register")
srv.Register(supportAgent, "/support")

// Set up central SIP routing
srv.SetupSIPRouting("/sip", true) // route="/sip", autoMap=true

// Register additional SIP username mappings
srv.RegisterSIPUsername("signup", "/register") // signup@domain → registration agent
srv.RegisterSIPUsername("help", "/support")    // help@domain → support agent
```

With server-level routing:
- Each agent is reachable via its name (when `autoMap` is true)
- Additional SIP usernames can be mapped to specific agent routes
- All SIP routing is handled at a single endpoint (`/sip` by default)

#### How SIP Routing Works

1. A SIP call comes in with a username (e.g., `support@yourdomain`)
2. The SDK extracts the username part (`support`)
3. The system checks if this username is registered:
   - In individual routing: The current agent checks its own username list
   - In server routing: The server checks its central mapping table (via `LookupSIPRoute`)
4. If a match is found, the call is routed to the appropriate agent

### Custom Routing

You can dynamically handle requests to different paths using routing callbacks. A `swml.RoutingCallback` has the signature `func(body map[string]any, headers map[string]any) *string` — return a pointer to a redirect URL, or `nil` to process the request normally:

```go
package main

import (
	"strings"

	"github.com/signalwire/signalwire-go/pkg/agent"
)

func main() {
	a := agent.NewAgentBase()

	// Enable custom routing at construction or anytime after
	a.RegisterRoutingCallback(handleCustomerRoute, "/customer")
	a.RegisterRoutingCallback(handleProductRoute, "/product")
}

// Define the routing handlers
func handleCustomerRoute(body map[string]any, headers map[string]any) *string {
	// Extract any relevant data
	customerID, _ := body["customer_id"].(string)

	// You can redirect to another agent/service if needed
	if strings.HasPrefix(customerID, "vip-") {
		url := "/vip-handler/" + customerID
		return &url
	}

	// Or return nil to process the request with the SWML request hook
	return nil
}

func handleProductRoute(body map[string]any, headers map[string]any) *string { return nil }
```

### Customizing SWML Requests

You can modify the SWML document based on request data by registering an `OnSwmlRequest` hook. The hook has signature `func(requestData map[string]any, callbackPath string, r *http.Request) map[string]any` — return a map of modifications to apply to the document, or `nil` to use the default:

```go
a.SetOnSwmlRequestHook(func(requestData map[string]any, callbackPath string, r *http.Request) map[string]any {
	if callerType, ok := requestData["caller_type"].(string); ok {
		// Example: change the AI behavior based on caller type
		if callerType == "vip" {
			return map[string]any{
				"sections": map[string]any{
					"main": []any{
						// Keep the first verb (answer)
						// Modify the AI verb parameters
						map[string]any{
							"ai": map[string]any{
								"params": map[string]any{
									"wait_for_user":         false,
									"end_of_speech_timeout": 500, // More responsive
								},
							},
						},
					},
				},
			}
		}
	}

	// You can also use callbackPath to serve different content based on the route
	if callbackPath == "/customer" {
		return map[string]any{
			"sections": map[string]any{
				"main": []any{
					map[string]any{"answer": map[string]any{}},
					map[string]any{"play": map[string]any{"url": "say:Welcome to our customer service line."}},
				},
			},
		}
	}

	// Return nil to use the default document
	return nil
})
```

### Conversation Summary Handling

Process conversation summaries by registering an `OnSummary` callback. Its signature is `func(summary map[string]any, rawData map[string]any)`:

```go
a.OnSummary(func(summary map[string]any, rawData map[string]any) {
	if summary != nil {
		// Log the summary
		log.Printf("conversation_summary: %v", summary)

		// Save the summary to a database, send notifications, etc.
		// ...
	}
})
```

### Custom Webhook URLs

You can override the default webhook URLs for SWAIG functions and post-prompt delivery:

```go
// Override the webhook URL for all SWAIG functions
a.SetWebHookURL("https://external-service.example.com/handle-swaig")

// Override the post-prompt delivery URL
a.SetPostPromptURL("https://analytics.example.com/conversation-summaries")

// These methods allow you to:
// 1. Send function calls to external services instead of handling them locally
// 2. Send conversation summaries to analytics services or other systems
// 3. Use special URLs with pre-configured authentication
```

### External Input Checking

The SDK provides a check-for-input endpoint that allows agents to check for new input from external systems. A client can POST to `/check_for_input`:

```go
package main

// Example client code that checks for new input

import (
	"bytes"
	"encoding/json"
	"net/http"
)

// checkForNewInput returns new messages if any, or nil otherwise.
func checkForNewInput(agentURL, conversationID, user, pass string) ([]any, error) {
	url := agentURL + "/check_for_input"
	payload, _ := json.Marshal(map[string]any{"conversation_id": conversationID})

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(user, pass)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var data struct {
			NewInput bool  `json:"new_input"`
			Messages []any `json:"messages"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, err
		}
		if data.NewInput {
			return data.Messages, nil
		}
	}

	return nil, nil
}

func main() { _ = checkForNewInput }
```

By default, the check_for_input endpoint returns an empty response. Enable it with `agent.WithCheckForInputOverride(true)` and customize behavior by supplying your own handler logic in your agent wiring; the endpoint performs basic-auth validation (via `ValidateBasicAuth`) before invoking your logic. A typical implementation:

- validates basic auth; returns HTTP 401 if it fails
- reads `conversation_id` from the POST body or query string; returns HTTP 400 if missing
- looks up new messages for the conversation from your store
- returns a JSON body of the form `{"status": "success", "conversation_id": "...", "new_input": true, "messages": [...]}`

This endpoint is useful for implementing asynchronous conversations where users might send messages through different channels that need to be incorporated into the agent conversation.

## Prefab Agents

Prefab agents are pre-configured agent implementations designed for specific use cases. They provide ready-to-use functionality with customization options, saving development time and ensuring consistent patterns. In Go they live in `github.com/signalwire/signalwire-go/pkg/prefabs` and each is constructed from an options struct.

### Built-in Prefabs

The SDK includes several built-in prefab agents:

#### InfoGathererAgent

Collects structured information from users. Each `Question` has a `KeyName` (where the answer is stored), `QuestionText`, and an optional `Confirm` flag:

```go
package main

import "github.com/signalwire/signalwire-go/pkg/prefabs"

func main() {
	questions := []prefabs.Question{
		{KeyName: "full_name", QuestionText: "What is your full name?"},
		{KeyName: "email", QuestionText: "What is your email address?", Confirm: true},
		{KeyName: "reason", QuestionText: "How can I help you today?"},
	}

	a := prefabs.NewInfoGathererAgent(prefabs.InfoGathererOptions{
		Name:      "info-gatherer",
		Route:     "/info-gatherer",
		Questions: &questions,
	})

	a.Run()
}
```

#### FAQBotAgent

Answers questions based on a set of FAQ entries. Each `FAQ` has a `Question`, `Answer`, and optional `Categories`:

```go
package main

import "github.com/signalwire/signalwire-go/pkg/prefabs"

func main() {
	a := prefabs.NewFAQBotAgent(prefabs.FAQBotOptions{
		Name:    "knowledge-base",
		Route:   "/knowledge-base",
		Persona: "I'm a product documentation assistant.",
		FAQs: []prefabs.FAQ{
			{
				Question: "How do I reset my password?",
				Answer:   "Use the 'Forgot password' link on the sign-in page.",
				Categories: []string{"account"},
			},
		},
	})

	a.Run()
}
```

#### ConciergeAgent

Acts as a virtual concierge for a venue, answering questions about amenities, services, hours, and directions:

```go
package main

import "github.com/signalwire/signalwire-go/pkg/prefabs"

func main() {
	a := prefabs.NewConciergeAgent(prefabs.ConciergeOptions{
		Name:      "concierge",
		Route:     "/concierge",
		VenueName: "Grand Hotel",
		Services:  []string{"room service", "valet parking", "spa booking"},
		Amenities: map[string]prefabs.Amenity{
			"pool": {Hours: "6am-10pm", Location: "3rd floor", Details: "Heated, towels provided"},
			"gym":  {Hours: "24/7", Location: "2nd floor"},
		},
		Hours:          "Front desk staffed 24/7",
		WelcomeMessage: "Welcome to the Grand Hotel. How can I help you today?",
	})

	a.Run()
}
```

#### SurveyAgent

Conducts structured surveys with different question types. Build questions with `NewSurveyQuestion` plus option functions (`WithQuestionID`, `WithQuestionType`, `WithQuestionScale`, `WithQuestionChoices`, `WithOptional`):

```go
package main

import "github.com/signalwire/signalwire-go/pkg/prefabs"

func main() {
	a := prefabs.NewSurveyAgent(prefabs.SurveyOptions{
		Name:       "satisfaction-survey",
		Route:      "/survey",
		SurveyName: "Customer Satisfaction",
		BrandName:  "Acme",
		Questions: []prefabs.SurveyQuestion{
			prefabs.NewSurveyQuestion(
				"How satisfied are you with our product?",
				prefabs.WithQuestionID("satisfaction"),
				prefabs.WithQuestionType("rating"),
				prefabs.WithQuestionScale(5),
			),
			prefabs.NewSurveyQuestion(
				"Do you have any specific feedback about how we can improve?",
				prefabs.WithQuestionID("feedback"),
				prefabs.WithQuestionType("open_ended"),
				prefabs.WithOptional(),
			),
		},
	})

	a.Run()
}
```

#### ReceptionistAgent

Handles call routing and department transfers. Each `Department` has a `Name`, `Description`, `Number`, and a `TransferSWML` flag (true when `Number` is a SWML transfer destination):

```go
package main

import "github.com/signalwire/signalwire-go/pkg/prefabs"

func main() {
	a := prefabs.NewReceptionistAgent(prefabs.ReceptionistOptions{
		Name:  "acme-receptionist",
		Route: "/reception",
		Voice: "rime.spore:mistv2",
		Departments: []prefabs.Department{
			{Name: "sales", Description: "For product inquiries and pricing", Number: "+15551235555"},
			{Name: "support", Description: "For technical assistance", Number: "+15551236666"},
			{Name: "billing", Description: "For payment and invoice questions", Number: "+15551237777"},
		},
		Greeting: "Thank you for calling ACME Corp. How may I direct your call?",
	})

	a.Run()
}
```

### Creating Your Own Prefabs

You can create your own prefab agents in Go by writing a constructor function that builds and configures an `*agent.AgentBase` (optionally embedding it in a struct to add methods and per-prefab state). The built-in prefabs follow this pattern: each embeds `*agent.AgentBase` and applies its options in its `New...` constructor.

#### Basic Prefab Structure

A well-designed prefab should:

1. Embed `*agent.AgentBase` (or build on another prefab)
2. Take configuration parameters in an options struct
3. Apply configuration to set up the agent
4. Provide appropriate default values
5. Include domain-specific tools

Example of a custom support agent prefab:

```go
package supportprefab

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// CustomerSupportOptions configures a CustomerSupportAgent.
type CustomerSupportOptions struct {
	ProductName    string
	SupportEmail   string
	EscalationPath string
	AgentOptions   []agent.AgentOption // forwarded to NewAgentBase (name, route, etc.)
}

// CustomerSupportAgent is a prefab that embeds AgentBase.
type CustomerSupportAgent struct {
	*agent.AgentBase
}

func NewCustomerSupportAgent(opts CustomerSupportOptions) *CustomerSupportAgent {
	base := agent.NewAgentBase(opts.AgentOptions...)
	a := &CustomerSupportAgent{AgentBase: base}

	// Configure prompt
	a.PromptAddSection("Personality",
		fmt.Sprintf("I am a customer support agent for %s.", opts.ProductName), nil)
	a.PromptAddSection("Goal", "Help customers solve their problems effectively.", nil)

	// Standard instructions (conditionally include escalation)
	instructions := []string{
		"Be professional but friendly.",
		"Verify the customer's identity before sharing account details.",
	}
	if opts.EscalationPath != "" {
		instructions = append(instructions,
			fmt.Sprintf("For complex issues, offer to escalate to %s.", opts.EscalationPath))
	}
	a.PromptAddSection("Instructions", "", instructions)

	// Register default tools
	a.DefineTool(agent.ToolDefinition{
		Name:        "escalate_issue",
		Description: "Escalate a customer issue to a human agent",
		Parameters: map[string]any{
			"issue_summary":  map[string]any{"type": "string", "description": "Brief summary of the issue"},
			"customer_email": map[string]any{"type": "string", "description": "Customer's email address"},
		},
		Handler: func(args, rawData map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("Issue escalated successfully.")
		},
	})

	a.DefineTool(agent.ToolDefinition{
		Name:        "send_support_email",
		Description: "Send a follow-up email to the customer",
		Parameters: map[string]any{
			"customer_email":   map[string]any{"type": "string"},
			"issue_summary":    map[string]any{"type": "string"},
			"resolution_steps": map[string]any{"type": "string"},
		},
		Handler: func(args, rawData map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("Follow-up email sent successfully.")
		},
	})

	return a
}
```

#### Using the Custom Prefab

<!-- snippet: no-compile references the supportprefab package + agent from the separate custom-prefab definition above -->
```go
supportAgent := supportprefab.NewCustomerSupportAgent(supportprefab.CustomerSupportOptions{
	ProductName:    "SignalWire Voice API",
	SupportEmail:   "support@example.com",
	EscalationPath: "tier 2 support",
	AgentOptions: []agent.AgentOption{
		agent.WithName("voice-support"),
		agent.WithRoute("/voice-support"),
	},
})

// Start the agent
supportAgent.Run()
```

#### Customizing Existing Prefabs

You can also wrap a built-in prefab to customize it — embed it and add configuration in your constructor:

```go
package enhancedgatherer

import (
	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/prefabs"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

type EnhancedGatherer struct {
	*prefabs.InfoGathererAgent
}

func NewEnhancedGatherer(opts prefabs.InfoGathererOptions) *EnhancedGatherer {
	base := prefabs.NewInfoGathererAgent(opts)
	g := &EnhancedGatherer{InfoGathererAgent: base}

	// Add an additional instruction
	g.PromptAddSection("Instructions", "", []string{"Verify all information carefully."})

	// Add an additional custom tool
	g.DefineTool(agent.ToolDefinition{
		Name:        "check_customer",
		Description: "Check customer status in database",
		Parameters: map[string]any{
			"email": map[string]any{"type": "string"},
		},
		Handler: func(args, rawData map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("Customer status: Active")
		},
	})

	return g
}
```

### Best Practices for Prefab Design

1. **Clear Documentation**: Document the purpose, parameters, and extension points
2. **Sensible Defaults**: Provide working defaults that make sense for the use case
3. **Error Handling**: Implement robust error handling with helpful messages
4. **Modular Design**: Keep prefabs focused on a specific use case
5. **Consistent Interface**: Maintain consistent patterns across related prefabs
6. **Extension Points**: Provide clear ways for others to extend your prefab
7. **Configuration Options**: Make all key behaviors configurable

### Making Prefabs Distributable

To create distributable prefabs that can be used across multiple projects:

1. **Package Structure**: Put your prefab in its own importable Go package
2. **Documentation**: Include clear usage examples in package doc comments
3. **Configuration**: Support both code and file-based configuration
4. **Testing**: Include tests for your prefab
5. **Publishing**: Publish it as a Go module so others can `go get` it

Example package structure:

```text
my-prefab-agents/
├── README.md
├── go.mod
├── examples/
│   └── support_agent_example/
│       └── main.go
├── support/
│   └── support.go
├── retail/
│   └── retail.go
└── internal/
    └── knowledgebase/
        └── knowledgebase.go
```

## API Reference

### Constructor Options (`agent.NewAgentBase(opts ...agent.AgentOption)`)

- `agent.WithName(name)`: Agent name/identifier
- `agent.WithRoute(route)`: HTTP route path (default: "/")
- `agent.WithHost(host)`: Host to bind to (default: "0.0.0.0")
- `agent.WithPort(port)`: Port to bind to (default: 3000)
- `agent.WithBasicAuth(user, password)`: Basic-auth credentials
- `agent.WithUsePom(usePom)`: Whether to use POM for prompts (default: true)
- `agent.WithTokenExpiry(secs)`: Security token expiry time (default: 3600)
- `agent.WithAutoAnswer(autoAnswer)`: Auto-answer calls
- `agent.WithRecordCall(record)`: Record calls
- `agent.WithSchemaPath(path)`: Optional path to schema.json file
- `agent.WithSuppressLogs(suppress)`: Whether to suppress structured logs (default: false)
- `agent.WithSigningKey(key)`: Signing key for tool tokens / signed webhooks

### Prompt Methods

- `PromptAddSection(title, body, bullets, opts...)` (options: `agent.WithNumbered`, `agent.WithNumberedBullets`, `agent.WithSubsections`)
- `PromptAddSubsection(parentTitle, title, body, bullets)`
- `PromptAddToSection(title, body, opts...)` (options: `agent.WithBullet`, `agent.WithBullets`)
- `SetPromptText(text)`
- `SetPromptPom(pom)`
- `SetPostPrompt(text)`

### SWAIG Methods

- `DefineTool(agent.ToolDefinition{Name, Description, Parameters, Required, Handler, Secure, Fillers, WebhookURL, ...})`
- `RegisterSwaigFunction(funcDef map[string]any)`
- `SetNativeFunctions(names []string)`
- `AddFunctionInclude(url, functions, metaData)`
- `SetFunctionIncludes(includes []map[string]any)`

### Configuration Methods

- `AddHint(hint)` and `AddHints(hints)`
- `AddPatternHint(hint, pattern, replace, ignoreCase...)`
- `AddPronunciation(replace, withText, ignoreCase...)`
- `AddLanguage(config map[string]any)` and `AddLanguageTyped(name, code, voice, speechFillers, functionFillers, engine, model, params...)`
- `SetParam(key, value)` and `SetParams(params map[string]any)`
- `SetGlobalData(data)` and `UpdateGlobalData(data)`

### State Methods

The Go SDK does not maintain per-call conversation state inside `AgentBase`. Persist any session state you need in your own store (a database, Redis, or a mutex-guarded in-memory map) keyed by `call_id`, and access it from your SWAIG handlers via the `rawData["call_id"]` value. See [Session Lifecycle Hooks](#session-lifecycle-hooks) for the `startup_hook` / `hangup_hook` pattern.

### SIP Routing Methods

- `EnableSIPRouting(autoMap, path)`: Enable SIP routing for an agent
- `RegisterSIPUsername(sipUsername)`: Register a SIP username for an agent
- `AutoMapSIPUsernames()`: Automatically register SIP usernames based on agent attributes

#### AgentServer SIP Methods

- `SetupSIPRouting(route, autoMap)`: Set up central SIP routing for a server
- `RegisterSIPUsername(username, route)`: Map a SIP username to an agent route
- `LookupSIPRoute(username)`: Resolve a SIP username to its agent route

### Service Methods

- `Run()`: Auto-detect the environment and start serving
- `RunWithMode(mode)` / `RunContext(ctx)`: Start serving with an explicit mode / context
- `Serve()`: Start the web server
- `AsRouter()`: Return an `http.Handler` for this agent
- `SetOnSwmlRequestHook(hook)`: Customize SWML based on request data and path
- `OnSummary(cb)`: Handle post-prompt summaries
- `OnFunctionCall(name, args, rawData)`: Process SWAIG function calls
- `RegisterRoutingCallback(callbackFn, path)`: Register a callback for custom path routing
- `SetWebHookURL(url)`: Override the default web hook URL
- `SetPostPromptURL(url)`: Override the default post-prompt URL

### Endpoint Methods

The SDK provides several endpoints for different purposes:

- Root endpoint (`/`): Serves the main SWML document
- SWAIG endpoint (`/swaig`): Handles SWAIG function calls
- Post-prompt endpoint (`/post_prompt`): Processes conversation summaries
- Check-for-input endpoint (`/check_for_input`): Supports checking for new input from external systems
- Debug endpoint (`/debug`): Serves the SWML document with debug headers
- Debug events endpoint (`/debug_events`): Receives real-time debug events from the AI module (see [Debug Events](#debug-events))
- SIP routing endpoint (configurable, default `/sip`): Handles SIP routing requests

## Testing

The SignalWire AI Agent SDK provides testing capabilities through the `swaig-test`
CLI tool. Unlike the Python tool (which loads an agent source file), the Go
`swaig-test` drives a **running** agent over HTTP via `--url` — so start the agent
in one terminal, then point the CLI at its URL. Function arguments are passed with
repeatable `--param key=value` flags.

### Local Agent Testing

Start the agent, then test it over HTTP:

```bash
# Terminal 1: run the agent (serves on whatever host:port/route it configures)
go run ./cmd/my_agent

# Terminal 2: exercise it over HTTP
# List available functions
swaig-test --url http://localhost:3000/ --list-tools

# Test a SWAIG function (pass args with --param key=value)
swaig-test --url http://localhost:3000/ --exec get_weather --param location="New York"

# Generate the SWML document
swaig-test --url http://localhost:3000/ --dump-swml
```

### Serverless Environment Simulation

The Go CLI implements Lambda simulation only. `--simulate-serverless lambda`
applies the Lambda mode-detection env vars around the invocation (and clears
`SWML_PROXY_URL_BASE`) so platform-specific URL generation is exercised. It
requires `--url` — Go agents are compiled binaries, so the CLI simulates by
running the live agent URL with the platform env applied. Other platforms
(CGI / Cloud Functions / Azure) are not implemented by the Go CLI.

```bash
# Lambda environment simulation while dumping SWML
swaig-test --url http://localhost:3000/ --simulate-serverless lambda --dump-swml

# Function execution in a simulated Lambda context
swaig-test --url http://localhost:3000/ --simulate-serverless lambda \
  --exec get_weather --param location="Miami"
```

For true in-process adapter dispatch (no running server), call
`SimulateDumpSWMLViaLambda` / `SimulateExecToolViaLambda` from `package main`
directly — see `cmd/swaig-test/simulate.go`.

### Inspecting a Compiled Example

To introspect a compiled example binary's tool registry without HTTP, use
`--example NAME` (list-tools only):

```bash
swaig-test --example my_example --list-tools
```

### Testing Best Practices

1. **Run the agent first**: the CLI drives a live server over `--url`.
2. **Pass args as `--param key=value`**: repeat the flag for multiple arguments.
3. **Use `--raw`**: emit compact JSON (e.g. to pipe into `jq`).
4. **Use `--verbose`**: show request/response details for debugging.

For more detailed testing documentation, see the [CLI Guide](cli_guide.md).

## Examples

### Simple Question-Answering Agent

```go
package main

import (
	"fmt"
	"time"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

func newSimpleAgent() *agent.AgentBase {
	a := agent.NewAgentBase(
		agent.WithName("simple"),
		agent.WithRoute("/simple"),
		agent.WithUsePom(true),
	)

	// Configure agent personality
	a.PromptAddSection("Personality", "You are a friendly and helpful assistant.", nil)
	a.PromptAddSection("Goal", "Help users with basic tasks and answer questions.", nil)
	a.PromptAddSection("Instructions", "", []string{
		"Be concise and direct in your responses.",
		"If you don't know something, say so clearly.",
		"Use the get_time function when asked about the current time.",
	})

	a.DefineTool(agent.ToolDefinition{
		Name:        "get_time",
		Description: "Get the current time",
		Handler: func(args, rawData map[string]any) *swaig.FunctionResult {
			now := time.Now()
			formattedTime := now.Format("15:04:05")
			return swaig.NewFunctionResult(fmt.Sprintf("The current time is %s", formattedTime))
		},
	})

	return a
}

func main() {
	a := newSimpleAgent()
	fmt.Println("Starting agent server...")
	fmt.Println("Note: Works in any deployment mode (server/CGI/Lambda)")
	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
```

### Multi-Language Customer Service Agent

```go
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

func newCustomerServiceAgent() *agent.AgentBase {
	a := agent.NewAgentBase(
		agent.WithName("customer-service"),
		agent.WithRoute("/support"),
		agent.WithUsePom(true),
	)

	// Configure agent personality
	a.PromptAddSection("Personality",
		"You are a helpful customer service representative for SignalWire.", nil)
	a.PromptAddSection("Knowledge",
		"You can answer questions about SignalWire products and services.", nil)
	a.PromptAddSection("Instructions", "", []string{
		"Greet customers politely",
		"Answer questions about SignalWire products",
		"Use check_account_status when customer asks about their account",
		"Use create_support_ticket for unresolved issues",
	})

	// Add language support
	a.AddLanguageTyped(
		"English",
		"en-US",
		"en-US-Neural2-F",
		[]string{"Let me think...", "One moment please..."},
		[]string{"I'm looking that up...", "Let me check that..."},
		"", "",
	)
	a.AddLanguage(map[string]any{
		"name":           "Spanish",
		"code":           "es",
		"voice":          "rime.spore:multilingual",
		"speech_fillers": []string{"Un momento por favor...", "Estoy pensando..."},
	})

	// Enable languages
	a.SetParams(map[string]any{"languages_enabled": true})

	// Add company information
	a.SetGlobalData(map[string]any{
		"company_name":  "SignalWire",
		"support_hours": "9am-5pm ET, Monday through Friday",
		"support_email": "support@signalwire.com",
	})

	a.DefineTool(agent.ToolDefinition{
		Name:        "check_account_status",
		Description: "Check the status of a customer's account",
		Parameters: map[string]any{
			"account_id": map[string]any{
				"type":        "string",
				"description": "The customer's account ID",
			},
		},
		Handler: func(args, rawData map[string]any) *swaig.FunctionResult {
			accountID, _ := args["account_id"].(string)
			// In a real implementation, this would query a database
			return swaig.NewFunctionResult(fmt.Sprintf("Account %s is in good standing.", accountID))
		},
	})

	a.DefineTool(agent.ToolDefinition{
		Name:        "create_support_ticket",
		Description: "Create a support ticket for an unresolved issue",
		Parameters: map[string]any{
			"issue": map[string]any{
				"type":        "string",
				"description": "Brief description of the issue",
			},
			"priority": map[string]any{
				"type":        "string",
				"description": "Ticket priority",
				"enum":        []string{"low", "medium", "high", "critical"},
			},
		},
		Handler: func(args, rawData map[string]any) *swaig.FunctionResult {
			issue, _ := args["issue"].(string)
			priority, _ := args["priority"].(string)
			if priority == "" {
				priority = "medium"
			}

			// Generate a ticket ID (in a real system, this would create a database entry)
			ticketID := fmt.Sprintf("TICKET-%04d", len(issue)%10000)

			return swaig.NewFunctionResult(fmt.Sprintf(
				"Support ticket %s has been created with %s priority. "+
					"A support representative will contact you shortly.", ticketID, priority))
		},
	})

	return a
}

func main() {
	a := newCustomerServiceAgent()
	fmt.Println("Starting customer service agent...")
	fmt.Println("Note: Works in any deployment mode (server/CGI/Lambda)")
	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
```

### Dynamic Agent Configuration Examples

For working examples of dynamic agent configuration, see these directories under `examples/`:

- **`simple_static/`**: Traditional static configuration approach
- **`comprehensive_dynamic/`**: Advanced multi-tier, multi-industry dynamic agent
- **`simple_agent/`**: Basic AI agent with prompt and tools
- **`swaig_features/`**: The full range of `FunctionResult` call-control actions

These examples demonstrate the progression from static to dynamic configuration and show real-world use cases like multi-tenant applications, A/B testing, and personalization.

For more examples, see the `examples` directory in the SignalWire AI Agent SDK repository.
