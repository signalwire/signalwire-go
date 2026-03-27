package swml

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/signalwire/signalwire-go/pkg/logging"
)

// RoutingCallback is a function called on incoming requests to customize responses.
// It receives the request and request body, and returns an optional SWML JSON override.
// If it returns nil, the default document is used.
type RoutingCallback func(r *http.Request, body map[string]any) map[string]any

// Service is the base SWML service that manages documents, HTTP endpoints, and auth.
// It provides auto-vivified verb methods driven by the SWML schema.
type Service struct {
	mu sync.RWMutex

	Name   string
	Route  string
	Host   string
	Port   int
	Logger *logging.Logger

	// Auth
	basicAuthUser     string
	basicAuthPassword string

	// SWML document
	document *Document

	// Schema for verb validation
	schema *Schema

	// Proxy detection
	proxyURLBase string

	// Routing callbacks
	routingCallbacks map[string]RoutingCallback

	// Server state
	server  *http.Server
	running bool

	// Verb method cache: maps verb name to whether it exists in schema
	verbCache map[string]bool
}

// ServiceOption is a functional option for configuring a Service.
type ServiceOption func(*Service)

// WithName sets the service name.
func WithName(name string) ServiceOption {
	return func(s *Service) { s.Name = name }
}

// WithRoute sets the HTTP route path.
func WithRoute(route string) ServiceOption {
	return func(s *Service) { s.Route = strings.TrimRight(route, "/") }
}

// WithHost sets the HTTP server bind host.
func WithHost(host string) ServiceOption {
	return func(s *Service) { s.Host = host }
}

// WithPort sets the HTTP server port.
func WithPort(port int) ServiceOption {
	return func(s *Service) { s.Port = port }
}

// WithBasicAuth sets explicit basic auth credentials.
func WithBasicAuth(user, password string) ServiceOption {
	return func(s *Service) {
		s.basicAuthUser = user
		s.basicAuthPassword = password
	}
}

// NewService creates a new SWML service with the given options.
func NewService(opts ...ServiceOption) *Service {
	s := &Service{
		Name:             "SWML",
		Route:            "/",
		Host:             "0.0.0.0",
		Port:             3000,
		Logger:           logging.New("swml-service"),
		document:         NewDocument(),
		routingCallbacks: make(map[string]RoutingCallback),
		verbCache:        make(map[string]bool),
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Port from env if not explicitly set
	if envPort := os.Getenv("PORT"); envPort != "" {
		var port int
		if _, err := fmt.Sscanf(envPort, "%d", &port); err == nil {
			s.Port = port
		}
	}

	// Auth from env if not explicitly set
	if s.basicAuthUser == "" {
		s.basicAuthUser = os.Getenv("SWML_BASIC_AUTH_USER")
	}
	if s.basicAuthPassword == "" {
		s.basicAuthPassword = os.Getenv("SWML_BASIC_AUTH_PASSWORD")
	}
	// Auto-generate if still empty
	if s.basicAuthUser == "" {
		s.basicAuthUser = s.Name
	}
	if s.basicAuthPassword == "" {
		s.basicAuthPassword = generatePassword()
	}

	// Proxy URL from env
	s.proxyURLBase = os.Getenv("SWML_PROXY_URL_BASE")

	// Load schema
	schema, err := GetSchema()
	if err != nil {
		s.Logger.Warn("failed to load SWML schema: %s", err)
	} else {
		s.schema = schema
		s.Logger.Debug("loaded schema with %d verbs", schema.VerbCount())
	}

	return s
}

// generatePassword creates a random password for auto-generated basic auth.
// Panics if the system has no entropy source — never fall back to a weak password.
func generatePassword() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic("crypto/rand failed: cannot generate secure password: " + err.Error())
	}
	return hex.EncodeToString(bytes)
}

// GetDocument returns the current SWML document.
func (s *Service) GetDocument() *Document {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.document
}

// ResetDocument resets the SWML document to empty.
func (s *Service) ResetDocument() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.document = NewDocument()
}

