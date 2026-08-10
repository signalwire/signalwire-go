// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Command secure-default-dump is the Go port's SECURE-DEFAULT (A1) dump program
// for the cross-port behavioral differ
// (porting-sdk/scripts/diff_port_secure_default.py, corpus
// porting-sdk/scripts/secure_default_corpus.py).
//
// The A1 contract: DefineTool defaults to SECURE fleet-wide. A tool defined
// WITHOUT an explicit secure opt-out MUST require SWAIG token validation, and the
// WIRE manifestation of that is the per-tool `__token` query parameter the
// rendered SWAIG webhook carries when the SWML is rendered with a call_id
// (reference agent_base.py:1040/1096-1100). A tool defined with an explicit
// secure=false gets NO `__token` and, just as load-bearing, NO per-tool
// `web_hook_url` KEY AT ALL — it falls back to the shared
// SWAIG.defaults.web_hook_url. Handing an insecure tool its own URL would put an
// unauthenticated, function-specific callback on the wire.
//
// For each corpus fixture this program defines the tool on a fresh AgentBase,
// renders the SWML with the FIXED corpus call_id, and emits the RENDERED
// functions[] entry verbatim — with nondeterministic token VALUES replaced by
// the corpus placeholder — under:
//
//	secure_default_true — the SDK-recorded secure flag for the tool.
//	rendered            — the functions[] entry, keys preserved exactly.
//
// The port does NOT classify. Every verdict (has_own_webhook, token_carrier) is
// derived by the differ FROM THE KEYS of this payload, so a port cannot report a
// topology it did not actually emit. That is the 2026-07-27 redesign: the
// previous “wire_reflects_secure“ boolean was the port's own conclusion about
// its own render, which made the gate structurally blind to a token in the wrong
// key and to an insecure tool carrying its own tokenless URL.
//
// The token VALUE is an HMAC over (call_id, tool, expiry, nonce) and varies per
// run, so it is redacted to "<TOKEN>"; every KEY and key path survives intact.
//
// Protocol: stdout = ONE JSON object mapping fixture id -> {secure_default_true,
// rendered}. Only stdout carries JSON; all diagnostics go to stderr.
//
// Run from the signalwire-go repo root:
//
//	go run ./cmd/secure-default-dump
package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/signalwire/signalwire-go/v3/pkg/agent"
	"github.com/signalwire/signalwire-go/v3/pkg/logging"
	"github.com/signalwire/signalwire-go/v3/pkg/swaig"
)

// callID mirrors secure_default_corpus.CALL_ID exactly. A FIXED call_id is what
// makes a secure tool deterministically mint a token.
const callID = "call-secure-default-fixture"

// tokenPlaceholder mirrors secure_default_corpus.TOKEN_PLACEHOLDER.
const tokenPlaceholder = "<TOKEN>"

// fixture mirrors one secure_default_corpus.CORPUS entry.
type fixture struct {
	id       string
	toolName string
	// expectSecure is the SDK-recorded secure flag this fixture asserts.
	expectSecure bool
	// explicitInsecure arms the secure=false opt-out (the second corpus case).
	explicitInsecure bool
}

var corpus = []fixture{
	// A1 (a) — NO explicit secure opt-out must default to SECURE, so the rendered
	// webhook carries a __token. This is the case that reds a port whose
	// DefineTool defaults insecure (which go did before this gate landed: a plain
	// `bool` field whose zero value is false).
	{id: "define_tool_default_is_secure", toolName: "sd_default_secure", expectSecure: true},
	// A1 (b) — an explicit secure=false must be INSECURE: no __token, and no
	// per-tool web_hook_url key whatsoever.
	{
		id: "define_tool_explicit_insecure", toolName: "sd_explicit_insecure",
		expectSecure: false, explicitInsecure: true,
	},
}

// dumpEntry is the per-fixture artifact: the SDK-recorded flag plus the raw
// rendered payload. No classification — the differ owns that.
type dumpEntry struct {
	SecureDefaultTrue bool           `json:"secure_default_true"`
	Rendered          map[string]any `json:"rendered"`
}

func main() { os.Exit(run()) }

func run() int {
	// Keep stdout PURE JSON — the differ does json.loads(proc.stdout).
	logging.SetGlobalLevel(logging.LevelOff)

	out := map[string]dumpEntry{}
	for _, f := range corpus {
		e, err := render(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "secure-default-dump: fixture %s: %v\n", f.id, err)
			return 1
		}
		out[f.id] = e
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "secure-default-dump: encode: %v\n", err)
		return 1
	}
	return 0
}

