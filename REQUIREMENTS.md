# SignalWire AI Agents Go SDK - Requirements & Progress Tracker

This document tracks the complete port of the SignalWire AI Agents SDK from Python/TypeScript to Go.

## Phase 1: Project Foundation

- [ ] Go module initialized (`github.com/signalwire/signalwire-agents-go`)
- [ ] Directory structure matching Python/TS layout
- [ ] `.gitignore` for Go projects
- [ ] `README.md` with project overview
- [ ] `CLAUDE.md` with development guidance
- [ ] Logging system (structured logging, levels, modes)
- [ ] Configuration system (env vars, config files, defaults)

## Phase 2: SWML Core

### SWMLService (Base Foundation)
- [ ] SWML document model (sections, verbs, rendering to JSON)
- [ ] Schema validation for SWML verbs (from schema.json)
- [ ] Auto-generated verb methods (answer, hangup, play, say, record, etc.)
- [ ] 5-phase call flow (pre-answer, answer, post-answer, AI, post-AI)
- [ ] HTTP server with routing (using standard library or chi)
- [ ] Basic auth (auto-generated credentials or env-based)
- [ ] Security headers (X-Content-Type-Options, X-Frame-Options, HSTS, CSP)
- [ ] CORS support
- [ ] Health/readiness endpoints (`/health`, `/ready`)
- [ ] Routing callbacks
- [ ] SIP username extraction from request body
- [ ] Proxy URL detection from request headers
- [ ] `SWML_PROXY_URL_BASE` env var support

### SWMLBuilder
- [ ] Fluent builder API for SWML documents
- [ ] Verb accumulation and section management
- [ ] `Build()` → map, `Render()` → JSON string
- [ ] `Reset()` for reuse

## Phase 3: Agent Core

### AgentBase
- [ ] Constructor with options (name, route, host, port, auth, etc.)
- [ ] Prompt management (raw text mode)
- [ ] Prompt Object Model (POM) builder (sections, bullets, subsections)
- [ ] `PromptAddSection()`, `PromptAddSubsection()`, `PromptAddToSection()`
- [ ] `SetPromptText()` / `SetPostPrompt()` for raw text
- [ ] Dynamic config callback (per-request agent customization)
- [ ] Ephemeral agent copies for dynamic config
- [ ] SWML rendering pipeline (5-phase document assembly)
- [ ] `OnRequest()` handler for SWML generation
- [ ] `OnSwmlRequest()` with request context
- [ ] `Run()` with auto-detection (server, CGI, Lambda, etc.)
- [ ] `Serve()` for direct HTTP server mode
- [ ] `AsRouter()` for embedding in larger servers
- [ ] Graceful shutdown support

### Tool/SWAIG System
- [ ] `DefineTool()` method (name, description, parameters, handler)
- [ ] Tool registration with JSON Schema parameter validation
- [ ] `RegisterSwaigFunction()` for raw function dicts (DataMap)
- [ ] `DefineTools()` to list all registered tools
- [ ] `OnFunctionCall()` handler for SWAIG webhook dispatch
- [ ] Secure tools with HMAC-SHA256 tokens (SessionManager)
- [ ] Token creation and validation with expiry
- [ ] Tool decorator equivalent (functional registration pattern)
- [ ] Type inference from Go function signatures → JSON Schema
- [ ] SWAIG query params support
- [ ] Webhook URL configuration

### SwaigFunctionResult (40+ actions)
- [ ] Constructor with response text and post_process flag
- [ ] `SetResponse()`, `SetPostProcess()`
- [ ] `AddAction()`, `AddActions()`
- [ ] `ToMap()` serialization

#### Call Control Actions
- [ ] `Connect()` (destination, final, from)
- [ ] `SwmlTransfer()` (dest, ai_response, final)
- [ ] `Hangup()`
- [ ] `Hold()` (timeout)
- [ ] `WaitForUser()` (enabled, timeout, answer_first)
- [ ] `Stop()`

