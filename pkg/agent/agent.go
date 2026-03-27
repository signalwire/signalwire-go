// Package agent provides the core AgentBase type that wires together SWML
// rendering, tool dispatch, prompt management, AI configuration, and HTTP
// serving into a single self-contained AI agent.
package agent

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/signalwire/signalwire-go/pkg/contexts"
	"github.com/signalwire/signalwire-go/pkg/logging"
	"github.com/signalwire/signalwire-go/pkg/security"
	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
	"github.com/signalwire/signalwire-go/pkg/swml"
)

// ---------------------------------------------------------------------------
// Type aliases
// ---------------------------------------------------------------------------

// ToolHandler is the signature for SWAIG function handlers.
type ToolHandler func(args map[string]any, rawData map[string]any) *swaig.FunctionResult

// DynamicConfigCallback is invoked on each request to mutate an ephemeral
// agent copy before rendering.  Headers and body params give the callback
// full context about the inbound request.
type DynamicConfigCallback func(queryParams map[string]string, bodyParams map[string]any, headers map[string]string, agent *AgentBase)

// SummaryCallback is called when a post-prompt summary arrives.
type SummaryCallback func(summary map[string]any, rawData map[string]any)

// DebugEventHandler is called for debug events if enabled.
type DebugEventHandler func(event map[string]any)

// ---------------------------------------------------------------------------
// ToolDefinition
// ---------------------------------------------------------------------------

// ToolDefinition describes a single SWAIG tool including its JSON Schema
// parameters and a Go handler function.
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON Schema for arguments
	Handler     ToolHandler
	Secure      bool
	Fillers     map[string][]string
	MetaData    map[string]any
	SwaigFields map[string]any // extra per-function SWAIG fields
}

// ---------------------------------------------------------------------------
// Verb helper (verb-name + config)
// ---------------------------------------------------------------------------

// verb is a pre-answer / post-answer / post-ai verb pair.
type verb struct {
	Name   string
	Config map[string]any
}

// ---------------------------------------------------------------------------
// AgentOption functional options
// ---------------------------------------------------------------------------

// AgentOption configures an AgentBase during construction.
type AgentOption func(*AgentBase)

// WithName sets the agent (and service) name.
func WithName(name string) AgentOption {
	return func(a *AgentBase) { a.name = name }
}

// WithRoute sets the HTTP route path the agent listens on.
func WithRoute(route string) AgentOption {
	return func(a *AgentBase) { a.route = route }
}

// WithHost sets the HTTP listen address.
func WithHost(host string) AgentOption {
	return func(a *AgentBase) { a.host = host }
}

// WithPort sets the HTTP listen port.
func WithPort(port int) AgentOption {
	return func(a *AgentBase) { a.port = port }
}

// WithBasicAuth sets explicit basic-auth credentials.
func WithBasicAuth(user, password string) AgentOption {
	return func(a *AgentBase) {
		a.basicAuthUser = user
		a.basicAuthPassword = password
	}
}

// WithAutoAnswer controls whether the answer verb is emitted automatically.
func WithAutoAnswer(autoAnswer bool) AgentOption {
	return func(a *AgentBase) { a.autoAnswer = autoAnswer }
}

// WithRecordCall enables or disables automatic call recording.
func WithRecordCall(record bool) AgentOption {
	return func(a *AgentBase) { a.recordCall = record }
}

// WithRecordFormat sets the recording format (e.g. "mp4", "wav").
func WithRecordFormat(format string) AgentOption {
	return func(a *AgentBase) { a.recordFormat = format }
}

// WithRecordStereo enables or disables stereo recording.
func WithRecordStereo(stereo bool) AgentOption {
	return func(a *AgentBase) { a.recordStereo = stereo }
}

// WithTokenExpiry sets the token expiry time in seconds for secure tools.
func WithTokenExpiry(secs int) AgentOption {
	return func(a *AgentBase) { a.tokenExpirySecs = secs }
}

// ---------------------------------------------------------------------------
// AgentBase
// ---------------------------------------------------------------------------

// AgentBase is the central agent struct that composes SWML service, tools,
// prompts, AI configuration, context management, and HTTP handling.
type AgentBase struct {
	mu          sync.RWMutex
	swmlService *swml.Service
	Logger      *logging.Logger

	// Construction parameters (forwarded to swml.Service)
	name              string
	route             string
	host              string
	port              int
	basicAuthUser     string
	basicAuthPassword string

	// Prompt management
	promptText  string           // raw text mode
	postPrompt  string
	usePom      bool             // default true
	pomSections []map[string]any // POM sections list

	// Tool management
	tools     map[string]*ToolDefinition // registered tools keyed by name
	toolOrder []string                   // insertion order

	// AI configuration
	hints              []string
	patternHints       []map[string]any
	languages          []map[string]any
	pronunciations     []map[string]any
	params             map[string]any // AI params like temperature
	globalData         map[string]any
	nativeFunctions    []string
	internalFillers    map[string]map[string][]string
	debugEventsLevel   int
	functionIncludes   []map[string]any
	promptLlmParams    map[string]any
	postPromptLlmParams map[string]any

	// Context/Steps
	contextBuilder *contexts.ContextBuilder // nil until DefineContexts called

	// Call flow verbs
	preAnswerVerbs  []verb
	answerConfig    map[string]any
	postAnswerVerbs []verb
	postAiVerbs     []verb
	autoAnswer      bool // default true
	recordCall      bool
	recordFormat    string
	recordStereo    bool

	// Web/HTTP
	dynamicConfigCallback DynamicConfigCallback
	webhookURL            string
	postPromptURL         string
	swaigQueryParams      map[string]string
	proxyURLBase          string

	// Session security
	sessionManager  *security.SessionManager
	tokenExpirySecs int

	// Lifecycle callbacks
	summaryCallback  SummaryCallback
	debugEventHandler DebugEventHandler

	// Skills
	skillManager *skills.SkillManager

	// SIP routing
	sipRoutingEnabled bool
	sipUsernames      map[string]bool

	// MCP integration
	mcpServers        []map[string]any // external MCP server configs
	mcpServerEnabled  bool             // expose /mcp endpoint
}

