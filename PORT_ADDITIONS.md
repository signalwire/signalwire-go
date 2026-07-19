# PORT_ADDITIONS.md
#
# Every symbol listed here is a public Go-port API that has no direct
# Python-reference counterpart. Format: `<key>: <rationale>` per line.
# `diff_port_surface.py --port-additions-actual` enforces that every
# silently-dropped symbol from cmd/enumerate-surface is documented here.
# Keys may be either Python-canonical paths (signalwire.relay.event.AIEvent)
# or Go-native short paths (relay.AIEvent / agent.WithSTT). Both forms
# match.

# --- Existing curated entries (preserved) ---
signalwire.relay.event.AIEvent: Go-only typed wrapper around AI action events; Python uses RelayEvent directly

# --- Tier-2 idiom additions: context.Context-aware entry points (IDIOM_PASS_JOURNAL §4) ---
# Additive *Context variants of the blocking/async entry points that honor ctx
# cancellation + deadline. The existing non-ctx methods are PRESERVED and delegate
# with context.Background(), so these add zero drift (no oracle method changes
# shape). Python's run()/serve/dial loops have no caller-supplied cancellation token.
# NOTE: the signature/surface enumerators only project methods listed in the
# adapter rename tables and only record port-only STRUCTS / FREE FUNCTIONS in
# port_additions_actual.json — these methods-on-mapped-structs and package-level
# vars are invisible to both diff gates; documented here for the audit trail.
relay.Client.RunContext: Go ctx-aware form of Run; stops cleanly on ctx cancel/deadline (equivalent to Stop), returns ctx.Err(). Non-ctx Run preserved, delegates with context.Background()
relay.Client.DialContext: Go ctx-aware form of Dial; aborts the dial on ctx cancel/deadline returning ctx.Err(), alongside the dial-timeout + client lifecycle. Non-ctx Dial preserved
server.AgentServer.RunContext: Go ctx-aware form of AgentServer.Run; on ctx cancel/deadline performs a graceful Shutdown (drain) then returns nil. Non-ctx Run preserved
agent.AgentBase.RunContext: Go ctx-aware form of AgentBase.Run; on ctx cancel/deadline triggers the existing graceful HTTP shutdown (drains in-flight) then returns nil. Composes with SetupGracefulShutdown. Non-ctx Run preserved

# --- #192: Run() serverless-mode auto-detection + dispatch ---
# Run() is now the universal entry point: it computes the execution mode via
# swml.GetExecutionMode() (the cross-language detection contract — CGI/Lambda/
# GCF/Azure/server) and dispatches, mirroring Python web_mixin.run() +
# serverless_mixin. Server mode serves HTTP (existing behavior, preserved); CGI
# dispatches the single request inline through pkg/serverless (env+stdin→stdout);
# the platform-runtime-driven modes (Lambda/GCF/Azure) return ErrServerlessUnsupported
# rather than silently binding a TCP listener that never receives traffic — those
# are served via the adapters (pkg/lambda wraps AsRouter(), driven from main() by
# aws-lambda-go; pkg/serverless provides the GCF/CGI handlers), not inline in Run().
# DetectRunMode and
# RunWithMode are the Go-idiomatic shape for Python run()'s implicit
# get_execution_mode() call and its force_mode= override arg respectively;
# Python folds both into run()'s param list, Go exposes them as methods on the
# mapped struct (invisible to both diff gates, like the *Context methods above).
# Tested in pkg/agent/run_serverless_test.go (detection fixtures, force-mode
# dispatch, server-mode still serves HTTP).
agent.AgentBase.DetectRunMode: Go accessor returning the swml.ExecutionMode Run() would dispatch on (from swml.GetExecutionMode()). Mirrors Python run()'s internal get_execution_mode() call; lets callers branch (e.g. wire a pkg/lambda adapter) before invoking Run. Method-on-mapped-struct: invisible to both diff gates
agent.AgentBase.RunWithMode: Go force-mode form of Run — dispatches on the supplied swml.ExecutionMode rather than auto-detecting. Mirrors Python run(force_mode=...). Server mode serves HTTP; CGI dispatches the request inline via pkg/serverless; Lambda/GCF/Azure return ErrServerlessUnsupported (served via the adapters). Method-on-mapped-struct: invisible to both diff gates
agent.ErrServerlessUnsupported: Go sentinel — Run/RunWithMode detected a platform-runtime-driven serverless mode (Lambda/GCF/Azure); the agent must be served via its http.Handler (AsRouter) using the platform adapter (pkg/lambda for Lambda, pkg/serverless for GCF; CGI dispatches inline and does not return this), not Run(). errors.Is-able. No Python counterpart (Python's run() handles each platform inline; Go's serverless handling is the AsRouter+adapter path)
# REST client ctx-aware variants (cloud-product #19436). doRequest now delegates to
# doRequestContext via http.NewRequestWithContext; the non-ctx verbs are PRESERVED and
# delegate with context.Background(). Python's REST client has no caller-supplied
# cancellation token. Methods-on-mapped-structs: invisible to both diff gates.
rest.HTTPClient.GetContext: Go ctx-aware form of HTTPClient.Get; request is cancelled on ctx cancel/deadline. Non-ctx Get preserved, delegates with context.Background()
rest.HTTPClient.PostContext: Go ctx-aware form of HTTPClient.Post. Non-ctx Post preserved
rest.HTTPClient.PutContext: Go ctx-aware form of HTTPClient.Put. Non-ctx Put preserved
rest.HTTPClient.PatchContext: Go ctx-aware form of HTTPClient.Patch. Non-ctx Patch preserved
rest.HTTPClient.DeleteContext: Go ctx-aware form of HTTPClient.Delete. Non-ctx Delete preserved

