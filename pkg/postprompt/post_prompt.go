// Copyright (c) 2026 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Package postprompt normalizes a post-prompt body from either engine.
//
// One conversation can run over voice and over text chat, and both engines
// produce "the post-prompt" — in different shapes. This package absorbs the
// divergence so an application sees one artifact whichever engine finished:
//
//	field              voice                      chat
//	app_name           "swml app"                 "ai_chat"
//	conversation_id    absent                     present at top level
//	full log           raw_call_log               raw_messages
//	summary arrives as summarize_conversation     a bare role:assistant turn
//	                   tool call                  inside call_log
//	post_prompt_data   parsed object              {"raw": "```json ...```"}
//
// conversation_type is a reliable top-level discriminator on both. A third
// post_prompt_data shape, {"parsed": [{...}], "raw": "..."}, is also handled.
//
// What a summary should contain is the application's decision (whatever its
// post-prompt asked the model to produce), so parsing here is schema-agnostic
// and returns the object as found. Nothing in this package fails: the
// conversation that produced the input is already over, so a malformed summary
// degrades rather than failing the request that delivered it.
package postprompt

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// dialogueRoles are the roles that are actual dialogue. Everything else in a
// call log is machinery: system (the prompt), system-log (lifecycle and step
// tracing), tool (function output), assistant-manual (filler speech).
var dialogueRoles = []string{"user", "assistant"}

var (
	fenceOpen  = regexp.MustCompile("^```[a-zA-Z]*\\s*")
	fenceClose = regexp.MustCompile("\\s*```$")
)

// NormalizedPostPrompt is one finished conversation leg, in a shape that does
// not vary by engine.
type NormalizedPostPrompt struct {
	// Medium is conversation_type as reported, e.g. "voice" or "chat"; "" when
	// the engine did not say.
	Medium string
	// ConversationID is present on chat, absent (nil) on voice. Callers that
	// need a stable key should fall back to their own (global_data, call_id).
	ConversationID *string
	// Summary is the parsed post_prompt_data — whatever keys the application's
	// post-prompt asked for. Empty when there was none or it could not be parsed;
	// a model that answered in prose yields {"summary": "<the prose>"}.
	Summary map[string]any
	// Dialogue is the user/assistant turns only, with tool calls and the chat
	// engine's summary echo removed.
	Dialogue []map[string]string
	// CallID is the platform call id, when present.
	CallID *string
	// Raw is the complete request body, untouched.
	Raw map[string]any
}

// StripJSONFence unwraps ```json ... ``` fencing. The chat engine hands the
// model's answer back verbatim, fence and all, where the voice engine parses it
// first.
func StripJSONFence(text string) string {
	stripped := strings.TrimSpace(text)
	if strings.HasPrefix(stripped, "```") {
		stripped = fenceOpen.ReplaceAllString(stripped, "")
		stripped = fenceClose.ReplaceAllString(stripped, "")
	}
	return strings.TrimSpace(stripped)
}

// unwrapParsed pulls the object out of a {"parsed": [...]} wrapper, if present.
func unwrapParsed(data map[string]any) map[string]any {
	switch p := data["parsed"].(type) {
	case map[string]any:
		if len(p) > 0 {
			return p
		}
	case []any:
		for _, item := range p {
			if m, ok := item.(map[string]any); ok && len(m) > 0 {
				return m
			}
		}
	}
	return nil
}

// ParsePostPromptData returns post_prompt_data as a plain map, whichever shape
// it arrived in, or an empty map when there is nothing usable.
func ParsePostPromptData(data any) map[string]any {
	m, ok := data.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	if unwrapped := unwrapParsed(m); unwrapped != nil {
		return unwrapped
	}
	// Flat shape: real keys already present (anything but raw/parsed).
	flat := map[string]any{}
	for k, v := range m {
		if k != "raw" && k != "parsed" {
			flat[k] = v
		}
	}
	if len(flat) > 0 {
		return flat
	}
	raw, ok := m["raw"].(string)
	if !ok || strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	unfenced := StripJSONFence(raw)
	var loaded any
	if err := json.Unmarshal([]byte(unfenced), &loaded); err != nil {
		// Prose instead of JSON. Still a summary.
		return map[string]any{"summary": unfenced}
	}
	if obj, ok := loaded.(map[string]any); ok {
		return obj
	}
	if str, ok := loaded.(string); ok {
		return map[string]any{"summary": str}
	}
	return map[string]any{"summary": fmt.Sprint(loaded)}
}

