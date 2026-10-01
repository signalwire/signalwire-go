// Copyright (c) 2026 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package aichat

// A browser-facing gateway for the SignalWire AI Chat service.
//
// A chat widget running in a page cannot hold a SignalWire API token: the token
// carries the whole project, so putting it in JavaScript hands every visitor the
// ability to run up billed turns. So the widget talks to a gateway mounted in your
// own app, which holds the credential server-side and forwards on the widget's
// behalf:
//
//	browser ──(publishable key)──▶ your app ──(project:token)──▶ chat service
//
// The browser learns the gateway's URL and a publishable key — not the project,
// the space, the token, or which agent config runs: the gateway injects
// config_url itself, so a key only ever reaches the one script it was issued for.
//
// What a stolen key gets: nothing to read (chat_log is filtered, conversation
// handles are signed so ids cannot be guessed), only the ability to talk, which
// costs money — which is why the caps (MaxNewConversations, MaxTurns) are the
// primary control. The origin allowlist stops a key pasted into someone else's
// page; it does not stop curl. Treat it as leak containment, not access control.
//
// Exactly one browser field is forwarded rather than overwritten:
// user_meta_data, the page context a widget collects about itself. It reaches the
// agent's config request as params.user_meta_data, only when a conversation is
// created. It is a visitor's claim, never authority.
//
// Size limits: the request body (MaxRequestBodyBytes, refused before parsing), a
// chat message (MaxMessageBytes of UTF-8, refused before a conversation is minted
// or a turn charged) and user_meta_data (MaxUserMetadataBytes serialized) are each
// answered with 413 past their limit.

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Gateway defaults and limits.
const (
	// DefaultHandleTTL is how long a conversation handle stays valid: it outlives
	// a page refresh but not a session left open overnight.
	DefaultHandleTTL = 24 * 60 * 60
	// ServiceDefaultConversationTimeout mirrors the chat service's own default
	// idle timeout, so the gateway can report a real number when not given one.
	ServiceDefaultConversationTimeout = 3600
	// DefaultMaxNewConversations is new conversations per window, per gateway.
	DefaultMaxNewConversations = 60
	// DefaultMaxTurns is turns per conversation, ever.
	DefaultMaxTurns = 200
	// DefaultWindowSeconds is the window MaxNewConversations counts over.
	DefaultWindowSeconds = 60
	// MaxUserMetadataBytes bounds the browser-volunteered user_meta_data,
	// serialized.
	MaxUserMetadataBytes = 8 * 1024
	// MaxMessageBytes bounds one typed message (UTF-8): a chat turn, and the text
	// a HandoffRouter's /say injects into a live call.
	MaxMessageBytes = 8 * 1024
	// MaxRequestBodyBytes bounds a whole request body, checked before parsing.
	MaxRequestBodyBytes = 64 * 1024
)

// allowedGatewayMethods are the browser-facing methods a gateway accepts.
var allowedGatewayMethods = map[string]bool{"start": true, "chat": true, "log": true, "end": true}

// visibleRoles are the transcript roles a browser may see. chat_log returns the
// whole conversation as the service holds it — the substituted system prompt,
// tool calls, tool results — so it is filtered down to what the visitor already
// watched go past.
var visibleRoles = map[string]bool{"user": true, "assistant": true}

// GatewayRejection is a request the gateway refused, with the HTTP status the
// browser should see. Deliberately coarse: the browser learns that it was refused
// and at most which bucket it fell into, never the caps' values or the allowlist.
type GatewayRejection struct { //nolint:errname // the reference class name (signalwire.ai_chat.gateway.GatewayRejection)
	// Status is the HTTP status to send back (401 bad key, 403 origin/handle,
	// 400 disallowed method, 413 over a size limit, 429 a cap was hit).
	Status int
	// Reason is a short, fixed explanation that reaches the browser.
	Reason string
}

// NewGatewayRejection builds a rejection with the status the route returns.
func NewGatewayRejection(status int, reason string) *GatewayRejection {
	return &GatewayRejection{Status: status, Reason: reason}
}

// Error implements error.
func (r *GatewayRejection) Error() string { return fmt.Sprintf("%d: %s", r.Status, r.Reason) }

