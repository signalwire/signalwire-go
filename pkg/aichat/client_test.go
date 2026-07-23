// Copyright (c) 2026 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package aichat

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// capturedRequest records what the fake server saw for a single JSON-RPC call.
type capturedRequest struct {
	method string
	params map[string]any
	auth   string
}

// newTestServer returns a client + a pointer to the last captured request. The
// handler responds with respond(method) so each test controls the result body.
func newTestServer(t *testing.T, respond func(method string, params map[string]any) map[string]any) (*Client, *capturedRequest) {
	t.Helper()
	captured := &capturedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req jsonRPCRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("server: bad request body: %v", err)
		}
		captured.method = req.Method
		captured.params = req.Params
		captured.auth = r.Header.Get("Authorization")
		resp := respond(req.Method, req.Params)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	client, err := NewClient(WithURL(srv.URL), WithProject("proj-1"), WithToken("tok-1"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client, captured
}

func ctx() context.Context { return context.Background() }

func TestCreateConversation(t *testing.T) {
	client, cptr := newTestServer(t, func(_ string, _ map[string]any) map[string]any {
		return map[string]any{"result": map[string]any{"status": "created", "initial_message": "hi there"}}
	})
	info, err := client.CreateConversation(ctx(), "conv-1", CreateOptions{ConfigURL: "http://cfg", Timeout: 30, Reinit: true})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if cptr.method != "create_conversation" {
		t.Errorf("wire method = %q, want create_conversation", cptr.method)
	}
	if cptr.params["id"] != "conv-1" || cptr.params["config_url"] != "http://cfg" {
		t.Errorf("params = %v, want id+config_url", cptr.params)
	}
	if cptr.params["conversation_timeout"] != float64(30) {
		t.Errorf("conversation_timeout = %v, want 30", cptr.params["conversation_timeout"])
	}
	if cptr.params["reinit"] != true {
		t.Errorf("reinit = %v, want true", cptr.params["reinit"])
	}
	if info.ID != "conv-1" || info.Status != "created" || info.InitialMessage != "hi there" {
		t.Errorf("info = %+v", info)
	}
}

func TestBasicAuthAndIdentityNeverInParams(t *testing.T) {
	client, cptr := newTestServer(t, func(_ string, _ map[string]any) map[string]any {
		return map[string]any{"result": map[string]any{"status": "created"}}
	})
	if _, err := client.CreateConversation(ctx(), "conv-1", CreateOptions{ConfigURL: "http://cfg"}); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("proj-1:tok-1"))
	if cptr.auth != wantAuth {
		t.Errorf("auth header = %q, want %q", cptr.auth, wantAuth)
	}
	for _, forbidden := range []string{"project_id", "project", "token", "api_token", "space", "space_id"} {
		if _, ok := cptr.params[forbidden]; ok {
			t.Errorf("identity key %q leaked into params", forbidden)
		}
	}
}

func TestChat(t *testing.T) {
	client, cptr := newTestServer(t, func(_ string, _ map[string]any) map[string]any {
		return map[string]any{"result": map[string]any{"response": "the reply", "user_event": map[string]any{"kind": "x"}}}
	})
	reply, err := client.Chat(ctx(), "conv-1", "hello")
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if cptr.method != "chat" {
		t.Errorf("wire method = %q, want chat", cptr.method)
	}
	if cptr.params["role"] != "user" {
		t.Errorf("default role = %v, want user", cptr.params["role"])
	}
	if reply.Text != "the reply" || reply.ConversationID != "conv-1" {
		t.Errorf("reply = %+v", reply)
	}
	if reply.UserEvent["kind"] != "x" {
		t.Errorf("user_event = %v", reply.UserEvent)
	}
}

func TestEndAndDelete(t *testing.T) {
	client, _ := newTestServer(t, func(method string, _ map[string]any) map[string]any {
		if method == "end_conversation" {
			return map[string]any{"result": map[string]any{"status": "ended", "id": "conv-1"}}
		}
		return map[string]any{"result": map[string]any{"status": "deleted", "id": "conv-1"}}
	})
	ended, err := client.End(ctx(), "conv-1")
	if err != nil || !ended {
		t.Errorf("End = %v, %v; want true, nil", ended, err)
	}
	deleted, err := client.Delete(ctx(), "conv-1")
	if err != nil || !deleted {
		t.Errorf("Delete = %v, %v; want true, nil", deleted, err)
	}
}

