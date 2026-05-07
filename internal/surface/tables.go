// Package surface holds the lookup tables that map Go SDK types and
// functions to their Python-canonical reference names. Both
// cmd/enumerate-surface (names-only) and cmd/enumerate-signatures
// (full signatures) consume these tables.
package surface

// classTarget is one Python class destination for a Go struct's methods.
// Each (pyModule, pyClass) pair is created exactly once; methods accumulate
// across multiple mappings that share the same target.
type ClassTarget struct {
	Module  string
	Class   string
	Methods map[string]string // goMethod -> pyMethod
	// Alias emits the class symbol itself with no methods.  Used for Python
	// stub classes like ``Room``, ``StopResponse``, ``ToolError``.
	Alias bool
	// SyntheticMethods are Python method names to emit unconditionally
	// (without a matching Go method).  Used for ``from_payload`` classmethods
	// on relay events that Go expresses through package-level factory
	// constructors.
	SyntheticMethods []string
}

// structTable maps a Go ``<shortPkg>.<StructName>`` to one or more Python
// class targets.  Unmapped structs are treated as port-only extensions
// (they must appear in PORT_ADDITIONS.md to avoid drift, but because we
// simply don't emit them they are silently dropped).
var StructTable = map[string][]ClassTarget{
	// --- agent package: AgentBase + all its mixins ------------------------
	"agent.AgentBase": {
		ClassTarget{
			Module: "signalwire.core.agent_base", Class: "AgentBase",
			Methods: map[string]string{
				"NewAgentBase":          "__init__",
				"GetName":               "get_name",
				"Pom":                   "pom",
				"AddAnswerVerb":         "add_answer_verb",
				"AddPostAiVerb":         "add_post_ai_verb",
				"AddPostAnswerVerb":     "add_post_answer_verb",
				"AddPreAnswerVerb":      "add_pre_answer_verb",
				"AddSwaigQueryParams":   "add_swaig_query_params",
				"ClearPostAiVerbs":      "clear_post_ai_verbs",
				"ClearPostAnswerVerbs":  "clear_post_answer_verbs",
				"ClearPreAnswerVerbs":   "clear_pre_answer_verbs",
				"ClearSwaigQueryParams": "clear_swaig_query_params",
				"EnableSipRouting":      "enable_sip_routing",
				"OnDebugEvent":          "on_debug_event",
				"OnSummary":             "on_summary",
				"RegisterSipUsername":   "register_sip_username",
				"SetPostPromptUrl":      "set_post_prompt_url",
				"SetWebHookUrl":         "set_web_hook_url",
			},
			// ``get_full_url`` and ``auto_map_sip_usernames`` are helpers
			// absent from the Go port's surface — see PORT_OMISSIONS.md.
		},
		ClassTarget{
			Module: "signalwire.core.mixins.prompt_mixin", Class: "PromptMixin",
			Methods: map[string]string{
				"SetPromptText":       "set_prompt_text",
				"SetPostPrompt":       "set_post_prompt",
				"SetPromptPom":        "set_prompt_pom",
				"PromptAddSection":    "prompt_add_section",
				"PromptAddToSection":  "prompt_add_to_section",
				"PromptAddSubsection": "prompt_add_subsection",
				"PromptHasSection":    "prompt_has_section",
				"GetPrompt":           "get_prompt",
				"GetPostPrompt":       "get_post_prompt",
				"DefineContexts":      "define_contexts",
				"Contexts":            "contexts",
				"ResetContexts":       "reset_contexts",
			},
		},
		// Python additionally extracted a ``PromptManager`` class that
		// PromptMixin delegates to.  The user-facing surface is
		// identical (``agent.prompt_manager.X`` ≡ ``agent.X``).  Project
		// the same set of methods to PromptManager so the cross-language
		// audit treats both paths as covered.
		ClassTarget{
			Module: "signalwire.core.agent.prompt.manager", Class: "PromptManager",
			Methods: map[string]string{
				"SetPromptText":       "set_prompt_text",
				"SetPostPrompt":       "set_post_prompt",
				"SetPromptPom":        "set_prompt_pom",
				"PromptAddSection":    "prompt_add_section",
				"PromptAddToSection":  "prompt_add_to_section",
				"PromptAddSubsection": "prompt_add_subsection",
				"PromptHasSection":    "prompt_has_section",
				"GetPrompt":           "get_prompt",
				"GetPostPrompt":       "get_post_prompt",
				"GetRawPrompt":        "get_raw_prompt",
				"GetContexts":         "get_contexts",
				"DefineContexts":      "define_contexts",
			},
		},
		ClassTarget{
			Module: "signalwire.core.mixins.tool_mixin", Class: "ToolMixin",
			Methods: map[string]string{
				"DefineTool":            "define_tool",
				"DefineTools":           "define_tools",
				"RegisterSwaigFunction": "register_swaig_function",
				"OnFunctionCall":        "on_function_call",
			},
		},
		ClassTarget{
			Module: "signalwire.core.agent.tools.registry", Class: "ToolRegistry",
			Methods: map[string]string{
				"DefineTool":            "define_tool",
				"RegisterSwaigFunction": "register_swaig_function",
				"HasFunction":           "has_function",
				"GetFunction":           "get_function",
				"GetAllFunctions":       "get_all_functions",
				"RemoveFunction":        "remove_function",
			},
		},
		ClassTarget{
			Module: "signalwire.core.mixins.auth_mixin", Class: "AuthMixin",
			Methods: map[string]string{
				"ValidateBasicAuth":       "validate_basic_auth",
				"GetBasicAuthCredentials": "get_basic_auth_credentials",
			},
		},
		ClassTarget{
			Module: "signalwire.core.mixins.ai_config_mixin", Class: "AIConfigMixin",
			Methods: map[string]string{
				"AddHint":                "add_hint",
				"AddHints":               "add_hints",
				"AddPatternHint":         "add_pattern_hint",
				"AddLanguage":            "add_language",
				"SetLanguages":           "set_languages",
				"AddPronunciation":       "add_pronunciation",
				"SetPronunciations":      "set_pronunciations",
				"SetParam":               "set_param",
				"SetParams":              "set_params",
				"SetGlobalData":          "set_global_data",
				"UpdateGlobalData":       "update_global_data",
				"SetNativeFunctions":     "set_native_functions",
				"SetInternalFillers":     "set_internal_fillers",
				"AddInternalFiller":      "add_internal_filler",
				"EnableDebugEvents":      "enable_debug_events",
				"AddFunctionInclude":     "add_function_include",
				"SetFunctionIncludes":    "set_function_includes",
				"SetPromptLlmParams":     "set_prompt_llm_params",
				"SetPostPromptLlmParams": "set_post_prompt_llm_params",
				"AddMcpServer":           "add_mcp_server",
				"EnableMcpServer":        "enable_mcp_server",
			},
		},
		ClassTarget{
			Module: "signalwire.core.mixins.skill_mixin", Class: "SkillMixin",
			Methods: map[string]string{
				"AddSkill":    "add_skill",
				"RemoveSkill": "remove_skill",
				"ListSkills":  "list_skills",
				"HasSkill":    "has_skill",
			},
		},
		ClassTarget{
			Module: "signalwire.core.mixins.web_mixin", Class: "WebMixin",
			Methods: map[string]string{
				"Run":                      "run",
				"Serve":                    "serve",
				"AsRouter":                 "as_router",
				"SetDynamicConfigCallback": "set_dynamic_config_callback",
				"ManualSetProxyUrl":        "manual_set_proxy_url",
				"EnableDebugRoutes":        "enable_debug_routes",
				"OnRequest":                "on_request",
				"OnSwmlRequest":            "on_swml_request",
			},
		},
		ClassTarget{
			// StateMixin in Python exposes ``validate_tool_token`` publicly
			// (and ``_create_tool_token`` as a private helper). Project the
			// public method only — ``CreateToolToken`` remains a Go-side
			// AgentBase facade (matches .NET), but it is not enumerated as
			// a private mixin method on the Python side.
			Module: "signalwire.core.mixins.state_mixin", Class: "StateMixin",
			Methods: map[string]string{
				"ValidateToolToken": "validate_tool_token",
			},
		},
	},

	// --- server package ---------------------------------------------------
	"server.AgentServer": {{
		Module: "signalwire.agent_server", Class: "AgentServer",
		Methods: map[string]string{
			"NewAgentServer":      "__init__",
			"GetAgent":            "get_agent",
			"GetAgents":           "get_agents",
			"Register":            "register",
			"Unregister":          "unregister",
			"RegisterSipUsername": "register_sip_username",
			"Run":                 "run",
			"ServeStaticFiles":    "serve_static_files",
			"SetupSipRouting":     "setup_sip_routing",
		},
	}},

	// --- pom package ------------------------------------------------------
	// Typed Prompt Object Model: matches signalwire.pom.pom byte-for-byte
	// for the canonical render scenarios (see pkg/pom/pom_test.go and
	// signalwire-python/tests/unit/pom/test_pom_render_parity.py).
	"pom.PromptObjectModel": {{
		Module: "signalwire.pom.pom", Class: "PromptObjectModel",
		Methods: map[string]string{
			"NewPromptObjectModel": "__init__",
			"AddSection":           "add_section",
			"FindSection":          "find_section",
			"ToList":               "to_dict",
			"ToJSON":               "to_json",
			"ToYAML":               "to_yaml",
			"RenderMarkdown":       "render_markdown",
			"RenderXML":            "render_xml",
			"AddPomAsSubsection":   "add_pom_as_subsection",
		},
		// from_json / from_yaml are package-level constructors in Go
		// (pom.FromJSON / pom.FromYAML); see freeFnTable below.
		SyntheticMethods: []string{"from_json", "from_yaml"},
	}},
	"pom.Section": {{
		Module: "signalwire.pom.pom", Class: "Section",
		Methods: map[string]string{
			"NewSection":     "__init__",
			"AddBody":        "add_body",
			"AddBullets":     "add_bullets",
			"AddSubsection":  "add_subsection",
			"ToMap":          "to_dict",
			"RenderMarkdown": "render_markdown",
			"RenderXML":      "render_xml",
		},
	}},

	// --- swml package -----------------------------------------------------
	"swml.Service": {{
		Module: "signalwire.core.swml_service", Class: "SWMLService",
		Methods: map[string]string{
			"NewService":              "__init__",
			"GetDocument":             "get_document",
			"ResetDocument":           "reset_document",
			"GetBasicAuthCredentials": "get_basic_auth_credentials",
			"ExecuteVerb":             "add_verb",
			"ExecuteVerbToSection":    "add_verb_to_section",
			"RegisterRoutingCallback": "register_routing_callback",
			"OnRequest":               "on_request",
			"Render":                  "render_document",
			"Serve":                   "serve",
			// schema_utils accessor — exposes the SchemaUtils helper as a
			// public attribute, matching Python's ``self.schema_utils``.
			"SchemaUtils": "schema_utils",
		},
	}},

	// --- SchemaUtils (signalwire.utils.schema_utils.SchemaUtils) -------
	"swml.SchemaUtils": {{
		Module: "signalwire.utils.schema_utils", Class: "SchemaUtils",
		Methods: map[string]string{
			"NewSchemaUtils":              "__init__",
			"LoadSchema":                  "load_schema",
			"GetAllVerbNames":             "get_all_verb_names",
			"GetVerbProperties":           "get_verb_properties",
			"GetVerbRequiredProperties":   "get_verb_required_properties",
			"GetVerbParameters":           "get_verb_parameters",
			"ValidateVerb":                "validate_verb",
			"ValidateDocument":            "validate_document",
		},
	}},

	// --- SchemaValidationError (Python's exception class) --------------
	"swml.SchemaValidationError": {{
		Module: "signalwire.utils.schema_utils", Class: "SchemaValidationError",
		Methods: map[string]string{
			"NewSchemaValidationError": "__init__",
		},
	}},

	// --- swaig package ----------------------------------------------------
	"swaig.FunctionResult": {{
		Module: "signalwire.core.function_result", Class: "FunctionResult",
		Methods: map[string]string{
			"NewFunctionResult":        "__init__",
			"SetResponse":              "set_response",
			"SetPostProcess":           "set_post_process",
			"AddAction":                "add_action",
			"AddActions":               "add_actions",
			"ToMap":                    "to_dict",
			"Connect":                  "connect",
			"SwmlTransfer":             "swml_transfer",
			"Hangup":                   "hangup",
			"Hold":                     "hold",
			"WaitForUser":              "wait_for_user",
			"Stop":                     "stop",
			"UpdateGlobalData":         "update_global_data",
			"RemoveGlobalData":         "remove_global_data",
			"SetMetadata":              "set_metadata",
			"RemoveMetadata":           "remove_metadata",
			"SwmlUserEvent":            "swml_user_event",
			"SwmlChangeStep":           "swml_change_step",
			"SwmlChangeContext":        "swml_change_context",
			"SwitchContext":            "switch_context",
			"ReplaceInHistory":         "replace_in_history",
			"Say":                      "say",
			"PlayBackgroundFile":       "play_background_file",
			"StopBackgroundFile":       "stop_background_file",
			"RecordCall":               "record_call",
			"StopRecordCall":           "stop_record_call",
			"AddDynamicHints":          "add_dynamic_hints",
			"ClearDynamicHints":        "clear_dynamic_hints",
			"SetEndOfSpeechTimeout":    "set_end_of_speech_timeout",
			"SetSpeechEventTimeout":    "set_speech_event_timeout",
			"ToggleFunctions":          "toggle_functions",
			"EnableFunctionsOnTimeout": "enable_functions_on_timeout",
			"Pay":                      "pay",
			"CreatePaymentPrompt":      "create_payment_prompt",
			"CreatePaymentAction":      "create_payment_action",
			"CreatePaymentParameter":   "create_payment_parameter",
			"JoinRoom":                 "join_room",
			"JoinConference":           "join_conference",
			"SendSms":                  "send_sms",
			"SipRefer":                 "sip_refer",
			"Tap":                      "tap",
			"StopTap":                  "stop_tap",
			"ExecuteSwml":              "execute_swml",
			"ExecuteRpc":               "execute_rpc",
			"RpcAiMessage":             "rpc_ai_message",
			"RpcAiUnhold":              "rpc_ai_unhold",
			"RpcDial":                  "rpc_dial",
			"SimulateUserInput":        "simulate_user_input",
			"EnableExtensiveData":      "enable_extensive_data",
			"UpdateSettings":           "update_settings",
		},
	}},

	// --- relay package ----------------------------------------------------
	"relay.Client": {{
		Module: "signalwire.relay.client", Class: "RelayClient",
		Methods: map[string]string{
			"NewRelayClient": "__init__",
			"OnCall":         "on_call",
			"OnMessage":      "on_message",
			"Run":            "run",
			"Stop":           "disconnect",
			"Dial":           "dial",
			"SendMessage":    "send_message",
		},
	}},
	"relay.Call": {{
		Module: "signalwire.relay.call", Class: "Call",
		Methods: map[string]string{
			"Answer":             "answer",
			"Hangup":             "hangup",
			"Pass":               "pass_",
			"Transfer":           "transfer",
			"Play":               "play",
			"PlayAndCollect":     "play_and_collect",
			"Collect":            "collect",
			"Record":             "record",
			"Connect":            "connect",
			"Disconnect":         "disconnect",
			"SendDigits":         "send_digits",
			"Detect":             "detect",
			"SendFax":            "send_fax",
			"ReceiveFax":         "receive_fax",
			"Tap":                "tap",
			"Stream":             "stream",
			"JoinConference":     "join_conference",
			"LeaveConference":    "leave_conference",
			"AI":                 "ai",
			"AmazonBedrock":      "amazon_bedrock",
			"AIMessage":          "ai_message",
			"AIHold":             "ai_hold",
			"AIUnhold":           "ai_unhold",
			"Hold":               "hold",
			"Unhold":             "unhold",
			"Denoise":            "denoise",
			"DenoiseStop":        "denoise_stop",
			"JoinRoom":           "join_room",
			"LeaveRoom":          "leave_room",
			"QueueEnter":         "queue_enter",
			"QueueLeave":         "queue_leave",
			"BindDigit":          "bind_digit",
			"ClearDigitBindings": "clear_digit_bindings",
			"UserEvent":          "user_event",
			"Echo":               "echo",
			"Pay":                "pay",
			"Transcribe":         "transcribe",
			"LiveTranscribe":     "live_transcribe",
			"LiveTranslate":      "live_translate",
			"Refer":              "refer",
			"On":                 "on",
			"WaitFor":            "wait_for",
			"WaitForEnded":       "wait_for_ended",
			"String":             "__repr__",
		},
		// Call's constructor is unexported (`newCall`); Python's
		// ``__init__`` is an internal contract method.  Omit here;
		// see PORT_OMISSIONS.md.
		SyntheticMethods: []string{"__init__"},
	}},

	// Relay actions: each Go struct maps 1:1 to a Python class.  Python's
	// ``__init__`` is synthesised because Go uses unexported factories.
	"relay.Action": {{
		Module: "signalwire.relay.call", Class: "Action",
		Methods: map[string]string{
			"Wait":   "wait",
			"IsDone": "is_done",
			"Result": "result",
		},
		SyntheticMethods: []string{"__init__"},
	}},
	"relay.PlayAction": {{
		Module: "signalwire.relay.call", Class: "PlayAction",
		Methods: map[string]string{
			"Pause":  "pause",
			"Resume": "resume",
			"Stop":   "stop",
			"Volume": "volume",
		},
		SyntheticMethods: []string{"__init__"},
	}},
	"relay.RecordAction": {{
		Module: "signalwire.relay.call", Class: "RecordAction",
		Methods: map[string]string{
			"Pause":  "pause",
			"Resume": "resume",
			"Stop":   "stop",
		},
		SyntheticMethods: []string{"__init__"},
	}},
	"relay.DetectAction": {{
		Module: "signalwire.relay.call", Class: "DetectAction",
		Methods:          map[string]string{"Stop": "stop"},
		SyntheticMethods: []string{"__init__"},
	}},
	"relay.CollectAction": {{
		Module: "signalwire.relay.call", Class: "CollectAction",
		Methods: map[string]string{
			"Stop":             "stop",
			"StartInputTimers": "start_input_timers",
			"Volume":           "volume",
		},
		SyntheticMethods: []string{"__init__"},
	}},
	"relay.StandaloneCollectAction": {{
		Module: "signalwire.relay.call", Class: "StandaloneCollectAction",
		Methods: map[string]string{
			"Stop":             "stop",
			"StartInputTimers": "start_input_timers",
		},
		SyntheticMethods: []string{"__init__"},
	}},
	"relay.FaxAction": {{
		Module: "signalwire.relay.call", Class: "FaxAction",
		Methods:          map[string]string{"Stop": "stop"},
		SyntheticMethods: []string{"__init__"},
	}},
	"relay.TapAction": {{
		Module: "signalwire.relay.call", Class: "TapAction",
		Methods:          map[string]string{"Stop": "stop"},
		SyntheticMethods: []string{"__init__"},
	}},
	"relay.StreamAction": {{
		Module: "signalwire.relay.call", Class: "StreamAction",
		Methods:          map[string]string{"Stop": "stop"},
		SyntheticMethods: []string{"__init__"},
	}},
	"relay.PayAction": {{
		Module: "signalwire.relay.call", Class: "PayAction",
		Methods:          map[string]string{"Stop": "stop"},
		SyntheticMethods: []string{"__init__"},
	}},
	"relay.TranscribeAction": {{
		Module: "signalwire.relay.call", Class: "TranscribeAction",
		Methods:          map[string]string{"Stop": "stop"},
		SyntheticMethods: []string{"__init__"},
	}},
	"relay.AIAction": {{
		Module: "signalwire.relay.call", Class: "AIAction",
		Methods:          map[string]string{"Stop": "stop"},
		SyntheticMethods: []string{"__init__"},
	}},

	"relay.Message": {{
		Module: "signalwire.relay.message", Class: "Message",
		Methods: map[string]string{
			"On":     "on",
			"Wait":   "wait",
			"Result": "result",
			"IsDone": "is_done",
			"String": "__repr__",
		},
		SyntheticMethods: []string{"__init__"},
	}},

	// Relay events: each Go ``New<Event>`` factory plays the role of Python's
	// ``from_payload`` classmethod.  We emit the Python ``from_payload``
	// method synthetically whenever the Go factory exists.
	"relay.RelayEvent":          {eventTarget("RelayEvent")},
	"relay.CallStateEvent":      {eventTarget("CallStateEvent")},
	"relay.CallReceiveEvent":    {eventTarget("CallReceiveEvent")},
	"relay.PlayEvent":           {eventTarget("PlayEvent")},
	"relay.RecordEvent":         {eventTarget("RecordEvent")},
	"relay.CollectEvent":        {eventTarget("CollectEvent")},
	"relay.ConnectEvent":        {eventTarget("ConnectEvent")},
	"relay.DetectEvent":         {eventTarget("DetectEvent")},
	"relay.FaxEvent":            {eventTarget("FaxEvent")},
	"relay.TapEvent":            {eventTarget("TapEvent")},
	"relay.StreamEvent":         {eventTarget("StreamEvent")},
	"relay.SendDigitsEvent":     {eventTarget("SendDigitsEvent")},
	"relay.DialEvent":           {eventTarget("DialEvent")},
	"relay.ReferEvent":          {eventTarget("ReferEvent")},
	"relay.DenoiseEvent":        {eventTarget("DenoiseEvent")},
	"relay.PayEvent":            {eventTarget("PayEvent")},
	"relay.QueueEvent":          {eventTarget("QueueEvent")},
	"relay.EchoEvent":           {eventTarget("EchoEvent")},
	"relay.TranscribeEvent":     {eventTarget("TranscribeEvent")},
	"relay.HoldEvent":           {eventTarget("HoldEvent")},
	"relay.ConferenceEvent":     {eventTarget("ConferenceEvent")},
	"relay.CallingErrorEvent":   {eventTarget("CallingErrorEvent")},
	"relay.MessageReceiveEvent": {eventTarget("MessageReceiveEvent")},
	"relay.MessageStateEvent":   {eventTarget("MessageStateEvent")},
	// relay.AIEvent has no Python counterpart; it's a port-only extension.
	// See PORT_ADDITIONS.md.

	// --- rest package -----------------------------------------------------
	"rest.RestClient": {{
		Module: "signalwire.rest.client", Class: "RestClient",
		Methods: map[string]string{
			"NewRestClient": "__init__",
		},
	}},
	"rest.HttpClient": {{
		Module: "signalwire.rest._base", Class: "HttpClient",
		Methods: map[string]string{
			"NewHttpClient": "__init__",
			"Get":           "get",
			"Post":          "post",
			"Put":           "put",
			"Patch":         "patch",
			"Delete":        "delete",
		},
	}},
	"rest.SignalWireRestError": {{
		Module: "signalwire.rest._base", Class: "SignalWireRestError",
		Methods: map[string]string{
			"NewSignalWireRestError": "__init__",
		},
	}},
	"rest.CrudResource": {{
		Module: "signalwire.rest._base", Class: "CrudResource",
		Methods: map[string]string{
			"List":   "list",
			"Get":    "get",
			"Create": "create",
			"Update": "update",
			"Delete": "delete",
		},
	}},
	"rest.PaginatedIterator": {{
		Module: "signalwire.rest._pagination", Class: "PaginatedIterator",
		Methods: map[string]string{
			"NewPaginatedIterator": "__init__",
			"Next":                 "__next__",
		},
		SyntheticMethods: []string{"__iter__"},
	}},

	// rest/namespaces/common.go (Resource struct = Python's BaseResource)
	"namespaces.Resource": {{
		Module: "signalwire.rest._base", Class: "BaseResource",
		Methods:          map[string]string{},
		SyntheticMethods: []string{"__init__"},
	}},

	// REST namespaces — one Go struct per Python class.
	"namespaces.AddressesNamespace": {{
		Module: "signalwire.rest.namespaces.addresses", Class: "AddressesResource",
		Methods: map[string]string{
			"NewAddressesNamespace": "__init__",
			"List":                  "list",
			"Get":                   "get",
			"Create":                "create",
			"Delete":                "delete",
		},
	}},
	"namespaces.CallingNamespace": {{
		Module: "signalwire.rest.namespaces.calling", Class: "CallingNamespace",
		Methods: map[string]string{
			"NewCallingNamespace":     "__init__",
			"Dial":                    "dial",
			"End":                     "end",
			"Update":                  "update",
			"Disconnect":              "disconnect",
			"Refer":                   "refer",
			"Transfer":                "transfer",
			"Play":                    "play",
			"PlayStop":                "play_stop",
			"PlayPause":               "play_pause",
			"PlayResume":              "play_resume",
			"PlayVolume":              "play_volume",
			"Record":                  "record",
			"RecordStop":              "record_stop",
			"RecordPause":             "record_pause",
			"RecordResume":            "record_resume",
			"Collect":                 "collect",
			"CollectStop":             "collect_stop",
			"CollectStartInputTimers": "collect_start_input_timers",
			"Detect":                  "detect",
			"DetectStop":              "detect_stop",
			"Stream":                  "stream",
			"StreamStop":              "stream_stop",
			"Tap":                     "tap",
			"TapStop":                 "tap_stop",
			"Transcribe":              "transcribe",
			"TranscribeStop":          "transcribe_stop",
			"LiveTranscribe":          "live_transcribe",
			"LiveTranslate":           "live_translate",
			"SendFaxStop":             "send_fax_stop",
			"ReceiveFaxStop":          "receive_fax_stop",
			"Denoise":                 "denoise",
			"DenoiseStop":             "denoise_stop",
			"AIHold":                  "ai_hold",
			"AIUnhold":                "ai_unhold",
			"AIMessage":               "ai_message",
			"AIStop":                  "ai_stop",
			"UserEvent":               "user_event",
		},
	}},
	"namespaces.ChatNamespace": {{
		Module: "signalwire.rest.namespaces.chat", Class: "ChatResource",
		Methods: map[string]string{
			"NewChatNamespace": "__init__",
			"CreateToken":      "create_token",
		},
	}},
	"namespaces.DatasphereNamespace": {{
		Module: "signalwire.rest.namespaces.datasphere", Class: "DatasphereNamespace",
		Methods: map[string]string{"NewDatasphereNamespace": "__init__"},
	}},
	"namespaces.DatasphereDocuments": {{
		Module: "signalwire.rest.namespaces.datasphere", Class: "DatasphereDocuments",
		Methods: map[string]string{
			"Search":      "search",
			"ListChunks":  "list_chunks",
			"GetChunk":    "get_chunk",
			"DeleteChunk": "delete_chunk",
		},
		SyntheticMethods: []string{"__init__"},
	}},
	"namespaces.ImportedNumbersNamespace": {{
		Module: "signalwire.rest.namespaces.imported_numbers", Class: "ImportedNumbersResource",
		Methods: map[string]string{
			"NewImportedNumbersNamespace": "__init__",
			"Create":                      "create",
		},
	}},
	"namespaces.LookupNamespace": {{
		Module: "signalwire.rest.namespaces.lookup", Class: "LookupResource",
		Methods: map[string]string{
			"NewLookupNamespace": "__init__",
			"PhoneNumber":        "phone_number",
		},
	}},
	"namespaces.MFANamespace": {{
		Module: "signalwire.rest.namespaces.mfa", Class: "MfaResource",
		Methods: map[string]string{
			"NewMFANamespace": "__init__",
			"SMS":             "sms",
			"Call":            "call",
			"Verify":          "verify",
		},
	}},
	"namespaces.NumberGroupsNamespace": {{
		Module: "signalwire.rest.namespaces.number_groups", Class: "NumberGroupsResource",
		Methods: map[string]string{
			"NewNumberGroupsNamespace": "__init__",
			"ListMemberships":          "list_memberships",
			"GetMembership":            "get_membership",
			"AddMembership":            "add_membership",
			"DeleteMembership":         "delete_membership",
		},
	}},
	"namespaces.PhoneNumbersNamespace": {{
		Module: "signalwire.rest.namespaces.phone_numbers", Class: "PhoneNumbersResource",
		Methods: map[string]string{
			"NewPhoneNumbersNamespace": "__init__",
			"Search":                   "search",
			"SetSwmlWebhook":           "set_swml_webhook",
			"SetCxmlWebhook":           "set_cxml_webhook",
			"SetCxmlApplication":       "set_cxml_application",
			"SetAiAgent":               "set_ai_agent",
			"SetCallFlow":              "set_call_flow",
			"SetRelayApplication":      "set_relay_application",
			"SetRelayTopic":            "set_relay_topic",
		},
	}},
	"namespaces.PubSubNamespace": {{
		Module: "signalwire.rest.namespaces.pubsub", Class: "PubSubResource",
		Methods: map[string]string{
			"NewPubSubNamespace": "__init__",
			"CreateToken":        "create_token",
		},
	}},
	"namespaces.QueuesNamespace": {{
		Module: "signalwire.rest.namespaces.queues", Class: "QueuesResource",
		Methods: map[string]string{
			"NewQueuesNamespace": "__init__",
			"ListMembers":        "list_members",
			"GetMember":          "get_member",
			"GetNextMember":      "get_next_member",
		},
	}},
	"namespaces.RecordingsNamespace": {{
		Module: "signalwire.rest.namespaces.recordings", Class: "RecordingsResource",
		Methods: map[string]string{
			"NewRecordingsNamespace": "__init__",
			"List":                   "list",
			"Get":                    "get",
			"Delete":                 "delete",
		},
	}},
	"namespaces.ShortCodesNamespace": {{
		Module: "signalwire.rest.namespaces.short_codes", Class: "ShortCodesResource",
		Methods: map[string]string{
			"NewShortCodesNamespace": "__init__",
			"List":                   "list",
			"Get":                    "get",
			"Update":                 "update",
		},
	}},
	"namespaces.SipProfileNamespace": {{
		Module: "signalwire.rest.namespaces.sip_profile", Class: "SipProfileResource",
		Methods: map[string]string{
			"NewSipProfileNamespace": "__init__",
			"Get":                    "get",
			"Update":                 "update",
		},
	}},
	"namespaces.VerifiedCallersNamespace": {{
		Module: "signalwire.rest.namespaces.verified_callers", Class: "VerifiedCallersResource",
		Methods: map[string]string{
			"NewVerifiedCallersNamespace": "__init__",
			"RedialVerification":          "redial_verification",
			"SubmitVerification":          "submit_verification",
		},
	}},

	// Fabric namespace
	"namespaces.FabricNamespace": {{
		Module: "signalwire.rest.namespaces.fabric", Class: "FabricNamespace",
		Methods: map[string]string{"NewFabricNamespace": "__init__"},
	}},
	"namespaces.FabricAddresses": {{
		Module: "signalwire.rest.namespaces.fabric", Class: "FabricAddresses",
		Methods: map[string]string{
			"List": "list",
			"Get":  "get",
		},
	}},
	"namespaces.FabricTokens": {{
		Module: "signalwire.rest.namespaces.fabric", Class: "FabricTokens",
		Methods: map[string]string{
			"CreateSubscriberToken":  "create_subscriber_token",
			"RefreshSubscriberToken": "refresh_subscriber_token",
			"CreateInviteToken":      "create_invite_token",
			"CreateGuestToken":       "create_guest_token",
			"CreateEmbedToken":       "create_embed_token",
		},
		SyntheticMethods: []string{"__init__"},
	}},
	"namespaces.ConferenceRoomsResource": {{
		Module: "signalwire.rest.namespaces.fabric", Class: "ConferenceRoomsResource",
		Methods: map[string]string{"ListAddresses": "list_addresses"},
	}},
	"namespaces.SubscribersResource": {{
		Module: "signalwire.rest.namespaces.fabric", Class: "SubscribersResource",
		Methods: map[string]string{
			"ListSIPEndpoints":   "list_sip_endpoints",
			"CreateSIPEndpoint":  "create_sip_endpoint",
			"GetSIPEndpoint":     "get_sip_endpoint",
			"UpdateSIPEndpoint":  "update_sip_endpoint",
			"DeleteSIPEndpoint":  "delete_sip_endpoint",
		},
	}},
	"namespaces.CallFlowsResource": {{
		Module: "signalwire.rest.namespaces.fabric", Class: "CallFlowsResource",
		Methods: map[string]string{
			"ListAddresses": "list_addresses",
			"ListVersions":  "list_versions",
			"DeployVersion": "deploy_version",
		},
	}},
	"namespaces.GenericResources": {{
		Module: "signalwire.rest.namespaces.fabric", Class: "GenericResources",
		Methods: map[string]string{
			"List":                    "list",
			"Get":                     "get",
			"Delete":                  "delete",
			"ListAddresses":           "list_addresses",
			"AssignPhoneRoute":        "assign_phone_route",
			"AssignDomainApplication": "assign_domain_application",
		},
	}},
	"namespaces.AutoMaterializedWebhookResource": {{
		Module: "signalwire.rest.namespaces.fabric", Class: "AutoMaterializedWebhook",
		Methods: map[string]string{"Create": "create"},
	}},
	"namespaces.CxmlApplicationsResource": {{
		Module: "signalwire.rest.namespaces.fabric", Class: "CxmlApplicationsResource",
		Methods: map[string]string{
			"Create": "create",
		},
	}},

	// Compat namespace
	"namespaces.CompatNamespace": {{
		Module: "signalwire.rest.namespaces.compat", Class: "CompatNamespace",
		Methods: map[string]string{"NewCompatNamespace": "__init__"},
	}},
	"namespaces.CompatAccounts": {{
		Module: "signalwire.rest.namespaces.compat", Class: "CompatAccounts",
		Methods: map[string]string{
			"List":   "list",
			"Get":    "get",
			"Create": "create",
			"Update": "update",
		},
		SyntheticMethods: []string{"__init__"},
		// __init__ is synthesized because Go uses NewCompatNamespace as the
		// composite constructor; CompatAccounts has no exported New<X>.
	}},
	"namespaces.CompatCalls": {{
		Module: "signalwire.rest.namespaces.compat", Class: "CompatCalls",
		Methods: map[string]string{
			"Update":          "update",
			"StartRecording":  "start_recording",
			"UpdateRecording": "update_recording",
			"StartStream":     "start_stream",
			"StopStream":      "stop_stream",
		},
	}},
	"namespaces.CompatMessages": {{
		Module: "signalwire.rest.namespaces.compat", Class: "CompatMessages",
		Methods: map[string]string{
			"Update":      "update",
			"ListMedia":   "list_media",
			"GetMedia":    "get_media",
			"DeleteMedia": "delete_media",
		},
	}},
	"namespaces.CompatFaxes": {{
		Module: "signalwire.rest.namespaces.compat", Class: "CompatFaxes",
		Methods: map[string]string{
			"Update":      "update",
			"ListMedia":   "list_media",
			"GetMedia":    "get_media",
			"DeleteMedia": "delete_media",
		},
	}},
	"namespaces.CompatConferences": {{
		Module: "signalwire.rest.namespaces.compat", Class: "CompatConferences",
		Methods: map[string]string{
			"List":              "list",
			"Get":               "get",
			"Update":            "update",
			"ListParticipants":  "list_participants",
			"GetParticipant":    "get_participant",
			"UpdateParticipant": "update_participant",
			"RemoveParticipant": "remove_participant",
			"ListRecordings":    "list_recordings",
			"GetRecording":      "get_recording",
			"UpdateRecording":   "update_recording",
			"DeleteRecording":   "delete_recording",
			"StartStream":       "start_stream",
			"StopStream":        "stop_stream",
		},
	}},
	"namespaces.CompatPhoneNumbers": {{
		Module: "signalwire.rest.namespaces.compat", Class: "CompatPhoneNumbers",
		Methods: map[string]string{
			"List":                   "list",
			"Get":                    "get",
			"Update":                 "update",
			"Delete":                 "delete",
			"ImportNumber":           "import_number",
			"Purchase":               "purchase",
			"SearchLocal":            "search_local",
			"SearchTollFree":         "search_toll_free",
			"ListAvailableCountries": "list_available_countries",
		},
		SyntheticMethods: []string{"__init__"},
	}},
	"namespaces.CompatApplications": {{
		Module: "signalwire.rest.namespaces.compat", Class: "CompatApplications",
		Methods: map[string]string{"Update": "update"},
	}},
	"namespaces.CompatLamlBins": {{
		Module: "signalwire.rest.namespaces.compat", Class: "CompatLamlBins",
		Methods: map[string]string{"Update": "update"},
	}},
	"namespaces.CompatQueues": {{
		Module: "signalwire.rest.namespaces.compat", Class: "CompatQueues",
		Methods: map[string]string{
			"Update":        "update",
			"ListMembers":   "list_members",
			"GetMember":     "get_member",
			"DequeueMember": "dequeue_member",
		},
	}},
	"namespaces.CompatRecordings": {{
		Module: "signalwire.rest.namespaces.compat", Class: "CompatRecordings",
		Methods: map[string]string{
			"List":   "list",
			"Get":    "get",
			"Delete": "delete",
		},
	}},
	"namespaces.CompatTranscriptions": {{
		Module: "signalwire.rest.namespaces.compat", Class: "CompatTranscriptions",
		Methods: map[string]string{
			"List":   "list",
			"Get":    "get",
			"Delete": "delete",
		},
	}},
	"namespaces.CompatTokens": {{
		Module: "signalwire.rest.namespaces.compat", Class: "CompatTokens",
		Methods: map[string]string{
			"Create": "create",
			"Update": "update",
			"Delete": "delete",
		},
	}},

	// Video namespace
	"namespaces.VideoNamespace": {{
		Module: "signalwire.rest.namespaces.video", Class: "VideoNamespace",
		Methods: map[string]string{"NewVideoNamespace": "__init__"},
	}},
	"namespaces.VideoRooms": {{
		Module: "signalwire.rest.namespaces.video", Class: "VideoRooms",
		Methods: map[string]string{
			"ListStreams":  "list_streams",
			"CreateStream": "create_stream",
		},
	}},
	"namespaces.VideoRoomTokens": {{
		Module: "signalwire.rest.namespaces.video", Class: "VideoRoomTokens",
		Methods: map[string]string{"Create": "create"},
	}},
	"namespaces.VideoRoomSessions": {{
		Module: "signalwire.rest.namespaces.video", Class: "VideoRoomSessions",
		Methods: map[string]string{
			"List":           "list",
			"Get":            "get",
			"ListEvents":     "list_events",
			"ListMembers":    "list_members",
			"ListRecordings": "list_recordings",
		},
	}},
	"namespaces.VideoRoomRecordings": {{
		Module: "signalwire.rest.namespaces.video", Class: "VideoRoomRecordings",
		Methods: map[string]string{
			"List":       "list",
			"Get":        "get",
			"Delete":     "delete",
			"ListEvents": "list_events",
		},
	}},
	"namespaces.VideoConferences": {{
		Module: "signalwire.rest.namespaces.video", Class: "VideoConferences",
		Methods: map[string]string{
			"ListStreams":           "list_streams",
			"CreateStream":          "create_stream",
			"ListConferenceTokens":  "list_conference_tokens",
		},
	}},
	"namespaces.VideoConferenceTokens": {{
		Module: "signalwire.rest.namespaces.video", Class: "VideoConferenceTokens",
		Methods: map[string]string{
			"Get":   "get",
			"Reset": "reset",
		},
	}},
	"namespaces.VideoStreams": {{
		Module: "signalwire.rest.namespaces.video", Class: "VideoStreams",
		Methods: map[string]string{
			"Get":    "get",
			"Update": "update",
			"Delete": "delete",
		},
	}},

	// Project / Registry / Logs namespaces
	"namespaces.ProjectNamespace": {{
		Module: "signalwire.rest.namespaces.project", Class: "ProjectNamespace",
		Methods: map[string]string{"NewProjectNamespace": "__init__"},
	}},
	"namespaces.ProjectTokens": {{
		Module: "signalwire.rest.namespaces.project", Class: "ProjectTokens",
		Methods: map[string]string{
			"Create": "create",
			"Update": "update",
			"Delete": "delete",
		},
		SyntheticMethods: []string{"__init__"},
	}},
	"namespaces.LogsNamespace": {{
		Module: "signalwire.rest.namespaces.logs", Class: "LogsNamespace",
		Methods: map[string]string{"NewLogsNamespace": "__init__"},
	}},
	"namespaces.MessageLogs": {{
		Module: "signalwire.rest.namespaces.logs", Class: "MessageLogs",
		Methods: map[string]string{"List": "list", "Get": "get"},
	}},
	"namespaces.VoiceLogs": {{
		Module: "signalwire.rest.namespaces.logs", Class: "VoiceLogs",
		Methods: map[string]string{"List": "list", "Get": "get", "ListEvents": "list_events"},
	}},
	"namespaces.FaxLogs": {{
		Module: "signalwire.rest.namespaces.logs", Class: "FaxLogs",
		Methods: map[string]string{"List": "list", "Get": "get"},
	}},
	"namespaces.ConferenceLogs": {{
		Module: "signalwire.rest.namespaces.logs", Class: "ConferenceLogs",
		Methods: map[string]string{"List": "list"},
	}},
	"namespaces.RegistryNamespace": {{
		Module: "signalwire.rest.namespaces.registry", Class: "RegistryNamespace",
		Methods: map[string]string{"NewRegistryNamespace": "__init__"},
	}},
	"namespaces.RegistryBrands": {{
		Module: "signalwire.rest.namespaces.registry", Class: "RegistryBrands",
		Methods: map[string]string{
			"List":           "list",
			"Create":         "create",
			"Get":            "get",
			"ListCampaigns":  "list_campaigns",
			"CreateCampaign": "create_campaign",
		},
	}},
	"namespaces.RegistryCampaigns": {{
		Module: "signalwire.rest.namespaces.registry", Class: "RegistryCampaigns",
		Methods: map[string]string{
			"Get":          "get",
			"Update":       "update",
			"ListNumbers":  "list_numbers",
			"ListOrders":   "list_orders",
			"CreateOrder":  "create_order",
		},
	}},
	"namespaces.RegistryOrders": {{
		Module: "signalwire.rest.namespaces.registry", Class: "RegistryOrders",
		Methods: map[string]string{"Get": "get"},
	}},
	"namespaces.RegistryNumbers": {{
		Module: "signalwire.rest.namespaces.registry", Class: "RegistryNumbers",
		Methods: map[string]string{"Delete": "delete"},
	}},

	// --- contexts package -------------------------------------------------
	"contexts.ContextBuilder": {{
		Module: "signalwire.core.contexts", Class: "ContextBuilder",
		Methods: map[string]string{
			"NewContextBuilder": "__init__",
			"AddContext":        "add_context",
			"GetContext":        "get_context",
			"Reset":             "reset",
			"ToMap":             "to_dict",
			"Validate":          "validate",
		},
	}},
	"contexts.Context": {{
		Module: "signalwire.core.contexts", Class: "Context",
		Methods: map[string]string{
			"AddStep":          "add_step",
			"GetStep":          "get_step",
			"RemoveStep":       "remove_step",
			"MoveStep":         "move_step",
			"SetInitialStep":   "set_initial_step",
			"SetValidContexts": "set_valid_contexts",
			"SetValidSteps":    "set_valid_steps",
			"SetPostPrompt":    "set_post_prompt",
			"SetSystemPrompt":  "set_system_prompt",
			"SetPrompt":        "set_prompt",
			"SetConsolidate":   "set_consolidate",
			"SetFullReset":     "set_full_reset",
			"SetUserPrompt":    "set_user_prompt",
			"SetIsolated":      "set_isolated",
			"AddSection":       "add_section",
			"AddBullets":       "add_bullets",
			"AddSystemSection": "add_system_section",
			"AddSystemBullets": "add_system_bullets",
			"SetEnterFillers":  "set_enter_fillers",
			"SetExitFillers":   "set_exit_fillers",
			"AddEnterFiller":   "add_enter_filler",
			"AddExitFiller":    "add_exit_filler",
			"ToMap":            "to_dict",
		},
		SyntheticMethods: []string{"__init__"},
	}},
	"contexts.Step": {{
		Module: "signalwire.core.contexts", Class: "Step",
		Methods: map[string]string{
			"SetText":               "set_text",
			"AddSection":            "add_section",
			"AddBullets":            "add_bullets",
			"SetStepCriteria":       "set_step_criteria",
			"SetFunctions":          "set_functions",
			"SetValidSteps":         "set_valid_steps",
			"SetValidContexts":      "set_valid_contexts",
			"SetEnd":                "set_end",
			"SetSkipUserTurn":       "set_skip_user_turn",
			"SetSkipToNextStep":     "set_skip_to_next_step",
			"SetGatherInfo":         "set_gather_info",
			"AddGatherQuestion":     "add_gather_question",
			"ClearSections":         "clear_sections",
			"SetResetSystemPrompt":  "set_reset_system_prompt",
			"SetResetUserPrompt":    "set_reset_user_prompt",
			"SetResetConsolidate":   "set_reset_consolidate",
			"SetResetFullReset":     "set_reset_full_reset",
			"ToMap":                 "to_dict",
		},
		SyntheticMethods: []string{"__init__"},
	}},
	"contexts.GatherInfo": {{
		Module: "signalwire.core.contexts", Class: "GatherInfo",
		Methods: map[string]string{
			"AddQuestion": "add_question",
			"ToMap":       "to_dict",
		},
		SyntheticMethods: []string{"__init__"},
	}},
	"contexts.GatherQuestion": {{
		Module: "signalwire.core.contexts", Class: "GatherQuestion",
		Methods:          map[string]string{"ToMap": "to_dict"},
		SyntheticMethods: []string{"__init__"},
	}},

	// --- datamap package --------------------------------------------------
	"datamap.DataMap": {{
		Module: "signalwire.core.data_map", Class: "DataMap",
		Methods: map[string]string{
			// Constructor lives as ``datamap.New``; remapped below via
			// the freeFnAsInit table so we still emit ``__init__``.
			"Purpose":            "purpose",
			"Description":        "description",
			"Parameter":          "parameter",
			"Params":             "params",
			"Body":               "body",
			"Expression":         "expression",
			"Webhook":            "webhook",
			"WebhookExpressions": "webhook_expressions",
			"Output":             "output",
			"FallbackOutput":     "fallback_output",
			"Foreach":            "foreach",
			"ErrorKeys":          "error_keys",
			"GlobalErrorKeys":    "global_error_keys",
			"ToSwaigFunction":    "to_swaig_function",
		},
	}},

	// --- security package -------------------------------------------------
	// The Go port exposes only the TokenFactory surface (CreateToken /
	// ValidateToken) from SessionManager; the full Python
	// session-management API is under PORT_OMISSIONS.md.
	"security.SessionManager": {{
		Module: "signalwire.core.security.session_manager", Class: "SessionManager",
		Methods: map[string]string{
			"NewSessionManager": "__init__",
			"CreateToken":       "create_tool_token",
			"ValidateToken":     "validate_tool_token",
		},
	}},

	// --- skills package ---------------------------------------------------
	"skills.SkillManager": {{
		Module: "signalwire.core.skill_manager", Class: "SkillManager",
		Methods: map[string]string{
			"NewSkillManager":  "__init__",
			"LoadSkill":        "load_skill",
			"UnloadSkill":      "unload_skill",
			"ListLoadedSkills": "list_loaded_skills",
			"HasSkill":         "has_skill",
			"GetSkill":         "get_skill",
		},
	}},
	"skills.BaseSkill": {{
		Module: "signalwire.core.skill_base", Class: "SkillBase",
		Methods: map[string]string{
			"GetParameterSchema": "get_parameter_schema",
			"GetHints":           "get_hints",
			"GetGlobalData":      "get_global_data",
			"GetPromptSections":  "get_prompt_sections",
			"Cleanup":            "cleanup",
			"GetInstanceKey":     "get_instance_key",
		},
		SyntheticMethods: []string{"__init__"},
	}},
	"skills.SkillRegistry": {{
		// Python's `signalwire.skills.registry.SkillRegistry` is an
		// instance class with `add_skill_directory` + `_external_paths`.
		// Go mirrors only the parity-relevant surface; the package-level
		// `RegisterSkill` / `GetSkillFactory` / `ListSkills` functions
		// remain the canonical Go API for compile-time skill registration
		// and are projected separately as `signalwire.*` free functions
		// (see FreeFnTable below).
		Module: "signalwire.skills.registry", Class: "SkillRegistry",
		Methods: map[string]string{
			"NewSkillRegistry":  "__init__",
			"AddSkillDirectory": "add_skill_directory",
		},
	}},

	// --- prefabs package --------------------------------------------------
	"prefabs.ConciergeAgent": {{
		Module: "signalwire.prefabs.concierge", Class: "ConciergeAgent",
		Methods: map[string]string{
			"NewConciergeAgent": "__init__",
			"OnSummary":         "on_summary",
			"CheckAvailability": "check_availability",
			"GetDirections":     "get_directions",
		},
	}},
	"prefabs.FAQBotAgent": {{
		Module: "signalwire.prefabs.faq_bot", Class: "FAQBotAgent",
		Methods: map[string]string{
			"NewFAQBotAgent": "__init__",
			"OnSummary":      "on_summary",
			"SearchFaqs":     "search_faqs",
		},
	}},
	"prefabs.InfoGathererAgent": {{
		Module: "signalwire.prefabs.info_gatherer", Class: "InfoGathererAgent",
		Methods: map[string]string{
			"NewInfoGathererAgent": "__init__",
			"OnSwmlRequest":        "on_swml_request",
			"SetQuestionCallback":  "set_question_callback",
			"StartQuestions":       "start_questions",
			"SubmitAnswer":         "submit_answer",
		},
	}},
	"prefabs.ReceptionistAgent": {{
		Module: "signalwire.prefabs.receptionist", Class: "ReceptionistAgent",
		Methods: map[string]string{
			"NewReceptionistAgent": "__init__",
			"OnSummary":            "on_summary",
		},
	}},
	"prefabs.SurveyAgent": {{
		Module: "signalwire.prefabs.survey", Class: "SurveyAgent",
		Methods: map[string]string{
			"NewSurveyAgent":   "__init__",
			"OnSummary":        "on_summary",
			"LogResponse":      "log_response",
			"ValidateResponse": "validate_response",
		},
	}},

	// --- livewire package -------------------------------------------------
	"livewire.Agent": {{
		Module: "signalwire.livewire", Class: "Agent",
		Methods: map[string]string{
			"NewAgent":            "__init__",
			"OnEnter":             "on_enter",
			"OnExit":              "on_exit",
			"OnUserTurnCompleted": "on_user_turn_completed",
			"LLMNode":             "llm_node",
			"STTNode":             "stt_node",
			"TTSNode":             "tts_node",
			"UpdateInstructions":  "update_instructions",
			"UpdateTools":         "update_tools",
			"Session":             "session",
		},
	}},
	"livewire.AgentSession": {{
		Module: "signalwire.livewire", Class: "AgentSession",
		Methods: map[string]string{
			"NewAgentSession": "__init__",
			"Start":           "start",
			"Say":             "say",
			"GenerateReply":   "generate_reply",
			"Interrupt":       "interrupt",
			"UpdateAgent":     "update_agent",
			"History":         "history",
			"Userdata":        "userdata",
		},
	}},
	"livewire.JobContext": {{
		Module: "signalwire.livewire", Class: "JobContext",
		Methods: map[string]string{
			"Connect":            "connect",
			"WaitForParticipant": "wait_for_participant",
		},
		SyntheticMethods: []string{"__init__"},
	}},
	"livewire.RunContext": {{
		Module: "signalwire.livewire", Class: "RunContext",
		Methods:          map[string]string{"Userdata": "userdata"},
		SyntheticMethods: []string{"__init__"},
	}},
	"livewire.JobProcess": {{
		Module: "signalwire.livewire", Class: "JobProcess",
		Methods:          map[string]string{},
		SyntheticMethods: []string{"__init__"},
	}},
	"livewire.Room":         {{Module: "signalwire.livewire", Class: "Room", Methods: map[string]string{}, Alias: true}},
	"livewire.StopResponse": {{Module: "signalwire.livewire", Class: "StopResponse", Methods: map[string]string{}, Alias: true}},
	"livewire.AgentHandoff": {{
		Module: "signalwire.livewire", Class: "AgentHandoff",
		Methods:          map[string]string{},
		SyntheticMethods: []string{"__init__"},
	}},
	"livewire.LiveServer": {{
		Module: "signalwire.livewire", Class: "AgentServer",
		Methods: map[string]string{
			"RTCSession": "rtc_session",
		},
		SyntheticMethods: []string{"__init__"},
	}},

	// Livewire plugins — Python classes are stubs (only __init__).
	"livewire.DeepgramSTT": {{
		Module: "signalwire.livewire.plugins", Class: "DeepgramSTT",
		Methods:          map[string]string{},
		SyntheticMethods: []string{"__init__"},
	}},
	"livewire.ElevenLabsTTS": {{
		Module: "signalwire.livewire.plugins", Class: "ElevenLabsTTS",
		Methods:          map[string]string{},
		SyntheticMethods: []string{"__init__"},
	}},
	"livewire.CartesiaTTS": {{
		Module: "signalwire.livewire.plugins", Class: "CartesiaTTS",
		Methods:          map[string]string{},
		SyntheticMethods: []string{"__init__"},
	}},
	"livewire.OpenAILLM": {{
		Module: "signalwire.livewire.plugins", Class: "OpenAILLM",
		Methods:          map[string]string{},
		SyntheticMethods: []string{"__init__"},
	}},
	"livewire.SileroVAD": {{
		Module: "signalwire.livewire.plugins", Class: "SileroVAD",
		Methods:          map[string]string{"Load": "load"},
		SyntheticMethods: []string{"__init__"},
	}},
	// Go-only livewire plugins (GoogleSTT, OpenAITTS) are port-only extensions.
}

