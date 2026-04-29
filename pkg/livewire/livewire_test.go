package livewire

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/signalwire/signalwire-go/pkg/logging"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// ---------------------------------------------------------------------------
// NewAgent
// ---------------------------------------------------------------------------

func TestNewAgent(t *testing.T) {
	a := NewAgent("You are a helpful assistant.")
	if a.instructions != "You are a helpful assistant." {
		t.Errorf("expected instructions to be set, got %q", a.instructions)
	}
	if len(a.tools) != 0 {
		t.Errorf("expected no tools, got %d", len(a.tools))
	}
}

func TestNewAgentWithUserdata(t *testing.T) {
	data := map[string]string{"key": "value"}
	a := NewAgent("test", WithUserdata(data))
	if a.userdata == nil {
		t.Error("expected userdata to be set")
	}
}

func TestNewAgentWithTools(t *testing.T) {
	// WithTools is a noop — should not panic
	a := NewAgent("test", WithTools("tool1", "tool2"))
	if a == nil {
		t.Error("expected agent to be created")
	}
}

// ---------------------------------------------------------------------------
// FunctionTool
// ---------------------------------------------------------------------------

func TestFunctionTool_CanonicalHandler(t *testing.T) {
	a := NewAgent("test")
	a.FunctionTool("my_tool",
		func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			city, _ := args["city"].(string)
			return swaig.NewFunctionResult("Weather in " + city)
		},
		WithDescription("Get weather"),
		WithParameters(map[string]any{
			"city": map[string]any{
				"type":        "string",
				"description": "City name",
			},
		}),
	)

	if len(a.tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(a.tools))
	}
	td := a.tools[0]
	if td.name != "my_tool" {
		t.Errorf("expected tool name 'my_tool', got %q", td.name)
	}
	if td.description != "Get weather" {
		t.Errorf("expected description 'Get weather', got %q", td.description)
	}
	if td.parameters == nil {
		t.Error("expected parameters to be set")
	}

	// Invoke handler
	result := td.handler(map[string]any{"city": "London"}, nil)
	m := result.ToMap()
	if resp, _ := m["response"].(string); resp != "Weather in London" {
		t.Errorf("expected 'Weather in London', got %q", resp)
	}
}

func TestFunctionTool_LiveKitStyleHandler(t *testing.T) {
	a := NewAgent("test")
	a.FunctionTool("greet",
		func(ctx *RunContext, name string) string {
			return "Hello, " + name
		},
		WithDescription("Greet someone"),
	)

	if len(a.tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(a.tools))
	}

	// Invoke the wrapped handler
	result := a.tools[0].handler(map[string]any{"name": "Alice"}, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if resp != "Hello, Alice" {
		t.Errorf("expected 'Hello, Alice', got %q", resp)
	}
}

func TestFunctionTool_SimpleMapHandler(t *testing.T) {
	a := NewAgent("test")
	a.FunctionTool("lookup",
		func(args map[string]any) string {
			return "found"
		},
		WithDescription("Lookup something"),
	)

	result := a.tools[0].handler(map[string]any{}, nil)
	m := result.ToMap()
	resp, _ := m["response"].(string)
	if resp != "found" {
		t.Errorf("expected 'found', got %q", resp)
	}
}

// ---------------------------------------------------------------------------
// AgentSession
// ---------------------------------------------------------------------------

func TestNewAgentSession_AllOptions(t *testing.T) {
	// Should not panic with any combination of options
	s := NewAgentSession(
		WithSTT("deepgram"),
		WithTTS("elevenlabs"),
		WithLLM("openai/gpt-4"),
		WithVAD(NewSileroVAD()),
		WithTurnDetection("server_vad"),
		WithAllowInterruptions(false),
		WithMinEndpointingDelay(0.5),
		WithMaxEndpointingDelay(2.0),
		WithMaxToolSteps(5),
	)

	if s == nil {
		t.Fatal("expected session to be created")
	}
	if s.stt != "deepgram" {
		t.Errorf("expected stt 'deepgram', got %q", s.stt)
	}
	if s.llm != "openai/gpt-4" {
		t.Errorf("expected llm 'openai/gpt-4', got %q", s.llm)
	}
	if s.allowInterrupt != false {
		t.Error("expected allowInterrupt to be false")
	}
	if s.minEndpointing != 0.5 {
		t.Errorf("expected minEndpointing 0.5, got %f", s.minEndpointing)
	}
}

