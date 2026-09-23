<!-- ══════════════════════════════════════════════════════════════════════════
BEFORE YOU ADD AN ENTRY TO THIS FILE — READ THIS.

Every entry here is a place the parity checker STOPS comparing. That is a real cost:
a divergence you list is a divergence no gate will ever catch again. So entries must
be RARE, and each one must earn its place. Default to skepticism: assume the entry is
NOT needed and make the case that it is.

The order of preference, always:
  1. FIX THE PORT so it matches the reference (add the missing member; make the
     signature match).
  2. FIX THE EMISSION so idiom folds onto the reference shape — the enumerator/emitter
     canonicalizes your language's spelling onto the oracle's (builder → __init__,
     getters → attributes, Result<T,E> → the plain return, CamelCase → the reference
     name, options-object/kwargs → the expanded param list, RAII/dispose → close).
     MOST divergences are idiom and belong here, not in this file.
  3. FIX THE REFERENCE if the oracle itself is wrong or stale (a Python-only symbol
     that leaked into the contract, a param the reference added and the oracle never
     re-enumerated). Fix Python / the oracle, then re-drift — do not paper over a
     broken reference with a per-port entry.
  4. Only when 1–3 genuinely cannot apply does an entry here become justified.

An entry is JUSTIFIED ONLY IF it is irreducible after correct emission — i.e. the
divergence survives because the two languages genuinely cannot express the same thing,
not because the emitter hasn't folded the idiom yet. If emission COULD fold it, the
entry is a bug in this file; go fix the emitter.

Each entry MUST state WHY, concretely, in one of these forms:
  • ADDITION — this symbol exists in the port but not the reference. Answer: is it
    genuine port-only surface with NO reference twin (say what it is and why the
    reference has no equivalent), or is it IDIOM the emitter should have folded (then
    it does not belong here — fold it)? A convenience/alias/back-compat wrapper is NOT
    a justification.
  • OMISSION — this reference symbol has no port member. Answer: WHY can it not exist
    here — what specific language feature is absent (e.g. no async-context-manager
    protocol, no __init__ method protocol)? "impossible:" means the construct cannot
    be expressed at all; if it merely LOOKS different, that's idiom → fold it, don't
    omit it. Cite a precedent when one exists (e.g. RelayClient omits the same dunder).
  • SIGNATURE — the symbol matches by name but its parameters differ. Answer: is the
    difference a foldable idiom collapse (options-object, leading context/self,
    builder) — then EXPAND it in the signature emitter so names+count match, don't list
    it — or a genuine reference-only parameter with no cross-language analogue?

If you cannot write a crisp, specific WHY that survives the "could emission fold this?"
test, the entry is not ready. Prove it's needed before you add it.
═══════════════════════════════════════════════════════════════════════════════ -->

# PORT_OMISSIONS.md
#
# Every symbol listed here is a public Python-reference API member that the
# Go port deliberately does not implement.  The format is one
# `<fully.qualified.symbol>: <rationale>` per line, as expected by
# porting-sdk/scripts/diff_port_surface.py.  Section headers (lines that
# begin with `#`) are ignored by the parser.
#
# Rationale conventions:
#   * "not_yet_implemented: <reason>" = tracked gap, future PR will add it.
#   * anything else = deliberate omission (subsystem skipped, Python-only
#     implementation detail, or port-specific architectural difference).

# --- Search subsystem ---

# --- CLI tooling ---

# --- MCP gateway service ---

# --- POM builder ---
# signalwire.pom.pom.PromptObjectModel and signalwire.pom.pom.Section are
# now implemented natively in Go as pom.PromptObjectModel and pom.Section
# (pkg/pom/pom.go).  Tests in pkg/pom/pom_test.go assert exact-string
# parity with the Python renderer; signalwire-python parity tests live in
# tests/unit/pom/test_pom_render_parity.py.
#
# pom_tool is a Python-only CLI that wraps the POM module — kept omitted
# because Go ships a library, not a CLI.

# --- Utils / web / auth helpers ---
signalwire.core.auth_handler.AuthHandler.verify_basic_auth: impossible: the reference's sole parameter is FastAPI's HTTPBasicCredentials object; Go has no framework credentials type and will not invent one. security.AuthHandler DOES verify basic credentials (VerifyBasicAuth(*http.Request) / VerifyBasicAuthPair(user, pass)) — the capability is present, only the framework-object parameter shape cannot be expressed
signalwire.core.auth_handler.AuthHandler.flask_decorator: impossible: Flask-specific decorator; no Go equivalent (Go uses net/http middleware)
signalwire.core.auth_handler.AuthHandler.get_auth_info: impossible: Python auth-helper accessor; Go folds auth state into middleware, no standalone class
signalwire.core.auth_handler.AuthHandler.get_fastapi_dependency: impossible: FastAPI-specific dependency factory; no Go equivalent (Go uses net/http middleware)
signalwire.core.auth_handler.AuthHandler.verify_api_key: impossible: Python auth-helper method; Go verifies API keys inside withAuth middleware, no standalone class
signalwire.core.auth_handler.AuthHandler.verify_bearer_token: impossible: Python auth-helper method; Go verifies bearer tokens inside withAuth middleware, no standalone class
signalwire.core.logging_config.configure_logging: impossible: wraps the Python logging library; Go uses pkg/logging (structured) with equivalent behaviour — no logging-lib configuration surface

