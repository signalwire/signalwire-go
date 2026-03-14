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

	"github.com/signalwire/signalwire-agents-go/pkg/agent"
	"github.com/signalwire/signalwire-agents-go/pkg/logging"
	"github.com/signalwire/signalwire-agents-go/pkg/swaig"
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

// ---------------------------------------------------------------------------
// RunContext
// ---------------------------------------------------------------------------

// RunContext mirrors a LiveKit RunContext — available inside tool handlers.
type RunContext struct {
	Session  *AgentSession
	Agent    *Agent
	Userdata any
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

// NewAgentSession creates a new AgentSession with the given options.
func NewAgentSession(opts ...SessionOption) *AgentSession {
	s := &AgentSession{
		logger:         logging.New("LiveWire"),
		allowInterrupt: true,
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

// Start binds the session to an agent and prepares the underlying SignalWire
// AgentBase for serving.
func (s *AgentSession) Start(ctx *JobContext, ag *Agent) error {
	s.currentAgent = ag

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

	// Register all tools from the agent
	for _, td := range ag.tools {
		def := agent.ToolDefinition{
			Name:        td.name,
			Description: td.description,
			Parameters:  td.parameters,
			Handler:     td.handler,
		}
		s.swAgent.DefineTool(def)
	}

	// Bind to JobContext
	if ctx != nil {
		ctx.agent = s.swAgent
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
	agent *agent.AgentBase
}

// Connect is a LiveKit compatibility noop — SignalWire agents connect
// automatically when the platform invokes the SWML endpoint.
func (j *JobContext) Connect() error {
	getGlobalNoop().once("connect", "JobContext.Connect(): SignalWire's control plane handles connection lifecycle at scale automatically")
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
type AgentHandoff struct {
	Agent *Agent
}

// ---------------------------------------------------------------------------
// StopResponse
// ---------------------------------------------------------------------------

// StopResponse signals that a tool should not trigger another LLM reply.
type StopResponse struct{}