func TestLog(t *testing.T) {
	client, cptr := newTestServer(t, func(_ string, _ map[string]any) map[string]any {
		return map[string]any{"result": map[string]any{
			"chat_log":      []any{map[string]any{"role": "user"}, map[string]any{"role": "assistant"}},
			"call_timeline": []any{map[string]any{"event": "start"}},
		}}
	})
	log, err := client.Log(ctx(), "conv-1")
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if cptr.method != "chat_log" {
		t.Errorf("wire method = %q, want chat_log", cptr.method)
	}
	if len(log.Messages) != 2 || len(log.CallTimeline) != 1 {
		t.Errorf("log = %+v", log)
	}
}

func TestSummarizeSuccess(t *testing.T) {
	client, _ := newTestServer(t, func(_ string, _ map[string]any) map[string]any {
		return map[string]any{"result": map[string]any{"summary": "a short summary"}}
	})
	summary, err := client.Summarize(ctx(), "conv-1")
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if summary != "a short summary" {
		t.Errorf("summary = %q", summary)
	}
}

// TestSummarizeErrorBranchSurfaces is the load-bearing one_of test: the {error}
// branch rides the SUCCESS envelope and MUST surface as a typed *SummaryError,
// never as an empty string.
func TestSummarizeErrorBranchSurfaces(t *testing.T) {
	client, _ := newTestServer(t, func(_ string, _ map[string]any) map[string]any {
		return map[string]any{"result": map[string]any{"error": "Failed to generate summary"}}
	})
	summary, err := client.Summarize(ctx(), "__summarize_error")
	if err == nil {
		t.Fatalf("Summarize swallowed the {error} branch; got summary=%q, want *SummaryError", summary)
	}
	var se *SummaryError
	if !errors.As(err, &se) {
		t.Fatalf("error type = %T, want *SummaryError", err)
	}
	if summary != "" {
		t.Errorf("summary = %q, want empty on failure", summary)
	}
	if se.HasCode {
		t.Errorf("SummaryError.HasCode = true, want false (no JSON-RPC code)")
	}
	if !strings.Contains(se.Message, "Failed to generate summary") {
		t.Errorf("message = %q", se.Message)
	}
	// The base family must also match via errors.As.
	var base *AIChatError
	if !errors.As(err, &base) {
		t.Errorf("*SummaryError does not unwrap to *AIChatError")
	}
}

func TestErrorCodeMapping(t *testing.T) {
	cases := []struct {
		code    int
		wantAs  func(error) bool
		wantErr string
	}{
		{-32001, func(e error) bool { var t *ConversationNotFoundError; return errors.As(e, &t) }, "ConversationNotFound"},
		{-32005, func(e error) bool { var t *RateLimitError; return errors.As(e, &t) }, "RateLimit"},
		{-32006, func(e error) bool { var t *RateLimitError; return errors.As(e, &t) }, "RateLimit"},
		{-32007, func(e error) bool { var t *ChatInProgressError; return errors.As(e, &t) }, "ChatInProgress"},
		{-32009, func(e error) bool { var t *AuthenticationError; return errors.As(e, &t) }, "Authentication"},
	}
	for _, tc := range cases {
		code := tc.code
		client, _ := newTestServer(t, func(_ string, _ map[string]any) map[string]any {
			return map[string]any{"error": map[string]any{"code": code, "message": "boom"}}
		})
		_, err := client.Chat(ctx(), "x", "y")
		if err == nil {
			t.Errorf("code %d: got nil error", code)
			continue
		}
		if !tc.wantAs(err) {
			t.Errorf("code %d: error type %T did not match %s", code, err, tc.wantErr)
		}
		var base *AIChatError
		if !errors.As(err, &base) || base.Code != code {
			t.Errorf("code %d: base.Code = %d", code, base.Code)
		}
	}
}

