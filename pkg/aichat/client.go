// Copyright (c) 2026 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Package aichat is a client for the SignalWire AI Chat service.
//
// The client speaks the standard SignalWire front-door protocol: HTTP Basic
// project:api_token with the space in the hostname —
// POST https://{space}.signalwire.com/api/ai/chat — carrying a JSON-RPC 2.0 body
// whose params are pure payload (identity never appears in the body; it rides the
// Basic-auth header only).
//
// A Chat call awaits a full LLM round trip (seconds, not milliseconds). The
// service streams keepalive whitespace ahead of a slow response body (proxy
// read-timeout protection), so liveness is byte-driven rather than wall-clock:
// there is no total-request timeout an idle turn could trip — only a per-read idle
// timeout, mirroring the Python reference's
// aiohttp.ClientTimeout(total=None, connect=10, sock_read=60). Leading whitespace
// is valid JSON, so the buffered decode is unaffected. Pass a context.Context to
// each call for cancellation.
//
// Mirrors the Python reference signalwire.ai_chat.AIChatClient.
//
// Example:
//
//	client, err := aichat.NewClient(aichat.WithSpace("myspace")) // env supplies creds
//	if err != nil {
//		log.Fatal(err)
//	}
//	if _, err := client.CreateConversation(ctx, "conv-1", aichat.CreateOptions{ConfigURL: cfgURL}); err != nil {
//		log.Fatal(err)
//	}
//	reply, err := client.Chat(ctx, "conv-1", "hello")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(reply.Text)
package aichat

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

// defaultPath is the endpoint path appended to a space-derived base URL.
const defaultPath = "/api/ai/chat"

// userAgent is the User-Agent header sent with every request.
const userAgent = "signalwire-go-ai-chat/1.0"

// defaultReadIdleTimeout bounds true byte-silence (a dead connection), NOT total
// turn length — mirroring the Python reference's sock_read=60. The service streams
// keepalive whitespace roughly every 10s, so a live-but-slow turn never trips it.
// net/http has no native per-read idle timeout, so this is applied via the
// transport's ResponseHeaderTimeout + a per-request context deadline the heartbeat
// keeps alive; a total wall-clock cap is deliberately absent so a slow-but-live
// turn is never severed by the client.
const defaultReadIdleTimeout = 60 * time.Second

// ── Response models ───────────────────────────────────────────────────

// ConversationInfo is the result of CreateConversation.
type ConversationInfo struct {
	// ID is the conversation id (echoed back — the caller's own input).
	ID string
	// Status is the lifecycle status the service reported (e.g. "created").
	Status string
	// InitialMessage is the opening assistant message, if the config produced one.
	InitialMessage string
}

// ChatResponse is the result of Chat.
type ChatResponse struct {
	// Text is the assistant's reply text (the wire "response" field).
	Text string
	// ConversationID is the conversation this reply belongs to.
	ConversationID string
	// UserEvent is an optional structured event the turn emitted, else nil.
	UserEvent map[string]any
}

// ChatLog is the result of Log.
type ChatLog struct {
	// Messages is the full message history (the wire "chat_log" field).
	Messages []map[string]any
	// CallTimeline is the call timeline (the wire "call_timeline" field).
	CallTimeline []map[string]any
}

// ── Client options ────────────────────────────────────────────────────

// Client is a client for the SignalWire AI Chat service. Construct it with
// NewClient; it is safe for concurrent use.
type Client struct {
	// URL is the fully-qualified endpoint requests are POSTed to.
	URL string

	authHeader      string
	httpClient      *http.Client
	readIdleTimeout time.Duration
	requestCounter  int
}

// Option configures a Client in NewClient (the functional-options idiom).
type Option func(*clientConfig)

type clientConfig struct {
	project         string
	token           string
	space           string
	url             string
	httpClient      *http.Client
	readIdleTimeout time.Duration
	hasReadIdle     bool
}

// WithProject sets the project id (Basic-auth username). Falls back to
// SIGNALWIRE_PROJECT_ID when unset.
func WithProject(project string) Option { return func(c *clientConfig) { c.project = project } }

// WithToken sets the API token (Basic-auth password). Falls back to
// SIGNALWIRE_API_TOKEN when unset.
func WithToken(token string) Option { return func(c *clientConfig) { c.token = token } }

