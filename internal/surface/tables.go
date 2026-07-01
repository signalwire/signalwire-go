// Package surface holds the lookup tables that map Go SDK types and
// functions to their Python-canonical reference names. Both
// cmd/enumerate-surface (names-only) and cmd/enumerate-signatures
// (full signatures) consume these tables.
package surface

import (
	"fmt"
	"maps"
)

// init folds the generated REST StructTable entries (GeneratedRESTStructTable,
// in struct_table_generated.go — same package surface, so no import cycle) into
// the master StructTable. The generated slice owns every per-resource REST entry
// (namespaces.AddressesNamespace, namespaces.AIAgents, …); the hand entries above
// keep only what the generator does not emit (namespaces.Resource + the six
// namespace containers). A key collision between the two is a real bug (a hand
// entry the generator now also owns) — fail loud rather than silently overwrite.
func init() {
	for k := range GeneratedRESTStructTable {
		if _, dup := StructTable[k]; dup {
			panic(fmt.Sprintf("surface: StructTable key %q defined both by hand and by GeneratedRESTStructTable — remove the hand entry", k))
		}
	}
	maps.Copy(StructTable, GeneratedRESTStructTable)
}

// ClassTarget is one Python class destination for a Go struct's methods.
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

// StructTable maps a Go “<shortPkg>.<StructName>“ to one or more Python
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
				"EnableSIPRouting":      "enable_sip_routing",
				"OnDebugEvent":          "on_debug_event",
				"OnSummary":             "on_summary",
				"RegisterSIPUsername":   "register_sip_username",
				"SetPostPromptURL":      "set_post_prompt_url",
				"SetWebHookURL":         "set_web_hook_url",
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
				"PostPrompt":          "get_post_prompt",
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
				"PostPrompt":          "get_post_prompt",
				"RawPrompt":           "get_raw_prompt",
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
				"Function":              "get_function",
				"AllFunctions":          "get_all_functions",
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
				"SetLanguageParams":      "set_language_params",
				"LanguageParams":         "get_language_params",
				"SetLanguages":           "set_languages",
				"SetMultilingual":        "set_multilingual",
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
				"ManualSetProxyURL":        "manual_set_proxy_url",
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
			"RegisterSIPUsername": "register_sip_username",
			"Run":                 "run",
			"ServeStaticFiles":    "serve_static_files",
			"SetupSIPRouting":     "setup_sip_routing",
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
			"NewSchemaUtils":            "__init__",
			"LoadSchema":                "load_schema",
			"GetAllVerbNames":           "get_all_verb_names",
			"GetVerbProperties":         "get_verb_properties",
			"GetVerbRequiredProperties": "get_verb_required_properties",
			"GetVerbParameters":         "get_verb_parameters",
			"ValidateVerb":              "validate_verb",
			"ValidateDocument":          "validate_document",
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
			"SIPRefer":                 "sip_refer",
			"Tap":                      "tap",
			"StopTap":                  "stop_tap",
			"ExecuteSwml":              "execute_swml",
			"ExecuteRPC":               "execute_rpc",
			"RPCAiMessage":             "rpc_ai_message",
			"RPCAiUnhold":              "rpc_ai_unhold",
			"RPCDial":                  "rpc_dial",
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
			"Answer":                 "answer",
			"Hangup":                 "hangup",
			"Pass":                   "pass_",
			"Transfer":               "transfer",
			"Play":                   "play",
			"PlayTTS":                "play_tts",
			"PlayAudio":              "play_audio",
			"PlaySilence":            "play_silence",
			"PlayRingtone":           "play_ringtone",
			"PlayAndCollect":         "play_and_collect",
			"PromptTTS":              "prompt_tts",
			"PromptAudio":            "prompt_audio",
			"Collect":                "collect",
			"DetectDigit":            "detect_digit",
			"DetectAnsweringMachine": "detect_answering_machine",
			"DetectFax":              "detect_fax",
			"WaitForAnswered":        "wait_for_answered",
			"WaitForRinging":         "wait_for_ringing",
			"WaitForEnding":          "wait_for_ending",
			"Record":                 "record",
			"Connect":                "connect",
			"Disconnect":             "disconnect",
			"SendDigits":             "send_digits",
			"Detect":                 "detect",
			"SendFax":                "send_fax",
			"ReceiveFax":             "receive_fax",
			"Tap":                    "tap",
			"Stream":                 "stream",
			"JoinConference":         "join_conference",
			"LeaveConference":        "leave_conference",
			"AI":                     "ai",
			"AmazonBedrock":          "amazon_bedrock",
			"AIMessage":              "ai_message",
			"AIHold":                 "ai_hold",
			"AIUnhold":               "ai_unhold",
			"Hold":                   "hold",
			"Unhold":                 "unhold",
			"Denoise":                "denoise",
			"DenoiseStop":            "denoise_stop",
			"JoinRoom":               "join_room",
			"LeaveRoom":              "leave_room",
			"QueueEnter":             "queue_enter",
			"QueueLeave":             "queue_leave",
			"BindDigit":              "bind_digit",
			"ClearDigitBindings":     "clear_digit_bindings",
			"UserEvent":              "user_event",
			"Echo":                   "echo",
			"Pay":                    "pay",
			"Transcribe":             "transcribe",
			"LiveTranscribe":         "live_transcribe",
			"LiveTranslate":          "live_translate",
			"Refer":                  "refer",
			"On":                     "on",
			"WaitFor":                "wait_for",
			"WaitForEnded":           "wait_for_ended",
			"String":                 "__repr__",
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
	"rest.HTTPClient": {{
		Module: "signalwire.rest._base", Class: "HttpClient",
		Methods: map[string]string{
			"NewHTTPClient": "__init__",
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
	// ADAPTER (base-placement rename): the Go `CrudResource` struct provides all
	// five CRUD verbs on one base, but the Python reference SPLITS them across two
	// bases — `ReadResource` carries get/list, `CrudResource(ReadResource)` adds
	// create/update/delete. To compare on the reference's placement (rename-not-
	// omission: keep the methods comparing, don't blind-spot them), map the Go
	// struct's Get/List onto `_base.ReadResource` and Create/Update/Delete onto
	// `_base.CrudResource`. This closes BOTH the SURFACE-DIFF `CrudResource.get/list`
	// addition AND the `_base.ReadResource[.get/.list]` missing-port (surface + the
	// two ReadResource.get/list DRIFT items) in one mapping.
	"rest.CrudResource": {
		{
			Module: "signalwire.rest._base", Class: "ReadResource",
			Methods: map[string]string{
				"List": "list",
				"Get":  "get",
			},
		},
		{
			Module: "signalwire.rest._base", Class: "CrudResource",
			Methods: map[string]string{
				"Create": "create",
				"Update": "update",
				"Delete": "delete",
			},
		},
	},
	"rest.PaginatedIterator": {{
		Module: "signalwire.rest._pagination", Class: "PaginatedIterator",
		Methods: map[string]string{
			"NewPaginatedIterator": "__init__",
			"Next":                 "__next__",
		},
		SyntheticMethods: []string{"__iter__"},
	}},

	// --- REST namespaces (adopted from the generated surface) --------------
	//
	// The per-resource StructTable entries (namespaces.AddressesNamespace,
	// namespaces.AIAgents, namespaces.FabricTokens, …) are GENERATED into
	// internal/surface/struct_table_generated.go (var GeneratedRESTStructTable)
	// from the x-sdk-* markup and merged into StructTable by the init() at the
	// bottom of this file. Only the entries the generator does NOT emit are kept
	// by hand below: the shared Resource base (Python BaseResource) and the six
	// namespace containers, which the generator's resource pass does not produce
	// but the oracle records in signalwire.rest.namespaces._client_tree_generated
	// (their resource-accessor fields auto-project as snake_case accessors).

	// rest/namespaces/common.go (Resource struct = Python's BaseResource)
	"namespaces.Resource": {{
		Module: "signalwire.rest._base", Class: "BaseResource",
		Methods:          map[string]string{},
		SyntheticMethods: []string{"__init__"},
	}},

	// Namespace containers (client_tree_generated.go). The oracle records these
	// in the _client_tree_generated module; each container's resource fields
	// (AIAgents, CallFlows, Documents, …) auto-project as snake_case accessor
	// methods, matching Python's __getattr__/property accessors.
	"namespaces.DatasphereNamespace": {{
		Module: "signalwire.rest.namespaces._client_tree_generated", Class: "DatasphereNamespace",
		Methods: map[string]string{"NewDatasphereNamespace": "__init__"},
	}},
	"namespaces.FabricNamespace": {{
		Module: "signalwire.rest.namespaces._client_tree_generated", Class: "FabricNamespace",
		Methods: map[string]string{"NewFabricNamespace": "__init__"},
	}},
	"namespaces.VideoNamespace": {{
		Module: "signalwire.rest.namespaces._client_tree_generated", Class: "VideoNamespace",
		Methods: map[string]string{"NewVideoNamespace": "__init__"},
	}},
	"namespaces.ProjectNamespace": {{
		Module: "signalwire.rest.namespaces._client_tree_generated", Class: "ProjectNamespace",
		Methods: map[string]string{"NewProjectNamespace": "__init__"},
	}},
	"namespaces.LogsNamespace": {{
		Module: "signalwire.rest.namespaces._client_tree_generated", Class: "LogsNamespace",
		Methods: map[string]string{"NewLogsNamespace": "__init__"},
	}},
	"namespaces.RegistryNamespace": {{
		Module: "signalwire.rest.namespaces._client_tree_generated", Class: "RegistryNamespace",
		Methods: map[string]string{"NewRegistryNamespace": "__init__"},
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
			"SetText":              "set_text",
			"AddSection":           "add_section",
			"AddBullets":           "add_bullets",
			"SetStepCriteria":      "set_step_criteria",
			"SetFunctions":         "set_functions",
			"SetValidSteps":        "set_valid_steps",
			"SetValidContexts":     "set_valid_contexts",
			"SetEnd":               "set_end",
			"SetSkipUserTurn":      "set_skip_user_turn",
			"SetSkipToNextStep":    "set_skip_to_next_step",
			"SetGatherInfo":        "set_gather_info",
			"AddGatherQuestion":    "add_gather_question",
			"ClearSections":        "clear_sections",
			"SetResetSystemPrompt": "set_reset_system_prompt",
			"SetResetUserPrompt":   "set_reset_user_prompt",
			"SetResetConsolidate":  "set_reset_consolidate",
			"SetResetFullReset":    "set_reset_full_reset",
			"ToMap":                "to_dict",
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

// FreeFnTable maps a Go “<shortPkg>.<FuncName>“ to a Python module-level
// function.  Only used for genuinely free-standing functions — factory
// constructors (“New<Struct>“) that should become “__init__“ are
// declared via “factoryInit“ instead.
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
	"contexts.CreateSimpleContext": {Module: "signalwire.core.contexts", Name: "create_simple_context"},
	"datamap.CreateSimpleAPITool":  {Module: "signalwire.core.data_map", Name: "create_simple_api_tool"},
	"datamap.CreateExpressionTool": {Module: "signalwire.core.data_map", Name: "create_expression_tool"},
	"relay.ParseEvent":             {Module: "signalwire.relay.event", Name: "parse_event"},

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

	// Standalone security hygiene utilities — Python ships these as
	// module-level free functions in signalwire.core.security.security_utils.
	// Go exposes them as package-level functions in pkg/security
	// (FilterSensitiveHeaders / RedactURL / IsValidHostname); the PascalCase +
	// all-caps URL initialism is the Go naming idiom, mapped here to the
	// snake_case canonical names the DRIFT gate compares.
	"security.FilterSensitiveHeaders": {Module: "signalwire.core.security.security_utils", Name: "filter_sensitive_headers"},
	"security.RedactURL":              {Module: "signalwire.core.security.security_utils", Name: "redact_url"},
	"security.IsValidHostname":        {Module: "signalwire.core.security.security_utils", Name: "is_valid_hostname"},
}

// FactoryInit maps a Go factory function (not a “New<Struct>“ that matches
// its struct name) to the class whose “__init__“ it satisfies.  Example:
// “datamap.New“ constructs “DataMap“ — lift it into the __init__ slot.
var FactoryInit = map[string]struct{ StructKey string }{
	"datamap.New": {StructKey: "datamap.DataMap"},
}

// eventTarget builds the standard relay event class target: the class is
// present, plus “from_payload“ emitted synthetically when Go's factory
// “New<Event>“ is present.
func eventTarget(cls string) ClassTarget {
	return ClassTarget{
		Module:           "signalwire.relay.event",
		Class:            cls,
		Methods:          map[string]string{},
		SyntheticMethods: []string{"from_payload"},
		Alias:            true,
	}
}
