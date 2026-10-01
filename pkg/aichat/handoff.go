// Copyright (c) 2026 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package aichat

// Moving one conversation between voice and text.
//
// ChatGateway lets a browser hold a text conversation. HandoffRouter is the other
// half a browser client needs: the three routes it calls to move that
// conversation to a phone call and back, and to type into a live call. The
// SignalWire address widget calls {gateway-url}/handoff, /escalate and /say
// against the same URL as the gateway, so mount both under one prefix.
//
// Mechanism vs policy: this type owns the wire contract — the routes, the nonce,
// the ordering guarantee and the spend guards. What a conversation IS (where a
// leg's transcript is written, what a resumed greeting says) is the
// application's, injected as callbacks.
//
// The nonce: a browser cannot be trusted to name a call (a page-supplied call id
// would let anyone inject speech into a stranger's call). The application puts a
// random handoff_nonce in the user variables of one dial, registers it here
// against that call's ids, and the browser presents it later. A nonce is
// registered once (the first registration stands); redemption for a handle is
// single use; typing is repeatable, bounded by MaxMessagesPerCall, until the
// nonce is redeemed or NonceTTL passes. An unknown nonce is answered exactly
// like an expired or redeemed one.
//
// The ordering guarantee: a medium never starts until the one it replaces has
// finished and its record is durable. /handoff ends the call and waits for
// CaptureLeg before minting a handle; /escalate ends the chat leg and waits
// before returning.
//
// Deployment: the nonce registry lives in this process. A redemption must reach
// the replica that served the dial — run one replica, use sticky routing, or
// supply a shared Registry. Registration, redemption and the typing count are
// atomic within one HandoffRouter only.

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/signalwire/signalwire-go/v3/pkg/logging"
)

// Handoff defaults.
const (
	// DefaultNonceTTL is seconds a nonce stays usable after registration.
	DefaultNonceTTL = 3600
	// DefaultMaxMessagesPerCall caps typed messages for one call.
	DefaultMaxMessagesPerCall = 200
	// DefaultCaptureTimeout is seconds to wait for CaptureLeg.
	DefaultCaptureTimeout = 8.0
)

// NonceEntry is what a nonce is a capability for. Redeemed marks a nonce /handoff
// has exchanged for a handle; the entry is kept until its TTL passes, so the
// nonce can be neither redeemed nor registered again.
type NonceEntry struct {
	// ConversationID is the conversation the nonce belongs to.
	ConversationID string
	// CallID is the call the nonce was registered against, or nil.
	CallID *string
	// IssuedAt is when the nonce was first registered, in seconds on the
	// process's monotonic clock.
	IssuedAt float64
	// Messages counts typed messages delivered through /say.
	Messages int
	// Redeemed reports whether /handoff consumed the nonce.
	Redeemed bool
}

// monotonicBase anchors monotonicSeconds; time.Since reads Go's monotonic clock.
var monotonicBase = time.Now()

// monotonicSeconds is seconds on this process's monotonic clock — the clock
// NonceEntry.IssuedAt is measured on (unaffected by wall-clock changes).
func monotonicSeconds() float64 { return time.Since(monotonicBase).Seconds() }

// HandoffRouterOptions configures NewHandoffRouter. A zero numeric field takes
// the documented default.
type HandoffRouterOptions struct {
	// Gateway owns the conversations: it mints handles and checks origins, so
	// both halves of the URL enforce one origin policy. Required.
	Gateway *ChatGateway `sw:"required" kind:"keyword"`
	// CaptureLeg is called as CaptureLeg(conversationID, medium) to end a leg and
	// write its record; return true only once the record is durable (an error
	// counts as not captured). Nil means no wait, and no ordering guarantee.
	CaptureLeg func(conversationID, medium string) (bool, error) `sw:"optional" kind:"keyword" gen:"optional<callable<list<string,string>,union<bool,bool>>>"`
	// EndCall is called as EndCall(callID) to hang the call up server-side.
	EndCall func(callID string) error `sw:"optional" kind:"keyword" gen:"optional<callable<list<string>,void>>"`
	// SendMessage is called as SendMessage(callID, text) for /say; an error
	// means not delivered. Nil leaves typing disabled (/say answers 404).
	SendMessage func(callID, text string) (bool, error) `sw:"optional" kind:"keyword" gen:"optional<callable<list<string,string>,union<bool,bool>>>"`
	// NextConversationID produces the id for the NEW leg. Defaults to appending
	// ".N" ("root" -> "root.1", "root.2" -> "root.3").
	NextConversationID func(conversationID string) string `sw:"optional" kind:"keyword" gen:"optional<callable<list<string>,string>>"`
	// NonceTTL is seconds a nonce stays usable after registration.
	NonceTTL int `sw:"optional" kind:"keyword"`
	// MaxMessagesPerCall caps typed messages for one call — each is a billable
	// turn, so this is a spend guard as much as an abuse one.
	MaxMessagesPerCall int `sw:"optional" kind:"keyword"`
	// CaptureTimeout is seconds to wait for CaptureLeg.
	CaptureTimeout float64 `sw:"optional" kind:"keyword"`
	// Registry is an optional shared nonce table. A redemption is stored by
	// assigning the marked entry back to its key.
	Registry map[string]NonceEntry `sw:"optional" kind:"keyword" gen:"optional<dict<string,class:signalwire.ai_chat.handoff.NonceEntry>>"`
}

