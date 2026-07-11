# SignalWire AI Agents SDK: Why the SDK, Not Raw SWML

## The Problem with Raw SWML

SWML (SignalWire Markup Language) is a JSON document format that defines how an agent behaves during a call -- 30+ verbs, an AI verb with dozens of parameters, SWAIG (SignalWire AI Gateway) function definitions with JSON Schema, post-prompt URLs, webhook authentication, language arrays, pronunciation rules, hints, global data, contexts, steps, gather configs. Writing it by hand means constructing deeply nested JSON, manually building authenticated webhook URLs, hand-coding parameter schemas, and deploying separate webhook servers for your tools. Every agent becomes a bespoke JSON engineering project.

The SDK eliminates all of this. You write Go. The SDK generates correct SWML, serves it over HTTP, and handles its own webhook callbacks -- all in one process, deployable to any platform.

---

## The Self-Referencing Pipeline

The SDK's core architectural insight is that the agent is both the **SWML generator** and the **SWAIG webhook handler** in a single stateless microservice.

```text
SignalWire requests SWML → Agent generates document
  ↓
SWML contains webhook URLs → URLs point back to the agent itself
  ↓
AI calls a function → SignalWire POSTs to agent's /swaig/ endpoint
  ↓
Agent executes function locally → Returns result to AI
  ↓
Call ends → SignalWire POSTs analytics to agent's /post_prompt/ endpoint
```

The agent auto-detects its own public URL -- including behind ngrok, load balancers, API Gateway, or any reverse proxy (via `X-Forwarded-Host`, `Forwarded` header, or `SWML_PROXY_URL_BASE` env var). It embeds Basic Auth credentials directly into the webhook URLs. It generates per-call security tokens for each function. The developer writes none of this:

```go
package main

import (
    "github.com/signalwire/signalwire-go/pkg/agent"
    "github.com/signalwire/signalwire-go/pkg/swaig"
)

func main() {
    a := agent.NewAgentBase(
        agent.WithName("weather"),
        agent.WithRoute("/weather"),
    )
    a.PromptAddSection("Role", "You help with weather.", nil)

    a.DefineTool(agent.ToolDefinition{
        Name:        "get_weather",
        Description: "Get weather",
        Parameters: map[string]any{
            "city": map[string]any{"type": "string"},
        },
        Required: []string{"city"},
        Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
            city, _ := args["city"].(string)
            // ... fetch weather ...
            return swaig.NewFunctionResult("72°F and sunny in " + city)
        },
    })

    a.Run()
}
```

That's a complete agent: HTTP server, SWML generation, authenticated webhook routing, function execution, and response formatting. The generated SWML contains the full AI configuration, function schemas, and webhook URLs pointing back to the running process -- all computed automatically.

---

## Prompt Object Model (POM)

Raw SWML prompts are flat strings. The SDK provides structured prompt building:

<!-- snippet-setup -->
```go
import (
	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/datamap"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// Shared agent established in prose above.
var a = agent.NewAgentBase()

// A context shared by the contexts/steps examples below.
var c = a.DefineContexts().AddContext("default")

var (
	_ = a
	_ = c
	_ = datamap.New
	_ = swaig.NewFunctionResult
)
```

```go
a.PromptAddSection("Role", "You are a travel booking assistant.", nil)
a.PromptAddSection("Rules", "", []string{
    "Never make up flight information",
    "Always confirm before booking",
    "Use the search tool for real data",
})
a.PromptAddSection("Personality", "Friendly but professional.", nil)
```

POM sections are rendered by the platform into a format the LLM understands with proper hierarchy. You can add subsections, append to existing sections, check if sections exist, and compose prompts programmatically -- including from skills that inject their own sections.

---

## Tools: Three Ways

### 1. Local Tools (Local Execution)

```go
package main

import (
    "github.com/signalwire/signalwire-go/pkg/agent"
    "github.com/signalwire/signalwire-go/pkg/swaig"
)

// order + db stand in for your own data layer.
type order struct{ ID, Status string }

func (o order) ToMap() map[string]any { return map[string]any{"id": o.ID, "status": o.Status} }

type orderDB struct{}

func (orderDB) Get(id string) order { return order{ID: id, Status: "shipped"} }

var db orderDB

func main() {
    a := agent.NewAgentBase()
    a.DefineTool(agent.ToolDefinition{
        Name:        "lookup_order",
        Description: "Look up an order",
        Parameters: map[string]any{
            "order_id": map[string]any{"type": "string"},
        },
        Required: []string{"order_id"},
        Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
            orderID, _ := args["order_id"].(string)
            order := db.Get(orderID)
            result := swaig.NewFunctionResult("Order " + order.ID + ": " + order.Status)
            result.AddAction("set_global_data", map[string]any{"current_order": order.ToMap()})
            return result
        },
    })
}
```

