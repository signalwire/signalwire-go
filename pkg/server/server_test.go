package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/logging"
)

// TestWithLogLevel_EnumOrString proves the user-facing WithLogLevel option sets
// the IDENTICAL global level whether given the typed logging.LevelNameDebug
// constant or the bare "debug" string (the Python reference uses a plain str).
// Real behavior — the option mutates the live global logging level.
func TestWithLogLevel_EnumOrString(t *testing.T) {
	original := logging.GetGlobalLevel()
	defer logging.SetGlobalLevel(original)

	// Typed constant path.
	_ = NewAgentServer(WithLogLevel(logging.LevelNameDebug))
	viaConst := logging.GetGlobalLevel()

	// Reset, then the bare-string path.
	logging.SetGlobalLevel(original)
	_ = NewAgentServer(WithLogLevel("debug"))
	viaStr := logging.GetGlobalLevel()

	if viaConst != logging.LevelDebug {
		t.Errorf("WithLogLevel(LevelNameDebug) set %v, want LevelDebug", viaConst)
	}
	if viaConst != viaStr {
		t.Errorf("typed const (%v) and string (%v) set different global levels", viaConst, viaStr)
	}
}

// ---------------------------------------------------------------------------
// Constructor tests
// ---------------------------------------------------------------------------

func TestNewAgentServer_Defaults(t *testing.T) {
	s := NewAgentServer()
	if s == nil {
		t.Fatal("expected non-nil server")
	}
	if s.host != "0.0.0.0" {
		t.Errorf("expected host=0.0.0.0, got %q", s.host)
	}
	if s.port != 3000 {
		t.Errorf("expected port=3000, got %d", s.port)
	}
	if s.logger == nil {
		t.Error("expected non-nil logger")
	}
	if s.agents == nil {
		t.Error("expected non-nil agents map")
	}
	if s.sipUsernames == nil {
		t.Error("expected non-nil sipUsernames map")
	}
	if s.staticDirs == nil {
		t.Error("expected non-nil staticDirs map")
	}
}

func TestNewAgentServer_WithOptions(t *testing.T) {
	s := NewAgentServer(
		WithServerHost("127.0.0.1"),
		WithServerPort(8080),
	)
	if s.host != "127.0.0.1" {
		t.Errorf("expected host=127.0.0.1, got %q", s.host)
	}
	if s.port != 8080 {
		t.Errorf("expected port=8080, got %d", s.port)
	}
}

// ---------------------------------------------------------------------------
// Register / Unregister tests
// ---------------------------------------------------------------------------

func TestRegister_ExplicitRoute(t *testing.T) {
	s := NewAgentServer()
	a := agent.NewAgentBase(agent.WithName("Bot1"))

	s.Register(a, "/bot1")

	got := s.GetAgent("/bot1")
	if got == nil {
		t.Fatal("expected agent at /bot1")
	}
	if got.GetName() != "Bot1" {
		t.Errorf("expected name=Bot1, got %q", got.GetName())
	}
}

func TestRegister_AgentDefaultRoute(t *testing.T) {
	s := NewAgentServer()
	a := agent.NewAgentBase(agent.WithName("Bot2"), agent.WithRoute("/mybot"))

	s.Register(a, "")

	got := s.GetAgent("/mybot")
	if got == nil {
		t.Fatal("expected agent at /mybot when route is empty")
	}
}

func TestRegister_NormalisesRoute(t *testing.T) {
	s := NewAgentServer()
	a := agent.NewAgentBase(agent.WithName("Bot3"))

	s.Register(a, "no-slash")

	got := s.GetAgent("/no-slash")
	if got == nil {
		t.Fatal("expected route to be normalised with leading /")
	}
}

func TestRegister_OverwritesSameRoute(t *testing.T) {
	s := NewAgentServer()
	a1 := agent.NewAgentBase(agent.WithName("First"))
	a2 := agent.NewAgentBase(agent.WithName("Second"))

	s.Register(a1, "/shared")
	s.Register(a2, "/shared")

	got := s.GetAgent("/shared")
	if got == nil {
		t.Fatal("expected agent at /shared")
	}
	if got.GetName() != "Second" {
		t.Errorf("expected overwritten agent name=Second, got %q", got.GetName())
	}

	// Should not duplicate the route in GetAgents
	agents := s.GetAgents()
	count := 0
	for _, e := range agents {
		if e.Route == "/shared" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 entry for /shared, got %d", count)
	}
}