// HandoffRouter serves the three routes a browser client needs beside a
// ChatGateway: /handoff (redeem a call's nonce for a chat handle), /escalate
// (end a chat leg before a call is placed) and /say (type into a live call).
type HandoffRouter struct {
	// Gateway owns the conversations.
	Gateway *ChatGateway
	// CaptureLeg ends a leg and writes its record (see HandoffRouterOptions).
	CaptureLeg func(conversationID, medium string) (bool, error)
	// EndCall hangs a call up server-side.
	EndCall func(callID string) error
	// SendMessage injects typed text into a live call.
	SendMessage func(callID, text string) (bool, error)
	// NextConversationID produces the id for a new leg.
	NextConversationID func(conversationID string) string
	// NonceTTL is seconds a nonce stays usable.
	NonceTTL int
	// MaxMessagesPerCall caps typed messages per call.
	MaxMessagesPerCall int
	// CaptureTimeout is seconds to wait for CaptureLeg.
	CaptureTimeout float64

	mu     sync.Mutex
	nonces map[string]NonceEntry
	now    func() float64
	logger *logging.Logger
}

// NewHandoffRouter configures the handoff; see HandoffRouterOptions.
func NewHandoffRouter(opts HandoffRouterOptions) (*HandoffRouter, error) {
	if opts.Gateway == nil {
		return nil, errors.New("aichat: HandoffRouter requires a Gateway")
	}
	h := &HandoffRouter{
		Gateway:            opts.Gateway,
		CaptureLeg:         opts.CaptureLeg,
		EndCall:            opts.EndCall,
		SendMessage:        opts.SendMessage,
		NextConversationID: opts.NextConversationID,
		NonceTTL:           orDefault(opts.NonceTTL, DefaultNonceTTL),
		MaxMessagesPerCall: orDefault(opts.MaxMessagesPerCall, DefaultMaxMessagesPerCall),
		CaptureTimeout:     opts.CaptureTimeout,
		nonces:             opts.Registry,
		now:                monotonicSeconds,
		logger:             logging.New("ai_chat.handoff"),
	}
	if h.CaptureTimeout == 0 {
		h.CaptureTimeout = DefaultCaptureTimeout
	}
	if h.NextConversationID == nil {
		h.NextConversationID = defaultNextConversationID
	}
	if h.nonces == nil {
		h.nonces = map[string]NonceEntry{}
	}
	return h, nil
}

// defaultNextConversationID: "root" -> "root.1"; "root.2" -> "root.3". "." because
// the chat service strips "~", "_" and "-" occur inside generated ids, and ":" is
// the gateway's handle delimiter.
func defaultNextConversationID(conversationID string) string {
	if i := strings.LastIndex(conversationID, "."); i > 0 {
		tail := conversationID[i+1:]
		if tail != "" && strings.Trim(tail, "0123456789") == "" {
			if n, err := strconv.Atoi(tail); err == nil {
				return conversationID[:i] + "." + strconv.Itoa(n+1)
			}
		}
	}
	return conversationID + ".1"
}

// RegisterOptions names what a nonce is registered against.
type RegisterOptions struct {
	// ConversationID is the conversation the nonce belongs to.
	ConversationID string `sw:"required" kind:"keyword"`
	// CallID is the call that carried the nonce, read from the platform's
	// request; empty when there is none.
	CallID string `sw:"optional" kind:"keyword" gen:"optional<string>"`
}