# --- Bedrock prefab agent ---


# --- Skill registry plumbing ---
signalwire.skills.registry.SkillRegistry.discover_skills: approved: 2026-07 user sign-off — Go registers skills at compile time via package-level RegisterSkill/ListSkills/GetSkillFactory; no runtime instance-method discovery
signalwire.skills.registry.SkillRegistry.get_all_skills_schema: approved: 2026-07 user sign-off — Go uses package-level skill registration; no instance schema-aggregation method
signalwire.skills.registry.SkillRegistry.get_skill_class: approved: 2026-07 user sign-off — Go uses package-level GetSkillFactory; no instance class lookup
signalwire.skills.registry.SkillRegistry.list_all_skill_sources: approved: 2026-07 user sign-off — Go registers skills at compile time; no runtime source enumeration
signalwire.skills.registry.SkillRegistry.list_skills: approved: 2026-07 user sign-off — Go uses the package-level ListSkills free function
signalwire.skills.registry.SkillRegistry.register_skill: approved: 2026-07 user sign-off — Go uses the package-level RegisterSkill free function

# --- Core mixins not split into Go ---
signalwire.core.mixins.mcp_server_mixin.MCPServerMixin: approved: 2026-07 user sign-off — MCP-server mixin is a Python marker class (no public methods); Go inlines MCP into AgentBase (AddMcpServer/EnableMcpServer)
signalwire.core.mixins.serverless_mixin.ServerlessMixin: approved: 2026-07 user sign-off — Python serverless mixin (Lambda detection + request handling); Go delegates serverless to platform adapters (pkg/lambda Handler), not an in-process AgentBase mixin
agentbase-family.handle_serverless_request: impossible: Python couples serverless request handling into the mixin; Go delegates to platform adapters (pkg/lambda) — no in-process AgentBase equivalent
agentbase-family.tool: impossible: Python @tool decorator relies on the decorator protocol; Go uses AgentBase.DefineTool(ToolDefinition{...})
agentbase-family.get_app: impossible: Python's WebMixin.get_app returns the FastAPI app object; Go has no framework app handle (AsRouter returns http.Handler)

# --- Core agent internal submodules ---
signalwire.core.agent.prompt.manager.PromptManager.__init__: impossible: Python internal submodule constructor; Go consolidates PromptManager into AgentBase (no separately-constructed manager)
signalwire.core.agent.tools.decorator.ToolDecorator: impossible: Python decorator-factory class relies on the decorator protocol; Go uses DefineTool struct-literals
signalwire.core.agent.tools.decorator.ToolDecorator.create_class_decorator: impossible: Python decorator-factory relies on the decorator protocol; Go uses DefineTool struct-literals
signalwire.core.agent.tools.decorator.ToolDecorator.create_instance_decorator: impossible: Python decorator-factory relies on the decorator protocol; Go uses DefineTool struct-literals
signalwire.core.agent.tools.registry.ToolRegistry.__init__: impossible: Python internal submodule constructor; Go consolidates the tool registry into AgentBase (no separately-constructed registry)
signalwire.core.agent.tools.registry.ToolRegistry.register_class_decorated_tools: impossible: registers @tool-decorated class methods discovered via the decorator protocol; Go has no decorator-discovery equivalent
signalwire.core.skill_base.SkillBase.define_tool: impossible: Python skill tool registration uses a decorator; Go uses BaseSkill.RegisterTools returning []ToolRegistration
signalwire.core.skill_base.SkillBase.validate_env_vars: impossible: Python validates required env vars via runtime introspection of a declared list; Go skills read os.Getenv directly in Setup (RequiredEnvVars declares the list)
signalwire.core.skill_base.SkillBase.validate_packages: impossible: Python validate_packages checks pip dependencies at runtime; Go dependencies are resolved at build time — no runtime package check

