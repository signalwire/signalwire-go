# SignalWire AI Agents SDK - Complete API Reference

This document provides a comprehensive reference for all public APIs in the SignalWire AI Agents SDK for Go.

Add the SDK to your module with `go get github.com/signalwire/signalwire-go`, then import the packages you need (`pkg/agent`, `pkg/swaig`, `pkg/datamap`, `pkg/contexts`, `pkg/skills`, `pkg/prefabs`, `pkg/server`).

## Table of Contents

1. [AgentBase](#agentbase) - Core agent functionality
2. [FunctionResult](#functionresult) - SWAIG (SignalWire AI Gateway) function response handling
3. [DataMap](#datamap) - Serverless API tools that execute on SignalWire's servers
4. [Context System](#context-system) - Structured workflows
5. [State Management](#state-management) - Persistent state
6. [Skills System](#skills-system) - Modular capabilities
7. [Utility Types](#utility-types) - Supporting types

---

## AgentBase

The `agent.AgentBase` struct is the foundation for creating AI agents. It builds on `SWMLService` (the layer for generating SWML -- SignalWire Markup Language -- documents) and provides comprehensive functionality for building conversational AI agents.

Unlike the Python SDK, the Go SDK does not use subclassing. You construct an agent with functional options and configure it with fluent methods (each returns the agent pointer for chaining).

### Constructor

```go
func NewAgentBase(opts ...AgentOption) *AgentBase
```

**Options** (each is an `agent.AgentOption`):

- `WithName(name string)`: Human-readable name for the agent
- `WithRoute(route string)`: HTTP route path for the agent (default: "/")
- `WithHost(host string)`: Host address to bind to (default: "0.0.0.0")
- `WithPort(port int)`: Port number to listen on (default: 3000)
- `WithBasicAuth(user, password string)`: Username/password for HTTP basic auth
- `WithUsePom(usePom bool)`: Whether to use Prompt Object Model (default: true)
- `WithTokenExpiry(secs int)`: Security token expiration time (default: 3600)
- `WithAutoAnswer(autoAnswer bool)`: Automatically answer incoming calls (default: true)
- `WithRecordCall(record bool)`: Record calls by default (default: false)
- `WithRecordFormat(format swaig.RecordFormat)`: Recording format: `swaig.FormatMP4`, `swaig.FormatWAV`, `swaig.FormatMP3` (default: mp4)
- `WithRecordStereo(stereo bool)`: Record in stereo (default: true)
- `WithDefaultWebhookURL(url string)`: Default webhook URL for functions
- `WithAgentID(id string)`: Unique identifier for the agent
- `WithNativeFunctions(names []string)`: Native function names to enable
- `WithSchemaPath(path string)`: Path to custom SWML schema file
- `WithSuppressLogs(suppress bool)`: Suppress logging output (default: false)
- `WithEnablePostPromptOverride(enable bool)`: Allow post-prompt URL override (default: false)
- `WithCheckForInputOverride(enable bool)`: Allow check-for-input URL override (default: false)
- `WithConfigFile(path string)`: Path to JSON configuration file. See [Configuration Guide](configuration.md) for details.

```go
a := agent.NewAgentBase(
	agent.WithName("my-agent"),
	agent.WithRoute("/agent"),
	agent.WithPort(3000),
	agent.WithUsePom(true),
)
```

### Core Methods

#### Deployment and Execution

##### `Run() error`
Auto-detects the deployment environment and runs the agent appropriately. Related methods: `RunWithMode(mode swml.ExecutionMode) error` forces a specific mode, `RunContext(ctx context.Context) error` runs with cancellation, and `DetectRunMode() swml.ExecutionMode` reports the mode `Run` would select.

```go
// Auto-detect environment
if err := a.Run(); err != nil {
	log.Fatal(err)
}

// Force server mode
if err := a.RunWithMode(swml.ExecutionModeServer); err != nil {
	log.Fatal(err)
}
```

##### `Serve() error`
Explicitly run as an HTTP server using the standard library `net/http`.

```go
if err := a.Serve(); err != nil {
	log.Fatal(err)
}
```

### Prompt Configuration

#### Text-Based Prompts

##### `SetPromptText(text string) *AgentBase`
Set the agent's prompt as raw text.

```go
a.SetPromptText("You are a helpful customer service agent.")
```

##### `SetPostPrompt(text string) *AgentBase`
Set additional text to append after the main prompt.

```go
a.SetPostPrompt("Always be polite and professional.")
```

#### LLM Parameter Configuration

##### `SetPromptLlmParams(params map[string]any) *AgentBase`
Set Language Model parameters for the main prompt. Any parameters are passed through to the SignalWire server, which validates and applies them based on the target model's capabilities.

**Common Parameters:**
- `temperature`: Controls randomness. Lower = more focused
- `top_p`: Nucleus sampling threshold
- `barge_confidence`: ASR confidence to interrupt
- `presence_penalty`: Topic diversity control
- `frequency_penalty`: Repetition control

Note: No defaults are sent unless explicitly set. Invalid parameters for the selected model are handled/ignored by the server.

```go
// Configure for consistent, professional responses
a.SetPromptLlmParams(map[string]any{
	"temperature":       0.3,
	"top_p":             0.9,
	"barge_confidence":  0.7,
	"presence_penalty":  0.1,
	"frequency_penalty": 0.2,
})
```

##### `SetPostPromptLlmParams(params map[string]any) *AgentBase`
Set Language Model parameters for the post-prompt. Note: `barge_confidence` is not applicable to post-prompt.

```go
// Configure for focused summaries
a.SetPostPromptLlmParams(map[string]any{
	"temperature": 0.2,
	"top_p":       0.9,
})
```

#### Structured Prompts (POM)

##### `PromptAddSection(title, body string, bullets []string, opts ...PromptSectionOption) *AgentBase`
Add a structured section to the prompt using the Prompt Object Model. Pass `""` for no body and `nil` for no bullets. Options: `agent.WithNumbered(bool)`, `agent.WithNumberedBullets(bool)`, `agent.WithSubsections([]map[string]any)`.

```go
// Simple section
a.PromptAddSection("Role", "You are a customer service representative.", nil)

// Section with bullets
a.PromptAddSection("Guidelines", "Follow these principles:",
	[]string{"Be helpful", "Stay professional", "Listen carefully"})

// Numbered bullets
a.PromptAddSection("Process", "Follow these steps:",
	[]string{"Greet the customer", "Identify their need", "Provide solution"},
	agent.WithNumberedBullets(true))
```

##### `PromptAddToSection(title, body string, opts ...PromptAddToSectionOption) *AgentBase`
Add content to an existing prompt section. Options: `agent.WithBullet(string)` to add a single bullet, `agent.WithBullets([]string)` to add several.

```go
// Add body text to existing section
a.PromptAddToSection("Guidelines", "Remember to always verify customer identity.")

// Add single bullet
a.PromptAddToSection("Process", "", agent.WithBullet("Document the interaction"))

// Add multiple bullets
a.PromptAddToSection("Process", "", agent.WithBullets([]string{"Follow up", "Close ticket"}))
```

##### `PromptAddSubsection(parentTitle, title, body string, bullets []string) *AgentBase`
Add a subsection to an existing prompt section.

```go
a.PromptAddSubsection(
	"Guidelines",
	"Escalation Rules",
	"Escalate when:",
	[]string{"Customer is angry", "Technical issue beyond scope"},
)
```

### Voice and Language Configuration

##### `AddLanguage(config map[string]any) *AgentBase`
Configure a voice/language using a config map (keys: `name`, `code`, `voice`, `speech_fillers`, `function_fillers`, `engine`, `model`).

```go
a.AddLanguage(map[string]any{"name": "English", "code": "en-US", "voice": "rime.spore"})
```

##### `AddLanguageTyped(name, code, voice string, speechFillers, functionFillers []string, engine, model string, params ...map[string]any) *AgentBase`
Typed variant of `AddLanguage`. Pass `nil`/`""` for unused fields.

```go
a.AddLanguageTyped(
	"English", "en-US", "nova.luna",
	[]string{"Let me think...", "One moment..."},   // speech fillers
	[]string{"Processing...", "Looking that up..."}, // function fillers
	"", "",
)
```

##### `SetLanguages(languages []map[string]any) *AgentBase`
Set multiple language configurations at once.

```go
a.SetLanguages([]map[string]any{
	{"name": "English", "code": "en-US", "voice": "rime.spore"},
	{"name": "Spanish", "code": "es-ES", "voice": "nova.luna"},
})
```

### Speech Recognition Configuration

##### `AddHint(hint string) *AgentBase`
Add a single speech recognition hint.

```go
a.AddHint("SignalWire")
```

##### `AddHints(hints []string) *AgentBase`
Add multiple speech recognition hints.

```go
a.AddHints([]string{"SignalWire", "SWML", "API", "webhook", "SIP"})
```

##### `AddPatternHint(hint, pattern, replace string, ignoreCase ...bool) *AgentBase`
Add a pattern-based hint for speech recognition.

```go
a.AddPatternHint(
	"phone number",
	`(\d{3})-(\d{3})-(\d{4})`,
	`($1) $2-$3`,
)
```

##### `AddPronunciation(replace, withText string, ignoreCase ...bool) *AgentBase`
Add pronunciation rules for text-to-speech.

```go
a.AddPronunciation("API", "A P I")
a.AddPronunciation("SWML", "swim-el")
```

##### `SetPronunciations(p []map[string]any) *AgentBase`
Set multiple pronunciation rules at once.

```go
a.SetPronunciations([]map[string]any{
	{"replace": "API", "with": "A P I"},
	{"replace": "SWML", "with": "swim-el", "ignore_case": true},
})
```

### AI Parameters Configuration

##### `SetParam(key string, value any) *AgentBase`
Set a single AI parameter.

```go
a.SetParam("ai_model", "gpt-4.1-nano")
a.SetParam("end_of_speech_timeout", 500)
```

##### `SetParams(params map[string]any) *AgentBase`
Set multiple AI parameters at once.

**Common Parameters:**
- `ai_model`: AI model to use ("gpt-4.1-nano", "gpt-4.1-mini", etc.)
- `end_of_speech_timeout`: Milliseconds to wait for speech end (default: 1000)
- `attention_timeout`: Milliseconds before attention timeout (default: 30000)
- `background_file_volume`: Volume for background audio (-60 to 0 dB)
- `temperature`: AI creativity/randomness (0.0 to 2.0)
- `max_tokens`: Maximum response length
- `top_p`: Nucleus sampling parameter (0.0 to 1.0)

```go
a.SetParams(map[string]any{
	"ai_model":               "gpt-4.1-nano",
	"end_of_speech_timeout":  500,
	"attention_timeout":      15000,
	"background_file_volume": -20,
	"temperature":            0.7,
})
```

### Global Data Management

##### `SetGlobalData(data map[string]any) *AgentBase`
Set global data available to the AI and functions.

```go
a.SetGlobalData(map[string]any{
	"company_name":      "Acme Corp",
	"support_hours":     "9 AM - 5 PM EST",
	"escalation_number": "+1-555-0123",
})
```

##### `UpdateGlobalData(data map[string]any) *AgentBase`
Update existing global data (merge with existing).

```go
a.UpdateGlobalData(map[string]any{
	"current_promotion": "20% off all services",
	"promotion_expires": "2024-12-31",
})
```

### Function Definition

##### `DefineTool(def ToolDefinition) *AgentBase`
Define a custom SWAIG function/tool. The `ToolDefinition` struct fields are:

```go
type ToolDefinition struct {
	Name           string
	Description    string
	Parameters     map[string]any // JSON Schema properties map
	Required       []string       // required parameter names
	Handler        ToolHandler    // func(args, rawData map[string]any) *swaig.FunctionResult
	Secure         bool           // require security token (default behavior: secure)
	Fillers        map[string][]string
	WaitFile       string
	WaitFileLoops  int
	WebhookURL     string // per-tool external webhook; overrides the agent webhook
	MetaData       map[string]any
	SwaigFields    map[string]any
	IsTypedHandler bool
}
```

```go
a.DefineTool(agent.ToolDefinition{
	Name:        "get_weather",
	Description: "Get current weather for a location",
	Parameters: map[string]any{
		"location": map[string]any{
			"type":        "string",
			"description": "City name",
		},
	},
	Required: []string{"location"},
	Fillers:  map[string][]string{"en-US": {"Checking weather...", "Looking up forecast..."}},
	Handler: func(args, rawData map[string]any) *swaig.FunctionResult {
		location, _ := args["location"].(string)
		if location == "" {
			location = "Unknown"
		}
		return swaig.NewFunctionResult(fmt.Sprintf("The weather in %s is sunny and 75°F", location))
	},
})
```

##### `RegisterSwaigFunction(funcDef map[string]any) *AgentBase`
Register a pre-built SWAIG function definition (for example, one produced by `DataMap.ToSwaigFunction()`).

```go
// Register a DataMap tool
weatherTool := datamap.New("get_weather").
	Webhook("GET", "https://api.weather.com/...", nil, "", false, nil)
a.RegisterSwaigFunction(weatherTool.ToSwaigFunction())
```

### Session Lifecycle Hooks

SignalWire AI agents support special SWAIG functions that are automatically called at specific points in the conversation lifecycle. Define them as regular tools whose names are exactly `startup_hook` and `hangup_hook`.

##### `startup_hook`
Called when a new conversation/call begins.

```go
a.DefineTool(agent.ToolDefinition{
	Name:        "startup_hook",
	Description: "Called when a new conversation starts to initialize state",
	Handler: func(args, rawData map[string]any) *swaig.FunctionResult {
		callID, _ := rawData["call_id"].(string)
		_ = callID
		// Initialize session resources, load user data, etc.
		return swaig.NewFunctionResult("Session initialized")
	},
})
```

##### `hangup_hook`
Called when a conversation/call ends.

```go
a.DefineTool(agent.ToolDefinition{
	Name:        "hangup_hook",
	Description: "Called when conversation ends to clean up resources",
	Handler: func(args, rawData map[string]any) *swaig.FunctionResult {
		callID, _ := rawData["call_id"].(string)
		_ = callID
		// Clean up resources, save session data, etc.
		return swaig.NewFunctionResult("Session ended")
	},
})
```

**Common Use Cases:**
- Loading user preferences at session start
- Initializing session-specific resources
- Logging conversation metrics
- Cleaning up temporary data
- Saving conversation summaries

### Skills System

##### `AddSkill(skillName skills.SkillName, params map[string]any) *AgentBase`
Add a modular skill to the agent. Skills are named by typed `skills.SkillName` constants.

**Available Skills:**
- `skills.SkillDatetime`: Current date/time information
- `skills.SkillMath`: Mathematical calculations
- `skills.SkillWebSearch`: Google Custom Search integration
- `skills.SkillDatasphere`: SignalWire DataSphere search
- `skills.SkillNativeVectorSearch`: Remote document search via a search server

```go
// Simple skill (nil params for defaults)
a.AddSkill(skills.SkillDatetime, nil)
a.AddSkill(skills.SkillMath, nil)

// Skill with configuration
a.AddSkill(skills.SkillWebSearch, map[string]any{
	"api_key":          "your-google-api-key",
	"search_engine_id": "your-search-engine-id",
	"num_results":      3,
})

// Multiple instances with different names
a.AddSkill(skills.SkillWebSearch, map[string]any{
	"api_key":          "your-api-key",
	"search_engine_id": "general-engine",
	"tool_name":        "search_general",
})
a.AddSkill(skills.SkillWebSearch, map[string]any{
	"api_key":          "your-api-key",
	"search_engine_id": "news-engine",
	"tool_name":        "search_news",
})
```

##### `RemoveSkill(skillName skills.SkillName) *AgentBase`
Remove a skill from the agent.

```go
a.RemoveSkill(skills.SkillWebSearch)
```

##### `ListSkills() []string`
Get the list of currently added skills.

```go
activeSkills := a.ListSkills()
fmt.Printf("Active skills: %v\n", activeSkills)
```

##### `HasSkill(skillName skills.SkillName) bool`
Check whether a skill is currently added.

```go
if a.HasSkill(skills.SkillWebSearch) {
	fmt.Println("Web search is available")
}
```

### Native Functions

##### `SetNativeFunctions(names []string) *AgentBase`
Enable specific native SWML functions.

**Available Native Functions:** `transfer`, `hangup`, `play`, `record`, `send_sms`

```go
a.SetNativeFunctions([]string{"transfer", "hangup", "send_sms"})
```

##### `SetInternalFillers(fillers map[string]map[string][]string) *AgentBase`
Set custom filler phrases for internal/native SWAIG functions (function name → language code → phrases).

**Available Internal Functions:** `next_step`, `change_context`, `check_time`, `wait_for_user`, `wait_seconds`, `get_visual_input`

```go
a.SetInternalFillers(map[string]map[string][]string{
	"next_step": {
		"en-US": {"Moving to the next step...", "Let's continue..."},
		"es":    {"Pasando al siguiente paso...", "Continuemos..."},
	},
	"check_time": {
		"en-US": {"Let me check the time...", "Getting current time..."},
	},
})
```

##### `AddInternalFiller(funcName, langCode string, fillers []string) *AgentBase`
Add internal fillers for a specific function and language.

```go
a.AddInternalFiller("next_step", "en-US", []string{
	"Great! Let's move to the next step...",
	"Perfect! Moving forward...",
})
```

### Function Includes

##### `AddFunctionInclude(url string, functions []string, metaData map[string]any) *AgentBase`
Include external SWAIG functions from another service.

```go
a.AddFunctionInclude(
	"https://external-service.com/swaig",
	[]string{"external_function1", "external_function2"},
	map[string]any{"service": "external", "version": "1.0"},
)
```

##### `SetFunctionIncludes(includes []map[string]any) *AgentBase`
Set multiple function includes at once.

```go
a.SetFunctionIncludes([]map[string]any{
	{"url": "https://service1.com/swaig", "functions": []string{"func1", "func2"}},
	{"url": "https://service2.com/swaig", "functions": []string{"func3"}, "meta_data": map[string]any{"priority": "high"}},
})
```

### Webhook Configuration

##### `SetWebHookURL(url string) *AgentBase`
Set the default webhook URL for SWAIG functions.

```go
a.SetWebHookURL("https://myserver.com/webhook")
```

##### `SetPostPromptURL(url string) *AgentBase`
Set the URL for post-prompt processing.

```go
a.SetPostPromptURL("https://myserver.com/post-prompt")
```

##### `AddSwaigQueryParams(params map[string]string) *AgentBase`
Add query parameters to be included in all SWAIG webhook URLs. Useful for preserving dynamic-configuration state across SWAIG callbacks.

```go
// In a dynamic config callback, preserve configuration parameters
a.SetDynamicConfigCallback(func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
	customerID := queryParams["customer_id"]
	if customerID != "" {
		// Pass through to SWAIG callbacks
		ep.AddSwaigQueryParams(map[string]string{"customer_id": customerID})
		ep.AddSkill(skills.SkillName("customer_lookup"), map[string]any{"customer_id": customerID})
	}
})
```

##### `ClearSwaigQueryParams() *AgentBase`
Clear all SWAIG query parameters.

```go
a.ClearSwaigQueryParams()
```

### Debug Events

##### `EnableDebugEvents(level int) *AgentBase`
Enable the debug event webhook for this agent. When enabled, the AI module POSTs real-time debug events to a `/debug_events` endpoint on this agent during calls. Events are automatically logged via the agent's structured logger and can optionally be handled with a custom callback via `OnDebugEvent`.

**Parameters:**
- `level` (int): Debug event verbosity level. `1` = high-level events (barge, errors, session start/end, step changes). `2+` = adds high-volume events (every LLM request/response, conversation_add).

```go
a.EnableDebugEvents(1) // level 1
a.EnableDebugEvents(2) // include high-volume events
```

**How it works:**
- Registers a `/debug_events` POST endpoint on the agent's HTTP server
- Auto-sets `debug_webhook_url` and `debug_webhook_level` in the SWML `params` during rendering
- The URL is built automatically using the same auth/proxy logic as other webhook URLs

**Event types at level 1:**

| Event label | Description |
|-------------|-------------|
| `session_start` | AI session started (model, TTS engine, voice, language) |
| `session_end` | AI session ended (reason, duration, token counts) |
| `barge` | User interrupted AI speech (barge type, elapsed ms) |
| `step_change` | Conversation step changed |
| `context_change` | Conversation context changed |
| `llm_error` | LLM error (fatal, retry, max_retries) |
| `voice_error` | TTS voice configuration or runtime error |
| `hold` | Call placed on hold or taken off hold |
| `filler` | Filler phrase spoken (thinking or function filler) |
| `consolidation` | Token consolidation triggered |
| `process_action` | Webhook action being processed |
| `gather_start` | Gather flow started |
| `gather_complete` | Gather flow completed |

**Additional events at level 2+:**

| Event label | Description |
|-------------|-------------|
| `llm_request` | LLM API request initiated (input tokens) |
| `llm_response` | LLM API response received (duration, output tokens) |
| `conversation_add` | Entry added to conversation history |

### Call Flow Verb Insertion

These methods customize the SWML call flow by inserting verbs at different stages of the call lifecycle.

##### `AddPreAnswerVerb(verbName string, config map[string]any) *AgentBase`
Add a verb to run before the call is answered (while still ringing).

**Safe pre-answer verbs:** `transfer`, `execute`, `return`, `label`, `goto`, `request`, `switch`, `cond`, `if`, `eval`, `set`, `unset`, `hangup`, `send_sms`, `sleep`, `stop_record_call`, `stop_denoise`, `stop_tap`

```go
// Send SMS before answering
a.AddPreAnswerVerb("send_sms", map[string]any{
	"to":   "+15551234567",
	"from": "+15559876543",
	"body": "Incoming call from AI agent",
})

// Set variables before answer
a.AddPreAnswerVerb("set", map[string]any{"call_start": "${system.timestamp}"})
```

##### `AddAnswerVerb(config map[string]any) *AgentBase`
Configure the answer verb that connects the call.

```go
// Set maximum call duration to 1 hour
a.AddAnswerVerb(map[string]any{"max_duration": 3600})
```

##### `AddPostAnswerVerb(verbName string, config map[string]any) *AgentBase`
Add a verb to run after the call is answered but before the AI starts.

```go
// Play welcome message before AI starts
a.AddPostAnswerVerb("play", map[string]any{
	"url": "say:Welcome to our AI assistant. This call may be recorded.",
})

// Add a brief pause
a.AddPostAnswerVerb("sleep", map[string]any{"duration": 1})
```

##### `AddPostAiVerb(verbName string, config map[string]any) *AgentBase`
Add a verb to run after the AI conversation ends.

```go
// Clean hangup after AI ends
a.AddPostAiVerb("hangup", map[string]any{})

// Transfer to human after AI conversation
a.AddPostAiVerb("transfer", map[string]any{"to": "+15551234567"})

// Log call completion
a.AddPostAiVerb("request", map[string]any{
	"url":    "https://myserver.com/call-complete",
	"method": "POST",
})
```

##### `ClearPreAnswerVerbs() *AgentBase` / `ClearPostAnswerVerbs() *AgentBase` / `ClearPostAiVerbs() *AgentBase`
Remove all pre-answer, post-answer, or post-AI verbs respectively.

**Method Chaining Example:**
```go
a.AddPreAnswerVerb("set", map[string]any{"source": "ai_agent"}).
	AddAnswerVerb(map[string]any{"max_duration": 1800}).
	AddPostAnswerVerb("play", map[string]any{"url": "say:Hello!"}).
	AddPostAiVerb("hangup", map[string]any{})
```

### Dynamic Configuration

##### `SetDynamicConfigCallback(cb DynamicConfigCallback) *AgentBase`
Set the callback for per-request dynamic configuration. `DynamicConfigCallback` is:

```go
type DynamicConfigCallback func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, agent *AgentBase)
```

```go
a.SetDynamicConfigCallback(func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
	// Configure based on request
	if queryParams["language"] == "spanish" {
		ep.AddLanguage(map[string]any{"name": "Spanish", "code": "es-ES", "voice": "nova.luna"})
	}

	// Set customer-specific data
	if customerID := headers["x-customer-id"]; customerID != "" {
		ep.SetGlobalData(map[string]any{"customer_id": customerID})
	}
})
```

### SIP Integration

##### `EnableSIPRouting(autoMap bool, path string) *AgentBase`
Enable SIP-based routing for voice calls.

```go
a.EnableSIPRouting(true, "/sip")
```

##### `RegisterSIPUsername(username string) *AgentBase`
Register a specific SIP username for this agent.

```go
a.RegisterSIPUsername("support")
a.RegisterSIPUsername("sales")
```

##### `RegisterRoutingCallback(callbackFn swml.RoutingCallback, path string)`
Register custom routing logic for calls. `swml.RoutingCallback` is `func(body map[string]any, headers map[string]any) *string` — return a pointer to a redirect route, or `nil` to process normally.

```go
func routeCall(body map[string]any, headers map[string]any) *string {
	sipUsername, _ := body["sip_username"].(string)
	switch sipUsername {
	case "support":
		route := "/support-agent"
		return &route
	case "sales":
		route := "/sales-agent"
		return &route
	}
	return nil
}

a.RegisterRoutingCallback(routeCall, "/sip")
```

### Utility Methods

##### `GetName() string`
Get the agent's name.

##### `AsRouter() http.Handler`
Get the agent as an `http.Handler` for embedding in a larger application.

```go
// Embed agent in a larger HTTP mux
mux := http.NewServeMux()
mux.Handle("/agent/", http.StripPrefix("/agent", a.AsRouter()))
```

### Event Handlers

##### `OnSummary(cb SummaryCallback) *AgentBase`
Register a handler for conversation summaries. This callback is triggered when the AI generates a summary based on your post-prompt configuration. `SummaryCallback` is `func(summary map[string]any, rawData map[string]any)`.

```go
a := agent.NewAgentBase(agent.WithName("summary-agent"), agent.WithRoute("/agent"))

// Configure post-prompt to request a JSON summary
a.SetPostPrompt(`
Return a JSON summary of the conversation:
{
    "topic": "MAIN_TOPIC",
    "satisfied": true/false,
    "follow_up_needed": true/false,
    "key_points": ["point1", "point2"]
}
`)

a.OnSummary(func(summary map[string]any, rawData map[string]any) {
	if summary != nil {
		// Access parsed JSON fields directly
		topic, _ := summary["topic"].(string)
		satisfied, _ := summary["satisfied"].(bool)
		fmt.Printf("Call about: %s, Customer satisfied: %v\n", topic, satisfied)

		// Save to database, send to CRM, trigger follow-up, etc.
		if followUp, _ := summary["follow_up_needed"].(bool); followUp {
			scheduleFollowUp(summary)
		}
	}

	// Access raw summary text if needed
	if pp, ok := rawData["post_prompt_data"].(map[string]any); ok {
		rawText, _ := pp["raw"].(string)
		fmt.Printf("Raw summary: %s\n", rawText)
	}
})
```

##### `OnDebugEvent(cb DebugEventHandler) *AgentBase`
Register a handler for debug webhook events. Requires `EnableDebugEvents` to be called first. `DebugEventHandler` is `func(event map[string]any)` — the event's label is available as `event["event_type"]` and the call id as `event["call_id"]`.

```go
a := agent.NewAgentBase(agent.WithName("my_agent"))
a.EnableDebugEvents(1)

a.OnDebugEvent(func(event map[string]any) {
	callID, _ := event["call_id"].(string)
	eventType, _ := event["event_type"].(string)
	switch eventType {
	case "llm_error":
		fmt.Printf("LLM error on call %s: %v\n", callID, event["event"])
	case "barge":
		fmt.Printf("Barge after %vms\n", event["barge_elapsed_ms"])
	case "session_end":
		fmt.Printf("Call ended: %v, duration: %vms\n", event["reason"], event["duration_ms"])
	}
})
```

> **Note:** Even without registering a handler, all debug events are automatically logged via the agent's structured logger when `EnableDebugEvents` is called.

##### `OnFunctionCall(name string, args map[string]any, rawData map[string]any) (any, error)`
Dispatch a SWAIG function call by name to its registered handler. Call this to invoke a tool programmatically (for example, from your own routing logic); it returns the tool's result.

```go
result, err := a.OnFunctionCall("get_weather", map[string]any{"location": "Miami"}, rawData)
```

##### `SetOnSwmlRequestHook(hook OnSwmlRequestHook) *AgentBase`
Register a hook to customize SWML generation per request. `OnSwmlRequestHook` is `func(requestData map[string]any, callbackPath string, r *http.Request) map[string]any` — return a map of modifications, or `nil` for the default document.

```go
a.SetOnSwmlRequestHook(func(requestData map[string]any, callbackPath string, r *http.Request) map[string]any {
	if callerType, _ := requestData["caller_type"].(string); callerType == "vip" {
		return map[string]any{
			"sections": map[string]any{
				"main": []any{
					map[string]any{"ai": map[string]any{"params": map[string]any{"wait_for_user": false}}},
				},
			},
		}
	}
	return nil
})
```

### Authentication

##### `ValidateBasicAuth(username, password string) bool`
Validate basic-auth credentials against the agent's configured (or environment-sourced) credentials.

```go
if a.ValidateBasicAuth(user, pass) {
	// authorized
}
```

##### `GetBasicAuthCredentials() (string, string)` / `GetBasicAuthCredentialsWithSource() (user, pass, source string)`
Get the basic-auth credentials from the constructor or environment; the `WithSource` variant also reports where they came from.

```go
user, pass := a.GetBasicAuthCredentials()
user, pass, source := a.GetBasicAuthCredentialsWithSource()
```

### Context System

##### `DefineContexts() *contexts.ContextBuilder`
Define structured workflow contexts for the agent.

```go
cb := a.DefineContexts()
cb.AddContext("greeting").
	AddStep("welcome").
	SetText("Welcome! How can I help?").
	SetValidSteps([]string{"next"})

cb.AddContext("main_menu").
	AddStep("menu").
	SetText("Choose: 1) Support 2) Sales 3) Billing").
	SetFunctions([]string{"transfer_to_support", "transfer_to_sales"})
```

`Contexts()` returns the current builder, and `ResetContexts()` clears it.

---

## FunctionResult

The `swaig.FunctionResult` type is used to create structured responses from SWAIG functions. It handles both natural-language responses and structured actions the agent should execute.

### Constructor

```go
func NewFunctionResult(response string) *FunctionResult
```

**Parameters:**
- `response` (string): Natural-language response text for the AI to speak (pass `""` for an actions-only result)

Post-processing (letting the AI take another turn before actions execute) is controlled with `SetPostProcess`.

```go
// Simple response
result := swaig.NewFunctionResult("The weather is sunny and 75°F")

// Empty response (actions only)
result := swaig.NewFunctionResult("")
```

### Core Methods

#### Response Configuration

##### `SetResponse(response string) *FunctionResult`
Set or update the natural-language response text.

```go
result := swaig.NewFunctionResult("")
result.SetResponse("I found your order information")
```

##### `SetPostProcess(postProcess bool) *FunctionResult`
Enable or disable post-processing for this result. When `true`, the AI responds to the user once more before the actions execute.

```go
result := swaig.NewFunctionResult("I'll help you with that")
result.SetPostProcess(true)
```

#### Action Management

##### `AddAction(name string, data any) *FunctionResult`
Add a structured action to execute. `data` can be a string, bool, map, or slice.

```go
// Simple action with boolean
result.AddAction("hangup", true)

// Action with string data
result.AddAction("play", "welcome.mp3")

// Action with object data
result.AddAction("set_global_data", map[string]any{"customer_id": "12345", "status": "verified"})

// Action with array data
result.AddAction("send_sms", []any{"+15551234567", "Your order is ready!"})
```

##### `AddActions(actions []map[string]any) *FunctionResult`
Add multiple actions at once.

```go
result.AddActions([]map[string]any{
	{"play": "hold_music.mp3"},
	{"set_global_data": map[string]any{"status": "on_hold"}},
	{"wait": 5000},
})
```

Other response accessors: `Response()`, `Actions()`, `PostProcess()`.

### Call Control Actions

#### Call Transfer and Connection

##### `Connect(destination string, final bool, from string) *FunctionResult`
Transfer or connect the call to another destination. `final=true` is a permanent transfer (call exits the agent); `final=false` returns the call to the agent if the far end hangs up. Pass `""` for `from` to keep the caller ID.

```go
// Permanent transfer to phone number
result.Connect("+15551234567", true, "")

// Temporary transfer to SIP address with custom caller ID
result.Connect("support@company.com", false, "+15559876543")
```

##### `SwmlTransfer(dest, aiResponse string, final bool) *FunctionResult`
Create a SWML-based transfer with an AI response for when the transfer completes.

```go
result.SwmlTransfer(
	"+15551234567",
	"You've been transferred back to me. How else can I help?",
	false,
)
```

##### `SIPRefer(toURI string) *FunctionResult`
Perform a SIP REFER transfer.

```go
result.SIPRefer("sip:support@company.com")
```

#### Call Management

##### `Hangup() *FunctionResult`
End the call immediately.

```go
result := swaig.NewFunctionResult("Thank you for calling. Goodbye!").Hangup()
```

##### `Hold(timeout int) *FunctionResult`
Put the call on hold for `timeout` seconds.

```go
result := swaig.NewFunctionResult("Please hold while I look that up").Hold(60)
```

##### `Stop() *FunctionResult`
Stop current audio playback or recording.

```go
result.Stop()
```

#### Audio Control

##### `Say(text string) *FunctionResult`
Add text for the AI to speak.

```go
result.Say("Please wait while I process your request")
```

##### `PlayBackgroundFile(filename string, wait bool) *FunctionResult`
Play an audio file in the background. When `wait` is true, block until the file finishes.

```go
// Play hold music in the background
result.PlayBackgroundFile("hold_music.mp3", false)

// Play announcement and wait for completion
result.PlayBackgroundFile("important_announcement.wav", true)
```

##### `StopBackgroundFile() *FunctionResult`
Stop background audio playback.

```go
result.StopBackgroundFile()
```

### Data Management Actions

##### `UpdateGlobalData(data map[string]any) *FunctionResult`
Update global data for the conversation (merge with existing).

```go
result.UpdateGlobalData(map[string]any{
	"last_interaction": "2024-01-15T10:30:00Z",
	"agent_notes":      "Customer satisfied with resolution",
})
```

##### `RemoveGlobalData(keys []string) *FunctionResult` / `RemoveGlobalDataKey(key string) *FunctionResult`
Remove one or more keys from global data.

```go
// Remove single key
result.RemoveGlobalDataKey("temporary_data")

// Remove multiple keys
result.RemoveGlobalData([]string{"temp1", "temp2", "cache_data"})
```

##### `SetMetadata(data map[string]any) *FunctionResult`
Set metadata for the conversation.

```go
result.SetMetadata(map[string]any{
	"call_type":  "support",
	"priority":   "high",
	"department": "technical",
})
```

##### `RemoveMetadata(keys []string) *FunctionResult` / `RemoveMetadataKey(key string) *FunctionResult`
Remove one or more metadata keys.

```go
result.RemoveMetadata([]string{"temporary_flag", "debug_info"})
```

### AI Behavior Control

##### `SetEndOfSpeechTimeout(ms int) *FunctionResult`
Adjust how long to wait for speech to end (milliseconds).

```go
result.SetEndOfSpeechTimeout(300)  // shorter, quick responses
result.SetEndOfSpeechTimeout(2000) // longer, thoughtful responses
```

##### `SetSpeechEventTimeout(ms int) *FunctionResult`
Set the timeout for speech events (milliseconds).

```go
result.SetSpeechEventTimeout(5000)
```

##### `WaitForUser(enabled *bool, timeout *int, answerFirst bool) *FunctionResult`
Control whether to wait for user input. `enabled` and `timeout` are pointers so they can be omitted (`nil`).

```go
enabled := true
timeout := 10000
result.WaitForUser(&enabled, &timeout, false)

// Don't wait for user
no := false
result.WaitForUser(&no, nil, false)
```

##### `ToggleFunctions(toggles []map[string]any) *FunctionResult`
Enable or disable specific functions.

```go
result.ToggleFunctions([]map[string]any{
	{"function": "transfer_to_sales", "active": true},
	{"function": "end_call", "active": false},
})
```

##### `EnableFunctionsOnTimeout(enabled bool) *FunctionResult`
Control whether functions are enabled when a timeout occurs.

```go
result.EnableFunctionsOnTimeout(false)
```

##### `EnableExtensiveData(enabled bool) *FunctionResult`
Enable extensive data collection.

```go
result.EnableExtensiveData(true)
```

##### `UpdateSettings(settings map[string]any) *FunctionResult`
Update various AI settings.

```go
result.UpdateSettings(map[string]any{
	"temperature":           0.8,
	"max_tokens":            150,
	"end_of_speech_timeout": 800,
})
```

### Context and Conversation Control

##### `SwitchContext(systemPrompt, userPrompt string, consolidate, fullReset, isolated bool) *FunctionResult`
Switch conversation context or reset the conversation. Pass `""` for a prompt to leave it unchanged.

```go
// Switch to technical support context
result.SwitchContext(
	"You are now a technical support specialist",
	"The customer needs technical help",
	false, false, false,
)

// Reset conversation completely
result.SwitchContext("", "", false, true, false)

// Consolidate conversation history
result.SwitchContext("", "", true, false, false)
```

##### `SimulateUserInput(text string) *FunctionResult`
Simulate user input for testing or automation.

```go
result.SimulateUserInput("I need help with my order")
```

### Communication Actions

##### `SendSms(toNumber, fromNumber, body string, media []string, tags []string, region string) *FunctionResult`
Send an SMS message. Pass `nil` for `media`/`tags` and `""` for `region` when unused.

```go
// Simple text message
result.SendSms("+15551234567", "+15559876543", "Your order #12345 has shipped!", nil, nil, "")

// Message with media and tags
result.SendSms(
	"+15551234567", "+15559876543", "Here's your receipt",
	[]string{"https://example.com/receipt.pdf"},
	[]string{"receipt", "order_12345"},
	"",
)
```

### Recording and Media

##### `RecordCall(controlID string, stereo bool, format RecordFormat, direction RecordDirection, opts *RecordCallOptions) *FunctionResult`
Start call recording. Formats: `swaig.FormatWAV`, `swaig.FormatMP3`, `swaig.FormatMP4`. Directions: `swaig.RecordDirectionBoth`, `swaig.RecordDirectionSpeak`, `swaig.RecordDirectionListen`. `RecordCallOptions` carries `Terminators`, `Beep`, `InputSensitivity`, `MaxLength`, `StatusURL`, and timeouts.

```go
// Basic recording
result.RecordCall("", false, swaig.FormatMP3, swaig.RecordDirectionBoth, nil)

// Recording with control ID and options
result.RecordCall("customer_call_001", true, swaig.FormatWAV, swaig.RecordDirectionBoth,
	&swaig.RecordCallOptions{
		Beep:        true,
		Terminators: "#*",
	})
```

##### `StopRecordCall(controlID string) *FunctionResult`
Stop call recording. Pass `""` to stop the default recording.

```go
result.StopRecordCall("")
result.StopRecordCall("customer_call_001")
```

### Conference and Room Management

##### `JoinRoom(name string) *FunctionResult`
Join a SignalWire room.

```go
result.JoinRoom("support_room_1")
```

##### `JoinConference(name string, opts *JoinConferenceOptions) *FunctionResult`
Join a conference call. All settings live on `JoinConferenceOptions` (`Muted`, `Beep`, `StartOnEnter *bool`, `EndOnExit`, `WaitURL`, `MaxParticipants`, `Record`, `Region`, `Trim`, `Coach`, status/recording callbacks).

```go
// Basic conference join
result.JoinConference("sales_meeting", nil)

// Conference with recording and settings
result.JoinConference("support_conference", &swaig.JoinConferenceOptions{
	Beep:            "onEnter",
	Record:          "record-from-start",
	MaxParticipants: 10,
})
```

### Payment Processing

##### `Pay(connectorURL string, opts *PayOptions) *FunctionResult`
Process a payment through the call. `PayOptions` carries all the settings (`InputMethod`, `PaymentMethod`, `Timeout`, `MaxAttempts`, `SecurityCode`, `PostalCode`, `TokenType`, `ChargeAmount`, `Currency`, `Language`, `Voice`, `Description`, `Parameters`, `Prompts`, etc.). Booleans that default to true (like `SecurityCode`) have a companion `...Set` flag to force `false`.

```go
// Basic payment processing
result.Pay("https://payment-processor.com/webhook", &swaig.PayOptions{
	ChargeAmount: "29.99",
	Description:  "Monthly subscription",
})

// Payment with custom settings
result.Pay("https://payment-processor.com/webhook", &swaig.PayOptions{
	InputMethod:  "voice",
	Timeout:      10,
	MaxAttempts:  3,
	ChargeAmount: "149.99",
	Currency:     "usd",
	Description:  "Premium service upgrade",
})
```

### Call Monitoring

##### `Tap(uri, controlID string, direction TapDirection, codec Codec, rtpPtime int, statusURL string) *FunctionResult`
Start call tapping/monitoring. Directions: `swaig.TapDirectionBoth`, `swaig.TapDirectionSpeak`, `swaig.TapDirectionHear`. Codecs: `swaig.CodecPCMU`, `swaig.CodecPCMA`.

```go
// Basic call tapping
result.Tap("sip:monitor@company.com", "", swaig.TapDirectionBoth, swaig.CodecPCMU, 20, "")

// Tap with specific settings
result.Tap("sip:quality@company.com", "quality_monitor_001", swaig.TapDirectionBoth, swaig.CodecPCMU, 20, "")
```

##### `StopTap(controlID string) *FunctionResult`
Stop call tapping.

```go
result.StopTap("")
result.StopTap("quality_monitor_001")
```

### Advanced SWML Execution

##### `ExecuteSwml(swmlContent any, transfer bool) *FunctionResult`
Execute custom SWML content.

```go
customSWML := map[string]any{
	"version": "1.0.0",
	"sections": map[string]any{
		"main": []any{
			map[string]any{"play": map[string]any{"url": "https://example.com/custom.mp3"}},
			map[string]any{"say": map[string]any{"text": "Custom SWML execution"}},
		},
	},
}
result.ExecuteSwml(customSWML, false)
```

### Utility Methods

##### `ToMap() map[string]any`
Convert the result to a map for serialization.

```go
result := swaig.NewFunctionResult("Hello world").AddAction("play", "music.mp3")
resultMap := result.ToMap()
// {"response": "Hello world", "action": [{"play": "music.mp3"}]}
```

### Static Helper Functions (payment prompts)

These package-level helpers build payment-prompt configurations for `PayOptions.Prompts`:

##### `CreatePaymentPrompt(forSituation string, actions []map[string]string, cardType, errorType string) map[string]any`

```go
prompt := swaig.CreatePaymentPrompt("card_number", []map[string]string{
	swaig.CreatePaymentAction("say", "Please enter your card number"),
}, "", "")
```

##### `CreatePaymentAction(actionType, phrase string) map[string]string`

```go
action := swaig.CreatePaymentAction("say", "Enter your card number")
```

##### `CreatePaymentParameter(name, value string) map[string]string`

```go
param := swaig.CreatePaymentParameter("merchant_id", "12345")
```

### Method Chaining

All action methods return the receiver, enabling fluent chaining:

```go
result := swaig.NewFunctionResult("I'll help you with that").
	SetPostProcess(true).
	UpdateGlobalData(map[string]any{"status": "helping"}).
	SetEndOfSpeechTimeout(800).
	AddAction("play", "thinking.mp3")

// Complex workflow
result := swaig.NewFunctionResult("Processing your payment").
	SetPostProcess(true).
	UpdateGlobalData(map[string]any{"payment_status": "processing"}).
	Pay("https://payments.com/webhook", &swaig.PayOptions{
		ChargeAmount: "99.99",
		Description:  "Service payment",
	}).
	SendSms("+15551234567", "+15559876543", "Payment confirmation will be sent shortly", nil, nil, "")
```

---

## DataMap

The `datamap.DataMap` type provides a declarative approach to creating SWAIG tools that integrate with REST APIs without requiring webhook infrastructure. DataMap tools execute on SignalWire's server infrastructure, eliminating the need to expose webhook endpoints.

### Constructor

```go
func New(functionName string) *DataMap
```

**Parameters:**
- `functionName` (string): Name of the SWAIG function this DataMap will create

```go
weatherMap := datamap.New("get_weather")
searchMap := datamap.New("search_docs")
```

### Core Configuration Methods

#### Function Metadata

##### `Purpose(description string) *DataMap`
Set the function description/purpose.

```go
dm := datamap.New("get_weather").Purpose("Get current weather information for any city")
```

##### `Description(description string) *DataMap`
Alias for `Purpose()`.

```go
dm := datamap.New("search_api").Description("Search our knowledge base for information")
```

#### Parameter Definition

##### `Parameter(name, paramType, desc string, required bool, enum []string) *DataMap`
Add a function parameter with JSON-schema validation. Pass `nil` for `enum` when there is no fixed value set.

```go
// Required string parameter
dm.Parameter("location", "string", "City name or ZIP code", true, nil)

// Optional number parameter
dm.Parameter("days", "number", "Number of forecast days", false, nil)

// Enum parameter with allowed values
dm.Parameter("units", "string", "Temperature units", false, []string{"celsius", "fahrenheit"})

// Boolean parameter
dm.Parameter("include_alerts", "boolean", "Include weather alerts", false, nil)

// Array parameter
dm.Parameter("categories", "array", "Search categories to include", false, nil)
```

### API Integration Methods

#### HTTP Webhook Configuration

##### `Webhook(method, url string, headers map[string]string, formParam string, inputArgsAsParams bool, requireArgs []string) *DataMap`
Configure an HTTP API call. Pass `nil` for `headers`/`requireArgs`, `""` for `formParam`, and `false` for `inputArgsAsParams` when unused.

**Variable Substitution in URLs:**
- `${args.parameter_name}`: Function argument values
- `${global_data.key}`: Call-wide data store (user info, call state — NOT credentials)
- `${meta_data.call_id}`: Call and function metadata

```go
// Simple GET request with parameter substitution
dm.Webhook("GET", "https://api.weather.com/v1/current?key=API_KEY&q=${args.location}", nil, "", false, nil)

// POST request with authentication headers
dm.Webhook("POST", "https://api.company.com/search",
	map[string]string{
		"Authorization": "Bearer YOUR_TOKEN",
		"Content-Type":  "application/json",
	}, "", false, nil)

// Webhook that requires specific arguments
dm.Webhook("GET", "https://api.service.com/data?id=${args.customer_id}", nil, "", false, []string{"customer_id"})
```

##### `Body(data map[string]any) *DataMap`
Set the JSON body for POST/PUT requests (supports `${variable}` substitution).

```go
dm.Body(map[string]any{
	"query": "${args.search_term}",
	"limit": 5,
	"filters": map[string]any{
		"category": "${args.category}",
		"active":   true,
	},
})
```

##### `Params(data map[string]any) *DataMap`
Set URL query parameters (supports `${variable}` substitution).

```go
dm.Params(map[string]any{
	"api_key": "YOUR_API_KEY",
	"q":       "${args.location}",
	"units":   "${args.units}",
	"lang":    "en",
})
```

#### Multiple Webhooks and Fallbacks

DataMap supports multiple webhook configurations for fallback scenarios — each `Webhook`/`Output` pair defines one attempt, and `FallbackOutput` is used if they all fail:

```go
dm := datamap.New("search_with_fallback").
	Purpose("Search with multiple API fallbacks").
	Parameter("query", "string", "Search query", true, nil).
	// Primary API
	Webhook("GET", "https://api.primary.com/search?q=${args.query}", nil, "", false, nil).
	Output(swaig.NewFunctionResult("Primary result: ${response.title}")).
	// Fallback API
	Webhook("GET", "https://api.fallback.com/search?q=${args.query}", nil, "", false, nil).
	Output(swaig.NewFunctionResult("Fallback result: ${response.title}")).
	// Final fallback if all APIs fail
	FallbackOutput(swaig.NewFunctionResult("Sorry, all search services are currently unavailable"))
```

### Response Processing

#### Basic Output

##### `Output(result *swaig.FunctionResult) *DataMap`
Set the response template for successful API calls.

**Variable Substitution in Outputs:**
- `${response.field}`: API response fields
- `${response.nested.field}`: Nested response fields
- `${response.array[0].field}`: Array element fields
- `${args.parameter}`: Original function arguments
- `${global_data.key}`: Call-wide data store

```go
// Simple response template
dm.Output(swaig.NewFunctionResult("Weather in ${args.location}: ${response.current.condition.text}, ${response.current.temp_f}°F"))

// Response with actions
dm.Output(swaig.NewFunctionResult("Found ${response.total_results} results").
	UpdateGlobalData(map[string]any{"last_search": "${args.query}"}).
	AddAction("play", "search_complete.mp3"))
```

##### `FallbackOutput(result *swaig.FunctionResult) *DataMap`
Set the response used when all webhooks fail.

```go
dm.FallbackOutput(swaig.NewFunctionResult("Sorry, the service is temporarily unavailable. Please try again later.").
	AddAction("play", "service_unavailable.mp3"))
```

#### Array Processing

##### `Foreach(config map[string]any) *DataMap`
Process array responses by iterating over elements.

```go
// Simple array processing
dm := datamap.New("search_docs").
	Webhook("GET", "https://api.docs.com/search?q=${args.query}", nil, "", false, nil).
	Foreach(map[string]any{"array": "${response.results}"}).
	Output(swaig.NewFunctionResult("Found: ${foreach.title} - ${foreach.summary}"))

// Advanced foreach configuration
dm.Foreach(map[string]any{
	"array": "${response.items}",
	"limit": 3, // process only first 3 items
	"filter": map[string]any{
		"field": "status",
		"value": "active",
	},
})
```

**Foreach Variable Access:**
- `${foreach.field}`: Current array element field
- `${foreach.nested.field}`: Nested fields in current element
- `${foreach_index}`: Current iteration index (0-based)
- `${foreach_count}`: Total number of items being processed

### Pattern-Based Processing

#### Expression Matching

##### `Expression(testValue, pattern string, output *swaig.FunctionResult, nomatchOutput *swaig.FunctionResult) *DataMap`
Add pattern-based responses without API calls. Pass `nil` for `nomatchOutput` when not needed. Use `ExpressionRegexp(testValue string, pattern *regexp.Regexp, ...)` to pass a pre-compiled `*regexp.Regexp`.

```go
controlMap := datamap.New("file_control").
	Purpose("Control file playback").
	Parameter("command", "string", "Playback command", true, nil).
	Parameter("filename", "string", "File to control", false, nil).
	// Start commands
	Expression("${args.command}", `start|play|begin`,
		swaig.NewFunctionResult("Starting playback").
			AddAction("start_playback", map[string]any{"file": "${args.filename}"}),
		nil).
	// Stop commands
	Expression("${args.command}", `stop|pause|halt`,
		swaig.NewFunctionResult("Stopping playback").
			AddAction("stop_playback", true),
		nil).
	// Volume commands
	Expression("${args.command}", `volume (\d+)`,
		swaig.NewFunctionResult("Setting volume to ${match.1}").
			AddAction("set_volume", "${match.1}"),
		nil)
```

**Pattern Matching Variables:**
- `${match.0}`: Full match
- `${match.1}`, `${match.2}`, etc.: Capture groups
- `${match.group_name}`: Named capture groups

### Error Handling

##### `ErrorKeys(keys []string) *DataMap`
Specify response fields that indicate errors.

```go
// Treat these response fields as errors
dm.ErrorKeys([]string{"error", "error_message", "status_code"})
```

##### `GlobalErrorKeys(keys []string) *DataMap`
Set global error keys for all webhooks in this DataMap.

```go
dm.GlobalErrorKeys([]string{"error", "message", "code"})
```

### Advanced Configuration

##### `WebhookExpressions(expressions []map[string]any) *DataMap`
Add expression-based webhook selection.

```go
dm.WebhookExpressions([]map[string]any{
	{
		"test":    "${args.type}",
		"pattern": "weather",
		"webhook": map[string]any{"method": "GET", "url": "https://weather-api.com/current?q=${args.location}"},
	},
	{
		"test":    "${args.type}",
		"pattern": "news",
		"webhook": map[string]any{"method": "GET", "url": "https://news-api.com/search?q=${args.query}"},
	},
})
```

### Complete DataMap Examples

#### Simple Weather API

```go
weatherTool := datamap.New("get_weather").
	Purpose("Get current weather information").
	Parameter("location", "string", "City name or ZIP code", true, nil).
	Parameter("units", "string", "Temperature units", false, []string{"celsius", "fahrenheit"}).
	Webhook("GET", "https://api.weather.com/v1/current?key=API_KEY&q=${args.location}&units=${args.units}", nil, "", false, nil).
	Output(swaig.NewFunctionResult("Weather in ${args.location}: ${response.current.condition.text}, ${response.current.temp_f}°F")).
	ErrorKeys([]string{"error"})

// Register with agent
a.RegisterSwaigFunction(weatherTool.ToSwaigFunction())
```

#### Search with Array Processing

```go
searchTool := datamap.New("search_knowledge").
	Purpose("Search company knowledge base").
	Parameter("query", "string", "Search query", true, nil).
	Parameter("category", "string", "Search category", false, []string{"docs", "faq", "policies"}).
	Webhook("POST", "https://api.company.com/search",
		map[string]string{"Authorization": "Bearer TOKEN"}, "", false, nil).
	Body(map[string]any{
		"query":    "${args.query}",
		"category": "${args.category}",
		"limit":    5,
	}).
	Foreach(map[string]any{"array": "${response.results}"}).
	Output(swaig.NewFunctionResult("Found: ${foreach.title} - ${foreach.summary}")).
	FallbackOutput(swaig.NewFunctionResult("Search service is temporarily unavailable"))
```

#### Command Processing (No API)

```go
controlTool := datamap.New("system_control").
	Purpose("Control system functions").
	Parameter("action", "string", "Action to perform", true, nil).
	Parameter("target", "string", "Target for the action", false, nil).
	// Restart commands
	Expression("${args.action}", `restart|reboot`,
		swaig.NewFunctionResult("Restarting ${args.target}").
			AddAction("restart_service", map[string]any{"service": "${args.target}"}),
		nil).
	// Status commands
	Expression("${args.action}", `status|check`,
		swaig.NewFunctionResult("Checking status of ${args.target}").
			AddAction("check_status", map[string]any{"service": "${args.target}"}),
		nil).
	// Default for unrecognized commands
	Expression("${args.action}", `.*`,
		swaig.NewFunctionResult("Unknown command: ${args.action}"),
		swaig.NewFunctionResult("Please specify a valid action"))
```

### Conversion and Registration

##### `ToSwaigFunction() map[string]any`
Convert the DataMap to a SWAIG function definition for registration.

```go
weatherMap := datamap.New("get_weather").
	Purpose("Get weather").
	Parameter("location", "string", "City", true, nil)

// Convert to a SWAIG function and register
swaigFunction := weatherMap.ToSwaigFunction()
a.RegisterSwaigFunction(swaigFunction)
```

### Convenience Functions

The SDK provides helper functions for common DataMap patterns:

##### `CreateSimpleAPITool(name, url, responseTemplate string, parameters map[string]map[string]any, method string, headers map[string]string, body map[string]any, errorKeys []string) *DataMap`

Create a simple API integration tool.

```go
weather := datamap.CreateSimpleAPITool(
	"get_weather",
	"https://api.weather.com/v1/current?key=API_KEY&q=${location}",
	"Weather in ${location}: ${response.current.condition.text}",
	map[string]map[string]any{
		"location": {"type": "string", "description": "City name", "required": true},
	},
	"GET", nil, nil, nil,
)

a.RegisterSwaigFunction(weather.ToSwaigFunction())
```

##### `CreateExpressionTool(name string, patterns map[string]ExpressionPattern, parameters map[string]map[string]any) *DataMap`

Create a pattern-based tool without API calls. An `ExpressionPattern` pairs a regex `Pattern` with a `*swaig.FunctionResult` `Result`.

```go
fileControl := datamap.CreateExpressionTool(
	"file_control",
	map[string]datamap.ExpressionPattern{
		"start": {Pattern: `start.*`, Result: swaig.NewFunctionResult("").AddAction("start_playback", true)},
		"stop":  {Pattern: `stop.*`, Result: swaig.NewFunctionResult("").AddAction("stop_playback", true)},
	},
	map[string]map[string]any{
		"command": {"type": "string", "description": "Playback command", "required": true},
	},
)

a.RegisterSwaigFunction(fileControl.ToSwaigFunction())
```

### Method Chaining

All DataMap methods return the receiver, enabling fluent chaining:

```go
completeTool := datamap.New("comprehensive_search").
	Purpose("Comprehensive search with fallbacks").
	Parameter("query", "string", "Search query", true, nil).
	Parameter("category", "string", "Search category", false, []string{"all", "docs", "faq"}).
	Webhook("GET", "https://primary-api.com/search?q=${args.query}&cat=${args.category}", nil, "", false, nil).
	Output(swaig.NewFunctionResult("Primary: ${response.title}")).
	Webhook("GET", "https://backup-api.com/search?q=${args.query}", nil, "", false, nil).
	Output(swaig.NewFunctionResult("Backup: ${response.title}")).
	FallbackOutput(swaig.NewFunctionResult("All search services unavailable")).
	ErrorKeys([]string{"error", "message"})
```

---

## Context System

The Context System enhances traditional prompt-based agents by adding structured workflows with sequential steps on top of a base prompt. Each step contains its own guidance, completion criteria, and function restrictions while building upon the agent's foundational prompt.

### ContextBuilder

The `contexts.ContextBuilder` is accessed via `agent.DefineContexts()` and provides the main interface for creating structured workflows.

#### Getting Started

```go
// Access the context builder
cb := a.DefineContexts()

// Create contexts and steps
cb.AddContext("greeting").
	AddStep("welcome").
	SetText("Welcome! How can I help you today?").
	SetStepCriteria("User has stated their need").
	SetValidSteps([]string{"next"})
```

##### `AddContext(name string) *Context`
Create a new context in the workflow.

```go
greetingContext := cb.AddContext("greeting")
mainMenuContext := cb.AddContext("main_menu")
supportContext := cb.AddContext("support")
```

### Context

The `Context` type represents a conversation context containing multiple steps. Key methods:

- `AddStep(name string) *Step`: Create a new step in this context
- `SetValidContexts(ctxs []string) *Context`: Set which contexts can be accessed from this one
- `SetPostPrompt(prompt string) *Context`: Override the agent's post-prompt when this context is active
- `SetSystemPrompt(prompt string) *Context`: Trigger a context switch with new system instructions (makes this a Context Switch Context)
- `SetConsolidate(consolidate bool) *Context`: Whether to consolidate conversation history when entering
- `SetFullReset(fullReset bool) *Context`: Whether to do a complete system-prompt replacement vs injection
- `SetUserPrompt(prompt string) *Context`: User message to inject when entering this context
- `SetPrompt(prompt string) *Context`: Simple string prompt applied to all steps
- `AddSection(title, body string) *Context`: Add a POM-style section to the context prompt
- `AddBullets(title string, bullets []string) *Context`: Add a POM-style bullet section

**Context Types:**

1. **Workflow Container Context** (no system prompt): Organizes steps without conversation state changes
2. **Context Switch Context** (has system prompt): Triggers conversation state changes when entered, processing entry parameters like a `context_switch` SWAIG action

**Prompt Hierarchy:** Base Agent Prompt → Context Prompt → Step Prompt

#### Usage Examples

```go
// Workflow container context (just organizes steps)
mainContext := cb.AddContext("main")
mainContext.SetPrompt("Follow standard customer service protocols")

// Context switch context (changes AI behavior)
billingContext := cb.AddContext("billing")
billingContext.SetSystemPrompt("You are now a billing specialist").
	SetConsolidate(true).
	SetUserPrompt("Customer needs billing assistance").
	AddSection("Department", "Billing Department").
	AddBullets("Services", []string{"Account inquiries", "Payments", "Refunds"})

// Full reset context (complete conversation reset)
managerContext := cb.AddContext("manager")
managerContext.SetSystemPrompt("You are a senior manager").
	SetFullReset(true).
	SetConsolidate(true)
```

### Step

Each `Step` is configured with fluent methods: `SetText(text)`, `AddSection(title, body)`, `AddBullets(title, bullets)`, `SetStepCriteria(criteria)`, `SetFunctions(functions)`, `SetValidSteps(steps)`, `SetValidContexts(ctxs)`, `SetEnd(end)`, and reset controls (`SetResetSystemPrompt`, `SetResetUserPrompt`, `SetResetConsolidate`, `SetResetFullReset`).

```go
step := ctx.AddStep("diagnose").
	SetText("Let me run some diagnostics to identify the issue.").
	SetFunctions([]string{"run_diagnostics", "check_system_status"}).
	SetStepCriteria("Diagnostics completed").
	SetValidSteps([]string{"resolve"})
```

---

## State Management

The Go SDK does not maintain per-call conversation state inside `AgentBase`. Persist any session state you need in your own store — a database, Redis, or a mutex-guarded in-memory map — keyed by `call_id`, and access it from your SWAIG handlers via `rawData["call_id"]`. The `startup_hook` / `hangup_hook` lifecycle tools (see [Session Lifecycle Hooks](#session-lifecycle-hooks)) are the natural place to initialize and clean up that state.

```go
// A minimal call-keyed state store
type stateStore struct {
	mu   sync.RWMutex
	data map[string]map[string]any
}

func (s *stateStore) Update(callID string, kv map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data[callID] == nil {
		s.data[callID] = map[string]any{}
	}
	for k, v := range kv {
		s.data[callID][k] = v
	}
}

func (s *stateStore) Get(callID string) (map[string]any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, ok := s.data[callID]
	return st, ok
}
```

---

## Skills System

The Skills System provides modular, reusable capabilities that can be added to any agent with `a.AddSkill(skills.<Name>, params)`.

### Available Built-in Skills

#### `skills.SkillDatetime`
Provides current date and time information.

**Parameters:**
- `timezone` (string): Timezone for date/time (default: system timezone)
- `format` (string): Custom date/time format string

```go
// Basic datetime skill
a.AddSkill(skills.SkillDatetime, nil)

// With timezone
a.AddSkill(skills.SkillDatetime, map[string]any{"timezone": "America/New_York"})
```

#### `skills.SkillMath`
Safe mathematical expression evaluation.

**Parameters:**
- `precision` (int): Decimal precision for results (default: 2)
- `max_expression_length` (int): Maximum expression length (default: 100)

```go
// Basic math skill
a.AddSkill(skills.SkillMath, nil)

// With custom precision
a.AddSkill(skills.SkillMath, map[string]any{"precision": 4})
```

#### `skills.SkillWebSearch`
Google Custom Search API integration with web scraping.

**Parameters:**
- `api_key` (string, required): Google Custom Search API key
- `search_engine_id` (string, required): Google Custom Search Engine ID
- `num_results` (int): Number of results to return (default: 3)
- `tool_name` (string): Custom tool name for multiple instances
- `delay` (float): Delay between requests in seconds
- `no_results_message` (string): Custom message when no results found

```go
// Basic web search
a.AddSkill(skills.SkillWebSearch, map[string]any{
	"api_key":          "your-google-api-key",
	"search_engine_id": "your-search-engine-id",
})

// Multiple search instances
a.AddSkill(skills.SkillWebSearch, map[string]any{
	"api_key":          "your-api-key",
	"search_engine_id": "general-engine-id",
	"tool_name":        "search_general",
	"num_results":      5,
})
```

#### `skills.SkillDatasphere`
SignalWire DataSphere knowledge search integration.

**Parameters:**
- `space_name` (string, required): DataSphere space name
- `project_id` (string, required): DataSphere project ID
- `token` (string, required): DataSphere access token
- `document_id` (string): Specific document to search
- `tool_name` (string): Custom tool name for multiple instances
- `count` (int): Number of results to return (default: 3)
- `tags` ([]string): Filter by document tags

```go
// Basic DataSphere search
a.AddSkill(skills.SkillDatasphere, map[string]any{
	"space_name": "my-space",
	"project_id": "my-project",
	"token":      "my-token",
})

// Multiple DataSphere instances
a.AddSkill(skills.SkillDatasphere, map[string]any{
	"space_name":  "my-space",
	"project_id":  "my-project",
	"token":       "my-token",
	"document_id": "drinks-menu",
	"tool_name":   "search_drinks",
	"count":       5,
})
```

#### `skills.SkillNativeVectorSearch`
Remote document search against a SignalWire search server. This skill is **remote-only** in the Go SDK — it queries a running search server over HTTP and has no local index file.

**Parameters:**
- `remote_url` (string, **required**): URL of the remote search server (e.g. `http://localhost:8001` or `http://user:pass@host:8001`). Basic-auth credentials may be embedded in the URL.
- `index_name` (string): Name of the index on the remote server (default: `"default"`)
- `tool_name` (string): Custom tool name (default: `"search_knowledge"`)
- `count` (int): Number of results to return (default: 5)
- `similarity_threshold` (float): Minimum similarity score 0.0-1.0 (default: 0.0). Higher values are stricter; lower values are more permissive.
- `tags` ([]string): Tags to filter search results
- `response_prefix` (string): Text prepended to results (default: "")
- `response_postfix` (string): Text appended to results (default: "")
- `max_content_length` (int): Maximum total response size in characters (default: 32768)
- `no_results_message` (string): Message when no results found; use `{query}` as a placeholder (default: `"No information found for '{query}'"`)
- `hints` ([]string): Speech-recognition hints for this skill
- `description` (string): Tool description shown to the AI (default: `"Search the knowledge base for information"`)

```go
// Basic remote search
a.AddSkill(skills.SkillNativeVectorSearch, map[string]any{
	"remote_url": "http://localhost:8001",
})

// With custom settings
a.AddSkill(skills.SkillNativeVectorSearch, map[string]any{
	"remote_url":           "http://user:pass@search.example.com:8001",
	"index_name":           "docs",
	"tool_name":            "search_docs",
	"count":                10,
	"similarity_threshold": 0.25,
})
```

### Creating Custom Skills

In Go, a skill implements the `skills.SkillBase` interface (it is an interface, not a base class to subclass). A skill:

- declares its `SkillName`, description, and version
- validates required parameters/environment in its setup step (return an error to fail loudly)
- registers one or more tools with the agent — either `DefineTool` handlers or `DataMap`-based tools via `RegisterSwaigFunction`
- contributes speech-recognition hints and prompt sections

A DataMap-backed skill's `register` step typically looks like:

```go
tool := datamap.New("custom_function").
	Description("Custom API integration").
	Parameter("query", "string", "Search query", true, nil).
	Webhook("GET", "https://api.example.com/search?key="+apiKey+"&q=${args.query}", nil, "", false, nil).
	Output(swaig.NewFunctionResult("Found: ${response.title}"))

a.RegisterSwaigFunction(tool.ToSwaigFunction())
```

Register your skill with the skill manager so it can be added by name. Use the built-in skills under `pkg/skills` as reference implementations of the interface.

---

## Utility Types

### ToolDefinition

`agent.ToolDefinition` describes a SWAIG function/tool passed to `DefineTool`. See [Function Definition](#function-definition) for its fields and usage.

### SWMLService

`AgentBase` builds on the SWMLService layer, which provides SWML document generation and HTTP serving. The agent exposes this through methods like `RenderSWML(requestData map[string]any, request *http.Request) map[string]any` (build the complete SWML document) and `HandleRequest(method, url string, headers map[string]string, body map[string]any) (int, map[string]string, string)` (handle an HTTP request and produce a response).

### Dynamic Configuration

The dynamic-configuration callback receives the (cloned) agent instance directly, allowing you to configure it based on request data.

```go
a.SetDynamicConfigCallback(func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ep *agent.AgentBase) {
	// Configure based on request
	if queryParams["lang"] == "es" {
		ep.AddLanguage(map[string]any{"name": "Spanish", "code": "es-ES", "voice": "nova.luna"})
	}

	// Customer-specific configuration
	if customerID := headers["x-customer-id"]; customerID != "" {
		ep.SetGlobalData(map[string]any{"customer_id": customerID})
		ep.PromptAddSection("Customer Context", fmt.Sprintf("You are helping customer %s", customerID), nil)
	}

	// Add skills dynamically
	if queryParams["enable_search"] == "true" {
		ep.AddSkill(skills.SkillWebSearch, map[string]any{"provider": "google"})
	}
})
```

---

## Environment Variables

The SDK supports various environment variables for configuration:

### Authentication
- `SWML_BASIC_AUTH_USER`: Basic auth username
- `SWML_BASIC_AUTH_PASSWORD`: Basic auth password

### SSL/HTTPS
- `SWML_SSL_ENABLED`: Enable SSL (true/false)
- `SWML_SSL_CERT_PATH`: Path to SSL certificate
- `SWML_SSL_KEY_PATH`: Path to SSL private key
- `SWML_DOMAIN`: Domain name for SSL

### Proxy Support
- `SWML_PROXY_URL_BASE`: Base URL for proxy server

### Skills Configuration
- `GOOGLE_SEARCH_API_KEY`: Google Custom Search API key
- `GOOGLE_SEARCH_ENGINE_ID`: Google Custom Search Engine ID
- `DATASPHERE_SPACE_NAME`: DataSphere space name
- `DATASPHERE_PROJECT_ID`: DataSphere project ID
- `DATASPHERE_TOKEN`: DataSphere access token

### Usage

```go
import "os"

// Set environment variables
os.Setenv("SWML_BASIC_AUTH_USER", "admin")
os.Setenv("SWML_BASIC_AUTH_PASSWORD", "secret")
os.Setenv("GOOGLE_SEARCH_API_KEY", "your-api-key")

// The agent will automatically use these
a := agent.NewAgentBase(agent.WithName("My Agent"))
a.AddSkill(skills.SkillWebSearch, map[string]any{
	"search_engine_id": "your-engine-id",
	// api_key will be read from the environment
})
```

---

## Complete Example

Here's a comprehensive example using multiple SDK components:

```go
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/datamap"
	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

func newComprehensiveAgent() *agent.AgentBase {
	a := agent.NewAgentBase(
		agent.WithName("Comprehensive Agent"),
		agent.WithAutoAnswer(true),
		agent.WithRecordCall(true),
	)

	// Configure voice and language
	a.AddLanguageTyped("English", "en-US", "rime.spore",
		[]string{"Let me check...", "One moment..."}, nil, "", "")

	// Add speech recognition hints
	a.AddHints([]string{"SignalWire", "customer service", "technical support"})

	// Configure AI parameters
	a.SetParams(map[string]any{
		"ai_model":              "gpt-4.1-nano",
		"end_of_speech_timeout": 800,
		"temperature":           0.7,
	})

	// Add skills
	a.AddSkill(skills.SkillDatetime, nil)
	a.AddSkill(skills.SkillMath, nil)
	a.AddSkill(skills.SkillWebSearch, map[string]any{
		"api_key":          "your-google-api-key",
		"search_engine_id": "your-engine-id",
		"num_results":      3,
	})

	// Set up structured workflow
	setupContexts(a)

	// Add custom tools
	registerCustomTools(a)

	// A tool handler that transfers to billing with state tracking
	a.DefineTool(agent.ToolDefinition{
		Name:        "transfer_to_billing",
		Description: "Transfer call to billing department",
		Handler: func(args, rawData map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("Transferring you to our billing department").
				UpdateGlobalData(map[string]any{"last_action": "transfer_to_billing"}).
				Connect("billing@company.com", false, "")
		},
	})

	// Handle conversation summaries
	a.OnSummary(func(summary map[string]any, rawData map[string]any) {
		fmt.Printf("Conversation completed: %v\n", summary)
	})

	// Set global data
	a.SetGlobalData(map[string]any{
		"company_name":  "Acme Corp",
		"support_hours": "9 AM - 5 PM EST",
		"version":       "2.0",
	})

	return a
}

func setupContexts(a *agent.AgentBase) {
	cb := a.DefineContexts()

	// Greeting context
	greeting := cb.AddContext("greeting")
	greeting.AddStep("welcome").
		SetText("Hello! Welcome to Acme Corp support. How can I help you today?").
		SetStepCriteria("Customer has explained their issue").
		SetValidSteps([]string{"next"})
	greeting.AddStep("categorize").
		AddSection("Current Task", "Categorize the customer's request").
		AddBullets("Categories", []string{
			"Technical issue - use diagnostic tools",
			"Billing question - transfer to billing",
			"General inquiry - handle directly",
		}).
		SetFunctions([]string{"transfer_to_billing", "run_diagnostics"}).
		SetStepCriteria("Request categorized and action taken")

	// Technical support context
	tech := cb.AddContext("technical_support")
	tech.AddStep("diagnose").
		SetText("Let me run some diagnostics to identify the issue.").
		SetFunctions([]string{"run_diagnostics", "check_system_status"}).
		SetStepCriteria("Diagnostics completed").
		SetValidSteps([]string{"resolve"})
	tech.AddStep("resolve").
		SetText("Based on the diagnostics, here's how we'll fix this.").
		SetFunctions([]string{"apply_fix", "schedule_technician"}).
		SetStepCriteria("Issue resolved or escalated")
}

func registerCustomTools(a *agent.AgentBase) {
	// Customer lookup tool (DataMap)
	lookupTool := datamap.New("lookup_customer").
		Description("Look up customer information").
		Parameter("customer_id", "string", "Customer ID", true, nil).
		Webhook("GET", "https://api.company.com/customers/${args.customer_id}",
			map[string]string{"Authorization": "Bearer YOUR_TOKEN"}, "", false, nil).
		Output(swaig.NewFunctionResult("Customer: ${response.name}, Status: ${response.status}")).
		ErrorKeys([]string{"error"})
	a.RegisterSwaigFunction(lookupTool.ToSwaigFunction())

	// System control tool (DataMap, expression-based)
	controlTool := datamap.New("system_control").
		Description("Control system functions").
		Parameter("action", "string", "Action to perform", true, nil).
		Parameter("target", "string", "Target system", false, nil).
		Expression("${args.action}", `restart|reboot`,
			swaig.NewFunctionResult("Restarting ${args.target}").
				AddAction("restart_system", map[string]any{"target": "${args.target}"}),
			nil).
		Expression("${args.action}", `status|check`,
			swaig.NewFunctionResult("Checking ${args.target} status").
				AddAction("check_status", map[string]any{"target": "${args.target}"}),
			nil)
	a.RegisterSwaigFunction(controlTool.ToSwaigFunction())
}

func main() {
	a := newComprehensiveAgent()
	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
```
