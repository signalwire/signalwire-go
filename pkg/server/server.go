// Package server provides AgentServer for hosting multiple AI agents on a
// single HTTP server with route-based dispatch.
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/logging"
	"github.com/signalwire/signalwire-go/pkg/swml"
)

// ---------------------------------------------------------------------------
// AgentServer
// ---------------------------------------------------------------------------

// AgentServer hosts multiple agents on a single HTTP server with route-based
// dispatch.  Each registered agent is mounted at its own route prefix and
// exposed via the agent's AsRouter() handler.
type AgentServer struct {
	mu     sync.RWMutex
	agents map[string]*agent.AgentBase // route -> agent
	order  []string                    // insertion order

	host string
	port int

	logger *logging.Logger

	// SIP routing
	sipEnabled   bool
	sipRoute     string
	sipUsernames map[string]string // sip username -> agent route

	// Static files
	staticDirs map[string]string // route -> directory
}

// ---------------------------------------------------------------------------
// ServerOption functional options
// ---------------------------------------------------------------------------

// ServerOption configures an AgentServer during construction.
type ServerOption func(*AgentServer)

// WithServerHost sets the listen address for the server.
func WithServerHost(host string) ServerOption {
	return func(s *AgentServer) { s.host = host }
}

// WithServerPort sets the listen port for the server.
func WithServerPort(port int) ServerOption {
	return func(s *AgentServer) { s.port = port }
}

// WithLogLevel sets the global log level for the server.
// Accepted values (case-insensitive): "debug", "info", "warn", "warning",
// "error", "off".  Mirrors Python AgentServer(log_level=...) behavior: the
// level is applied globally via logging.SetGlobalLevel so all loggers in the
// process are affected.  The default level is "info".
func WithLogLevel(level string) ServerOption {
	return func(s *AgentServer) {
		logging.SetGlobalLevel(logging.ParseLevel(level))
	}
}

// ---------------------------------------------------------------------------
// RunOption functional options
// ---------------------------------------------------------------------------

// RunOption overrides server settings at run time.
type RunOption func(*AgentServer)

// WithRunHost overrides the listen address when calling Run.
func WithRunHost(host string) RunOption {
	return func(s *AgentServer) { s.host = host }
}

// WithRunPort overrides the listen port when calling Run.
func WithRunPort(port int) RunOption {
	return func(s *AgentServer) { s.port = port }
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

// NewAgentServer creates a new AgentServer with the given options.
// Default host is "0.0.0.0" and default port is 3000.
func NewAgentServer(opts ...ServerOption) *AgentServer {
	s := &AgentServer{
		agents:       make(map[string]*agent.AgentBase),
		order:        make([]string, 0),
		host:         "0.0.0.0",
		port:         3000,
		sipUsernames: make(map[string]string),
		staticDirs:   make(map[string]string),
	}

	for _, opt := range opts {
		opt(s)
	}

	s.logger = logging.New("AgentServer")

	return s
}

// ---------------------------------------------------------------------------
// Agent registration
// ---------------------------------------------------------------------------

// Register adds an agent to the server at the given route.  If route is
// empty the agent's configured route (via WithRoute) is used instead.
// The route is normalised to always start with "/".
func (s *AgentServer) Register(a *agent.AgentBase, route string) {
	if route == "" {
		route = a.GetRoute()
	}
	if route == "" {
		route = "/"
	}
	// Ensure the route starts with "/"
	if route[0] != '/' {
		route = "/" + route
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.agents[route]; !exists {
		s.order = append(s.order, route)
	}
	s.agents[route] = a

	s.logger.Info("registered agent %q at route %s", a.GetName(), route)
}

// Unregister removes the agent at the given route.  Returns true if an agent
// was found and removed.
func (s *AgentServer) Unregister(route string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.agents[route]; !exists {
		return false
	}

	delete(s.agents, route)

	// Remove from insertion-order slice
	for i, r := range s.order {
		if r == route {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}

	s.logger.Info("unregistered agent at route %s", route)
	return true
}

// AgentEntry pairs a route with its agent for listing purposes.
type AgentEntry struct {
	Route string
	Agent *agent.AgentBase
}

// GetAgents returns all registered agents in insertion order.
func (s *AgentServer) GetAgents() []AgentEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]AgentEntry, 0, len(s.order))
	for _, route := range s.order {
		if a, ok := s.agents[route]; ok {
			result = append(result, AgentEntry{Route: route, Agent: a})
		}
	}
	return result
}