// freeFnTable maps a Go ``<shortPkg>.<FuncName>`` to a Python module-level
// function.  Only used for genuinely free-standing functions — factory
// constructors (``New<Struct>``) that should become ``__init__`` are
// declared via ``factoryInit`` instead.
var FreeFnTable = map[string]struct{ Module, Name string }{
	// Top-level signalwire package entry points
	"agent.RunAgent":              {Module: "signalwire", Name: "run_agent"},
	"agent.StartAgent":            {Module: "signalwire", Name: "start_agent"},
	"skills.RegisterSkill":        {Module: "signalwire", Name: "register_skill"},
	"skills.AddSkillDirectory":    {Module: "signalwire", Name: "add_skill_directory"},
	"skills.ListSkills":           {Module: "signalwire", Name: "list_skills"},
	"skills.ListSkillsWithParams": {Module: "signalwire", Name: "list_skills_with_params"},
	"rest.NewRestClient":          {Module: "signalwire", Name: "RestClient"},

	// Core modules
	"contexts.CreateSimpleContext":  {Module: "signalwire.core.contexts", Name: "create_simple_context"},
	"datamap.CreateSimpleApiTool":   {Module: "signalwire.core.data_map", Name: "create_simple_api_tool"},
	"datamap.CreateExpressionTool":  {Module: "signalwire.core.data_map", Name: "create_expression_tool"},
	"relay.ParseEvent":              {Module: "signalwire.relay.event", Name: "parse_event"},

	// Utilities — SSRF guard, projects onto Python's free function
	// signalwire.utils.url_validator.validate_url.
	"util.ValidateURL": {Module: "signalwire.utils.url_validator", Name: "validate_url"},

	// Execution-mode helpers (cross-port serverless detection contract):
	// Python ships get_execution_mode in signalwire.core.logging_config and
	// is_serverless_mode in signalwire.utils. Go places both under pkg/util
	// for cohesion; the projections below restore the Python paths.
	"util.GetExecutionMode": {Module: "signalwire.core.logging_config", Name: "get_execution_mode"},
	"util.IsServerlessMode": {Module: "signalwire.utils", Name: "is_serverless_mode"},

	// Livewire
	"livewire.FunctionTool": {Module: "signalwire.livewire", Name: "function_tool"},
	"livewire.RunApp":       {Module: "signalwire.livewire", Name: "run_app"},

	// Webhook signature validation — Python ships these as module-level free
	// functions in signalwire.core.security.webhook_validator. Go exposes
	// them as ValidateWebhookSignature / ValidateRequest in pkg/security.
	"security.ValidateWebhookSignature": {Module: "signalwire.core.security.webhook_validator", Name: "validate_webhook_signature"},
	"security.ValidateRequest":          {Module: "signalwire.core.security.webhook_validator", Name: "validate_request"},
}

// factoryInit maps a Go factory function (not a ``New<Struct>`` that matches
// its struct name) to the class whose ``__init__`` it satisfies.  Example:
// ``datamap.New`` constructs ``DataMap`` — lift it into the __init__ slot.
var FactoryInit = map[string]struct{ StructKey string }{
	"datamap.New": {StructKey: "datamap.DataMap"},
}


// eventTarget builds the standard relay event class target: the class is
// present, plus ``from_payload`` emitted synthetically when Go's factory
// ``New<Event>`` is present.
func eventTarget(cls string) ClassTarget {
	return ClassTarget{
		Module:           "signalwire.relay.event",
		Class:            cls,
		Methods:          map[string]string{},
		SyntheticMethods: []string{"from_payload"},
		Alias:            true,
	}
}
