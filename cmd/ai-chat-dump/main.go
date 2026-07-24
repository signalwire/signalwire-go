// Copyright (c) 2026 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Command ai-chat-dump is the Go port's AI-CHAT dump program for the cross-port
// wire-behavioral gate (porting-sdk/scripts/diff_port_ai_chat.py, on the
// ai-chat-client branch — a COORDINATED pass).
//
// The gate boots the in-process mock_ai_chat server, exports MOCK_AI_CHAT_URL +
// SIGNALWIRE_PROJECT_ID / SIGNALWIRE_API_TOKEN into this program's env, runs it,
// and asserts the JSON it prints (+ the wire requests the mock recorded) speak the
// AI Chat protocol per the vendored spec (ai-chat-specs/ai-chat.yaml).
//
// This mirrors porting-sdk/scripts/ai_chat_dump_reference.py EXACTLY: it drives the
// Go aichat.Client through the shared ai_chat_corpus and emits ONE JSON object to
// stdout (nothing else — no log noise), keyed by corpus step:
//
//	success steps (create/chat/end/delete/log/summarize):
//	    { wire_method, decoded: { <spec result fields> } }
//	summarize_failed (the summarize {error} one_of branch — must SURFACE, not swallow):
//	    { wire_method:"summarize", raised:true, error_type, message }
//	error steps (err_notfound/err_ratelimit/err_inprogress/err_auth/err_unmapped):
//	    { raised:true, error_code, error_type }
//
// The corpus (steps + SUMMARIZE_ERROR_ID + ERROR_STEPS + force_error_id) is data,
// identical for every language; it is mirrored inline here from ai_chat_corpus.py.
//
// Run from the repo root against a running mock:
//
//	MOCK_AI_CHAT_URL=http://127.0.0.1:PORT/api/ai/chat go run ./cmd/ai-chat-dump
//
// Nothing but the JSON object is written to stdout on success.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/signalwire/signalwire-go/v3/pkg/aichat"
)

// ── the shared corpus (mirror of porting-sdk/scripts/ai_chat_corpus.py) ──────

// summarizeErrorID is the sentinel conversation id that makes summarize return its
// {error} branch.
const summarizeErrorID = "__summarize_error"

// errorStep pairs a corpus step id with the JSON-RPC code the raised error MUST
// carry. Ordered to match the reference's iteration order.
type errorStep struct {
	step string
	code int
}

var errorSteps = []errorStep{
	{"err_notfound", -32001},   // ConversationNotFound
	{"err_ratelimit", -32005},  // RateLimit
	{"err_inprogress", -32007}, // ChatInProgress
	{"err_auth", -32009},       // Authentication
	{"err_unmapped", -32602},   // base AIChatError (unmapped code)
}

// forceErrorID is the sentinel conversation id that makes the mock return code.
func forceErrorID(code int) string {
	return fmt.Sprintf("__err_%d", code)
}

func run(url string) (map[string]any, error) {
	ctx := context.Background()
	out := map[string]any{}
	client, err := aichat.NewClient(aichat.WithURL(url))
	if err != nil {
		return nil, err
	}

	// ── success steps ──────────────────────────────────────────────────
	info, err := client.CreateConversation(ctx, "conv-1", aichat.CreateOptions{
		ConfigURL: "http://cfg",
		Timeout:   30,
		Reinit:    true,
	})
	if err != nil {
		return nil, fmt.Errorf("create: %w", err)
	}
	out["create"] = map[string]any{
		"wire_method": "create_conversation",
		"decoded":     map[string]any{"status": info.Status, "id": info.ID, "initial_message": info.InitialMessage},
	}

	reply, err := client.Chat(ctx, "conv-1", "hello", aichat.ChatOptions{Timeout: 30, Reinit: true})
	if err != nil {
		return nil, fmt.Errorf("chat: %w", err)
	}
	out["chat"] = map[string]any{
		"wire_method": "chat",
		"decoded":     map[string]any{"response": reply.Text, "user_event": reply.UserEvent},
	}

	// end/delete return bool idiomatically; the wire result also carries the
	// conversation id (the caller's own input, echoed). Report both the derived
	// status and the id operated on — mirroring the reference dump.
	ended, err := client.End(ctx, "conv-1")
	if err != nil {
		return nil, fmt.Errorf("end: %w", err)
	}
	out["end"] = map[string]any{
		"wire_method": "end_conversation",
		"decoded":     map[string]any{"status": statusOr(ended, "ended"), "id": "conv-1"},
	}

	deleted, err := client.Delete(ctx, "conv-1")
	if err != nil {
		return nil, fmt.Errorf("delete: %w", err)
	}
	out["delete"] = map[string]any{
		"wire_method": "delete",
		"decoded":     map[string]any{"status": statusOr(deleted, "deleted"), "id": "conv-1"},
	}

	log, err := client.Log(ctx, "conv-1")
	if err != nil {
		return nil, fmt.Errorf("log: %w", err)
	}
	out["log"] = map[string]any{
		"wire_method": "chat_log",
		"decoded":     map[string]any{"chat_log": log.Messages, "call_timeline": log.CallTimeline},
	}

	summary, err := client.Summarize(ctx, "conv-1")
	if err != nil {
		return nil, fmt.Errorf("summarize: %w", err)
	}
	out["summarize"] = map[string]any{"wire_method": "summarize", "decoded": map[string]any{"summary": summary}}

	// ── summarize one_of {error} branch: must SURFACE, not swallow ───────
	swallowed, err := client.Summarize(ctx, summarizeErrorID)
	var summaryErr *aichat.SummaryError
	switch {
	case err == nil:
		out["summarize_failed"] = map[string]any{
			"wire_method": "summarize",
			"raised":      false,
			"decoded":     map[string]any{"summary": swallowed},
		}
	case errors.As(err, &summaryErr):
		out["summarize_failed"] = map[string]any{
			"wire_method": "summarize",
			"raised":      true,
			"error_type":  typeName(err),
			"message":     summaryErr.Message,
		}
	default:
		return nil, fmt.Errorf("summarize_failed: unexpected error: %w", err)
	}

	// ── error-code steps (JSON-RPC error object) ─────────────────────────
	for _, es := range errorSteps {
		_, err := client.Chat(ctx, forceErrorID(es.code), "x")
		if err == nil {
			out[es.step] = map[string]any{"raised": false}
			continue
		}
		var aiErr *aichat.AIChatError
		if !errors.As(err, &aiErr) {
			return nil, fmt.Errorf("%s: unexpected error type: %w", es.step, err)
		}
		out[es.step] = map[string]any{
			"raised":     true,
			"error_code": aiErr.Code,
			"error_type": typeName(err),
		}
	}

	return out, nil
}

// statusOr returns want when ok, else "?" — mirroring the reference's derived
// status reporting for the boolean end/delete results.
func statusOr(ok bool, want string) string {
	if ok {
		return want
	}
	return "?"
}

// typeName reports the concrete typed-error class name (e.g.
// "ConversationNotFoundError"), the Go analogue of the reference's
// type(e).__name__. It unwraps a pointer to reach the struct name.
func typeName(err error) string {
	t := reflect.TypeOf(err)
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil {
		return "error"
	}
	return t.Name()
}

func main() {
	url := os.Getenv("MOCK_AI_CHAT_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr, "MOCK_AI_CHAT_URL not set")
		os.Exit(2)
	}
	out, err := run(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ai-chat-dump: %v\n", err)
		os.Exit(1)
	}
	enc, err := json.Marshal(out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ai-chat-dump: encode: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(enc))
}
