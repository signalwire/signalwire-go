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
WithCancel: context.WithCancel cancellable context
NotifyContext: signal.NotifyContext signal-aware context

## Go standard library — `math/rand`

Intn: rand.Intn random integer in range

## Go standard library — `sync`

Add: sync.WaitGroup.Add counter increment
Done: sync.WaitGroup.Done counter decrement

## Go standard library — `errors`

As: errors.As typed unwrap

## Port-only illustrative references

Publish: illustrative PubSub.Publish reference inside a comment in examples/rest_demo/main.go
NewSignalWireClient: legacy pre-2.0 constructor kept as a "Before" example in docs/MIGRATION-2.0.md
ToolHandler: swaig.ToolHandler and agent.ToolHandler type references inside a comment in examples/skills_demo/main.go
fn: anonymous-struct field name (op.fn()) used to iterate a table-driven operation list in rest/examples/rest_calling_play_and_record.go

## Python standard library referenced from legacy Python code blocks

The top-level `docs/*.md` files carry over Python code blocks from the
upstream Python SDK while the Go-native rewrite is in progress. These
references are Python stdlib methods that appear inside those blocks.

Thread: python threading.Thread (appears in docs/web_service.md Python code)
abspath: python os.path.abspath
basicConfig: python logging.basicConfig
getLogger: python logging.getLogger
setLevel: python logger.setLevel
isoformat: python datetime.isoformat
fromisoformat: python datetime.fromisoformat
total_seconds: python timedelta.total_seconds

## Python SDK dunder / protected helpers in legacy Python code blocks

Names starting with `__` or `_` that appear in Python code blocks in the
top-level `docs/*.md` files. These are internal Python-SDK conventions
(dunder constructors, private helpers) and never surface in the Go port.

__init__: python class constructor shown in Python code blocks
_check_basic_auth: python-SDK private helper illustrated in docs/agent_guide.md
_configure_instructions: python-SDK private helper illustrated in docs/agent_guide.md
_get_new_messages: python-SDK private helper illustrated in docs/agent_guide.md
_register_custom_tools: python-SDK private helper illustrated in docs/api_reference.md
_register_default_tools: python-SDK private helper illustrated in docs/agent_guide.md
_setup_contexts: python-SDK private helper illustrated in docs/api_reference.md
_setup_static_config: python-SDK private helper illustrated in docs/agent_guide.md
_test_api_connection: python-SDK private helper illustrated in docs/third_party_skills.md

## Python SDK setter-style method names in legacy Python code blocks

setGoal: python-SDK setter illustrated in docs/agent_guide.md Python code block
setInstructions: python-SDK setter illustrated in docs/agent_guide.md Python code block
setPersonality: python-SDK setter illustrated in docs/agent_guide.md Python code block

## Python-SDK method names in legacy Python code blocks (top-level `docs/`)

These are Python-SDK method names referenced in ```python``` fences inside
the top-level `docs/*.md` files. The Go port implements the same behaviour
under Go-idiomatic CamelCase identifiers (which the audit resolves against
`port_surface_go.json`). Each line below documents that the snake_case
name is the Python-reference equivalent of the corresponding Go method.
The long-term fix is to rewrite each block to Go; see PORT_OMISSIONS.md for
the subset deliberately not ported. Until that rewrite lands, these names
are non-claims of Go API.