#### State & Data Management
- [ ] `UpdateGlobalData()` / `RemoveGlobalData()`
- [ ] `SetMetadata()` / `RemoveMetadata()`
- [ ] `SwmlUserEvent()`
- [ ] `SwmlChangeStep()` / `SwmlChangeContext()`
- [ ] `SwitchContext()` (system_prompt, user_prompt, consolidate, full_reset, isolated)
- [ ] `ReplaceInHistory()`

#### Media Control
- [ ] `Say()`
- [ ] `PlayBackgroundFile()` / `StopBackgroundFile()`
- [ ] `RecordCall()` / `StopRecordCall()`

#### Speech & AI Config
- [ ] `AddDynamicHints()` / `ClearDynamicHints()`
- [ ] `SetEndOfSpeechTimeout()` / `SetSpeechEventTimeout()`
- [ ] `ToggleFunctions()` / `EnableFunctionsOnTimeout()`
- [ ] `EnableExtensiveData()`
- [ ] `UpdateSettings()`

#### Advanced Features
- [ ] `ExecuteSwml()`
- [ ] `JoinConference()` / `JoinRoom()`
- [ ] `SipRefer()`
- [ ] `Tap()` / `StopTap()`
- [ ] `SendSms()`
- [ ] `Pay()`

#### RPC Actions
- [ ] `ExecuteRpc()`
- [ ] `RpcDial()` / `RpcAiMessage()` / `RpcAiUnhold()`
- [ ] `SimulateUserInput()`

### DataMap (Server-side Tools)
- [ ] Fluent builder: `NewDataMap("name")`
- [ ] `Purpose()` / `Description()`
- [ ] `Parameter()` (name, type, description, required, enum)
- [ ] `Expression()` (test_value, pattern, output, nomatch_output)
- [ ] `Webhook()` (method, URL, headers, form_param, input_args_as_params, require_args)
- [ ] `WebhookExpressions()`
- [ ] `Body()` / `Params()`
- [ ] `Foreach()`
- [ ] `Output()` / `FallbackOutput()`
- [ ] `ErrorKeys()` / `GlobalErrorKeys()`
- [ ] `ToSwaigFunction()` serialization
- [ ] `CreateSimpleApiTool()` helper
- [ ] `CreateExpressionTool()` helper

### Contexts & Steps System
- [ ] `ContextBuilder` with `AddContext()` / `GetContext()` / `Validate()`
- [ ] `Context` with full API:
  - [ ] `AddStep()`, `GetStep()`, `RemoveStep()`, `MoveStep()`
  - [ ] `SetValidContexts()`, `SetValidSteps()`
  - [ ] `SetPostPrompt()`, `SetSystemPrompt()`, `SetPrompt()`
  - [ ] `SetConsolidate()`, `SetFullReset()`, `SetUserPrompt()`
  - [ ] `SetIsolated()`
  - [ ] `AddSection()`, `AddBullets()`, `AddSystemSection()`, `AddSystemBullets()`
  - [ ] Enter/exit fillers
  - [ ] `ToMap()` serialization
- [ ] `Step` with full API:
  - [ ] `SetText()`, `AddSection()`, `AddBullets()`
  - [ ] `SetStepCriteria()`, `SetFunctions()`
  - [ ] `SetValidSteps()`, `SetValidContexts()`
  - [ ] `SetEnd()`, `SetSkipUserTurn()`, `SetSkipToNextStep()`
  - [ ] GatherInfo support (questions, output_key, completion_action)
  - [ ] Reset settings (system_prompt, user_prompt, consolidate, full_reset)
  - [ ] `ToMap()` serialization
- [ ] `GatherInfo` and `GatherQuestion` types
- [ ] `CreateSimpleContext()` helper

