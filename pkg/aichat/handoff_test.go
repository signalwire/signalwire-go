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
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type handoffRig struct {
	router *HandoffRouter
	gw     *ChatGateway
	mu     sync.Mutex
	events []string
	said   []string
}

func (r *handoffRig) record(e string) {
	r.mu.Lock()
	r.events = append(r.events, e)
	r.mu.Unlock()
}

func newHandoffRig(t *testing.T, mutate func(*HandoffRouterOptions)) *handoffRig {
	t.Helper()
	rig := &handoffRig{gw: newGateway(t, ChatGatewayOptions{AllowedOrigins: []string{"https://shop.example.com"}})}
	opts := HandoffRouterOptions{
		Gateway: rig.gw,
		CaptureLeg: func(conversationID, medium string) (bool, error) {
			rig.record("capture:" + conversationID + ":" + medium)
			return true, nil
		},
		EndCall: func(callID string) error {
			rig.record("end:" + callID)
			return nil
		},
		SendMessage: func(callID, text string) (bool, error) {
			rig.mu.Lock()
			rig.said = append(rig.said, callID+":"+text)
			rig.mu.Unlock()
			return true, nil
		},
	}
	if mutate != nil {
		mutate(&opts)
	}
	h, err := NewHandoffRouter(opts)
	if err != nil {
		t.Fatal(err)
	}
	rig.router = h
	return rig
}

func TestRedeemEndsTheCallBeforeCaptureAndMintsANewLeg(t *testing.T) {
	rig := newHandoffRig(t, nil)
	rig.router.Register("n1", RegisterOptions{ConversationID: "conv", CallID: "call-1"})
	handle := rig.router.Redeem("n1")
	if handle == nil {
		t.Fatal("redeem returned nil")
	}
	id, err := rig.gw.ReadHandle(*handle)
	if err != nil || id != "conv.1" {
		t.Errorf("new leg id = %q, %v", id, err)
	}
	if strings.Join(rig.events, ",") != "end:call-1,capture:conv:voice" {
		t.Errorf("ordering = %v", rig.events)
	}
	if rig.router.Redeem("n1") != nil {
		t.Error("a nonce redeemed twice")
	}
	if rig.router.Redeem("never-registered") != nil {
		t.Error("unknown nonce redeemed")
	}
	if got := defaultNextConversationID("root.2"); got != "root.3" {
		t.Errorf("next id = %s", got)
	}
	if got := defaultNextConversationID("root.x"); got != "root.x.1" {
		t.Errorf("next id = %s", got)
	}
}

func TestRegistrationIsFirstWins(t *testing.T) {
	rig := newHandoffRig(t, nil)
	rig.router.Register("n", RegisterOptions{ConversationID: "conv", CallID: "call-1"})
	if !rig.router.Say("n", "one") {
		t.Fatal("say failed")
	}
	rig.router.Register("n", RegisterOptions{ConversationID: "conv", CallID: "call-1"})  // repeat: keeps the count
	rig.router.Register("n", RegisterOptions{ConversationID: "other", CallID: "call-2"}) // conflicting: ignored
	if e := rig.router.nonces["n"]; e.Messages != 1 || *e.CallID != "call-1" || e.ConversationID != "conv" {
		t.Errorf("entry = %+v", e)
	}
	_ = rig.router.Redeem("n")
	rig.router.Register("n", RegisterOptions{ConversationID: "conv", CallID: "call-1"})
	if rig.router.Redeem("n") != nil {
		t.Error("a redeemed nonce came back to life")
	}
	if rig.router.Say("n", "after") {
		t.Error("a redeemed nonce could type")
	}
}

func TestNoncesExpire(t *testing.T) {
	rig := newHandoffRig(t, func(o *HandoffRouterOptions) { o.NonceTTL = 60 })
	rig.router.Register("n", RegisterOptions{ConversationID: "conv", CallID: "call-1"})
	rig.router.now = func() float64 { return monotonicSeconds() + 61 }
	if rig.router.Say("n", "late") || rig.router.Redeem("n") != nil {
		t.Error("expired nonce still usable")
	}
}

func TestSayCapsAndFailures(t *testing.T) {
	rig := newHandoffRig(t, func(o *HandoffRouterOptions) { o.MaxMessagesPerCall = 2 })
	rig.router.Register("n", RegisterOptions{ConversationID: "conv", CallID: "call-1"})
	if !rig.router.Say("n", "  hi  ") || rig.said[0] != "call-1:hi" {
		t.Fatalf("said = %v", rig.said)
	}
	if rig.router.Say("n", "   ") {
		t.Error("empty text delivered")
	}
	if !rig.router.Say("n", "two") || rig.router.Say("n", "three") {
		t.Error("cap not enforced at 2")
	}
	if rig.router.Say("n", strings.Repeat("a", MaxMessageBytes+1)) {
		t.Error("oversized text delivered")
	}

	failing := newHandoffRig(t, func(o *HandoffRouterOptions) {
		o.SendMessage = func(string, string) (bool, error) { return false, errors.New("down") }
	})
	failing.router.Register("n", RegisterOptions{ConversationID: "conv", CallID: "call-1"})
	if failing.router.Say("n", "hi") {
		t.Error("failed delivery reported success")
	}
	if e := failing.router.nonces["n"]; e.Messages != 0 {
		t.Errorf("slot not given back: %+v", e)
	}

	disabled := newHandoffRig(t, func(o *HandoffRouterOptions) { o.SendMessage = nil })
	disabled.router.Register("n", RegisterOptions{ConversationID: "conv", CallID: "call-1"})
	if disabled.router.Say("n", "hi") {
		t.Error("typing without a sender")
	}
}

