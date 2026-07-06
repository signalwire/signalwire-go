// Command swml-dump is the Go port's SWML dump program for the cross-port SWML
// differ (porting-sdk/scripts/diff_port_swml.py).
//
// For each swml_corpus case it builds an AgentBase, applies the setter chain,
// renders the SWML document, and extracts the observed dotted path (e.g.
// "ai.prompt.pom") — emitting ONE JSON object mapping
//
//	case-id -> extracted-fragment
//
// to stdout. The differ canonicalizes both sides and byte-compares against the
// python oracle. Only stdout carries JSON.
//
// Run from the signalwire-go repo root:
//
//	go run ./cmd/swml-dump
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/pkg/agent"
)

// newAgent constructs a demo AgentBase (name "demo", route "/demo") with POM
// enabled so prompt_add_section renders into ai.prompt.pom, matching the oracle.
func newAgent() *agent.AgentBase {
	return agent.NewAgentBase(
		agent.WithName("demo"),
		agent.WithRoute("/demo"),
		agent.WithUsePom(true),
	)
}

// extract walks a dotted path into a rendered SWML doc. "ai.prompt" means: find
// the ai verb in sections.main, then index into it — the Go mirror of
// diff_port_swml._extract.
func extract(doc map[string]any, path string) any {
	var ai any
	if sections, ok := doc["sections"].(map[string]any); ok {
		if main, ok := sections["main"].([]any); ok {
			for _, sec := range main {
				if m, ok := sec.(map[string]any); ok {
					if v, ok := m["ai"]; ok {
						ai = v
						break
					}
				}
			}
		}
	}
	var node any
	if ai != nil {
		node = map[string]any{"ai": ai}
	} else {
		node = doc
	}
	for _, part := range splitDots(path) {
		m, ok := node.(map[string]any)
		if !ok {
			return nil
		}
		node = m[part]
	}
	return node
}

// pick reduces a map fragment to the listed keys (mirrors the oracle's `pick`).
func pick(frag any, keys []string) any {
	m, ok := frag.(map[string]any)
	if !ok {
		return frag
	}
	out := map[string]any{}
	for _, k := range keys {
		out[k] = m[k]
	}
	return out
}

func splitDots(s string) []string {
	var out []string
	start := 0
	for i := range len(s) {
		if s[i] == '.' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func render(a *agent.AgentBase) map[string]any {
	return a.RenderSWML(nil, nil)
}

func main() {
	out := map[string]any{}

	// swml_set_prompt_llm_params: two set_prompt_llm_params calls MERGE.
	{
		a := newAgent()
		a.SetPromptLlmParams(map[string]any{"temperature": 0.5})
		a.SetPromptLlmParams(map[string]any{"top_p": 0.9})
		out["swml_set_prompt_llm_params"] = pick(extract(render(a), "ai.prompt"), []string{"temperature", "top_p"})
	}

	// swml_set_post_prompt_llm_params: establish a post-prompt, then merge params.
	{
		a := newAgent()
		a.SetPostPrompt("Summarize the call.")
		a.SetPostPromptLlmParams(map[string]any{"temperature": 0.3})
		a.SetPostPromptLlmParams(map[string]any{"top_p": 0.8})
		out["swml_set_post_prompt_llm_params"] = pick(extract(render(a), "ai.post_prompt"), []string{"temperature", "top_p"})
	}

	// swml_add_language: engine/model/voice carried into ai.languages.
	{
		a := newAgent()
		a.AddLanguage(map[string]any{
			"name": "English", "code": "en-US", "voice": "rime.spore",
			"engine": "rime", "model": "mistv2",
		})
		out["swml_add_language"] = extract(render(a), "ai.languages")
	}

	// swml_add_pattern_hint: structured hint into ai.hints.
	{
		a := newAgent()
		a.AddPatternHint("SignalWire", "signal wire", "SignalWire", true)
		out["swml_add_pattern_hint"] = extract(render(a), "ai.hints")
	}

	// swml_add_hint: a plain string hint.
	{
		a := newAgent()
		a.AddHint("SignalWire")
		out["swml_add_hint"] = extract(render(a), "ai.hints")
	}

	// swml_prompt_add_section: POM sections render into ai.prompt.pom.
	{
		a := newAgent()
		a.PromptAddSection("Role", "You are a helpful assistant.", nil)
		a.PromptAddSection("Rules", "", []string{"Be concise", "Be accurate"})
		out["swml_prompt_add_section"] = extract(render(a), "ai.prompt.pom")
	}

	// swml_add_pronunciation: renders into ai.pronounce.
	{
		a := newAgent()
		a.AddPronunciation("SW", "SignalWire", true)
		out["swml_add_pronunciation"] = extract(render(a), "ai.pronounce")
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "swml-dump: encode failed: %v\n", err)
		os.Exit(1)
	}
}