// NewAgentBase creates a new AgentBase with default values and applies the
// provided functional options.
func NewAgentBase(opts ...AgentOption) *AgentBase {
	a := &AgentBase{
		name:         "Agent",
		route:        "/",
		host:         "0.0.0.0",
		port:         3000,
		usePom:       true,
		autoAnswer:   true,
		recordFormat: "mp4",
		recordStereo: true,
		tokenExpirySecs: 3600,

		// Initialize all maps and slices
		pomSections:        make([]map[string]any, 0),
		tools:              make(map[string]*ToolDefinition),
		toolOrder:          make([]string, 0),
		hints:              make([]string, 0),
		patternHints:       make([]map[string]any, 0),
		languages:          make([]map[string]any, 0),
		pronunciations:     make([]map[string]any, 0),
		params:             make(map[string]any),
		globalData:         make(map[string]any),
		internalFillers:    make(map[string]map[string][]string),
		functionIncludes:   make([]map[string]any, 0),
		promptLlmParams:    make(map[string]any),
		postPromptLlmParams: make(map[string]any),
		answerConfig:       make(map[string]any),
		swaigQueryParams:   make(map[string]string),
		sipUsernames:       make(map[string]bool),
		mcpServers:         make([]map[string]any, 0),
	}

	for _, opt := range opts {
		opt(a)
	}

	// Build swml.Service options from agent config
	svcOpts := []swml.ServiceOption{
		swml.WithName(a.name),
		swml.WithRoute(a.route),
		swml.WithHost(a.host),
		swml.WithPort(a.port),
	}
	if a.basicAuthUser != "" && a.basicAuthPassword != "" {
		svcOpts = append(svcOpts, swml.WithBasicAuth(a.basicAuthUser, a.basicAuthPassword))
	}
	a.swmlService = swml.NewService(svcOpts...)
	a.Logger = logging.New(a.name)

	// Proxy URL from env or service
	if a.proxyURLBase == "" {
		a.proxyURLBase = os.Getenv("SWML_PROXY_URL_BASE")
	}

	// Session manager for secure tools
	a.sessionManager = security.NewSessionManager(a.tokenExpirySecs)

	// Skill manager
	a.skillManager = skills.NewSkillManager()

	return a
}

// ---------------------------------------------------------------------------
// Accessor methods
// ---------------------------------------------------------------------------

// GetRoute returns the agent's configured route path.
func (a *AgentBase) GetRoute() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.route
}

// GetName returns the agent's name.
func (a *AgentBase) GetName() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.name
}

// ---------------------------------------------------------------------------
// Prompt methods
// ---------------------------------------------------------------------------

// SetPromptText sets the agent prompt to raw text, disabling POM mode.
func (a *AgentBase) SetPromptText(text string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.promptText = text
	a.usePom = false
	return a
}

// SetPostPrompt sets the post-prompt text used for conversation summary.
func (a *AgentBase) SetPostPrompt(text string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.postPrompt = text
	return a
}

// SetPromptPom sets the POM sections directly and enables POM mode.
func (a *AgentBase) SetPromptPom(pom []map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pomSections = pom
	a.usePom = true
	return a
}

// PromptAddSection appends a new section to the POM prompt.
func (a *AgentBase) PromptAddSection(title string, body string, bullets []string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()

	section := map[string]any{"title": title}
	if body != "" {
		section["body"] = body
	}
	if len(bullets) > 0 {
		section["bullets"] = bullets
	}
	a.pomSections = append(a.pomSections, section)
	a.usePom = true
	return a
}

// PromptAddToSection finds an existing POM section by title and appends text.
func (a *AgentBase) PromptAddToSection(title string, text string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i, sec := range a.pomSections {
		if sec["title"] == title {
			existing, _ := sec["body"].(string)
			if existing != "" {
				a.pomSections[i]["body"] = existing + "\n" + text
			} else {
				a.pomSections[i]["body"] = text
			}
			return a
		}
	}
	return a
}

// PromptAddSubsection adds a subsection under an existing parent section.
func (a *AgentBase) PromptAddSubsection(parentTitle string, title string, body string, bullets []string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()

	subsection := map[string]any{"title": title}
	if body != "" {
		subsection["body"] = body
	}
	if len(bullets) > 0 {
		subsection["bullets"] = bullets
	}

	for i, sec := range a.pomSections {
		if sec["title"] == parentTitle {
			subs, _ := sec["subsections"].([]map[string]any)
			subs = append(subs, subsection)
			a.pomSections[i]["subsections"] = subs
			return a
		}
	}
	return a
}

// PromptHasSection returns true if a POM section with the given title exists.
func (a *AgentBase) PromptHasSection(title string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, sec := range a.pomSections {
		if sec["title"] == title {
			return true
		}
	}
	return false
}

// GetPrompt returns the current prompt.  If POM mode is active, it returns
// []map[string]any; otherwise it returns the raw string.
func (a *AgentBase) GetPrompt() any {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.usePom {
		result := make([]map[string]any, len(a.pomSections))
		copy(result, a.pomSections)
		return result
	}
	return a.promptText
}

// ---------------------------------------------------------------------------
// Tool methods
// ---------------------------------------------------------------------------

// DefineTool registers a tool (SWAIG function) with the agent.
func (a *AgentBase) DefineTool(def ToolDefinition) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, exists := a.tools[def.Name]; !exists {
		a.toolOrder = append(a.toolOrder, def.Name)
	}
	a.tools[def.Name] = &def
	return a
}

