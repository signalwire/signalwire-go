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
// secure=false gets NO `__token` — it falls back to the shared, unauthenticated
// SWAIG defaults.web_hook_url.
//
// For each corpus fixture this program defines the tool on a fresh AgentBase,
// renders the SWML with the FIXED corpus call_id, and reduces to the
// deterministic pair the differ compares against the python golden:
//
//	secure_default_true  — the SDK-recorded secure flag for the tool.
//	wire_reflects_secure — a `__token` is present on the rendered webhook IFF the
//	                       tool is secure (secure -> token; insecure -> none).
//
// The token VALUE is an HMAC over (call_id, tool, expiry, nonce) and varies per
// run, so it is NOT compared — only its PRESENCE folds into the boolean. That
// keeps the golden deterministic while the behavior producing it stays real and
// unfakeable: a port cannot report a token for its secure default without
// actually minting one onto the wire.
//
// Protocol: stdout = ONE JSON object mapping fixture id -> classification. Only
// stdout carries JSON; all diagnostics go to stderr.
//
// Run from the signalwire-go repo root:
//
//	go run ./cmd/secure-default-dump
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/signalwire/signalwire-go/v3/pkg/agent"
	"github.com/signalwire/signalwire-go/v3/pkg/logging"
	"github.com/signalwire/signalwire-go/v3/pkg/swaig"
)

// callID mirrors secure_default_corpus.CALL_ID exactly. A FIXED call_id is what
// makes a secure tool deterministically mint a token.
const callID = "call-secure-default-fixture"

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
	// A1 (b) — an explicit secure=false must be INSECURE: NO __token. This pins
	// the other direction, so a port that blindly tokenizes every tool reds here.
	{
		id: "define_tool_explicit_insecure", toolName: "sd_explicit_insecure",
		expectSecure: false, explicitInsecure: true,
	},
}

// classification is the per-fixture artifact the differ compares.
type classification struct {
	SecureDefaultTrue  bool `json:"secure_default_true"`
	WireReflectsSecure bool `json:"wire_reflects_secure"`
}

func main() { os.Exit(run()) }

func run() int {
	// Keep stdout PURE JSON — the differ does json.loads(proc.stdout).
	logging.SetGlobalLevel(logging.LevelOff)

	out := map[string]classification{}
	for _, f := range corpus {
		c, err := classify(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "secure-default-dump: fixture %s: %v\n", f.id, err)
			return 1
		}
		out[f.id] = c
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "secure-default-dump: encode: %v\n", err)
		return 1
	}
	return 0
}

// classify builds a fresh agent, defines the fixture's single tool, renders the
// SWML with the fixed corpus call_id, and reduces to the fixture artifact.
func classify(f fixture) (classification, error) {
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
		return classification{}, fmt.Errorf("expected 1 registered tool, got %d", len(tools))
	}
	isSecure := tools[0].IsSecure()

	// Render WITH the fixed call_id so a secure tool mints its __token.
	doc := a.RenderSWMLForCall(nil, nil, callID)
	entry, ok := swaigFunctionByName(doc, f.toolName)
	if !ok {
		return classification{}, fmt.Errorf("tool %q absent from the rendered SWAIG functions", f.toolName)
	}
	tokenPresent := webhookHasToken(entry)

	return classification{
		SecureDefaultTrue: isSecure,
		// The wire "reflects" secure when a token is present IFF the tool is
		// secure: secure -> token present; insecure -> token correctly absent.
		WireReflectsSecure: tokenPresent == f.expectSecure,
	}, nil
}

// webhookHasToken reports whether a rendered SWAIG function entry's webhook
// carries the reserved `__token` query parameter — the wire reflection of
// secure. Mirrors the oracle's _webhook_has_token.
func webhookHasToken(entry map[string]any) bool {
	url, _ := entry["web_hook_url"].(string)
	return strings.Contains(url, "__token=")
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