func TestOverlappingSaysCannotPassTheCap(t *testing.T) {
	rig := newHandoffRig(t, func(o *HandoffRouterOptions) { o.MaxMessagesPerCall = 5 })
	rig.router.Register("n", RegisterOptions{ConversationID: "conv", CallID: "call-1"})
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if rig.router.Say("n", "x") {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if ok != 5 {
		t.Errorf("delivered %d, want 5", ok)
	}
}

func TestCaptureTimeoutAndPanicDoNotBlockTheSwitch(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	slow := newHandoffRig(t, func(o *HandoffRouterOptions) {
		o.CaptureTimeout = 0.05
		o.CaptureLeg = func(string, string) (bool, error) { <-release; return true, nil }
	})
	slow.router.Register("n", RegisterOptions{ConversationID: "conv", CallID: "call-1"})
	if slow.router.Redeem("n") == nil {
		t.Error("capture timeout blocked the handoff")
	}
	panicky := newHandoffRig(t, func(o *HandoffRouterOptions) {
		o.CaptureLeg = func(string, string) (bool, error) { panic("boom") }
	})
	panicky.router.Register("n", RegisterOptions{ConversationID: "conv", CallID: "call-1"})
	if panicky.router.Redeem("n") == nil {
		t.Error("a panicking capture blocked the handoff")
	}
}

func TestEscalate(t *testing.T) {
	rig := newHandoffRig(t, nil)
	h := rig.gw.MintHandle("conv-9")
	if !rig.router.Escalate(h) || rig.events[0] != "capture:conv-9:chat" {
		t.Errorf("escalate: %v", rig.events)
	}
	if rig.router.Escalate("forged.handle") {
		t.Error("forged handle escalated")
	}
}

func handoffPost(t *testing.T, h http.Handler, path, body, origin string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestHandoffRoutes(t *testing.T) {
	rig := newHandoffRig(t, nil)
	r := rig.router.Router()
	rig.router.Register("n", RegisterOptions{ConversationID: "conv", CallID: "call-1"})

	if code, _ := handoffPost(t, r, "/say", `{"nonce":"n","text":"hi"}`, "https://evil.example"); code != 403 {
		t.Errorf("unlisted origin: %d", code)
	}
	if code, out := handoffPost(t, r, "/say", `{"nonce":"n","text":"hi"}`, "https://shop.example.com"); code != 200 || out["ok"] != true {
		t.Errorf("say: %d %v", code, out)
	}
	if code, _ := handoffPost(t, r, "/say", `{"nonce":"zzz","text":"`+strings.Repeat("a", MaxMessageBytes+1)+`"}`, ""); code != 413 {
		t.Errorf("oversized say (size answer precedes the nonce lookup): %d", code)
	}
	if code, _ := handoffPost(t, r, "/say", `{"nonce":"unknown","text":"hi"}`, ""); code != 404 {
		t.Errorf("unknown nonce: %d", code)
	}
	if code, _ := handoffPost(t, r, "/handoff", `{"nonce":"`+strings.Repeat("a", MaxRequestBodyBytes)+`"}`, ""); code != 413 {
		t.Errorf("oversized body: %d", code)
	}
	code, out := handoffPost(t, r, "/handoff", `{"nonce":"n"}`, "")
	if handle, _ := out["handle"].(string); code != 200 || handle == "" {
		t.Fatalf("handoff: %d %v", code, out)
	}
	if code, out := handoffPost(t, r, "/handoff", `{"nonce":"n"}`, ""); code != 404 || out["error"] != "not found" {
		t.Errorf("spent nonce: %d %v", code, out)
	}
	if code, _ := handoffPost(t, r, "/escalate", `{}`, ""); code != 400 {
		t.Errorf("escalate without handle: %d", code)
	}
	if code, _ := handoffPost(t, r, "/escalate", `{"handle":"`+rig.gw.MintHandle("c")+`"}`, ""); code != 200 {
		t.Errorf("escalate: %d", code)
	}
}

func TestNewHandoffRouterRequiresGateway(t *testing.T) {
	if _, err := NewHandoffRouter(HandoffRouterOptions{}); err == nil {
		t.Error("nil gateway accepted")
	}
}