// RegisterSwaigFunction registers a raw SWAIG function definition (e.g. for
// DataMap tools that don't have a Go handler).
func (a *AgentBase) RegisterSwaigFunction(funcDef map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()

	name, _ := funcDef["function"].(string)
	if name == "" {
		return a
	}

	def := &ToolDefinition{
		Name:        name,
		SwaigFields: funcDef,
	}
	if _, exists := a.tools[name]; !exists {
		a.toolOrder = append(a.toolOrder, name)
	}
	a.tools[name] = def
	return a
}

// DefineTools returns all registered tool definitions in insertion order.
func (a *AgentBase) DefineTools() []*ToolDefinition {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]*ToolDefinition, 0, len(a.toolOrder))
	for _, name := range a.toolOrder {
		if t, ok := a.tools[name]; ok {
			result = append(result, t)
		}
	}
	return result
}

// OnFunctionCall dispatches a SWAIG function call to the registered handler.
func (a *AgentBase) OnFunctionCall(name string, args map[string]any, rawData map[string]any) (any, error) {
	a.mu.RLock()
	tool, ok := a.tools[name]
	a.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown tool: %q", name)
	}
	if tool.Handler == nil {
		return nil, fmt.Errorf("tool %q has no handler (may be a DataMap tool)", name)
	}
	result := tool.Handler(args, rawData)
	if result == nil {
		return nil, nil
	}
	return result.ToMap(), nil
}

// ---------------------------------------------------------------------------
// AI Config methods
// ---------------------------------------------------------------------------

// AddHint adds a single speech-recognition hint.
func (a *AgentBase) AddHint(hint string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.hints = append(a.hints, hint)
	return a
}

// AddHints adds multiple speech-recognition hints.
func (a *AgentBase) AddHints(hints []string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.hints = append(a.hints, hints...)
	return a
}

// AddPatternHint adds a pattern-based speech-recognition hint.
func (a *AgentBase) AddPatternHint(pattern string, hint string, language string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	ph := map[string]any{"pattern": pattern, "hint": hint}
	if language != "" {
		ph["language"] = language
	}
	a.patternHints = append(a.patternHints, ph)
	return a
}

// AddLanguage adds a language configuration.
func (a *AgentBase) AddLanguage(config map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.languages = append(a.languages, config)
	return a
}

// SetLanguages replaces all language configurations.
func (a *AgentBase) SetLanguages(languages []map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.languages = languages
	return a
}

// AddPronunciation adds a pronunciation override.
func (a *AgentBase) AddPronunciation(phrase, pronunciation, languageCode string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pronunciations = append(a.pronunciations, map[string]any{
		"replace": phrase,
		"with":    pronunciation,
		"lang":    languageCode,
	})
	return a
}

// SetPronunciations replaces all pronunciation overrides.
func (a *AgentBase) SetPronunciations(p []map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pronunciations = p
	return a
}

// SetParam sets a single AI parameter (e.g. temperature, top_p).
func (a *AgentBase) SetParam(key string, value any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.params[key] = value
	return a
}

// SetParams replaces all AI parameters.
func (a *AgentBase) SetParams(params map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.params = params
	return a
}

// SetGlobalData replaces all global data.
func (a *AgentBase) SetGlobalData(data map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.globalData = data
	return a
}

// UpdateGlobalData merges data into existing global data.
func (a *AgentBase) UpdateGlobalData(data map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	for k, v := range data {
		a.globalData[k] = v
	}
	return a
}

// SetNativeFunctions sets the list of native function names.
func (a *AgentBase) SetNativeFunctions(names []string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.nativeFunctions = names
	return a
}

// SetInternalFillers replaces all internal fillers.
func (a *AgentBase) SetInternalFillers(fillers map[string]map[string][]string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.internalFillers = fillers
	return a
}

// AddInternalFiller adds fillers for a specific function and language.
func (a *AgentBase) AddInternalFiller(funcName, langCode string, fillers []string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.internalFillers[funcName] == nil {
		a.internalFillers[funcName] = make(map[string][]string)
	}
	a.internalFillers[funcName][langCode] = fillers
	return a
}

// EnableDebugEvents sets the debug events level (0 = off).
func (a *AgentBase) EnableDebugEvents(level int) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.debugEventsLevel = level
	return a
}

// AddFunctionInclude adds a remote SWAIG function include.
func (a *AgentBase) AddFunctionInclude(url string, functions []string, metaData map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	include := map[string]any{"url": url}
	if len(functions) > 0 {
		include["functions"] = functions
	}
	if len(metaData) > 0 {
		include["meta_data"] = metaData
	}
	a.functionIncludes = append(a.functionIncludes, include)
	return a
}

// SetFunctionIncludes replaces all function includes.
func (a *AgentBase) SetFunctionIncludes(includes []map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.functionIncludes = includes
	return a
}

// SetPromptLlmParams sets LLM parameters for the main prompt.
func (a *AgentBase) SetPromptLlmParams(params map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.promptLlmParams = params
	return a
}

// SetPostPromptLlmParams sets LLM parameters for the post-prompt.
func (a *AgentBase) SetPostPromptLlmParams(params map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.postPromptLlmParams = params
	return a
}

// ---------------------------------------------------------------------------
// Verb management
// ---------------------------------------------------------------------------

// AddPreAnswerVerb adds a SWML verb to execute before the answer.
func (a *AgentBase) AddPreAnswerVerb(verbName string, config map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.preAnswerVerbs = append(a.preAnswerVerbs, verb{Name: verbName, Config: config})
	return a
}

// AddAnswerVerb configures the answer verb. Merged with defaults at render time.
func (a *AgentBase) AddAnswerVerb(config map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	for k, v := range config {
		a.answerConfig[k] = v
	}
	return a
}

