// Copyright (c) 2026 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package aichat

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeService is an in-process AI Chat service: it records each JSON-RPC call and
// answers per method.
type fakeService struct {
	mu    sync.Mutex
	calls []jsonRPCRequest
	srv   *httptest.Server
	// chatBody, when set, is written verbatim as the chat response (keepalive
	// padding included).
	chatBody string
	log      []map[string]any
}

func newFakeService(t *testing.T) (*fakeService, *Client) {
	t.Helper()
	fs := &fakeService{}
	fs.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)
		fs.mu.Lock()
		fs.calls = append(fs.calls, req)
		chatBody, log := fs.chatBody, fs.log
		fs.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "chat":
			if chatBody != "" {
				_, _ = io.WriteString(w, chatBody)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"response": "hello", "conversation_id": req.Params["id"]}})
		case "create_conversation":
			_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"status": "created", "initial_message": "Hi, how can I help?"}})
		case "chat_log":
			_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"chat_log": log}})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{}})
		}
	}))
	t.Cleanup(fs.srv.Close)
	c, err := NewClient(WithURL(fs.srv.URL), WithProject("p"), WithToken("t"))
	if err != nil {
		t.Fatal(err)
	}
	return fs, c
}

func (fs *fakeService) last() jsonRPCRequest {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.calls[len(fs.calls)-1]
}