The SDK converts this into a SWAIG function definition with JSON Schema parameters, creates a secure webhook URL, routes inbound POST requests to the handler, parses arguments, and formats the response -- including the 20+ SWAIG actions (transfer, hold, context_switch, toggle_functions, etc.) that tools can return.

A tool's `Parameters` map is the JSON Schema `properties` object; `Required` lists the required fields. The `Handler` receives the parsed `args` plus the full `rawData` webhook payload and returns a `*swaig.FunctionResult`:

```go
package main

import (
    "github.com/signalwire/signalwire-go/pkg/agent"
    "github.com/signalwire/signalwire-go/pkg/swaig"
)

// order + db stand in for your own data layer.
type order struct{ ID, Status string }

type orderDB struct{}

func (orderDB) Get(id string) order { return order{ID: id, Status: "shipped"} }

var db orderDB

func main() {
    a := agent.NewAgentBase()
    a.DefineTool(agent.ToolDefinition{
        Name:        "lookup_order",
        Description: "Look up an order by ID",
        Parameters: map[string]any{
            "order_id": map[string]any{"type": "string", "description": "The order identifier"},
        },
        Required: []string{"order_id"},
        Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
            orderID, _ := args["order_id"].(string)
            order := db.Get(orderID)
            return swaig.NewFunctionResult("Order " + order.ID + ": " + order.Status)
        },
    })
}
```

### 2. DataMap (Server-Side Execution)

```go
dm := datamap.New("check_stock").
    Purpose("Check product stock levels").
    Parameter("sku", "string", "Product SKU", true, nil).
    Webhook("GET", "https://api.warehouse.com/stock/${args.sku}", nil, "", false, nil).
    Output(swaig.NewFunctionResult("Stock for ${args.sku}: ${response.quantity} units")).
    FallbackOutput(swaig.NewFunctionResult("Could not check stock right now"))

a.RegisterSwaigFunction(dm.ToSwaigFunction())
```

DataMap tools execute on SignalWire's servers -- no webhook needed. The SDK generates the `data_map` structure in the SWML with variable expansion (`${args.*}`, `${response.*}`, `${global_data.*}`), foreach iteration, expression matching, and error handling. Your agent never receives the callback; SignalWire handles the entire API call.

### 3. Skills (Packaged Integrations)

```go
a.AddSkill("web_search", map[string]any{"api_key": "...", "engine_id": "..."})
a.AddSkill("datetime", nil)
a.AddSkill("math", nil)
```

One line. The skill auto-registers its tools, injects prompt sections, adds speech hints, and validates dependencies. No manual wiring.

---

## The Skills System

Skills are self-contained modules that package tools, prompts, hints, and configuration into a single `AddSkill()` call. Each skill:

- Implements the `SkillBase` interface (embed `skills.BaseSkill`) with `Setup()` and `RegisterTools()` methods
- Declares `RequiredEnvVars()` for dependency validation (Go compiles its dependencies in, so there is no package-list to validate)
- Returns `[]skills.ToolRegistration` from `RegisterTools()` to register SWAIG functions
- Can inject prompt sections via `GetPromptSections()`
- Can provide speech hints via `GetHints()`
- Can contribute global data via `GetGlobalData()`
- Supports multiple instances with different configs (e.g., two `web_search` skills with different engines) by overriding `SupportsMultipleInstances()`

**Built-in skills:** `datetime`, `math`, `web_search`, `wikipedia_search`, `weather_api`, `google_maps`, `datasphere`, `datasphere_serverless`, `native_vector_search`, `spider`, `mcp_gateway`, `swml_transfer`, `play_background_file`, `info_gatherer`, `api_ninjas_trivia`, `joke`, `claude_skills`, `custom_skills`. Blank-import `_ "github.com/signalwire/signalwire-go/pkg/skills/all"` so each skill's `init()` registers it.

The elegance is composability: skills don't know about each other, but they all register cleanly into the same agent. A single agent can combine web search, datetime, a custom booking tool, and a DataMap stock checker -- all declared during construction, all generating correct SWML with proper function definitions, all routed to the right handler.

