// Package agent provides the core AgentBase type that wires together SWML
// rendering, tool dispatch, prompt management, AI configuration, and HTTP
// serving into a single self-contained AI agent.
package agent

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"

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
//
// Python equivalent: signalwire.core.mixins.tool_mixin.ToolMixin.define_tool
// Added fields to match Python: WebhookURL (webhook_url param), Required
// (required param for required argument names), IsTypedHandler (is_typed_handler).
type ToolDefinition struct {
	Name           string
	Description    string
	Parameters     map[string]any // JSON Schema for arguments (properties map)
	Required       []string       // Required parameter names included in the JSON Schema envelope
	Handler        ToolHandler
	Secure         bool
	Fillers        map[string][]string
	WaitFile       string         // URL to audio file to play while the function executes
	WaitFileLoops  int            // Number of times to loop WaitFile (0 = no loop)
	WebhookURL     string         // Per-tool webhook URL; overrides the agent-level webhook when non-empty
	MetaData       map[string]any
	SwaigFields    map[string]any // extra per-function SWAIG fields
	IsTypedHandler bool           // whether handler uses typed structs (Python: is_typed_handler)
}

// ValidateArgs validates the provided args map against the tool's parameter schema.
//
// It constructs a JSON Schema envelope from Parameters and Required (matching the
// shape emitted by buildSwaigFunctions) and validates args against that schema using
// encoding/json round-trip comparison.  When Parameters is nil or empty the function
// returns (true, nil) immediately, mirroring the Python SDK's behaviour of skipping
// validation when no schema is declared.
//
// Go's standard library does not include a JSON Schema validator, so this
// implementation performs a best-effort structural check:
//   - Every key listed in Required must be present in args.
//   - No third-party dependency is introduced; the check is intentionally lightweight.
//
// A full JSON Schema validator (e.g. github.com/xeipuuv/gojsonschema) can be
// swapped in by replacing the body of this method.
func (td *ToolDefinition) ValidateArgs(args map[string]any) (bool, []string) {
	if len(td.Parameters) == 0 {
		return true, nil
	}

	var errs []string

	// Check required parameters are present.
	for _, req := range td.Required {
		if _, ok := args[req]; !ok {
			errs = append(errs, "'"+req+"' is a required property")
		}
	}

	if len(errs) > 0 {
		return false, errs
	}
	return true, nil
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
	return func(a *AgentBase) { a.pendingName = name }
}

// WithRoute sets the HTTP route path the agent listens on.
func WithRoute(route string) AgentOption {
	return func(a *AgentBase) { a.pendingRoute = route }
}

// WithHost sets the HTTP listen address.
func WithHost(host string) AgentOption {
	return func(a *AgentBase) { a.pendingHost = host }
}

// WithPort sets the HTTP listen port.
func WithPort(port int) AgentOption {
	return func(a *AgentBase) { a.pendingPort = port }
}