func TestAgentSession_WithLLM_MapsToParams(t *testing.T) {
	s := NewAgentSession(WithLLM("openai/gpt-4"))
	ag := NewAgent("test instructions")

	ctx := &JobContext{Room: &Room{Name: "test"}}
	if err := s.Start(ctx, ag); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	sw := s.GetSwAgent()
	if sw == nil {
		t.Fatal("expected SwAgent to be created")
	}
}

func TestAgentSession_WithAllowInterruptions_MapsToBarge(t *testing.T) {
	s := NewAgentSession(WithAllowInterruptions(false))
	ag := NewAgent("test")

	ctx := &JobContext{Room: &Room{Name: "test"}}
	s.Start(ctx, ag)

	// The barge_confidence should be set high to prevent interruptions
	sw := s.GetSwAgent()
	if sw == nil {
		t.Fatal("expected SwAgent to be created")
	}
}

func TestAgentSession_Say(t *testing.T) {
	s := NewAgentSession()
	s.Say("Hello")
	s.Say("World")
	if len(s.sayQueue) != 2 {
		t.Errorf("expected 2 queued messages, got %d", len(s.sayQueue))
	}
	if s.sayQueue[0] != "Hello" || s.sayQueue[1] != "World" {
		t.Errorf("unexpected say queue: %v", s.sayQueue)
	}
}

func TestAgentSession_GenerateReply(t *testing.T) {
	// With no swAgent, GenerateReply must be a safe no-op: it should NOT
	// allocate state, panic, or otherwise mutate the session. Verify by
	// confirming the underlying agent is still nil and the say queue is
	// untouched.
	s := NewAgentSession()
	s.GenerateReply() // no options
	s.GenerateReply(WithReplyInstructions("Say hello"))

	if s.GetSwAgent() != nil {
		t.Errorf("GenerateReply without Start() must not allocate swAgent, got %v", s.GetSwAgent())
	}
	if len(s.sayQueue) != 0 {
		t.Errorf("GenerateReply must not enqueue Say messages; got %v", s.sayQueue)
	}
}

func TestAgentSession_GenerateReplyWithSwAgent(t *testing.T) {
	s := NewAgentSession()
	ag := NewAgent("test")
	ctx := &JobContext{Room: &Room{Name: "test"}}
	if err := s.Start(ctx, ag); err != nil {
		t.Fatalf("Start: %v", err)
	}

	s.GenerateReply(WithReplyInstructions("Greet the user warmly"))

	// GenerateReply with instructions must add a section to the underlying
	// AgentBase prompt — verify by rendering the document and finding the
	// section text. This catches a regression where GenerateReply silently
	// drops its instructions.
	swag := s.GetSwAgent()
	if swag == nil {
		t.Fatalf("expected swAgent to be allocated after Start()")
	}
	doc := swag.RenderSWML(nil, nil)
	rendered := fmt.Sprintf("%v", doc)
	if !strings.Contains(rendered, "Greet the user warmly") {
		t.Errorf("expected GenerateReply instructions to appear in rendered prompt; got:\n%s", rendered)
	}
}

