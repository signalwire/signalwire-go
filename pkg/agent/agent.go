// Package agent provides the core AgentBase type that wires together SWML
// rendering, tool dispatch, prompt management, AI configuration, and HTTP
// serving into a single self-contained AI agent.
package agent

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	neturl "net/url"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/signalwire/signalwire-go/v3/pkg/contexts"
	"github.com/signalwire/signalwire-go/v3/pkg/logging"
	"github.com/signalwire/signalwire-go/v3/pkg/pom"
	"github.com/signalwire/signalwire-go/v3/pkg/security"
	"github.com/signalwire/signalwire-go/v3/pkg/serverless"
	"github.com/signalwire/signalwire-go/v3/pkg/skills"
	"github.com/signalwire/signalwire-go/v3/pkg/swaig"
	"github.com/signalwire/signalwire-go/v3/pkg/swml"
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

// OnSwmlRequestHook is the function-field hook that user code can set to
// override the default SWML-request customization behavior. Returning a
// non-nil map applies modifications to the rendered SWML; returning nil
// uses the default rendering unchanged.
//
// Matches Python: web_mixin.WebMixin.on_swml_request — Go has no method
// inheritance, so we expose the override as a settable function field.
type OnSwmlRequestHook func(requestData map[string]any, callbackPath string, r *http.Request) map[string]any

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
	WaitFile       string // URL to audio file to play while the function executes
	WaitFileLoops  int    // Number of times to loop WaitFile (0 = no loop)
	WebhookURL     string // Per-tool webhook URL; overrides the agent-level webhook when non-empty
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