### AI Configuration
- [ ] `AddHint()` / `AddHints()`
- [ ] `AddPatternHint()`
- [ ] `AddLanguage()` / `SetLanguages()`
- [ ] `AddPronunciation()` / `SetPronunciations()`
- [ ] `SetParam()` / `SetParams()`
- [ ] `SetGlobalData()` / `UpdateGlobalData()`
- [ ] `SetNativeFunctions()`
- [ ] `SetInternalFillers()` / `AddInternalFiller()`
- [ ] `EnableDebugEvents()`
- [ ] `AddFunctionInclude()` / `SetFunctionIncludes()`
- [ ] `SetPromptLlmParams()` / `SetPostPromptLlmParams()`

### Verb Management
- [ ] `AddPreAnswerVerb()` / `ClearPreAnswerVerbs()`
- [ ] `AddAnswerVerb()`
- [ ] `AddPostAnswerVerb()` / `ClearPostAnswerVerbs()`
- [ ] `AddPostAiVerb()` / `ClearPostAiVerbs()`

### SIP Routing
- [ ] `EnableSipRouting()`
- [ ] `RegisterSipUsername()`
- [ ] `AutoMapSipUsernames()`

### Web/HTTP
- [ ] `SetDynamicConfigCallback()`
- [ ] `ManualSetProxyUrl()`
- [ ] `SetWebHookUrl()` / `SetPostPromptUrl()`
- [ ] `AddSwaigQueryParams()` / `ClearSwaigQueryParams()`
- [ ] `EnableDebugRoutes()`

### Lifecycle Callbacks
- [ ] `OnSummary()` callback for post-prompt processing
- [ ] `OnDebugEvent()` callback

### Auth
- [ ] Basic auth with auto-generated credentials
- [ ] `SWML_BASIC_AUTH_USER` / `SWML_BASIC_AUTH_PASSWORD` env vars
- [ ] Auth validation per platform (HTTP, CGI, Lambda)

### Session/State
- [ ] SessionManager with HMAC-SHA256 tokens
- [ ] Token creation with call_id, function_name, expiry
- [ ] Token validation with timing-safe comparison
- [ ] Thread-safe global data (sync.RWMutex)

## Phase 4: Skills System

### SkillBase Interface
- [ ] `SkillBase` interface definition
- [ ] Required attributes: Name, Description, Version, RequiredPackages, RequiredEnvVars
- [ ] `Setup() bool`
- [ ] `RegisterTools()`
- [ ] `GetHints() []string`
- [ ] `GetGlobalData() map[string]any`
- [ ] `GetPromptSections() []map[string]any`
- [ ] `Cleanup()`
- [ ] `ValidateEnvVars() bool`
- [ ] `GetInstanceKey() string`
- [ ] `GetParameterSchema() map[string]any`
- [ ] `DefineTool()` wrapper
- [ ] Skill data namespacing (`GetSkillData`, `UpdateSkillData`)
- [ ] Multi-instance support (`SupportsMultipleInstances`)

### SkillManager
- [ ] `LoadSkill()` with validation pipeline
- [ ] `UnloadSkill()`
- [ ] `ListLoadedSkills()`
- [ ] `HasSkill()` / `GetSkill()`

### SkillRegistry
- [ ] On-demand skill loading
- [ ] Built-in skill discovery
- [ ] External skill directory support
- [ ] `ListSkills()` / `GetAllSkillsSchema()`
- [ ] `RegisterSkill()` / `AddSkillDirectory()`

### Built-in Skills (port all 18)
- [ ] datetime
- [ ] math
- [ ] joke
- [ ] weather_api
- [ ] web_search
- [ ] wikipedia_search
- [ ] google_maps
- [ ] spider
- [ ] datasphere
- [ ] datasphere_serverless
- [ ] swml_transfer
- [ ] play_background_file
- [ ] api_ninjas_trivia
- [ ] native_vector_search (network mode only, skip local vector)
- [ ] info_gatherer
- [ ] claude_skills
- [ ] mcp_gateway
- [ ] custom_skills

## Phase 5: Prefab Agents

- [ ] InfoGathererAgent (sequential field collection)
- [ ] SurveyAgent (multi-type questions with validation)
- [ ] ReceptionistAgent (department routing + transfer)
- [ ] FAQBotAgent (keyword matching + fallback)
- [ ] ConciergeAgent (venue/service information)