---

## Contexts and Steps: Priming the State Machine

The contexts/steps system lets you define structured workflows declaratively. Instead of hoping the LLM follows instructions about conversation flow, you mechanically enforce it:

```go
ctx := a.DefineContexts()

greeting := ctx.AddContext("default")
step1 := greeting.AddStep("welcome")
step1.SetText("Greet the user and ask how you can help.")
step1.SetValidSteps([]string{"collect_info"})
step1.SetFunctions([]string{"check_hours"}) // Only this tool available here

step2 := greeting.AddStep("collect_info")
step2.SetText("Collect the user's name and email.")
step2.SetStepCriteria("User has provided both name and email")
step2.SetGatherInfo("user_profile", "", "")
step2.AddGatherQuestion("name", "What is your name?")
step2.AddGatherQuestion("email", "What is your email?")
step2.SetValidSteps([]string{"confirm"})

step3 := greeting.AddStep("confirm")
step3.SetText("Confirm the information and say goodbye.")
step3.SetFunctions("none") // No tools -- just confirm and end
```

This generates SWML with a complete contexts/steps structure. The platform enforces navigation rules, restricts which functions are available at each step, collects structured data with typed questions and confirmation, and tracks transitions with trigger attribution in the enriched call_log. The LLM can't skip steps, can't call restricted tools, and can't navigate to disallowed contexts -- not because it was told not to, but because the mechanisms don't exist in its world. This is PGI (Programmatically Governed Inference) in practice.

**Multi-context** agents can define separate conversation modes (e.g., "sales" and "support") with isolated function sets, and use `set_valid_contexts()` to control switching. Context transitions support 4-mode reset (consolidate x full_reset) with conversation history summarization or archival.

---

## Programmatically Governed Inference (PGI)

The contexts/steps system is the SDK's implementation of a broader architectural discipline: **Programmatically Governed Inference**. PGI starts from a single design rule: *do not tell the AI anything it does not need to know.*

Current AI models are extraordinarily good at language -- understanding loosely phrased human input, mapping intent onto structured actions, and rendering system decisions back into natural speech. They are also inconsistent, non-deterministic, and prone to confident error. These are not bugs that will be fixed in the next model generation. They are properties of probabilistic inference itself. The industry's dominant response -- prompt harder and hope ("prompt and pray") -- treats the model as the brain of the system. PGI rejects this entirely. The model is not the brain. It is a controlled participant inside a deterministic system that was always in charge.

### The Four Layers

PGI is enforced through four layers of constraint, each operating independently. Only the first depends on the model's cooperation. The remaining three are mechanical.

**Layer 1: Semantic Constraints** -- The model receives a prompt describing its role and instructions for how to behave. This is the weakest layer; it depends on probabilistic compliance. PGI treats it as guidance, not enforcement. The remaining layers are the law.

**Layer 2: Schema Constraints** -- At each step, the model sees only the tools registered for that step. Tools belonging to other steps do not exist in its function schema. The model cannot call them, reference them, or reason about them. This is the difference between telling someone not to open a door and removing the door from the building.

**Layer 3: Transition Constraints** -- Each step defines which steps it can transition to. The platform validates every transition against this whitelist. The model cannot skip phases, loop back to completed steps, or jump to unreachable states. The conversational flow is governed by the same deterministic logic as any well-designed state machine.

**Layer 4: Execution Authority** -- When the model calls a tool, it is making a request, not issuing a command. The tool handler accesses authoritative state, applies business logic, and returns both a response for the model to speak and a set of actions for the platform to execute. The model does not update state. The model does not decide what happens next. The platform does.

### PGI in Practice: Blackjack

```go
betting := c.AddStep("betting")
betting.SetFunctions([]string{"place_bet"})
betting.SetValidSteps([]string{"playing"})

playing := c.AddStep("playing")
playing.SetFunctions([]string{"hit", "stand", "double_down"})
playing.SetValidSteps([]string{"hand_complete"})

lost := c.AddStep("you_lost")
lost.SetFunctions([]string{})
lost.SetValidSteps([]string{})
```

During the betting step, the model can only call `place_bet`. It cannot deal cards, draw cards, or resolve hands because those functions are not in its schema. When the tool handler transitions to the playing step, `place_bet` disappears and `hit`, `stand`, `double_down` appear. The model's capabilities change not because it was told to behave differently, but because the available operations were mechanically replaced.

