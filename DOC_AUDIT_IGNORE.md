# DOC_AUDIT_IGNORE

Every identifier listed here is intentionally skipped by
`porting-sdk/scripts/audit_docs.py`. Each line follows
`<name>: <rationale>` — the rationale explains *why* the identifier is
legitimately referenced in docs or examples without appearing in the Go
port surface.

Grouped by category. Keep the rationale concise.

## Go standard library — `fmt`

Printf: fmt.Printf formatted print to stdout
Println: fmt.Println line print to stdout
Sprintf: fmt.Sprintf formatted string
Fprintf: fmt.Fprintf formatted writer output
Fprintln: fmt.Fprintln writer line output

## Go standard library — `log`

Fatal: log.Fatal terminating error
Fatalf: log.Fatalf formatted terminating error

## Go standard library — `os` / `os/signal`

Getenv: os.Getenv environment variable lookup
Exit: os.Exit process termination

## Go standard library — `strconv`

Atoi: strconv.Atoi string-to-int conversion
ParseFloat: strconv.ParseFloat string-to-float conversion

## Go standard library — `strings`

Split: strings.Split token splitter
ToLower: strings.ToLower case conversion
ToUpper: strings.ToUpper case conversion

## Go standard library — `encoding/json`

MarshalIndent: json.MarshalIndent pretty JSON serialisation

## Go standard library — `time`

Now: time.Now current-time getter
Format: time.Time.Format timestamp formatting
Duration: time.Duration constructor (e.g. time.Duration(n)*time.Second)

## Go standard library — `context` / `os/signal`

Background: context.Background root context
WithTimeout: context.WithTimeout cancellable context
NotifyContext: signal.NotifyContext signal-aware context

## Go standard library — `math/rand`

Intn: rand.Intn random integer in range

## Go standard library — `sync`

Add: sync.WaitGroup.Add counter increment
Done: sync.WaitGroup.Done counter decrement

## Go standard library — `errors`

As: errors.As typed unwrap
Is: errors.Is sentinel comparison (docs/client-reference.md: errors.Is(err, relay.ErrDialTimeout)) — Go stdlib, sibling of errors.As

## Port-only illustrative references

Publish: illustrative PubSub.Publish reference inside a comment in examples/rest_demo/main.go
NewSignalWireClient: legacy pre-2.0 constructor kept as a "Before" example in docs/MIGRATION-2.0.md
ToolHandler: swaig.ToolHandler and agent.ToolHandler type references inside a comment in examples/skills_demo/main.go
fn: anonymous-struct field name (op.fn()) used to iterate a table-driven operation list in rest/examples/rest_calling_play_and_record.go
Uuid: real generated public type `type Uuid string` (pkg/rest/namespaces/relay_rest_types_generated.go); it is the declared type of id fields such as CallingNamespaceUpdateParams.Id, so docs must write namespaces.Uuid(callID) to convert a string variable. The surface enumerator records structs/marked-enums/methods/functions only, not scalar named-type aliases, so this real type is invisible to audit_docs — not a doc bug or an absent symbol.

## Python standard library referenced from legacy Python code blocks

The top-level `docs/*.md` files carry over Python code blocks from the
upstream Python SDK while the Go-native rewrite is in progress. These
references are Python stdlib methods that appear inside those blocks.

## Python-SDK method names in legacy Python code blocks (top-level `docs/`)

These are Python-SDK method names referenced in ```python``` fences inside
the top-level `docs/*.md` files. The Go port implements the same behaviour
under Go-idiomatic CamelCase identifiers (which the audit resolves against
`port_surface_go.json`). Each line below documents that the snake_case
name is the Python-reference equivalent of the corresponding Go method.
The long-term fix is to rewrite each block to Go; see PORT_OMISSIONS.md for
the subset deliberately not ported. Until that rewrite lands, these names
are non-claims of Go API.

