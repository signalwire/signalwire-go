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
signalwire.core.data_map.DataMap.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.core.function_result.FunctionResult.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.core.security.session_manager.SessionManager.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.core.skill_base.SkillBase.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.core.skill_manager.SkillManager.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.core.swml_service.SWMLService.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.livewire.Agent.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.livewire.AgentHandoff.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.livewire.AgentServer.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.livewire.AgentSession.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.livewire.JobContext.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.livewire.JobProcess.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.livewire.RunContext.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.livewire.plugins.CartesiaTTS.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.livewire.plugins.DeepgramSTT.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.livewire.plugins.ElevenLabsTTS.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.livewire.plugins.OpenAILLM.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.livewire.plugins.SileroVAD.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
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
signalwire.rest._base.HttpClient.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest._pagination.PaginatedIterator.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.client.RestClient.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.addresses.AddressesResource.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.calling.CallingNamespace.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.chat.ChatResource.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.compat.CompatAccounts.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.compat.CompatNamespace.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.compat.CompatPhoneNumbers.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.datasphere.DatasphereDocuments.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.datasphere.DatasphereNamespace.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.fabric.FabricNamespace.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.fabric.FabricTokens.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.imported_numbers.ImportedNumbersResource.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.logs.LogsNamespace.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.lookup.LookupResource.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.mfa.MfaResource.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.number_groups.NumberGroupsResource.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.phone_numbers.PhoneNumbersResource.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.project.ProjectNamespace.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.project.ProjectTokens.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.pubsub.PubSubResource.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.queues.QueuesResource.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.recordings.RecordingsResource.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.registry.RegistryNamespace.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.short_codes.ShortCodesResource.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.sip_profile.SipProfileResource.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.verified_callers.VerifiedCallersResource.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.rest.namespaces.video.VideoNamespace.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.search.DocumentProcessor.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.search.IndexBuilder.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.search.SearchEngine.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.search.SearchService.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.search.search_service.SearchRequest.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.search.search_service.SearchResponse.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs
signalwire.search.search_service.SearchResult.__init__: Go uses NewX factory function as constructor; param shape may differ from Python kwargs

## Idiom: Go fluent API returns *Self for chaining

signalwire.agent_server.AgentServer.get_agent: Go fluent API returns *Self for chaining
signalwire.agent_server.AgentServer.get_agents: Go fluent API returns *Self for chaining
signalwire.core.contexts.Context.get_step: Go fluent API returns *Self for chaining
signalwire.core.contexts.ContextBuilder.get_context: Go fluent API returns *Self for chaining
signalwire.core.mixins.tool_mixin.ToolMixin.define_tools: Go fluent API returns *Self for chaining
signalwire.core.mixins.web_mixin.WebMixin.as_router: Go fluent API returns *Self for chaining
signalwire.core.swml_service.SWMLService.get_document: Go fluent API returns *Self for chaining
signalwire.livewire.Agent.update_tools: Go fluent API returns *Self for chaining
signalwire.relay.event.parse_event: Go fluent API returns *Self for chaining

## Backlog: real signature divergences (418 symbols)

Real Go port maintenance — parameter renames, missing optionals,
type imprecisions. Triage in a separate sweep.