# --- Tier-2 idiom additions: AgentServer graceful shutdown (IDIOM_PASS_JOURNAL §4) ---
server.AgentServer.Shutdown: Go graceful shutdown — stops accepting new connections and drains in-flight requests bounded by ctx's deadline (net/http.Server.Shutdown). Returns ErrServerNotRunning when not serving. No Python-reference equivalent (AgentServer has no graceful-shutdown surface)

# --- Tier-2 idiom additions: errors.Is-able sentinel errors (IDIOM_PASS_JOURNAL §4) ---
# Package-level sentinels wrapped with %w at their return sites so callers branch
# with errors.Is instead of scraping strings. Python uses RelayError + bare
# exceptions with no sentinel set. RelayError gained an Unwrap() so a single value
# satisfies BOTH errors.As(*RelayError) (existing) and errors.Is(sentinel) (new).
relay.ErrNotConnected: Go sentinel — operation needs a live WS connection but none exists (or it was torn down). errors.Is-able
relay.ErrDialTimeout: Go sentinel — Dial received no answering calling.call.dial event before its dial-timeout. errors.Is-able; also a *RelayError
relay.ErrDialFailed: Go sentinel — server reported a terminal "failed" dial_state (no device answered). errors.Is-able; also a *RelayError
relay.ErrExecuteTimeout: Go sentinel — a JSON-RPC request got no response within its deadline. errors.Is-able
server.ErrServerNotRunning: Go sentinel — Shutdown called with no server currently serving (before Run, or after stop). errors.Is-able

# --- Tier-3 idiom additions: typed RELAY state accessors (IDIOM_PASS_JOURNAL §4 "Tier 3") ---
# Typed-kind accessors ALONGSIDE the existing bare-string accessors (which are
# PRESERVED for parity with the Python reference's bare str). The typed kinds
# (relay.CallState / DialState / MessageState — documented in the structs block)
# give callers IsTerminal()/IsKnown() + compile-time distinctness of the three
# vocabularies; their underlying string equals the string accessor byte-for-byte
# → zero wire change. Methods-on-mapped-structs: invisible to both diff gates
# (same as the *Context methods above) — documented here for the audit trail.
relay.Call.CallState: Go typed-kind accessor returning relay.CallState alongside the bare-string Call.State() (kept). String == State() exactly
relay.Message.MessageState: Go typed-kind accessor returning relay.MessageState alongside the bare-string Message.State() (kept). String == State() exactly
relay.DialEvent.DialStateTyped: Go typed-kind accessor returning relay.DialState alongside the bare-string DialEvent.DialState field (kept). String == DialState exactly

# --- #188: register_routing_callback return-shape idiom (NOT a capability gap) ---
# Python's register_routing_callback (web_mixin.py:1227) callback returns a route
# string -> HTTP 307 redirect (web_mixin.py:704-711). Go's RegisterRoutingCallback
# callback returns a map[string]any that REPLACES the rendered SWML response
# document for that path (agent.go:2369; dispatched in the SWML handler at
# agent.go ~3293-3300 — non-nil map is served as the doc, else the default
# RenderSWML doc).
#
# This is an IDIOM difference, not a Go-only capability. Python ALREADY supports
# response-document override: on_swml_request(body, callback_path, request)
# returns an optional modifications dict that _render_swml merges into / rebuilds
# the served document (web_mixin.py:716-725). Go simply surfaces that same
# "customize the response document per request" power through the per-path
# routing-callback's return value rather than through an overridable hook method.
# Same capability, more ergonomic registration shape — so there is nothing to
# "port to Python"; the reference is not missing this.
#
# It still appears here only because the per-path callback's map return type
# differs from Python's str return type for the same-named method, so the diff
# would otherwise flag the shape. Used by examples swmlservice_ai_sidecar,
# dynamic_swml_service, swml_service_routing; behavioral test at
# pkg/agent/routing_callback_test.go (callback receives the live *http.Request and
# its returned map replaces the document). The callback function type is a
# parameter type, not an enumerated public symbol, so it is invisible to both
# diff gates; documented here for the audit trail.
signalwire.core.mixins.web_mixin.WebMixin.register_routing_callback: Go's RegisterRoutingCallback callback returns map[string]any (replace the rendered SWML document) vs Python's str (HTTP 307 redirect) — an idiom difference, NOT a capability gap: Python already supports response-document override via on_swml_request -> _render_swml. Same power, more ergonomic per-path registration shape. Behavioral test in pkg/agent/routing_callback_test.go