// AddPostAnswerVerb adds a SWML verb to execute after the answer.
func (a *AgentBase) AddPostAnswerVerb(verbName string, config map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.postAnswerVerbs = append(a.postAnswerVerbs, verb{Name: verbName, Config: config})
	return a
}

// AddPostAiVerb adds a SWML verb to execute after the AI verb.
func (a *AgentBase) AddPostAiVerb(verbName string, config map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.postAiVerbs = append(a.postAiVerbs, verb{Name: verbName, Config: config})
	return a
}

// ClearPreAnswerVerbs removes all pre-answer verbs.
func (a *AgentBase) ClearPreAnswerVerbs() *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.preAnswerVerbs = nil
	return a
}

// ClearPostAnswerVerbs removes all post-answer verbs.
func (a *AgentBase) ClearPostAnswerVerbs() *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.postAnswerVerbs = nil
	return a
}

// ClearPostAiVerbs removes all post-AI verbs.
func (a *AgentBase) ClearPostAiVerbs() *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.postAiVerbs = nil
	return a
}

// ---------------------------------------------------------------------------
// Context methods
// ---------------------------------------------------------------------------

// DefineContexts returns the context builder, creating it if needed.
func (a *AgentBase) DefineContexts() *contexts.ContextBuilder {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.contextBuilder == nil {
		a.contextBuilder = contexts.NewContextBuilder()
	}
	return a.contextBuilder
}

// Contexts is an alias for DefineContexts.
func (a *AgentBase) Contexts() *contexts.ContextBuilder {
	return a.DefineContexts()
}

// ---------------------------------------------------------------------------
// Web/HTTP methods
// ---------------------------------------------------------------------------

// SetDynamicConfigCallback sets a callback invoked on each request to allow
// per-request agent customisation.
func (a *AgentBase) SetDynamicConfigCallback(cb DynamicConfigCallback) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.dynamicConfigCallback = cb
	return a
}

// ManualSetProxyUrl overrides the proxy URL base used for webhook URL generation.
func (a *AgentBase) ManualSetProxyUrl(url string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.proxyURLBase = url
	return a
}

// SetWebHookUrl explicitly sets the webhook URL used in SWAIG function defs.
func (a *AgentBase) SetWebHookUrl(url string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.webhookURL = url
	return a
}

// SetPostPromptUrl sets the URL for post-prompt summary delivery.
func (a *AgentBase) SetPostPromptUrl(url string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.postPromptURL = url
	return a
}

// AddSwaigQueryParams adds query parameters that will be appended to SWAIG
// webhook URLs.
func (a *AgentBase) AddSwaigQueryParams(params map[string]string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	for k, v := range params {
		a.swaigQueryParams[k] = v
	}
	return a
}

// ClearSwaigQueryParams removes all SWAIG query parameters.
func (a *AgentBase) ClearSwaigQueryParams() *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.swaigQueryParams = make(map[string]string)
	return a
}

// EnableDebugRoutes is a placeholder for adding debug HTTP routes.
func (a *AgentBase) EnableDebugRoutes() *AgentBase {
	// Future: register /debug routes on the mux
	return a
}

// ---------------------------------------------------------------------------
// MCP methods
// ---------------------------------------------------------------------------

// MCPServerConfig holds configuration for an external MCP server connection.
type MCPServerConfig struct {
	URL          string
	Headers      map[string]string
	Resources    bool
	ResourceVars map[string]string
}

// AddMcpServer adds an external MCP server for tool discovery and invocation.
// Tools are discovered via the MCP protocol at session start and registered as
// SWAIG functions. Returns self for method chaining.
func (a *AgentBase) AddMcpServer(cfg MCPServerConfig) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()

	server := map[string]any{"url": cfg.URL}
	if len(cfg.Headers) > 0 {
		server["headers"] = cfg.Headers
	}
	if cfg.Resources {
		server["resources"] = true
	}
	if len(cfg.ResourceVars) > 0 {
		server["resource_vars"] = cfg.ResourceVars
	}
	a.mcpServers = append(a.mcpServers, server)
	return a
}

// EnableMcpServer exposes this agent's tools as an MCP server endpoint at /mcp.
// The endpoint speaks JSON-RPC 2.0 (MCP protocol) and supports initialize,
// tools/list, tools/call, and ping. Returns self for method chaining.
func (a *AgentBase) EnableMcpServer() *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.mcpServerEnabled = true
	return a
}

// buildMcpToolList converts registered tools to MCP tool format.
func (a *AgentBase) buildMcpToolList() []map[string]any {
	tools := make([]map[string]any, 0)
	for _, name := range a.toolOrder {
		td, ok := a.tools[name]
		if !ok {
			continue
		}
		tool := map[string]any{
			"name":        td.Name,
			"description": td.Description,
		}
		if td.Parameters != nil {
			tool["inputSchema"] = map[string]any{
				"type":       "object",
				"properties": td.Parameters,
			}
		} else {
			tool["inputSchema"] = map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			}
		}
		tools = append(tools, tool)
	}
	return tools
}