// DialogueOptions carries the optional parameters of [DialogueTurns].
type DialogueOptions struct {
	// Roles are the roles to keep (nil = user and assistant).
	Roles []string `sw:"optional" kind:"keyword"`
	// DropEcho is exact content to treat as the chat engine's summary echo and
	// drop: the chat engine appends its own post-prompt output to call_log as a
	// bare role:assistant turn, identifiable only by being byte-identical to
	// post_prompt_data.raw.
	DropEcho *string `kind:"keyword"`
}

// DialogueTurns extracts the real dialogue from a call log (call_log /
// raw_call_log / raw_messages): it drops non-dialogue roles, entries carrying
// tool_calls, and empty content, returning [{"role": ..., "content": ...}, ...]
// in order.
func DialogueTurns(callLog any, opts DialogueOptions) []map[string]string {
	entries, ok := callLog.([]any)
	if !ok {
		if typed, ok2 := callLog.([]map[string]any); ok2 {
			for _, e := range typed {
				entries = append(entries, e)
			}
		} else {
			return []map[string]string{}
		}
	}
	roles := opts.Roles
	if roles == nil {
		roles = dialogueRoles
	}
	echo := ""
	if opts.DropEcho != nil {
		echo = strings.TrimSpace(*opts.DropEcho)
	}
	out := []map[string]string{}
	for _, raw := range entries {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		role, _ := entry["role"].(string)
		if !contains(roles, role) {
			continue
		}
		if tc, has := entry["tool_calls"]; has && truthy(tc) {
			continue
		}
		content, ok := entry["content"].(string)
		if !ok || strings.TrimSpace(content) == "" {
			continue
		}
		if echo != "" && strings.TrimSpace(content) == echo {
			continue
		}
		out = append(out, map[string]string{"role": role, "content": content})
	}
	return out
}

// NormalizePostPrompt normalizes a post-prompt body from either engine. A body
// it cannot make sense of yields a NormalizedPostPrompt with empty fields.
//
//	leg := postprompt.NormalizePostPrompt(body)
//	if len(leg.Dialogue) > 0 { store(leg.ConversationID, leg.Medium, leg.Summary, leg.Dialogue) }
func NormalizePostPrompt(body any) NormalizedPostPrompt {
	b, ok := body.(map[string]any)
	if !ok {
		return NormalizedPostPrompt{Summary: map[string]any{}, Dialogue: []map[string]string{}, Raw: map[string]any{}}
	}
	summary := ParsePostPromptData(b["post_prompt_data"])
	// The echo is compared against the RAW string the engine returned, not the
	// parsed summary — the assistant turn carries the fence too.
	var dropEcho *string
	if ppd, ok := b["post_prompt_data"].(map[string]any); ok {
		if raw, ok := ppd["raw"].(string); ok && raw != "" {
			dropEcho = &raw
		}
	}
	var log any = []any{}
	for _, key := range []string{"call_log", "raw_call_log", "raw_messages"} {
		if v, has := b[key]; has && truthy(v) {
			log = v
			break
		}
	}
	medium := ""
	if v, has := b["conversation_type"]; has && truthy(v) {
		if s, ok := v.(string); ok {
			medium = s
		} else {
			medium = fmt.Sprint(v)
		}
	}
	return NormalizedPostPrompt{
		Medium:         medium,
		ConversationID: optString(b["conversation_id"]),
		Summary:        summary,
		Dialogue:       DialogueTurns(log, DialogueOptions{DropEcho: dropEcho}),
		CallID:         optString(b["call_id"]),
		Raw:            b,
	}
}

func optString(v any) *string {
	if s, ok := v.(string); ok && s != "" {
		return &s
	}
	return nil
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// truthy mirrors the reference's truthiness test on a decoded JSON value.
func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0
	case []any:
		return len(t) > 0
	case []map[string]any:
		return len(t) > 0
	case map[string]any:
		return len(t) > 0
	}
	return true
}