add_hints: Python AgentBase.add_hints — Go AgentBase.AddHints
add_language: Python AgentBase.add_language — Go AgentBase.AddLanguage
add_section: Python PromptMixin.add_section — Go AgentBase.PromptAddSection
body: Python Section attribute name in docs/api_reference.md python block
connect: Python FunctionResult.connect — Go FunctionResult.Connect
debug: Python logger.debug level method in docs/swml_service_guide.md python block
description: Python docstring keyword shown as a field in docs/api_reference.md
error: Python logger.error level method in docs/agent_guide.md python block
error_keys: Python DataMap keyword shown in docs/api_reference.md
expression: Python DataMap.expression keyword shown in docs/api_reference.md
foreach: Python DataMap.foreach keyword shown in docs/api_reference.md
get_config: Python config helper illustrated in docs/configuration.md python block
hangup: Python FunctionResult.hangup — Go FunctionResult.Hangup
hold: Python FunctionResult.hold — Go FunctionResult.Hold
info: Python logger.info level method in docs/agent_guide.md python block
output: Python DataMap.output keyword shown in docs/api_reference.md
parameter: Python DataMap keyword shown in docs/api_reference.md
params: Python DataMap keyword shown in docs/api_reference.md
pay: Python FunctionResult.pay — Go FunctionResult.Pay
play_background_file: Python FunctionResult.play_background_file — Go FunctionResult.PlayBackgroundFile
purpose: Python DataMap field name shown in docs/api_reference.md
record_call: Python FunctionResult.record_call — Go FunctionResult.RecordCall
register: Python AgentServer.register — Go AgentServer.Register
register_routing_callback: Python SWMLService.register_routing_callback — Go Service.RegisterRoutingCallback
replace_in_history: Python FunctionResult.replace_in_history — Go FunctionResult.ReplaceInHistory
run: Python AgentBase.run — Go AgentBase.Run
say: Python FunctionResult.say — Go FunctionResult.Say
send_sms: Python FunctionResult.send_sms — Go FunctionResult.SendSms
serve: Python AgentBase.serve — Go AgentBase.Serve
set_functions: Python SWAIG keyword shown in docs/api_reference.md
set_global_data: Python AgentBase.set_global_data — Go AgentBase.SetGlobalData
set_params: Python AgentBase.set_params — Go AgentBase.SetParams
set_post_prompt_llm_params: Python AgentBase.set_post_prompt_llm_params — Go AgentBase.SetPostPromptLlmParams
set_prompt: Python PromptMixin.set_prompt — Go AgentBase.SetPromptText (renamed)
set_prompt_llm_params: Python AgentBase.set_prompt_llm_params — Go AgentBase.SetPromptLlmParams
set_text: Python Section.set_text — Go Section.SetText
set_valid_contexts: Python Context.set_valid_contexts — Go Context.SetValidContexts
set_valid_steps: Python Context.set_valid_steps — Go Context.SetValidSteps
setup: Python skills system hook referenced in docs/architecture.md
start: Python web-service start — FastAPI/uvicorn illustrated in docs/security.md
stop: Python FunctionResult.stop — Go FunctionResult.Stop
stop_record_call: Python FunctionResult.stop_record_call — Go FunctionResult.StopRecordCall
stop_tap: Python FunctionResult.stop_tap — Go FunctionResult.StopTap
swml_transfer: Python FunctionResult.swml_transfer — Go FunctionResult.SwmlTransfer
tap: Python FunctionResult.tap — Go FunctionResult.Tap
toggle_functions: Python FunctionResult.toggle_functions — Go FunctionResult.ToggleFunctions
tool: Python @tool decorator reference in docs/agent_guide.md python block
update: Python skill update method illustrated in docs/skills_parameter_schema.md
wait_for_user: Python FunctionResult.wait_for_user — Go FunctionResult.WaitForUser
warning: Python logger.warning level method in docs/agent_guide.md python block
webhook: Python SWAIG.webhook field name in docs/api_reference.md

## Go stdlib referenced by harness/example code

The audit harnesses (relay_audit_harness, skills_audit_harness,
rest_audit_harness) and other examples make heavy use of Go stdlib
methods that aren't part of the SignalWire surface. Each line below
documents the stdlib origin so a reviewer can spot a real symbol typo.