// handleMcpRequest processes a single MCP JSON-RPC 2.0 request and returns
// the response map.
func (a *AgentBase) handleMcpRequest(body map[string]any) map[string]any {
	jsonrpc, _ := body["jsonrpc"].(string)
	method, _ := body["method"].(string)
	reqID := body["id"]
	params, _ := body["params"].(map[string]any)
	if params == nil {
		params = make(map[string]any)
	}

	if jsonrpc != "2.0" {
		return mcpError(reqID, -32600, "Invalid JSON-RPC version")
	}

	// Initialize handshake
	if method == "initialize" {
		return map[string]any{
			"jsonrpc": "2.0",
			"id":      reqID,
			"result": map[string]any{
				"protocolVersion": "2025-06-18",
				"capabilities":   map[string]any{"tools": map[string]any{}},
				"serverInfo": map[string]any{
					"name":    a.name,
					"version": "1.0.0",
				},
			},
		}
	}

	// Initialized notification
	if method == "notifications/initialized" {
		return map[string]any{"jsonrpc": "2.0", "id": reqID, "result": map[string]any{}}
	}

	// List tools
	if method == "tools/list" {
		a.mu.RLock()
		tools := a.buildMcpToolList()
		a.mu.RUnlock()
		return map[string]any{
			"jsonrpc": "2.0",
			"id":      reqID,
			"result":  map[string]any{"tools": tools},
		}
	}

	// Call tool
	if method == "tools/call" {
		toolName, _ := params["name"].(string)
		arguments, _ := params["arguments"].(map[string]any)
		if arguments == nil {
			arguments = make(map[string]any)
		}

		a.mu.RLock()
		tool, ok := a.tools[toolName]
		a.mu.RUnlock()

		if !ok || tool == nil {
			return mcpError(reqID, -32602, fmt.Sprintf("Unknown tool: %s", toolName))
		}
		if tool.Handler == nil {
			return mcpError(reqID, -32602, fmt.Sprintf("Tool %s has no handler", toolName))
		}

		rawData := map[string]any{
			"function": toolName,
			"argument": map[string]any{"parsed": []any{arguments}},
		}

		result := tool.Handler(arguments, rawData)

		responseText := ""
		if result != nil {
			m := result.ToMap()
			if r, ok := m["response"].(string); ok {
				responseText = r
			}
		}

		return map[string]any{
			"jsonrpc": "2.0",
			"id":      reqID,
			"result": map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": responseText},
				},
				"isError": false,
			},
		}
	}

	// Ping
	if method == "ping" {
		return map[string]any{"jsonrpc": "2.0", "id": reqID, "result": map[string]any{}}
	}

	return mcpError(reqID, -32601, fmt.Sprintf("Method not found: %s", method))
}

// mcpError builds a JSON-RPC error response.
func mcpError(reqID any, code int, message string) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      reqID,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
}

// handleMcp is the HTTP handler for the /mcp endpoint.
func (a *AgentBase) handleMcp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAgentRequestBody)
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(mcpError(nil, -32700, "Parse error"))
		return
	}

	resp := a.handleMcpRequest(body)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ---------------------------------------------------------------------------
// SIP methods
// ---------------------------------------------------------------------------

// EnableSipRouting enables SIP routing for the agent.
func (a *AgentBase) EnableSipRouting(autoMap bool, path string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sipRoutingEnabled = true
	return a
}

// RegisterSipUsername registers a SIP username that this agent handles.
func (a *AgentBase) RegisterSipUsername(username string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sipUsernames[username] = true
	return a
}

// ---------------------------------------------------------------------------
// Skills integration
// ---------------------------------------------------------------------------

// AddSkill loads a skill by name with optional params and registers its tools.
func (a *AgentBase) AddSkill(skillName string, params map[string]any) *AgentBase {
	if params == nil {
		params = map[string]any{}
	}
	factory := skills.GetSkillFactory(skillName)
	if factory == nil {
		a.Logger.Error("unknown skill: %s", skillName)
		return a
	}
	skill := factory(params)
	ok, errMsg := a.skillManager.LoadSkill(skill)
	if !ok {
		a.Logger.Error("failed to load skill %s: %s", skillName, errMsg)
		return a
	}
	// Register the skill's tools with the agent
	for _, tool := range skill.RegisterTools() {
		handler := tool.Handler
		td := ToolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
			Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
				return handler(args, rawData)
			},
			Secure: tool.Secure,
		}
		if tool.Fillers != nil {
			td.Fillers = tool.Fillers
		}
		if tool.SwaigFields != nil {
			td.SwaigFields = tool.SwaigFields
		}
		a.DefineTool(td)
	}
	// Add hints
	hints := skill.GetHints()
	if len(hints) > 0 {
		a.AddHints(hints)
	}
	// Add global data
	gd := skill.GetGlobalData()
	if len(gd) > 0 {
		a.UpdateGlobalData(gd)
	}
	// Add prompt sections (unless skip_prompt param is set)
	skipPrompt, _ := params["skip_prompt"].(bool)
	if !skipPrompt {
		for _, section := range skill.GetPromptSections() {
			title, _ := section["title"].(string)
			body, _ := section["body"].(string)
			if bullets, ok := section["bullets"].([]string); ok {
				a.PromptAddSection(title, body, bullets)
			} else {
				a.PromptAddSection(title, body, nil)
			}
		}
	}
	return a
}

// RemoveSkill unloads a skill by name.
func (a *AgentBase) RemoveSkill(skillName string) *AgentBase {
	a.skillManager.UnloadSkill(skillName)
	return a
}

// ListSkills returns the names of loaded skills.
func (a *AgentBase) ListSkills() []string {
	return a.skillManager.ListLoadedSkills()
}

// HasSkill returns whether a skill is loaded.
func (a *AgentBase) HasSkill(skillName string) bool {
	return a.skillManager.HasSkill(skillName)
}

// ---------------------------------------------------------------------------
// Lifecycle callbacks
// ---------------------------------------------------------------------------

// OnSummary registers a callback for post-prompt summaries.
func (a *AgentBase) OnSummary(cb SummaryCallback) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.summaryCallback = cb
	return a
}

// OnDebugEvent registers a callback for debug events.
func (a *AgentBase) OnDebugEvent(cb DebugEventHandler) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.debugEventHandler = cb
	return a
}

// ---------------------------------------------------------------------------
// SWML Rendering
// ---------------------------------------------------------------------------