// WithSpace sets the space name; the client builds
// https://{space}.signalwire.com/api/ai/chat. Falls back to SIGNALWIRE_SPACE.
func WithSpace(space string) Option { return func(c *clientConfig) { c.space = space } }

// WithURL sets a fully-qualified endpoint URL, used verbatim (highest precedence).
func WithURL(url string) Option { return func(c *clientConfig) { c.url = url } }

// WithHTTPClient injects a custom *http.Client (dependency injection for tests /
// custom transports). When set, the read-idle/connect timeouts are the caller's
// responsibility.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *clientConfig) { c.httpClient = hc }
}

// WithReadIdleTimeout overrides the per-read idle timeout (byte-silence, NOT total
// turn length). Default 60s. A value <= 0 disables it.
func WithReadIdleTimeout(d time.Duration) Option {
	return func(c *clientConfig) { c.readIdleTimeout = d; c.hasReadIdle = true }
}

// NewClient constructs a Client. project is required (option or
// SIGNALWIRE_PROJECT_ID); a target URL must resolve from url, space, or
// SIGNALWIRE_SPACE.
func NewClient(opts ...Option) (*Client, error) {
	cfg := &clientConfig{}
	for _, o := range opts {
		o(cfg)
	}
	project := firstNonEmpty(cfg.project, os.Getenv("SIGNALWIRE_PROJECT_ID"))
	token := firstNonEmpty(cfg.token, os.Getenv("SIGNALWIRE_API_TOKEN"))
	space := firstNonEmpty(cfg.space, os.Getenv("SIGNALWIRE_SPACE"))

	if project == "" {
		return nil, fmt.Errorf("project is required: pass aichat.WithProject or set the SIGNALWIRE_PROJECT_ID environment variable")
	}

	url, err := resolveURL(cfg.url, space)
	if err != nil {
		return nil, err
	}

	readIdle := defaultReadIdleTimeout
	if cfg.hasReadIdle {
		readIdle = cfg.readIdleTimeout
	}

	hc := cfg.httpClient
	if hc == nil {
		hc = newHTTPClient()
	}

	return &Client{
		URL:             url,
		authHeader:      "Basic " + base64.StdEncoding.EncodeToString([]byte(project+":"+token)),
		httpClient:      hc,
		readIdleTimeout: readIdle,
	}, nil
}

// newHTTPClient builds the default *http.Client. It sets a bounded connect timeout
// and a ResponseHeaderTimeout (the closest net/http analogue to sock_read: it bounds
// silence waiting for response headers), but deliberately leaves the overall
// Client.Timeout at 0 — a slow-but-live turn must never be severed by a wall-clock
// cap. The per-call read-idle deadline is applied per request in do().
func newHTTPClient() *http.Client {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		// http.DefaultTransport is always an *http.Transport in the standard
		// library; fall back to a fresh one if a test replaced it.
		return &http.Client{}
	}
	tr := transport.Clone()
	tr.ResponseHeaderTimeout = defaultReadIdleTimeout
	return &http.Client{Transport: tr}
}

func resolveURL(url, space string) (string, error) {
	if url != "" {
		return url, nil
	}
	if space != "" {
		return "https://" + space + ".signalwire.com" + defaultPath, nil
	}
	return "", fmt.Errorf("no service URL: pass aichat.WithURL or aichat.WithSpace / set SIGNALWIRE_SPACE")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// ── Wire ───────────────────────────────────────────────────────────────

type jsonRPCRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params"`
	ID      string         `json:"id"`
}

type jsonRPCError struct {
	Code    *int   `json:"code"`
	Message string `json:"message"`
}

type jsonRPCResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *jsonRPCError   `json:"error"`
}