// GetAgent returns the agent registered at the given route, or nil if none.
func (s *AgentServer) GetAgent(route string) *agent.AgentBase {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.agents[route]
}

// ---------------------------------------------------------------------------
// SIP routing
// ---------------------------------------------------------------------------

// SetupSipRouting enables a central SIP routing endpoint.  When autoMap is
// true, all currently registered agents are automatically mapped using their
// route as the SIP username.
func (s *AgentServer) SetupSipRouting(route string, autoMap bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sipEnabled = true
	s.sipRoute = route
	if s.sipRoute == "" {
		s.sipRoute = "/sip"
	}

	if autoMap {
		for r := range s.agents {
			// Use route without leading "/" as username
			username := r
			if len(username) > 1 && username[0] == '/' {
				username = username[1:]
			}
			s.sipUsernames[username] = r
		}
	}

	s.logger.Info("SIP routing enabled at %s (autoMap=%v)", s.sipRoute, autoMap)
}

// RegisterSipUsername maps a SIP username to an agent route so that
// inbound SIP calls for that username are routed to the correct agent.
func (s *AgentServer) RegisterSipUsername(username, route string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sipUsernames[username] = route
	s.logger.Info("SIP username %q mapped to route %s", username, route)
}

// ---------------------------------------------------------------------------
// Static files
// ---------------------------------------------------------------------------

// ServeStaticFiles registers a directory to be served at the given route.
func (s *AgentServer) ServeStaticFiles(directory, route string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.staticDirs[route] = directory
	s.logger.Info("serving static files from %q at %s", directory, route)
}

// ---------------------------------------------------------------------------
// Global routing callbacks
// ---------------------------------------------------------------------------