// ChatGatewayOptions configures NewChatGateway. A zero numeric field takes the
// documented default.
type ChatGatewayOptions struct {
	// ConfigURL is the agent this key may talk to. Injected on every call and
	// never accepted from the request. Required.
	ConfigURL string `sw:"required" kind:"keyword"`
	// Key is the publishable key the widget carries. Empty falls back to
	// SIGNALWIRE_CHAT_GATEWAY_KEY, then to a generated "pk_…" key.
	Key string `sw:"optional" kind:"keyword" gen:"optional<string>"`
	// AllowedOrigins are the origins permitted to use this key. Localhost is
	// always allowed; anything else must be listed.
	AllowedOrigins []string `sw:"optional" kind:"keyword" gen:"union<list<string>,tuple<string,any>>"`
	// Client is the AI Chat client to forward with. Nil builds one from the
	// environment (and the gateway then owns it).
	Client *Client `sw:"optional" kind:"keyword"`
	// Secret is the HMAC key for signing handles: a []byte or a string. Nil
	// falls back to SIGNALWIRE_CHAT_GATEWAY_SECRET, then to random bytes per
	// process — which invalidates outstanding handles on restart, so set it when
	// running more than one replica.
	Secret any `sw:"optional" kind:"keyword" gen:"optional<union<bytes,string>>"`
	// HandleTTL is seconds a handle stays valid (default DefaultHandleTTL).
	HandleTTL int `sw:"optional" kind:"keyword"`
	// ConversationTimeout is idle seconds before the service ends a
	// conversation, passed on every create. 0 leaves it to the service default.
	ConversationTimeout int `sw:"optional" kind:"keyword" gen:"optional<int>"`
	// MaxNewConversations is new conversations per WindowSeconds.
	MaxNewConversations int `sw:"optional" kind:"keyword"`
	// MaxTurns is turns a single conversation may run.
	MaxTurns int `sw:"optional" kind:"keyword"`
	// WindowSeconds is the window for MaxNewConversations.
	WindowSeconds int `sw:"optional" kind:"keyword"`
}

// ChatGateway is a server-side proxy that lets a browser chat without holding a
// token. Mount its Router() on the agent:
//
//	gw, err := aichat.NewChatGateway(aichat.ChatGatewayOptions{
//	    ConfigURL:      "https://my-agent.example.com/swml",
//	    Key:            "pk_live_...",
//	    AllowedOrigins: []string{"https://shop.example.com"},
//	})
//	a.Mount(gw.Router(), agent.MountOptions{Prefix: "/chat"})
//
// Counters live in this process; behind several replicas the effective caps
// multiply by the replica count.
type ChatGateway struct {
	// ConfigURL is the SWML config the gateway always sends upstream.
	ConfigURL string
	// Key is the publishable key the browser presents.
	Key string
	// AllowedOrigins is the set of origins (no trailing slash) allowed to call.
	AllowedOrigins map[string]bool
	// HandleTTL is seconds a signed handle stays valid.
	HandleTTL int
	// ConversationTimeout is the idle timeout passed upstream (0 = service default).
	ConversationTimeout int
	// MaxNewConversations caps conversations minted per window.
	MaxNewConversations int
	// MaxTurns caps turns per conversation.
	MaxTurns int
	// WindowSeconds is the length of the window MaxNewConversations counts over.
	WindowSeconds int

	client     *Client
	ownsClient bool
	secret     []byte
	mu         sync.Mutex
	mints      []time.Time
	turns      map[string]turnCount
	now        func() time.Time
}

type turnCount struct {
	count int
	last  time.Time
}

// NewChatGateway builds a gateway that fronts one agent for browser traffic.
func NewChatGateway(opts ChatGatewayOptions) (*ChatGateway, error) {
	if opts.ConfigURL == "" {
		return nil, errors.New("config_url is required — it is what a key is scoped to")
	}
	key := firstNonEmpty(opts.Key, os.Getenv("SIGNALWIRE_CHAT_GATEWAY_KEY"))
	if key == "" {
		key = "pk_" + randomToken(24)
	}
	origins := map[string]bool{}
	for _, o := range opts.AllowedOrigins {
		origins[strings.TrimRight(o, "/")] = true
	}
	g := &ChatGateway{
		ConfigURL:           opts.ConfigURL,
		Key:                 key,
		AllowedOrigins:      origins,
		HandleTTL:           orDefault(opts.HandleTTL, DefaultHandleTTL),
		ConversationTimeout: opts.ConversationTimeout,
		MaxNewConversations: orDefault(opts.MaxNewConversations, DefaultMaxNewConversations),
		MaxTurns:            orDefault(opts.MaxTurns, DefaultMaxTurns),
		WindowSeconds:       orDefault(opts.WindowSeconds, DefaultWindowSeconds),
		client:              opts.Client,
		turns:               map[string]turnCount{},
		now:                 time.Now,
	}
	if g.client == nil {
		c, err := NewClient()
		if err != nil {
			return nil, err
		}
		g.client = c
		g.ownsClient = true
	}
	switch s := opts.Secret.(type) {
	case nil:
		if env := os.Getenv("SIGNALWIRE_CHAT_GATEWAY_SECRET"); env != "" {
			g.secret = []byte(env)
		} else {
			g.secret = make([]byte, 32)
			if _, err := rand.Read(g.secret); err != nil {
				return nil, fmt.Errorf("aichat: generate gateway secret: %w", err)
			}
		}
	case []byte:
		g.secret = append([]byte(nil), s...)
	case string:
		g.secret = []byte(s)
	default:
		return nil, fmt.Errorf("aichat: Secret must be []byte or string, got %T", opts.Secret)
	}
	return g, nil
}

