// Command state-dump is the Go port's STATE dump program for the cross-port
// state differ (porting-sdk/scripts/diff_port_state.py).
//
// For each state_corpus case it builds the target object, applies the mutation
// chain via the Go SDK's native API, reads the observable state through the
// public accessor / rendered representation, and prints ONE JSON object mapping
//
//	case-id -> observed-state
//
// to stdout. The differ canonicalizes both sides and byte-compares against the
// python oracle. Only stdout carries JSON.
//
// Run from the signalwire-go repo root:
//
//	go run ./cmd/state-dump
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/signalwire/signalwire-go/v3/pkg/agent"
	"github.com/signalwire/signalwire-go/v3/pkg/contexts"
	"github.com/signalwire/signalwire-go/v3/pkg/prefabs"
	"github.com/signalwire/signalwire-go/v3/pkg/server"
	"github.com/signalwire/signalwire-go/v3/pkg/skills"
	"github.com/signalwire/signalwire-go/v3/pkg/swml"
)

// greetVerbHandler is a minimal custom VerbHandler for the "greet" verb — the Go
// analog of the corpus's throwaway __register_verb__ handler.
type greetVerbHandler struct{ name string }

func (h greetVerbHandler) GetVerbName() string { return h.name }
func (h greetVerbHandler) ValidateConfig(map[string]any) (bool, []string) {
	return true, nil
}
func (h greetVerbHandler) BuildConfig(params map[string]any) (map[string]any, error) {
	return params, nil
}

func demoAgent() *agent.AgentBase {
	return agent.NewAgentBase(agent.WithName("demo"), agent.WithRoute("/demo"))
}

func main() {
	out := map[string]any{}

	// ---- global_data: set MERGES into the accumulated global data ----
	{
		a := demoAgent()
		a.SetGlobalData(map[string]any{"company": "SignalWire", "tier": "gold"})
		out["state_set_global_data"] = a.GetGlobalData()
	}
	{
		a := demoAgent()
		a.UpdateGlobalData(map[string]any{"k1": "v1"})
		a.UpdateGlobalData(map[string]any{"k2": "v2"})
		out["state_update_global_data"] = a.GetGlobalData()
	}
	{
		// MERGE semantics: overlapping key wins, sibling survives.
		a := demoAgent()
		a.SetGlobalData(map[string]any{"a": 1, "b": 2})
		a.SetGlobalData(map[string]any{"b": 99, "c": 3})
		out["state_global_data_merge"] = a.GetGlobalData()
	}

	// ---- sip-username registration on AgentBase (lowercased set) ----
	{
		a := demoAgent()
		a.RegisterSIPUsername("Bob")
		a.RegisterSIPUsername("alice")
		out["state_register_sip_username"] = a.GetSIPUsernames()
	}
	{
		// dedup + case-fold: "Bob","BOB","bob" collapse to one.
		a := demoAgent()
		a.RegisterSIPUsername("Bob")
		a.RegisterSIPUsername("BOB")
		a.RegisterSIPUsername("bob")
		out["state_register_sip_username_dedup"] = a.GetSIPUsernames()
	}

	// ---- AgentServer sip-username mapping (username -> route) + lookup ----
	{
		s := server.NewAgentServer()
		s.SetupSIPRouting("/sip", false)
		s.RegisterSIPUsername("Bob", "/agent")
		s.RegisterSIPUsername("sales", "/sales")
		lookupBob, _ := s.LookupSIPRoute("bob")
		lookupBOB, _ := s.LookupSIPRoute("BOB")
		var lookupMissing any
		if r, ok := s.LookupSIPRoute("nope"); ok {
			lookupMissing = r
		} else {
			lookupMissing = nil
		}
		out["server_sip_username_mapping"] = map[string]any{
			"mapping":        s.SIPUsernameMapping(),
			"lookup_bob":     lookupBob,
			"lookup_BOB":     lookupBOB,
			"lookup_missing": lookupMissing,
		}
	}
	{
		// unregister removes the agent route from the registry.
		s := server.NewAgentServer()
		s.Register(agent.NewAgentBase(agent.WithName("agent"), agent.WithRoute("/agent")), "/agent")
		s.Register(agent.NewAgentBase(agent.WithName("other"), agent.WithRoute("/other")), "/other")
		s.Unregister("/agent")
		agents := s.GetAgents()
		routes := make([]string, 0, len(agents))
		for _, e := range agents {
			routes = append(routes, e.Route)
		}
		out["server_unregister"] = routes
	}

	// ---- routing-callback registration on SWMLService (path-normalized) ----
	{
		svc := swml.NewService(swml.WithName("svc"), swml.WithRoute("/svc"))
		noop := func(map[string]any, map[string]any) *string { return nil }
		svc.RegisterRoutingCallback("/sip/", noop)
		svc.RegisterRoutingCallback("voice", noop)
		out["state_register_routing_callback"] = svc.RoutingCallbackPaths()
	}

	// ---- verb-handler registration (VerbHandlerRegistry: ai preloaded) ----
	{
		// A fresh Service preloads only the "ai" handler (mirrors Python's
		// VerbHandlerRegistry.__init__); register a custom "greet".
		svc := swml.NewService(swml.WithName("svc"), swml.WithRoute("/svc"))
		svc.RegisterVerbHandler(greetVerbHandler{name: "greet"})
		out["state_register_verb_handler"] = map[string]any{
			"verbs":       svc.VerbHandlerNames(),
			"has_greet":   svc.HasVerbHandler("greet"),
			"has_ai":      svc.HasVerbHandler("ai"),
			"has_missing": svc.HasVerbHandler("nope"),
		}
	}

	// ---- skill registration (SkillRegistry: name -> factory, idempotent) ----
	{
		reg := skills.NewSkillRegistry()
		noopFactory := func(map[string]any) skills.SkillBase { return nil }
		reg.RegisterSkill("custom_alpha", noopFactory)
		reg.RegisterSkill("custom_beta", noopFactory)
		reg.RegisterSkill("custom_alpha", noopFactory) // idempotent
		out["state_register_skill"] = reg.RegisteredNames()
	}

	// ---- InfoGatherer.submit_answer: records answer + advances index ----
	{
		ig := prefabs.NewInfoGathererAgent(prefabs.InfoGathererOptions{Name: "demo", Route: "/demo"})
		out["infogatherer_submit_answer_first"] = submitAnswerDelta(ig,
			map[string]any{"answer": "Alice"},
			map[string]any{"global_data": map[string]any{
				"questions": []any{
					map[string]any{"key_name": "name", "question_text": "What is your name?"},
					map[string]any{"key_name": "email", "question_text": "What is your email?"},
				},
				"question_index": float64(0),
				"answers":        []any{},
			}})
	}
	{
		ig := prefabs.NewInfoGathererAgent(prefabs.InfoGathererOptions{Name: "demo", Route: "/demo"})
		out["infogatherer_submit_answer_last"] = submitAnswerDelta(ig,
			map[string]any{"answer": "a@b.com"},
			map[string]any{"global_data": map[string]any{
				"questions": []any{
					map[string]any{"key_name": "name", "question_text": "What is your name?"},
					map[string]any{"key_name": "email", "question_text": "What is your email?"},
				},
				"question_index": float64(1),
				"answers":        []any{map[string]any{"key_name": "name", "answer": "Alice"}},
			}})
	}

	// ---- contexts/steps navigation (valid_steps rendered per step) ----
	{
		a := demoAgent()
		cb := a.DefineContexts()
		ctx := cb.AddContext("default")
		ctx.AddStep("greet").SetText("Greet the caller.").SetValidSteps([]string{"collect"})
		ctx.AddStep("collect").SetText("Collect their info.").SetValidSteps([]string{"greet"})
		out["state_contexts_navigation"] = contextsNav(cb)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "state-dump: encode failed: %v\n", err)
		os.Exit(1)
	}
}