// buildWebhookURL constructs the full webhook URL for SWAIG functions,
// including basic auth, route suffix, and query parameters.
func (a *AgentBase) buildWebhookURL() string {
	if a.webhookURL != "" {
		return a.webhookURL
	}

	user, pass := a.swmlService.GetBasicAuthCredentials()
	baseURL := a.swmlService.GetFullURL(false)

	// Insert credentials
	scheme := "http://"
	if strings.HasPrefix(baseURL, "https://") {
		scheme = "https://"
	}
	rest := strings.TrimPrefix(strings.TrimPrefix(baseURL, "http://"), "https://")
	authedBase := fmt.Sprintf("%s%s:%s@%s", scheme, user, pass, rest)

	route := strings.TrimRight(a.swmlService.Route, "/")
	url := authedBase
	if !strings.HasSuffix(url, route) {
		url = strings.TrimRight(url, "/") + route
	}
	url = strings.TrimRight(url, "/") + "/swaig"

	// Add query parameters
	if len(a.swaigQueryParams) > 0 {
		params := make([]string, 0, len(a.swaigQueryParams))
		for k, v := range a.swaigQueryParams {
			params = append(params, fmt.Sprintf("%s=%s", k, v))
		}
		url += "?" + strings.Join(params, "&")
	}

	return url
}

// buildPostPromptURL constructs the post-prompt URL.
func (a *AgentBase) buildPostPromptURL() string {
	if a.postPromptURL != "" {
		return a.postPromptURL
	}

	user, pass := a.swmlService.GetBasicAuthCredentials()
	baseURL := a.swmlService.GetFullURL(false)

	scheme := "http://"
	if strings.HasPrefix(baseURL, "https://") {
		scheme = "https://"
	}
	rest := strings.TrimPrefix(strings.TrimPrefix(baseURL, "http://"), "https://")
	authedBase := fmt.Sprintf("%s%s:%s@%s", scheme, user, pass, rest)

	route := strings.TrimRight(a.swmlService.Route, "/")
	url := authedBase
	if !strings.HasSuffix(url, route) {
		url = strings.TrimRight(url, "/") + route
	}
	return strings.TrimRight(url, "/") + "/post_prompt"
}

// buildSwaigFunctions returns the SWAIG functions array for the AI verb.
func (a *AgentBase) buildSwaigFunctions(webhookURL string) []map[string]any {
	functions := make([]map[string]any, 0, len(a.toolOrder)+len(a.functionIncludes))

	for _, name := range a.toolOrder {
		tool, ok := a.tools[name]
		if !ok {
			continue
		}

		// DataMap tools use their raw SwaigFields directly
		if tool.SwaigFields != nil && tool.Handler == nil {
			functions = append(functions, tool.SwaigFields)
			continue
		}

		fn := map[string]any{
			"function":     tool.Name,
			"purpose":      tool.Description,
			"web_hook_url": webhookURL,
		}

		if tool.Parameters != nil {
			fn["argument"] = map[string]any{
				"type":       "object",
				"properties": tool.Parameters,
			}
		}

		if tool.Secure {
			fn["meta_data_token"] = "secure_token"
		}

		if tool.Fillers != nil {
			fn["fillers"] = tool.Fillers
		}
		if tool.MetaData != nil {
			fn["meta_data"] = tool.MetaData
		}

		// Merge any extra SwaigFields
		if tool.SwaigFields != nil {
			for k, v := range tool.SwaigFields {
				if _, exists := fn[k]; !exists {
					fn[k] = v
				}
			}
		}

		functions = append(functions, fn)
	}

	return functions
}

// RenderSWML builds the complete SWML document for a request.
func (a *AgentBase) RenderSWML(requestData map[string]any, request *http.Request) map[string]any {
	a.mu.RLock()
	defer a.mu.RUnlock()

	doc := swml.NewDocument()

	// 1. Pre-answer verbs
	for _, v := range a.preAnswerVerbs {
		doc.AddVerb(v.Name, v.Config)
	}

	// 2. Answer verb
	if a.autoAnswer {
		answerCfg := map[string]any{
			"max_duration": 14400,
		}
		for k, v := range a.answerConfig {
			answerCfg[k] = v
		}
		doc.AddVerb("answer", answerCfg)
	}

	// 3. Record call if enabled
	if a.recordCall {
		recordCfg := map[string]any{
			"format": a.recordFormat,
			"stereo": a.recordStereo,
		}
		doc.AddVerb("record_call", recordCfg)
	}

	// 4. Post-answer verbs
	for _, v := range a.postAnswerVerbs {
		doc.AddVerb(v.Name, v.Config)
	}

	// 5. Build AI verb config
	aiConfig := make(map[string]any)

	// Prompt
	if a.usePom && len(a.pomSections) > 0 {
		aiConfig["prompt"] = map[string]any{
			"pom": a.pomSections,
		}
	} else if a.promptText != "" {
		aiConfig["prompt"] = map[string]any{
			"text": a.promptText,
		}
	}

	// Post-prompt
	if a.postPrompt != "" {
		postPromptConfig := map[string]any{
			"text": a.postPrompt,
		}
		aiConfig["post_prompt"] = postPromptConfig
	}

	// Post-prompt URL
	if a.postPrompt != "" || a.summaryCallback != nil {
		aiConfig["post_prompt_url"] = a.buildPostPromptURL()
	}

	// Params
	if len(a.params) > 0 {
		aiConfig["params"] = a.params
	}

	// Hints
	if len(a.hints) > 0 {
		aiConfig["hints"] = a.hints
	}

	// Languages
	if len(a.languages) > 0 {
		aiConfig["languages"] = a.languages
	}

	// Pronunciations
	if len(a.pronunciations) > 0 {
		aiConfig["pronounce"] = a.pronunciations
	}

	// SWAIG functions
	webhookURL := a.buildWebhookURL()
	swaigFunctions := a.buildSwaigFunctions(webhookURL)
	if len(swaigFunctions) > 0 || len(a.functionIncludes) > 0 {
		swaigConfig := map[string]any{}
		if len(swaigFunctions) > 0 {
			swaigConfig["functions"] = swaigFunctions
		}
		if len(a.functionIncludes) > 0 {
			swaigConfig["includes"] = a.functionIncludes
		}
		aiConfig["SWAIG"] = swaigConfig
	}

	// Global data
	if len(a.globalData) > 0 {
		aiConfig["global_data"] = a.globalData
	}

	// Native functions
	if len(a.nativeFunctions) > 0 {
		aiConfig["native_functions"] = a.nativeFunctions
	}

	// Pattern hints
	if len(a.patternHints) > 0 {
		aiConfig["pattern_hints"] = a.patternHints
	}

	// Contexts
	if a.contextBuilder != nil {
		ctxMap, err := a.contextBuilder.ToMap()
		if err == nil && len(ctxMap) > 0 {
			aiConfig["contexts"] = ctxMap
		}
	}

	// Prompt LLM params
	if len(a.promptLlmParams) > 0 {
		if promptCfg, ok := aiConfig["prompt"].(map[string]any); ok {
			for k, v := range a.promptLlmParams {
				promptCfg[k] = v
			}
		}
	}

	// Post-prompt LLM params
	if len(a.postPromptLlmParams) > 0 {
		if ppCfg, ok := aiConfig["post_prompt"].(map[string]any); ok {
			for k, v := range a.postPromptLlmParams {
				ppCfg[k] = v
			}
		}
	}

	// Debug events
	if a.debugEventsLevel > 0 {
		aiConfig["debug_events"] = a.debugEventsLevel
	}

	// MCP servers
	if len(a.mcpServers) > 0 {
		aiConfig["mcp_servers"] = a.mcpServers
	}

	// 6. Add AI verb
	doc.AddVerb("ai", aiConfig)

	// 7. Post-AI verbs
	for _, v := range a.postAiVerbs {
		doc.AddVerb(v.Name, v.Config)
	}

	return doc.ToMap()
}

