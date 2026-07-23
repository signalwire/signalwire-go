# PORT_SIGNATURE_OMISSIONS.md

Documented signature divergences between this Go port and the Python
reference. Each entry excuses signature drift on a symbol that exists
in both. Names-only divergences live in PORT_OMISSIONS.md /
PORT_ADDITIONS.md and are inherited automatically.

Format:
    <fully.qualified.symbol>: <one-line rationale>

Every excused divergence names a specific, honest Go-idiom reason. There is
NO open "maintenance backlog" — the 2026-07 burndown either fixed the real
divergence (tightened a type, implemented a missing option/param, corrected a
wire key) or excused it under one of the genuine-Go-idiom categories below.
Each excused divergence falls into one of:

- **Functional options** (`go-idiom-options-collapse` / `go-variadic-options` /
  `go-idiom-options-struct`): Go collapses Python's many keyword args into one
  variadic `...XxxOption` arg (or a typed options struct). Wire bytes unchanged.
- **No keyword-only / no defaults** (`go-no-keyword-only` / `go-kwargs-catchall`
  / `go-variadic-optional-scalar` / `go-no-defaults-extension`): Go has no
  keyword-only params, no default-valued params, and no `**kwargs` catch-all;
  these become positionals, trailing variadic scalars, or are dropped when the
  canonical call supplies nothing.
- **Typed handlers / multi-return** (`go-typed-handler` / `go-multi-return`):
  Go uses concrete func-typed callbacks and `(T, error)`/tuple multi-returns
  where Python uses untyped callables / value tuples.
- **Factory constructors** (`go-factory-ctor` / `go-typed-factory` /
  `go-package-fn`): Go `NewX` factories and package-level functions in place of
  Python `__init__`/classmethods.
- **Sum types** (`go-multi-union`): a single static Go type cannot represent a
  Python `union<A,B>`; Go picks one member (wire-neutral).
- **Wire-neutral spellings** (`go-wire-neutral-string`): a Go bare `string`
  where the reference has a format-annotated `uuid`/newtype — same wire bytes.
- **Reference-oracle gaps** (`reference-oracle gap`): the symbol exists in the
  reference SOURCE but is absent from `python_signatures.json` (a griffe
  blindspot, e.g. the whole `signalwire.livewire` package); not a port defect.


## Idiom: Go NewX factory constructors

signalwire.agent_server.AgentServer.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.core.agent_base.AgentBase.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.core.contexts.Context.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.core.contexts.ContextBuilder.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.core.contexts.GatherInfo.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.core.contexts.GatherQuestion.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.core.contexts.Step.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.core.function_result.FunctionResult.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.core.security.session_manager.SessionManager.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.core.skill_base.SkillBase.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.core.skill_manager.SkillManager.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.core.swml_service.SWMLService.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.prefabs.concierge.ConciergeAgent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.prefabs.faq_bot.FAQBotAgent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.prefabs.info_gatherer.InfoGathererAgent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.prefabs.receptionist.ReceptionistAgent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.prefabs.survey.SurveyAgent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.call.AIAction.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.call.Action.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.call.Call.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.call.CollectAction.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.call.DetectAction.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.call.FaxAction.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.call.PayAction.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.call.PlayAction.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.call.RecordAction.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.call.StandaloneCollectAction.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.call.StreamAction.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.call.TapAction.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.call.TranscribeAction.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.client.RelayClient.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.CallReceiveEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.CallStateEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.CallingErrorEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.CollectEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.ConferenceEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.ConnectEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.DenoiseEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.DetectEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.DialEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.EchoEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.FaxEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.HoldEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.MessageReceiveEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.MessageStateEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.PayEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.PlayEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.QueueEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.RecordEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.ReferEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.RelayEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.SendDigitsEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.StreamEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.TapEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.event.TranscribeEvent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.relay.message.Message.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs

## Idiom: Go fluent API returns *Self for chaining

signalwire.agent_server.AgentServer.get_agents: Go fluent API returns *Self for chaining
signalwire.core.mixins.tool_mixin.ToolMixin.define_tools: Go fluent API returns *Self for chaining

## Idiom: PromptManager projects from AgentBase (fluent *AgentBase return)

signalwire.core.agent.prompt.manager.PromptManager.prompt_add_section: Go's PromptManager methods project from agent.AgentBase; Go returns *AgentBase for chaining and uses an options struct in place of Python's title/body/bullets/numbered/numbered_bullets/subsections kwargs
signalwire.core.agent.prompt.manager.PromptManager.prompt_add_to_section: Go's PromptManager methods project from agent.AgentBase; Go returns *AgentBase for chaining and uses an options struct in place of Python's title/body/bullet/bullets kwargs
signalwire.core.agent.prompt.manager.PromptManager.define_contexts: Go's PromptManager.DefineContexts returns *ContextBuilder for fluent chaining and takes no Python-style ``contexts`` dict (Go callers build via Builder pattern)

## Idiom: ToolRegistry projects from AgentBase (fluent *AgentBase return + ToolDefinition struct)

signalwire.core.agent.tools.registry.ToolRegistry.define_tool: Go's ToolRegistry methods project from agent.AgentBase; DefineTool accepts a single *ToolDefinition struct in place of Python's 13 kwargs and returns *AgentBase for chaining
signalwire.core.agent.tools.registry.ToolRegistry.get_function: Go's ToolRegistry returns the port's ``*ToolDefinition`` value type instead of Python's union of SWAIGFunction/dict (Python's untyped registry vs Go's typed one)
signalwire.core.agent.tools.registry.ToolRegistry.get_all_functions: Go's ToolRegistry returns ``map[string]*ToolDefinition`` instead of Python's union of SWAIGFunction/dict (Python's untyped registry vs Go's typed one)