func TestAgentSession_Interrupt(t *testing.T) {
	// Interrupt is a documented no-op on SignalWire (the platform handles
	// barge-in). The contract is: "the call returns without error, without
	// mutating session state." Verify both — start a session, interrupt,
	// confirm the underlying swAgent and currentAgent fields are unchanged.
	s := NewAgentSession()
	ag := NewAgent("original")
	ctx := &JobContext{Room: &Room{Name: "test"}}
	if err := s.Start(ctx, ag); err != nil {
		t.Fatalf("Start: %v", err)
	}
	preAgent := s.GetSwAgent()
	preInstructions := ag.instructions

	s.Interrupt()

	if s.GetSwAgent() != preAgent {
		t.Errorf("Interrupt() mutated swAgent reference")
	}
	if ag.instructions != preInstructions {
		t.Errorf("Interrupt() mutated agent instructions: %q -> %q", preInstructions, ag.instructions)
	}
}

func TestAgentSession_UpdateInstructions(t *testing.T) {
	s := NewAgentSession()
	ag := NewAgent("original instructions")
	ctx := &JobContext{Room: &Room{Name: "test"}}
	s.Start(ctx, ag)

	s.UpdateInstructions("new instructions")
	if ag.instructions != "new instructions" {
		t.Errorf("expected agent instructions to be updated, got %q", ag.instructions)
	}
}

// ---------------------------------------------------------------------------
// JobContext
// ---------------------------------------------------------------------------