func orDefault(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}

func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("aichat: crypto/rand failed: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// LastActivity returns the epoch SECONDS of the newest message, or nil if nothing
// is dated. It bootstraps a browser's idle clock across a reload. The service
// stamps messages in microseconds; every role counts, because the service's idle
// clock runs off any write.
func (g *ChatGateway) LastActivity(messages []map[string]any) *float64 {
	var newest int64
	found := false
	for _, msg := range messages {
		ts, ok := intTimestamp(msg["timestamp"])
		if ok && ts > 0 && (!found || ts > newest) {
			newest, found = ts, true
		}
	}
	if !found {
		return nil
	}
	v := float64(newest) / 1_000_000
	return &v
}

// intTimestamp reads an integral JSON number (decoded as float64 or json.Number
// or a Go int), refusing fractional and boolean values.
func intTimestamp(v any) (int64, bool) {
	switch t := v.(type) {
	case int:
		return int64(t), true
	case int64:
		return t, true
	case float64:
		if t == float64(int64(t)) {
			return int64(t), true
		}
	}
	return 0, false
}

// EffectiveTimeout is the idle seconds a conversation actually gets: the
// configured timeout, or the service's documented default when unset — so the
// number handed to a browser is never empty.
func (g *ChatGateway) EffectiveTimeout() int {
	if g.ConversationTimeout != 0 {
		return g.ConversationTimeout
	}
	return ServiceDefaultConversationTimeout
}

// Close releases the upstream client if this gateway built it; a client passed
// in through ChatGatewayOptions.Client belongs to the caller and is left open.
func (g *ChatGateway) Close() error {
	if g.ownsClient {
		return g.client.Close()
	}
	return nil
}

// MintHandle issues a signed handle for a conversation (a new random id when
// none is given). The browser never names a conversation: signing means a
// caller can only present handles this gateway issued.
func (g *ChatGateway) MintHandle(conversationID ...string) string {
	id := ""
	if len(conversationID) > 0 {
		id = conversationID[0]
	}
	if id == "" {
		id = "chat-" + randomToken(18)
	}
	expires := g.now().Unix() + int64(g.HandleTTL)
	payload := []byte(id + ":" + strconv.FormatInt(expires, 10))
	mac := hmac.New(sha256.New, g.secret)
	mac.Write(payload)
	return b64(payload) + "." + b64(mac.Sum(nil))
}

// ReadHandle returns the conversation id inside a handle, or a *GatewayRejection:
// 400 "malformed handle", 403 "invalid handle" (bad signature) or 403 "expired
// handle". Signature first, expiry second, both before the id is trusted.
func (g *ChatGateway) ReadHandle(handle string) (string, error) {
	raw, sig, ok := strings.Cut(handle, ".")
	if !ok {
		return "", NewGatewayRejection(400, "malformed handle")
	}
	payload, err1 := unb64(raw)
	given, err2 := unb64(sig)
	if err1 != nil || err2 != nil {
		return "", NewGatewayRejection(400, "malformed handle")
	}
	mac := hmac.New(sha256.New, g.secret)
	mac.Write(payload)
	if !hmac.Equal(given, mac.Sum(nil)) {
		return "", NewGatewayRejection(403, "invalid handle")
	}
	text := string(payload)
	i := strings.LastIndex(text, ":")
	if i < 0 {
		return "", NewGatewayRejection(400, "malformed handle")
	}
	expires, err := strconv.ParseInt(text[i+1:], 10, 64)
	if err != nil {
		return "", NewGatewayRejection(400, "malformed handle")
	}
	if g.now().Unix() > expires {
		return "", NewGatewayRejection(403, "expired handle")
	}
	return text[:i], nil
}

// CheckOrigin allows localhost always and anything else only when listed. A nil
// origin (no Origin header) is allowed: browsers always send one for the
// cross-origin POSTs this serves, so absence means a non-browser caller.
func (g *ChatGateway) CheckOrigin(origin *string) error {
	if origin == nil {
		return nil
	}
	host := ""
	if u, err := url.Parse(*origin); err == nil {
		host = strings.ToLower(u.Hostname())
	}
	switch host {
	case "localhost", "127.0.0.1", "::1", "[::1]":
		return nil
	}
	if strings.HasSuffix(host, ".localhost") {
		return nil
	}
	if g.AllowedOrigins[strings.TrimRight(*origin, "/")] {
		return nil
	}
	return NewGatewayRejection(403, "origin not allowed")
}

// CheckKey verifies the publishable key the browser sent (nil when absent),
// compared in constant time. A missing or wrong key is a 401 rejection.
func (g *ChatGateway) CheckKey(presented *string) error {
	if presented == nil || *presented == "" || !hmac.Equal([]byte(*presented), []byte(g.Key)) {
		return NewGatewayRejection(401, "bad key")
	}
	return nil
}

// VisibleMessages is the transcript a browser may redraw and nothing else: user
// and assistant turns with text, reduced to role, content and timestamp (epoch
// seconds). The system prompt and tool traffic never leave the server.
func (g *ChatGateway) VisibleMessages(messages []map[string]any) []map[string]any {
	out := []map[string]any{}
	for _, msg := range messages {
		role, _ := msg["role"].(string)
		content, ok := msg["content"].(string)
		if !visibleRoles[role] || !ok || strings.TrimSpace(content) == "" {
			continue
		}
		entry := map[string]any{"role": role, "content": content}
		if ts, ok := intTimestamp(msg["timestamp"]); ok && ts > 0 {
			entry["timestamp"] = float64(ts) / 1_000_000
		}
		out = append(out, entry)
	}
	return out
}

func (g *ChatGateway) chargeMint() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := g.now()
	cutoff := now.Add(-time.Duration(g.WindowSeconds) * time.Second)
	kept := g.mints[:0]
	for _, t := range g.mints {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	g.mints = kept
	if len(g.mints) >= g.MaxNewConversations {
		return NewGatewayRejection(429, "too many new conversations")
	}
	g.mints = append(g.mints, now)
	return nil
}

func (g *ChatGateway) chargeTurn(conversationID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := g.now()
	// A handle cannot outlive its TTL, so anything older can never be charged again.
	cutoff := now.Add(-time.Duration(g.HandleTTL) * time.Second)
	for k, v := range g.turns {
		if !v.last.After(cutoff) {
			delete(g.turns, k)
		}
	}
	tc := g.turns[conversationID]
	if tc.count >= g.MaxTurns {
		return NewGatewayRejection(429, "conversation turn limit reached")
	}
	g.turns[conversationID] = turnCount{count: tc.count + 1, last: now}
	return nil
}

// ReadUserMetadata validates the page context a browser volunteered
// (body["user_meta_data"]). Absent, null and empty all return nil. Not an object
// is a 400 rejection; over MaxUserMetadataBytes serialized is a 413.
func (g *ChatGateway) ReadUserMetadata(body map[string]any) (map[string]any, error) {
	raw, present := body["user_meta_data"]
	if !present || raw == nil {
		return nil, nil //nolint:nilnil // nil map = no metadata, the reference's None
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, NewGatewayRejection(400, "user_meta_data must be an object")
	}
	if len(m) == 0 {
		return nil, nil //nolint:nilnil // nil map = no metadata, the reference's None
	}
	encoded, err := json.Marshal(m)
	if err != nil {
		return nil, NewGatewayRejection(400, "user_meta_data must be JSON-serializable")
	}
	if len(encoded) > MaxUserMetadataBytes {
		return nil, NewGatewayRejection(413, "user_meta_data too large")
	}
	return m, nil
}

// PrepareOptions carries what Prepare needs from the request besides its body.
type PrepareOptions struct {
	// Origin is the request's Origin header, or nil when absent.
	Origin *string `sw:"required" kind:"keyword"`
	// Key is the publishable key presented (the Bearer token), or nil.
	Key *string `sw:"required" kind:"keyword"`
}

// Prepare validates a browser request and builds the upstream JSON-RPC call:
// the method, its params, and the handle minted for a newly created conversation
// (nil otherwise). Everything the browser could use to widen its access is
// rejected or overwritten: the method must be one of four, the conversation
// comes from a signed handle, and config_url is the gateway's. A chat message
// over MaxMessageBytes is refused with 413 before a conversation is minted or a
// turn charged. Rejections are *GatewayRejection errors.
func (g *ChatGateway) Prepare(body map[string]any, opts PrepareOptions) (string, map[string]any, *string, error) {
	if err := g.CheckKey(opts.Key); err != nil {
		return "", nil, nil, err
	}
	if err := g.CheckOrigin(opts.Origin); err != nil {
		return "", nil, nil, err
	}
	method := "chat"
	if m, present := body["method"]; present {
		s, ok := m.(string)
		if !ok || !allowedGatewayMethods[s] {
			return "", nil, nil, NewGatewayRejection(400, "method not allowed")
		}
		method = s
	}
	// Read before minting so a malformed bag costs the caller nothing.
	userMetadata, err := g.ReadUserMetadata(body)
	if err != nil {
		return "", nil, nil, err
	}
	message, isString := body["message"].(string)
	if method == "chat" && isString && len(message) > MaxMessageBytes {
		return "", nil, nil, NewGatewayRejection(413, "message too large")
	}

	var conversationID string
	var minted *string
	if handle, _ := body["handle"].(string); handle != "" {
		conversationID, err = g.ReadHandle(handle)
		if err != nil {
			return "", nil, nil, err
		}
	} else if h, present := body["handle"]; present && h != nil && h != "" && h != false {
		return "", nil, nil, NewGatewayRejection(400, "malformed handle")
	} else if method == "end" || method == "log" {
		return "", nil, nil, NewGatewayRejection(400, method+" requires a handle")
	} else {
		if err := g.chargeMint(); err != nil {
			return "", nil, nil, err
		}
		h := g.MintHandle()
		minted = &h
		if conversationID, err = g.ReadHandle(h); err != nil {
			return "", nil, nil, err
		}
	}

	switch method {
	case "end":
		return "end_conversation", map[string]any{"id": conversationID}, nil, nil
	case "log":
		// Scoped to the conversation named INSIDE the signed handle.
		return "chat_log", map[string]any{"id": conversationID}, nil, nil
	case "start":
		// Opens the conversation with no user message, so the agent speaks first.
		params := map[string]any{"id": conversationID, "config_url": g.ConfigURL}
		if g.ConversationTimeout != 0 {
			params["conversation_timeout"] = g.ConversationTimeout
		}
		if userMetadata != nil {
			params["user_meta_data"] = userMetadata
		}
		return "create_conversation", params, minted, nil
	}

	if !isString || strings.TrimSpace(message) == "" {
		return "", nil, nil, NewGatewayRejection(400, "message is required")
	}
	if err := g.chargeTurn(conversationID); err != nil {
		return "", nil, nil, err
	}
	// config_url on every chat so the service auto-creates on the first one and
	// ignores it after; the timeout and metadata ride along for the same reason —
	// any chat may be the one that creates.
	params := map[string]any{"id": conversationID, "message": message, "config_url": g.ConfigURL}
	if g.ConversationTimeout != 0 {
		params["conversation_timeout"] = g.ConversationTimeout
	}
	if userMetadata != nil {
		params["user_meta_data"] = userMetadata
	}
	return "chat", params, minted, nil
}

// Router returns the HTTP handler exposing this gateway at "/" (mount it under a
// prefix with agent.Mount).
//
// POST takes {"method": "start"|"chat"|"log"|"end", "handle"?, "message"?,
// "user_meta_data"?} with the key in "Authorization: Bearer". A chat streams the
// service's JSON-RPC response body through unbuffered — the service pads slow
// turns with keepalive whitespace so proxies do not sever the connection, and
// collecting the body here would swallow that padding. A newly minted handle
// rides back in the X-Chat-Handle header. OPTIONS answers CORS preflight. A body
// over MaxRequestBodyBytes is answered 413 without being parsed.
func (g *ChatGateway) Router() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "" && r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodOptions:
			g.preflight(w, r)
		case http.MethodPost:
			g.proxy(w, r)
		default:
			w.Header().Set("Allow", "POST, OPTIONS")
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		}
	})
}