## Idiom: Go typed-result returns vs Python serialized/dynamic returns

signalwire.core.skill_base.SkillBase.logger: type-class divergence; Go's Logger field is typed as *logging.Logger; Python returns the result of get_logger() helper (same role, different declared type)

## Idiom: Go typed options vs Python kwargs / typed signature divergences

signalwire.core.mixins.auth_mixin.AuthMixin.get_basic_auth_credentials: Go's GetBasicAuthCredentials returns the resolved auth string only (no include_source kwarg); Python supports an include_source flag that causes it to return a (user, pass, source) tuple
signalwire.core.security.security_utils.filter_sensitive_headers: type-idiom divergence — Python parametrizes the header dict with a generic ``_V`` TypeVar (``dict[str, _V]`` in and out); Go uses a concrete ``map[string]string``. Same wire behavior (headers are string→string); Go has no need for the value-type generic.

## POM (signalwire.pom.pom) — Go idiom

signalwire.pom.pom.PromptObjectModel.__init__: go-factory-ctor — Go uses NewPromptObjectModel() with no params; Python __init__ accepts an optional debug kwarg (Go logging is package-level, no per-instance debug flag)
signalwire.pom.pom.PromptObjectModel.add_section: go-variadic-options — Go takes (title string, opts ...SectionOption) using functional options (WithBody/WithBullets/WithNumbered/WithNumberedBullets); Python uses 5 named kwargs
signalwire.pom.pom.PromptObjectModel.from_json: go-package-fn — Go exposes pom.FromJSON(string) as a package-level constructor function (Go convention) where Python uses a classmethod accepting Union[str, dict]
signalwire.pom.pom.PromptObjectModel.from_yaml: go-package-fn — Go exposes pom.FromYAML(string) as a package-level constructor function (Go convention) where Python uses a classmethod accepting Union[str, dict]
signalwire.pom.pom.Section.__init__: go-factory-ctor — Go uses NewSection(title) plus functional-option mutators (WithBody/WithBullets/...); Python __init__ accepts title + 4 named kwargs
signalwire.pom.pom.Section.add_subsection: go-variadic-options — Go takes (title string, opts ...SectionOption); Python uses 5 named kwargs

## RELAY Call: functional-options idiom (2026-07 backlog burndown)
#
# Go's relay Call methods express Python's per-verb keyword arguments as the
# functional-options idiom (PORT_PHILOSOPHY_GO.md §Construction / Dave Cheney):
# `Method(<positional> , opts ...XxxOption)`. Each `With*` helper writes exactly
# one wire key, so the emitted RELAY frame is byte-identical to Python's — the
# divergence is purely the surface SHAPE (N keyword args collapse to one variadic
# options arg). The signature audit compares param count/kind; the wire bytes are
# policed by the shared mock (pkg/relay/*_mock_test.go). Every option set below was
# audited against call.py for full field coverage + correct wire keys during this
# burndown (which also FIXED three real wire bugs: bind_digit method→bind_method,
# amazon_bedrock own-RPC, ai_params nested under `params`; see git log + the new
# regression tests in actions_mock_test.go).