The `you_lost` step has zero functions and zero valid transitions. The game is over. A user can beg, negotiate, or attempt social engineering. None of it works, because the mechanism for continuing does not exist. There is nothing for the model to comply with or resist. The interaction is structurally complete.

The tool handler demonstrates execution authority -- the model has no idea a step change is about to happen:

```go
package main

import (
    "fmt"

    "github.com/signalwire/signalwire-go/pkg/swaig"
)

// drawCard, calculateHand, formatCard stand in for your own game logic.
func drawCard(game map[string]any) string      { return "Ace of Spades" }
func calculateHand(game map[string]any) int    { return 20 }
func formatCard(card string) string            { return card }

func handleHit(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
    globalData, _ := rawData["global_data"].(map[string]any)
    game, _ := globalData["game_state"].(map[string]any)
    card := drawCard(game)
    score := calculateHand(game)

    result := swaig.NewFunctionResult(
        fmt.Sprintf("You drew %s. Your total is %d.", formatCard(card), score),
    )
    result.UpdateGlobalData(map[string]any{"game_state": game})

    if score > 21 {
        result.SwmlChangeStep("you_lost")
    }

    return result
}

func main() { _ = handleHit }
```

The model speaks the result. The platform changes the step. The model's world changes without its participation.

### Data Isolation

PGI extends to how data flows through the system. The model operates on a projection of reality, not the full truth. Authoritative state lives in structured data (`global_data`) that the model sees only in curated subsets. In a blackjack game, the model knows the player's chip count and visible cards. It does not know the deck composition, the dealer's hidden card, or the internal scoring calculations. In an ordering system, the model knows which items have been added. It does not know the internal pricing logic, tax calculations, or inventory state.

The model cannot hallucinate a price it has never seen. It cannot promise availability it has no knowledge of. It can only report what the system tells it to report.

### Why PGI, Not Guardrails

PGI produces a property that makes it fundamentally different from guardrails, output filtering, or any other containment strategy: **the model does not know it is being governed.** It does not know that other tools exist elsewhere in the system. It does not know that a state machine is managing the interaction. It sees its current world -- a prompt, a set of functions, a conversation history -- and operates within it. There is nothing to reason around, nothing to game, nothing to circumvent.

The strongest test of any PGI system: replace the model with a rigid scripted menu ("press 1 for tacos, press 2 for drinks") and the system would still produce correct outcomes. The tool handlers would still validate input, enforce business rules, and manage state. The experience would be worse, but every order would be accurate and every transition would follow the rules. The model makes the interaction natural. The software makes it correct. In a PGI system, those are independent properties.

The SDK's contexts/steps/function restrictions are the primitives that make PGI mechanical rather than aspirational. The developer defines steps, scopes tools to steps, declares transitions, and writes tool handlers that return structured results with platform actions. The platform enforces all of it. The developer brings domain expertise. The SDK provides the governance infrastructure.

---

## Deployment: One `Run()` Call

```go
a = agent.NewAgentBase(agent.WithName("my-agent"), agent.WithRoute("/agent"))
a.Run()
```

That single call auto-detects the environment and does the right thing:

| Environment | Detection | What Happens |
|-------------|-----------|--------------|
| **Standalone** | Default | Starts uvicorn HTTP server with FastAPI |
| **AWS Lambda** | Lambda context object | Returns Lambda-formatted response |
| **Google Cloud Functions** | GCF environment markers | Returns Flask-compatible response |
| **Azure Functions** | Azure context object | Returns Azure HttpResponse |
| **CGI** | CGI environment variables | Reads stdin, writes stdout |

Each mode handles authentication differently (HTTP Basic Auth, API Gateway authorizers, function-level auth), constructs webhook URLs using the correct public endpoint (Lambda function URL, GCF URL, Azure app URL), and formats request/response bodies per platform. You write one agent, deploy it anywhere.

For standalone mode, the SDK provides:
- Kubernetes health (`/health`) and readiness (`/ready`) probes
- SSL/TLS support via `SWML_SSL_ENABLED`, `SWML_SSL_CERT`, `SWML_SSL_KEY`
- CORS configuration
- Debug endpoint (`/debug`) for inspection

---

## Multi-Agent Hosting

```go
import "github.com/signalwire/signalwire-go/pkg/server"

salesAgent := agent.NewAgentBase(agent.WithName("sales"))
supportAgent := agent.NewAgentBase(agent.WithName("support"))
triageAgent := agent.NewAgentBase(agent.WithName("triage"))

srv := server.NewAgentServer(
    server.WithServerHost("0.0.0.0"),
    server.WithServerPort(3000),
)
srv.Register(salesAgent, "/sales")
srv.Register(supportAgent, "/support")
srv.Register(triageAgent, "/triage")
srv.Run()
```