// TestUnmappedCodeFallsToBase: an unmapped JSON-RPC code surfaces as the base
// *AIChatError, not a typed subclass.
func TestUnmappedCodeFallsToBase(t *testing.T) {
	client, _ := newTestServer(t, func(_ string, _ map[string]any) map[string]any {
		return map[string]any{"error": map[string]any{"code": -32602, "message": "invalid params"}}
	})
	_, err := client.Chat(ctx(), "x", "y")
	if err == nil {
		t.Fatal("got nil error")
	}
	// It must NOT be any typed subclass.
	for _, is := range []func(error) bool{
		func(e error) bool { var t *ConversationNotFoundError; return errors.As(e, &t) },
		func(e error) bool { var t *RateLimitError; return errors.As(e, &t) },
		func(e error) bool { var t *ChatInProgressError; return errors.As(e, &t) },
		func(e error) bool { var t *AuthenticationError; return errors.As(e, &t) },
	} {
		if is(err) {
			t.Errorf("unmapped code -32602 matched a typed subclass: %T", err)
		}
	}
	// Having excluded every typed subclass above, the family error must be the
	// base *AIChatError carrying the unmapped code.
	var base *AIChatError
	if !errors.As(err, &base) {
		t.Fatalf("error type = %T, want the base *AIChatError", err)
	}
	if base.Code != -32602 {
		t.Errorf("base.Code = %d, want -32602", base.Code)
	}
}

// TestSlowKeepaliveWhitespace: leading keepalive whitespace is valid JSON and must
// not break the decode (byte-driven liveness, mirroring the reference).
func TestSlowKeepaliveWhitespace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, "   \n  \t ")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		_, _ = io.WriteString(w, `{"result":{"response":"ok"}}`)
	}))
	defer srv.Close()
	client, err := NewClient(WithURL(srv.URL), WithProject("p"), WithToken("t"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	reply, err := client.Chat(ctx(), "conv-1", "hi")
	if err != nil {
		t.Fatalf("Chat with keepalive whitespace: %v", err)
	}
	if reply.Text != "ok" {
		t.Errorf("reply.Text = %q, want ok", reply.Text)
	}
}

func TestNewClientRequiresProject(t *testing.T) {
	t.Setenv("SIGNALWIRE_PROJECT_ID", "")
	_, err := NewClient(WithURL("http://x"))
	if err == nil {
		t.Fatal("expected error when project is missing")
	}
}

func TestResolveURLFromSpace(t *testing.T) {
	t.Setenv("SIGNALWIRE_SPACE", "")
	client, err := NewClient(WithProject("p"), WithSpace("myspace"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	want := "https://myspace.signalwire.com/api/ai/chat"
	if client.URL != want {
		t.Errorf("URL = %q, want %q", client.URL, want)
	}
}

func TestNewClientRequiresURLOrSpace(t *testing.T) {
	t.Setenv("SIGNALWIRE_SPACE", "")
	_, err := NewClient(WithProject("p"))
	if err == nil {
		t.Fatal("expected error when neither url nor space resolves")
	}
}

// TestSummarizeSamplingParams: pointer-valued sampling params are sent only when
// set, under their wire names.
func TestSummarizeSamplingParams(t *testing.T) {
	client, cptr := newTestServer(t, func(_ string, _ map[string]any) map[string]any {
		return map[string]any{"result": map[string]any{"summary": "s"}}
	})
	temp := 0.5
	maxTok := 128
	if _, err := client.Summarize(ctx(), "conv-1", SummarizeOptions{
		SummaryPrompt: "brief",
		Temperature:   &temp,
		MaxTokens:     &maxTok,
	}); err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if cptr.params["summary_prompt"] != "brief" {
		t.Errorf("summary_prompt = %v", cptr.params["summary_prompt"])
	}
	if cptr.params["temperature"] != 0.5 {
		t.Errorf("temperature = %v", cptr.params["temperature"])
	}
	if cptr.params["max_tokens"] != float64(128) {
		t.Errorf("max_tokens = %v", cptr.params["max_tokens"])
	}
	if _, ok := cptr.params["top_p"]; ok {
		t.Errorf("top_p should be absent when unset")
	}
}