After: time.After channel-based timeout
Close: io.Closer.Close (used on http.Response.Body, ws.Conn, etc.)
Do: http.Client.Do
Encode: encoding/json.Encoder.Encode
GetString: encoding/json or stdlib accessor in skills_audit_harness
Grow: strings.Builder.Grow capacity hint
HasPrefix: strings.HasPrefix
Handler: http.Handler interface or net/http.Handler type
Index: strings.Index
IndexByte: strings.IndexByte
Load: sync/atomic.Bool.Load / atomic.Value.Load
New: errors.New / time.New / generic stdlib constructor
NewEncoder: encoding/json.NewEncoder
NewHandler: lambda.NewHandler / generic stdlib New constructors
NewRequest: net/http.NewRequest
ReadAll: io.ReadAll
Set: http.Header.Set / url.Values.Set
SetEscapeHTML: encoding/json.Encoder.SetEscapeHTML
Sleep: time.Sleep
Sprint: fmt.Sprint
Store: sync/atomic.Bool.Store / atomic.Value.Store
Switch: dynamic dispatch keyword (not a method) — false-positive in audit
ToMap: anonymous toString-like helper sometimes appearing in logging code
TrimPrefix: strings.TrimPrefix
TrimRight: strings.TrimRight
TrimSpace: strings.TrimSpace
Unmarshal: encoding/json.Unmarshal
WriteByte: strings.Builder.WriteByte / bytes.Buffer.WriteByte
WriteString: strings.Builder.WriteString / bytes.Buffer.WriteString

## Go SDK constructors / methods (Python-name-mapped surface gap)

The Go port's `port_surface.json` is Python-shaped (Go struct methods are
mapped onto Python class methods so diff_port_surface can compare).
Go-idiomatic constructors (`NewXxx`) map to Python's `__init__`, which
the audit's CamelCase translation doesn't cover. The names below are
real Go SDK exports, listed to acknowledge the audit's translation
limitation (NOT to hide undefined symbols).

NewAgentBase: agent.NewAgentBase — Python AgentBase.__init__ equivalent
NewAgentServer: server.NewAgentServer — Python AgentServer.__init__
NewConciergeAgent: prefabs.NewConciergeAgent — Python ConciergeAgent.__init__
NewFAQBotAgent: prefabs.NewFAQBotAgent — Python FAQBotAgent.__init__
NewFunctionResult: swaig.NewFunctionResult — Python FunctionResult.__init__
NewInfoGathererAgent: prefabs.NewInfoGathererAgent — Python InfoGathererAgent.__init__
NewReceptionistAgent: prefabs.NewReceptionistAgent — Python ReceptionistAgent.__init__
NewRelayClient: relay.NewRelayClient — Python RelayClient.__init__
NewRestClient: rest.NewRestClient — Python RestClient.__init__
NewService: swml.NewService — Python SWMLService.__init__
NewSkillManager: skills.NewSkillManager — Python SkillManager.__init__
NewSurveyAgent: prefabs.NewSurveyAgent — Python SurveyAgent.__init__
NewCallStateEvent: relay.NewCallStateEvent — factory for Python CallStateEvent.from_payload

## Go With* options (Python uses keyword args; audit can't map kwargs)

The Python SDK uses `__init__(name=..., route=...)` keyword args. Go
uses `New(WithName(...), WithRoute(...))` functional options. The
audit can't bridge that idiom — every functional option below is a
real Go SDK export (one per Python kwarg) but doesn't appear under a
Python class name.

WithAIParams: relay/rest WithAI option (e.g. AI.Hold + ai_params)
WithAIPrompt: rest CallingNamespace.WithAIPrompt option
WithAutoAnswer: rest WithAutoAnswer option
WithBasicAuth: swml.WithBasicAuth option
WithConferenceBeep: rest WithConferenceBeep option
WithConferenceMuted: rest WithConferenceMuted option
WithConfirm: rest WithConfirm option
WithConnectRingback: rest WithConnectRingback option
WithContexts: relay.WithContexts option
WithDialFromNumber: relay.WithDialFromNumber option
WithDialTimeout: relay.WithDialTimeout option
WithFunctions: rest WithFunctions option
WithHost: swml.WithHost option
WithMaxActiveCalls: relay.WithMaxActiveCalls option
WithMessageMedia: relay.WithMessageMedia option
WithMessageRegion: relay.WithMessageRegion option
WithMessageTags: relay.WithMessageTags option
WithName: swml.WithName option
WithPort: swml.WithPort option
WithProject: relay.WithProject / rest.WithProject option
WithRecordCall: rest WithRecordCall option
WithRecordDirection: rest WithRecordDirection option
WithRecordFormat: rest WithRecordFormat option
WithRecordStereo: rest WithRecordStereo option
WithRoute: swml.WithRoute option
WithServerPort: server.WithServerPort option
WithSpace: relay.WithSpace / rest.WithSpace option
WithStreamCodec: rest WithStreamCodec option
WithToken: relay.WithToken / rest.WithToken option
WithType: rest WithType option

## Audit harness / example helper methods