// submitAnswerDelta drives InfoGatherer.SubmitAnswer and reduces the result to
// the observable delta (mirrors diff_port_state._observe "submit_answer_delta"):
// the set_global_data action's question_index + answers, plus a `done` flag
// derived from the completion message.
func submitAnswerDelta(ig *prefabs.InfoGathererAgent, args, rawData map[string]any) map[string]any {
	res := ig.SubmitAnswer(args, rawData)
	m := res.ToMap()
	var gd map[string]any
	if actions, ok := m["action"].([]map[string]any); ok {
		for _, act := range actions {
			if v, ok := act["set_global_data"].(map[string]any); ok {
				gd = v
				break
			}
		}
	}
	// action may serialize as []any depending on ToMap; handle both.
	if gd == nil {
		if actions, ok := m["action"].([]any); ok {
			for _, a := range actions {
				if act, ok := a.(map[string]any); ok {
					if v, ok := act["set_global_data"].(map[string]any); ok {
						gd = v
						break
					}
				}
			}
		}
	}
	resp, _ := m["response"].(string)
	return map[string]any{
		"question_index": gd["question_index"],
		"answers":        gd["answers"],
		// `done` mirrors the oracle's _is_complete: the completion message contains
		// "All questions have been answered" (vs a next-question instruction).
		"done": strings.Contains(resp, "All questions have been answered"),
	}
}

// contextsNav renders the builder and reduces to per-context {name, valid_steps}.
func contextsNav(cb *contexts.ContextBuilder) map[string]any {
	m, err := cb.ToMap()
	if err != nil {
		fmt.Fprintf(os.Stderr, "state-dump: contexts ToMap failed: %v\n", err)
		os.Exit(1)
	}
	nav := map[string]any{}
	for cname, cdoc := range m {
		cm, _ := cdoc.(map[string]any)
		steps, _ := cm["steps"].([]map[string]any)
		reduced := make([]map[string]any, 0, len(steps))
		for _, s := range steps {
			reduced = append(reduced, map[string]any{
				"name":        s["name"],
				"valid_steps": s["valid_steps"],
			})
		}
		nav[cname] = reduced
	}
	return nav
}
