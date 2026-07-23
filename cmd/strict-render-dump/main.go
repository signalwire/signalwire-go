// Command strict-render-dump is the Go port's SWML STRICT-RENDER dump program
// for the cross-port strict-render differ
// (porting-sdk/scripts/diff_port_strict_render.py).
//
// The strict-render contract: building/rendering an SWML document with a
// MISSHAPEN config, an UNKNOWN verb, or a MISSPELLED/unknown key must RAISE
// (return an error / panic) — not silently drop or accept it. A VALID build
// must still render.
//
// For each strict_render_corpus case this program builds the document in Go
// idiom and reports the observed OUTCOME:
//
//	"raised" — the build returned an error or panicked (the contract's teeth)
//	"ok"     — the build completed cleanly
//
// It emits ONE JSON object mapping case-id -> "raised"|"ok" to stdout (JSON
// only). The differ compares each outcome against the python oracle.
//
// The corpus targets two objects:
//   - SWMLService cases exercise Service.ExecuteVerb(name, config) on a
//     schema-validation-ON service (add_verb in the python reference).
//   - AgentBase cases exercise the contexts / tool-registry surface:
//     DefineTool + DefineContexts -> AddContext -> AddStep -> SetText /
//     SetFunctions / SetValidContexts, then ContextBuilder.ToMap() (which runs
//     Validate — the python _ctx_validate).
//
// Run from the signalwire-go repo root:
//
//	go run ./cmd/strict-render-dump
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/v3/pkg/agent"
	"github.com/signalwire/signalwire-go/v3/pkg/contexts"
	"github.com/signalwire/signalwire-go/v3/pkg/swaig"
	"github.com/signalwire/signalwire-go/v3/pkg/swml"
)

// outcome runs build and classifies the result: any error OR panic is "raised";
// a clean return is "ok". Mirrors the python differ's try/except -> raised.
func outcome(build func() error) (result string) {
	defer func() {
		if r := recover(); r != nil {
			result = "raised"
		}
	}()
	if err := build(); err != nil {
		return "raised"
	}
	return "ok"
}

// newService builds a schema-validation-ON SWMLService (the add_verb target).
func newService() *swml.Service {
	return swml.NewService(
		swml.WithName("s"),
		swml.WithRoute("/s"),
		swml.WithSchemaValidation(true),
	)
}

// addVerb is the SWMLService corpus verb: add_verb(name, config).
func addVerb(name string, config map[string]any) func() error {
	return func() error {
		return newService().ExecuteVerb(name, config)
	}
}

// ---- AgentBase contexts helpers (the _ctx_* corpus verbs) -----------------

// noopHandler is a define_tool handler stand-in (the corpus only needs the tool
// registered, never invoked).
func noopHandler(map[string]any, map[string]any) *swaig.FunctionResult { return nil }

func main() {
	out := map[string]string{}

	// ================================================================
	// Verb-level strict render (SWMLService, validation ON)
	// ================================================================
	out["strict_unknown_verb"] = outcome(addVerb("foobar", map[string]any{}))
	out["strict_answer_misspelled_key"] = outcome(addVerb("answer", map[string]any{"maxduration": 5}))
	out["strict_answer_unknown_key"] = outcome(addVerb("answer", map[string]any{"wibble": 1}))
	out["strict_play_misspelled_key"] = outcome(addVerb("play", map[string]any{"urlz": []any{"say:hi"}}))
	out["strict_play_valid_plus_unknown_key"] = outcome(addVerb("play", map[string]any{"url": "say:hi", "foo": 1}))
	out["strict_record_misspelled_key"] = outcome(addVerb("record", map[string]any{"formatt": "wav"}))
	out["strict_answer_wrong_type"] = outcome(addVerb("answer", map[string]any{"max_duration": "notanumber"}))
	out["strict_ai_misspelled_top_key"] = outcome(addVerb("ai", map[string]any{"prompt": map[string]any{"text": "hi"}, "temperatur": 0.5}))
	out["strict_ai_unknown_top_key"] = outcome(addVerb("ai", map[string]any{"prompt": map[string]any{"text": "hi"}, "zzz": 1}))
	out["strict_ai_missing_prompt"] = outcome(addVerb("ai", map[string]any{"post_prompt": map[string]any{"text": "bye"}}))

	// good documents must still render
	out["strict_answer_ok"] = outcome(addVerb("answer", map[string]any{"max_duration": 5}))
	out["strict_play_ok"] = outcome(addVerb("play", map[string]any{"url": "say:hi"}))
	out["strict_ai_ok"] = outcome(addVerb("ai", map[string]any{"prompt": map[string]any{"text": "hi"}}))
	out["strict_ai_params_open_ok"] = outcome(addVerb("ai", map[string]any{
		"prompt": map[string]any{"text": "hi"},
		"params": map[string]any{"some_future_param": 1},
	}))

	// ================================================================
	// Contexts-level strict render (AgentBase; dangling refs)
	// ================================================================

	// strict_dangling_step_function: order_status registered, step whitelists
	// an unregistered non-native 'get_datetime' -> dangling -> raise.
	out["strict_dangling_step_function"] = outcome(func() error {
		a := agent.NewAgentBase(agent.WithName("a"), agent.WithRoute("/a"))
		a.DefineTool(agent.ToolDefinition{Name: "order_status", Description: "look up an order", Parameters: map[string]any{}, Handler: noopHandler})
		cb := a.DefineContexts()
		st := cb.AddContext("default").AddStep("help")
		st.SetText("help")
		st.SetFunctions([]string{"order_status", "get_datetime"})
		_, err := cb.ToMap()
		return err
	})

	// strict_registered_step_function_ok: step whitelists a registered tool.
	out["strict_registered_step_function_ok"] = outcome(func() error {
		a := agent.NewAgentBase(agent.WithName("a"), agent.WithRoute("/a"))
		a.DefineTool(agent.ToolDefinition{Name: "order_status", Description: "look up an order", Parameters: map[string]any{}, Handler: noopHandler})
		cb := a.DefineContexts()
		st := cb.AddContext("default").AddStep("help")
		st.SetText("help")
		st.SetFunctions([]string{"order_status"})
		_, err := cb.ToMap()
		return err
	})

	// strict_reserved_native_function_ok: reserved natives are not dangling.
	out["strict_reserved_native_function_ok"] = outcome(func() error {
		a := agent.NewAgentBase(agent.WithName("a"), agent.WithRoute("/a"))
		cb := a.DefineContexts()
		st := cb.AddContext("default").AddStep("help")
		st.SetText("help")
		st.SetFunctions([]string{"next_step", "change_context"})
		_, err := cb.ToMap()
		return err
	})

	// strict_dangling_valid_context: valid_contexts references an undefined context.
	out["strict_dangling_valid_context"] = outcome(func() error {
		a := agent.NewAgentBase(agent.WithName("a"), agent.WithRoute("/a"))
		cb := a.DefineContexts()
		st := cb.AddContext("default").AddStep("help")
		st.SetText("help")
		st.SetValidContexts([]string{"nowhere"})
		_, err := cb.ToMap()
		return err
	})

	// Reference contexts import so the package is used even if a future refactor
	// drops the direct reference above (keeps the idiom explicit).
	_ = contexts.ReservedNativeToolNames

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "strict-render-dump: encode failed: %v\n", err)
		os.Exit(1)
	}
}