These are Go-SDK methods added during this port's audit work. They
exist in code but the surface enumerator chose not to map them to
Python names (no Python equivalent) — the audit treats them as
unresolved. Each is a legitimate Go SDK export documented in code.

Notify: relay.Client.Notify — fire-and-forget JSON-RPC notify
OnEvent: relay.Client.OnEvent — generic event hook
RegisterTools: skills.SkillBase.RegisterTools — listed in port_surface.json under SkillBase
RenderPretty: swml.Document.RenderPretty — pretty-print method
RenderSWML: agent.AgentBase.RenderSWML — SWML rendering entry point
Response: swaig.FunctionResult.Response — accessor method
Setup: skills.SkillBase.Setup — listed in port_surface.json under SkillBase
SetBaseURL: rest.HttpClient.SetBaseURL / RestClient.SetBaseURL — base URL override

## Other Go-idiomatic surface

AI: top-level constants/keyword (e.g. AI.Hold) — appears in relay docs
AIHold: rest CallingNamespace.AIHold method
AIMessage: rest CallingNamespace.AIMessage method
AIStop: rest CallingNamespace.AIStop method
AIUnhold: rest CallingNamespace.AIUnhold method
CallID: relay.Call.CallID accessor — returned by Call construction
CreateSIPEndpoint: namespaces.SubscribersResource.CreateSIPEndpoint
DeleteSIPEndpoint: namespaces.SubscribersResource.DeleteSIPEndpoint
GetExecutionMode: lambda.GetExecutionMode — serverless detection
GetSIPEndpoint: namespaces.SubscribersResource.GetSIPEndpoint
GetSkillFactory: skills.GetSkillFactory — registry lookup
ListSIPEndpoints: namespaces.SubscribersResource.ListSIPEndpoints
Name: swaig.Tool.Name accessor / generic getter — false-positive
Prompt: agent.AgentBase.Prompt or contexts.Prompt accessor
Reason: relay event field accessor
Setup: skills.SkillBase.Setup — already listed; second hit-form ignored
SMS: messaging-related comment in examples
State: relay.Call.State accessor / FSM state
String: relay.Call.String / generic Stringer interface
UpdateSIPEndpoint: namespaces.SubscribersResource.UpdateSIPEndpoint
Version: skills.SkillBase.Version / agent.Version constant

## Go standard library / doc-example references (2026-07-06 doc conversion)

Abs: math.Abs — Go stdlib
Contains: strings.Contains — Go stdlib
Decode: json.Decoder.Decode — Go stdlib
NewDecoder: json.NewDecoder — Go stdlib
NewReader: strings/bytes.NewReader — Go stdlib
Errorf: fmt.Errorf — Go stdlib
Marshal: json.Marshal — Go stdlib
Join: strings.Join — Go stdlib
Parse: url.Parse / template.Parse — Go stdlib
Print: fmt.Print — Go stdlib
Handle: http.ServeMux.Handle — Go stdlib
ListenAndServe: http.ListenAndServe — Go stdlib
NewServeMux: http.NewServeMux — Go stdlib
StripPrefix: http.StripPrefix — Go stdlib
SetBasicAuth: http.Request.SetBasicAuth — Go stdlib
Lock: sync.Mutex.Lock — Go stdlib
Unlock: sync.Mutex.Unlock — Go stdlib
RLock: sync.RWMutex.RLock — Go stdlib
RUnlock: sync.RWMutex.RUnlock — Go stdlib
Seconds: time.Duration.Seconds — Go stdlib
Since: time.Since — Go stdlib
Setenv: os.Setenv — Go stdlib
Stat: os.Stat — Go stdlib
go: Go `go` keyword highlighted in a code block — false-positive, not an identifier
init: Go language built-in package-init function referenced in a comment ("// Each imported package registers its skill via init().") in docs/third_party_skills.md — not an SDK symbol
SkillName: skills.SkillName typed-string conversion (e.g. skills.SkillName("weather")) — real port type used in AddSkill examples
NewCustomerSupportAgent: user-defined example agent constructor in docs/agent_guide.md
newMyAgent: doc-local example constructor defined and called within the same snippet (func newMyAgent() ... / func main() { _, _ = newMyAgent() }) in docs/skills_parameter_schema.md — not an SDK symbol
registerTools: user-defined method on an illustrative custom-prefab example in docs/architecture.md
testAPIConnection: user-defined helper method on an illustrative skill example in docs/third_party_skills.md
