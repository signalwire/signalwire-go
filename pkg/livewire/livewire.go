// Package livewire provides a LiveKit-compatible API surface that runs on
// SignalWire's platform.  Developers can use familiar LiveKit struct and
// function names — just change the import path to get SignalWire's
// infrastructure handling STT, TTS, VAD, LLM, and call control.
package livewire

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/logging"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// ---------------------------------------------------------------------------
// Banner
// ---------------------------------------------------------------------------

const banner = `
    __    _            _       ___
   / /   (_)   _____  | |     / (_)_______
  / /   / / | / / _ \ | | /| / / / ___/ _ \
 / /___/ /| |/ /  __/ | |/ |/ / / /  /  __/
/_____/_/ |___/\___/  |__/|__/_/_/   \___/

 LiveKit-compatible agents powered by SignalWire
`

// printBanner writes the ASCII banner to stderr, using ANSI cyan if a terminal
// is detected (TERM env var is non-empty).
func printBanner() {
	if os.Getenv("TERM") != "" {
		fmt.Fprintf(os.Stderr, "\033[36m%s\033[0m\n", banner)
	} else {
		fmt.Fprintf(os.Stderr, "%s\n", banner)
	}
}

// ---------------------------------------------------------------------------
// "Did You Know?" Tips
// ---------------------------------------------------------------------------

var tips = []string{
	"SignalWire agents support DataMap tools that execute server-side — no webhook infrastructure needed. See: docs/datamap_guide.md",
	"SignalWire Contexts & Steps give you mechanical state control over conversations — no prompt engineering needed. See: docs/contexts_guide.md",
	"SignalWire agents can transfer calls between agents with a single SwmlTransfer() action",
	"SignalWire handles 18 built-in skills (datetime, math, web search, etc.) with one-liner integration via agent.AddSkill()",
	"SignalWire agents support SMS, conferencing, call recording, and SIP — all from the same agent",
	"Your agent's entire AI pipeline (STT, LLM, TTS, VAD) runs in SignalWire's cloud — zero infrastructure to manage",
	"SignalWire prefab agents (Survey, Receptionist, FAQ, Concierge) give you production patterns in 10 lines of code",
	"SignalWire's RELAY client gives you real-time WebSocket call control with 57+ methods — play, record, detect, conference, and more",
	"SignalWire agents auto-generate SWML documents — the platform handles media, turn detection, and barge-in for you",
	"You can host multiple agents on one server with AgentServer — each with its own route, prompt, and tools",
}

func printTip() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	tip := tips[r.Intn(len(tips))]
	fmt.Fprintf(os.Stderr, "\n\U0001f4a1 Did you know?  %s\n\n", tip)
}

// ---------------------------------------------------------------------------
// Noop logging helpers
// ---------------------------------------------------------------------------

// noopTracker ensures each noop message is only printed once.
type noopTracker struct {
	mu     sync.Mutex
	logged map[string]bool
	logger *logging.Logger
}

func newNoopTracker(logger *logging.Logger) *noopTracker {
	return &noopTracker{
		logged: make(map[string]bool),
		logger: logger,
	}
}

func (t *noopTracker) once(key, message string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.logged[key] {
		return
	}
	t.logged[key] = true
	t.logger.Info("[LiveWire] %s", message)
}

// global noop tracker (used before a session exists)
var (
	globalNoopOnce sync.Once
	globalNoop     *noopTracker
)

func getGlobalNoop() *noopTracker {
	globalNoopOnce.Do(func() {
		globalNoop = newNoopTracker(logging.New("LiveWire"))
	})
	return globalNoop
}

// ---------------------------------------------------------------------------
// toolDef — internal tool representation
// ---------------------------------------------------------------------------

type toolDef struct {
	name        string
	description string
	parameters  map[string]any
	handler     func(args map[string]any, rawData map[string]any) *swaig.FunctionResult
}

// ---------------------------------------------------------------------------
// Agent
// ---------------------------------------------------------------------------