// WithBasicAuth sets explicit basic-auth credentials.
func WithBasicAuth(user, password string) AgentOption {
	return func(a *AgentBase) {
		a.pendingBasicAuthUser = user
		a.pendingBasicAuthPassword = password
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

// WithAIVerbName overrides the SWML verb name used for the AI section.
// The default is "ai".  Set to "amazon_bedrock" for BedrockAgent.
func WithAIVerbName(name string) AgentOption {
	return func(a *AgentBase) { a.aiVerbName = name }
}

// WithUsePom controls whether Prompt Object Model (POM) mode is active.
// When true (default), structured prompt sections are used; when false,
// raw text from SetPromptText is used.
//
// Python equivalent: use_pom parameter in AgentBase.__init__
func WithUsePom(usePom bool) AgentOption {
	return func(a *AgentBase) { a.usePom = usePom }
}

// WithDefaultWebhookURL sets the default webhook URL for all SWAIG functions.
// When set, this URL is used as the fallback for all tools that do not specify
// their own WebhookURL.
//
// Python equivalent: default_webhook_url parameter in AgentBase.__init__
func WithDefaultWebhookURL(url string) AgentOption {
	return func(a *AgentBase) { a.defaultWebhookURL = url }
}

// WithAgentID sets a fixed agent ID. If not provided, a UUID is generated
// automatically in NewAgentBase.
//
// Python equivalent: agent_id parameter in AgentBase.__init__
// Python behavior: self.agent_id = agent_id or str(uuid.uuid4())
func WithAgentID(id string) AgentOption {
	return func(a *AgentBase) { a.AgentID = id }
}

// WithNativeFunctions sets the initial list of native (built-in) SWAIG
// function names to include in the SWAIG object on every rendered document.
//
// Python equivalent: native_functions parameter in AgentBase.__init__
func WithNativeFunctions(names []string) AgentOption {
	return func(a *AgentBase) {
		if names != nil {
			a.nativeFunctions = names
		}
	}
}

// WithSchemaPath sets the path to an optional SWML schema file used for
// validation. If empty, no schema validation is performed.
//
// Python equivalent: schema_path parameter in AgentBase.__init__
func WithSchemaPath(path string) AgentOption {
	return func(a *AgentBase) { a.schemaPath = path }
}

// WithSuppressLogs disables verbose structured logging from the agent.
// When true, info-level agent lifecycle logs are suppressed.
//
// Python equivalent: suppress_logs parameter in AgentBase.__init__
func WithSuppressLogs(suppress bool) AgentOption {
	return func(a *AgentBase) { a.suppressLogs = suppress }
}

// WithEnablePostPromptOverride allows subclasses to override the post-prompt
// URL with a custom handler. When enabled, the agent registers a
// /post_prompt_override endpoint and routes summary callbacks through it.
//
// Python equivalent: enable_post_prompt_override parameter in AgentBase.__init__
func WithEnablePostPromptOverride(enable bool) AgentOption {
	return func(a *AgentBase) { a.enablePostPromptOverride = enable }
}

// WithCheckForInputOverride enables the /check_for_input endpoint, which
// allows external systems to inject input into an active AI session.
//
// Python equivalent: check_for_input_override parameter in AgentBase.__init__
func WithCheckForInputOverride(enable bool) AgentOption {
	return func(a *AgentBase) { a.checkForInputOverride = enable }
}

// WithConfigFile sets the path to an optional YAML/JSON service configuration
// file. When provided, the file is loaded at startup and its values are merged
// with (but do not override) explicit constructor parameters.
//
// Python equivalent: config_file parameter in AgentBase.__init__
func WithConfigFile(path string) AgentOption {
	return func(a *AgentBase) { a.configFile = path }
}

// WithSchemaValidation controls whether the rendered SWML document is
// validated against the SWML schema before serving. Defaults to true.
// Can also be disabled via the SWML_SKIP_SCHEMA_VALIDATION=1 environment
// variable.
//
// Python equivalent: schema_validation parameter in AgentBase.__init__
func WithSchemaValidation(validate bool) AgentOption {
	return func(a *AgentBase) { a.schemaValidation = validate }
}

// ---------------------------------------------------------------------------
// AgentBase
// ---------------------------------------------------------------------------

// AgentBase is the central agent struct. It embeds *swml.Service so that
// Service's fields and methods (Name, Route, Host, Port, basic auth, the
// HTTP server, the tool registry, etc.) are promoted onto AgentBase. The
// agent-specific state below is layered on top.
type AgentBase struct {
	mu sync.RWMutex
	*swml.Service
	Logger *logging.Logger

	// Pending construction-time options. Filled by With* options before
	// Service is built; consumed when NewAgentBase calls swml.NewService.
	// Once Service is non-nil these are unused.
	pendingName              string
	pendingRoute             string
	pendingHost              string
	pendingPort              int
	pendingBasicAuthUser     string
	pendingBasicAuthPassword string

	// Agent identity — matches Python: self.agent_id = agent_id or str(uuid.uuid4())
	// Exported so callers can read the assigned ID without a getter.
	AgentID string

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
	dynamicConfigCallback      DynamicConfigCallback
	webhookURL                 string
	postPromptURL              string
	defaultWebhookURL          string // Python: _default_webhook_url
	swaigQueryParams           map[string]string
	proxyURLBase               string
	suppressLogs               bool // Python: _suppress_logs
	enablePostPromptOverride   bool // Python: enable_post_prompt_override
	checkForInputOverride      bool // Python: check_for_input_override
	configFile                 string // Python: config_file
	schemaPath                 string // Python: schema_path
	schemaValidation           bool   // Python: schema_validation (default true)

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

	// SIP redirect callbacks. Distinct from swml.Service.routingCallbacks:
	// these callbacks return a route string and trigger an HTTP 307 redirect,
	// matching Python web_mixin.register_routing_callback semantics
	// (web_mixin.py:621-635). The swml.Service callbacks return a document
	// override (richer Go-only semantics).
	sipRoutingCallbacks map[string]func(r *http.Request, body map[string]any) string

	// MCP integration
	mcpServers        []map[string]any // external MCP server configs
	mcpServerEnabled  bool             // expose /mcp endpoint

	// AI verb overrides — used by specialised sub-agents (e.g. BedrockAgent)
	// aiVerbName replaces the literal "ai" key in the SWML document.
	// promptTransformer, when non-nil, is called with the assembled prompt
	// map before it is embedded in the AI verb config.
	aiVerbName        string
	promptTransformer func(map[string]any) map[string]any

	// Graceful shutdown
	shutdownCh chan struct{} // closed by SetupGracefulShutdown signal handler
}

// NewAgentBase creates a new AgentBase with default values and applies the
// provided functional options.
func NewAgentBase(opts ...AgentOption) *AgentBase {
	a := &AgentBase{
		pendingName:      "Agent",
		pendingRoute:     "/",
		pendingHost:      "0.0.0.0",
		pendingPort:      3000,
		usePom:           true,
		autoAnswer:       true,
		recordFormat:     "mp4",
		recordStereo:     true,
		tokenExpirySecs:  3600,
		schemaValidation: true, // Python default: schema_validation=True
		aiVerbName:       "ai",

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
		sipRoutingCallbacks: make(map[string]func(r *http.Request, body map[string]any) string),
		mcpServers:         make([]map[string]any, 0),
	}

	for _, opt := range opts {
		opt(a)
	}

	// Auto-generate agent ID if not provided by WithAgentID.
	// Python equivalent: self.agent_id = agent_id or str(uuid.uuid4())
	if a.AgentID == "" {
		a.AgentID = generateUUID()
	}

	// Build the embedded Service from the collected pending options.
	svcOpts := []swml.ServiceOption{
		swml.WithName(a.pendingName),
		swml.WithRoute(a.pendingRoute),
		swml.WithHost(a.pendingHost),
		swml.WithPort(a.pendingPort),
	}
	if a.pendingBasicAuthUser != "" && a.pendingBasicAuthPassword != "" {
		svcOpts = append(svcOpts, swml.WithBasicAuth(a.pendingBasicAuthUser, a.pendingBasicAuthPassword))
	}
	a.Service = swml.NewService(svcOpts...)
	a.Logger = logging.New(a.Name)

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

// generateUUID produces a random UUID v4 string using crypto/rand.
// Used to generate AgentID when none is provided, matching Python's
// str(uuid.uuid4()) behavior.
func generateUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// fallback: use os.Getpid-based deterministic value
		return fmt.Sprintf("agent-%d", os.Getpid())
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ---------------------------------------------------------------------------
// Accessor methods
// ---------------------------------------------------------------------------

// GetRoute returns the agent's configured route path.
func (a *AgentBase) GetRoute() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.Route
}

// GetName returns the agent's name.
func (a *AgentBase) GetName() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.Name
}

// GetFullURL returns the full URL for this agent's endpoint, optionally
// embedding basic-auth credentials.
//
// Python equivalent: AgentBase.get_full_url(include_auth=False) (agent_base.py:325)
//
// The Python implementation handles serverless URL construction (CGI / Lambda /
// Cloud Functions / Azure) inline. In the Go SDK, serverless URL construction
// lives in pkg/lambda; this method delegates server-mode URL building to the
// embedded swml.Service and matches Python's server-mode behavior.
func (a *AgentBase) GetFullURL(includeAuth bool) string {
	return a.Service.GetFullURL(includeAuth)
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
//
// Python equivalent: prompt_mixin.PromptMixin.prompt_add_section
// Added params to match Python signature: numbered, numberedBullets, subsections.
// - numbered: if true the section itself is rendered with a numeric marker
// - numberedBullets: if true the bullet list is rendered with numbers
// - subsections: optional list of child section maps (each with "title", "body", "bullets")
func (a *AgentBase) PromptAddSection(title string, body string, bullets []string, opts ...PromptSectionOption) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()

	cfg := &promptSectionCfg{}
	for _, o := range opts {
		o(cfg)
	}

	section := map[string]any{"title": title}
	if body != "" {
		section["body"] = body
	}
	if len(bullets) > 0 {
		section["bullets"] = bullets
	}
	if cfg.numbered {
		section["numbered"] = true
	}
	if cfg.numberedBullets {
		section["numbered_bullets"] = true
	}
	if len(cfg.subsections) > 0 {
		section["subsections"] = cfg.subsections
	}
	a.pomSections = append(a.pomSections, section)
	a.usePom = true
	return a
}

// promptSectionCfg holds optional POM section configuration.
type promptSectionCfg struct {
	numbered        bool
	numberedBullets bool
	subsections     []map[string]any
}

// PromptSectionOption is a functional option for PromptAddSection.
type PromptSectionOption func(*promptSectionCfg)

// WithNumbered marks the section as numbered.
// Python equivalent: numbered=True in prompt_add_section
func WithNumbered(v bool) PromptSectionOption {
	return func(c *promptSectionCfg) { c.numbered = v }
}

// WithNumberedBullets marks the bullets list as numbered.
// Python equivalent: numbered_bullets=True in prompt_add_section
func WithNumberedBullets(v bool) PromptSectionOption {
	return func(c *promptSectionCfg) { c.numberedBullets = v }
}

// WithSubsections attaches child sections to the parent section.
// Python equivalent: subsections=[...] in prompt_add_section
func WithSubsections(subs []map[string]any) PromptSectionOption {
	return func(c *promptSectionCfg) { c.subsections = subs }
}

// PromptAddToSection finds an existing POM section by title and appends
// text and/or bullets. If the section does not exist, it is a no-op.
//
// Python equivalent: prompt_mixin.PromptMixin.prompt_add_to_section
// Added params to match Python signature: bullet (single bullet string) and
// bullets ([]string list). When body is non-empty it is appended to the
// section body. When bullet is non-empty it is added to the bullets list.
// When bullets is non-nil its elements are appended to the bullets list.
func (a *AgentBase) PromptAddToSection(title string, body string, opts ...PromptAddToSectionOption) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()

	cfg := &promptAddToSectionCfg{}
	for _, o := range opts {
		o(cfg)
	}

	for i, sec := range a.pomSections {
		if sec["title"] == title {
			// Append body text
			if body != "" {
				existing, _ := sec["body"].(string)
				if existing != "" {
					a.pomSections[i]["body"] = existing + "\n" + body
				} else {
					a.pomSections[i]["body"] = body
				}
			}
			// Append bullets
			newBullets := make([]string, 0)
			if cfg.bullet != "" {
				newBullets = append(newBullets, cfg.bullet)
			}
			newBullets = append(newBullets, cfg.bullets...)
			if len(newBullets) > 0 {
				existing, _ := sec["bullets"].([]string)
				a.pomSections[i]["bullets"] = append(existing, newBullets...)
			}
			return a
		}
	}
	return a
}