add_action: Python AgentBase.add_action — Go SwaigFunctionResult.AddAction
add_actions: Python AgentBase.add_actions — Go SwaigFunctionResult.AddActions
add_answer_verb: Python AgentBase.add_answer_verb — Go AgentBase.AddAnswerVerb
add_application: Python SWAIG method — referenced in docs/swaig_reference.md python block
add_bullets: Python Section.add_bullets — Go Section.AddBullets
add_context: Python ContextBuilder.add_context — Go ContextBuilder.AddContext
add_directory: Python WebService.add_directory — Go WebService.AddDirectory equivalent
add_function_include: Python AgentBase.add_function_include — Go AgentBase.AddFunctionInclude
add_gather_question: Python GatherInfo.add_gather_question — Go GatherInfo.AddGatherQuestion
add_hint: Python AgentBase.add_hint — Go AgentBase.AddHint
add_hints: Python AgentBase.add_hints — Go AgentBase.AddHints
add_internal_filler: Python AgentBase.add_internal_filler — Go AgentBase.AddInternalFiller
add_language: Python AgentBase.add_language — Go AgentBase.AddLanguage
add_mcp_server: Python AgentBase.add_mcp_server — Go AgentBase.AddMcpServer
add_pattern_hint: Python AgentBase.add_pattern_hint — Go AgentBase.AddPatternHint
add_post_ai_verb: Python AgentBase.add_post_ai_verb — Go AgentBase.AddPostAiVerb
add_post_answer_verb: Python AgentBase.add_post_answer_verb — Go AgentBase.AddPostAnswerVerb
add_pre_answer_verb: Python AgentBase.add_pre_answer_verb — Go AgentBase.AddPreAnswerVerb
add_pronunciation: Python AgentBase.add_pronunciation — Go AgentBase.AddPronunciation
add_section: Python PromptMixin.add_section — Go AgentBase.PromptAddSection
add_skill: Python SkillMixin.add_skill — Go AgentBase.AddSkill
add_step: Python Context.add_step — Go Context.AddStep
add_swaig_query_params: Python AgentBase.add_swaig_query_params — Go AgentBase.AddSwaigQueryParams
add_verb: Python SWMLService.add_verb — Go Service.ExecuteVerb
add_verb_to_section: Python SWMLService.add_verb_to_section — Go Service.ExecuteVerbToSection
alert_ops_team: Python user-defined function illustrated in docs/api_reference.md python block
allow_functions: Python SWAIG param shown in docs/api_reference.md python block
apply_custom_config: Python user-defined method illustrated in docs/agent_guide.md python block
apply_default_config: Python user-defined method illustrated in docs/agent_guide.md python block
as_router: Python WebMixin.as_router — Go AgentBase.AsRouter
body: Python Section attribute name in docs/api_reference.md python block
build_document: Python SWMLService.build_document — user hook in docs/swml_service_guide.md python block
build_voicemail_document: Python user-defined method in docs/swml_service_guide.md python block
clear_swaig_query_params: Python AgentBase.clear_swaig_query_params — Go AgentBase.ClearSwaigQueryParams
connect: Python FunctionResult.connect — Go FunctionResult.Connect
create_payment_action: Python FunctionResult.create_payment_action — Go FunctionResult.CreatePaymentAction
create_payment_parameter: Python FunctionResult.create_payment_parameter — Go FunctionResult.CreatePaymentParameter
create_payment_prompt: Python FunctionResult.create_payment_prompt — Go FunctionResult.CreatePaymentPrompt
debug: Python logger.debug level method in docs/swml_service_guide.md python block
define_contexts: Python AgentBase.define_contexts — Go AgentBase.DefineContexts
define_tool: Python AgentBase.define_tool — Go AgentBase.DefineTool
delete_state: Python state manager method in docs/agent_guide.md python block
description: Python docstring keyword shown as a field in docs/api_reference.md
enable_debug_events: Python AgentBase.enable_debug_events — Go AgentBase.EnableDebugEvents
enable_extensive_data: Python FunctionResult.enable_extensive_data — Go FunctionResult.EnableExtensiveData
enable_functions_on_timeout: Python FunctionResult.enable_functions_on_timeout — Go FunctionResult.EnableFunctionsOnTimeout
enable_mcp_server: Python AgentBase.enable_mcp_server — Go AgentBase.EnableMcpServer
enable_record_call: Python-only toggle illustrated in docs/sdk_features.md
enable_sip_routing: Python AgentBase.enable_sip_routing — Go AgentBase.EnableSipRouting
error: Python logger.error level method in docs/agent_guide.md python block
error_keys: Python DataMap keyword shown in docs/api_reference.md
execute_swml: Python FunctionResult.execute_swml — Go FunctionResult.ExecuteSwml
expression: Python DataMap.expression keyword shown in docs/api_reference.md
fallback_output: Python DataMap keyword shown in docs/api_reference.md
foreach: Python DataMap.foreach keyword shown in docs/api_reference.md
get_config: Python config helper illustrated in docs/configuration.md python block
get_customer_config: Python user-defined helper in docs/agent_guide.md python block
get_customer_settings: Python user-defined helper in docs/agent_guide.md python block
get_customer_tier: Python user-defined helper in docs/agent_guide.md python block
get_document: Python SWMLService.get_document — Go Service.GetDocument
get_full_url: Python AgentBase.get_full_url — Python-only helper, see PORT_OMISSIONS.md
get_parameter_schema: Python SkillBase.get_parameter_schema illustrated in docs/skills_parameter_schema.md
get_section: Python config helper illustrated in docs/configuration.md python block
get_state: Python state manager method in docs/agent_guide.md python block
global_error_keys: Python DataMap keyword shown in docs/api_reference.md
handle_serverless_request: Python AgentBase.handle_serverless_request — Python-only helper, see PORT_OMISSIONS.md
hangup: Python FunctionResult.hangup — Go FunctionResult.Hangup
has_config: Python config helper illustrated in docs/configuration.md python block
has_skill: Python SkillMixin.has_skill — Go AgentBase.HasSkill
hold: Python FunctionResult.hold — Go FunctionResult.Hold
include_router: Python FastAPI router helper illustrated in docs/api_reference.md
info: Python logger.info level method in docs/agent_guide.md python block
is_valid_customer: Python user-defined helper in docs/agent_guide.md python block
join_conference: Python FunctionResult.join_conference — Go FunctionResult.JoinConference
join_room: Python FunctionResult.join_room — Go FunctionResult.JoinRoom
list_all_skill_sources: Python skills helper illustrated in docs/third_party_skills.md
list_skills: Python SkillMixin.list_skills — Go AgentBase.ListSkills
load_skill: Python SkillManager.load_skill — Python-internal hook referenced in docs/architecture.md
load_user_preferences: Python user-defined helper in docs/agent_guide.md python block
on_completion_go_to: Python Context keyword shown in docs/api_reference.md
on_debug_event: Python AgentBase.on_debug_event — Go AgentBase.OnDebugEvent
on_function_call: Python ToolMixin.on_function_call — Go AgentBase.OnFunctionCall
output: Python DataMap.output keyword shown in docs/api_reference.md
parameter: Python DataMap keyword shown in docs/api_reference.md
params: Python DataMap keyword shown in docs/api_reference.md
pay: Python FunctionResult.pay — Go FunctionResult.Pay
play_background_file: Python FunctionResult.play_background_file — Go FunctionResult.PlayBackgroundFile
prompt_add_section: Python PromptMixin.prompt_add_section — Go AgentBase.PromptAddSection
prompt_add_subsection: Python PromptMixin.prompt_add_subsection — Go AgentBase.PromptAddSubsection
prompt_add_to_section: Python PromptMixin.prompt_add_to_section — Go AgentBase.PromptAddToSection
purpose: Python DataMap field name shown in docs/api_reference.md
record_call: Python FunctionResult.record_call — Go FunctionResult.RecordCall
register: Python AgentServer.register — Go AgentServer.Register
register_customer_route: Python user-defined route in docs/swml_service_guide.md python block
register_default_tools: Python skills system hook referenced in docs/architecture.md
register_knowledge_base_tool: Python user-defined helper in docs/agent_guide.md python block
register_product_route: Python user-defined route in docs/swml_service_guide.md python block
register_routing_callback: Python SWMLService.register_routing_callback — Go Service.RegisterRoutingCallback
register_sip_username: Python AgentBase.register_sip_username — Go AgentBase.RegisterSipUsername
register_swaig_function: Python ToolMixin.register_swaig_function — Go AgentBase.RegisterSwaigFunction
register_tools: Python skills system hook referenced in docs/architecture.md
register_verb_handler: Python SWMLService.register_verb_handler — Python-only helper, see PORT_OMISSIONS.md
remove_directory: Python WebService.remove_directory — Go WebService equivalent
remove_global_data: Python FunctionResult.remove_global_data — Go FunctionResult.RemoveGlobalData
remove_metadata: Python FunctionResult.remove_metadata — Go FunctionResult.RemoveMetadata
remove_skill: Python SkillMixin.remove_skill — Go AgentBase.RemoveSkill
replace_in_history: Python FunctionResult.replace_in_history — Go FunctionResult.ReplaceInHistory
reset_document: Python SWMLService.reset_document — Go Service.ResetDocument
run: Python AgentBase.run — Go AgentBase.Run
say: Python FunctionResult.say — Go FunctionResult.Say
schedule_follow_up: Python user-defined helper in docs/api_reference.md python block
send_sms: Python FunctionResult.send_sms — Go FunctionResult.SendSms
send_to_analytics: Python user-defined helper in docs/agent_guide.md python block
serve: Python AgentBase.serve — Go AgentBase.Serve
set_consolidate: Python Context.set_consolidate — Go Context.SetConsolidate
set_dynamic_config_callback: Python WebMixin.set_dynamic_config_callback — Go AgentBase.SetDynamicConfigCallback
set_end_of_speech_timeout: Python FunctionResult.set_end_of_speech_timeout — Go FunctionResult.SetEndOfSpeechTimeout
set_full_reset: Python Context.set_full_reset — Go Context.SetFullReset
set_function_includes: Python AgentBase.set_function_includes — Go AgentBase.SetFunctionIncludes
set_functions: Python SWAIG keyword shown in docs/api_reference.md
set_gather_info: Python GatherInfo builder method — Go GatherInfo setter
set_global_data: Python AgentBase.set_global_data — Go AgentBase.SetGlobalData
set_internal_fillers: Python AgentBase.set_internal_fillers — Go AgentBase.SetInternalFillers
set_languages: Python AgentBase.set_languages — Go AgentBase.SetLanguages
set_metadata: Python FunctionResult.set_metadata — Go FunctionResult.SetMetadata
set_native_functions: Python AgentBase.set_native_functions — Go AgentBase.SetNativeFunctions
set_param: Python AgentBase.set_param — Go AgentBase.SetParam
set_params: Python AgentBase.set_params — Go AgentBase.SetParams
set_post_process: Python FunctionResult.set_post_process — Go FunctionResult.SetPostProcess
set_post_prompt: Python PromptMixin.set_post_prompt — Go AgentBase.SetPostPrompt
set_post_prompt_llm_params: Python AgentBase.set_post_prompt_llm_params — Go AgentBase.SetPostPromptLlmParams
set_post_prompt_url: Python AgentBase.set_post_prompt_url — Go AgentBase.SetPostPromptUrl
set_prompt: Python PromptMixin.set_prompt — Go AgentBase.SetPromptText (renamed)
set_prompt_llm_params: Python AgentBase.set_prompt_llm_params — Go AgentBase.SetPromptLlmParams
set_prompt_text: Python PromptMixin.set_prompt_text — Go AgentBase.SetPromptText
set_pronunciations: Python AgentBase.set_pronunciations — Go AgentBase.SetPronunciations
set_response: Python FunctionResult.set_response — Go FunctionResult.SetResponse
set_speech_event_timeout: Python FunctionResult.set_speech_event_timeout — Go FunctionResult.SetSpeechEventTimeout
set_step_criteria: Python Step.set_step_criteria — Go Step.SetStepCriteria equivalent
set_system_prompt: Python Context.set_system_prompt — Go Context.SetSystemPrompt
set_text: Python Section.set_text — Go Section.SetText
set_user_prompt: Python Context.set_user_prompt — Go Context.SetUserPrompt
set_valid_contexts: Python Context.set_valid_contexts — Go Context.SetValidContexts
set_valid_steps: Python Context.set_valid_steps — Go Context.SetValidSteps
set_web_hook_url: Python AgentBase.set_web_hook_url — Go AgentBase.SetWebHookUrl
setup: Python skills system hook referenced in docs/architecture.md
setup_google_search: Python skills helper illustrated in docs/skills_system.md
setup_sip_routing: Python AgentServer.setup_sip_routing — Go AgentServer.SetupSipRouting
simulate_user_input: Python FunctionResult.simulate_user_input — Go FunctionResult.SimulateUserInput
sip_refer: Python FunctionResult.sip_refer — Go FunctionResult.SipRefer
start: Python web-service start — FastAPI/uvicorn illustrated in docs/security.md
stop: Python FunctionResult.stop — Go FunctionResult.Stop
stop_background_file: Python FunctionResult.stop_background_file — Go FunctionResult.StopBackgroundFile
stop_record_call: Python FunctionResult.stop_record_call — Go FunctionResult.StopRecordCall
stop_tap: Python FunctionResult.stop_tap — Go FunctionResult.StopTap
switch_context: Python FunctionResult.switch_context — Go FunctionResult.SwitchContext
swml_change_context: Python FunctionResult.swml_change_context — Go FunctionResult.SwmlChangeContext
swml_change_step: Python FunctionResult.swml_change_step — Go FunctionResult.SwmlChangeStep
swml_transfer: Python FunctionResult.swml_transfer — Go FunctionResult.SwmlTransfer
tap: Python FunctionResult.tap — Go FunctionResult.Tap
to_dict: Python object.to_dict — Python convention; Go port marshals via encoding/json
to_swaig_function: Python SWAIG tool method illustrated in docs/api_reference.md
toggle_functions: Python FunctionResult.toggle_functions — Go FunctionResult.ToggleFunctions
tool: Python @tool decorator reference in docs/agent_guide.md python block
unload_skill: Python SkillManager.unload_skill — Python-internal hook referenced in docs/architecture.md
update: Python skill update method illustrated in docs/skills_parameter_schema.md
update_global_data: Python FunctionResult.update_global_data — Go FunctionResult.UpdateGlobalData
update_settings: Python FunctionResult.update_settings — Go FunctionResult.UpdateSettings
update_state: Python state manager method in docs/agent_guide.md python block
validate_env_vars: Python SkillBase.validate_env_vars — Python-only helper, see PORT_OMISSIONS.md
validate_packages: Python SkillBase.validate_packages — Python-only helper, see PORT_OMISSIONS.md
wait_for_user: Python FunctionResult.wait_for_user — Go FunctionResult.WaitForUser
warning: Python logger.warning level method in docs/agent_guide.md python block
webhook: Python SWAIG.webhook field name in docs/api_reference.md
webhook_expressions: Python DataMap keyword shown in docs/api_reference.md

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