## Phase 6: AgentServer (Multi-Agent Hosting)

- [ ] `Register()` / `Unregister()` agents by route
- [ ] `GetAgents()` / `GetAgent()`
- [ ] `SetupSipRouting()` / `RegisterSipUsername()`
- [ ] `Run()` with platform auto-detection
- [ ] Health/readiness endpoints
- [ ] Security middleware
- [ ] Static file serving

## Phase 7: RELAY Client (WebSocket Call Control)

### Connection Management
- [ ] WebSocket connection to SignalWire Blade protocol
- [ ] JSON-RPC 2.0 message framing
- [ ] Auto-reconnect with exponential backoff (1s → 30s max)
- [ ] Half-open connection detection (server ping monitoring)
- [ ] `signalwire.disconnect` handling with restart flag
- [ ] Request queuing during disconnect
- [ ] Protocol/authorization_state session persistence

### Authentication
- [ ] Legacy auth (project_id + token)
- [ ] JWT auth
- [ ] Fast re-auth via authorization_state
- [ ] Context subscription/unsubscription

### Four Correlation Mechanisms
- [ ] JSON-RPC `id` → pending request futures (channels)
- [ ] `call_id` → Call object routing
- [ ] `control_id` → Action object tracking (per-call map)
- [ ] `tag` → dial correlation (async call_id resolution)

### Event ACK
- [ ] Immediate ACK for every `signalwire.event`
- [ ] Server ping response (`signalwire.ping`)

### Call Object (57+ methods)

#### Lifecycle
- [ ] `Answer()`
- [ ] `Hangup(reason)`
- [ ] `Pass()`
- [ ] `Transfer(dest)`

#### Audio
- [ ] `Play(media, volume, direction, loop)` → PlayAction
- [ ] `PlayAndCollect(media, collect, ...)` → CollectAction
- [ ] `Collect(digits, speech, ...)` → StandaloneCollectAction

#### Recording
- [ ] `Record(audio)` → RecordAction

#### Bridging
- [ ] `Connect(devices, ringback, tag, max_duration)`
- [ ] `Disconnect()`

#### DTMF
- [ ] `SendDigits(digits)`

#### Detection
- [ ] `Detect(detect, timeout)` → DetectAction

#### Advanced
- [ ] `SendFax()` / `ReceiveFax()` → FaxAction
- [ ] `Tap(tap, device)` → TapAction
- [ ] `Stream(url, name, codec)` → StreamAction
- [ ] `Pay(connector_url, amount, currency)` → PayAction
- [ ] `Transcribe(status_url)` → TranscribeAction
- [ ] `LiveTranscribe()` / `LiveTranslate()`
- [ ] `Refer(device, status_url)`
- [ ] `Echo(timeout)`

#### Conferencing
- [ ] `JoinConference()` / `LeaveConference()`

#### AI
- [ ] `AI(prompt, SWAIG, params)` → AIAction
- [ ] `AmazonBedrock()`
- [ ] `AIMessage()` / `AIHold()` / `AIUnhold()`

#### Hold/Denoise/Rooms/Queues
- [ ] `Hold()` / `Unhold()`
- [ ] `Denoise()` / `DenoiseStop()`
- [ ] `JoinRoom()` / `LeaveRoom()`
- [ ] `QueueEnter()` / `QueueLeave()`
- [ ] `BindDigit()` / `ClearDigitBindings()`
- [ ] `UserEvent()`

### Action Objects
- [ ] Base action: `Wait()`, `Stop()`, `IsDone()`, `Result()`, `Completed()`
- [ ] `PlayAction` (+ Pause, Resume, Volume)
- [ ] `RecordAction`
- [ ] `DetectAction`
- [ ] `CollectAction` (+ play_and_collect gotcha: filter by event type)
- [ ] `StandaloneCollectAction`
- [ ] `FaxAction`
- [ ] `TapAction`
- [ ] `StreamAction`
- [ ] `PayAction`
- [ ] `TranscribeAction`
- [ ] `AIAction`
- [ ] `on_completed` callback support