// Agent mirrors a LiveKit Agent — it holds instructions and tool definitions.
type Agent struct {
	instructions string
	tools        []toolDef
	userdata     any

	// session is set by AgentSession.Start — mirrors Python Agent._session.
	session *AgentSession

	// Lifecycle callback hooks — set via OnEnter, OnExit, OnUserTurnCompleted.
	onEnterFn        func()
	onExitFn         func()
	onUserTurnFn     func(turnCtx any, newMessage any)
}

// AgentOption configures an Agent during construction.
type AgentOption func(*Agent)

// WithTools is a LiveKit-compatible noop — use FunctionTool to register tools.
func WithTools(tools ...any) AgentOption {
	return func(a *Agent) {
		getGlobalNoop().once("WithTools", "WithTools(): use FunctionTool() to register tools on SignalWire agents")
	}
}

// WithUserdata attaches arbitrary user data to the agent.
func WithUserdata(data any) AgentOption {
	return func(a *Agent) { a.userdata = data }
}

// WithMCPServers is a LiveKit-compatible noop — MCP servers are not yet
// supported in LiveWire.  Tools should be registered via FunctionTool.
// Mirrors Python Agent(mcp_servers=...) which emits a one-time noop warning.
func WithMCPServers(servers ...any) AgentOption {
	return func(a *Agent) {
		getGlobalNoop().once("WithMCPServers", "WithMCPServers(): MCP servers are not yet supported in LiveWire — register tools via FunctionTool")
	}
}

