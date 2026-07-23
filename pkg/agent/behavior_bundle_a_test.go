// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package agent

import (
	"encoding/json"
	"github.com/signalwire/signalwire-go/v3/pkg/swaig"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Regression tests for the "Category A" agent behavior bundle:
//   #177  AddPreAnswerVerb warns on unknown/unsafe verbs (warn-only)
//   #178  EnableDebugEvents wires the debug webhook so OnDebugEvent fires
//   #179  EnableDebugRoutes / AsRouter registers /debug and /debug_events
//   #181  DefineContextsFromMap populates the ContextBuilder from a map
//   #183  OnSummary receives the extracted summary as arg1, raw body as arg2
//   #186  RegisterRoutingCallback callback receives the live *http.Request
//   #187  RegisterRoutingCallback normalizes its path (leading/trailing slash)

// captureStderr (defined in webhook_signing_test.go) swaps os.Stderr for a
// pipe and returns what fn writes; a logging.Logger created inside fn binds to
// the swapped os.Stderr at construction time, so its warnings are captured.

// ---------------------------------------------------------------------------
// #177 — AddPreAnswerVerb: unsafe verbs are invalid → panic (Python parity:
// add_pre_answer_verb raises ValueError); auto-answer verbs (play/connect) warn.
// ---------------------------------------------------------------------------

func TestAddPreAnswerVerb_PanicsOnUnsafeVerb(t *testing.T) {
	a := NewAgentBase(WithName("t"))
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected AddPreAnswerVerb to panic on an unsafe verb")
		}
		msg, _ := r.(string)
		if !strings.Contains(msg, "not safe for pre-answer use") {
			t.Errorf("panic message should explain the unsafe verb, got: %v", r)
		}
		// On panic the verb must NOT have been registered.
		if len(a.preAnswerVerbs) != 0 {
			t.Errorf("unsafe verb must not be registered, got %v", a.preAnswerVerbs)
		}
	}()
	a.AddPreAnswerVerb("definitely_not_a_verb", map[string]any{})
}

func TestAddPreAnswerVerb_NoWarnForSafeVerb(t *testing.T) {
	out := captureStderr(t, func() {
		a := NewAgentBase(WithName("t"))
		a.AddPreAnswerVerb("hangup", map[string]any{})
	})
	if strings.Contains(out, "not safe for pre-answer use") {
		t.Errorf("safe verb should not warn, got stderr:\n%s", out)
	}
}

func TestAddPreAnswerVerb_WarnsOnAutoAnswerWithoutFlag(t *testing.T) {
	out := captureStderr(t, func() {
		a := NewAgentBase(WithName("t"))
		// "play" auto-answers unless auto_answer:false is present.
		a.AddPreAnswerVerb("play", map[string]any{"url": "ring.mp3"})
	})
	if !strings.Contains(out, "pre_answer_verb_will_answer") {
		t.Errorf("expected auto-answer warning for play, got stderr:\n%s", out)
	}
}