// promptAddToSectionCfg holds optional config for PromptAddToSection.
type promptAddToSectionCfg struct {
	bullet  string
	bullets []string
}

// PromptAddToSectionOption is a functional option for PromptAddToSection.
type PromptAddToSectionOption func(*promptAddToSectionCfg)

// WithBullet adds a single bullet point to an existing section.
// Python equivalent: bullet= param in prompt_add_to_section
func WithBullet(b string) PromptAddToSectionOption {
	return func(c *promptAddToSectionCfg) { c.bullet = b }
}

// WithBullets adds multiple bullet points to an existing section.
// Python equivalent: bullets= param in prompt_add_to_section
func WithBullets(bs []string) PromptAddToSectionOption {
	return func(c *promptAddToSectionCfg) { c.bullets = bs }
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

// GetPostPrompt returns the current post-prompt text. Returns an empty string
// if no post-prompt has been set.
//
// Python equivalent: prompt_mixin.PromptMixin.get_post_prompt (prompt_mixin.py line 374)
func (a *AgentBase) GetPostPrompt() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.postPrompt
}

// ---------------------------------------------------------------------------
// Tool methods
// ---------------------------------------------------------------------------

// DefineTool registers a tool (SWAIG function) with the agent.
//
// # How this becomes a tool the model sees
//
// A SWAIG function is exactly the same concept as a "tool" in native
// OpenAI / Anthropic tool calling. On every LLM turn, the SDK renders
// each registered SWAIG function into the OpenAI tool schema:
//
//	{
//	  "type": "function",
//	  "function": {
//	    "name":        "your_name_here",
//	    "description": "your description text",
//	    "parameters":  { /* your JSON schema */ }
//	  }
//	}
//
// That schema is sent to the model as part of the same API call that
// produces the next assistant message. The model reads:
//
//   - the function Description to decide WHEN to call this tool
//   - each parameter "description" (inside Parameters) to decide HOW to
//     fill in that argument from the user's utterance
//
// This means descriptions are prompt engineering, not developer
// comments. A vague Description is the #1 cause of "the model has the
// right tool but doesn't call it" failures.
//
// # Bad vs good descriptions
//
// BAD:
//
//	Description: "Lookup function"
//	Parameters:  {"id": {"type": "string", "description": "the id"}}
//
// GOOD:
//
//	Description: "Look up a customer's account details by account number. "+
//	    "Use this BEFORE quoting any account-specific info (balance, "+
//	    "plan, status). Do not use for general product questions.",
//	Parameters: map[string]any{
//	    "account_number": map[string]any{
//	        "type": "string",
//	        "description": "The customer's 8-digit account number, no "+
//	            "dashes or spaces. Ask the user if they don't provide it.",
//	    },
//	},
//
// # Tool count matters
//
// LLM tool selection accuracy degrades past ~7-8 simultaneously-active
// tools per call. Use Step.SetFunctions() to partition tools across
// steps so only the relevant subset is active at any moment.
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

// HasFunction reports whether a SWAIG function with the given name is
// registered. (Python parity: ``ToolRegistry.has_function``.)
func (a *AgentBase) HasFunction(name string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	_, ok := a.tools[name]
	return ok
}

// GetFunction returns the registered tool definition for the given
// name, or nil when no such function is registered. (Python parity:
// ``ToolRegistry.get_function``.)
func (a *AgentBase) GetFunction(name string) *ToolDefinition {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if t, ok := a.tools[name]; ok {
		return t
	}
	return nil
}

// GetAllFunctions returns a snapshot of all registered SWAIG functions
// keyed by name. The returned map is a copy — subsequent registrations
// do not mutate it. (Python parity: ``ToolRegistry.get_all_functions``.)
func (a *AgentBase) GetAllFunctions() map[string]*ToolDefinition {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make(map[string]*ToolDefinition, len(a.tools))
	for k, v := range a.tools {
		out[k] = v
	}
	return out
}

// RemoveFunction removes a registered SWAIG function. Returns true when
// the function was found and removed; false when it wasn't registered.
// (Python parity: ``ToolRegistry.remove_function``.)
func (a *AgentBase) RemoveFunction(name string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.tools[name]; !ok {
		return false
	}
	delete(a.tools, name)
	for i, n := range a.toolOrder {
		if n == name {
			a.toolOrder = append(a.toolOrder[:i], a.toolOrder[i+1:]...)
			break
		}
	}
	return true
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

// AddPatternHint adds a pattern-based speech-recognition hint with regex
// replacement semantics.
//
// Python equivalent: ai_config_mixin.AIConfigMixin.add_pattern_hint
// Python signature: add_pattern_hint(hint, pattern, replace, ignore_case=False)
//
// The Python implementation appends to self._hints (not a separate patternHints
// list) as a dict with keys "hint", "pattern", "replace", "ignore_case".
// The Go implementation stores in patternHints and merges into the rendered
// "hints" array at render time.
//
// Parameters:
//   - hint:       the hint text the model receives
//   - pattern:    regex pattern for the spoken word/phrase
//   - replace:    replacement string for the matched pattern
//   - ignoreCase: when true, matching is case-insensitive
func (a *AgentBase) AddPatternHint(hint string, pattern string, replace string, ignoreCase ...bool) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	ph := map[string]any{
		"hint":    hint,
		"pattern": pattern,
		"replace": replace,
	}
	if len(ignoreCase) > 0 && ignoreCase[0] {
		ph["ignore_case"] = true
	}
	a.patternHints = append(a.patternHints, ph)
	return a
}