NewAgent: livewire.NewAgent constructor (LiveKit-compat shim)
NewAgentBase: agent.NewAgentBase — Python AgentBase.__init__ equivalent
NewAgentServer: server.NewAgentServer — Python AgentServer.__init__
NewAgentSession: livewire.NewAgentSession — LiveKit-compat shim constructor
NewConciergeAgent: prefabs.NewConciergeAgent — Python ConciergeAgent.__init__
NewDocument: swml.NewDocument — Python Document.__init__
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
WithDescription: rest WithDescription option
WithDialFromNumber: relay.WithDialFromNumber option
WithDialTimeout: relay.WithDialTimeout option
WithFunctions: rest WithFunctions option
WithHost: swml.WithHost option
WithLLM: rest WithLLM option
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
WithReplyInstructions: livewire.WithReplyInstructions option
WithRoute: swml.WithRoute option
WithServerPort: server.WithServerPort option
WithSpace: relay.WithSpace / rest.WithSpace option
WithStreamCodec: rest WithStreamCodec option
WithStreamDirection: rest WithStreamDirection option
WithSTT: livewire.WithSTT option
WithToken: relay.WithToken / rest.WithToken option
WithTTS: livewire.WithTTS option
WithType: rest WithType option

## Audit harness / example helper methods

These are Go-SDK methods added during this port's audit work. They
exist in code but the surface enumerator chose not to map them to
Python names (no Python equivalent) — the audit treats them as
unresolved. Each is a legitimate Go SDK export documented in code.

Notify: relay.Client.Notify — fire-and-forget JSON-RPC notify
OnEvent: relay.Client.OnEvent — generic event hook
RTCSession: livewire.LiveServer.RTCSession — LiveKit-compat shim
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
FunctionTool: livewire.FunctionTool helper — LiveKit-compat shim
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