signalwire.RestClient: BACKLOG / param-count-mismatch/ reference has 2 param(s), port has 3/ reference=['args', 'kwargs'] port=['projec; return-mismatch/
signalwire.add_skill_directory: BACKLOG / param-mismatch/ param[0] (path)/ type 'any' vs 'string'
signalwire.agent_server.AgentServer.register: BACKLOG / param-mismatch/ param[1] (agent)/ name 'agent' vs 'a'; param-mismatch/ param[2] (route)/ type 'optional<string>' vs 'str
signalwire.agent_server.AgentServer.run: BACKLOG / param-count-mismatch/ reference has 5 param(s), port has 2/ reference=['self', 'event', 'context', 'ho
signalwire.agent_server.AgentServer.serve_static_files: BACKLOG / param-mismatch/ param[2] (route)/ required False vs True; default '/' vs '<absent>'
signalwire.agent_server.AgentServer.setup_sip_routing: BACKLOG / param-mismatch/ param[1] (route)/ required False vs True; default '/sip' vs '<absent>'; param-mismatch/ param[2] (auto_m
signalwire.core.agent_base.AgentBase.add_answer_verb: BACKLOG / param-mismatch/ param[1] (config)/ type 'optional<dict<string,any>>' vs 'dict<string,any>'; requ
signalwire.core.agent_base.AgentBase.enable_sip_routing: BACKLOG / param-mismatch/ param[1] (auto_map)/ required False vs True; default True vs '<absent>'; param-mismatch/ param[2] (path)
signalwire.core.agent_base.AgentBase.on_debug_event: BACKLOG / param-mismatch/ param[1] (handler)/ name 'handler' vs 'cb'; type 'class/Callable' vs 'class/Debu; return-mismatch/ retur
signalwire.core.agent_base.AgentBase.on_summary: BACKLOG / param-count-mismatch/ reference has 3 param(s), port has 2/ reference=['self', 'summary', 'raw_data'] ; return-mismatch/
signalwire.core.agent_base.AgentBase.register_sip_username: BACKLOG / param-mismatch/ param[1] (sip_username)/ name 'sip_username' vs 'username'
signalwire.core.contexts.Context.add_enter_filler: BACKLOG / param-mismatch/ param[1] (language_code)/ name 'language_code' vs 'lang_code'
signalwire.core.contexts.Context.add_exit_filler: BACKLOG / param-mismatch/ param[1] (language_code)/ name 'language_code' vs 'lang_code'
signalwire.core.contexts.Context.add_step: BACKLOG / param-count-mismatch/ reference has 7 param(s), port has 2/ reference=['self', 'name', 'task', 'bullet
signalwire.core.contexts.Context.set_enter_fillers: BACKLOG / param-mismatch/ param[1] (enter_fillers)/ name 'enter_fillers' vs 'fillers'
signalwire.core.contexts.Context.set_exit_fillers: BACKLOG / param-mismatch/ param[1] (exit_fillers)/ name 'exit_fillers' vs 'fillers'
signalwire.core.contexts.Context.set_post_prompt: BACKLOG / param-mismatch/ param[1] (post_prompt)/ name 'post_prompt' vs 'prompt'
signalwire.core.contexts.Context.set_system_prompt: BACKLOG / param-mismatch/ param[1] (system_prompt)/ name 'system_prompt' vs 'prompt'
signalwire.core.contexts.Context.set_user_prompt: BACKLOG / param-mismatch/ param[1] (user_prompt)/ name 'user_prompt' vs 'prompt'
signalwire.core.contexts.Context.set_valid_contexts: BACKLOG / param-mismatch/ param[1] (contexts)/ name 'contexts' vs 'ctxs'
signalwire.core.contexts.ContextBuilder.validate: BACKLOG / return-mismatch/ returns 'void' vs 'any'
signalwire.core.contexts.GatherInfo.add_question: BACKLOG / param-mismatch/ param[3] (kwargs)/ name 'kwargs' vs 'opts'; kind 'var_keyword' vs 'positional'; 
signalwire.core.contexts.Step.add_gather_question: BACKLOG / param-count-mismatch/ reference has 7 param(s), port has 4/ reference=['self', 'key', 'question', 'typ
signalwire.core.contexts.Step.set_functions: BACKLOG / param-mismatch/ param[1] (functions)/ type 'union<list<string>,string>' vs 'any'
signalwire.core.contexts.Step.set_gather_info: BACKLOG / param-mismatch/ param[1] (output_key)/ type 'optional<string>' vs 'string'; required False vs Tr; param-mismatch/ param[
signalwire.core.contexts.Step.set_reset_system_prompt: BACKLOG / param-mismatch/ param[1] (system_prompt)/ name 'system_prompt' vs 'prompt'
signalwire.core.contexts.Step.set_reset_user_prompt: BACKLOG / param-mismatch/ param[1] (user_prompt)/ name 'user_prompt' vs 'prompt'
signalwire.core.contexts.create_simple_context: BACKLOG / param-mismatch/ param[0] (name)/ required False vs True; default 'default' vs '<absent>'
signalwire.core.data_map.DataMap.expression: BACKLOG / param-mismatch/ param[2] (pattern)/ type 'union<class/Pattern,string>' vs 'string'; param-mismatch/ param[4] (nomatch_ou
signalwire.core.data_map.DataMap.foreach: BACKLOG / param-mismatch/ param[1] (foreach_config)/ name 'foreach_config' vs 'config'
signalwire.core.data_map.DataMap.parameter: BACKLOG / param-mismatch/ param[3] (description)/ name 'description' vs 'desc'; param-mismatch/ param[4] (required)/ required Fals
signalwire.core.data_map.DataMap.webhook: BACKLOG / param-mismatch/ param[3] (headers)/ type 'optional<dict<string,string>>' vs 'dict<string,string>; param-mismatch/ param[
signalwire.core.data_map.create_expression_tool: BACKLOG / param-mismatch/ param[1] (patterns)/ type 'dict<string,tuple<string,class/signalwire.core.functi; param-mismatch/ param[
signalwire.core.data_map.create_simple_api_tool: BACKLOG / param-mismatch/ param[3] (parameters)/ type 'optional<dict<string,class/Dict>>' vs 'dict<string,; param-mismatch/ param[
signalwire.core.function_result.FunctionResult.add_dynamic_hints: BACKLOG / param-mismatch/ param[1] (hints)/ type 'list<union<dict<string,any>,string>>' vs 'list<any>'
signalwire.core.function_result.FunctionResult.connect: BACKLOG / param-mismatch/ param[2] (final)/ required False vs True; default True vs '<absent>'; param-mismatch/ param[3] (from_add
signalwire.core.function_result.FunctionResult.enable_extensive_data: BACKLOG / param-mismatch/ param[1] (enabled)/ required False vs True; default True vs '<absent>'
signalwire.core.function_result.FunctionResult.enable_functions_on_timeout: BACKLOG / param-mismatch/ param[1] (enabled)/ required False vs True; default True vs '<absent>'
signalwire.core.function_result.FunctionResult.execute_rpc: BACKLOG / param-mismatch/ param[2] (params)/ type 'optional<dict<string,any>>' vs 'dict<string,any>'; requ; param-mismatch/ param[
signalwire.core.function_result.FunctionResult.execute_swml: BACKLOG / param-mismatch/ param[2] (transfer)/ required False vs True; default False vs '<absent>'
signalwire.core.function_result.FunctionResult.hold: BACKLOG / param-mismatch/ param[1] (timeout)/ required False vs True; default 300 vs '<absent>'
signalwire.core.function_result.FunctionResult.join_conference: BACKLOG / param-count-mismatch/ reference has 19 param(s), port has 3/ reference=['self', 'name', 'muted', 'beep
signalwire.core.function_result.FunctionResult.pay: BACKLOG / param-count-mismatch/ reference has 20 param(s), port has 3/ reference=['self', 'payment_connector_url
signalwire.core.function_result.FunctionResult.play_background_file: BACKLOG / param-mismatch/ param[2] (wait)/ required False vs True; default False vs '<absent>'
signalwire.core.function_result.FunctionResult.record_call: BACKLOG / param-count-mismatch/ reference has 12 param(s), port has 6/ reference=['self', 'control_id', 'stereo'
signalwire.core.function_result.FunctionResult.remove_global_data: BACKLOG / param-mismatch/ param[1] (keys)/ type 'union<list<string>,string>' vs 'list<string>'
signalwire.core.function_result.FunctionResult.remove_metadata: BACKLOG / param-mismatch/ param[1] (keys)/ type 'union<list<string>,string>' vs 'list<string>'
signalwire.core.function_result.FunctionResult.replace_in_history: BACKLOG / param-mismatch/ param[1] (text)/ type 'union<bool,string>' vs 'any'; required False vs True; def
signalwire.core.function_result.FunctionResult.rpc_ai_message: BACKLOG / param-mismatch/ param[3] (role)/ required False vs True; default 'system' vs '<absent>'
signalwire.core.function_result.FunctionResult.rpc_dial: BACKLOG / param-mismatch/ param[4] (device_type)/ required False vs True; default 'phone' vs '<absent>'
signalwire.core.function_result.FunctionResult.send_sms: BACKLOG / param-mismatch/ param[3] (body)/ type 'optional<string>' vs 'string'; required False vs True; de; param-mismatch/ param[
signalwire.core.function_result.FunctionResult.set_end_of_speech_timeout: BACKLOG / param-mismatch/ param[1] (milliseconds)/ name 'milliseconds' vs 'ms'
signalwire.core.function_result.FunctionResult.set_speech_event_timeout: BACKLOG / param-mismatch/ param[1] (milliseconds)/ name 'milliseconds' vs 'ms'
signalwire.core.function_result.FunctionResult.stop_record_call: BACKLOG / param-mismatch/ param[1] (control_id)/ type 'optional<string>' vs 'string'; required False vs Tr
signalwire.core.function_result.FunctionResult.stop_tap: BACKLOG / param-mismatch/ param[1] (control_id)/ type 'optional<string>' vs 'string'; required False vs Tr
signalwire.core.function_result.FunctionResult.switch_context: BACKLOG / param-count-mismatch/ reference has 5 param(s), port has 6/ reference=['self', 'system_prompt', 'user_
signalwire.core.function_result.FunctionResult.swml_transfer: BACKLOG / param-mismatch/ param[3] (final)/ required False vs True; default True vs '<absent>'
signalwire.core.function_result.FunctionResult.tap: BACKLOG / param-mismatch/ param[2] (control_id)/ type 'optional<string>' vs 'string'; required False vs Tr; param-mismatch/ param[
signalwire.core.function_result.FunctionResult.toggle_functions: BACKLOG / param-mismatch/ param[1] (function_toggles)/ name 'function_toggles' vs 'toggles'
signalwire.core.function_result.FunctionResult.wait_for_user: BACKLOG / param-mismatch/ param[1] (enabled)/ required False vs True; default None vs '<absent>'; param-mismatch/ param[2] (timeou
signalwire.core.mixins.ai_config_mixin.AIConfigMixin.add_function_include: BACKLOG / param-mismatch/ param[3] (meta_data)/ type 'optional<dict<string,any>>' vs 'dict<string,any>'; r
signalwire.core.mixins.ai_config_mixin.AIConfigMixin.add_internal_filler: BACKLOG / param-mismatch/ param[1] (function_name)/ name 'function_name' vs 'func_name'; param-mismatch/ param[2] (language_code)/
signalwire.core.mixins.ai_config_mixin.AIConfigMixin.add_language: BACKLOG / param-count-mismatch/ reference has 8 param(s), port has 2/ reference=['self', 'name', 'code', 'voice'
signalwire.core.mixins.ai_config_mixin.AIConfigMixin.add_mcp_server: BACKLOG / param-count-mismatch/ reference has 5 param(s), port has 2/ reference=['self', 'url', 'headers', 'reso
signalwire.core.mixins.ai_config_mixin.AIConfigMixin.add_pattern_hint: BACKLOG / param-mismatch/ param[4] (ignore_case)/ type 'bool' vs 'list<bool>'; required False vs True; def
signalwire.core.mixins.ai_config_mixin.AIConfigMixin.add_pronunciation: BACKLOG / param-mismatch/ param[3] (ignore_case)/ type 'bool' vs 'list<bool>'; required False vs True; def
signalwire.core.mixins.ai_config_mixin.AIConfigMixin.enable_debug_events: BACKLOG / param-mismatch/ param[1] (level)/ required False vs True; default 1 vs '<absent>'
signalwire.core.mixins.ai_config_mixin.AIConfigMixin.set_internal_fillers: BACKLOG / param-mismatch/ param[1] (internal_fillers)/ name 'internal_fillers' vs 'fillers'
signalwire.core.mixins.ai_config_mixin.AIConfigMixin.set_native_functions: BACKLOG / param-mismatch/ param[1] (function_names)/ name 'function_names' vs 'names'
signalwire.core.mixins.ai_config_mixin.AIConfigMixin.set_post_prompt_llm_params: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin
signalwire.core.mixins.ai_config_mixin.AIConfigMixin.set_prompt_llm_params: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin
signalwire.core.mixins.ai_config_mixin.AIConfigMixin.set_pronunciations: BACKLOG / param-mismatch/ param[1] (pronunciations)/ name 'pronunciations' vs 'p'
signalwire.core.mixins.prompt_mixin.PromptMixin.contexts: BACKLOG / missing-reference/ in port, not in reference
signalwire.core.mixins.prompt_mixin.PromptMixin.define_contexts: BACKLOG / param-count-mismatch/ reference has 2 param(s), port has 1/ reference=['self', 'contexts'] port=['self; return-mismatch/
signalwire.core.mixins.prompt_mixin.PromptMixin.get_prompt: BACKLOG / return-mismatch/ returns 'union<list<dict<string,any>>,string>' vs 'any'
signalwire.core.mixins.prompt_mixin.PromptMixin.prompt_add_section: BACKLOG / param-count-mismatch/ reference has 7 param(s), port has 5/ reference=['self', 'title', 'body', 'bulle
signalwire.core.mixins.prompt_mixin.PromptMixin.prompt_add_subsection: BACKLOG / param-mismatch/ param[3] (body)/ required False vs True; default '' vs '<absent>'; param-mismatch/ param[4] (bullets)/ t
signalwire.core.mixins.prompt_mixin.PromptMixin.prompt_add_to_section: BACKLOG / param-count-mismatch/ reference has 5 param(s), port has 4/ reference=['self', 'title', 'body', 'bulle
signalwire.core.mixins.skill_mixin.SkillMixin.add_skill: BACKLOG / param-mismatch/ param[2] (params)/ type 'optional<dict<string,any>>' vs 'dict<string,any>'; requ
signalwire.core.mixins.tool_mixin.ToolMixin.define_tool: BACKLOG / param-count-mismatch/ reference has 11 param(s), port has 2/ reference=['self', 'name', 'description',
signalwire.core.mixins.tool_mixin.ToolMixin.on_function_call: BACKLOG / param-mismatch/ param[3] (raw_data)/ type 'optional<dict<string,any>>' vs 'dict<string,any>'; re
signalwire.core.mixins.tool_mixin.ToolMixin.register_swaig_function: BACKLOG / param-mismatch/ param[1] (function_dict)/ name 'function_dict' vs 'func_def'
signalwire.core.mixins.web_mixin.WebMixin.manual_set_proxy_url: BACKLOG / param-mismatch/ param[1] (proxy_url)/ name 'proxy_url' vs 'url'
signalwire.core.mixins.web_mixin.WebMixin.run: BACKLOG / param-count-mismatch/ reference has 6 param(s), port has 1/ reference=['self', 'event', 'context', 'fo
signalwire.core.mixins.web_mixin.WebMixin.serve: BACKLOG / param-count-mismatch/ reference has 3 param(s), port has 1/ reference=['self', 'host', 'port'] port=['; return-mismatch/
signalwire.core.mixins.web_mixin.WebMixin.set_dynamic_config_callback: BACKLOG / param-mismatch/ param[1] (callback)/ name 'callback' vs 'cb'; type 'callable<list<dict<any,any>,
signalwire.core.skill_base.SkillBase.get_parameter_schema: BACKLOG / param-mismatch/ param[0] (cls)/ name 'cls' vs 'self'; kind 'cls' vs 'self'
signalwire.core.skill_manager.SkillManager.get_skill: BACKLOG / param-mismatch/ param[1] (skill_identifier)/ name 'skill_identifier' vs 'key'; return-mismatch/ returns 'optional<class/
signalwire.core.skill_manager.SkillManager.has_skill: BACKLOG / param-mismatch/ param[1] (skill_identifier)/ name 'skill_identifier' vs 'key'
signalwire.core.skill_manager.SkillManager.load_skill: BACKLOG / param-count-mismatch/ reference has 4 param(s), port has 2/ reference=['self', 'skill_name', 'skill_cl; return-mismatch/
signalwire.core.skill_manager.SkillManager.unload_skill: BACKLOG / param-mismatch/ param[1] (skill_identifier)/ name 'skill_identifier' vs 'key'
signalwire.core.swml_service.SWMLService.add_verb: BACKLOG / param-mismatch/ param[2] (config)/ type 'union<dict<string,any>,int>' vs 'any'; return-mismatch/ returns 'bool' vs 'any'
signalwire.core.swml_service.SWMLService.add_verb_to_section: BACKLOG / param-mismatch/ param[1] (section_name)/ name 'section_name' vs 'section'; param-mismatch/ param[3] (config)/ type 'unio
signalwire.core.swml_service.SWMLService.get_basic_auth_credentials: BACKLOG / param-count-mismatch/ reference has 2 param(s), port has 1/ reference=['self', 'include_source'] port=; return-mismatch/
signalwire.core.swml_service.SWMLService.on_request: BACKLOG / param-mismatch/ param[1] (request_data)/ type 'optional<dict<any,any>>' vs 'dict<string,any>'; r; param-mismatch/ param[
signalwire.core.swml_service.SWMLService.register_routing_callback: BACKLOG / param-mismatch/ param[1] (callback_fn)/ name 'callback_fn' vs 'path'; type 'callable<list<class/; param-mismatch/ param[
signalwire.core.swml_service.SWMLService.serve: BACKLOG / param-count-mismatch/ reference has 7 param(s), port has 1/ reference=['self', 'host', 'port', 'ssl_ce; return-mismatch/
signalwire.list_skills: BACKLOG / param-count-mismatch/ reference has 2 param(s), port has 0/ reference=['args', 'kwargs'] port=[]; return-mismatch/ retur
signalwire.list_skills_with_params: BACKLOG / return-mismatch/ returns 'any' vs 'dict<string,dict<string,dict<string,any>>>'
signalwire.livewire.Agent.on_enter: BACKLOG / param-count-mismatch/ reference has 1 param(s), port has 2/ reference=['self'] port=['self', 'fn']; return-mismatch/ ret
signalwire.livewire.Agent.on_exit: BACKLOG / param-count-mismatch/ reference has 1 param(s), port has 2/ reference=['self'] port=['self', 'fn']; return-mismatch/ ret
signalwire.livewire.Agent.on_user_turn_completed: BACKLOG / param-count-mismatch/ reference has 3 param(s), port has 2/ reference=['self', 'turn_ctx', 'new_messag; return-mismatch/
signalwire.livewire.AgentServer.rtc_session: BACKLOG / param-count-mismatch/ reference has 6 param(s), port has 3/ reference=['self', 'func', 'agent_name', '; return-mismatch/
signalwire.livewire.AgentSession.generate_reply: BACKLOG / param-mismatch/ param[1] (instructions)/ name 'instructions' vs 'opts'; kind 'keyword' vs 'posit; return-mismatch/ retur
signalwire.livewire.AgentSession.interrupt: BACKLOG / return-mismatch/ returns 'any' vs 'void'
signalwire.livewire.AgentSession.say: BACKLOG / return-mismatch/ returns 'any' vs 'void'
signalwire.livewire.AgentSession.start: BACKLOG / param-mismatch/ param[1] (agent)/ name 'agent' vs 'ctx'; type 'class/signalwire.livewire.Agent' ; param-mismatch/ param[
signalwire.livewire.AgentSession.update_agent: BACKLOG / param-mismatch/ param[1] (agent)/ name 'agent' vs 'ag'; return-mismatch/ returns 'any' vs 'void'
signalwire.livewire.JobContext.wait_for_participant: BACKLOG / param-mismatch/ param[1] (identity)/ kind 'keyword' vs 'positional'; type 'any' vs 'string'; req
signalwire.livewire.plugins.SileroVAD.load: BACKLOG / param-mismatch/ param[0] (cls)/ name 'cls' vs 'self'; kind 'cls' vs 'self'; return-mismatch/ returns 'any' vs 'class/sig
signalwire.livewire.run_app: BACKLOG / return-mismatch/ returns 'any' vs 'void'
signalwire.prefabs.info_gatherer.InfoGathererAgent.set_question_callback: BACKLOG / param-mismatch/ param[1] (callback)/ name 'callback' vs 'cb'; type 'callable<list<dict<any,any>,; return-mismatch/ retur
signalwire.register_skill: BACKLOG / param-count-mismatch/ reference has 1 param(s), port has 2/ reference=['skill_class'] port=['name', 'f; return-mismatch/
signalwire.relay.call.AIAction.stop: BACKLOG / return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.Action.is_done: BACKLOG / missing-reference/ in port, not in reference
signalwire.relay.call.Action.wait: BACKLOG / param-mismatch/ param[1] (timeout)/ name 'timeout' vs 'ctx'; type 'optional<float>' vs 'any'; re
signalwire.relay.call.Call.__repr__: BACKLOG / missing-reference/ in port, not in reference
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
signalwire.relay.call.Call.denoise: BACKLOG / return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.Call.denoise_stop: BACKLOG / return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.Call.detect: BACKLOG / param-count-mismatch/ reference has 6 param(s), port has 3/ reference=['self', 'detect', 'timeout', 'c
signalwire.relay.call.Call.disconnect: BACKLOG / return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.Call.echo: BACKLOG / param-count-mismatch/ reference has 4 param(s), port has 3/ reference=['self', 'timeout', 'status_url'; return-mismatch/
signalwire.relay.call.Call.hangup: BACKLOG / param-mismatch/ param[1] (reason)/ required False vs True; default 'hangup' vs '<absent>'; return-mismatch/ returns 'dic
signalwire.relay.call.Call.hold: BACKLOG / return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.Call.join_conference: BACKLOG / param-count-mismatch/ reference has 22 param(s), port has 3/ reference=['self', 'name', 'muted', 'beep; return-mismatch/
signalwire.relay.call.Call.join_room: BACKLOG / param-count-mismatch/ reference has 4 param(s), port has 3/ reference=['self', 'name', 'status_url', '; return-mismatch/
signalwire.relay.call.Call.leave_conference: BACKLOG / param-count-mismatch/ reference has 3 param(s), port has 2/ reference=['self', 'conference_id', 'kwarg; return-mismatch/
signalwire.relay.call.Call.leave_room: BACKLOG / param-count-mismatch/ reference has 2 param(s), port has 1/ reference=['self', 'kwargs'] port=['self']; return-mismatch/
signalwire.relay.call.Call.live_transcribe: BACKLOG / param-count-mismatch/ reference has 3 param(s), port has 2/ reference=['self', 'action', 'kwargs'] por; return-mismatch/
signalwire.relay.call.Call.live_translate: BACKLOG / param-count-mismatch/ reference has 4 param(s), port has 3/ reference=['self', 'action', 'status_url',; return-mismatch/
signalwire.relay.call.Call.on: BACKLOG / param-mismatch/ param[2] (handler)/ type 'class/signalwire.relay.call.EventHandler' vs 'callable
signalwire.relay.call.Call.pass_: BACKLOG / return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.Call.pay: BACKLOG / param-count-mismatch/ reference has 22 param(s), port has 3/ reference=['self', 'payment_connector_url
signalwire.relay.call.Call.play: BACKLOG / param-count-mismatch/ reference has 8 param(s), port has 3/ reference=['self', 'media', 'volume', 'dir
signalwire.relay.call.Call.play_and_collect: BACKLOG / param-count-mismatch/ reference has 7 param(s), port has 4/ reference=['self', 'media', 'collect', 'vo
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
signalwire.relay.call.Call.unhold: BACKLOG / return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.Call.user_event: BACKLOG / param-mismatch/ param[1] (event)/ name 'event' vs 'event_name'; kind 'keyword' vs 'positional'; ; param-mismatch/ param[
signalwire.relay.call.Call.wait_for: BACKLOG / param-mismatch/ param[1] (event_type)/ name 'event_type' vs 'ctx'; type 'string' vs 'any'; param-mismatch/ param[2] (pre
signalwire.relay.call.Call.wait_for_ended: BACKLOG / param-mismatch/ param[1] (timeout)/ name 'timeout' vs 'ctx'; type 'optional<float>' vs 'any'; re
signalwire.relay.call.CollectAction.start_input_timers: BACKLOG / return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.CollectAction.stop: BACKLOG / return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.CollectAction.volume: BACKLOG / param-mismatch/ param[1] (volume)/ name 'volume' vs 'db'; return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.DetectAction.stop: BACKLOG / return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.FaxAction.stop: BACKLOG / return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.PayAction.stop: BACKLOG / return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.PlayAction.pause: BACKLOG / return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.PlayAction.resume: BACKLOG / return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.PlayAction.stop: BACKLOG / return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.PlayAction.volume: BACKLOG / param-mismatch/ param[1] (volume)/ name 'volume' vs 'db'; return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.RecordAction.pause: BACKLOG / param-mismatch/ param[1] (behavior)/ type 'optional<string>' vs 'list<string>'; required False v; return-mismatch/ retur
signalwire.relay.call.RecordAction.resume: BACKLOG / return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.RecordAction.stop: BACKLOG / return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.StandaloneCollectAction.start_input_timers: BACKLOG / return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.StandaloneCollectAction.stop: BACKLOG / return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.StreamAction.stop: BACKLOG / return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.TapAction.stop: BACKLOG / return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.call.TranscribeAction.stop: BACKLOG / return-mismatch/ returns 'dict<any,any>' vs 'any'
signalwire.relay.client.RelayClient.dial: BACKLOG / param-count-mismatch/ reference has 5 param(s), port has 3/ reference=['self', 'devices', 'tag', 'max_
signalwire.relay.client.RelayClient.on_call: BACKLOG / param-mismatch/ param[1] (handler)/ type 'class/signalwire.relay.client.CallHandler' vs 'callabl; return-mismatch/ retur
signalwire.relay.client.RelayClient.on_message: BACKLOG / param-mismatch/ param[1] (handler)/ type 'class/signalwire.relay.client.MessageHandler' vs 'call; return-mismatch/ retur
signalwire.relay.client.RelayClient.run: BACKLOG / return-mismatch/ returns 'void' vs 'any'
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
signalwire.relay.message.Message.is_done: BACKLOG / missing-reference/ in port, not in reference
signalwire.relay.message.Message.on: BACKLOG / param-mismatch/ param[1] (handler)/ type 'class/Callable' vs 'callable<list<class/signalwire.rel
signalwire.relay.message.Message.wait: BACKLOG / param-mismatch/ param[1] (timeout)/ name 'timeout' vs 'ctx'; type 'optional<float>' vs 'any'; re
signalwire.rest._base.CrudResource.create: BACKLOG / missing-reference/ in port, not in reference
signalwire.rest._base.CrudResource.delete: BACKLOG / missing-reference/ in port, not in reference
signalwire.rest._base.CrudResource.get: BACKLOG / missing-reference/ in port, not in reference
signalwire.rest._base.CrudResource.list: BACKLOG / missing-reference/ in port, not in reference
signalwire.rest._base.CrudResource.update: BACKLOG / missing-reference/ in port, not in reference
signalwire.rest._base.HttpClient.delete: BACKLOG / missing-reference/ in port, not in reference
signalwire.rest._base.HttpClient.get: BACKLOG / missing-reference/ in port, not in reference
signalwire.rest._base.HttpClient.patch: BACKLOG / missing-reference/ in port, not in reference
signalwire.rest._base.HttpClient.post: BACKLOG / missing-reference/ in port, not in reference
signalwire.rest._base.HttpClient.put: BACKLOG / missing-reference/ in port, not in reference
signalwire.rest._pagination.PaginatedIterator.__next__: BACKLOG / missing-reference/ in port, not in reference
signalwire.rest.namespaces.addresses.AddressesResource.create: BACKLOG / param-mismatch/ param[1] (kwargs)/ name 'kwargs' vs 'data'; kind 'var_keyword' vs 'positional'; ; return-mismatch/ retur
signalwire.rest.namespaces.addresses.AddressesResource.delete: BACKLOG / param-mismatch/ param[1] (address_id)/ name 'address_id' vs 'id'; type 'any' vs 'string'; return-mismatch/ returns 'any'
signalwire.rest.namespaces.addresses.AddressesResource.get: BACKLOG / param-mismatch/ param[1] (address_id)/ name 'address_id' vs 'id'; type 'any' vs 'string'; return-mismatch/ returns 'any'
signalwire.rest.namespaces.addresses.AddressesResource.list: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin; return-mismatch/ retur
signalwire.rest.namespaces.calling.CallingNamespace.ai_hold: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.ai_message: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.ai_stop: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.ai_unhold: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.collect: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.collect_start_input_timers: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.collect_stop: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.denoise: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.denoise_stop: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.detect: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.detect_stop: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.dial: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin; return-mismatch/ retur
signalwire.rest.namespaces.calling.CallingNamespace.disconnect: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.end: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.live_transcribe: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.live_translate: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.play: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.play_pause: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.play_resume: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.play_stop: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.play_volume: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.receive_fax_stop: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.record: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.record_pause: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.record_resume: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.record_stop: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.refer: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.send_fax_stop: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.stream: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.stream_stop: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.tap: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.tap_stop: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.transcribe: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.transcribe_stop: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.transfer: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.calling.CallingNamespace.update: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin; return-mismatch/ retur
signalwire.rest.namespaces.calling.CallingNamespace.user_event: BACKLOG / param-mismatch/ param[1] (call_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.chat.ChatResource.create_token: BACKLOG / param-mismatch/ param[1] (kwargs)/ name 'kwargs' vs 'data'; kind 'var_keyword' vs 'positional'; ; return-mismatch/ retur
signalwire.rest.namespaces.compat.CompatAccounts.create: BACKLOG / param-mismatch/ param[1] (kwargs)/ name 'kwargs' vs 'data'; kind 'var_keyword' vs 'positional'; ; return-mismatch/ retur
signalwire.rest.namespaces.compat.CompatAccounts.get: BACKLOG / param-mismatch/ param[1] (sid)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.compat.CompatAccounts.list: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin; return-mismatch/ retur
signalwire.rest.namespaces.compat.CompatAccounts.update: BACKLOG / param-mismatch/ param[1] (sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs 'data'; kind
signalwire.rest.namespaces.compat.CompatApplications.update: BACKLOG / param-mismatch/ param[1] (sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs 'data'; kind
signalwire.rest.namespaces.compat.CompatCalls.start_recording: BACKLOG / param-mismatch/ param[1] (call_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs 'data';
signalwire.rest.namespaces.compat.CompatCalls.start_stream: BACKLOG / param-mismatch/ param[1] (call_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs 'data';
signalwire.rest.namespaces.compat.CompatCalls.stop_stream: BACKLOG / param-mismatch/ param[1] (call_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (stream_sid)/ type 'any' vs 'strin
signalwire.rest.namespaces.compat.CompatCalls.update: BACKLOG / param-mismatch/ param[1] (sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs 'data'; kind
signalwire.rest.namespaces.compat.CompatCalls.update_recording: BACKLOG / param-mismatch/ param[1] (call_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (recording_sid)/ type 'any' vs 'st
signalwire.rest.namespaces.compat.CompatConferences.delete_recording: BACKLOG / param-mismatch/ param[1] (conference_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (recording_sid)/ type 'any' 
signalwire.rest.namespaces.compat.CompatConferences.get: BACKLOG / param-mismatch/ param[1] (sid)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.compat.CompatConferences.get_participant: BACKLOG / param-mismatch/ param[1] (conference_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (call_sid)/ type 'any' vs 's
signalwire.rest.namespaces.compat.CompatConferences.get_recording: BACKLOG / param-mismatch/ param[1] (conference_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (recording_sid)/ type 'any' 
signalwire.rest.namespaces.compat.CompatConferences.list: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin; return-mismatch/ retur
signalwire.rest.namespaces.compat.CompatConferences.list_participants: BACKLOG / param-mismatch/ param[1] (conference_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword'
signalwire.rest.namespaces.compat.CompatConferences.list_recordings: BACKLOG / param-mismatch/ param[1] (conference_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword'
signalwire.rest.namespaces.compat.CompatConferences.remove_participant: BACKLOG / param-mismatch/ param[1] (conference_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (call_sid)/ type 'any' vs 's
signalwire.rest.namespaces.compat.CompatConferences.start_stream: BACKLOG / param-mismatch/ param[1] (conference_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs '
signalwire.rest.namespaces.compat.CompatConferences.stop_stream: BACKLOG / param-mismatch/ param[1] (conference_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (stream_sid)/ type 'any' vs 
signalwire.rest.namespaces.compat.CompatConferences.update: BACKLOG / param-mismatch/ param[1] (sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs 'data'; kind
signalwire.rest.namespaces.compat.CompatConferences.update_participant: BACKLOG / param-mismatch/ param[1] (conference_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (call_sid)/ type 'any' vs 's
signalwire.rest.namespaces.compat.CompatConferences.update_recording: BACKLOG / param-mismatch/ param[1] (conference_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (recording_sid)/ type 'any' 
signalwire.rest.namespaces.compat.CompatFaxes.delete_media: BACKLOG / param-mismatch/ param[1] (fax_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (media_sid)/ type 'any' vs 'string'
signalwire.rest.namespaces.compat.CompatFaxes.get_media: BACKLOG / param-mismatch/ param[1] (fax_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (media_sid)/ type 'any' vs 'string'
signalwire.rest.namespaces.compat.CompatFaxes.list_media: BACKLOG / param-mismatch/ param[1] (fax_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.compat.CompatFaxes.update: BACKLOG / param-mismatch/ param[1] (sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs 'data'; kind
signalwire.rest.namespaces.compat.CompatLamlBins.update: BACKLOG / param-mismatch/ param[1] (sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs 'data'; kind
signalwire.rest.namespaces.compat.CompatMessages.delete_media: BACKLOG / param-mismatch/ param[1] (message_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (media_sid)/ type 'any' vs 'str
signalwire.rest.namespaces.compat.CompatMessages.get_media: BACKLOG / param-mismatch/ param[1] (message_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (media_sid)/ type 'any' vs 'str
signalwire.rest.namespaces.compat.CompatMessages.list_media: BACKLOG / param-mismatch/ param[1] (message_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs
signalwire.rest.namespaces.compat.CompatMessages.update: BACKLOG / param-mismatch/ param[1] (sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs 'data'; kind
signalwire.rest.namespaces.compat.CompatPhoneNumbers.delete: BACKLOG / param-mismatch/ param[1] (sid)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.compat.CompatPhoneNumbers.get: BACKLOG / param-mismatch/ param[1] (sid)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.compat.CompatPhoneNumbers.import_number: BACKLOG / param-mismatch/ param[1] (kwargs)/ name 'kwargs' vs 'data'; kind 'var_keyword' vs 'positional'; ; return-mismatch/ retur
signalwire.rest.namespaces.compat.CompatPhoneNumbers.list: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin; return-mismatch/ retur
signalwire.rest.namespaces.compat.CompatPhoneNumbers.list_available_countries: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin; return-mismatch/ retur
signalwire.rest.namespaces.compat.CompatPhoneNumbers.purchase: BACKLOG / param-mismatch/ param[1] (kwargs)/ name 'kwargs' vs 'data'; kind 'var_keyword' vs 'positional'; ; return-mismatch/ retur
signalwire.rest.namespaces.compat.CompatPhoneNumbers.search_local: BACKLOG / param-mismatch/ param[1] (country)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.compat.CompatPhoneNumbers.search_toll_free: BACKLOG / param-mismatch/ param[1] (country)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.compat.CompatPhoneNumbers.update: BACKLOG / param-mismatch/ param[1] (sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs 'data'; kind
signalwire.rest.namespaces.compat.CompatQueues.dequeue_member: BACKLOG / param-mismatch/ param[1] (queue_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (call_sid)/ type 'any' vs 'string
signalwire.rest.namespaces.compat.CompatQueues.get_member: BACKLOG / param-mismatch/ param[1] (queue_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (call_sid)/ type 'any' vs 'string
signalwire.rest.namespaces.compat.CompatQueues.list_members: BACKLOG / param-mismatch/ param[1] (queue_sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs '
signalwire.rest.namespaces.compat.CompatQueues.update: BACKLOG / param-mismatch/ param[1] (sid)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs 'data'; kind
signalwire.rest.namespaces.compat.CompatRecordings.delete: BACKLOG / param-mismatch/ param[1] (sid)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.compat.CompatRecordings.get: BACKLOG / param-mismatch/ param[1] (sid)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.compat.CompatRecordings.list: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin; return-mismatch/ retur
signalwire.rest.namespaces.compat.CompatTokens.create: BACKLOG / param-mismatch/ param[1] (kwargs)/ name 'kwargs' vs 'data'; kind 'var_keyword' vs 'positional'; ; return-mismatch/ retur
signalwire.rest.namespaces.compat.CompatTranscriptions.delete: BACKLOG / param-mismatch/ param[1] (sid)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.compat.CompatTranscriptions.get: BACKLOG / param-mismatch/ param[1] (sid)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.compat.CompatTranscriptions.list: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin; return-mismatch/ retur
signalwire.rest.namespaces.datasphere.DatasphereDocuments.delete_chunk: BACKLOG / param-mismatch/ param[1] (document_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (chunk_id)/ type 'any' vs 'stri
signalwire.rest.namespaces.datasphere.DatasphereDocuments.get_chunk: BACKLOG / param-mismatch/ param[1] (document_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (chunk_id)/ type 'any' vs 'stri
signalwire.rest.namespaces.datasphere.DatasphereDocuments.list_chunks: BACKLOG / param-mismatch/ param[1] (document_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs
signalwire.rest.namespaces.datasphere.DatasphereDocuments.search: BACKLOG / param-mismatch/ param[1] (kwargs)/ name 'kwargs' vs 'data'; kind 'var_keyword' vs 'positional'; ; return-mismatch/ retur
signalwire.rest.namespaces.fabric.AutoMaterializedWebhook.create: BACKLOG / param-mismatch/ param[1] (kwargs)/ name 'kwargs' vs 'data'; kind 'var_keyword' vs 'positional'; ; return-mismatch/ retur
signalwire.rest.namespaces.fabric.CallFlowsResource.deploy_version: BACKLOG / param-mismatch/ param[1] (resource_id)/ name 'resource_id' vs 'id'; type 'any' vs 'string'; param-mismatch/ param[2] (kw
signalwire.rest.namespaces.fabric.CallFlowsResource.list_addresses: BACKLOG / param-mismatch/ param[1] (resource_id)/ name 'resource_id' vs 'id'; type 'any' vs 'string'; param-mismatch/ param[2] (pa
signalwire.rest.namespaces.fabric.CallFlowsResource.list_versions: BACKLOG / param-mismatch/ param[1] (resource_id)/ name 'resource_id' vs 'id'; type 'any' vs 'string'; param-mismatch/ param[2] (pa
signalwire.rest.namespaces.fabric.ConferenceRoomsResource.list_addresses: BACKLOG / param-mismatch/ param[1] (resource_id)/ name 'resource_id' vs 'id'; type 'any' vs 'string'; param-mismatch/ param[2] (pa
signalwire.rest.namespaces.fabric.FabricAddresses.get: BACKLOG / param-mismatch/ param[1] (address_id)/ name 'address_id' vs 'id'; type 'any' vs 'string'; return-mismatch/ returns 'any'
signalwire.rest.namespaces.fabric.FabricAddresses.list: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin; return-mismatch/ retur
signalwire.rest.namespaces.fabric.FabricTokens.create_embed_token: BACKLOG / param-mismatch/ param[1] (kwargs)/ name 'kwargs' vs 'data'; kind 'var_keyword' vs 'positional'; ; return-mismatch/ retur
signalwire.rest.namespaces.fabric.FabricTokens.create_guest_token: BACKLOG / param-mismatch/ param[1] (kwargs)/ name 'kwargs' vs 'data'; kind 'var_keyword' vs 'positional'; ; return-mismatch/ retur
signalwire.rest.namespaces.fabric.FabricTokens.create_invite_token: BACKLOG / param-mismatch/ param[1] (kwargs)/ name 'kwargs' vs 'data'; kind 'var_keyword' vs 'positional'; ; return-mismatch/ retur
signalwire.rest.namespaces.fabric.FabricTokens.create_subscriber_token: BACKLOG / param-mismatch/ param[1] (kwargs)/ name 'kwargs' vs 'data'; kind 'var_keyword' vs 'positional'; ; return-mismatch/ retur
signalwire.rest.namespaces.fabric.FabricTokens.refresh_subscriber_token: BACKLOG / param-mismatch/ param[1] (kwargs)/ name 'kwargs' vs 'data'; kind 'var_keyword' vs 'positional'; ; return-mismatch/ retur
signalwire.rest.namespaces.fabric.GenericResources.assign_domain_application: BACKLOG / param-mismatch/ param[1] (resource_id)/ name 'resource_id' vs 'id'; type 'any' vs 'string'; param-mismatch/ param[2] (kw
signalwire.rest.namespaces.fabric.GenericResources.assign_phone_route: BACKLOG / param-mismatch/ param[1] (resource_id)/ name 'resource_id' vs 'id'; type 'any' vs 'string'; param-mismatch/ param[2] (kw
signalwire.rest.namespaces.fabric.GenericResources.delete: BACKLOG / param-mismatch/ param[1] (resource_id)/ name 'resource_id' vs 'id'; type 'any' vs 'string'; return-mismatch/ returns 'an
signalwire.rest.namespaces.fabric.GenericResources.get: BACKLOG / param-mismatch/ param[1] (resource_id)/ name 'resource_id' vs 'id'; type 'any' vs 'string'; return-mismatch/ returns 'an
signalwire.rest.namespaces.fabric.GenericResources.list: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin; return-mismatch/ retur
signalwire.rest.namespaces.fabric.GenericResources.list_addresses: BACKLOG / param-mismatch/ param[1] (resource_id)/ name 'resource_id' vs 'id'; type 'any' vs 'string'; param-mismatch/ param[2] (pa
signalwire.rest.namespaces.fabric.SubscribersResource.create_sip_endpoint: BACKLOG / param-mismatch/ param[1] (subscriber_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs 'd
signalwire.rest.namespaces.fabric.SubscribersResource.delete_sip_endpoint: BACKLOG / param-mismatch/ param[1] (subscriber_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (endpoint_id)/ type 'any' vs 
signalwire.rest.namespaces.fabric.SubscribersResource.get_sip_endpoint: BACKLOG / param-mismatch/ param[1] (subscriber_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (endpoint_id)/ type 'any' vs 
signalwire.rest.namespaces.fabric.SubscribersResource.list_sip_endpoints: BACKLOG / param-mismatch/ param[1] (subscriber_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' 
signalwire.rest.namespaces.fabric.SubscribersResource.update_sip_endpoint: BACKLOG / param-mismatch/ param[1] (subscriber_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (endpoint_id)/ type 'any' vs 
signalwire.rest.namespaces.imported_numbers.ImportedNumbersResource.create: BACKLOG / param-mismatch/ param[1] (kwargs)/ name 'kwargs' vs 'data'; kind 'var_keyword' vs 'positional'; ; return-mismatch/ retur
signalwire.rest.namespaces.logs.ConferenceLogs.list: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin; return-mismatch/ retur
signalwire.rest.namespaces.logs.FaxLogs.get: BACKLOG / param-mismatch/ param[1] (log_id)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.logs.FaxLogs.list: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin; return-mismatch/ retur
signalwire.rest.namespaces.logs.MessageLogs.get: BACKLOG / param-mismatch/ param[1] (log_id)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.logs.MessageLogs.list: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin; return-mismatch/ retur
signalwire.rest.namespaces.logs.VoiceLogs.get: BACKLOG / param-mismatch/ param[1] (log_id)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.logs.VoiceLogs.list: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin; return-mismatch/ retur
signalwire.rest.namespaces.logs.VoiceLogs.list_events: BACKLOG / param-mismatch/ param[1] (log_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'pos
signalwire.rest.namespaces.lookup.LookupResource.phone_number: BACKLOG / param-mismatch/ param[1] (e164)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'posit
signalwire.rest.namespaces.mfa.MfaResource.call: BACKLOG / param-mismatch/ param[1] (kwargs)/ name 'kwargs' vs 'data'; kind 'var_keyword' vs 'positional'; ; return-mismatch/ retur
signalwire.rest.namespaces.mfa.MfaResource.sms: BACKLOG / param-mismatch/ param[1] (kwargs)/ name 'kwargs' vs 'data'; kind 'var_keyword' vs 'positional'; ; return-mismatch/ retur
signalwire.rest.namespaces.mfa.MfaResource.verify: BACKLOG / param-mismatch/ param[1] (request_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs 'data
signalwire.rest.namespaces.number_groups.NumberGroupsResource.add_membership: BACKLOG / param-mismatch/ param[1] (group_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs 'data';
signalwire.rest.namespaces.number_groups.NumberGroupsResource.delete_membership: BACKLOG / param-mismatch/ param[1] (membership_id)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.number_groups.NumberGroupsResource.get_membership: BACKLOG / param-mismatch/ param[1] (membership_id)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.number_groups.NumberGroupsResource.list_memberships: BACKLOG / param-mismatch/ param[1] (group_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'p
signalwire.rest.namespaces.phone_numbers.PhoneNumbersResource.search: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin; return-mismatch/ retur
signalwire.rest.namespaces.phone_numbers.PhoneNumbersResource.set_ai_agent: BACKLOG / param-mismatch/ param[1] (resource_id)/ name 'resource_id' vs 'sid'; param-mismatch/ param[3] (extra)/ kind 'var_keyword
signalwire.rest.namespaces.phone_numbers.PhoneNumbersResource.set_call_flow: BACKLOG / param-count-mismatch/ reference has 5 param(s), port has 4/ reference=['self', 'resource_id', 'flow_id; return-mismatch/
signalwire.rest.namespaces.phone_numbers.PhoneNumbersResource.set_cxml_application: BACKLOG / param-mismatch/ param[1] (resource_id)/ name 'resource_id' vs 'sid'; param-mismatch/ param[3] (extra)/ kind 'var_keyword
signalwire.rest.namespaces.phone_numbers.PhoneNumbersResource.set_cxml_webhook: BACKLOG / param-count-mismatch/ reference has 6 param(s), port has 4/ reference=['self', 'resource_id', 'url', '; return-mismatch/
signalwire.rest.namespaces.phone_numbers.PhoneNumbersResource.set_relay_application: BACKLOG / param-mismatch/ param[1] (resource_id)/ name 'resource_id' vs 'sid'; param-mismatch/ param[3] (extra)/ kind 'var_keyword
signalwire.rest.namespaces.phone_numbers.PhoneNumbersResource.set_relay_topic: BACKLOG / param-count-mismatch/ reference has 5 param(s), port has 4/ reference=['self', 'resource_id', 'topic',; return-mismatch/
signalwire.rest.namespaces.phone_numbers.PhoneNumbersResource.set_swml_webhook: BACKLOG / param-mismatch/ param[1] (resource_id)/ name 'resource_id' vs 'sid'; param-mismatch/ param[3] (extra)/ kind 'var_keyword
signalwire.rest.namespaces.project.ProjectTokens.create: BACKLOG / param-mismatch/ param[1] (kwargs)/ name 'kwargs' vs 'data'; kind 'var_keyword' vs 'positional'; ; return-mismatch/ retur
signalwire.rest.namespaces.project.ProjectTokens.delete: BACKLOG / param-mismatch/ param[1] (token_id)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.project.ProjectTokens.update: BACKLOG / param-mismatch/ param[1] (token_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs 'data';
signalwire.rest.namespaces.pubsub.PubSubResource.create_token: BACKLOG / param-mismatch/ param[1] (kwargs)/ name 'kwargs' vs 'data'; kind 'var_keyword' vs 'positional'; ; return-mismatch/ retur
signalwire.rest.namespaces.queues.QueuesResource.get_member: BACKLOG / param-mismatch/ param[1] (queue_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (member_id)/ type 'any' vs 'string
signalwire.rest.namespaces.queues.QueuesResource.get_next_member: BACKLOG / param-mismatch/ param[1] (queue_id)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.queues.QueuesResource.list_members: BACKLOG / param-mismatch/ param[1] (queue_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'p
signalwire.rest.namespaces.recordings.RecordingsResource.delete: BACKLOG / param-mismatch/ param[1] (recording_id)/ name 'recording_id' vs 'id'; type 'any' vs 'string'; return-mismatch/ returns '
signalwire.rest.namespaces.recordings.RecordingsResource.get: BACKLOG / param-mismatch/ param[1] (recording_id)/ name 'recording_id' vs 'id'; type 'any' vs 'string'; return-mismatch/ returns '
signalwire.rest.namespaces.recordings.RecordingsResource.list: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin; return-mismatch/ retur
signalwire.rest.namespaces.registry.RegistryBrands.create: BACKLOG / param-mismatch/ param[1] (kwargs)/ name 'kwargs' vs 'data'; kind 'var_keyword' vs 'positional'; ; return-mismatch/ retur
signalwire.rest.namespaces.registry.RegistryBrands.create_campaign: BACKLOG / param-mismatch/ param[1] (brand_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs 'data';
signalwire.rest.namespaces.registry.RegistryBrands.get: BACKLOG / param-mismatch/ param[1] (brand_id)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.registry.RegistryBrands.list: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin; return-mismatch/ retur
signalwire.rest.namespaces.registry.RegistryBrands.list_campaigns: BACKLOG / param-mismatch/ param[1] (brand_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'p
signalwire.rest.namespaces.registry.RegistryCampaigns.create_order: BACKLOG / param-mismatch/ param[1] (campaign_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs 'dat
signalwire.rest.namespaces.registry.RegistryCampaigns.get: BACKLOG / param-mismatch/ param[1] (campaign_id)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.registry.RegistryCampaigns.list_numbers: BACKLOG / param-mismatch/ param[1] (campaign_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs
signalwire.rest.namespaces.registry.RegistryCampaigns.list_orders: BACKLOG / param-mismatch/ param[1] (campaign_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs
signalwire.rest.namespaces.registry.RegistryCampaigns.update: BACKLOG / param-mismatch/ param[1] (campaign_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs 'dat
signalwire.rest.namespaces.registry.RegistryNumbers.delete: BACKLOG / param-mismatch/ param[1] (number_id)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.registry.RegistryOrders.get: BACKLOG / param-mismatch/ param[1] (order_id)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.short_codes.ShortCodesResource.get: BACKLOG / param-mismatch/ param[1] (short_code_id)/ name 'short_code_id' vs 'id'; type 'any' vs 'string'; return-mismatch/ returns
signalwire.rest.namespaces.short_codes.ShortCodesResource.list: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin; return-mismatch/ retur
signalwire.rest.namespaces.short_codes.ShortCodesResource.update: BACKLOG / param-mismatch/ param[1] (short_code_id)/ name 'short_code_id' vs 'id'; type 'any' vs 'string'; param-mismatch/ param[2]
signalwire.rest.namespaces.sip_profile.SipProfileResource.get: BACKLOG / return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.sip_profile.SipProfileResource.update: BACKLOG / param-mismatch/ param[1] (kwargs)/ name 'kwargs' vs 'data'; kind 'var_keyword' vs 'positional'; ; return-mismatch/ retur
signalwire.rest.namespaces.verified_callers.VerifiedCallersResource.redial_verification: BACKLOG / param-mismatch/ param[1] (caller_id)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.verified_callers.VerifiedCallersResource.submit_verification: BACKLOG / param-mismatch/ param[1] (caller_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs 'data'
signalwire.rest.namespaces.video.VideoConferenceTokens.get: BACKLOG / param-mismatch/ param[1] (token_id)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.video.VideoConferenceTokens.reset: BACKLOG / param-mismatch/ param[1] (token_id)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.video.VideoConferences.create_stream: BACKLOG / param-mismatch/ param[1] (conference_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs 'd
signalwire.rest.namespaces.video.VideoConferences.list_conference_tokens: BACKLOG / param-mismatch/ param[1] (conference_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' 
signalwire.rest.namespaces.video.VideoConferences.list_streams: BACKLOG / param-mismatch/ param[1] (conference_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' 
signalwire.rest.namespaces.video.VideoRoomRecordings.delete: BACKLOG / param-mismatch/ param[1] (recording_id)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.video.VideoRoomRecordings.get: BACKLOG / param-mismatch/ param[1] (recording_id)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.video.VideoRoomRecordings.list: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin; return-mismatch/ retur
signalwire.rest.namespaces.video.VideoRoomRecordings.list_events: BACKLOG / param-mismatch/ param[1] (recording_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' v
signalwire.rest.namespaces.video.VideoRoomSessions.get: BACKLOG / param-mismatch/ param[1] (session_id)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.video.VideoRoomSessions.list: BACKLOG / param-mismatch/ param[1] (params)/ kind 'var_keyword' vs 'positional'; type 'any' vs 'dict<strin; return-mismatch/ retur
signalwire.rest.namespaces.video.VideoRoomSessions.list_events: BACKLOG / param-mismatch/ param[1] (session_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 
signalwire.rest.namespaces.video.VideoRoomSessions.list_members: BACKLOG / param-mismatch/ param[1] (session_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 
signalwire.rest.namespaces.video.VideoRoomSessions.list_recordings: BACKLOG / param-mismatch/ param[1] (session_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 
signalwire.rest.namespaces.video.VideoRoomTokens.create: BACKLOG / param-mismatch/ param[1] (kwargs)/ name 'kwargs' vs 'data'; kind 'var_keyword' vs 'positional'; ; return-mismatch/ retur
signalwire.rest.namespaces.video.VideoRooms.create_stream: BACKLOG / param-mismatch/ param[1] (room_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs 'data'; 
signalwire.rest.namespaces.video.VideoRooms.list_streams: BACKLOG / param-mismatch/ param[1] (room_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (params)/ kind 'var_keyword' vs 'po
signalwire.rest.namespaces.video.VideoStreams.delete: BACKLOG / param-mismatch/ param[1] (stream_id)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.video.VideoStreams.get: BACKLOG / param-mismatch/ param[1] (stream_id)/ type 'any' vs 'string'; return-mismatch/ returns 'any' vs 'dict<string,any>'
signalwire.rest.namespaces.video.VideoStreams.update: BACKLOG / param-mismatch/ param[1] (stream_id)/ type 'any' vs 'string'; param-mismatch/ param[2] (kwargs)/ name 'kwargs' vs 'data'
signalwire.search.preprocess_document_content: BACKLOG / missing-port/ in reference, not in port
signalwire.search.preprocess_query: BACKLOG / missing-port/ in reference, not in port
