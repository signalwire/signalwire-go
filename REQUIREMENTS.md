# SignalWire AI Agents Go SDK - Requirements & Progress Tracker

This document tracks the complete port of the SignalWire AI Agents SDK from Python/TypeScript to Go.

## Phase 1: Project Foundation

- [x] Go module initialized (`github.com/signalwire/signalwire-agents-go`)
- [x] Directory structure matching Python/TS layout
- [x] `.gitignore` for Go projects
- [x] `README.md` with project overview
- [x] `CLAUDE.md` with development guidance
- [x] Logging system (structured logging, levels, modes)
- [x] Configuration system (env vars, config files, defaults)

## Phase 2: SWML Core

### SWMLService (Base Foundation)
- [x] SWML document model (sections, verbs, rendering to JSON)
- [x] Schema validation for SWML verbs (from schema.json)
- [x] Auto-generated verb methods (all 38 verbs from schema)
- [x] 5-phase call flow (pre-answer, answer, post-answer, AI, post-AI)
- [x] HTTP server with routing (net/http standard library)
- [x] Basic auth (auto-generated credentials or env-based)
- [x] Security headers (X-Content-Type-Options, X-Frame-Options)
- [x] Health/readiness endpoints (`/health`, `/ready`)
- [x] Routing callbacks
- [x] SIP username extraction from request body
- [x] `SWML_PROXY_URL_BASE` env var support

### SWMLBuilder
- [x] Verb accumulation and section management
- [x] `Render()` → JSON string, `RenderPretty()` → indented JSON
- [x] `Reset()` for reuse

## Phase 3: Agent Core

### AgentBase
- [x] Constructor with functional options (name, route, host, port, auth, etc.)
- [x] Prompt management (raw text mode)
- [x] Prompt Object Model (POM) builder (sections, bullets, subsections)
- [x] `PromptAddSection()`, `PromptAddSubsection()`, `PromptAddToSection()`
- [x] `SetPromptText()` / `SetPostPrompt()` for raw text
- [x] Dynamic config callback (per-request agent customization)
- [x] Ephemeral agent copies for dynamic config
- [x] SWML rendering pipeline (5-phase document assembly)
- [x] `OnRequest()` handler for SWML generation
- [x] `Run()` / `Serve()` for HTTP server mode
- [x] `AsRouter()` for embedding in larger servers

### Tool/SWAIG System
- [x] `DefineTool()` method (name, description, parameters, handler)
- [x] `RegisterSwaigFunction()` for raw function dicts (DataMap)
- [x] `DefineTools()` to list all registered tools
- [x] `OnFunctionCall()` handler for SWAIG webhook dispatch
- [x] Secure tools with HMAC-SHA256 tokens (SessionManager)
- [x] Token creation and validation with expiry
- [x] SWAIG query params support
- [x] Webhook URL configuration

### SwaigFunctionResult (40+ actions)
- [x] Constructor with response text and post_process flag
- [x] `SetResponse()`, `SetPostProcess()`
- [x] `AddAction()`, `AddActions()`
- [x] `ToMap()` serialization

#### Call Control Actions
- [x] `Connect()`, `SwmlTransfer()`, `Hangup()`, `Hold()`, `WaitForUser()`, `Stop()`

#### State & Data Management
- [x] `UpdateGlobalData()` / `RemoveGlobalData()`
- [x] `SetMetadata()` / `RemoveMetadata()`
- [x] `SwmlUserEvent()`, `SwmlChangeStep()`, `SwmlChangeContext()`
- [x] `SwitchContext()`, `ReplaceInHistory()`

#### Media Control
- [x] `Say()`, `PlayBackgroundFile()` / `StopBackgroundFile()`, `RecordCall()` / `StopRecordCall()`

#### Speech & AI Config
- [x] `AddDynamicHints()` / `ClearDynamicHints()`
- [x] `SetEndOfSpeechTimeout()` / `SetSpeechEventTimeout()`
- [x] `ToggleFunctions()` / `EnableFunctionsOnTimeout()`, `EnableExtensiveData()`, `UpdateSettings()`