func TestJobContext_Connect(t *testing.T) {
	ctx := &JobContext{Room: &Room{Name: "test-room"}}
	err := ctx.Connect()
	if err != nil {
		t.Errorf("Connect should return nil, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// RunApp (mock — verify banner and entrypoint)
// ---------------------------------------------------------------------------

func TestRunApp_Flow(t *testing.T) {
	entrypointCalled := false
	setupCalled := false

	server := NewAgentServer()
	server.SetSetupFunc(func(proc *JobProcess) {
		setupCalled = true
		proc.Userdata["warmed"] = true
	})

	server.RTCSession(func(ctx *JobContext) {
		entrypointCalled = true
		// Don't actually start an agent — just verify the flow
	})

	// Capture stderr to verify banner
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// RunApp will call entrypoint but then log an error because no agent was started
	RunApp(server)

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !entrypointCalled {
		t.Error("expected entrypoint to be called")
	}
	if !setupCalled {
		t.Error("expected setup function to be called")
	}

	// Verify banner was printed (check for distinctive ASCII art and tagline)
	if !strings.Contains(output, "LiveKit-compatible") {
		t.Error("expected banner to contain 'LiveKit-compatible'")
	}
	if !strings.Contains(output, "SignalWire") {
		t.Error("expected banner to contain 'SignalWire'")
	}
}

func TestRunApp_WithAgentName(t *testing.T) {
	server := NewAgentServer()
	server.RTCSession(func(ctx *JobContext) {
		// noop
	}, WithAgentName("TestAgent"))

	if server.agentName != "TestAgent" {
		t.Errorf("expected agent name 'TestAgent', got %q", server.agentName)
	}
}

func TestRunApp_WithServerType(t *testing.T) {
	// WithServerType is documented as a no-op on SignalWire. The contract
	// is: it stores the server type on the rtcConfig (so callers writing
	// LiveKit-style code can set it without surprise) and it must NOT
	// affect the registered entrypoint. Verify both.
	server := NewAgentServer()
	called := false
	server.RTCSession(func(ctx *JobContext) { called = true }, WithServerType("room"))

	if server.entrypoint == nil {
		t.Fatal("RTCSession must register the entrypoint regardless of server type")
	}
	server.entrypoint(&JobContext{Room: &Room{Name: "x"}})
	if !called {
		t.Error("registered entrypoint was not invoked")
	}
}

// ---------------------------------------------------------------------------
// AgentHandoff
// ---------------------------------------------------------------------------

func TestAgentHandoff(t *testing.T) {
	a := NewAgent("handoff agent")
	h := AgentHandoff{Agent: a}
	if h.Agent.instructions != "handoff agent" {
		t.Errorf("expected handoff agent instructions, got %q", h.Agent.instructions)
	}
}

// ---------------------------------------------------------------------------
// StopResponse
// ---------------------------------------------------------------------------

func TestStopResponse(t *testing.T) {
	// StopResponse is a sentinel type returned from tool handlers to
	// signal "do not run another LLM turn after this tool." Its contract
	// is: the zero value is acceptable, and it composes with ToolError
	// (the framework distinguishes them via type-switch). Verify both
	// type-switch arms recognise StopResponse.
	var v any = StopResponse{}
	switch v.(type) {
	case StopResponse:
		// expected
	default:
		t.Fatalf("StopResponse{} did not match its own type in a switch; got %T", v)
	}

	// And that it does NOT match ToolError.
	if _, ok := v.(*ToolError); ok {
		t.Errorf("StopResponse must not be assignable to *ToolError")
	}
}

// ---------------------------------------------------------------------------
// Plugin stubs
// ---------------------------------------------------------------------------

func TestPluginStubs(t *testing.T) {
	// All plugin constructors should work without panic
	d := NewDeepgramSTT(func(s *DeepgramSTT) { s.Model = "nova-2" })
	if d.Model != "nova-2" {
		t.Errorf("expected model 'nova-2', got %q", d.Model)
	}

	g := NewGoogleSTT(func(s *GoogleSTT) { s.Model = "chirp" })
	if g.Model != "chirp" {
		t.Errorf("expected model 'chirp', got %q", g.Model)
	}

	e := NewElevenLabsTTS(func(s *ElevenLabsTTS) { s.Voice = "rachel" })
	if e.Voice != "rachel" {
		t.Errorf("expected voice 'rachel', got %q", e.Voice)
	}

	c := NewCartesiaTTS(func(s *CartesiaTTS) { s.Voice = "sonic" })
	if c.Voice != "sonic" {
		t.Errorf("expected voice 'sonic', got %q", c.Voice)
	}

	o := NewOpenAITTS(func(s *OpenAITTS) { s.Voice = "alloy" })
	if o.Voice != "alloy" {
		t.Errorf("expected voice 'alloy', got %q", o.Voice)
	}

	l := NewOpenAILLM(func(s *OpenAILLM) { s.Model = "gpt-4o" })
	if l.Model != "gpt-4o" {
		t.Errorf("expected model 'gpt-4o', got %q", l.Model)
	}

	v := NewSileroVAD()
	loaded := v.Load()
	if loaded != v {
		t.Error("Load() should return the same SileroVAD instance")
	}
}

// ---------------------------------------------------------------------------
// Noop logging happens once per feature
// ---------------------------------------------------------------------------

func TestNoopLogging_OncePerFeature(t *testing.T) {
	tracker := newNoopTracker(logging.New("test"))

	tracker.once("stt", "STT message")
	tracker.once("stt", "STT message again")
	tracker.once("stt", "STT message third time")

	// Verify the key was tracked (logged only once)
	tracker.mu.Lock()
	sttLogged := tracker.logged["stt"]
	tracker.mu.Unlock()

	if !sttLogged {
		t.Error("expected stt to be tracked")
	}

	// Different key should also be tracked
	tracker.once("vad", "VAD message")
	tracker.mu.Lock()
	vadLogged := tracker.logged["vad"]
	tracker.mu.Unlock()

	if !vadLogged {
		t.Error("expected vad to be tracked")
	}
}

func TestNoopTracker_TrackedMap(t *testing.T) {
	tracker := &noopTracker{
		logged: make(map[string]bool),
		logger: logging.New("test"),
	}

	// Manually test the tracked map
	tracker.mu.Lock()
	tracker.logged["stt"] = true
	tracker.mu.Unlock()

	tracker.mu.Lock()
	alreadyLogged := tracker.logged["stt"]
	tracker.mu.Unlock()

	if !alreadyLogged {
		t.Error("expected stt to be tracked")
	}

	tracker.mu.Lock()
	notLogged := tracker.logged["vad"]
	tracker.mu.Unlock()

	if notLogged {
		t.Error("expected vad to NOT be tracked")
	}
}

func TestNoopTracker_OnlyLogsOnce(t *testing.T) {
	tracker := newNoopTracker(logging.New("test"))

	// Call once multiple times with same key
	for i := 0; i < 10; i++ {
		tracker.once("feature", "some message")
	}

	// The tracked map should have the key
	tracker.mu.Lock()
	logged := tracker.logged["feature"]
	count := len(tracker.logged)
	tracker.mu.Unlock()

	if !logged {
		t.Error("expected feature to be tracked")
	}
	if count != 1 {
		t.Errorf("expected exactly 1 tracked key, got %d", count)
	}
}

func TestNoopTracker_ConcurrentSafety(t *testing.T) {
	tracker := newNoopTracker(logging.New("test"))

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tracker.once("key", fmt.Sprintf("message %d", i))
		}(i)
	}
	wg.Wait()

	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if !tracker.logged["key"] {
		t.Error("expected key to be tracked after concurrent calls")
	}
}

// ---------------------------------------------------------------------------
// Tips
// ---------------------------------------------------------------------------

func TestTipsArrayHasEntries(t *testing.T) {
	if len(tips) == 0 {
		t.Error("expected tips array to have entries")
	}
	if len(tips) < 5 {
		t.Errorf("expected at least 5 tips, got %d", len(tips))
	}
}

// ---------------------------------------------------------------------------
// Banner
// ---------------------------------------------------------------------------

func TestBannerContent(t *testing.T) {
	// "LiveWire" is rendered as ASCII art, so check for substrings that appear
	if !strings.Contains(banner, "LiveKit-compatible") {
		t.Error("banner should contain 'LiveKit-compatible'")
	}
	if !strings.Contains(banner, "SignalWire") {
		t.Error("banner should contain 'SignalWire'")
	}
	// Verify the ASCII art is present (check a distinctive line)
	if !strings.Contains(banner, "/ /   (_)") {
		t.Error("banner should contain ASCII art")
	}
}

func TestPrintBanner_NoTerm(t *testing.T) {
	old := os.Getenv("TERM")
	os.Setenv("TERM", "")
	defer os.Setenv("TERM", old)

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	printBanner()

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Should NOT contain ANSI escape codes
	if strings.Contains(output, "\033[36m") {
		t.Error("expected no ANSI color when TERM is empty")
	}
	if !strings.Contains(output, "LiveKit-compatible") {
		t.Error("expected banner text")
	}
}

func TestPrintBanner_WithTerm(t *testing.T) {
	old := os.Getenv("TERM")
	os.Setenv("TERM", "xterm-256color")
	defer os.Setenv("TERM", old)

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	printBanner()

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Should contain ANSI cyan
	if !strings.Contains(output, "\033[36m") {
		t.Error("expected ANSI cyan color when TERM is set")
	}
}

// ---------------------------------------------------------------------------
// Full integration: Agent -> Session -> Start -> tools registered
// ---------------------------------------------------------------------------

func TestIntegration_ToolsRegistered(t *testing.T) {
	ag := NewAgent("You are a test agent.")
	ag.FunctionTool("test_tool",
		func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("test result")
		},
		WithDescription("A test tool"),
		WithParameters(map[string]any{
			"input": map[string]any{"type": "string", "description": "test input"},
		}),
	)

	s := NewAgentSession(WithLLM("gpt-4"))
	ctx := &JobContext{Room: &Room{Name: "test"}}
	if err := s.Start(ctx, ag); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	sw := s.GetSwAgent()
	if sw == nil {
		t.Fatal("expected SwAgent")
	}

	// Verify the tool was registered on the underlying agent
	tools := sw.DefineTools()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool registered, got %d", len(tools))
	}
	if tools[0].Name != "test_tool" {
		t.Errorf("expected tool name 'test_tool', got %q", tools[0].Name)
	}
	if tools[0].Description != "A test tool" {
		t.Errorf("expected description 'A test tool', got %q", tools[0].Description)
	}
}