One process, multiple agents, route-based dispatch. Each agent gets its own SWML endpoint and SWAIG callback routing. SIP routing can map usernames to specific agents.

---

## Dynamic Configuration and Multi-Tenancy

```go
// loadTenantConfig stands in for your own per-tenant config lookup.
loadTenantConfig := func(tenant string) struct {
    CompanyInfo string
    Tier        string
} {
    return struct {
        CompanyInfo string
        Tier        string
    }{CompanyInfo: "Acme Inc.", Tier: "premium"}
}

tenantConfig := func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, ag *agent.AgentBase) {
    tenant := headers["X-Tenant-ID"]
    if tenant == "" {
        tenant = "default"
    }
    config := loadTenantConfig(tenant)
    ag.PromptAddSection("Company", config.CompanyInfo, nil)
    ag.SetGlobalData(map[string]any{"tenant_id": tenant, "tier": config.Tier})
    if config.Tier == "premium" {
        ag.AddSkill("advanced_search", nil)
    }
}

a.SetDynamicConfigCallback(tenantConfig)
```

Each inbound request creates an **ephemeral copy** of the agent. The callback customizes it per-request -- different prompts, skills, global data, languages, tools. The original agent is unchanged. This enables multi-tenancy from a single deployment: one agent instance serves hundreds of tenants with tailored behavior.

---

## Search System

The Go SDK provides knowledge-base search through the built-in `native_vector_search`
skill, which connects to a **remote search server** over HTTP. The skill is
remote-only: it does not build or read local index files, and the Go SDK ships no
index-building CLI or local search backend. You run the search server separately
(it exposes `/health` and `/search` endpoints) and point the skill at it:

```go
a.AddSkill("native_vector_search", map[string]any{
    "remote_url":  "http://localhost:8001",       // required
    "index_name":  "knowledge",                    // index to query on the server
    "tool_name":   "search_docs",
    "description":  "Search product documentation",
})
```

The skill:
- Sends the AI's query to the remote server's `/search` endpoint and formats the
  returned results (content, source filename/section, and relevance score) for the AI.
- Supports HTTP Basic auth via `http://user:pass@host:8001` in `remote_url`.
- Validates the URL for SSRF protection (private/loopback addresses are rejected
  unless `SWML_ALLOW_PRIVATE_URLS` is set).
- Accepts `count`, `similarity_threshold`, `tags`, `response_prefix`,
  `response_postfix`, `max_content_length`, and `no_results_message` parameters to
  tune queries and response formatting.

How documents are ingested, chunked, embedded, and stored is the responsibility of
the remote search server, not the Go SDK.

---

## Prefab Agents

Production-ready patterns for common use cases:

```go
import "github.com/signalwire/signalwire-go/pkg/prefabs"

// Collect structured data
questions := []prefabs.Question{
    {KeyName: "name", QuestionText: "What is your name?"},
    {KeyName: "issue", QuestionText: "Describe your issue", Confirm: true},
}
gatherer := prefabs.NewInfoGathererAgent(prefabs.InfoGathererOptions{
    Questions: &questions,
})

// Route calls to departments
receptionist := prefabs.NewReceptionistAgent(prefabs.ReceptionistOptions{
    Departments: []prefabs.Department{
        {Name: "Sales", Number: "+15551234567", Description: "Product inquiries"},
        {Name: "Support", Number: "+15559876543", Description: "Technical help"},
    },
})

_, _ = gatherer, receptionist
```

Five prefabs: **InfoGatherer**, **Survey**, **Receptionist**, **FAQ**, **Concierge** —
constructed via `prefabs.New*Agent(Options{...})` (see
`examples/{prefab_info_gatherer,prefab_survey,receptionist,concierge,faq_bot}/main.go`).
Each generates complete SWML with appropriate prompts, tools, and workflows. You
instantiate, customize, deploy.

---

## AI Configuration

Everything the platform supports, the SDK exposes as methods:

