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
				// RECONCILE: both helpers ARE present on the Go AgentBase
				// (pkg/agent/agent.go) — map to their Python-canonical names.
				"AutoMapSIPUsernames": "auto_map_sip_usernames",
				"GetFullURL":          "get_full_url",
			},
			// handle_request: the reference emits it as an AgentBase SURFACE
			// symbol (an override of SWMLService.handle_request) but records the
			// SIGNATURE only on the defining SWMLService class. Go likewise
			// defines AgentBase.HandleRequest (pkg/agent/agent.go) overriding the
			// promoted swml.Service.HandleRequest. Emit it as a surface-only
			// synthetic (SyntheticMethods is consumed by enumerate-surface only),
			// so SURFACE-DIFF sees AgentBase.handle_request while the signature is
			// carried once on SWMLService — matching the oracle exactly.
			SyntheticMethods: []string{"handle_request"},
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
				// RECONCILE: present on Go AgentBase (pkg/agent/agent.go).
				"RegisterRoutingCallback": "register_routing_callback",
				"SetupGracefulShutdown":   "setup_graceful_shutdown",
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

	// --- bedrock agent ----------------------------------------------------
	// IMPLEMENTED: pkg/agent/bedrock.go — AgentBase specialisation that renders
	// the amazon_bedrock verb + Bedrock inference setters, mirroring
	// signalwire.agents.bedrock.BedrockAgent.
	"agent.BedrockAgent": {{
		Module: "signalwire.agents.bedrock", Class: "BedrockAgent",
		Methods: map[string]string{
			"NewBedrockAgent":        "__init__",
			"SetVoice":               "set_voice",
			"SetInferenceParams":     "set_inference_params",
			"SetLLMModel":            "set_llm_model",
			"SetLLMTemperature":      "set_llm_temperature",
			"SetPromptLLMParams":     "set_prompt_llm_params",
			"SetPostPromptLLMParams": "set_post_prompt_llm_params",
			"String":                 "__repr__",
		},
	}},

	// --- web service ------------------------------------------------------
	// IMPLEMENTED: pkg/web/web_service.go — standalone static-file HTTP service,
	// mirroring signalwire.web.web_service.WebService (the reference surface
	// records only __init__/add_directory/remove_directory/start/stop; app +
	// security are Python @property accessors not surfaced).
	"web.WebService": {{
		Module: "signalwire.web.web_service", Class: "WebService",
		Methods: map[string]string{
			"NewWebService":   "__init__",
			"AddDirectory":    "add_directory",
			"RemoveDirectory": "remove_directory",
			"Start":           "start",
			"Stop":            "stop",
		},
	}},

	// --- config loader ----------------------------------------------------
	// IMPLEMENTED: pkg/agent/config_loader.go — JSON config with ${VAR|default}
	// env substitution, mirroring signalwire.core.config_loader.ConfigLoader.
	"agent.ConfigLoader": {{
		Module: "signalwire.core.config_loader", Class: "ConfigLoader",
		Methods: map[string]string{
			"NewConfigLoader": "__init__",
			"HasConfig":       "has_config",
			"GetConfigFile":   "get_config_file",
			"GetConfig":       "get_config",
			"SubstituteVars":  "substitute_vars",
			"Get":             "get",
			"GetSection":      "get_section",
			"MergeWithEnv":    "merge_with_env",
		},
	}},

	// --- server package ---------------------------------------------------
	"server.AgentServer": {{
		Module: "signalwire.agent_server", Class: "AgentServer",
		Methods: map[string]string{
			"NewAgentServer":                "__init__",
			"GetAgent":                      "get_agent",
			"GetAgents":                     "get_agents",
			"Register":                      "register",
			"Unregister":                    "unregister",
			"RegisterSIPUsername":           "register_sip_username",
			"Run":                           "run",
			"ServeStaticFiles":              "serve_static_files",
			"SetupSIPRouting":               "setup_sip_routing",
			"RegisterGlobalRoutingCallback": "register_global_routing_callback",
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
	// IMPLEMENTED: pkg/pom/pom_builder.go — fluent builder wrapping
	// PromptObjectModel, mirroring signalwire.core.pom_builder.PomBuilder.
	"pom.PomBuilder": {{
		Module: "signalwire.core.pom_builder", Class: "PomBuilder",
		Methods: map[string]string{
			"NewPomBuilder":  "__init__",
			"AddSection":     "add_section",
			"AddToSection":   "add_to_section",
			"AddSubsection":  "add_subsection",
			"HasSection":     "has_section",
			"GetSection":     "get_section",
			"RenderMarkdown": "render_markdown",
			"RenderXML":      "render_xml",
			"ToDict":         "to_dict",
			"ToJSON":         "to_json",
		},
		// from_sections is a Python classmethod; Go exposes pom.FromSections
		// (free function) — projected as a class member via the free-fn table.
		SyntheticMethods: []string{"from_sections"},
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
	"swml.Service": {
		{
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
				"HandleRequest":           "handle_request",
				"Render":                  "render_document",
				"Serve":                   "serve",
				// schema_utils accessor — exposes the SchemaUtils helper as a
				// public attribute, matching Python's ``self.schema_utils``.
				"SchemaUtils": "schema_utils",
				// RECONCILE: all present on the Go swml.Service (pkg/swml/service.go +
				// verb_handler.go) — map to the Python-canonical names.
				"AddSection":            "add_section",
				"AsRouter":              "as_router",
				"FullValidationEnabled": "full_validation_enabled",
				"ManualSetProxyURL":     "manual_set_proxy_url",
				"RegisterVerbHandler":   "register_verb_handler",
				"Stop":                  "stop",
			},
			// extract_sip_username is a Python @classmethod; Go exposes the equivalent
			// as the package-level swml.ExtractSIPUsername. Emit it as a class member
			// so the reference's SWMLService.extract_sip_username is PRESENT.
			SyntheticMethods: []string{"extract_sip_username"},
		},
		// RECONCILE: the Python SWMLBuilder (signalwire.core.swml_builder) is a
		// fluent SWML document builder; Go folds the identical builder surface
		// onto swml.Service (verbs are methods on Service). Project Service's verb
		// + build methods onto SWMLBuilder so the reference symbols are PRESENT.
		{
			Module: "signalwire.core.swml_builder", Class: "SWMLBuilder",
			Methods: map[string]string{
				"NewService":    "__init__",
				"AI":            "ai",
				"Answer":        "answer",
				"Hangup":        "hangup",
				"Play":          "play",
				"Say":           "say",
				"AddSection":    "add_section",
				"Render":        "render",
				"ResetDocument": "reset",
			},
			// build == render's alias (both return the document); GetDocument
			// serves the build role in Go.
			SyntheticMethods: []string{"build"},
		},
		// RECONCILE: the Python VerbHandlerRegistry (signalwire.core.swml_handler)
		// is realised in Go as an inline map on swml.Service with Register/Get/Has
		// accessors — project them onto the reference's registry class.
		{
			Module: "signalwire.core.swml_handler", Class: "VerbHandlerRegistry",
			Methods: map[string]string{
				"NewService":          "__init__",
				"RegisterVerbHandler": "register_handler",
				"GetVerbHandler":      "get_handler",
				"HasVerbHandler":      "has_handler",
			},
		},
	},

	// --- swml verb handlers (signalwire.core.swml_handler) ----------------
	// AIVerbHandler is a concrete Go struct (pkg/swml/ai_verb_handler.go);
	// SWMLVerbHandler is the reference's abstract base — Go models it as the
	// VerbHandler interface (pkg/swml/verb_handler.go) with the same contract.
	"swml.AIVerbHandler": {
		{
			Module: "signalwire.core.swml_handler", Class: "AIVerbHandler",
			Methods: map[string]string{
				"GetVerbName":    "get_verb_name",
				"ValidateConfig": "validate_config",
				"BuildConfig":    "build_config",
			},
		},
		// SWMLVerbHandler is the reference's abstract base (get_verb_name/
		// validate_config/build_config). Go models the base as the VerbHandler
		// interface (pkg/swml/verb_handler.go); the concrete AIVerbHandler
		// provides the contract, so project the base's method set here to keep the
		// reference symbol PRESENT (the interface itself is not a Go struct the
		// walker records).
		{
			Module: "signalwire.core.swml_handler", Class: "SWMLVerbHandler",
			Methods: map[string]string{
				"GetVerbName":    "get_verb_name",
				"ValidateConfig": "validate_config",
				"BuildConfig":    "build_config",
			},
		},
	},

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
			// RECONCILE: present on Go swml.SchemaUtils (pkg/swml/schema_utils.go).
			"FullValidationAvailable": "full_validation_available",
			"GenerateMethodSignature": "generate_method_signature",
			"GenerateMethodBody":      "generate_method_body",
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
		// create_payment_* are @staticmethods on Python's FunctionResult; Go
		// ships them as package-level helpers (swaig.CreatePaymentPrompt/Action/
		// Parameter, pkg/swaig/function_result.go) that build the same payment
		// config maps. RECONCILE-IN-EMIT: surface them under FunctionResult (the
		// reference's class-static placement) since the helpers exist.
		SyntheticMethods: []string{"create_payment_prompt", "create_payment_action", "create_payment_parameter"},
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
			// RECONCILE: the Go relay.Client exposes these directly (pkg/relay/
			// client.go) — they were wrongly documented as unexported. Map them
			// to their Python-canonical names so the symbols are PRESENT.
			"Connect":       "connect",
			"Execute":       "execute",
			"Receive":       "receive",
			"Unreceive":     "unreceive",
			"RelayProtocol": "relay_protocol",
		},
	}},
	// RECONCILE: Go ships a typed relay.RelayError (pkg/relay/error.go) — the
	// Python-reference RelayError exception. NewRelayError satisfies __init__.
	"relay.RelayError": {{
		Module: "signalwire.relay.client", Class: "RelayError",
		Methods: map[string]string{
			"NewRelayError": "__init__",
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
			"Pause":            "pause",
			"Resume":           "resume",
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
	// RequestOptions (plan 4.2): the per-request transport envelope. Go models
	// it as a value struct with public fields (Timeout/Retries/RetryOnStatus/
	// RetryBackoff/AbortSignal) plus a Merge method — the reference
	// RequestOptions with its merge(). Construction is a Go struct literal (no
	// NewRequestOptions factory), so __init__ + the abort_signal accessor are
	// signature-only idiom divergences (PORT_SIGNATURE_OMISSIONS.md); the SURFACE
	// oracle records only merge(), which this mapping projects.
	"rest.RequestOptions": {{
		Module: "signalwire.rest._request_options", Class: "RequestOptions",
		Methods: map[string]string{
			"Merge": "merge",
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
	// The one Go rest.SignalWireRestError struct projects onto BOTH Python REST
	// error classes. Python models a transport failure as a SUBCLASS
	// SignalWireRestTransportError(SignalWireRestError); Go folds it into this same
	// struct with a Transport bool discriminator (StatusCode 0 == status_code=None)
	// + the NewSignalWireRestTransportError constructor. Mapping the two
	// constructors onto the two classes' __init__ keeps the reference's
	// SignalWireRestTransportError comparing EQUAL (rename-not-omission) instead of
	// surfacing as a missing-port, while NewSignalWireRestTransportError stays a
	// PORT_ADDITION (Go's spelling of the transport error).
	"rest.SignalWireRestError": {
		{
			Module: "signalwire.rest._base", Class: "SignalWireRestError",
			Methods: map[string]string{
				"NewSignalWireRestError": "__init__",
			},
		},
		{
			Module: "signalwire.rest._base", Class: "SignalWireRestTransportError",
			Methods: map[string]string{
				"NewSignalWireRestTransportError": "__init__",
			},
		},
	},
	// ADAPTER (base-placement rename): the Go `namespaces.CrudResource` struct (the
	// single CRUD base every generated REST resource embeds — pkg/rest/namespaces/
	// common.go) provides all five CRUD verbs on one base, but the Python reference
	// SPLITS them across two bases — `ReadResource` carries get/list,
	// `CrudResource(ReadResource)` adds create/update/delete. To compare on the
	// reference's placement (rename-not-omission: keep the methods comparing, don't
	// blind-spot them), map the Go struct's Get/List onto `_base.ReadResource` and
	// Create/Update/Delete onto `_base.CrudResource`. This closes BOTH the
	// SURFACE-DIFF `CrudResource.get/list` addition AND the
	// `_base.ReadResource[.get/.list]` missing-port (surface + the two
	// ReadResource.get/list DRIFT items) in one mapping.
	"namespaces.CrudResource": {
		{
			Module: "signalwire.rest._base", Class: "ReadResource",
			Methods: map[string]string{
				"List":     "list",
				"Get":      "get",
				"Paginate": "paginate",
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
	// RECONCILE: Go's namespaces.CrudWithAddresses (pkg/rest/namespaces/common.go)
	// IS the Python _base.CrudWithAddresses mixin — CRUD (promoted via embedded
	// CrudResource) + ListAddresses. The FabricResource / FabricResourcePUT
	// reference bases are empty aliases of this same mixin, so they carry no
	// distinct surface (kept impossible-tagged in PORT_OMISSIONS).
	"namespaces.CrudWithAddresses": {{
		Module: "signalwire.rest._base", Class: "CrudWithAddresses",
		Methods: map[string]string{
			"ListAddresses": "list_addresses",
		},
	}},
	// namespaces.Paginator is the LIVE paginator — the value every resource's
	// Paginate() accessor returns. It represents Python's _pagination.
	// PaginatedIterator class surface (__init__/__next__/__iter__). The former
	// orphan rest.PaginatedIterator, which mapped here but no accessor returned,
	// was retired in plan 6.2-go; its adapter mapping moved to the live type so
	// the Python class stays represented (rename, not omission — DRIFT/SURFACE
	// stay 0).
	"namespaces.Paginator": {{
		Module: "signalwire.rest._pagination", Class: "PaginatedIterator",
		Methods: map[string]string{
			"NewPaginator": "__init__",
			"Next":         "__next__",
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
			"SetHistory":       "set_history",
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
			"SetHistory":           "set_history",
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
	// Go's SessionManager mirrors the full Python session-management surface:
	// the tool-token pair (create_tool_token/validate_tool_token), the underlying
	// session-token pair (generate_token/validate_token), token debugging, and
	// session lifecycle + metadata (pkg/security/session_manager.go).
	"security.SessionManager": {{
		Module: "signalwire.core.security.session_manager", Class: "SessionManager",
		Methods: map[string]string{
			"NewSessionManager":    "__init__",
			"CreateToken":          "create_tool_token",
			"ValidateToken":        "validate_tool_token",
			"GenerateToken":        "generate_token",
			"ValidateSessionToken": "validate_token",
			"CreateSession":        "create_session",
			"DebugToken":           "debug_token",
			"ActivateSession":      "activate_session",
			"EndSession":           "end_session",
			"GetSessionMetadata":   "get_session_metadata",
			"SetSessionMetadata":   "set_session_metadata",
		},
	}},

	// IMPLEMENTED: pkg/security/security_config.go — HTTP security settings
	// (SSL/hosts/CORS/headers/HSTS/basic-auth) from SWML_* env, mirroring
	// signalwire.core.security_config.SecurityConfig. get_ssl_context_kwargs
	// returns a primitive path-string dict ({ssl_certfile, ssl_keyfile}), which
	// Go exposes as GetSSLContextKwargs -> map[string]any (fed into crypto/tls
	// via swml.WithTLS).
	"security.SecurityConfig": {{
		Module: "signalwire.core.security_config", Class: "SecurityConfig",
		Methods: map[string]string{
			"NewSecurityConfig":   "__init__",
			"LoadFromEnv":         "load_from_env",
			"ValidateSSLConfig":   "validate_ssl_config",
			"GetBasicAuth":        "get_basic_auth",
			"GetSecurityHeaders":  "get_security_headers",
			"GetSSLContextKwargs": "get_ssl_context_kwargs",
			"ShouldAllowHost":     "should_allow_host",
			"GetCORSConfig":       "get_cors_config",
			"GetURLScheme":        "get_url_scheme",
			"LogConfig":           "log_config",
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
			// RECONCILE: present on Go BaseSkill (pkg/skills/skill_base.go).
			"GetSkillData":    "get_skill_data",
			"UpdateSkillData": "update_skill_data",
		},
		// register_tools + setup are the two abstract contract methods every
		// concrete skill implements (the SkillBase Go interface declares them);
		// the reference records them on SkillBase. Emit synthetically so the base
		// carries the contract. define_tool/validate_env_vars/validate_packages
		// have no BaseSkill equivalent (impossible-tagged in PORT_OMISSIONS).
		SyntheticMethods: []string{"__init__", "register_tools", "setup"},
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
	// RECONCILE: these livewire types ARE present in Go (pkg/livewire/livewire.go
	// + plugins.go) — surface them under their Python-canonical names.
	"livewire.ChatContext": {{
		Module: "signalwire.livewire", Class: "ChatContext",
		Methods: map[string]string{
			"NewChatContext": "__init__",
			"Append":         "append",
		},
	}},
	"livewire.ToolError": {{
		Module: "signalwire.livewire", Class: "ToolError",
		Methods: map[string]string{},
		Alias:   true,
	}},
	"livewire.InferenceLLM": {{
		Module: "signalwire.livewire", Class: "InferenceLLM",
		Methods: map[string]string{"NewInferenceLLM": "__init__"},
	}},
	"livewire.InferenceSTT": {{
		Module: "signalwire.livewire", Class: "InferenceSTT",
		Methods: map[string]string{"NewInferenceSTT": "__init__"},
	}},
	"livewire.InferenceTTS": {{
		Module: "signalwire.livewire", Class: "InferenceTTS",
		Methods: map[string]string{"NewInferenceTTS": "__init__"},
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

	// RequestOptions resolution helpers (plan 4.2). Python exposes resolve()
	// and status_is_retryable() as module-level functions of
	// signalwire.rest._request_options; Go exposes the same two as package-level
	// rest.Resolve / rest.StatusIsRetryable (rename-not-omission).
	"rest.Resolve":           {Module: "signalwire.rest._request_options", Name: "resolve"},
	"rest.StatusIsRetryable": {Module: "signalwire.rest._request_options", Name: "status_is_retryable"},

	// Typed-handler schema inference. Python's signalwire.core.agent.tools.
	// type_inference reflects a handler's signature at runtime; Go builds the
	// same JSON-Schema from the typed Params declaration (pkg/swaig/type_inference.go).
	// Projected onto the reference module-level free functions.
	"swaig.InferSchema":               {Module: "signalwire.core.agent.tools.type_inference", Name: "infer_schema"},
	"swaig.CreateTypedHandlerWrapper": {Module: "signalwire.core.agent.tools.type_inference", Name: "create_typed_handler_wrapper"},

	// config_loader.find_config_file is a Python @staticmethod; Go exposes it as
	// the package-level agent.FindConfigFile. The surface diff records it under
	// the ConfigLoader class (staticmethod placement) via the free-fn projection.
	"agent.FindConfigFile": {Module: "signalwire.core.config_loader", Name: "ConfigLoader.find_config_file"},

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

	// Decomposed webhook-validation core — the framework-free decision unit
	// signalwire.core.security.webhook_middleware.validate(method,url,headers,
	// body) -> optional<(status,headers,body)>. Go exposes it as
	// security.Validate returning *WebhookRejection (nil = pass, a
	// {Status,Headers,Body} triple = reject); the *WebhookRejection type is
	// aliased to the canonical tuple<int,dict<string,string>,string> in
	// type_aliases.yaml, so the pointer enumerates to the oracle's
	// optional<tuple<int,dict<string,string>,string>> return. The http.Handler
	// WebhookMiddleware STAYS a PORT_ADDITION framework-wrapper idiom over this
	// core; only the decomposed decision core is the required cross-port symbol.
	"security.Validate": {Module: "signalwire.core.security.webhook_middleware", Name: "validate"},

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

// SkillContract records one Go built-in skill's Python-canonical class surface.
// Each built-in skill in pkg/skills/builtin/*.go embeds skills.BaseSkill and
// overrides a subset of the SkillBase contract; the remaining contract methods
// are PROMOTED from the embedded BaseSkill, so the concrete skill struct
// genuinely PROVIDES every method the Python reference records for it (declared
// override or inherited default). Rather than blind-spot the whole batch under a
// PORT_OMISSIONS excuse ("Go ships via *Skill structs"), we RECONCILE-IN-EMIT:
// project each Go skill struct onto its Python-canonical
// `signalwire.skills.<name>.skill.<Class>` module+class with the reference's
// exact method set, so the symbols are PRESENT and compare EQUAL.
//
// Method names are the Python-canonical snake_case (the reference's own leaves);
// the Go members they correspond to are RegisterTools/GetHints/Setup/Cleanup/
// GetParameterSchema/GetInstanceKey/GetGlobalData/GetPromptSections (mapped via
// the standard goNameToSnake fold: register_tools←RegisterTools, etc.).
// `Synthetic` names are Python contract methods Go expresses differently but
// equivalently: `__init__` (Go `New<Skill>` factory), `get_tools` (Go returns
// the tool list via RegisterTools), `search_wiki` (Go registers it as a tool
// handler). ClassName is the REFERENCE class casing (e.g. `ApiNinjasTriviaSkill`,
// `WeatherApiSkill`), which differs from the Go struct's initialism casing.
type SkillContract struct {
	// GoStruct is the short `<pkg>.<Struct>` key as it appears in the walk
	// (all built-in skills are package `builtin`, except spider which is its
	// own sub-package `spider`). Used to verify the struct is present (fail
	// loud on a renamed/removed skill).
	GoStruct string
	// Module is the Python-canonical per-skill module.
	Module string
	// ClassName is the Python-reference class name (reference casing).
	ClassName string
	// Methods are the contract method leaves that map 1:1 from a Go member via
	// goNameToSnake (register_tools, get_hints, setup, cleanup,
	// get_parameter_schema, get_instance_key, get_global_data,
	// get_prompt_sections). Each is verified present on the struct (declared or
	// promoted from BaseSkill).
	Methods []string
	// Synthetic are Python contract methods Go expresses via a factory / tool
	// registration (__init__, get_tools, search_wiki) — emitted unconditionally.
	Synthetic []string
}

// SkillContractTable is the per-built-in-skill projection consumed by BOTH
// cmd/enumerate-surface and cmd/enumerate-signatures (kept in lockstep). The
// method sets are the Python reference's own per-skill surface (each skill
// records a DIFFERENT subset — see signalwire-python/signalwire/skills/<n>/skill.py).
// mcp_gateway is intentionally absent: the Python reference does not surface a
// signalwire.skills.mcp_gateway.skill module (Go ships the skill as a port
// extension, recorded in PORT_ADDITIONS).
var SkillContractTable = []SkillContract{
	{GoStruct: "builtin.APINinjasTriviaSkill", Module: "signalwire.skills.api_ninjas_trivia.skill", ClassName: "ApiNinjasTriviaSkill",
		Methods:   []string{"get_instance_key", "get_parameter_schema", "register_tools", "setup"},
		Synthetic: []string{"__init__", "get_tools"}},
	{GoStruct: "builtin.ClaudeSkillsSkill", Module: "signalwire.skills.claude_skills.skill", ClassName: "ClaudeSkillsSkill",
		Methods: []string{"get_hints", "get_instance_key", "get_parameter_schema", "register_tools", "setup"}},
	{GoStruct: "builtin.DataSphereSkill", Module: "signalwire.skills.datasphere.skill", ClassName: "DataSphereSkill",
		Methods: []string{"cleanup", "get_global_data", "get_hints", "get_instance_key", "get_parameter_schema", "get_prompt_sections", "register_tools", "setup"}},
	{GoStruct: "builtin.DataSphereServerlessSkill", Module: "signalwire.skills.datasphere_serverless.skill", ClassName: "DataSphereServerlessSkill",
		Methods: []string{"get_global_data", "get_hints", "get_instance_key", "get_parameter_schema", "get_prompt_sections", "register_tools", "setup"}},
	{GoStruct: "builtin.DateTimeSkill", Module: "signalwire.skills.datetime.skill", ClassName: "DateTimeSkill",
		Methods: []string{"get_hints", "get_parameter_schema", "get_prompt_sections", "register_tools", "setup"}},
	{GoStruct: "builtin.GoogleMapsSkill", Module: "signalwire.skills.google_maps.skill", ClassName: "GoogleMapsSkill",
		Methods: []string{"get_hints", "get_parameter_schema", "get_prompt_sections", "register_tools", "setup"}},
	{GoStruct: "builtin.InfoGathererSkill", Module: "signalwire.skills.info_gatherer.skill", ClassName: "InfoGathererSkill",
		Methods: []string{"get_global_data", "get_instance_key", "get_parameter_schema", "register_tools", "setup"}},
	{GoStruct: "builtin.JokeSkill", Module: "signalwire.skills.joke.skill", ClassName: "JokeSkill",
		Methods: []string{"get_global_data", "get_hints", "get_parameter_schema", "get_prompt_sections", "register_tools", "setup"}},
	{GoStruct: "builtin.MathSkill", Module: "signalwire.skills.math.skill", ClassName: "MathSkill",
		Methods: []string{"get_hints", "get_parameter_schema", "get_prompt_sections", "register_tools", "setup"}},
	{GoStruct: "builtin.NativeVectorSearchSkill", Module: "signalwire.skills.native_vector_search.skill", ClassName: "NativeVectorSearchSkill",
		Methods: []string{"cleanup", "get_global_data", "get_hints", "get_instance_key", "get_parameter_schema", "get_prompt_sections", "register_tools", "setup"}},
	{GoStruct: "builtin.PlayBackgroundFileSkill", Module: "signalwire.skills.play_background_file.skill", ClassName: "PlayBackgroundFileSkill",
		Methods:   []string{"get_instance_key", "get_parameter_schema", "register_tools", "setup"},
		Synthetic: []string{"__init__", "get_tools"}},
	{GoStruct: "spider.SpiderSkill", Module: "signalwire.skills.spider.skill", ClassName: "SpiderSkill",
		Methods:   []string{"cleanup", "get_hints", "get_instance_key", "get_parameter_schema", "register_tools", "setup"},
		Synthetic: []string{"__init__"}},
	{GoStruct: "builtin.SWMLTransferSkill", Module: "signalwire.skills.swml_transfer.skill", ClassName: "SWMLTransferSkill",
		Methods: []string{"get_hints", "get_instance_key", "get_parameter_schema", "get_prompt_sections", "register_tools", "setup"}},
	{GoStruct: "builtin.WeatherAPISkill", Module: "signalwire.skills.weather_api.skill", ClassName: "WeatherApiSkill",
		Methods:   []string{"get_parameter_schema", "register_tools", "setup"},
		Synthetic: []string{"__init__", "get_tools"}},
	{GoStruct: "builtin.WebSearchSkill", Module: "signalwire.skills.web_search.skill", ClassName: "WebSearchSkill",
		Methods: []string{"get_global_data", "get_hints", "get_instance_key", "get_parameter_schema", "get_prompt_sections", "register_tools", "setup"}},
	{GoStruct: "builtin.WikipediaSearchSkill", Module: "signalwire.skills.wikipedia_search.skill", ClassName: "WikipediaSearchSkill",
		Methods:   []string{"get_hints", "get_parameter_schema", "get_prompt_sections", "register_tools", "setup"},
		Synthetic: []string{"search_wiki"}},
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
