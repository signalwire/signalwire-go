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
signalwire.livewire.plugins.GoogleSTT: Go-only plugin stub; matches WithSTT("google") at AgentSession construction
signalwire.livewire.plugins.OpenAITTS: Go-only plugin stub; matches WithTTS("openai") at AgentSession construction

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
builtin.SpiderSkill: Go skill implementation; matches the Python skill of the same name structurally
builtin.WeatherAPISkill: Go skill implementation; matches the Python skill of the same name structurally
builtin.WebSearchSkill: Go skill implementation; matches the Python skill of the same name structurally
builtin.WikipediaSearchSkill: Go skill implementation; matches the Python skill of the same name structurally
datamap.ExpressionPattern: Go-only struct; no direct Python counterpart
lambda.Handler: Go-only struct; no direct Python counterpart
livewire.ChatContext: Go-only struct; no direct Python counterpart
livewire.ChatMessage: Go-only struct; no direct Python counterpart
livewire.GoogleSTT: Go livewire plugin stub; resolves WithSTT/WithTTS provider strings
livewire.InferenceLLM: Go-only struct; no direct Python counterpart
livewire.InferenceSTT: Go livewire plugin stub; resolves WithSTT/WithTTS provider strings
livewire.InferenceTTS: Go livewire plugin stub; resolves WithSTT/WithTTS provider strings
livewire.OpenAITTS: Go livewire plugin stub; resolves WithSTT/WithTTS provider strings
livewire.ToolError: Go-only struct; no direct Python counterpart
logging.Logger: Go-only struct; no direct Python counterpart
logging.LogLevel: Go-only defined-string type (closed set of log-level names: debug/info/warn/warning/error/off) + LevelName* typed constants; server.WithLogLevel takes it for autocomplete + call-site typo checking, while Go's untyped-constant auto-conversion keeps a bare "debug" string compiling — parity with the reference's plain str log_level. ParseLevel(string(LogLevel)) resolves it to the internal Level, so it adds zero signature drift (it appears on no oracle method param). Distinct from the internal Level severity int.
namespaces.CallFlowOptions: Go-only options struct; encodes Python kwargs for the matching constructor
namespaces.CrudResource: Go REST resource type; Python uses dynamic resource accessors via __getattr__
namespaces.CrudWithAddresses: Go-only struct; no direct Python counterpart
namespaces.CxmlApplicationsResource: Go REST resource type; Python uses dynamic resource accessors via __getattr__
namespaces.CxmlWebhookOptions: Go-only options struct; encodes Python kwargs for the matching constructor
namespaces.RelayTopicOptions: Go-only options struct; encodes Python kwargs for the matching constructor
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
relay.CollectParams: Go-only struct; no direct Python counterpart
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
contexts.WithPrompt: Go functional-options helper; encodes a Python kwarg for the matching constructor
contexts.WithType: Go functional-options helper; encodes a Python kwarg for the matching constructor
lambda.NewHandler: Go factory constructor for a port-only struct; Python equivalent does not exist
livewire.NewAgentServer: Go factory constructor for a port-only struct; Python equivalent does not exist
livewire.NewChatContext: Go factory constructor for a port-only struct; Python equivalent does not exist
livewire.NewGoogleSTT: Go factory constructor for a port-only struct; Python equivalent does not exist
livewire.NewInferenceLLM: Go factory constructor for a port-only struct; Python equivalent does not exist
livewire.NewInferenceSTT: Go factory constructor for a port-only struct; Python equivalent does not exist
livewire.NewInferenceTTS: Go factory constructor for a port-only struct; Python equivalent does not exist
livewire.NewOpenAITTS: Go factory constructor for a port-only struct; Python equivalent does not exist
livewire.NewToolError: Go factory constructor for a port-only struct; Python equivalent does not exist
livewire.WithAgentName: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithAllowInterruptions: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithDescription: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithInferenceLLMModel: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithInferenceSTTModel: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithLLM: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithMCPServers: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithMaxEndpointingDelay: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithMaxToolSteps: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithMinEndpointingDelay: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithMinInterruptionDuration: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithOnRequest: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithOnSessionEnd: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithParameters: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithPreemptiveGeneration: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithRecord: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithReplyInstructions: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithRoom: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithSTT: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithServerType: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithSessionMCPServers: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithSessionTools: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithSessionUserdata: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithTTS: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithTools: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithTurnDetection: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithUserdata: Go functional-options helper; encodes a Python kwarg for the matching constructor
livewire.WithVAD: Go functional-options helper; encodes a Python kwarg for the matching constructor
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
namespaces.ResetDeprecationWarnOnce: Go-only public function; no direct Python counterpart
namespaces.SetDeprecationLogger: Go-only public function; no direct Python counterpart
prefabs.NewBedrockAgent: Go factory constructor for a port-only struct; Python equivalent does not exist
prefabs.NewSurveyQuestion: Go factory constructor for a port-only struct; Python equivalent does not exist
prefabs.WithOptional: Go functional-options helper; encodes a Python kwarg for the matching constructor
prefabs.WithQuestionChoices: Go functional-options helper; encodes a Python kwarg for the matching constructor
prefabs.WithQuestionID: Go functional-options helper; encodes a Python kwarg for the matching constructor
prefabs.WithQuestionScale: Go functional-options helper; encodes a Python kwarg for the matching constructor
prefabs.WithQuestionType: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.NewAIEvent: Go factory constructor for a port-only struct; Python equivalent does not exist
relay.NewRelayClient: Go factory constructor for a port-only struct; Python equivalent does not exist
relay.NewRelayError: Go factory constructor for a port-only struct; Python equivalent does not exist
relay.WithAIEngine: Go functional-options helper; encodes a Python kwarg for the matching constructor
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
relay.WithConferenceDeaf: Go functional-options helper; encodes a Python kwarg for the matching constructor
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
relay.WithStreamDirection: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithTTSGender: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithTTSLanguage: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithTTSVoice: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithTTSVolume: Go functional-options helper; encodes a Python kwarg for the matching constructor
relay.WithToken: Go functional-options helper; encodes a Python kwarg for the matching constructor
rest.NewCrudResourcePUT: Go factory constructor for a port-only struct; Python equivalent does not exist
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