func TestUnregister_Existing(t *testing.T) {
	s := NewAgentServer()
	a := agent.NewAgentBase(agent.WithName("Bot"))

	s.Register(a, "/bot")
	removed := s.Unregister("/bot")

	if !removed {
		t.Error("expected Unregister to return true")
	}
	if s.GetAgent("/bot") != nil {
		t.Error("expected agent to be removed")
	}
	if len(s.GetAgents()) != 0 {
		t.Error("expected empty agent list after unregister")
	}
}

func TestUnregister_NonExistent(t *testing.T) {
	s := NewAgentServer()
	removed := s.Unregister("/nope")
	if removed {
		t.Error("expected Unregister to return false for non-existent route")
	}
}

// ---------------------------------------------------------------------------
// GetAgents tests
// ---------------------------------------------------------------------------

func TestGetAgents_InsertionOrder(t *testing.T) {
	s := NewAgentServer()
	s.Register(agent.NewAgentBase(agent.WithName("Alpha")), "/alpha")
	s.Register(agent.NewAgentBase(agent.WithName("Beta")), "/beta")
	s.Register(agent.NewAgentBase(agent.WithName("Gamma")), "/gamma")

	agents := s.GetAgents()
	if len(agents) != 3 {
		t.Fatalf("expected 3 agents, got %d", len(agents))
	}
	expected := []string{"/alpha", "/beta", "/gamma"}
	for i, e := range agents {
		if e.Route != expected[i] {
			t.Errorf("position %d: expected route=%s, got %s", i, expected[i], e.Route)
		}
	}
}

func TestGetAgents_Empty(t *testing.T) {
	s := NewAgentServer()
	agents := s.GetAgents()
	if len(agents) != 0 {
		t.Errorf("expected 0 agents, got %d", len(agents))
	}
}

// ---------------------------------------------------------------------------
// GetAgent tests
// ---------------------------------------------------------------------------

func TestGetAgent_Exists(t *testing.T) {
	s := NewAgentServer()
	a := agent.NewAgentBase(agent.WithName("Finder"))
	s.Register(a, "/find-me")

	got := s.GetAgent("/find-me")
	if got == nil {
		t.Fatal("expected to find agent")
	}
	if got.GetName() != "Finder" {
		t.Errorf("expected name=Finder, got %q", got.GetName())
	}
}

func TestGetAgent_NotFound(t *testing.T) {
	s := NewAgentServer()
	got := s.GetAgent("/nonexistent")
	if got != nil {
		t.Error("expected nil for non-existent route")
	}
}

// ---------------------------------------------------------------------------
// SIP routing tests
// ---------------------------------------------------------------------------

func TestSetupSipRouting(t *testing.T) {
	s := NewAgentServer()
	s.Register(agent.NewAgentBase(agent.WithName("SipBot")), "/sipbot")

	s.SetupSIPRouting("/sip", false)

	if !s.sipEnabled {
		t.Error("expected sipEnabled=true")
	}
	if s.sipRoute != "/sip" {
		t.Errorf("expected sipRoute=/sip, got %q", s.sipRoute)
	}
}

func TestSetupSipRouting_AutoMap(t *testing.T) {
	s := NewAgentServer()
	s.Register(agent.NewAgentBase(agent.WithName("Sales")), "/sales")
	s.Register(agent.NewAgentBase(agent.WithName("Support")), "/support")

	s.SetupSIPRouting("", true)

	if s.sipRoute != "/sip" {
		t.Errorf("expected default sipRoute=/sip, got %q", s.sipRoute)
	}

	// Auto-mapped usernames should be route without leading "/"
	if s.sipUsernames["sales"] != "/sales" {
		t.Errorf("expected sipUsernames[sales]=/sales, got %q", s.sipUsernames["sales"])
	}
	if s.sipUsernames["support"] != "/support" {
		t.Errorf("expected sipUsernames[support]=/support, got %q", s.sipUsernames["support"])
	}
}

