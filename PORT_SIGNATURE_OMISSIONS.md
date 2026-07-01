# PORT_SIGNATURE_OMISSIONS.md

Documented signature divergences between this Go port and the Python
reference. Each entry excuses signature drift on a symbol that exists
in both. Names-only divergences live in PORT_OMISSIONS.md /
PORT_ADDITIONS.md and are inherited automatically.

Format:
    <fully.qualified.symbol>: <one-line rationale>

Excused divergences fall into:

1. **Idiom-level** (deliberate, not fixable without breaking Go API style):
   - Go uses NewX factory functions as constructors; param shapes follow
     Go conventions, not Python kwarg lists.
   - Go methods return *Self for fluent chaining; Python returns None.
   - Go has no defaults — every parameter is required.

2. **Port maintenance backlog** (tracked here; will be reduced as the Go
   port catches up to Python signature parity).


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
signalwire.livewire.Agent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.livewire.AgentSession.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
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
signalwire.rest.namespaces.datasphere_resources_generated.DatasphereDocuments.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.fabric_resources_generated.FabricTokens.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.project_resources_generated.ProjectTokens.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs

## Idiom: Go fluent API returns *Self for chaining

signalwire.agent_server.AgentServer.get_agents: Go fluent API returns *Self for chaining
signalwire.core.mixins.tool_mixin.ToolMixin.define_tools: Go fluent API returns *Self for chaining
signalwire.core.mixins.web_mixin.WebMixin.as_router: Go fluent API returns *Self for chaining
signalwire.core.swml_service.SWMLService.get_document: Go fluent API returns *Self for chaining

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
signalwire.core.mixins.web_mixin.WebMixin.on_swml_request: Go takes a typed *http.Request param; Python takes Optional[fastapi.Request] (FastAPI vs net/http binding difference)
signalwire.core.security.security_utils.filter_sensitive_headers: type-idiom divergence — Python parametrizes the header dict with a generic ``_V`` TypeVar (``dict[str, _V]`` in and out); Go uses a concrete ``map[string]string``. Same wire behavior (headers are string→string); Go has no need for the value-type generic.

## POM (signalwire.pom.pom) — Go idiom

signalwire.pom.pom.PromptObjectModel.__init__: go-factory-ctor — Go uses NewPromptObjectModel() with no params; Python __init__ accepts an optional debug kwarg (Go logging is package-level, no per-instance debug flag)
signalwire.pom.pom.PromptObjectModel.add_section: go-variadic-options — Go takes (title string, opts ...SectionOption) using functional options (WithBody/WithBullets/WithNumbered/WithNumberedBullets); Python uses 5 named kwargs
signalwire.pom.pom.PromptObjectModel.from_json: go-package-fn — Go exposes pom.FromJSON(string) as a package-level constructor function (Go convention) where Python uses a classmethod accepting Union[str, dict]
signalwire.pom.pom.PromptObjectModel.from_yaml: go-package-fn — Go exposes pom.FromYAML(string) as a package-level constructor function (Go convention) where Python uses a classmethod accepting Union[str, dict]
signalwire.pom.pom.Section.__init__: go-factory-ctor — Go uses NewSection(title) plus functional-option mutators (WithBody/WithBullets/...); Python __init__ accepts title + 4 named kwargs
signalwire.pom.pom.Section.add_subsection: go-variadic-options — Go takes (title string, opts ...SectionOption); Python uses 5 named kwargs

## Backlog: real signature divergences (418 symbols)

Real Go port maintenance — parameter renames, missing optionals,
type imprecisions. Triage in a separate sweep.