func newGateway(t *testing.T, opts ChatGatewayOptions) *ChatGateway {
	t.Helper()
	if opts.ConfigURL == "" {
		opts.ConfigURL = "https://agent.example.com/swml"
	}
	if opts.Key == "" {
		opts.Key = "pk_test"
	}
	if opts.Secret == nil {
		opts.Secret = "s3cret"
	}
	if opts.Client == nil {
		_, opts.Client = newFakeService(t)
	}
	g, err := NewChatGateway(opts)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func sp(s string) *string { return &s }

func rejection(t *testing.T, err error) *GatewayRejection {
	t.Helper()
	var rej *GatewayRejection
	if !errors.As(err, &rej) {
		t.Fatalf("want *GatewayRejection, got %v", err)
	}
	return rej
}

func TestNewChatGatewayRequiresConfigURL(t *testing.T) {
	_, c := newFakeService(t)
	if _, err := NewChatGateway(ChatGatewayOptions{Client: c}); err == nil {
		t.Fatal("empty ConfigURL accepted")
	}
}

func TestHandleRoundTripsAndForgeriesAreRefused(t *testing.T) {
	g := newGateway(t, ChatGatewayOptions{})
	h := g.MintHandle("conv-1")
	id, err := g.ReadHandle(h)
	if err != nil || id != "conv-1" {
		t.Fatalf("ReadHandle = %q, %v", id, err)
	}
	other := newGateway(t, ChatGatewayOptions{Secret: []byte("different")})
	if rej := rejection(t, func() error { _, e := other.ReadHandle(h); return e }()); rej.Status != 403 || rej.Reason != "invalid handle" {
		t.Errorf("foreign handle: %+v", rej)
	}
	for _, junk := range []string{"", "nodot", "!!!.???"} {
		rej := rejection(t, func() error { _, e := g.ReadHandle(junk); return e }())
		if rej.Status != 400 || rej.Reason != "malformed handle" {
			t.Errorf("ReadHandle(%q): %+v", junk, rej)
		}
	}
	g.now = func() time.Time { return time.Now().Add(time.Duration(DefaultHandleTTL+10) * time.Second) }
	if rej := rejection(t, func() error { _, e := g.ReadHandle(h); return e }()); rej.Status != 403 || rej.Reason != "expired handle" {
		t.Errorf("expired: %+v", rej)
	}
}

func TestOriginPolicy(t *testing.T) {
	g := newGateway(t, ChatGatewayOptions{AllowedOrigins: []string{"https://shop.example.com/"}})
	for _, o := range []string{"http://localhost:3000", "http://127.0.0.1", "http://app.localhost", "https://shop.example.com"} {
		if err := g.CheckOrigin(sp(o)); err != nil {
			t.Errorf("%s refused: %v", o, err)
		}
	}
	if err := g.CheckOrigin(nil); err != nil {
		t.Errorf("missing origin refused: %v", err)
	}
	if rej := rejection(t, g.CheckOrigin(sp("https://evil.example.com"))); rej.Status != 403 {
		t.Errorf("unlisted origin: %+v", rej)
	}
}

func TestPrepareEnforcesKeyMethodAndOwnership(t *testing.T) {
	g := newGateway(t, ChatGatewayOptions{})
	if rej := rejection(t, prepErr(g, map[string]any{"message": "hi"}, nil)); rej.Status != 401 {
		t.Errorf("no key: %+v", rej)
	}
	if rej := rejection(t, prepErr(g, map[string]any{"message": "hi"}, sp("wrong"))); rej.Status != 401 {
		t.Errorf("wrong key: %+v", rej)
	}
	if rej := rejection(t, prepErr(g, map[string]any{"method": "summarize"}, sp("pk_test"))); rej.Status != 400 {
		t.Errorf("method: %+v", rej)
	}
	method, params, minted, err := g.Prepare(map[string]any{
		"message": "hi", "config_url": "https://evil", "id": "theirs",
	}, PrepareOptions{Key: sp("pk_test")})
	if err != nil || method != "chat" || minted == nil {
		t.Fatalf("Prepare = %q %v %v", method, minted, err)
	}
	if params["config_url"] != g.ConfigURL {
		t.Errorf("config_url = %v, want the gateway's", params["config_url"])
	}
	id, _ := g.ReadHandle(*minted)
	if params["id"] != id || id == "theirs" {
		t.Errorf("conversation id = %v (minted %s)", params["id"], id)
	}
	// The second turn reuses the handle: nothing minted.
	_, params2, minted2, err := g.Prepare(map[string]any{"message": "again", "handle": *minted}, PrepareOptions{Key: sp("pk_test")})
	if err != nil || minted2 != nil || params2["id"] != id {
		t.Errorf("second turn: %v %v %v", params2, minted2, err)
	}
}

func prepErr(g *ChatGateway, body map[string]any, key *string) error {
	_, _, _, err := g.Prepare(body, PrepareOptions{Key: key})
	return err
}

func TestPrepareMethods(t *testing.T) {
	g := newGateway(t, ChatGatewayOptions{ConversationTimeout: 900})
	key := sp("pk_test")
	for _, m := range []string{"end", "log"} {
		if rej := rejection(t, prepErr(g, map[string]any{"method": m}, key)); rej.Status != 400 || rej.Reason != m+" requires a handle" {
			t.Errorf("%s without handle: %+v", m, rej)
		}
	}
	method, params, minted, err := g.Prepare(map[string]any{"method": "start", "user_meta_data": map[string]any{"url": "https://shop"}}, PrepareOptions{Key: key})
	if err != nil || method != "create_conversation" || minted == nil {
		t.Fatalf("start: %q %v %v", method, minted, err)
	}
	if params["conversation_timeout"] != 900 || params["config_url"] != g.ConfigURL {
		t.Errorf("start params = %v", params)
	}
	if um, _ := params["user_meta_data"].(map[string]any); um["url"] != "https://shop" {
		t.Errorf("user_meta_data = %v", params["user_meta_data"])
	}
	method, params, _, _ = g.Prepare(map[string]any{"method": "log", "handle": *minted, "id": "other"}, PrepareOptions{Key: key})
	id, _ := g.ReadHandle(*minted)
	if method != "chat_log" || params["id"] != id || len(params) != 1 {
		t.Errorf("log: %q %v", method, params)
	}
	method, params, _, _ = g.Prepare(map[string]any{"method": "end", "handle": *minted}, PrepareOptions{Key: key})
	if method != "end_conversation" || params["id"] != id {
		t.Errorf("end: %q %v", method, params)
	}
	if rej := rejection(t, prepErr(g, map[string]any{"message": "  "}, key)); rej.Status != 400 || rej.Reason != "message is required" {
		t.Errorf("empty message: %+v", rej)
	}
}

func TestCaps(t *testing.T) {
	g := newGateway(t, ChatGatewayOptions{MaxNewConversations: 2, MaxTurns: 2})
	key := sp("pk_test")
	handles := make([]string, 0, 2)
	for range 2 {
		_, _, minted, err := g.Prepare(map[string]any{"message": "hi"}, PrepareOptions{Key: key})
		if err != nil {
			t.Fatal(err)
		}
		handles = append(handles, *minted)
	}
	if rej := rejection(t, prepErr(g, map[string]any{"message": "hi"}, key)); rej.Status != 429 {
		t.Errorf("mint cap: %+v", rej)
	}
	// Conversation 0 has used 1 turn: one more, then capped; conversation 1 unaffected.
	if err := prepErr(g, map[string]any{"message": "x", "handle": handles[0]}, key); err != nil {
		t.Fatal(err)
	}
	if rej := rejection(t, prepErr(g, map[string]any{"message": "x", "handle": handles[0]}, key)); rej.Status != 429 {
		t.Errorf("turn cap: %+v", rej)
	}
	if err := prepErr(g, map[string]any{"message": "x", "handle": handles[1]}, key); err != nil {
		t.Errorf("other conversation capped: %v", err)
	}
}

func TestSizeLimits(t *testing.T) {
	g := newGateway(t, ChatGatewayOptions{MaxNewConversations: 1})
	key := sp("pk_test")
	// Bytes, not characters: 4-byte runes push it over well before 8192 chars.
	big := strings.Repeat("\U0001F600", MaxMessageBytes/4+1)
	if rej := rejection(t, prepErr(g, map[string]any{"message": big}, key)); rej.Status != 413 {
		t.Errorf("oversized message: %+v", rej)
	}
	if rej := rejection(t, prepErr(g, map[string]any{"message": "hi", "user_meta_data": "nope"}, key)); rej.Status != 400 {
		t.Errorf("non-object metadata: %+v", rej)
	}
	if rej := rejection(t, prepErr(g, map[string]any{"message": "hi", "user_meta_data": map[string]any{"x": strings.Repeat("a", MaxUserMetadataBytes)}}, key)); rej.Status != 413 {
		t.Errorf("oversized metadata: %+v", rej)
	}
	// None of the refusals above spent the single mint.
	if err := prepErr(g, map[string]any{"message": strings.Repeat("a", MaxMessageBytes)}, key); err != nil {
		t.Errorf("message at the limit refused (or a refusal charged the mint): %v", err)
	}
}

func TestTranscriptFilteringAndTimes(t *testing.T) {
	g := newGateway(t, ChatGatewayOptions{})
	msgs := []map[string]any{
		{"role": "system", "content": "SECRET PROMPT", "timestamp": float64(9e15)},
		{"role": "user", "content": "hi", "timestamp": float64(1_700_000_000_000_000)},
		{"role": "assistant", "content": "", "tool_calls": []any{"x"}},
		{"role": "tool", "content": "tool result"},
		{"role": "assistant", "content": "hello", "timestamp": float64(1_700_000_001_000_000), "id": "m2"},
	}
	vis := g.VisibleMessages(msgs)
	if len(vis) != 2 || vis[0]["content"] != "hi" || vis[1]["content"] != "hello" {
		t.Fatalf("visible = %v", vis)
	}
	if vis[1]["timestamp"] != float64(1_700_000_001) || vis[1]["id"] != nil {
		t.Errorf("entry = %v", vis[1])
	}
	if la := g.LastActivity(msgs); la == nil || *la != 9e9 {
		t.Errorf("last activity = %v (every role counts)", la)
	}
	if g.LastActivity([]map[string]any{{"role": "user"}}) != nil {
		t.Error("undated transcript reported a time")
	}
	if g.EffectiveTimeout() != ServiceDefaultConversationTimeout {
		t.Errorf("effective timeout = %d", g.EffectiveTimeout())
	}
}

func post(t *testing.T, h http.Handler, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRouterFullExchange(t *testing.T) {
	fs, c := newFakeService(t)
	fs.chatBody = "   \n  {\"result\":{\"response\":\"padded\"}}"
	g := newGateway(t, ChatGatewayOptions{Client: c, AllowedOrigins: []string{"https://shop.example.com"}})
	router := g.Router()
	auth := map[string]string{"Authorization": "Bearer pk_test", "Origin": "https://shop.example.com"}

	rec := post(t, router, `{"method":"start","user_meta_data":{"title":"Shop"}}`, auth)
	if rec.Code != 200 {
		t.Fatalf("start: %d %s", rec.Code, rec.Body)
	}
	handle := rec.Header().Get("X-Chat-Handle")
	var start map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &start)
	if handle == "" || start["greeting"] != "Hi, how can I help?" || start["timeout"] != float64(ServiceDefaultConversationTimeout) {
		t.Fatalf("start response = %v handle=%q", start, handle)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://shop.example.com" {
		t.Errorf("CORS header missing: %v", rec.Header())
	}
	if um, _ := fs.last().Params["user_meta_data"].(map[string]any); um["title"] != "Shop" {
		t.Errorf("upstream create params = %v", fs.last().Params)
	}

	// A chat relays the service body verbatim — keepalive padding included.
	rec = post(t, router, `{"message":"hello","handle":"`+handle+`"}`, auth)
	if rec.Code != 200 || rec.Body.String() != fs.chatBody {
		t.Errorf("chat relay = %d %q", rec.Code, rec.Body.String())
	}
	if fs.last().Method != "chat" || fs.last().Params["config_url"] != g.ConfigURL {
		t.Errorf("upstream chat = %+v", fs.last())
	}

	fs.log = []map[string]any{{"role": "system", "content": "prompt"}, {"role": "user", "content": "hello", "timestamp": 2_000_000}}
	rec = post(t, router, `{"method":"log","handle":"`+handle+`"}`, auth)
	var log map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &log)
	if msgs, _ := log["messages"].([]any); len(msgs) != 1 || log["last_activity"] != float64(2) {
		t.Errorf("log = %v", log)
	}

	rec = post(t, router, `{"method":"end","handle":"`+handle+`"}`, auth)
	if rec.Code != 200 || fs.last().Method != "end_conversation" {
		t.Errorf("end: %d %v", rec.Code, fs.last())
	}
}