# --- Go-only structs (port-only public types) ---
agent.MCPServerConfig: Go-only config struct; not part of Python public API
agent.ToolDefinition: Go-only struct; no direct Python counterpart
builtin.APINinjasTriviaSkill: Go skill implementation; matches the Python skill of the same name structurally
builtin.ClaudeSkillsSkill: Go skill implementation; matches the Python skill of the same name structurally
builtin.CustomSkillsSkill: Go skill implementation; matches the Python skill of the same name structurally
builtin.DataSphereServerlessSkill: Go skill implementation; matches the Python skill of the same name structurally
builtin.DataSphereSkill: Go skill implementation; matches the Python skill of the same name structurally
builtin.DateTimeSkill: Go skill implementation; matches the Python skill of the same name structurally
builtin.GoogleMapsSkill: Go skill implementation; matches the Python skill of the same name structurally
builtin.InfoGathererSkill: Go skill implementation; matches the Python skill of the same name structurally
builtin.JokeSkill: Go skill implementation; matches the Python skill of the same name structurally
builtin.MCPGatewaySkill: Go skill implementation; matches the Python skill of the same name structurally
builtin.MathSkill: Go skill implementation; matches the Python skill of the same name structurally
builtin.NativeVectorSearchSkill: Go skill implementation; matches the Python skill of the same name structurally
builtin.PlayBackgroundFileSkill: Go skill implementation; matches the Python skill of the same name structurally
builtin.SWMLTransferSkill: Go skill implementation; matches the Python skill of the same name structurally
spider.SpiderSkill: Go skill implementation (own `spider` sub-package); matches the Python skill of the same name structurally, surfaced via the skill-contract projection under signalwire.skills.spider.skill
builtin.WeatherAPISkill: Go skill implementation; matches the Python skill of the same name structurally
builtin.WebSearchSkill: Go skill implementation; matches the Python skill of the same name structurally
builtin.WikipediaSearchSkill: Go skill implementation; matches the Python skill of the same name structurally
datamap.ExpressionPattern: Go-only struct; no direct Python counterpart
lambda.Handler: Go-only struct; no direct Python counterpart
serverless.Handler: Go-only struct; the CGI / Google Cloud Functions dispatch adapter (analog of pkg/lambda for the non-Lambda serverless platforms), wrapping the agent's http.Handler. Python's ServerlessMixin dispatches these inline; Go's serverless request handling is the AsRouter+adapter path
serverless.CGIResult: Go-only struct; the CGI dispatch outcome (status/headers/body) that WriteCGI serializes. No Python counterpart (Python writes the CGI response inline via stdout)
logging.Logger: Go-only struct; no direct Python counterpart
logging.LogLevel: Go-only defined-string type (closed set of log-level names: debug/info/warn/warning/error/off) + LevelName* typed constants; server.WithLogLevel takes it for autocomplete + call-site typo checking, while Go's untyped-constant auto-conversion keeps a bare "debug" string compiling — parity with the reference's plain str log_level. ParseLevel(string(LogLevel)) resolves it to the internal Level, so it adds zero signature drift (it appears on no oracle method param). Distinct from the internal Level severity int.
namespaces.CrudResource: Go REST resource type; Python uses dynamic resource accessors via __getattr__
namespaces.CrudWithAddresses: Go-only struct; no direct Python counterpart
namespaces.CxmlApplicationsResource: Go REST resource type; Python uses dynamic resource accessors via __getattr__
prefabs.Amenity: Go-only struct; no direct Python counterpart
prefabs.BedrockAgent: Go-only struct; no direct Python counterpart
prefabs.BedrockOptions: Go-only options struct; encodes Python kwargs for the matching constructor
prefabs.ConciergeOptions: Go-only options struct; encodes Python kwargs for the matching constructor
prefabs.Department: Go-only struct; no direct Python counterpart
prefabs.FAQ: Go-only struct; no direct Python counterpart
prefabs.FAQBotOptions: Go-only options struct; encodes Python kwargs for the matching constructor
prefabs.InfoGathererOptions: Go-only options struct; encodes Python kwargs for the matching constructor
prefabs.Question: Go-only struct; no direct Python counterpart
prefabs.ReceptionistOptions: Go-only options struct; encodes Python kwargs for the matching constructor
prefabs.SurveyOptions: Go-only options struct; encodes Python kwargs for the matching constructor
prefabs.SurveyQuestion: Go-only struct; no direct Python counterpart
relay.AIEvent: Go-only struct; no direct Python counterpart
relay.CallState: Go-only defined-string kind (call lifecycle: created/ringing/answered/ending/ended) + Call*/typed-const set; Call.CallState() returns it ALONGSIDE the bare-string Call.State() (parity — string accessor kept). IsTerminal() (ended) + IsKnown() predicates; server-emitted+growable so unknown values flow through (no exhaustive-switch break). Underlying string == State() byte-for-byte → zero wire change + invisible to both signature & surface enumerators (no oracle symbol). Grounded in Python relay/constants.py CALL_STATES. DISTINCT type from DialState/MessageState — never conflated.
relay.CollectParams: Go-only struct; no direct Python counterpart
relay.Device: Go-only struct typing the {type, params} object passed across connect/refer/dial/tap (Type string — discriminant is NOT schema-enumerated; Params any). Purely additive — the raw map[string]any / [][]map[string]any path is unchanged; ToMap()/MarshalJSON + DeviceList/DeviceGroups yield the IDENTICAL wire shape (nil Params → "params":{}). Grounded in porting-sdk/relay-protocol/calling.{connect,dial,refer,tap}.params.json. Invisible to both enumerators (no oracle symbol).
relay.DialState: Go-only defined-string kind (dial outcome: dialing/answered/failed) read from wire dial_state; DialEvent.DialStateTyped() returns it ALONGSIDE the bare-string DialEvent.DialState field (parity kept). IsTerminal() (answered|failed; dialing is progress) + IsKnown(). Grounded in Python relay/client.py:950 dial_state docstring + :1006 ("dialing" is progress). A SEPARATE vocabulary from CallState/MessageState — distinct Go type, never conflated. Invisible to both enumerators.
relay.MessageState: Go-only defined-string kind (delivery lifecycle: queued/initiated/sent/delivered/undelivered/failed/received) + Msg*/typed-const set; Message.MessageState() returns it ALONGSIDE the bare-string Message.State() (parity kept). IsTerminal() mirrors Python MESSAGE_TERMINAL_STATES (delivered/undelivered/failed — inbound "received" excluded, matching Python exactly; the internal isTerminalMessageState keeps "received" terminal for Wait() short-circuit, a separate behavior-only concern) + IsKnown(). Underlying string == State() byte-for-byte. Grounded in Python relay/constants.py MESSAGE_STATE_*/MESSAGE_TERMINAL_STATES. DISTINCT type from CallState/DialState. Invisible to both enumerators.
relay.RelayError: Go-only struct; no direct Python counterpart
relay.TTSGender: Go-only defined-string type (closed set of TTS voice genders: male/female) + GenderMale/GenderFemale typed constants; WithTTSGender (the play_tts/prompt_tts gender option) takes it for autocomplete + call-site typo checking, while Go's untyped-constant auto-conversion keeps a bare "female" string compiling — parity with the reference's plain str gender. Stored on the wire as a plain string, so it adds zero signature drift (it appears on no oracle method param).
server.AgentEntry: Go-only struct; no direct Python counterpart
skills.ToolRegistration: Go-only struct; no direct Python counterpart
skills.SkillName: Go-only defined-string type (closed set of the 18 built-in skill names) + Skill* typed constants; AddSkill/RemoveSkill/HasSkill take it for autocomplete + call-site typo checking, while Go's untyped-constant auto-conversion keeps bare "datetime" / SkillName("custom") strings compiling — parity with the reference's str. Wire-identical to string, so signature drift stays 0 (the union<class:...,string> the enumerator emits absorbs against the reference's str). Mirrors the PHP SkillName backed-enum proof.
swaig.Codec: Go-only defined-string type (closed set of SWAIG-tap audio codecs: PCMU/PCMA) + Codec* typed constants; FunctionResult.Tap takes it for autocomplete + call-site typo checking, while Go's untyped-constant auto-conversion keeps a bare "PCMU" string compiling — parity with the reference's str codec (validated valid_codecs=["PCMU","PCMA"] at function_result.py:1217). Wire-identical to string, so signature drift stays 0 (the union<class:swaig.Codec,string> the enumerator emits for tap's codec param absorbs against the reference's str). DISTINCT from the larger RELAY connect/stream codec superset (genuinely open/multi-value → left a bare string); never reuse this type there.
swaig.JoinConferenceOptions: Go-only options struct; encodes Python kwargs for the matching constructor
swaig.PayOptions: Go-only options struct; encodes Python kwargs for the matching constructor
swaig.RecordCallOptions: Go-only options struct; encodes Python kwargs for the matching constructor
swaig.RecordDirection: Go-only defined-string type (closed set of record_call audio directions: speak/listen/both) + RecordDirection* typed constants; FunctionResult.RecordCall takes it for autocomplete + call-site typo checking, while Go's untyped-constant auto-conversion keeps a bare "both" string compiling — parity with the reference's str direction (validated valid_directions=["speak","listen","both"] at function_result.py:917). Wire-identical to string, so signature drift stays 0 (the union<class:swaig.RecordDirection,string> the enumerator emits for record_call's direction param absorbs against the reference's str). DISTINCT from swaig.TapDirection (tap uses "hear" where record_call uses "listen") — never unify the two.
swaig.RecordFormat: Go-only defined-string type (closed set of recording formats: mp3/wav/mp4) + Format* typed constants; FunctionResult.RecordCall (and the relay/agent WithRecordFormat options) take it for autocomplete + call-site typo checking, while Go's untyped-constant auto-conversion keeps a bare "wav" string compiling — parity with the reference's str format. Wire-identical to string, so signature drift stays 0 (the union<class:swaig.RecordFormat,string> the enumerator emits for record_call's format param absorbs against the reference's str).
swaig.TapDirection: Go-only defined-string type (closed set of tap audio directions: speak/hear/both) + TapDirection* typed constants; FunctionResult.Tap takes it for autocomplete + call-site typo checking, while Go's untyped-constant auto-conversion keeps a bare "both" string compiling — parity with the reference's str direction (validated valid_directions=["speak","hear","both"] at function_result.py:1212). Wire-identical to string, so signature drift stays 0 (the union<class:swaig.TapDirection,string> the enumerator emits for tap's direction param absorbs against the reference's str). DISTINCT from swaig.RecordDirection (record_call uses "listen" where tap uses "hear") — never unify the two.

# --- Tier-2 flagship: typed SWAIG tool-parameter builder (IDIOM_PASS_JOURNAL §4) ---
# A fluent, type-safe builder over the SAME wire shape ToolDefinition.Parameters
# (a JSON-Schema *properties* map[string]any) + ToolDefinition.Required ([]string)
# already carry. Replaces the hand-written nested-map blob (concierge.go:180+,
# survey.go:285+); produces byte-identical output (reflect.DeepEqual / JSON-equal,
# proven by TestParamsByteIdentical* + the agent render integration tests). Purely
# additive — the untyped Parameters path is unchanged. Integrates the Tier-1 typed
# enums via the *Values() helpers (Params.Enum(name, swaig.RecordFormatValues(), …)
# → schema "enum":[…]). No Python-reference counterpart (Python hand-builds the same
# dict); the signatures enumerator doesn't see these (not in StructTable/FreeFnTable)
# so signature drift stays 0, but the SURFACE enumerator records port-only structs +
# free functions, so each is listed here for the --port-additions-actual audit trail.
swaig.Params: Go-only fluent builder struct accumulating SWAIG tool-parameter properties + the top-level required list; renders to ToolDefinition.Parameters (map[string]any properties map) via Properties()/Build() and ToolDefinition.Required ([]string) via RequiredNames()/Build(). Typed convenience over the untyped Parameters blob — byte-identical wire output. No Python counterpart.
swaig.Prop: Go-only struct — a single JSON-Schema property under construction (produced by the Prop* kind constructors, used as Array item / Object schemas). No Python counterpart.
swaig.NewParams: Go-only constructor returning an empty *Params builder. No Python counterpart.
swaig.PropString: Go-only kind constructor returning a string-typed *Prop (for Array items / standalone use). No Python counterpart.
swaig.PropNumber: Go-only kind constructor returning a number-typed (float) *Prop. No Python counterpart.
swaig.PropInteger: Go-only kind constructor returning an integer-typed *Prop. No Python counterpart.
swaig.PropBoolean: Go-only kind constructor returning a boolean-typed *Prop. No Python counterpart.
swaig.PropEnum: Go-only kind constructor returning a string-typed *Prop constrained to a closed set (JSON-Schema "enum"); pairs with the *Values() helpers. No Python counterpart.
swaig.PropArray: Go-only kind constructor returning an array-typed *Prop whose elements match a given item *Prop. No Python counterpart.
swaig.PropObject: Go-only kind constructor returning an object-typed *Prop with nested properties + nested required list (from a *Params). No Python counterpart.
swaig.Default: Go-only PropOption setting a property's JSON-Schema "default" keyword. No Python counterpart.
swaig.Format: Go-only PropOption setting a property's JSON-Schema "format" keyword (e.g. "date"). No Python counterpart.
swaig.WithEnum: Go-only PropOption attaching a JSON-Schema "enum" closed set to a property. No Python counterpart.
swaig.Required: Go-only PropOption marking the enclosing property required (inline alternative to Params.Required(names...)). No Python counterpart. (Note: the same name is also the variadic fluent setter method Params.Required(...) — a method, invisible to the surface enumerator; this entry is the free-function option.)
swaig.RecordFormatValues: Go-only helper returning the RecordFormat closed set as wire strings (mp3/wav/mp4) for Params.Enum/PropEnum/WithEnum — bridges the Tier-1 enum into schema "enum". No Python counterpart.
swaig.RecordDirectionValues: Go-only helper returning the RecordDirection closed set as wire strings (speak/listen/both) for schema "enum". No Python counterpart.
swaig.TapDirectionValues: Go-only helper returning the TapDirection closed set as wire strings (speak/hear/both) for schema "enum". No Python counterpart.
swaig.CodecValues: Go-only helper returning the Codec closed set as wire strings (PCMU/PCMA) for schema "enum". No Python counterpart.
swml.AIVerbHandler: Go-only struct; no direct Python counterpart
swml.Document: Go-only struct; no direct Python counterpart
swml.Schema: Go-only struct; no direct Python counterpart
swml.SecurityConfig: Go-only config struct; not part of Python public API
swml.ToolDefinition: Go-only struct; no direct Python counterpart
swml.VerbInfo: Go-only struct; no direct Python counterpart

# --- Go-only functions (functional-options helpers, factory constructors, package utilities) ---
agent.WithAIVerbName: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithAgentID: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithAutoAnswer: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithBasicAuth: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithBullet: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithBullets: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithCheckForInputOverride: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithConfigFile: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithDefaultWebhookURL: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithEnablePostPromptOverride: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithHost: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithName: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithNativeFunctions: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithNumbered: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithNumberedBullets: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithPort: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithRecordCall: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithRecordFormat: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithRecordStereo: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithRoute: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithSchemaPath: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithSchemaValidation: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithSubsections: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithSuppressLogs: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithTokenExpiry: Go functional-options helper; encodes a Python kwarg for the matching constructor
agent.WithUsePom: Go functional-options helper; encodes a Python kwarg for the matching constructor
builtin.NewAPINinjasTrivia: Go factory constructor for a port-only struct; Python equivalent does not exist
builtin.NewClaudeSkills: Go factory constructor for a port-only struct; Python equivalent does not exist
builtin.NewCustomSkills: Go factory constructor for a port-only struct; Python equivalent does not exist
builtin.NewDataSphere: Go factory constructor for a port-only struct; Python equivalent does not exist
builtin.NewDataSphereServerless: Go factory constructor for a port-only struct; Python equivalent does not exist
builtin.NewDateTime: Go factory constructor for a port-only struct; Python equivalent does not exist
builtin.NewGoogleMaps: Go factory constructor for a port-only struct; Python equivalent does not exist
builtin.NewInfoGatherer: Go factory constructor for a port-only struct; Python equivalent does not exist
builtin.NewJoke: Go factory constructor for a port-only struct; Python equivalent does not exist
builtin.NewMCPGateway: Go factory constructor for a port-only struct; Python equivalent does not exist
builtin.NewMath: Go factory constructor for a port-only struct; Python equivalent does not exist
builtin.NewNativeVectorSearch: Go factory constructor for a port-only struct; Python equivalent does not exist
builtin.NewPlayBackgroundFile: Go factory constructor for a port-only struct; Python equivalent does not exist
builtin.NewSWMLTransfer: Go factory constructor for a port-only struct; Python equivalent does not exist
builtin.NewSpider: Go factory constructor for a port-only struct; Python equivalent does not exist
builtin.NewWeatherAPI: Go factory constructor for a port-only struct; Python equivalent does not exist
builtin.NewWebSearch: Go factory constructor for a port-only struct; Python equivalent does not exist
builtin.NewWikipediaSearch: Go factory constructor for a port-only struct; Python equivalent does not exist
contexts.WithConfirm: Go functional-options helper; encodes a Python kwarg for the matching constructor
contexts.WithFunctions: Go functional-options helper; encodes a Python kwarg for the matching constructor
contexts.WithIsolated: Go functional-options helper; encodes the per-question isolated kwarg (GatherQuestion.isolated) for AddGatherQuestion
contexts.WithPrompt: Go functional-options helper; encodes a Python kwarg for the matching constructor
contexts.WithType: Go functional-options helper; encodes a Python kwarg for the matching constructor
lambda.NewHandler: Go factory constructor for a port-only struct; Python equivalent does not exist
serverless.NewHandler: Go factory constructor for the port-only serverless.Handler (CGI/GCF adapter); Python equivalent does not exist
serverless.WriteCGI: Go-only public function serializing a CGIResult to the CGI response wire format (Status line + headers + body); Python writes the CGI response inline
logging.GetGlobalLevel: Go-only public function; no direct Python counterpart
logging.IsSuppressed: Go-only public function; no direct Python counterpart
logging.New: Go factory constructor for a port-only struct; Python equivalent does not exist
logging.ParseLevel: Go-only public function; no direct Python counterpart
logging.ResetLoggingConfiguration: Go-only public function; no direct Python counterpart
logging.SetGlobalLevel: Go-only public function; no direct Python counterpart
logging.Suppress: Go-only public function; no direct Python counterpart
logging.Unsuppress: Go-only public function; no direct Python counterpart
namespaces.AllPhoneCallHandlers: Go-only public function; no direct Python counterpart
namespaces.NewCrudResource: Go factory constructor for a port-only struct; Python equivalent does not exist
namespaces.NewCrudResourcePUT: Go factory constructor for a port-only struct; Python equivalent does not exist
namespaces.NewCrudWithAddresses: Go factory constructor for a port-only struct; Python equivalent does not exist
namespaces.NewCrudWithAddressesPUT: Go factory constructor for a port-only struct; Python equivalent does not exist
prefabs.NewBedrockAgent: Go factory constructor for a port-only struct; Python equivalent does not exist
prefabs.NewSurveyQuestion: Go factory constructor for a port-only struct; Python equivalent does not exist
prefabs.WithOptional: Go functional-options helper; encodes a Python kwarg for the matching constructor
prefabs.WithQuestionChoices: Go functional-options helper; encodes a Python kwarg for the matching constructor
prefabs.WithQuestionID: Go functional-options helper; encodes a Python kwarg for the matching constructor
prefabs.WithQuestionScale: Go functional-options helper; encodes a Python kwarg for the matching constructor
prefabs.WithQuestionType: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.DeviceGroups: Go-only helper — converts serial groups of parallel typed relay.Devices into the [][]map[string]any shape Connect/Dial take, byte-identical to hand-built nesting; no Python counterpart
relay.DeviceList: Go-only helper — converts a flat list of typed relay.Devices into one parallel [][]map[string]any leg for Connect/Dial; no Python counterpart
relay.NewAIEvent: Go factory constructor for a port-only struct; Python equivalent does not exist
relay.NewDevice: Go factory constructor for the port-only relay.Device struct; Python uses a raw {type, params} dict
relay.NewRelayClient: Go factory constructor for a port-only struct; Python equivalent does not exist
relay.NewRelayError: Go factory constructor for a port-only struct; Python equivalent does not exist
relay.WithAIParams: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithAIPostPrompt: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithAIPrompt: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithAMDDetectInterruptions: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithAMDDetectMessageEnd: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithAMDEndSilenceTimeout: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithAMDInitialTimeout: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithAMDMachineVoiceThreshold: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithAMDMachineWordsThreshold: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithAMDTimeout: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithAudioVolume: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithConferenceBeep: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithConferenceMuted: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithConnectRingback: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithContexts: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithDialFromNumber: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithDialMaxDuration: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithDialTimeout: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithDigitDigits: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithDigitTimeout: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithEnvDefaults: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithFaxDetectTimeout: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithFaxHeaderInfo: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithFaxTone: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithJWT: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithMaxActiveCalls: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithMessageContext: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithMessageMedia: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithMessageOnCompleted: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithMessageRegion: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithMessageTags: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithPayChargeAmount: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithPayCurrency: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithPayDescription: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithPayInputMethod: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithPayLanguage: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithPayMaxAttempts: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithPayMinPostalCodeLength: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithPayParameters: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithPayPaymentMethod: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithPayPostalCode: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithPayPrompts: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithPaySecurityCode: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithPayStatusURL: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithPayTimeout: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithPayTokenType: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithPayValidCardTypes: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithPayVoice: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithPlayVolume: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithProject: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithRecordBeep: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithRecordDirection: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithRecordEndSilenceTimeout: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithRecordFormat: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithRecordInitialTimeout: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithRecordStereo: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithRecordTerminators: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithRingtoneDuration: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithRingtoneVolume: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithSpace: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithStreamCodec: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithTTSGender: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithTTSLanguage: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithTTSVoice: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithTTSVolume: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithToken: Go functional-options helper; encodes a Python kwarg for the matching constructor
security.WithDebugMode: Go functional-options helper; encodes a Python kwarg for the matching constructor
security.WithSecret: Go functional-options helper; encodes a Python kwarg for the matching constructor
server.WithLogLevel: Go functional-options helper; encodes a Python kwarg for the matching constructor
server.WithRunHost: Go functional-options helper; encodes a Python kwarg for the matching constructor
server.WithRunPort: Go functional-options helper; encodes a Python kwarg for the matching constructor
server.WithServerHost: Go functional-options helper; encodes a Python kwarg for the matching constructor
server.WithServerPort: Go functional-options helper; encodes a Python kwarg for the matching constructor
skills.GetSkillFactory: Go-only public function; no direct Python counterpart
swaig.CreatePaymentAction: Go-only public function; no direct Python counterpart
swaig.CreatePaymentParameter: Go-only public function; no direct Python counterpart
swaig.CreatePaymentPrompt: Go-only public function; no direct Python counterpart
swml.ExtractSIPUsername: Go-only public function; no direct Python counterpart
swml.GetExecutionMode: Go-only public function; no direct Python counterpart
swml.GetSchema: Go-only public function; no direct Python counterpart
swml.IsServerlessMode: Go-only public function; no direct Python counterpart
swml.LoadSchemaFromFile: Go-only public function; no direct Python counterpart
swml.NewAIVerbHandler: Go factory constructor for a port-only struct; Python equivalent does not exist
swml.NewDocument: Go factory constructor for a port-only struct; Python equivalent does not exist
swml.ValidateURL: Go-only public function; no direct Python counterpart
swml.WithAPIKey: Go functional-options helper; encodes a Python kwarg for the matching constructor
swml.WithBasicAuth: Go functional-options helper; encodes a Python kwarg for the matching constructor
swml.WithBearerToken: Go functional-options helper; encodes a Python kwarg for the matching constructor
swml.WithConfigFile: Go functional-options helper; encodes a Python kwarg for the matching constructor
swml.WithDomain: Go functional-options helper; encodes a Python kwarg for the matching constructor
swml.WithHost: Go functional-options helper; encodes a Python kwarg for the matching constructor
swml.WithName: Go functional-options helper; encodes a Python kwarg for the matching constructor
swml.WithPort: Go functional-options helper; encodes a Python kwarg for the matching constructor
swml.WithRoute: Go functional-options helper; encodes a Python kwarg for the matching constructor
swml.WithSchemaPath: Go functional-options helper; encodes a Python kwarg for the matching constructor
swml.WithSchemaValidation: Go functional-options helper; encodes a Python kwarg for the matching constructor
swml.WithSecurityConfig: Go functional-options helper; encodes a Python kwarg for the matching constructor
swml.WithTLS: Go functional-options helper; encodes a Python kwarg for the matching constructor

# --- Go-only public Logger field auto-projected onto each struct that embeds it ---
signalwire.core.agent_base.AgentBase.logger: Go's AgentBase exposes a public ``Logger *logging.Logger`` field; auto-projected as ``logger`` accessor on the Python-canonical class
signalwire.core.mixins.ai_config_mixin.AIConfigMixin.logger: AIConfigMixin methods are projected from agent.AgentBase, which exposes a public ``Logger *logging.Logger`` field
signalwire.core.mixins.prompt_mixin.PromptMixin.logger: PromptMixin methods are projected from agent.AgentBase, which exposes a public ``Logger *logging.Logger`` field
signalwire.core.mixins.skill_mixin.SkillMixin.logger: SkillMixin methods are projected from agent.AgentBase, which exposes a public ``Logger *logging.Logger`` field
signalwire.core.mixins.tool_mixin.ToolMixin.logger: ToolMixin methods are projected from agent.AgentBase, which exposes a public ``Logger *logging.Logger`` field
signalwire.core.mixins.web_mixin.WebMixin.logger: WebMixin methods are projected from agent.AgentBase, which exposes a public ``Logger *logging.Logger`` field
signalwire.core.swml_service.SWMLService.logger: Go's swml.Service exposes a public ``Logger *logging.Logger`` field; auto-projected as ``logger`` accessor on the Python-canonical class


# --- Go-only fields on REST base resources (Python uses dynamic attribute lookup) ---
signalwire.rest._base.BaseResource.http: Go's namespaces.Resource exposes a public ``http`` HTTPClient field; Python uses dynamic attribute lookup via __init__
signalwire.rest._base.CrudResource.client: Go's namespaces.CrudResource exposes a public ``client`` HTTPClient field; Python uses dynamic attribute lookup via __init__
signalwire.rest._base.ReadResource.client: same Go namespaces.CrudResource ``client`` field, surfaced under the ReadResource half of the CrudResource->ReadResource base-placement adapter (internal/surface/tables.go); Python's ReadResource uses dynamic attribute lookup

# --- Go projections of Python attributes the Python adapter drops from surface but keeps in signatures ---
# Python's enumerate-surface omits these as instance properties; signatures keeps them.
# Go projects them via the StructTable rename map so the signature audit aligns; surface side excused here.
signalwire.core.agent_base.AgentBase.pom: Go's Pom() method projects to Python's pom property; Python's signatures index includes it but the surface index drops it as an instance attribute
signalwire.core.swml_service.SWMLService.schema_utils: Go's SchemaUtils field projects to Python's schema_utils property; Python's signatures index includes it but the surface index drops it as an instance attribute
signalwire.relay.call.Action.result: Go's Result() method projects to Python's result property; Python's signatures index includes it but the surface index drops it as an instance attribute

# --- ReadResource.paginate() surfaced on the concrete read-only subclasses ---
# The Python SIGNATURES oracle records ReadResource.paginate() ON each concrete
# read-only subclass (FabricAddresses/FaxLogs/MessageLogs/VideoRoomSessions/
# VoiceLogs — verified against python_signatures.json), so the port emits a real
# Paginate() on each to keep DRIFT clean (paginate is NOT a _CRUD_METHODS verb, so
# the crud_base structural excusal that covers list/get does NOT cover it). The
# Python SURFACE oracle, however, records paginate() only on the _base.ReadResource
# BASE (like list/get) and lists just __init__ on the subclass — so the subclass
# copies read as port additions here. The port's _base.ReadResource surface already
# carries paginate (via the CrudResource->ReadResource base-placement adapter),
# matching the reference base; these five are the subclass projections the signature
# audit requires, excused on the surface side. (Same shape as the pom/schema_utils
# signatures-keeps/surface-drops entries above.)
signalwire.rest.namespaces.fabric_resources_generated.FabricAddresses.paginate: ReadResource.paginate() on the concrete subclass — required by the signatures oracle (records it per-subclass), surface oracle records it only on the ReadResource base
signalwire.rest.namespaces.fax_resources_generated.FaxLogs.paginate: ReadResource.paginate() on the concrete subclass — required by the signatures oracle (records it per-subclass), surface oracle records it only on the ReadResource base
signalwire.rest.namespaces.message_resources_generated.MessageLogs.paginate: ReadResource.paginate() on the concrete subclass — required by the signatures oracle (records it per-subclass), surface oracle records it only on the ReadResource base
signalwire.rest.namespaces.video_resources_generated.VideoRoomSessions.paginate: ReadResource.paginate() on the concrete subclass — required by the signatures oracle (records it per-subclass), surface oracle records it only on the ReadResource base
signalwire.rest.namespaces.voice_resources_generated.VoiceLogs.paginate: ReadResource.paginate() on the concrete subclass — required by the signatures oracle (records it per-subclass), surface oracle records it only on the ReadResource base

## SWML-verbs generated-payload reserved-word fields (port emits what the reference can't name)

The reference's TypedDict generator cannot name a field that is a Python keyword, so it
drops `else` to a `# non-identifier field 'else'` comment (the wire key still round-trips
at runtime). Go struct field tags have no such restriction, so the generated SWML-verb
configs legitimately type the field — the port is MORE faithful to the wire than the
reference can express. This is the read-side analog of the `from`→`From` reserved-word
handling. Keyed by the gen-payload fold token.

gen-payload.CondElse.else: generated SWML-verb config field the Python reference drops because `else` is a Python keyword (recorded as a `# non-identifier field` comment); the wire key is real and the Go struct types it
gen-payload.CondReg.else: generated SWML-verb config field the Python reference drops because `else` is a Python keyword (recorded as a `# non-identifier field` comment); the wire key is real and the Go struct types it

# --- Port-only helper structs (options/results) ---
swml.PlayOptions: Go options struct for swml.Service.Play — the idiomatic Go named-options shape that replaced the 7-positional-pointer signature (plan 6.2-go). Its fields unfold 1:1 back to the Python play(url, urls, volume, say_voice, say_language, say_gender, auto_answer) keyword params (enumerate-signatures optionsStructUnfoldMethods), so signature drift stays 0; the struct type itself is port-only call-shape plumbing.
swml.AIOptions: Go options struct for swml.Service.AI — the idiomatic Go named-options shape that replaced the 6-positional signature (plan 6.2-go). Fields unfold to ai(prompt_text, prompt_pom, post_prompt, post_prompt_url, swaig, **kwargs) with the Extra map folding to the reference **kwargs tail; port-only call-shape plumbing.
web.Options: Go options struct for web.NewWebService — the idiomatic Go constructor-options shape for the WebService static-file server (Python passes a flat kwarg list). Call-shape plumbing, not oracle surface.
swml.ValidationResult: Go struct returned by swml schema validation — an idiomatic typed result the Python reference expresses as a (bool, errors) tuple. Port-only helper type.
namespaces.Paginator: Go value returned by CrudResource.Paginate()/the ReadResource-subclass Paginate() — the LIVE paginator that walks a list endpoint's links.next cursor. It REPRESENTS Python's _pagination.PaginatedIterator class (adapter StructTable maps NewPaginator->__init__, Next->__next__, synthetic __iter__), so it is the port's counterpart to that class, not a bare addition; the former orphan rest.PaginatedIterator (no Paginate() returned it) was retired in plan 6.2-go and its mapping moved here. The Paginate() return type also folds to that class ref (enumerate-signatures goLocalAliases) so signatures compare EQUAL. Lives in the namespaces package (not rest) to avoid the rest->namespaces import cycle.
namespaces.NewPaginator: constructor for namespaces.Paginator (above) — maps to _pagination.PaginatedIterator.__init__; Go-idiom factory. Python constructs PaginatedIterator inline in ReadResource.paginate().

# --- Extras escape hatch (typed-first surface) ---

Every generated Create/Update/Set* params struct carries an `Extras map[string]any`
field. The typed fields ARE the intended surface — reach for a typed field whenever one
exists; `Extras` is the escape hatch for genuinely-open / forward-compat wire keys the
typed surface does not (yet) model (the analog of Python's `**kwargs` tail and the closed-
create-params + `extras` design in TYPED_SURFACE_STRATEGY §4). This preserves Python's
dynamic-kwargs capability 1:1 while keeping the typed surface the default.

Merge semantics (`namespaces.mergeExtra`, common.go): the generated wrapper writes the
typed params into the request body first (each unconditionally, including its Go zero
value), then merges `Extras` — so on a key collision the Extras value wins (last-writer-
wins, Extras applied last). That override is deliberate: because a typed param is written
even at its zero value, an Extras entry of the same name is the only way to supply a wire
value the caller left on the typed zero. A key set only via Extras still reaches the wire
and is caught by the strict mock's 400-on-unknown-key in the test lanes. Not oracle surface
(the field folds away like the other map[string]any open-param plumbing).