// request POSTs one JSON-RPC call and returns its decoded result object.
//
// Success/failure is decided by the JSON-RPC BODY, not the HTTP status: the
// service's keepalive heartbeat commits 200 before the turn's outcome is known, so
// a slow error can arrive as 200 + {"error": …}. The status is never gated on here
// (mirrors the Python reference).
func (c *Client) request(ctx context.Context, method string, params map[string]any) (map[string]any, error) {
	c.requestCounter++
	payload := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      "req-" + strconv.Itoa(c.requestCounter),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("aichat: encode request: %w", err)
	}

	// A per-request read-idle deadline. Because the mock (and the real proxy)
	// heartbeat well within the window, a live-but-slow turn never trips it, while a
	// truly dead connection is severed after readIdleTimeout of silence. <= 0
	// disables it.
	if c.readIdleTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.readIdleTimeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("aichat: build request: %w", err)
	}
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("aichat: %s request failed: %w", method, err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Buffer the whole body then decode. Leading keepalive whitespace is valid JSON,
	// so a plain decode handles it — no need to strip.
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &AIChatError{Code: resp.StatusCode, HasCode: true, Message: fmt.Sprintf("read response failed (HTTP %d)", resp.StatusCode)}
	}
	var decoded jsonRPCResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, &AIChatError{Code: resp.StatusCode, HasCode: true, Message: fmt.Sprintf("non-JSON response (HTTP %d)", resp.StatusCode)}
	}

	if decoded.Error != nil {
		if decoded.Error.Code != nil {
			return nil, newTypedError(*decoded.Error.Code, decoded.Error.Message)
		}
		return nil, &AIChatError{Message: decoded.Error.Message}
	}

	if len(decoded.Result) == 0 {
		return map[string]any{}, nil
	}
	var result map[string]any
	if err := json.Unmarshal(decoded.Result, &result); err != nil {
		return map[string]any{}, nil
	}
	return result, nil
}

// ── Per-call option types ──────────────────────────────────────────────

// CreateOptions are the options for CreateConversation. ConfigURL is required.
type CreateOptions struct {
	// ConfigURL locates the agent config (required).
	ConfigURL string
	// UserMessage is the opening user message to send with the create (wire
	// "user_message").
	UserMessage string
	// Timeout is the conversation inactivity timeout in seconds (wire
	// "conversation_timeout"). Omitted when 0.
	Timeout int
	// Reinit reinitializes an existing conversation.
	Reinit bool
	// UserMetadata is arbitrary caller metadata (wire "user_meta_data").
	UserMetadata map[string]any
}

// ChatOptions are the options for Chat. All fields are optional.
type ChatOptions struct {
	// Role is the message role ("user" or "system"). Defaults to "user".
	Role string
	// ConfigURL, when set, auto-creates the conversation if it doesn't exist yet.
	ConfigURL string
	// Timeout is the conversation inactivity timeout in seconds (applies to the
	// auto-create). Omitted when 0.
	Timeout int
	// Reinit reinitializes an existing conversation (applies to the auto-create).
	Reinit bool
	// UserMetadata is arbitrary caller metadata (wire "user_meta_data").
	UserMetadata map[string]any
}

// SummarizeOptions are the sampling / prompt options for Summarize. All fields are
// optional.
type SummarizeOptions struct {
	// SummaryPrompt is a custom prompt steering the summary (wire "summary_prompt").
	SummaryPrompt string
	// Temperature, TopP, FrequencyPenalty, PresencePenalty, and MaxTokens are
	// sampling parameters, each sent only when its pointer is non-nil.
	Temperature      *float64
	TopP             *float64
	FrequencyPenalty *float64
	PresencePenalty  *float64
	MaxTokens        *int
}

// ── API methods ────────────────────────────────────────────────────────

// CreateConversation creates a conversation (or, with Reinit, reinitializes an
// existing one) and optionally sends its opening user message. opts.ConfigURL is
// required.
func (c *Client) CreateConversation(ctx context.Context, conversationID string, opts CreateOptions) (ConversationInfo, error) {
	params := map[string]any{"id": conversationID, "config_url": opts.ConfigURL}
	if opts.UserMessage != "" {
		params["user_message"] = opts.UserMessage
	}
	if opts.Timeout != 0 {
		params["conversation_timeout"] = opts.Timeout
	}
	if len(opts.UserMetadata) > 0 {
		params["user_meta_data"] = opts.UserMetadata
	}
	if opts.Reinit {
		params["reinit"] = true
	}
	result, err := c.request(ctx, "create_conversation", params)
	if err != nil {
		return ConversationInfo{}, err
	}
	return ConversationInfo{
		ID:             conversationID,
		Status:         stringField(result, "status", "created"),
		InitialMessage: stringField(result, "initial_message", ""),
	}, nil
}