// NewAgent creates a new Agent with the given instructions and options.
func NewAgent(instructions string, opts ...AgentOption) *Agent {
	a := &Agent{
		instructions: instructions,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// ToolOption configures a tool definition.
type ToolOption func(*toolDef)

// WithDescription sets the tool description.
func WithDescription(desc string) ToolOption {
	return func(t *toolDef) { t.description = desc }
}

// WithParameters sets explicit JSON-Schema parameters for a tool.
func WithParameters(params map[string]any) ToolOption {
	return func(t *toolDef) { t.parameters = params }
}

// FunctionTool registers a named tool on the agent.  The handler must be
//
//	func(args map[string]any, rawData map[string]any) *swaig.FunctionResult
//
// or a LiveKit-style handler that will be wrapped:
//
//	func(ctx *RunContext, location string) string
//
// In the LiveKit-style case the function's string parameters are inferred and
// the return string is wrapped into a FunctionResult automatically.
func (a *Agent) FunctionTool(name string, handler any, opts ...ToolOption) *Agent {
	td := toolDef{name: name}
	for _, opt := range opts {
		opt(&td)
	}

	switch h := handler.(type) {
	case func(args map[string]any, rawData map[string]any) *swaig.FunctionResult:
		td.handler = h
	default:
		// Wrap LiveKit-style handlers that accept (ctx, args...) string.
		// We store a generic wrapper that just returns the description as a
		// placeholder — real LiveKit-style reflection is out of scope; users
		// should prefer the canonical handler signature.
		td.handler = func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			// Best-effort: call string-returning handlers with a simple adapter
			if fn, ok := handler.(func(*RunContext, string) string); ok {
				// Pull the first string arg
				var arg string
				for _, v := range args {
					if s, ok := v.(string); ok {
						arg = s
						break
					}
				}
				result := fn(&RunContext{Userdata: a.userdata, Agent: a}, arg)
				return swaig.NewFunctionResult(result)
			}
			if fn, ok := handler.(func(*RunContext) string); ok {
				result := fn(&RunContext{Userdata: a.userdata, Agent: a})
				return swaig.NewFunctionResult(result)
			}
			if fn, ok := handler.(func(map[string]any) string); ok {
				result := fn(args)
				return swaig.NewFunctionResult(result)
			}
			return swaig.NewFunctionResult("handler type not supported")
		}
	}

	a.tools = append(a.tools, td)
	return a
}

// Instructions returns the agent's current instructions string.
// Mirrors Python Agent.instructions (public read/write attribute, line 290).
func (a *Agent) Instructions() string {
	return a.instructions
}

// Session returns the AgentSession currently bound to this agent, or nil if
// the agent has not been started.
// Mirrors Python Agent.session property (lines 334–340).
func (a *Agent) Session() *AgentSession {
	return a.session
}

// OnEnter registers a callback to be invoked when the agent enters a session.
// Mirrors Python Agent.on_enter lifecycle hook (line 346).
// Returns the Agent for method chaining.
func (a *Agent) OnEnter(fn func()) *Agent {
	a.onEnterFn = fn
	return a
}

// OnExit registers a callback to be invoked when the agent exits a session.
// Mirrors Python Agent.on_exit lifecycle hook (line 350).
// Returns the Agent for method chaining.
func (a *Agent) OnExit(fn func()) *Agent {
	a.onExitFn = fn
	return a
}

// OnUserTurnCompleted registers a callback invoked when the user finishes
// speaking.  The two arguments mirror Python's turn_ctx and new_message
// parameters (line 354), typed as any to avoid a LiveKit dependency.
// Returns the Agent for method chaining.
func (a *Agent) OnUserTurnCompleted(fn func(turnCtx any, newMessage any)) *Agent {
	a.onUserTurnFn = fn
	return a
}

// UpdateTools replaces the agent's tool list. Mirrors Python
// Agent.update_tools (livewire/__init__.py:394) which accepts List[Any]
// and stores self._tools = list(tools). In Go, the parameter is []any
// to keep the unexported toolDef out of the public signature (the
// original typed []toolDef made the method uncallable from external
// packages). Elements that aren't recognized tools are silently
// skipped, matching Python's permissive storage semantics.
// Returns the Agent for method chaining.
func (a *Agent) UpdateTools(tools []any) *Agent {
	a.tools = coerceTools(tools)
	return a
}

// coerceTools narrows a []any to []toolDef by extracting recognized
// internal tool representations. Anything else is silently dropped —
// matches Python's behavior where non-@function_tool callables in the
// list are ignored by _register_function_tool's _livewire_tool filter
// (livewire/__init__.py:592-594).
func coerceTools(in []any) []toolDef {
	out := make([]toolDef, 0, len(in))
	for _, t := range in {
		switch td := t.(type) {
		case toolDef:
			out = append(out, td)
		case *toolDef:
			if td != nil {
				out = append(out, *td)
			}
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// RunContext
// ---------------------------------------------------------------------------

// RunContext mirrors a LiveKit RunContext — available inside tool handlers.
type RunContext struct {
	Session      *AgentSession
	Agent        *Agent
	Userdata     any
	SpeechHandle any
	FunctionCall any
}

// ---------------------------------------------------------------------------
// AgentSession
// ---------------------------------------------------------------------------

// AgentSession mirrors a LiveKit AgentSession — it binds an Agent to the
// SignalWire platform and manages the call lifecycle.
type AgentSession struct {
	stt            string
	tts            string
	llm            string
	vad            string
	turnDetection  string
	allowInterrupt bool
	minEndpointing float64
	maxEndpointing float64
	maxToolSteps   int

	// Session-level tools — merged with agent tools in Start().
	// Mirrors Python AgentSession._tools (line 459).
	sessionTools []toolDef

	// mcpServers stores the value passed to WithSessionMCPServers.
	// No-op on SignalWire; stored for parity with Python AgentSession._mcp_servers.
	mcpServers any

	// userdata holds arbitrary caller data at the session level.
	// Mirrors Python AgentSession._userdata (line 460).
	userdata any

	// minInterruptionDuration mirrors Python AgentSession._min_interruption_duration.
	// No-op on SignalWire; barge-in is handled automatically.
	minInterruptionDuration float64

	// preemptiveGeneration mirrors Python AgentSession._preemptive_generation.
	// No-op on SignalWire.
	preemptiveGeneration bool

	// history stores the conversation turn history.
	// Mirrors Python AgentSession._history (line 480).
	history []map[string]string

	// internal
	swAgent      *agent.AgentBase
	currentAgent *Agent
	logger       *logging.Logger
	noop         *noopTracker

	// queued text to say
	sayQueue []string
}

// SessionOption configures an AgentSession.
type SessionOption func(*AgentSession)

// WithSTT sets the STT provider — noop on SignalWire (handled by the control plane).
func WithSTT(provider string) SessionOption {
	return func(s *AgentSession) {
		s.stt = provider
		s.logNoop("stt", fmt.Sprintf(`WithSTT(%q): SignalWire's control plane handles speech recognition at scale — no configuration needed`, provider))
	}
}

// WithTTS sets the TTS provider — noop on SignalWire (voice can be configured via languages).
func WithTTS(provider string) SessionOption {
	return func(s *AgentSession) {
		s.tts = provider
		s.logNoop("tts", fmt.Sprintf(`WithTTS(%q): SignalWire's control plane handles text-to-speech at scale — no configuration needed`, provider))
	}
}

// WithLLM sets the LLM model — this maps to SignalWire AI params.
func WithLLM(model string) SessionOption {
	return func(s *AgentSession) {
		s.llm = model
	}
}

// WithVAD sets the VAD provider — noop on SignalWire (handled by the control plane).
func WithVAD(vad any) SessionOption {
	return func(s *AgentSession) {
		s.vad = fmt.Sprintf("%v", vad)
		s.logNoop("vad", "WithVAD(): SignalWire's control plane handles voice activity detection at scale automatically")
	}
}

// WithTurnDetection sets the turn detection mode — noop on SignalWire.
func WithTurnDetection(mode string) SessionOption {
	return func(s *AgentSession) {
		s.turnDetection = mode
		s.logNoop("turn_detection", fmt.Sprintf(`WithTurnDetection(%q): SignalWire's control plane handles turn detection at scale automatically`, mode))
	}
}

// WithAllowInterruptions maps to barge configuration on SignalWire.
func WithAllowInterruptions(allow bool) SessionOption {
	return func(s *AgentSession) {
		s.allowInterrupt = allow
	}
}

// WithMinEndpointingDelay maps to end_of_speech_timeout on SignalWire.
func WithMinEndpointingDelay(d float64) SessionOption {
	return func(s *AgentSession) {
		s.minEndpointing = d
	}
}

// WithMaxEndpointingDelay maps to AI params on SignalWire.
func WithMaxEndpointingDelay(d float64) SessionOption {
	return func(s *AgentSession) {
		s.maxEndpointing = d
	}
}

// WithMaxToolSteps sets the maximum tool call chain depth — noop on SignalWire.
func WithMaxToolSteps(n int) SessionOption {
	return func(s *AgentSession) {
		s.maxToolSteps = n
		s.logNoop("max_tool_steps", fmt.Sprintf("WithMaxToolSteps(%d): SignalWire's control plane handles tool execution depth at scale automatically", n))
	}
}

// WithSessionTools appends session-level tools to the AgentSession.  These are
// merged with the bound Agent's tools in Start().
// Mirrors Python AgentSession(tools=...) which stores list(tools or []) on
// self._tools (line 459) and merges them in _build_sw_agent() (line 591).
func WithSessionTools(tools ...any) SessionOption {
	return func(s *AgentSession) {
		s.sessionTools = append(s.sessionTools, coerceTools(tools)...)
	}
}

// WithSessionMCPServers stores the MCP servers value on the session — noop on
// SignalWire.  Mirrors Python AgentSession(mcp_servers=...) which emits a
// one-time noop warning (lines 450–456).
func WithSessionMCPServers(servers any) SessionOption {
	return func(s *AgentSession) {
		s.mcpServers = servers
		s.logNoop("mcp_servers", "WithSessionMCPServers(): MCP servers are not yet supported in LiveWire — register tools via FunctionTool")
	}
}

// WithSessionUserdata attaches arbitrary user data to the session.
// Mirrors Python AgentSession(userdata=...) which stores the value as
// self._userdata (line 460).
func WithSessionUserdata(data any) SessionOption {
	return func(s *AgentSession) { s.userdata = data }
}

// WithMinInterruptionDuration sets the minimum interruption duration — noop on
// SignalWire where barge-in timing is handled automatically.
// Mirrors Python AgentSession(min_interruption_duration=0.5) (line 419).
func WithMinInterruptionDuration(d float64) SessionOption {
	return func(s *AgentSession) {
		s.minInterruptionDuration = d
		s.logNoop("min_interruption_duration", fmt.Sprintf("WithMinInterruptionDuration(%g): SignalWire's control plane handles barge-in timing automatically", d))
	}
}

// WithPreemptiveGeneration enables or disables preemptive generation — noop on
// SignalWire.  Mirrors Python AgentSession(preemptive_generation=False) (line 423).
func WithPreemptiveGeneration(enabled bool) SessionOption {
	return func(s *AgentSession) {
		s.preemptiveGeneration = enabled
		s.logNoop("preemptive_generation", fmt.Sprintf("WithPreemptiveGeneration(%v): SignalWire's control plane handles generation pipelining automatically", enabled))
	}
}

// NewAgentSession creates a new AgentSession with the given options.
// Mirrors Python AgentSession.__init__ which initializes _history to [] (line 480)
// and _userdata to {} when not provided (line 460).
func NewAgentSession(opts ...SessionOption) *AgentSession {
	s := &AgentSession{
		logger:                  logging.New("LiveWire"),
		allowInterrupt:          true,
		minInterruptionDuration: 0.5,
		history:                 []map[string]string{},
	}
	s.noop = newNoopTracker(s.logger)
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// logNoop logs a noop message once for the given feature key.
func (s *AgentSession) logNoop(key, message string) {
	if s.noop != nil {
		s.noop.once(key, message)
	} else {
		getGlobalNoop().once(key, message)
	}
}

// Userdata returns the session-level userdata.
// Mirrors Python AgentSession.userdata property getter (line 489).
func (s *AgentSession) Userdata() any {
	return s.userdata
}

// SetUserdata sets the session-level userdata.
// Mirrors Python AgentSession.userdata property setter (line 493).
func (s *AgentSession) SetUserdata(val any) {
	s.userdata = val
}

// History returns the conversation turn history (read-only).
// Mirrors Python AgentSession.history property (line 497).
func (s *AgentSession) History() []map[string]string {
	return s.history
}

// UpdateAgent swaps in a new Agent mid-session.
// Mirrors Python AgentSession.update_agent (line 528) which sets
// self._agent = agent and agent.session = self.
func (s *AgentSession) UpdateAgent(ag *Agent) {
	s.currentAgent = ag
	ag.session = s
}

// startConfig holds options for Start().
type startConfig struct {
	room   *Room
	record bool
}

// StartOption configures a Start() call.
type StartOption func(*startConfig)

// WithRoom sets the room for the session start.
// Mirrors Python AgentSession.start(room=...) keyword-only param (line 504).
func WithRoom(room *Room) StartOption {
	return func(c *startConfig) { c.room = room }
}

// WithRecord enables call recording for the session.
// Mirrors Python AgentSession.start(record=False) keyword-only param (line 504).
func WithRecord(record bool) StartOption {
	return func(c *startConfig) { c.record = record }
}

// Start binds the session to an agent and prepares the underlying SignalWire
// AgentBase for serving.
//
// Mirrors Python AgentSession.start(agent, *, room=None, record=False) (line 504).
// The room and record parameters are accepted via StartOption functional options
// (WithRoom and WithRecord) following Go idioms.
func (s *AgentSession) Start(ctx *JobContext, ag *Agent, opts ...StartOption) error {
	// Apply start options (room, record).
	cfg := &startConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	s.currentAgent = ag

	// Bind the back-reference so agent.Session() works.
	// Mirrors Python AgentSession.start(): agent.session = self (line 507).
	ag.session = s

	// Build a real SignalWire agent
	agOpts := []agent.AgentOption{
		agent.WithName("LiveWireAgent"),
		agent.WithRoute("/"),
	}
	s.swAgent = agent.NewAgentBase(agOpts...)

	// Set prompt from agent instructions
	s.swAgent.SetPromptText(ag.instructions)

	// Map LLM model if provided
	if s.llm != "" {
		// Strip provider prefix if present (e.g. "openai/gpt-4" -> model param)
		model := s.llm
		if idx := strings.Index(model, "/"); idx >= 0 {
			model = model[idx+1:]
		}
		s.swAgent.SetParam("model", model)
	}

	// Map barge / interruption settings
	if !s.allowInterrupt {
		s.swAgent.SetParam("barge_confidence", 1.0)
	}

	// Map endpointing delays
	if s.minEndpointing > 0 {
		s.swAgent.SetParam("end_of_speech_timeout", int(s.minEndpointing*1000))
	}
	if s.maxEndpointing > 0 {
		s.swAgent.SetParam("attention_timeout", int(s.maxEndpointing*1000))
	}

	// Register session-level tools first, then agent tools.
	// Mirrors Python _build_sw_agent(): all_tools = list(self._tools) + list(agent._tools) (line 591).
	allTools := append(s.sessionTools, ag.tools...)
	for _, td := range allTools {
		def := agent.ToolDefinition{
			Name:        td.name,
			Description: td.description,
			Parameters:  td.parameters,
			Handler:     td.handler,
		}
		s.swAgent.DefineTool(def)
	}

	// Log noop for room/record params if supplied — these have no direct
	// SignalWire equivalent (barge and recording are handled by the platform).
	if cfg.room != nil {
		s.logNoop("start_room", "Start(WithRoom(...)): SignalWire's control plane manages media rooms at scale automatically")
	}
	if cfg.record {
		s.logNoop("start_record", "Start(WithRecord(true)): call recording is configured via the SignalWire control plane")
	}

	// Bind to JobContext
	if ctx != nil {
		ctx.agent = s.swAgent
	}

	// Fire the on_enter lifecycle hook if registered.
	// Mirrors Python AgentSession.start() which calls agent.on_enter() (implicitly
	// via the session binding) — in Go the callback is invoked synchronously here.
	if ag.onEnterFn != nil {
		ag.onEnterFn()
	}

	return nil
}

// Say queues text to be spoken by the agent.
func (s *AgentSession) Say(text string) {
	s.sayQueue = append(s.sayQueue, text)
}

// replyConfig holds configuration for GenerateReply.
type replyConfig struct {
	instructions string
}

// ReplyOption configures a GenerateReply call.
type ReplyOption func(*replyConfig)

// WithReplyInstructions sets the instructions for the generated reply.
func WithReplyInstructions(inst string) ReplyOption {
	return func(c *replyConfig) { c.instructions = inst }
}

// GenerateReply triggers the agent to speak. On SignalWire this is handled
// by the prompt; reply instructions are appended to the prompt.
func (s *AgentSession) GenerateReply(opts ...ReplyOption) {
	cfg := &replyConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.instructions != "" && s.swAgent != nil {
		// Append the reply instructions to the prompt
		s.swAgent.PromptAddSection("Initial Greeting", cfg.instructions, nil)
	}
}

// Interrupt interrupts current speech — noop on SignalWire (barge-in is automatic).
func (s *AgentSession) Interrupt() {
	s.logNoop("interrupt", "Interrupt(): SignalWire handles barge-in automatically via its control plane")
}

// UpdateInstructions changes the agent's prompt mid-session.
func (s *AgentSession) UpdateInstructions(instructions string) {
	if s.swAgent != nil {
		s.swAgent.SetPromptText(instructions)
	}
	if s.currentAgent != nil {
		s.currentAgent.instructions = instructions
	}
}

// GetSwAgent returns the underlying SignalWire AgentBase (for testing/advanced use).
func (s *AgentSession) GetSwAgent() *agent.AgentBase {
	return s.swAgent
}

// ---------------------------------------------------------------------------
// JobContext
// ---------------------------------------------------------------------------

// JobContext mirrors a LiveKit JobContext — provides room and connection info.
type JobContext struct {
	Room  *Room
	Proc  *JobProcess
	agent *agent.AgentBase
}

// Connect is a LiveKit compatibility noop — SignalWire agents connect
// automatically when the platform invokes the SWML endpoint.
func (j *JobContext) Connect() error {
	getGlobalNoop().once("connect", "JobContext.Connect(): SignalWire's control plane handles connection lifecycle at scale automatically")
	return nil
}

// WaitForParticipant is a LiveKit compatibility noop — SignalWire handles
// participant management automatically.
// Mirrors Python JobContext.wait_for_participant(*, identity=None) (line 670).
func (j *JobContext) WaitForParticipant(identity string) error {
	getGlobalNoop().once("wait_for_participant", "JobContext.WaitForParticipant(): SignalWire's control plane handles participant management automatically")
	return nil
}

// ---------------------------------------------------------------------------
// Room
// ---------------------------------------------------------------------------

// Room is a stub — SignalWire doesn't use the LiveKit room abstraction.
type Room struct {
	Name string
}

// ---------------------------------------------------------------------------
// LiveServer (mirrors LiveKit AgentServer)
// ---------------------------------------------------------------------------

// LiveServer mirrors a LiveKit AgentServer — it registers entrypoints and
// starts the agent.
type LiveServer struct {
	setupFunc  func(*JobProcess)
	entrypoint func(*JobContext)
	agentName  string
	logger     *logging.Logger
}

// NewAgentServer creates a new LiveServer.
func NewAgentServer() *LiveServer {
	return &LiveServer{
		logger: logging.New("LiveWire"),
	}
}

// rtcConfig holds RTC session options.
type rtcConfig struct {
	agentName  string
	serverType string
}

// RTCOption configures an RTC session.
type RTCOption func(*rtcConfig)

// WithAgentName sets the agent name for the RTC session.
func WithAgentName(name string) RTCOption {
	return func(c *rtcConfig) { c.agentName = name }
}

// WithServerType sets the server type ("room" or "publisher") — noop on SignalWire.
func WithServerType(t string) RTCOption {
	return func(c *rtcConfig) {
		c.serverType = t
		getGlobalNoop().once("server_type", fmt.Sprintf(`WithServerType(%q): SignalWire's control plane handles server topology at scale automatically`, t))
	}
}

// WithOnRequest accepts a request callback — noop on SignalWire.
// Mirrors Python AgentServer.rtc_session(on_request=...) which silently ignores
// the parameter for LiveKit portability.
func WithOnRequest(fn func()) RTCOption {
	return func(c *rtcConfig) {
		getGlobalNoop().once("on_request", "WithOnRequest(): SignalWire's control plane handles request routing automatically")
	}
}

// WithOnSessionEnd accepts a session-end callback — noop on SignalWire.
// Mirrors Python AgentServer.rtc_session(on_session_end=...) which silently
// ignores the parameter for LiveKit portability.
func WithOnSessionEnd(fn func()) RTCOption {
	return func(c *rtcConfig) {
		getGlobalNoop().once("on_session_end", "WithOnSessionEnd(): SignalWire's control plane handles session lifecycle automatically")
	}
}

// SetSetupFunc sets the prewarm/setup function — noop on SignalWire.
func (s *LiveServer) SetSetupFunc(fn func(*JobProcess)) {
	s.setupFunc = fn
	getGlobalNoop().once("setup_func", "SetupFunc: Warm process pools not needed — SignalWire's control plane manages media infrastructure at scale")
}

// RTCSession registers the session entrypoint function.
func (s *LiveServer) RTCSession(fn func(*JobContext), opts ...RTCOption) {
	cfg := &rtcConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.agentName != "" {
		s.agentName = cfg.agentName
	}
	s.entrypoint = fn
}

// ---------------------------------------------------------------------------
// JobProcess
// ---------------------------------------------------------------------------

// JobProcess mirrors a LiveKit JobProcess — used for prewarm/setup.
type JobProcess struct {
	Userdata map[string]any
}

// ---------------------------------------------------------------------------
// RunApp
// ---------------------------------------------------------------------------

// RunApp starts the LiveWire agent — prints the banner, a random tip, invokes
// the setup function (if any), calls the entrypoint, and starts the
// underlying SignalWire agent server.
func RunApp(server *LiveServer) {
	printBanner()

	// Run setup if registered
	if server.setupFunc != nil {
		proc := &JobProcess{Userdata: make(map[string]any)}
		server.setupFunc(proc)
	}

	// Create a JobContext
	ctx := &JobContext{
		Room: &Room{Name: "livewire-room"},
		Proc: &JobProcess{Userdata: make(map[string]any)},
	}

	// Call the entrypoint — this should create session, agent, tools, etc.
	if server.entrypoint != nil {
		server.entrypoint(ctx)
	}

	// Print a random tip right before starting
	printTip()

	// Start the underlying SignalWire agent
	if ctx.agent != nil {
		if err := ctx.agent.Run(); err != nil {
			server.logger.Error("agent error: %s", err)
		}
	} else {
		server.logger.Error("no agent was started — did you call session.Start()?")
	}
}

// ---------------------------------------------------------------------------
// AgentHandoff
// ---------------------------------------------------------------------------

// AgentHandoff signals a handoff to another agent in multi-agent scenarios.
// Mirrors Python AgentHandoff(agent, *, returns=None) (line 153).
type AgentHandoff struct {
	Agent   *Agent
	Returns any
}

// ---------------------------------------------------------------------------
// StopResponse
// ---------------------------------------------------------------------------

// StopResponse signals that a tool should not trigger another LLM reply.
type StopResponse struct{}

// ---------------------------------------------------------------------------
// ToolError
// ---------------------------------------------------------------------------

// ToolError signals a tool execution error. Return a *ToolError from a tool
// handler to tell the framework the tool failed; the error message is forwarded
// to the LLM as a tool-failure notification rather than triggering a normal
// LLM reply. Parallel to StopResponse in this file.
type ToolError struct {
	Message string
}

// Error implements the built-in error interface.
func (e *ToolError) Error() string { return e.Message }

// NewToolError constructs a ToolError with the given message.
func NewToolError(message string) *ToolError { return &ToolError{Message: message} }

// ---------------------------------------------------------------------------
// ChatContext
// ---------------------------------------------------------------------------

// ChatMessage holds a single role/content pair in a conversation history.
// The JSON tags match the dict keys produced by the Python ChatContext.append()
// implementation: {"role": ..., "content": ...}.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatContext buffers a conversation as an ordered list of role/content
// messages.  It mirrors the Python livewire.ChatContext stub which is
// API-compatible with the livekit-agents ChatContext shape.
type ChatContext struct {
	Messages []ChatMessage
}

// NewChatContext returns an empty ChatContext ready for use.
func NewChatContext() *ChatContext { return &ChatContext{} }

// Append adds a role/content message to the context and returns the receiver
// for method chaining.  If role is empty it defaults to "user"; if content is
// empty it defaults to "" (empty string), matching the Python defaults
// role="user", text="".
func (c *ChatContext) Append(role, content string) *ChatContext {
	if role == "" {
		role = "user"
	}
	c.Messages = append(c.Messages, ChatMessage{Role: role, Content: content})
	return c
}

// ---------------------------------------------------------------------------
// InferenceTTS
// ---------------------------------------------------------------------------

// InferenceTTS is a no-op stub providing LiveKit import compatibility.
// SignalWire's control plane handles text-to-speech; this type exists so
// code written for livekit/agents inference.TTS can be dropped in unchanged.
type InferenceTTS struct {
	Model string
}

// NewInferenceTTS creates an InferenceTTS stub with the given model hint.
// The model value is stored for compatibility but is otherwise unused.
func NewInferenceTTS(model string) *InferenceTTS {
	getGlobalNoop().once("inference_tts", fmt.Sprintf("NewInferenceTTS(%q): SignalWire's control plane handles text-to-speech -- inference stubs are no-ops", model))
	return &InferenceTTS{Model: model}
}