func TestAddPreAnswerVerb_NoWarnForAutoAnswerWithFalseFlag(t *testing.T) {
	out := captureStderr(t, func() {
		a := NewAgentBase(WithName("t"))
		a.AddPreAnswerVerb("play", map[string]any{"url": "ring.mp3", "auto_answer": false})
	})
	if strings.Contains(out, "pre_answer_verb_will_answer") {
		t.Errorf("auto_answer:false should suppress the warning, got stderr:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// #178 / #179 — EnableDebugEvents wires the webhook; /debug_events route
// dispatches to OnDebugEvent.
// ---------------------------------------------------------------------------

func TestDebugEventsRouteDispatchesToHandler(t *testing.T) {
	a := NewAgentBase(WithName("t"), WithBasicAuth("u", "p"))
	a.EnableDebugEvents(1)

	var got map[string]any
	a.OnDebugEvent(func(event map[string]any) {
		got = event
	})

	payload := `{"label":"thinking","call_id":"abc","data":{"x":1}}`
	req := httptest.NewRequest(http.MethodPost, "/debug_events", strings.NewReader(payload))
	req.SetBasicAuth("u", "p")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	a.AsRouter().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("/debug_events status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if got == nil {
		t.Fatal("OnDebugEvent handler did not fire")
	}
	if got["label"] != "thinking" || got["call_id"] != "abc" {
		t.Errorf("debug event body not delivered intact: %v", got)
	}
}

func TestEnableDebugEvents_WiresWebhookIntoParams(t *testing.T) {
	a := NewAgentBase(WithName("t"), WithBasicAuth("u", "p"))
	a.EnableDebugEvents(3)
	doc := a.RenderSWML(nil, nil)
	sections, _ := doc["sections"].(map[string]any)
	main, _ := sections["main"].([]any)
	for _, v := range main {
		vm, _ := v.(map[string]any)
		aiCfg, ok := vm["ai"].(map[string]any)
		if !ok {
			continue
		}
		params, ok := aiCfg["params"].(map[string]any)
		if !ok {
			t.Fatal("expected params in AI config")
		}
		if params["debug_webhook_level"] != 3 {
			t.Errorf("debug_webhook_level = %v, want 3", params["debug_webhook_level"])
		}
		url, _ := params["debug_webhook_url"].(string)
		if !strings.HasSuffix(stripQuery(url), "/debug_events") {
			t.Errorf("debug_webhook_url = %q, want path ending /debug_events", url)
		}
		return
	}
	t.Fatal("AI verb not found")
}

func stripQuery(u string) string {
	if i := strings.IndexByte(u, '?'); i >= 0 {
		return u[:i]
	}
	return u
}

// EnableDebugRoutes is a chaining no-op (matches Python); the routes it
// documents are registered unconditionally by AsRouter.
func TestEnableDebugRoutes_RegistersDebugRoute(t *testing.T) {
	a := NewAgentBase(WithName("t"), WithBasicAuth("u", "p")).EnableDebugRoutes()
	a.SetPromptText("hi")

	req := httptest.NewRequest(http.MethodGet, "/debug", nil)
	req.SetBasicAuth("u", "p")
	rec := httptest.NewRecorder()
	a.AsRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("/debug status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var doc map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("/debug did not return a SWML document: %v", err)
	}
	if _, ok := doc["sections"]; !ok {
		t.Errorf("/debug response missing sections: %v", doc)
	}
}

// ---------------------------------------------------------------------------
// #181 — DefineContextsFromMap populates the ContextBuilder from a map.
// ---------------------------------------------------------------------------

func TestDefineContextsFromMap_PopulatesBuilder(t *testing.T) {
	a := NewAgentBase(WithName("t"))
	// Register the tool the step whitelists so its set_functions reference is
	// not dangling. STRICT-RENDER (Wave-2 P#5) rejects a step that whitelists a
	// function which is neither a registered SWAIG tool nor a reserved native
	// tool, matching the python reference's ContextBuilder.validate().
	a.DefineTool(ToolDefinition{
		Name:        "get_time",
		Description: "get the current time",
		Parameters:  map[string]any{},
		Handler:     func(map[string]any, map[string]any) *swaig.FunctionResult { return nil },
	})
	a.DefineContextsFromMap(map[string]any{
		"default": map[string]any{
			"steps": []any{
				map[string]any{
					"name":          "greet",
					"text":          "Greet the caller.",
					"step_criteria": "greeting done",
					"functions":     []any{"get_time"},
					"valid_steps":   []any{"finish"},
				},
				map[string]any{
					"name": "finish",
					"text": "Say goodbye.",
					"end":  true,
				},
			},
		},
	})

	m, err := a.DefineContexts().ToMap()
	if err != nil {
		t.Fatalf("ToMap: %v", err)
	}
	ctx, ok := m["default"].(map[string]any)
	if !ok {
		t.Fatalf("default context not built: %v", m)
	}
	steps, ok := ctx["steps"].([]map[string]any)
	if !ok || len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %#v", ctx["steps"])
	}
	step0Text, _ := steps[0]["text"].(string)
	if steps[0]["name"] != "greet" || !strings.Contains(step0Text, "Greet") {
		t.Errorf("step 0 not populated: %v", steps[0])
	}
	if steps[0]["step_criteria"] != "greeting done" {
		t.Errorf("step_criteria not set: %v", steps[0]["step_criteria"])
	}
	if steps[1]["end"] != true {
		t.Errorf("step 1 end flag not set: %v", steps[1])
	}
}

// ---------------------------------------------------------------------------
// #183 — OnSummary receives extracted summary as arg1, raw body as arg2.
// ---------------------------------------------------------------------------

func TestOnSummary_ExtractsSummaryFromPostPromptData(t *testing.T) {
	a := NewAgentBase(WithName("t"), WithBasicAuth("u", "p"))

	var gotSummary, gotRaw map[string]any
	a.OnSummary(func(summary, rawData map[string]any) {
		gotSummary = summary
		gotRaw = rawData
	})

	// post_prompt_data.parsed[0] is the canonical structured summary.
	payload := `{"post_prompt_data":{"parsed":[{"topic":"billing","sentiment":"happy"}],"raw":"ignored"},"call_id":"c1"}`
	req := httptest.NewRequest(http.MethodPost, "/post_prompt", strings.NewReader(payload))
	req.SetBasicAuth("u", "p")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	a.AsRouter().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("/post_prompt status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if gotSummary == nil {
		t.Fatal("summary was not extracted")
	}
	if gotSummary["topic"] != "billing" || gotSummary["sentiment"] != "happy" {
		t.Errorf("summary = %v, want extracted parsed[0]", gotSummary)
	}
	// arg2 must be the full raw body, not the summary.
	if gotRaw == nil || gotRaw["call_id"] != "c1" {
		t.Errorf("rawData = %v, want full body with call_id", gotRaw)
	}
	if _, ok := gotRaw["post_prompt_data"]; !ok {
		t.Errorf("rawData should contain post_prompt_data, got %v", gotRaw)
	}
}

func TestFindSummary_ExtractionOrder(t *testing.T) {
	// (1) top-level summary wins.
	if s := findSummary(map[string]any{
		"summary":          map[string]any{"k": "v"},
		"post_prompt_data": map[string]any{"parsed": []any{map[string]any{"k": "other"}}},
	}); s["k"] != "v" {
		t.Errorf("top-level summary should win, got %v", s)
	}
	// (2) parsed[0] when no top-level summary.
	if s := findSummary(map[string]any{
		"post_prompt_data": map[string]any{"parsed": []any{map[string]any{"k": "p0"}}},
	}); s["k"] != "p0" {
		t.Errorf("parsed[0] expected, got %v", s)
	}
	// (3) raw JSON parsed when no parsed array.
	if s := findSummary(map[string]any{
		"post_prompt_data": map[string]any{"raw": `{"k":"fromraw"}`},
	}); s["k"] != "fromraw" {
		t.Errorf("raw JSON should be parsed, got %v", s)
	}
	// nil body.
	if s := findSummary(nil); s != nil {
		t.Errorf("nil body should yield nil, got %v", s)
	}
}

// ---------------------------------------------------------------------------
// #186 — RegisterRoutingCallback receives the parsed body and request headers.
// ---------------------------------------------------------------------------

func TestRoutingCallbackReceivesBodyAndHeaders(t *testing.T) {
	a := NewAgentBase(WithName("t"), WithRoute("/svc"), WithBasicAuth("u", "p"))

	var gotBody map[string]any
	var gotHeader string
	a.RegisterRoutingCallback(func(body map[string]any, headers map[string]any) *string {
		gotBody = body
		if v, ok := headers["X-Test-Header"].(string); ok {
			gotHeader = v
		}
		route := "/routed"
		return &route
	}, "/agents")

	req := httptest.NewRequest(http.MethodPost, "/agents", strings.NewReader(`{"caller":"a"}`))
	req.SetBasicAuth("u", "p")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Header", "present")
	rec := httptest.NewRecorder()
	a.AsRouter().ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Location") != "/routed" {
		t.Errorf("Location = %q, want /routed", rec.Header().Get("Location"))
	}
	if gotBody["caller"] != "a" {
		t.Errorf("body not threaded through: %v", gotBody)
	}
	if gotHeader != "present" {
		t.Errorf("request header not threaded through: %q", gotHeader)
	}
}

// ---------------------------------------------------------------------------
// #187 — RegisterRoutingCallback normalizes the path.
// ---------------------------------------------------------------------------

func TestRoutingCallbackPathNormalization(t *testing.T) {
	a := NewAgentBase(WithName("t"), WithRoute("/svc"), WithBasicAuth("u", "p"))
	// Registered without a leading slash and with a trailing slash; must
	// normalize to "/agents" and still match.
	a.RegisterRoutingCallback(func(body map[string]any, headers map[string]any) *string {
		route := "/routed"
		return &route
	}, "agents/")

	if code, _ := dispatchRedirectAt(t, a, "/agents"); code != http.StatusTemporaryRedirect {
		t.Fatal(`callback registered as "agents/" should match /agents`)
	}
	if code, _ := dispatchRedirectAt(t, a, "/agents/"); code != http.StatusTemporaryRedirect {
		t.Fatal(`callback registered as "agents/" should also match /agents/`)
	}
}

func TestNormalizeCallbackPath(t *testing.T) {
	cases := map[string]string{
		"":         "/sip",
		"agents":   "/agents",
		"agents/":  "/agents",
		"/agents":  "/agents",
		"/agents/": "/agents",
		"/":        "/",
	}
	for in, want := range cases {
		if got := normalizeCallbackPath(in); got != want {
			t.Errorf("normalizeCallbackPath(%q) = %q, want %q", in, got, want)
		}
	}
}