signalwire.RestClient: BACKLOG / param-count-mismatch/ reference has 2 param(s), port has 3/ reference=['args', 'kwargs'] port=['projec; return-mismatch/
signalwire.agent_server.AgentServer.run: BACKLOG / param-count-mismatch/ reference has 5 param(s), port has 2/ reference=['self', 'event', 'context', 'ho
signalwire.core.agent_base.AgentBase.on_debug_event: BACKLOG / param-mismatch/ param[1] (handler)/ name 'handler' vs 'cb'; type 'class/Callable' vs 'class/Debu; return-mismatch/ retur
signalwire.core.agent_base.AgentBase.on_summary: BACKLOG / param-count-mismatch/ reference has 3 param(s), port has 2/ reference=['self', 'summary', 'raw_data'] ; return-mismatch/
signalwire.core.contexts.Context.add_step: BACKLOG / param-count-mismatch/ reference has 7 param(s), port has 2/ reference=['self', 'name', 'task', 'bullet
signalwire.core.contexts.GatherInfo.add_question: BACKLOG / param-mismatch/ param[3] (kwargs)/ name 'kwargs' vs 'opts'; kind 'var_keyword' vs 'positional'; 
signalwire.core.contexts.Step.add_gather_question: BACKLOG / param-count-mismatch/ reference has 7 param(s), port has 4/ reference=['self', 'key', 'question', 'typ
signalwire.core.data_map.DataMap.expression: BACKLOG / param-mismatch/ param[2] (pattern)/ type 'union<class/Pattern,string>' vs 'string'; param-mismatch/ param[4] (nomatch_ou
signalwire.core.data_map.create_expression_tool: BACKLOG / param-mismatch/ param[1] (patterns)/ type 'dict<string,tuple<string,class/signalwire.core.functi; param-mismatch/ param[
signalwire.core.data_map.create_simple_api_tool: BACKLOG / param-mismatch/ param[3] (parameters)/ type 'optional<dict<string,class/Dict>>' vs 'dict<string,; param-mismatch/ param[
signalwire.core.function_result.FunctionResult.join_conference: BACKLOG / param-count-mismatch/ reference has 19 param(s), port has 3/ reference=['self', 'name', 'muted', 'beep
signalwire.core.function_result.FunctionResult.pay: BACKLOG / param-count-mismatch/ reference has 20 param(s), port has 3/ reference=['self', 'payment_connector_url
signalwire.core.function_result.FunctionResult.record_call: BACKLOG / param-count-mismatch/ reference has 12 param(s), port has 6/ reference=['self', 'control_id', 'stereo'
signalwire.core.function_result.FunctionResult.remove_global_data: BACKLOG / param-mismatch/ param[1] (keys)/ type 'union<list<string>,string>' vs 'list<string>'
signalwire.core.function_result.FunctionResult.remove_metadata: BACKLOG / param-mismatch/ param[1] (keys)/ type 'union<list<string>,string>' vs 'list<string>'
signalwire.core.function_result.FunctionResult.switch_context: BACKLOG / param-count-mismatch/ reference has 5 param(s), port has 6/ reference=['self', 'system_prompt', 'user_
signalwire.core.mixins.ai_config_mixin.AIConfigMixin.add_language: BACKLOG / param-count-mismatch/ reference has 8 param(s), port has 2/ reference=['self', 'name', 'code', 'voice'
signalwire.core.mixins.ai_config_mixin.AIConfigMixin.add_mcp_server: BACKLOG / param-count-mismatch/ reference has 5 param(s), port has 2/ reference=['self', 'url', 'headers', 'reso
signalwire.core.mixins.ai_config_mixin.AIConfigMixin.add_pattern_hint: BACKLOG / param-mismatch/ param[4] (ignore_case)/ type 'bool' vs 'list<bool>'; required False vs True; def
signalwire.core.mixins.ai_config_mixin.AIConfigMixin.add_pronunciation: BACKLOG / param-mismatch/ param[3] (ignore_case)/ type 'bool' vs 'list<bool>'; required False vs True; def
signalwire.core.mixins.prompt_mixin.PromptMixin.define_contexts: BACKLOG / param-count-mismatch/ reference has 2 param(s), port has 1/ reference=['self', 'contexts'] port=['self; return-mismatch/
signalwire.core.mixins.prompt_mixin.PromptMixin.prompt_add_section: BACKLOG / param-count-mismatch/ reference has 7 param(s), port has 5/ reference=['self', 'title', 'body', 'bulle
signalwire.core.mixins.prompt_mixin.PromptMixin.prompt_add_to_section: BACKLOG / param-count-mismatch/ reference has 5 param(s), port has 4/ reference=['self', 'title', 'body', 'bulle
signalwire.core.mixins.tool_mixin.ToolMixin.define_tool: BACKLOG / param-count-mismatch/ reference has 11 param(s), port has 2/ reference=['self', 'name', 'description',
signalwire.core.mixins.web_mixin.WebMixin.run: BACKLOG / param-count-mismatch/ reference has 6 param(s), port has 1/ reference=['self', 'event', 'context', 'fo
signalwire.core.mixins.web_mixin.WebMixin.serve: BACKLOG / param-count-mismatch/ reference has 3 param(s), port has 1/ reference=['self', 'host', 'port'] port=['; return-mismatch/
signalwire.core.mixins.web_mixin.WebMixin.set_dynamic_config_callback: BACKLOG / param-mismatch/ param[1] (callback)/ name 'callback' vs 'cb'; type 'callable<list<dict<any,any>,
signalwire.core.skill_manager.SkillManager.get_skill: BACKLOG / param-mismatch/ param[1] (skill_identifier)/ name 'skill_identifier' vs 'key'; return-mismatch/ returns 'optional<class/
signalwire.core.skill_manager.SkillManager.load_skill: BACKLOG / param-count-mismatch/ reference has 4 param(s), port has 2/ reference=['self', 'skill_name', 'skill_cl; return-mismatch/
signalwire.core.swml_service.SWMLService.get_basic_auth_credentials: BACKLOG / param-count-mismatch/ reference has 2 param(s), port has 1/ reference=['self', 'include_source'] port=; return-mismatch/
signalwire.core.swml_service.SWMLService.register_routing_callback: BACKLOG / param-mismatch/ param[1] (callback_fn)/ name 'callback_fn' vs 'path'; type 'callable<list<class/; param-mismatch/ param[
signalwire.core.swml_service.SWMLService.serve: BACKLOG / param-count-mismatch/ reference has 7 param(s), port has 1/ reference=['self', 'host', 'port', 'ssl_ce; return-mismatch/
signalwire.list_skills: BACKLOG / param-count-mismatch/ reference has 2 param(s), port has 0/ reference=['args', 'kwargs'] port=[]; return-mismatch/ retur
signalwire.livewire.AgentServer.rtc_session: BACKLOG / param-count-mismatch/ reference has 6 param(s), port has 3/ reference=['self', 'func', 'agent_name', '; return-mismatch/
signalwire.livewire.AgentSession.generate_reply: BACKLOG / param-mismatch/ param[1] (instructions)/ name 'instructions' vs 'opts'; kind 'keyword' vs 'posit; return-mismatch/ retur
signalwire.livewire.AgentSession.say: BACKLOG / return-mismatch/ returns 'any' vs 'void'
signalwire.livewire.AgentSession.start: BACKLOG / param-mismatch/ param[1] (agent)/ name 'agent' vs 'ctx'; type 'class/signalwire.livewire.Agent' ; param-mismatch/ param[
signalwire.livewire.plugins.SileroVAD.load: BACKLOG / param-mismatch/ param[0] (cls)/ name 'cls' vs 'self'; kind 'cls' vs 'self'; return-mismatch/ returns 'any' vs 'class/sig
signalwire.livewire.run_app: BACKLOG / return-mismatch/ returns 'any' vs 'void'
signalwire.prefabs.info_gatherer.InfoGathererAgent.set_question_callback: BACKLOG / param-mismatch/ param[1] (callback)/ name 'callback' vs 'cb'; type 'callable<list<dict<any,any>,; return-mismatch/ retur
signalwire.register_skill: BACKLOG / param-count-mismatch/ reference has 1 param(s), port has 2/ reference=['skill_class'] port=['name', 'f; return-mismatch/
signalwire.relay.call.Call.ai: BACKLOG / param-count-mismatch/ reference has 16 param(s), port has 2/ reference=['self', 'control_id', 'agent',
signalwire.relay.call.Call.ai_hold: BACKLOG / param-mismatch/ param[1] (timeout)/ name 'timeout' vs 'control_id'; kind 'keyword' vs 'positiona; param-mismatch/ param[
signalwire.relay.call.Call.ai_message: BACKLOG / param-mismatch/ param[1] (message_text)/ name 'message_text' vs 'control_id'; kind 'keyword' vs ; param-mismatch/ param[
signalwire.relay.call.Call.ai_unhold: BACKLOG / param-mismatch/ param[1] (prompt)/ name 'prompt' vs 'control_id'; kind 'keyword' vs 'positional'; param-mismatch/ param[
signalwire.relay.call.Call.amazon_bedrock: BACKLOG / param-count-mismatch/ reference has 8 param(s), port has 2/ reference=['self', 'prompt', 'SWAIG', 'ai_; return-mismatch/
signalwire.relay.call.Call.answer: BACKLOG / param-count-mismatch/ reference has 2 param(s), port has 1/ reference=['self', 'kwargs'] port=['self']; return-mismatch/
signalwire.relay.call.Call.bind_digit: BACKLOG / param-count-mismatch/ reference has 7 param(s), port has 6/ reference=['self', 'digits', 'bind_method'; return-mismatch/
signalwire.relay.call.Call.clear_digit_bindings: BACKLOG / param-count-mismatch/ reference has 3 param(s), port has 2/ reference=['self', 'realm', 'kwargs'] port; return-mismatch/
signalwire.relay.call.Call.collect: BACKLOG / param-count-mismatch/ reference has 11 param(s), port has 2/ reference=['self', 'digits', 'speech', 'i
signalwire.relay.call.Call.connect: BACKLOG / param-count-mismatch/ reference has 8 param(s), port has 3/ reference=['self', 'devices', 'ringback', ; return-mismatch/
signalwire.relay.call.Call.detect: BACKLOG / param-count-mismatch/ reference has 6 param(s), port has 3/ reference=['self', 'detect', 'timeout', 'c
signalwire.relay.call.Call.detect_answering_machine: Go collapses Python's keyword-only AMD args (initial_timeout/end_silence_timeout/machine_voice_threshold/machine_words_threshold/detect_interruptions/detect_message_end/timeout/on_completed) into variadic AMDOption; emits the same {"type":"machine","params":{...only-provided...}} detect media
signalwire.relay.call.Call.detect_digit: Go collapses Python's keyword-only digits/timeout/on_completed into variadic DetectDigitOption; emits the same {"type":"digit","params":{digits?}} detect media
signalwire.relay.call.Call.detect_fax: Go collapses Python's keyword-only tone/timeout/on_completed into variadic DetectFaxOption; emits the same {"type":"fax","params":{tone?}} detect media
signalwire.relay.call.Call.echo: BACKLOG / param-count-mismatch/ reference has 4 param(s), port has 3/ reference=['self', 'timeout', 'status_url'; return-mismatch/
signalwire.relay.call.Call.join_conference: BACKLOG / param-count-mismatch/ reference has 22 param(s), port has 3/ reference=['self', 'name', 'muted', 'beep; return-mismatch/
signalwire.relay.call.Call.join_room: BACKLOG / param-count-mismatch/ reference has 4 param(s), port has 3/ reference=['self', 'name', 'status_url', '; return-mismatch/
signalwire.relay.call.Call.leave_conference: BACKLOG / param-count-mismatch/ reference has 3 param(s), port has 2/ reference=['self', 'conference_id', 'kwarg; return-mismatch/
signalwire.relay.call.Call.leave_room: BACKLOG / param-count-mismatch/ reference has 2 param(s), port has 1/ reference=['self', 'kwargs'] port=['self']; return-mismatch/
signalwire.relay.call.Call.live_transcribe: BACKLOG / param-count-mismatch/ reference has 3 param(s), port has 2/ reference=['self', 'action', 'kwargs'] por; return-mismatch/
signalwire.relay.call.Call.live_translate: BACKLOG / param-count-mismatch/ reference has 4 param(s), port has 3/ reference=['self', 'action', 'status_url',; return-mismatch/
signalwire.relay.call.Call.on: BACKLOG / param-mismatch/ param[2] (handler)/ type 'class/signalwire.relay.call.EventHandler' vs 'callable
signalwire.relay.call.Call.pay: BACKLOG / param-count-mismatch/ reference has 22 param(s), port has 3/ reference=['self', 'payment_connector_url
signalwire.relay.call.Call.play: BACKLOG / param-count-mismatch/ reference has 8 param(s), port has 3/ reference=['self', 'media', 'volume', 'dir
signalwire.relay.call.Call.play_and_collect: BACKLOG / param-count-mismatch/ reference has 7 param(s), port has 4/ reference=['self', 'media', 'collect', 'vo
signalwire.relay.call.Call.play_audio: Go collapses Python's keyword-only volume/on_completed into variadic AudioOption; emits the same {"type":"audio","params":{"url":...}} play media
signalwire.relay.call.Call.play_ringtone: Go collapses Python's keyword-only duration/volume/on_completed into variadic RingtoneOption; emits the same {"type":"ringtone","params":{"name":...,duration?}} play media
signalwire.relay.call.Call.play_silence: Go drops Python's keyword-only on_completed (no functional callback variant); emits the same {"type":"silence","params":{"duration":...}} play media
signalwire.relay.call.Call.play_tts: Go collapses Python's keyword-only language/gender/voice/volume/on_completed into variadic TTSOption; emits the same {"type":"tts","params":{"text":...,language?,gender?,voice?}} play media
signalwire.relay.call.Call.prompt_audio: Go collapses Python's keyword-only volume/on_completed into variadic AudioOption; emits the same {"type":"audio","params":{"url":...}} play_and_collect media
signalwire.relay.call.Call.prompt_tts: Go collapses Python's keyword-only language/gender/voice/volume/on_completed into variadic TTSOption; emits the same {"type":"tts","params":{"text":...,language?,gender?,voice?}} play_and_collect media
signalwire.relay.call.Call.queue_enter: BACKLOG / param-count-mismatch/ reference has 5 param(s), port has 3/ reference=['self', 'queue_name', 'control_; return-mismatch/
signalwire.relay.call.Call.queue_leave: BACKLOG / param-count-mismatch/ reference has 6 param(s), port has 4/ reference=['self', 'queue_name', 'control_; return-mismatch/
signalwire.relay.call.Call.receive_fax: BACKLOG / param-count-mismatch/ reference has 4 param(s), port has 1/ reference=['self', 'control_id', 'on_compl
signalwire.relay.call.Call.record: BACKLOG / param-count-mismatch/ reference has 5 param(s), port has 2/ reference=['self', 'audio', 'control_id', 
signalwire.relay.call.Call.refer: BACKLOG / param-count-mismatch/ reference has 4 param(s), port has 3/ reference=['self', 'device', 'status_url',; return-mismatch/
signalwire.relay.call.Call.send_digits: BACKLOG / param-count-mismatch/ reference has 3 param(s), port has 2/ reference=['self', 'digits', 'control_id']; return-mismatch/
signalwire.relay.call.Call.send_fax: BACKLOG / param-count-mismatch/ reference has 7 param(s), port has 4/ reference=['self', 'document', 'identity',
signalwire.relay.call.Call.stream: BACKLOG / param-count-mismatch/ reference has 12 param(s), port has 3/ reference=['self', 'url', 'name', 'codec'
signalwire.relay.call.Call.tap: BACKLOG / param-count-mismatch/ reference has 6 param(s), port has 3/ reference=['self', 'tap', 'device', 'contr
signalwire.relay.call.Call.transcribe: BACKLOG / param-count-mismatch/ reference has 5 param(s), port has 2/ reference=['self', 'control_id', 'status_u
signalwire.relay.call.Call.transfer: BACKLOG / param-count-mismatch/ reference has 3 param(s), port has 2/ reference=['self', 'dest', 'kwargs'] port=; return-mismatch/
signalwire.relay.call.Call.user_event: BACKLOG / param-mismatch/ param[1] (event)/ name 'event' vs 'event_name'; kind 'keyword' vs 'positional'; ; param-mismatch/ param[
signalwire.relay.call.Call.wait_for: BACKLOG / param-mismatch/ param[1] (event_type)/ name 'event_type' vs 'ctx'; type 'string' vs 'any'; param-mismatch/ param[2] (pre
signalwire.relay.call.RecordAction.pause: BACKLOG / param-mismatch/ param[1] (behavior)/ type 'optional<string>' vs 'list<string>'; required False v; return-mismatch/ retur
signalwire.relay.client.RelayClient.dial: BACKLOG / param-count-mismatch/ reference has 5 param(s), port has 3/ reference=['self', 'devices', 'tag', 'max_
signalwire.relay.client.RelayClient.on_call: BACKLOG / param-mismatch/ param[1] (handler)/ type 'class/signalwire.relay.client.CallHandler' vs 'callabl; return-mismatch/ retur
signalwire.relay.client.RelayClient.on_message: BACKLOG / param-mismatch/ param[1] (handler)/ type 'class/signalwire.relay.client.MessageHandler' vs 'call; return-mismatch/ retur
signalwire.relay.client.RelayClient.send_message: BACKLOG / param-count-mismatch/ reference has 9 param(s), port has 5/ reference=['self', 'to_number', 'from_numb
signalwire.relay.event.CallReceiveEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.CallStateEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.CallingErrorEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.CollectEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.ConferenceEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.ConnectEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.DenoiseEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.DetectEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.DialEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.EchoEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.FaxEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.HoldEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.MessageReceiveEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.MessageStateEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.PayEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.PlayEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.QueueEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.RecordEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.ReferEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.RelayEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.SendDigitsEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.StreamEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.TapEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.event.TranscribeEvent.from_payload: BACKLOG / missing-port/ in reference, not in port
signalwire.relay.message.Message.on: BACKLOG / param-mismatch/ param[1] (handler)/ type 'class/Callable' vs 'callable<list<class/signalwire.rel
signalwire.rest.namespaces.relay_rest_resources_generated.PhoneNumbers.set_ai_agent: BACKLOG / param-mismatch/ param[1] (resource_id)/ name 'resource_id' vs 'sid'; param-mismatch/ param[3] (extra)/ kind 'var_keyword
signalwire.rest.namespaces.relay_rest_resources_generated.PhoneNumbers.set_call_flow: BACKLOG / param-count-mismatch/ reference has 5 param(s), port has 4/ reference=['self', 'resource_id', 'flow_id; return-mismatch/
signalwire.rest.namespaces.relay_rest_resources_generated.PhoneNumbers.set_cxml_application: BACKLOG / param-mismatch/ param[1] (resource_id)/ name 'resource_id' vs 'sid'; param-mismatch/ param[3] (extra)/ kind 'var_keyword
signalwire.rest.namespaces.relay_rest_resources_generated.PhoneNumbers.set_cxml_webhook: BACKLOG / param-count-mismatch/ reference has 6 param(s), port has 4/ reference=['self', 'resource_id', 'url', '; return-mismatch/
signalwire.rest.namespaces.relay_rest_resources_generated.PhoneNumbers.set_relay_application: BACKLOG / param-mismatch/ param[1] (resource_id)/ name 'resource_id' vs 'sid'; param-mismatch/ param[3] (extra)/ kind 'var_keyword
signalwire.rest.namespaces.relay_rest_resources_generated.PhoneNumbers.set_relay_topic: BACKLOG / param-count-mismatch/ reference has 5 param(s), port has 4/ reference=['self', 'resource_id', 'topic',; return-mismatch/
signalwire.rest.namespaces.relay_rest_resources_generated.PhoneNumbers.set_swml_webhook: BACKLOG / param-mismatch/ param[1] (resource_id)/ name 'resource_id' vs 'sid'; param-mismatch/ param[3] (extra)/ kind 'var_keyword