// Register records what a nonce is a capability for. Call it from the
// dynamic-config callback of the dial that carried the nonce, reading callID from
// the request the platform sent — never from anything the browser supplied.
//
// The first registration stands: registering a nonce already in the table
// changes nothing (its call, registration time and typed-message count are kept,
// and a redeemed nonce stays redeemed); a conflicting re-registration is logged.
// Once the entry's NonceTTL has passed, the nonce can be registered again.
func (h *HandoffRouter) Register(nonce string, opts RegisterOptions) {
	if nonce == "" {
		return
	}
	conversationID := opts.ConversationID
	var call *string
	if opts.CallID != "" {
		c := opts.CallID
		call = &c
	}
	h.mu.Lock()
	h.prune()
	existing, found := h.nonces[nonce]
	if !found {
		h.nonces[nonce] = NonceEntry{ConversationID: conversationID, CallID: call, IssuedAt: h.now()}
	}
	h.mu.Unlock()
	if found {
		if existing.Redeemed || existing.ConversationID != conversationID || !sameCall(existing.CallID, call) {
			h.logger.Warn("handoff_nonce_already_registered: conversation_id=%s redeemed=%t",
				existing.ConversationID, existing.Redeemed)
		}
		return
	}
	h.logger.Info("handoff_nonce_registered: conversation_id=%s", conversationID)
}

func sameCall(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// prune drops entries, redeemed ones included, whose TTL has passed. Call with
// the lock held.
func (h *HandoffRouter) prune() {
	cutoff := h.now() - float64(h.NonceTTL)
	for n, e := range h.nonces {
		if e.IssuedAt < cutoff {
			delete(h.nonces, n)
		}
	}
}

// lookup returns the live entry for nonce: false if unknown, expired or redeemed.
// Call with the lock held.
func (h *HandoffRouter) lookup(nonce string) (NonceEntry, bool) {
	if nonce == "" {
		return NonceEntry{}, false
	}
	h.prune()
	e, ok := h.nonces[nonce]
	if !ok || e.Redeemed {
		return NonceEntry{}, false
	}
	return e, true
}

// capture waits for the application's CaptureLeg, bounded by CaptureTimeout.
// Never fails: a timeout, error or panic is logged and reported as not captured.
func (h *HandoffRouter) capture(conversationID, medium string) bool {
	if h.CaptureLeg == nil {
		return false
	}
	type result struct {
		ok  bool
		err error
	}
	done := make(chan result, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- result{err: fmt.Errorf("panic: %v", r)}
			}
		}()
		ok, err := h.CaptureLeg(conversationID, medium)
		done <- result{ok: ok, err: err}
	}()
	timer := time.NewTimer(time.Duration(h.CaptureTimeout * float64(time.Second)))
	defer timer.Stop()
	select {
	case r := <-done:
		if r.err != nil {
			h.logger.Error("handoff_capture_failed: conversation_id=%s error=%v", conversationID, r.err)
			return false
		}
		return r.ok
	case <-timer.C:
		h.logger.Warn("handoff_capture_timeout: conversation_id=%s medium=%s note=starting the next medium without this leg's record",
			conversationID, medium)
		return false
	}
}

// Redeem exchanges a nonce for a chat handle. Single use: it ends the call, waits
// for its record, and only then mints a handle for a new leg of the same
// conversation. Returns nil for an unknown, expired or already redeemed nonce —
// deliberately indistinguishable.
func (h *HandoffRouter) Redeem(nonce string) *string {
	h.mu.Lock()
	entry, ok := h.lookup(nonce)
	if !ok {
		h.mu.Unlock()
		return nil
	}
	// Consumed even if what follows fails: a nonce is one attempt. Written back
	// so a shared registry stores the change.
	entry.Redeemed = true
	h.nonces[nonce] = entry
	h.mu.Unlock()

	if entry.CallID != nil && h.EndCall != nil {
		if err := safeCall(func() error { return h.EndCall(*entry.CallID) }); err != nil {
			h.logger.Warn("handoff_end_call_failed: error=%v", err)
		}
	}
	h.capture(entry.ConversationID, "voice")

	var next string
	if err := safeCall(func() error { next = h.NextConversationID(entry.ConversationID); return nil }); err != nil {
		h.logger.Error("handoff_mint_failed: error=%v", err)
		return nil
	}
	handle := h.Gateway.MintHandle(next)
	h.logger.Info("handoff_redeemed: conversation_id=%s", entry.ConversationID)
	return &handle
}

// Escalate ends a chat leg and waits for its record, before a call is placed, so
// a voice leg started immediately afterwards finds the text leg already
// recorded. False for a handle that does not verify.
func (h *HandoffRouter) Escalate(handle string) bool {
	conversationID, err := h.Gateway.ReadHandle(handle)
	if err != nil {
		return false
	}
	h.capture(conversationID, "chat")
	h.logger.Info("handoff_escalated: conversation_id=%s", conversationID)
	return true
}