# --- Go-only fields on livewire context structs (LiveKit-style typed handles) ---
signalwire.livewire.AgentHandoff.agent: Go's AgentHandoff embeds a typed ``Agent *Agent`` reference; Python's AgentHandoff is an empty stub class
signalwire.livewire.JobContext.proc: Go's JobContext embeds a typed ``Proc *JobProcess`` reference; Python's JobContext is an empty stub class
signalwire.livewire.JobContext.room: Go's JobContext embeds a typed ``Room *Room`` reference; Python's JobContext is an empty stub class
signalwire.livewire.RunContext.agent: Go's RunContext embeds a typed ``Agent *Agent`` reference; Python's RunContext is an empty stub class
signalwire.livewire.RunContext.session: Go's RunContext embeds a typed ``Session *AgentSession`` reference; Python's RunContext is an empty stub class

# --- Go-only fields on REST base resources (Python uses dynamic attribute lookup) ---
signalwire.rest._base.BaseResource.http: Go's namespaces.Resource exposes a public ``http`` HTTPClient field; Python uses dynamic attribute lookup via __init__
signalwire.rest._base.CrudResource.client: Go's namespaces.CrudResource exposes a public ``client`` HTTPClient field; Python uses dynamic attribute lookup via __init__

# --- Go projections of Python attributes the Python adapter drops from surface but keeps in signatures ---
# Python's enumerate-surface omits these as instance properties; signatures keeps them.
# Go projects them via the StructTable rename map so the signature audit aligns; surface side excused here.
signalwire.core.agent_base.AgentBase.pom: Go's Pom() method projects to Python's pom property; Python's signatures index includes it but the surface index drops it as an instance attribute
signalwire.core.swml_service.SWMLService.schema_utils: Go's SchemaUtils field projects to Python's schema_utils property; Python's signatures index includes it but the surface index drops it as an instance attribute
signalwire.relay.call.Action.result: Go's Result() method projects to Python's result property; Python's signatures index includes it but the surface index drops it as an instance attribute