func TestRegisterSipUsername(t *testing.T) {
	s := NewAgentServer()
	s.RegisterSIPUsername("alice", "/alice-agent")

	if s.sipUsernames["alice"] != "/alice-agent" {
		t.Errorf("expected alice -> /alice-agent, got %q", s.sipUsernames["alice"])
	}
}

// TestSIPRouting_ServedPathExtractsAndRoutes is Tier-2 behavioral contract #6:
// a registered SIP username, POSTed to the SERVED /sip endpoint, must (a) fire
// the SIP handler, (b) have its username EXTRACTED from the SIP body (call.to =
// "sip:<user>@domain"), and (c) be ROUTED to the mapped agent route. A
// stored-but-unconsulted mapping would never route — the endpoint would 404 or
// ignore the mapping. This exercises the mapping through the served mux, not
// just the in-memory map assertion the other SIP tests make.
func TestSIPRouting_ServedPathExtractsAndRoutes(t *testing.T) {
	s := NewAgentServer()
	s.Register(agent.NewAgentBase(agent.WithName("Support"), agent.WithRoute("/support")), "/support")
	s.SetupSIPRouting("/sip", false)
	s.RegisterSIPUsername("support", "/support")

	mux := s.buildMux()

	// SIP-shaped body: the username lives in call.to as a SIP URI.
	body := `{"call":{"to":"sip:support@example.sip.signalwire.com","from":"sip:caller@x"}}`
	req := httptest.NewRequest("POST", "/sip", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("served /sip: status = %d, want 200 (routed); body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("SIP response not JSON: %v; body=%s", err, rr.Body.String())
	}
	// (c) routed to the mapped agent route.
	if resp["route"] != "/support" {
		t.Errorf("routed route = %v, want /support (username 'support' extracted from call.to and consulted)", resp["route"])
	}
	if resp["action"] != "redirect" {
		t.Errorf("action = %v, want redirect", resp["action"])
	}
}

// TestSIPRouting_ServedPathUnregisteredUsername confirms the served /sip
// endpoint actually consults the mapping: an unregistered extracted username
// yields 404, not a route. A stored-but-unconsulted implementation could not
// distinguish registered from unregistered.
func TestSIPRouting_ServedPathUnregisteredUsername(t *testing.T) {
	s := NewAgentServer()
	s.Register(agent.NewAgentBase(agent.WithName("Support"), agent.WithRoute("/support")), "/support")
	s.SetupSIPRouting("/sip", false)
	s.RegisterSIPUsername("support", "/support")

	mux := s.buildMux()

	body := `{"call":{"to":"sip:nobody@example.sip.signalwire.com"}}`
	req := httptest.NewRequest("POST", "/sip", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("unregistered username: status = %d, want 404 (mapping consulted, no match)", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Static file serving tests
// ---------------------------------------------------------------------------

func TestServeStaticFiles_Setup(t *testing.T) {
	s := NewAgentServer()
	s.ServeStaticFiles("/tmp/static", "/assets")

	if s.staticDirs["/assets"] != "/tmp/static" {
		t.Errorf("expected staticDirs[/assets]=/tmp/static, got %q", s.staticDirs["/assets"])
	}
}

func TestServeStaticFiles_HTTP(t *testing.T) {
	// Create a temp directory with a test file
	dir := t.TempDir()
	testFile := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	s := NewAgentServer()
	s.ServeStaticFiles(dir, "/static")

	mux := s.buildMux()
	req := httptest.NewRequest("GET", "/static/hello.txt", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "hello world") {
		t.Errorf("expected file content, got %q", body)
	}

	// Check security headers
	if rr.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("expected X-Content-Type-Options header")
	}
	if rr.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("expected X-Frame-Options header")
	}
	if rr.Header().Get("Cache-Control") != "no-store" {
		t.Error("expected Cache-Control header")
	}
}

// ---------------------------------------------------------------------------
// Health / readiness endpoint tests
// ---------------------------------------------------------------------------

func TestHTTP_HealthEndpoint(t *testing.T) {
	s := NewAgentServer()
	mux := s.buildMux()

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}
	if body["status"] != "healthy" {
		t.Errorf("expected status=healthy, got %q", body["status"])
	}
}

func TestHTTP_ReadyEndpoint(t *testing.T) {
	s := NewAgentServer()
	mux := s.buildMux()

	req := httptest.NewRequest("GET", "/ready", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode ready response: %v", err)
	}
	if body["status"] != "ready" {
		t.Errorf("expected status=ready, got %q", body["status"])
	}
}

// ---------------------------------------------------------------------------
// Root index endpoint tests
// ---------------------------------------------------------------------------

func TestHTTP_RootIndex_ListsAgents(t *testing.T) {
	s := NewAgentServer()
	s.Register(agent.NewAgentBase(agent.WithName("Bot1")), "/bot1")
	s.Register(agent.NewAgentBase(agent.WithName("Bot2")), "/bot2")

	mux := s.buildMux()
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode index response: %v", err)
	}

	agentsRaw, ok := body["agents"].([]any)
	if !ok {
		t.Fatal("expected agents array in response")
	}
	if len(agentsRaw) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agentsRaw))
	}

	first, ok := agentsRaw[0].(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", agentsRaw[0])
	}
	if first["route"] != "/bot1" {
		t.Errorf("expected first agent route=/bot1, got %v", first["route"])
	}
	if first["name"] != "Bot1" {
		t.Errorf("expected first agent name=Bot1, got %v", first["name"])
	}
}