// RegisterGlobalRoutingCallback registers a routing callback across all
// currently-registered agents at the given path.  The callback fires on every
// incoming request to that path and can return an SWML document override (or
// nil to fall through to the agent's default response).
//
// This is the Go equivalent of Python's
// AgentServer.register_global_routing_callback(callback_fn, path).
func (s *AgentServer) RegisterGlobalRoutingCallback(path string, cb swml.RoutingCallback) {
	// Normalise the path to start with "/"
	if len(path) == 0 || path[0] != '/' {
		path = "/" + path
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, a := range s.agents {
		a.RegisterRoutingCallback(path, cb)
	}

	s.logger.Info("registered global routing callback at %s on %d agent(s)", path, len(s.agents))
}

// ---------------------------------------------------------------------------
// HTTP server
// ---------------------------------------------------------------------------

// Run starts the HTTP server.  This is a blocking call.  Optional RunOption
// values can override host and port at start time.
//
// Serverless dispatch: unlike Python's AgentServer.run() which auto-detects
// CGI and Lambda environments, Run() is HTTP-server-only.  For AWS Lambda
// deployments use the pkg/lambda package instead.  CGI mode has no Go
// equivalent; deploy as a standard HTTP service behind a reverse proxy.
func (s *AgentServer) Run(opts ...RunOption) error {
	for _, opt := range opts {
		opt(s)
	}

	mux := s.buildMux()

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	s.logger.Info("AgentServer starting on %s", addr)

	s.mu.RLock()
	for _, route := range s.order {
		if a, ok := s.agents[route]; ok {
			s.logger.Info("  %s -> %s", route, a.GetName())
		}
	}
	s.mu.RUnlock()

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	return srv.ListenAndServe()
}

// buildMux assembles an http.ServeMux with all agent routes, health checks,
// SIP routing, static file serving, and a root index.
func (s *AgentServer) buildMux() *http.ServeMux {
	mux := http.NewServeMux()

	// ---------------------------------------------------------------
	// Health / readiness probes (no auth)
	// ---------------------------------------------------------------
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	})

	// ---------------------------------------------------------------
	// Agent routes
	// ---------------------------------------------------------------
	s.mu.RLock()
	agents := make(map[string]*agent.AgentBase, len(s.agents))
	for k, v := range s.agents {
		agents[k] = v
	}
	sipEnabled := s.sipEnabled
	sipRoute := s.sipRoute
	sipUsernames := make(map[string]string, len(s.sipUsernames))
	for k, v := range s.sipUsernames {
		sipUsernames[k] = v
	}
	staticDirs := make(map[string]string, len(s.staticDirs))
	for k, v := range s.staticDirs {
		staticDirs[k] = v
	}
	order := make([]string, len(s.order))
	copy(order, s.order)
	s.mu.RUnlock()

	for route, a := range agents {
		handler := a.AsRouter()
		// Strip the route prefix before forwarding to the agent handler
		if route != "/" {
			mux.Handle(route+"/", http.StripPrefix(route, handler))
			mux.Handle(route, http.StripPrefix(route, handler))
		} else {
			// Root route agents are handled specially via the index handler
			// so they don't swallow everything.
			mux.Handle("/_root/", http.StripPrefix("/_root", handler))
			mux.Handle("/_root", http.StripPrefix("/_root", handler))
		}
	}

	// ---------------------------------------------------------------
	// SIP routing endpoint
	// ---------------------------------------------------------------
	if sipEnabled && sipRoute != "" {
		mux.HandleFunc(sipRoute, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}

			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}

			// Extract SIP username from the request body
			username := extractSipUsername(body)
			if username == "" {
				http.Error(w, "missing SIP username", http.StatusBadRequest)
				return
			}

			agentRoute, ok := sipUsernames[username]
			if !ok {
				http.Error(w, fmt.Sprintf("no agent for SIP username %q", username), http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"action": "redirect",
				"route":  agentRoute,
			})
		})
	}

	// ---------------------------------------------------------------
	// Static file routes
	// ---------------------------------------------------------------
	for route, dir := range staticDirs {
		fs := http.FileServer(http.Dir(dir))
		handler := http.StripPrefix(route, fs)

		// Add security headers
		secured := addSecurityHeaders(handler)
		mux.Handle(route+"/", secured)
		mux.Handle(route, secured)
	}

	// ---------------------------------------------------------------
	// Root index: list registered agents
	// ---------------------------------------------------------------
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only handle exact "/" path
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		type agentInfo struct {
			Route string `json:"route"`
			Name  string `json:"name"`
		}

		entries := make([]agentInfo, 0, len(order))
		for _, route := range order {
			if a, ok := agents[route]; ok {
				entries = append(entries, agentInfo{
					Route: route,
					Name:  a.GetName(),
				})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"agents": entries,
		})
	})

	return mux
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// extractSipUsername extracts the SIP username from an inbound SIP routing
// request body.  It checks common field paths used by SignalWire.
func extractSipUsername(body map[string]any) string {
	// Try top-level "sip_username"
	if u, ok := body["sip_username"].(string); ok && u != "" {
		return u
	}
	// Try nested "call.to" or "to"
	if to, ok := body["to"].(string); ok && to != "" {
		return to
	}
	if call, ok := body["call"].(map[string]any); ok {
		if to, ok := call["to"].(string); ok && to != "" {
			return to
		}
	}
	return ""
}

// addSecurityHeaders wraps an http.Handler to include standard security
// response headers.
func addSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