// ---------------------------------------------------------------------------
// HTTP Server
// ---------------------------------------------------------------------------

// Run starts the HTTP server for the agent.  This is a blocking call.
func (a *AgentBase) Run() error {
	return a.buildAndServe()
}

// Serve is an alias for Run.
func (a *AgentBase) Serve() error {
	return a.Run()
}

// AsRouter returns an http.Handler for embedding in a custom server.
func (a *AgentBase) AsRouter() http.Handler {
	return a.buildMux()
}

// buildAndServe creates the HTTP server and starts listening.
func (a *AgentBase) buildAndServe() error {
	mux := a.buildMux()

	user, _ := a.swmlService.GetBasicAuthCredentials()
	addr := fmt.Sprintf("%s:%d", a.swmlService.Host, a.swmlService.Port)

	a.Logger.Info("serving agent %q on %s%s", a.name, addr, a.swmlService.Route)
	a.Logger.Info("auth user: %s", user)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	return server.ListenAndServe()
}

// buildMux creates the HTTP mux with all agent routes.
func (a *AgentBase) buildMux() *http.ServeMux {
	mux := http.NewServeMux()

	route := a.swmlService.Route
	if route == "" {
		route = "/"
	}
	route = strings.TrimRight(route, "/")
	if route == "" {
		route = "/"
	}

	// Health endpoints (no auth)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	})

	// Main SWML endpoint (with auth)
	swmlRoute := route
	if swmlRoute == "/" {
		mux.HandleFunc("/", a.withAuth(a.handleSWML))
	} else {
		mux.HandleFunc(swmlRoute, a.withAuth(a.handleSWML))
		// Also handle without trailing slash
		mux.HandleFunc(swmlRoute+"/", a.withAuth(a.handleSWML))
	}

	// SWAIG function dispatch endpoint
	swaigRoute := route + "/swaig"
	if route == "/" {
		swaigRoute = "/swaig"
	}
	mux.HandleFunc(swaigRoute, a.withAuth(a.handleSwaig))

	// Post-prompt summary endpoint
	ppRoute := route + "/post_prompt"
	if route == "/" {
		ppRoute = "/post_prompt"
	}
	mux.HandleFunc(ppRoute, a.withAuth(a.handlePostPrompt))

	// MCP server endpoint (no auth — MCP clients authenticate via headers)
	if a.mcpServerEnabled {
		mcpRoute := route + "/mcp"
		if route == "/" {
			mcpRoute = "/mcp"
		}
		mux.HandleFunc(mcpRoute, a.handleMcp)
	}

	return mux
}

// maxAgentRequestBody is the maximum request body size (1MB).
const maxAgentRequestBody = 1 << 20

// handleSWML serves the SWML document for the agent.
func (a *AgentBase) handleSWML(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, maxAgentRequestBody)
		json.NewDecoder(r.Body).Decode(&body)
	}

	a.mu.RLock()
	hasDynamic := a.dynamicConfigCallback != nil
	a.mu.RUnlock()

	var doc map[string]any
	if hasDynamic {
		doc = a.handleDynamicConfig(body, r)
	} else {
		doc = a.RenderSWML(body, r)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

// handleDynamicConfig creates an ephemeral agent copy, applies the dynamic
// config callback, and renders from the copy.
func (a *AgentBase) handleDynamicConfig(body map[string]any, r *http.Request) map[string]any {
	// Clone the agent
	ephemeral := a.clone()

	// Extract parameters for the callback
	queryParams := make(map[string]string)
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			queryParams[k] = v[0]
		}
	}

	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	// Apply the callback
	a.mu.RLock()
	cb := a.dynamicConfigCallback
	a.mu.RUnlock()

	if cb != nil {
		cb(queryParams, body, headers, ephemeral)
	}

	return ephemeral.RenderSWML(body, r)
}