// GetBasicAuthCredentials returns the (username, password) for basic auth.
func (s *Service) GetBasicAuthCredentials() (string, string) {
	return s.basicAuthUser, s.basicAuthPassword
}

// --- Schema-driven verb methods ---

// ExecuteVerb adds any SWML verb to the document, validated against the schema.
// This is the core method that all verb convenience methods delegate to.
// For most verbs, config should be a map[string]any of verb parameters.
// For "sleep", config should be an integer (milliseconds).
func (s *Service) ExecuteVerb(verbName string, config any) error {
	if s.schema != nil && !s.schema.IsValidVerb(verbName) {
		return fmt.Errorf("unknown SWML verb: %q", verbName)
	}
	return s.document.AddVerb(verbName, config)
}

// ExecuteVerbToSection adds a SWML verb to a named section.
func (s *Service) ExecuteVerbToSection(section, verbName string, config any) error {
	if s.schema != nil && !s.schema.IsValidVerb(verbName) {
		return fmt.Errorf("unknown SWML verb: %q", verbName)
	}
	return s.document.AddVerbToSection(section, verbName, config)
}

// --- Auto-vivified convenience methods for all 38 SWML verbs ---
// These are generated from the schema verb list. The AI verb is included here
// but AgentBase will override it with its own implementation.

// Answer adds the answer verb to the document.
func (s *Service) Answer(config map[string]any) error {
	return s.ExecuteVerb("answer", filterNilValues(config))
}

// Hangup adds the hangup verb.
func (s *Service) Hangup(config map[string]any) error {
	return s.ExecuteVerb("hangup", filterNilValues(config))
}

// Play adds the play verb.
func (s *Service) Play(config map[string]any) error {
	return s.ExecuteVerb("play", filterNilValues(config))
}

// Record adds the record verb.
func (s *Service) Record(config map[string]any) error {
	return s.ExecuteVerb("record", filterNilValues(config))
}

// RecordCall adds the record_call verb.
func (s *Service) RecordCall(config map[string]any) error {
	return s.ExecuteVerb("record_call", filterNilValues(config))
}

// StopRecordCall adds the stop_record_call verb.
func (s *Service) StopRecordCall(config map[string]any) error {
	return s.ExecuteVerb("stop_record_call", filterNilValues(config))
}

// Sleep adds the sleep verb. Duration is in milliseconds.
func (s *Service) Sleep(duration int) error {
	return s.ExecuteVerb("sleep", duration)
}

// Connect adds the connect verb.
func (s *Service) Connect(config map[string]any) error {
	return s.ExecuteVerb("connect", filterNilValues(config))
}

// SendDigits adds the send_digits verb.
func (s *Service) SendDigits(config map[string]any) error {
	return s.ExecuteVerb("send_digits", filterNilValues(config))
}

// SendSMS adds the send_sms verb.
func (s *Service) SendSMS(config map[string]any) error {
	return s.ExecuteVerb("send_sms", filterNilValues(config))
}

// SendFax adds the send_fax verb.
func (s *Service) SendFax(config map[string]any) error {
	return s.ExecuteVerb("send_fax", filterNilValues(config))
}

// ReceiveFax adds the receive_fax verb.
func (s *Service) ReceiveFax(config map[string]any) error {
	return s.ExecuteVerb("receive_fax", filterNilValues(config))
}

// SIPRefer adds the sip_refer verb.
func (s *Service) SIPRefer(config map[string]any) error {
	return s.ExecuteVerb("sip_refer", filterNilValues(config))
}

// AI adds the ai verb. AgentBase overrides this with its own AI rendering.
func (s *Service) AI(config map[string]any) error {
	return s.ExecuteVerb("ai", filterNilValues(config))
}

// AmazonBedrock adds the amazon_bedrock verb.
func (s *Service) AmazonBedrock(config map[string]any) error {
	return s.ExecuteVerb("amazon_bedrock", filterNilValues(config))
}

// Cond adds the cond verb (conditional logic).
func (s *Service) Cond(config map[string]any) error {
	return s.ExecuteVerb("cond", filterNilValues(config))
}

// Switch adds the switch verb.
func (s *Service) Switch(config map[string]any) error {
	return s.ExecuteVerb("switch", filterNilValues(config))
}