func headerPtr(r *http.Request, name string) *string {
	vals := r.Header.Values(name)
	if len(vals) == 0 {
		return nil
	}
	return &vals[0]
}

// cors returns CORS headers for an allowed origin, and none otherwise.
func (g *ChatGateway) cors(origin *string) map[string]string {
	if origin == nil || g.CheckOrigin(origin) != nil {
		return map[string]string{}
	}
	return map[string]string{
		"Access-Control-Allow-Origin":   *origin,
		"Access-Control-Expose-Headers": "X-Chat-Handle",
		"Vary":                          "Origin",
	}
}

func (g *ChatGateway) preflight(w http.ResponseWriter, r *http.Request) {
	headers := g.cors(headerPtr(r, "Origin"))
	if len(headers) > 0 {
		headers["Access-Control-Allow-Headers"] = "Authorization, Content-Type"
		headers["Access-Control-Allow-Methods"] = "POST, OPTIONS"
		headers["Access-Control-Max-Age"] = "600"
	}
	for k, v := range headers {
		w.Header().Set(k, v)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (g *ChatGateway) proxy(w http.ResponseWriter, r *http.Request) {
	origin := headerPtr(r, "Origin")
	var key *string
	if auth := r.Header.Get("Authorization"); len(auth) >= 7 && strings.EqualFold(auth[:7], "bearer ") {
		k := auth[7:]
		key = &k
	}
	cors := g.cors(origin)
	for k, v := range cors {
		w.Header().Set(k, v)
	}

	parsed, err := readJSONBody(r, MaxRequestBodyBytes)
	var method string
	var params map[string]any
	var minted *string
	if err == nil {
		body, ok := parsed.(map[string]any)
		if !ok {
			err = NewGatewayRejection(400, "body must be an object")
		} else {
			method, params, minted, err = g.Prepare(body, PrepareOptions{Origin: origin, Key: key})
		}
	}
	if err != nil {
		var rej *GatewayRejection
		if errors.As(err, &rej) {
			writeJSON(w, rej.Status, map[string]any{"error": rej.Reason})
		} else {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "bad request"})
		}
		return
	}

	ctx := r.Context()
	id, _ := params["id"].(string)
	switch method {
	case "end_conversation":
		if _, err := g.client.End(ctx, id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "upstream error"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ended"})
		return
	case "create_conversation":
		opts := CreateOptions{ConfigURL: g.ConfigURL, Timeout: g.ConversationTimeout}
		if um, ok := params["user_meta_data"].(map[string]any); ok {
			opts.UserMetadata = um
		}
		info, err := g.client.CreateConversation(ctx, id, opts)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "upstream error"})
			return
		}
		if minted != nil {
			w.Header().Set("X-Chat-Handle", *minted)
		}
		var greeting any
		if info.InitialMessage != "" {
			greeting = info.InitialMessage
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"greeting": greeting,
			"status":   info.Status,
			"timeout":  g.EffectiveTimeout(),
		})
		return
	case "chat_log":
		log, err := g.client.Log(ctx, id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "upstream error"})
			return
		}
		var last any
		if la := g.LastActivity(log.Messages); la != nil {
			last = *la
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"messages":      g.VisibleMessages(log.Messages),
			"timeout":       g.EffectiveTimeout(),
			"last_activity": last,
		})
		return
	}

	if minted != nil {
		w.Header().Set("X-Chat-Handle", *minted)
	}
	resp, err := g.client.RawPost(ctx, method, params)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "upstream error"})
		return
	}
	defer func() { _ = resp.Body.Close() }()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)
	buf := make([]byte, 4096)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if rerr != nil {
			return
		}
	}
}

// readJSONBody parses a request's JSON body, refusing one over limit bytes: a
// declared Content-Length over the limit is refused before anything is read, and
// a chunked or understated body is abandoned as soon as it passes the limit.
func readJSONBody(r *http.Request, limit int) (any, error) {
	if r.ContentLength > int64(limit) {
		return nil, NewGatewayRejection(413, "request too large")
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, NewGatewayRejection(413, "request too large")
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func writeJSON(w http.ResponseWriter, status int, body map[string]any) {
	data, err := json.Marshal(body)
	if err != nil {
		status, data = http.StatusInternalServerError, []byte(`{"error":"internal error"}`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func b64(raw []byte) string { return base64.RawURLEncoding.EncodeToString(raw) }

func unb64(text string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(strings.TrimRight(text, "="))
}