// render builds a fresh agent, defines the fixture's single tool, renders the
// SWML with the fixed corpus call_id, and returns the redacted functions[] entry.
func render(f fixture) (dumpEntry, error) {
	a := agent.NewAgentBase(
		agent.WithName("secure-default-fixture"),
		agent.WithRoute("/sd"),
		agent.WithBasicAuth("u", "p"),
	)
	a.SetPromptText("secure default fixture")

	def := agent.ToolDefinition{
		Name:        f.toolName,
		Description: "secure-default fixture tool",
		Parameters:  map[string]any{},
		Handler: func(map[string]any, map[string]any) *swaig.FunctionResult {
			return swaig.NewFunctionResult("ok")
		},
	}
	// ONLY the explicit-insecure fixture sets the field. The default fixture
	// leaves it unset — that is the whole point of the A1 case.
	if f.explicitInsecure {
		insecure := false
		def.Secure = &insecure
	}
	a.DefineTool(def)

	// Read back the SDK-recorded secure flag for this tool.
	tools := a.DefineTools()
	if len(tools) != 1 {
		return dumpEntry{}, fmt.Errorf("expected 1 registered tool, got %d", len(tools))
	}
	isSecure := tools[0].IsSecure()

	// Render WITH the fixed call_id so a secure tool mints its __token.
	doc := a.RenderSWMLForCall(nil, nil, callID)
	entry, ok := swaigFunctionByName(doc, f.toolName)
	if !ok {
		return dumpEntry{}, fmt.Errorf("tool %q absent from the rendered SWAIG functions", f.toolName)
	}

	return dumpEntry{SecureDefaultTrue: isSecure, Rendered: redact(entry)}, nil
}

// isTokenish reports whether a key names a security token by the differ's rule:
// a case-insensitive "token" suffix.
func isTokenish(key string) bool {
	return strings.HasSuffix(strings.ToLower(key), "token")
}

// redact replaces every nondeterministic token VALUE (an HMAC) with the corpus
// placeholder while preserving every KEY and key path exactly — both a
// token-suffixed field and a token-suffixed query parameter on a URL value.
// Mirrors diff_port_secure_default.redact_entry so the differ's re-application
// is a no-op (idempotent).
func redact(entry map[string]any) map[string]any {
	out := make(map[string]any, len(entry))
	for k, v := range entry {
		s, isStr := v.(string)
		switch {
		case isStr && isTokenish(k):
			out[k] = tokenPlaceholder
		case isStr && (strings.Contains(s, "://") || strings.HasPrefix(s, "/")):
			out[k] = redactURLTokens(s)
		default:
			out[k] = v
		}
	}
	return out
}

// redactURLTokens replaces the VALUE of every token-suffixed query parameter in
// a URL with the placeholder, leaving the URL untouched when it carries none.
func redactURLTokens(raw string) string {
	q := strings.Index(raw, "?")
	if q < 0 {
		return raw
	}
	pairs := strings.Split(raw[q+1:], "&")
	anyToken := false
	for _, pair := range pairs {
		key := pair
		if eq := strings.Index(pair, "="); eq >= 0 {
			key = pair[:eq]
		}
		if decoded, err := url.QueryUnescape(key); err == nil {
			key = decoded
		}
		if isTokenish(key) {
			anyToken = true
			break
		}
	}
	if !anyToken {
		return raw
	}

	rebuilt := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		key, eq := pair, strings.Index(pair, "=")
		if eq >= 0 {
			key = pair[:eq]
		}
		decoded := key
		if d, err := url.QueryUnescape(key); err == nil {
			decoded = d
		}
		if eq >= 0 && isTokenish(decoded) {
			rebuilt = append(rebuilt, key+"="+tokenPlaceholder)
			continue
		}
		rebuilt = append(rebuilt, pair)
	}
	return raw[:q+1] + strings.Join(rebuilt, "&")
}

// swaigFunctionByName walks sections.main -> the `ai` verb -> SWAIG.functions and
// returns the entry for one tool.
func swaigFunctionByName(doc map[string]any, name string) (map[string]any, bool) {
	sections, ok := doc["sections"].(map[string]any)
	if !ok {
		return nil, false
	}
	main, ok := sections["main"].([]any)
	if !ok {
		return nil, false
	}
	for _, raw := range main {
		verb, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		ai, ok := verb["ai"].(map[string]any)
		if !ok {
			continue
		}
		sw, ok := ai["SWAIG"].(map[string]any)
		if !ok {
			continue
		}
		fns, ok := sw["functions"].([]map[string]any)
		if !ok {
			continue
		}
		for _, fn := range fns {
			if fnName, _ := fn["function"].(string); fnName == name {
				return fn, true
			}
		}
	}
	return nil, false
}