signalwire.relay.call.Call.play: go-idiom-options-collapse — Go Play(media, opts ...PlayOption) collapses Python's volume/direction/loop/control_id/on_completed keyword-only args into one variadic options arg; all fields available via With* helpers; same {play:[...],volume?} wire frame
signalwire.relay.call.Call.play_and_collect: go-idiom-options-collapse — Go PlayAndCollect(media, collect, opts ...PlayOption) collapses Python's volume/control_id/on_completed; same wire frame
signalwire.relay.call.Call.play_tts: go-idiom-options-collapse — Go PlayTTS(text, opts ...TTSOption) collapses Python's language/gender/voice/volume/on_completed; same {type:tts,params:{text,language?,gender?,voice?}} + sibling volume
signalwire.relay.call.Call.play_audio: go-idiom-options-collapse — Go PlayAudio(url, opts ...AudioOption) collapses Python's volume/on_completed; same {type:audio,params:{url}} + sibling volume
signalwire.relay.call.Call.play_ringtone: go-idiom-options-collapse — Go PlayRingtone(name, opts ...RingtoneOption) collapses Python's duration/volume/on_completed; same {type:ringtone,params:{name,duration?}} + sibling volume
signalwire.relay.call.Call.play_silence: go-idiom-options-collapse — Go PlaySilence(duration) takes duration positionally; Python's keyword-only on_completed is the sole omitted convenience callback; same {type:silence,params:{duration}} wire frame
signalwire.relay.call.Call.prompt_tts: go-idiom-options-collapse — Go PromptTTS(text, collect, opts ...TTSOption) collapses Python's language/gender/voice/volume/on_completed; same wire frame
signalwire.relay.call.Call.prompt_audio: go-idiom-options-collapse — Go PromptAudio(url, collect, opts ...AudioOption) collapses Python's volume/on_completed; same wire frame
signalwire.relay.call.Call.record: go-idiom-options-collapse — Go Record(opts ...RecordOption) collapses Python's audio/control_id/on_completed (plus Go's own convenience record knobs folded into the audio map); same {record:{audio:{...}}} wire frame
signalwire.relay.call.Call.connect: go-idiom-options-collapse — Go Connect(devices, opts ...ConnectOption) collapses Python's ringback/tag/max_duration/max_price_per_minute/status_url into one variadic options arg; all fields available via WithConnect* helpers; same calling.connect params
signalwire.relay.call.Call.stream: go-idiom-options-collapse — Go Stream(url, opts ...StreamOption) collapses Python's name/codec/track/status_url/status_url_method/authorization_bearer_token/custom_parameters/control_id into one variadic options arg; all wire fields available via WithStream* helpers; same calling.stream params
signalwire.relay.call.Call.join_conference: go-idiom-options-collapse — Go JoinConference(name, opts ...ConferenceOption) collapses Python's 19 keyword-only conference knobs into one variadic options arg; all calling.join_conference schema fields available via WithConference* helpers; same wire params
signalwire.relay.call.Call.pay: go-idiom-options-collapse — Go Pay(connectorURL, opts ...PayOption) collapses Python's 19 pay keyword-only args into one variadic options arg; all fields available via WithPay* helpers (input_method→"input"); same calling.pay params
signalwire.relay.call.Call.ai: go-idiom-options-collapse — Go AI(opts ...AIOption) collapses Python's agent/prompt/post_prompt/post_prompt_url/post_prompt_auth_*/global_data/pronounce/hints/languages/SWAIG/ai_params/control_id into one variadic options arg; all fields available via WithAI* helpers (ai_params nested under "params"); same calling.ai params
signalwire.relay.call.Call.amazon_bedrock: go-idiom-options-collapse — Go AmazonBedrock(opts ...AIOption) collapses Python's prompt/SWAIG/ai_params/global_data/post_prompt/post_prompt_url via the shared AIOption set; dispatches the dedicated calling.amazon_bedrock RPC; same wire params
signalwire.relay.call.Call.send_fax: go-idiom-options-collapse — Go SendFax(document, identity, opts ...FaxOption) collapses Python's header_info/control_id/on_completed; same calling.send_fax params
signalwire.relay.call.Call.receive_fax: go-idiom-options-collapse — Go ReceiveFax(opts ...FaxOption) collapses Python's control_id/on_completed; same calling.receive_fax params
signalwire.relay.call.Call.detect_answering_machine: go-idiom-options-collapse — Go DetectAnsweringMachine(opts ...AMDOption) collapses Python's keyword-only AMD tuning args (initial_timeout/end_silence_timeout/machine_voice_threshold/machine_words_threshold/detect_interruptions/detect_message_end/timeout) into one variadic options arg; same {type:machine,params:{…only-provided…}} detect media
signalwire.relay.call.Call.detect_digit: go-idiom-options-collapse — Go DetectDigit(opts ...DetectDigitOption) collapses Python's digits/timeout; same {type:digit,params:{digits?}} detect media
signalwire.relay.call.Call.detect_fax: go-idiom-options-collapse — Go DetectFax(opts ...DetectFaxOption) collapses Python's tone/timeout; same {type:fax,params:{tone?}} detect media
signalwire.relay.call.Call.detect: go-variadic-optionals — Go Detect(detect, timeout *float64, controlID ...string) models Python's keyword-only control_id via a trailing variadic-scalar and on_completed via the action's Wait()/On(); same calling.detect params
signalwire.relay.call.Call.tap: go-variadic-optionals — Go Tap(tap, device, controlID ...string) models Python's keyword-only control_id via a trailing variadic-scalar; on_completed via the returned TapAction; same calling.tap params
signalwire.relay.call.Call.transcribe: go-variadic-optionals — Go Transcribe(statusURL, controlID ...string) models Python's keyword-only control_id via a trailing variadic-scalar; on_completed via the returned TranscribeAction; same calling.transcribe params
signalwire.relay.call.Call.collect: go-idiom-options-struct — Go Collect(params *CollectParams) takes a typed options struct carrying Python's digits/speech/initial_timeout/partial_results/continuous/send_start_of_input/start_input_timers/control_id/on_completed fields; same calling.collect params

## RELAY Call: Python keyword-only / **kwargs args Go models positionally (go-idiom)
#
# Go has no keyword-only parameters and no **kwargs catch-all; Python's `*` and
# `**kwargs` become plain positional args (or are dropped when they carry no
# caller-visible field — the canonical call never supplies them). Wire frame is
# unchanged.