// clone creates a shallow copy of the agent suitable for dynamic config.
func (a *AgentBase) clone() *AgentBase {
	a.mu.RLock()
	defer a.mu.RUnlock()

	c := &AgentBase{
		swmlService: a.swmlService,
		Logger:      a.Logger,
		name:        a.name,
		route:       a.route,
		host:        a.host,
		port:        a.port,

		promptText: a.promptText,
		postPrompt: a.postPrompt,
		usePom:     a.usePom,

		autoAnswer:   a.autoAnswer,
		recordCall:   a.recordCall,
		recordFormat: a.recordFormat,
		recordStereo: a.recordStereo,

		webhookURL:    a.webhookURL,
		postPromptURL: a.postPromptURL,
		proxyURLBase:  a.proxyURLBase,

		tokenExpirySecs: a.tokenExpirySecs,
		sessionManager:  a.sessionManager,
		debugEventsLevel: a.debugEventsLevel,

		contextBuilder: a.contextBuilder,
	}

	// Deep-copy slices and maps
	c.pomSections = make([]map[string]any, len(a.pomSections))
	copy(c.pomSections, a.pomSections)

	c.tools = make(map[string]*ToolDefinition, len(a.tools))
	for k, v := range a.tools {
		c.tools[k] = v
	}
	c.toolOrder = make([]string, len(a.toolOrder))
	copy(c.toolOrder, a.toolOrder)

	c.hints = make([]string, len(a.hints))
	copy(c.hints, a.hints)

	c.patternHints = make([]map[string]any, len(a.patternHints))
	copy(c.patternHints, a.patternHints)

	c.languages = make([]map[string]any, len(a.languages))
	copy(c.languages, a.languages)

	c.pronunciations = make([]map[string]any, len(a.pronunciations))
	copy(c.pronunciations, a.pronunciations)

	c.params = make(map[string]any, len(a.params))
	for k, v := range a.params {
		c.params[k] = v
	}

	c.globalData = make(map[string]any, len(a.globalData))
	for k, v := range a.globalData {
		c.globalData[k] = v
	}

	c.nativeFunctions = make([]string, len(a.nativeFunctions))
	copy(c.nativeFunctions, a.nativeFunctions)

	c.internalFillers = make(map[string]map[string][]string, len(a.internalFillers))
	for k, v := range a.internalFillers {
		inner := make(map[string][]string, len(v))
		for k2, v2 := range v {
			inner[k2] = make([]string, len(v2))
			copy(inner[k2], v2)
		}
		c.internalFillers[k] = inner
	}

	c.functionIncludes = make([]map[string]any, len(a.functionIncludes))
	copy(c.functionIncludes, a.functionIncludes)

	c.promptLlmParams = make(map[string]any, len(a.promptLlmParams))
	for k, v := range a.promptLlmParams {
		c.promptLlmParams[k] = v
	}
	c.postPromptLlmParams = make(map[string]any, len(a.postPromptLlmParams))
	for k, v := range a.postPromptLlmParams {
		c.postPromptLlmParams[k] = v
	}

	c.preAnswerVerbs = make([]verb, len(a.preAnswerVerbs))
	copy(c.preAnswerVerbs, a.preAnswerVerbs)
	c.postAnswerVerbs = make([]verb, len(a.postAnswerVerbs))
	copy(c.postAnswerVerbs, a.postAnswerVerbs)
	c.postAiVerbs = make([]verb, len(a.postAiVerbs))
	copy(c.postAiVerbs, a.postAiVerbs)

	c.answerConfig = make(map[string]any, len(a.answerConfig))
	for k, v := range a.answerConfig {
		c.answerConfig[k] = v
	}

	c.swaigQueryParams = make(map[string]string, len(a.swaigQueryParams))
	for k, v := range a.swaigQueryParams {
		c.swaigQueryParams[k] = v
	}

	c.sipUsernames = make(map[string]bool, len(a.sipUsernames))
	for k, v := range a.sipUsernames {
		c.sipUsernames[k] = v
	}

	c.mcpServers = make([]map[string]any, len(a.mcpServers))
	copy(c.mcpServers, a.mcpServers)
	c.mcpServerEnabled = a.mcpServerEnabled

	return c
}

// handleSwaig dispatches incoming SWAIG function calls.
func (a *AgentBase) handleSwaig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAgentRequestBody)
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	funcName, _ := body["function"].(string)
	if funcName == "" {
		http.Error(w, "missing function name", http.StatusBadRequest)
		return
	}

	args, _ := body["argument"].(map[string]any)
	if args == nil {
		// Try parsing argument as JSON string
		if argStr, ok := body["argument"].(string); ok && argStr != "" {
			json.Unmarshal([]byte(argStr), &args)
		}
		if args == nil {
			args = make(map[string]any)
		}
	}

	result, err := a.OnFunctionCall(funcName, args, body)
	if err != nil {
		a.Logger.Error("function call %q failed: %s", funcName, err)
		errResult := swaig.NewFunctionResult(fmt.Sprintf("Error: %s", err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(errResult.ToMap())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if result != nil {
		json.NewEncoder(w).Encode(result)
	} else {
		json.NewEncoder(w).Encode(swaig.NewFunctionResult("ok").ToMap())
	}
}

// handlePostPrompt handles the post-prompt summary callback.
func (a *AgentBase) handlePostPrompt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAgentRequestBody)
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	a.mu.RLock()
	cb := a.summaryCallback
	a.mu.RUnlock()

	if cb != nil {
		cb(body, body)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// withAuth wraps a handler with basic auth middleware.
func (a *AgentBase) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cache-Control", "no-store")

		user, pass := a.swmlService.GetBasicAuthCredentials()
		reqUser, reqPass, ok := r.BasicAuth()
		userMatch := subtle.ConstantTimeCompare([]byte(reqUser), []byte(user)) == 1
		passMatch := subtle.ConstantTimeCompare([]byte(reqPass), []byte(pass)) == 1
		if !ok || !userMatch || !passMatch {
			w.Header().Set("WWW-Authenticate", `Basic realm="Agent"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}