// Say delivers typed text into the live call the nonce names. It does not consume
// the nonce: typing is repeatable until the nonce is redeemed or NonceTTL passes,
// up to MaxMessagesPerCall messages. No other request field is forwarded. Text
// over MaxMessageBytes is refused.
func (h *HandoffRouter) Say(nonce, text string) bool {
	if h.SendMessage == nil {
		return false
	}
	cleaned := strings.TrimSpace(text)
	if cleaned == "" || len(cleaned) > MaxMessageBytes {
		return false
	}
	h.mu.Lock()
	entry, ok := h.lookup(nonce)
	if !ok || entry.CallID == nil || *entry.CallID == "" {
		h.mu.Unlock()
		return false
	}
	if entry.Messages >= h.MaxMessagesPerCall {
		h.mu.Unlock()
		h.logger.Warn("handoff_say_cap_reached: call_id=%s", *entry.CallID)
		return false
	}
	// Take the message's slot before delivering, so overlapping requests can't
	// all pass the cap.
	entry.Messages++
	h.nonces[nonce] = entry
	h.mu.Unlock()

	err := safeCall(func() error { _, e := h.SendMessage(*entry.CallID, cleaned); return e })
	if err == nil {
		return true
	}
	h.logger.Error("handoff_say_failed: error=%v", err)
	h.mu.Lock()
	// Not delivered: give the slot back, if the table still holds this
	// registration (matched by value; a shared registry may hold a copy).
	if cur, ok := h.nonces[nonce]; ok && cur.Messages > 0 &&
		cur.ConversationID == entry.ConversationID && sameCall(cur.CallID, entry.CallID) &&
		cur.IssuedAt == entry.IssuedAt {
		cur.Messages--
		h.nonces[nonce] = cur
	}
	h.mu.Unlock()
	return false
}

// safeCall runs an application callback, turning a panic into an error.
func safeCall(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return fn()
}

// Router returns the HTTP handler serving /handoff, /escalate and /say. Mount it
// at the SAME prefix as the gateway's router — the browser derives all three
// paths from one configured URL:
//
//	a.Mount(gw.Router(), agent.MountOptions{Prefix: "/chat"})
//	a.Mount(handoff.Router(), agent.MountOptions{Prefix: "/chat"})
//
// Every route answers 413 for a body over MaxRequestBodyBytes, and /say for text
// over MaxMessageBytes, before the nonce is looked up.
func (h *HandoffRouter) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /handoff", h.handleHandoff)
	mux.HandleFunc("POST /escalate", h.handleEscalate)
	mux.HandleFunc("POST /say", h.handleSay)
	return mux
}

// forbiddenOrigin writes 403 for a disallowed Origin and reports whether it did.
func (h *HandoffRouter) forbiddenOrigin(w http.ResponseWriter, r *http.Request) bool {
	if h.Gateway.CheckOrigin(headerPtr(r, "Origin")) != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "origin not allowed"})
		return true
	}
	return false
}

// body returns the JSON object sent ({} for anything else), or a 413 rejection.
func (h *HandoffRouter) body(r *http.Request) (map[string]any, *GatewayRejection) {
	data, err := readJSONBody(r, MaxRequestBodyBytes)
	if err != nil {
		var rej *GatewayRejection
		if errors.As(err, &rej) {
			return nil, rej
		}
		return map[string]any{}, nil
	}
	m, ok := data.(map[string]any)
	if !ok {
		return map[string]any{}, nil
	}
	return m, nil
}

func (h *HandoffRouter) handleHandoff(w http.ResponseWriter, r *http.Request) {
	if h.forbiddenOrigin(w, r) {
		return
	}
	data, rej := h.body(r)
	if rej != nil {
		writeJSON(w, rej.Status, map[string]any{"error": rej.Reason})
		return
	}
	nonce, ok := data["nonce"].(string)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	handle := h.Redeem(nonce)
	if handle == nil {
		// Same answer for unknown, expired and already-redeemed.
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"handle": *handle})
}

func (h *HandoffRouter) handleEscalate(w http.ResponseWriter, r *http.Request) {
	if h.forbiddenOrigin(w, r) {
		return
	}
	data, rej := h.body(r)
	if rej != nil {
		writeJSON(w, rej.Status, map[string]any{"error": rej.Reason})
		return
	}
	handle, ok := data["handle"].(string)
	if !ok || handle == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "bad request"})
		return
	}
	if !h.Escalate(handle) {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *HandoffRouter) handleSay(w http.ResponseWriter, r *http.Request) {
	if h.forbiddenOrigin(w, r) {
		return
	}
	data, rej := h.body(r)
	if rej != nil {
		writeJSON(w, rej.Status, map[string]any{"error": rej.Reason})
		return
	}
	nonce, okN := data["nonce"].(string)
	text := ""
	okT := true
	if raw, present := data["text"]; present {
		text, okT = raw.(string)
	}
	if !okN || !okT {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	if len(text) > MaxMessageBytes {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]any{"error": "message too large"})
		return
	}
	if !h.Say(nonce, text) {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