func TestRouterRejections(t *testing.T) {
	g := newGateway(t, ChatGatewayOptions{})
	router := g.Router()
	if rec := post(t, router, `{"message":"hi"}`, map[string]string{"Authorization": "Bearer nope"}); rec.Code != 401 {
		t.Errorf("bad key: %d", rec.Code)
	}
	rec := post(t, router, `{"message":"hi"}`, map[string]string{"Authorization": "Bearer pk_test", "Origin": "https://evil.example"})
	if rec.Code != 403 || rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("unlisted origin: %d %v", rec.Code, rec.Header())
	}
	if rec := post(t, router, `not json`, map[string]string{"Authorization": "Bearer pk_test"}); rec.Code != 400 {
		t.Errorf("bad json: %d", rec.Code)
	}
	if rec := post(t, router, `[1]`, map[string]string{"Authorization": "Bearer pk_test"}); rec.Code != 400 {
		t.Errorf("non-object: %d", rec.Code)
	}
	huge := `{"message":"` + strings.Repeat("a", MaxRequestBodyBytes) + `"}`
	if rec := post(t, router, huge, map[string]string{"Authorization": "Bearer pk_test"}); rec.Code != 413 {
		t.Errorf("oversized body: %d", rec.Code)
	}
	// Preflight: allow headers only for an allowed origin.
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	pre := httptest.NewRecorder()
	router.ServeHTTP(pre, req)
	if pre.Code != 204 || pre.Header().Get("Access-Control-Allow-Methods") != "POST, OPTIONS" {
		t.Errorf("preflight: %d %v", pre.Code, pre.Header())
	}
}

func TestRawPostLeavesBodyUnread(t *testing.T) {
	fs, c := newFakeService(t)
	fs.chatBody = "  {\"result\":{}}"
	resp, err := c.RawPost(ctx(), "chat", map[string]any{"id": "c1", "message": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if string(raw) != fs.chatBody {
		t.Errorf("body = %q", raw)
	}
	if got := fs.last(); got.Method != "chat" || got.JSONRPC != "2.0" || got.ID == "" {
		t.Errorf("request = %+v", got)
	}
}