# --- Core SWML / SWAIG / function_result internals ---
signalwire.core.swaig_function.SWAIGFunction.__call__: impossible: Python callable protocol (__call__) has no Go equivalent
signalwire.core.swaig_function.SWAIGFunction.__init__: impossible: constructor of the callable wrapper class Go does not model
signalwire.core.swaig_function.SWAIGFunction.execute: impossible: Python SWAIGFunction.execute invokes the wrapped callable; Go invokes via swaig.ToolHandler func values
signalwire.core.swaig_function.SWAIGFunction.to_swaig: impossible: serialises the callable-wrapper to a SWAIG entry; Go builds SWAIG entries from ToolDefinition directly
signalwire.core.swml_builder.SWMLBuilder.__getattr__: impossible: Python dynamic attribute dispatch (__getattr__) has no Go equivalent; verbs are explicit methods on swml.Service
signalwire.core.swml_renderer.SwmlRenderer: impossible: Python SwmlRenderer is a stateless render helper; Go folds rendering into swml.Service.Render / swaig.FunctionResult — no separate renderer type
signalwire.core.swml_renderer.SwmlRenderer.render_function_response_swml: impossible: Go builds function-response SWML via swaig.FunctionResult — no separate renderer
signalwire.core.swml_renderer.SwmlRenderer.render_swml: impossible: Go folds SWML rendering into swml.Service.Render — no separate renderer
signalwire.core.swml_service.SWMLService.__getattr__: impossible: Python dynamic attribute dispatch (__getattr__) has no Go equivalent; verbs are explicit methods on swml.Service

# --- Relay Call / Client / Message ---
signalwire.relay.client.RelayClient.__aenter__: impossible: Python async context-manager protocol (__aenter__) has no Go equivalent; Go uses explicit Connect()/Stop()
signalwire.relay.client.RelayClient.__aexit__: impossible: Python async context-manager protocol (__aexit__) has no Go equivalent; Go uses explicit Connect()/Stop()
signalwire.relay.client.RelayClient.__del__: impossible: Python __del__ finalizer has no Go equivalent; Go GC + Stop() release the WebSocket
signalwire.relay.message.Message.__repr__: impossible: Python __repr__ object-protocol method has no Go analog (Stringer not surfaced as a reference method)

# --- AI-Chat async context-manager protocol (signalwire.ai_chat.client) ---
signalwire.ai_chat.client.AIChatClient.__aenter__: impossible: Python async context-manager protocol (__aenter__) has no Go equivalent; the Go aichat.Client wraps a stateless, connection-pooled *http.Client (nothing to enter) and the TS OO cousin omits it identically
signalwire.ai_chat.client.AIChatClient.__aexit__: impossible: Python async context-manager protocol (__aexit__) has no Go equivalent; the Go aichat.Client has no owned session to tear down on exit, mirroring RelayClient.__aexit__

# --- REST namespace omissions ---

# --- Prefab internal handlers ---


# --- Misc not-yet-implemented items ---

# --- Idiom: Python class accessors that Go folds into private fields or package-level helpers ---
signalwire.agent_server.AgentServer.agents: impossible: Python exposes ``agents`` as a public dict attribute; Go keeps the map private (``agents map[string]*agent.AgentBase``) and exposes it via the ``GetAgents()`` accessor (idiomatic Go private-field + renamed accessor — the ``agents`` member name genuinely cannot exist)
agentbase-family.skill_manager: impossible: Python exposes ``self.skill_manager`` (a SkillManager composition handle); Go folds the SkillManager into a private ``skillManager`` field and surfaces the user-facing methods (AddSkill, RemoveSkill, ListSkills, HasSkill) directly on AgentBase — no public composition-handle accessor
signalwire.core.skill_manager.SkillManager.loaded_skills: impossible: Python exposes ``loaded_skills`` as a public dict attribute; Go keeps the map private (``loadedSkills map[string]SkillBase``) and exposes it via the ``ListLoadedSkills()`` accessor (private-field + renamed accessor)
signalwire.core.swml_service.SWMLService.security: impossible: Python exposes a ``security`` property returning a SecurityConfig composition handle; Go folds auth state into private fields on Service (basicAuthUser, bearerToken, apiKey, ...) configured via WithSecurityConfig/WithBasicAuth/WithBearerToken/WithAPIKey options — no ``security`` accessor
signalwire.core.swml_service.SWMLService.verb_registry: impossible: Python exposes a ``verb_registry`` property returning a VerbHandlerRegistry; Go uses a private ``verbHandlers`` map on Service and exposes RegisterVerbHandler directly — no registry composition handle
signalwire.pom.pom.PromptObjectModel.sections: impossible: Python exposes a ``sections`` list PROPERTY; Go promotes it to an exported struct FIELD ``Sections []*Section`` (direct field access is idiomatic Go), which the signature/surface enumerators record as a field, not a zero-arg method member — the property-shaped accessor cannot exist
signalwire.pom.pom.Section.subsections: impossible: Python exposes a ``subsections`` list PROPERTY; Go promotes it to an exported struct FIELD ``Subsections []*Section`` (direct field access is idiomatic Go) — the property-shaped accessor cannot exist