// WithRecordFormat sets the recording format (e.g. "mp4", "wav"). The
// parameter is the defined string type swaig.RecordFormat: the Format*
// constants give autocomplete + a compile-time typo check, while Go's
// untyped-constant auto-conversion keeps a bare "wav" literal compiling. It is
// stored as a plain string so the emitted SWML is unchanged.
func WithRecordFormat(format swaig.RecordFormat) AgentOption {
	return func(a *AgentBase) { a.recordFormat = string(format) }
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

// WithSigningKey sets the SignalWire Signing Key used to validate inbound
// webhook signatures. When non-empty, signed routes (POST /, /swaig,
// /post_prompt, and any registered routing callbacks) are wrapped with
// security.WebhookMiddleware — unsigned or mis-signed requests are
// rejected with HTTP 403 before reaching the handler.
//
// When this option is unset, AgentBase falls back to the
// SIGNALWIRE_SIGNING_KEY environment variable. When neither is set, the
// agent accepts unsigned requests and emits a one-time WARN log on
// startup, per the SignalWire webhooks specification §"AgentBase integration".
//
// Python equivalent: AgentBase(signing_key="...") parameter.
func WithSigningKey(key string) AgentOption {
	return func(a *AgentBase) { a.signingKey = key }
}

// WithSigningKeyTrustProxy enables X-Forwarded-Proto / X-Forwarded-Host
// honoring during URL reconstruction. Set true when AgentBase runs behind
// a reverse proxy / ngrok / load balancer that terminates TLS upstream;
// without it the validator sees the internal scheme/host and the signature
// will mismatch.
//
// No Python-equivalent flag — Python's web_mixin reads X-Forwarded-* headers
// unconditionally; in Go we make it explicit because forging these headers
// is a real attack on naive deployments.
func WithSigningKeyTrustProxy(trust bool) AgentOption {
	return func(a *AgentBase) { a.signingKeyTrustProxy = trust }
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
	promptText  string // raw text mode
	postPrompt  string
	usePom      bool             // default true
	pomSections []map[string]any // POM sections list

	// Tool management
	tools     map[string]*ToolDefinition // registered tools keyed by name
	toolOrder []string                   // insertion order

	// AI configuration
	hints               []string
	patternHints        []map[string]any
	languages           []map[string]any
	multilingual        map[string]any
	pronunciations      []map[string]any
	params              map[string]any // AI params like temperature
	globalData          map[string]any
	nativeFunctions     []string
	internalFillers     map[string]map[string][]string
	debugEventsLevel    int
	functionIncludes    []map[string]any
	promptLlmParams     map[string]any
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
	dynamicConfigCallback    DynamicConfigCallback
	onSwmlRequestHook        OnSwmlRequestHook
	webhookURL               string
	postPromptURL            string
	defaultWebhookURL        string // Python: _default_webhook_url
	swaigQueryParams         map[string]string
	proxyURLBase             string
	suppressLogs             bool   // Python: _suppress_logs
	enablePostPromptOverride bool   // Python: enable_post_prompt_override
	checkForInputOverride    bool   // Python: check_for_input_override
	configFile               string // Python: config_file
	schemaPath               string // Python: schema_path
	schemaValidation         bool   // Python: schema_validation (default true)

	// Session security
	sessionManager  *security.SessionManager
	tokenExpirySecs int

	// Webhook signature validation. When signingKey is non-empty, signed
	// routes (POST /, /swaig, /post_prompt, plus any registered routing
	// callbacks) are wrapped with security.WebhookMiddleware which rejects
	// unsigned / mis-signed POSTs with HTTP 403. When unset, AgentBase
	// emits a startup warning and accepts unsigned requests — matching the
	// Python AgentBase behavior in porting-sdk/webhooks.md §"AgentBase
	// integration".
	//
	// Python parity: AgentBase.__init__(signing_key=...) and the
	// SIGNALWIRE_SIGNING_KEY env-var fallback applied in NewAgentBase.
	signingKey           string
	signingKeyTrustProxy bool

	// Lifecycle callbacks
	summaryCallback   SummaryCallback
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
	mcpServers       []map[string]any // external MCP server configs
	mcpServerEnabled bool             // expose /mcp endpoint

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
		pomSections:         make([]map[string]any, 0),
		tools:               make(map[string]*ToolDefinition),
		toolOrder:           make([]string, 0),
		hints:               make([]string, 0),
		patternHints:        make([]map[string]any, 0),
		languages:           make([]map[string]any, 0),
		pronunciations:      make([]map[string]any, 0),
		params:              make(map[string]any),
		globalData:          make(map[string]any),
		internalFillers:     make(map[string]map[string][]string),
		functionIncludes:    make([]map[string]any, 0),
		promptLlmParams:     make(map[string]any),
		postPromptLlmParams: make(map[string]any),
		answerConfig:        make(map[string]any),
		swaigQueryParams:    make(map[string]string),
		sipUsernames:        make(map[string]bool),
		sipRoutingCallbacks: make(map[string]func(r *http.Request, body map[string]any) string),
		mcpServers:          make([]map[string]any, 0),
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

	// Webhook signing key — fall back to env var when no explicit key was
	// supplied via WithSigningKey. Empty after fallback ⇒ validation is
	// disabled and we emit a one-time startup warning so production
	// deployments don't silently accept unsigned webhooks.
	if a.signingKey == "" {
		a.signingKey = os.Getenv("SIGNALWIRE_SIGNING_KEY")
	}
	if a.signingKey == "" {
		a.Logger.Warn("[signalwire] webhook signature validation is disabled — set SigningKey or SIGNALWIRE_SIGNING_KEY to enable")
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

// Pom returns a typed PromptObjectModel built from the agent's current
// POM sections. Returns nil when use_pom is false (Matches Python:
// “self.pom“ is “None“ when “use_pom=False“). The returned value
// is a deep copy / fresh build — mutations don't affect the agent's
// internal state.
//
// Python equivalent: “agent.pom“ instance attribute (agent_base.py
// line 209), which is a “PromptObjectModel“ instance.
func (a *AgentBase) Pom() *pom.PromptObjectModel {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if !a.usePom {
		return nil
	}
	// Build the typed POM from the agent's section maps. We rebuild on
	// every call so callers can safely mutate the returned object.
	result := pom.NewPromptObjectModel()
	for _, sec := range a.pomSections {
		s := agentSectionToPom(sec)
		if s != nil {
			result.Sections = append(result.Sections, s)
		}
	}
	return result
}

// agentSectionToPom converts one map-shaped section (the legacy storage
// inside AgentBase) into a typed *pom.Section. Used by Pom() to bridge
// between the historical map storage and the typed POM API.
func agentSectionToPom(m map[string]any) *pom.Section {
	if m == nil {
		return nil
	}
	s := &pom.Section{}
	if t, ok := m["title"].(string); ok {
		t2 := t
		s.Title = &t2
	}
	if b, ok := m["body"].(string); ok {
		s.Body = b
	}
	if bs, ok := m["bullets"].([]string); ok {
		s.Bullets = append([]string(nil), bs...)
	} else if bs, ok := m["bullets"].([]any); ok {
		for _, x := range bs {
			if str, ok := x.(string); ok {
				s.Bullets = append(s.Bullets, str)
			}
		}
	}
	if n, ok := m["numbered"].(bool); ok {
		s.Numbered = &n
	}
	// Accept both legacy snake_case ("numbered_bullets") and the JSON
	// schema's camelCase ("numberedBullets") spelling.
	if nb, ok := m["numbered_bullets"].(bool); ok {
		s.NumberedBullets = nb
	}
	if nb, ok := m["numberedBullets"].(bool); ok {
		s.NumberedBullets = nb
	}
	if subs, ok := m["subsections"].([]map[string]any); ok {
		for _, sub := range subs {
			if child := agentSectionToPom(sub); child != nil {
				s.Subsections = append(s.Subsections, child)
			}
		}
	} else if subs, ok := m["subsections"].([]any); ok {
		for _, raw := range subs {
			if sub, ok := raw.(map[string]any); ok {
				if child := agentSectionToPom(sub); child != nil {
					s.Subsections = append(s.Subsections, child)
				}
			}
		}
	}
	return s
}

// PostPrompt returns the current post-prompt text. Returns an empty string
// if no post-prompt has been set.
//
// Python equivalent: prompt_mixin.PromptMixin.get_post_prompt (prompt_mixin.py line 374)
func (a *AgentBase) PostPrompt() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.postPrompt
}

// RawPrompt returns the raw prompt text whatever “SetPromptText“ stored,
// regardless of POM mode. Returns an empty string when no raw prompt has
// been set.
//
// Python equivalent: prompt_manager.PromptManager.get_raw_prompt
func (a *AgentBase) RawPrompt() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.promptText
}

// GetContexts returns the contexts as a serialised map (the same shape SWML
// expects), or nil when no contexts have been defined yet. This mirrors
// Python's “PromptManager.get_contexts“ which returns the contexts dict
// or “None“.
//
// Python equivalent: prompt_manager.PromptManager.get_contexts
func (a *AgentBase) GetContexts() map[string]any {
	a.mu.RLock()
	cb := a.contextBuilder
	a.mu.RUnlock()
	if cb == nil {
		return nil
	}
	m, err := cb.ToMap()
	if err != nil {
		return nil
	}
	return m
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
// registered. (Matches Python: “ToolRegistry.has_function“.)
func (a *AgentBase) HasFunction(name string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	_, ok := a.tools[name]
	return ok
}

// Function returns the registered tool definition for the given
// name, or nil when no such function is registered. (Matches Python:
// “ToolRegistry.get_function“.)
func (a *AgentBase) Function(name string) *ToolDefinition {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if t, ok := a.tools[name]; ok {
		return t
	}
	return nil
}

// AllFunctions returns a snapshot of all registered SWAIG functions
// keyed by name. The returned map is a copy — subsequent registrations
// do not mutate it. (Matches Python: “ToolRegistry.get_all_functions“.)
func (a *AgentBase) AllFunctions() map[string]*ToolDefinition {
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
// (Matches Python: “ToolRegistry.remove_function“.)
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
		//nolint:nilnil // a SWAIG handler legitimately produces no result and no
		// error (mirrors Python's handler returning None); not an error condition.
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
// The Python implementation appends to self._hints (not a separate list) as a
// dict with keys "hint", "pattern", "replace", "ignore_case", so the structured
// hint renders inside the SWML ai.hints array alongside plain-string hints. The
// Go implementation stores in patternHints and merges into that same rendered
// "hints" array at render time. Matching Python, a call with any of hint,
// pattern, or replace empty is a no-op (the hint is only attached when all three
// are non-empty).
//
// Parameters:
//   - hint:       the hint text the model receives
//   - pattern:    regex pattern for the spoken word/phrase
//   - replace:    replacement string for the matched pattern
//   - ignoreCase: when true, matching is case-insensitive
func (a *AgentBase) AddPatternHint(hint string, pattern string, replace string, ignoreCase ...bool) *AgentBase {
	// Python guards: `if hint and pattern and replace`. Attach only when all
	// three are non-empty; otherwise this is a no-op (still chainable).
	if hint == "" || pattern == "" || replace == "" {
		return a
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	ph := map[string]any{
		"hint":        hint,
		"pattern":     pattern,
		"replace":     replace,
		"ignore_case": len(ignoreCase) > 0 && ignoreCase[0],
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
//	function_fillers=None, engine=None, model=None, params=None)
//
// Parameters:
//   - name:            display name (e.g. "English")
//   - code:            BCP-47 language code (e.g. "en-US")
//   - voice:           TTS voice name; may use "engine.voice:model" combined format
//   - speechFillers:   filler phrases for natural speech pauses
//   - functionFillers: filler phrases played during SWAIG function calls
//   - engine:          explicit TTS engine name (e.g. "elevenlabs")
//   - model:           explicit TTS model name (e.g. "eleven_turbo_v2_5")
//   - params:          optional per-language params dict (engine-specific tuning,
//     voice settings, etc.). Variadic — passing a single non-empty
//     map[string]any emits the SWML language object's "params" key.
//     Empty or omitted → key not emitted.
func (a *AgentBase) AddLanguageTyped(name, code, voice string, speechFillers, functionFillers []string, engine, model string, params ...map[string]any) *AgentBase {
	lang := map[string]any{
		"name": name,
		"code": code,
	}

	// Voice formatting: prefer explicit engine/model params; then try to parse
	// "engine.voice:model" combined format; otherwise use voice string as-is.
	switch {
	case engine != "" || model != "":
		lang["voice"] = voice
		if engine != "" {
			lang["engine"] = engine
		}
		if model != "" {
			lang["model"] = model
		}
	case strings.Contains(voice, ".") && strings.Contains(voice, ":"):
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
	default:
		lang["voice"] = voice
	}

	// Fillers
	switch {
	case len(speechFillers) > 0 && len(functionFillers) > 0:
		lang["speech_fillers"] = speechFillers
		lang["function_fillers"] = functionFillers
	case len(speechFillers) > 0:
		lang["fillers"] = speechFillers
	case len(functionFillers) > 0:
		lang["fillers"] = functionFillers
	}

	// Per-language params (engine-specific tuning, voice settings, etc.).
	// Only emit the "params" key when a non-empty map was passed, matching the
	// Python behavior of not polluting SWML with empty objects.
	if len(params) > 0 && len(params[0]) > 0 {
		lang["params"] = params[0]
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.languages = append(a.languages, lang)
	return a
}

// SetLanguageParams sets (or replaces) the per-language params dict on an
// already-added language. Useful when language entries are built up via
// AddLanguage/AddLanguageTyped first and engine-specific tuning is added
// later (e.g., from a config loader).
//
// Python equivalent: ai_config_mixin.AIConfigMixin.set_language_params
// Python signature: set_language_params(code, params)
//
// Parameters:
//   - code:   language code as previously passed to AddLanguage (e.g. "en-US")
//   - params: engine-specific params dict to attach. Empty/nil removes the key.
//
// Returns the AgentBase for chaining. No-op if the code isn't found.
func (a *AgentBase) SetLanguageParams(code string, params map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, lang := range a.languages {
		if c, _ := lang["code"].(string); c == code {
			if len(params) > 0 {
				lang["params"] = params
			} else {
				delete(lang, "params")
			}
			break
		}
	}
	return a
}

// LanguageParams reads the per-language params dict for a previously-added
// language.
//
// Python equivalent: ai_config_mixin.AIConfigMixin.get_language_params
// Python signature: get_language_params(code) -> Optional[Dict[str, Any]]
//
// Returns the params map if set, or nil otherwise (including when the code is
// unknown). Callers can distinguish "no params set" from "empty params set" by
// the fact that empty maps are never stored (SetLanguageParams with an empty
// dict removes the key).
func (a *AgentBase) LanguageParams(code string) map[string]any {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, lang := range a.languages {
		if c, _ := lang["code"].(string); c == code {
			if p, ok := lang["params"].(map[string]any); ok {
				return p
			}
			return nil
		}
	}
	return nil
}

// SetLanguages replaces all language configurations.
func (a *AgentBase) SetLanguages(languages []map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.languages = languages
	return a
}

// SetMultilingual configures ASR-driven multilingual mode (Mode B).
//
// Python equivalent: ai_config_mixin.AIConfigMixin.set_multilingual
// Python signature: set_multilingual(config) -> AgentBase
//
// Emits a top-level multilingual object on the AI verb. The recognizer runs in
// code-switching mode and the agent answers in whatever language the caller
// actually spoke - the model does not pick the language. This is mutually
// exclusive with SetLanguages; if both are set the server uses multilingual and
// ignores languages.
//
// Parameters:
//   - config: the multilingual config object (languages, allowed,
//     start_language, min_switch_words, fillers, etc.).
func (a *AgentBase) SetMultilingual(config map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(config) > 0 {
		a.multilingual = config
	}
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

// SetGlobalData merges data into the global data (later keys win, siblings
// survive) — matching Python's set_global_data, which is a .update() not a
// replace. Use ClearGlobalData first if a full replace is intended.
func (a *AgentBase) SetGlobalData(data map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	// MERGE, not replace — Python's set_global_data does self._global_data.update(data)
	// (ai_config_mixin.py:327) so skills and other callers each contribute keys
	// without clobbering siblings. A later key with the same name wins; other
	// keys survive. (This is identical to UpdateGlobalData by design.)
	if a.globalData == nil {
		a.globalData = make(map[string]any, len(data))
	}
	for k, v := range data {
		a.globalData[k] = v
	}
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

// GetGlobalData returns a copy of the accumulated global data.
func (a *AgentBase) GetGlobalData() map[string]any {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make(map[string]any, len(a.globalData))
	for k, v := range a.globalData {
		out[k] = v
	}
	return out
}

// GetSIPUsernames returns the registered SIP usernames (lowercased, sorted).
func (a *AgentBase) GetSIPUsernames() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]string, 0, len(a.sipUsernames))
	for u := range a.sipUsernames {
		out = append(out, u)
	}
	sort.Strings(out)
	return out
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

// SetPromptLlmParams merges LLM parameters into the main prompt's params.
// Python (ai_config_mixin.py:669) does self._prompt_llm_params.update(params) —
// a MERGE, so repeated calls with distinct keys accumulate rather than replace.
func (a *AgentBase) SetPromptLlmParams(params map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.promptLlmParams == nil {
		a.promptLlmParams = make(map[string]any)
	}
	for k, v := range params {
		a.promptLlmParams[k] = v
	}
	return a
}

// SetPostPromptLlmParams merges LLM parameters into the post-prompt's params.
// Python (ai_config_mixin.py:703) does self._post_prompt_llm_params.update(params) —
// a MERGE, so repeated calls with distinct keys accumulate rather than replace.
func (a *AgentBase) SetPostPromptLlmParams(params map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.postPromptLlmParams == nil {
		a.postPromptLlmParams = make(map[string]any)
	}
	for k, v := range params {
		a.postPromptLlmParams[k] = v
	}
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

// preAnswerSafeVerbs mirrors Python AgentBase._PRE_ANSWER_SAFE_VERBS
// (agent_base.py:331) — the set of SWML verbs known to be safe to run while
// the call is still ringing (before the answer).
var preAnswerSafeVerbs = map[string]struct{}{
	"transfer":         {},
	"execute":          {},
	"return":           {},
	"label":            {},
	"goto":             {},
	"request":          {},
	"switch":           {},
	"cond":             {},
	"if":               {},
	"eval":             {},
	"set":              {},
	"unset":            {},
	"hangup":           {},
	"send_sms":         {},
	"sleep":            {},
	"stop_record_call": {},
	"stop_denoise":     {},
	"stop_tap":         {},
}

// preAnswerAutoAnswerVerbs mirrors Python AgentBase._AUTO_ANSWER_VERBS
// (agent_base.py:351) — verbs that answer the call unless "auto_answer" is
// explicitly false.
var preAnswerAutoAnswerVerbs = map[string]struct{}{
	"play":    {},
	"connect": {},
}

// sortedPreAnswerSafeVerbs returns the safe-verb names in sorted order for a
// stable warning message (Python sorts the set when formatting).
func sortedPreAnswerSafeVerbs() []string {
	names := make([]string, 0, len(preAnswerSafeVerbs))
	for v := range preAnswerSafeVerbs {
		names = append(names, v)
	}
	sort.Strings(names)
	return names
}

// AddPreAnswerVerb adds a SWML verb to execute before the answer.
//
// Python equivalent: AgentBase.add_pre_answer_verb (agent_base.py:546).
// Pre-answer verbs run while the call is still ringing, so only certain verbs
// are safe. A verb that is genuinely unsafe before answer is an INVALID input:
// Python raises ValueError, and the Go port panics (a chaining builder that
// returns *AgentBase cannot return an error; panic matches the port's
// convention for build-time invalid args — see datamap.Expression). Verbs that
// answer the call (play, connect) instead get a warning unless
// "auto_answer": false is present, mirroring Python's _AUTO_ANSWER_VERBS branch.
func (a *AgentBase) AddPreAnswerVerb(verbName string, config map[string]any) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, isAutoAnswer := preAnswerAutoAnswerVerbs[verbName]; isAutoAnswer {
		if aa, ok := config["auto_answer"]; !ok || aa != false {
			a.Logger.Warn(
				"pre_answer_verb_will_answer: verb=%q hint=add 'auto_answer': false to prevent %s from answering the call",
				verbName, verbName,
			)
		}
	} else if _, safe := preAnswerSafeVerbs[verbName]; !safe {
		panic(fmt.Sprintf(
			"AddPreAnswerVerb: verb %q is not safe for pre-answer use. Safe verbs: %s",
			verbName, strings.Join(sortedPreAnswerSafeVerbs(), ", "),
		))
	}

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

// DefineContextsFromMap populates the agent's ContextBuilder from a
// fully-formed contexts map and returns the agent for chaining.
//
// Python equivalent: AgentBase.define_contexts({...}) (prompt_mixin.py:131 →
// prompt manager define_contexts, manager.py:75), which accepts a dict in the
// canonical SWML contexts shape — the same shape ContextBuilder.to_dict()
// emits. The accepted shape is:
//
//	{
//	  "<context-name>": {
//	    "steps": [
//	      {"name": "...", "text": "...", "step_criteria": "...",
//	       "functions": "none" | ["fn", ...],
//	       "valid_steps": ["..."], "valid_contexts": ["..."],
//	       "end": bool, "skip_user_turn": bool, "skip_to_next_step": bool}
//	    ],
//	    "valid_contexts": ["..."], "valid_steps": ["..."],
//	    "initial_step": "...", "post_prompt": "...", "system_prompt": "...",
//	    "user_prompt": "...", "prompt": "...", "consolidate": bool,
//	    "full_reset": bool, "isolated": bool
//	  }
//	}
//
// Unlike the chained DefineContexts builder, this populates the existing
// ContextBuilder in one call so callers can hand it a deserialised map.
func (a *AgentBase) DefineContextsFromMap(cfg map[string]any) *AgentBase {
	builder := a.DefineContexts()

	a.mu.Lock()
	defer a.mu.Unlock()

	// Deterministic order so re-runs and the SWML output are stable.
	names := make([]string, 0, len(cfg))
	for name := range cfg {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		raw, ok := cfg[name].(map[string]any)
		if !ok {
			continue
		}
		ctx := builder.AddContext(name)

		if steps, ok := raw["steps"].([]any); ok {
			for _, s := range steps {
				stepMap, ok := s.(map[string]any)
				if !ok {
					continue
				}
				stepName, _ := stepMap["name"].(string)
				if stepName == "" {
					continue
				}
				step := ctx.AddStep(stepName)
				if text, ok := stepMap["text"].(string); ok {
					step.SetText(text)
				}
				if criteria, ok := stepMap["step_criteria"].(string); ok {
					step.SetStepCriteria(criteria)
				}
				if fns, ok := stepMap["functions"]; ok {
					if coerced := coerceFunctions(fns); coerced != nil {
						step.SetFunctions(coerced)
					}
				}
				if vs := coerceStringSlice(stepMap["valid_steps"]); vs != nil {
					step.SetValidSteps(vs)
				}
				if vc := coerceStringSlice(stepMap["valid_contexts"]); vc != nil {
					step.SetValidContexts(vc)
				}
				if end, ok := stepMap["end"].(bool); ok && end {
					step.SetEnd(true)
				}
				if skip, ok := stepMap["skip_user_turn"].(bool); ok && skip {
					step.SetSkipUserTurn(true)
				}
				if skip, ok := stepMap["skip_to_next_step"].(bool); ok && skip {
					step.SetSkipToNextStep(true)
				}
			}
		}

		if initial, ok := raw["initial_step"].(string); ok {
			ctx.SetInitialStep(initial)
		}
		if vc := coerceStringSlice(raw["valid_contexts"]); vc != nil {
			ctx.SetValidContexts(vc)
		}
		if vs := coerceStringSlice(raw["valid_steps"]); vs != nil {
			ctx.SetValidSteps(vs)
		}
		if pp, ok := raw["post_prompt"].(string); ok {
			ctx.SetPostPrompt(pp)
		}
		if sp, ok := raw["system_prompt"].(string); ok {
			ctx.SetSystemPrompt(sp)
		}
		if up, ok := raw["user_prompt"].(string); ok {
			ctx.SetUserPrompt(up)
		}
		if p, ok := raw["prompt"].(string); ok {
			ctx.SetPrompt(p)
		}
		if c, ok := raw["consolidate"].(bool); ok && c {
			ctx.SetConsolidate(true)
		}
		if fr, ok := raw["full_reset"].(bool); ok && fr {
			ctx.SetFullReset(true)
		}
		if iso, ok := raw["isolated"].(bool); ok && iso {
			ctx.SetIsolated(true)
		}
	}

	return a
}

// coerceFunctions normalises the SWML "functions" value, which may be the
// string "none" or a list of tool names, into the form SetFunctions accepts.
func coerceFunctions(v any) any {
	switch t := v.(type) {
	case string:
		return t
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// coerceStringSlice converts a JSON-decoded []any (or []string) into []string,
// returning nil when the value is absent or not a list.
func coerceStringSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
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

// ManualSetProxyURL overrides the proxy URL base used for webhook URL generation.
func (a *AgentBase) ManualSetProxyURL(url string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.proxyURLBase = url
	return a
}

// SetWebHookURL explicitly sets the webhook URL used in SWAIG function defs.
func (a *AgentBase) SetWebHookURL(url string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.webhookURL = url
	return a
}

// SetPostPromptURL sets the URL for post-prompt summary delivery.
func (a *AgentBase) SetPostPromptURL(url string) *AgentBase {
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

// EnableDebugRoutes enables the agent's debug HTTP routes (/debug and
// /debug_events).
//
// Python equivalent: web_mixin.enable_debug_routes (web_mixin.py:1343), which
// is a backward-compatibility no-op returning self because the debug routes
// are registered unconditionally in _register_routes. The Go port mirrors that:
// AsRouter always registers /debug and /debug_events, so this method exists for
// API compatibility and chaining and simply returns the agent.
func (a *AgentBase) EnableDebugRoutes() *AgentBase {
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

// OnSwmlRequest is the primary customization point for the user to modify
// the SWML document based on request data. If a hook has been registered
// via SetOnSwmlRequestHook the hook is invoked; otherwise this returns nil
// (no modification).
//
// Python equivalent: web_mixin.WebMixin.on_swml_request (web_mixin.py line 1287)
// Python signature: on_swml_request(request_data, callback_path, request) -> Optional[dict]
//
// Go has no method overriding via embedded structs alone — the hook field
// is the idiomatic Go equivalent of Python's overridable on_swml_request.
// The third *http.Request argument is preserved on the Go-native signature
// (the cross-language audit projects only the first two args). Returning a
// non-nil map applies modifications to the rendered SWML; returning nil
// uses the default rendering unchanged.
func (a *AgentBase) OnSwmlRequest(requestData map[string]any, callbackPath string, r *http.Request) map[string]any {
	a.mu.RLock()
	hook := a.onSwmlRequestHook
	a.mu.RUnlock()
	if hook != nil {
		return hook(requestData, callbackPath, r)
	}
	return nil
}

// SetOnSwmlRequestHook registers a function that customizes the SWML
// response on a per-request basis. The hook receives the parsed body,
// the callback path (for routing-callback dispatch), and the raw
// *http.Request for header / query inspection. Returning a non-nil map
// applies modifications to the rendered SWML; returning nil falls
// through to the default rendering.
//
// Matches Python: this is the Go-idiomatic way of "overriding"
// on_swml_request — Go has no method inheritance.
func (a *AgentBase) SetOnSwmlRequestHook(hook OnSwmlRequestHook) *AgentBase {
	a.mu.Lock()
	a.onSwmlRequestHook = hook
	a.mu.Unlock()
	return a
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
// “auth_mixin.AuthMixin.get_basic_auth_credentials(include_source=True)“
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
// Matches Python: state_mixin.StateMixin.validate_tool_token. Python rejects
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
// SessionManager. Returns an empty string when minting fails (Matches Python:
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
			tool["inputSchema"] = ensureParameterStructure(td.Parameters, td.Required)
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
				"capabilities":    map[string]any{"tools": map[string]any{}},
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
		if err := json.NewEncoder(w).Encode(mcpError(nil, -32700, "Parse error")); err != nil {
			a.Logger.Warn("failed to write MCP parse-error response: %s", err)
		}
		return
	}

	resp := a.handleMcpRequest(body)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		a.Logger.Warn("failed to write MCP response: %s", err)
	}
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

// EnableSIPRouting enables SIP-based routing for this agent.
//
// Python equivalent: AgentBase.enable_sip_routing(auto_map=True, path="/sip")
//
// This registers a routing callback at the given path that checks incoming
// SIP usernames against the agent's registered username set. When autoMap is
// true, AutoMapSIPUsernames is called to derive common usernames from the
// agent name and route.
//
// The Python implementation (agent_base.py line 612) creates a sip_routing_callback
// that extracts the SIP username from the body, checks it against _sip_usernames,
// and returns None in both the matched and unmatched case — letting the normal
// routing continue. It then calls register_routing_callback to register the
// callback, and optionally calls auto_map_sip_usernames.
func (a *AgentBase) EnableSIPRouting(autoMap bool, path string) *AgentBase {
	// Build SIP routing callback that matches Python behavior: it extracts the
	// SIP username and returns nil (no redirect) in both the matched and
	// unmatched case, letting normal processing continue.
	cb := func(body map[string]any, _ map[string]any) *string {
		sipUsername := swml.ExtractSIPUsername(body)
		if sipUsername != "" {
			username := strings.ToLower(sipUsername)
			a.mu.RLock()
			_, matched := a.sipUsernames[username]
			a.mu.RUnlock()
			_ = matched // matched or not, routing continues (returns nil)
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
		a.AutoMapSIPUsernames()
	}

	return a
}

// RegisterRoutingCallback registers a callback function that is invoked for
// incoming POST requests at the given path to determine routing.
//
// Python equivalent: web_mixin.WebMixin.register_routing_callback
// Python signature: register_routing_callback(callback_fn, path="/sip")
//
// The callback receives the parsed request body and the request headers —
// callback_fn(body, headers) — and returns a non-nil route string to redirect
// the request (the framework issues an HTTP 307 Temporary Redirect preserving
// method + body) or nil to let normal SWML processing continue. This method
// delegates to swml.Service.RegisterRoutingCallback.
func (a *AgentBase) RegisterRoutingCallback(callbackFn swml.RoutingCallback, path string) {
	a.Service.RegisterRoutingCallback(normalizeCallbackPath(path), callbackFn)
}

// normalizeCallbackPath mirrors Python web_mixin.register_routing_callback's
// path handling: default the empty path to "/sip", ensure a single leading
// slash, and strip any trailing slash so e.g. "agents/" registers (and later
// matches) as "/agents".
func normalizeCallbackPath(path string) string {
	if path == "" {
		return "/sip"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
	}
	if path == "" {
		path = "/"
	}
	return path
}

// RegisterSIPRoutingCallback registers a callback whose string return value
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
func (a *AgentBase) RegisterSIPRoutingCallback(
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

// AutoMapSIPUsernames automatically registers common SIP usernames derived
// from this agent's name and route.
//
// Python equivalent: AgentBase.auto_map_sip_usernames (agent_base.py line 674)
//
// Derives usernames by:
//  1. Stripping non-alphanumeric/underscore chars from the agent name (lowercased)
//  2. Stripping non-alphanumeric/underscore chars from the route (lowercased)
//  3. If the cleaned name is longer than 3 chars, also registers a vowel-stripped variant
func (a *AgentBase) AutoMapSIPUsernames() *AgentBase {
	nonAlpha := regexp.MustCompile(`[^a-z0-9_]`)

	a.mu.RLock()
	name := a.Name
	route := a.Route
	a.mu.RUnlock()

	cleanName := nonAlpha.ReplaceAllString(strings.ToLower(name), "")
	if cleanName != "" {
		a.RegisterSIPUsername(cleanName)
	}

	cleanRoute := nonAlpha.ReplaceAllString(strings.ToLower(route), "")
	if cleanRoute != "" && cleanRoute != cleanName {
		a.RegisterSIPUsername(cleanRoute)
	}

	// Register vowel-stripped variant if name is long enough
	if len(cleanName) > 3 {
		vowels := regexp.MustCompile(`[aeiou]`)
		noVowels := vowels.ReplaceAllString(cleanName, "")
		if noVowels != cleanName && len(noVowels) > 2 {
			a.RegisterSIPUsername(noVowels)
		}
	}

	return a
}

// RegisterSIPUsername registers a SIP username that this agent handles.
func (a *AgentBase) RegisterSIPUsername(username string) *AgentBase {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sipUsernames[strings.ToLower(username)] = true
	return a
}

// ---------------------------------------------------------------------------
// Skills integration
// ---------------------------------------------------------------------------

// AddSkill loads a skill by name with optional params and registers its tools.
//
// skillName is a skills.SkillName (a defined string type). The built-in
// skills.Skill* constants give autocomplete + call-site typo checking; because
// Go auto-converts untyped string-constant literals, a bare "datetime" literal
// or skills.SkillName("custom") for a third-party skill compiles identically —
// compatibility with the Python reference's str parameter.
func (a *AgentBase) AddSkill(skillName skills.SkillName, params map[string]any) *AgentBase {
	if params == nil {
		params = map[string]any{}
	}
	name := string(skillName)
	factory := skills.GetSkillFactory(name)
	if factory == nil {
		a.Logger.Error("unknown skill: %s", name)
		return a
	}
	skill := factory(params)
	ok, errMsg := a.skillManager.LoadSkill(skill)
	if !ok {
		a.Logger.Error("failed to load skill %s: %s", name, errMsg)
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

// RemoveSkill unloads a skill by name. Accepts a skills.SkillName constant or
// any string literal (Go auto-converts), mirroring AddSkill.
func (a *AgentBase) RemoveSkill(skillName skills.SkillName) *AgentBase {
	a.skillManager.UnloadSkill(string(skillName))
	return a
}

// ListSkills returns the names of loaded skills.
func (a *AgentBase) ListSkills() []string {
	return a.skillManager.ListLoadedSkills()
}

// HasSkill returns whether a skill is loaded. Accepts a skills.SkillName
// constant or any string literal (Go auto-converts), mirroring AddSkill.
func (a *AgentBase) HasSkill(skillName skills.SkillName) bool {
	return a.skillManager.HasSkill(string(skillName))
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

// buildEndpointURL constructs the authed webhook URL for an arbitrary agent
// endpoint (e.g. "debug_events"), including swaigQueryParams. Mirrors Python
// _build_webhook_url(endpoint, query_params) (used at agent_base.py:1240 for
// the debug_events webhook).
func (a *AgentBase) buildEndpointURL(endpoint string) string {
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
	url = strings.TrimRight(url, "/") + "/" + endpoint

	if len(a.swaigQueryParams) > 0 {
		params := make([]string, 0, len(a.swaigQueryParams))
		for k, v := range a.swaigQueryParams {
			params = append(params, fmt.Sprintf("%s=%s", k, v))
		}
		sort.Strings(params)
		url += "?" + strings.Join(params, "&")
	}

	return url
}

// ensureParameterStructure normalizes a tool's declared parameters into the
// SWML/JSON-Schema envelope, mirroring the Python reference
// SWAIGFunction._ensure_parameter_structure:
//
//   - a COMPLETE schema (already has both "type" and "properties") is passed
//     through unchanged — NOT re-wrapped. This is the pass-through path a caller
//     hits when they hand DefineTool a full {type,properties,required} schema;
//     double-wrapping it would bury the real schema under a spurious outer
//     {type:object, properties:{...the schema...}}.
//   - otherwise, params is treated as a bare properties map and wrapped in
//     {type:object, properties:params}, with required appended when non-empty.
func ensureParameterStructure(params map[string]any, required []string) map[string]any {
	if len(params) == 0 {
		return map[string]any{"type": "object", "properties": map[string]any{}}
	}
	_, hasType := params["type"]
	_, hasProps := params["properties"]
	if hasType && hasProps {
		return params
	}
	out := map[string]any{"type": "object", "properties": params}
	if len(required) > 0 {
		out["required"] = required
	}
	return out
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
			fn["parameters"] = ensureParameterStructure(tool.Parameters, tool.Required)
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
		if err := doc.AddVerb(v.Name, v.Config); err != nil {
			a.Logger.Warn("failed to add pre-answer verb %q: %s", v.Name, err)
		}
	}

	// 2. Answer verb
	if a.autoAnswer {
		answerCfg := map[string]any{
			"max_duration": 14400,
		}
		for k, v := range a.answerConfig {
			answerCfg[k] = v
		}
		if err := doc.AddVerb("answer", answerCfg); err != nil {
			a.Logger.Warn("failed to add answer verb: %s", err)
		}
	}

	// 3. Record call if enabled
	if a.recordCall {
		recordCfg := map[string]any{
			"format": a.recordFormat,
			"stereo": a.recordStereo,
		}
		if err := doc.AddVerb("record_call", recordCfg); err != nil {
			a.Logger.Warn("failed to add record_call verb: %s", err)
		}
	}

	// 4. Post-answer verbs
	for _, v := range a.postAnswerVerbs {
		if err := doc.AddVerb(v.Name, v.Config); err != nil {
			a.Logger.Warn("failed to add post-answer verb %q: %s", v.Name, err)
		}
	}

	// 5. Build AI verb config
	aiConfig := make(map[string]any)

	// Prompt. Python's get_prompt() returns the POM list (possibly empty) when
	// usePom is active, else the prompt text; build_config ALWAYS emits a
	// "prompt" block from that base — so an agent with usePom but no sections
	// still renders prompt:{pom:[]}. Mirroring this is what lets prompt LLM
	// params (below) merge into ai.prompt even when no base text/section is set.
	if a.usePom {
		pom := a.pomSections
		if pom == nil {
			pom = []map[string]any{}
		}
		aiConfig["prompt"] = map[string]any{
			"pom": pom,
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

	// Hints — a single mixed array of plain-string hints (AddHint/AddHints) and
	// structured pattern hints (AddPatternHint), matching Python, whose
	// add_pattern_hint appends structured dicts into the same self._hints list
	// that renders under ai.hints. (Python has NO separate pattern_hints key.)
	if len(a.hints) > 0 || len(a.patternHints) > 0 {
		merged := make([]any, 0, len(a.hints)+len(a.patternHints))
		for _, h := range a.hints {
			merged = append(merged, h)
		}
		for _, ph := range a.patternHints {
			merged = append(merged, ph)
		}
		aiConfig["hints"] = merged
	}

	// Languages
	if len(a.languages) > 0 {
		aiConfig["languages"] = a.languages
	}

	// Multilingual (ASR-driven mode; top-level multilingual object)
	if len(a.multilingual) > 0 {
		aiConfig["multilingual"] = a.multilingual
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

	// (Pattern hints are merged into the "hints" array above, matching Python;
	// there is no separate pattern_hints key in the SWML ai block.)

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

	// Debug events. Python (agent_base.py:1232-1246) wires a debug webhook so
	// the platform POSTs debug events back to the agent's /debug_events
	// endpoint: it sets params.debug_webhook_url and params.debug_webhook_level
	// (no separate ai.debug_events key is emitted). Mirror that here so the
	// /debug_events route (registered in AsRouter) actually receives events and
	// OnDebugEvent fires.
	if a.debugEventsLevel > 0 {
		params, ok := aiConfig["params"].(map[string]any)
		if !ok {
			params = map[string]any{}
			aiConfig["params"] = params
		}
		params["debug_webhook_url"] = a.buildEndpointURL("debug_events")
		params["debug_webhook_level"] = a.debugEventsLevel
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
	if err := doc.AddVerb(verbName, aiConfig); err != nil {
		a.Logger.Warn("failed to add AI verb %q: %s", verbName, err)
	}

	// 7. Post-AI verbs
	for _, v := range a.postAiVerbs {
		if err := doc.AddVerb(v.Name, v.Config); err != nil {
			a.Logger.Warn("failed to add post-AI verb %q: %s", v.Name, err)
		}
	}

	return doc.ToMap()
}

// ---------------------------------------------------------------------------
// HTTP Server
// ---------------------------------------------------------------------------

// ErrServerlessUnsupported is returned by Run / RunWithMode when a serverless
// execution mode (AWS Lambda, Google Cloud Functions, Azure Functions, or CGI)
// is detected. Go's serverless request handling is NOT performed inline by
// Run: it lives in the dedicated adapter pkg/lambda, which wraps the agent's
// http.Handler (AgentBase.AsRouter) and is driven from main() by the platform
// runtime (e.g. aws-lambda-go's lambda.Start). Run returning this error rather
// than silently serving HTTP avoids binding a TCP listener that would never
// receive traffic inside a function runtime. errors.Is-able.
var ErrServerlessUnsupported = errors.New("agent: serverless execution mode detected; serve the agent via its http.Handler (AsRouter) using the platform adapter (e.g. pkg/lambda) rather than Run()")

// Run is the universal entry point for the agent. It auto-detects the runtime
// execution mode from the process environment and dispatches accordingly,
// mirroring Python's run() (web_mixin.py:341 + serverless_mixin.py):
//
//   - server                → start the long-running HTTP server (blocking)
//   - cgi                   → dispatch the single CGI request through the
//     agent handler (pkg/serverless ServeCGI: env + stdin → stdout), then return
//   - lambda / gcf / azure  → return ErrServerlessUnsupported (these are
//     served via the platform adapter — pkg/lambda, pkg/serverless — wired from
//     main() because the platform runtime owns the event loop)
//
// Detection order matches the cross-language SDK contract (see
// swml.GetExecutionMode). To override the detected mode (e.g. in tests or an
// explicit deployment), use RunWithMode.
//
// Run delegates server mode to RunContext with context.Background(); use
// RunContext directly to drive a graceful shutdown from a context.
func (a *AgentBase) Run() error {
	return a.RunWithMode(a.DetectRunMode())
}

// DetectRunMode reports the execution mode Run would dispatch on, derived from
// the process environment via swml.GetExecutionMode. Exposed so callers can
// branch (e.g. wire a pkg/lambda adapter) before invoking Run. Mirrors
// Python's get_execution_mode() as consumed by run().
func (a *AgentBase) DetectRunMode() swml.ExecutionMode {
	return swml.GetExecutionMode()
}

// RunWithMode is the force-mode form of Run: it dispatches on the supplied mode
// rather than auto-detecting, mirroring Python run(force_mode=...). Server mode
// serves HTTP (blocking); any serverless mode returns ErrServerlessUnsupported
// (wrapped with the mode name) because Go handles those via the platform
// adapter (AsRouter + pkg/lambda), not inline. This is a Go-port addition
// documented in PORT_ADDITIONS.md.
func (a *AgentBase) RunWithMode(mode swml.ExecutionMode) error {
	switch mode {
	case swml.ModeServer:
		return a.RunContext(context.Background())
	case swml.ModeCGI:
		// CGI is invoked once per request by the CGI host, which hands the
		// request off via the process environment + stdin and expects the
		// response on stdout, then the process exits. This is a real dispatch
		// (mirrors Python's serverless_mixin CGI branch), not an unsupported
		// mode: serve the request through the agent's http.Handler.
		return serverless.NewHandler(a.AsRouter()).ServeCGI(context.Background())
	case swml.ModeLambda, swml.ModeGoogleCloudFunction, swml.ModeAzureFunction:
		// Lambda / GCF / Azure are driven by the platform runtime's own event
		// loop (lambda.Start / functions.HTTP / the Azure worker), which calls
		// the adapter from main() — Run() cannot host that loop itself. The
		// dispatch happens in the adapter (pkg/lambda, pkg/serverless), wired
		// from main(); Run() returns the descriptive error directing the caller
		// there rather than binding a dead TCP listener.
		return fmt.Errorf("%w (detected mode: %q)", ErrServerlessUnsupported, mode)
	default:
		// Unknown mode: be conservative and serve HTTP (matches Python's
		// run(), whose final else branch falls through to serve()).
		return a.RunContext(context.Background())
	}
}

// RunContext is the context-aware form of Run. It blocks serving HTTP exactly
// like Run, but when ctx is cancelled (or its deadline passes) it triggers the
// same graceful HTTP shutdown that SetupGracefulShutdown performs on a signal —
// draining in-flight requests — then returns nil. It composes with
// SetupGracefulShutdown: whichever of (ctx, SIGTERM/SIGINT) fires first wins.
//
// This is a Go-port addition (the Python reference's run()/serve loop has no
// caller-supplied cancellation token); documented in PORT_ADDITIONS.md.
func (a *AgentBase) RunContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	// Ensure the shutdown channel exists before we spawn the watcher so a
	// ctx cancellation that races buildAndServe's own initialisation still
	// has a channel to close. buildAndServe reuses a non-nil shutdownCh.
	a.mu.Lock()
	if a.shutdownCh == nil {
		a.shutdownCh = make(chan struct{})
	}
	ch := a.shutdownCh
	a.mu.Unlock()

	// Bridge ctx cancellation onto the existing graceful-shutdown channel
	// (the same one SetupGracefulShutdown closes on a signal). The watcher
	// exits when serving ends so it never leaks past this call.
	watcherDone := make(chan struct{})
	defer close(watcherDone)
	go func() {
		select {
		case <-ctx.Done():
			a.mu.Lock()
			if a.shutdownCh != nil {
				select {
				case <-a.shutdownCh:
					// already closed by a signal / earlier cancellation
				default:
					close(a.shutdownCh)
				}
			}
			a.mu.Unlock()
		case <-ch:
			// graceful shutdown initiated elsewhere (signal handler)
		case <-watcherDone:
		}
	}()

	return a.buildAndServe()
}

// Serve starts the long-running HTTP server for this agent unconditionally,
// mirroring Python's serve() (web_mixin.py:175) which always serves and does
// no execution-mode detection — that is Run()'s job. Use Run for the universal
// auto-detecting entry point; use Serve to force HTTP serving regardless of
// the detected environment. This is a blocking call.
func (a *AgentBase) Serve() error {
	return a.RunContext(context.Background())
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

	user, pass, source := a.Service.GetBasicAuthCredentialsWithSource()
	addr := fmt.Sprintf("%s:%d", a.Service.Host, a.Service.Port)

	a.Logger.Info("serving agent %q on %s%s", a.Name, addr, a.Service.Route)
	a.Logger.Info("auth user: %s", user)

	// First-run auth wall: when the password was auto-generated (no
	// SWML_BASIC_AUTH_PASSWORD env / no WithBasicAuth), it exists only in this
	// process, so a developer has no way to authenticate an incoming request
	// unless we surface it. Print the full basic-auth credentials to stderr ONCE
	// at startup so a first run is actually usable. Suppressed when the password
	// came from the environment or was set explicitly (the developer already
	// knows it) and when logs are suppressed.
	if source == "auto-generated" && !logging.IsSuppressed() {
		fmt.Fprintf(os.Stderr,
			"[signalwire] auto-generated basic-auth credentials for agent %q: "+
				"user=%q password=%q (set SWML_BASIC_AUTH_USER / "+
				"SWML_BASIC_AUTH_PASSWORD or agent.WithBasicAuth(...) to pin them)\n",
			a.Name, user, pass)
	}

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
		// Bound the request-header read so a slow/incomplete header write
		// (Slowloris) cannot pin a connection open indefinitely. net/http
		// applies no header-read deadline by default, so this must be set
		// explicitly. 20s is generous for a well-behaved SignalWire client.
		ReadHeaderTimeout: 20 * time.Second,
	}

	// If SetupGracefulShutdown is active, spin a goroutine that waits for
	// the shutdown channel and then asks the server to shut down cleanly.
	go func() {
		<-ch
		a.Logger.Info("graceful shutdown: stopping HTTP server")
		if err := server.Shutdown(context.Background()); err != nil {
			a.Logger.Warn("graceful shutdown: server.Shutdown returned: %s", err)
		}
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
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "healthy"}); err != nil {
			a.Logger.Warn("failed to write health response: %s", err)
		}
	})
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ready"}); err != nil {
			a.Logger.Warn("failed to write ready response: %s", err)
		}
	})

	// Main SWML endpoint (with auth + signature on POST)
	swmlRoute := route
	if swmlRoute == "/" {
		mux.HandleFunc("/", a.withAuth(a.withSignedPost(a.handleSWML)))
	} else {
		mux.HandleFunc(swmlRoute, a.withAuth(a.withSignedPost(a.handleSWML)))
		// Also handle without trailing slash
		mux.HandleFunc(swmlRoute+"/", a.withAuth(a.withSignedPost(a.handleSWML)))
	}

	// SWAIG function dispatch endpoint (signed on POST)
	swaigRoute := route + "/swaig"
	if route == "/" {
		swaigRoute = "/swaig"
	}
	mux.HandleFunc(swaigRoute, a.withAuth(a.withSignedPost(a.handleSwaig)))

	// Post-prompt summary endpoint (signed on POST)
	ppRoute := route + "/post_prompt"
	if route == "/" {
		ppRoute = "/post_prompt"
	}
	mux.HandleFunc(ppRoute, a.withAuth(a.withSignedPost(a.handlePostPrompt)))

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

	// Debug routes — matches Python web_mixin._register_routes (web_mixin.py
	// lines 422-472), which unconditionally registers /debug (a SWML dump for
	// inspection, behaves like the main endpoint) and /debug_events (the
	// webhook the platform POSTs debug events to when EnableDebugEvents is on).
	// These are what EnableDebugRoutes documents; in Python that method is a
	// backward-compat no-op because the routes are always registered here.
	debugRoute := route + "/debug"
	debugEventsRoute := route + "/debug_events"
	if route == "/" {
		debugRoute = "/debug"
		debugEventsRoute = "/debug_events"
	}
	mux.HandleFunc(debugRoute, a.withAuth(a.withSignedPost(a.handleSWML)))
	mux.HandleFunc(debugRoute+"/", a.withAuth(a.withSignedPost(a.handleSWML)))
	mux.HandleFunc(debugEventsRoute, a.withAuth(a.withSignedPost(a.handleDebugEvents)))
	mux.HandleFunc(debugEventsRoute+"/", a.withAuth(a.withSignedPost(a.handleDebugEvents)))

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
		mux.HandleFunc(path, a.withAuth(a.withSignedPost(a.handleSWML)))
		mux.HandleFunc(path+"/", a.withAuth(a.withSignedPost(a.handleSWML)))
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
		mux.HandleFunc(path, a.withAuth(a.withSignedPost(a.handleSWML)))
		mux.HandleFunc(path+"/", a.withAuth(a.withSignedPost(a.handleSWML)))
	}

	return mux
}

// maxAgentRequestBody is the maximum request body size (1MB).
const maxAgentRequestBody = 1 << 20

// handleSWML serves the SWML document for the agent.
//
// This is the thin framework adapter: it captures the raw body, method, URL, and
// headers from the *http.Request, then delegates the auth / SIP-307 /
// routing-callback-307 / dynamic-config / render DECISION to the shared
// handleRequestWithContext core — the SAME core the framework-free HandleRequest
// entry point uses. It marshals the returned (status, headers, body) triple back
// into the http.ResponseWriter, including the 307 redirect (Location) and 401
// (WWW-Authenticate). Keeping a single decision core is what guarantees the
// served path (serve() / AsRouter()) behaves identically to HandleRequest —
// notably that a routing callback returning a redirect yields a real 307, not a
// stray 200. rust (swml/router.rs) and php (SWML/Service::dispatchFromGlobals)
// mirror this shape.
func (a *AgentBase) handleSWML(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, maxAgentRequestBody)
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			// Best-effort parse: an undecodable body leaves body nil and
			// downstream rendering falls back to defaults.
			body = nil
		}
	}

	headers := headerStringMap(r.Header)
	status, respHeaders, respBody := a.handleRequestWithContext(r.Method, r.URL.String(), headers, body, r)

	for k, v := range respHeaders {
		w.Header().Set(k, v)
	}
	w.WriteHeader(status)
	if respBody != "" {
		// respBody is a JSON document produced by json.Marshal in
		// handleRequestWithContext (Content-Type application/json) or an empty
		// redirect body — never attacker-controlled markup, so the gosec G705
		// XSS-taint flag is a false positive here.
		if _, err := w.Write([]byte(respBody)); err != nil { //nolint:gosec // G705: JSON body, not HTML; Content-Type is application/json
			a.Logger.Warn("failed to write SWML response: %s", err)
		}
	}
}

// headerStringMap collapses an http.Header into a plain map[string]string keeping
// the first value per key, matching the (method, url, headers, body) primitive
// surface of handleRequestWithContext / HandleRequest.
func headerStringMap(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, v := range h {
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	return out
}

// HandleRequest is the framework-free request-dispatch core for an agent,
// overriding swml.Service.HandleRequest so the agent's full SWML render (prompt,
// tools, dynamic config, on_swml_request modifications) is produced over plain
// primitives instead of *http.Request objects.
//
// It mirrors the Python signalwire.core.agent_base.AgentBase.handle_request
// override: proxy detection, basic-auth, routing-callback (307 redirect), then
// the agent's rendered SWML document with any on_request modifications applied.
//
// Parameters and return follow swml.Service.HandleRequest:
// (method, url, headers, body) -> (status, responseHeaders, bodyString).
func (a *AgentBase) HandleRequest(method string, url string, headers map[string]string, body map[string]any) (int, map[string]string, string) {
	return a.handleRequestWithContext(method, url, headers, body, nil)
}

// handleRequestWithContext is the single decision core behind BOTH the
// framework-free HandleRequest entry point and the served handleSWML handler. It
// performs proxy detection, basic-auth (401), SIP redirect-routing (307),
// routing-callback dispatch (307), dynamic-config rendering, and on_swml_request
// modifications over plain primitives, returning a (status, headers, body_string)
// triple.
//
// The optional r may be nil. When non-nil (the served path) it is threaded into
// the SIP-routing callback, the dynamic-config callback, and on_swml_request so
// those subclass hooks still receive the raw request; when nil (the primitive
// path) SIP routing is skipped (its callbacks are *http.Request-typed) and
// dynamic config / on_swml_request run request-free. Routing-callback 307
// redirect and 401 auth behave identically on both paths — that equivalence is the
// point of routing the served path through here.
func (a *AgentBase) handleRequestWithContext(
	method string,
	url string,
	headers map[string]string,
	body map[string]any,
	r *http.Request,
) (int, map[string]string, string) {
	a.Service.DetectProxyFromPrimitives(url, headers)

	if !a.Service.CheckBasicAuthHeaders(headers) {
		return 401,
			map[string]string{"WWW-Authenticate": "Basic"},
			`{"error":"Unauthorized"}`
	}

	// SIP redirect-routing dispatch. Python web_mixin._handle_request
	// (web_mixin.py:621-635) checks _routing_callbacks; on a non-empty string
	// return, responds with HTTP 307 Temporary Redirect. Go registers SIP
	// callbacks separately (they return a string and carry the raw request), so
	// they only run on the served path where r is available.
	if r != nil && method == http.MethodPost && body != nil {
		sipMatchPath := strings.TrimRight(pathFromURL(url), "/")
		if sipMatchPath == "" {
			sipMatchPath = "/"
		}
		a.mu.RLock()
		sipCb, sipMatched := a.sipRoutingCallbacks[sipMatchPath]
		a.mu.RUnlock()
		if sipMatched {
			if route := sipCb(r, body); route != "" {
				return 307, map[string]string{"Location": route}, ""
			}
		}
	}

	callbackPath := a.Service.CallbackPathForURL(url)

	// Routing-callback dispatch → 307 redirect. Python web_mixin._handle_request
	// (web_mixin.py:628-635): callback_fn(body, headers) -> route|None; on a
	// non-None route, issue an HTTP 307.
	if method == http.MethodPost && len(body) > 0 && callbackPath != "" {
		if cb, ok := a.Service.RoutingCallbackFor(callbackPath); ok {
			if route := cb(body, headerStringMapAnyFromMap(headers)); route != nil {
				return 307, map[string]string{"Location": *route}, ""
			}
		}
	}

	// on_swml_request modifications. Thread the raw request when present so
	// subclasses that override OnSwmlRequest still receive it (served path);
	// the primitive path passes nil.
	modifications := a.OnSwmlRequest(body, callbackPath, r)

	a.mu.RLock()
	hasDynamic := a.dynamicConfigCallback != nil
	a.mu.RUnlock()

	var doc map[string]any
	switch {
	case hasDynamic && r != nil:
		doc = a.handleDynamicConfig(body, r)
	default:
		doc = a.RenderSWML(body, r)
	}
	if len(modifications) > 0 {
		applySwmlModifications(doc, modifications)
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return 500, map[string]string{}, `{"error":"render failed"}`
	}
	return 200, map[string]string{"Content-Type": "application/json"}, string(out)
}

// pathFromURL extracts the path component from a raw URL string, tolerating a
// bare path (no scheme/host). Used to match SIP-routing callbacks registered by
// path.
func pathFromURL(rawURL string) string {
	if u, err := neturl.Parse(rawURL); err == nil && u.Path != "" {
		return u.Path
	}
	// Fall back to the substring before any query/fragment.
	p := rawURL
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	return p
}

// headerStringMapAnyFromMap converts a plain string headers map to map[string]any
// for the RoutingCallback headers argument.
func headerStringMapAnyFromMap(h map[string]string) map[string]any {
	out := make(map[string]any, len(h))
	for k, v := range h {
		out[k] = v
	}
	return out
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

		tokenExpirySecs:  a.tokenExpirySecs,
		sessionManager:   a.sessionManager,
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

	if a.multilingual != nil {
		c.multilingual = make(map[string]any, len(a.multilingual))
		for k, v := range a.multilingual {
			c.multilingual[k] = v
		}
	}

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
// extractSwaigArgs unwraps the SWAIG request body's arguments to the flat args
// dict a tool handler expects, mirroring the python reference
// (core/swml_service.py _handle_swaig_request). Extraction order:
//
//  1. body["argument"] is an object with a non-empty "parsed" array  -> parsed[0]
//  2. body["argument"] is an object with a non-empty "raw" string     -> json.Unmarshal(raw)
//  3. body["argument"] is a bare JSON string                          -> json.Unmarshal(string)
//  4. body["arguments"] is an object (the flat fallback some           -> arguments
//     integrations + the platform both accept)
//  5. otherwise                                                        -> {}
//
// A result that does not decode to a JSON object yields an empty map (never nil),
// so a handler always receives a usable args map.
func extractSwaigArgs(body map[string]any) map[string]any {
	if arg, ok := body["argument"].(map[string]any); ok {
		// (1) parsed[0]
		if parsed, ok := arg["parsed"].([]any); ok && len(parsed) > 0 {
			if first, ok := parsed[0].(map[string]any); ok {
				return first
			}
		}
		// (2) raw (JSON string)
		if raw, ok := arg["raw"].(string); ok && raw != "" {
			var m map[string]any
			if json.Unmarshal([]byte(raw), &m) == nil && m != nil {
				return m
			}
		}
		return map[string]any{}
	}
	// (3) argument as a bare JSON string.
	if argStr, ok := body["argument"].(string); ok && argStr != "" {
		var m map[string]any
		if json.Unmarshal([]byte(argStr), &m) == nil && m != nil {
			return m
		}
	}
	// (4) flat {"arguments": {...}} fallback.
	if flat, ok := body["arguments"].(map[string]any); ok && flat != nil {
		return flat
	}
	// (5) nothing usable.
	return map[string]any{}
}

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

	// Argument extraction. The real platform (mod_openai) POSTs a tool call as
	// the PLATFORM-NESTED shape {"argument": {"parsed": [{...args...}], "raw":
	// "<json>"}} — NOT a flat args dict. Mirror the python reference
	// (core/swml_service.py _handle_swaig_request, ~:832-851): unwrap
	// argument.parsed[0], else argument.raw (JSON), else a flat {"arguments":{...}}
	// fallback, else {}. The pre-GO-7 code read body["argument"] straight into
	// args, so on a real platform call the handler received {"parsed":..,"raw":..}
	// (i.e. NO real args) — SWAIG-HTTP fixture PSDK-7's platform_nested case.
	args := extractSwaigArgs(body)

	result, err := a.OnFunctionCall(funcName, args, body)
	if err != nil {
		a.Logger.Error("function call %q failed: %s", funcName, err)
		errResult := swaig.NewFunctionResult(fmt.Sprintf("Error: %s", err))
		w.Header().Set("Content-Type", "application/json")
		if encErr := json.NewEncoder(w).Encode(errResult.ToMap()); encErr != nil {
			a.Logger.Warn("failed to write function-error response: %s", encErr)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if result != nil {
		if err := json.NewEncoder(w).Encode(result); err != nil {
			a.Logger.Warn("failed to write function result: %s", err)
		}
	} else {
		if err := json.NewEncoder(w).Encode(swaig.NewFunctionResult("ok").ToMap()); err != nil {
			a.Logger.Warn("failed to write function result: %s", err)
		}
	}
}

// findSummary extracts a structured summary from a post-prompt request body,
// mirroring Python AgentBase._find_summary_in_post_data (agent_base.py:1494)
// and the TypeScript AgentBase.findSummary. The extraction order is exact:
//
//  1. body["summary"]
//  2. body["post_prompt_data"]["parsed"][0] when "parsed" is a non-empty array
//  3. body["post_prompt_data"]["raw"] parsed as JSON; on parse failure the raw
//     value itself ("raw-as-both" in Python/TS)
//
// Returns nil when no summary is present. Because the OnSummary callback's
// first argument is typed map[string]any, a successful extraction that is not
// a JSON object (e.g. a non-JSON raw string in the raw-as-both branch, or a
// scalar/array) is returned as nil — the raw body is always available as the
// callback's second argument regardless.
func findSummary(body map[string]any) map[string]any {
	if body == nil {
		return nil
	}

	if s, ok := body["summary"]; ok {
		if m, ok := s.(map[string]any); ok {
			return m
		}
		return nil
	}

	pdata, ok := body["post_prompt_data"].(map[string]any)
	if !ok {
		return nil
	}

	if parsed, ok := pdata["parsed"].([]any); ok && len(parsed) > 0 {
		if m, ok := parsed[0].(map[string]any); ok {
			return m
		}
		return nil
	}

	if rawVal, ok := pdata["raw"]; ok && rawVal != nil {
		// Already a structured object — return as-is.
		if m, ok := rawVal.(map[string]any); ok {
			return m
		}
		// String raw text: try to parse JSON. On success return the object;
		// on failure fall back to the raw value (Python/TS "raw-as-both"),
		// which here can only surface a JSON object.
		if rawStr, ok := rawVal.(string); ok {
			var parsed map[string]any
			if err := json.Unmarshal([]byte(rawStr), &parsed); err == nil {
				return parsed
			}
		}
	}

	return nil
}

// handleDebugEvents receives debug-event webhooks the platform POSTs back to
// the agent when debug events are enabled (EnableDebugEvents). It parses the
// JSON body and dispatches it to the registered OnDebugEvent handler.
//
// Python equivalent: web_mixin._handle_debug_events_request (web_mixin.py:1081)
// which parses the body and invokes self._debug_event_handler(event_type, body).
// The Go DebugEventHandler takes the event body map (matching the TypeScript
// onDebugEvent(body) shape); the raw body is passed through unchanged.
func (a *AgentBase) handleDebugEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"POST required"}`, http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAgentRequestBody)
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		// Match Python: an unparseable body yields an error status but not a
		// hard HTTP failure.
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "invalid JSON"}); err != nil {
			a.Logger.Warn("failed to write debug_events error response: %s", err)
		}
		return
	}

	a.mu.RLock()
	handler := a.debugEventHandler
	a.mu.RUnlock()

	if handler != nil {
		handler(body)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		a.Logger.Warn("failed to write debug_events response: %s", err)
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
		cb(findSummary(body), body)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		a.Logger.Warn("failed to write postprompt response: %s", err)
	}
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
	if err := json.NewEncoder(w).Encode(map[string]any{
		"status":          "success",
		"conversation_id": conversationID,
		"new_input":       false,
		"messages":        []any{},
	}); err != nil {
		a.Logger.Warn("failed to write check_for_input response: %s", err)
	}
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
			// JSON error body {"error":"Unauthorized"} — matches Python's 401
			// challenge (auth_mixin._send_lambda_auth_challenge and the FastAPI
			// path both return json {"error":"Unauthorized"}) and Go's own
			// framework-free SWMLService.HandleRequest 401. http.Error would emit
			// a plain-text "Unauthorized\n" body, diverging from every other port.
			w.Header().Set("WWW-Authenticate", `Basic realm="Agent"`)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			if _, err := w.Write([]byte(`{"error":"Unauthorized"}`)); err != nil {
				a.Logger.Warn("failed to write 401 body: %s", err)
			}
			return
		}

		next(w, r)
	}
}

// withSignedPost wraps a handler so that POST requests are gated by
// security.WebhookMiddleware (when a signing key is configured) and
// non-POST requests pass through untouched. This is the right shape for
// the SWML and /swaig endpoints, which legitimately serve GET (health-style
// document fetches and SWAIG schema introspection respectively).
//
// When no signing key is configured this is a passthrough — startup logs
// the disabled-validation warning so operators are aware.
func (a *AgentBase) withSignedPost(next http.HandlerFunc) http.HandlerFunc {
	if a.signingKey == "" {
		return next
	}
	mw := security.WebhookMiddleware(a.signingKey, &security.WebhookOpts{
		TrustProxy:   a.signingKeyTrustProxy,
		ProxyURLBase: a.proxyURLBase,
	})
	wrapped := mw(next)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next(w, r)
			return
		}
		wrapped.ServeHTTP(w, r)
	}
}