// Chat sends a message and awaits a full LLM round trip (expect seconds). Passing
// opts.ConfigURL auto-creates the conversation if it doesn't exist yet; Timeout and
// Reinit apply to that auto-create, with the same meaning as on CreateConversation.
func (c *Client) Chat(ctx context.Context, conversationID, message string, opts ...ChatOptions) (ChatResponse, error) {
	var o ChatOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	role := o.Role
	if role == "" {
		role = "user"
	}
	params := map[string]any{"id": conversationID, "message": message, "role": role}
	if o.ConfigURL != "" {
		params["config_url"] = o.ConfigURL
	}
	if len(o.UserMetadata) > 0 {
		params["user_meta_data"] = o.UserMetadata
	}
	if o.Timeout != 0 {
		params["conversation_timeout"] = o.Timeout
	}
	if o.Reinit {
		params["reinit"] = true
	}
	result, err := c.request(ctx, "chat", params)
	if err != nil {
		return ChatResponse{}, err
	}
	return ChatResponse{
		Text:           stringField(result, "response", ""),
		ConversationID: conversationID,
		UserEvent:      mapField(result, "user_event"),
	}, nil
}

// Close releases any resources the client owns. The client wraps a stateless,
// connection-pooled *http.Client (shared, or caller-injected via WithHTTPClient),
// which has no per-client resource to release, so Close is a no-op that completes
// the lifecycle contract — the Go analogue of the Python reference's close()
// (which releases its owned aiohttp ClientSession). It always returns nil and is
// safe to call more than once.
func (c *Client) Close() error { return nil }

// End ends a conversation (triggers server-side post-processing / archival). It
// returns true when the service reported the conversation ended.
func (c *Client) End(ctx context.Context, conversationID string) (bool, error) {
	result, err := c.request(ctx, "end_conversation", map[string]any{"id": conversationID})
	if err != nil {
		return false, err
	}
	return stringField(result, "status", "") == "ended", nil
}

// Delete permanently deletes a conversation and its data (idempotent). It returns
// true when the service reported the conversation deleted.
func (c *Client) Delete(ctx context.Context, conversationID string) (bool, error) {
	result, err := c.request(ctx, "delete", map[string]any{"id": conversationID})
	if err != nil {
		return false, err
	}
	return stringField(result, "status", "") == "deleted", nil
}

// Log returns the full message history plus the call timeline.
func (c *Client) Log(ctx context.Context, conversationID string) (ChatLog, error) {
	result, err := c.request(ctx, "chat_log", map[string]any{"id": conversationID})
	if err != nil {
		return ChatLog{}, err
	}
	return ChatLog{
		Messages:     mapSliceField(result, "chat_log"),
		CallTimeline: mapSliceField(result, "call_timeline"),
	}, nil
}

// Summarize returns an AI summary of the conversation (rate limited server-side).
//
// The service returns EXACTLY ONE of {summary} or {error} — BOTH on the success
// envelope — so a failed generation surfaces as a *SummaryError, never as an empty
// string. On the {error} branch this returns ("", *SummaryError).
func (c *Client) Summarize(ctx context.Context, conversationID string, opts ...SummarizeOptions) (string, error) {
	var o SummarizeOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	params := map[string]any{"id": conversationID}
	if o.SummaryPrompt != "" {
		params["summary_prompt"] = o.SummaryPrompt
	}
	if o.Temperature != nil {
		params["temperature"] = *o.Temperature
	}
	if o.TopP != nil {
		params["top_p"] = *o.TopP
	}
	if o.FrequencyPenalty != nil {
		params["frequency_penalty"] = *o.FrequencyPenalty
	}
	if o.PresencePenalty != nil {
		params["presence_penalty"] = *o.PresencePenalty
	}
	if o.MaxTokens != nil {
		params["max_tokens"] = *o.MaxTokens
	}
	result, err := c.request(ctx, "summarize", params)
	if err != nil {
		return "", err
	}
	_, hasError := result["error"]
	_, hasSummary := result["summary"]
	if hasError && !hasSummary {
		return "", &SummaryError{&AIChatError{Message: fmt.Sprint(result["error"])}}
	}
	return stringField(result, "summary", ""), nil
}

// ── decode helpers ─────────────────────────────────────────────────────

func stringField(m map[string]any, key, def string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return def
}

func mapField(m map[string]any, key string) map[string]any {
	if v, ok := m[key].(map[string]any); ok {
		return v
	}
	return nil
}

func mapSliceField(m map[string]any, key string) []map[string]any {
	arr, ok := m[key].([]any)
	if !ok {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(arr))
	for _, e := range arr {
		if mm, ok := e.(map[string]any); ok {
			out = append(out, mm)
		}
	}
	return out
}