func TestHTTP_RootIndex_Empty(t *testing.T) {
	s := NewAgentServer()
	mux := s.buildMux()

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	agentsRaw, ok := body["agents"].([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", body["agents"])
	}
	if len(agentsRaw) != 0 {
		t.Errorf("expected 0 agents, got %d", len(agentsRaw))
	}
}

// ---------------------------------------------------------------------------
// SIP endpoint tests
// ---------------------------------------------------------------------------

func TestHTTP_SipRouting(t *testing.T) {
	s := NewAgentServer()
	s.Register(agent.NewAgentBase(agent.WithName("Sales")), "/sales")
	s.SetupSIPRouting("/sip", false)
	s.RegisterSIPUsername("alice", "/sales")

	mux := s.buildMux()

	payload := `{"sip_username":"alice"}`
	req := httptest.NewRequest("POST", "/sip", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["route"] != "/sales" {
		t.Errorf("expected route=/sales, got %q", body["route"])
	}
}

func TestHTTP_SipRouting_UnknownUsername(t *testing.T) {
	s := NewAgentServer()
	s.SetupSIPRouting("/sip", false)

	mux := s.buildMux()

	payload := `{"sip_username":"unknown"}`
	req := httptest.NewRequest("POST", "/sip", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown SIP username, got %d", rr.Code)
	}
}

func TestHTTP_SipRouting_MethodNotAllowed(t *testing.T) {
	s := NewAgentServer()
	s.SetupSIPRouting("/sip", false)

	mux := s.buildMux()

	req := httptest.NewRequest("GET", "/sip", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET on SIP route, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// RunOption tests
// ---------------------------------------------------------------------------

func TestRunOptions(t *testing.T) {
	s := NewAgentServer()

	// Apply run options to verify they modify the server
	WithRunHost("192.168.1.1")(s)
	WithRunPort(9090)(s)

	if s.host != "192.168.1.1" {
		t.Errorf("expected host=192.168.1.1, got %q", s.host)
	}
	if s.port != 9090 {
		t.Errorf("expected port=9090, got %d", s.port)
	}
}

// ---------------------------------------------------------------------------
// extractSIPUsername tests
// ---------------------------------------------------------------------------

func TestExtractSipUsername_TopLevel(t *testing.T) {
	body := map[string]any{"sip_username": "bob"}
	if got := extractSIPUsername(body); got != "bob" {
		t.Errorf("expected bob, got %q", got)
	}
}

func TestExtractSipUsername_To(t *testing.T) {
	body := map[string]any{"to": "charlie"}
	if got := extractSIPUsername(body); got != "charlie" {
		t.Errorf("expected charlie, got %q", got)
	}
}

func TestExtractSipUsername_Nested(t *testing.T) {
	body := map[string]any{"call": map[string]any{"to": "dave"}}
	if got := extractSIPUsername(body); got != "dave" {
		t.Errorf("expected dave, got %q", got)
	}
}

func TestExtractSipUsername_Empty(t *testing.T) {
	body := map[string]any{"other": "value"}
	if got := extractSIPUsername(body); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// RegisterGlobalSIPRoutingCallback (Python register_global_routing_callback
// with redirect-string semantics)
// ---------------------------------------------------------------------------

func TestRegisterGlobalSipRoutingCallback_FansOutToAllAgents(t *testing.T) {
	s := NewAgentServer()

	a1 := agent.NewAgentBase(agent.WithName("a1"), agent.WithBasicAuth("u", "p"))
	a2 := agent.NewAgentBase(agent.WithName("a2"), agent.WithBasicAuth("u", "p"))
	s.Register(a1, "/a1")
	s.Register(a2, "/a2")

	const target = "https://elsewhere.example/global"
	s.RegisterGlobalSIPRoutingCallback("/sip", func(r *http.Request, body map[string]any) string {
		return target
	})

	// Hit each agent's /sip endpoint and confirm both produce the redirect.
	for _, route := range []string{"/a1/sip", "/a2/sip"} {
		req := httptest.NewRequest(http.MethodPost, route, strings.NewReader("{}"))
		req.SetBasicAuth("u", "p")
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		s.buildMux().ServeHTTP(rec, req)

		if rec.Code != http.StatusTemporaryRedirect {
			t.Errorf("%s: status = %d, want 307; body=%s", route, rec.Code, rec.Body.String())
			continue
		}
		if loc := rec.Header().Get("Location"); loc != target {
			t.Errorf("%s: Location = %q, want %q", route, loc, target)
		}
	}
}

func TestRegisterGlobalSipRoutingCallback_NormalizesPath(t *testing.T) {
	s := NewAgentServer()
	a := agent.NewAgentBase(agent.WithName("a"), agent.WithBasicAuth("u", "p"))
	s.Register(a, "/a")

	// Trailing slash should be stripped; leading slash should be added.
	s.RegisterGlobalSIPRoutingCallback("sip/", func(r *http.Request, body map[string]any) string {
		return "https://x.example"
	})

	req := httptest.NewRequest(http.MethodPost, "/a/sip", strings.NewReader("{}"))
	req.SetBasicAuth("u", "p")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.buildMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("normalized path did not register correctly; status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// Ensure the routing-callback (RegisterGlobalRoutingCallback, (body,headers)->
// route|nil) and the SIP-redirect (RegisterGlobalSIPRoutingCallback) variants
// coexist when registered on different paths — both issue a 307 redirect.
func TestRegisterGlobalRoutingCallbacks_Coexist(t *testing.T) {
	s := NewAgentServer()
	a := agent.NewAgentBase(agent.WithName("a"), agent.WithBasicAuth("u", "p"))
	s.Register(a, "/a")

	s.RegisterGlobalRoutingCallback("/override", func(body map[string]any, headers map[string]any) *string {
		route := "/routed"
		return &route
	})
	s.RegisterGlobalSIPRoutingCallback("/sip", func(r *http.Request, body map[string]any) string {
		return "https://elsewhere.example"
	})

	// Routing-callback path returns a 307 redirect to the route.
	req := httptest.NewRequest(http.MethodPost, "/a/override", strings.NewReader(`{"x":1}`))
	req.SetBasicAuth("u", "p")
	rec := httptest.NewRecorder()
	s.buildMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusTemporaryRedirect {
		t.Errorf("routing path: status=%d, want 307", rec.Code)
	} else if loc := rec.Header().Get("Location"); loc != "/routed" {
		t.Errorf("routing path: Location=%q, want /routed", loc)
	}

	// SIP path returns 307.
	req = httptest.NewRequest(http.MethodPost, "/a/sip", strings.NewReader("{}"))
	req.SetBasicAuth("u", "p")
	rec = httptest.NewRecorder()
	s.buildMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusTemporaryRedirect {
		t.Errorf("sip path: status=%d, want 307", rec.Code)
	}
}