signalwire.core.security.webhook_middleware.make_webhook_validation_dependency: impossible: FastAPI dependency factory; Go exposes equivalent as security.WebhookMiddleware (http.Handler middleware) — see PORT_ADDITIONS.md

# --- REST _base empty alias classes ---
signalwire.rest._base.FabricResource: impossible: Python empty base class (FabricResource(CrudResource)); Go aliases it to namespaces.CrudWithAddresses — no distinct surface to emit
signalwire.rest._base.FabricResourcePUT: impossible: Python empty base-class variant; Go aliases it to namespaces.CrudWithAddresses via NewCrudWithAddressesPUT — no distinct surface

# =====================================================================
# BACKLOG: generated typed-payload modules (SWML/SWAIG types-generation pass) —
# NOT YET ADOPTED. The REST *_resources_generated METHOD surface AND the REST
# field-level *_types_generated wire types are adopted (one Go type per
# components/schemas entry, surfaced as <ns>_types_generated classes). The RELAY
# WS protocol types (relay.protocol_types_generated) and the read-side SWAIG+SWML
# webhook types (swml_webhooks_types_generated) are ALSO now adopted — generated by
# cmd/generate-rest into pkg/relay/protocol_types_generated.go (123 structs) and
# pkg/rest/namespaces/swml_webhooks_types_generated.go (9 structs) — so their
# omissions have been removed (they are IMPLEMENTED, not deferred).
#
# What remains deferred is the CORE SWML/SWAIG generated-type surface: the SWML
# verb config types (core.swml_verbs_generated) and the read-side SWAIG payload
# types (core.post_prompt_generated / core.swaig_request_generated /
# core.swaig_actions_generated) — the D-workstream. Grouped here so the gate can
# distinguish "known-deferred" from "unexpected". These are REAL reference symbols
# the port has not generated yet — nothing invented.
#
# Note: the PostPromptData / SwaigArgument gen-type folds are now SATISFIED by the
# swml_webhooks_types_generated module above (each leaf duplicates into that module
# AND a still-deferred core.* module in the reference; the surface diff folds the
# duplicated leaf to gen-type.<Leaf>, which the port's swml_webhooks copy matches),
# so their omission entries have been removed.
# ---------------------------------------------------------------------
# (d) Deferred NON-REST generated types (SWML verbs / SWAIG read-side payloads):
# SWML/SWAIG core *_generated typed-payload modules (D workstream):

# --- LiveWire (LiveKit-agents compat shim) — Go is not a LiveKit agents SDK language ---

# --- Composition-attribute handles the reference surfaces (via the signature
# --- oracle's composition enrich) that Go implements with a different idiom.
# --- Each is a self-only Python property returning an SDK class; Go either folds
# --- the state into private fields with a renamed accessor, or exposes a plain
# --- (non-SDK-typed) field, so the reference member name genuinely cannot exist.
signalwire.core.pom_builder.PomBuilder.pom: impossible: Python's PomBuilder.pom returns the built PromptObjectModel; Go's builder returns the *PromptObjectModel directly from Build()/its terminal methods rather than exposing a ``pom`` composition handle
signalwire.web.web_service.WebService.security: impossible: Python's WebService.security returns a SecurityConfig composition handle; Go folds auth state into private fields configured via the WithSecurityConfig/WithBasicAuth/... options — no ``security`` accessor

# --- Raw-keyed twins of the agentbase-family.* omissions above. The SURFACE diff
# --- folds these members to `agentbase-family.<m>` (so the family keys above match
# --- there); the SIGNATURE diff (diff_port_signatures.py) does NOT fold, matching
# --- the reference's raw mixin/AgentBase path, so it needs the un-folded key to
# --- inherit the same excusal. Same omission, two key vocabularies.
signalwire.core.mixins.tool_mixin.ToolMixin.tool: impossible: Python @tool decorator relies on the decorator protocol; Go uses AgentBase.DefineTool(ToolDefinition{...})
signalwire.core.mixins.web_mixin.WebMixin.get_app: impossible: Python's WebMixin.get_app returns the FastAPI app object; Go has no framework app handle (AsRouter returns http.Handler)
signalwire.core.mixins.serverless_mixin.ServerlessMixin.handle_serverless_request: impossible: Python couples serverless request handling into the mixin; Go delegates to platform adapters (pkg/lambda) — no in-process AgentBase equivalent
signalwire.core.agent_base.AgentBase.skill_manager: impossible: Python exposes ``self.skill_manager`` (a SkillManager composition handle); Go folds the SkillManager into a private ``skillManager`` field and surfaces the user-facing methods (AddSkill/RemoveSkill/ListSkills/HasSkill) directly on AgentBase