```go
// LLM tuning
a.SetPromptLlmParams(map[string]any{
    "temperature":      0.3,
    "top_p":            0.9,
    "barge_confidence": 0.7,
})

// Multi-language
a.AddLanguageTyped("Spanish", "es", "google.es-ES-Neural2-A",
    []string{"Un momento..."}, []string{"Buscando..."}, "", "")

// Speech recognition
a.AddHints([]string{"SignalWire", "SWML", "SWAIG"})
a.AddPronunciation("SignalWire", "Signal Wire")

// Vision, thinking, inner dialog
a.SetParams(map[string]any{"enable_vision": true, "vision_model": "gpt-4o"})
a.SetParams(map[string]any{"enable_thinking": true, "thinking_model": "o4-mini"})

// Interruption control
a.SetParams(map[string]any{
    "barge_match_string": "^(stop|cancel|nevermind)$",
    "barge_min_words":    2,
    "barge_confidence":   0.8,
})

// Native functions with custom fillers
a.SetNativeFunctions([]string{"check_time", "wait_for_user"})
a.AddInternalFiller("check_time", "en", []string{"Let me check the time..."})

// Call flow verbs
a.AddPreAnswerVerb("play", map[string]any{"url": "ringback.wav"})
a.AddPostAiVerb("hangup", map[string]any{})
```

Call recording is configured at construction via functional options:

```go
a = agent.NewAgentBase(
    agent.WithName("recorder"),
    agent.WithRecordCall(true),
    agent.WithRecordFormat("wav"),
    agent.WithRecordStereo(true),
)
```

Each of these would require understanding and manually constructing the correct SWML JSON structure. The SDK provides named methods with proper defaults.

---

## swaig-test CLI

The Go `swaig-test` drives a **running** agent over HTTP — start the agent, then
point the CLI at its `--url`. Function args are passed with `--param key=value`.

```bash
# List available tools
swaig-test --url http://localhost:3000/ --list-tools

# Execute a specific tool (args via --param key=value)
swaig-test --url http://localhost:3000/ --exec get_weather --param city="San Francisco"

# Dump generated SWML for inspection
swaig-test --url http://localhost:3000/ --dump-swml

# Lambda serverless environment simulation (the only platform the Go CLI implements)
swaig-test --url http://localhost:3000/ --simulate-serverless lambda --dump-swml

# Multi-agent: target a specific agent by its route in the URL path
swaig-test --url http://localhost:3000/support --list-tools
```

---

## Authentication

The SDK handles auth automatically:

- **Auto-generated credentials:** If no env vars set, generates `user_XXXX` / random password and prints to console
- **Environment variables:** `SWML_BASIC_AUTH_USER` / `SWML_BASIC_AUTH_PASSWORD`
- **Embedded in URLs:** Webhook URLs include `user:pass@host` automatically
- **Per-function tokens:** Secure functions get `__token=...` query params with expiration
- **Platform-specific:** Different auth handling for Lambda, CGI, GCF, Azure (each platform has its own auth mechanism)

---

## What You'd Have to Build Without the SDK

| Capability | Without SDK | With SDK |
|-----------|-------------|----------|
| SWML document | Hand-craft JSON | Auto-generated from Go |
| Webhook server | Build and deploy separately | Built into the agent process |
| URL routing | Manual FastAPI/Flask setup | Automatic route registration |
| Auth tokens | Manual JWT/token system | Auto-generated per call/function |
| Proxy detection | Parse headers yourself | Automatic (ngrok, LB, CDN) |
| Tool schemas | Write JSON Schema by hand | `DefineTool` with a `Parameters` map |
| Serverless deploy | Platform-specific handler code | `a.Run()` auto-detects |
| Multi-language | Manually construct language arrays | `AddLanguage()` one-liner |
| State machine | Manually build contexts JSON | Fluent `DefineContexts()` API |
| Structured data collection | Build gather configs by hand | `AddGatherQuestion()` chain |
| Search/RAG | Wire up a search client | `AddSkill("native_vector_search", ...)` against a remote search server |
| Multi-agent | Separate deployments + router | `server.AgentServer` with route registration |
| Dynamic config | Custom middleware | `SetDynamicConfigCallback()` |
| Post-call analytics | Parse raw webhook payload | `OnSummary()` callback |
| Health checks | Manual endpoints | Built-in `/health` and `/ready` |
| Call recording | Manual SWML verb insertion | `WithRecordCall()` option |
| SSL/TLS | Manual cert configuration | Env var driven |

The SDK turns what would be a multi-file infrastructure project into a handful of Go calls. The SWML is correct by construction. The webhooks route themselves. The auth is automatic. The deployment is universal. The developer focuses on what the agent should *do*, not how to wire it together.