// AddLanguage adds a language configuration as a raw map.
func (a *AgentBase) AddLanguage(config map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.languages = append(a.languages, config)
	return a
}

// AddLanguageTyped adds a language configuration using typed named parameters,
// matching the Python SDK's add_language method signature exactly.
//
// Python equivalent: ai_config_mixin.AIConfigMixin.add_language
// Python signature: add_language(name, code, voice, speech_fillers=None,
//
//	function_fillers=None, engine=None, model=None)
//
// Parameters:
//   - name:            display name (e.g. "English")
//   - code:            BCP-47 language code (e.g. "en-US")
//   - voice:           TTS voice name; may use "engine.voice:model" combined format
//   - speechFillers:   filler phrases for natural speech pauses
//   - functionFillers: filler phrases played during SWAIG function calls
//   - engine:          explicit TTS engine name (e.g. "elevenlabs")
//   - model:           explicit TTS model name (e.g. "eleven_turbo_v2_5")
func (a *AgentBase) AddLanguageTyped(name, code, voice string, speechFillers, functionFillers []string, engine, model string) *AgentBase {
	lang := map[string]any{
		"name": name,
		"code": code,
	}

	// Voice formatting: prefer explicit engine/model params; then try to parse
	// "engine.voice:model" combined format; otherwise use voice string as-is.
	if engine != "" || model != "" {
		lang["voice"] = voice
		if engine != "" {
			lang["engine"] = engine
		}
		if model != "" {
			lang["model"] = model
		}
	} else if strings.Contains(voice, ".") && strings.Contains(voice, ":") {
		// Parse "engine.voice:model" combined format
		parts := strings.SplitN(voice, ":", 2)
		if len(parts) == 2 {
			modelPart := parts[1]
			evParts := strings.SplitN(parts[0], ".", 2)
			if len(evParts) == 2 {
				lang["engine"] = evParts[0]
				lang["voice"] = evParts[1]
				lang["model"] = modelPart
			} else {
				lang["voice"] = voice
			}
		} else {
			lang["voice"] = voice
		}
	} else {
		lang["voice"] = voice
	}

	// Fillers
	if len(speechFillers) > 0 && len(functionFillers) > 0 {
		lang["speech_fillers"] = speechFillers
		lang["function_fillers"] = functionFillers
	} else if len(speechFillers) > 0 {
		lang["fillers"] = speechFillers
	} else if len(functionFillers) > 0 {
		lang["fillers"] = functionFillers
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.languages = append(a.languages, lang)
	return a
}

// SetLanguages replaces all language configurations.
func (a *AgentBase) SetLanguages(languages []map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.languages = languages
	return a
}

// AddPronunciation adds a pronunciation override rule.
//
// Python equivalent: ai_config_mixin.AIConfigMixin.add_pronunciation
// Python signature: add_pronunciation(replace, with_text, ignore_case=False)
//
// Parameters:
//   - replace:    the word or expression to match
//   - withText:   the phonetic spelling to substitute
//   - ignoreCase: when true, matching ignores case (Python: ignore_case)
func (a *AgentBase) AddPronunciation(replace, withText string, ignoreCase ...bool) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	rule := map[string]any{
		"replace": replace,
		"with":    withText,
	}
	if len(ignoreCase) > 0 && ignoreCase[0] {
		rule["ignore_case"] = true
	}
	a.pronunciations = append(a.pronunciations, rule)
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

// SupportedInternalFillerNames is the complete set of internal SWAIG
// function names that accept fillers, matching the SWAIGInternalFiller
// schema definition. Any name outside this set is silently ignored by
// the runtime — SetInternalFillers / AddInternalFiller warn if you pass
// an unknown name.
//
// Notable absences: change_step, gather_submit, and arbitrary user-defined
// SWAIG function names are NOT supported.
var SupportedInternalFillerNames = map[string]struct{}{
	"hangup":                  {}, // AI is hanging up the call
	"check_time":              {}, // AI is checking the time
	"wait_for_user":           {}, // AI is waiting for user input
	"wait_seconds":            {}, // deliberate pause / wait period
	"adjust_response_latency": {}, // AI is adjusting response timing
	"next_step":               {}, // transitioning between steps in prompt.contexts
	"change_context":          {}, // switching between contexts in prompt.contexts
	"get_visual_input":        {}, // processing visual input (enable_vision)
	"get_ideal_strategy":      {}, // thinking (enable_thinking)
}

// supportedInternalFillerNamesSorted returns a sorted slice copy of the
// supported names for use in warning messages.
func supportedInternalFillerNamesSorted() []string {
	out := make([]string, 0, len(SupportedInternalFillerNames))
	for n := range SupportedInternalFillerNames {
		out = append(out, n)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

// SetInternalFillers replaces all internal fillers.
//
// Internal fillers are short phrases the AI agent speaks (via TTS) while
// an internal/native function is running, so the caller doesn't hear
// dead air during transitions or background work.
//
// Supported function names (match the SWAIGInternalFiller schema):
//
//	hangup                  — when the agent is hanging up
//	check_time              — when checking the time
//	wait_for_user           — when waiting for user input
//	wait_seconds            — during deliberate pauses
//	adjust_response_latency — when adjusting response timing
//	next_step               — transitioning between steps in prompt.contexts
//	change_context          — switching between contexts in prompt.contexts
//	get_visual_input        — processing visual input (enable_vision)
//	get_ideal_strategy      — thinking (enable_thinking)
//
// Notably NOT supported: change_step, gather_submit, or arbitrary
// user-defined SWAIG function names. The runtime only honors fillers for
// the names listed above; everything else is silently ignored at the
// SWML level. This method warns at registration time if you pass an
// unknown name so you catch the typo early.
func (a *AgentBase) SetInternalFillers(fillers map[string]map[string][]string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	unknown := make([]string, 0)
	for name := range fillers {
		if _, ok := SupportedInternalFillerNames[name]; !ok {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		// Stable sort the unknown list.
		for i := 1; i < len(unknown); i++ {
			for j := i; j > 0 && unknown[j-1] > unknown[j]; j-- {
				unknown[j-1], unknown[j] = unknown[j], unknown[j-1]
			}
		}
		a.Logger.Warn(
			"unknown_internal_filler_names: %v. SetInternalFillers received "+
				"names that the SWML schema does not recognize. Those entries "+
				"will be ignored by the runtime. Supported names: %v",
			unknown, supportedInternalFillerNamesSorted(),
		)
	}
	a.internalFillers = fillers
	return a
}

// AddInternalFiller adds fillers for a specific function and language.
//
// See SetInternalFillers for the complete list of supported funcName
// values (SupportedInternalFillerNames) and what fillers do. Names
// outside the supported set log a warning and are stored but will not
// play at runtime.
func (a *AgentBase) AddInternalFiller(funcName, langCode string, fillers []string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := SupportedInternalFillerNames[funcName]; !ok {
		a.Logger.Warn(
			"unknown_internal_filler_name: %q. AddInternalFiller received a "+
				"function name the SWML schema does not recognize. The entry "+
				"will be stored but the runtime will not play these fillers. "+
				"Supported names: %v",
			funcName, supportedInternalFillerNamesSorted(),
		)
	}
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

// SetPromptTransformer installs a hook that is called with the assembled
// prompt map before it is placed into the AI verb config.  The function
// may return a new map or mutate and return the same map.  Set to nil to
// remove a previously installed transformer.
//
// This is used by specialised agents (e.g. BedrockAgent) that need to
// add or filter prompt-level keys without reimplementing all of RenderSWML.
func (a *AgentBase) SetPromptTransformer(fn func(map[string]any) map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.promptTransformer = fn
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

// DefineContexts returns the context builder, creating it if needed. The
// builder is attached to this agent so Validate() can check user-defined
// tool names against reserved native tool names (next_step, change_context,
// gather_submit).
func (a *AgentBase) DefineContexts() *contexts.ContextBuilder {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.contextBuilder == nil {
		a.contextBuilder = contexts.NewContextBuilder()
		a.contextBuilder.AttachAgent(a)
	}
	return a.contextBuilder
}

// ListToolNames returns the names of every registered SWAIG tool in
// insertion order. Implements contexts.ToolLister.
func (a *AgentBase) ListToolNames() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	names := make([]string, 0, len(a.toolOrder))
	names = append(names, a.toolOrder...)
	return names
}

// Contexts is an alias for DefineContexts.
func (a *AgentBase) Contexts() *contexts.ContextBuilder {
	return a.DefineContexts()
}

// ResetContexts removes all contexts, returning the agent to a
// no-contexts state. This is a convenience wrapper around
// DefineContexts().Reset(). Use it in a dynamic config callback when
// you need to rebuild contexts from scratch for a specific request.
func (a *AgentBase) ResetContexts() *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.contextBuilder != nil {
		a.contextBuilder.Reset()
	}
	return a
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

// OnRequest is called on every SWML request before rendering. Subclasses can
// override this method to inspect or transform the request data. It delegates
// to OnSwmlRequest.
//
// Python equivalent: web_mixin.WebMixin.on_request (web_mixin.py line 1266)
// Python signature: on_request(request_data, callback_path) -> Optional[dict]
//
// Returns nil to proceed with default rendering, or a non-nil map containing
// SWML document overrides.
func (a *AgentBase) OnRequest(requestData map[string]any, callbackPath string) map[string]any {
	return a.OnSwmlRequest(requestData, callbackPath, nil)
}

// OnSwmlRequest is the primary customization point for subclasses to modify
// the SWML document based on request data. The default implementation returns
// nil (no modification).
//
// Python equivalent: web_mixin.WebMixin.on_swml_request (web_mixin.py line 1287)
// Python signature: on_swml_request(request_data, callback_path, request) -> Optional[dict]
//
// Override this method in a subclass to inspect query params, headers, or body
// fields and return a map of SWML document overrides, or nil for no changes.
func (a *AgentBase) OnSwmlRequest(requestData map[string]any, callbackPath string, r *http.Request) map[string]any {
	return nil
}

// SetupGracefulShutdown registers OS signal handlers for SIGTERM and SIGINT
// that initiate a graceful HTTP server shutdown. This is useful for Kubernetes
// deployments where the pod receives SIGTERM before termination.
//
// Python equivalent: web_mixin.WebMixin.setup_graceful_shutdown (web_mixin.py line 1405)
// Python behavior: registers signal.SIGTERM and signal.SIGINT handlers that
// call sys.exit(0) after optional cleanup.
//
// The Go implementation uses signal.NotifyContext so that the active HTTP
// server (if started via Run/Serve) can shut down cleanly. Call this before
// Run().
func (a *AgentBase) SetupGracefulShutdown() {
	go func() {
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
		defer stop()
		<-ctx.Done()
		a.Logger.Info("shutdown signal received, initiating graceful shutdown")
		// Signal the HTTP server to stop by closing a dedicated channel.
		// buildAndServe honours a.shutdownCh if it is non-nil.
		a.mu.Lock()
		if a.shutdownCh != nil {
			select {
			case <-a.shutdownCh:
				// already closed
			default:
				close(a.shutdownCh)
			}
		}
		a.mu.Unlock()
	}()
}

// ValidateBasicAuth validates the provided username and password against the
// agent's configured basic auth credentials using a constant-time comparison.
//
// Python equivalent: auth_mixin.AuthMixin.validate_basic_auth (auth_mixin.py line 24)
// Python behavior: hmac.compare_digest(username, exp_user) and compare_digest(password, exp_pass)
func (a *AgentBase) ValidateBasicAuth(username, password string) bool {
	user, pass := a.Service.GetBasicAuthCredentials()
	userMatch := subtle.ConstantTimeCompare([]byte(username), []byte(user)) == 1
	passMatch := subtle.ConstantTimeCompare([]byte(password), []byte(pass)) == 1
	return userMatch && passMatch
}

// GetBasicAuthCredentials returns the (username, password) configured for
// this agent's HTTP basic auth.
//
// Python equivalent: auth_mixin.AuthMixin.get_basic_auth_credentials (auth_mixin.py line 42)
// Python behavior: returns (username, password) tuple from self._basic_auth
func (a *AgentBase) GetBasicAuthCredentials() (string, string) {
	return a.Service.GetBasicAuthCredentials()
}

// GetBasicAuthCredentialsWithSource returns the basic-auth credentials
// plus a string indicating their SOURCE — one of "provided",
// "environment", or "generated". Mirrors Python's
// ``auth_mixin.AuthMixin.get_basic_auth_credentials(include_source=True)``
// (auth_mixin.py line 42-73).
func (a *AgentBase) GetBasicAuthCredentialsWithSource() (user, pass, source string) {
	user, pass = a.Service.GetBasicAuthCredentials()
	envUser := os.Getenv("SWML_BASIC_AUTH_USER")
	envPass := os.Getenv("SWML_BASIC_AUTH_PASSWORD")
	switch {
	case envUser != "" && envPass != "" && user == envUser && pass == envPass:
		source = "environment"
	case strings.HasPrefix(user, "user_") && len(pass) > 20:
		source = "generated"
	default:
		source = "provided"
	}
	return
}

// ValidateToolToken verifies that a SWAIG tool security token is authentic,
// unexpired, and matches the given function name and call ID. Returns false
// when the function is not registered, the SessionManager rejects the token,
// or the validation panics for any reason.
//
// Python parity: state_mixin.StateMixin.validate_tool_token. Python rejects
// unknown function names up-front and swallows exceptions, returning false.
func (a *AgentBase) ValidateToolToken(functionName, token, callID string) (ok bool) {
	if !a.HasFunction(functionName) {
		return false
	}
	defer func() {
		if r := recover(); r != nil {
			ok = false
		}
	}()
	return a.sessionManager.ValidateToken(functionName, token, callID)
}

// CreateToolToken mints a per-call SWAIG-function token via the agent's
// SessionManager. Returns an empty string when minting fails (Python parity:
// state_mixin.StateMixin._create_tool_token, which catches all exceptions and
// returns "" on error).
func (a *AgentBase) CreateToolToken(toolName, callID string) (token string) {
	defer func() {
		if r := recover(); r != nil {
			token = ""
		}
	}()
	return a.sessionManager.CreateToken(toolName, callID)
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
					"name":    a.Name,
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
// Routing callbacks
// ---------------------------------------------------------------------------

// (RegisterRoutingCallback is defined below at the (callbackFn, path)
// signature — the duplicate (path, cb) declaration was removed during
// the merge with main, which already carries the Python-aligned form.)

// ---------------------------------------------------------------------------
// SIP methods
// ---------------------------------------------------------------------------

// EnableSipRouting enables SIP-based routing for this agent.
//
// Python equivalent: AgentBase.enable_sip_routing(auto_map=True, path="/sip")
//
// This registers a routing callback at the given path that checks incoming
// SIP usernames against the agent's registered username set. When autoMap is
// true, AutoMapSipUsernames is called to derive common usernames from the
// agent name and route.
//
// The Python implementation (agent_base.py line 612) creates a sip_routing_callback
// that extracts the SIP username from the body, checks it against _sip_usernames,
// and returns None in both the matched and unmatched case — letting the normal
// routing continue. It then calls register_routing_callback to register the
// callback, and optionally calls auto_map_sip_usernames.
func (a *AgentBase) EnableSipRouting(autoMap bool, path string) *AgentBase {
	// Build SIP routing callback that matches Python behavior
	cb := func(r *http.Request, body map[string]any) map[string]any {
		sipUsername := swml.ExtractSIPUsername(body)
		if sipUsername != "" {
			username := strings.ToLower(sipUsername)
			a.mu.RLock()
			_, matched := a.sipUsernames[username]
			a.mu.RUnlock()
			if matched {
				// Username matched this agent — let normal processing continue
				return nil
			}
			// Not matched — let routing continue
		}
		return nil
	}

	// Register routing callback on the swml.Service
	if path == "" {
		path = "/sip"
	}
	a.Service.RegisterRoutingCallback(path, cb)

	a.mu.Lock()
	a.sipRoutingEnabled = true
	a.mu.Unlock()

	// Auto-map common usernames if requested
	if autoMap {
		a.AutoMapSipUsernames()
	}

	return a
}

// RegisterRoutingCallback registers a callback function that is invoked for
// incoming requests at the given path to determine routing.
//
// Python equivalent: web_mixin.WebMixin.register_routing_callback
// Python signature: register_routing_callback(callback_fn, path="/sip")
//
// The callback receives the HTTP request and the parsed body. It should return
// a non-nil map to override the response, or nil to let normal processing continue.
// This method delegates to swml.Service.RegisterRoutingCallback.
//
// For Python-aligned redirect semantics (callback returns a route string and
// the framework issues an HTTP 307 redirect), use RegisterSipRoutingCallback.
func (a *AgentBase) RegisterRoutingCallback(callbackFn func(r *http.Request, body map[string]any) map[string]any, path string) {
	if path == "" {
		path = "/sip"
	}
	a.Service.RegisterRoutingCallback(path, callbackFn)
}

// RegisterSipRoutingCallback registers a callback whose string return value
// triggers an HTTP 307 Temporary Redirect to that route. An empty return
// value (or a GET / non-POST request) lets normal SWML processing continue.
//
// Python equivalent: web_mixin.WebMixin.register_routing_callback
// Python signature: register_routing_callback(callback_fn, path="/sip")
//
// The Python callback returns Optional[str]; on a non-None return the
// framework responds with HTTP 307 + Location: route (web_mixin.py:628-635).
// This method preserves that behavior, in contrast to RegisterRoutingCallback
// which returns a response document override (a richer Go-only mechanism).
//
// Use this form when porting Python code that relies on redirect-based SIP
// or route-dispatch patterns.
func (a *AgentBase) RegisterSipRoutingCallback(
	callbackFn func(r *http.Request, body map[string]any) string,
	path string,
) {
	if path == "" {
		path = "/sip"
	}
	path = strings.TrimRight(path, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	a.mu.Lock()
	if a.sipRoutingCallbacks == nil {
		a.sipRoutingCallbacks = make(map[string]func(r *http.Request, body map[string]any) string)
	}
	a.sipRoutingCallbacks[path] = callbackFn
	a.mu.Unlock()
}

// sipRoutingCallbackPaths returns the paths registered for SIP-redirect
// callbacks, sorted for deterministic HTTP endpoint registration.
func (a *AgentBase) sipRoutingCallbackPaths() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	paths := make([]string, 0, len(a.sipRoutingCallbacks))
	for p := range a.sipRoutingCallbacks {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

// AutoMapSipUsernames automatically registers common SIP usernames derived
// from this agent's name and route.
//
// Python equivalent: AgentBase.auto_map_sip_usernames (agent_base.py line 674)
//
// Derives usernames by:
//  1. Stripping non-alphanumeric/underscore chars from the agent name (lowercased)
//  2. Stripping non-alphanumeric/underscore chars from the route (lowercased)
//  3. If the cleaned name is longer than 3 chars, also registers a vowel-stripped variant
func (a *AgentBase) AutoMapSipUsernames() *AgentBase {
	nonAlpha := regexp.MustCompile(`[^a-z0-9_]`)

	a.mu.RLock()
	name := a.Name
	route := a.Route
	a.mu.RUnlock()

	cleanName := nonAlpha.ReplaceAllString(strings.ToLower(name), "")
	if cleanName != "" {
		a.RegisterSipUsername(cleanName)
	}

	cleanRoute := nonAlpha.ReplaceAllString(strings.ToLower(route), "")
	if cleanRoute != "" && cleanRoute != cleanName {
		a.RegisterSipUsername(cleanRoute)
	}

	// Register vowel-stripped variant if name is long enough
	if len(cleanName) > 3 {
		vowels := regexp.MustCompile(`[aeiou]`)
		noVowels := vowels.ReplaceAllString(cleanName, "")
		if noVowels != cleanName && len(noVowels) > 2 {
			a.RegisterSipUsername(noVowels)
		}
	}

	return a
}

// RegisterSipUsername registers a SIP username that this agent handles.
func (a *AgentBase) RegisterSipUsername(username string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sipUsernames[strings.ToLower(username)] = true
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
//
// Python precedence (agent_base.py:840-844): compute default from the
// route, then override with _web_hook_url_override if set. Go exposes the
// override as defaultWebhookURL via WithDefaultWebhookURL — check it first.
func (a *AgentBase) buildWebhookURL() string {
	if a.defaultWebhookURL != "" {
		return a.defaultWebhookURL
	}
	if a.webhookURL != "" {
		return a.webhookURL
	}

	user, pass := a.Service.GetBasicAuthCredentials()
	baseURL := a.Service.GetFullURL(false)

	// Insert credentials
	scheme := "http://"
	if strings.HasPrefix(baseURL, "https://") {
		scheme = "https://"
	}
	rest := strings.TrimPrefix(strings.TrimPrefix(baseURL, "http://"), "https://")
	authedBase := fmt.Sprintf("%s%s:%s@%s", scheme, user, pass, rest)

	route := strings.TrimRight(a.Service.Route, "/")
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

	user, pass := a.Service.GetBasicAuthCredentials()
	baseURL := a.Service.GetFullURL(false)

	scheme := "http://"
	if strings.HasPrefix(baseURL, "https://") {
		scheme = "https://"
	}
	rest := strings.TrimPrefix(strings.TrimPrefix(baseURL, "http://"), "https://")
	authedBase := fmt.Sprintf("%s%s:%s@%s", scheme, user, pass, rest)

	route := strings.TrimRight(a.Service.Route, "/")
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

		// Determine the effective webhook URL: per-tool override takes precedence.
		effectiveWebhook := webhookURL
		if tool.WebhookURL != "" {
			effectiveWebhook = tool.WebhookURL
		}

		fn := map[string]any{
			"function":     tool.Name,
			"description":  tool.Description,
			"web_hook_url": effectiveWebhook,
		}

		if tool.Parameters != nil {
			params := map[string]any{
				"type":       "object",
				"properties": tool.Parameters,
			}
			if len(tool.Required) > 0 {
				params["required"] = tool.Required
			}
			fn["parameters"] = params
		}

		if tool.Secure {
			fn["meta_data_token"] = "secure_token"
		}

		if tool.Fillers != nil {
			fn["fillers"] = tool.Fillers
		}
		if tool.WaitFile != "" {
			fn["wait_file"] = tool.WaitFile
		}
		if tool.WaitFileLoops > 0 {
			fn["wait_file_loops"] = tool.WaitFileLoops
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

	// Apply prompt transformer (used by specialised agents like BedrockAgent)
	if a.promptTransformer != nil {
		if promptCfg, ok := aiConfig["prompt"].(map[string]any); ok {
			aiConfig["prompt"] = a.promptTransformer(promptCfg)
		}
	}

	// 6. Add AI verb (name may be overridden, e.g. "amazon_bedrock")
	verbName := a.aiVerbName
	if verbName == "" {
		verbName = "ai"
	}
	doc.AddVerb(verbName, aiConfig)

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

// buildAndServe creates the HTTP server and starts listening. If
// SetupGracefulShutdown was called before Run, it honours the shutdown channel
// and performs a graceful server shutdown on signal receipt.
func (a *AgentBase) buildAndServe() error {
	mux := a.buildMux()

	user, _ := a.Service.GetBasicAuthCredentials()
	addr := fmt.Sprintf("%s:%d", a.Service.Host, a.Service.Port)

	a.Logger.Info("serving agent %q on %s%s", a.Name, addr, a.Service.Route)
	a.Logger.Info("auth user: %s", user)

	// Initialise shutdown channel so SetupGracefulShutdown can signal us.
	a.mu.Lock()
	if a.shutdownCh == nil {
		a.shutdownCh = make(chan struct{})
	}
	ch := a.shutdownCh
	a.mu.Unlock()

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// If SetupGracefulShutdown is active, spin a goroutine that waits for
	// the shutdown channel and then asks the server to shut down cleanly.
	go func() {
		<-ch
		a.Logger.Info("graceful shutdown: stopping HTTP server")
		server.Shutdown(context.Background()) //nolint:errcheck
	}()

	err := server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// buildMux creates the HTTP mux with all agent routes.
func (a *AgentBase) buildMux() *http.ServeMux {
	mux := http.NewServeMux()

	route := a.Service.Route
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

	// Check-for-input endpoint — matches Python web_mixin.py lines 390-396,
	// which registers /check_for_input (and /check_for_input/) on both GET
	// and POST, unconditionally. Python validates conversation_id and returns
	// a default empty-input response.
	checkRoute := route + "/check_for_input"
	if route == "/" {
		checkRoute = "/check_for_input"
	}
	mux.HandleFunc(checkRoute, a.withAuth(a.handleCheckForInput))
	mux.HandleFunc(checkRoute+"/", a.withAuth(a.handleCheckForInput))

	// MCP server endpoint (no auth — MCP clients authenticate via headers)
	if a.mcpServerEnabled {
		mcpRoute := route + "/mcp"
		if route == "/" {
			mcpRoute = "/mcp"
		}
		mux.HandleFunc(mcpRoute, a.handleMcp)
	}

	// Routing-callback endpoints — matches Python web_mixin.py lines 427-447
	// which registers an HTTP endpoint per routing callback. Without this,
	// callbacks registered via RegisterRoutingCallback / AgentServer
	// .RegisterGlobalRoutingCallback are stored in the swml service but never
	// dispatched.
	for _, cbPath := range a.Service.RoutingCallbackPaths() {
		// Skip the root path — already handled by the main SWML endpoint above.
		if cbPath == "/" || cbPath == swmlRoute {
			continue
		}
		path := strings.TrimRight(cbPath, "/")
		if path == "" {
			continue
		}
		// Register both with and without trailing slash, matching Python.
		mux.HandleFunc(path, a.withAuth(a.handleSWML))
		mux.HandleFunc(path+"/", a.withAuth(a.handleSWML))
	}

	// SIP redirect-routing endpoints. Python registers these alongside swml
	// routing callbacks (single _routing_callbacks map). Go separates them
	// because the return semantics differ (string→307 redirect vs document
	// override). handleSWML inspects sipRoutingCallbacks first and emits the
	// redirect when the callback returns a non-empty string.
	for _, cbPath := range a.sipRoutingCallbackPaths() {
		if cbPath == "/" || cbPath == swmlRoute {
			continue
		}
		path := strings.TrimRight(cbPath, "/")
		if path == "" {
			continue
		}
		mux.HandleFunc(path, a.withAuth(a.handleSWML))
		mux.HandleFunc(path+"/", a.withAuth(a.handleSWML))
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

	// SIP redirect-routing dispatch. Python web_mixin._handle_request
	// (web_mixin.py:621-635) checks _routing_callbacks; on a non-None string
	// return, responds with HTTP 307 Temporary Redirect. Mirror that here
	// using sipRoutingCallbacks (the parallel string-return registration).
	// On an empty string return, fall through to the normal SWML pipeline so
	// the same endpoint can serve as both a redirector and a document source.
	sipMatchPath := strings.TrimRight(r.URL.Path, "/")
	if sipMatchPath == "" {
		sipMatchPath = "/"
	}
	a.mu.RLock()
	sipCb, sipMatched := a.sipRoutingCallbacks[sipMatchPath]
	a.mu.RUnlock()
	if sipMatched && r.Method == http.MethodPost && body != nil {
		if route := sipCb(r, body); route != "" {
			http.Redirect(w, r, route, http.StatusTemporaryRedirect)
			return
		}
	}

	// Routing-callback dispatch. Python web_mixin._handle_request (line 620)
	// checks whether the request URL matches a registered routing callback
	// and, if so, delegates document generation to that callback. Match the
	// URL path against both the exact registered key and the trim-right form
	// (so /agents/ matches /agents).
	urlPath := r.URL.Path
	swmlRoute := strings.TrimRight(a.Service.Route, "/")
	if swmlRoute == "" {
		swmlRoute = "/"
	}
	isSwmlRoute := urlPath == swmlRoute || urlPath == swmlRoute+"/"
	var callbackPath string
	if !isSwmlRoute {
		for _, p := range a.Service.RoutingCallbackPaths() {
			trim := strings.TrimRight(p, "/")
			if trim == "" {
				continue
			}
			if urlPath == trim || urlPath == trim+"/" {
				callbackPath = p
				break
			}
		}
	}

	a.mu.RLock()
	hasDynamic := a.dynamicConfigCallback != nil
	a.mu.RUnlock()

	// Python web_mixin._handle_request (line 642) calls on_swml_request before
	// rendering and passes modifications into _render_swml, which merges them
	// into the AI verb config. Mirror that here: collect modifications first,
	// then apply to the rendered doc.
	//
	// OnRequest is the public hook documented in SWMLService.on_request;
	// AgentBase.OnRequest delegates to OnSwmlRequest. Call OnSwmlRequest
	// directly so the *http.Request reaches subclasses (matching Python, which
	// passes the request).
	modifications := a.OnSwmlRequest(body, callbackPath, r)

	var doc map[string]any
	if callbackPath != "" {
		// The swml service's OnRequest dispatches to the registered callback
		// and returns the callback's result; if the callback returns nil,
		// OnRequest falls back to the default rendered document.
		doc = a.Service.OnRequest(body, callbackPath)
	} else if hasDynamic {
		doc = a.handleDynamicConfig(body, r)
	} else {
		doc = a.RenderSWML(body, r)
	}

	if len(modifications) > 0 {
		applySwmlModifications(doc, modifications)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

// applySwmlModifications merges on_swml_request modifications into the AI
// verb config of a rendered SWML document. Matches Python
// AgentBase._render_swml modifications handling: "global_data" is
// deep-merged into the existing global_data map; other keys overwrite
// their counterparts in the AI verb config.
func applySwmlModifications(doc, modifications map[string]any) {
	sections, ok := doc["sections"].(map[string]any)
	if !ok {
		return
	}
	mainVerbs, ok := sections["main"].([]any)
	if !ok {
		return
	}
	for _, v := range mainVerbs {
		verb, ok := v.(map[string]any)
		if !ok {
			continue
		}
		aiCfg, ok := verb["ai"].(map[string]any)
		if !ok {
			continue
		}
		for k, val := range modifications {
			if k == "global_data" {
				gdIn, ok := val.(map[string]any)
				if !ok || len(gdIn) == 0 {
					continue
				}
				existing, _ := aiCfg["global_data"].(map[string]any)
				if existing == nil {
					existing = map[string]any{}
				}
				for gk, gv := range gdIn {
					existing[gk] = gv
				}
				aiCfg["global_data"] = existing
				continue
			}
			aiCfg[k] = val
		}
		return
	}
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
		Service: a.Service,
		Logger:  a.Logger,

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

	c.aiVerbName = a.aiVerbName
	c.promptTransformer = a.promptTransformer

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

// handleCheckForInput serves the /check_for_input endpoint. Matches Python
// web_mixin._handle_check_for_input_request:
//   - GET: conversation_id from query string
//   - POST: conversation_id from JSON body
//   - validates conversation_id (<=256 chars, alphanumeric + -_.)
//   - returns 400 on missing/invalid conversation_id
//   - returns {"status":"success","conversation_id":...,"new_input":false,"messages":[]}
func (a *AgentBase) handleCheckForInput(w http.ResponseWriter, r *http.Request) {
	var conversationID string

	if r.Method == http.MethodPost {
		ct := r.Header.Get("Content-Type")
		if ct != "" && !strings.HasPrefix(ct, "application/json") {
			http.Error(w, `{"error":"Content-Type must be application/json"}`, http.StatusUnsupportedMediaType)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxAgentRequestBody)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"Invalid JSON in request body"}`, http.StatusBadRequest)
			return
		}
		if cid, ok := body["conversation_id"].(string); ok {
			conversationID = cid
		}
	} else {
		conversationID = r.URL.Query().Get("conversation_id")
	}

	if conversationID == "" {
		http.Error(w, `{"error":"Missing conversation_id parameter"}`, http.StatusBadRequest)
		return
	}
	if !isValidConversationID(conversationID) {
		http.Error(w, `{"error":"Invalid conversation_id format"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":          "success",
		"conversation_id": conversationID,
		"new_input":       false,
		"messages":        []any{},
	})
}

// isValidConversationID mirrors Python's check: <=256 chars and each char
// is alphanumeric or one of '-', '_', '.'.
func isValidConversationID(id string) bool {
	if len(id) > 256 {
		return false
	}
	for _, c := range id {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c == '-' || c == '_' || c == '.':
		default:
			return false
		}
	}
	return true
}

// withAuth wraps a handler with basic auth middleware.
func (a *AgentBase) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cache-Control", "no-store")

		user, pass := a.Service.GetBasicAuthCredentials()
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