signalwire.relay.call.Call.ai_hold: go-no-keyword-only — Go AIHold(controlID, timeout, prompt string) takes Python's keyword-only timeout/prompt as positionals (Go has no keyword-only params); same calling.ai_hold params
signalwire.relay.call.Call.ai_unhold: go-no-keyword-only — Go AIUnhold(controlID, prompt string) takes Python's keyword-only prompt as a positional; same calling.ai_unhold params
signalwire.relay.call.Call.ai_message: go-no-keyword-only — Go AIMessage(controlID, text, role string, reset, globalData map) takes Python's keyword-only message_text/role/reset/global_data as positionals; same calling.ai_message params
signalwire.relay.call.Call.user_event: go-no-keyword-only — Go UserEvent(eventName string, extra ...map[string]any) takes Python's keyword `event` positionally and its **kwargs as a trailing variadic map; same user_event wire frame
signalwire.relay.call.Call.clear_digit_bindings: go-kwargs-catchall — Python clear_digit_bindings(*, realm, **kwargs); Go ClearDigitBindings(realm string) takes realm positionally, omits **kwargs; same wire frame
signalwire.relay.call.Call.live_translate: go-kwargs-catchall — Python live_translate(action, status_url, **kwargs); Go LiveTranslate(action map, statusURL string) omits **kwargs; same wire frame
signalwire.relay.call.Call.echo: go-kwargs-catchall — Python echo(timeout, status_url, **kwargs); Go Echo(timeout *float64, statusURL string) omits **kwargs; same calling.echo params
signalwire.relay.call.Call.refer: go-kwargs-catchall — Python refer(device, status_url, **kwargs); Go Refer(device map, statusURL string) omits **kwargs; same calling.refer params
signalwire.relay.call.Call.join_room: go-kwargs-catchall — Python join_room(name, status_url, **kwargs); Go JoinRoom(name, statusURL string) omits **kwargs; same wire frame
signalwire.relay.call.Call.bind_digit: go-kwargs-catchall — Python bind_digit(digits, bind_method, *, bind_params, realm, max_triggers, **kwargs); Go BindDigit(digits, method, bindParams, realm, maxTriggers) takes the keyword-only args positionally and omits **kwargs (emits the fixed bind_method wire key); same calling.bind_digit params
signalwire.relay.call.Call.send_digits: go-no-keyword-only — Go SendDigits(digits string) omits Python's optional control_id (Go auto-generates the control_id); same calling.send_digits digits
signalwire.relay.call.Call.queue_enter: go-no-keyword-only — Go QueueEnter(name, statusURL string) omits Python's optional caller-supplied control_id (auto-generated); same calling.queue.enter params
signalwire.relay.call.Call.queue_leave: go-no-keyword-only — Go QueueLeave(name, queueID, statusURL string) omits Python's optional caller-supplied control_id (auto-generated); same calling.queue.leave params
signalwire.core.security.webhook_middleware.validate: go-no-keyword-only — Go security.Validate(method, url string, headers map[string]string, body, signingKey string) takes Python's keyword-only signing_key as a trailing positional (Go has no keyword-only params); returns *WebhookRejection (nil=pass, {Status,Headers,Body}=reject) aliased to the oracle's optional<tuple<int,dict<string,string>,string>>; identical decomposed webhook-validation contract

## RELAY Call: typed handler / event-loop idioms (go-typed-handler)
signalwire.relay.call.Call.on: go-typed-handler — Go On(eventType string, handler func(*RelayEvent)) takes a concrete Go func value where Python's EventHandler is a class-typed callback; equivalent contract
signalwire.relay.call.Call.wait_for: go-context-signature — Go WaitFor(ctx context.Context, eventType string, predicate func(*RelayEvent) bool) uses Go's context idiom + a typed predicate; Python's wait_for(event_type, predicate, timeout) folds timeout into ctx and types the predicate/event as Go funcs — same wait contract

## RELAY Client: functional-options / typed-handler idioms
signalwire.relay.client.RelayClient.dial: go-idiom-options-collapse — Go Dial(devices, opts ...DialOption) collapses Python's tag/max_duration/dial_timeout into one variadic options arg (WithDial* helpers); same dial contract
signalwire.relay.client.RelayClient.send_message: go-idiom-options-collapse — Go SendMessage(to, from, body string, opts ...MessageOption) collapses Python's context/media/tags/region/on_completed into one variadic options arg (WithMessage* helpers); same messaging.send params
signalwire.relay.client.RelayClient.on_call: go-typed-handler — Go OnCall(handler func(*Call)) takes a concrete Go func value where Python's on_call registers a CallHandler class; Go returns void (register-only) vs Python returns the handler; equivalent registration contract
signalwire.relay.client.RelayClient.on_message: go-typed-handler — Go OnMessage(handler func(*Message)) takes a concrete Go func value where Python's on_message registers a MessageHandler class; Go returns void; equivalent registration contract

## AgentBase / mixins: options-struct + typed-handler + fluent idioms
signalwire.core.mixins.tool_mixin.ToolMixin.define_tool: go-idiom-options-struct — Go DefineTool(def ToolDefinition) accepts a single typed struct in place of Python's 11 kwargs (name/description/parameters/handler/secure/fillers/webhook_url/required/is_typed_handler/swaig_fields); returns *AgentBase for chaining; same tool registration
signalwire.core.mixins.ai_config_mixin.AIConfigMixin.add_language: go-idiom-options-struct — Go AddLanguage(config map) takes the language config as one map (AddLanguageTyped exposes the full name/code/voice/speech_fillers/function_fillers/engine/model/params arg list); same add_language wire shape
signalwire.core.mixins.ai_config_mixin.AIConfigMixin.add_mcp_server: go-idiom-options-struct — Go AddMcpServer(cfg MCPServerConfig) takes a typed config struct in place of Python's url/headers/resources/resource_vars kwargs; same MCP-server config
signalwire.core.mixins.ai_config_mixin.AIConfigMixin.add_pattern_hint: go-variadic-optional-scalar — Go AddPatternHint(hint, pattern, replace string, ignoreCase ...bool) models Python's optional ignore_case=False keyword via a trailing variadic-scalar (Go has no default-valued params); same pattern-hint wire shape
signalwire.core.mixins.ai_config_mixin.AIConfigMixin.add_pronunciation: go-variadic-optional-scalar — Go AddPronunciation(replace, withText string, ignoreCase ...bool) models Python's optional ignore_case=False keyword via a trailing variadic-scalar; same pronunciation wire shape
signalwire.core.mixins.prompt_mixin.PromptMixin.prompt_add_section: go-idiom-options-collapse — Go PromptAddSection(title, body string, bullets []string, opts ...) collapses Python's numbered/numbered_bullets/subsections keyword args into functional options; returns *AgentBase; same POM section
signalwire.core.mixins.prompt_mixin.PromptMixin.prompt_add_to_section: go-idiom-options-collapse — Go PromptAddToSection(title, body string, opts ...) collapses Python's singular bullet + bullets kwargs into functional options; returns *AgentBase; same POM append
signalwire.core.mixins.prompt_mixin.PromptMixin.define_contexts: go-fluent-builder — Go DefineContexts() takes no args and returns *ContextBuilder for fluent construction where Python define_contexts(contexts) takes a contexts dict and returns a union<AgentBase,ContextBuilder>; same context definition
signalwire.core.mixins.web_mixin.WebMixin.run: go-idiom-noargs — Go Run() runs the agent HTTP server; Python run(event, context, force_mode, host, port) folds the serving knobs into the agent/server config (Go host/port set via options); same serve behavior
signalwire.core.mixins.web_mixin.WebMixin.serve: go-idiom-noargs — Go Serve() serves with configured host/port; Python serve(host, port) passes them per-call (Go configures them on the agent); same serve behavior
signalwire.core.mixins.web_mixin.WebMixin.set_dynamic_config_callback: go-typed-handler — Go SetDynamicConfigCallback(cb DynamicConfigCallback) takes a concrete Go func-typed callback where Python takes an untyped callable; same callback contract
signalwire.core.agent_base.AgentBase.on_debug_event: go-typed-handler — Go OnDebugEvent(cb DebugEventHandler) takes a concrete Go func-typed handler and returns *AgentBase for chaining where Python's on_debug_event takes/returns a callable; same debug-event contract
signalwire.core.agent_base.AgentBase.on_summary: go-typed-handler — Go OnSummary(cb SummaryCallback) takes a func-typed callback in place of Python's (summary, raw_data) positional handler shape; same summary contract
signalwire.core.skill_manager.SkillManager.load_skill: go-idiom-typed — Go LoadSkill(skill SkillBase) takes a constructed SkillBase and returns (bool, string) multi-return where Python load_skill(skill_name, skill_class, params) takes a name+class+params triple and returns a tuple<bool,string>; same load-outcome contract
signalwire.prefabs.info_gatherer.InfoGathererAgent.set_question_callback: go-typed-handler — Go SetQuestionCallback takes a Go func-typed callback returning []Question where Python's callback returns a list of dicts; same question-callback contract