// Execute adds the execute verb (run another SWML section).
func (s *Service) Execute(config map[string]any) error {
	return s.ExecuteVerb("execute", filterNilValues(config))
}

// Return adds the return verb.
func (s *Service) Return(config map[string]any) error {
	return s.ExecuteVerb("return", filterNilValues(config))
}

// Goto adds the goto verb.
func (s *Service) Goto(config map[string]any) error {
	return s.ExecuteVerb("goto", filterNilValues(config))
}

// Label adds the label verb.
func (s *Service) Label(config map[string]any) error {
	return s.ExecuteVerb("label", filterNilValues(config))
}

// Set adds the set verb (set variables).
func (s *Service) Set(config map[string]any) error {
	return s.ExecuteVerb("set", filterNilValues(config))
}

// Unset adds the unset verb.
func (s *Service) Unset(config map[string]any) error {
	return s.ExecuteVerb("unset", filterNilValues(config))
}

// Transfer adds the transfer verb.
func (s *Service) Transfer(config map[string]any) error {
	return s.ExecuteVerb("transfer", filterNilValues(config))
}

// Tap adds the tap verb (media tapping).
func (s *Service) Tap(config map[string]any) error {
	return s.ExecuteVerb("tap", filterNilValues(config))
}

// StopTap adds the stop_tap verb.
func (s *Service) StopTap(config map[string]any) error {
	return s.ExecuteVerb("stop_tap", filterNilValues(config))
}

// Denoise adds the denoise verb.
func (s *Service) Denoise(config map[string]any) error {
	return s.ExecuteVerb("denoise", filterNilValues(config))
}

// StopDenoise adds the stop_denoise verb.
func (s *Service) StopDenoise(config map[string]any) error {
	return s.ExecuteVerb("stop_denoise", filterNilValues(config))
}

// JoinRoom adds the join_room verb.
func (s *Service) JoinRoom(config map[string]any) error {
	return s.ExecuteVerb("join_room", filterNilValues(config))
}

// JoinConference adds the join_conference verb.
func (s *Service) JoinConference(config map[string]any) error {
	return s.ExecuteVerb("join_conference", filterNilValues(config))
}

// Prompt adds the prompt verb.
func (s *Service) Prompt(config map[string]any) error {
	return s.ExecuteVerb("prompt", filterNilValues(config))
}

// EnterQueue adds the enter_queue verb.
func (s *Service) EnterQueue(config map[string]any) error {
	return s.ExecuteVerb("enter_queue", filterNilValues(config))
}

// Request adds the request verb (HTTP request).
func (s *Service) Request(config map[string]any) error {
	return s.ExecuteVerb("request", filterNilValues(config))
}

// Pay adds the pay verb.
func (s *Service) Pay(config map[string]any) error {
	return s.ExecuteVerb("pay", filterNilValues(config))
}

// DetectMachine adds the detect_machine verb.
func (s *Service) DetectMachine(config map[string]any) error {
	return s.ExecuteVerb("detect_machine", filterNilValues(config))
}

// LiveTranscribe adds the live_transcribe verb.
func (s *Service) LiveTranscribe(config map[string]any) error {
	return s.ExecuteVerb("live_transcribe", filterNilValues(config))
}

// LiveTranslate adds the live_translate verb.
func (s *Service) LiveTranslate(config map[string]any) error {
	return s.ExecuteVerb("live_translate", filterNilValues(config))
}

// UserEvent adds the user_event verb.
func (s *Service) UserEvent(config map[string]any) error {
	return s.ExecuteVerb("user_event", filterNilValues(config))
}

// --- HTTP server ---

// GetFullURL returns the full URL for this service including auth.
func (s *Service) GetFullURL(includeAuth bool) string {
	scheme := "http"
	host := s.Host
	if host == "0.0.0.0" {
		host = "localhost"
	}

	if s.proxyURLBase != "" {
		base := strings.TrimRight(s.proxyURLBase, "/")
		if includeAuth {
			return fmt.Sprintf("%s%s", insertAuth(base, s.basicAuthUser, s.basicAuthPassword), s.Route)
		}
		return base + s.Route
	}

	base := fmt.Sprintf("%s://%s:%d", scheme, host, s.Port)
	if includeAuth {
		return fmt.Sprintf("%s%s", insertAuth(base, s.basicAuthUser, s.basicAuthPassword), s.Route)
	}
	return base + s.Route
}