#### Advanced Features
- [x] `ExecuteSwml()`, `JoinConference()` / `JoinRoom()`, `SipRefer()`
- [x] `Tap()` / `StopTap()`, `SendSms()`, `Pay()`

#### RPC Actions
- [x] `ExecuteRpc()`, `RpcDial()`, `RpcAiMessage()`, `RpcAiUnhold()`, `SimulateUserInput()`

### DataMap (Server-side Tools)
- [x] Fluent builder: `New("name")`
- [x] `Purpose()` / `Description()` / `Parameter()` / `Expression()`
- [x] `Webhook()` / `WebhookExpressions()` / `Body()` / `Params()` / `Foreach()`
- [x] `Output()` / `FallbackOutput()` / `ErrorKeys()` / `GlobalErrorKeys()`
- [x] `ToSwaigFunction()` serialization
- [x] `CreateSimpleApiTool()` / `CreateExpressionTool()` helpers

### Contexts & Steps System
- [x] `ContextBuilder` with `AddContext()` / `GetContext()` / `Validate()`
- [x] `Context` with full API (AddStep, navigation, prompts, fillers, sections)
- [x] `Step` with full API (text, criteria, functions, gather info, reset settings)
- [x] `GatherInfo` and `GatherQuestion` types
- [x] `CreateSimpleContext()` helper

### AI Configuration
- [x] Hints, pattern hints, languages, pronunciations, params
- [x] Global data, native functions, internal fillers
- [x] Debug events, function includes, LLM params

### Verb Management
- [x] Pre-answer, answer, post-answer, post-AI verb management with clear methods

### SIP Routing
- [x] `EnableSipRouting()`, `RegisterSipUsername()`

### Web/HTTP
- [x] Dynamic config callback, proxy URL, webhook URL, post-prompt URL, query params

### Lifecycle Callbacks
- [x] `OnSummary()`, `OnDebugEvent()`

### Auth & Session
- [x] Basic auth with auto-generated credentials
- [x] SessionManager with HMAC-SHA256 tokens
- [x] Timing-safe token validation

## Phase 4: Skills System

- [x] `SkillBase` interface with full contract
- [x] `BaseSkill` with default implementations
- [x] `SkillManager` (load, unload, validate, lifecycle)
- [x] `SkillRegistry` (global registry with factory functions)
- [x] `ToolHandler` type definition

### Built-in Skills (all 18 ported)
- [x] datetime, math, joke
- [x] weather_api, web_search, wikipedia_search, google_maps
- [x] spider, datasphere, datasphere_serverless
- [x] swml_transfer, play_background_file, api_ninjas_trivia
- [x] native_vector_search (network mode only)
- [x] info_gatherer, claude_skills, mcp_gateway, custom_skills

## Phase 5: Prefab Agents

- [x] InfoGathererAgent (sequential field collection)
- [x] SurveyAgent (multi-type questions with validation)
- [x] ReceptionistAgent (department routing + transfer)
- [x] FAQBotAgent (keyword matching + scoring)
- [x] ConciergeAgent (venue/service information)

## Phase 6: AgentServer (Multi-Agent Hosting)

- [x] `Register()` / `Unregister()` agents by route
- [x] `GetAgents()` / `GetAgent()`
- [x] `SetupSipRouting()` / `RegisterSipUsername()`
- [x] `Run()` with HTTP server
- [x] Health/readiness endpoints
- [x] Security headers
- [x] Static file serving

## Phase 7: RELAY Client (WebSocket Call Control)

### Connection & Auth
- [x] WebSocket connection (gorilla/websocket)
- [x] JSON-RPC 2.0 message framing
- [x] Auto-reconnect with exponential backoff
- [x] Legacy auth (project_id + token) and JWT
- [x] Authorization state for fast reconnect
- [x] Context subscription/unsubscription

### Correlation & Events
- [x] JSON-RPC `id` → pending request channels
- [x] `call_id` → Call object routing
- [x] `control_id` → Action object tracking
- [x] `tag` → dial correlation
- [x] Event ACK and server ping handling