### Event System (22 event types)
- [ ] `CallStateEvent`
- [ ] `CallReceiveEvent`
- [ ] `PlayEvent`
- [ ] `RecordEvent`
- [ ] `CollectEvent`
- [ ] `ConnectEvent`
- [ ] `DetectEvent`
- [ ] `FaxEvent`
- [ ] `TapEvent`
- [ ] `StreamEvent`
- [ ] `SendDigitsEvent`
- [ ] `DialEvent`
- [ ] `ReferEvent`
- [ ] `DenoiseEvent`
- [ ] `PayEvent`
- [ ] `QueueEvent`
- [ ] `EchoEvent`
- [ ] `TranscribeEvent`
- [ ] `HoldEvent`
- [ ] `ConferenceEvent`
- [ ] `CallingErrorEvent`
- [ ] `MessageReceiveEvent` / `MessageStateEvent`

### Messaging (SMS/MMS)
- [ ] `SendMessage()` (to, from, body, media, context, tags)
- [ ] Message object with state tracking
- [ ] `Wait()` for delivery confirmation
- [ ] `OnMessage` handler for inbound messages
- [ ] Delivery state machine (queued→initiated→sent→delivered)

### Client Configuration
- [ ] `NewRelayClient(options)` constructor
- [ ] Context subscription
- [ ] `OnCall` handler registration
- [ ] `OnMessage` handler registration
- [ ] `Run()` / `Stop()`
- [ ] Concurrency limits (`RELAY_MAX_ACTIVE_CALLS`, `RELAY_MAX_CONNECTIONS`)
- [ ] Environment variables: PROJECT_ID, API_TOKEN, JWT_TOKEN, SPACE

## Phase 8: REST Client (HTTP API)

### Base HTTP Client
- [ ] `requests.Session` equivalent with connection pooling
- [ ] Basic Auth (project_id:token)
- [ ] JSON request/response handling
- [ ] Error handling (SignalWireRestError)
- [ ] Base URL construction from space hostname

### Resource Patterns
- [ ] `BaseResource` with path helpers
- [ ] `CrudResource` with List/Create/Get/Update/Delete
- [ ] Pagination support (iterator pattern)

### Namespaces (18+)
- [ ] `Fabric` (ai_agents, call_flows, subscribers, numbers, connectors, routes, etc.)
- [ ] `Calling` (37 command dispatch methods: dial, play, record, collect, etc.)
- [ ] `PhoneNumbers` (search, purchase, update, release)
- [ ] `Datasphere` (documents, chunks, search)
- [ ] `Video` (rooms, sessions, conferences, recordings, streams, tokens)
- [ ] `Compat` (Twilio-compatible LAML API)
- [ ] `Addresses`
- [ ] `Queues`
- [ ] `Recordings`
- [ ] `NumberGroups`
- [ ] `VerifiedCallers`
- [ ] `SipProfile`
- [ ] `Lookup`
- [ ] `ShortCodes`
- [ ] `ImportedNumbers`
- [ ] `MFA`
- [ ] `Registry`
- [ ] `Logs`
- [ ] `Project`
- [ ] `PubSub`
- [ ] `Chat`

### Client Constructor
- [ ] `NewSignalWireClient(project, token, host)` or env vars
- [ ] Lazy namespace initialization
- [ ] Shared HTTP client with connection pooling

## Phase 9: Serverless Support

- [ ] AWS Lambda handler adapter
- [ ] Google Cloud Functions adapter
- [ ] Azure Functions adapter
- [ ] CGI mode adapter
- [ ] Auto-detection of execution environment

## Phase 10: CLI Tools

- [ ] `swaig-test` equivalent
  - [ ] `--list-tools`
  - [ ] `--exec` tool execution
  - [ ] `--dump-swml` output
  - [ ] `--simulate-serverless`
  - [ ] Agent discovery from Go files