// RegisterRoutingCallback registers a callback for a specific path.
func (s *Service) RegisterRoutingCallback(path string, cb RoutingCallback) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.routingCallbacks[path] = cb
}

// OnRequest generates the SWML response for an incoming request.
// It checks routing callbacks first, then returns the default document.
func (s *Service) OnRequest(requestData map[string]any, callbackPath string) map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check routing callbacks
	if callbackPath != "" {
		if cb, ok := s.routingCallbacks[callbackPath]; ok {
			result := cb(nil, requestData)
			if result != nil {
				return result
			}
		}
	}

	return s.document.ToMap()
}

// Render returns the SWML document as a JSON string.
func (s *Service) Render() (string, error) {
	return s.document.Render()
}

// RenderPretty returns the SWML document as an indented JSON string.
func (s *Service) RenderPretty() (string, error) {
	return s.document.RenderPretty()
}

// ExtractSIPUsername extracts a SIP username from a request body.
func ExtractSIPUsername(body map[string]any) string {
	// Look in call.to field for SIP URI
	callData, ok := body["call"].(map[string]any)
	if !ok {
		return ""
	}
	to, ok := callData["to"].(string)
	if !ok {
		return ""
	}
	// Parse SIP URI: sip:username@domain
	if strings.HasPrefix(to, "sip:") {
		to = to[4:]
	}
	if idx := strings.Index(to, "@"); idx > 0 {
		return to[:idx]
	}
	return to
}

// --- Helpers ---

// filterNilValues removes nil values from a map, matching Python's behavior
// of skipping None kwargs in verb methods.
func filterNilValues(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		if v != nil {
			result[k] = v
		}
	}
	return result
}

// insertAuth inserts basic auth credentials into a URL.
func insertAuth(baseURL, user, password string) string {
	// Insert user:pass@ after the scheme://
	if idx := strings.Index(baseURL, "://"); idx >= 0 {
		scheme := baseURL[:idx+3]
		rest := baseURL[idx+3:]
		return fmt.Sprintf("%s%s:%s@%s", scheme, user, password, rest)
	}
	return baseURL
}

// Serve starts the HTTP server. This is a blocking call.
func (s *Service) Serve() error {
	mux := s.buildMux()

	s.mu.Lock()
	s.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.Host, s.Port),
		Handler: mux,
	}
	s.running = true
	s.mu.Unlock()

	s.Logger.Info("serving on %s:%d%s", s.Host, s.Port, s.Route)
	s.Logger.Info("auth user: %s", s.basicAuthUser)

	return s.server.ListenAndServe()
}

// Stop gracefully stops the HTTP server.
func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.server != nil {
		s.running = false
		return s.server.Close()
	}
	return nil
}

// buildMux creates the HTTP handler with auth middleware and routes.
func (s *Service) buildMux() *http.ServeMux {
	mux := http.NewServeMux()

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
	route := s.Route
	if route == "" {
		route = "/"
	}
	mux.HandleFunc(route, s.withSecurity(s.handleSWML))

	return mux
}

// maxRequestBody is the maximum allowed request body size (1MB).
const maxRequestBody = 1 << 20

// handleSWML serves the SWML document.
func (s *Service) handleSWML(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
		json.NewDecoder(r.Body).Decode(&body)
	}

	doc := s.OnRequest(body, r.URL.Path)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

// withSecurity wraps a handler with auth and security headers.
func (s *Service) withSecurity(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cache-Control", "no-store")

		// Basic auth check (timing-safe comparison)
		user, pass, ok := r.BasicAuth()
		userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(s.basicAuthUser)) == 1
		passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(s.basicAuthPassword)) == 1
		if !ok || !userMatch || !passMatch {
			w.Header().Set("WWW-Authenticate", `Basic realm="SWML Service"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}