### Call Object (30+ methods)
- [x] Lifecycle: Answer, Hangup, Pass, Transfer
- [x] Audio: Play, PlayAndCollect, Collect
- [x] Recording, Bridging, DTMF, Detection
- [x] Fax, Tap, Stream, Pay, Transcribe
- [x] Conferencing, AI, Hold/Denoise, Room/Queue
- [x] BindDigit, UserEvent, Echo

### Action Objects (11 types)
- [x] Base Action with Wait/Stop/IsDone/OnCompleted
- [x] PlayAction (Pause, Resume, Volume)
- [x] RecordAction, DetectAction, CollectAction, FaxAction
- [x] TapAction, StreamAction, PayAction, TranscribeAction, AIAction

### Event System & Messaging
- [x] 22+ typed event types
- [x] SMS/MMS messaging with delivery tracking

## Phase 8: REST Client (HTTP API)

- [x] HttpClient with Basic Auth, JSON, connection pooling
- [x] CrudResource with List/Create/Get/Update/Delete
- [x] PaginatedIterator
- [x] SignalWireRestError

### Namespaces (21 implemented)
- [x] Fabric (16 sub-resources), Calling (37 commands)
- [x] PhoneNumbers, Datasphere, Video, Compat (LAML)
- [x] Addresses, Queues, Recordings, NumberGroups
- [x] VerifiedCallers, SipProfile, Lookup, ShortCodes
- [x] ImportedNumbers, MFA, Registry, Logs
- [x] Project, PubSub, Chat

## Phase 9: Serverless Support

- [ ] AWS Lambda handler adapter
- [ ] Google Cloud Functions adapter
- [ ] Azure Functions adapter
- [ ] CGI mode adapter
- [ ] Auto-detection of execution environment

## Phase 10: CLI Tools

- [ ] `swaig-test` equivalent (in progress)
  - [ ] `--dump-swml` output
  - [ ] `--list-tools`
  - [ ] `--exec` tool execution
  - [ ] `--param` parameter passing
  - [ ] `--raw` compact JSON output

## Phase 11: Documentation & Examples

### Documentation
- [x] `docs/` — 17 guides copied from Python SDK
- [x] `pkg/relay/` — RELAY implementation guide + 5 doc files
- [x] `pkg/rest/` — REST README + 6 doc files

### Examples (in progress)
- [ ] `examples/simple_agent/`
- [ ] `examples/simple_dynamic_agent/`
- [ ] `examples/multi_agent_server/`
- [ ] `examples/contexts_demo/`
- [ ] `examples/datamap_demo/`
- [ ] `examples/skills_demo/`
- [ ] `examples/session_state/`
- [ ] `examples/call_flow/`
- [ ] `examples/relay_demo/`
- [ ] `examples/rest_demo/`
- [ ] `examples/prefab_info_gatherer/`
- [ ] `examples/prefab_survey/`

## Phase 12: Testing

- [x] Test framework (standard `testing` package)
- [x] Logging tests (7 tests)
- [x] SWML tests — document, schema, service (28 tests)
- [x] Agent tests (66 tests)
- [x] SwaigFunctionResult tests (76 tests)
- [x] SessionManager tests (14 tests)
- [x] DataMap tests (19 tests)
- [x] Context/Step tests (42 tests)
- [x] Skill system tests (27 tests)
- [x] Prefab tests (31 tests)
- [x] AgentServer tests (27 tests)
- [x] RELAY client tests (43 tests)
- [x] REST client tests (20 tests)
- [ ] CI configuration

---

## Summary Stats

- **82 Go source files**
- **~14,600 lines of production code**
- **~8,400 lines of test code**
- **~400 tests passing across 14 packages**
- **9 external dependencies** (gorilla/websocket, google/uuid)

## Explicitly Excluded

- **Search/RAG system** — No vector/transformer models in Go
- **pgvector backend** — Depends on search system
- **sw-search CLI** — Depends on search system
- **BedrockAgent** — Can be added later if needed