## SWMLService: Go-idiomatic serve/auth/routing signatures
signalwire.core.swml_service.SWMLService.serve: go-idiom-noargs — Go Serve() serves with configured host/port/TLS where Python serve(host, port, ssl_cert, ssl_key, ssl_enabled, domain) passes them per-call (Go configures them on the service); same serve behavior
signalwire.core.swml_service.SWMLService.get_basic_auth_credentials: go-multi-return — Go GetBasicAuthCredentials() returns (user, pass) as a two-value multi-return; Python get_basic_auth_credentials(include_source) has an include_source flag toggling a 2- vs 3-tuple return (GetBasicAuthCredentialsWithSource is the Go 3-value variant); same credential contract
signalwire.core.swml_service.SWMLService.register_routing_callback: go-idiom param-order — Go RegisterRoutingCallback(path string, cb swml.RoutingCallback) places the path first (consistent with every other Go registration method: RegisterVerbHandler, RegisterGlobalRoutingCallback) where Python register_routing_callback(callback_fn, path="/sip") places the callback first; the callback TYPE now matches exactly (callable<list<dict<string,any>,dict<string,any>>,optional<string>> = (body, headers) -> route|nil). Pure param-order swap, same routing registration (dotnet documents the identical (path, callback) swap).

## FunctionResult: functional-options + genuine port extension
signalwire.core.function_result.FunctionResult.join_conference: go-idiom-options-collapse — Go JoinConference(name, opts ...) collapses Python's 17 conference keyword args into functional options; same conference action
signalwire.core.function_result.FunctionResult.pay: go-idiom-options-collapse — Go Pay(connectorURL, opts ...) collapses Python's 18 pay keyword args into functional options; same pay action
signalwire.core.function_result.FunctionResult.record_call: go-idiom-options-collapse — Go RecordCall(controlID, stereo, format, direction, opts ...) collapses Python's terminators/beep/input_sensitivity/initial_timeout/end_silence_timeout/max_length/status_url tail into functional options; same record_call action
signalwire.core.function_result.FunctionResult.switch_context: go-no-defaults-extension — Go SwitchContext(systemPrompt, userPrompt string, consolidate, fullReset, isolated bool) adds a 5th `isolated` param — a real SignalWire context_switch wire field (Context.set_isolated in the reference; matches the php port's documented isolated extension). Go has no default-valued params, so it is a required positional rather than an optional keyword; same context_switch action
signalwire.core.function_result.FunctionResult.remove_global_data: go-multi-union — Python keys is union<list<string>,string> (a genuine sum type); a single static Go type cannot represent it, so Go narrows to the []string form (RemoveMetadata/RemoveGlobalData accept a slice); same remove_global_data action
signalwire.core.function_result.FunctionResult.remove_metadata: go-multi-union — Python keys is union<list<string>,string>; Go narrows to the []string form (a single static type cannot express the sum type); same remove_metadata action

## DataMap: multi-union + typed-pattern idioms
signalwire.core.data_map.DataMap.expression: go-multi-union — Python pattern is union<class:Pattern,string> (a compiled-regex OR a string); Go takes the string form (regex compiled from it) — a single static type cannot express the sum type; same expression wire shape
signalwire.core.data_map.create_expression_tool: go-typed-map — Go create_expression_tool takes dict<string,class:ExpressionPattern> where Python takes dict<string,tuple<string,FunctionResult>>; Go's ExpressionPattern struct carries the same (output, result) pair as a named type instead of a positional tuple; same tool shape

## Contexts: functional-options / options-struct idioms
signalwire.core.contexts.Context.add_step: go-fluent-builder — Go AddStep(name string) returns the step builder; Python add_step(name, task, bullets, criteria, functions, valid_steps) passes all step fields up-front (Go sets them fluently on the returned step); same step definition
signalwire.core.contexts.Step.add_gather_question: go-idiom-options-collapse — Go Step.AddGatherQuestion(key, question string, opts ...) collapses Python's type/confirm/prompt/functions keyword args into functional options; same gather-question definition
signalwire.core.contexts.GatherInfo.add_question: go-idiom-options — Go GatherInfo.AddQuestion(..., opts ...) models Python's **kwargs as trailing functional options; same question definition

## Top-level factory / registration functions (go-idiom)
signalwire.RestClient: go-typed-factory — Go RestClient(project, token, space string) is a typed factory taking the three concrete auth positionals where Python's RestClient(*args, **kwargs) is an untyped passthrough constructor; same client construction
signalwire.register_skill: go-typed-registration — Go RegisterSkill(name string, factory ...) registers a skill by name + factory where Python register_skill(skill_class) registers a class; Go's package-level registration idiom (no class objects); same registry effect
signalwire.agent_server.AgentServer.run: go-idiom-options-collapse — Go AgentServer.Run(opts ...RunOption) collapses Python's event/context/host/port serving args into functional options; same server run

## Top-level list_skills: reference-oracle gap
# signalwire/__init__.py defines BOTH list_skills() and list_skills_with_params(),
# but python_signatures records only list_skills_with_params — a dropped-symbol
# oracle gap. The Go port surfaces list_skills; excused as an oracle gap.
signalwire.list_skills: reference-oracle gap — signalwire.list_skills defined at signalwire/__init__.py:70 but absent from python_signatures (only list_skills_with_params recorded)

## Generated REST set_methods: uuid-format arg (go-wire-neutral-string)
# The reference types the set_method binding arg as the generated `uuid`
# scalar-format alias (a str with a format annotation); Go's generated set_method
# emits it as a bare `string`. Wire-identical (a uuid IS a string on the wire) — a
# wire-neutral spelling difference (RULES.md §2/§3), not a bug. The generator's
# optional set_method args (version/fallback_url/status_callback_url) are now emitted
# as *string params (the missing-param fix in cmd/generate-rest during this burndown).
signalwire.rest.namespaces.relay_rest_resources_generated.PhoneNumbers.set_call_flow: go-wire-neutral-string — flow_id bound field is uuid-format (gen:uuid) in the reference; Go emits it as a bare string (wire-identical); version optional arg now emitted as *string
signalwire.rest.namespaces.relay_rest_resources_generated.PhoneNumbers.set_ai_agent: go-wire-neutral-string — agent_id bound field is uuid-format (gen:uuid) in the reference; Go emits it as a bare string (wire-identical)

## Surface-reconcile signature idiom (2026-07: surface parity → 0)

# Constructors take a Go options struct / functional options, not Python kwargs.
signalwire.core.security_config.SecurityConfig.__init__: Go NewSecurityConfig() takes no args (loads defaults+env); Python takes config_file/service_name kwargs
signalwire.web.web_service.WebService.__init__: Go NewWebService(Options{...}) takes an options struct; Python takes a flat kwarg list
signalwire.core.swml_builder.SWMLBuilder.__init__: Go swml.NewService takes functional ServiceOptions; Python SWMLBuilder(service) takes an SWMLService
signalwire.core.swml_handler.VerbHandlerRegistry.__init__: Go swml.NewService takes functional options; Python VerbHandlerRegistry() takes none (the registry is an inline map on Service)

# Fluent builders return the receiver (*Self chaining); Python returns None/bool.
signalwire.core.swml_builder.SWMLBuilder.add_section: Go returns the builder (*PomBuilder/*Service) for chaining; Python returns None
signalwire.core.swml_builder.SWMLBuilder.reset: Go returns the builder for chaining; Python returns None

# Go accessors/handlers use Go-idiomatic types (structs, error tuples folded to
# multi-return, RawMessage) that differ from the Python signature shapes.
signalwire.agent_server.AgentServer.register_global_routing_callback: go-idiom param-order — Go RegisterGlobalRoutingCallback(path string, cb swml.RoutingCallback) places the path first (Go registration convention) where Python register_global_routing_callback(callback_fn, path) places the callback first; the callback TYPE now matches exactly (callable<list<dict<string,any>,dict<string,any>>,optional<string>>). Pure param-order swap.
signalwire.core.security.session_manager.SessionManager.set_session_metadata: Go SetSessionMetadata(sessionID, metadata map) stores a map and returns void; Python set_session_metadata(call_id,key,value) sets one key and returns bool
signalwire.core.pom_builder.PomBuilder.add_section: Go AddSection omits the nested `subsections` kwarg (subsections are added via AddSubsection); param shape differs
signalwire.core.pom_builder.PomBuilder.add_to_section: Go AddToSection takes (title, body, bullets); Python also accepts a singular `bullet` — folded into `bullets` in Go
signalwire.core.swml_handler.AIVerbHandler.build_config: Go BuildConfig(params map) takes the verb params as one map; Python spreads them as explicit kwargs
signalwire.core.swml_handler.VerbHandlerRegistry.get_handler: Go GetVerbHandler returns the VerbHandler interface; Python returns optional<SWMLVerbHandler>
signalwire.core.swml_handler.VerbHandlerRegistry.register_handler: Go RegisterVerbHandler takes the VerbHandler interface; Python takes an SWMLVerbHandler base
signalwire.core.swml_service.SWMLService.register_verb_handler: Go RegisterVerbHandler takes the VerbHandler interface; Python takes an SWMLVerbHandler base
signalwire.prefabs.info_gatherer.InfoGathererAgent.on_swml_request: Go OnSwmlRequest matches the DynamicConfigCallback shape (queryParams,bodyParams,headers,agent); Python on_swml_request(request_data,callback_path,request) differs
signalwire.relay.client.RelayClient.execute: Go Execute returns json.RawMessage (defer-decoded); Python returns a decoded dict<string,any>

# Surface-only skill methods: the reference records these in the signatures oracle
# but Go expresses them via RegisterTools / a tool handler, so there is no separate
# Go method with this exact signature (the surface symbol is present via the skill
# contract projection; the signature has no Go counterpart to compare).
signalwire.skills.api_ninjas_trivia.skill.ApiNinjasTriviaSkill.get_tools: Go returns the tool list via RegisterTools (no separate get_tools method)
signalwire.skills.play_background_file.skill.PlayBackgroundFileSkill.get_tools: Go returns the tool list via RegisterTools (no separate get_tools method)
signalwire.skills.weather_api.skill.WeatherApiSkill.get_tools: Go returns the tool list via RegisterTools (no separate get_tools method)
signalwire.skills.spider.skill.SpiderSkill.__init__: Go uses NewSpider factory; the reference records a per-skill __init__ signature Go expresses via the factory
signalwire.skills.wikipedia_search.skill.WikipediaSearchSkill.search_wiki: Go registers the wiki search as a tool handler (handleSearch), not a public search_wiki method

# mcp_gateway CLIENT skill (MCPGatewaySkill) — Go-idiom method rename (NOT an omission).
# Go implements ALL SIX oracle methods as real public methods on *MCPGatewaySkill
# (pkg/skills/builtin/mcp_gateway.go); each is the SAME canonical method, only spelled
# in Go's exported PascalCase. The go signature enumerator does not walk the builtin
# concrete-skill packages (it enumerates the core surface), so these snake↔PascalCase
# renames are reconciled here in the adapter rather than emitted — the wire/behaviour
# contract is identical (secure-default verify_ssl opt-in verified by verify_ssl_parity +
# tls_verify). go-idiom-pascalcase rename:
#   setup                 -> Setup()                    (skills.go:47)
#   register_tools        -> RegisterTools()            (skills.go:146)
#   get_global_data       -> GetGlobalData()            (skills.go:393)
#   get_hints             -> GetHints()                 (skills.go:407)
#   get_prompt_sections   -> GetPromptSections()        (skills.go:417)
#   get_parameter_schema  -> GetParameterSchema()       (skills.go:438)
signalwire.skills.mcp_gateway.skill.MCPGatewaySkill.setup: go-idiom-pascalcase rename — Go Setup() is the same canonical method; the go enumerator does not walk builtin concrete-skill packages, so the snake↔PascalCase rename is reconciled in the adapter
signalwire.skills.mcp_gateway.skill.MCPGatewaySkill.register_tools: go-idiom-pascalcase rename — Go RegisterTools() is the same canonical method (returns the tool list), reconciled in the adapter (enumerator does not walk builtin skills)
signalwire.skills.mcp_gateway.skill.MCPGatewaySkill.get_global_data: go-idiom-pascalcase rename — Go GetGlobalData() is the same canonical method, reconciled in the adapter (enumerator does not walk builtin skills)
signalwire.skills.mcp_gateway.skill.MCPGatewaySkill.get_hints: go-idiom-pascalcase rename — Go GetHints() is the same canonical method, reconciled in the adapter (enumerator does not walk builtin skills)
signalwire.skills.mcp_gateway.skill.MCPGatewaySkill.get_prompt_sections: go-idiom-pascalcase rename — Go GetPromptSections() is the same canonical method, reconciled in the adapter (enumerator does not walk builtin skills)
signalwire.skills.mcp_gateway.skill.MCPGatewaySkill.get_parameter_schema: go-idiom-pascalcase rename — Go GetParameterSchema() is the same canonical method, reconciled in the adapter (enumerator does not walk builtin skills)

## BedrockAgent (C2-BEDROCK, Wave 2): reference now HAS the signatures
# Cluster-1 C1-O1 added BedrockAgent to python_signatures (it was previously only
# in python_surface). The prior `reference-oracle gap` excuse is therefore STALE
# and removed. The Go implementation (pkg/agent/bedrock.go) surfaces these methods;
# any residual difference is go's named-parameter idiom, reconciled below, NOT an
# oracle gap. The 7 set_* / __repr__ methods now match the reference signature
# with NO excuse (removed). Only __init__ diverges — go collapses Python's 7
# construction kwargs into a single BedrockOptions struct (go's named-parameter idiom).
signalwire.agents.bedrock.BedrockAgent.__init__: go-idiom-options — Go NewBedrockAgent(opts BedrockOptions) collapses Python's name/route/system_prompt/voice_id/temperature/top_p/max_tokens kwargs into one options struct (pkg/agent/bedrock.go); same fields, same Bedrock defaults, wire/behaviour-neutral
signalwire.web.web_service.WebService.app: Python @property returning the FastAPI app; Go has no framework app handle (not surfaced)
signalwire.web.web_service.WebService.security: Go WebService.Security() accessor exists but the reference records it as a @property with a distinct signature; not part of the compared surface

## Surface-reconcile signature lockstep (2026-07 cleanup: removed stale surface omissions)
signalwire.core.agent.prompt.manager.PromptManager.logger: Go PromptManager exposes a Logger field; the reference records no logger signature
signalwire.core.agent.tools.registry.ToolRegistry.logger: Go ToolRegistry exposes a Logger field; the reference records no logger signature
signalwire.core.mixins.auth_mixin.AuthMixin.logger: Go exposes a Logger field; the reference records no logger signature
signalwire.core.mixins.state_mixin.StateMixin.logger: Go exposes a Logger field; the reference records no logger signature
signalwire.core.swml_builder.SWMLBuilder.logger: Go swml.Service exposes a Logger field projected onto SWMLBuilder; reference records no logger signature
signalwire.core.swml_handler.VerbHandlerRegistry.logger: Go swml.Service exposes a Logger field projected onto VerbHandlerRegistry; reference records no logger signature
signalwire.skills.registry.SkillRegistry.logger: reference records a SkillRegistry.logger the Go instance registry does not expose (package-level registration idiom)
signalwire.core.function_result.FunctionResult.create_payment_action: Go exposes swaig.CreatePaymentAction as a package helper (staticmethod placement); no instance method signature to compare
signalwire.core.function_result.FunctionResult.create_payment_parameter: Go exposes swaig.CreatePaymentParameter as a package helper; no instance method signature to compare
signalwire.core.function_result.FunctionResult.create_payment_prompt: Go exposes swaig.CreatePaymentPrompt as a package helper; no instance method signature to compare
signalwire.core.pom_builder.PomBuilder.from_sections: Go exposes pom.FromSections as a package helper (classmethod placement); no instance method signature to compare
signalwire.core.pom_builder.PomBuilder.pom: Go PomBuilder holds an unexported pom field; the reference records no `pom` accessor signature — surfaced only as an internal
signalwire.core.skill_base.SkillBase.register_tools: abstract contract method surfaced synthetically on SkillBase; concrete skills carry the signature
signalwire.core.skill_base.SkillBase.setup: abstract contract method surfaced synthetically on SkillBase; concrete skills carry the signature
signalwire.core.swml_builder.SWMLBuilder.build: build is surfaced synthetically (Go GetDocument/Render serve the build role); no distinct Go method signature
signalwire.core.swml_handler.AIVerbHandler.validate_config: Go ValidateConfig signature differs from the reference SWMLVerbHandler.validate_config shape
signalwire.core.swml_service.SWMLService.extract_sip_username: extract_sip_username surfaced synthetically (Go swml.ExtractSIPUsername package func); no instance-method signature
signalwire.prefabs.concierge.ConciergeAgent.on_summary: Go OnSummary matches the SummaryCallback shape (summary,rawData); reference on_summary signature differs
signalwire.prefabs.faq_bot.FAQBotAgent.on_summary: Go OnSummary matches the SummaryCallback shape; reference on_summary signature differs
signalwire.prefabs.receptionist.ReceptionistAgent.on_summary: Go OnSummary matches the SummaryCallback shape; reference on_summary signature differs
signalwire.prefabs.survey.SurveyAgent.on_summary: Go OnSummary matches the SummaryCallback shape; reference on_summary signature differs
signalwire.rest._base.BaseResource.__init__: Go namespaces.Resource is inline-initialised by namespace constructors; no public NewResource factory signature to compare
signalwire.utils.schema_utils.SchemaUtils.generate_method_body: Go GenerateMethodBody exists but the reference signatures oracle does not record it under this module
signalwire.utils.schema_utils.SchemaUtils.generate_method_signature: Go GenerateMethodSignature exists but the reference signatures oracle does not record it under this module
signalwire.core.agent.tools.type_inference.infer_schema: go-idiom typed-params-builder input — Python infer_schema reflects a handler func's signature/type-hints at runtime (callable input) to derive the schema; Go has no runtime func-signature reflection, so the typed declaration is supplied via the fluent swaig.Params builder — InferSchema(p *swaig.Params) returns the SAME 5-tuple (parameters, required, description, isTyped, hasRawData). Only param[0]'s input FORM differs (typed-builder vs callable); the schema-derivation role and return shape match exactly. (rust/java omit this as impossible; go realizes it via the builder — the create_typed_handler_wrapper cousin matches the reference signature verbatim.)
signalwire.rest._base.SignalWireRestTransportError.__init__: go-error-cause-wrap — the Go transport error is the same rest.SignalWireRestError struct (folded via a Transport bool discriminator); its NewSignalWireRestTransportError(cause, body, url, method) constructor takes one EXTRA leading `cause error` arg beyond the reference's (body, url, method) so the underlying net/context error is preserved for errors.Is/errors.Unwrap — the Go equivalent of Python's `raise SignalWireRestTransportError(...) from exc`, which Python expresses via __cause__ (not a constructor param). Body still defaults to cause.Error() when empty, so the reference's (body, url, method) roles all map through.
signalwire.rest._request_options.RequestOptions.__init__: go-struct-literal — Go's RequestOptions (plan 4.2) is a value struct with public fields (Timeout/Retries/RetryOnStatus/RetryBackoff/AbortSignal); a caller constructs it with a composite literal (rest.RequestOptions{Retries: intPtr(1)}), so there is no NewRequestOptions factory to project as __init__. The five optional reference kwargs are the five public fields (same functional surface); construction is the literal, not a constructor call. SURFACE-oracle-invisible (only merge() is in the surface oracle).
signalwire.rest._request_options.RequestOptions.abort_signal: go-ctx-abort-primitive — the reference's abort_signal is a cooperative-cancellation object (the private _AbortSignal protocol); Go's cancellation primitive IS context.Context, exposed as the public RequestOptions.AbortSignal field and threaded onto the outgoing request's context (per the cross-port design: 'go uses context.Context'). A stdlib context.Context field is not projected as an SDK-class accessor, so the reference's abort_signal accessor has no matching Go method signature. SURFACE-oracle-invisible (only merge() is in the surface oracle).