## Phase 11: Documentation & Examples

### Documentation (mirror Python/TS docs/)
- [ ] `docs/architecture.md`
- [ ] `docs/agent_guide.md`
- [ ] `docs/api_reference.md`
- [ ] `docs/swaig_reference.md`
- [ ] `docs/datamap_guide.md`
- [ ] `docs/contexts_guide.md`
- [ ] `docs/skills_system.md`
- [ ] `docs/cli_guide.md`
- [ ] `docs/security.md`
- [ ] `docs/configuration.md`

### Examples (mirror key examples)
- [ ] `examples/simple_agent/` — basic agent with tools
- [ ] `examples/simple_dynamic_agent/` — per-request configuration
- [ ] `examples/multi_agent_server/` — multiple agents, one server
- [ ] `examples/contexts_demo/` — multi-step workflows
- [ ] `examples/datamap_demo/` — server-side tools
- [ ] `examples/skills_demo/` — skills integration
- [ ] `examples/session_state/` — global data, callbacks
- [ ] `examples/call_flow/` — verb management, debug events
- [ ] `examples/relay_demo/` — RELAY call control
- [ ] `examples/rest_demo/` — REST API usage

## Phase 12: Testing

- [ ] Test framework setup (standard `testing` package)
- [ ] Core tests (SWMLService, AgentBase, SwaigFunctionResult)
- [ ] DataMap tests
- [ ] Context/Step tests
- [ ] Skill system tests
- [ ] Prefab tests
- [ ] RELAY client tests
- [ ] REST client tests
- [ ] Integration tests
- [ ] CI configuration

---

## Explicitly Excluded (Per Requirements)

- **Search/RAG system** — No vector/transformer models in Go
- **pgvector backend** — Depends on search system
- **sw-search CLI** — Depends on search system
- **BedrockAgent** — Can be added later if needed

---

## Go-Specific Design Decisions

### Architecture (Composition over Inheritance)
- Go has no class inheritance → use struct embedding and interfaces
- Python's 8 mixins → separate manager structs composed in AgentBase
- Method chaining → methods return pointer receiver `*AgentBase`

### Concurrency
- Python's asyncio → Go goroutines + channels
- Thread-safe global data → `sync.RWMutex`
- RELAY WebSocket → goroutine per connection, channels for correlation
- Action.Wait() → channel-based blocking

### Error Handling
- Python exceptions → Go `error` returns
- Validation errors → typed error types
- RELAY call-gone (404/410) → handled gracefully, not panics

### HTTP Server
- FastAPI/Hono → `net/http` with chi or standard mux
- Middleware pattern for auth, CORS, security headers
- `context.Context` propagation throughout

### Configuration
- Environment variables with `os.Getenv()`
- Functional options pattern for constructors
- Config struct with defaults

### Serialization
- JSON marshaling via `encoding/json`
- Struct tags for field mapping
- Custom marshalers for complex types (SwaigFunctionResult actions)

### Module Layout
```
github.com/signalwire/signalwire-agents-go/
├── pkg/
│   ├── swml/           # SWML document model + builder
│   ├── agent/          # AgentBase, AI config, prompts
│   ├── swaig/          # SwaigFunctionResult, tool registry
│   ├── datamap/        # DataMap builder
│   ├── contexts/       # ContextBuilder, Context, Step
│   ├── skills/         # SkillBase, SkillManager, built-in skills
│   ├── prefabs/        # InfoGatherer, Survey, Receptionist, FAQ, Concierge
│   ├── server/         # AgentServer (multi-agent hosting)
│   ├── relay/          # RELAY WebSocket client
│   ├── rest/           # REST HTTP client
│   ├── security/       # SessionManager, auth
│   └── logging/        # Structured logging
├── cmd/
│   └── swaig-test/     # CLI tool
├── examples/           # Example agents
├── docs/               # Documentation
├── internal/           # Internal packages (schema utils, etc.)
└── tests/              # Integration tests
```
